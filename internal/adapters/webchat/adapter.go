package webchat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the WebChat channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	hub            *Hub
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *Config
}

// Config holds WebChat adapter configuration
type Config struct {
	WidgetTitle       string `json:"widget_title"`
	WidgetColor       string `json:"widget_color"`
	WelcomeMessage    string `json:"welcome_message"`
	OfflineMessage    string `json:"offline_message"`
	AvatarURL         string `json:"avatar_url"`
	AllowAttachments  bool   `json:"allow_attachments"`
	MaxFileSize       int64  `json:"max_file_size"`
	AllowedFileTypes  string `json:"allowed_file_types"`
	RequireEmail      bool   `json:"require_email"`
	RequireName       bool   `json:"require_name"`
	BusinessHoursOnly bool   `json:"business_hours_only"`
}

// NewAdapter creates a new WebChat adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeWebChat,
		Name:        "Web Chat",
		Description: "Real-time web chat widget for websites",
		Version:     "1.0.0",
		Author:      "Linktor Team",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{
				plugin.ContentTypeText,
				plugin.ContentTypeImage,
				plugin.ContentTypeDocument,
			},
			SupportsMedia:           true,
			SupportsLocation:        false,
			SupportsTemplates:       false,
			SupportsInteractive:     true,
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: true,
			SupportsReactions:       false,
			SupportsReplies:         true,
			SupportsForwarding:      false,
			MaxMessageLength:        4096,
			MaxMediaSize:            10 * 1024 * 1024, // 10MB
			MaxAttachments:          5,
			SupportedMediaTypes: []string{
				"image/jpeg", "image/png", "image/gif", "image/webp",
				"application/pdf", "application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeWebChat, info),
		config:      &Config{},
	}
}

// Initialize configures the adapter
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	// Parse configuration
	a.config = &Config{
		WidgetTitle:      getOrDefault(config, "widget_title", "Chat with us"),
		WidgetColor:      getOrDefault(config, "widget_color", "#007bff"),
		WelcomeMessage:   getOrDefault(config, "welcome_message", "Hello! How can we help you today?"),
		OfflineMessage:   getOrDefault(config, "offline_message", "We're currently offline. Leave a message and we'll get back to you."),
		AvatarURL:        config["avatar_url"],
		AllowAttachments: config["allow_attachments"] == "true",
		RequireEmail:     config["require_email"] == "true",
		RequireName:      config["require_name"] == "true",
	}

	return nil
}

// Connect starts the WebSocket hub
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.hub != nil {
		return nil // Already connected
	}

	a.hub = NewHub()
	go a.hub.Run()

	a.SetConnected(true)
	return nil
}

// Disconnect stops the WebSocket hub
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.hub != nil {
		a.hub.Stop()
		a.hub = nil
	}

	a.SetConnected(false)
	return nil
}

// SendMessage sends a message to a WebSocket client
func (a *Adapter) SendMessage(ctx context.Context, msg *plugin.OutboundMessage) (*plugin.SendResult, error) {
	a.mu.RLock()
	hub := a.hub
	a.mu.RUnlock()

	if hub == nil {
		return &plugin.SendResult{
			Success: false,
			Status:  plugin.MessageStatusFailed,
			Error:   "adapter not connected",
		}, nil
	}

	// Find the client by recipient ID (session ID)
	client := hub.GetClient(msg.RecipientID)
	if client == nil {
		return &plugin.SendResult{
			Success: false,
			Status:  plugin.MessageStatusFailed,
			Error:   "client not connected",
		}, nil
	}

	// Build WebSocket message
	wsMsg := &WebSocketMessage{
		Type: MessageTypeMessage,
		Payload: MessagePayload{
			ID:          msg.ID,
			ContentType: string(msg.ContentType),
			Content:     msg.Content,
			SenderType:  "user",
			SenderID:    msg.Metadata["sender_id"],
			SenderName:  msg.Metadata["sender_name"],
			Attachments: convertAttachments(msg.Attachments),
			Timestamp:   time.Now().Format(time.RFC3339),
		},
	}

	if err := client.SendMessage(wsMsg); err != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	return &plugin.SendResult{
		Success:    true,
		ExternalID: msg.ID,
		Status:     plugin.MessageStatusDelivered, // WebSocket delivery is instant
		Timestamp:  time.Now(),
	}, nil
}

