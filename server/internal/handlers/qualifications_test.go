package handlers

import (
	"context"
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

func TestQualifications(t *testing.T) {
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
	router.Get("/api/v2/qualification-types", wrap(api.ListQualificationTypes))
	router.Post("/api/v2/qualification-types", wrap(api.CreateQualificationType))
	router.Get("/api/v2/qualifications", wrap(api.ListQualifications))
	router.Post("/api/v2/members/{id}/qualifications", wrap(api.CreateQualification))

	var m openapi.Member
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 1, "firstName": "Ada", "lastName": "Lovelace", "email": "ada@example.org",
	}, &m); code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}
	call := func(role openapi.AppRole, method, path, body string) *httptest.ResponseRecorder {
		token, err := auth.NewToken("sec@example.org", role, secret)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	for _, role := range []openapi.AppRole{openapi.READONLY, openapi.TREASURER} {
		if rec := call(role, http.MethodGet, "/api/v2/qualifications", ""); rec.Code != http.StatusForbidden {
			t.Errorf("%s may list: %d", role, rec.Code)
		}
	}
	rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/qualification-types", `{"name":"Führungszeugnis","validityMonths":60,"inspectionOnly":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create type: %d %s", rec.Code, rec.Body)
	}
	typeID := between(rec.Body.String(), `"id":"`, `"`)
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/qualification-types", `{"name":"Führungszeugnis"}`); rec.Code != http.StatusConflict {
		t.Errorf("duplicate type: %d, want 409", rec.Code)
	}

	issued := time.Now().AddDate(-5, 0, 20).Format(dateLayout) // expires in 20 days
	rec = call(openapi.SECRETARY, http.MethodPost, "/api/v2/members/"+m.Id+"/qualifications",
		`{"typeId":"`+typeID+`","issuedDate":"`+issued+`","note":"Inhalt des Zeugnisses"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create qualification: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.ADMIN, http.MethodGet, "/api/v2/qualifications?warnDays=30", "")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"status":"expiring"`) || strings.Contains(body, "Inhalt") {
		t.Errorf("list: %d %s", rec.Code, body)
	}
	rec = call(openapi.ADMIN, http.MethodGet, "/api/v2/qualifications?warnDays=5", "")
	if !strings.Contains(rec.Body.String(), `"status":"valid"`) {
		t.Errorf("warnDays=5: %s", rec.Body)
	}
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/members/"+m.Id+"/qualifications", `{"typeId":"nope","issuedDate":"2025-01-01"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown type: %d, want 400", rec.Code)
	}
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	return s[:strings.Index(s, b)]
}
