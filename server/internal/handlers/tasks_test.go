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

func TestTasks(t *testing.T) {
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
	router.Get("/api/v2/tasks", wrap(api.ListTasks))
	router.Post("/api/v2/tasks", wrap(api.CreateTask))
	router.Put("/api/v2/tasks/{id}", wrap(api.UpdateTask))
	router.Delete("/api/v2/tasks/{id}", wrap(api.DeleteTask))

	var m openapi.Member
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 1, "firstName": "Ada", "lastName": "Lovelace", "email": "ada@example.org",
	}, &m); code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}
	call := func(email string, role openapi.AppRole, method, path, body string) *httptest.ResponseRecorder {
		token, err := auth.NewToken(email, role, secret)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	past := time.Now().AddDate(0, 0, -3).Format(dateLayout)
	rec := call("x@example.org", openapi.READONLY, http.MethodPost, "/api/v2/tasks", `{"title":"x"}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("readonly create: %d, want 403", rec.Code)
	}
	if rec := call("x@example.org", openapi.TREASURER, http.MethodPost, "/api/v2/tasks", `{"title":"x","assigneeMemberId":"`+m.Id+`","assigneeRoleId":"r"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("member and role: %d, want 400", rec.Code)
	}
	rec = call("x@example.org", openapi.TREASURER, http.MethodPost, "/api/v2/tasks",
		`{"title":"Jahresmeldung LSB","dueDate":"`+past+`","recurrence":"yearly","assigneeMemberId":"`+m.Id+`"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"overdue":true`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	id := between(rec.Body.String(), `"id":"`, `"`)

	if rec := call("ADA@example.org", openapi.READONLY, http.MethodGet, "/api/v2/tasks?filter=mine", ""); !strings.Contains(rec.Body.String(), "Jahresmeldung") {
		t.Errorf("mine for the assignee: %s", rec.Body)
	}
	if rec := call("other@example.org", openapi.READONLY, http.MethodGet, "/api/v2/tasks?filter=mine", ""); strings.Contains(rec.Body.String(), "Jahresmeldung") {
		t.Errorf("mine for someone else: %s", rec.Body)
	}
	if rec := call("x@example.org", openapi.READONLY, http.MethodGet, "/api/v2/tasks?filter=bogus", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad filter: %d", rec.Code)
	}

	// Finishing the yearly task creates next year's occurrence.
	rec = call("x@example.org", openapi.TREASURER, http.MethodPut, "/api/v2/tasks/"+id,
		`{"title":"Jahresmeldung LSB","dueDate":"`+past+`","recurrence":"yearly","status":"done","assigneeMemberId":"`+m.Id+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("finish: %d %s", rec.Code, rec.Body)
	}
	all := call("x@example.org", openapi.READONLY, http.MethodGet, "/api/v2/tasks", "").Body.String()
	if strings.Count(all, `"title"`) != 2 || !strings.Contains(all, `"status":"open"`) || strings.Contains(all, `"overdue":true`) {
		t.Errorf("after finish: %s", all)
	}
	if rec := call("x@example.org", openapi.TREASURER, http.MethodDelete, "/api/v2/tasks/"+id, ""); rec.Code != http.StatusNoContent {
		t.Errorf("delete: %d", rec.Code)
	}
	if rec := call("x@example.org", openapi.TREASURER, http.MethodDelete, "/api/v2/tasks/"+id, ""); rec.Code != http.StatusNotFound {
		t.Errorf("delete twice: %d, want 404", rec.Code)
	}
}
