package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/donation"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

type donationSettingsJSON struct {
	IssuerName       string  `json:"issuerName"`
	IssuerAddress    string  `json:"issuerAddress"`
	TaxOffice        string  `json:"taxOffice"`
	TaxNumber        string  `json:"taxNumber"`
	ExemptionKind    string  `json:"exemptionKind"`
	ExemptionDate    string  `json:"exemptionDate"`
	AssessmentPeriod *string `json:"assessmentPeriod,omitempty"`
	Purposes         string  `json:"purposes"`
	FeesDeductible   bool    `json:"feesDeductible"`
	Signatory        *string `json:"signatory,omitempty"`
	Place            *string `json:"place,omitempty"`
	// Read only: whether receipts issued today would no longer be
	// recognised because the exemption is too old, and until when they are.
	ExemptionExpired bool   `json:"exemptionExpired"`
	ValidUntil       string `json:"validUntil"`
}

func toDonationSettingsJSON(i donation.Issuer) donationSettingsJSON {
	opt := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	return donationSettingsJSON{IssuerName: i.Name, IssuerAddress: i.Address, TaxOffice: i.TaxOffice, TaxNumber: i.TaxNumber,
		ExemptionKind: i.ExemptionKind, ExemptionDate: i.ExemptionDate.Format(dateLayout),
		AssessmentPeriod: opt(i.AssessmentPeriod), Purposes: i.Purposes, FeesDeductible: i.FeesDeductible,
		Signatory: opt(i.Signatory), Place: opt(i.Place), ExemptionExpired: i.ExemptionExpired(time.Now()),
		ValidUntil: i.ExemptionDate.AddDate(i.ValidYears(), 0, 0).Format(dateLayout)}
}

type donorJSON struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Street     *string `json:"street,omitempty"`
	PostalCode *string `json:"postalCode,omitempty"`
	City       *string `json:"city,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

func toDonorJSON(d store.Donor) donorJSON {
	return donorJSON{ID: d.ID, Name: d.Name, Street: d.Street, PostalCode: d.PostalCode, City: d.City, Notes: d.Notes}
}

type donorRefJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type donationJSON struct {
	TransactionID string           `json:"transactionId"`
	Date          string           `json:"date"`
	Description   string           `json:"description"`
	Amount        float64          `json:"amount"`
	Category      *string          `json:"category,omitempty"`
	Member        *honorMemberJSON `json:"member,omitempty"`
	Donor         *donorRefJSON    `json:"donor,omitempty"`
	ExpenseWaiver bool             `json:"expenseWaiver"`
	ReceiptNumber *string          `json:"receiptNumber,omitempty"`
}

func toDonationJSON(e store.DonationEntry) donationJSON {
	out := donationJSON{TransactionID: e.TransactionID, Date: e.Date.Format(dateLayout), Description: e.Description,
		Amount: db.FromCents(e.AmountCents), Category: e.Category, ExpenseWaiver: e.ExpenseWaiver, ReceiptNumber: e.ReceiptNumber}
	if e.Member != nil {
		m := memberRefJSON(*e.Member)
		out.Member = &m
	}
	if e.Donor != nil {
		out.Donor = &donorRefJSON{ID: e.Donor.ID, Name: e.Donor.Name}
	}
	return out
}

type candidateItemJSON struct {
	Ref           string  `json:"ref"`
	Kind          string  `json:"kind"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	ExpenseWaiver bool    `json:"expenseWaiver"`
	Description   string  `json:"description"`
	ReceiptID     *string `json:"receiptId,omitempty"`
	ReceiptNumber *string `json:"receiptNumber,omitempty"`
}

type candidateJSON struct {
	Key     string              `json:"key"`
	Member  *honorMemberJSON    `json:"member,omitempty"`
	Donor   *donorRefJSON       `json:"donor,omitempty"`
	Name    string              `json:"name"`
	Address string              `json:"address"`
	Items   []candidateItemJSON `json:"items"`
}

