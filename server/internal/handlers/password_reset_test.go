package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// The reset password used to live only in memory, so it was lost on restart
// (#184).
func TestResetPasswordIsPersisted(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SERVER_CONFIG_FILE", cfgPath)

	router := chi.NewRouter()
	cfg := &config.Config{AdminEmail: "admin@example.org", AdminPassword: "old-hash"}
	openapi.HandlerWithOptions(New(cfg, nil), openapi.ChiServerOptions{BaseURL: "/api/v2", BaseRouter: router})

	const token = "test-reset-token"
	resetTokensMu.Lock()
	resetTokens[token] = tokenEntry{email: cfg.AdminEmail, expiresAt: time.Now().Add(time.Minute)}
	resetTokensMu.Unlock()

	body := map[string]any{"token": token, "newPassword": "new-secret-123"}
	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", body, nil); code != http.StatusOK {
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

	if code := doJSON(t, router, http.MethodPost, "/api/v2/auth/reset-password", body, nil); code != http.StatusBadRequest {
		t.Errorf("reusing the token: %d, want 400", code)
	}
}
