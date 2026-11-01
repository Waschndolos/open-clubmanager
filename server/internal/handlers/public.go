package handlers

import "net/http"

// publicOperations may be called without a token at any time: signing in,
// recovering a password and checking whether the server is up.
var publicOperations = map[string]bool{
	"POST /api/v2/auth/login":           true,
	"POST /api/v2/auth/logout":          true,
	"POST /api/v2/auth/refresh-token":   true,
	"POST /api/v2/auth/forgot-password": true,
	"POST /api/v2/auth/reset-password":  true,
	"GET /api/v2/setup/status":          true,
	"GET /api/v2/system/health":         true,
	"GET /api/v2/system/meta":           true,
}

// setupOperations may be called without a token only until the first-run
// setup has created the admin. Afterwards they need a token like any other.
var setupOperations = map[string]bool{
	"POST /api/v2/setup/configure-database": true,
	"POST /api/v2/setup/initialize":         true,
	"GET /api/v2/settings/database":         true,
	"POST /api/v2/validation/check-db-path": true,
	"POST /api/v2/validation/check-db-url":  true,
}

// AllowAnonymous reports whether r may pass the auth middleware without a
// token. Every operation not listed here requires one.
func (a *API) AllowAnonymous(r *http.Request) bool {
	op := r.Method + " " + r.URL.Path
	if publicOperations[op] {
		return true
	}
	return setupOperations[op] && a.setupPending()
}

// setupPending reports whether the first-run setup has not created the admin
// yet.
func (a *API) setupPending() bool {
	a.setupMu.Lock()
	defer a.setupMu.Unlock()
	return a.cfg.AdminPassword == ""
}
