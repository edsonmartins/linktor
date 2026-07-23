package outbound

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

type fakeLastInbound struct {
	last *time.Time
	err  error
}

func (f *fakeLastInbound) LastInboundAt(ctx context.Context, conversationID string) (*time.Time, error) {
	return f.last, f.err
}

func windowMsg(content Content) *Message {
	return &Message{ID: "m1", TenantID: "t1", ChannelID: "c1", ConversationID: "conv1", To: "+5511999999999", Content: content}
}

func newWindowGuard(mode PolicyMode, provider LastInboundProvider, inner Sender, now time.Time) Sender {
	g := NewSessionWindowPolicy(mode, provider)(waOfficialChannel(), inner)
	if wg, ok := g.(*sessionWindowGuard); ok {
		wg.now = func() time.Time { return now }
	}
	return g
}

func waOfficialChannel() *entity.Channel {
	ch := entity.NewChannel("t1", entity.ChannelTypeWhatsAppOfficial, "wa", "")
	ch.ID = "c1"
	return ch
}

func TestParsePolicyMode(t *testing.T) {
	cases := map[string]PolicyMode{
		"off": PolicyModeOff, "dry_run": PolicyModeDryRun, "enforce": PolicyModeEnforce,
		"": PolicyModeDryRun, "banana": PolicyModeDryRun,
	}
	for in, want := range cases {
		if got := ParsePolicyMode(in); got != want {
			t.Fatalf("ParsePolicyMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWindowGuard_DryRunNeverBlocks(t *testing.T) {
	now := time.Now()
	stale := now.Add(-30 * time.Hour)
	inner := &countingSender{}
	g := newWindowGuard(PolicyModeDryRun, &fakeLastInbound{last: &stale}, inner, now)

	if _, err := g.Send(context.Background(), windowMsg(Text{Body: "hi"})); err != nil {
		t.Fatalf("dry-run must send normally, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestWindowGuard_EnforceBlocksOutsideWindow(t *testing.T) {
	now := time.Now()
	stale := now.Add(-25 * time.Hour)
	inner := &countingSender{}
	g := newWindowGuard(PolicyModeEnforce, &fakeLastInbound{last: &stale}, inner, now)

	_, err := g.Send(context.Background(), windowMsg(Text{Body: "hi"}))
	if err == nil || !IsPermanent(err) {
		t.Fatalf("out-of-window free-form must be permanently blocked, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called for an out-of-window send")
	}
}

func TestWindowGuard_EnforceBlocksWhenNeverInbound(t *testing.T) {
	inner := &countingSender{}
	g := newWindowGuard(PolicyModeEnforce, &fakeLastInbound{last: nil}, inner, time.Now())

	if _, err := g.Send(context.Background(), windowMsg(Text{Body: "hi"})); err == nil {
		t.Fatal("free-form with no inbound history must be blocked under enforcement")
	}
	if inner.calls != 0 {
		t.Fatal("provider was called")
	}
}

func TestWindowGuard_InsideWindowIsAllowed(t *testing.T) {
	now := time.Now()
	recent := now.Add(-23 * time.Hour)
	inner := &countingSender{}
	g := newWindowGuard(PolicyModeEnforce, &fakeLastInbound{last: &recent}, inner, now)

	if _, err := g.Send(context.Background(), windowMsg(Text{Body: "hi"})); err != nil {
		t.Fatalf("inbound within 24h must allow free-form, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestWindowGuard_TemplateIsExemptFromWindow(t *testing.T) {
	inner := &countingSender{}
	g := newWindowGuard(PolicyModeEnforce, &fakeLastInbound{last: nil}, inner, time.Now())

	tpl := windowMsg(Template{Name: "hello", Language: "pt_BR"})
	if _, err := g.Send(context.Background(), tpl); err != nil {
		t.Fatalf("approved template must bypass the window, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestWindowGuard_ProviderErrorFailsOpen(t *testing.T) {
	inner := &countingSender{}
	g := newWindowGuard(PolicyModeEnforce, &fakeLastInbound{err: errors.New("db down")}, inner, time.Now())

	if _, err := g.Send(context.Background(), windowMsg(Text{Body: "hi"})); err != nil {
		t.Fatalf("policy guard must fail open on lookup error (unlike the sandbox guard), got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestWindowGuard_OnlyWrapsWhatsAppOfficial(t *testing.T) {
	inner := &countingSender{}
	tg := entity.NewChannel("t1", entity.ChannelTypeTelegram, "tg", "")
	s := NewSessionWindowPolicy(PolicyModeEnforce, &fakeLastInbound{})(tg, inner)
	if s != Sender(inner) {
		t.Fatal("non-whatsapp_official channel must not be wrapped")
	}

	off := NewSessionWindowPolicy(PolicyModeOff, &fakeLastInbound{})(waOfficialChannel(), inner)
	if off != Sender(inner) {
		t.Fatal("mode off must not wrap")
	}
}
