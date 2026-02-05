package email

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the Email channel adapter with multi-provider support
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	client         *Client
	imapClient     *IMAPClient
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *Config
	stopCh         chan struct{}
	imapRunning    bool
}

// NewAdapter creates a new Email adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeEmail,
		Name:        "Email",
		Description: "Email channel with multi-provider support (SMTP, SendGrid, Mailgun, SES, Postmark)",
		Version:     "1.0.0",
		Author:      "Linktor Team",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{
				plugin.ContentTypeText,
				plugin.ContentTypeDocument,
			},
			SupportsMedia:           true, // Attachments
			SupportsLocation:        false,
			SupportsTemplates:       true, // HTML templates
			SupportsInteractive:     false,
			SupportsReadReceipts:    false,
			SupportsTypingIndicator: false,
			SupportsReactions:       false,
			SupportsReplies:         true, // Email threading
			SupportsForwarding:      false,
			MaxMessageLength:        0,                   // No practical limit
			MaxMediaSize:            25 * 1024 * 1024,    // 25MB typical limit
			MaxAttachments:          10,
			SupportedMediaTypes: []string{
				"application/pdf",
				"application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.ms-excel",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"image/jpeg",
				"image/png",
				"image/gif",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeEmail, info),
		config:      &Config{},
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	a.config = &Config{
		Provider:  Provider(config["provider"]),
		FromEmail: config["from_email"],
		FromName:  config["from_name"],
		ReplyTo:   config["reply_to"],

		// SMTP
		SMTPHost:       config["smtp_host"],
		SMTPUsername:   config["smtp_username"],
		SMTPPassword:   config["smtp_password"],
		SMTPEncryption: config["smtp_encryption"],

		// IMAP
		IMAPHost:     config["imap_host"],
		IMAPUsername: config["imap_username"],
		IMAPPassword: config["imap_password"],
		IMAPFolder:   config["imap_folder"],

		// SendGrid
		SendGridAPIKey: config["sendgrid_api_key"],

		// Mailgun
		MailgunDomain: config["mailgun_domain"],
		MailgunAPIKey: config["mailgun_api_key"],
		MailgunRegion: config["mailgun_region"],

		// SES
		SESRegion:      config["ses_region"],
		SESAccessKeyID: config["ses_access_key_id"],
		SESSecretKey:   config["ses_secret_key"],

		// Postmark
		PostmarkServerToken: config["postmark_server_token"],

		// Webhook
		WebhookSecret: config["webhook_secret"],
	}

	// Parse numeric config
	if port := config["smtp_port"]; port != "" {
		fmt.Sscanf(port, "%d", &a.config.SMTPPort)
	}
	if port := config["imap_port"]; port != "" {
		fmt.Sscanf(port, "%d", &a.config.IMAPPort)
	}
	if interval := config["imap_poll_interval"]; interval != "" {
		fmt.Sscanf(interval, "%d", &a.config.IMAPPollInterval)
	}

	return nil
}

// Connect establishes connection to the email provider
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create email client
	client, err := NewClient(a.config)
	if err != nil {
		return fmt.Errorf("failed to create email client: %w", err)
	}

	// Test connection
	if err := client.TestConnection(ctx); err != nil {
		return fmt.Errorf("failed to verify email connection: %w", err)
	}

	a.client = client
	a.stopCh = make(chan struct{})

	// Start IMAP polling if configured (for SMTP provider)
	if a.config.Provider == ProviderSMTP && a.config.IMAPHost != "" {
		imapClient, err := NewIMAPClient(a.config)
		if err == nil {
			if err := imapClient.Connect(ctx); err == nil {
				a.imapClient = imapClient
				go a.startIMAPPolling()
			}
		}
	}

	a.SetConnected(true)
	return nil
}

// Disconnect closes the email connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stopCh != nil {
		close(a.stopCh)
	}

	if a.imapClient != nil {
		a.imapClient.StopPolling()
		a.imapClient.Disconnect()
		a.imapClient = nil
	}

	a.client = nil
	a.SetConnected(false)
	return nil
}

