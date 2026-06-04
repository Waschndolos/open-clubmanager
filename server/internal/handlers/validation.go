package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	path := strings.TrimSpace(req.Path)
	if path == "" {
		i18n := "error.emptypath"
		writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: false, I18nToken: &i18n})
		return
	}

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		i18n := "error.parentdirnotfound"
		writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: false, I18nToken: &i18n})
		return
	}

	// Try creating/opening the file to verify write permission.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		i18n := "error.notwritable"
		writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: false, I18nToken: &i18n})
		return
	}
	_ = f.Close()

	writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: true})
}

// ValidateDbUrl validates a database URL (MySQL connection string).
func (a *API) ValidateDbUrl(w http.ResponseWriter, r *http.Request) {
	var req openapi.ValidationDatabaseUrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	url := strings.TrimSpace(req.DatabaseUrl)
	if url == "" {
		i18n := "error.emptyurl"
		writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: false, I18nToken: &i18n})
		return
	}

	if req.Mode == openapi.MysqlShared {
		lower := strings.ToLower(url)
		if !strings.HasPrefix(lower, "mysql://") && !strings.HasPrefix(lower, "mysqls://") {
			i18n := "error.invalidmysqlurl"
			writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: false, I18nToken: &i18n})
			return
		}
	}

	writeJSON(w, http.StatusOK, openapi.ValidationResponse{Valid: true})
}

