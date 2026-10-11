package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ValidateDbPath checks whether a given file system path can be used as a
// SQLite database (parent directory must exist and the path must be writable).
func (a *API) ValidateDbPath(w http.ResponseWriter, r *http.Request) {
	var req openapi.ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	writeJSON(w, http.StatusOK, validateSQLitePath(req.Path))
}

// validateSQLitePath checks that path can hold a SQLite database: the parent
// directory must exist and the file must be creatable/writable.
func validateSQLitePath(raw string) openapi.ValidationResponse {
	invalid := func(token string) openapi.ValidationResponse {
		return openapi.ValidationResponse{Valid: false, I18nToken: &token}
	}
	if strings.TrimSpace(raw) == "" {
		return invalid("error.emptypath")
	}
	path := dbpkg.CleanPath(strings.TrimSpace(raw))

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		return invalid("error.parentdirnotfound")
	}

	// Try creating/opening the file to verify write permission.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return invalid("error.notwritable")
	}
	_ = f.Close()
	return openapi.ValidationResponse{Valid: true}
}

// ValidateDbUrl validates a database location. Only local SQLite files are
// supported, so this applies the same checks as ValidateDbPath.
func (a *API) ValidateDbUrl(w http.ResponseWriter, r *http.Request) {
	var req openapi.ValidationDatabaseUrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	writeJSON(w, http.StatusOK, validateSQLitePath(req.DatabaseUrl))
}
