package whatsapp_official

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
	"github.com/msgfy/linktor/pkg/plugin"
)

// OutboundConsumer consumes outbound messages from NATS and sends them via WhatsApp
type OutboundConsumer struct {
	adapter      *Adapter
	producer     *nats.Producer
	consumer     *nats.Consumer
	deduplicator *MessageDeduplicator
	mu           sync.RWMutex
	running      bool
	cancelFunc   context.CancelFunc
}

// NewOutboundConsumer creates a new outbound message consumer
func NewOutboundConsumer(adapter *Adapter, producer *nats.Producer, consumer *nats.Consumer) *OutboundConsumer {
	return &OutboundConsumer{
		adapter:      adapter,
		producer:     producer,
		consumer:     consumer,
		deduplicator: NewMessageDeduplicator(5 * time.Minute),
	}
}

// Start starts consuming outbound messages
func (c *OutboundConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	// Subscribe to WhatsApp outbound messages
	err := c.consumer.SubscribeOutbound(ctx, "whatsapp", c.handleOutbound)
	if err != nil {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		return fmt.Errorf("failed to subscribe to outbound messages: %w", err)
	}

	// Also subscribe to whatsapp_official channel type
	err = c.consumer.SubscribeOutbound(ctx, "whatsapp_official", c.handleOutbound)
	if err != nil {
		logger.Warn("Failed to subscribe to whatsapp_official outbound: " + err.Error())
	}

	logger.Info("WhatsApp outbound consumer started")
	return nil
}

// Stop stops the consumer
func (c *OutboundConsumer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	c.running = false
	logger.Info("WhatsApp outbound consumer stopped")
}

// handleOutbound handles an outbound message from NATS
func (c *OutboundConsumer) handleOutbound(ctx context.Context, msg *nats.OutboundMessage) error {
	// Check for duplicate
	if c.deduplicator.IsDuplicate(msg.ID) {
		logger.Warn("Duplicate message detected, skipping: " + msg.ID)
		return nil
	}

	// Convert NATS message to plugin message
	pluginMsg := &plugin.OutboundMessage{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		RecipientID:    msg.RecipientID,
		ContentType:    plugin.ContentType(msg.ContentType),
		Content:        msg.Content,
		Metadata:       msg.Metadata,
	}

	// Convert attachments
	if len(msg.Attachments) > 0 {
		pluginMsg.Attachments = make([]*plugin.Attachment, len(msg.Attachments))
		for i, att := range msg.Attachments {
			pluginMsg.Attachments[i] = &plugin.Attachment{
				Type:         att.Type,
				URL:          att.URL,
				Filename:     att.Filename,
				MimeType:     att.MimeType,
				SizeBytes:    att.SizeBytes,
				ThumbnailURL: att.ThumbnailURL,
				Metadata:     att.Metadata,
			}
		}
	}

	// Send via adapter
	result, err := c.adapter.SendMessage(ctx, pluginMsg)
	if err != nil {
		c.publishStatus(ctx, msg, "failed", err.Error())
		return err
	}

	if !result.Success {
		c.publishStatus(ctx, msg, "failed", result.Error)
		return fmt.Errorf("send failed: %s", result.Error)
	}

	// Publish success status
	c.publishStatus(ctx, msg, "sent", "")

	// Mark as processed
	c.deduplicator.MarkProcessed(msg.ID)

	return nil
}

// publishStatus publishes a status update for a message
func (c *OutboundConsumer) publishStatus(ctx context.Context, msg *nats.OutboundMessage, status, errorMsg string) {
	if c.producer == nil {
		return
	}

	statusUpdate := &nats.StatusUpdate{
		MessageID:    msg.ID,
		ChannelType:  msg.ChannelType,
		Status:       status,
		ErrorMessage: errorMsg,
		Timestamp:    time.Now(),
	}

	if err := c.producer.PublishStatusUpdate(ctx, statusUpdate); err != nil {
		logger.Error("Failed to publish status update: " + err.Error())
	}
}

// MessageDeduplicator prevents duplicate message processing
type MessageDeduplicator struct {
	mu       sync.RWMutex
	messages map[string]time.Time
	ttl      time.Duration
}

// NewMessageDeduplicator creates a new message deduplicator
func NewMessageDeduplicator(ttl time.Duration) *MessageDeduplicator {
	d := &MessageDeduplicator{
		messages: make(map[string]time.Time),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go d.cleanup()

	return d
}

// IsDuplicate checks if a message ID has been seen recently
func (d *MessageDeduplicator) IsDuplicate(messageID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	_, exists := d.messages[messageID]
	return exists
}

// MarkProcessed marks a message as processed
func (d *MessageDeduplicator) MarkProcessed(messageID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.messages[messageID] = time.Now()
}

// cleanup periodically removes expired entries
func (d *MessageDeduplicator) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for id, ts := range d.messages {
			if now.Sub(ts) > d.ttl {
				delete(d.messages, id)
			}
		}
		d.mu.Unlock()
	}
}

// SessionAwareConsumer adds session window checking to outbound consumer
type SessionAwareConsumer struct {
	*OutboundConsumer
	templateSender *TemplateSender
	defaultTemplate string
	defaultLanguage string
}

