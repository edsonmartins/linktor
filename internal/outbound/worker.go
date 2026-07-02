package outbound

import (
	"context"
	"encoding/json"
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

	if err := w.limiter.wait(ctx, msg.ChannelID); err != nil {
		// Context cancelled/expired while throttled: do not send, let NATS
		// redeliver rather than blocking past AckWait (which would double-send).
		return err
	}

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
		Metadata:            meta,
	}

	switch raw.ContentType {
	case "template":
		msg.Content = Template{
			Name:           meta["template_name"],
			Language:       meta["template_language"],
			ComponentsJSON: meta["template_components"],
		}
	case "interactive":
		msg.Content = translateInteractive(raw, meta)
	case "image", "video", "audio", "document":
		msg.Content = mediaContent(MediaType(raw.ContentType), raw, meta)
	default: // "text", "", anything else falls back to text
		// An agent may attach a file on an otherwise-text message; deliver the
		// first attachment as media rather than silently dropping it.
		if len(raw.Attachments) > 0 && meta["media_url"] == "" && meta["media_id"] == "" {
			att := raw.Attachments[0]
			msg.Content = mediaContent(mediaTypeFromAttachment(att), raw, meta)
			break
		}
		msg.Content = Text{
			Body:       raw.Content,
			PreviewURL: strings.EqualFold(meta["preview_url"], "true"),
		}
	}

	return msg
}

// mediaContent builds a Media payload, sourcing the URL/filename from the first
// attachment when the metadata does not already carry a media_url/media_id.
// Previously translate only read meta["media_url"], so agent/bot attachments
// (which arrive in raw.Attachments) were never delivered.
func mediaContent(mt MediaType, raw *nats.OutboundMessage, meta map[string]string) Media {
	url := meta["media_url"]
	mediaID := meta["media_id"]
	filename := meta["filename"]
	if url == "" && mediaID == "" && len(raw.Attachments) > 0 {
		att := raw.Attachments[0]
		url = att.URL
		if filename == "" {
			filename = att.Filename
		}
	}
	return Media{
		Type:     mt,
		URL:      url,
		MediaID:  mediaID,
		Caption:  raw.Content,
		Filename: filename,
	}
}

// mediaTypeFromAttachment maps an attachment's declared type/MIME to a MediaType,
// defaulting to document for anything unrecognized.
func mediaTypeFromAttachment(att nats.AttachmentData) MediaType {
	switch {
	case att.Type == "image" || strings.HasPrefix(att.MimeType, "image/"):
		return MediaImage
	case att.Type == "video" || strings.HasPrefix(att.MimeType, "video/"):
		return MediaVideo
	case att.Type == "audio" || strings.HasPrefix(att.MimeType, "audio/"):
		return MediaAudio
	default:
		return MediaDocument
	}
}

// translateInteractive reconstructs the channel-agnostic Interactive payload the
// send path produced (metadata["quick_replies"] + interactive_body). When no
// buttons survive it falls back to plain text so nothing is dropped.
func translateInteractive(raw *nats.OutboundMessage, meta map[string]string) Content {
	body := meta["interactive_body"]
	if body == "" {
		body = raw.Content
	}
	var buttons []InteractiveButton
	if qr := meta["quick_replies"]; qr != "" {
		var generic []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal([]byte(qr), &generic); err == nil {
			for _, g := range generic {
				buttons = append(buttons, InteractiveButton{ID: g.ID, Title: g.Title})
			}
		}
	}
	if len(buttons) == 0 {
		return Text{Body: body}
	}
	return Interactive{Body: body, Buttons: buttons}
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

// wait blocks until this channel's rate-limit slot is free, or until ctx is
// done. Returning ctx.Err() lets the caller abort the send instead of sleeping
// past the NATS AckWait deadline (which would trigger a redelivery and a
// duplicate send). The sleep is interruptible for the same reason.
func (cl *channelLimiter) wait(ctx context.Context, channelID string) error {
	if cl.minInterval <= 0 {
		return ctx.Err()
	}
	select {
	case <-cl.mu:
	case <-ctx.Done():
		return ctx.Err()
	}
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
	if sleep <= 0 {
		return nil
	}
	t := time.NewTimer(sleep)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
