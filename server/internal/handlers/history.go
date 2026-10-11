package handlers

import (
	"net/http"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ListHistory returns all audit-log entries, newest first.
func (a *API) ListHistory(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	entries, err := st.ListAudit(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list history")
		return
	}
	result := make([]openapi.AuditLog, len(entries))
	for i, e := range entries {
		result[i] = openapi.AuditLog{
			Id: e.ID, Action: openapi.AuditLogAction(e.Action), Entity: e.Entity, EntityId: e.EntityID,
			UserId: e.UserID, Data: e.Data, CreatedAt: e.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, result)
}
