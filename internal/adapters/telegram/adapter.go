package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the Telegram Bot API channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu             sync.RWMutex
	client         *Client
	messageHandler plugin.MessageHandler
	statusHandler  plugin.StatusHandler
	config         *TelegramConfig
}

// NewAdapter creates a new Telegram adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeTelegram,
		Name:        "Telegram",
		Description: "Telegram Bot API integration",
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
				plugin.ContentTypeContact,
			},
			SupportsMedia:           true,
			SupportsLocation:        true,
			SupportsTemplates:       false,
			SupportsInteractive:     true, // Inline keyboards
			SupportsReadReceipts:    false,
			SupportsTypingIndicator: true,
			SupportsReactions:       false,
			SupportsReplies:         true,
			SupportsForwarding:      false,
			MaxMessageLength:        4096,
			MaxMediaSize:            50 * 1024 * 1024, // 50MB for documents
			MaxAttachments:          1,
			SupportedMediaTypes: []string{
				"image/jpeg", "image/png", "image/gif", "image/webp",
				"video/mp4",
				"audio/mpeg", "audio/ogg", "audio/mp4",
				"application/pdf",
				"application/zip",
				"application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeTelegram, info),
		config:      &TelegramConfig{},
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	// Parse configuration
	a.config = &TelegramConfig{
		BotToken: config["bot_token"],
		BotName:  config["bot_name"],
	}

	// Validate required fields
	if a.config.BotToken == "" {
		return fmt.Errorf("bot_token is required")
	}

	return nil
}

// Connect establishes connection to the Telegram Bot API
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create API client
	client, err := NewClient(a.config.BotToken)
	if err != nil {
		return fmt.Errorf("failed to create telegram client: %w", err)
	}

	a.client = client

	// Verify connection by getting bot info
	botInfo, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("failed to verify Telegram connection: %w", err)
	}

	// Store bot name
	if a.config.BotName == "" {
		a.config.BotName = botInfo.UserName
	}

	a.SetConnected(true)
	return nil
}

// Disconnect closes the Telegram connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.client = nil
	a.SetConnected(false)

	return nil
}

// SendMessage sends a message via Telegram
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

	// Parse chat ID from recipient ID
	var chatID int64
	if _, err := fmt.Sscanf(msg.RecipientID, "%d", &chatID); err != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     fmt.Sprintf("invalid chat_id: %s", msg.RecipientID),
			Timestamp: time.Now(),
		}, nil
	}

	// Get reply-to message ID if present
	var replyToMsgID int64
	if replyTo, ok := msg.Metadata["reply_to_id"]; ok && replyTo != "" {
		fmt.Sscanf(replyTo, "%d", &replyToMsgID)
	}

	// Get parse mode
	parseMode := msg.Metadata["parse_mode"]

	var messageID int
	var sendErr error

	switch msg.ContentType {
	case plugin.ContentTypeText:
		// Check if we have an inline keyboard
		keyboard := a.buildKeyboardFromMetadata(msg)
		if keyboard != nil {
			result, err := client.SendMessageWithKeyboard(chatID, msg.Content, parseMode, keyboard, replyToMsgID)
			if err == nil {
				messageID = result.MessageID
			}
			sendErr = err
		} else {
			result, err := client.SendMessage(chatID, msg.Content, parseMode, replyToMsgID)
			if err == nil {
				messageID = result.MessageID
			}
			sendErr = err
		}

	case plugin.ContentTypeImage:
		mediaURL := a.getMediaURL(msg)
		if mediaURL == "" {
			return &plugin.SendResult{
				Success:   false,
				Status:    plugin.MessageStatusFailed,
				Error:     "no media URL provided for image",
				Timestamp: time.Now(),
			}, nil
		}
		result, err := client.SendPhoto(chatID, mediaURL, msg.Content, replyToMsgID)
		if err == nil {
			messageID = result.MessageID
		}
		sendErr = err

	case plugin.ContentTypeVideo:
		mediaURL := a.getMediaURL(msg)
		if mediaURL == "" {
			return &plugin.SendResult{
				Success:   false,
				Status:    plugin.MessageStatusFailed,
				Error:     "no media URL provided for video",
				Timestamp: time.Now(),
			}, nil
		}
		result, err := client.SendVideo(chatID, mediaURL, msg.Content, replyToMsgID)
		if err == nil {
			messageID = result.MessageID
		}
		sendErr = err

	case plugin.ContentTypeAudio:
		mediaURL := a.getMediaURL(msg)
		if mediaURL == "" {
			return &plugin.SendResult{
				Success:   false,
				Status:    plugin.MessageStatusFailed,
				Error:     "no media URL provided for audio",
				Timestamp: time.Now(),
			}, nil
		}
		result, err := client.SendAudio(chatID, mediaURL, msg.Content, replyToMsgID)
		if err == nil {
			messageID = result.MessageID
		}
		sendErr = err

	case plugin.ContentTypeDocument:
		mediaURL := a.getMediaURL(msg)
		if mediaURL == "" {
			return &plugin.SendResult{
				Success:   false,
				Status:    plugin.MessageStatusFailed,
				Error:     "no media URL provided for document",
				Timestamp: time.Now(),
			}, nil
		}
		result, err := client.SendDocument(chatID, mediaURL, msg.Content, replyToMsgID)
		if err == nil {
			messageID = result.MessageID
		}
		sendErr = err

	default:
		// Default to text
		result, err := client.SendMessage(chatID, msg.Content, parseMode, replyToMsgID)
		if err == nil {
			messageID = result.MessageID
		}
		sendErr = err
	}

	if sendErr != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     sendErr.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	externalID := ""
	if messageID > 0 {
		externalID = fmt.Sprintf("%d", messageID)
	}

	return &plugin.SendResult{
		Success:    true,
		ExternalID: externalID,
		Status:     plugin.MessageStatusSent,
		Timestamp:  time.Now(),
	}, nil
}

