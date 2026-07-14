package webhook

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
)

// ChannelLookup is the subset of the channel repository the dispatcher needs to
// resolve a channel's outbound webhook endpoint and signing secret.
type ChannelLookup interface {
	FindByID(ctx context.Context, id string) (*entity.Channel, error)
}

// EventSubscriber is the subset of the NATS consumer the dispatcher needs.
type EventSubscriber interface {
	SubscribeEvents(ctx context.Context, handler nats.EventHandler) error
}

// DeliveryPublisher enqueues a webhook for durable delivery. *nats.Producer
// satisfies it; the durable webhooks stream gives crash-safe retry + DLQ.
type DeliveryPublisher interface {
	PublishWebhookDelivery(ctx context.Context, webhook *nats.WebhookDelivery) error
}

// Credential key (in channel.Credentials) holding the per-channel HMAC secret
// used to sign the outbound webhook. The delivery URL is channel.WebhookURL.
const credWebhookSecret = "webhook_secret"

// cfgWebhookEvents is the channel.Config key holding the comma-separated list of
// `linktor-channel-v1` event types the channel's outbound webhook is subscribed
// to (e.g. "message.received,message.read"). An empty/absent value means deliver
// all events, preserving the original always-deliver behaviour.
const cfgWebhookEvents = "webhook_events"

// subscribedTo reports whether the channel's outbound webhook should receive
// eventType. When webhook_events is empty (or "*" is present) every event is
// delivered; otherwise only the listed types are.
func subscribedTo(channel *entity.Channel, eventType string) bool {
	raw := strings.TrimSpace(channel.Config[cfgWebhookEvents])
	if raw == "" {
		return true
	}
	for _, e := range strings.Split(raw, ",") {
		e = strings.TrimSpace(e)
		if e == "*" || e == eventType {
			return true
		}
	}
	return false
}

// Dispatcher subscribes to internal message events and enqueues them for durable
// delivery to each channel's configured external webhook (e.g. DeskLenz) as
// `linktor-channel-v1` envelopes. It is channel-agnostic: the same path serves
// every channel type and only `channelType` differs in the payload.
//
// Delivery is durable, not fire-and-forget: the dispatcher serializes the
// envelope and enqueues it on the durable webhooks stream (carrying the exact
// body bytes + channel id). A separate DeliveryWorker consumes that stream,
// signs with a fresh timestamp and performs the HTTP POST — so a crash between
// event and delivery does not lose the webhook, and NATS owns retry/DLQ.
type Dispatcher struct {
	deliveries DeliveryPublisher
	channels   ChannelLookup
	now        func() time.Time
	newID      func() string
}

// NewDispatcher builds an outbound-webhook dispatcher that enqueues onto the
// durable webhooks stream via deliveries.
func NewDispatcher(deliveries DeliveryPublisher, channels ChannelLookup) *Dispatcher {
	return &Dispatcher{
		deliveries: deliveries,
		channels:   channels,
		now:        time.Now,
		newID:      func() string { return uuid.New().String() },
	}
}

// Start subscribes to the events stream and begins dispatching.
func (d *Dispatcher) Start(ctx context.Context, sub EventSubscriber) error {
	return sub.SubscribeEvents(ctx, d.handle)
}

// handle converts an internal event into an enqueued outbound webhook. Unknown
// or irrelevant event types are ignored (acked) so the dispatcher coexists with
// other event consumers on the shared events stream. A non-nil error redelivers
// the event (the enqueue is durable, so this only happens on NATS publish
// failure), preserving at-least-once semantics end to end.
func (d *Dispatcher) handle(ctx context.Context, event *nats.Event) error {
	if event == nil {
		return nil
	}

	switch event.Type {
	case nats.EventMessageReceived:
		return d.dispatchInbound(ctx, event)
	case nats.EventMessageSent:
		return d.dispatchStatus(ctx, event, TypeMessageSent)
	case nats.EventMessageDelivered:
		return d.dispatchStatus(ctx, event, TypeMessageDelivered)
	case nats.EventMessageRead:
		return d.dispatchStatus(ctx, event, TypeMessageRead)
	case nats.EventMessageFailed:
		return d.dispatchStatus(ctx, event, TypeMessageFailed)
	case nats.EventContactCreated:
		return d.dispatchContact(ctx, event, TypeContactCreated)
	case nats.EventConversationCreated:
		return d.dispatchConversation(ctx, event, TypeConversationCreated)
	case nats.EventConversationAssigned:
		return d.dispatchConversation(ctx, event, TypeConversationAssigned)
	case nats.EventConversationResolved:
		return d.dispatchConversation(ctx, event, TypeConversationResolved)
	case nats.EventConversationReopened:
		return d.dispatchConversation(ctx, event, TypeConversationReopened)
	case nats.EventConversationEscalated:
		return d.dispatchConversation(ctx, event, TypeConversationEscalated)
	}
	return nil
}

