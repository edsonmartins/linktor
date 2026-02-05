package facebook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the Facebook Messenger channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	client         *Client
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *FacebookConfig
}

// NewAdapter creates a new Facebook Messenger adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeFacebook,
		Name:        "Facebook Messenger",
		Description: "Facebook Messenger via Meta Graph API",
		Version:     "1.0.0",
		Author:      "Linktor Team",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{
				plugin.ContentTypeText,
				plugin.ContentTypeImage,
				plugin.ContentTypeVideo,
				plugin.ContentTypeAudio,
				plugin.ContentTypeDocument,
				plugin.ContentTypeLocation,
			},
			SupportsMedia:           true,
			SupportsLocation:        true,
			SupportsTemplates:       true,
			SupportsInteractive:     true,
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: true,
			SupportsReactions:       false,
			SupportsReplies:         false,
			SupportsForwarding:      false,
			MaxMessageLength:        2000,
			MaxMediaSize:            25 * 1024 * 1024, // 25MB
			MaxAttachments:          1,
			SupportedMediaTypes:     []string{"image/jpeg", "image/png", "image/gif", "video/mp4", "audio/mpeg", "application/pdf"},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeFacebook, info),
		config:      &FacebookConfig{},
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	// Parse configuration
	a.config = &FacebookConfig{
		AppID:           config["app_id"],
		AppSecret:       config["app_secret"],
		PageID:          config["page_id"],
		PageAccessToken: config["page_access_token"],
		UserAccessToken: config["user_access_token"],
		VerifyToken:     config["verify_token"],
		InstagramID:     config["instagram_id"],
	}

	return nil
}

// Connect establishes connection to the Facebook Messenger API
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate config
	if err := a.config.Validate(); err != nil {
		return err
	}

	// Create client
	client, err := NewClient(a.config)
	if err != nil {
		return fmt.Errorf("failed to create facebook client: %w", err)
	}

	a.client = client

	// Verify connection by getting page info
	_, err = client.GetPageInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify Facebook connection: %w", err)
	}

	// Subscribe to webhooks
	if err := client.SubscribeToWebhooks(ctx); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: Failed to subscribe to webhooks: %v\n", err)
	}

	a.SetConnected(true)
	return nil
}

// Disconnect closes the Facebook connection
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

// SendMessage sends a message via Facebook Messenger
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

	case plugin.ContentTypeDocument:
		if len(msg.Attachments) > 0 {
			resp, err = client.SendFile(ctx, msg.RecipientID, msg.Attachments[0].URL)
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

// SendTypingIndicator sends a typing indicator
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	if indicator.IsTyping {
		return client.SendTypingOn(ctx, indicator.RecipientID)
	}
	return client.SendTypingOff(ctx, indicator.RecipientID)
}

// SendReadReceipt sends a read receipt
func (a *Adapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	return client.MarkSeen(ctx, receipt.RecipientID)
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
	return "/webhooks/facebook"
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
	if !IsMessengerWebhook(payload) {
		return nil
	}

	fbMessages := ExtractMessages(payload)
	var messages []*plugin.InboundMessage

	for _, fbMsg := range fbMessages {
		// Skip echo messages
		if fbMsg.IsEcho {
			continue
		}

		msg := &plugin.InboundMessage{
			ID:          uuid.New().String(),
			ExternalID:  fbMsg.ExternalID,
			SenderID:    fbMsg.SenderID,
			ContentType: plugin.ContentTypeText,
			Content:     fbMsg.Text,
			Timestamp:   fbMsg.Timestamp,
			Metadata: map[string]string{
				"page_id":      fbMsg.PageID,
				"recipient_id": fbMsg.RecipientID,
			},
		}

		// Handle quick reply payload
		if fbMsg.QuickReply != "" {
			msg.Metadata["quick_reply"] = fbMsg.QuickReply
		}

		// Handle attachments
		if len(fbMsg.Attachments) > 0 {
			att := fbMsg.Attachments[0]
			msg.ContentType = contentTypeFromFB(att.Type)
			msg.Attachments = []*plugin.Attachment{
				{
					Type: GetAttachmentType(att.Type),
					URL:  att.URL,
				},
			}

			// Handle location
			if att.Type == "location" {
				msg.Metadata["lat"] = fmt.Sprintf("%f", att.Lat)
				msg.Metadata["long"] = fmt.Sprintf("%f", att.Long)
			}
		}

		messages = append(messages, msg)
	}

	return messages
}

// contentTypeFromFB converts Facebook attachment type to plugin ContentType
func contentTypeFromFB(fbType string) plugin.ContentType {
	switch fbType {
	case "image":
		return plugin.ContentTypeImage
	case "video":
		return plugin.ContentTypeVideo
	case "audio":
		return plugin.ContentTypeAudio
	case "file":
		return plugin.ContentTypeDocument
	case "location":
		return plugin.ContentTypeLocation
	default:
		return plugin.ContentTypeText
	}
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
