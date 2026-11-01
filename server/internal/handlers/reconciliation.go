package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/reconcile"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

type reconAllocationJSON struct {
	FeeID  string  `json:"feeId"`
	Amount float64 `json:"amount"`
}

type reconSuggestionJSON struct {
	Confidence  reconcile.Confidence  `json:"confidence"`
	Reasons     []string              `json:"reasons"`
	Allocations []reconAllocationJSON `json:"allocations"`
	Unallocated float64               `json:"unallocated"`
}

type reconPaymentJSON struct {
	ID          string          `json:"id"`
	FeeID       string          `json:"feeId"`
	Member      honorMemberJSON `json:"member"`
	Year        int             `json:"year"`
	Description *string         `json:"description,omitempty"`
	Amount      float64         `json:"amount"`
}

type reconEntryJSON struct {
	TransactionID    string               `json:"transactionId"`
	Date             string               `json:"date"`
	Description      string               `json:"description"`
	Amount           float64              `json:"amount"`
	Type             string               `json:"type"`
	CounterpartyName *string              `json:"counterpartyName,omitempty"`
	CounterpartyIBAN *string              `json:"counterpartyIban,omitempty"`
	EndToEndID       *string              `json:"endToEndId,omitempty"`
	IsReturn         bool                 `json:"isReturn"`
	ReturnReason     *string              `json:"returnReason,omitempty"`
	Status           string               `json:"status"`
	Suggestion       *reconSuggestionJSON `json:"suggestion,omitempty"`
	Payments         []reconPaymentJSON   `json:"payments"`
}

type reconFeeJSON struct {
	FeeID        string          `json:"feeId"`
	Member       honorMemberJSON `json:"member"`
	Year         int             `json:"year"`
	Description  *string         `json:"description,omitempty"`
	DueDate      string          `json:"dueDate"`
	Amount       float64         `json:"amount"`
	Paid         float64         `json:"paid"`
	PaidDate     *string         `json:"paidDate,omitempty"`
	Open         bool            `json:"open"`
	InCollection bool            `json:"inCollection"`
}

func entryStatus(e store.ReconciliationEntry) string {
	switch {
	case len(e.Payments) > 0:
		return "reconciled"
	case e.DismissedAt != nil:
		return "dismissed"
	case e.Open():
		return "open"
	}
	// An expense that is not a returned debit: nothing to reconcile.
	return "none"
}

func toReconEntryJSON(e store.ReconciliationEntry, s *reconcile.Suggestion) reconEntryJSON {
	out := reconEntryJSON{TransactionID: e.TransactionID, Date: e.Date.Format(dateLayout), Description: e.Description,
		Amount: db.FromCents(e.AmountCents), Type: e.Type, CounterpartyName: e.Bank.CounterpartyName,
		CounterpartyIBAN: e.Bank.CounterpartyIBAN, EndToEndID: e.Bank.EndToEndID, IsReturn: e.Bank.IsReturn,
		ReturnReason: e.Bank.ReturnReason, Status: entryStatus(e), Payments: make([]reconPaymentJSON, 0, len(e.Payments))}
	for _, p := range e.Payments {
		amount := p.AmountCents
		if amount < 0 {
			amount = -amount
		}
		out.Payments = append(out.Payments, reconPaymentJSON{ID: p.ID, FeeID: p.FeeID, Member: memberRefJSON(p.Member),
			Year: p.FeeYear, Description: p.FeeDesc, Amount: db.FromCents(amount)})
	}
	if s != nil && out.Status == "open" {
		js := &reconSuggestionJSON{Confidence: s.Confidence, Reasons: s.Reasons,
			Allocations: make([]reconAllocationJSON, 0, len(s.Allocations)), Unallocated: db.FromCents(s.UnallocatedCents)}
		for _, al := range s.Allocations {
			js.Allocations = append(js.Allocations, reconAllocationJSON{FeeID: al.FeeID, Amount: db.FromCents(al.AmountCents)})
		}
		out.Suggestion = js
	}
	return out
}

