package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/pkg/plugin"
	"go.mau.fi/whatsmeow/types"
)

// ConnectionHandler is called when connection state changes
type ConnectionHandler func(ctx context.Context, connected bool, reason string) error

// Adapter implements the WhatsApp (unofficial) channel adapter using whatsmeow
type Adapter struct {
	*plugin.BaseAdapter

	mu                sync.RWMutex
	client            *Client
	messageHandler    plugin.MessageHandler
	statusHandler     plugin.StatusHandler
	connectionHandler ConnectionHandler
	config            *Config
	stopCh            chan struct{}
	eventLoopDone     chan struct{}
}

// NewAdapter creates a new WhatsApp unofficial adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeWhatsApp,
		Name:        "WhatsApp (Unofficial)",
		Description: "WhatsApp via whatsmeow multi-device protocol",
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
			SupportsTemplates:       false, // Not available in unofficial API
			SupportsInteractive:     false, // Limited support
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: true,
			SupportsReactions:       true,
			SupportsReplies:         true,
			SupportsForwarding:      false, // Complex to implement
			MaxMessageLength:        65536,
			MaxMediaSize:            100 * 1024 * 1024, // 100MB
			MaxAttachments:          1,
			SupportedMediaTypes: []string{
				"image/jpeg", "image/png", "image/webp",
				"video/mp4", "video/3gpp",
				"audio/aac", "audio/mp4", "audio/mpeg", "audio/amr", "audio/ogg",
				"application/pdf", "application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeWhatsApp, info),
		config:      &Config{},
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	a.config = &Config{
		ChannelID:    config["channel_id"],
		DatabasePath: config["database_path"],
		DeviceName:   config["device_name"],
		PlatformType: config["platform_type"],
		LogLevel:     config["log_level"],
	}

	if a.config.LogLevel == "" {
		a.config.LogLevel = "WARN"
	}

	return nil
}

// Connect establishes connection to WhatsApp
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil && a.client.IsConnected() {
		return nil
	}

	// Create client
	client, err := NewClient(a.config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	a.client = client
	a.stopCh = make(chan struct{})
	a.eventLoopDone = make(chan struct{})

	// Connect
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Start event loop
	go a.eventLoop()

	a.SetConnected(true)
	return nil
}

// Disconnect closes the WhatsApp connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stopCh != nil {
		close(a.stopCh)
		<-a.eventLoopDone
	}

	if a.client != nil {
		a.client.Disconnect()
		if err := a.client.Close(); err != nil {
			return err
		}
	}

	a.client = nil
	a.SetConnected(false)
	return nil
}

// Login initiates QR code login and returns a channel for QR events
func (a *Adapter) Login(ctx context.Context) (<-chan QRCodeEvent, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrClientNotReady
	}

	return client.Login(ctx)
}

// LoginWithPairCode initiates phone number pairing
func (a *Adapter) LoginWithPairCode(ctx context.Context, phoneNumber string) (*PairCodeResponse, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrClientNotReady
	}

	return client.LoginWithPairCode(ctx, phoneNumber)
}

// Logout logs out from WhatsApp
func (a *Adapter) Logout(ctx context.Context) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return nil
	}

	return client.Logout(ctx)
}

// IsLoggedIn returns true if the client is logged in
func (a *Adapter) IsLoggedIn() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.client != nil && a.client.IsLoggedIn()
}

// GetDeviceInfo returns information about the connected device
func (a *Adapter) GetDeviceInfo() *DeviceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.client == nil {
		return &DeviceInfo{State: DeviceStateDisconnected}
	}

	return a.client.GetDeviceInfo()
}

