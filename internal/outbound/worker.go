package outbound

import (
	"context"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
)

// Subscriber is the subset of the NATS consumer the worker needs.
type Subscriber interface {
	SubscribeOutbound(ctx context.Context, channelType string, handler nats.OutboundHandler) error
}

// StatusPublisher is the subset of the NATS producer the worker needs.
type StatusPublisher interface {
	PublishStatusUpdate(ctx context.Context, status *nats.StatusUpdate) error
}

// Worker consumes outbound messages from NATS and delivers them via the Sender
// resolved for each message's channel. It is channel-agnostic: every supported
// channel type flows through the same delivery, retry, and status path.
//
// Retry is handled by NATS: returning a (transient) error NAKs the message for
// redelivery; returning nil acks it. PermanentError results are recorded and
// acked so they are not retried.
type Worker struct {
	subscriber   Subscriber
	publisher    StatusPublisher
	resolver     *Resolver
	campaignRepo repository.CampaignRepository // optional; nil disables campaign tracking
	limiter      *channelLimiter
}

// NewWorker creates a delivery worker. campaignRepo may be nil.
func NewWorker(subscriber Subscriber, publisher StatusPublisher, resolver *Resolver, campaignRepo repository.CampaignRepository, perChannelRatePerSec int) *Worker {
	return &Worker{
		subscriber:   subscriber,
		publisher:    publisher,
		resolver:     resolver,
		campaignRepo: campaignRepo,
		limiter:      newChannelLimiter(perChannelRatePerSec),
	}
}

// Start subscribes the worker to every channel type the resolver supports.
func (w *Worker) Start(ctx context.Context) error {
	for _, channelType := range w.resolver.SupportedTypes() {
		if err := w.subscriber.SubscribeOutbound(ctx, channelType, w.handle); err != nil {
			return err
		}
		logger.Info("outbound delivery worker subscribed for " + channelType)
	}
	return nil
}

// handle delivers one outbound message. Its return value drives NATS ack/nak:
//   - nil            -> ack (delivered, or permanently failed and recorded)
//   - non-nil error  -> nak  -> redelivered (transient failure)
func (w *Worker) handle(ctx context.Context, raw *nats.OutboundMessage) error {
	// Honor a campaign cancellation that landed after enqueue.
	if raw.Metadata != nil && raw.Metadata["campaign_id"] != "" && w.campaignRepo != nil {
		if c, err := w.campaignRepo.FindByID(ctx, raw.Metadata["campaign_id"]); err == nil && c.Status == entity.CampaignStatusCancelled {
			return nil // ack and drop; campaign was cancelled
		}
	}

	msg := translate(raw)

	sender, err := w.resolver.For(ctx, msg.ChannelID)
	if err != nil {
		// No sender / bad credentials is not retryable.
		w.recordFailure(ctx, raw, msg, err.Error())
		logger.Error("outbound: cannot resolve sender for channel " + msg.ChannelID + ": " + err.Error())
		return nil
	}

	w.limiter.wait(msg.ChannelID)

	receipt, err := sender.Send(ctx, msg)
	if err != nil {
		if IsPermanent(err) {
			w.recordFailure(ctx, raw, msg, err.Error())
			logger.Error("outbound: permanent send failure for " + msg.ID + ": " + err.Error())
			return nil // ack: do not retry a permanent failure
		}
		logger.Warn("outbound: transient send failure for " + msg.ID + ", will retry: " + err.Error())
		return err // nak -> retry
	}

	w.recordSuccess(ctx, raw, msg, receipt)
	return nil
}

func (w *Worker) recordSuccess(ctx context.Context, raw *nats.OutboundMessage, msg *Message, receipt *Receipt) {
	if msg.CampaignRecipientID != "" {
		// Campaign sends have no Message row; update the recipient directly.
		// Delivery/read status later flows in via the status webhook pipeline.
		if w.campaignRepo != nil {
			_ = w.campaignRepo.UpdateRecipientStatus(ctx, msg.CampaignRecipientID, entity.RecipientSent, receipt.ProviderMessageID, "")
		}
		return
	}
	// Conversation message: feed the message status pipeline.
	w.publishStatus(ctx, raw, "sent", "", receipt.ProviderMessageID)
}

func (w *Worker) recordFailure(ctx context.Context, raw *nats.OutboundMessage, msg *Message, reason string) {
	if msg.CampaignRecipientID != "" {
		if w.campaignRepo != nil {
			_ = w.campaignRepo.UpdateRecipientStatus(ctx, msg.CampaignRecipientID, entity.RecipientFailed, "", reason)
		}
		return
	}
	w.publishStatus(ctx, raw, "failed", reason, "")
}

func (w *Worker) publishStatus(ctx context.Context, raw *nats.OutboundMessage, status, errMsg, providerID string) {
	if w.publisher == nil {
		return
	}
	_ = w.publisher.PublishStatusUpdate(ctx, &nats.StatusUpdate{
		MessageID:    raw.ID,
		ExternalID:   providerID,
		ChannelType:  raw.ChannelType,
		Status:       status,
		ErrorMessage: errMsg,
		Timestamp:    time.Now(),
	})
}

// translate converts a transport-level nats.OutboundMessage into the typed
// domain Message the senders understand.
func translate(raw *nats.OutboundMessage) *Message {
	meta := raw.Metadata
	if meta == nil {
		meta = map[string]string{}
	}

	msg := &Message{
		ID:                  raw.ID,
		TenantID:            raw.TenantID,
		ChannelID:           raw.ChannelID,
		To:                  raw.RecipientID,
		CampaignID:          meta["campaign_id"],
		CampaignRecipientID: meta["campaign_recipient_id"],
	}

	switch raw.ContentType {
	case "template":
		msg.Content = Template{
			Name:           meta["template_name"],
			Language:       meta["template_language"],
			ComponentsJSON: meta["template_components"],
		}
	case "image", "video", "audio", "document":
		msg.Content = Media{
			Type:     MediaType(raw.ContentType),
			URL:      meta["media_url"],
			MediaID:  meta["media_id"],
			Caption:  raw.Content,
			Filename: meta["filename"],
		}
	default: // "text", "", anything else falls back to text
		msg.Content = Text{
			Body:       raw.Content,
			PreviewURL: strings.EqualFold(meta["preview_url"], "true"),
		}
	}

	return msg
}

// channelLimiter enforces a minimum spacing between sends per channel so a
// burst (e.g. a large campaign) does not exceed provider throughput tiers.
type channelLimiter struct {
	minInterval time.Duration
	mu          chan struct{} // 1-slot mutex via channel to allow per-key locking cheaply
	last        map[string]time.Time
}

func newChannelLimiter(ratePerSec int) *channelLimiter {
	var interval time.Duration
	if ratePerSec > 0 {
		interval = time.Second / time.Duration(ratePerSec)
	}
	cl := &channelLimiter{
		minInterval: interval,
		mu:          make(chan struct{}, 1),
		last:        make(map[string]time.Time),
	}
	cl.mu <- struct{}{}
	return cl
}

func (cl *channelLimiter) wait(channelID string) {
	if cl.minInterval <= 0 {
		return
	}
	<-cl.mu
	now := time.Now()
	next := cl.last[channelID].Add(cl.minInterval)
	var sleep time.Duration
	if now.Before(next) {
		sleep = next.Sub(now)
		cl.last[channelID] = next
	} else {
		cl.last[channelID] = now
	}
	cl.mu <- struct{}{}
	if sleep > 0 {
		time.Sleep(sleep)
	}
}
