package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

// Refresh tokens keep a login alive while the app is in use. They are opaque
// random strings; the database only holds their SHA-256 hash.
//
// A browser on the same site as the backend (the dev setup) gets the token in
// an httpOnly, SameSite=Strict cookie scoped to the auth endpoints, so page
// scripts never see it. The desktop app cannot use that cookie: its pages are
// served from tauri://localhost or http://tauri.localhost, which is a
// different site than the backend on http://localhost, so a SameSite cookie
// is never sent and WebKit's third-party cookie blocking drops it anyway.
// Such clients ask for the token in the response body at login
// (refreshTokenInBody) and send it back in the X-Refresh-Token header. The
// reply always uses the channel the token came in, so a script cannot turn a
// cookie session into a readable token.
const (
	refreshCookieName = "refreshToken"
	refreshHeaderName = "X-Refresh-Token"
	refreshCookiePath = "/api/v2/auth"
	// refreshTokenTTL is how long a login survives without any refresh. Every
	// refresh issues a token valid for this long again.
	refreshTokenTTL = 12 * time.Hour
)

// refreshReuseGrace tolerates a just-replaced token, so two refreshes racing
// with the same token (several browser tabs sharing the cookie) do not revoke
// the login. Within it the old token gets a new access token but no new
// refresh token. A variable for tests.
var refreshReuseGrace = 30 * time.Second

var errNoRefreshToken = errors.New("no refresh token")

// refreshTokenFromRequest returns the presented refresh token and whether it
// came in the header (rather than the cookie).
func refreshTokenFromRequest(r *http.Request) (token string, viaHeader bool) {
	if h := r.Header.Get(refreshHeaderName); h != "" {
		return h, true
	}
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		return c.Value, false
	}
	return "", false
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setRefreshCookie(w http.ResponseWriter, r *http.Request, token string) {
	// No Expires: the cookie ends with the browser session, like the
	// sessionStorage copy the desktop app keeps. The server enforces the TTL.
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
}

// deliverRefreshToken puts a new refresh token into resp or a cookie.
func deliverRefreshToken(w http.ResponseWriter, r *http.Request, resp *openapi.AccessTokenResponse, token string, inBody bool) {
	if inBody {
		resp.RefreshToken = &token
		return
	}
	setRefreshCookie(w, r, token)
}

// issueRefreshToken starts a new refresh token family for a fresh login. It
// returns "" without a database, where logins only last as long as the
// access token.
func (a *API) issueRefreshToken(ctx context.Context, email string, role openapi.AppRole) (string, error) {
	st := a.store()
	if st == nil {
		return "", nil
	}
	var userID string
	if email != a.cfg.AdminEmail {
		user, err := st.GetUserByEmail(ctx, email)
		if err != nil {
			return "", err
		}
		userID = user.ID
	}
	token, err := newRefreshToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	if err := st.CreateRefreshToken(ctx, hashRefreshToken(token), db.NewID(), email, string(role), userID, now, now.Add(refreshTokenTTL)); err != nil {
		return "", err
	}
	return token, nil
}

// RefreshAccessToken exchanges a refresh token for a new access token and a
// new refresh token. The access token of the request is ignored, so it works
// after the access token expired. The refresh token is checked against the
// account like an access token (ValidateSession): deleted users, changed
// roles and password changes end the login.
func (a *API) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	token, viaHeader := refreshTokenFromRequest(r)
	resp, err := a.refresh(w, r, token, viaHeader)
	if err != nil {
		if !errors.Is(err, errNoRefreshToken) && !errors.Is(err, auth.ErrSessionRevoked) {
			log.Printf("auth: refresh: %v", err)
		}
		if !viaHeader {
			clearRefreshCookie(w, r)
		}
		writeJSON(w, http.StatusUnauthorized, openapi.ErrorResponse{Error: "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request, token string, viaHeader bool) (openapi.AccessTokenResponse, error) {
	ctx := r.Context()
	var resp openapi.AccessTokenResponse
	st := a.store()
	if token == "" || st == nil {
		return resp, errNoRefreshToken
	}
	stored, err := st.GetRefreshToken(ctx, hashRefreshToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return resp, errNoRefreshToken
	}
	if err != nil {
		return resp, err
	}

	now := time.Now()
	revoke := func(reason error) (openapi.AccessTokenResponse, error) {
		if err := st.RevokeRefreshTokenFamily(ctx, stored.FamilyID); err != nil {
			log.Printf("auth: revoke refresh tokens: %v", err)
		}
		return resp, reason
	}
	if !now.Before(stored.ExpiresAt) {
		return revoke(auth.ErrSessionRevoked)
	}
	reused := stored.ReplacedAt != nil
	if reused && now.Sub(*stored.ReplacedAt) > refreshReuseGrace {
		// A replaced token came back: it was copied. End the login.
		log.Printf("auth: replaced refresh token reused for %s; login revoked", stored.Email)
		return revoke(auth.ErrSessionRevoked)
	}
	claims := &auth.Claims{Email: stored.Email, AppRole: openapi.AppRole(stored.AppRole)}
	claims.IssuedAt = jwt.NewNumericDate(stored.CreatedAt)
	if err := a.ValidateSession(ctx, claims); err != nil {
		return revoke(err)
	}

	if !reused {
		next, err := newRefreshToken()
		if err != nil {
			return resp, err
		}
		err = st.RotateRefreshToken(ctx, stored, hashRefreshToken(next), now, now.Add(refreshTokenTTL))
		switch {
		case err == nil:
			deliverRefreshToken(w, r, &resp, next, viaHeader)
		case errors.Is(err, store.ErrNotFound):
			// A concurrent refresh rotated it first; treat as within grace.
		default:
			return resp, err
		}
	}

	access, err := auth.NewToken(claims.Email, claims.AppRole, a.cfg.JWTSecret)
	if err != nil {
		return resp, err
	}
	resp.AccessToken = access
	return resp, nil
}

// Logout revokes the presented refresh token's login and clears the cookie.
// Access tokens already issued stay valid until they expire (15 minutes);
// the client discards its copy.
func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	if token, _ := refreshTokenFromRequest(r); token != "" {
		if st := a.store(); st != nil {
			if stored, err := st.GetRefreshToken(r.Context(), hashRefreshToken(token)); err == nil {
				if err := st.RevokeRefreshTokenFamily(r.Context(), stored.FamilyID); err != nil {
					log.Printf("auth: logout: %v", err)
				}
			}
		}
	}
	clearRefreshCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// revokeAdminRefreshTokens ends all logins of the administrator from
// config.json, after its password changed.
func (a *API) revokeAdminRefreshTokens(ctx context.Context) {
	if st := a.store(); st != nil && a.cfg.AdminEmail != "" {
		if err := st.RevokeRefreshTokensForEmail(ctx, a.cfg.AdminEmail); err != nil {
			log.Printf("auth: revoke admin refresh tokens: %v", err)
		}
	}
}