// SendMessage sends an email
func (a *Adapter) SendMessage(ctx context.Context, msg *plugin.OutboundMessage) (*plugin.SendResult, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     "adapter not connected",
			Timestamp: time.Now(),
		}, nil
	}

	// Build email from outbound message
	email := a.buildOutboundEmail(msg)

	result, err := client.Send(ctx, email)
	if err != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	if !result.Success {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     result.Error,
			Timestamp: result.Timestamp,
		}, nil
	}

	return &plugin.SendResult{
		Success:    true,
		ExternalID: result.ExternalID,
		Status:     plugin.MessageStatusSent,
		Timestamp:  result.Timestamp,
	}, nil
}

// buildOutboundEmail converts a plugin.OutboundMessage to an OutboundEmail
func (a *Adapter) buildOutboundEmail(msg *plugin.OutboundMessage) *OutboundEmail {
	email := &OutboundEmail{
		To:       []string{msg.RecipientID},
		Metadata: make(map[string]string),
	}

	// Set subject from metadata
	if subject, ok := msg.Metadata["subject"]; ok {
		email.Subject = subject
	} else {
		email.Subject = "Message from Linktor"
	}

	// Set body based on content type
	if msg.ContentType == plugin.ContentTypeText {
		// Check if HTML is provided
		if html, ok := msg.Metadata["html_body"]; ok {
			email.HTMLBody = html
			email.TextBody = msg.Content // Plain text fallback
		} else {
			email.TextBody = msg.Content
		}
	} else {
		email.TextBody = msg.Content
	}

	// CC and BCC from metadata
	if cc, ok := msg.Metadata["cc"]; ok && cc != "" {
		email.CC = strings.Split(cc, ",")
	}
	if bcc, ok := msg.Metadata["bcc"]; ok && bcc != "" {
		email.BCC = strings.Split(bcc, ",")
	}

	// Reply-To
	if replyTo, ok := msg.Metadata["reply_to"]; ok {
		email.ReplyTo = replyTo
	}

	// Threading
	if inReplyTo, ok := msg.Metadata["in_reply_to"]; ok {
		email.InReplyTo = inReplyTo
	}
	if references, ok := msg.Metadata["references"]; ok {
		email.References = references
	}

	// Convert attachments
	for _, att := range msg.Attachments {
		emailAtt := &Attachment{
			Filename:    att.Filename,
			ContentType: att.MimeType,
			URL:         att.URL,
		}
		if data, ok := att.Metadata["content"]; ok {
			emailAtt.Content = []byte(data)
		}
		email.Attachments = append(email.Attachments, emailAtt)
	}

	// Copy metadata for tracking
	for k, v := range msg.Metadata {
		if k == "track_opens" || k == "track_clicks" || k == "tag" {
			email.Metadata[k] = v
		}
	}

	return email
}

// SetMessageHandler sets the handler for inbound messages
func (a *Adapter) SetMessageHandler(handler plugin.MessageHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messageHandler = handler
}

// SetStatusHandler sets the handler for status updates
func (a *Adapter) SetStatusHandler(handler plugin.StatusHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.statusHandler = handler
}

// GetWebhookPath returns the webhook path for this adapter
func (a *Adapter) GetWebhookPath() string {
	return "/webhooks/email"
}

// ValidateWebhook validates an incoming webhook request
func (a *Adapter) ValidateWebhook(headers map[string]string, body []byte) bool {
	// Validation depends on provider
	// This is a basic implementation - each provider has specific validation
	return true
}

// ProcessWebhook processes an incoming webhook payload
func (a *Adapter) ProcessWebhook(ctx context.Context, provider Provider, body []byte, headers map[string]string) error {
	a.mu.RLock()
	msgHandler := a.messageHandler
	statusHandler := a.statusHandler
	a.mu.RUnlock()

	payload, err := ParseWebhook(provider, body, headers)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	switch payload.Type {
	case "inbound":
		if msgHandler != nil && payload.IncomingEmail != nil {
			inbound := a.convertToInboundMessage(payload.IncomingEmail)
			return msgHandler(ctx, inbound)
		}

	case "status":
		if statusHandler != nil && payload.StatusCallback != nil {
			status := a.convertToStatusCallback(payload.StatusCallback)
			return statusHandler(ctx, status)
		}

	case "subscription_confirmation":
		// SES subscription confirmation - log for manual handling
		return nil
	}

	return nil
}

