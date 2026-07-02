//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// seedConversation creates the minimal tenant/channel/contact/conversation graph
// required to insert messages, returning the conversation ID.
func seedConversation(t *testing.T, ctx context.Context, db *PostgresDB) string {
	t.Helper()

	tenantID := uuid.New().String()
	channelID := uuid.New().String()
	contactID := uuid.New().String()
	convID := uuid.New().String()
	slug := "t-" + uuid.New().String()[:8]

	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, "test", slug); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO channels (id, tenant_id, name, type) VALUES ($1, $2, $3, $4)`,
		channelID, tenantID, "ch", "whatsapp"); err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO contacts (id, tenant_id, name) VALUES ($1, $2, $3)`,
		contactID, tenantID, "contact"); err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO conversations (id, tenant_id, channel_id, contact_id) VALUES ($1, $2, $3, $4)`,
		convID, tenantID, channelID, contactID); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	return convID
}

func insertMessage(t *testing.T, ctx context.Context, repo *MessageRepository, convID string, sender entity.SenderType, status entity.MessageStatus, createdAt time.Time) *entity.Message {
	t.Helper()
	m := &entity.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		SenderType:     sender,
		ContentType:    entity.ContentTypeText,
		Content:        "hi",
		Metadata:       map[string]string{},
		Status:         status,
		CreatedAt:      createdAt,
	}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create message: %v", err)
	}
	return m
}

// TestUpdateStatus_DoesNotRegress verifies a stale "sent" event cannot overwrite
// a message already marked "delivered" / "read".
func TestUpdateStatus_DoesNotRegress(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewMessageRepository(db)
	convID := seedConversation(t, ctx, db)
	m := insertMessage(t, ctx, repo, convID, entity.SenderTypeUser, entity.MessageStatusSent, time.Now())

	// Advance forward: sent -> delivered -> read.
	if err := repo.UpdateStatus(ctx, m.ID, entity.MessageStatusDelivered, ""); err != nil {
		t.Fatalf("to delivered: %v", err)
	}
	if err := repo.UpdateStatus(ctx, m.ID, entity.MessageStatusRead, ""); err != nil {
		t.Fatalf("to read: %v", err)
	}

	// A redelivered "sent" must be a no-op (no error, no regression).
	if err := repo.UpdateStatus(ctx, m.ID, entity.MessageStatusSent, ""); err != nil {
		t.Fatalf("stale sent should be a benign no-op, got: %v", err)
	}

	got, err := repo.FindByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Status != entity.MessageStatusRead {
		t.Fatalf("status regressed: want read, got %s", got.Status)
	}
}

// TestMarkAsRead_IgnoresOutbound verifies only inbound (contact) messages get
// read receipts; the agent's own outbound messages are left untouched.
func TestMarkAsRead_IgnoresOutbound(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewMessageRepository(db)
	convID := seedConversation(t, ctx, db)

	base := time.Now().Add(-time.Hour)
	inbound := insertMessage(t, ctx, repo, convID, entity.SenderTypeContact, entity.MessageStatusDelivered, base)
	outbound := insertMessage(t, ctx, repo, convID, entity.SenderTypeUser, entity.MessageStatusDelivered, base.Add(time.Minute))

	if err := repo.MarkAsRead(ctx, convID, outbound.ID); err != nil {
		t.Fatalf("mark as read: %v", err)
	}

	gotIn, _ := repo.FindByID(ctx, inbound.ID)
	if gotIn.Status != entity.MessageStatusRead {
		t.Fatalf("inbound message should be read, got %s", gotIn.Status)
	}
	gotOut, _ := repo.FindByID(ctx, outbound.ID)
	if gotOut.Status == entity.MessageStatusRead {
		t.Fatalf("outbound message must NOT be marked read (corrupts delivery metrics)")
	}
}

// TestFindByTenant_FilteredTotalMatchesFilter verifies the paginated total honors
// the status filter (COUNT applies the same predicate as the page query).
func TestFindByTenant_FilteredTotalMatchesFilter(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed one tenant with 3 conversations: 2 open, 1 resolved.
	tenantID := uuid.New().String()
	channelID := uuid.New().String()
	contactID := uuid.New().String()
	slug := "t-" + uuid.New().String()[:8]
	if _, err := db.Pool.Exec(ctx, `INSERT INTO tenants (id, name, slug) VALUES ($1,$2,$3)`, tenantID, "t", slug); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO channels (id, tenant_id, name, type) VALUES ($1,$2,$3,$4)`, channelID, tenantID, "c", "whatsapp"); err != nil {
		t.Fatalf("channel: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO contacts (id, tenant_id, name) VALUES ($1,$2,$3)`, contactID, tenantID, "x"); err != nil {
		t.Fatalf("contact: %v", err)
	}
	for _, st := range []string{"open", "open", "resolved"} {
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO conversations (id, tenant_id, channel_id, contact_id, status) VALUES ($1,$2,$3,$4,$5)`,
			uuid.New().String(), tenantID, channelID, contactID, st); err != nil {
			t.Fatalf("conversation: %v", err)
		}
	}

	repo := NewConversationRepository(db)
	params := NewListParams()
	params.Filters["status"] = "open"

	convs, total, err := repo.FindByTenant(ctx, tenantID, params)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if total != 2 {
		t.Fatalf("filtered total should be 2 (open only), got %d", total)
	}
	if len(convs) != 2 {
		t.Fatalf("page should hold 2 open conversations, got %d", len(convs))
	}
}
