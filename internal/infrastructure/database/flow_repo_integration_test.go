//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// seedFlowTenant inserts the minimal tenant row a flow's FK requires.
func seedFlowTenant(t *testing.T, ctx context.Context, db *PostgresDB) string {
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

// TestFlowRepo_MetaFlowIDRoundTrip verifies the Meta-side flow ID survives a
// create/read/list/update cycle (the bug: it was never persisted).
func TestFlowRepo_MetaFlowIDRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	tenantID := seedFlowTenant(t, ctx, db)
	repo := NewFlowRepository(db)

	flow := entity.NewFlow(tenantID, "meta-flow", entity.FlowTriggerManual, "")
	flow.ID = uuid.New().String()
	flow.MetaFlowID = "META-XYZ-1"
	if err := repo.Create(ctx, flow); err != nil {
		t.Fatalf("create: %v", err)
	}

	// FindByID must read the persisted Meta ID.
	got, err := repo.FindByID(ctx, flow.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got.MetaFlowID != "META-XYZ-1" {
		t.Fatalf("FindByID MetaFlowID = %q, want %q", got.MetaFlowID, "META-XYZ-1")
	}

	// FindByTenant (scanFlowFromRows path) must read it too.
	list, _, err := repo.FindByTenant(ctx, tenantID, nil, &repository.ListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("find by tenant: %v", err)
	}
	if len(list) != 1 || list[0].MetaFlowID != "META-XYZ-1" {
		t.Fatalf("FindByTenant did not return persisted MetaFlowID: %+v", list)
	}

	// Update must persist a changed Meta ID.
	got.MetaFlowID = "META-XYZ-2"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	reloaded, err := repo.FindByID(ctx, flow.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.MetaFlowID != "META-XYZ-2" {
		t.Fatalf("after update MetaFlowID = %q, want %q", reloaded.MetaFlowID, "META-XYZ-2")
	}
}
