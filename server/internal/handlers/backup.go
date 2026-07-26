package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// sqliteDBPath returns the cleaned file-system path from the configured
// DatabasePath (which may carry a "file:" prefix).
func (a *API) sqliteDBPath() (string, error) {
	if a.cfg.DatabasePath == "" {
		return "", fmt.Errorf("database path is not configured")
	}
	p := strings.TrimPrefix(a.cfg.DatabasePath, "file:")
	p = strings.TrimPrefix(p, "//")
	return filepath.Clean(p), nil
}

// BackupDatabase creates a ZIP archive containing the SQLite database file and
// streams it as a download. Returns 400 when running in mysql-shared mode.
func (a *API) BackupDatabase(w http.ResponseWriter, r *http.Request) {
	if a.cfg.DatabaseURL != "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{
			Error: "backup is only supported in sqlite-local mode; use your database server's own backup tools for mysql-shared mode",
		})
		return
	}

	dbPath, err := a.sqliteDBPath()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, openapi.ErrorResponse{Error: err.Error()})
		return
	}

	// Read DB file while holding the read-lock so the file is not replaced
	// concurrently by a restore operation.
	a.dbMu.RLock()
	data, err := os.ReadFile(dbPath)
	a.dbMu.RUnlock()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to read database file: " + err.Error()})
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	fw, err := zw.Create("clubmanager.db")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create zip entry"})
		return
	}
	if _, err := fw.Write(data); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to write zip entry"})
		return
	}
	if err := zw.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to finalize zip archive"})
		return
	}

	filename := fmt.Sprintf("clubmanager-backup-%s.zip", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// RestoreDatabase accepts a multipart/form-data upload containing a ZIP backup
// file, replaces the active SQLite database, and reconnects. Returns 400 when
// running in mysql-shared mode.
func (a *API) RestoreDatabase(w http.ResponseWriter, r *http.Request) {
	if a.cfg.DatabaseURL != "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{
			Error: "restore is only supported in sqlite-local mode; use your database server's own restore tools for mysql-shared mode",
		})
		return
	}

	dbPath, err := a.sqliteDBPath()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, openapi.ErrorResponse{Error: err.Error()})
		return
	}

	// Accept up to 100 MiB.
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "failed to parse multipart form: " + err.Error()})
		return
	}

	file, _, err := r.FormFile("backup")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "backup file is required (field name: backup)"})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to read uploaded file"})
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid zip file: " + err.Error()})
		return
	}

	var dbEntry *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".db") {
			dbEntry = f
			break
		}
	}
	if dbEntry == nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "no .db file found inside the backup archive"})
		return
	}

	rc, err := dbEntry.Open()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to open database entry in zip"})
		return
	}
	defer rc.Close()

	dbData, err := io.ReadAll(rc)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to read database from zip"})
		return
	}

	// Close the active connection before replacing the file.
	a.dbMu.Lock()
	if a.db != nil {
		_ = a.db.Close()
		a.db = nil
	}
	a.dbMu.Unlock()

	if err := os.WriteFile(dbPath, dbData, 0o600); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to write restored database file: " + err.Error()})
		return
	}

	// Reopen and migrate so the restored DB is immediately usable.
	if db, openErr := dbpkg.Open("sqlite-local", a.cfg.DatabasePath); openErr == nil {
		_ = dbpkg.Migrate(db)
		a.dbMu.Lock()
		a.db = db
		a.dbMu.Unlock()
	}

	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}