func toCandidateJSON(c store.DonationCandidate) candidateJSON {
	out := candidateJSON{Key: c.Key(), Name: c.Name, Address: c.Address, Items: make([]candidateItemJSON, 0, len(c.Items))}
	if c.Member != nil {
		m := memberRefJSON(*c.Member)
		out.Member = &m
	}
	if c.Donor != nil {
		out.Donor = &donorRefJSON{ID: c.Donor.ID, Name: c.Donor.Name}
	}
	for _, it := range c.Items {
		out.Items = append(out.Items, candidateItemJSON{Ref: it.Ref(), Kind: it.Kind, Date: it.Date.Format(dateLayout),
			Amount: db.FromCents(it.AmountCents), ExpenseWaiver: it.ExpenseWaiver, Description: it.Description,
			ReceiptID: it.ReceiptID, ReceiptNumber: it.ReceiptNumber})
	}
	return out
}

type receiptItemJSON struct {
	Date          string  `json:"date"`
	Kind          string  `json:"kind"`
	Amount        float64 `json:"amount"`
	ExpenseWaiver bool    `json:"expenseWaiver"`
	TransactionID *string `json:"transactionId,omitempty"`
	FeeID         *string `json:"feeId,omitempty"`
}

type receiptJSON struct {
	ID               string            `json:"id"`
	Number           string            `json:"number"`
	Year             int               `json:"year"`
	Kind             string            `json:"kind"`
	MemberID         *string           `json:"memberId,omitempty"`
	DonorID          *string           `json:"donorId,omitempty"`
	RecipientName    string            `json:"recipientName"`
	RecipientAddress string            `json:"recipientAddress"`
	PeriodStart      string            `json:"periodStart"`
	PeriodEnd        string            `json:"periodEnd"`
	Total            float64           `json:"total"`
	IssueDate        string            `json:"issueDate"`
	DuplicateOf      *string           `json:"duplicateOf,omitempty"`
	CreatedBy        *string           `json:"createdBy,omitempty"`
	CreatedAt        string            `json:"createdAt"`
	VoidedAt         *string           `json:"voidedAt,omitempty"`
	VoidReason       *string           `json:"voidReason,omitempty"`
	Items            []receiptItemJSON `json:"items"`
}

func toReceiptJSON(r store.DonationReceipt) receiptJSON {
	out := receiptJSON{ID: r.ID, Number: r.Number, Year: r.Year, Kind: r.Kind, MemberID: r.MemberID, DonorID: r.DonorID,
		RecipientName: r.RecipientName, RecipientAddress: r.RecipientAddress, PeriodStart: r.PeriodStart.Format(dateLayout),
		PeriodEnd: r.PeriodEnd.Format(dateLayout), Total: db.FromCents(r.TotalCents), IssueDate: r.IssueDate.Format(dateLayout),
		DuplicateOf: r.DuplicateOf, CreatedBy: r.CreatedBy, CreatedAt: db.FormatTime(r.CreatedAt),
		VoidReason: r.VoidReason, Items: make([]receiptItemJSON, 0, len(r.Items))}
	if r.VoidedAt != nil {
		v := db.FormatTime(*r.VoidedAt)
		out.VoidedAt = &v
	}
	for _, it := range r.Items {
		out.Items = append(out.Items, receiptItemJSON{Date: it.Date.Format(dateLayout), Kind: it.Kind,
			Amount: db.FromCents(it.AmountCents), ExpenseWaiver: it.ExpenseWaiver, TransactionID: it.TransactionID, FeeID: it.FeeID})
	}
	return out
}

// donationAccess checks that the caller may use donation receipts: only
// roles that manage finances (the treasurer), on a device that may read (or,
// with write, change) the finances and read the members' addresses.
func (a *API) donationAccess(w http.ResponseWriter, r *http.Request, write bool) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok && slices.Contains(financeEditors, claims.AppRole) && a.collab.CanRead(r.Context(), db.ScopeMembersContact)
	if allowed && write {
		allowed = a.collab.CanWrite(r.Context(), db.ScopeFinance)
	} else if allowed {
		allowed = a.collab.CanRead(r.Context(), db.ScopeFinance)
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access donation receipts"})
	}
	return allowed
}

