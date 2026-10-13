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

func TestEvents(t *testing.T) {
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
		req.Header.Set("Accept-Language", "de-DE,de;q=0.9")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec := call(openapi.ADMIN, http.MethodPost, "/api/v2/groups", `{"name":"Jugend"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: %d %s", rec.Code, rec.Body)
	}
	groupID := between(rec.Body.String(), `"id":"`, `"`)

	training := `{"title":"Jugendtraining","category":"training","start":"2026-10-13T16:00:00Z","end":"2026-10-13T17:30:00Z",
		"timeZone":"Europe/Berlin","recurrence":"weekly","recurrenceUntil":"2026-11-03","location":"Strand, Nord","groupIds":["` + groupID + `"]}`
	if rec := call(openapi.READONLY, http.MethodPost, "/api/v2/events", training); rec.Code != http.StatusForbidden {
		t.Errorf("read-only may create: %d", rec.Code)
	}
	for _, bad := range []string{
		`{"title":"","category":"training","start":"2026-10-13T16:00:00Z","end":"2026-10-13T17:00:00Z"}`,
		`{"title":"X","category":"party","start":"2026-10-13T16:00:00Z","end":"2026-10-13T17:00:00Z"}`,
		`{"title":"X","category":"other","start":"2026-10-13T16:00:00Z","end":"2026-10-13T15:00:00Z"}`,
		`{"title":"X","category":"other","start":"2026-10-13T16:00:00Z","end":"2026-10-13T17:00:00Z","timeZone":"Mars/Olympus"}`,
	} {
		if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/events", bad); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", bad, rec.Code)
		}
	}
	rec = call(openapi.SECRETARY, http.MethodPost, "/api/v2/events", training)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created openapi.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || len(created.Groups) != 1 {
		t.Fatalf("created event: %v %s", err, rec.Body)
	}
	id := created.Id
	if rec := call(openapi.SECRETARY, http.MethodPost, "/api/v2/events",
		`{"title":"Sommerfest","category":"festival","start":"2026-10-24T00:00:00+02:00","end":"2026-10-25T00:00:00+02:00","allDay":true,"timeZone":"Europe/Berlin"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create all-day: %d %s", rec.Code, rec.Body)
	}

	if rec := call(openapi.TREASURER, http.MethodPost, "/api/v2/events/"+id+"/exceptions", `{"date":"2026-10-20"}`); rec.Code != http.StatusOK {
		t.Fatalf("exception: %d %s", rec.Code, rec.Body)
	}

	rec = call(openapi.READONLY, http.MethodGet, "/api/v2/events?from=2026-10-01&to=2026-11-30", "")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Count(body, `"eventId"`) != 4 || strings.Contains(body, "2026-10-20") {
		t.Errorf("list: %d %s", rec.Code, body)
	}
	// Club-wide events match every group filter; group events only their group.
	rec = call(openapi.READONLY, http.MethodGet, "/api/v2/events?from=2026-10-01&to=2026-11-30&groupId=other", "")
	if strings.Count(rec.Body.String(), `"eventId"`) != 1 {
		t.Errorf("other group: %s", rec.Body)
	}
	if rec := call(openapi.READONLY, http.MethodGet, "/api/v2/events?from=2026-01-01&to=2027-12-31", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("range too long: %d", rec.Code)
	}

	rec = call(openapi.READONLY, http.MethodGet, "/api/v2/events/"+id+"/ics?date=2026-10-27", "")
	ics := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/calendar") ||
		strings.Count(ics, "BEGIN:VEVENT") != 1 || !strings.Contains(ics, "DTSTART:20261027T170000Z") ||
		!strings.Contains(ics, `LOCATION:Strand\, Nord`) || !strings.Contains(ics, "CATEGORIES:Training") ||
		!strings.Contains(rec.Header().Get("Content-Disposition"), "jugendtraining.ics") {
		t.Errorf("event ics: %d %v\n%s", rec.Code, rec.Header(), ics)
	}
	if rec := call(openapi.READONLY, http.MethodGet, "/api/v2/events/"+id+"/ics?date=2026-10-20", ""); rec.Code != http.StatusNotFound {
		t.Errorf("ics of excluded occurrence: %d", rec.Code)
	}
	rec = call(openapi.READONLY, http.MethodGet, "/api/v2/calendar.ics?from=2026-10-01&to=2026-12-31", "")
	ics = rec.Body.String()
	if rec.Code != http.StatusOK || strings.Count(ics, "BEGIN:VEVENT") != 4 || !strings.Contains(ics, "DTSTART;VALUE=DATE:20261024") ||
		!strings.Contains(ics, "CATEGORIES:Fest") {
		t.Errorf("calendar ics: %d\n%s", rec.Code, ics)
	}

	if rec := call(openapi.SECRETARY, http.MethodDelete, "/api/v2/events/"+id, ""); rec.Code != http.StatusNoContent {
		t.Errorf("delete: %d", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodGet, "/api/v2/events/"+id, ""); rec.Code != http.StatusNotFound {
		t.Errorf("get deleted: %d", rec.Code)
	}
}
