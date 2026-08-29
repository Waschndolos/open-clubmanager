package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
)

type documentRecord struct {
	Id          int       `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Category    string    `json:"category"`
	Filename    string    `json:"filename"`
	StoragePath string    `json:"storagePath"`
	Size        int64     `json:"size"`
	MimeType    string    `json:"mimeType"`
	UploadedBy  string    `json:"uploadedBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type updateDocumentRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
}

func (a *API) ListDocuments(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	query := `SELECT "id","title","description","category","filename","storagePath","size","mimeType","uploadedBy","createdAt","updatedAt" FROM "Document"`
	args := make([]interface{}, 0, 1)
	if category != "" {
		query += ` WHERE "category"=?`
		args = append(args, category)
	}
	query += ` ORDER BY "createdAt" DESC`

	rows, err := a.getDB().Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list documents"})
		return
	}
	defer rows.Close()

	result := make([]documentRecord, 0)
	for rows.Next() {
		doc, scanErr := scanDocument(rows)
		if scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to scan document"})
			return
		}
		result = append(result, doc)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}

	maxBytes := int64(a.cfg.DocumentMaxMB) * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+(1<<20))
	if err := r.ParseMultipartForm(maxBytes + (1 << 20)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "too large") {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds maximum upload size"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form data"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	if header.Size > maxBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds maximum upload size"})
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category is required"})
		return
	}
	description := strings.TrimSpace(r.FormValue("description"))
	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	dbDir := databaseDirectory(a.cfg.DatabasePath)
	docsDir := filepath.Join(dbDir, "documents")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create document storage"})
		return
	}

	safeOriginal := sanitizeFilename(header.Filename)
	if safeOriginal == "" {
		safeOriginal = "document.bin"
	}
	storedName := fmt.Sprintf("%d-%s", time.Now().UTC().UnixNano(), safeOriginal)
	storagePath := filepath.Join("documents", storedName)
	fullPath, pathErr := resolveStoredDocumentPath(dbDir, storagePath)
	if pathErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve document path"})
		return
	}

	targetFile, err := os.OpenFile(fullPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store uploaded file"})
		return
	}
	defer targetFile.Close()

	copied, err := io.Copy(targetFile, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store uploaded file"})
		return
	}
	if copied > maxBytes {
		_ = os.Remove(fullPath)
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds maximum upload size"})
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(safeOriginal))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.getDB().Exec(
		`INSERT INTO "Document" ("title","description","category","filename","storagePath","size","mimeType","uploadedBy","createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?,?,?)`,
		title, optStr(descriptionPtr), category, header.Filename, storagePath, copied, mimeType, currentUserID(r), now, now,
	)
	if err != nil {
		_ = os.Remove(fullPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save document metadata"})
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		_ = os.Remove(fullPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save document metadata"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "CREATE", "Document", int(id64), currentUserID(r), "")

	doc, err := getDocumentByID(a.getDB(), int(id64))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load uploaded document"})
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *API) DownloadDocument(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	doc, err := getDocumentByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load document"})
		return
	}

	fullPath, pathErr := resolveStoredDocumentPath(databaseDirectory(a.cfg.DatabasePath), doc.StoragePath)
	if pathErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid document path"})
		return
	}

	file, err := os.Open(fullPath)
	if os.IsNotExist(err) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document file not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to open document"})
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", doc.MimeType)
	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": doc.Filename})
	if contentDisposition == "" {
		contentDisposition = fmt.Sprintf("attachment; filename=%q", doc.Filename)
	}
	w.Header().Set("Content-Disposition", contentDisposition)
	w.Header().Set("Content-Length", strconv.FormatInt(doc.Size, 10))
	http.ServeContent(w, r, doc.Filename, doc.UpdatedAt, file)
}

func (a *API) UpdateDocument(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	var req updateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
		return
	}

	doc, err := getDocumentByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load document"})
		return
	}

	title := doc.Title
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
	}
	if title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	category := doc.Category
	if req.Category != nil {
		category = strings.TrimSpace(*req.Category)
	}
	if category == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category is required"})
		return
	}

	description := doc.Description
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.getDB().Exec(
		`UPDATE "Document" SET "title"=?,"description"=?,"category"=?,"updatedAt"=? WHERE "id"=?`,
		title, optStr(description), category, now, id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update document"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", "Document", id, currentUserID(r), "")

	updated, err := getDocumentByID(a.getDB(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load updated document"})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *API) DeleteDocument(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}

	doc, err := getDocumentByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load document"})
		return
	}

	fullPath, pathErr := resolveStoredDocumentPath(databaseDirectory(a.cfg.DatabasePath), doc.StoragePath)
	if pathErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid document path"})
		return
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete document file"})
		return
	}

	_, err = a.getDB().Exec(`DELETE FROM "Document" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete document"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "Document", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func getDocumentByID(db *sql.DB, id int) (documentRecord, error) {
	row := db.QueryRow(
		`SELECT "id","title","description","category","filename","storagePath","size","mimeType","uploadedBy","createdAt","updatedAt" FROM "Document" WHERE "id"=?`,
		id,
	)
	return scanDocumentRow(row)
}

func scanDocument(rows *sql.Rows) (documentRecord, error) {
	var doc documentRecord
	var description sql.NullString
	var createdAtStr, updatedAtStr string
	err := rows.Scan(
		&doc.Id, &doc.Title, &description, &doc.Category, &doc.Filename, &doc.StoragePath, &doc.Size, &doc.MimeType, &doc.UploadedBy, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return doc, err
	}
	doc.Description = nullStrPtr(description)
	doc.CreatedAt, _ = parseTime(createdAtStr)
	doc.UpdatedAt, _ = parseTime(updatedAtStr)
	return doc, nil
}

func scanDocumentRow(row *sql.Row) (documentRecord, error) {
	var doc documentRecord
	var description sql.NullString
	var createdAtStr, updatedAtStr string
	err := row.Scan(
		&doc.Id, &doc.Title, &description, &doc.Category, &doc.Filename, &doc.StoragePath, &doc.Size, &doc.MimeType, &doc.UploadedBy, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return doc, err
	}
	doc.Description = nullStrPtr(description)
	doc.CreatedAt, _ = parseTime(createdAtStr)
	doc.UpdatedAt, _ = parseTime(updatedAtStr)
	return doc, nil
}

func databaseDirectory(databasePath string) string {
	path := strings.TrimPrefix(databasePath, "file:")
	path = strings.TrimPrefix(path, "//")
	if path == "" {
		return "."
	}
	return filepath.Dir(filepath.Clean(path))
}

func sanitizeFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == "" {
		return ""
	}
	var out strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			out.WriteRune(r)
		} else {
			out.WriteRune('_')
		}
	}
	return strings.Trim(out.String(), "._")
}

func resolveStoredDocumentPath(dbDir, storagePath string) (string, error) {
	cleaned := filepath.Clean(storagePath)
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", errors.New("invalid storage path")
	}
	fullPath := filepath.Clean(filepath.Join(dbDir, cleaned))
	return fullPath, nil
}