func writeDonationError(w http.ResponseWriter, err error, notFound, failure string) {
	var items *store.ReceiptItemsError
	status, code, msg := http.StatusUnprocessableEntity, "", ""
	switch {
	case errors.As(err, &items):
		if len(items.Invalid) > 0 {
			code, msg = "RECEIPT_ITEMS_INVALID", "some items are no donation or paid fee of the donor in one year: "+strings.Join(items.Invalid, ", ")
		} else {
			status, code, msg = http.StatusConflict, "RECEIPT_ITEMS_CERTIFIED", "some items are already certified: "+strings.Join(items.Certified, ", ")
		}
	case errors.Is(err, store.ErrDonationsNotConfigured):
		code, msg = "DONATIONS_NOT_CONFIGURED", "the club's exemption data has not been set up"
	case errors.Is(err, store.ErrDonationCertified):
		status, code, msg = http.StatusConflict, "DONATION_CERTIFIED", "the donation is certified by a receipt; void the receipt first"
	case errors.Is(err, store.ErrNotIncome):
		code, msg = "NOT_INCOME", "only an income can be a donation"
	case errors.Is(err, store.ErrDonorInUse):
		status, code, msg = http.StatusConflict, "DONOR_IN_USE", "the donor has donations"
	case errors.Is(err, store.ErrReceiptVoided):
		code, msg = "RECEIPT_VOIDED", "the receipt is voided"
	case errors.Is(err, store.ErrMissingAddress):
		code, msg = "MISSING_ADDRESS", "the donor has no complete postal address"
	default:
		writeStoreError(w, err, notFound, failure)
		return
	}
	writeJSON(w, status, openapi.ErrorResponse{Code: &code, Error: msg})
}

// GetDonationSettings: GET /api/v2/donations/settings. Responds with 204
// before the exemption data has been set up.
func (a *API) GetDonationSettings(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, false) {
		return
	}
	i, err := st.GetDonationSettings(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to load donation settings")
		return
	}
	if i == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, toDonationSettingsJSON(*i))
}

// SetDonationSettings: PUT /api/v2/donations/settings.
func (a *API) SetDonationSettings(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	var req donationSettingsJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	trim := func(s *string) string {
		if s == nil {
			return ""
		}
		return strings.TrimSpace(*s)
	}
	i := donation.Issuer{Name: strings.TrimSpace(req.IssuerName), Address: strings.TrimSpace(req.IssuerAddress),
		TaxOffice: strings.TrimSpace(req.TaxOffice), TaxNumber: strings.TrimSpace(req.TaxNumber), ExemptionKind: req.ExemptionKind,
		AssessmentPeriod: trim(req.AssessmentPeriod), Purposes: strings.TrimSpace(req.Purposes), FeesDeductible: req.FeesDeductible,
		Signatory: trim(req.Signatory), Place: trim(req.Place)}
	date, err := time.Parse(dateLayout, req.ExemptionDate)
	switch {
	case i.Name == "" || len(i.Name) > 200:
		badRequest(w, "issuerName must have 1 to 200 characters")
	case i.Address == "" || len(i.Address) > 500:
		badRequest(w, "issuerAddress must have 1 to 500 characters")
	case i.TaxOffice == "" || i.TaxNumber == "":
		badRequest(w, "taxOffice and taxNumber are required")
	case i.ExemptionKind != donation.ExemptionNotice && i.ExemptionKind != donation.ExemptionAssessment:
		badRequest(w, "exemptionKind must be notice or assessment")
	case err != nil:
		badRequest(w, "exemptionDate must be a date (YYYY-MM-DD)")
	case date.After(time.Now()):
		badRequest(w, "exemptionDate must not be in the future")
	case i.Purposes == "" || len(i.Purposes) > 1000:
		badRequest(w, "purposes must have 1 to 1000 characters")
	default:
		i.ExemptionDate = date
		if err := st.SetDonationSettings(r.Context(), i); err != nil {
			writeStoreError(w, err, "", "failed to save donation settings")
			return
		}
		st.Audit(r.Context(), "UPDATE", "DonationSettings", "club", currentUserID(r))
		writeJSON(w, http.StatusOK, toDonationSettingsJSON(i))
	}
}

