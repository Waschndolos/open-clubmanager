package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/sepa"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

type sepaCreditorJSON struct {
	Name               string  `json:"name"`
	CreditorID         string  `json:"creditorId"`
	IBAN               string  `json:"iban"`
	BIC                *string `json:"bic,omitempty"`
	RemittanceTemplate *string `json:"remittanceTemplate,omitempty"`
}

type sepaCandidateJSON struct {
	FeeID        string          `json:"feeId"`
	Member       honorMemberJSON `json:"member"`
	Amount       float64         `json:"amount"`
	Year         int             `json:"year"`
	Description  *string         `json:"description,omitempty"`
	DueDate      string          `json:"dueDate"`
	DebtorName   string          `json:"debtorName"`
	IBAN         *string         `json:"iban,omitempty"`
	MandateDate  *string         `json:"mandateDate,omitempty"`
	MandateRef   *string         `json:"mandateReference,omitempty"`
	SequenceType string          `json:"sequenceType"`
	Remittance   string          `json:"remittance"`
	Issues       []string        `json:"issues"`
}

type sepaRunItemJSON struct {
	FeeID        string          `json:"feeId"`
	Member       honorMemberJSON `json:"member"`
	Amount       float64         `json:"amount"`
	SequenceType string          `json:"sequenceType"`
	MandateRef   string          `json:"mandateReference"`
	DebtorName   string          `json:"debtorName"`
	IBAN         string          `json:"iban"`
	Remittance   string          `json:"remittance"`
}

type sepaRunJSON struct {
	ID             string            `json:"id"`
	CollectionDate string            `json:"collectionDate"`
	PaidDate       *string           `json:"paidDate,omitempty"`
	CreatedAt      string            `json:"createdAt"`
	Total          float64           `json:"total"`
	Count          int               `json:"count"`
	Items          []sepaRunItemJSON `json:"items"`
}

func toSEPARunJSON(r store.SEPARun) sepaRunJSON {
	out := sepaRunJSON{ID: r.ID, CollectionDate: r.CollectionDate.Format(dateLayout), PaidDate: formatDate(r.PaidDate),
		CreatedAt: db.FormatTime(r.CreatedAt), Total: db.FromCents(r.TotalCents()), Count: len(r.Items),
		Items: make([]sepaRunItemJSON, 0, len(r.Items))}
	for _, it := range r.Items {
		out.Items = append(out.Items, sepaRunItemJSON{FeeID: it.FeeID, Member: memberRefJSON(it.Member),
			Amount: db.FromCents(it.AmountCents), SequenceType: it.SequenceType, MandateRef: it.MandateRef,
			DebtorName: it.DebtorName, IBAN: it.IBAN, Remittance: it.Remittance})
	}
	return out
}

// sepaAccess checks that the caller may use direct debits: only roles that
// manage finances (the treasurer), on a device that may read (or, with
// write, change) both the bank data and the finances.
func (a *API) sepaAccess(w http.ResponseWriter, r *http.Request, write bool) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok && slices.Contains(financeEditors, claims.AppRole)
	for _, s := range []db.Scope{db.ScopeMembersBank, db.ScopeFinance} {
		if !allowed {
			break
		}
		if write {
			allowed = a.collab.CanWrite(r.Context(), s)
		} else {
			allowed = a.collab.CanRead(r.Context(), s)
		}
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access direct debits"})
	}
	return allowed
}

func writeSEPAError(w http.ResponseWriter, err error, notFound, failure string) {
	var invalid *store.SEPAInvalidError
	code := ""
	msg := ""
	switch {
	case errors.As(err, &invalid):
		code, msg = "SEPA_FEES_INVALID", "some fees cannot be collected: "+strings.Join(invalid.FeeIDs, ", ")
	case errors.Is(err, store.ErrSEPANotConfigured):
		code, msg = "SEPA_NOT_CONFIGURED", "the creditor data has not been set up"
	case errors.Is(err, store.ErrSEPARunPaid):
		code, msg = "SEPA_RUN_PAID", "the run has already been confirmed"
	case errors.Is(err, store.ErrSEPANoFees):
		code, msg = "SEPA_NO_FEES", "no fees selected"
	default:
		writeStoreError(w, err, notFound, failure)
		return
	}
	writeJSON(w, http.StatusUnprocessableEntity, openapi.ErrorResponse{Code: &code, Error: msg})
}

