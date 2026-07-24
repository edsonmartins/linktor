package outbound

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/metrics"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

// fakeChecker is a mutable in-memory AllowlistChecker. Entries are normalized
// E.164 recipients. Calls counts consultations so tests can prove the guard
// queries at send time instead of capturing state.
type fakeChecker struct {
	mu      sync.Mutex
	allowed map[string]bool
	err     error
	Calls   int
}

func newFakeChecker(recipients ...string) *fakeChecker {
	f := &fakeChecker{allowed: map[string]bool{}}
	for _, r := range recipients {
		f.allowed[r] = true
	}
	return f
}

func (f *fakeChecker) remove(recipient string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.allowed, recipient)
}

func (f *fakeChecker) IsAllowed(ctx context.Context, tenantID, channelID, recipient string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	if f.err != nil {
		return false, f.err
	}
	return f.allowed[recipient], nil
}

// countingSender records how many messages actually reached the provider side.
type countingSender struct{ calls int }

func (c *countingSender) Send(ctx context.Context, m *Message) (*Receipt, error) {
	c.calls++
	return &Receipt{ProviderMessageID: "prov-1"}, nil
}

func guardedMsg(to string) *Message {
	return &Message{ID: "m1", TenantID: "t1", ChannelID: "c1", To: to, Content: Text{Body: "hi"}}
}

func TestSandboxGuard_BlocksUnlistedRecipientBeforeProvider(t *testing.T) {
	inner := &countingSender{}
	checker := newFakeChecker("+5544999999999")
	g := newSandboxGuard(inner, checker, "t1", "c1", "whatsapp_official")

	_, err := g.Send(context.Background(), guardedMsg("+5511888887777"))

	if err == nil {
		t.Fatal("unlisted recipient was sent, want block")
	}
	if !IsPermanent(err) {
		t.Fatalf("block must be permanent (no retry), got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called for a blocked recipient")
	}
	if strings.Contains(err.Error(), "5511888887777") {
		t.Fatalf("error leaks the full recipient number: %v", err)
	}
}

