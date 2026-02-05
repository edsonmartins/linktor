package instagram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the Instagram DM channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	client         *Client
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *InstagramConfig
}

// NewAdapter creates a new Instagram DM adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeInstagram,
		Name:        "Instagram Direct Messages",
		Description: "Instagram DM via Meta Graph API",
		Version:     "1.0.0",
		Author:      "Linktor Team",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{
				plugin.ContentTypeText,
				plugin.ContentTypeImage,
				plugin.ContentTypeVideo,
				plugin.ContentTypeAudio,
			},
			SupportsMedia:           true,
			SupportsLocation:        false,
			SupportsTemplates:       false,
			SupportsInteractive:     false,
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: false,
			SupportsReactions:       true,
			SupportsReplies:         false,
			SupportsForwarding:      false,
			MaxMessageLength:        1000,
			MaxMediaSize:            8 * 1024 * 1024, // 8MB
			MaxAttachments:          1,
			SupportedMediaTypes:     []string{"image/jpeg", "image/png", "video/mp4", "audio/mp4"},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeInstagram, info),
		config:      &InstagramConfig{},
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	// Parse configuration
	a.config = &InstagramConfig{
		InstagramID:     config["instagram_id"],
		AccessToken:     config["access_token"],
		AppID:           config["app_id"],
		AppSecret:       config["app_secret"],
		VerifyToken:     config["verify_token"],
		PageID:          config["page_id"],
		PageAccessToken: config["page_access_token"],
	}

	// Parse expiration if provided
	if expiresAt := config["expires_at"]; expiresAt != "" {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			a.config.ExpiresAt = t
		}
	}

	return nil
}

// Connect establishes connection to the Instagram API
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate config
	if err := a.config.Validate(); err != nil {
		return err
	}

	// Check token expiration
	if a.config.IsExpired() {
		return ErrTokenExpired
	}

	// Create client
	client, err := NewClient(a.config)
	if err != nil {
		return fmt.Errorf("failed to create instagram client: %w", err)
	}

	a.client = client

	// Verify connection by getting account info
	_, err = client.GetAccountInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify Instagram connection: %w", err)
	}

	// Subscribe to webhooks
	if err := client.SubscribeToWebhooks(ctx); err != nil {
		fmt.Printf("Warning: Failed to subscribe to webhooks: %v\n", err)
	}

	a.SetConnected(true)
	return nil
}

// Disconnect closes the Instagram connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil {
		// Unsubscribe from webhooks
		if err := a.client.UnsubscribeFromWebhooks(ctx); err != nil {
			fmt.Printf("Warning: Failed to unsubscribe from webhooks: %v\n", err)
		}
	}

	a.client = nil
	a.SetConnected(false)
	return nil
}

// SendMessage sends a message via Instagram DM
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

	var resp *meta.SendMessageResponse
	var err error

	switch msg.ContentType {
	case plugin.ContentTypeText:
		resp, err = client.SendTextMessage(ctx, msg.RecipientID, msg.Content)

	case plugin.ContentTypeImage:
		if len(msg.Attachments) > 0 {
			resp, err = client.SendImage(ctx, msg.RecipientID, msg.Attachments[0].URL)
		}

	case plugin.ContentTypeVideo:
		if len(msg.Attachments) > 0 {
			resp, err = client.SendVideo(ctx, msg.RecipientID, msg.Attachments[0].URL)
		}

	case plugin.ContentTypeAudio:
		if len(msg.Attachments) > 0 {
			resp, err = client.SendAudio(ctx, msg.RecipientID, msg.Attachments[0].URL)
		}

	default:
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     fmt.Sprintf("unsupported content type: %s", msg.ContentType),
			Timestamp: time.Now(),
		}, nil
	}

	if err != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	return &plugin.SendResult{
		Success:    true,
		ExternalID: resp.MessageID,
		Status:     plugin.MessageStatusSent,
		Timestamp:  time.Now(),
	}, nil
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
	return "/webhooks/instagram"
}

// ValidateWebhook validates an incoming webhook request
func (a *Adapter) ValidateWebhook(headers map[string]string, body []byte) bool {
	if a.config.AppSecret == "" {
		return true
	}
	signature := headers["X-Hub-Signature-256"]
	return meta.ValidateWebhookSignature(a.config.AppSecret, body, signature)
}

// ProcessWebhook processes a webhook payload and returns inbound messages
func (a *Adapter) ProcessWebhook(payload *WebhookPayload) []*plugin.InboundMessage {
	if !IsInstagramWebhook(payload) && !IsInstagramViaPageWebhook(payload) {
		return nil
	}

	igMessages := ExtractMessages(payload)
	var messages []*plugin.InboundMessage

	for _, igMsg := range igMessages {
		// Skip echo messages
		if igMsg.IsEcho {
			continue
		}

		// Skip deleted messages
		if igMsg.IsDeleted {
			continue
		}

		msg := &plugin.InboundMessage{
			ID:          uuid.New().String(),
			ExternalID:  igMsg.ExternalID,
			SenderID:    igMsg.SenderID,
			ContentType: plugin.ContentTypeText,
			Content:     igMsg.Text,
			Timestamp:   igMsg.Timestamp,
			Metadata: map[string]string{
				"instagram_id": igMsg.InstagramID,
				"recipient_id": igMsg.RecipientID,
			},
		}

		// Handle attachments
		if len(igMsg.Attachments) > 0 {
			att := igMsg.Attachments[0]
			msg.ContentType = contentTypeFromIG(att.Type)
			msg.Attachments = []*plugin.Attachment{
				{
					Type: GetAttachmentType(att.Type),
					URL:  att.URL,
				},
			}
		}

		messages = append(messages, msg)
	}

	return messages
}

// contentTypeFromIG converts Instagram attachment type to plugin ContentType
func contentTypeFromIG(igType string) plugin.ContentType {
	switch igType {
	case "image":
		return plugin.ContentTypeImage
	case "video":
		return plugin.ContentTypeVideo
	case "audio":
		return plugin.ContentTypeAudio
	case "share":
		return plugin.ContentTypeText // Shared posts are converted to text
	default:
		return plugin.ContentTypeText
	}
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
