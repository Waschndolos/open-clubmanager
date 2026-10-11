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

func TestExportMemberData(t *testing.T) {
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
	router.Get("/api/v2/members/{id}/data-export", func(w http.ResponseWriter, r *http.Request) {
		auth.Middleware(secret)(http.HandlerFunc(api.ExportMemberData)).
			ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), openapi.BearerAuthScopes, []string{})))
	})

	var m openapi.Member
	if code := doJSON(t, router, http.MethodPost, "/api/v2/members", map[string]any{
		"number": 7, "firstName": "Ada", "lastName": "Lovelace", "email": "ada@example.org", "iban": "DE02120300000000202051",
	}, &m); code != http.StatusCreated {
		t.Fatalf("create member: %d", code)
	}

	get := func(role openapi.AppRole, path string) *httptest.ResponseRecorder {
		token, err := auth.NewToken("sec@example.org", role, secret)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	if rec := get(openapi.READONLY, "/api/v2/members/"+m.Id+"/data-export"); rec.Code != http.StatusForbidden {
		t.Errorf("readonly: %d, want 403", rec.Code)
	}
	if rec := get(openapi.SECRETARY, "/api/v2/members/nope/data-export"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown member: %d, want 404", rec.Code)
	}

	rec := get(openapi.SECRETARY, "/api/v2/members/"+m.Id+"/data-export?format=json")
	if rec.Code != http.StatusOK {
		t.Fatalf("json: %d %s", rec.Code, rec.Body)
	}
	var report struct {
		Sections []struct {
			Key     string
			Records []map[string]any
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	found := map[string]int{}
	for _, s := range report.Sections {
		found[s.Key] = len(s.Records)
	}
	if found["masterData"] != 1 || found["bank"] != 1 || !strings.Contains(rec.Body.String(), "DE02120300000000202051") {
		t.Errorf("sections = %v", found)
	}
	if _, ok := found["fees"]; !ok {
		t.Errorf("fees section missing: %v", found)
	}

	rec = get(openapi.ADMIN, "/api/v2/members/"+m.Id+"/data-export")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "%PDF-") || !strings.Contains(rec.Body.String(), "Lovelace") {
		t.Errorf("pdf: %d, %.20q", rec.Code, rec.Body.String())
	}

	entries, err := api.store().ListAudit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if e.Entity == "MemberDataExport" && e.EntityID == m.Id && e.UserID == "sec@example.org" {
			n++
		}
	}
	if n != 2 {
		t.Errorf("audit entries for exports = %d, want 2", n)
	}
}