// SendTypingIndicator sends a typing indicator to a client
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	a.mu.RLock()
	hub := a.hub
	a.mu.RUnlock()

	if hub == nil {
		return fmt.Errorf("adapter not connected")
	}

	client := hub.GetClient(indicator.RecipientID)
	if client == nil {
		return fmt.Errorf("client not connected")
	}

	wsMsg := &WebSocketMessage{
		Type: MessageTypeTyping,
		Payload: MessagePayload{
			IsTyping: indicator.IsTyping,
		},
	}

	return client.SendMessage(wsMsg)
}

// SendReadReceipt sends a read receipt to a client
func (a *Adapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	a.mu.RLock()
	hub := a.hub
	a.mu.RUnlock()

	if hub == nil {
		return fmt.Errorf("adapter not connected")
	}

	client := hub.GetClient(receipt.RecipientID)
	if client == nil {
		return fmt.Errorf("client not connected")
	}

	wsMsg := &WebSocketMessage{
		Type: MessageTypeRead,
		Payload: MessagePayload{
			ID: receipt.MessageID,
		},
	}

	return client.SendMessage(wsMsg)
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

// GetHub returns the WebSocket hub
func (a *Adapter) GetHub() *Hub {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.hub
}

// GetConfig returns the adapter configuration
func (a *Adapter) GetConfig() *Config {
	return a.config
}

// HandleInboundMessage processes an incoming message from a WebSocket client
func (a *Adapter) HandleInboundMessage(ctx context.Context, sessionID string, msg *MessagePayload) error {
	a.mu.RLock()
	handler := a.messageHandler
	a.mu.RUnlock()

	if handler == nil {
		return fmt.Errorf("no message handler configured")
	}

	// Convert attachments
	var attachments []*plugin.Attachment
	for _, att := range msg.Attachments {
		attachments = append(attachments, &plugin.Attachment{
			Type:      att.Type,
			URL:       att.URL,
			Filename:  att.Filename,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}

	inbound := &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  msg.ID,
		SenderID:    sessionID,
		SenderName:  msg.SenderName,
		ContentType: plugin.ContentType(msg.ContentType),
		Content:     msg.Content,
		Metadata: map[string]string{
			"session_id": sessionID,
		},
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return handler(ctx, inbound)
}

// HandleClientConnect is called when a new client connects
func (a *Adapter) HandleClientConnect(ctx context.Context, sessionID string, metadata map[string]string) error {
	// Send welcome message if configured
	if a.config.WelcomeMessage != "" {
		a.mu.RLock()
		hub := a.hub
		a.mu.RUnlock()

		if hub != nil {
			client := hub.GetClient(sessionID)
			if client != nil {
				wsMsg := &WebSocketMessage{
					Type: MessageTypeMessage,
					Payload: MessagePayload{
						ID:          uuid.New().String(),
						ContentType: "text",
						Content:     a.config.WelcomeMessage,
						SenderType:  "system",
						Timestamp:   time.Now().Format(time.RFC3339),
					},
				}
				client.SendMessage(wsMsg)
			}
		}
	}

	return nil
}

// HandleClientDisconnect is called when a client disconnects
func (a *Adapter) HandleClientDisconnect(ctx context.Context, sessionID string) error {
	// Could publish an event here if needed
	return nil
}

// Helper functions

func getOrDefault(config map[string]string, key, defaultValue string) string {
	if v, ok := config[key]; ok && v != "" {
		return v
	}
	return defaultValue
}

func convertAttachments(attachments []*plugin.Attachment) []AttachmentPayload {
	result := make([]AttachmentPayload, 0, len(attachments))
	for _, att := range attachments {
		result = append(result, AttachmentPayload{
			Type:      att.Type,
			URL:       att.URL,
			Filename:  att.Filename,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}
	return result
}

// Ensure Adapter implements ChannelAdapter
var _ plugin.ChannelAdapter = (*Adapter)(nil)
