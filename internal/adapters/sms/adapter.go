package sms

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the Twilio SMS channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	client         *Client
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *TwilioConfig
}

// NewAdapter creates a new SMS/Twilio adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeSMS,
		Name:        "SMS (Twilio)",
		Description: "SMS messaging via Twilio",
		Version:     "1.0.0",
		Author:      "Linktor Team",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{
				plugin.ContentTypeText,
				plugin.ContentTypeImage,
			},
			SupportsMedia:           true, // MMS
			SupportsLocation:        false,
			SupportsTemplates:       false,
			SupportsInteractive:     false,
			SupportsReadReceipts:    false,
			SupportsTypingIndicator: false,
			SupportsReactions:       false,
			SupportsReplies:         false,
			SupportsForwarding:      false,
			MaxMessageLength:        1600, // SMS segment limit
			MaxMediaSize:            5 * 1024 * 1024, // 5MB for MMS
			MaxAttachments:          10, // MMS supports up to 10 media items
			SupportedMediaTypes: []string{
				"image/jpeg", "image/png", "image/gif",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeSMS, info),
		config:      &TwilioConfig{},
	}
}

// Initialize configures the adapter with Twilio credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	// Parse configuration
	a.config = &TwilioConfig{
		AccountSID:          config["account_sid"],
		AuthToken:           config["auth_token"],
		APIKeySID:           config["api_key_sid"],
		APIKeySecret:        config["api_key_secret"],
		PhoneNumber:         config["phone_number"],
		MessagingServiceSID: config["messaging_service_sid"],
		StatusCallbackURL:   config["status_callback_url"],
	}

	// Validate required fields
	if a.config.AccountSID == "" {
		return fmt.Errorf("account_sid is required")
	}

	if a.config.AuthToken == "" && (a.config.APIKeySID == "" || a.config.APIKeySecret == "") {
		return fmt.Errorf("either auth_token or api_key_sid+api_key_secret is required")
	}

	if a.config.PhoneNumber == "" && a.config.MessagingServiceSID == "" {
		return fmt.Errorf("either phone_number or messaging_service_sid is required")
	}

	return nil
}

// Connect establishes connection to Twilio API
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create API client
	client, err := NewClient(a.config)
	if err != nil {
		return fmt.Errorf("failed to create twilio client: %w", err)
	}

	a.client = client

	// Verify connection by fetching account info
	_, err = client.GetAccountInfo()
	if err != nil {
		return fmt.Errorf("failed to verify Twilio connection: %w", err)
	}

	a.SetConnected(true)
	return nil
}

// Disconnect closes the Twilio connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.client = nil
	a.SetConnected(false)

	return nil
}

// SendMessage sends an SMS/MMS via Twilio
func (a *Adapter) SendMessage(ctx context.Context, msg *plugin.OutboundMessage) (*plugin.SendResult, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return &plugin.SendResult{
			Success: false,
			Status:  plugin.MessageStatusFailed,
			Error:   "adapter not connected",
		}, nil
	}

	// Validate recipient phone number
	to := msg.RecipientID
	if !ValidatePhoneNumber(to) {
		// Try to format it
		to = FormatPhoneNumber(to, "")
		if !ValidatePhoneNumber(to) {
			return &plugin.SendResult{
				Success:   false,
				Status:    plugin.MessageStatusFailed,
				Error:     fmt.Sprintf("invalid phone number: %s", msg.RecipientID),
				Timestamp: time.Now(),
			}, nil
		}
	}

	// Collect media URLs for MMS
	var mediaURLs []string
	if msg.ContentType == plugin.ContentTypeImage {
		if len(msg.Attachments) > 0 {
			for _, att := range msg.Attachments {
				if att.URL != "" {
					mediaURLs = append(mediaURLs, att.URL)
				}
			}
		} else if mediaURL := msg.Metadata["media_url"]; mediaURL != "" {
			mediaURLs = append(mediaURLs, mediaURL)
		}
	}

	// Send message
	result, err := client.SendMessage(to, msg.Content, mediaURLs)
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
			Timestamp: time.Now(),
		}, nil
	}

	// Map Twilio status to plugin status
	status := plugin.MessageStatusSent
	if result.Status == StatusQueued {
		status = plugin.MessageStatusPending
	}

	return &plugin.SendResult{
		Success:    true,
		ExternalID: result.MessageSID,
		Status:     status,
		Timestamp:  time.Now(),
	}, nil
}

// SendTypingIndicator - SMS doesn't support typing indicators
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	// SMS doesn't support typing indicators
	return nil
}

// SendReadReceipt - SMS doesn't support read receipts
func (a *Adapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	// SMS doesn't support read receipts
	return nil
}

