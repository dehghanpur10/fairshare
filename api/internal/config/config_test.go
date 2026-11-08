package config_test

import (
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dehghanpur10/fairshare/api/internal/config"
)

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

const secret = "0123456789abcdef0123456789abcdef"

func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{
		"DATABASE_URL": "postgres://localhost/fairshare",
		"JWT_SECRET":   secret,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != ":8080" || cfg.Env != "development" {
		t.Errorf("got addr %q env %q, want :8080 development", cfg.Addr, cfg.Env)
	}
	if cfg.TokenTTL != 24*time.Hour {
		t.Errorf("got ttl %v, want 24h", cfg.TokenTTL)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("got log level %v, want info", cfg.LogLevel)
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("got origins %v, want none", cfg.AllowedOrigins)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{
		"APP_ENV":              "production",
		"HTTP_ADDR":            ":9000",
		"DATABASE_URL":         "postgres://localhost/fairshare",
		"JWT_SECRET":           secret,
		"TOKEN_TTL":            "15m",
		"CORS_ALLOWED_ORIGINS": "https://app.example.com, http://localhost:5173 ,",
		"LOG_LEVEL":            "debug",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TokenTTL != 15*time.Minute || cfg.LogLevel != slog.LevelDebug || cfg.Addr != ":9000" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	want := []string{"https://app.example.com", "http://localhost:5173"}
	if !slices.Equal(cfg.AllowedOrigins, want) {
		t.Errorf("got origins %v, want %v", cfg.AllowedOrigins, want)
	}
}

func TestLoadReportsEveryProblem(t *testing.T) {
	_, err := config.Load(env(map[string]string{
		"JWT_SECRET": "short",
		"TOKEN_TTL":  "soon",
		"LOG_LEVEL":  "loud",
	}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL", "JWT_SECRET", "TOKEN_TTL", "LOG_LEVEL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

func TestDatabaseURLFromParts(t *testing.T) {
	got, err := config.DatabaseURL(env(map[string]string{
		"DB_HOST":     "db.internal",
		"DB_NAME":     "fairshare",
		"DB_USER":     "app",
		"DB_PASSWORD": "p@ss/word",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "postgres://app:p%40ss%2Fword@db.internal:5432/fairshare?sslmode=require"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
