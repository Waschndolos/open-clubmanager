package handlers

import (
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

func TestFeeModelsAndBillingRuns(t *testing.T) {
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
	router.Get("/api/v2/fee-models", wrap(api.ListFeeModels))
	router.Post("/api/v2/fee-models", wrap(api.CreateFeeModel))
	router.Get("/api/v2/fee-models/members", wrap(api.ListMemberFeeModels))
	router.Put("/api/v2/fee-models/{id}", wrap(api.UpdateFeeModel))
	router.Delete("/api/v2/fee-models/{id}", wrap(api.DeleteFeeModel))
	router.Get("/api/v2/members/{id}/fee-model", wrap(api.GetMemberFeeModel))
	router.Put("/api/v2/members/{id}/fee-model", wrap(api.SetMemberFeeModel))
	router.Get("/api/v2/fee-runs", wrap(api.ListFeeRuns))
	router.Post("/api/v2/fee-runs", wrap(api.CreateFeeRun))
	router.Post("/api/v2/fee-runs/preview", wrap(api.PreviewFeeRun))
	router.Delete("/api/v2/fee-runs/{id}", wrap(api.DeleteFeeRun))

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
	decode := func(rec *httptest.ResponseRecorder, out any) {
		t.Helper()
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %q: %v", rec.Body, err)
		}
	}

	// Models: adults from 18 pro rata, youth up to 17, passive without age limit.
	for _, role := range []openapi.AppRole{openapi.SECRETARY, openapi.READONLY, openapi.TRAINER} {
		if rec := call(role, http.MethodPost, "/api/v2/fee-models", `{"name":"X","amount":1,"interval":"yearly"}`); rec.Code != http.StatusForbidden {
			t.Errorf("%s may create fee models: %d", role, rec.Code)
		}
	}
	for _, body := range []string{
		`{"name":"","amount":10,"interval":"yearly"}`,
		`{"name":"A","amount":0,"interval":"yearly"}`,
		`{"name":"A","amount":10,"interval":"weekly"}`,
		`{"name":"A","amount":10,"interval":"yearly","minAge":20,"maxAge":10}`,
	} {
		if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-models", body); rec.Code != http.StatusBadRequest {
			t.Errorf("invalid model %s accepted: %d", body, rec.Code)
		}
	}
	models := map[string]feeModelJSON{}
	for key, body := range map[string]string{
		"adult":   `{"name":"Erwachsene","amount":120,"interval":"yearly","minAge":18,"proRata":true}`,
		"youth":   `{"name":"Jugend","amount":15,"interval":"quarterly","maxAge":17}`,
		"passive": `{"name":"Passiv","amount":30,"interval":"yearly"}`,
	} {
		rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-models", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create model %s: %d %s", key, rec.Code, rec.Body)
		}
		var m feeModelJSON
		decode(rec, &m)
		models[key] = m
	}
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-models", `{"name":"Passiv","amount":10,"interval":"yearly"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "FEE_MODEL_NAME_TAKEN") {
		t.Errorf("duplicate name: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.READONLY, http.MethodGet, "/api/v2/fee-models", ""); rec.Code != http.StatusOK {
		t.Errorf("read-only login cannot list models: %d", rec.Code)
	}

	members := []map[string]any{
		{"number": 1, "firstName": "Jannik", "lastName": "Petersen", "email": "jannik@example.org", "birthday": "1980-03-10T00:00:00.000Z"},
		// Joins on 1 April: 9 of 12 months. Saved by the member dialog in
		// Germany as local midnight, i.e. on the previous day in UTC.
		{"number": 2, "firstName": "Merle", "lastName": "Hansen", "email": "merle@example.org", "birthday": "1990-06-01T00:00:00.000Z",
			"entryDate": "2027-03-31T22:00:00.000Z"},
		// 17 on 1 January, 18 on 1 July: youth for the whole year (age at the period start).
		{"number": 3, "firstName": "Ole", "lastName": "Jensen", "email": "ole@example.org", "birthday": "2009-06-15T00:00:00.000Z"},
		// Leaves on 1 April: only the first quarter (local midnight in winter time).
		{"number": 4, "firstName": "Kira", "lastName": "Thomsen", "email": "kira@example.org", "birthday": "2015-01-31T23:00:00.000Z",
			"exitDate": "2027-03-31T22:00:00.000Z"},
		// Left before the year.
		{"number": 5, "firstName": "Mats", "lastName": "Andresen", "email": "mats@example.org", "birthday": "1970-01-01T00:00:00.000Z",
			"exitDate": "2026-12-31T00:00:00.000Z"},
		{"number": 6, "firstName": "Greta", "lastName": "Möller", "email": "greta@example.org"}, // no birthday
		{"number": 7, "firstName": "Hauke", "lastName": "Albers", "email": "hauke@example.org", "birthday": "1950-01-01T00:00:00.000Z"},
		{"number": 8, "firstName": "Sophie", "lastName": "Lorenzen", "email": "sophie@example.org", "birthday": "1960-01-01T00:00:00.000Z"},
	}
	ids := make([]string, len(members))
	for i, body := range members {
		var m openapi.Member
		if code := doJSON(t, router, http.MethodPost, "/api/v2/members", body, &m); code != http.StatusCreated {
			t.Fatalf("create member: %d", code)
		}
		ids[i] = m.Id
	}
	// Hauke is an honorary member, Sophie passive.
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/members/"+ids[6]+"/fee-model", `{"mode":"exempt"}`); rec.Code != http.StatusOK {
		t.Fatalf("set exempt: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/members/"+ids[7]+"/fee-model", `{"mode":"model","feeModelId":"nope"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown model accepted: %d", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodPut, "/api/v2/members/"+ids[7]+"/fee-model", `{"mode":"exempt"}`); rec.Code != http.StatusForbidden {
		t.Errorf("secretary may assign fee models: %d", rec.Code)
	}
	if rec := call(openapi.TREASURER, http.MethodPut, "/api/v2/members/"+ids[7]+"/fee-model",
		`{"mode":"model","feeModelId":"`+models["passive"].ID+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("set model: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.TREASURER, http.MethodGet, "/api/v2/members/"+ids[7]+"/fee-model", ""); !strings.Contains(rec.Body.String(), models["passive"].ID) {
		t.Errorf("get assignment: %s", rec.Body)
	}
	var onDay []memberFeeModelJSON
	decode(call(openapi.READONLY, http.MethodGet, "/api/v2/fee-models/members?date=2027-01-01", ""), &onDay)
	byNumber := map[int]memberFeeModelJSON{}
	for _, m := range onDay {
		byNumber[m.Member.Number] = m
	}
	if len(onDay) != 6 || byNumber[3].Model == nil || byNumber[3].Model.Name != "Jugend" || byNumber[7].Mode != "exempt" ||
		byNumber[8].Model.Name != "Passiv" || byNumber[6].Model != nil {
		t.Errorf("models on 1 January: %+v", onDay)
	}

	// Preview the year 2027.
	year := `{"from":"2027-01","to":"2027-12","dueDate":"2027-03-01"}`
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs/preview", `{"from":"2027-12","to":"2027-01"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("reversed range accepted: %d", rec.Code)
	}
	rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs/preview", year)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body)
	}
	var plan feeRunPlanJSON
	decode(rec, &plan)
	got := map[string]float64{}
	for _, it := range plan.Items {
		got[it.Member.FirstName+" "+it.Period.Label] += it.Amount
	}
	want := map[string]float64{
		"Jannik 2027": 120, "Merle 2027": 90,
		"Ole 2027-Q1": 15, "Ole 2027-Q2": 15, "Ole 2027-Q3": 15, "Ole 2027-Q4": 15,
		"Kira 2027-Q1": 15, "Sophie 2027": 30,
	}
	if len(got) != len(want) {
		t.Errorf("preview items %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, want %v", k, got[k], v)
		}
	}
	if plan.Count != 8 || plan.Total != 315 {
		t.Errorf("preview count %d total %v", plan.Count, plan.Total)
	}
	skipped := map[string]string{}
	for _, s := range plan.Skipped {
		skipped[s.Member.FirstName] = s.Reason
	}
	if len(skipped) != 2 || skipped["Greta"] != "NO_MODEL" || skipped["Hauke"] != "EXEMPT" {
		t.Errorf("skipped: %v", skipped)
	}
	for _, it := range plan.Items {
		if it.Member.FirstName == "Merle" && (!it.ProRata || it.Description != "Erwachsene 2027") {
			t.Errorf("Merle's item: %+v", it)
		}
		if it.Member.FirstName == "Ole" && it.Period.Label == "2027-Q3" && it.DueDate != "2027-07-01" {
			t.Errorf("Q3 due on %s", it.DueDate)
		}
	}

	// Nothing is saved by the preview; the run creates the fees once.
	var fees []openapi.MemberFee
	doJSON(t, router, http.MethodGet, "/api/v2/finance/memberfees", nil, &fees)
	if len(fees) != 0 {
		t.Fatalf("preview created %d fees", len(fees))
	}
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/fee-runs", year); rec.Code != http.StatusForbidden {
		t.Errorf("secretary may create a run: %d", rec.Code)
	}
	rec = call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs", year)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create run: %d %s", rec.Code, rec.Body)
	}
	var run feeRunJSON
	decode(rec, &run)
	if run.Count != 8 || run.Total != 315 || run.From != "2027-01" || run.To != "2027-12" {
		t.Errorf("run: %+v", run)
	}
	doJSON(t, router, http.MethodGet, "/api/v2/finance/memberfees", nil, &fees)
	if len(fees) != 8 {
		t.Fatalf("%d fees after the run", len(fees))
	}

	// The same period again is detected and refused.
	decode(call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs/preview", year), &plan)
	if plan.Count != 0 {
		t.Errorf("second preview has %d items", plan.Count)
	}
	already := 0
	for _, s := range plan.Skipped {
		if s.Reason == "ALREADY_CHARGED" {
			already++
		}
	}
	if already != 8 {
		t.Errorf("%d periods reported as already charged", already)
	}
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs", year); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "FEE_RUN_EMPTY") {
		t.Errorf("second run: %d %s", rec.Code, rec.Body)
	}
	// An overlapping range charges only the new periods (the 2028 ones).
	decode(call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs/preview", `{"from":"2027-07","to":"2028-03"}`), &plan)
	oleAdult := false
	for _, it := range plan.Items {
		if !strings.HasPrefix(it.Period.Label, "2028") {
			t.Errorf("overlapping range charges %s %s again", it.Member.FirstName, it.Period.Label)
		}
		oleAdult = oleAdult || (it.Member.FirstName == "Ole" && it.Model.Name == "Erwachsene")
	}
	if !oleAdult {
		t.Error("Ole, 18 on 1 January 2028, does not change to the adult model")
	}

	// Deleting a fee frees its period; deleting the run removes the open fees.
	var jannik string
	for _, f := range fees {
		if f.Member.FirstName == "Jannik" {
			jannik = f.Id
		}
	}
	if code := doJSON(t, router, http.MethodPut, "/api/v2/finance/memberfees/"+jannik,
		map[string]any{"paidDate": "2027-03-05T00:00:00.000Z"}, nil); code != http.StatusOK {
		t.Fatalf("mark paid: %d", code)
	}
	rec = call(openapi.TREASURER, http.MethodDelete, "/api/v2/fee-runs/"+run.ID, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"removed":7`) {
		t.Fatalf("delete run: %d %s", rec.Code, rec.Body)
	}
	var runs []feeRunJSON
	decode(call(openapi.TREASURER, http.MethodGet, "/api/v2/fee-runs", ""), &runs)
	if len(runs) != 1 || runs[0].Count != 1 || runs[0].PaidCount != 1 {
		t.Errorf("runs after delete: %+v", runs)
	}
	decode(call(openapi.TREASURER, http.MethodPost, "/api/v2/fee-runs/preview", year), &plan)
	if plan.Count != 7 {
		t.Errorf("after deleting the run %d periods can be charged, want 7", plan.Count)
	}

	// Deleting a model sends its members back to the selection by age.
	if rec := call(openapi.TREASURER, http.MethodDelete, "/api/v2/fee-models/"+models["passive"].ID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete model: %d", rec.Code)
	}
	if rec := call(openapi.TREASURER, http.MethodGet, "/api/v2/members/"+ids[7]+"/fee-model", ""); !strings.Contains(rec.Body.String(), `"mode":"auto"`) {
		t.Errorf("assignment after deleting the model: %s", rec.Body)
	}
}