// UploadMedia - Twilio MMS uses URLs, not pre-upload
func (a *Adapter) UploadMedia(ctx context.Context, media *plugin.Media) (*plugin.MediaUpload, error) {
	// Twilio doesn't require pre-uploading media
	// Media must be hosted at a publicly accessible URL
	return &plugin.MediaUpload{
		Success: false,
		Error:   "twilio MMS requires media at a public URL, upload to a file storage first",
	}, nil
}

// DownloadMedia - Not implemented for SMS
func (a *Adapter) DownloadMedia(ctx context.Context, mediaID string) (*plugin.Media, error) {
	// MMS media is accessed via URL, not downloaded through Twilio
	return nil, fmt.Errorf("download media via URL directly, not through twilio adapter")
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
	return "/api/v1/webhooks/sms"
}

// ValidateWebhook validates a Twilio webhook request
func (a *Adapter) ValidateWebhook(headers map[string]string, body []byte) bool {
	// Twilio provides X-Twilio-Signature header for validation
	// For now, we trust webhooks from our registered URL
	// TODO: Implement signature validation using auth token
	return true
}

// HandleWebhook processes incoming webhook requests
func (a *Adapter) HandleWebhook(ctx context.Context, body []byte) error {
	a.mu.RLock()
	msgHandler := a.messageHandler
	a.mu.RUnlock()

	// Parse webhook
	payload, webhookType, err := ParseWebhook(body)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	switch webhookType {
	case WebhookTypeIncoming:
		if msgHandler == nil {
			return fmt.Errorf("no message handler configured")
		}

		// Convert to plugin format
		inbound := a.toInboundMessage(payload)

		// Call handler
		return msgHandler(ctx, inbound)

	case WebhookTypeStatus:
		a.mu.RLock()
		statusHandler := a.statusHandler
		a.mu.RUnlock()

		if statusHandler == nil {
			// Status handler is optional
			return nil
		}

		// Convert to plugin format
		status := a.toStatusCallback(payload)

		// Call handler
		return statusHandler(ctx, status)
	}

	return nil
}

// toInboundMessage converts a Twilio webhook payload to plugin format
func (a *Adapter) toInboundMessage(payload *WebhookPayload) *plugin.InboundMessage {
	inbound := &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  payload.MessageSID,
		SenderID:    payload.From,
		ContentType: plugin.ContentTypeText,
		Content:     payload.Body,
		Timestamp:   time.Now(),
		Metadata: map[string]string{
			"account_sid":  payload.AccountSID,
			"from_city":    payload.FromCity,
			"from_state":   payload.FromState,
			"from_zip":     payload.FromZip,
			"from_country": payload.FromCountry,
			"to":           payload.To,
		},
	}

	// Handle MMS media
	numMedia := 0
	if payload.NumMedia != "" {
		fmt.Sscanf(payload.NumMedia, "%d", &numMedia)
	}

	if numMedia > 0 {
		inbound.ContentType = plugin.ContentTypeImage
		// Media URLs would be in MediaUrl0, MediaUrl1, etc.
		// For now, we'd need to parse these from the form data
	}

	return inbound
}

// toStatusCallback converts a Twilio status webhook to plugin format
func (a *Adapter) toStatusCallback(payload *WebhookPayload) *plugin.StatusCallback {
	twilioStatus := payload.MessageStatus
	if twilioStatus == "" {
		twilioStatus = payload.SmsStatus
	}

	// Map Twilio status to plugin status
	var status plugin.MessageStatus
	switch ParseMessageStatus(twilioStatus) {
	case StatusDelivered:
		status = plugin.MessageStatusDelivered
	case StatusRead:
		status = plugin.MessageStatusRead
	case StatusFailed, StatusUndelivered:
		status = plugin.MessageStatusFailed
	case StatusSent, StatusQueued, StatusSending, StatusAccepted:
		status = plugin.MessageStatusSent
	default:
		status = plugin.MessageStatusPending
	}

	return &plugin.StatusCallback{
		ExternalID:   payload.MessageSID,
		Status:       status,
		ErrorMessage: payload.ErrorMessage,
		Timestamp:    time.Now(),
	}
}

// GetClient returns the Twilio client
func (a *Adapter) GetClient() *Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// GetConfig returns the adapter configuration
func (a *Adapter) GetConfig() *TwilioConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
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
		if a.config.PhoneNumber != "" {
			status.Metadata["phone_number"] = a.config.PhoneNumber
		}
		if a.config.MessagingServiceSID != "" {
			status.Metadata["messaging_service_sid"] = a.config.MessagingServiceSID
		}
	} else {
		status.Status = "disconnected"
	}

	return status
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
