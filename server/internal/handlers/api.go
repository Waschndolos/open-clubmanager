package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

type API struct {
	openapi.Unimplemented
	cfg config.Config
}

func New(cfg config.Config) *API {
	return &API{cfg: cfg}
}

func (a *API) GetSystemHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openapi.HealthResponse{
		Status:  "ok",
		Service: "open-clubmanager-api-v2",
	})
}

func (a *API) GetSystemMeta(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, openapi.MetaResponse{
		ApiVersion:            "v2",
		UsesLegacyJwtFallback: false,
		Now:                   now,
	})
}

func (a *API) GetSetupStatus(w http.ResponseWriter, _ *http.Request) {
	mode := openapi.SqliteLocal
	if a.cfg.DatabaseURL != "" {
		mode = openapi.MysqlShared
	}

	writeJSON(w, http.StatusOK, openapi.SetupStatus{
		SetupRequired:      a.cfg.AdminPassword == "",
		UserCount:          0,
		DatabaseConfigured: a.cfg.DatabasePath != "" || a.cfg.DatabaseURL != "",
		DatabaseMode:       mode,
	})
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	if a.cfg.AdminEmail == "" || a.cfg.AdminPassword == "" {
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "admin credentials not configured"})
		return
	}

	var req openapi.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if string(req.Email) != a.cfg.AdminEmail || bcrypt.CompareHashAndPassword([]byte(a.cfg.AdminPassword), []byte(req.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
		return
	}

	token, err := auth.NewToken(string(req.Email), a.cfg.JWTSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create access token"})
		return
	}

	writeJSON(w, http.StatusOK, openapi.AccessTokenResponse{AccessToken: token})
}

func (a *API) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, openapi.UserProfile{Email: openapi_types.Email(claims.Email)})
}

func (a *API) GetDbPath(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"dbPath": a.cfg.DatabasePath})
}

func (a *API) GetDatabaseSettings(w http.ResponseWriter, _ *http.Request) {
	mode := openapi.SqliteLocal
	url := a.cfg.DatabasePath
	if a.cfg.DatabaseURL != "" {
		mode = openapi.MysqlShared
		url = a.cfg.DatabaseURL
	}
	writeJSON(w, http.StatusOK, openapi.ConfigureDatabaseResponse{
		Mode:        mode,
		DatabaseUrl: url,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
