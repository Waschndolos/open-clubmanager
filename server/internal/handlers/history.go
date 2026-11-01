package handlers

import (
	"net/http"
	"slices"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// financeAuditEntities are the audit-log entities that belong to the
// finances: the cash book, member fees and fee models, direct debits, the
// payment reconciliation and donation receipts. Only roles that may read the
// finances see their history entries.
var financeAuditEntities = map[string]bool{
	"FinanceTransaction": true,
	"Transaction":        true, // booked from an inventory maintenance
	"MemberFee":          true,
	"FeeModel":           true,
	"MemberFeeModel":     true,
	"FeeRun":             true,
	"SepaCreditor":       true,
	"SepaMandate":        true,
	"SepaRun":            true,
	"Reconciliation":     true,
	"Donor":              true,
	"Donation":           true,
	"DonationReceipt":    true,
	"DonationSettings":   true,
}

// canReadFinance reports whether the login behind r may read the finances on
// this device, like financeAccess but without writing a response.
func (a *API) canReadFinance(r *http.Request) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	return ok && slices.Contains(financeEditors, claims.AppRole) && a.collab.CanRead(r.Context(), db.ScopeFinance)
}

// ListHistory returns all audit-log entries, newest first. Entries about
// finances are left out for roles that may not read the finances.
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
	finance := a.canReadFinance(r)
	result := make([]openapi.AuditLog, 0, len(entries))
	for _, e := range entries {
		if !finance && financeAuditEntities[e.Entity] {
			continue
		}
		result = append(result, openapi.AuditLog{
			Id: e.ID, Action: openapi.AuditLogAction(e.Action), Entity: e.Entity, EntityId: e.EntityID,
			UserId: e.UserID, Data: e.Data, CreatedAt: e.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, result)
}
