package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	claimsKey contextKey = "claims"
)

// Claims holds the JWT payload for an authenticated app user.
type Claims struct {
	Email   string          `json:"email"`
	AppRole openapi.AppRole `json:"appRole"`
	jwt.RegisteredClaims
}

// NewToken mints a signed JWT for the given email and app role.
func NewToken(email string, role openapi.AppRole, secret string) (string, error) {
	claims := Claims{
		Email:   email,
		AppRole: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// SessionCheck re-validates the claims of a correctly signed token against the
// current account state, so that deleted users, role changes and password
// changes take effect immediately instead of when the token expires.
type SessionCheck func(ctx context.Context, claims *Claims) error

// ErrSessionRevoked is returned by a SessionCheck for a token that no longer
// matches its account.
var ErrSessionRevoked = errors.New("session revoked")

// Middleware requires a valid bearer token on every request it wraps.
// Optional checks run after the signature check; any error rejects the
// request.
func Middleware(secret string, checks ...SessionCheck) openapi.MiddlewareFunc {
	return MiddlewareAllowing(secret, nil, checks...)
}

// MiddlewareAllowing requires a valid bearer token on every request except
// those anonymous reports as open to callers without one. Authentication is
// the default: an operation without a security requirement in the OpenAPI
// spec is not public unless anonymous names it. A valid token on an open
// request still attaches its claims. A token rejected by one of the checks
// counts as no token.
func MiddlewareAllowing(secret string, anonymous func(*http.Request) bool, checks ...SessionCheck) openapi.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			open := anonymous != nil && anonymous(r)

			claims, err := parseToken(r.Header.Get("Authorization"), secret)
			for _, check := range checks {
				if err == nil {
					err = check(r.Context(), claims)
				}
			}
			if err != nil {
				if open {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}

// parseToken validates the bearer token in an Authorization header.
func parseToken(rawAuth, secret string) (*Claims, error) {
	tokenString, err := extractBearer(rawAuth)
	if err != nil {
		return nil, err
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected jwt signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

func extractBearer(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("invalid authorization header")
	}
	return parts[1], nil
}
