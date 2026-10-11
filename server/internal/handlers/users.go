package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// requireAdmin returns true when the request's JWT claims show the ADMIN role.
// It writes a 403 Forbidden response and returns false otherwise.
func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.AppRole != openapi.ADMIN {
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Error: "admin role required"})
		return false
	}
	return true
}

// ListAppUsers returns all app users. Only admins may call this endpoint.
func (a *API) ListAppUsers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	users, err := st.ListUsers(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to query users")
		return
	}
	out := make([]openapi.AppUser, len(users))
	for i, u := range users {
		out[i] = toAPIUser(u)
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateAppUser creates a new app user. Only admins may call this endpoint.
func (a *API) CreateAppUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	st, ok := a.requireStore(w)
	if !ok {
		return
	}

	var req openapi.CreateAppUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if string(req.Email) == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "email and password are required"})
		return
	}

	if !req.AppRole.Valid() {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid appRole"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
		return
	}

	u, err := st.CreateUser(r.Context(), string(req.Email), string(hash), string(req.AppRole))
	if errors.Is(err, store.ErrConflict) {
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "email already in use"})
		return
	}
	if err != nil {
		writeStoreError(w, err, "", "failed to create user")
		return
	}
	writeJSON(w, http.StatusCreated, toAPIUser(u))
}

// UpdateAppUser updates an app user's email or role. Only admins may call this endpoint.
func (a *API) UpdateAppUser(w http.ResponseWriter, r *http.Request, id string) {
	if !requireAdmin(w, r) {
		return
	}
	st, ok := a.requireStore(w)
	if !ok {
		return
	}

	var req openapi.UpdateAppUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if req.AppRole != nil && !req.AppRole.Valid() {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid appRole"})
		return
	}

	var email, role *string
	if req.Email != nil {
		e := string(*req.Email)
		email = &e
	}
	if req.AppRole != nil {
		ar := string(*req.AppRole)
		role = &ar
	}

	u, err := st.UpdateUser(r.Context(), id, email, role)
	if errors.Is(err, store.ErrConflict) {
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "email already in use"})
		return
	}
	if err != nil {
		writeStoreError(w, err, "user not found", "failed to update user")
		return
	}
	writeJSON(w, http.StatusOK, toAPIUser(u))
}

// DeleteAppUser deletes an app user. Only admins may call this endpoint.
func (a *API) DeleteAppUser(w http.ResponseWriter, r *http.Request, id string) {
	if !requireAdmin(w, r) {
		return
	}
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	if err := st.DeleteUser(r.Context(), id); err != nil {
		writeStoreError(w, err, "user not found", "failed to delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword lets the currently authenticated user change their own password.
// Admins may omit currentPassword; other users must supply it.
func (a *API) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "unauthorized"})
		return
	}
	st, ok := a.requireStore(w)
	if !ok {
		return
	}

	var req openapi.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "newPassword is required"})
		return
	}
	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "newPassword must be at least 8 characters"})
		return
	}

	email := claims.Email

	// Check whether this is the config-based admin.
	if email == a.cfg.AdminEmail {
		if claims.AppRole != openapi.ADMIN && req.CurrentPassword != nil {
			if bcrypt.CompareHashAndPassword([]byte(a.cfg.AdminPassword), []byte(*req.CurrentPassword)) != nil {
				writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Error: "current password is incorrect"})
				return
			}
		} else if claims.AppRole != openapi.ADMIN && req.CurrentPassword == nil {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "currentPassword is required"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
			return
		}
		a.cfg.AdminPassword = string(hash)
		if err := saveConfig(a); err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save new password"})
			return
		}
		writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
		return
	}

	// DB user: verify current password unless caller is admin.
	user, err := st.GetUserByEmail(r.Context(), email)
	if err != nil {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "user not found"})
		return
	}

	if claims.AppRole != openapi.ADMIN {
		if req.CurrentPassword == nil || *req.CurrentPassword == "" {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "currentPassword is required"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(*req.CurrentPassword)) != nil {
			writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Error: "current password is incorrect"})
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
		return
	}

	if err := st.SetUserPassword(r.Context(), email, string(hash)); err != nil {
		writeStoreError(w, err, "user not found", "failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func toAPIUser(u store.User) openapi.AppUser {
	return openapi.AppUser{
		Id: u.ID, Email: openapi_types.Email(u.Email), AppRole: openapi.AppRole(u.AppRole), CreatedAt: u.CreatedAt,
	}
}