// suggest matches an open entry against the fees: payments against the
// amounts still due on open fees, returned debits against paid fees.
func suggest(e store.ReconciliationEntry, fees []store.ReconcileFee, members []reconcile.Member) *reconcile.Suggestion {
	if !e.Open() {
		return nil
	}
	in := reconcile.Entry{AmountCents: e.AmountCents, Text: e.Description, Return: e.Bank.IsReturn}
	if e.Bank.CounterpartyName != nil {
		in.Name = *e.Bank.CounterpartyName
	}
	if e.Bank.CounterpartyIBAN != nil {
		in.IBAN = *e.Bank.CounterpartyIBAN
	}
	if e.Bank.EndToEndID != nil {
		in.EndToEndID = *e.Bank.EndToEndID
	}
	candidates := make([]reconcile.Fee, 0, len(fees))
	for _, f := range fees {
		amount := f.AmountCents - f.PaidCents
		if e.Bank.IsReturn {
			amount = min(f.PaidCents, f.AmountCents)
		} else if !f.Open() {
			continue
		}
		if amount > 0 {
			candidates = append(candidates, reconcile.Fee{ID: f.ID, MemberID: f.Member.ID, AmountCents: amount, DueDate: f.DueDate})
		}
	}
	return reconcile.Match(in, members, candidates)
}

// reconciliationSuggestions returns the suggestions for all open entries,
// keyed by transaction id.
func reconciliationSuggestions(ctx context.Context, st *store.Store) (map[string]*reconcile.Suggestion, error) {
	entries, fees, members, err := reconciliationData(ctx, st)
	if err != nil {
		return nil, err
	}
	out := map[string]*reconcile.Suggestion{}
	for _, e := range entries {
		out[e.TransactionID] = suggest(e, fees, members)
	}
	return out, nil
}

