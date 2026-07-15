package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

	// Native calling (VoIP) — isolated from the messaging event loop.
	callGateway   *CallGateway
	callHandler   CallHandler
	callAudioSink PeerAudioSink
	callRecorder  *CallRecorder
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
			SupportsInteractive:     true,  // Native-flow buttons/lists (see interactive.go)
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: true,
			SupportsReactions:       true,
			SupportsReplies:         true,
			SupportsForwarding:      true, // Text forward via metadata is_forwarded (edit.go)
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
		ChannelID:     config["channel_id"],
		DatabasePath:  config["database_path"],
		DeviceName:    config["device_name"],
		PlatformType:  config["platform_type"],
		LogLevel:      config["log_level"],
		RecordCalls:   config["record_calls"] == "true",
		RecordingsDir: config["recordings_dir"],
	}

	if a.config.LogLevel == "" {
		a.config.LogLevel = "WARN"
	}

	// Config-driven call recording; EnableCallRecording can still toggle it at
	// runtime.
	if a.config.RecordCalls {
		a.callRecorder = NewCallRecorder(a.config.RecordingsDir, 16000)
	}

	return nil
}

// Connect establishes connection to WhatsApp
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil {
		if a.client.IsConnected() {
			return nil
		}
		// Clean up the previous (stale) client before creating a new one to
		// avoid leaking its goroutine, event loop and SQLite handle.
		a.teardownLocked()
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

	// Bring up the native calling gateway on the live whatsmeow client. The
	// handler closure reads the adapter's current call handler so SetCallHandler
	// works whether it is called before or after Connect.
	if raw := client.GetRawClient(); raw != nil {
		g := NewCallGateway(raw, nil, func(cctx context.Context, evt CallEvent) {
			a.mu.RLock()
			h := a.callHandler
			a.mu.RUnlock()
			if h != nil {
				h(cctx, evt)
			}
		})
		g.AudioSink = a.callAudioSink
		g.Recorder = a.callRecorder
		a.callGateway = g
		raw.AddEventHandler(g.HandleWhatsmeowEvent)
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

	return a.teardownLocked()
}

// teardownLocked stops the event loop and closes the underlying client.
// It must be called with a.mu held. It is safe to call multiple times: the
// stop channel is niled after being closed (avoiding a double-close panic) and
// the client is zeroed even when Close() returns an error.
func (a *Adapter) teardownLocked() error {
	if a.callGateway != nil {
		a.callGateway.Shutdown(context.Background())
		a.callGateway = nil
	}

	if a.stopCh != nil {
		close(a.stopCh)
		<-a.eventLoopDone
		a.stopCh = nil
		a.eventLoopDone = nil
	}

	var err error
	if a.client != nil {
		a.client.Disconnect()
		err = a.client.Close()
	}

	a.client = nil
	a.SetConnected(false)
	return err
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
		switch {
		case msg.Metadata["is_forwarded"] == "true":
			// Forward a text body tagged with the "Forwarded" label.
			resp, err = client.ForwardText(ctx, msg.RecipientID, msg.Content)
		case msg.Metadata["reply_to_id"] != "":
			resp, err = client.SendTextMessageWithReply(ctx, msg.RecipientID, msg.Content, msg.Metadata["reply_to_id"], msg.Metadata["quoted_text"])
		default:
			resp, err = client.SendTextMessage(ctx, msg.RecipientID, msg.Content)
		}

	case plugin.ContentTypeImage:
		if len(msg.Attachments) > 0 {
			att := msg.Attachments[0]
			// Media data should be provided via metadata["media_data"] or fetched from URL.
			// Declare mediaData separately so the send error propagates to the outer err
			// (using := here would shadow err and leave resp nil -> panic below).
			var mediaData []byte
			mediaData, err = getMediaData(ctx, att)
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
			var mediaData []byte
			mediaData, err = getMediaData(ctx, att)
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
			var mediaData []byte
			mediaData, err = getMediaData(ctx, att)
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
			var mediaData []byte
			mediaData, err = getMediaData(ctx, att)
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

	case plugin.ContentTypeInteractive:
		// Content carries the interactive payload as JSON (see InteractivePayload).
		resp, err = sendInteractive(ctx, client, msg.RecipientID, msg.Content)

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

	// Defensive guard: never dereference a nil response even if a send path
	// somehow returned (nil, nil).
	if resp == nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     "send returned no response",
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

// SetCallHandler registers the handler for native call lifecycle events. It may
// be called before or after Connect.
func (a *Adapter) SetCallHandler(handler CallHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.callHandler = handler
}

// SetCallAudioSink registers a sink for the remote peer's decoded PCM during
// calls (recording / speech-to-text). Applies to the active gateway if any.
func (a *Adapter) SetCallAudioSink(sink PeerAudioSink) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.callAudioSink = sink
	if a.callGateway != nil {
		a.callGateway.AudioSink = sink
	}
}

// EnableCallRecording turns on per-call WAV recording of the peer's audio,
// written under dir (default "media/recordings"). The ended CallEvent carries
// the resulting RecordingPath. Recording never blocks the live call — the audio
// callback only buffers in memory; the file is written at call teardown.
func (a *Adapter) EnableCallRecording(dir string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	rec := NewCallRecorder(dir, 16000)
	a.callRecorder = rec
	if a.callGateway != nil {
		a.callGateway.Recorder = rec
	}
}

// PlaceCall starts an outgoing native WhatsApp call and returns its call id.
func (a *Adapter) PlaceCall(ctx context.Context, to string, isVideo bool) (string, error) {
	a.mu.RLock()
	g := a.callGateway
	a.mu.RUnlock()
	if g == nil {
		return "", ErrClientNotReady
	}
	return g.PlaceCall(ctx, to, isVideo)
}

// AcceptCall answers a ringing inbound call.
func (a *Adapter) AcceptCall(ctx context.Context, callID string) error {
	a.mu.RLock()
	g := a.callGateway
	a.mu.RUnlock()
	if g == nil {
		return ErrClientNotReady
	}
	return g.AcceptCall(ctx, callID)
}

// RejectCall declines a ringing inbound call.
func (a *Adapter) RejectCall(ctx context.Context, callID string) error {
	a.mu.RLock()
	g := a.callGateway
	a.mu.RUnlock()
	if g == nil {
		return ErrClientNotReady
	}
	return g.RejectCall(ctx, callID)
}

// EndCall hangs up an in-progress call.
func (a *Adapter) EndCall(ctx context.Context, callID string) error {
	a.mu.RLock()
	g := a.callGateway
	a.mu.RUnlock()
	if g == nil {
		return ErrClientNotReady
	}
	return g.EndCall(ctx, callID)
}

// EditMessage edits the text of a message previously sent on this channel.
func (a *Adapter) EditMessage(ctx context.Context, chat, messageID, newText string) (*SendMessageResponse, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	return client.EditMessage(ctx, chat, messageID, newText)
}

// RevokeMessage deletes a message for everyone. Pass an empty sender to revoke
// your own message; in groups an admin passes the original sender's JID.
func (a *Adapter) RevokeMessage(ctx context.Context, chat, sender, messageID string) (*SendMessageResponse, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	return client.RevokeMessage(ctx, chat, sender, messageID)
}

// SendTypingIndicator sends a typing indicator
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	jid, err := resolveRecipientJID(indicator.RecipientID)
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
	chatJID, err := resolveRecipientJID(receipt.RecipientID)
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

	// Lifecycle ctx for handlers: cancelled when the adapter stops so
	// in-flight processing does not outlive the event loop.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-stopCh
		cancel()
	}()

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
					// Resolve @lid senders to their phone-number JID so the
					// conversation keys on a stable phone identity.
					if isLID(v.SenderJID) {
						if pn := client.ResolvePN(ctx, v.SenderJID); pn.User != "" && !isLID(pn) {
							v.SenderPN = pn
						}
					}
					inbound := convertToInboundMessage(v)
					// Eagerly download+decrypt inbound media so the encrypted
					// CDN blob becomes usable bytes for the application.
					enrichInboundMedia(ctx, v, inbound, client)
					if err := msgHandler(ctx, inbound); err != nil {
						// Log error but continue
					}
				}

			case *Receipt:
				if statusHandler != nil {
					status := convertToStatusCallback(v)
					if err := statusHandler(ctx, status); err != nil {
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
					if err := connHandler(ctx, connected, string(v.State)); err != nil {
						// Log error but continue
					}
				}

			case LogoutEvent:
				a.mu.Lock()
				a.SetConnected(false)
				a.mu.Unlock()

				// Notify connection handler about logout
				if connHandler != nil {
					if err := connHandler(ctx, false, v.Reason); err != nil {
						// Log error but continue
					}
				}
			}
		}
	}
}

