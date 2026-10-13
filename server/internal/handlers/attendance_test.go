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

func TestAttendanceAccess(t *testing.T) {
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	const secret = "test-secret"
	api := New(&config.Config{JWTSecret: secret}, conn)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router,
		Middlewares: []openapi.MiddlewareFunc{api.ScopeGuard, auth.Middleware(secret)}})
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			auth.Middleware(secret)(h).ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), openapi.BearerAuthScopes, []string{})))
		}
	}
	router.Get("/api/v2/attendance/occurrences", wrap(api.ListAttendanceOccurrences))
	router.Get("/api/v2/attendance/sheet", wrap(api.GetAttendanceSheet))
	router.Put("/api/v2/attendance/sheet", wrap(api.SetAttendance))
	router.Get("/api/v2/attendance/stats", wrap(api.AttendanceStats))
	router.Get("/api/v2/attendance/export", wrap(api.ExportAttendance))
	router.Post("/api/v2/tasks", wrap(api.CreateTask))

	call := func(email string, role openapi.AppRole, method, path, body string) *httptest.ResponseRecorder {
		token, err := auth.NewToken(email, role, secret)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept-Language", "de")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	admin := func(method, path, body string) string {
		rec := call("admin@example.org", openapi.ADMIN, method, path, body)
		if rec.Code >= 300 {
			t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body)
		}
		return rec.Body.String()
	}
	id := func(body string) string {
		var v struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(body), &v)
		return v.ID
	}

	youth := id(admin(http.MethodPost, "/api/v2/groups", `{"name":"Jugend"}`))
	adults := id(admin(http.MethodPost, "/api/v2/groups", `{"name":"Erwachsene"}`))
	// The trainer login is linked to member 1 (same e-mail), who is in the youth group.
	admin(http.MethodPost, "/api/v2/members", `{"number":1,"firstName":"Coach","lastName":"Kite","email":"coach@example.org","groupIds":["`+youth+`"]}`)
	ida := id(admin(http.MethodPost, "/api/v2/members", `{"number":2,"firstName":"Ida","lastName":"Kite","email":"ida@example.org","groupIds":["`+youth+`"]}`))
	event := func(title, group string) string {
		audience := ""
		if group != "" {
			audience = `,"groupIds":["` + group + `"]`
		}
		return id(admin(http.MethodPost, "/api/v2/events", `{"title":"`+title+`","category":"training","start":"2026-10-13T15:30:00Z",
			"end":"2026-10-13T17:00:00Z","timeZone":"Europe/Berlin","recurrence":"weekly"`+audience+`}`))
	}
	youthTraining, adultTraining, clubEvent := event("Jugendtraining", youth), event("Erwachsene", adults), event("Vereinsabend", "")

	trainer := func(method, path, body string) *httptest.ResponseRecorder {
		return call("coach@example.org", openapi.TRAINER, method, path, body)
	}
	set := func(eventID string) string {
		return `{"eventId":"` + eventID + `","date":"2026-10-13","entries":[{"memberId":"` + ida + `","status":"present"}]}`
	}

	if rec := trainer(http.MethodPut, "/api/v2/attendance/sheet", set(youthTraining)); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"present":1`) {
		t.Errorf("trainer records own group: %d %s", rec.Code, rec.Body)
	}
	for _, other := range []string{adultTraining, clubEvent} {
		if rec := trainer(http.MethodPut, "/api/v2/attendance/sheet", set(other)); rec.Code != http.StatusForbidden {
			t.Errorf("trainer records other event: %d", rec.Code)
		}
		if rec := trainer(http.MethodGet, "/api/v2/attendance/sheet?eventId="+other+"&date=2026-10-13", ""); rec.Code != http.StatusForbidden {
			t.Errorf("trainer sees other event: %d", rec.Code)
		}
	}
	rec := trainer(http.MethodGet, "/api/v2/attendance/occurrences?from=2026-10-13&to=2026-10-13", "")
	if rec.Code != http.StatusOK || strings.Count(rec.Body.String(), `"eventId"`) != 1 || !strings.Contains(rec.Body.String(), youthTraining) {
		t.Errorf("trainer occurrences: %d %s", rec.Code, rec.Body)
	}
	// Trainers are read-only elsewhere.
	if rec := trainer(http.MethodPost, "/api/v2/tasks", `{"title":"X","status":"open","recurrence":"none"}`); rec.Code != http.StatusForbidden {
		t.Errorf("trainer creates task: %d", rec.Code)
	}

	if rec := call("ro@example.org", openapi.READONLY, http.MethodPut, "/api/v2/attendance/sheet", set(adultTraining)); rec.Code != http.StatusForbidden {
		t.Errorf("read-only records: %d", rec.Code)
	}
	if rec := call("ro@example.org", openapi.READONLY, http.MethodGet, "/api/v2/attendance/sheet?eventId="+adultTraining+"&date=2026-10-13", ""); rec.Code != http.StatusOK {
		t.Errorf("read-only reads: %d", rec.Code)
	}
	if rec := call("sec@example.org", openapi.SECRETARY, http.MethodPut, "/api/v2/attendance/sheet",
		`{"eventId":"`+adultTraining+`","date":"2026-10-13","entries":[{"memberId":"`+ida+`","status":"late"}]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown status: %d", rec.Code)
	}
	admin(http.MethodPut, "/api/v2/attendance/sheet", set(adultTraining))

	// The trainer's statistics and exports only contain the own group.
	rec = trainer(http.MethodGet, "/api/v2/attendance/stats?from=2026-10-01&to=2026-10-31", "")
	if !strings.Contains(rec.Body.String(), `"present":1`) || strings.Count(rec.Body.String(), `"eventId"`) != 1 {
		t.Errorf("trainer stats: %s", rec.Body)
	}
	rec = trainer(http.MethodGet, "/api/v2/attendance/export?from=2026-10-01&to=2026-10-31&format=csv", "")
	csv := rec.Body.String()
	if rec.Code != http.StatusOK || strings.Count(csv, "\n") != 2 || !strings.Contains(csv, "2026-10-13,17:30,Jugendtraining,2,Ida,Kite,present") {
		t.Errorf("trainer csv: %d %q", rec.Code, csv)
	}
	rec = call("admin@example.org", openapi.ADMIN, http.MethodGet, "/api/v2/attendance/export?from=2026-10-01&to=2026-10-31&format=pdf", "")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "%PDF-1.4") || !strings.Contains(rec.Body.String(), "Anwesenheitsliste") {
		t.Errorf("pdf: %d %.80q", rec.Code, rec.Body.String())
	}
	if rec := call("admin@example.org", openapi.ADMIN, http.MethodGet, "/api/v2/attendance/export?from=2026-10-01&to=2026-10-31&format=xls", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown format: %d", rec.Code)
	}
}
