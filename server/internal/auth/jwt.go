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

func Middleware(secret string) openapi.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicRoute(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			rawAuth := r.Header.Get("Authorization")
			tokenString, err := extractBearer(rawAuth)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, errors.New("unexpected jwt signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}

func isPublicRoute(method, path string) bool {
	switch {
	case method == http.MethodGet && path == "/api/v2/system/health":
		return true
	case method == http.MethodGet && path == "/api/v2/system/meta":
		return true
	case method == http.MethodGet && path == "/api/v2/setup/status":
		return true
	case method == http.MethodPost && path == "/api/v2/setup/configure-database":
		return true
	case method == http.MethodPost && path == "/api/v2/setup/initialize":
		return true
	case method == http.MethodPost && path == "/api/v2/auth/login":
		return true
	case method == http.MethodPost && path == "/api/v2/auth/refresh-token":
		return true
	case method == http.MethodPost && path == "/api/v2/auth/logout":
		return true
	case method == http.MethodPost && path == "/api/v2/auth/forgot-password":
		return true
	case method == http.MethodPost && path == "/api/v2/auth/reset-password":
		return true
	default:
		return false
	}
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
