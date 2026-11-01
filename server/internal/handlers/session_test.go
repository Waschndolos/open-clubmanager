package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

func newSessionTestAPI(t *testing.T) (*API, http.Handler) {
	t.Helper()
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))

	hash, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	const secret = "test-secret"
	api := New(&config.Config{JWTSecret: secret, AdminEmail: "admin@example.org", AdminPassword: string(hash)}, conn)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router,
		Middlewares: []openapi.MiddlewareFunc{api.ScopeGuard, auth.Middleware(secret, api.ValidateSession)}})
	return api, router
}

func sessionCall(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func loginToken(t *testing.T, h http.Handler, email, password string) string {
	t.Helper()
	rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "", `{"email":"`+email+`","password":"`+password+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
	}
	var out openapi.AccessTokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.AccessToken
}

// TestSessionRevocation checks that tokens stop working as soon as the user
// is deleted, gets another role or changes the password.
func TestSessionRevocation(t *testing.T) {
	api, h := newSessionTestAPI(t)
	admin := loginToken(t, h, "admin@example.org", "admin-password")

	createUser := func(email string) string {
		rec := sessionCall(t, h, http.MethodPost, "/api/v2/users", admin,
			`{"email":"`+email+`","password":"user-password","appRole":"SECRETARY"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create user: %d %s", rec.Code, rec.Body)
		}
		var u openapi.AppUser
		if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
			t.Fatal(err)
		}
		return u.Id
	}
	profile := func(token string) int {
		return sessionCall(t, h, http.MethodGet, "/api/v2/auth/profile", token, "").Code
	}
	// Tokens carry whole seconds; wait so that a change happens in a later
	// second than the token was issued in.
	nextSecond := func() { time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second))) }

	t.Run("deleted user", func(t *testing.T) {
		id := createUser("deleted@example.org")
		token := loginToken(t, h, "deleted@example.org", "user-password")
		if code := profile(token); code != http.StatusOK {
			t.Fatalf("before delete: %d", code)
		}
		if rec := sessionCall(t, h, http.MethodDelete, "/api/v2/users/"+id, admin, ""); rec.Code != http.StatusNoContent {
			t.Fatalf("delete: %d %s", rec.Code, rec.Body)
		}
		if code := profile(token); code != http.StatusUnauthorized {
			t.Fatalf("after delete: %d, want 401", code)
		}
	})

	t.Run("role change", func(t *testing.T) {
		id := createUser("demoted@example.org")
		token := loginToken(t, h, "demoted@example.org", "user-password")
		if rec := sessionCall(t, h, http.MethodPut, "/api/v2/users/"+id, admin, `{"appRole":"READONLY"}`); rec.Code != http.StatusOK {
			t.Fatalf("update: %d %s", rec.Code, rec.Body)
		}
		if code := profile(token); code != http.StatusUnauthorized {
			t.Fatalf("after role change: %d, want 401", code)
		}
		if code := profile(loginToken(t, h, "demoted@example.org", "user-password")); code != http.StatusOK {
			t.Fatalf("new login: %d", code)
		}
	})

	t.Run("password change", func(t *testing.T) {
		createUser("rotated@example.org")
		token := loginToken(t, h, "rotated@example.org", "user-password")
		nextSecond()
		if rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/change-password", token,
			`{"currentPassword":"user-password","newPassword":"new-password"}`); rec.Code != http.StatusOK {
			t.Fatalf("change password: %d %s", rec.Code, rec.Body)
		}
		if code := profile(token); code != http.StatusUnauthorized {
			t.Fatalf("after password change: %d, want 401", code)
		}
		if code := profile(loginToken(t, h, "rotated@example.org", "new-password")); code != http.StatusOK {
			t.Fatalf("new login: %d", code)
		}
	})

	t.Run("forged admin role", func(t *testing.T) {
		createUser("forged@example.org")
		token, err := auth.NewToken("forged@example.org", openapi.ADMIN, "test-secret")
		if err != nil {
			t.Fatal(err)
		}
		if code := profile(token); code != http.StatusUnauthorized {
			t.Fatalf("role mismatch: %d, want 401", code)
		}
	})

	// Last: revokes the admin token the other subtests use.
	t.Run("admin password change", func(t *testing.T) {
		old := loginToken(t, h, "admin@example.org", "admin-password")
		nextSecond()
		if rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/change-password", old,
			`{"newPassword":"new-admin-password"}`); rec.Code != http.StatusOK {
			t.Fatalf("change password: %d %s", rec.Code, rec.Body)
		}
		if code := profile(old); code != http.StatusUnauthorized {
			t.Fatalf("after password change: %d, want 401", code)
		}
		if code := profile(loginToken(t, h, "admin@example.org", "new-admin-password")); code != http.StatusOK {
			t.Fatalf("new login: %d", code)
		}
		if api.cfg.AdminTokensValidAfter == 0 {
			t.Fatal("cutoff not recorded")
		}
	})
}

// TestLoginThrottling checks the lockout after repeated failed logins.
func TestLoginThrottling(t *testing.T) {
	api, h := newSessionTestAPI(t)
	now := time.Now()
	api.logins.now = func() time.Time { return now }

	attempt := func(password string) *httptest.ResponseRecorder {
		return sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "",
			`{"email":"admin@example.org","password":"`+password+`"}`)
	}
	for i := 0; i < loginAccountThreshold; i++ {
		if rec := attempt("wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d, want 401", i+1, rec.Code)
		}
	}
	// Locked: even the correct password is refused.
	rec := attempt("admin-password")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "30" {
		t.Fatalf("locked: %d Retry-After=%q, want 429 / 30", rec.Code, rec.Header().Get("Retry-After"))
	}

	// The lockout doubles with every further failure.
	now = now.Add(31 * time.Second)
	if rec := attempt("wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after lockout: %d, want 401", rec.Code)
	}
	if rec := attempt("admin-password"); rec.Header().Get("Retry-After") != "60" {
		t.Fatalf("second lockout: Retry-After=%q, want 60", rec.Header().Get("Retry-After"))
	}

	// A successful login clears the account's failures.
	now = now.Add(61 * time.Second)
	if rec := attempt("admin-password"); rec.Code != http.StatusOK {
		t.Fatalf("after lockout: %d, want 200", rec.Code)
	}
	if rec := attempt("wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after reset: %d, want 401", rec.Code)
	}

	// Failures from one source count across accounts.
	for i := 0; i < loginSourceThreshold; i++ {
		sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "", `{"email":"user`+string(rune('a'+i))+`@example.org","password":"x"}`)
	}
	if rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "", `{"email":"new@example.org","password":"x"}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("source lockout: %d, want 429", rec.Code)
	}
}
