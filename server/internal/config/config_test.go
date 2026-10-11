package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateSecretGeneratesAndReuses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "jwt_secret")

	first, err := LoadOrCreateSecret(path)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(first) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(first))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", perm)
	}

	second, err := LoadOrCreateSecret(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if first != second {
		t.Fatal("secret changed between starts")
	}

	other, err := LoadOrCreateSecret(filepath.Join(t.TempDir(), "jwt_secret"))
	if err != nil {
		t.Fatalf("generate other: %v", err)
	}
	if other == first {
		t.Fatal("two generated secrets are identical")
	}
}

func clearEnv(t *testing.T) {
	for _, k := range []string{"PORT", "BIND_ADDRESS", "JWT_SECRET", "JWT_ACCESS_SECRET", "JWT_SECRET_FILE", "DATABASE_PATH", "DATABASE_URL"} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaultsToLoopbackAndGeneratedSecret(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	// Existing installs may have the old hard-coded secret persisted.
	if err := os.WriteFile(cfgPath, []byte(`{"jwtSecret":"dev_access_secret_change_me"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SERVER_CONFIG_FILE", cfgPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.BindAddress != "127.0.0.1" {
		t.Fatalf("expected loopback bind address, got %q", cfg.BindAddress)
	}
	if cfg.JWTSecret == "" || cfg.JWTSecret == legacyDefaultJWTSecret {
		t.Fatalf("expected generated secret, got %q", cfg.JWTSecret)
	}
	stored, err := os.ReadFile(filepath.Join(dir, "jwt_secret"))
	if err != nil {
		t.Fatalf("secret file not written: %v", err)
	}
	if string(stored) != cfg.JWTSecret+"\n" {
		t.Fatal("secret file content does not match loaded secret")
	}

	// Saving the config must not copy the generated secret into config.json.
	if err := Save(cfg, cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}
	raw, _ := os.ReadFile(cfgPath)
	var saved map[string]any
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if _, ok := saved["jwtSecret"]; ok {
		t.Fatal("generated secret leaked into config.json")
	}

	again, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again.JWTSecret != cfg.JWTSecret {
		t.Fatal("secret not reused on second start")
	}
}

func TestLoadExplicitSettingsWin(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	t.Setenv("SERVER_CONFIG_FILE", filepath.Join(dir, "config.json"))
	t.Setenv("BIND_ADDRESS", "0.0.0.0")
	t.Setenv("JWT_SECRET", "explicit-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.BindAddress != "0.0.0.0" || cfg.JWTSecret != "explicit-secret" {
		t.Fatalf("unexpected config: bind=%q secret=%q", cfg.BindAddress, cfg.JWTSecret)
	}
	if _, err := os.Stat(filepath.Join(dir, "jwt_secret")); !os.IsNotExist(err) {
		t.Fatal("secret file must not be created when a secret is configured")
	}
}
