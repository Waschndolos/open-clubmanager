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

// The user management enforces the password policy of the setup and refuses
// e-mail addresses of the administrator or of another user (#192).
func TestAppUserValidation(t *testing.T) {
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	const secret = "test-secret"
	api := New(&config.Config{JWTSecret: secret, AdminEmail: "Admin@Example.org"}, conn)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router,
		Middlewares: []openapi.MiddlewareFunc{auth.Middleware(secret)}})

	token, err := auth.NewToken("Admin@Example.org", openapi.ADMIN, secret)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body string) (int, openapi.ErrorResponse, openapi.AppUser) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var e openapi.ErrorResponse
		var u openapi.AppUser
		if rec.Code >= 300 {
			_ = json.Unmarshal(rec.Body.Bytes(), &e)
		} else {
			_ = json.Unmarshal(rec.Body.Bytes(), &u)
		}
		return rec.Code, e, u
	}
	code := func(e openapi.ErrorResponse) string {
		if e.Code == nil {
			return ""
		}
		return *e.Code
	}

	cases := []struct {
		name, body string
		status     int
		code       string
	}{
		{"short password", `{"email":"weak@example.org","password":"a","appRole":"READONLY"}`, http.StatusBadRequest, "PASSWORD_TOO_SHORT"},
		{"7 characters", `{"email":"weak@example.org","password":"1234567","appRole":"READONLY"}`, http.StatusBadRequest, "PASSWORD_TOO_SHORT"},
		{"admin e-mail", `{"email":"admin@example.org","password":"Testpass123!","appRole":"READONLY"}`, http.StatusConflict, "EMAIL_IN_USE"},
		{"valid", `{"email":"pia@example.org","password":"Testpass123!","appRole":"READONLY"}`, http.StatusCreated, ""},
		{"same e-mail in other case", `{"email":"PIA@example.org","password":"Testpass123!","appRole":"SECRETARY"}`, http.StatusConflict, "EMAIL_IN_USE"},
	}
	for _, c := range cases {
		status, e, _ := call(http.MethodPost, "/api/v2/users", c.body)
		if status != c.status || code(e) != c.code {
			t.Errorf("%s: %d %q, want %d %q", c.name, status, code(e), c.status, c.code)
		}
	}

	status, _, other := call(http.MethodPost, "/api/v2/users", `{"email":"tom@example.org","password":"Testpass123!","appRole":"READONLY"}`)
	if status != http.StatusCreated {
		t.Fatalf("create tom: %d", status)
	}
	path := "/api/v2/users/" + other.Id
	for _, email := range []string{"admin@EXAMPLE.org", "Pia@example.org"} {
		if status, e, _ := call(http.MethodPut, path, `{"email":"`+email+`"}`); status != http.StatusConflict || code(e) != "EMAIL_IN_USE" {
			t.Errorf("rename to %s: %d %q, want 409 EMAIL_IN_USE", email, status, code(e))
		}
	}
	// Keeping the own address (in another case) and changing the role is fine.
	if status, _, u := call(http.MethodPut, path, `{"email":"Tom@example.org","appRole":"TREASURER"}`); status != http.StatusOK || u.AppRole != openapi.TREASURER {
		t.Errorf("update own e-mail case and role: %d %+v", status, u)
	}
}

// The setup applies the same password policy.
func TestPasswordPolicyOnSetup(t *testing.T) {
	_, router := newPublicTestRouter(t, "")
	body := map[string]any{"email": "admin@example.org", "password": "short"}
	if code := doJSON(t, router, http.MethodPost, "/api/v2/setup/initialize", body, nil); code != http.StatusBadRequest {
		t.Errorf("setup with a short password: %d, want 400", code)
	}
}
