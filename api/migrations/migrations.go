// Package migrations embeds the SQL schema migrations and applies them.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"hash/fnv"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var files embed.FS

// Up applies every pending migration to the connection's current schema and
// returns how many ran. A Postgres advisory lock makes concurrent runs safe:
// one applies, the rest wait and then find nothing left to do.
func Up(ctx context.Context, db *sql.DB) (int, error) {
	var schema string
	if err := db.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		return 0, fmt.Errorf("read current schema: %w", err)
	}
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockID(lockID(schema)))
	if err != nil {
		return 0, fmt.Errorf("create migration lock: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, files, goose.WithSessionLocker(locker))
	if err != nil {
		return 0, fmt.Errorf("load migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return 0, fmt.Errorf("apply migrations: %w", err)
	}
	return len(results), nil
}

// lockID keys the advisory lock on the schema being migrated, so migrating
// two schemas in the same database (as the tests do) never contends.
func lockID(schema string) int64 {
	h := fnv.New64a()
	h.Write([]byte("fairshare/migrations/" + schema))
	return int64(h.Sum64() >> 1)
}
