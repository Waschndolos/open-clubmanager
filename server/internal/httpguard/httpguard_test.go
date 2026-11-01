package httpguard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware(t *testing.T) {
	g := New([]string{"http://localhost:5173"}, []string{"club.example.org"})
	handler := g.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name, method, path, host, origin, contentType, body string
		want                                                int
		wantCORS                                            bool
	}{
		{name: "native client", method: "GET", path: "/api/v2/members", host: "localhost:3001", want: 200},
		{name: "tauri origin", method: "GET", path: "/api/v2/members", host: "localhost:3001", origin: "tauri://localhost", want: 200, wantCORS: true},
		{name: "windows tauri origin", method: "GET", path: "/x", host: "127.0.0.1:3001", origin: "http://tauri.localhost", want: 200, wantCORS: true},
		{name: "configured dev origin", method: "GET", path: "/x", host: "localhost:3001", origin: "http://localhost:5173", want: 200, wantCORS: true},
		{name: "other localhost port", method: "GET", path: "/x", host: "localhost:3001", origin: "http://localhost:8080", want: 403},
		{name: "foreign site", method: "POST", path: "/x", host: "localhost:3001", origin: "https://evil.example", contentType: "application/json", body: "{}", want: 403},
		{name: "null origin", method: "GET", path: "/x", host: "localhost:3001", origin: "null", want: 403},
		{name: "preflight", method: "OPTIONS", path: "/x", host: "localhost:3001", origin: "tauri://localhost", want: 204, wantCORS: true},
		{name: "foreign preflight", method: "OPTIONS", path: "/x", host: "localhost:3001", origin: "http://localhost:8080", want: 403},
		{name: "rebinding host", method: "GET", path: "/x", host: "attacker.example:3001", want: 421},
		{name: "configured host", method: "GET", path: "/x", host: "club.example.org", want: 200},
		{name: "ipv6 host", method: "GET", path: "/x", host: "[::1]:3001", want: 200},
		{name: "json write", method: "PUT", path: "/x", host: "localhost:3001", contentType: "application/json; charset=utf-8", body: "{}", want: 200},
		{name: "text/plain write", method: "POST", path: "/api/v2/setup/initialize", host: "localhost:3001", contentType: "text/plain", body: "{}", want: 415},
		{name: "form write", method: "POST", path: "/x", host: "localhost:3001", contentType: "application/x-www-form-urlencoded", body: "a=b", want: 415},
		{name: "missing content type", method: "POST", path: "/x", host: "localhost:3001", body: "{}", want: 415},
		{name: "bodyless write", method: "POST", path: "/api/v2/auth/refresh-token", host: "localhost:3001", want: 200},
		{name: "multipart restore", method: "POST", path: "/api/v2/system/restore", host: "localhost:3001", contentType: "multipart/form-data; boundary=x", body: "--x--", want: 200},
		{name: "multipart elsewhere", method: "POST", path: "/api/v2/members", host: "localhost:3001", contentType: "multipart/form-data; boundary=x", body: "--x--", want: 415},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var req *http.Request
			if c.body != "" {
				req = httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			} else {
				req = httptest.NewRequest(c.method, c.path, nil)
			}
			req.Host = c.host
			if c.origin != "" {
				req.Header.Set("Origin", c.origin)
			}
			if c.contentType != "" {
				req.Header.Set("Content-Type", c.contentType)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("status %d, want %d (%s)", rec.Code, c.want, rec.Body)
			}
			gotCORS := rec.Header().Get("Access-Control-Allow-Origin") != ""
			if gotCORS != c.wantCORS {
				t.Fatalf("CORS headers present = %v, want %v", gotCORS, c.wantCORS)
			}
			if gotCORS && rec.Header().Get("Access-Control-Allow-Origin") != c.origin {
				t.Fatalf("Allow-Origin %q, want %q", rec.Header().Get("Access-Control-Allow-Origin"), c.origin)
			}
		})
	}
}

// The desktop app sends its refresh token in a header, which a cross-origin
// request may only carry when the preflight allows it.
func TestPreflightAllowsRefreshHeader(t *testing.T) {
	handler := New(nil, nil).Middleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodOptions, "/api/v2/auth/refresh-token", nil)
	req.Host = "localhost:3001"
	req.Header.Set("Origin", "tauri://localhost")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "X-Refresh-Token") {
		t.Fatalf("Allow-Headers %q lacks X-Refresh-Token", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}