// ListDonors: GET /api/v2/donations/donors.
func (a *API) ListDonors(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, false) {
		return
	}
	list, err := st.ListDonors(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list donors")
		return
	}
	out := make([]donorJSON, 0, len(list))
	for _, d := range list {
		out = append(out, toDonorJSON(d))
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeDonor(w http.ResponseWriter, r *http.Request) (store.DonorInput, bool) {
	var req donorJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return store.DonorInput{}, false
	}
	in := store.DonorInput{Name: strings.TrimSpace(req.Name), Street: optionalTrimmed(req.Street),
		PostalCode: optionalTrimmed(req.PostalCode), City: optionalTrimmed(req.City), Notes: optionalTrimmed(req.Notes)}
	if in.Name == "" || len(in.Name) > 200 {
		badRequest(w, "name must have 1 to 200 characters")
		return in, false
	}
	return in, true
}

// CreateDonor: POST /api/v2/donations/donors.
func (a *API) CreateDonor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	in, ok := decodeDonor(w, r)
	if !ok {
		return
	}
	d, err := st.CreateDonor(r.Context(), in)
	if err != nil {
		writeStoreError(w, err, "", "failed to create donor")
		return
	}
	st.Audit(r.Context(), "CREATE", "Donor", d.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toDonorJSON(d))
}

// UpdateDonor: PUT /api/v2/donations/donors/{id}.
func (a *API) UpdateDonor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	in, ok := decodeDonor(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	d, err := st.UpdateDonor(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "donor not found", "failed to update donor")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Donor", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toDonorJSON(d))
}

// DeleteDonor: DELETE /api/v2/donations/donors/{id}.
func (a *API) DeleteDonor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteDonor(r.Context(), id); err != nil {
		writeDonationError(w, err, "donor not found", "failed to delete donor")
		return
	}
	st.Audit(r.Context(), "DELETE", "Donor", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ListDonations: GET /api/v2/donations lists the incomes that are assigned
// to a donor or have a donation category.
func (a *API) ListDonations(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, false) {
		return
	}
	list, err := st.ListDonations(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list donations")
		return
	}
	out := make([]donationJSON, 0, len(list))
	for _, e := range list {
		out = append(out, toDonationJSON(e))
	}
	writeJSON(w, http.StatusOK, out)
}

