package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

func newCampaignTestService() (*CampaignService, *testutil.MockCampaignRepository, *testutil.MockProducer) {
	repo := testutil.NewMockCampaignRepository()
	channelRepo := testutil.NewMockChannelRepository()
	ch := entity.NewChannel("t1", entity.ChannelTypeWhatsAppOfficial, "wa", "")
	ch.ID = "ch1"
	_ = channelRepo.Create(context.Background(), ch)
	producer := testutil.NewMockProducer()
	return NewCampaignService(repo, channelRepo, producer), repo, producer
}

func TestCampaignCreateValidation(t *testing.T) {
	svc, _, _ := newCampaignTestService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, "t1", "u1", &CampaignInput{ChannelID: "ch1", TemplateName: "x", Recipients: []RecipientInput{{Phone: "+1"}}}); err == nil {
		t.Fatal("expected error for missing name")
	}
	if _, err := svc.Create(ctx, "t1", "u1", &CampaignInput{Name: "c", ChannelID: "ch1", Recipients: []RecipientInput{{Phone: "+1"}}}); err == nil {
		t.Fatal("expected error for missing template")
	}
	if _, err := svc.Create(ctx, "t1", "u1", &CampaignInput{Name: "c", ChannelID: "ch1", TemplateName: "x"}); err == nil {
		t.Fatal("expected error for no recipients")
	}
	if _, err := svc.Create(ctx, "t1", "u1", &CampaignInput{Name: "c", ChannelID: "missing", TemplateName: "x", Recipients: []RecipientInput{{Phone: "+1"}}}); err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

