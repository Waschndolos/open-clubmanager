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

// BackupDatabase creates a ZIP archive containing a consistent snapshot of
// the SQLite database and streams it as a download.
func (a *API) BackupDatabase(w http.ResponseWriter, r *http.Request) {
	a.dbMu.RLock()
	conn := a.conn
	a.dbMu.RUnlock()
	if conn == nil {
		writeJSON(w, http.StatusServiceUnavailable, openapi.ErrorResponse{Error: "database not configured yet"})
		return
	}

	// VACUUM INTO writes a consistent copy including data still in the WAL,
	// which copying the database file would miss.
	tmpDir, err := os.MkdirTemp("", "clubmanager-backup-")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create temporary directory"})
		return
	}
	defer os.RemoveAll(tmpDir)
	snapshot := filepath.Join(tmpDir, "clubmanager.db")
	if _, err := conn.ExecContext(r.Context(), `VACUUM INTO ?`, snapshot); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to snapshot database: " + err.Error()})
		return
	}
	data, err := os.ReadFile(snapshot)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to read database snapshot: " + err.Error()})
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
// file, replaces the active SQLite database, and reconnects. Backups taken
// before the UUID schema are migrated on reopen.
func (a *API) RestoreDatabase(w http.ResponseWriter, r *http.Request) {
	if a.cfg.DatabasePath == "" {
		writeJSON(w, http.StatusServiceUnavailable, openapi.ErrorResponse{Error: "database path is not configured"})
		return
	}
	dbPath := dbpkg.CleanPath(a.cfg.DatabasePath)

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

	// Stage the upload next to the database so the final rename is atomic.
	staged := dbPath + ".restore"
	if err := os.WriteFile(staged, dbData, 0o600); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to write restored database file: " + err.Error()})
		return
	}
	defer os.Remove(staged)

	// Close the active connection before replacing the file, and drop its WAL
	// so stale pages are not replayed into the restored database.
	a.dbMu.Lock()
	if a.conn != nil {
		_ = a.conn.Close()
		a.conn, a.st = nil, nil
	}
	a.dbMu.Unlock()
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	if err := os.Rename(staged, dbPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to replace database file: " + err.Error()})
		return
	}

	// Reopen (and migrate) so the restored DB is immediately usable.
	if err := a.openDatabase(r.Context(), dbPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to open restored database: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}
