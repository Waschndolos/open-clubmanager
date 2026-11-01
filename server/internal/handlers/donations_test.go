package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// TestDonationReceipts walks through the donation receipts of a club year:
// settings, donors, assigning donations, single and annual receipts,
// duplicates and voiding.
func TestDonationReceipts(t *testing.T) {
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	const secret = "test-secret"
	api := New(&config.Config{JWTSecret: secret}, conn)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router})
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			auth.Middleware(secret)(h).ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), openapi.BearerAuthScopes, []string{})))
		}
	}
	router.Get("/api/v2/donations", wrap(api.ListDonations))
	router.Get("/api/v2/donations/settings", wrap(api.GetDonationSettings))
	router.Put("/api/v2/donations/settings", wrap(api.SetDonationSettings))
	router.Get("/api/v2/donations/donors", wrap(api.ListDonors))
	router.Post("/api/v2/donations/donors", wrap(api.CreateDonor))
	router.Delete("/api/v2/donations/donors/{id}", wrap(api.DeleteDonor))
	router.Get("/api/v2/donations/candidates", wrap(api.ListDonationCandidates))
	router.Get("/api/v2/donations/receipts", wrap(api.ListDonationReceipts))
	router.Post("/api/v2/donations/receipts", wrap(api.CreateDonationReceipt))
	router.Post("/api/v2/donations/receipts/annual", wrap(api.CreateAnnualDonationReceipts))
	router.Get("/api/v2/donations/receipts/pdf", wrap(api.DownloadDonationReceipts))
	router.Get("/api/v2/donations/receipts/{id}/pdf", wrap(api.DownloadDonationReceipts))
	router.Post("/api/v2/donations/receipts/{id}/duplicate", wrap(api.DuplicateDonationReceipt))
	router.Post("/api/v2/donations/receipts/{id}/void", wrap(api.VoidDonationReceipt))
	router.Put("/api/v2/donations/{id}", wrap(api.SetDonation))
	router.Delete("/api/v2/donations/{id}", wrap(api.RemoveDonation))

	call := func(role openapi.AppRole, method, path string, body any, out any) *httptest.ResponseRecorder {
		t.Helper()
		token, err := auth.NewToken("kasse@example.org", role, secret)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if out != nil && rec.Code < 300 {
			if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
				t.Fatalf("%s %s: decode %q: %v", method, path, rec.Body.String(), err)
			}
		}
		return rec
	}
	kasse := func(method, path string, body any, out any) *httptest.ResponseRecorder {
		t.Helper()
		return call(openapi.TREASURER, method, path, body, out)
	}
	expect := func(rec *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		if rec.Code != status || !strings.Contains(rec.Body.String(), code) {
			t.Fatalf("want %d %s, got %d %s", status, code, rec.Code, rec.Body)
		}
	}

	// Members: Merle with an address, Ole without one.
	var merle, ole openapi.Member
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{"number": 1, "firstName": "Merle",
		"lastName": "Hansen", "email": "merle@example.org", "street": "Strandweg 2", "postalCode": "25980", "city": "Sylt"}, &merle); code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{"number": 2, "firstName": "Ole",
		"lastName": "Jensen", "email": "ole@example.org"}, &ole); code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}
	tx := func(date, desc string, amount float64, typ, category string) string {
		t.Helper()
		body := map[string]any{"date": date + "T00:00:00.000Z", "description": desc, "amount": amount, "type": typ}
		if category != "" {
			body["category"] = category
		}
		var out openapi.FinanceTransaction
		if code := doJSON(t, router, http.MethodPost, "/api/v2/finance/transactions", body, &out); code != http.StatusCreated {
			t.Fatalf("create transaction: %d", code)
		}
		return out.Id
	}
	tx1 := tx("2025-03-01", "Spende Hansen", 150, "income", "")
	tx2 := tx("2025-11-15", "Aufwandsspende Hansen", 40, "income", "Spende")
	tx3 := tx("2025-05-05", "Spende Nordlicht", 500, "income", "Spende")
	tx4 := tx("2025-06-01", "Spende Jensen", 20, "income", "Spende")
	tx5 := tx("2025-06-02", "Neue Segel", 900, "expense", "Spende")
	tx6 := tx("2025-07-01", "Spende unbekannt", 25, "income", "Spenden")
	tx7 := tx("2024-12-20", "Spende Hansen", 30, "income", "Spende")

	// Only the treasurer and admin may use donation receipts.
	for _, role := range []openapi.AppRole{openapi.SECRETARY, openapi.READONLY, openapi.TRAINER} {
		expect(call(role, http.MethodGet, "/api/v2/donations", nil, nil), http.StatusForbidden, "FORBIDDEN")
	}
	expect(kasse(http.MethodGet, "/api/v2/donations/settings", nil, nil), http.StatusNoContent, "")
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx1}}, nil),
		http.StatusUnprocessableEntity, "DONATIONS_NOT_CONFIGURED")

	// Settings: validated, and an old exemption is flagged.
	settings := map[string]any{"issuerName": "Kiteclub Nordwind e.V.", "issuerAddress": "Deichweg 1\n25980 Sylt",
		"taxOffice": "Nordfriesland", "taxNumber": "15/290/12345", "exemptionKind": "ruling", "exemptionDate": "2019-01-10",
		"assessmentPeriod": "2017", "purposes": "des Sports", "place": "Sylt", "signatory": "Kai Kasse"}
	expect(kasse(http.MethodPut, "/api/v2/donations/settings", settings, nil), http.StatusBadRequest, "exemptionKind")
	settings["exemptionKind"] = "notice"
	var saved donationSettingsJSON
	expect(kasse(http.MethodPut, "/api/v2/donations/settings", settings, &saved), http.StatusOK, "")
	if !saved.ExemptionExpired || saved.ValidUntil != "2024-01-10" {
		t.Errorf("exemption of 2019 must be expired: %+v", saved)
	}
	settings["exemptionDate"], settings["assessmentPeriod"] = "2023-05-02", "2022"
	expect(kasse(http.MethodPut, "/api/v2/donations/settings", settings, &saved), http.StatusOK, "")
	if saved.ExemptionExpired {
		t.Errorf("exemption of 2023 must be valid: %+v", saved)
	}

	// An external donor.
	var donor donorJSON
	expect(kasse(http.MethodPost, "/api/v2/donations/donors", map[string]any{"name": " "}, nil), http.StatusBadRequest, "name")
	expect(kasse(http.MethodPost, "/api/v2/donations/donors", map[string]any{"name": "Nordlicht GmbH", "street": "Hafenstr. 5",
		"postalCode": "25980", "city": "Sylt"}, &donor), http.StatusCreated, "")

	// Assign the donations.
	expect(kasse(http.MethodPut, "/api/v2/donations/"+tx5, map[string]any{"memberId": merle.Id}, nil), http.StatusUnprocessableEntity, "NOT_INCOME")
	expect(kasse(http.MethodPut, "/api/v2/donations/"+tx1, map[string]any{"memberId": merle.Id, "donorId": donor.ID}, nil),
		http.StatusBadRequest, "exactly one")
	for _, a := range []struct {
		tx   string
		body map[string]any
	}{
		{tx1, map[string]any{"memberId": merle.Id}},
		{tx2, map[string]any{"memberId": merle.Id, "expenseWaiver": true}},
		{tx3, map[string]any{"donorId": donor.ID}},
		{tx4, map[string]any{"memberId": ole.Id}},
		{tx7, map[string]any{"memberId": merle.Id}},
	} {
		expect(kasse(http.MethodPut, "/api/v2/donations/"+a.tx, a.body, nil), http.StatusOK, "")
	}
	var donations []donationJSON
	expect(kasse(http.MethodGet, "/api/v2/donations", nil, &donations), http.StatusOK, "")
	byTx := map[string]donationJSON{}
	for _, d := range donations {
		byTx[d.TransactionID] = d
	}
	if len(donations) != 6 || byTx[tx6].Member != nil || byTx[tx6].Donor != nil {
		t.Errorf("donations: want the 5 assigned and the unassigned one with a donation category: %+v", donations)
	}
	if _, ok := byTx[tx5]; ok {
		t.Error("an expense is listed as donation")
	}
	if c := byTx[tx1].Category; c == nil || *c != "Spende" {
		t.Errorf("an assigned donation without category gets the category Spende: %v", c)
	}
	if !byTx[tx2].ExpenseWaiver || byTx[tx3].Donor == nil || byTx[tx3].Donor.Name != "Nordlicht GmbH" {
		t.Errorf("assignment not stored: %+v %+v", byTx[tx2], byTx[tx3])
	}

	// Candidates of 2025; fees are not deductible for a sports club.
	var candidates []candidateJSON
	expect(kasse(http.MethodGet, "/api/v2/donations/candidates?year=2025", nil, &candidates), http.StatusOK, "")
	if len(candidates) != 3 || candidates[0].Name != "Merle Hansen" || len(candidates[0].Items) != 2 ||
		candidates[0].Address != "Strandweg 2\n25980 Sylt" || candidates[2].Name != "Ole Jensen" || candidates[2].Address != "" {
		t.Fatalf("candidates: %+v", candidates)
	}

	// A single receipt, then the same again gives a duplicate.
	var first, dup receiptJSON
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx1},
		"issueDate": "2026-01-10"}, &first), http.StatusCreated, "")
	if first.Number != "2025-0001" || first.Kind != "single" || first.Total != 150 || first.DuplicateOf != nil ||
		first.RecipientAddress != "Strandweg 2\n25980 Sylt" || first.CreatedBy == nil || *first.CreatedBy != "kasse@example.org" {
		t.Errorf("first receipt: %+v", first)
	}
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx1}}, &dup),
		http.StatusCreated, "")
	if dup.Number != first.Number || dup.DuplicateOf == nil || *dup.DuplicateOf != first.ID || dup.IssueDate != "2026-01-10" {
		t.Errorf("re-created receipt must be a duplicate of the first: %+v", dup)
	}
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx1, tx2}}, nil),
		http.StatusConflict, "RECEIPT_ITEMS_CERTIFIED")
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx3}}, nil),
		http.StatusUnprocessableEntity, "RECEIPT_ITEMS_INVALID")
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx2, tx7}}, nil),
		http.StatusUnprocessableEntity, "RECEIPT_ITEMS_INVALID")
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": ole.Id, "items": []string{tx4}}, nil),
		http.StatusUnprocessableEntity, "MISSING_ADDRESS")
	expect(kasse(http.MethodPut, "/api/v2/donations/"+tx1, map[string]any{"donorId": donor.ID}, nil), http.StatusConflict, "DONATION_CERTIFIED")
	expect(kasse(http.MethodDelete, "/api/v2/donations/donors/"+donor.ID, nil, nil), http.StatusConflict, "DONOR_IN_USE")

	pdf := kasse(http.MethodGet, "/api/v2/donations/receipts/"+dup.ID+"/pdf", nil, nil)
	if pdf.Code != http.StatusOK || pdf.Header().Get("Content-Type") != "application/pdf" ||
		!strings.Contains(pdf.Body.String(), "DUPLIKAT") || !strings.Contains(pdf.Header().Get("Content-Disposition"), "2025-0001-duplikat") {
		t.Errorf("duplicate PDF: %d %v", pdf.Code, pdf.Header())
	}
	if pdf := kasse(http.MethodGet, "/api/v2/donations/receipts/"+first.ID+"/pdf", nil, nil); strings.Contains(pdf.Body.String(), "DUPLIKAT") {
		t.Error("the original is marked as duplicate")
	}

	// The annual run certifies what is left; Ole has no address.
	var annual struct {
		Created []receiptJSON   `json:"created"`
		Skipped []candidateJSON `json:"skipped"`
	}
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts/annual", map[string]any{"year": 2025, "issueDate": "2026-01-15"}, &annual),
		http.StatusCreated, "")
	if len(annual.Created) != 2 || len(annual.Skipped) != 1 || annual.Skipped[0].Name != "Ole Jensen" {
		t.Fatalf("annual run: %+v", annual)
	}
	for _, rc := range annual.Created {
		if rc.Number != "2025-0002" && rc.Number != "2025-0003" {
			t.Errorf("annual receipt number %s", rc.Number)
		}
	}
	annualID := annual.Created[0].ID
	if again := kasse(http.MethodPost, "/api/v2/donations/receipts/annual", map[string]any{"year": 2025}, &annual); again.Code != http.StatusCreated ||
		len(annual.Created) != 0 {
		t.Errorf("a second annual run must not certify anything again: %s", again.Body)
	}
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts/annual", map[string]any{"year": 2999}, nil), http.StatusBadRequest, "year")

	// Voiding frees the items: a new receipt gets a new number.
	var voided, renewed receiptJSON
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts/"+first.ID+"/void", map[string]any{"reason": "Betrag falsch"}, &voided),
		http.StatusOK, "")
	if voided.VoidedAt == nil || voided.VoidReason == nil {
		t.Errorf("voided receipt: %+v", voided)
	}
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts/"+first.ID+"/void", map[string]any{}, nil), http.StatusUnprocessableEntity, "RECEIPT_VOIDED")
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts/"+first.ID+"/duplicate", nil, nil), http.StatusUnprocessableEntity, "RECEIPT_VOIDED")
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{tx1}}, &renewed),
		http.StatusCreated, "")
	if renewed.Number != "2025-0004" || renewed.DuplicateOf != nil {
		t.Errorf("receipt after voiding: %+v", renewed)
	}
	if !strings.Contains(kasse(http.MethodGet, "/api/v2/donations/receipts/"+first.ID+"/pdf", nil, nil).Body.String(), "STORNIERT") {
		t.Error("a voided receipt must be marked on the PDF")
	}

	// The log lists every receipt, including the duplicate (also voided).
	var log []receiptJSON
	expect(kasse(http.MethodGet, "/api/v2/donations/receipts?year=2025", nil, &log), http.StatusOK, "")
	if len(log) != 5 {
		t.Fatalf("log: want 5 receipts, got %d", len(log))
	}
	for _, rc := range log {
		if rc.ID == dup.ID && rc.VoidedAt == nil {
			t.Error("the duplicate of a voided receipt must be voided with it")
		}
	}
	both := kasse(http.MethodGet, "/api/v2/donations/receipts/pdf?ids="+annualID+","+renewed.ID, nil, nil)
	if both.Code != http.StatusOK || strings.Count(both.Body.String(), "/Type /Page ") < 2 {
		t.Errorf("combined PDF: %d", both.Code)
	}
	if rec := kasse(http.MethodGet, "/api/v2/donations/receipts/pdf?ids=", nil, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("PDF without ids: %d", rec.Code)
	}

	// Clubs whose fees are deductible can certify paid membership fees.
	var fee openapi.MemberFee
	if code := doJSON(t, router, http.MethodPost, "/api/v2/finance/memberfees", map[string]any{"memberId": merle.Id, "amount": 90.5,
		"dueDate": "2025-03-01T00:00:00.000Z", "paidDate": "2025-04-01T00:00:00.000Z", "year": 2025}, &fee); code != http.StatusCreated {
		t.Fatalf("create fee: %d", code)
	}
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{fee.Id}}, nil),
		http.StatusUnprocessableEntity, "RECEIPT_ITEMS_INVALID")
	settings["feesDeductible"] = true
	expect(kasse(http.MethodPut, "/api/v2/donations/settings", settings, nil), http.StatusOK, "")
	var withFee receiptJSON
	expect(kasse(http.MethodPost, "/api/v2/donations/receipts", map[string]any{"memberId": merle.Id, "items": []string{fee.Id}}, &withFee),
		http.StatusCreated, "")
	if withFee.Items[0].Kind != "membership_fee" || withFee.Total != 90.5 {
		t.Errorf("fee receipt: %+v", withFee)
	}

	// A donation that is not certified can be unassigned.
	expect(kasse(http.MethodDelete, "/api/v2/donations/"+tx4, nil, nil), http.StatusOK, "")
	expect(kasse(http.MethodDelete, "/api/v2/donations/"+tx4, nil, nil), http.StatusNotFound, "")
}
