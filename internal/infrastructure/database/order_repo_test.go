//go:build integration

// Order repository transactional tests. Tagged `integration` (like
// migrations_integration_test.go) because OrderRepository.Create runs real SQL
// against PostgreSQL — there is no in-memory seam. Run with:
//
//	go test -tags=integration ./internal/infrastructure/database/ -run Order
//
// Requires a reachable PostgreSQL (the dev docker-compose `postgres` service).
// The default `go test ./internal/infrastructure/database/ -run Order` (no tag)
// compiles the package without this file and finds no Order tests, so it passes.
package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
)

func newOrderTestRepo(t *testing.T) (*OrderRepository, *PostgresDB) {
	t.Helper()
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.RunMigrations(ctx); err != nil {
		db.Close()
		t.Fatalf("migrations: %v", err)
	}
	return NewOrderRepository(db), db
}

// TestOrderRepository_CreateRollsBackOnItemFailure is the regression test for the
// non-transactional Create bug: previously the order INSERT and the per-item
// INSERT loop ran on separate auto-commit calls, so a mid-loop item failure left
// a persisted order whose stored total no longer matched its (missing) items.
// Two items share a primary key so the second item INSERT fails mid-loop; the
// whole order must roll back — no order row and no item rows may persist.
func TestOrderRepository_CreateRollsBackOnItemFailure(t *testing.T) {
	ctx := context.Background()
	repo, db := newOrderTestRepo(t)
	defer db.Close()

	orgID := "org-" + uuid.New().String()
	orderID := "ord-" + uuid.New().String()
	dupItemID := "item-" + uuid.New().String() // deliberately reused to force a PK violation

	order := &entity.Order{
		ID:             orderID,
		OrganizationID: orgID,
		ChannelID:      "ch-" + uuid.New().String(),
		CustomerPhone:  "+5511999999999",
		Status:         entity.OrderStatusPending,
		Subtotal:       3000,
		Total:          3000,
		Currency:       "BRL",
		Items: []entity.OrderItem{
			{ID: dupItemID, ProductID: "p1", ProductName: "First", Quantity: 1, UnitPrice: 1000, TotalPrice: 1000, Currency: "BRL"},
			// Same ID as the first item -> duplicate primary key -> the second
			// INSERT fails partway through the loop.
			{ID: dupItemID, ProductID: "p2", ProductName: "Second", Quantity: 1, UnitPrice: 2000, TotalPrice: 2000, Currency: "BRL"},
		},
	}

	if err := repo.Create(ctx, order); err == nil {
		t.Fatal("expected Create to fail on the duplicate item id")
	}

	var orderCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE id = $1`, orderID).Scan(&orderCount); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 0 {
		t.Fatalf("expected the order to be rolled back, found %d order row(s)", orderCount)
	}

	var itemCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM order_items WHERE order_id = $1`, orderID).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 0 {
		t.Fatalf("expected no items persisted after rollback, found %d", itemCount)
	}
}

// TestOrderRepository_CreatePersistsOrderWithItems checks the happy path still
// commits the order and all its items atomically.
func TestOrderRepository_CreatePersistsOrderWithItems(t *testing.T) {
	ctx := context.Background()
	repo, db := newOrderTestRepo(t)
	defer db.Close()

	orgID := "org-" + uuid.New().String()
	orderID := "ord-" + uuid.New().String()
	order := &entity.Order{
		ID:             orderID,
		OrganizationID: orgID,
		ChannelID:      "ch-" + uuid.New().String(),
		CustomerPhone:  "+5511988887777",
		Status:         entity.OrderStatusPending,
		Subtotal:       3000,
		Total:          3000,
		Currency:       "BRL",
		Items: []entity.OrderItem{
			{ProductID: "p1", ProductName: "First", Quantity: 1, UnitPrice: 1000, TotalPrice: 1000, Currency: "BRL"},
			{ProductID: "p2", ProductName: "Second", Quantity: 1, UnitPrice: 2000, TotalPrice: 2000, Currency: "BRL"},
		},
	}
	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Clean up regardless of assertions (FK ON DELETE CASCADE removes items).
	defer func() { _ = repo.Delete(ctx, orgID, orderID) }()

	got, err := repo.GetByID(ctx, orgID, orderID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Total != 3000 {
		t.Fatalf("expected total 3000, got %d", got.Total)
	}
	items, err := repo.GetOrderItems(ctx, orderID)
	if err != nil {
		t.Fatalf("GetOrderItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 persisted items, got %d", len(items))
	}
}