// SetDonation: PUT /api/v2/donations/{id} {"memberId"|"donorId",
// "expenseWaiver"} marks the transaction as a donation of that member or donor.
func (a *API) SetDonation(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	var req struct {
		MemberID      *string `json:"memberId"`
		DonorID       *string `json:"donorId"`
		ExpenseWaiver bool    `json:"expenseWaiver"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	member, donor := optionalTrimmed(req.MemberID), optionalTrimmed(req.DonorID)
	if (member == nil) == (donor == nil) {
		badRequest(w, "exactly one of memberId and donorId is required")
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.SetDonation(r.Context(), id, member, donor, req.ExpenseWaiver); err != nil {
		writeDonationError(w, err, "transaction not found", "failed to assign donation")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Donation", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// RemoveDonation: DELETE /api/v2/donations/{id} removes the donor.
func (a *API) RemoveDonation(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.RemoveDonation(r.Context(), id); err != nil {
		writeDonationError(w, err, "donation not found", "failed to remove donation")
		return
	}
	st.Audit(r.Context(), "DELETE", "Donation", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func queryYear(r *http.Request) (int, bool) {
	v := r.URL.Query().Get("year")
	if v == "" {
		return time.Now().Year(), true
	}
	y, err := strconv.Atoi(v)
	return y, err == nil && y >= 1900 && y <= 9999
}

// ListDonationCandidates: GET /api/v2/donations/candidates?year= lists the
// members and donors with donations (and deductible paid fees) in the year.
func (a *API) ListDonationCandidates(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, false) {
		return
	}
	year, ok := queryYear(r)
	if !ok {
		badRequest(w, "year must be a year")
		return
	}
	issuer, err := st.GetDonationSettings(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to load donation settings")
		return
	}
	list, err := st.DonationCandidates(r.Context(), year, issuer != nil && issuer.FeesDeductible)
	if err != nil {
		writeStoreError(w, err, "", "failed to list donors")
		return
	}
	out := make([]candidateJSON, 0, len(list))
	for _, c := range list {
		out = append(out, toCandidateJSON(c))
	}
	writeJSON(w, http.StatusOK, out)
}

// ListDonationReceipts: GET /api/v2/donations/receipts?year= is the log of
// issued receipts (year 0 or missing: all years).
func (a *API) ListDonationReceipts(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, false) {
		return
	}
	year := 0
	if v := r.URL.Query().Get("year"); v != "" {
		var ok bool
		if year, ok = queryYear(r); !ok {
			badRequest(w, "year must be a year")
			return
		}
	}
	list, err := st.ListReceipts(r.Context(), year)
	if err != nil {
		writeStoreError(w, err, "", "failed to list receipts")
		return
	}
	out := make([]receiptJSON, 0, len(list))
	for _, rc := range list {
		out = append(out, toReceiptJSON(rc))
	}
	writeJSON(w, http.StatusOK, out)
}

func parseIssueDate(w http.ResponseWriter, v string) (time.Time, bool) {
	if v == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), true
	}
	d, err := time.Parse(dateLayout, v)
	if err != nil {
		badRequest(w, "issueDate must be a date (YYYY-MM-DD)")
		return d, false
	}
	return d, true
}

// CreateDonationReceipt: POST /api/v2/donations/receipts {"memberId"|"donorId",
// "items": [transaction or fee ids], "issueDate"}. One item gives a single
// receipt, several a collective one; the items of an issued receipt give a
// duplicate of it.
func (a *API) CreateDonationReceipt(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	var req struct {
		MemberID  *string  `json:"memberId"`
		DonorID   *string  `json:"donorId"`
		Items     []string `json:"items"`
		IssueDate string   `json:"issueDate"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<18)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	member, donor := optionalTrimmed(req.MemberID), optionalTrimmed(req.DonorID)
	if (member == nil) == (donor == nil) {
		badRequest(w, "exactly one of memberId and donorId is required")
		return
	}
	if len(req.Items) == 0 {
		badRequest(w, "items must not be empty")
		return
	}
	date, ok := parseIssueDate(w, req.IssueDate)
	if !ok {
		return
	}
	list, err := st.CreateReceipts(r.Context(), []store.ReceiptRequest{{MemberID: member, DonorID: donor, ItemRefs: req.Items}},
		date, currentUserID(r))
	if err != nil {
		writeDonationError(w, err, "", "failed to create receipt")
		return
	}
	st.Audit(r.Context(), "CREATE", "DonationReceipt", list[0].ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toReceiptJSON(list[0]))
}

