package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

type reconResponse struct {
	Entries []reconEntryJSON `json:"entries"`
	Fees    []reconFeeJSON   `json:"fees"`
}

// TestPaymentReconciliation imports the example statement of testdata/ into
// a club like the demo data and walks through the review.
func TestPaymentReconciliation(t *testing.T) {
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
	router.Post("/api/v2/households", wrap(api.CreateHousehold))
	router.Get("/api/v2/sepa/candidates", wrap(api.ListSEPACandidates))
	router.Get("/api/v2/finance/memberfees", wrap(api.ListMemberFees))
	router.Get("/api/v2/reconciliation", wrap(api.GetReconciliation))
	router.Post("/api/v2/reconciliation/confirm-certain", wrap(api.ConfirmCertainReconciliations))
	router.Post("/api/v2/reconciliation/{id}/confirm", wrap(api.ConfirmReconciliation))
	router.Post("/api/v2/reconciliation/{id}/dismiss", wrap(api.DismissReconciliation))
	router.Post("/api/v2/reconciliation/{id}/reopen", wrap(api.ReopenReconciliation))

	call := func(role openapi.AppRole, method, path string, body any, out any) int {
		t.Helper()
		token, err := auth.NewToken("kasse@example.org", role, secret)
		if err != nil {
			t.Fatal(err)
		}
		var data []byte
		if body != nil {
			if data, err = json.Marshal(body); err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, path, strings.NewReader(string(data)))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if out != nil && rec.Code < 300 {
			if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
				t.Fatalf("%s %s: %v: %s", method, path, err, rec.Body)
			}
		}
		return rec.Code
	}

	// The members of the demo data with open fees, and Sophie Lorenzen,
	// whose paid fee is returned. Example IBANs, never real accounts.
	type demo struct {
		number       int
		first, last  string
		iban, holder string
		amount       float64
		paid         bool
	}
	demos := []demo{
		{1006, "Sophie", "Lorenzen", "DE02100500000054540402", "Sophie Lorenzen", 180, true},
		{1007, "Mats", "Andresen", "DE75512108001245126199", "Mats Andresen", 180, false},
		{1010, "Nele", "Kruse", "", "", 90, false},
		{1011, "Paul", "Kruse", "", "", 120, false},
		{1012, "Tjark", "Albers", "DE88100900001234567892", "Familie Albers", 90, false},
		{1013, "Ida", "Wendt", "", "", 180, false},
		{1015, "Greta", "Möller", "DE89370400440532013000", "Greta Möller", 180, false},
	}
	member := map[int]string{}
	fee := map[int]string{}
	for _, d := range demos {
		body := map[string]any{"number": d.number, "firstName": d.first, "lastName": d.last,
			"email": strings.ToLower(d.first+"."+d.last) + "@example.org"}
		if d.iban != "" {
			body["iban"], body["accountHolder"] = d.iban, d.holder
		}
		var m openapi.Member
		if code := doJSON(t, router, http.MethodPost, "/api/v2/members", body, &m); code != http.StatusCreated {
			t.Fatalf("create member: %d", code)
		}
		member[d.number] = m.Id
		f := map[string]any{"memberId": m.Id, "amount": d.amount, "dueDate": "2026-03-01T00:00:00.000Z", "year": 2026,
			"description": "Jahresbeitrag 2026"}
		if d.paid {
			f["paidDate"] = "2026-01-15T00:00:00.000Z"
		}
		var created openapi.MemberFee
		if code := doJSON(t, router, http.MethodPost, "/api/v2/finance/memberfees", f, &created); code != http.StatusCreated {
			t.Fatalf("create fee: %d", code)
		}
		fee[d.number] = created.Id
	}
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/households", map[string]any{"name": "Familie Kruse",
		"memberIds": []string{member[1010], member[1011]}}, nil); code != http.StatusCreated {
		t.Fatalf("create household: %d", code)
	}

	xml, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "camt053-reconciliation.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var imported openapi.Camt053ImportResponse
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/finance/transactions/import/camt053",
		map[string]string{"xml": string(xml)}, &imported); code != http.StatusOK {
		t.Fatalf("import: %d", code)
	}
	if imported.ImportedCount != 8 || imported.MatchedMemberCount != 6 || imported.MemberFeesMarkedPaid != 0 {
		t.Errorf("import result %+v", imported)
	}

	for _, role := range []openapi.AppRole{openapi.SECRETARY, openapi.READONLY, openapi.TRAINER} {
		if code := call(role, http.MethodGet, "/api/v2/reconciliation", nil, nil); code != http.StatusForbidden {
			t.Errorf("%s may open the reconciliation: %d", role, code)
		}
	}

	load := func() map[string]reconEntryJSON {
		t.Helper()
		var res reconResponse
		if code := call(openapi.TREASURER, http.MethodGet, "/api/v2/reconciliation", nil, &res); code != http.StatusOK {
			t.Fatalf("get reconciliation: %d", code)
		}
		out := map[string]reconEntryJSON{}
		for _, e := range res.Entries {
			key := e.Description
			if e.CounterpartyName != nil {
				key = *e.CounterpartyName
			}
			out[key] = e
		}
		return out
	}
	entries := load()
	want := map[string]reconcileExpectation{
		"Ida Wendt":             {"open", "certain", []string{fee[1013]}},
		"Familie Albers":        {"open", "certain", []string{fee[1012]}},
		"Mats Andresen":         {"open", "probable", []string{fee[1007]}},
		"Birte Kruse":           {"open", "probable", []string{fee[1010], fee[1011]}},
		"G. Moeller":            {"open", "unclear", []string{fee[1015]}},
		"Sophie Lorenzen":       {"open", "certain", []string{fee[1006]}},
		"Surfshop Fehmarn GmbH": {"open", "", nil},
		"Kite Repair Service":   {"none", "", nil},
	}
	for name, w := range want {
		e, ok := entries[name]
		if !ok {
			t.Fatalf("entry of %s missing", name)
		}
		w.check(t, name, e)
	}
	if e := entries["Sophie Lorenzen"]; !e.IsReturn || e.ReturnReason == nil || *e.ReturnReason != "MD06" || e.Type != "expense" {
		t.Errorf("return not recognised: %+v", e)
	}
	if e := entries["Mats Andresen"]; e.CounterpartyIBAN == nil || *e.CounterpartyIBAN != "DE75512108001245126199" {
		t.Errorf("payer IBAN: %+v", e.CounterpartyIBAN)
	}

	// Nothing is paid before a confirmation.
	feeOf := func(number int) openapi.MemberFee {
		t.Helper()
		var f openapi.MemberFee
		var list []openapi.MemberFee
		doJSONAs(t, router, secret, openapi.TREASURER, "/api/v2/finance/memberfees", &list)
		for _, x := range list {
			if x.Id == fee[number] {
				f = x
			}
		}
		return f
	}
	if feeOf(1013).PaidDate != nil {
		t.Fatal("fee paid without confirmation")
	}

	// Confirming the certain suggestions pays Ida and Tjark and reverses
	// Sophie's direct debit.
	var confirmed map[string]int
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/reconciliation/confirm-certain", nil, &confirmed); code != http.StatusOK ||
		confirmed["confirmed"] != 3 {
		t.Fatalf("confirm certain: %d %v", code, confirmed)
	}
	if f := feeOf(1013); f.PaidDate == nil || f.PaidDate.Format(dateLayout) != "2026-10-05" {
		t.Errorf("Ida's fee: %v", f.PaidDate)
	}
	if feeOf(1012).PaidDate == nil {
		t.Error("Tjark's fee not paid")
	}
	if feeOf(1006).PaidDate != nil {
		t.Error("returned fee still paid")
	}
	// The returned fee can be collected again by direct debit.
	var candidates []sepaCandidateJSON
	call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/candidates", nil, &candidates)
	found := false
	for _, c := range candidates {
		if c.FeeID == fee[1006] && c.Amount == 180 {
			found = true
		}
	}
	if !found {
		t.Errorf("returned fee not collectable: %+v", candidates)
	}

	// Mats pays the first instalment: the fee stays open with 90 € due,
	// and a direct debit would only collect the rest.
	mats := entries["Mats Andresen"]
	confirm := func(e reconEntryJSON, allocations []reconAllocationJSON) (int, reconEntryJSON) {
		var out reconEntryJSON
		code := call(openapi.TREASURER, http.MethodPost, "/api/v2/reconciliation/"+e.TransactionID+"/confirm",
			map[string]any{"allocations": allocations}, &out)
		return code, out
	}
	if code, _ := confirm(mats, []reconAllocationJSON{{FeeID: fee[1007], Amount: 120}}); code != http.StatusUnprocessableEntity {
		t.Errorf("more than the transaction accepted: %d", code)
	}
	if code, out := confirm(mats, mats.Suggestion.Allocations); code != http.StatusOK || out.Status != "reconciled" {
		t.Fatalf("confirm partial payment: %d %+v", code, out)
	}
	if feeOf(1007).PaidDate != nil {
		t.Error("partially paid fee marked paid")
	}
	call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/candidates", nil, &candidates)
	for _, c := range candidates {
		if c.FeeID == fee[1007] && c.Amount != 90 {
			t.Errorf("direct debit of a partly paid fee: %v", c.Amount)
		}
	}
	if code, _ := confirm(mats, mats.Suggestion.Allocations); code != http.StatusUnprocessableEntity {
		t.Errorf("confirmed twice: %d", code)
	}

	// The family transfer is split between both children.
	kruse := entries["Birte Kruse"]
	if code, _ := confirm(kruse, kruse.Suggestion.Allocations); code != http.StatusOK {
		t.Fatalf("confirm family transfer: %d", code)
	}
	if feeOf(1010).PaidDate == nil || feeOf(1011).PaidDate == nil {
		t.Error("family fees not paid")
	}

	// Greta's unclear transfer is corrected by hand to a part of her fee;
	// the sponsoring is dismissed.
	greta := entries["G. Moeller"]
	if code, _ := confirm(greta, []reconAllocationJSON{{FeeID: fee[1015], Amount: 100}}); code != http.StatusOK {
		t.Fatalf("confirm corrected: %d", code)
	}
	var dismissed reconEntryJSON
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/reconciliation/"+entries["Surfshop Fehmarn GmbH"].TransactionID+"/dismiss",
		nil, &dismissed); code != http.StatusOK || dismissed.Status != "dismissed" {
		t.Errorf("dismiss: %d %+v", code, dismissed)
	}
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/reconciliation/"+entries["Kite Repair Service"].TransactionID+"/confirm",
		map[string]any{"allocations": []reconAllocationJSON{{FeeID: fee[1015], Amount: 10}}}, nil); code != http.StatusUnprocessableEntity {
		t.Errorf("expense assigned: %d", code)
	}

	// Undoing the return makes Sophie's fee paid again, with its old date.
	var reopened reconEntryJSON
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/reconciliation/"+entries["Sophie Lorenzen"].TransactionID+"/reopen",
		nil, &reopened); code != http.StatusOK || reopened.Status != "open" || reopened.Suggestion == nil {
		t.Fatalf("reopen: %d %+v", code, reopened)
	}
	if f := feeOf(1006); f.PaidDate == nil || f.PaidDate.Format(dateLayout) != "2026-01-15" {
		t.Errorf("paid date after undo: %v", f.PaidDate)
	}

	// Returned again after it was collected once more: the fee is due once,
	// not twice.
	if code, _ := confirm(reopened, reopened.Suggestion.Allocations); code != http.StatusOK {
		t.Fatalf("confirm return again: %d", code)
	}
	recollected := feeOf(1006)
	if recollected.PaidDate != nil {
		t.Fatal("returned fee still paid")
	}
	if code := doJSON(t, router, http.MethodPut, "/api/v2/finance/memberfees/"+fee[1006],
		map[string]any{"paidDate": "2026-11-20T00:00:00.000Z"}, nil); code != http.StatusOK {
		t.Fatalf("mark fee paid: %d", code)
	}
	second := strings.Replace(strings.Replace(string(xml), "RUECKLASTSCHRIFT Jahresbeitrag 2026", "RUECKLASTSCHRIFT 2 Jahresbeitrag 2026", 1),
		"2026-10-19", "2026-11-25", 2)
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/finance/transactions/import/camt053",
		map[string]string{"xml": second}, nil); code != http.StatusOK {
		t.Fatalf("second import: %d", code)
	}
	var res reconResponse
	call(openapi.TREASURER, http.MethodGet, "/api/v2/reconciliation", nil, &res)
	var secondReturn *reconEntryJSON
	for i, e := range res.Entries {
		if strings.HasPrefix(e.Description, "RUECKLASTSCHRIFT 2") {
			secondReturn = &res.Entries[i]
		}
	}
	if secondReturn == nil || secondReturn.Suggestion == nil || len(secondReturn.Suggestion.Allocations) != 1 ||
		secondReturn.Suggestion.Allocations[0].FeeID != fee[1006] {
		t.Fatalf("second return: %+v", secondReturn)
	}
	if code, _ := confirm(*secondReturn, secondReturn.Suggestion.Allocations); code != http.StatusOK {
		t.Fatalf("confirm second return: %d", code)
	}
	call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/candidates", nil, &candidates)
	for _, c := range candidates {
		if c.FeeID == fee[1006] && c.Amount != 180 {
			t.Errorf("due after two returns: %v", c.Amount)
		}
	}
	// Undo the second return: the fee is paid again with the date set by hand.
	if code := call(openapi.TREASURER, http.MethodPost, "/api/v2/reconciliation/"+secondReturn.TransactionID+"/reopen", nil, nil); code != http.StatusOK {
		t.Fatalf("reopen second return: %d", code)
	}
	if f := feeOf(1006); f.PaidDate == nil || f.PaidDate.Format(dateLayout) != "2026-11-20" {
		t.Errorf("paid date after undoing the second return: %v", f.PaidDate)
	}

	// Deleting a transaction settles its fees again.
	if code := doJSON(t, router, http.MethodDelete, "/api/v2/finance/transactions/"+entries["Ida Wendt"].TransactionID, nil, nil); code != http.StatusOK {
		t.Fatalf("delete transaction: %d", code)
	}
	if feeOf(1013).PaidDate != nil {
		t.Error("fee still paid after its payment was deleted")
	}

	// The entries of the first import (the second one repeated them).
	entries = load()
	for name, status := range map[string]string{"Mats Andresen": "reconciled", "Birte Kruse": "reconciled",
		"G. Moeller": "reconciled", "Surfshop Fehmarn GmbH": "dismissed", "Sophie Lorenzen": "reconciled"} {
		if entries[name].Status != status {
			t.Errorf("%s: status %s, want %s", name, entries[name].Status, status)
		}
	}
	if p := entries["Birte Kruse"].Payments; len(p) != 2 || p[0].Amount+p[1].Amount != 210 {
		t.Errorf("family payments: %+v", p)
	}
}

type reconcileExpectation struct {
	status     string
	confidence string
	fees       []string
}

func (w reconcileExpectation) check(t *testing.T, name string, e reconEntryJSON) {
	t.Helper()
	if e.Status != w.status {
		t.Errorf("%s: status %s, want %s", name, e.Status, w.status)
	}
	if w.confidence == "" {
		if e.Suggestion != nil {
			t.Errorf("%s: unexpected suggestion %+v", name, e.Suggestion)
		}
		return
	}
	if e.Suggestion == nil {
		t.Errorf("%s: no suggestion", name)
		return
	}
	if string(e.Suggestion.Confidence) != w.confidence {
		t.Errorf("%s: confidence %s, want %s (%v)", name, e.Suggestion.Confidence, w.confidence, e.Suggestion.Reasons)
	}
	var got []string
	for _, a := range e.Suggestion.Allocations {
		got = append(got, a.FeeID)
	}
	if strings.Join(got, ",") != strings.Join(w.fees, ",") {
		t.Errorf("%s: fees %v, want %v", name, got, w.fees)
	}
}
