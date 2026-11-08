// Package config loads runtime settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"
)

// Config holds everything the API server needs at startup.
type Config struct {
	Env            string
	Addr           string
	DatabaseURL    string
	JWTSecret      []byte
	TokenTTL       time.Duration
	AllowedOrigins []string
	LogLevel       slog.Level
}

// Load reads the server configuration. getenv is os.Getenv outside of tests.
// All problems are reported together so a misconfigured deploy fails once
// with the full list instead of one variable at a time.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:  valueOr(getenv("APP_ENV"), "development"),
		Addr: valueOr(getenv("HTTP_ADDR"), ":8080"),
	}
	var errs []error

	dsn, err := DatabaseURL(getenv)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.DatabaseURL = dsn

	cfg.JWTSecret = []byte(getenv("JWT_SECRET"))
	if len(cfg.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 bytes"))
	}

	cfg.TokenTTL, err = time.ParseDuration(valueOr(getenv("TOKEN_TTL"), "24h"))
	if err != nil || cfg.TokenTTL <= 0 {
		errs = append(errs, fmt.Errorf("TOKEN_TTL must be a positive duration like 24h, got %q", getenv("TOKEN_TTL")))
	}

	for origin := range strings.SplitSeq(getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, origin)
		}
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(valueOr(getenv("LOG_LEVEL"), "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}

	return cfg, errors.Join(errs...)
}

// DatabaseURL returns DATABASE_URL if set. Otherwise it builds a URL from
// DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD and DB_SSLMODE, which is
// how ECS hands over the RDS credentials it reads from Secrets Manager.
func DatabaseURL(getenv func(string) string) (string, error) {
	if dsn := getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	host, name, user := getenv("DB_HOST"), getenv("DB_NAME"), getenv("DB_USER")
	if host == "" || name == "" || user == "" {
		return "", errors.New("set DATABASE_URL, or DB_HOST, DB_NAME, DB_USER and DB_PASSWORD")
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, getenv("DB_PASSWORD")),
		Host:     net.JoinHostPort(host, valueOr(getenv("DB_PORT"), "5432")),
		Path:     "/" + name,
		RawQuery: url.Values{"sslmode": {valueOr(getenv("DB_SSLMODE"), "require")}}.Encode(),
	}
	return u.String(), nil
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
