//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// seedTenantWithContacts creates a tenant and n contacts, returning the tenant
// ID and the contact IDs. Used to exercise the tenant-scoped identity uniqueness
// index introduced by migration 00007.
func seedTenantWithContacts(t *testing.T, ctx context.Context, db *PostgresDB, n int) (string, []string) {
	t.Helper()
	tenantID := uuid.New().String()
	slug := "t-" + uuid.New().String()[:8]
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, "test", slug); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		cid := uuid.New().String()
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO contacts (id, tenant_id, name) VALUES ($1, $2, $3)`,
			cid, tenantID, "contact"); err != nil {
			t.Fatalf("seed contact: %v", err)
		}
		ids = append(ids, cid)
	}
	return tenantID, ids
}

// TestCreateIdentityIfAbsent_TenantUniqueness proves the WS10-GETORCREATE fix:
// the partial unique index on (tenant_id, channel_type, identifier) blocks two
// different contacts in the same tenant from claiming the same identity, and
// CreateIdentityIfAbsent reports inserted=false (not an error) on that conflict.
func TestCreateIdentityIfAbsent_TenantUniqueness(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewContactRepository(db)
	tenantID, contacts := seedTenantWithContacts(t, ctx, db, 2)

	ident := func(contactID string) *entity.ContactIdentity {
		return &entity.ContactIdentity{
			ID:          uuid.New().String(),
			ContactID:   contactID,
			TenantID:    tenantID,
			ChannelType: "whatsapp",
			Identifier:  "+5511888888888",
			Metadata:    map[string]string{},
			CreatedAt:   time.Now(),
		}
	}

	// First contact wins the identity.
	inserted, err := repo.CreateIdentityIfAbsent(ctx, ident(contacts[0]))
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !inserted {
		t.Fatal("expected first CreateIdentityIfAbsent to insert (true)")
	}

	// A different contact in the same tenant loses: no error, inserted=false.
	inserted, err = repo.CreateIdentityIfAbsent(ctx, ident(contacts[1]))
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if inserted {
		t.Fatal("expected second CreateIdentityIfAbsent to be a no-op (false)")
	}

	// Only one identity row exists for the (tenant, channel_type, identifier).
	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contact_identities WHERE tenant_id = $1 AND channel_type = $2 AND identifier = $3`,
		tenantID, "whatsapp", "+5511888888888").Scan(&count); err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 identity row, got %d", count)
	}

	// FindByIdentity resolves to the winning contact.
	c, err := repo.FindByIdentity(ctx, tenantID, "whatsapp", "+5511888888888")
	if err != nil {
		t.Fatalf("find by identity: %v", err)
	}
	if c.ID != contacts[0] {
		t.Fatalf("expected winner %s, got %s", contacts[0], c.ID)
	}
}

// TestCreateIdentityIfAbsent_DifferentTenants confirms the uniqueness is scoped
// per tenant: the same (channel_type, identifier) may coexist across tenants.
func TestCreateIdentityIfAbsent_DifferentTenants(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewContactRepository(db)
	tenantA, contactsA := seedTenantWithContacts(t, ctx, db, 1)
	tenantB, contactsB := seedTenantWithContacts(t, ctx, db, 1)

	mk := func(tenantID, contactID string) *entity.ContactIdentity {
		return &entity.ContactIdentity{
			ID:          uuid.New().String(),
			ContactID:   contactID,
			TenantID:    tenantID,
			ChannelType: "whatsapp",
			Identifier:  "+5511777777777",
			Metadata:    map[string]string{},
			CreatedAt:   time.Now(),
		}
	}

	if ins, err := repo.CreateIdentityIfAbsent(ctx, mk(tenantA, contactsA[0])); err != nil || !ins {
		t.Fatalf("tenant A insert: ins=%v err=%v", ins, err)
	}
	if ins, err := repo.CreateIdentityIfAbsent(ctx, mk(tenantB, contactsB[0])); err != nil || !ins {
		t.Fatalf("tenant B insert should succeed (different tenant): ins=%v err=%v", ins, err)
	}
}
