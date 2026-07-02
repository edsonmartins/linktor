//go:build integration

package database

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/pressly/goose/v3"
)

// Run with: LINKTOR_TEST_DB_DSN unused — uses discrete env vars below.
//
//	go test -tags=integration ./internal/infrastructure/database/ -run TestMigrations
//
// Requires a reachable PostgreSQL (the dev docker-compose `postgres` service).
func testDBConfig(t *testing.T) *config.DatabaseConfig {
	t.Helper()
	getenv := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	return &config.DatabaseConfig{
		Host:         getenv("LINKTOR_DATABASE_HOST", "localhost"),
		Port:         5432,
		User:         getenv("LINKTOR_DATABASE_USER", "linktor"),
		Password:     getenv("LINKTOR_DATABASE_PASSWORD", "linktor"),
		Database:     getenv("LINKTOR_DATABASE_DATABASE", "linktor"),
		SSLMode:      "disable",
		MaxOpenConns: 5,
		MaxIdleConns: 1,
		MaxLifetime:  5,
	}
}

func TestMigrations_FreshAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()

	// Start from a clean schema so we exercise the fresh-database path.
	if _, err := db.Pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}

	// First run: builds the full schema and records the baseline version.
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("first migration: %v", err)
	}

	// Second run: must be a no-op (idempotent) and not error.
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("second migration (idempotency): %v", err)
	}

	// goose recorded the baseline version.
	var maxVersion int64
	if err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version`).Scan(&maxVersion); err != nil {
		t.Fatalf("read goose version: %v", err)
	}
	if maxVersion < baselineVersion {
		t.Fatalf("expected goose version >= %d, got %d", baselineVersion, maxVersion)
	}

	// Spot-check a representative set of tables exists.
	for _, table := range []string{"tenants", "users", "channels", "messages", "campaigns", "audit_logs", "roles"} {
		var exists bool
		if err := db.Pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
			table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist after migration", table)
		}
	}
}

func TestMigrations_Rollback(t *testing.T) {
	ctx := context.Background()
	cfg := testDBConfig(t)
	db, err := NewPostgresDB(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()

	if _, err := db.Pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	// Roll the baseline back via a goose provider over the same DSN.
	sqlDB, err := sql.Open("pgx", db.dsn)
	if err != nil {
		t.Fatalf("open sql.DB: %v", err)
	}
	defer sqlDB.Close()
	sub, _ := fs.Sub(migrationsFS, "migrations")
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, sub,
		goose.WithGoMigrations(baselineMigration()))
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.Down(ctx); err != nil {
		t.Fatalf("migrate down: %v", err)
	}

	// App tables are gone; goose's bookkeeping table remains.
	var tenantsExists bool
	if err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='tenants')`).
		Scan(&tenantsExists); err != nil {
		t.Fatalf("check tenants: %v", err)
	}
	if tenantsExists {
		t.Fatal("expected tenants table to be dropped after rollback")
	}

	// Up again must rebuild cleanly (proves down/up cycle is sound).
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("re-migrate up after rollback: %v", err)
	}
}