// getMediaURL extracts media URL from message attachments or metadata
func (a *Adapter) getMediaURL(msg *plugin.OutboundMessage) string {
	if len(msg.Attachments) > 0 {
		return msg.Attachments[0].URL
	}
	return msg.Metadata["media_url"]
}

// buildKeyboardFromMetadata builds an inline keyboard from message metadata
func (a *Adapter) buildKeyboardFromMetadata(msg *plugin.OutboundMessage) *InlineKeyboard {
	// Check for quick replies or buttons in metadata
	quickReplies, hasQuickReplies := msg.Metadata["quick_replies"]
	if !hasQuickReplies || quickReplies == "" {
		return nil
	}

	// Parse quick replies (format: "text1|data1,text2|data2")
	// This is a simple format, could be enhanced with JSON parsing
	var buttons [][]InlineKeyboardButton
	var row []InlineKeyboardButton

	// For now, return nil - can be enhanced to parse JSON format
	// Similar to how Chatwoot handles quick replies

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	if len(buttons) == 0 {
		return nil
	}

	return &InlineKeyboard{Buttons: buttons}
}

// SendTypingIndicator sends a typing indicator
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	var chatID int64
	if _, err := fmt.Sscanf(indicator.RecipientID, "%d", &chatID); err != nil {
		return fmt.Errorf("invalid chat_id: %s", indicator.RecipientID)
	}

	if indicator.IsTyping {
		return client.SendTyping(chatID)
	}

	return nil
}

// SendReadReceipt - Telegram doesn't support read receipts via Bot API
func (a *Adapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	// Telegram Bot API doesn't support marking messages as read
	return nil
}

// UploadMedia - Telegram handles media via URLs, not pre-upload
func (a *Adapter) UploadMedia(ctx context.Context, media *plugin.Media) (*plugin.MediaUpload, error) {
	// Telegram Bot API doesn't require pre-uploading media
	// Media can be sent directly via URL
	return &plugin.MediaUpload{
		Success: false,
		Error:   "telegram doesn't require pre-upload, send media via URL",
	}, nil
}