// CreateAnnualDonationReceipts: POST /api/v2/donations/receipts/annual
// {"year", "issueDate"} issues receipts for every member and donor with
// items of the year not certified yet: a collective receipt, or a single one
// for one item. Donors without an address are skipped and reported.
func (a *API) CreateAnnualDonationReceipts(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	var req struct {
		Year      int    `json:"year"`
		IssueDate string `json:"issueDate"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if req.Year < 1900 || req.Year > time.Now().Year() {
		badRequest(w, "year must not be in the future")
		return
	}
	date, ok := parseIssueDate(w, req.IssueDate)
	if !ok {
		return
	}
	issuer, err := st.GetDonationSettings(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to load donation settings")
		return
	}
	if issuer == nil {
		writeDonationError(w, store.ErrDonationsNotConfigured, "", "")
		return
	}
	candidates, err := st.DonationCandidates(r.Context(), req.Year, issuer.FeesDeductible)
	if err != nil {
		writeStoreError(w, err, "", "failed to list donors")
		return
	}
	var reqs []store.ReceiptRequest
	skipped := []candidateJSON{}
	for _, c := range candidates {
		open := c.Open()
		if len(open) == 0 {
			continue
		}
		if c.Address == "" {
			skipped = append(skipped, toCandidateJSON(c))
			continue
		}
		rr := store.ReceiptRequest{}
		if c.Member != nil {
			rr.MemberID = &c.Member.ID
		} else {
			rr.DonorID = &c.Donor.ID
		}
		for _, it := range open {
			rr.ItemRefs = append(rr.ItemRefs, it.Ref())
		}
		reqs = append(reqs, rr)
	}
	created := []receiptJSON{}
	if len(reqs) > 0 {
		list, err := st.CreateReceipts(r.Context(), reqs, date, currentUserID(r))
		if err != nil {
			writeDonationError(w, err, "", "failed to create receipts")
			return
		}
		for _, rc := range list {
			st.Audit(r.Context(), "CREATE", "DonationReceipt", rc.ID, currentUserID(r))
			created = append(created, toReceiptJSON(rc))
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"created": created, "skipped": skipped})
}

// DuplicateDonationReceipt: POST /api/v2/donations/receipts/{id}/duplicate
// issues a duplicate, marked as such, with the number of the original.
func (a *API) DuplicateDonationReceipt(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	rc, err := st.DuplicateReceipt(r.Context(), chi.URLParam(r, "id"), currentUserID(r))
	if err != nil {
		writeDonationError(w, err, "receipt not found", "failed to create duplicate")
		return
	}
	st.Audit(r.Context(), "CREATE", "DonationReceipt", rc.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toReceiptJSON(rc))
}

// VoidDonationReceipt: POST /api/v2/donations/receipts/{id}/void {"reason"}.
func (a *API) VoidDonationReceipt(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, true) {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	id := chi.URLParam(r, "id")
	rc, err := st.VoidReceipt(r.Context(), id, strings.TrimSpace(req.Reason))
	if err != nil {
		writeDonationError(w, err, "receipt not found", "failed to void receipt")
		return
	}
	st.Audit(r.Context(), "UPDATE", "DonationReceipt", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toReceiptJSON(rc))
}

// DownloadDonationReceipts: GET /api/v2/donations/receipts/pdf?ids=a,b
// returns the receipts as one PDF, each on its own page.
func (a *API) DownloadDonationReceipts(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.donationAccess(w, r, false) {
		return
	}
	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	if id := chi.URLParam(r, "id"); id != "" {
		ids = []string{id}
	}
	var docs []donation.Receipt
	name := ""
	for _, id := range ids {
		if id = strings.TrimSpace(id); id == "" {
			continue
		}
		rc, err := st.GetReceipt(r.Context(), id)
		if err != nil {
			writeStoreError(w, err, "receipt not found", "failed to load receipt")
			return
		}
		docs = append(docs, rc.Document())
		name = rc.Number
		if rc.DuplicateOf != nil {
			name += "-duplikat"
		}
	}
	if len(docs) == 0 {
		badRequest(w, "ids must name at least one receipt")
		return
	}
	if len(docs) > 1 {
		name = time.Now().Format(dateLayout)
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="zuwendungsbestaetigung-`+name+`.pdf"`)
	_, _ = w.Write(donation.PDF(docs))
}
