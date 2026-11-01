// Package httpguard protects the loopback API against requests from untrusted
// browser contexts: cross-origin calls (CORS), cross-site request forgery and
// DNS rebinding.
package httpguard

import (
	"mime"
	"net"
	"net/http"
	"strings"
)

// DefaultOrigins are the origins of the packaged Tauri 2 app:
//   - tauri://localhost       – macOS / Linux
//   - http://tauri.localhost  – Windows WebView2 (default scheme)
//   - https://tauri.localhost – Windows WebView2 with useHttpsScheme
//
// Development origins (e.g. the Vite dev server) are not trusted by default;
// they are added explicitly via configuration.
var DefaultOrigins = []string{"tauri://localhost", "http://tauri.localhost", "https://tauri.localhost"}

// multipartPaths may receive multipart/form-data bodies (file uploads). All
// other state-changing requests must send JSON.
var multipartPaths = map[string]bool{"/api/v2/system/restore": true}

// Guard holds the allow-lists. The zero value is not usable; use New.
type Guard struct {
	origins map[string]bool
	hosts   map[string]bool
}

// New creates a guard that trusts DefaultOrigins plus extraOrigins, and the
// loopback host names plus extraHosts. Host entries are host names without
// port; IP literals are always accepted because DNS rebinding needs a name.
func New(extraOrigins, extraHosts []string) *Guard {
	g := &Guard{origins: map[string]bool{}, hosts: map[string]bool{"localhost": true}}
	for _, o := range append(append([]string{}, DefaultOrigins...), extraOrigins...) {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
			g.origins[strings.ToLower(o)] = true
		}
	}
	for _, h := range extraHosts {
		if h = strings.TrimSpace(h); h != "" {
			g.hosts[strings.ToLower(h)] = true
		}
	}
	return g
}

// SplitList splits a comma-separated configuration value.
func SplitList(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

// OriginAllowed reports whether origin may call the API from a browser.
func (g *Guard) OriginAllowed(origin string) bool {
	return g.origins[strings.ToLower(origin)]
}

// HostAllowed reports whether the Host header names this server: an IP
// literal, localhost or a configured host name.
func (g *Guard) HostAllowed(hostHeader string) bool {
	host := hostHeader
	if h, _, err := net.SplitHostPort(hostHeader); err == nil {
		host = h
	}
	host = strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
	if host == "" {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	return g.hosts[host]
}

// Middleware rejects requests with an unknown Host header or a foreign
// Origin, adds CORS headers for trusted origins, answers preflights and
// requires a JSON body on state-changing requests. The content-type rule
// makes every browser write a preflighted (non-simple) request, so a page
// cannot submit one without passing the origin check first.
func (g *Guard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.HostAllowed(r.Host) {
			http.Error(w, "invalid host header", http.StatusMisdirectedRequest)
			return
		}

		// Browsers send Origin on every cross-origin request and on
		// same-origin writes; native clients (curl, tests) send none.
		if origin := r.Header.Get("Origin"); origin != "" {
			if !g.OriginAllowed(origin) {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			h.Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if hasBody(r) && isStateChanging(r.Method) && !contentTypeAllowed(r) {
			http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isStateChanging(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}

func hasBody(r *http.Request) bool {
	return r.ContentLength > 0 || (r.ContentLength < 0 && r.Body != nil && r.Body != http.NoBody)
}

func contentTypeAllowed(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	switch mediaType {
	case "application/json":
		return true
	case "multipart/form-data":
		return multipartPaths[r.URL.Path]
	}
	return false
}
