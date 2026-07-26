package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
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
	if !a.requireDB(w) {
		return
	}

	rows, err := a.db.Query(`SELECT "id", "email", "appRole", "createdAt" FROM "User" ORDER BY "id"`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to query users"})
		return
	}
	defer rows.Close()

	users := []openapi.AppUser{}
	for rows.Next() {
		var u openapi.AppUser
		var createdAt string
		if err := rows.Scan(&u.Id, &u.Email, &u.AppRole, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to read user row"})
			return
		}
		u.CreatedAt = parseTimestamp(createdAt)
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to iterate users"})
		return
	}

	writeJSON(w, http.StatusOK, users)
}

// CreateAppUser creates a new app user. Only admins may call this endpoint.
func (a *API) CreateAppUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if !a.requireDB(w) {
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

	result, err := a.db.Exec(
		`INSERT INTO "User" ("email", "password", "appRole") VALUES (?, ?, ?)`,
		string(req.Email), string(hash), string(req.AppRole),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "email already in use"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create user"})
		return
	}

	id, _ := result.LastInsertId()
	u, ok := a.loadUserByID(w, int(id))
	if !ok {
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// UpdateAppUser updates an app user's email or role. Only admins may call this endpoint.
func (a *API) UpdateAppUser(w http.ResponseWriter, r *http.Request, id int) {
	if !requireAdmin(w, r) {
		return
	}
	if !a.requireDB(w) {
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

	// Build an update query using only the fields that were provided.
	setClauses := []string{}
	args := []any{}

	if req.Email != nil {
		setClauses = append(setClauses, `"email" = ?`)
		args = append(args, string(*req.Email))
	}
	if req.AppRole != nil {
		setClauses = append(setClauses, `"appRole" = ?`)
		args = append(args, string(*req.AppRole))
	}

	if len(setClauses) == 0 {
		u, ok := a.loadUserByID(w, id)
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, u)
		return
	}

	query := `UPDATE "User" SET `
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += `, "updatedAt" = CURRENT_TIMESTAMP WHERE "id" = ?`
	args = append(args, id)

	res, err := a.db.Exec(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update user"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "user not found"})
		return
	}

	u, ok := a.loadUserByID(w, id)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// DeleteAppUser permanently deletes an app user. Only admins may call this endpoint.
func (a *API) DeleteAppUser(w http.ResponseWriter, r *http.Request, id int) {
	if !requireAdmin(w, r) {
		return
	}
	if !a.requireDB(w) {
		return
	}

	res, err := a.db.Exec(`DELETE FROM "User" WHERE "id" = ?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete user"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "user not found"})
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
	if !a.requireDB(w) {
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
	var storedHash string
	err := a.db.QueryRow(`SELECT "password" FROM "User" WHERE "email" = ?`, email).Scan(&storedHash)
	if err != nil {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "user not found"})
		return
	}

	if claims.AppRole != openapi.ADMIN {
		if req.CurrentPassword == nil || *req.CurrentPassword == "" {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "currentPassword is required"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(*req.CurrentPassword)) != nil {
			writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Error: "current password is incorrect"})
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
		return
	}

	if _, err := a.db.Exec(
		`UPDATE "User" SET "password" = ?, "updatedAt" = CURRENT_TIMESTAMP WHERE "email" = ?`,
		string(hash), email,
	); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update password"})
		return
	}

	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// loadUserByID fetches a single AppUser from the database by its primary key.
func (a *API) loadUserByID(w http.ResponseWriter, id int) (openapi.AppUser, bool) {
	var u openapi.AppUser
	var email string
	var createdAt string
	err := a.db.QueryRow(
		`SELECT "id", "email", "appRole", "createdAt" FROM "User" WHERE "id" = ?`, id,
	).Scan(&u.Id, &email, &u.AppRole, &createdAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "user not found"})
		return openapi.AppUser{}, false
	}
	u.Email = openapi_types.Email(email)
	u.CreatedAt = parseTimestamp(createdAt)
	return u, true
}

// parseTimestamp tries multiple time formats used by SQLite and MySQL.
func parseTimestamp(s string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// isUniqueConstraintError returns true for SQLite/MySQL unique constraint violations.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "Duplicate entry")
}
