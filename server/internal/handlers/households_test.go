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

func TestHouseholdsAndGuardians(t *testing.T) {
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
	router.Get("/api/v2/households", wrap(api.ListHouseholds))
	router.Post("/api/v2/households", wrap(api.CreateHousehold))
	router.Put("/api/v2/households/{id}", wrap(api.UpdateHousehold))
	router.Get("/api/v2/members/{id}/guardians", wrap(api.ListGuardians))
	router.Post("/api/v2/members/{id}/guardians", wrap(api.CreateGuardian))

	ids := make([]string, 3)
	for i, name := range []string{"Pia", "Ole", "Max"} {
		var m openapi.Member
		if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{
			"number": i + 1, "firstName": name, "lastName": "Kite", "email": strings.ToLower(name) + "@example.org",
		}, &m); code != http.StatusCreated {
			t.Fatalf("create member: %d", code)
		}
		ids[i] = m.Id
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

	family := `{"name":"Familie Kite","payerId":"` + ids[0] + `","shareAddress":true,"memberIds":["` + ids[0] + `","` + ids[1] + `"]}`
	if rec := call(openapi.READONLY, http.MethodPost, "/api/v2/households", family); rec.Code != http.StatusForbidden {
		t.Errorf("read-only may create: %d", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/households",
		`{"name":"X","payerId":"`+ids[2]+`","memberIds":["`+ids[0]+`"]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("payer outside household: %d, want 400", rec.Code)
	}
	rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/households", family)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"payerMissing":false`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/households",
		`{"name":"Andere","memberIds":["`+ids[2]+`","`+ids[1]+`"]}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "MEMBER_IN_OTHER_HOUSEHOLD") {
		t.Errorf("member in two households: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.READONLY, http.MethodGet, "/api/v2/households", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Familie Kite") {
		t.Errorf("list: %d %s", rec.Code, rec.Body)
	}

	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/members/"+ids[1]+"/guardians", `{"relation":"Mutter"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("guardian without name: %d, want 400", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/members/"+ids[1]+"/guardians",
		`{"memberId":"`+ids[0]+`","relation":"Mutter"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create guardian: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.ADMIN, http.MethodGet, "/api/v2/members/"+ids[1]+"/guardians", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"firstName":"Pia"`) {
		t.Errorf("list guardians: %d %s", rec.Code, rec.Body)
	}
}