// convertToInboundMessage converts an IncomingMessage to plugin.InboundMessage
func convertToInboundMessage(msg *IncomingMessage) *plugin.InboundMessage {
	// Prefer the resolved phone number as the stable sender identity; keep the
	// raw LID in metadata when a resolution happened.
	senderID := msg.SenderJID.User
	if msg.SenderPN.User != "" {
		senderID = msg.SenderPN.User
	}

	inbound := &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  msg.ExternalID,
		SenderID:    senderID,
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

	if msg.SenderPN.User != "" {
		inbound.Metadata["sender_pn"] = msg.SenderPN.String()
		if isLID(msg.SenderJID) {
			inbound.Metadata["lid"] = msg.SenderJID.String()
		}
	}

	// Flag edits so downstream can update the original message instead of
	// inserting a duplicate.
	if msg.IsEdit {
		inbound.Metadata["edited"] = "true"
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
	case "interactive_reply":
		// A tapped native-flow button/list or template button. Surface it as
		// text (the display label) plus the selected option id in metadata so
		// bots/flows can branch on the id.
		inbound.ContentType = plugin.ContentTypeText
		inbound.Metadata["interactive_reply"] = "true"
		if msg.SelectedID != "" {
			inbound.Metadata["selected_id"] = msg.SelectedID
		}
	}

	// Convert attachments
	for _, att := range msg.Attachments {
		pa := &plugin.Attachment{
			Type:      att.Type,
			URL:       att.URL,
			MimeType:  att.MimeType,
			SizeBytes: int64(att.FileSize),
			Filename:  att.Filename,
		}

		// Location messages carry no downloadable media; surface their
		// coordinates (and name/address) via metadata so downstream
		// normalizers can reconstruct the location payload.
		if att.Type == "location" {
			lat := strconv.FormatFloat(att.Latitude, 'f', -1, 64)
			lon := strconv.FormatFloat(att.Longitude, 'f', -1, 64)
			inbound.Metadata["latitude"] = lat
			inbound.Metadata["longitude"] = lon
			if att.Filename != "" {
				inbound.Metadata["name"] = att.Filename
			}
			if att.Caption != "" {
				inbound.Metadata["address"] = att.Caption
			}
			pa.Metadata = map[string]string{
				"latitude":  lat,
				"longitude": lon,
			}
		}

		inbound.Attachments = append(inbound.Attachments, pa)
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

// mediaDownloader downloads and decrypts an inbound WhatsApp media message.
// *Client satisfies it via its DownloadMedia method; tests fake it to exercise
// the enrichment/mapping logic without a live whatsmeow connection.
type mediaDownloader interface {
	DownloadMedia(ctx context.Context, msg any) ([]byte, error)
}

// enrichInboundMedia eagerly downloads+decrypts each inbound media attachment
// and attaches the resulting bytes to the corresponding plugin attachment.
//
// whatsmeow media messages reference an encrypted blob on WhatsApp's CDN; the
// plain URL is ciphertext that no downstream HTTP consumer can decrypt. We
// therefore download+decrypt here and carry the plaintext as base64 in the
// attachment metadata (symmetric with how outbound media reads
// metadata["data"]), dropping the useless encrypted URL. src and inbound share
// a 1:1 attachment ordering because both are built from msg.Attachments.
func enrichInboundMedia(ctx context.Context, src *IncomingMessage, inbound *plugin.InboundMessage, downloader mediaDownloader) {
	if src == nil || inbound == nil || downloader == nil {
		return
	}

	for i := range src.Attachments {
		ref := src.Attachments[i].download
		if ref == nil {
			continue
		}
		if i >= len(inbound.Attachments) {
			break
		}

		pa := inbound.Attachments[i]
		if pa.Metadata == nil {
			pa.Metadata = make(map[string]string)
		}

		data, err := downloader.DownloadMedia(ctx, ref)
		if err != nil {
			pa.Metadata["download_error"] = err.Error()
			continue
		}

		pa.Metadata["data"] = base64.StdEncoding.EncodeToString(data)
		pa.Metadata["data_encoding"] = "base64"
		pa.SizeBytes = int64(len(data))
		// The encrypted CDN URL is unusable once we hold the plaintext; clear it
		// so no downstream consumer stores the ciphertext blob.
		pa.URL = ""
	}
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
func getMediaData(ctx context.Context, att *plugin.Attachment) ([]byte, error) {
	// Check if data is provided in metadata as base64
	if att.Metadata != nil {
		if b64Data, ok := att.Metadata["data"]; ok && b64Data != "" {
			return decodeBase64(b64Data)
		}
	}

	// Fetch from URL if provided
	if att.URL != "" {
		return fetchMediaFromURL(ctx, att.URL)
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

// maxMediaFetchBytes caps how much data fetchMediaFromURL will read from a
// remote URL to protect against memory-exhaustion via oversized responses.
const maxMediaFetchBytes = 25 * 1024 * 1024 // 25 MB

// fetchMediaFromURL fetches media content from a URL with SSRF protections:
// it only allows http/https, rejects hosts that resolve to private,
// loopback, link-local or otherwise non-public addresses, honours ctx, and
// bounds the amount of data read.
func fetchMediaFromURL(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid media URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported media URL scheme %q: only http/https allowed", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("media URL is missing a host")
	}

	// Resolve the host and reject any address that is not publicly routable.
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve media host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("media host %q resolves to a disallowed address %s", host, ip)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create media request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch media: status %d", resp.StatusCode)
	}

	// Read at most maxMediaFetchBytes (+1 to detect overflow).
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaFetchBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read media: %w", err)
	}
	if len(data) > maxMediaFetchBytes {
		return nil, fmt.Errorf("media exceeds maximum allowed size of %d bytes", maxMediaFetchBytes)
	}

	return data, nil
}

// isBlockedIP reports whether an IP must not be fetched from, to prevent SSRF
// against internal services and cloud metadata endpoints (e.g. 169.254.169.254).
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
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