// SendMessage sends a message via WhatsApp
func (a *Adapter) SendMessage(ctx context.Context, msg *plugin.OutboundMessage) (*plugin.SendResult, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     "adapter not connected",
			Timestamp: time.Now(),
		}, nil
	}

	var resp *SendMessageResponse
	var err error

	switch msg.ContentType {
	case plugin.ContentTypeText:
		// Check for reply
		if replyTo, ok := msg.Metadata["reply_to_id"]; ok && replyTo != "" {
			quotedText := msg.Metadata["quoted_text"]
			resp, err = client.SendTextMessageWithReply(ctx, msg.RecipientID, msg.Content, replyTo, quotedText)
		} else {
			resp, err = client.SendTextMessage(ctx, msg.RecipientID, msg.Content)
		}

	case plugin.ContentTypeImage:
		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			// Media data should be provided via metadata["media_data"] or fetched from URL
			mediaData, err := getMediaData(att)
			if err != nil {
				return &plugin.SendResult{
					Success: false,
					Status:  plugin.MessageStatusFailed,
					Error:   err.Error(),
				}, nil
			}
			resp, err = client.SendImageMessage(ctx, msg.RecipientID, mediaData, att.MimeType, msg.Content)
		} else {
			return &plugin.SendResult{
				Success: false,
				Status:  plugin.MessageStatusFailed,
				Error:   "no image attachment provided",
			}, nil
		}

	case plugin.ContentTypeVideo:
		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			mediaData, err := getMediaData(att)
			if err != nil {
				return &plugin.SendResult{
					Success: false,
					Status:  plugin.MessageStatusFailed,
					Error:   err.Error(),
				}, nil
			}
			resp, err = client.SendVideoMessage(ctx, msg.RecipientID, mediaData, att.MimeType, msg.Content)
		} else {
			return &plugin.SendResult{
				Success: false,
				Status:  plugin.MessageStatusFailed,
				Error:   "no video attachment provided",
			}, nil
		}

	case plugin.ContentTypeAudio:
		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			mediaData, err := getMediaData(att)
			if err != nil {
				return &plugin.SendResult{
					Success: false,
					Status:  plugin.MessageStatusFailed,
					Error:   err.Error(),
				}, nil
			}
			ptt := msg.Metadata["ptt"] == "true"
			resp, err = client.SendAudioMessage(ctx, msg.RecipientID, mediaData, att.MimeType, ptt)
		} else {
			return &plugin.SendResult{
				Success: false,
				Status:  plugin.MessageStatusFailed,
				Error:   "no audio attachment provided",
			}, nil
		}

	case plugin.ContentTypeDocument:
		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			mediaData, err := getMediaData(att)
			if err != nil {
				return &plugin.SendResult{
					Success: false,
					Status:  plugin.MessageStatusFailed,
					Error:   err.Error(),
				}, nil
			}
			resp, err = client.SendDocumentMessage(ctx, msg.RecipientID, mediaData, att.MimeType, att.Filename, msg.Content)
		} else {
			return &plugin.SendResult{
				Success: false,
				Status:  plugin.MessageStatusFailed,
				Error:   "no document attachment provided",
			}, nil
		}

	case plugin.ContentTypeLocation:
		lat, lon := 0.0, 0.0
		fmt.Sscanf(msg.Metadata["latitude"], "%f", &lat)
		fmt.Sscanf(msg.Metadata["longitude"], "%f", &lon)
		name := msg.Metadata["name"]
		address := msg.Metadata["address"]
		resp, err = client.SendLocationMessage(ctx, msg.RecipientID, lat, lon, name, address)

	default:
		return &plugin.SendResult{
			Success: false,
			Status:  plugin.MessageStatusFailed,
			Error:   fmt.Sprintf("unsupported content type: %s", msg.ContentType),
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
		Timestamp:  resp.Timestamp,
	}, nil
}

// SendTypingIndicator sends a typing indicator
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	jid, err := types.ParseJID(indicator.RecipientID)
	if err != nil {
		jid = types.NewJID(indicator.RecipientID, types.DefaultUserServer)
	}

	state := ChatPresencePaused
	if indicator.IsTyping {
		state = ChatPresenceComposing
	}

	return client.SendChatPresence(ctx, jid, state)
}

// SendReadReceipt marks messages as read
func (a *Adapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	// RecipientID is the chat/sender JID
	chatJID, err := types.ParseJID(receipt.RecipientID)
	if err != nil {
		chatJID = types.NewJID(receipt.RecipientID, types.DefaultUserServer)
	}

	// For direct chats, sender is same as chat
	return client.MarkAsRead(ctx, []string{receipt.MessageID}, chatJID, chatJID)
}

