// Command fairshare runs the Fairshare API.
//
// Usage:
//
//	fairshare serve     start the HTTP server (default)
//	fairshare migrate   apply pending database migrations and exit
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/dehghanpur10/fairshare/api/internal/auth"
	"github.com/dehghanpur10/fairshare/api/internal/config"
	"github.com/dehghanpur10/fairshare/api/internal/httpapi"
	"github.com/dehghanpur10/fairshare/api/internal/service"
	"github.com/dehghanpur10/fairshare/api/internal/store"
	"github.com/dehghanpur10/fairshare/api/migrations"
)

// shutdownTimeout must stay below the ECS stop timeout (30s by default) so
// in-flight requests finish before the task is killed.
const shutdownTimeout = 20 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "fairshare:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "serve":
		return serve(ctx, getenv, stdout)
	case "migrate":
		return migrate(ctx, getenv, stdout)
	default:
		return fmt.Errorf("unknown command %q, want serve or migrate", cmd)
	}
}

func serve(ctx context.Context, getenv func(string) string, stdout io.Writer) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := newLogger(stdout, cfg.Env, cfg.LogLevel)

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	st := store.New(pool)

	svc, err := service.New(st, auth.NewHasher(auth.DefaultCost), auth.NewTokens(cfg.JWTSecret, cfg.TokenTTL))
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(svc, st, logger, cfg.AllowedOrigins),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "env", cfg.Env)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down", "timeout", shutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("stopped")
	return nil
}

func migrate(ctx context.Context, getenv func(string) string, stdout io.Writer) error {
	dsn, err := config.DatabaseURL(getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := newLogger(stdout, getenv("APP_ENV"), slog.LevelInfo)

	pool, err := store.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	applied, err := migrations.Up(ctx, db)
	if err != nil {
		return err
	}
	logger.Info("migrations applied", "count", applied)
	return nil
}

// newLogger logs JSON in production, where CloudWatch can index the fields,
// and readable text everywhere else.
func newLogger(w io.Writer, env string, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if env == "production" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