func (d *Dispatcher) dispatchInbound(ctx context.Context, event *nats.Event) error {
	p := payload(event.Payload)

	channelID := p.str("channel_id")
	channel := d.resolveChannel(ctx, channelID)
	if channel == nil {
		return nil
	}

	data := InboundData{
		Message: InboundMessagePayload{
			ID:          p.str("message_id"),
			Direction:   "inbound",
			ContentType: p.str("content_type"),
			Content:     buildContent(p),
			SenderID:    p.str("sender_id"),
			SenderType:  string(entity.SenderTypeContact),
			Metadata:    inboundMetadata(p),
		},
		ConversationID: p.str("conversation_id"),
		ContactID:      p.str("contact_id"),
		ChannelID:      channelID,
		ChannelType:    channelType(channel, p),
	}

	return d.deliver(ctx, channel, TypeMessageReceived, event.TenantID, dedupKey(p.str("message_id"), "received"), data)
}

func (d *Dispatcher) dispatchStatus(ctx context.Context, event *nats.Event, eventType string) error {
	p := payload(event.Payload)

	channel := d.resolveChannel(ctx, p.str("channel_id"))
	if channel == nil {
		return nil
	}

	data := StatusData{
		MessageID: p.str("message_id"),
		Status:    p.str("status"),
		Direction: "outbound",
		Timestamp: d.now().UTC(),
		Error:     p.str("error"),
	}

	return d.deliver(ctx, channel, eventType, event.TenantID, dedupKey(p.str("message_id"), eventType), data)
}

// dispatchContact converts a contact lifecycle event into an outbound webhook.
// The contact.created event carries the origin channel_id so the delivery routes
// to that channel's endpoint (contacts are not otherwise bound to one channel).
func (d *Dispatcher) dispatchContact(ctx context.Context, event *nats.Event, eventType string) error {
	p := payload(event.Payload)

	channel := d.resolveChannel(ctx, p.str("channel_id"))
	if channel == nil {
		return nil
	}

	data := ContactData{
		ContactID:   p.str("contact_id"),
		Name:        p.str("name"),
		Phone:       p.str("phone"),
		Email:       p.str("email"),
		ChannelID:   p.str("channel_id"),
		ChannelType: channelType(channel, p),
	}

	// contact.created is one-shot per contact → deterministic dedup key.
	return d.deliver(ctx, channel, eventType, event.TenantID, dedupKey(p.str("contact_id"), eventType), data)
}

// dispatchConversation converts a conversation lifecycle event into an outbound
// webhook. All conversation events already carry channel_id.
func (d *Dispatcher) dispatchConversation(ctx context.Context, event *nats.Event, eventType string) error {
	p := payload(event.Payload)

	channel := d.resolveChannel(ctx, p.str("channel_id"))
	if channel == nil {
		return nil
	}

	data := ConversationData{
		ConversationID: p.str("conversation_id"),
		ChannelID:      p.str("channel_id"),
		ContactID:      p.str("contact_id"),
		Status:         p.str("status"),
		AssignedUserID: p.str("assigned_user_id"),
		ChannelType:    channelType(channel, p),
	}

	// Only creation is one-shot; assign/resolve/reopen/escalate can legitimately
	// repeat for the same conversation, so those mint a fresh id (no dedup) to
	// avoid collapsing distinct occurrences at the stream.
	dedup := ""
	if eventType == TypeConversationCreated {
		dedup = dedupKey(p.str("conversation_id"), eventType)
	}
	return d.deliver(ctx, channel, eventType, event.TenantID, dedup, data)
}

