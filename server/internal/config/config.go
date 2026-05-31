package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Port          int    `json:"port"`
	DatabasePath  string `json:"databasePath"`
	DatabaseURL   string `json:"databaseUrl"`
	JWTSecret     string `json:"jwtSecret"`
	AdminEmail    string `json:"adminEmail"`
	AdminPassword string `json:"adminPasswordHash"`
}

func Load() (Config, error) {
	cfg := Config{
		Port:         3001,
		DatabasePath: "./clubmanager.db",
		JWTSecret:    "dev_access_secret_change_me",
	}

	if path := os.Getenv("SERVER_CONFIG_FILE"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return Config{}, err
		}
		defer f.Close()

		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return Config{}, err
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

	if cfg.Port <= 0 {
		return Config{}, errors.New("port must be greater than 0")
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("jwt secret must not be empty")
	}

	return cfg, nil
}
