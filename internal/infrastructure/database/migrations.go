package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	// Registers the "pgx" database/sql driver used by goose.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// migrationsFS holds the numbered .sql migrations applied after the baseline.
// `all:` keeps the .gitkeep so the embed succeeds while the directory has no
// .sql files yet.
//
//go:embed all:migrations
var migrationsFS embed.FS

const baselineVersion = 1

// baselineMigration provisions the legacy idempotent schema as version 1.
// Because every baseline statement is guarded (IF NOT EXISTS / DO blocks),
// applying it to an existing database is a safe no-op; goose simply records
// the version so future migrations layer on top.
func baselineMigration() *goose.Migration {
	return goose.NewGoMigration(
		baselineVersion,
		&goose.GoFunc{RunTx: upBaseline},
		&goose.GoFunc{RunTx: downBaseline},
	)
}

func upBaseline(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range baselineStatements() {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("baseline migration failed: %w", err)
		}
	}
	return nil
}

// downBaseline drops every application table (children cascade), leaving goose's
// own version table intact. Rolling back the baseline wipes the schema, so it is
// intended for development/test resets only.
func downBaseline(ctx context.Context, tx *sql.Tx) error {
	const dropAll = `
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
    ) LOOP
        EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END $$;`
	if _, err := tx.ExecContext(ctx, dropAll); err != nil {
		return fmt.Errorf("baseline rollback failed: %w", err)
	}
	return nil
}

// runGooseMigrations opens a dedicated database/sql connection (goose requires
// one; the app otherwise uses pgxpool) and applies all pending migrations: the
// Go baseline plus any numbered .sql files under migrations/.
func runGooseMigrations(ctx context.Context, dsn string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer sqlDB.Close()

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("resolve migrations fs: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		sub,
		goose.WithGoMigrations(baselineMigration()),
	)
	if err != nil {
		return fmt.Errorf("build migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
