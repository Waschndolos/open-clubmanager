package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

func TestSEPADirectDebit(t *testing.T) {
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
	router.Get("/api/v2/sepa/creditor", wrap(api.GetSEPACreditor))
	router.Put("/api/v2/sepa/creditor", wrap(api.SetSEPACreditor))
	router.Get("/api/v2/sepa/candidates", wrap(api.ListSEPACandidates))
	router.Get("/api/v2/finance/memberfees", wrap(api.ListMemberFees))
	router.Get("/api/v2/sepa/runs", wrap(api.ListSEPARuns))
	router.Post("/api/v2/sepa/runs", wrap(api.CreateSEPARun))
	router.Get("/api/v2/sepa/runs/{id}/xml", wrap(api.DownloadSEPARun))
	router.Post("/api/v2/sepa/runs/{id}/confirm", wrap(api.ConfirmSEPARun))
	router.Delete("/api/v2/sepa/runs/{id}", wrap(api.CancelSEPARun))
	router.Put("/api/v2/members/{id}/sepa-mandate", wrap(api.SetSEPAMandate))

	// Example IBANs from bank documentation, never real accounts.
	members := []map[string]any{
		{"number": 1, "firstName": "Jannik", "lastName": "Petersen", "email": "jannik@example.org", "iban": "DE89 3704 0044 0532 0130 00",
			"bic": "COBADEFFXXX", "sepaMandateDate": "2020-04-01T00:00:00.000Z"},
		{"number": 2, "firstName": "Merle", "lastName": "Hansen", "email": "merle@example.org", "iban": "DE02500105170137075030",
			"accountHolder": "Familie Hansen", "sepaMandateDate": "2021-05-15T00:00:00.000Z"},
		{"number": 3, "firstName": "Ole", "lastName": "Jensen", "email": "ole@example.org", "iban": "DE02120300000000202051"}, // no mandate
	}
	ids := make([]string, len(members))
	fees := make([]string, len(members))
	for i, body := range members {
		var m openapi.Member
		if code := doJSON(t, router, http.MethodPost, "/api/v2/members", body, &m); code != http.StatusCreated {
			t.Fatalf("create member: %d", code)
		}
		ids[i] = m.Id
		var f openapi.MemberFee
		if code := doJSON(t, router, http.MethodPost, "/api/v2/finance/memberfees", map[string]any{
			"memberId": m.Id, "amount": 90.5, "dueDate": "2026-03-01T00:00:00.000Z", "year": 2026,
			"description": "Jahresbeitrag 2026",
		}, &f); code != http.StatusCreated {
			t.Fatalf("create fee: %d", code)
		}
		fees[i] = f.Id
	}
	// The API refuses IBANs with bad check digits (#193), but older data or a
	// pre-validation device of a collaboration club may still hold one.
	if _, err := conn.ExecContext(context.Background(),
		`UPDATE member_bank_details SET iban = 'DE02500105170137075031' WHERE member_id = ?`, ids[2]); err != nil {
		t.Fatal(err)
	}

	call := func(role openapi.AppRole, method, path, body string) *httptest.ResponseRecorder {
		token, err := auth.NewToken("kasse@example.org", role, secret)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	for _, role := range []openapi.AppRole{openapi.SECRETARY, openapi.READONLY, openapi.TRAINER} {
		if rec := call(role, http.MethodGet, "/api/v2/sepa/candidates", ""); rec.Code != http.StatusForbidden {
			t.Errorf("%s may list candidates: %d", role, rec.Code)
		}
	}
	if rec := call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/creditor", ""); rec.Code != http.StatusNoContent {
		t.Errorf("creditor before setup: %d", rec.Code)
	}
	tomorrow := time.Now().AddDate(0, 0, 2).Format(dateLayout)
	run := `{"collectionDate":"` + tomorrow + `","feeIds":["` + fees[0] + `"]}`
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/sepa/runs", run); rec.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(rec.Body.String(), "SEPA_NOT_CONFIGURED") {
		t.Errorf("run without creditor: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/sepa/creditor",
		`{"name":"Kiteclub","creditorId":"DE97ZZZ09999999999","iban":"DE02120300000000202051"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid creditor id accepted: %d", rec.Code)
	}
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/sepa/creditor",
		`{"name":"Kiteclub Nordwind e.V.","creditorId":"de98 zzz0 9999 9999 99","iban":"DE02 1203 0000 0000 2020 51",
		"remittanceTemplate":"{description} Nr. {number}"}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"creditorId":"DE98ZZZ09999999999"`) {
		t.Fatalf("set creditor: %d %s", rec.Code, rec.Body)
	}

	// Mandate references: member 1 and 2 get one, duplicates are rejected.
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/members/"+ids[0]+"/sepa-mandate", `{"mandateReference":"NW 1"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("reference with space accepted: %d", rec.Code)
	}
	for i, ref := range []string{"NW-1", "NW-2"} {
		if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/members/"+ids[i]+"/sepa-mandate", `{"mandateReference":"`+ref+`"}`); rec.Code != http.StatusOK {
			t.Fatalf("set mandate: %d %s", rec.Code, rec.Body)
		}
	}
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/members/"+ids[2]+"/sepa-mandate", `{"mandateReference":"NW-1"}`); rec.Code != http.StatusConflict {
		t.Errorf("duplicate reference: %d", rec.Code)
	}

	var candidates []sepaCandidateJSON
	rec := call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/candidates", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &candidates); err != nil || len(candidates) != 3 {
		t.Fatalf("candidates: %d %s", rec.Code, rec.Body)
	}
	byFee := map[string]sepaCandidateJSON{}
	for _, c := range candidates {
		byFee[c.FeeID] = c
	}
	if c := byFee[fees[0]]; len(c.Issues) != 0 || c.SequenceType != "FRST" || c.Remittance != "Jahresbeitrag 2026 Nr. 1" ||
		*c.IBAN != "DE89370400440532013000" {
		t.Errorf("candidate 1: %+v", c)
	}
	if c := byFee[fees[1]]; c.DebtorName != "Familie Hansen" {
		t.Errorf("debtor name: %+v", c)
	}
	if c := byFee[fees[2]]; strings.Join(c.Issues, ",") != "INVALID_IBAN,MISSING_MANDATE,MISSING_MANDATE_REFERENCE" {
		t.Errorf("issues of member 3: %v", c.Issues)
	}

	// A run with an invalid fee is rejected as a whole.
	all := `{"collectionDate":"` + tomorrow + `","feeIds":["` + strings.Join(fees, `","`) + `"]}`
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/sepa/runs", all); rec.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(rec.Body.String(), fees[2]) {
		t.Errorf("run with invalid fee: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/sepa/runs",
		`{"collectionDate":"2020-01-01","feeIds":["`+fees[0]+`"]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("past collection date: %d", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/sepa/runs", run); rec.Code != http.StatusForbidden {
		t.Errorf("secretary may create run: %d", rec.Code)
	}
	var created sepaRunJSON
	rec = call(openapi.TREASURER, http.MethodPost, "/api/v2/sepa/runs",
		`{"collectionDate":"`+tomorrow+`","feeIds":["`+fees[0]+`","`+fees[1]+`"]}`)
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &created) != nil || created.Count != 2 || created.Total != 181 {
		t.Fatalf("create run: %d %s", rec.Code, rec.Body)
	}

	// The fees are being collected now and no longer open.
	rec = call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/candidates", "")
	if strings.Contains(rec.Body.String(), fees[0]) || !strings.Contains(rec.Body.String(), fees[2]) {
		t.Errorf("collected fees still open: %s", rec.Body)
	}

	rec = call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/runs/"+created.ID+"/xml", "")
	xml := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), "lastschrift-"+tomorrow) ||
		strings.Count(xml, "<DrctDbtTxInf>") != 2 || !strings.Contains(xml, "<SeqTp>FRST</SeqTp>") ||
		!strings.Contains(xml, "<MndtId>NW-2</MndtId>") || !strings.Contains(xml, "<CtrlSum>181.00</CtrlSum>") ||
		!strings.Contains(xml, "<Nm>Familie Hansen</Nm>") {
		t.Errorf("xml: %d %s", rec.Code, xml)
	}

	// Cancelling opens the fees again; the next run is the first one again.
	if rec := call(openapi.TREASURER, http.MethodDelete, "/api/v2/sepa/runs/"+created.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.TREASURER, http.MethodPost, "/api/v2/sepa/runs", run)
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &created) != nil || created.Items[0].SequenceType != "FRST" {
		t.Fatalf("second run: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.TREASURER, http.MethodPost, "/api/v2/sepa/runs/"+created.ID+"/confirm", `{"paidDate":"`+tomorrow+`"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"paidDate":"`+tomorrow+`"`) {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.TREASURER, http.MethodDelete, "/api/v2/sepa/runs/"+created.ID, ""); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("cancel confirmed run: %d", rec.Code)
	}
	var fee openapi.MemberFee
	for _, f := range listFees(t, router, secret) {
		if f.Id == fees[0] {
			fee = f
		}
	}
	if fee.PaidDate == nil || fee.PaidDate.Format(dateLayout) != tomorrow {
		t.Errorf("fee not paid: %+v", fee)
	}

	// A new fee of the same member is now collected as recurring.
	var f openapi.MemberFee
	doJSON(t, router, http.MethodPost, "/api/v2/finance/memberfees", map[string]any{
		"memberId": ids[0], "amount": 10, "dueDate": "2027-03-01T00:00:00.000Z", "year": 2027}, &f)
	rec = call(openapi.TREASURER, http.MethodGet, "/api/v2/sepa/candidates", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &candidates); err != nil {
		t.Fatal(err)
	}
	for _, c := range candidates {
		if c.FeeID == f.Id && (c.SequenceType != "RCUR" || c.Remittance != "Beitrag 2027 Nr. 1") {
			t.Errorf("follow-up fee: %+v", c)
		}
	}
}

func listFees(t *testing.T, h http.Handler, secret string) []openapi.MemberFee {
	var list []openapi.MemberFee
	if code := doJSONAs(t, h, secret, openapi.TREASURER, "/api/v2/finance/memberfees", &list); code != http.StatusOK {
		t.Fatalf("list fees: %d", code)
	}
	return list
}
