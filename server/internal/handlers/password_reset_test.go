package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// newResetTestAPI serves the public reset endpoints on a fresh database and
// returns the config file path.
func newResetTestAPI(t *testing.T) (*API, chi.Router, *sql.DB, string) {
	t.Helper()
	conn, err := dbpkg.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SERVER_CONFIG_FILE", cfgPath)

	api := New(&config.Config{AdminEmail: "admin@example.org", AdminPassword: "old-hash"}, conn)
	router := chi.NewRouter()
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router})
	return api, router, conn, cfgPath
}

func resetBody(token, password string) map[string]any {
	return map[string]any{"token": token, "newPassword": password}
}

// The reset password used to live only in memory, so it was lost on restart
// (#184). It also ends the admin's older sessions.
func TestResetPasswordIsPersisted(t *testing.T) {
	api, router, _, cfgPath := newResetTestAPI(t)

	const token = "test-reset-token"
	if err := api.store().CreatePasswordResetToken(context.Background(), dbpkg.AdminUserID, token, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	oldSession := &auth.Claims{Email: "admin@example.org", AppRole: openapi.ADMIN}
	oldSession.IssuedAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))

	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody(token, "new-secret-123"), nil); code != http.StatusOK {
		t.Fatalf("reset: %d", code)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config not saved: %v", err)
	}
	var saved config.Config
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(saved.AdminPassword), []byte("new-secret-123")) != nil {
		t.Errorf("saved admin password does not match the new password")
	}
	if err := api.ValidateSession(context.Background(), oldSession); err == nil {
		t.Errorf("an admin session from before the reset is still valid")
	}

	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody(token, "new-secret-123"), nil); code != http.StatusBadRequest {
		t.Errorf("reusing the token: %d, want 400", code)
	}
}

// Reset tokens are kept in the database, hashed, single-use and with expiry
// (#184).
func TestResetTokensAreStoredHashed(t *testing.T) {
	api, router, conn, _ := newResetTestAPI(t)
	ctx := context.Background()
	st := api.store()

	// Only the hash of the token is stored.
	if err := st.CreatePasswordResetToken(ctx, dbpkg.AdminUserID, "plain-token", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := conn.QueryRow(`SELECT token_hash FROM password_reset_tokens`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == "plain-token" || len(stored) != 64 {
		t.Errorf("stored token %q is not a SHA-256 hash", stored)
	}

	// The token survives a restart: a new API on the same database redeems it.
	restarted := New(api.cfg, conn)
	if id, err := restarted.store().ConsumePasswordResetToken(ctx, "plain-token"); err != nil || id != dbpkg.AdminUserID {
		t.Errorf("redeem after restart: %q %v", id, err)
	}

	// A new request replaces the older link of the same account.
	if err := st.CreatePasswordResetToken(ctx, dbpkg.AdminUserID, "first", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := st.CreatePasswordResetToken(ctx, dbpkg.AdminUserID, "second", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody("first", "new-secret-123"), nil); code != http.StatusBadRequest {
		t.Errorf("superseded token: %d, want 400", code)
	}

	// Expired tokens are refused and cleaned up.
	if err := st.CreatePasswordResetToken(ctx, "some-user", "expired", time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody("expired", "new-secret-123"), nil); code != http.StatusBadRequest {
		t.Errorf("expired token: %d, want 400", code)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = 'some-user'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d expired tokens left", n)
	}

	// The password policy applies before the token is spent.
	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody("second", "short"), nil); code != http.StatusBadRequest {
		t.Errorf("short password: %d, want 400", code)
	}
	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody("second", "new-secret-123"), nil); code != http.StatusOK {
		t.Errorf("valid token after a refused short password: %d, want 200", code)
	}
}

// Users of the database can reset their password as well; the reset ends
// their older sessions.
func TestResetPasswordOfDatabaseUser(t *testing.T) {
	api, router, conn, _ := newResetTestAPI(t)
	ctx := context.Background()
	st := api.store()

	u, err := st.CreateUser(ctx, "pia@example.org", "old-hash", string(openapi.SECRETARY))
	if err != nil {
		t.Fatal(err)
	}

	// The forgot-password request stores a token for the user, and none for
	// unknown addresses.
	for _, email := range []string{"pia@example.org", "nobody@example.org"} {
		if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/forgot-password", map[string]any{"email": email}, nil); code != http.StatusOK {
			t.Errorf("forgot %s: %d", email, code)
		}
	}
	var owners []string
	rows, err := conn.Query(`SELECT user_id FROM password_reset_tokens`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		owners = append(owners, id)
	}
	rows.Close()
	if len(owners) != 1 || owners[0] != u.ID {
		t.Errorf("tokens stored for %v, want only %s", owners, u.ID)
	}

	if err := st.CreatePasswordResetToken(ctx, u.ID, "pia-token", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	// Sessions carry whole seconds; move the account's last change back so a
	// session from before the reset is valid until then.
	if _, err := conn.Exec(`UPDATE users SET updated_at = ? WHERE id = ?`, dbpkg.FormatTime(time.Now().Add(-time.Hour)), u.ID); err != nil {
		t.Fatal(err)
	}
	oldSession := &auth.Claims{Email: u.Email, AppRole: openapi.SECRETARY}
	oldSession.IssuedAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	if err := api.ValidateSession(ctx, oldSession); err != nil {
		t.Fatalf("session before the reset: %v", err)
	}

	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", resetBody("pia-token", "new-secret-123"), nil); code != http.StatusOK {
		t.Fatalf("reset: %d", code)
	}
	got, err := st.GetUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte("new-secret-123")) != nil {
		t.Errorf("user password was not changed")
	}
	if err := api.ValidateSession(ctx, oldSession); err == nil {
		t.Errorf("a session from before the reset is still valid")
	}
}
