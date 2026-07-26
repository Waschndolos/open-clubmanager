package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

type API struct {
	openapi.Unimplemented
	cfg         *config.Config
	cfgFilePath string
	db          *sql.DB
	dbMu        sync.RWMutex
}

func New(cfg *config.Config, db *sql.DB) *API {
	cfgPath, _ := config.DefaultConfigFilePath()
	if envPath := os.Getenv("SERVER_CONFIG_FILE"); envPath != "" {
		cfgPath = envPath
	}
	return &API{cfg: cfg, cfgFilePath: cfgPath, db: db}
}

// getDB returns the active database connection, or nil if not yet initialised.
func (a *API) getDB() *sql.DB {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	return a.db
}

// requireDB writes a 503 error and returns false when the DB is unavailable.
func (a *API) requireDB(w http.ResponseWriter) bool {
	if a.getDB() == nil {
		writeJSON(w, http.StatusServiceUnavailable, openapi.ErrorResponse{Error: "database not configured yet"})
		return false
	}
	return true
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
	mode := openapi.SqliteLocal
	if a.cfg.DatabaseURL != "" {
		mode = openapi.MysqlShared
	}

	databaseConfigured := a.cfg.DatabasePath != "" || a.cfg.DatabaseURL != ""
	if mode == openapi.SqliteLocal {
		sqlitePath := strings.TrimPrefix(a.cfg.DatabasePath, "file:")
		sqlitePath = strings.TrimPrefix(sqlitePath, "//")
		sqlitePath = filepath.Clean(sqlitePath)
		if sqlitePath == "" {
			databaseConfigured = false
		} else if _, err := os.Stat(sqlitePath); err != nil {
			databaseConfigured = false
		}
	}

	setupRequired := a.cfg.AdminPassword == "" || a.getDB() == nil

	writeJSON(w, http.StatusOK, openapi.SetupStatus{
		SetupRequired:      setupRequired,
		UserCount:          0,
		DatabaseConfigured: databaseConfigured,
		DatabaseMode:       mode,
	})
}

// ConfigureSetupDatabase persists the chosen database mode and connection
// string, then immediately opens the database and runs migrations.
func (a *API) ConfigureSetupDatabase(w http.ResponseWriter, r *http.Request) {
	var req openapi.ConfigureDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if req.Mode == openapi.MysqlShared {
		if req.DatabaseUrl == nil || *req.DatabaseUrl == "" {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "databaseUrl is required for mysql-shared mode"})
			return
		}
		a.cfg.DatabaseURL = *req.DatabaseUrl
		a.cfg.DatabasePath = ""
	} else {
		url := "./clubmanager.db"
		if req.DatabaseUrl != nil && *req.DatabaseUrl != "" {
			url = *req.DatabaseUrl
		}
		a.cfg.DatabasePath = url
		a.cfg.DatabaseURL = ""
	}

	if err := config.Save(*a.cfg, a.cfgFilePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save configuration"})
		return
	}

	// Open DB connection immediately so subsequent handlers can use it.
	mode := string(req.Mode)
	dsn := a.cfg.DatabasePath
	if a.cfg.DatabaseURL != "" {
		dsn = a.cfg.DatabaseURL
	}
	if db, err := dbpkg.Open(mode, dsn); err == nil {
		if migrateErr := dbpkg.Migrate(db); migrateErr == nil {
			a.dbMu.Lock()
			a.db = db
			a.dbMu.Unlock()
		}
	}

	writeJSON(w, http.StatusOK, openapi.ConfigureDatabaseResponse{
		Mode:        req.Mode,
		DatabaseUrl: a.cfg.DatabasePath + a.cfg.DatabaseURL,
	})
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
	} else if db := a.getDB(); db != nil {
		// Fall back to database users.
		var hashedPwd, appRole string
		err := db.QueryRow(
			`SELECT "password", "appRole" FROM "User" WHERE "email" = ?`, email,
		).Scan(&hashedPwd, &appRole)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.Password)) != nil {
			writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "invalid credentials"})
			return
		}
		role = openapi.AppRole(appRole)
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

// UpdateDatabaseSettings updates the database configuration and reconnects.
func (a *API) UpdateDatabaseSettings(w http.ResponseWriter, r *http.Request) {
	var req openapi.DatabaseSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if req.Mode == openapi.MysqlShared {
		a.cfg.DatabaseURL = req.DatabaseUrl
		a.cfg.DatabasePath = ""
	} else {
		a.cfg.DatabasePath = req.DatabaseUrl
		a.cfg.DatabaseURL = ""
	}

	if err := config.Save(*a.cfg, a.cfgFilePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save configuration"})
		return
	}

	if db, err := dbpkg.Open(string(req.Mode), req.DatabaseUrl); err == nil {
		_ = dbpkg.Migrate(db)
		a.dbMu.Lock()
		a.db = db
		a.dbMu.Unlock()
	}

	writeJSON(w, http.StatusOK, openapi.ConfigureDatabaseResponse{
		Mode:        req.Mode,
		DatabaseUrl: req.DatabaseUrl,
	})
}

// SetDbPath updates the SQLite database path.
func (a *API) SetDbPath(w http.ResponseWriter, r *http.Request) {
	var req openapi.SetDbPathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	a.cfg.DatabasePath = req.DbPath
	a.cfg.DatabaseURL = ""

	if err := config.Save(*a.cfg, a.cfgFilePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save configuration"})
		return
	}

	if db, err := dbpkg.Open("sqlite-local", req.DbPath); err == nil {
		_ = dbpkg.Migrate(db)
		a.dbMu.Lock()
		a.db = db
		a.dbMu.Unlock()
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
