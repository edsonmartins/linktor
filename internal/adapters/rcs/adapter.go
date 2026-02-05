package rcs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the RCS channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	client         *Client
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *Config
}

// NewAdapter creates a new RCS adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeRCS,
		Name:        "RCS Business Messaging",
		Description: "Rich Communication Services integration",
		Version:     "1.0.0",
		Author:      "Linktor Team",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{
				plugin.ContentTypeText,
				plugin.ContentTypeImage,
				plugin.ContentTypeVideo,
				plugin.ContentTypeDocument,
				plugin.ContentTypeLocation,
				plugin.ContentTypeInteractive,
			},
			SupportsMedia:           true,
			SupportsLocation:        true,
			SupportsTemplates:       false,
			SupportsInteractive:     true, // Rich cards, carousels
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: true,
			SupportsReactions:       false,
			SupportsReplies:         true,
			SupportsForwarding:      false,
			MaxMessageLength:        3072,
			MaxMediaSize:            10 * 1024 * 1024, // 10MB
			MaxAttachments:          1,
			SupportedMediaTypes: []string{
				"image/jpeg", "image/png", "image/gif",
				"video/mp4", "video/3gpp",
				"audio/mp3", "audio/aac",
				"application/pdf",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeRCS, info),
		config:      &Config{},
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	a.config = &Config{
		Provider:      Provider(config["provider"]),
		AgentID:       config["agent_id"],
		APIKey:        config["api_key"],
		APISecret:     config["api_secret"],
		WebhookURL:    config["webhook_url"],
		WebhookSecret: config["webhook_secret"],
		BaseURL:       config["base_url"],
		SenderID:      config["sender_id"],
	}

	if a.config.Provider == "" {
		a.config.Provider = ProviderZenvia // Default provider
	}

	return nil
}

// Connect establishes connection to the RCS provider
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	client, err := NewClient(a.config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	a.client = client

	// Verify connection by getting agent info
	_, err = client.GetAgentInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify RCS connection: %w", err)
	}

	a.SetConnected(true)
	return nil
}

// Disconnect closes the RCS connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.client = nil
	a.SetConnected(false)
	return nil
}

// SendMessage sends a message via RCS
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

	// Build RCS message
	rcsMsg := &OutboundMessage{
		To:       msg.RecipientID,
		Metadata: msg.Metadata,
	}

	switch msg.ContentType {
	case plugin.ContentTypeText:
		rcsMsg.Text = msg.Content

		// Check for rich card in metadata
		if msg.Metadata != nil {
			if cardTitle, ok := msg.Metadata["card_title"]; ok {
				rcsMsg.Card = &RichCard{
					Title:       cardTitle,
					Description: msg.Content,
				}
				if mediaURL, ok := msg.Metadata["card_media_url"]; ok {
					rcsMsg.Card.MediaURL = mediaURL
					rcsMsg.Card.MediaType = msg.Metadata["card_media_type"]
				}
				rcsMsg.Text = ""
			}
		}

	case plugin.ContentTypeImage:
		if len(msg.Attachments) > 0 {
			rcsMsg.MediaURL = msg.Attachments[0].URL
			rcsMsg.MediaType = msg.Attachments[0].MimeType
			if msg.Content != "" {
				// Add as rich card with caption
				rcsMsg.Card = &RichCard{
					Description: msg.Content,
					MediaURL:    rcsMsg.MediaURL,
					MediaType:   rcsMsg.MediaType,
				}
				rcsMsg.MediaURL = ""
			}
		}

	case plugin.ContentTypeVideo:
		if len(msg.Attachments) > 0 {
			rcsMsg.MediaURL = msg.Attachments[0].URL
			rcsMsg.MediaType = msg.Attachments[0].MimeType
		}

	case plugin.ContentTypeDocument:
		if len(msg.Attachments) > 0 {
			rcsMsg.MediaURL = msg.Attachments[0].URL
			rcsMsg.MediaType = msg.Attachments[0].MimeType
		}

	case plugin.ContentTypeLocation:
		if msg.Metadata != nil {
			lat, lon := 0.0, 0.0
			fmt.Sscanf(msg.Metadata["latitude"], "%f", &lat)
			fmt.Sscanf(msg.Metadata["longitude"], "%f", &lon)

			rcsMsg.Suggestions = []Suggestion{
				{
					Type: SuggestionTypeLocation,
					Text: msg.Content,
					Location: &Location{
						Latitude:  lat,
						Longitude: lon,
						Label:     msg.Metadata["name"],
					},
				},
			}
		}

	case plugin.ContentTypeInteractive:
		// Parse suggestions from metadata
		if msg.Metadata != nil {
			if suggestionsJSON, ok := msg.Metadata["suggestions"]; ok {
				// Parse suggestions JSON
				_ = suggestionsJSON // Would parse JSON to []Suggestion
			}
		}
		rcsMsg.Text = msg.Content

	default:
		rcsMsg.Text = msg.Content
	}

	// Add suggestions if provided in metadata
	if msg.Metadata != nil {
		if reply1, ok := msg.Metadata["quick_reply_1"]; ok {
			rcsMsg.Suggestions = append(rcsMsg.Suggestions, Suggestion{
				Type:         SuggestionTypeReply,
				Text:         reply1,
				PostbackData: msg.Metadata["quick_reply_1_data"],
			})
		}
		if reply2, ok := msg.Metadata["quick_reply_2"]; ok {
			rcsMsg.Suggestions = append(rcsMsg.Suggestions, Suggestion{
				Type:         SuggestionTypeReply,
				Text:         reply2,
				PostbackData: msg.Metadata["quick_reply_2_data"],
			})
		}
		if reply3, ok := msg.Metadata["quick_reply_3"]; ok {
			rcsMsg.Suggestions = append(rcsMsg.Suggestions, Suggestion{
				Type:         SuggestionTypeReply,
				Text:         reply3,
				PostbackData: msg.Metadata["quick_reply_3_data"],
			})
		}
	}

	// Send message
	result, err := client.SendMessage(ctx, rcsMsg)
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

	return &plugin.SendResult{
		Success:    true,
		ExternalID: result.MessageID,
		Status:     plugin.MessageStatusSent,
		Timestamp:  result.Timestamp,
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
	return "/webhooks/rcs"
}

