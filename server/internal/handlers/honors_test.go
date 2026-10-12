package handlers

import (
	"context"
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

func TestJubileesAndHonors(t *testing.T) {
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
	router.Get("/api/v2/jubilees", wrap(api.ListJubilees))
	router.Get("/api/v2/honors", wrap(api.ListHonors))
	router.Delete("/api/v2/honors/{id}", wrap(api.DeleteHonor))
	router.Post("/api/v2/members/{id}/honors", wrap(api.CreateHonor))
	router.Put("/api/v2/members/{id}/membership-adjustment", wrap(api.SetMembershipAdjust))

	var m openapi.Member
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 1, "firstName": "Ada", "lastName": "=Lovelace", "email": "ada@example.org", "birthday": "1950-05-10T00:00:00Z", "entryDate": "2000-05-01T00:00:00Z",
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

	const q = "/api/v2/jubilees?from=2025-01-01&to=2025-12-31&years=25&ages=75"
	rec := call(openapi.READONLY, http.MethodGet, q, "")
	if rec.Code != http.StatusOK || strings.Count(rec.Body.String(), `"kind"`) != 2 {
		t.Fatalf("jubilees: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.READONLY, http.MethodPost, "/api/v2/members/"+m.Id+"/honors", `{}`); rec.Code != http.StatusForbidden {
		t.Errorf("readonly create honor: %d, want 403", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/members/"+m.Id+"/honors", `{"kind":"membership","title":"x","date":"2025-05-01"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("missing milestone: %d, want 400", rec.Code)
	}

	rec = call(openapi.SECRETARY, http.MethodPost, "/api/v2/members/"+m.Id+"/honors",
		`{"kind":"membership","milestone":25,"title":"Ehrennadel Silber","date":"2025-05-01"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create honor: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.ADMIN, http.MethodGet, q, "")
	if strings.Count(rec.Body.String(), `"kind"`) != 1 || strings.Contains(rec.Body.String(), `"membership"`) {
		t.Errorf("honored stage still suggested: %s", rec.Body)
	}

	// A correction of -5 years moves the 25 year stage to 2030, so it leaves the range.
	if rec := call(openapi.ADMIN, http.MethodPut, "/api/v2/members/"+m.Id+"/membership-adjustment", `{"years":-5}`); rec.Code != http.StatusOK {
		t.Fatalf("adjust: %d", rec.Code)
	}
	rec = call(openapi.ADMIN, http.MethodGet, "/api/v2/jubilees?from=2025-01-01&to=2025-12-31&years=20&ages=75&format=csv", "")
	if !strings.Contains(rec.Body.String(), "'=Lovelace") || !strings.Contains(rec.Body.String(), "membership,20") {
		t.Errorf("csv: %s", rec.Body)
	}
	if rec := call(openapi.ADMIN, http.MethodGet, "/api/v2/jubilees?years=abc", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad years: %d, want 400", rec.Code)
	}
}
