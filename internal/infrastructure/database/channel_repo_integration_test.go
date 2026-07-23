//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	apperrors "github.com/msgfy/linktor/pkg/errors"
)

func seedTenant(t *testing.T, ctx context.Context, db *PostgresDB) string {
	t.Helper()
	tenantID := uuid.New().String()
	slug := "t-" + uuid.New().String()[:8]
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, "test", slug); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenantID
}

func newTestChannel(tenantID string, env entity.ChannelEnvironment) *entity.Channel {
	ch := entity.NewChannel(tenantID, entity.ChannelTypeWhatsAppOfficial, "env-test", "+5511999999999")
	ch.ID = uuid.New().String()
	ch.Environment = env
	return ch
}

// TestChannelEnvironment_RoundTrip verifies environment survives the full
// create → get → update → get cycle against a real database. This guards
// against the known failure mode of a column existing in the schema while the
// repository silently drops it (see source/is_imported precedent on messages).
func TestChannelEnvironment_RoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewChannelRepository(db, nil)
	tenantID := seedTenant(t, ctx, db)

	t.Run("default is production", func(t *testing.T) {
		ch := newTestChannel(tenantID, "") // caller did not choose
		if err := repo.Create(ctx, ch); err != nil {
			t.Fatalf("create: %v", err)
		}
		got, err := repo.FindByID(ctx, ch.ID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if got.Environment != entity.ChannelEnvironmentProduction {
			t.Fatalf("environment = %q, want production", got.Environment)
		}
	})

	t.Run("sandbox survives create-get-update-get", func(t *testing.T) {
		ch := newTestChannel(tenantID, entity.ChannelEnvironmentSandbox)
		if err := repo.Create(ctx, ch); err != nil {
			t.Fatalf("create: %v", err)
		}

		got, err := repo.FindByID(ctx, ch.ID)
		if err != nil {
			t.Fatalf("find after create: %v", err)
		}
		if got.Environment != entity.ChannelEnvironmentSandbox {
			t.Fatalf("environment after create = %q, want sandbox", got.Environment)
		}

		// Update mutating other fields — and adversarially flipping the struct's
		// Environment — must leave the persisted environment untouched: the
		// repository's UPDATE deliberately omits the column.
		got.Name = "renamed"
		got.Environment = entity.ChannelEnvironmentProduction
		if err := repo.Update(ctx, got); err != nil {
			t.Fatalf("update: %v", err)
		}

		got2, err := repo.FindByID(ctx, ch.ID)
		if err != nil {
			t.Fatalf("find after update: %v", err)
		}
		if got2.Name != "renamed" {
			t.Fatalf("name = %q, want renamed", got2.Name)
		}
		if got2.Environment != entity.ChannelEnvironmentSandbox {
			t.Fatalf("environment after update = %q, want sandbox (immutable)", got2.Environment)
		}
	})

	t.Run("listing carries environment", func(t *testing.T) {
		channels, _, err := repo.FindByTenant(ctx, tenantID, nil)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		seen := map[entity.ChannelEnvironment]bool{}
		for _, c := range channels {
			seen[c.Environment] = true
		}
		if !seen[entity.ChannelEnvironmentSandbox] || !seen[entity.ChannelEnvironmentProduction] {
			t.Fatalf("listing lost environments, saw %v", seen)
		}
	})
}

// TestChannelEnvironment_InvalidRejectedAtDomainEdge verifies an out-of-set
// value is rejected by validation before any persistence, not by the database.
func TestChannelEnvironment_InvalidRejectedAtDomainEdge(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewChannelRepository(db, nil)
	tenantID := seedTenant(t, ctx, db)

	ch := newTestChannel(tenantID, entity.ChannelEnvironment("staging"))
	err = repo.Create(ctx, ch)
	if err == nil {
		t.Fatal("create with invalid environment succeeded, want validation error")
	}
	if !apperrors.IsValidation(err) {
		t.Fatalf("error = %v, want validation error", err)
	}

	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM channels WHERE id = $1`, ch.ID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("invalid channel was persisted (count=%d)", count)
	}
}

// TestChannelEnvironment_PreexistingRowsAreProduction simulates a channel row
// written before the environment column existed (raw insert relying on the
// column default) and verifies it reads back as production.
func TestChannelEnvironment_PreexistingRowsAreProduction(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewChannelRepository(db, nil)
	tenantID := seedTenant(t, ctx, db)

	channelID := uuid.New().String()
	now := time.Now()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO channels (id, tenant_id, name, type, enabled, connection_status, credentials, config, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, true, 'disconnected', '{}', '{}', $5, $5)`,
		channelID, tenantID, "legacy", "whatsapp_official", now); err != nil {
		t.Fatalf("raw insert: %v", err)
	}

	got, err := repo.FindByID(ctx, channelID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Environment != entity.ChannelEnvironmentProduction {
		t.Fatalf("environment = %q, want production", got.Environment)
	}
}