// ValidateWebhook validates a webhook request
func (a *Adapter) ValidateWebhook(headers map[string]string, body []byte) bool {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return false
	}

	signature := headers["X-Signature"]
	if signature == "" {
		signature = headers["x-signature"]
	}
	if signature == "" {
		signature = headers["X-Hub-Signature-256"]
	}
	if signature == "" {
		signature = headers["x-hub-signature-256"]
	}

	return client.ValidateWebhook(signature, body)
}

// ProcessWebhook processes an incoming webhook payload
func (a *Adapter) ProcessWebhook(ctx context.Context, body []byte) error {
	a.mu.RLock()
	client := a.client
	msgHandler := a.messageHandler
	statusHandler := a.statusHandler
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	payload, err := client.ParseWebhook(body)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	switch payload.Type {
	case "message":
		if msgHandler != nil && payload.Message != nil {
			inbound := convertToInboundMessage(payload.Message)
			return msgHandler(ctx, inbound)
		}

	case "status":
		if statusHandler != nil && payload.Status != nil {
			status := convertToStatusCallback(payload.Status)
			return statusHandler(ctx, status)
		}
	}

	return nil
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
		status.Metadata["provider"] = string(a.config.Provider)
		status.Metadata["agent_id"] = a.config.AgentID
	} else {
		status.Status = "disconnected"
	}

	return status
}

// GetClient returns the RCS client
func (a *Adapter) GetClient() *Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// convertToInboundMessage converts an RCS IncomingMessage to plugin.InboundMessage
func convertToInboundMessage(msg *IncomingMessage) *plugin.InboundMessage {
	inbound := &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  msg.ExternalID,
		SenderID:    msg.SenderPhone,
		Content:     msg.Text,
		ContentType: plugin.ContentTypeText,
		Timestamp:   msg.Timestamp,
		Metadata: map[string]string{
			"agent_id": msg.AgentID,
		},
	}

	// Handle media
	if msg.MediaURL != "" {
		inbound.ContentType = plugin.ContentTypeImage // Default to image
		if msg.MediaType != "" {
			switch {
			case len(msg.MediaType) >= 6 && msg.MediaType[:6] == "video/":
				inbound.ContentType = plugin.ContentTypeVideo
			case len(msg.MediaType) >= 6 && msg.MediaType[:6] == "audio/":
				inbound.ContentType = plugin.ContentTypeAudio
			case msg.MediaType == "application/pdf":
				inbound.ContentType = plugin.ContentTypeDocument
			}
		}
		inbound.Attachments = []*plugin.Attachment{
			{
				Type:     string(inbound.ContentType),
				URL:      msg.MediaURL,
				MimeType: msg.MediaType,
			},
		}
	}

	// Handle location
	if msg.Location != nil {
		inbound.ContentType = plugin.ContentTypeLocation
		inbound.Content = msg.Location.Label
		inbound.Metadata["latitude"] = fmt.Sprintf("%f", msg.Location.Latitude)
		inbound.Metadata["longitude"] = fmt.Sprintf("%f", msg.Location.Longitude)
	}

	// Handle suggestion postback
	if msg.Suggestion != nil {
		inbound.Metadata["postback_data"] = msg.Suggestion.PostbackData
		inbound.Metadata["suggestion_text"] = msg.Suggestion.Text
	}

	return inbound
}

// convertToStatusCallback converts an RCS DeliveryReport to plugin.StatusCallback
func convertToStatusCallback(report *DeliveryReport) *plugin.StatusCallback {
	var status plugin.MessageStatus

	switch report.Status {
	case StatusPending:
		status = plugin.MessageStatusPending
	case StatusSent:
		status = plugin.MessageStatusSent
	case StatusDelivered:
		status = plugin.MessageStatusDelivered
	case StatusRead:
		status = plugin.MessageStatusRead
	case StatusFailed:
		status = plugin.MessageStatusFailed
	default:
		status = plugin.MessageStatusPending
	}

	return &plugin.StatusCallback{
		MessageID:    report.MessageID,
		ExternalID:   report.MessageID,
		Status:       status,
		ErrorMessage: report.Error,
		Timestamp:    report.Timestamp,
	}
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