func optionalTrimmed(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

// GetSEPACreditor: GET /api/v2/sepa/creditor. Responds with 204 before the
// creditor data has been set up.
func (a *API) GetSEPACreditor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, false) {
		return
	}
	c, err := st.GetSEPACreditor(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to load creditor")
		return
	}
	if c == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, sepaCreditorJSON{Name: c.Name, CreditorID: c.CreditorID, IBAN: c.IBAN, BIC: c.BIC,
		RemittanceTemplate: c.RemittanceTemplate})
}

// SetSEPACreditor: PUT /api/v2/sepa/creditor.
func (a *API) SetSEPACreditor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, true) {
		return
	}
	var req sepaCreditorJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	c := store.SEPACreditor{Name: strings.TrimSpace(req.Name), CreditorID: sepa.NormalizeCreditorID(req.CreditorID),
		IBAN: sepa.NormalizeIBAN(req.IBAN), RemittanceTemplate: optionalTrimmed(req.RemittanceTemplate)}
	if bic := optionalTrimmed(req.BIC); bic != nil {
		v := sepa.NormalizeIBAN(*bic)
		c.BIC = &v
	}
	switch {
	case c.Name == "" || len(c.Name) > 70:
		badRequest(w, "name must have 1 to 70 characters")
	case !sepa.ValidCreditorID(c.CreditorID):
		badRequest(w, "creditorId is not a valid SEPA creditor identifier")
	case !sepa.ValidIBAN(c.IBAN):
		badRequest(w, "iban is not a valid IBAN")
	case c.BIC != nil && !sepa.ValidBIC(*c.BIC):
		badRequest(w, "bic is not a valid BIC")
	case c.RemittanceTemplate != nil && len(*c.RemittanceTemplate) > 200:
		badRequest(w, "remittanceTemplate may have at most 200 characters")
	default:
		if err := st.SetSEPACreditor(r.Context(), c); err != nil {
			writeStoreError(w, err, "", "failed to save creditor")
			return
		}
		st.Audit(r.Context(), "UPDATE", "SepaCreditor", "club", currentUserID(r))
		writeJSON(w, http.StatusOK, sepaCreditorJSON{Name: c.Name, CreditorID: c.CreditorID, IBAN: c.IBAN, BIC: c.BIC,
			RemittanceTemplate: c.RemittanceTemplate})
	}
}

type mandateJSON struct {
	MandateReference *string `json:"mandateReference"`
}

// GetSEPAMandate: GET /api/v2/members/{id}/sepa-mandate.
func (a *API) GetSEPAMandate(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "bank details", db.ScopeMembersBank) {
		return
	}
	ref, err := st.GetMandateReference(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to load mandate")
		return
	}
	writeJSON(w, http.StatusOK, mandateJSON{MandateReference: ref})
}

// SetSEPAMandate: PUT /api/v2/members/{id}/sepa-mandate. An empty reference
// removes it.
func (a *API) SetSEPAMandate(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "bank details", db.ScopeMembersBank) {
		return
	}
	var req mandateJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	ref := optionalTrimmed(req.MandateReference)
	if ref != nil && !sepa.ValidMandateReference(*ref) {
		code := "INVALID_MANDATE_REFERENCE"
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Code: &code,
			Error: "mandateReference may have up to 35 letters, digits and + ? / - : ( ) . , ' without spaces"})
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.SetMandateReference(r.Context(), id, ref); err != nil {
		if errors.Is(err, store.ErrConflict) {
			code := "MANDATE_REFERENCE_TAKEN"
			writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code, Error: "another member already has this mandate reference"})
			return
		}
		writeStoreError(w, err, "member not found", "failed to save mandate")
		return
	}
	st.Audit(r.Context(), "UPDATE", "SepaMandate", id, currentUserID(r))
	writeJSON(w, http.StatusOK, mandateJSON{MandateReference: ref})
}

