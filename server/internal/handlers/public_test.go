package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

const publicTestSecret = "test-secret"

// newPublicTestRouter wires the generated routes as in main, for a server
// whose setup is complete unless adminPassword is empty.
func newPublicTestRouter(t *testing.T, adminPassword string) (*API, chi.Router) {
	t.Helper()
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	api := New(&config.Config{
		JWTSecret:     publicTestSecret,
		AdminEmail:    "admin@example.org",
		AdminPassword: adminPassword,
		DatabasePath:  filepath.Join(t.TempDir(), "club.db"),
	}, conn)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router,
		Middlewares: []openapi.MiddlewareFunc{api.ScopeGuard, auth.MiddlewareAllowing(publicTestSecret, api.AllowAnonymous)}})
	router.Post("/api/v2/system/backup", auth.Middleware(publicTestSecret)(http.HandlerFunc(api.BackupDatabase)).ServeHTTP)
	router.Post("/api/v2/system/restore", auth.Middleware(publicTestSecret)(http.HandlerFunc(api.RestoreDatabase)).ServeHTTP)
	return api, router
}

func serve(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func tokenFor(t *testing.T, role openapi.AppRole) string {
	t.Helper()
	token, err := auth.NewToken("user@example.org", role, publicTestSecret)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// TestRoutesRequireToken checks that every route refuses callers without a
// token, apart from the explicitly public ones (#179).
func TestRoutesRequireToken(t *testing.T) {
	_, router := newPublicTestRouter(t, "$2a$10$configured")
	param := regexp.MustCompile(`\{[^}]+\}`)

	checked := 0
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if publicOperations[method+" "+route] {
			return nil
		}
		path := param.ReplaceAllStringFunc(route, func(p string) string {
			if p == "{date}" {
				return "2026-01-01"
			}
			return "00000000-0000-0000-0000-000000000000"
		})
		if rec := serve(router, method, path, `{}`, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: status %d, want 401", method, route, rec.Code)
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 90 {
		t.Fatalf("only %d routes checked, the walk looks broken", checked)
	}

	if rec := serve(router, http.MethodGet, "/api/v2/system/health", "", ""); rec.Code != http.StatusOK {
		t.Errorf("health without token: status %d, want 200", rec.Code)
	}
	if rec := serve(router, http.MethodGet, "/api/v2/members", "", tokenFor(t, openapi.READONLY)); rec.Code != http.StatusOK {
		t.Errorf("members with token: status %d, want 200", rec.Code)
	}
}

// TestSetupRunsOnce checks that the first-run setup cannot replace the admin
// once it exists (#180).
func TestSetupRunsOnce(t *testing.T) {
	api, router := newPublicTestRouter(t, "")
	admin := `{"email":"admin@example.org","password":"first-password"}`

	if rec := serve(router, http.MethodGet, "/api/v2/settings/database", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("database settings during setup: status %d, want 200", rec.Code)
	}
	if rec := serve(router, http.MethodPost, "/api/v2/setup/initialize", admin, ""); rec.Code != http.StatusCreated {
		t.Fatalf("first initialize: status %d, want 201: %s", rec.Code, rec.Body)
	}
	hash := api.cfg.AdminPassword

	takeover := `{"email":"attacker@example.org","password":"second-password"}`
	if rec := serve(router, http.MethodPost, "/api/v2/setup/initialize", takeover, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous initialize after setup: status %d, want 401", rec.Code)
	}
	if rec := serve(router, http.MethodPost, "/api/v2/setup/initialize", takeover, tokenFor(t, openapi.ADMIN)); rec.Code != http.StatusConflict {
		t.Errorf("initialize after setup: status %d, want 409", rec.Code)
	}
	if api.cfg.AdminEmail != "admin@example.org" || api.cfg.AdminPassword != hash {
		t.Fatal("admin credentials were replaced")
	}

	dbReq := `{"mode":"sqlite-local","databaseUrl":"` + filepath.Join(t.TempDir(), "other.db") + `"}`
	if rec := serve(router, http.MethodPost, "/api/v2/setup/configure-database", dbReq, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous configure-database after setup: status %d, want 401", rec.Code)
	}
	if rec := serve(router, http.MethodPost, "/api/v2/setup/configure-database", dbReq, tokenFor(t, openapi.READONLY)); rec.Code != http.StatusForbidden {
		t.Errorf("configure-database as READONLY: status %d, want 403", rec.Code)
	}
	if rec := serve(router, http.MethodPost, "/api/v2/settings/database", dbReq, tokenFor(t, openapi.SECRETARY)); rec.Code != http.StatusForbidden {
		t.Errorf("database settings as SECRETARY: status %d, want 403", rec.Code)
	}
	if rec := serve(router, http.MethodGet, "/api/v2/setup/status", "", ""); !strings.Contains(rec.Body.String(), `"setupRequired":false`) {
		t.Errorf("setup status after setup: %s", rec.Body)
	}
}

// TestBackupRestoreNeedAdmin checks that only admins may download or replace
// the database (#181).
func TestBackupRestoreNeedAdmin(t *testing.T) {
	_, router := newPublicTestRouter(t, "$2a$10$configured")

	for _, role := range []openapi.AppRole{openapi.READONLY, openapi.SECRETARY, openapi.TREASURER} {
		for _, path := range []string{"/api/v2/system/backup", "/api/v2/system/restore"} {
			if rec := serve(router, http.MethodPost, path, "", tokenFor(t, role)); rec.Code != http.StatusForbidden {
				t.Errorf("%s as %s: status %d, want 403", path, role, rec.Code)
			}
		}
	}
	rec := serve(router, http.MethodPost, "/api/v2/system/backup", "", tokenFor(t, openapi.ADMIN))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" {
		t.Errorf("backup as ADMIN: status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}