// DownloadMedia downloads media from Telegram servers
func (a *Adapter) DownloadMedia(ctx context.Context, mediaID string) (*plugin.Media, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("adapter not connected")
	}

	data, filePath, err := client.DownloadFile(mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	return &plugin.Media{
		ID:        mediaID,
		Data:      data,
		Filename:  filePath,
		SizeBytes: int64(len(data)),
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
	return "/api/v1/webhooks/telegram"
}

// ValidateWebhook validates a webhook request
func (a *Adapter) ValidateWebhook(headers map[string]string, body []byte) bool {
	// Telegram webhooks can be validated via secret token in URL
	// or by setting a secret_token in setWebhook
	// For now, we trust the webhook if it comes to our registered URL
	return true
}

// HandleWebhook processes incoming webhook requests
func (a *Adapter) HandleWebhook(ctx context.Context, body []byte) error {
	a.mu.RLock()
	msgHandler := a.messageHandler
	a.mu.RUnlock()

	if msgHandler == nil {
		return fmt.Errorf("no message handler configured")
	}

	// Parse webhook
	update, err := ParseWebhook(body)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Extract incoming message
	incoming := ExtractIncomingMessage(update)
	if incoming == nil {
		// Not a message we handle (e.g., channel post, group message)
		return nil
	}

	// Convert to plugin format
	inbound := a.toInboundMessage(incoming)

	// Call handler
	return msgHandler(ctx, inbound)
}

// toInboundMessage converts a Telegram message to plugin format
func (a *Adapter) toInboundMessage(msg *IncomingMessage) *plugin.InboundMessage {
	inbound := &plugin.InboundMessage{
		ID:         uuid.New().String(),
		ExternalID: fmt.Sprintf("%d", msg.MessageID),
		SenderID:   fmt.Sprintf("%d", msg.ChatID),
		SenderName: msg.FromFirstName,
		Content:    msg.Text,
		Timestamp:  msg.Timestamp,
		Metadata: map[string]string{
			"from_user_id": fmt.Sprintf("%d", msg.FromUserID),
			"username":     msg.FromUsername,
			"first_name":   msg.FromFirstName,
			"last_name":    msg.FromLastName,
			"chat_id":      fmt.Sprintf("%d", msg.ChatID),
		},
	}

	// Add full name if available
	if msg.FromLastName != "" {
		inbound.SenderName = msg.FromFirstName + " " + msg.FromLastName
	}

	// Set content type based on message type
	switch msg.MessageType {
	case MessageTypeText:
		inbound.ContentType = plugin.ContentTypeText
		inbound.Content = msg.Text
	case MessageTypePhoto:
		inbound.ContentType = plugin.ContentTypeImage
		inbound.Content = msg.Caption
		if msg.MediaFileID != "" {
			inbound.Attachments = []*plugin.Attachment{{
				Type: "image",
				Metadata: map[string]string{
					"file_id": msg.MediaFileID,
				},
			}}
		}
	case MessageTypeVideo:
		inbound.ContentType = plugin.ContentTypeVideo
		inbound.Content = msg.Caption
		if msg.MediaFileID != "" {
			inbound.Attachments = []*plugin.Attachment{{
				Type:     "video",
				MimeType: msg.MediaMimeType,
				Metadata: map[string]string{
					"file_id": msg.MediaFileID,
				},
			}}
		}
	case MessageTypeAudio:
		inbound.ContentType = plugin.ContentTypeAudio
		inbound.Content = msg.Caption
		if msg.MediaFileID != "" {
			inbound.Attachments = []*plugin.Attachment{{
				Type:      "audio",
				MimeType:  msg.MediaMimeType,
				Filename:  msg.MediaFileName,
				SizeBytes: msg.MediaFileSize,
				Metadata: map[string]string{
					"file_id": msg.MediaFileID,
				},
			}}
		}
	case MessageTypeVoice:
		inbound.ContentType = plugin.ContentTypeAudio
		if msg.MediaFileID != "" {
			inbound.Attachments = []*plugin.Attachment{{
				Type:     "voice",
				MimeType: msg.MediaMimeType,
				Metadata: map[string]string{
					"file_id": msg.MediaFileID,
				},
			}}
		}
	case MessageTypeDocument:
		inbound.ContentType = plugin.ContentTypeDocument
		inbound.Content = msg.Caption
		if msg.MediaFileID != "" {
			inbound.Attachments = []*plugin.Attachment{{
				Type:      "document",
				MimeType:  msg.MediaMimeType,
				Filename:  msg.MediaFileName,
				SizeBytes: msg.MediaFileSize,
				Metadata: map[string]string{
					"file_id": msg.MediaFileID,
				},
			}}
		}
	case MessageTypeLocation:
		inbound.ContentType = plugin.ContentTypeLocation
		if msg.Location != nil {
			inbound.Content = fmt.Sprintf("%f,%f", msg.Location.Latitude, msg.Location.Longitude)
			inbound.Metadata["latitude"] = fmt.Sprintf("%f", msg.Location.Latitude)
			inbound.Metadata["longitude"] = fmt.Sprintf("%f", msg.Location.Longitude)
		}
	case MessageTypeContact:
		inbound.ContentType = plugin.ContentTypeContact
		if msg.Contact != nil {
			inbound.Content = msg.Contact.PhoneNumber
			inbound.Metadata["contact_phone"] = msg.Contact.PhoneNumber
			inbound.Metadata["contact_first_name"] = msg.Contact.FirstName
			inbound.Metadata["contact_last_name"] = msg.Contact.LastName
		}
	default:
		inbound.ContentType = plugin.ContentTypeText
	}

	// Handle reply
	if msg.ReplyToMsgID != nil {
		inbound.Metadata["reply_to_id"] = fmt.Sprintf("%d", *msg.ReplyToMsgID)
	}

	// Mark if edited
	if msg.IsEdited {
		inbound.Metadata["is_edited"] = "true"
	}

	return inbound
}

// GetClient returns the Telegram client
func (a *Adapter) GetClient() *Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// GetConfig returns the adapter configuration
func (a *Adapter) GetConfig() *TelegramConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// SetupWebhook configures the Telegram webhook
func (a *Adapter) SetupWebhook(webhookURL string) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	return client.SetWebhook(webhookURL)
}

// DeleteWebhook removes the Telegram webhook
func (a *Adapter) DeleteWebhook() error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	return client.DeleteWebhook()
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
		if a.config.BotName != "" {
			status.Metadata["bot_name"] = a.config.BotName
		}
	} else {
		status.Status = "disconnected"
	}

	return status
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
