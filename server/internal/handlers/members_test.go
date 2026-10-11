package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// newTestRouter serves the generated routes (without JWT middleware) on top of
// a fresh database.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	router := chi.NewRouter()
	openapi.HandlerWithOptions(New(&config.Config{}, conn), openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router})
	return router
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, &buf))
	if out != nil && rec.Code < 300 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec.Code
}

func TestMembersEndpoints(t *testing.T) {
	h := newTestRouter(t)

	var role openapi.NamedEntity
	if code := doJSON(t, h, http.MethodPost, "/api/v2/roles", map[string]string{"name": "Kassenwart"}, &role); code != http.StatusCreated {
		t.Fatalf("create role: %d", code)
	}

	var created openapi.Member
	code := doJSON(t, h, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 42, "firstName": "Grace", "lastName": "Hopper", "email": "grace@example.org",
		"iban": "DE02120300000000202051", "roleIds": []string{role.Id},
	}, &created)
	if code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}
	if created.Id == "" || created.Iban == nil || *created.Iban != "DE02120300000000202051" || len(created.Roles) != 1 {
		t.Errorf("created = %+v", created)
	}

	if code := doJSON(t, h, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 42, "firstName": "Dup", "lastName": "Dup", "email": "dup@example.org",
	}, nil); code != http.StatusConflict {
		t.Errorf("duplicate number: %d, want 409", code)
	}

	var list openapi.MembersListResponse
	if code := doJSON(t, h, http.MethodGet, "/api/v2/members?search=grace", nil, &list); code != http.StatusOK {
		t.Fatalf("list: %d", code)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].Id != created.Id || string(list.Items[0].Email) != "grace@example.org" {
		t.Errorf("list = %+v", list)
	}

	var got openapi.Member
	if code := doJSON(t, h, http.MethodGet, "/api/v2/members/"+created.Id, nil, &got); code != http.StatusOK || got.LastName != "Hopper" {
		t.Errorf("get: %d %+v", code, got)
	}

	if code := doJSON(t, h, http.MethodDelete, "/api/v2/members/"+created.Id, nil, nil); code != http.StatusOK {
		t.Errorf("delete: %d", code)
	}
	if code := doJSON(t, h, http.MethodGet, "/api/v2/members/"+created.Id, nil, nil); code != http.StatusNotFound {
		t.Errorf("get after delete: %d, want 404", code)
	}
}

// SQLite no longer enforces these with UNIQUE indexes (docs/sync-protocol.md
// §14.2); the store's pre-save checks must keep the API responses unchanged.
func TestUniquenessConflictsReturn409(t *testing.T) {
	h := newTestRouter(t)

	conflict := func(method, path string, body any) {
		t.Helper()
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, &buf))
		var resp openapi.ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if rec.Code != http.StatusConflict || resp.Error != "a record with the same unique value already exists" {
			t.Errorf("%s %s: %d %q, want 409 with the uniqueness message", method, path, rec.Code, rec.Body.String())
		}
	}

	member := func(number int, email string) map[string]any {
		return map[string]any{"number": number, "firstName": "F", "lastName": "L", "email": email}
	}
	var a, b openapi.Member
	if code := doJSON(t, h, http.MethodPost, "/api/v2/members", member(1, "a@example.org"), &a); code != http.StatusCreated {
		t.Fatalf("create a: %d", code)
	}
	if code := doJSON(t, h, http.MethodPost, "/api/v2/members", member(2, "b@example.org"), &b); code != http.StatusCreated {
		t.Fatalf("create b: %d", code)
	}
	conflict(http.MethodPost, "/api/v2/members", member(1, "c@example.org"))
	conflict(http.MethodPost, "/api/v2/members", member(3, "a@example.org"))
	conflict(http.MethodPut, "/api/v2/members/"+b.Id, member(1, "b@example.org"))
	conflict(http.MethodPut, "/api/v2/members/"+b.Id, member(2, "a@example.org"))
	if code := doJSON(t, h, http.MethodPut, "/api/v2/members/"+b.Id, member(2, "b@example.org"), nil); code != http.StatusOK {
		t.Errorf("update keeping own values: %d", code)
	}

	for _, res := range []string{"roles", "groups", "sections"} {
		var x, y openapi.NamedEntity
		doJSON(t, h, http.MethodPost, "/api/v2/"+res, map[string]string{"name": "X"}, &x)
		if code := doJSON(t, h, http.MethodPost, "/api/v2/"+res, map[string]string{"name": "Y"}, &y); code != http.StatusCreated {
			t.Fatalf("create %s: %d", res, code)
		}
		conflict(http.MethodPost, "/api/v2/"+res, map[string]string{"name": "X"})
		conflict(http.MethodPut, "/api/v2/"+res+"/"+y.Id, map[string]string{"name": "X"})
	}
}
