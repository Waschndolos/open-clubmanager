package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultBindAddress keeps the API reachable from this machine only. Server
// deployments that must be reachable from the network set BIND_ADDRESS
// (or "bindAddress" in the config file) to e.g. 0.0.0.0 explicitly.
const DefaultBindAddress = "127.0.0.1"

// legacyDefaultJWTSecret was the hard-coded fallback of earlier versions. It
// may still be persisted in existing config files and is treated as unset.
const legacyDefaultJWTSecret = "dev_access_secret_change_me"

// jwtSecretFileName is the file (next to config.json) holding the generated
// JWT secret when none is configured explicitly.
const jwtSecretFileName = "jwt_secret"

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type Config struct {
	Port           int                    `json:"port"`
	BindAddress    string                 `json:"bindAddress,omitempty"`
	DatabasePath   string                 `json:"databasePath"`
	JWTSecret      string                 `json:"jwtSecret,omitempty"`
	AdminEmail     string                 `json:"adminEmail"`
	AdminPassword  string                 `json:"adminPasswordHash"`
	AppBaseURL     string                 `json:"appBaseUrl"`
	SMTP           SMTPConfig             `json:"smtp"`
	AppPreferences map[string]interface{} `json:"appPreferences,omitempty"`

	// jwtSecretGenerated is true when JWTSecret was read from / written to the
	// generated secret file. Such a secret is never copied into config.json.
	jwtSecretGenerated bool
}

// DefaultConfigFilePath returns the platform-appropriate path for the persisted
// config file (~/.config/open-clubmanager/config.json on Linux/macOS).
func DefaultConfigFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "open-clubmanager", "config.json"), nil
}

// Save persists cfg as JSON to path, creating parent directories as needed.
func Save(cfg Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if cfg.jwtSecretGenerated {
		cfg.JWTSecret = ""
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func Load() (Config, error) {
	cfg := Config{
		Port:         3001,
		BindAddress:  DefaultBindAddress,
		DatabasePath: "./clubmanager.db",
	}

	// Prefer an explicitly configured file; fall back to the default location.
	configPath := os.Getenv("SERVER_CONFIG_FILE")
	if configPath == "" {
		if defaultPath, err := DefaultConfigFilePath(); err == nil {
			configPath = defaultPath
		}
	}

	if configPath != "" {
		f, err := os.Open(configPath)
		if err != nil && !os.IsNotExist(err) {
			return Config{}, err
		}
		if err == nil {
			defer f.Close()
			if err := json.NewDecoder(f).Decode(&cfg); err != nil {
				return Config{}, err
			}
		}
	}

	if value := os.Getenv("PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, err
		}
		cfg.Port = port
	}

	if value := os.Getenv("BIND_ADDRESS"); value != "" {
		cfg.BindAddress = value
	}
	if cfg.BindAddress == "" {
		cfg.BindAddress = DefaultBindAddress
	}

	if value := os.Getenv("DATABASE_PATH"); value != "" {
		cfg.DatabasePath = value
	}

	if value := os.Getenv("JWT_SECRET"); value != "" {
		cfg.JWTSecret = value
	}
	if value := os.Getenv("JWT_ACCESS_SECRET"); value != "" {
		cfg.JWTSecret = value
	}

	if value := os.Getenv("ADMIN_EMAIL"); value != "" {
		cfg.AdminEmail = value
	}

	if value := os.Getenv("ADMIN_PASSWORD_HASH"); value != "" {
		cfg.AdminPassword = value
	}

	if value := os.Getenv("APP_BASE_URL"); value != "" {
		cfg.AppBaseURL = value
	}

	if value := os.Getenv("SMTP_HOST"); value != "" {
		cfg.SMTP.Host = value
	}
	if value := os.Getenv("SMTP_PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, err
		}
		cfg.SMTP.Port = port
	}
	if value := os.Getenv("SMTP_USER"); value != "" {
		cfg.SMTP.User = value
	}
	if value := os.Getenv("SMTP_PASSWORD"); value != "" {
		cfg.SMTP.Password = value
	}
	if value := os.Getenv("SMTP_FROM"); value != "" {
		cfg.SMTP.From = value
	}

	if cfg.Port <= 0 {
		return Config{}, errors.New("port must be greater than 0")
	}

	if cfg.JWTSecret == legacyDefaultJWTSecret {
		cfg.JWTSecret = ""
	}
	if cfg.JWTSecret == "" {
		secretPath := os.Getenv("JWT_SECRET_FILE")
		if secretPath == "" {
			if configPath == "" {
				return Config{}, errors.New("jwt secret not configured and no config directory available to store a generated one")
			}
			secretPath = filepath.Join(filepath.Dir(configPath), jwtSecretFileName)
		}
		secret, err := LoadOrCreateSecret(secretPath)
		if err != nil {
			return Config{}, fmt.Errorf("jwt secret: %w", err)
		}
		cfg.JWTSecret = secret
		cfg.jwtSecretGenerated = true
	}

	return cfg, nil
}

// LoadOrCreateSecret returns the secret stored at path. If the file does not
// exist, a new 256-bit random secret is generated and written with 0600
// permissions so that later starts reuse it (and issued tokens stay valid).
func LoadOrCreateSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		secret := strings.TrimSpace(string(data))
		if secret == "" {
			return "", fmt.Errorf("secret file %s is empty", path)
		}
		return secret, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(buf)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	// O_EXCL: never overwrite a secret another process created concurrently.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return LoadOrCreateSecret(path)
		}
		return "", err
	}
	if _, err := f.WriteString(secret + "\n"); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return secret, nil
}
