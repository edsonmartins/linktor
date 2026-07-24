//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
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

// ---------------------------------------------------------------------------
// Sandbox allowlist repository
// ---------------------------------------------------------------------------

func TestSandboxAllowlistRepository(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewSandboxAllowlistRepository(db)
	channelRepo := NewChannelRepository(db, nil)
	tenantA := seedTenant(t, ctx, db)
	tenantB := seedTenant(t, ctx, db)

	sandboxCh := newTestChannel(tenantA, entity.ChannelEnvironmentSandbox)
	if err := channelRepo.Create(ctx, sandboxCh); err != nil {
		t.Fatalf("create channel: %v", err)
	}
	otherCh := newTestChannel(tenantA, entity.ChannelEnvironmentSandbox)
	if err := channelRepo.Create(ctx, otherCh); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	tenantWide := &entity.SandboxAllowlistEntry{
		ID: uuid.New().String(), TenantID: tenantA,
		Recipient: "+5544999999999", CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, tenantWide); err != nil {
		t.Fatalf("create tenant-wide entry: %v", err)
	}
	channelScoped := &entity.SandboxAllowlistEntry{
		ID: uuid.New().String(), TenantID: tenantA, ChannelID: sandboxCh.ID,
		Recipient: "+5544888888888", CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, channelScoped); err != nil {
		t.Fatalf("create channel-scoped entry: %v", err)
	}

	t.Run("duplicate tenant-wide entry conflicts", func(t *testing.T) {
		dup := &entity.SandboxAllowlistEntry{
			ID: uuid.New().String(), TenantID: tenantA,
			Recipient: "+5544999999999", CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, dup)
		if err == nil {
			t.Fatal("duplicate entry accepted, want conflict")
		}
	})

	t.Run("IsAllowed tenant-wide entry covers every channel", func(t *testing.T) {
		for _, chID := range []string{sandboxCh.ID, otherCh.ID} {
			allowed, err := repo.IsAllowed(ctx, tenantA, chID, "+5544999999999")
			if err != nil {
				t.Fatalf("IsAllowed: %v", err)
			}
			if !allowed {
				t.Fatalf("tenant-wide recipient not allowed on channel %s", chID)
			}
		}
	})

	t.Run("IsAllowed channel-scoped entry covers only its channel", func(t *testing.T) {
		allowed, err := repo.IsAllowed(ctx, tenantA, sandboxCh.ID, "+5544888888888")
		if err != nil {
			t.Fatalf("IsAllowed: %v", err)
		}
		if !allowed {
			t.Fatal("channel-scoped recipient not allowed on its own channel")
		}
		allowed, err = repo.IsAllowed(ctx, tenantA, otherCh.ID, "+5544888888888")
		if err != nil {
			t.Fatalf("IsAllowed: %v", err)
		}
		if allowed {
			t.Fatal("channel-scoped recipient must not be allowed on another channel")
		}
	})

	t.Run("IsAllowed is tenant-scoped", func(t *testing.T) {
		allowed, err := repo.IsAllowed(ctx, tenantB, sandboxCh.ID, "+5544999999999")
		if err != nil {
			t.Fatalf("IsAllowed: %v", err)
		}
		if allowed {
			t.Fatal("another tenant's entry must never authorize a recipient")
		}
	})

	t.Run("cross-tenant read and delete are rejected", func(t *testing.T) {
		if _, err := repo.FindByID(ctx, tenantB, tenantWide.ID); err == nil {
			t.Fatal("cross-tenant FindByID succeeded, want not-found")
		}
		if err := repo.Delete(ctx, tenantB, tenantWide.ID); err == nil {
			t.Fatal("cross-tenant Delete succeeded, want not-found")
		}
		entries, err := repo.FindByTenant(ctx, tenantA)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("entries = %d, want 2 (cross-tenant delete must not remove)", len(entries))
		}
	})

	t.Run("delete removes and takes effect on next IsAllowed", func(t *testing.T) {
		if err := repo.Delete(ctx, tenantA, tenantWide.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		allowed, err := repo.IsAllowed(ctx, tenantA, sandboxCh.ID, "+5544999999999")
		if err != nil {
			t.Fatalf("IsAllowed: %v", err)
		}
		if allowed {
			t.Fatal("removed recipient still allowed")
		}
	})
}

