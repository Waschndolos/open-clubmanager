package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// refreshCall posts to an auth endpoint with the refresh token as cookie
// (cookie != "") and/or header (header != "") and an optional bearer token.
func refreshCall(t *testing.T, h http.Handler, path, bearer, cookie, header string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: cookie})
	}
	if header != "" {
		req.Header.Set(refreshHeaderName, header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func refreshCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			return c
		}
	}
	return nil
}

func decodeTokens(t *testing.T, rec *httptest.ResponseRecorder) openapi.AccessTokenResponse {
	t.Helper()
	var out openapi.AccessTokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	return out
}

// loginCookie logs in and returns the access token and refresh cookie value.
func loginCookie(t *testing.T, h http.Handler, email, password string) (string, string) {
	t.Helper()
	rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "", `{"email":"`+email+`","password":"`+password+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
	}
	c := refreshCookie(rec)
	if c == nil || c.Value == "" {
		t.Fatalf("login %s: no refresh cookie", email)
	}
	return decodeTokens(t, rec).AccessToken, c.Value
}

// loginBody logs in asking for the refresh token in the body (desktop app).
func loginBody(t *testing.T, h http.Handler, email, password string) (string, string) {
	t.Helper()
	rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "",
		`{"email":"`+email+`","password":"`+password+`","refreshTokenInBody":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
	}
	if refreshCookie(rec) != nil {
		t.Fatal("body login also set a cookie")
	}
	out := decodeTokens(t, rec)
	if out.RefreshToken == nil || *out.RefreshToken == "" {
		t.Fatal("body login: no refresh token in body")
	}
	return out.AccessToken, *out.RefreshToken
}

func expiredToken(t *testing.T, email string, role openapi.AppRole) string {
	t.Helper()
	claims := auth.Claims{Email: email, AppRole: role, RegisteredClaims: jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRefreshCookie(t *testing.T) {
	_, h := newSessionTestAPI(t)
	profile := func(token string) int {
		return sessionCall(t, h, http.MethodGet, "/api/v2/auth/profile", token, "").Code
	}

	rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/login", "", `{"email":"admin@example.org","password":"admin-password"}`)
	c := refreshCookie(rec)
	if c == nil {
		t.Fatal("login set no refresh cookie")
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/v2/auth" {
		t.Fatalf("cookie attributes: %+v", c)
	}
	if decodeTokens(t, rec).RefreshToken != nil {
		t.Fatal("cookie login leaked the refresh token into the body")
	}
	first := c.Value

	// The access token expired: the refresh still works and rotates.
	expired := expiredToken(t, "admin@example.org", openapi.ADMIN)
	if code := profile(expired); code != http.StatusUnauthorized {
		t.Fatalf("expired access token accepted: %d", code)
	}
	rec = refreshCall(t, h, "/api/v2/auth/refresh-token", expired, first, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh with expired access token: %d %s", rec.Code, rec.Body)
	}
	out := decodeTokens(t, rec)
	if out.RefreshToken != nil {
		t.Fatal("cookie refresh leaked the refresh token into the body")
	}
	if code := profile(out.AccessToken); code != http.StatusOK {
		t.Fatalf("refreshed access token: %d", code)
	}
	c = refreshCookie(rec)
	if c == nil || c.Value == "" || c.Value == first {
		t.Fatalf("refresh did not rotate the cookie: %+v", c)
	}
	second := c.Value

	// Without any access token.
	rec = refreshCall(t, h, "/api/v2/auth/refresh-token", "", second, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh without access token: %d %s", rec.Code, rec.Body)
	}
	third := refreshCookie(rec).Value

	// A refresh racing with the one that replaced its token still gets an
	// access token, but no new refresh token.
	rec = refreshCall(t, h, "/api/v2/auth/refresh-token", "", second, "")
	if rec.Code != http.StatusOK || refreshCookie(rec) != nil {
		t.Fatalf("racing refresh: %d cookie=%v", rec.Code, refreshCookie(rec))
	}

	// Outside the grace period a replaced token is a stolen copy: the whole
	// login is revoked, including the newest token.
	defer func(g time.Duration) { refreshReuseGrace = g }(refreshReuseGrace)
	refreshReuseGrace = 0
	rec = refreshCall(t, h, "/api/v2/auth/refresh-token", "", first, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused token: %d, want 401", rec.Code)
	}
	if c := refreshCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Fatalf("failed refresh did not clear the cookie: %+v", c)
	}
	if rec := refreshCall(t, h, "/api/v2/auth/refresh-token", "", third, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("newest token after reuse: %d, want 401", rec.Code)
	}

	for name, rec := range map[string]*httptest.ResponseRecorder{
		"missing": refreshCall(t, h, "/api/v2/auth/refresh-token", "", "", ""),
		"unknown": refreshCall(t, h, "/api/v2/auth/refresh-token", "", "garbage", ""),
		"bearer only": refreshCall(t, h, "/api/v2/auth/refresh-token",
			loginToken(t, h, "admin@example.org", "admin-password"), "", ""),
	} {
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: %d, want 401", name, rec.Code)
		}
	}
}

func TestRefreshBody(t *testing.T) {
	_, h := newSessionTestAPI(t)
	_, first := loginBody(t, h, "admin@example.org", "admin-password")

	rec := refreshCall(t, h, "/api/v2/auth/refresh-token", "", "", first)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rec.Code, rec.Body)
	}
	if refreshCookie(rec) != nil {
		t.Fatal("header refresh set a cookie")
	}
	out := decodeTokens(t, rec)
	if out.RefreshToken == nil || *out.RefreshToken == first {
		t.Fatalf("header refresh did not rotate: %v", out.RefreshToken)
	}
	if code := sessionCall(t, h, http.MethodGet, "/api/v2/auth/profile", out.AccessToken, "").Code; code != http.StatusOK {
		t.Fatalf("refreshed access token: %d", code)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	_, h := newSessionTestAPI(t)
	_, cookie := loginCookie(t, h, "admin@example.org", "admin-password")
	_, other := loginBody(t, h, "admin@example.org", "admin-password")

	rec := refreshCall(t, h, "/api/v2/auth/logout", "", cookie, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body)
	}
	if c := refreshCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Fatalf("logout did not clear the cookie: %+v", c)
	}
	if rec := refreshCall(t, h, "/api/v2/auth/refresh-token", "", cookie, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: %d, want 401", rec.Code)
	}
	// Other logins of the same account are not affected.
	if rec := refreshCall(t, h, "/api/v2/auth/refresh-token", "", "", other); rec.Code != http.StatusOK {
		t.Fatalf("other login after logout: %d", rec.Code)
	}
}

// TestRefreshRevocation checks that refresh tokens end with the same account
// changes that revoke access tokens.
func TestRefreshRevocation(t *testing.T) {
	_, h := newSessionTestAPI(t)
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
	refresh := func(token string) int {
		return refreshCall(t, h, "/api/v2/auth/refresh-token", "", token, "").Code
	}

	t.Run("password change", func(t *testing.T) {
		createUser("pw@example.org")
		access, token := loginCookie(t, h, "pw@example.org", "user-password")
		// In the same second as the login: the explicit revocation, not the
		// issue time, ends the login.
		if rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/change-password", access,
			`{"currentPassword":"user-password","newPassword":"new-password"}`); rec.Code != http.StatusOK {
			t.Fatalf("change password: %d %s", rec.Code, rec.Body)
		}
		if code := refresh(token); code != http.StatusUnauthorized {
			t.Fatalf("after password change: %d, want 401", code)
		}
		_, fresh := loginCookie(t, h, "pw@example.org", "new-password")
		if code := refresh(fresh); code != http.StatusOK {
			t.Fatalf("new login: %d", code)
		}
	})

	t.Run("role change", func(t *testing.T) {
		id := createUser("role@example.org")
		_, token := loginCookie(t, h, "role@example.org", "user-password")
		if rec := sessionCall(t, h, http.MethodPut, "/api/v2/users/"+id, admin, `{"appRole":"READONLY"}`); rec.Code != http.StatusOK {
			t.Fatalf("update: %d %s", rec.Code, rec.Body)
		}
		if code := refresh(token); code != http.StatusUnauthorized {
			t.Fatalf("after role change: %d, want 401", code)
		}
	})

	t.Run("deleted user", func(t *testing.T) {
		id := createUser("gone@example.org")
		_, token := loginCookie(t, h, "gone@example.org", "user-password")
		if rec := sessionCall(t, h, http.MethodDelete, "/api/v2/users/"+id, admin, ""); rec.Code != http.StatusNoContent {
			t.Fatalf("delete: %d %s", rec.Code, rec.Body)
		}
		if code := refresh(token); code != http.StatusUnauthorized {
			t.Fatalf("after delete: %d, want 401", code)
		}
	})

	// Last: changes the admin password the other subtests use.
	t.Run("admin password change", func(t *testing.T) {
		_, token := loginCookie(t, h, "admin@example.org", "admin-password")
		if rec := sessionCall(t, h, http.MethodPost, "/api/v2/auth/change-password", admin,
			`{"newPassword":"new-admin-password"}`); rec.Code != http.StatusOK {
			t.Fatalf("change password: %d %s", rec.Code, rec.Body)
		}
		if code := refresh(token); code != http.StatusUnauthorized {
			t.Fatalf("after admin password change: %d, want 401", code)
		}
	})
}

// Without a database a login still works, but issues no refresh token.
func TestLoginWithoutDatabaseIssuesNoRefreshToken(t *testing.T) {
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(t.TempDir(), "config.json"))
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	api := New(&config.Config{JWTSecret: "test-secret", AdminEmail: "admin@example.org", AdminPassword: string(hash)}, nil)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router,
		Middlewares: []openapi.MiddlewareFunc{auth.MiddlewareAllowing("test-secret", api.AllowAnonymous, api.ValidateSession)}})

	rec := sessionCall(t, router, http.MethodPost, "/api/v2/auth/login", "", `{"email":"admin@example.org","password":"admin-password"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	if refreshCookie(rec) != nil || strings.Contains(rec.Body.String(), "refreshToken") {
		t.Fatalf("refresh token issued without a database: %s", rec.Body)
	}
	if rec := refreshCall(t, router, "/api/v2/auth/refresh-token", "", "x", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh without a database: %d, want 401", rec.Code)
	}
}
