package handlers

import (
	"context"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ValidateSession is an auth.SessionCheck: it rejects tokens whose account
// was deleted, renamed or given another role, and tokens issued before the
// account's last credential change. Tokens carry whole seconds, so a change
// only revokes tokens issued in an earlier second.
func (a *API) ValidateSession(ctx context.Context, claims *auth.Claims) error {
	var issued time.Time
	if claims.IssuedAt != nil {
		issued = claims.IssuedAt.Time
	}

	if a.cfg.AdminEmail != "" && claims.Email == a.cfg.AdminEmail {
		if claims.AppRole != openapi.ADMIN || issued.Unix() < a.cfg.AdminTokensValidAfter {
			return auth.ErrSessionRevoked
		}
		return nil
	}

	st := a.store()
	if st == nil {
		return auth.ErrSessionRevoked
	}
	user, err := st.GetUserByEmail(ctx, claims.Email)
	if err != nil {
		return auth.ErrSessionRevoked
	}
	if openapi.AppRole(user.AppRole) != claims.AppRole || issued.Before(user.UpdatedAt.Truncate(time.Second)) {
		return auth.ErrSessionRevoked
	}
	return nil
}
