//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// seedConversationGraph inserts the minimal tenant/channel/contact rows a
// conversation references and returns their IDs.
func seedConversationGraph(t *testing.T, ctx context.Context, db *PostgresDB) (tenantID, channelID, contactID string) {
	t.Helper()

	tenantID = uuid.New().String()
	channelID = uuid.New().String()
	contactID = uuid.New().String()
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
	return tenantID, channelID, contactID
}

// TestConversationTagsMetadata_RoundTrip is the regression guard for
// WS10-PERSIST-FIELDS: tags and metadata set on a conversation must survive
// Create -> Update -> FindByID instead of being silently dropped by the repo.
func TestConversationTagsMetadata_RoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewConversationRepository(db)
	tenantID, channelID, contactID := seedConversationGraph(t, ctx, db)

	// Create with initial tags + metadata.
	conv := entity.NewConversation(tenantID, contactID, channelID)
	conv.ID = uuid.New().String()
	conv.Tags = []string{"vip", "billing"}
	conv.Metadata = map[string]string{"source": "webchat", "priority_reason": "sla"}

	if err := repo.Create(ctx, conv); err != nil {
		t.Fatalf("create: %v", err)
	}

	created, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find after create: %v", err)
	}
	assertStringSet(t, "tags after create", created.Tags, []string{"vip", "billing"})
	if got := created.Metadata["source"]; got != "webchat" {
		t.Fatalf("metadata source after create: want webchat, got %q", got)
	}
	if got := created.Metadata["priority_reason"]; got != "sla" {
		t.Fatalf("metadata priority_reason after create: want sla, got %q", got)
	}

	// Update tags (the primary fix path) and mutate metadata.
	created.Tags = []string{"vip", "resolved-later", "escalated"}
	created.Metadata["escalation_reason"] = "timeout"
	if err := repo.Update(ctx, created); err != nil {
		t.Fatalf("update: %v", err)
	}

	reloaded, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find after update: %v", err)
	}
	assertStringSet(t, "tags after update", reloaded.Tags,
		[]string{"vip", "resolved-later", "escalated"})
	if got := reloaded.Metadata["escalation_reason"]; got != "timeout" {
		t.Fatalf("metadata escalation_reason after update: want timeout, got %q", got)
	}
	if got := reloaded.Metadata["source"]; got != "webchat" {
		t.Fatalf("metadata source should persist through update: want webchat, got %q", got)
	}
}

// TestConversationTagsMetadata_EmptyDefaults verifies a conversation created
// with no tags/metadata reads back as empty (non-nil) values, never a nil map
// or a scan panic on NULL columns.
func TestConversationTagsMetadata_EmptyDefaults(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewConversationRepository(db)
	tenantID, channelID, contactID := seedConversationGraph(t, ctx, db)

	conv := entity.NewConversation(tenantID, contactID, channelID)
	conv.ID = uuid.New().String()
	if err := repo.Create(ctx, conv); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Fatalf("empty tags expected, got %v", got.Tags)
	}
	if got.Metadata == nil {
		t.Fatalf("metadata must default to a non-nil map")
	}
	if len(got.Metadata) != 0 {
		t.Fatalf("empty metadata expected, got %v", got.Metadata)
	}
}

func assertStringSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: want %v, got %v", label, want, got)
	}
	wantSet := make(map[string]bool, len(want))
	for _, w := range want {
		wantSet[w] = true
	}
	for _, g := range got {
		if !wantSet[g] {
			t.Fatalf("%s: unexpected value %q (want %v, got %v)", label, g, want, got)
		}
	}
}
