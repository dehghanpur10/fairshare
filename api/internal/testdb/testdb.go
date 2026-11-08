// Package testdb gives each test its own freshly migrated Postgres schema.
package testdb

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/dehghanpur10/fairshare/api/internal/store"
	"github.com/dehghanpur10/fairshare/api/migrations"
)

// New returns a pool whose search_path points at a new schema with every
// migration applied. The schema is dropped when the test finishes, so tests
// can run in parallel without seeing each other's rows. The test is skipped
// when TEST_DATABASE_URL is not set.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := t.Context()

	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	exec(t, dsn, "CREATE SCHEMA "+schema)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := store.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		exec(t, dsn, "DROP SCHEMA "+schema+" CASCADE")
	})

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	if _, err := migrations.Up(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func exec(t *testing.T, dsn, sql string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