func TestCampaignCreateAndEnqueue(t *testing.T) {
	svc, repo, producer := newCampaignTestService()
	ctx := context.Background()

	camp, err := svc.Create(ctx, "t1", "u1", &CampaignInput{
		Name: "promo", ChannelID: "ch1", TemplateName: "welcome", TemplateLanguage: "pt_BR",
		Recipients: []RecipientInput{
			{Phone: "+551", Params: []string{"John"}},
			{Phone: "+552"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if camp.TotalRecipients != 2 || camp.Status != entity.CampaignStatusDraft {
		t.Fatalf("unexpected campaign: %+v", camp)
	}

	// Enqueue synchronously (white-box) and verify the outbound jobs.
	svc.enqueue(camp)

	if len(producer.OutboundMessages) != 2 {
		t.Fatalf("expected 2 outbound messages, got %d", len(producer.OutboundMessages))
	}
	for _, r := range repo.Recipients {
		if r.Status != entity.RecipientQueued {
			t.Fatalf("recipient %s should be queued, got %s", r.Phone, r.Status)
		}
	}
	// The recipient with params must carry template_components (the param-delivery fix).
	var withParams int
	for _, msg := range producer.OutboundMessages {
		if msg.Metadata["template_name"] != "welcome" {
			t.Fatalf("missing template_name in metadata: %+v", msg.Metadata)
		}
		if msg.Metadata["campaign_recipient_id"] == "" {
			t.Fatal("missing campaign_recipient_id linkage")
		}
		if msg.Metadata["template_components"] != "" {
			withParams++
		}
	}
	if withParams != 1 {
		t.Fatalf("expected exactly 1 message with template_components, got %d", withParams)
	}
}

func TestCampaignSweepStale(t *testing.T) {
	svc, repo, _ := newCampaignTestService()
	ctx := context.Background()

	camp, _ := svc.Create(ctx, "t1", "u1", &CampaignInput{
		Name: "c", ChannelID: "ch1", TemplateName: "x",
		Recipients: []RecipientInput{{Phone: "+1"}, {Phone: "+2"}},
	})
	repo.Campaigns[camp.ID].Status = entity.CampaignStatusProcessing
	svc.enqueue(camp) // -> recipients queued

	n, err := svc.SweepStale(ctx, time.Minute)
	if err != nil {
		t.Fatalf("SweepStale: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 campaign swept, got %d", n)
	}
	for _, r := range repo.Recipients {
		if r.Status != entity.RecipientFailed {
			t.Fatalf("stale queued recipient should be failed, got %s", r.Status)
		}
	}
	if repo.Campaigns[camp.ID].FailedCount != 2 {
		t.Fatalf("expected failed_count=2 after recount, got %d", repo.Campaigns[camp.ID].FailedCount)
	}
}

func TestCampaignDeriveStatusCompletes(t *testing.T) {
	svc, repo, _ := newCampaignTestService()
	ctx := context.Background()

	camp, _ := svc.Create(ctx, "t1", "u1", &CampaignInput{
		Name: "c", ChannelID: "ch1", TemplateName: "x",
		Recipients: []RecipientInput{{Phone: "+1"}, {Phone: "+2"}},
	})
	repo.Campaigns[camp.ID].Status = entity.CampaignStatusProcessing
	for _, r := range repo.Recipients {
		r.Status = entity.RecipientSent
	}

	got, err := svc.Get(ctx, "t1", camp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != entity.CampaignStatusCompleted {
		t.Fatalf("expected completed, got %s", got.Status)
	}
	if got.SentCount != 2 {
		t.Fatalf("expected sent_count=2, got %d", got.SentCount)
	}
}

func TestCampaignCancel(t *testing.T) {
	svc, repo, _ := newCampaignTestService()
	ctx := context.Background()
	camp, _ := svc.Create(ctx, "t1", "u1", &CampaignInput{
		Name: "c", ChannelID: "ch1", TemplateName: "x", Recipients: []RecipientInput{{Phone: "+1"}},
	})
	if err := svc.Cancel(ctx, "t1", camp.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if repo.Campaigns[camp.ID].Status != entity.CampaignStatusCancelled {
		t.Fatal("campaign should be cancelled")
	}
}

// TestCampaignCancelRejectsTerminal verifies Cancel is refused for campaigns
// already in a terminal state (completed or cancelled): flipping them would be
// misleading and there is nothing left to stop.
func TestCampaignCancelRejectsTerminal(t *testing.T) {
	svc, repo, _ := newCampaignTestService()
	ctx := context.Background()
	camp, _ := svc.Create(ctx, "t1", "u1", &CampaignInput{
		Name: "c", ChannelID: "ch1", TemplateName: "x", Recipients: []RecipientInput{{Phone: "+1"}},
	})

	// A completed campaign cannot be cancelled.
	repo.Campaigns[camp.ID].Status = entity.CampaignStatusCompleted
	if err := svc.Cancel(ctx, "t1", camp.ID); err == nil {
		t.Fatal("expected Cancel to reject a completed campaign")
	}
	if repo.Campaigns[camp.ID].Status != entity.CampaignStatusCompleted {
		t.Fatal("completed campaign must stay completed after a rejected cancel")
	}

	// An already-cancelled campaign cannot be cancelled again.
	repo.Campaigns[camp.ID].Status = entity.CampaignStatusCancelled
	if err := svc.Cancel(ctx, "t1", camp.ID); err == nil {
		t.Fatal("expected Cancel to reject an already-cancelled campaign")
	}
}

// cancelOnFirstPublish flips the campaign to cancelled the moment the first
// outbound job is published, simulating an operator cancelling mid-dispatch.
type cancelOnFirstPublish struct {
	*testutil.MockProducer
	repo   *testutil.MockCampaignRepository
	campID string
	fired  bool
}

func (p *cancelOnFirstPublish) PublishOutbound(ctx context.Context, msg *nats.OutboundMessage) error {
	if !p.fired {
		p.fired = true
		if c, ok := p.repo.Campaigns[p.campID]; ok {
			c.Status = entity.CampaignStatusCancelled
		}
	}
	return p.MockProducer.PublishOutbound(ctx, msg)
}

// TestCampaignCancelStopsDispatch verifies that cancelling a campaign while its
// enqueue loop is running stops further sends: the loop re-reads the status
// within the batch (every cancelCheckInterval) and halts, so only the recipients
// enqueued before the check are published — never the whole recipient list.
func TestCampaignCancelStopsDispatch(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewMockCampaignRepository()
	channelRepo := testutil.NewMockChannelRepository()
	ch := entity.NewChannel("t1", entity.ChannelTypeWhatsAppOfficial, "wa", "")
	ch.ID = "ch1"
	_ = channelRepo.Create(ctx, ch)
	prod := &cancelOnFirstPublish{MockProducer: testutil.NewMockProducer(), repo: repo}
	svc := NewCampaignService(repo, channelRepo, prod)

	// More recipients than cancelCheckInterval so the in-batch re-read triggers.
	total := cancelCheckInterval * 3
	recips := make([]RecipientInput, 0, total)
	for i := 0; i < total; i++ {
		recips = append(recips, RecipientInput{Phone: "+" + strconv.Itoa(1000+i)})
	}
	camp, err := svc.Create(ctx, "t1", "u1", &CampaignInput{
		Name: "promo", ChannelID: "ch1", TemplateName: "welcome", Recipients: recips,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	prod.campID = camp.ID
	repo.Campaigns[camp.ID].Status = entity.CampaignStatusProcessing

	// Dispatch synchronously; the producer cancels on the first publish and the
	// loop must stop at the next in-batch status re-read.
	svc.enqueue(camp)

	published := len(prod.MockProducer.OutboundMessages)
	if published != cancelCheckInterval {
		t.Fatalf("expected dispatch to stop at %d publishes after cancel, got %d", cancelCheckInterval, published)
	}
	if published >= total {
		t.Fatalf("cancel did not stop dispatch: published %d of %d recipients", published, total)
	}

	// The remaining recipients must be left untouched (still pending), not queued.
	var pending, queued int
	for _, r := range repo.Recipients {
		switch r.Status {
		case entity.RecipientPending:
			pending++
		case entity.RecipientQueued:
			queued++
		}
	}
	if queued != cancelCheckInterval {
		t.Fatalf("expected %d queued recipients, got %d", cancelCheckInterval, queued)
	}
	if pending != total-cancelCheckInterval {
		t.Fatalf("expected %d recipients left pending after cancel, got %d", total-cancelCheckInterval, pending)
	}
}