// convertToInboundMessage converts an IncomingEmail to plugin.InboundMessage
func (a *Adapter) convertToInboundMessage(email *IncomingEmail) *plugin.InboundMessage {
	inbound := &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  email.MessageID,
		SenderID:    email.From,
		SenderName:  email.FromName,
		ContentType: plugin.ContentTypeText,
		Timestamp:   email.ReceivedAt,
		Metadata: map[string]string{
			"subject":    email.Subject,
			"message_id": email.MessageID,
		},
	}

	// Prefer text body, but include both
	if email.TextBody != "" {
		inbound.Content = email.TextBody
	} else if email.HTMLBody != "" {
		inbound.Content = email.HTMLBody
		inbound.Metadata["content_type"] = "text/html"
	}

	// Add recipients to metadata
	if len(email.To) > 0 {
		inbound.Metadata["to"] = strings.Join(email.To, ",")
	}
	if len(email.CC) > 0 {
		inbound.Metadata["cc"] = strings.Join(email.CC, ",")
	}

	// Threading info
	if email.InReplyTo != "" {
		inbound.Metadata["in_reply_to"] = email.InReplyTo
	}
	if email.References != "" {
		inbound.Metadata["references"] = email.References
	}

	// Spam info
	if email.SpamScore > 0 {
		inbound.Metadata["spam_score"] = fmt.Sprintf("%.2f", email.SpamScore)
	}

	// Convert attachments
	for _, att := range email.Attachments {
		inbound.Attachments = append(inbound.Attachments, &plugin.Attachment{
			Type:      "document",
			Filename:  att.Filename,
			MimeType:  att.ContentType,
			SizeBytes: att.Size,
			URL:       att.URL,
			Metadata: map[string]string{
				"content_id": att.ContentID,
			},
		})
	}

	return inbound
}

// convertToStatusCallback converts an email StatusCallback to plugin.StatusCallback
func (a *Adapter) convertToStatusCallback(status *StatusCallback) *plugin.StatusCallback {
	var pluginStatus plugin.MessageStatus

	switch status.Status {
	case StatusQueued:
		pluginStatus = plugin.MessageStatusPending
	case StatusSent:
		pluginStatus = plugin.MessageStatusSent
	case StatusDelivered:
		pluginStatus = plugin.MessageStatusDelivered
	case StatusOpened, StatusClicked:
		pluginStatus = plugin.MessageStatusRead
	case StatusBounced, StatusFailed:
		pluginStatus = plugin.MessageStatusFailed
	case StatusSpam, StatusUnsubscribed:
		pluginStatus = plugin.MessageStatusFailed
	default:
		pluginStatus = plugin.MessageStatusSent
	}

	return &plugin.StatusCallback{
		MessageID:    status.MessageID,
		ExternalID:   status.ExternalID,
		Status:       pluginStatus,
		ErrorMessage: status.ErrorMessage,
		Timestamp:    status.Timestamp,
	}
}

// startIMAPPolling starts the IMAP polling goroutine
func (a *Adapter) startIMAPPolling() {
	a.mu.Lock()
	a.imapRunning = true
	imapClient := a.imapClient
	stopCh := a.stopCh
	a.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stopCh
		cancel()
	}()

	imapClient.StartPolling(ctx, func(email *IncomingEmail) {
		a.mu.RLock()
		handler := a.messageHandler
		a.mu.RUnlock()

		if handler != nil {
			inbound := a.convertToInboundMessage(email)
			handler(context.Background(), inbound)
		}
	})

	a.mu.Lock()
	a.imapRunning = false
	a.mu.Unlock()
}

// GetConnectionStatus returns detailed connection status
func (a *Adapter) GetConnectionStatus() *plugin.ConnectionStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := &plugin.ConnectionStatus{
		Connected: a.IsConnected(),
		Metadata:  make(map[string]string),
	}

	if a.IsConnected() {
		status.Status = "connected"
		if a.client != nil {
			status.Metadata["provider"] = a.client.GetProviderName()
		}
		status.Metadata["from_email"] = a.config.FromEmail
		if a.imapRunning {
			status.Metadata["imap_polling"] = "active"
		}
	} else {
		status.Status = "disconnected"
	}

	return status
}

// GetClient returns the email client
func (a *Adapter) GetClient() *Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// GetConfig returns the adapter configuration
func (a *Adapter) GetConfig() *Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
