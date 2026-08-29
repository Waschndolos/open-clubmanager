package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type Config struct {
	Port           int                    `json:"port"`
	DatabasePath   string                 `json:"databasePath"`
	DatabaseURL    string                 `json:"databaseUrl"`
	DocumentMaxMB  int                    `json:"documentMaxMb"`
	JWTSecret      string                 `json:"jwtSecret"`
	AdminEmail     string                 `json:"adminEmail"`
	AdminPassword  string                 `json:"adminPasswordHash"`
	AppBaseURL     string                 `json:"appBaseUrl"`
	SMTP           SMTPConfig             `json:"smtp"`
	AppPreferences map[string]interface{} `json:"appPreferences,omitempty"`
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
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func Load() (Config, error) {
	cfg := Config{
		Port:          3001,
		DatabasePath:  "./clubmanager.db",
		DocumentMaxMB: 20,
		JWTSecret:     "dev_access_secret_change_me",
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

	if value := os.Getenv("DATABASE_PATH"); value != "" {
		cfg.DatabasePath = value
	}

	if value := os.Getenv("DATABASE_URL"); value != "" {
		cfg.DatabaseURL = value
	}
	if value := os.Getenv("DOCUMENT_MAX_MB"); value != "" {
		maxMB, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, err
		}
		cfg.DocumentMaxMB = maxMB
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

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("jwt secret must not be empty")
	}
	if cfg.DocumentMaxMB <= 0 {
		return Config{}, errors.New("document max upload size must be greater than 0")
	}

	return cfg, nil
}