func TestSandboxGuard_AllowsListedRecipient_NormalizingBothSides(t *testing.T) {
	inner := &countingSender{}
	// Entry stored normalized; message carries the raw provider shape.
	checker := newFakeChecker("+5544999999999")
	g := newSandboxGuard(inner, checker, "t1", "c1", "whatsapp_official")

	if _, err := g.Send(context.Background(), guardedMsg("55 44 99999-9999")); err != nil {
		t.Fatalf("allowlisted recipient blocked: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestSandboxGuard_RemovalTakesEffectOnNextSend(t *testing.T) {
	inner := &countingSender{}
	checker := newFakeChecker("+5544999999999")
	g := newSandboxGuard(inner, checker, "t1", "c1", "whatsapp")

	if _, err := g.Send(context.Background(), guardedMsg("+5544999999999")); err != nil {
		t.Fatalf("first send should pass: %v", err)
	}

	checker.remove("+5544999999999")

	_, err := g.Send(context.Background(), guardedMsg("+5544999999999"))
	if err == nil || !IsPermanent(err) {
		t.Fatalf("send after removal must be permanently blocked, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1 (second send blocked)", inner.calls)
	}
	if checker.Calls != 2 {
		t.Fatalf("allowlist consultations = %d, want 2 — the guard must query at send time, never cache", checker.Calls)
	}
}

func TestSandboxGuard_CheckerErrorWithholdsSendAsTransient(t *testing.T) {
	inner := &countingSender{}
	checker := newFakeChecker("+5544999999999")
	checker.err = errors.New("db down")
	g := newSandboxGuard(inner, checker, "t1", "c1", "whatsapp_official")

	_, err := g.Send(context.Background(), guardedMsg("+5544999999999"))
	if err == nil {
		t.Fatal("send must be withheld when the allowlist cannot be checked")
	}
	if IsPermanent(err) {
		t.Fatalf("checker failure must be transient (retryable), got permanent: %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called while the allowlist was unavailable (fail-open)")
	}
}

func TestSandboxGuard_UnsupportedSandboxTypeFailsClosed(t *testing.T) {
	inner := &countingSender{}
	g := newSandboxGuard(inner, newFakeChecker(), "t1", "c1", "webchat")

	_, err := g.Send(context.Background(), guardedMsg("session-123"))
	if err == nil || !IsPermanent(err) {
		t.Fatalf("sandbox channel type without defined semantics must fail closed, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called for an unsupported sandbox type")
	}
}

// ---------------------------------------------------------------------------
// Resolver integration: guard applied per environment
// ---------------------------------------------------------------------------

func seedResolverChannel(t *testing.T, repo *testutil.MockChannelRepository, id string, env entity.ChannelEnvironment) {
	t.Helper()
	ch := entity.NewChannel("t1", entity.ChannelTypeWhatsAppOfficial, "wa", "")
	ch.ID = id
	ch.Environment = env
	ch.Config = map[string]string{"access_token": "x", "phone_number_id": "1"}
	if err := repo.Create(context.Background(), ch); err != nil {
		t.Fatalf("seed channel: %v", err)
	}
}

func TestResolver_WrapsSandboxAndLeavesProductionUntouched(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	seedResolverChannel(t, repo, "prod-ch", entity.ChannelEnvironmentProduction)
	seedResolverChannel(t, repo, "sandbox-ch", entity.ChannelEnvironmentSandbox)

	inner := &countingSender{}
	r := NewResolver(repo)
	r.Register(factoryFunc{t: "whatsapp_official", s: inner})
	r.SetSandboxAllowlist(newFakeChecker()) // empty allowlist

	prodSender, err := r.For(context.Background(), "prod-ch")
	if err != nil {
		t.Fatalf("For(prod): %v", err)
	}
	if _, err := prodSender.Send(context.Background(), guardedMsg("+5511888887777")); err != nil {
		t.Fatalf("production channel must be unaffected by the guard, got %v", err)
	}

	sandboxSender, err := r.For(context.Background(), "sandbox-ch")
	if err != nil {
		t.Fatalf("For(sandbox): %v", err)
	}
	if _, err := sandboxSender.Send(context.Background(), guardedMsg("+5511888887777")); err == nil {
		t.Fatal("sandbox channel with empty allowlist must block every recipient")
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1 (only the production send)", inner.calls)
	}
}

func TestResolver_SandboxWithoutCheckerFailsClosed(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	seedResolverChannel(t, repo, "sandbox-ch", entity.ChannelEnvironmentSandbox)

	r := NewResolver(repo)
	r.Register(factoryFunc{t: "whatsapp_official", s: &countingSender{}})
	// SetSandboxAllowlist deliberately NOT called.

	if _, err := r.For(context.Background(), "sandbox-ch"); err == nil {
		t.Fatal("resolver built an unguarded sender for a sandbox channel")
	}
}

type factoryFunc struct {
	t string
	s Sender
}

func (f factoryFunc) ChannelType() string                   { return f.t }
func (f factoryFunc) New(map[string]string) (Sender, error) { return f.s, nil }

// ---------------------------------------------------------------------------
// Worker paths: API/conversation, campaign, retry — all through the funnel
// ---------------------------------------------------------------------------

type captureStatusPublisher struct{ updates []*nats.StatusUpdate }

func (c *captureStatusPublisher) PublishStatusUpdate(ctx context.Context, s *nats.StatusUpdate) error {
	c.updates = append(c.updates, s)
	return nil
}

func newGuardedWorker(t *testing.T, checker AllowlistChecker) (*Worker, *countingSender, *captureStatusPublisher) {
	t.Helper()
	repo := testutil.NewMockChannelRepository()
	seedResolverChannel(t, repo, "sandbox-ch", entity.ChannelEnvironmentSandbox)

	inner := &countingSender{}
	r := NewResolver(repo)
	r.Register(factoryFunc{t: "whatsapp_official", s: inner})
	r.SetSandboxAllowlist(checker)

	pub := &captureStatusPublisher{}
	w := NewWorker(nil, pub, r, testutil.NewMockCampaignRepository(), 0)
	return w, inner, pub
}

func TestWorker_ConversationSendOnSandboxIsBlockedNoisily(t *testing.T) {
	w, inner, pub := newGuardedWorker(t, newFakeChecker())

	raw := &nats.OutboundMessage{
		ID: "m1", TenantID: "t1", ChannelID: "sandbox-ch",
		ChannelType: "whatsapp_official", RecipientID: "+5511888887777",
		ContentType: "text", Content: "hi",
	}
	if err := w.handle(context.Background(), raw); err != nil {
		t.Fatalf("permanent block must ack (nil), got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called for a blocked sandbox send")
	}
	if len(pub.updates) != 1 || pub.updates[0].Status != "failed" {
		t.Fatalf("expected one failed StatusUpdate, got %+v", pub.updates)
	}
	if !strings.Contains(pub.updates[0].ErrorMessage, "sandbox guard") {
		t.Fatalf("failure reason must be identifiable as a sandbox-guard block: %q", pub.updates[0].ErrorMessage)
	}
	// Machine-readable reason for the timeline (WP-K): the console reads this
	// instead of parsing the error text.
	if pub.updates[0].BlockedReason != metrics.BlockReasonAllowlist {
		t.Fatalf("BlockedReason = %q, want %q", pub.updates[0].BlockedReason, metrics.BlockReasonAllowlist)
	}
	if strings.Contains(pub.updates[0].ErrorMessage, "5511888887777") {
		t.Fatalf("status update leaks the full recipient: %q", pub.updates[0].ErrorMessage)
	}
}

func TestWorker_ProviderFailureHasNoBlockedReason(t *testing.T) {
	// A provider rejection (not a guard block) must NOT carry a BlockedReason,
	// so the console can tell it apart from a local block.
	repo := testutil.NewMockChannelRepository()
	seedResolverChannel(t, repo, "sandbox-ch", entity.ChannelEnvironmentProduction)
	failing := senderFunc(func(context.Context, *Message) (*Receipt, error) {
		return nil, Permanentf("provider 400: invalid recipient")
	})
	r := NewResolver(repo)
	r.Register(factoryFunc{t: "whatsapp_official", s: failing})
	r.SetSandboxAllowlist(newFakeChecker())
	pub := &captureStatusPublisher{}
	w := NewWorker(nil, pub, r, testutil.NewMockCampaignRepository(), 0)

	raw := &nats.OutboundMessage{
		ID: "mp", TenantID: "t1", ChannelID: "sandbox-ch",
		ChannelType: "whatsapp_official", RecipientID: "+5511888887777",
		ContentType: "text", Content: "hi",
	}
	if err := w.handle(context.Background(), raw); err != nil {
		t.Fatalf("permanent failure must ack, got %v", err)
	}
	if len(pub.updates) != 1 || pub.updates[0].Status != "failed" {
		t.Fatalf("expected one failed update, got %+v", pub.updates)
	}
	if pub.updates[0].BlockedReason != "" {
		t.Fatalf("provider failure must not carry BlockedReason, got %q", pub.updates[0].BlockedReason)
	}
}

func TestWorker_CampaignSendOnSandboxIsBlocked(t *testing.T) {
	w, inner, _ := newGuardedWorker(t, newFakeChecker())

	raw := &nats.OutboundMessage{
		ID: "m2", TenantID: "t1", ChannelID: "sandbox-ch",
		ChannelType: "whatsapp_official", RecipientID: "+5511888887777",
		ContentType: "text", Content: "promo",
		Metadata: map[string]string{"campaign_id": "camp1", "campaign_recipient_id": "rcpt1"},
	}
	if err := w.handle(context.Background(), raw); err != nil {
		t.Fatalf("campaign block must ack (nil), got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called for a blocked campaign send — the guard must cover campaigns")
	}
}

func TestWorker_RetryPathStaysGuarded(t *testing.T) {
	checker := newFakeChecker()
	checker.err = errors.New("allowlist temporarily unavailable")
	w, inner, _ := newGuardedWorker(t, checker)

	raw := &nats.OutboundMessage{
		ID: "m3", TenantID: "t1", ChannelID: "sandbox-ch",
		ChannelType: "whatsapp_official", RecipientID: "+5511888887777",
		ContentType: "text", Content: "hi",
	}
	// Transient guard failure → worker NAKs (returns error) → NATS redelivers.
	if err := w.handle(context.Background(), raw); err == nil {
		t.Fatal("transient guard failure must NAK for redelivery")
	}
	// The redelivery goes through the same funnel and is still guarded.
	checker.err = nil
	if err := w.handle(context.Background(), raw); err != nil {
		t.Fatalf("redelivery of blocked recipient must ack after permanent block, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called on retry despite the guard")
	}
}

func TestWorker_GuardBlockIsDistinguishableInChannelLog(t *testing.T) {
	w, _, _ := newGuardedWorker(t, newFakeChecker())
	logs := &captureLogger{}
	w.SetChannelLogger(logs)

	raw := &nats.OutboundMessage{
		ID: "m9", TenantID: "t1", ChannelID: "sandbox-ch",
		ChannelType: "whatsapp_official", RecipientID: "+5511888887777",
		ContentType: "text", Content: "hi",
	}
	if err := w.handle(context.Background(), raw); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(logs.entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(logs.entries))
	}
	entry := logs.entries[0]
	if entry.metadata["blocked_by"] != "allowlist" {
		t.Fatalf("blocked_by = %q, want allowlist — operator must distinguish guard block from provider failure", entry.metadata["blocked_by"])
	}
	if !strings.Contains(entry.message, "bloqueado por guarda") {
		t.Fatalf("log message must identify a guard block: %q", entry.message)
	}
	if strings.Contains(entry.message, "5511888887777") || strings.Contains(entry.metadata["to"], "5511888887777") {
		t.Fatalf("guard-block log leaks the full recipient: %+v", entry)
	}
}

func TestBlockedReason(t *testing.T) {
	err := Blocked("allowlist", "sandbox guard: recipient blocked")
	if !IsPermanent(err) {
		t.Fatal("Blocked must be permanent")
	}
	reason, ok := BlockedReason(err)
	if !ok || reason != "allowlist" {
		t.Fatalf("BlockedReason = %q/%v", reason, ok)
	}
	if _, ok := BlockedReason(Permanentf("provider 400")); ok {
		t.Fatal("plain permanent error must not read as a guard block")
	}
}