// dedupKey builds a stable id for an outbound webhook from the source message id
// and the event kind. Using it as the NATS WithMsgID dedup key means a redelivery
// of the same source event collapses to one external POST (instead of generating
// a fresh random id each time and defeating JetStream dedup). Falls back to ""
// when the message id is unknown so deliver() can mint a random id.
func dedupKey(messageID, kind string) string {
	if messageID == "" {
		return ""
	}
	return messageID + "_" + kind
}

// resolveChannel loads the channel and verifies it has an outbound webhook
// configured. Returns nil (skip) when no webhook is set or the channel is gone.
func (d *Dispatcher) resolveChannel(ctx context.Context, channelID string) *entity.Channel {
	if channelID == "" {
		return nil
	}
	channel, err := d.channels.FindByID(ctx, channelID)
	if err != nil || channel == nil {
		return nil
	}
	if channel.WebhookURL == "" {
		return nil // no external consumer configured for this channel
	}
	return channel
}

// deliver serializes the envelope and enqueues it on the durable webhooks
// stream. The exact body bytes and channel id ride along so the DeliveryWorker
// can sign with a fresh timestamp at send time (keeping retries inside the
// anti-replay window) without the signing secret ever touching the stream.
func (d *Dispatcher) deliver(ctx context.Context, channel *entity.Channel, eventType, tenantID, dedup string, data interface{}) error {
	// Per-channel event subscription filter: skip (ack) events the channel's
	// outbound webhook is not subscribed to, so unwanted events never reach the
	// durable stream.
	if !subscribedTo(channel, eventType) {
		return nil
	}

	id := "evt_" + d.newID()
	if dedup != "" {
		// Deterministic id so a redelivered source event dedups at the stream.
		id = "evt_" + dedup
	}
	env := &Envelope{
		ID:        id,
		Type:      eventType,
		Timestamp: d.now().UTC(),
		TenantID:  tenantID,
		Data:      data,
	}

	body, err := MarshalEnvelope(env)
	if err != nil {
		// A marshal failure is deterministic — retrying won't help, so drop (ack).
		logger.Error("webhook: failed to marshal envelope: " + err.Error())
		return nil
	}

	delivery := &nats.WebhookDelivery{
		ID:        env.ID,
		TenantID:  tenantID,
		ChannelID: channel.ID,
		URL:       channel.WebhookURL,
		EventType: eventType,
		Body:      body,
		Timestamp: d.now().UTC(),
	}

	if err := d.deliveries.PublishWebhookDelivery(ctx, delivery); err != nil {
		// Enqueue failed: return the error so the source event is redelivered.
		logger.Warn("webhook: failed to enqueue delivery for channel " + channel.ID + ": " + err.Error())
		return err
	}
	return nil
}

// channelType prefers the live channel type and falls back to the event payload.
func channelType(channel *entity.Channel, p eventPayload) string {
	if channel != nil && channel.Type != "" {
		return string(channel.Type)
	}
	return p.str("channel_type")
}

func buildContent(p eventPayload) MessageContent {
	content := MessageContent{Text: p.str("content")}
	if media := p.media(); media != nil {
		content.Media = media
	}
	return content
}

// inboundMetadata surfaces the provider sender name (and other passthrough
// metadata) the consumer uses to label the contact.
func inboundMetadata(p eventPayload) map[string]string {
	meta := map[string]string{}
	if name := p.str("sender_name"); name != "" {
		meta["senderName"] = name
	}
	if ext := p.str("external_id"); ext != "" {
		meta["externalId"] = ext
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

// eventPayload is a defensive reader over the JSON-decoded event payload map.
type eventPayload map[string]interface{}

func payload(m map[string]interface{}) eventPayload {
	if m == nil {
		return eventPayload{}
	}
	return eventPayload(m)
}

func (p eventPayload) str(key string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// media extracts the first attachment (if any) from the payload's "attachments"
// list. Attachments arrive as a JSON array of objects after NATS round-trip.
func (p eventPayload) media() *MediaPayload {
	raw, ok := p["attachments"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	first, ok := raw[0].(map[string]interface{})
	if !ok {
		return nil
	}
	att := eventPayload(first)
	url := att.str("url")
	if url == "" {
		return nil
	}
	media := &MediaPayload{
		URL:      url,
		MimeType: att.str("mime_type"),
		Filename: att.str("filename"),
		Caption:  att.str("caption"),
	}
	if size, ok := first["size_bytes"].(float64); ok {
		media.Size = int64(size)
	}
	return media
}
