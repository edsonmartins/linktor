package outbound

import (
	"context"
	"errors"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

func TestTranslateText(t *testing.T) {
	raw := &nats.OutboundMessage{
		ID: "m1", ChannelID: "c1", RecipientID: "+5511999",
		ContentType: "text", Content: "hello",
		Metadata: map[string]string{"preview_url": "true"},
	}
	msg := translate(raw)
	txt, ok := msg.Content.(Text)
	if !ok {
		t.Fatalf("expected Text, got %T", msg.Content)
	}
	if txt.Body != "hello" || !txt.PreviewURL {
		t.Fatalf("bad text mapping: %+v", txt)
	}
	if msg.To != "+5511999" {
		t.Fatalf("recipient not mapped: %q", msg.To)
	}
}

func TestTranslateTemplateCarriesComponents(t *testing.T) {
	raw := &nats.OutboundMessage{
		ID: "m2", ChannelID: "c1", RecipientID: "+55",
		ContentType: "template",
		Metadata: map[string]string{
			"template_name":         "welcome",
			"template_language":     "pt_BR",
			"template_components":   `[{"type":"body","parameters":[{"type":"text","text":"John"}]}]`,
			"campaign_id":           "camp1",
			"campaign_recipient_id": "rec1",
		},
	}
	msg := translate(raw)
	tpl, ok := msg.Content.(Template)
	if !ok {
		t.Fatalf("expected Template, got %T", msg.Content)
	}
	if tpl.Name != "welcome" || tpl.Language != "pt_BR" {
		t.Fatalf("bad template mapping: %+v", tpl)
	}
	if tpl.ComponentsJSON == "" {
		t.Fatal("components must be carried through (the param-delivery bug)")
	}
	if msg.CampaignID != "camp1" || msg.CampaignRecipientID != "rec1" {
		t.Fatalf("campaign linkage lost: %+v", msg)
	}
}

func TestTranslateMedia(t *testing.T) {
	raw := &nats.OutboundMessage{
		ID: "m3", ChannelID: "c1", RecipientID: "+55",
		ContentType: "image", Content: "caption",
		Metadata: map[string]string{"media_url": "https://x/y.png"},
	}
	msg := translate(raw)
	md, ok := msg.Content.(Media)
	if !ok {
		t.Fatalf("expected Media, got %T", msg.Content)
	}
	if md.Type != MediaImage || md.URL != "https://x/y.png" || md.Caption != "caption" {
		t.Fatalf("bad media mapping: %+v", md)
	}
}

func TestIsPermanent(t *testing.T) {
	if !IsPermanent(Permanentf("bad template")) {
		t.Fatal("Permanentf should be permanent")
	}
	if IsPermanent(errors.New("network timeout")) {
		t.Fatal("plain error should be transient")
	}
}

// fakeFactory builds a sender that records nothing; used to test the resolver.
type fakeFactory struct{ built int }

func (f *fakeFactory) ChannelType() string { return "whatsapp_official" }
func (f *fakeFactory) New(map[string]string) (Sender, error) {
	f.built++
	return senderFunc(func(context.Context, *Message) (*Receipt, error) { return &Receipt{}, nil }), nil
}

type senderFunc func(context.Context, *Message) (*Receipt, error)

func (s senderFunc) Send(ctx context.Context, m *Message) (*Receipt, error) { return s(ctx, m) }

func TestResolverBuildsOnceAndInvalidates(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	ch := entity.NewChannel("t1", entity.ChannelTypeWhatsAppOfficial, "wa", "")
	ch.ID = "c1"
	ch.Config = map[string]string{"access_token": "x", "phone_number_id": "1"}
	_ = repo.Create(context.Background(), ch)

	f := &fakeFactory{}
	r := NewResolver(repo)
	r.Register(f)

	if !r.Supports("whatsapp_official") {
		t.Fatal("should support registered type")
	}

	for i := 0; i < 3; i++ {
		if _, err := r.For(context.Background(), "c1"); err != nil {
			t.Fatalf("For: %v", err)
		}
	}
	if f.built != 1 {
		t.Fatalf("expected sender built once (cached), got %d", f.built)
	}

	r.Invalidate("c1")
	if _, err := r.For(context.Background(), "c1"); err != nil {
		t.Fatalf("For after invalidate: %v", err)
	}
	if f.built != 2 {
		t.Fatalf("expected rebuild after invalidate, got %d", f.built)
	}
}

func TestResolverUnsupportedType(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	ch := entity.NewChannel("t1", entity.ChannelTypeTelegram, "tg", "")
	ch.ID = "c2"
	_ = repo.Create(context.Background(), ch)

	r := NewResolver(repo)
	if _, err := r.For(context.Background(), "c2"); err == nil {
		t.Fatal("expected error for unsupported channel type")
	}
}