// UploadMedia uploads media (not applicable for whatsmeow - media is uploaded inline)
func (a *Adapter) UploadMedia(ctx context.Context, media *plugin.Media) (*plugin.MediaUpload, error) {
	// In whatsmeow, media is uploaded when sending the message
	return &plugin.MediaUpload{
		Success: true,
		MediaID: uuid.New().String(), // Return a temporary ID
	}, nil
}

// DownloadMedia downloads media from a message
func (a *Adapter) DownloadMedia(ctx context.Context, mediaID string) (*plugin.Media, error) {
	// This would require storing message references
	return nil, fmt.Errorf("download by ID not supported, use GetRawClient().Download() with message")
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

// SetConnectionHandler sets the handler for connection state changes
func (a *Adapter) SetConnectionHandler(handler ConnectionHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connectionHandler = handler
}

// GetConnectionStatus returns detailed connection status
func (a *Adapter) GetConnectionStatus() *plugin.ConnectionStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := &plugin.ConnectionStatus{
		Connected: a.IsConnected(),
		Metadata:  make(map[string]string),
	}

	if a.client != nil {
		info := a.client.GetDeviceInfo()
		status.Metadata["state"] = string(info.State)
		status.Metadata["jid"] = info.JID
		status.Metadata["phone"] = info.PhoneNumber

		if a.IsConnected() {
			status.Status = "connected"
		} else if info.State == DeviceStateQRPending {
			status.Status = "qr_pending"
		} else {
			status.Status = "disconnected"
		}
	} else {
		status.Status = "disconnected"
	}

	return status
}

// GetClient returns the underlying WhatsApp client
func (a *Adapter) GetClient() *Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// eventLoop processes events from the WhatsApp client
func (a *Adapter) eventLoop() {
	defer close(a.eventLoopDone)

	a.mu.RLock()
	client := a.client
	stopCh := a.stopCh
	a.mu.RUnlock()

	if client == nil {
		return
	}

	eventCh := client.GetEventChannel()

	for {
		select {
		case <-stopCh:
			return

		case evt, ok := <-eventCh:
			if !ok {
				return
			}

			a.mu.RLock()
			msgHandler := a.messageHandler
			statusHandler := a.statusHandler
			connHandler := a.connectionHandler
			a.mu.RUnlock()

			switch v := evt.(type) {
			case *IncomingMessage:
				if msgHandler != nil && !v.IsFromMe {
					inbound := convertToInboundMessage(v)
					if err := msgHandler(context.Background(), inbound); err != nil {
						// Log error but continue
					}
				}

			case *Receipt:
				if statusHandler != nil {
					status := convertToStatusCallback(v)
					if err := statusHandler(context.Background(), status); err != nil {
						// Log error but continue
					}
				}

			case ConnectionEvent:
				connected := v.State == DeviceStateConnected
				a.mu.Lock()
				a.SetConnected(connected)
				a.mu.Unlock()

				// Notify connection handler
				if connHandler != nil {
					if err := connHandler(context.Background(), connected, string(v.State)); err != nil {
						// Log error but continue
					}
				}

			case LogoutEvent:
				a.mu.Lock()
				a.SetConnected(false)
				a.mu.Unlock()

				// Notify connection handler about logout
				if connHandler != nil {
					if err := connHandler(context.Background(), false, v.Reason); err != nil {
						// Log error but continue
					}
				}
			}
		}
	}
}

