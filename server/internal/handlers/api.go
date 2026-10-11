package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

type API struct {
	openapi.Unimplemented
	cfg         *config.Config
	cfgFilePath string

	dbMu sync.RWMutex
	conn *sql.DB
	st   *store.Store
}

// New creates the API. conn may be nil when no database is configured yet.
func New(cfg *config.Config, conn *sql.DB) *API {
	cfgPath, _ := config.DefaultConfigFilePath()
	if envPath := os.Getenv("SERVER_CONFIG_FILE"); envPath != "" {
		cfgPath = envPath
	}
	a := &API{cfg: cfg, cfgFilePath: cfgPath}
	if conn != nil {
		a.conn, a.st = conn, store.New(conn)
	}
	return a
}

// store returns the active store, or nil if no database is open.
func (a *API) store() *store.Store {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	return a.st
}

// requireStore returns the active store, or writes a 503 error and returns
// false when no database is open.
func (a *API) requireStore(w http.ResponseWriter) (*store.Store, bool) {
	st := a.store()
	if st == nil {
		writeJSON(w, http.StatusServiceUnavailable, openapi.ErrorResponse{Error: "database not configured yet"})
		return nil, false
	}
	return st, true
}

// openDatabase opens (and migrates) the SQLite database at path and makes it
// the active one, closing the previous connection.
func (a *API) openDatabase(ctx context.Context, path string) error {
	conn, err := dbpkg.Open(ctx, path)
	if err != nil {
		return err
	}
	a.dbMu.Lock()
	old := a.conn
	a.conn, a.st = conn, store.New(conn)
	a.dbMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// writeStoreError maps store errors to HTTP responses.
func writeStoreError(w http.ResponseWriter, err error, notFound, failure string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: notFound})
	case errors.Is(err, store.ErrConflict):
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Error: "a record with the same unique value already exists"})
	case errors.Is(err, store.ErrInvalidReference):
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "referenced record does not exist"})
	default:
		log.Printf("%s: %v", failure, err)
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: failure})
	}
}

// currentUserID extracts the authenticated user's e-mail from the JWT and uses
// it as the audit-log user identifier.
func currentUserID(r *http.Request) string {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return "unknown"
	}
	return claims.Email
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
	databaseConfigured := false
	if a.cfg.DatabasePath != "" {
		_, err := os.Stat(dbpkg.CleanPath(a.cfg.DatabasePath))
		databaseConfigured = err == nil
	}

	setupRequired := a.cfg.AdminPassword == "" || a.store() == nil

	writeJSON(w, http.StatusOK, openapi.SetupStatus{
		SetupRequired:      setupRequired,
		UserCount:          0,
		DatabaseConfigured: databaseConfigured,
		DatabaseMode:       openapi.SqliteLocal,
	})
}

// ConfigureSetupDatabase opens (creating and migrating) the SQLite database at
// the given path and persists the path on success.
func (a *API) ConfigureSetupDatabase(w http.ResponseWriter, r *http.Request) {
	var req openapi.ConfigureDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	if !req.Mode.Valid() {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "unsupported database mode"})
		return
	}

	path := defaultDatabasePath
	if req.DatabaseUrl != nil && *req.DatabaseUrl != "" {
		path = *req.DatabaseUrl
	}
	if !a.switchDatabase(w, r, path) {
		return
	}

	writeJSON(w, http.StatusOK, openapi.ConfigureDatabaseResponse{
		Mode:        openapi.SqliteLocal,
		DatabaseUrl: a.cfg.DatabasePath,
	})
}

// defaultDatabasePath is used when setup does not name a database file.
const defaultDatabasePath = "./clubmanager.db"

// switchDatabase opens the database at path, makes it active and persists the
// path. It writes an error response and returns false on failure.
func (a *API) switchDatabase(w http.ResponseWriter, r *http.Request, path string) bool {
	if err := a.openDatabase(r.Context(), path); err != nil {
		log.Printf("open database %s: %v", path, err)
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to open database: " + err.Error()})
		return false
	}
	a.cfg.DatabasePath = path
	if err := config.Save(*a.cfg, a.cfgFilePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save configuration"})
		return false
	}
	return true
}

// InitializeSetup hashes the admin password and saves the admin credentials to
// the config file, completing the first-run setup.
func (a *API) InitializeSetup(w http.ResponseWriter, r *http.Request) {
	var req openapi.SetupInitializeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if string(req.Email) == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "email and password are required"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
		return
	}

	a.cfg.AdminEmail = string(req.Email)
	a.cfg.AdminPassword = string(hash)

	if err := config.Save(*a.cfg, a.cfgFilePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save configuration"})
		return
	}

	writeJSON(w, http.StatusCreated, openapi.SuccessResponse{Success: true})
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

	email := string(req.Email)
	role := openapi.ADMIN

	// Try the config-based admin first (backwards compatibility).
	if email == a.cfg.AdminEmail {
		if bcrypt.CompareHashAndPassword([]byte(a.cfg.AdminPassword), []byte(req.Password)) != nil {
			writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
			return
		}
	} else if st := a.store(); st != nil {
		// Fall back to database users.
		user, err := st.GetUserByEmail(r.Context(), email)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
			writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
			return
		}
		role = openapi.AppRole(user.AppRole)
	} else {
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
		return
	}

	token, err := auth.NewToken(email, role, a.cfg.JWTSecret)
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
	writeJSON(w, http.StatusOK, openapi.UserProfile{
		Email:   openapi_types.Email(claims.Email),
		AppRole: &claims.AppRole,
	})
}

// Logout is a no-op for JWT-based auth – the client discards the token.
func (a *API) Logout(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// RefreshAccessToken issues a new access token when the caller still holds a
// valid token (stateless refresh – no refresh-token table lookup required for
// the single-admin desktop deployment).
func (a *API) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "unauthorized"})
		return
	}
	token, err := auth.NewToken(claims.Email, claims.AppRole, a.cfg.JWTSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to refresh token"})
		return
	}
	writeJSON(w, http.StatusOK, openapi.AccessTokenResponse{AccessToken: token})
}

func (a *API) GetDbPath(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"dbPath": a.cfg.DatabasePath})
}

func (a *API) GetDatabaseSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openapi.ConfigureDatabaseResponse{
		Mode:        openapi.SqliteLocal,
		DatabaseUrl: a.cfg.DatabasePath,
	})
}

// UpdateDatabaseSettings switches to the SQLite database at the given path.
func (a *API) UpdateDatabaseSettings(w http.ResponseWriter, r *http.Request) {
	var req openapi.DatabaseSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	if !req.Mode.Valid() || req.DatabaseUrl == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "a database path is required"})
		return
	}
	if !a.switchDatabase(w, r, req.DatabaseUrl) {
		return
	}
	writeJSON(w, http.StatusOK, openapi.ConfigureDatabaseResponse{
		Mode:        openapi.SqliteLocal,
		DatabaseUrl: req.DatabaseUrl,
	})
}

// SetDbPath updates the SQLite database path.
func (a *API) SetDbPath(w http.ResponseWriter, r *http.Request) {
	var req openapi.SetDbPathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DbPath == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	if !a.switchDatabase(w, r, req.DbPath) {
		return
	}
	dbPath := req.DbPath
	writeJSON(w, http.StatusOK, openapi.SetDbPathResponse{DbPath: &dbPath, Status: "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// saveConfig is a convenience helper used by handlers that modify the config.
func saveConfig(a *API) error {
	return config.Save(*a.cfg, a.cfgFilePath)
}
