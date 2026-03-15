package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultDatabaseURL = "postgres://postgres:postgres@postgres:5432/shipyard?sslmode=disable"
	DefaultAPIKey      = "dev-shipyard-key"
	DefaultMinIOUser   = "minioadmin"
	DefaultMinIOPass   = "minioadmin"
)

func AllowInsecureDefaults() bool {
	raw := strings.TrimSpace(os.Getenv("SHIPYARD_ALLOW_INSECURE_DEFAULTS"))
	if raw == "" {
		return false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return parsed
}

func ValidateAPIKey(apiKey string) error {
	if strings.TrimSpace(apiKey) == "" {
		return errors.New("SHIPYARD_API_KEY is required")
	}
	if apiKey == DefaultAPIKey {
		return errors.New("SHIPYARD_API_KEY must not use default development value")
	}
	return nil
}

func ValidateDatabaseURL(dsn string) error {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return errors.New("DATABASE_URL is required")
	}
	if trimmed == DefaultDatabaseURL || strings.Contains(trimmed, "postgres:postgres@") {
		return errors.New("DATABASE_URL uses default postgres credentials")
	}
	return nil
}

func ValidateMinIOCredentials(accessKey, secretKey string) error {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		return errors.New("MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required")
	}
	if accessKey == DefaultMinIOUser && secretKey == DefaultMinIOPass {
		return errors.New("MINIO credentials must not use default development values")
	}
	return nil
}
