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

// TestAppRoleWrites checks that the generated write endpoints honour the app
// role of the login (roleWrites), not only the device scopes.
func TestAppRoleWrites(t *testing.T) {
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	const secret = "test-secret"
	api := New(&config.Config{JWTSecret: secret}, conn)
	router := chi.NewRouter()
	// As in main: the generated routes run behind the JWT middleware.
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router,
		Middlewares: []openapi.MiddlewareFunc{api.ScopeGuard, auth.Middleware(secret)}})

	call := func(role openapi.AppRole, method, path, body string) *httptest.ResponseRecorder {
		token, err := auth.NewToken("user@example.org", role, secret)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	const missing = "00000000-0000-0000-0000-000000000000"
	member := `{"number":1,"firstName":"Pia","lastName":"Kite","email":"pia@example.org"}`
	data := []struct{ method, path, body string }{
		{http.MethodPost, "/api/v2/members", member},
		{http.MethodPut, "/api/v2/members/" + missing, member},
		{http.MethodDelete, "/api/v2/members/" + missing, ""},
		{http.MethodPost, "/api/v2/roles", `{"name":"Kassenprüfer"}`},
		{http.MethodPost, "/api/v2/groups", `{"name":"Jugend"}`},
		{http.MethodPost, "/api/v2/sections", `{"name":"Segeln"}`},
		{http.MethodDelete, "/api/v2/groups/" + missing, ""},
		{http.MethodPost, "/api/v2/inventory/items", `{"name":"Boot"}`},
		{http.MethodPost, "/api/v2/inventory/loans", `{}`},
		{http.MethodDelete, "/api/v2/inventory/items/" + missing, ""},
	}
	finance := []struct{ method, path, body string }{
		{http.MethodPost, "/api/v2/finance/transactions", `{"description":"Beitrag","amount":10,"date":"2026-10-13"}`},
		{http.MethodDelete, "/api/v2/finance/transactions/" + missing, ""},
		{http.MethodPost, "/api/v2/finance/memberfees", `{}`},
		{http.MethodPost, "/api/v2/finance/reset", ""},
	}

	// Trainers only record attendance; roles unknown to the rule stay
	// read-only as well.
	readOnly := []openapi.AppRole{openapi.READONLY, openapi.TRAINER, openapi.AppRole("GUEST")}
	for _, role := range readOnly {
		for _, c := range append(data, finance...) {
			if rec := call(role, c.method, c.path, c.body); rec.Code != http.StatusForbidden ||
				!strings.Contains(rec.Body.String(), `"FORBIDDEN"`) {
				t.Errorf("%s %s %s: %d %s, want 403", role, c.method, c.path, rec.Code, rec.Body)
			}
		}
	}
	for _, c := range finance {
		if rec := call(openapi.SECRETARY, c.method, c.path, c.body); rec.Code != http.StatusForbidden {
			t.Errorf("SECRETARY %s %s: %d, want 403", c.method, c.path, rec.Code)
		}
	}
	for _, role := range []openapi.AppRole{openapi.ADMIN, openapi.TREASURER, openapi.SECRETARY} {
		for _, c := range data {
			if rec := call(role, c.method, c.path, c.body); rec.Code == http.StatusForbidden {
				t.Errorf("%s %s %s: forbidden: %s", role, c.method, c.path, rec.Body)
			}
		}
	}
	for _, role := range []openapi.AppRole{openapi.ADMIN, openapi.TREASURER} {
		for _, c := range finance {
			if rec := call(role, c.method, c.path, c.body); rec.Code == http.StatusForbidden {
				t.Errorf("%s %s %s: forbidden: %s", role, c.method, c.path, rec.Body)
			}
		}
	}

	// Reading stays open to every role.
	for _, role := range readOnly {
		if rec := call(role, http.MethodGet, "/api/v2/members", ""); rec.Code != http.StatusOK {
			t.Errorf("%s may not list members: %d", role, rec.Code)
		}
	}
	if rec := call(openapi.READONLY, http.MethodGet, "/api/v2/members", ""); !strings.Contains(rec.Body.String(), "Pia") {
		t.Errorf("members created by data editors missing: %s", rec.Body)
	}
}
