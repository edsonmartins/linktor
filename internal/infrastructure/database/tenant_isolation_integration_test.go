//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	apperrors "github.com/msgfy/linktor/pkg/errors"
)

// seedBareTenant inserts a tenant and returns its id.
func seedBareTenant(t *testing.T, ctx context.Context, db *PostgresDB) string {
	t.Helper()
	id := uuid.New().String()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		id, "test", "t-"+uuid.New().String()[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return id
}

// TestBotRepository_TenantIsolation_DefenseInDepth proves the SQL-layer backstop
// (INV-001): a mutation carrying the WRONG tenant touches nothing, even though
// the row id is valid. This guards against a future caller that skips the
// service *ForTenant ownership check.
func TestBotRepository_TenantIsolation_DefenseInDepth(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewBotRepository(db)
	owner := seedBareTenant(t, ctx, db)
	attacker := seedBareTenant(t, ctx, db)

	bot := entity.NewBot(owner, "Owner Bot", entity.BotTypeAI, entity.AIProviderOpenAI, "gpt-4")
	bot.ID = uuid.New().String()
	if err := repo.Create(ctx, bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	t.Run("cross-tenant Delete deletes nothing", func(t *testing.T) {
		err := repo.Delete(ctx, bot.ID, attacker)
		if err == nil {
			t.Fatal("cross-tenant Delete succeeded, want not-found")
		}
		if !apperrors.IsNotFound(err) {
			t.Fatalf("error = %v, want not-found", err)
		}
		if _, err := repo.FindByID(ctx, bot.ID); err != nil {
			t.Fatalf("bot must still exist after rejected cross-tenant delete: %v", err)
		}
	})

	t.Run("cross-tenant UpdateStatus changes nothing", func(t *testing.T) {
		if err := repo.UpdateStatus(ctx, bot.ID, attacker, entity.BotStatusActive); err == nil {
			t.Fatal("cross-tenant UpdateStatus succeeded, want not-found")
		}
	})

	t.Run("cross-tenant Update changes nothing", func(t *testing.T) {
		// Load the real bot, tamper with a field, but stamp the attacker tenant:
		// the WHERE tenant_id guard must reject it.
		forged := *bot
		forged.TenantID = attacker
		forged.Name = "HACKED"
		if err := repo.Update(ctx, &forged); err == nil {
			t.Fatal("cross-tenant Update succeeded, want not-found")
		}
		got, err := repo.FindByID(ctx, bot.ID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if got.Name != "Owner Bot" {
			t.Fatalf("bot name was tampered cross-tenant: %q", got.Name)
		}
	})

	t.Run("same-tenant operations still work", func(t *testing.T) {
		if err := repo.UpdateStatus(ctx, bot.ID, owner, entity.BotStatusActive); err != nil {
			t.Fatalf("same-tenant UpdateStatus should succeed: %v", err)
		}
		if err := repo.Delete(ctx, bot.ID, owner); err != nil {
			t.Fatalf("same-tenant Delete should succeed: %v", err)
		}
	})
}

// TestUserRepository_TenantIsolation_DefenseInDepth mirrors the above for users:
// an admin of tenant A cannot delete/update a user of tenant B even by id.
func TestUserRepository_TenantIsolation_DefenseInDepth(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewUserRepository(db)
	owner := seedBareTenant(t, ctx, db)
	attacker := seedBareTenant(t, ctx, db)

	user := entity.NewUser(owner, "owner-"+uuid.New().String()[:8]+"@test.local", "hash", "Owner", entity.UserRoleAgent)
	user.ID = uuid.New().String()
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	t.Run("cross-tenant Delete deletes nothing", func(t *testing.T) {
		if err := repo.Delete(ctx, user.ID, attacker); err == nil {
			t.Fatal("cross-tenant Delete succeeded, want not-found")
		}
		if _, err := repo.FindByID(ctx, user.ID); err != nil {
			t.Fatalf("user must still exist after rejected cross-tenant delete: %v", err)
		}
	})

	t.Run("cross-tenant Update changes nothing", func(t *testing.T) {
		forged := *user
		forged.TenantID = attacker
		forged.Name = "HACKED"
		if err := repo.Update(ctx, &forged); err == nil {
			t.Fatal("cross-tenant Update succeeded, want not-found")
		}
		got, err := repo.FindByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if got.Name != "Owner" {
			t.Fatalf("user name was tampered cross-tenant: %q", got.Name)
		}
	})

	t.Run("same-tenant Delete works", func(t *testing.T) {
		if err := repo.Delete(ctx, user.ID, owner); err != nil {
			t.Fatalf("same-tenant Delete should succeed: %v", err)
		}
	})
}