// TestConversationEnvironment_RoundTrip guards INV-018's persistence leg: the
// denormalized environment must survive create → read against a real database
// (same column-not-persisted failure mode as WP1).
func TestConversationEnvironment_RoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	channelRepo := NewChannelRepository(db, nil)
	convRepo := NewConversationRepository(db)
	tenantID := seedTenant(t, ctx, db)

	ch := newTestChannel(tenantID, entity.ChannelEnvironmentSandbox)
	if err := channelRepo.Create(ctx, ch); err != nil {
		t.Fatalf("create channel: %v", err)
	}
	contactID := uuid.New().String()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO contacts (id, tenant_id, name) VALUES ($1, $2, $3)`,
		contactID, tenantID, "contact"); err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	conv := &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   ch.ID,
		ContactID:   contactID,
		Environment: entity.ChannelEnvironmentSandbox,
		Status:      entity.ConversationStatusOpen,
		Priority:    entity.ConversationPriorityNormal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := convRepo.Create(ctx, conv); err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	got, err := convRepo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Environment != entity.ChannelEnvironmentSandbox {
		t.Fatalf("environment = %q, want sandbox", got.Environment)
	}

	// Legacy row (raw insert without environment) reads back as production.
	legacyID := uuid.New().String()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO conversations (id, tenant_id, channel_id, contact_id) VALUES ($1, $2, $3, $4)`,
		legacyID, tenantID, ch.ID, contactID); err != nil {
		t.Fatalf("raw insert: %v", err)
	}
	legacy, err := convRepo.FindByID(ctx, legacyID)
	if err != nil {
		t.Fatalf("find legacy: %v", err)
	}
	if legacy.Environment != entity.ChannelEnvironmentProduction {
		t.Fatalf("legacy environment = %q, want production", legacy.Environment)
	}
}

// TestMessageRepository_LastInboundAt covers the durable source of the 24h
// window policy: only contact messages count, and no history returns nil.
func TestMessageRepository_LastInboundAt(t *testing.T) {
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

	last, err := repo.LastInboundAt(ctx, convID)
	if err != nil {
		t.Fatalf("LastInboundAt: %v", err)
	}
	if last != nil {
		t.Fatalf("expected nil for a conversation with no inbound, got %v", last)
	}

	inboundAt := time.Now().Add(-2 * time.Hour).Truncate(time.Millisecond)
	insertMessage(t, ctx, repo, convID, entity.SenderTypeContact, entity.MessageStatusDelivered, inboundAt)
	insertMessage(t, ctx, repo, convID, entity.SenderTypeUser, entity.MessageStatusSent, time.Now())

	last, err = repo.LastInboundAt(ctx, convID)
	if err != nil {
		t.Fatalf("LastInboundAt: %v", err)
	}
	if last == nil {
		t.Fatal("expected inbound timestamp, got nil")
	}
	if !last.Equal(inboundAt) {
		t.Fatalf("last inbound = %v, want %v (agent message must not count)", last, inboundAt)
	}
}

