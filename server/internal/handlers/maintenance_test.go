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

func TestInventoryMaintenance(t *testing.T) {
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
	router.Get("/api/v2/inventory/maintenance", wrap(api.ListMaintenanceStates))
	router.Put("/api/v2/inventory/items/{id}/status", wrap(api.SetItemStatus))
	router.Get("/api/v2/inventory/items/{id}/maintenance", wrap(api.ListItemMaintenance))
	router.Post("/api/v2/inventory/items/{id}/maintenance", wrap(api.RecordMaintenance))

	var m openapi.Member
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 1, "firstName": "Ole", "lastName": "Kite", "email": "ole@example.org",
	}, &m); code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}
	var item openapi.InventoryItem
	if code := doJSON(t, router, http.MethodPost, "/api/v2/inventory/items", map[string]any{
		"name": "Rettungsboot", "category": "Sicherheit", "quantity": 1, "location": "Slip",
	}, &item); code != http.StatusCreated {
		t.Fatalf("create item: %d", code)
	}
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
	base := "/api/v2/inventory/items/" + item.Id

	if rec := call(openapi.READONLY, http.MethodPut, base+"/status", `{"condition":"damaged"}`); rec.Code != http.StatusForbidden {
		t.Errorf("read-only may set status: %d", rec.Code)
	}
	if rec := call(openapi.SECRETARY, http.MethodPut, base+"/status", `{"condition":"broken"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown condition: %d, want 400", rec.Code)
	}
	due := time.Now().AddDate(0, 0, -3).Format(dateLayout)
	if rec := call(openapi.SECRETARY, http.MethodPut, base+"/status",
		`{"condition":"damaged","intervalMonths":12,"nextDue":"`+due+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("set status: %d %s", rec.Code, rec.Body)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v2/inventory/loans", strings.NewReader(
		`{"itemId":"`+item.Id+`","memberId":"`+m.Id+`","loanedAt":"2026-06-01T10:00:00Z","notes":""}`)))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "ITEM_NOT_READY") {
		t.Errorf("lend damaged item: %d %s", rec.Code, rec.Body)
	}

	rec = call(openapi.READONLY, http.MethodGet, "/api/v2/inventory/maintenance", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"due":"overdue"`) {
		t.Errorf("states: %d %s", rec.Code, rec.Body)
	}

	today := time.Now().Format(dateLayout)
	withExpense := `{"date":"` + today + `","cost":80,"condition":"ready","bookExpense":{"description":"Wartung Rettungsboot"}}`
	if rec := call(openapi.SECRETARY, http.MethodPost, base+"/maintenance", withExpense); rec.Code != http.StatusForbidden {
		t.Errorf("secretary may book expense: %d", rec.Code)
	}
	rec = call(openapi.TREASURER, http.MethodPost, base+"/maintenance", withExpense)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"transactionId"`) {
		t.Fatalf("record maintenance: %d %s", rec.Code, rec.Body)
	}
	rec = call(openapi.ADMIN, http.MethodGet, "/api/v2/inventory/maintenance", "")
	if !strings.Contains(rec.Body.String(), `"condition":"ready"`) || !strings.Contains(rec.Body.String(), `"due":"ok"`) {
		t.Errorf("after maintenance: %s", rec.Body)
	}
}
