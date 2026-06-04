package handlers

import (
	"net/http"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ListHistory returns all audit-log entries, newest first.
func (a *API) ListHistory(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	rows, err := a.getDB().Query(
		`SELECT "id","action","entity","entityId","userId","data","createdAt" FROM "AuditLog" ORDER BY "createdAt" DESC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list history"})
		return
	}
	defer rows.Close()

	result := []openapi.AuditLog{}
	for rows.Next() {
		var entry openapi.AuditLog
		var createdAtStr string
		var data *string

		if err := rows.Scan(&entry.Id, (*string)(&entry.Action), &entry.Entity, &entry.EntityId, &entry.UserId, &data, &createdAtStr); err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan audit entry"})
			return
		}
		entry.Data = data
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			entry.CreatedAt = t
		}
		result = append(result, entry)
	}
	writeJSON(w, http.StatusOK, result)
}