// TestConversationEnvironmentFilter verifies the listing filter runs in the
// database (WP-H: filtering must happen in the backend, never client-side).
func TestConversationEnvironmentFilter(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	channelRepo := NewChannelRepository(db, nil)
	convRepo := NewConversationRepository(db)
	tenantID := seedTenant(t, ctx, db)
	contactID := uuid.New().String()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO contacts (id, tenant_id, name) VALUES ($1, $2, $3)`,
		contactID, tenantID, "filter-contact"); err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	mkConv := func(env entity.ChannelEnvironment) {
		ch := newTestChannel(tenantID, env)
		if err := channelRepo.Create(ctx, ch); err != nil {
			t.Fatalf("seed channel: %v", err)
		}
		conv := &entity.Conversation{
			ID: uuid.New().String(), TenantID: tenantID, ChannelID: ch.ID,
			ContactID: contactID, Environment: env,
			Status: entity.ConversationStatusOpen, Priority: entity.ConversationPriorityNormal,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := convRepo.Create(ctx, conv); err != nil {
			t.Fatalf("seed conversation: %v", err)
		}
	}
	mkConv(entity.ChannelEnvironmentSandbox)
	mkConv(entity.ChannelEnvironmentProduction)
	mkConv(entity.ChannelEnvironmentProduction)

	list := func(env string) int {
		params := repository.NewListParams()
		if env != "" {
			params.Filters["environment"] = env
		}
		convs, _, err := convRepo.FindByTenant(ctx, tenantID, params)
		if err != nil {
			t.Fatalf("list env=%q: %v", env, err)
		}
		for _, c := range convs {
			if env != "" && string(c.Environment) != env {
				t.Fatalf("filter env=%q returned conversation with environment %q", env, c.Environment)
			}
		}
		return len(convs)
	}

	if n := list(""); n != 3 {
		t.Fatalf("no filter: %d conversations, want 3", n)
	}
	if n := list("sandbox"); n != 1 {
		t.Fatalf("sandbox filter: %d conversations, want 1", n)
	}
	if n := list("production"); n != 2 {
		t.Fatalf("production filter: %d conversations, want 2", n)
	}
}

// TestMarkFailedWithBlockedReason verifies the guard-block reason lands in
// metadata.blocked_by (WP-K), distinguishable from a plain provider failure.
func TestMarkFailedWithBlockedReason(t *testing.T) {
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
	msg := insertMessage(t, ctx, repo, convID, entity.SenderTypeUser, entity.MessageStatusPending, time.Now())

	if err := repo.MarkFailedWithBlockedReason(ctx, msg.ID,
		"sandbox guard: recipient +55*******99 is not in the tenant's sandbox allowlist",
		"allowlist"); err != nil {
		t.Fatalf("mark blocked: %v", err)
	}

	got, err := repo.FindByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Status != entity.MessageStatusFailed {
		t.Fatalf("status = %q, want failed", got.Status)
	}
	if got.Metadata["blocked_by"] != "allowlist" {
		t.Fatalf("metadata.blocked_by = %q, want allowlist", got.Metadata["blocked_by"])
	}
	// Pre-existing metadata keys survive the jsonb_set merge.
	msg2 := &entity.Message{
		ID: uuid.New().String(), ConversationID: convID, SenderType: entity.SenderTypeUser,
		ContentType: entity.ContentTypeText, Content: "x",
		Metadata: map[string]string{"existing": "kept"}, Status: entity.MessageStatusPending,
		CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, msg2); err != nil {
		t.Fatalf("create msg2: %v", err)
	}
	if err := repo.MarkFailedWithBlockedReason(ctx, msg2.ID, "err", "window_24h"); err != nil {
		t.Fatalf("mark msg2: %v", err)
	}
	got2, _ := repo.FindByID(ctx, msg2.ID)
	if got2.Metadata["existing"] != "kept" || got2.Metadata["blocked_by"] != "window_24h" {
		t.Fatalf("metadata merge lost keys: %+v", got2.Metadata)
	}
}

// TestAuditLogFilters covers the WP-L additions: actor (substring across
// email/name/id) and period (created_at range).
func TestAuditLogFilters(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewAuditLogRepository(db)
	tenantID := seedTenant(t, ctx, db)

	insert := func(actorEmail, action string, at time.Time) {
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO audit_logs (id, tenant_id, actor_id, actor_email, action, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.New().String(), tenantID, uuid.New().String(), actorEmail, action, at); err != nil {
			t.Fatalf("insert audit: %v", err)
		}
	}
	now := time.Now().UTC()
	insert("alice@acme.com", "channel.create", now.Add(-48*time.Hour))
	insert("bob@acme.com", "channel.update", now.Add(-1*time.Hour))
	insert("alice@acme.com", "sandbox_allowlist.add", now.Add(-30*time.Minute))

	count := func(f *repository.AuditLogFilters) int64 {
		_, total, err := repo.FindByTenant(ctx, tenantID, f, nil)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		return total
	}

	if got := count(&repository.AuditLogFilters{Actor: "alice"}); got != 2 {
		t.Fatalf("actor=alice: %d, want 2", got)
	}
	if got := count(&repository.AuditLogFilters{Action: "channel.create"}); got != 1 {
		t.Fatalf("action filter: %d, want 1", got)
	}
	if got := count(&repository.AuditLogFilters{StartTime: now.Add(-2 * time.Hour)}); got != 2 {
		t.Fatalf("start filter: %d, want 2", got)
	}
	if got := count(&repository.AuditLogFilters{
		Actor:     "alice",
		StartTime: now.Add(-2 * time.Hour),
	}); got != 1 {
		t.Fatalf("actor+period: %d, want 1", got)
	}
}