// convertToInboundMessage converts an IncomingMessage to plugin.InboundMessage
func convertToInboundMessage(msg *IncomingMessage) *plugin.InboundMessage {
	inbound := &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  msg.ExternalID,
		SenderID:    msg.SenderJID.User,
		SenderName:  msg.SenderName,
		Content:     msg.Text,
		ContentType: plugin.ContentTypeText,
		Timestamp:   msg.Timestamp,
		Metadata: map[string]string{
			"chat_jid":   msg.ChatJID.String(),
			"sender_jid": msg.SenderJID.String(),
			"is_group":   fmt.Sprintf("%t", msg.IsGroup),
			"msg_type":   msg.MessageType,
		},
	}

	// Set content type based on message type
	switch msg.MessageType {
	case "image":
		inbound.ContentType = plugin.ContentTypeImage
	case "video":
		inbound.ContentType = plugin.ContentTypeVideo
	case "audio", "ptt":
		inbound.ContentType = plugin.ContentTypeAudio
	case "document":
		inbound.ContentType = plugin.ContentTypeDocument
	case "location":
		inbound.ContentType = plugin.ContentTypeLocation
	case "contact":
		inbound.ContentType = plugin.ContentTypeContact
	case "sticker":
		inbound.ContentType = plugin.ContentTypeImage
		inbound.Metadata["is_sticker"] = "true"
	}

	// Convert attachments
	for _, att := range msg.Attachments {
		inbound.Attachments = append(inbound.Attachments, &plugin.Attachment{
			Type:      att.Type,
			URL:       att.URL,
			MimeType:  att.MimeType,
			SizeBytes: int64(att.FileSize),
			Filename:  att.Filename,
		})
	}

	// Handle reply info
	if msg.ReplyTo != nil {
		inbound.Metadata["reply_to_id"] = msg.ReplyTo.MessageID
		inbound.Metadata["quoted_text"] = msg.ReplyTo.Text
	}

	// Handle reaction
	if msg.Reaction != nil {
		inbound.Metadata["reaction"] = msg.Reaction.Emoji
		inbound.Metadata["reaction_message_id"] = msg.Reaction.MessageID
	}

	return inbound
}

// convertToStatusCallback converts a Receipt to plugin.StatusCallback
func convertToStatusCallback(receipt *Receipt) *plugin.StatusCallback {
	var status plugin.MessageStatus

	switch receipt.Type {
	case ReceiptTypeDelivered:
		status = plugin.MessageStatusDelivered
	case ReceiptTypeRead:
		status = plugin.MessageStatusRead
	default:
		status = plugin.MessageStatusDelivered
	}

	// Return callback for first message ID
	messageID := ""
	if len(receipt.MessageIDs) > 0 {
		messageID = receipt.MessageIDs[0]
	}

	return &plugin.StatusCallback{
		MessageID:  messageID,
		ExternalID: messageID,
		Status:     status,
		Timestamp:  receipt.Timestamp,
	}
}

// getMediaData extracts media data from an attachment
// It checks for base64 data in metadata or fetches from URL
func getMediaData(att *plugin.Attachment) ([]byte, error) {
	// Check if data is provided in metadata as base64
	if att.Metadata != nil {
		if b64Data, ok := att.Metadata["data"]; ok && b64Data != "" {
			return decodeBase64(b64Data)
		}
	}

	// Fetch from URL if provided
	if att.URL != "" {
		return fetchMediaFromURL(att.URL)
	}

	return nil, fmt.Errorf("no media data provided: attach via URL or metadata[data] as base64")
}

// decodeBase64 decodes a base64 string
func decodeBase64(data string) ([]byte, error) {
	// Remove data URL prefix if present
	if idx := len("data:"); len(data) > idx {
		if colonIdx := indexOf(data, ";base64,"); colonIdx > 0 {
			data = data[colonIdx+8:]
		}
	}

	// Standard library base64 decode
	decoded := make([]byte, len(data))
	n, err := decodeBase64Std(decoded, []byte(data))
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}

// decodeBase64Std is a wrapper for encoding/base64
func decodeBase64Std(dst, src []byte) (int, error) {
	return base64Decode(dst, src)
}

// indexOf finds the index of substr in s
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// fetchMediaFromURL fetches media content from a URL
func fetchMediaFromURL(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch media: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// httpClient is a shared HTTP client for fetching media
var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

// base64Decode decodes base64
func base64Decode(dst, src []byte) (int, error) {
	return base64.StdEncoding.Decode(dst, src)
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
