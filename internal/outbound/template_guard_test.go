package outbound

import (
	"context"
	"errors"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

type fakeTemplateProvider struct {
	template *entity.Template
	err      error
	calls    int
}

func (f *fakeTemplateProvider) FindByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error) {
	f.calls++
	return f.template, f.err
}

func templateMsg() *Message {
	return &Message{ID: "m1", TenantID: "t1", ChannelID: "c1", To: "+5511999999999",
		Content: Template{Name: "promo", Language: "pt_BR"}}
}

func newTemplateGuard(mode PolicyMode, p TemplateStatusProvider, inner Sender) Sender {
	return NewTemplateStatusPolicy(mode, p)(waOfficialChannel(), inner)
}

func TestTemplateGuard_EnforceBlocksRejected(t *testing.T) {
	inner := &countingSender{}
	p := &fakeTemplateProvider{template: &entity.Template{Status: entity.TemplateStatusRejected}}
	g := newTemplateGuard(PolicyModeEnforce, p, inner)

	_, err := g.Send(context.Background(), templateMsg())
	if err == nil || !IsPermanent(err) {
		t.Fatalf("rejected template must be permanently blocked, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatal("provider was called for a rejected template")
	}
}

func TestTemplateGuard_ApprovedProceeds(t *testing.T) {
	inner := &countingSender{}
	p := &fakeTemplateProvider{template: &entity.Template{Status: entity.TemplateStatusApproved}}
	g := newTemplateGuard(PolicyModeEnforce, p, inner)

	if _, err := g.Send(context.Background(), templateMsg()); err != nil {
		t.Fatalf("approved template must send, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestTemplateGuard_UnknownStatusAllowsWithLog(t *testing.T) {
	inner := &countingSender{}
	// Not found locally (never synced): fail open, Meta stays the validator.
	p := &fakeTemplateProvider{err: errors.New("template not found")}
	g := newTemplateGuard(PolicyModeEnforce, p, inner)

	if _, err := g.Send(context.Background(), templateMsg()); err != nil {
		t.Fatalf("unknown template status must allow the send, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestTemplateGuard_DryRunNeverBlocks(t *testing.T) {
	inner := &countingSender{}
	p := &fakeTemplateProvider{template: &entity.Template{Status: entity.TemplateStatusPaused}}
	g := newTemplateGuard(PolicyModeDryRun, p, inner)

	if _, err := g.Send(context.Background(), templateMsg()); err != nil {
		t.Fatalf("dry-run must send normally, got %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", inner.calls)
	}
}

func TestTemplateGuard_ConsultsStatusOnEverySend(t *testing.T) {
	inner := &countingSender{}
	p := &fakeTemplateProvider{template: &entity.Template{Status: entity.TemplateStatusApproved}}
	g := newTemplateGuard(PolicyModeEnforce, p, inner)

	for i := 0; i < 3; i++ {
		if _, err := g.Send(context.Background(), templateMsg()); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	if p.calls != 3 {
		t.Fatalf("status lookups = %d, want 3 — Meta recategorizes templates, so the status is consulted at send time, never cached", p.calls)
	}

	// Recategorization under our feet takes effect on the next send.
	p.template = &entity.Template{Status: entity.TemplateStatusPaused}
	if _, err := g.Send(context.Background(), templateMsg()); err == nil {
		t.Fatal("recategorized (paused) template must be blocked on the next send")
	}
}

func TestTemplateGuard_NonTemplateContentPassesThrough(t *testing.T) {
	inner := &countingSender{}
	p := &fakeTemplateProvider{}
	g := newTemplateGuard(PolicyModeEnforce, p, inner)

	if _, err := g.Send(context.Background(), windowMsg(Text{Body: "hi"})); err != nil {
		t.Fatalf("text message must not consult template status, got %v", err)
	}
	if p.calls != 0 {
		t.Fatal("template provider consulted for non-template content")
	}
}