// NewSessionAwareConsumer creates a session-aware consumer
func NewSessionAwareConsumer(
	adapter *Adapter,
	producer *nats.Producer,
	consumer *nats.Consumer,
	defaultTemplate string,
	defaultLanguage string,
) *SessionAwareConsumer {
	return &SessionAwareConsumer{
		OutboundConsumer: NewOutboundConsumer(adapter, producer, consumer),
		templateSender:   NewTemplateSender(adapter.GetClient()),
		defaultTemplate:  defaultTemplate,
		defaultLanguage:  defaultLanguage,
	}
}

// handleOutboundWithSession handles outbound messages with session awareness
func (c *SessionAwareConsumer) handleOutboundWithSession(ctx context.Context, msg *nats.OutboundMessage) error {
	// Check if we're within the 24-hour session window
	if !c.adapter.IsSessionValid(msg.RecipientID) {
		// Outside session - need to send template message
		return c.sendTemplateMessage(ctx, msg)
	}

	// Within session - send regular message
	return c.handleOutbound(ctx, msg)
}

// sendTemplateMessage sends a template message when outside the 24-hour window
func (c *SessionAwareConsumer) sendTemplateMessage(ctx context.Context, msg *nats.OutboundMessage) error {
	// Check if message specifies a template
	templateName := msg.Metadata["template_name"]
	if templateName == "" {
		templateName = c.defaultTemplate
	}
	if templateName == "" {
		return fmt.Errorf("session expired and no template configured")
	}

	languageCode := msg.Metadata["template_language"]
	if languageCode == "" {
		languageCode = c.defaultLanguage
	}
	if languageCode == "" {
		languageCode = "en"
	}

	// Build template with body params if content is provided
	var template *TemplateObject
	if msg.Content != "" {
		template = NewTemplateBuilder(templateName, languageCode).
			AddBodyParameters(msg.Content).
			Build()
	} else {
		template = NewTemplateBuilder(templateName, languageCode).Build()
	}

	// Add components from metadata if provided
	if componentsJSON, ok := msg.Metadata["template_components"]; ok && componentsJSON != "" {
		var components []TemplateComponent
		if err := json.Unmarshal([]byte(componentsJSON), &components); err == nil {
			template.Components = append(template.Components, components...)
		}
	}

	// Send template
	resp, err := c.templateSender.SendTemplate(ctx, msg.RecipientID, template)
	if err != nil {
		c.publishStatus(ctx, msg, "failed", err.Error())
		return err
	}

	if len(resp.Messages) == 0 {
		c.publishStatus(ctx, msg, "failed", "no message ID returned")
		return fmt.Errorf("template send returned no message ID")
	}

	c.publishStatus(ctx, msg, "sent", "")
	return nil
}

// WebhookConsumer processes inbound webhooks and publishes to NATS
type WebhookConsumer struct {
	adapter  *Adapter
	producer *nats.Producer
	tenantID string
	channelID string
}

// NewWebhookConsumer creates a new webhook consumer
func NewWebhookConsumer(adapter *Adapter, producer *nats.Producer, tenantID, channelID string) *WebhookConsumer {
	return &WebhookConsumer{
		adapter:   adapter,
		producer:  producer,
		tenantID:  tenantID,
		channelID: channelID,
	}
}

// ProcessWebhook processes a webhook and publishes messages to NATS
func (c *WebhookConsumer) ProcessWebhook(ctx context.Context, body []byte) error {
	processor := NewWebhookProcessor(c.adapter.GetConfig())

	// Parse webhook
	payload, err := processor.ParseWebhook(body)
	if err != nil {
		return err
	}

	// Process messages
	messages := processor.ExtractMessages(payload)
	for _, msg := range messages {
		inbound := &nats.InboundMessage{
			ID:          msg.ExternalID,
			TenantID:    c.tenantID,
			ChannelID:   c.channelID,
			ChannelType: "whatsapp_official",
			ExternalID:  msg.ExternalID,
			ContentType: string(msg.ContentType),
			Content:     msg.Content,
			Metadata:    msg.Metadata,
			Timestamp:   msg.Timestamp,
		}

		// Convert attachments
		for _, att := range msg.Attachments {
			inbound.Attachments = append(inbound.Attachments, nats.AttachmentData{
				Type:         att.Type,
				URL:          att.URL,
				Filename:     att.Filename,
				MimeType:     att.MimeType,
				SizeBytes:    att.SizeBytes,
				ThumbnailURL: att.ThumbnailURL,
				Metadata:     att.Metadata,
			})
		}

		if err := c.producer.PublishInbound(ctx, inbound); err != nil {
			logger.Error("Failed to publish inbound message: " + err.Error())
			continue
		}
	}

	// Process statuses
	statuses := processor.ExtractStatuses(payload)
	for _, status := range statuses {
		statusUpdate := &nats.StatusUpdate{
			MessageID:    status.MessageID,
			ChannelType:  "whatsapp_official",
			Status:       string(status.Status),
			ErrorMessage: status.ErrorMessage,
			Timestamp:    status.Timestamp,
		}

		if err := c.producer.PublishStatusUpdate(ctx, statusUpdate); err != nil {
			logger.Error("Failed to publish status update: " + err.Error())
			continue
		}
	}

	return nil
}