func reconciliationData(ctx context.Context, st *store.Store) ([]store.ReconciliationEntry, []store.ReconcileFee, []reconcile.Member, error) {
	entries, err := st.ReconciliationEntries(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	fees, err := st.ReconcileFees(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	members, err := st.ReconcileMembers(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return entries, fees, members, nil
}

func writeReconcileError(w http.ResponseWriter, err error, failure string) {
	var invalid *store.AllocationError
	code := ""
	msg := ""
	switch {
	case errors.As(err, &invalid):
		code, msg = "FEES_NOT_ASSIGNABLE", "some fees cannot take the assigned amount: "+strings.Join(invalid.FeeIDs, ", ")
	case errors.Is(err, store.ErrAllocationExceeds):
		code, msg = "AMOUNT_EXCEEDED", "the assigned amounts exceed the amount of the transaction"
	case errors.Is(err, store.ErrNoAllocations):
		code, msg = "NO_FEES", "no fees assigned"
	case errors.Is(err, store.ErrAlreadyReconciled):
		code, msg = "ALREADY_RECONCILED", "the transaction has already been assigned"
	case errors.Is(err, store.ErrNotReconcilable):
		code, msg = "NOT_RECONCILABLE", "only incoming payments and returned direct debits can be assigned"
	default:
		writeStoreError(w, err, "transaction not found", failure)
		return
	}
	writeJSON(w, http.StatusUnprocessableEntity, openapi.ErrorResponse{Code: &code, Error: msg})
}

// GetReconciliation: GET /api/v2/reconciliation lists the imported
// transactions, with a suggestion for each open one, and the fees that can
// be assigned.
func (a *API) GetReconciliation(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeBankAccess(w, r, false, "the payment reconciliation") {
		return
	}
	entries, fees, members, err := reconciliationData(r.Context(), st)
	if err != nil {
		writeStoreError(w, err, "", "failed to load the reconciliation")
		return
	}
	out := struct {
		Entries []reconEntryJSON `json:"entries"`
		Fees    []reconFeeJSON   `json:"fees"`
	}{Entries: make([]reconEntryJSON, 0, len(entries)), Fees: make([]reconFeeJSON, 0, len(fees))}
	for _, e := range entries {
		out.Entries = append(out.Entries, toReconEntryJSON(e, suggest(e, fees, members)))
	}
	for _, f := range fees {
		out.Fees = append(out.Fees, reconFeeJSON{FeeID: f.ID, Member: memberRefJSON(f.Member), Year: f.Year,
			Description: f.Description, DueDate: f.DueDate.Format(dateLayout), Amount: db.FromCents(f.AmountCents),
			Paid: db.FromCents(f.PaidCents), PaidDate: formatDate(f.PaidDate), Open: f.Open(), InCollection: f.InCollection})
	}
	writeJSON(w, http.StatusOK, out)
}

// ConfirmReconciliation: POST /api/v2/reconciliation/{id}/confirm
// {"allocations":[{"feeId","amount"}]} assigns the transaction to fees.
func (a *API) ConfirmReconciliation(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeBankAccess(w, r, true, "the payment reconciliation") {
		return
	}
	var req struct {
		Allocations []reconAllocationJSON `json:"allocations"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	allocations := make([]store.Allocation, 0, len(req.Allocations))
	for _, al := range req.Allocations {
		allocations = append(allocations, store.Allocation{FeeID: al.FeeID, AmountCents: db.ToCents(al.Amount)})
	}
	id := chi.URLParam(r, "id")
	entry, err := st.ConfirmReconciliation(r.Context(), id, allocations)
	if err != nil {
		writeReconcileError(w, err, "failed to confirm the assignment")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Reconciliation", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toReconEntryJSON(entry, nil))
}

// ConfirmCertainReconciliations: POST /api/v2/reconciliation/confirm-certain
// confirms the suggestions of all open entries rated certain, oldest entry
// first, re-matching after each one. Responds with the number confirmed.
func (a *API) ConfirmCertainReconciliations(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeBankAccess(w, r, true, "the payment reconciliation") {
		return
	}
	confirmed := 0
	done := map[string]bool{}
	for {
		entries, fees, members, err := reconciliationData(r.Context(), st)
		if err != nil {
			writeStoreError(w, err, "", "failed to load the reconciliation")
			return
		}
		var next *store.ReconciliationEntry
		var allocations []store.Allocation
		// Entries are newest first; take the oldest certain one.
		for i := len(entries) - 1; i >= 0; i-- {
			s := suggest(entries[i], fees, members)
			if done[entries[i].TransactionID] || s == nil || s.Confidence != reconcile.Certain {
				continue
			}
			next = &entries[i]
			for _, al := range s.Allocations {
				allocations = append(allocations, store.Allocation{FeeID: al.FeeID, AmountCents: al.AmountCents})
			}
			break
		}
		if next == nil {
			break
		}
		done[next.TransactionID] = true
		if _, err := st.ConfirmReconciliation(r.Context(), next.TransactionID, allocations); err != nil {
			writeReconcileError(w, err, "failed to confirm the assignment")
			return
		}
		st.Audit(r.Context(), "UPDATE", "Reconciliation", next.TransactionID, currentUserID(r))
		confirmed++
	}
	writeJSON(w, http.StatusOK, map[string]int{"confirmed": confirmed})
}

// DismissReconciliation: POST /api/v2/reconciliation/{id}/dismiss marks a
// transaction as reviewed without a fee (e.g. a donation).
func (a *API) DismissReconciliation(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeBankAccess(w, r, true, "the payment reconciliation") {
		return
	}
	id := chi.URLParam(r, "id")
	entry, err := st.DismissReconciliation(r.Context(), id)
	if err != nil {
		writeReconcileError(w, err, "failed to dismiss the transaction")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Reconciliation", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toReconEntryJSON(entry, nil))
}

// ReopenReconciliation: POST /api/v2/reconciliation/{id}/reopen undoes the
// assignment or dismissal; the fees are settled again without it.
func (a *API) ReopenReconciliation(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeBankAccess(w, r, true, "the payment reconciliation") {
		return
	}
	id := chi.URLParam(r, "id")
	entry, err := st.ReopenReconciliation(r.Context(), id)
	if err != nil {
		writeReconcileError(w, err, "failed to reopen the transaction")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Reconciliation", id, currentUserID(r))
	_, fees, members, err := reconciliationData(r.Context(), st)
	if err != nil {
		writeJSON(w, http.StatusOK, toReconEntryJSON(entry, nil))
		return
	}
	writeJSON(w, http.StatusOK, toReconEntryJSON(entry, suggest(entry, fees, members)))
}
