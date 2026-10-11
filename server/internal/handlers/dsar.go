package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/dsar"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ExportMemberData creates the data subject access report (GDPR Art. 15/20)
// for one member as a PDF (default) or JSON download:
// GET /api/v2/members/{id}/data-export?format=pdf|json.
//
// Only users who manage members (admin, secretary) and whose device may read
// every scope the report covers get it. The export is written to the audit log.
func (a *API) ExportMemberData(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || (claims.AppRole != openapi.ADMIN && claims.AppRole != openapi.SECRETARY) {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not create a data export"})
		return
	}
	for _, sc := range dsar.Scopes() {
		if !a.collab.CanRead(r.Context(), sc) {
			code := "FORBIDDEN"
			writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role has no full access to this member's data"})
			return
		}
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}
	if format != "pdf" && format != "json" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "format must be pdf or json"})
		return
	}

	id := chi.URLParam(r, "id")
	report, err := dsar.Build(r.Context(), st, id, claims.Email)
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to create data export")
		return
	}
	st.Audit(r.Context(), "CREATE", "MemberDataExport", id, claims.Email)

	filename := fmt.Sprintf("data-export-%s.%s", id, format)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	_, _ = w.Write(report.PDF())
}