// ListSEPACandidates: GET /api/v2/sepa/candidates?template=… lists the open
// fees with their issues and the remittance text the template yields.
func (a *API) ListSEPACandidates(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, false) {
		return
	}
	template := r.URL.Query().Get("template")
	if !r.URL.Query().Has("template") {
		c, err := st.GetSEPACreditor(r.Context())
		if err != nil {
			writeStoreError(w, err, "", "failed to load creditor")
			return
		}
		if c != nil && c.RemittanceTemplate != nil {
			template = *c.RemittanceTemplate
		}
	}
	list, err := st.SEPACandidates(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list open fees")
		return
	}
	out := make([]sepaCandidateJSON, 0, len(list))
	for _, c := range list {
		j := sepaCandidateJSON{FeeID: c.FeeID, Member: memberRefJSON(c.Member), Amount: db.FromCents(c.AmountCents),
			Year: c.Year, Description: c.Description, DueDate: c.DueDate.Format(dateLayout), DebtorName: c.DebtorName(),
			MandateDate: formatDate(c.MandateDate), MandateRef: c.MandateRef, SequenceType: c.SequenceType,
			Remittance: c.Remittance(template), Issues: c.Issues}
		if c.IBAN != nil {
			iban := sepa.NormalizeIBAN(*c.IBAN)
			j.IBAN = &iban
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

// ListSEPARuns: GET /api/v2/sepa/runs.
func (a *API) ListSEPARuns(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, false) {
		return
	}
	runs, err := st.ListSEPARuns(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list runs")
		return
	}
	out := make([]sepaRunJSON, 0, len(runs))
	for _, run := range runs {
		out = append(out, toSEPARunJSON(run))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateSEPARun: POST /api/v2/sepa/runs. The fees are marked as being
// collected until the run is confirmed or cancelled.
func (a *API) CreateSEPARun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, true) {
		return
	}
	var req struct {
		CollectionDate     string   `json:"collectionDate"`
		RemittanceTemplate *string  `json:"remittanceTemplate"`
		FeeIDs             []string `json:"feeIds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	date, err := time.Parse(dateLayout, req.CollectionDate)
	if err != nil {
		badRequest(w, "collectionDate must be a date (YYYY-MM-DD)")
		return
	}
	today := time.Now()
	if !date.After(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)) {
		badRequest(w, "collectionDate must be in the future")
		return
	}
	if req.RemittanceTemplate != nil && len(*req.RemittanceTemplate) > 200 {
		badRequest(w, "remittanceTemplate may have at most 200 characters")
		return
	}
	run, err := st.CreateSEPARun(r.Context(), store.SEPARunInput{CollectionDate: date,
		RemittanceTemplate: req.RemittanceTemplate, FeeIDs: req.FeeIDs})
	if err != nil {
		writeSEPAError(w, err, "", "failed to create run")
		return
	}
	st.Audit(r.Context(), "CREATE", "SepaRun", run.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toSEPARunJSON(run))
}

// DownloadSEPARun: GET /api/v2/sepa/runs/{id}/xml returns the pain.008 file.
func (a *API) DownloadSEPARun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, false) {
		return
	}
	run, err := st.GetSEPARun(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err, "run not found", "failed to load run")
		return
	}
	out, err := run.Initiation().XML()
	if err != nil {
		writeStoreError(w, err, "", "failed to write direct debit file")
		return
	}
	name := "lastschrift-" + run.CollectionDate.Format(dateLayout) + "-" + run.ID[len(run.ID)-8:]
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.xml"`)
	_, _ = w.Write(out)
}

// ConfirmSEPARun: POST /api/v2/sepa/runs/{id}/confirm {"paidDate"} marks the
// run's fees as paid.
func (a *API) ConfirmSEPARun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, true) {
		return
	}
	var req struct {
		PaidDate string `json:"paidDate"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	date, err := time.Parse(dateLayout, req.PaidDate)
	if err != nil {
		badRequest(w, "paidDate must be a date (YYYY-MM-DD)")
		return
	}
	id := chi.URLParam(r, "id")
	run, err := st.ConfirmSEPARun(r.Context(), id, date)
	if err != nil {
		writeSEPAError(w, err, "run not found", "failed to confirm run")
		return
	}
	st.Audit(r.Context(), "UPDATE", "SepaRun", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toSEPARunJSON(run))
}

// CancelSEPARun: DELETE /api/v2/sepa/runs/{id}; the fees are open again.
func (a *API) CancelSEPARun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.sepaAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.CancelSEPARun(r.Context(), id); err != nil {
		writeSEPAError(w, err, "run not found", "failed to cancel run")
		return
	}
	st.Audit(r.Context(), "DELETE", "SepaRun", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}
