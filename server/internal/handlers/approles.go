package handlers

import (
	"net/http"
	"slices"
	"strings"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// Which app roles (of the login) may change which data. The device scopes of
// a club (ScopeGuard) apply on top. The lists name the roles that may write,
// so roles not listed here, such as READONLY or newer restricted roles, stay
// read-only.
var (
	// dataEditors may change club data in general: members, club structure,
	// inventory, events, households and tasks.
	dataEditors = []openapi.AppRole{openapi.ADMIN, openapi.TREASURER, openapi.SECRETARY}
	// financeEditors may change transactions and member fees.
	financeEditors = []openapi.AppRole{openapi.ADMIN, openapi.TREASURER}
)

// roleWrites maps the generated write endpoints to the roles that may use them.
var roleWrites = []struct {
	prefix string
	roles  []openapi.AppRole
}{
	{"/api/v2/members", dataEditors},
	{"/api/v2/roles", dataEditors},
	{"/api/v2/groups", dataEditors},
	{"/api/v2/sections", dataEditors},
	{"/api/v2/finance", financeEditors},
	{"/api/v2/inventory", dataEditors},
	{"/api/v2/events", dataEditors},
}

// canEditData reports whether an app role may change club data in general.
func canEditData(role openapi.AppRole) bool { return slices.Contains(dataEditors, role) }

// roleMayWrite reports whether the login behind r may write to path. Paths
// without a rule are left to their handlers.
func roleMayWrite(r *http.Request, path string) bool {
	for _, rw := range roleWrites {
		if strings.HasPrefix(path, rw.prefix) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			return ok && slices.Contains(rw.roles, claims.AppRole)
		}
	}
	return true
}
