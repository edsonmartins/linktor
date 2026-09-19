package outbound

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

// The provider fetches attachment media by URL with no session, so the worker
// must hand it a signed media-proxy URL — while the NATS payload (and the
// stored message behind it) keeps the unsigned reference.
func TestWorker_SignsMediaURLBeforeProviderFetch(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	seedResolverChannel(t, repo, "prod-ch", entity.ChannelEnvironmentProduction)

	var sentURL string
	r := NewResolver(repo)
	r.Register(factoryFunc{t: "whatsapp_official", s: senderFunc(func(_ context.Context, m *Message) (*Receipt, error) {
		sentURL = m.Content.(Media).URL
		return &Receipt{}, nil
	})})
	w := NewWorker(nil, &captureStatusPublisher{}, r, testutil.NewMockCampaignRepository(), 0)
	w.SetMediaURLSigner(func(u string) string { return u + "?exp=1&sig=x" })

	stored := "https://api.linktor.dev/api/v1/media/attachments/t1/a.jpg"
	raw := &nats.OutboundMessage{
		ID: "m1", TenantID: "t1", ChannelID: "prod-ch", ChannelType: "whatsapp_official",
		RecipientID: "+5511888887777", ContentType: "image",
		Attachments: []nats.AttachmentData{{Type: "image", URL: stored}},
	}
	if err := w.handle(context.Background(), raw); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if sentURL != stored+"?exp=1&sig=x" {
		t.Fatalf("provider got %q, want the signed URL", sentURL)
	}
	if raw.Attachments[0].URL != stored {
		t.Fatalf("payload mutated: %q", raw.Attachments[0].URL)
	}
}
