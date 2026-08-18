package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo), registered as "sqlite"
)

// Client wraps the whatsmeow client
type Client struct {
	mu      sync.RWMutex
	client  *whatsmeow.Client
	store   *sqlstore.Container
	device  *store.Device
	config  *Config
	state   DeviceState
	logger  waLog.Logger
	eventCh chan any
	qrCh    chan QRCodeEvent
	stopCh  chan struct{}

	// lidPNCache memoizes LID→PN resolutions with a TTL so we don't hit the
	// device store on every inbound message from a hidden-user (@lid) sender.
	lidPNMu    sync.Mutex
	lidPNCache map[types.JID]lidCacheEntry
}

// NewClient creates a new WhatsApp client
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config.SetDefaults()

	// Ensure storage directory exists
	dir := filepath.Dir(config.DatabasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create logger
	logger := waLog.Stdout("WhatsApp", config.LogLevel, true)

	// Open the session store with the pure-Go modernc SQLite driver (no cgo) so
	// the binary builds and deploys statically. whatsmeow still generates SQLite
	// SQL (dialect "sqlite3"); only the underlying driver changes. WAL + a busy
	// timeout keep the single-file store resilient under concurrent access.
	dbURI := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)",
		config.DatabasePath)
	db, err := sql.Open("sqlite", dbURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	container := sqlstore.NewWithDB(db, "sqlite3", logger)
	if err := container.Upgrade(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to upgrade database schema: %w", err)
	}

	return &Client{
		store:   container,
		config:  config,
		state:   DeviceStateDisconnected,
		logger:  logger,
		eventCh: make(chan any, 1024),
		qrCh:    make(chan QRCodeEvent, 10),
		stopCh:  make(chan struct{}),
	}, nil
}

// Connect connects to WhatsApp
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil && c.client.IsConnected() {
		return ErrAlreadyLoggedIn
	}

	// Get or create device
	device, err := c.getOrCreateDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}
	c.device = device

	// Create client
	client := whatsmeow.NewClient(device, c.logger)
	client.EnableAutoReconnect = c.config.AutoReconnect
	client.AutoTrustIdentity = c.config.AutoTrustIdentity

	// proxy_*: route the connection through a SOCKS5 proxy when configured. Set
	// before any connect so it applies to the whole session.
	if addr := c.proxyAddress(); addr != "" {
		if err := client.SetProxyAddress(addr); err != nil {
			c.logger.Warnf("invalid proxy address: %v", err)
		}
	}

	c.client = client

	// Register event handler
	client.AddEventHandler(c.handleEvent)

	// Check if already logged in
	if client.Store.ID == nil {
		c.state = DeviceStateQRPending
		return nil // Need to call Login() to get QR code
	}

	// Already have stored credentials, try to connect
	c.state = DeviceStateConnecting
	if err := client.Connect(); err != nil {
		c.state = DeviceStateDisconnected
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.state = DeviceStateConnected
	return nil
}

// Login initiates the QR code login process
func (c *Client) Login(ctx context.Context) (<-chan QRCodeEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, ErrClientNotReady
	}

	if c.client.Store.ID != nil && c.client.IsConnected() {
		return nil, ErrAlreadyLoggedIn
	}

	// Ensure disconnected before login
	if c.client.IsConnected() {
		c.client.Disconnect()
	}

	// Recreate the QR channel for each login attempt. The forwarding goroutine
	// below closes c.qrCh when it finishes; reusing the same channel across a
	// re-login (e.g. after a QR expiry) would close an already-closed channel
	// and panic.
	c.qrCh = make(chan QRCodeEvent, 10)

	// Get QR channel from whatsmeow
	qrChan, err := c.client.GetQRChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR channel: %w", err)
	}

	c.state = DeviceStateQRPending

	// Start goroutine to forward QR events
	go func() {
		defer close(c.qrCh)

		qrCount := 0
		for evt := range qrChan {
			switch evt.Event {
			case whatsmeow.QRChannelEventCode:
				// qrcode_max_count (0 = unlimited): give up after N regenerations
				// so a never-scanned login doesn't churn QR codes forever.
				qrCount++
				if max := c.config.QRCodeMaxCount; max > 0 && qrCount > max {
					c.logger.Warnf("qrcode_max_count (%d) reached; giving up QR login", max)
					c.mu.Lock()
					c.state = DeviceStateDisconnected
					c.mu.Unlock()
					return
				}
				c.qrCh <- QRCodeEvent{
					Code:      evt.Code,
					ExpiresAt: time.Now().Add(evt.Timeout),
				}
			case whatsmeow.QRChannelEventPasskeyRequest:
				// Passkey-locked ("Shortcake") account: the QR flow can't proceed.
				// Surface the WebAuthn challenge so the owner can sign it in their
				// browser; the assertion comes back via SubmitPasskeyResponse.
				if evt.PasskeyRequest != nil && evt.PasskeyRequest.PublicKey != nil {
					c.qrCh <- QRCodeEvent{PasskeyChallenge: evt.PasskeyRequest.PublicKey}
				}
			case whatsmeow.QRChannelEventPasskeyResponse:
				// SkipHandoffUX == false: whatsmeow did NOT auto-confirm. The code is
				// informational ("check your phone"); confirm so the flow proceeds to
				// the owner's on-device approval.
				if err := c.client.SendPasskeyConfirmation(ctx); err != nil {
					c.logger.Warnf("failed to send passkey confirmation: %v", err)
				}
			case "success":
				c.mu.Lock()
				c.state = DeviceStateConnected
				c.mu.Unlock()
				return
			case "timeout":
				c.mu.Lock()
				c.state = DeviceStateDisconnected
				c.mu.Unlock()
				return
			case whatsmeow.QRChannelEventError:
				c.logger.Warnf("whatsapp pairing error: %v", evt.Error)
			}
		}
	}()

	// Connect to trigger QR code generation
	if err := c.client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect for QR: %w", err)
	}

	return c.qrCh, nil
}

// proxyAddress builds the SOCKS5 proxy URL from the channel config, or "" when
// no proxy is configured. Credentials are URL-escaped.
func (c *Client) proxyAddress() string {
	if c.config == nil || c.config.ProxyHost == "" || c.config.ProxyPort == 0 {
		return ""
	}
	auth := ""
	if c.config.ProxyUser != "" {
		auth = url.QueryEscape(c.config.ProxyUser)
		if c.config.ProxyPass != "" {
			auth += ":" + url.QueryEscape(c.config.ProxyPass)
		}
		auth += "@"
	}
	return fmt.Sprintf("socks5://%s%s:%d", auth, c.config.ProxyHost, c.config.ProxyPort)
}

// SubmitPasskeyResponse forwards a WebAuthn assertion (produced by the account
// owner's authenticator, relayed via the Linktor passkey extension) to WhatsApp
// to complete passkey-based device linking. whatsmeow then auto-routes the
// continuation; the confirmation is handled in the QR-channel loop above.
func (c *Client) SubmitPasskeyResponse(ctx context.Context, assertion *types.WebAuthnResponse) error {
	c.mu.RLock()
	cli := c.client
	c.mu.RUnlock()
	if cli == nil {
		return ErrClientNotReady
	}
	return cli.SendPasskeyResponse(ctx, assertion)
}

// LoginWithPairCode initiates phone number pairing
func (c *Client) LoginWithPairCode(ctx context.Context, phoneNumber string) (*PairCodeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, ErrClientNotReady
	}

	if c.client.Store.ID != nil && c.client.IsConnected() {
		return nil, ErrAlreadyLoggedIn
	}

	// Ensure disconnected before login
	if c.client.IsConnected() {
		c.client.Disconnect()
	}

	c.state = DeviceStateConnecting

	// Connect first
	if err := c.client.Connect(); err != nil {
		c.state = DeviceStateDisconnected
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Request pairing code
	code, err := c.client.PairPhone(ctx, phoneNumber, true, whatsmeow.PairClientChrome, c.config.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pair code: %w", err)
	}

	return &PairCodeResponse{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

// Logout logs out from WhatsApp
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil
	}

	if err := c.client.Logout(ctx); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	c.state = DeviceStateLoggedOut
	return nil
}

// Disconnect disconnects from WhatsApp without logging out
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		c.client.Disconnect()
	}
	c.state = DeviceStateDisconnected
}

// IsLoggedIn returns true if the client is logged in
func (c *Client) IsLoggedIn() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client != nil && c.client.Store.ID != nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client != nil && c.client.IsConnected()
}

// GetState returns the current device state
func (c *Client) GetState() DeviceState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetDeviceInfo returns information about the connected device
func (c *Client) GetDeviceInfo() *DeviceInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.device == nil || c.device.ID == nil {
		return &DeviceInfo{
			State: c.state,
		}
	}

	return &DeviceInfo{
		ID:          c.device.ID.String(),
		JID:         c.device.ID.User,
		PhoneNumber: c.device.ID.User,
		DisplayName: c.config.DeviceName,
		State:       c.state,
		Platform:    c.config.PlatformType,
	}
}

// SendTextMessage sends a text message
func (c *Client) SendTextMessage(ctx context.Context, to, text string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		// Try to format as phone number
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendTextMessageWithReply sends a text message as a reply
func (c *Client) SendTextMessageWithReply(ctx context.Context, to, text, replyToID string, quotedText string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID: proto.String(replyToID),
				QuotedMessage: &waE2E.Message{
					Conversation: proto.String(quotedText),
				},
			},
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send reply: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendImageMessage sends an image message
func (c *Client) SendImageMessage(ctx context.Context, to string, imageData []byte, mimeType, caption string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	// Upload to WhatsApp servers
	uploaded, err := client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			Mimetype:      proto.String(mimeType),
			FileLength:    proto.Uint64(uint64(len(imageData))),
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
			Caption:       proto.String(caption),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send image: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendVideoMessage sends a video message
func (c *Client) SendVideoMessage(ctx context.Context, to string, videoData []byte, mimeType, caption string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	uploaded, err := client.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			Mimetype:      proto.String(mimeType),
			FileLength:    proto.Uint64(uint64(len(videoData))),
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
			Caption:       proto.String(caption),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send video: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendAudioMessage sends an audio message
func (c *Client) SendAudioMessage(ctx context.Context, to string, audioData []byte, mimeType string, ptt bool) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	uploaded, err := client.Upload(ctx, audioData, whatsmeow.MediaAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio: %w", err)
	}

	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			Mimetype:      proto.String(mimeType),
			FileLength:    proto.Uint64(uint64(len(audioData))),
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
			PTT:           proto.Bool(ptt),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send audio: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendDocumentMessage sends a document message
func (c *Client) SendDocumentMessage(ctx context.Context, to string, docData []byte, mimeType, filename, caption string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	uploaded, err := client.Upload(ctx, docData, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}

	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			Mimetype:      proto.String(mimeType),
			FileLength:    proto.Uint64(uint64(len(docData))),
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
			FileName:      proto.String(filename),
			Caption:       proto.String(caption),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send document: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendStickerMessage sends a sticker message
func (c *Client) SendStickerMessage(ctx context.Context, to string, stickerData []byte) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	uploaded, err := client.Upload(ctx, stickerData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("failed to upload sticker: %w", err)
	}

	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			Mimetype:      proto.String("image/webp"),
			FileLength:    proto.Uint64(uint64(len(stickerData))),
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
			IsAnimated:    proto.Bool(false),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send sticker: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendLocationMessage sends a location message
func (c *Client) SendLocationMessage(ctx context.Context, to string, lat, lon float64, name, address string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	msg := &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  proto.Float64(lat),
			DegreesLongitude: proto.Float64(lon),
			Name:             proto.String(name),
			Address:          proto.String(address),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send location: %w", err)
	}

	return &SendMessageResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendReaction sends a reaction to a message
func (c *Client) SendReaction(ctx context.Context, to, messageID, emoji string) error {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	jid, err := resolveRecipientJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	// Build reaction message using the client's BuildReaction helper
	reactionMsg := client.BuildReaction(jid, client.Store.ID.ToNonAD(), messageID, emoji)

	_, err = client.SendMessage(ctx, jid, reactionMsg)
	return err
}

// MarkAsRead marks messages as read
func (c *Client) MarkAsRead(ctx context.Context, messageIDs []string, chatJID types.JID, senderJID types.JID) error {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	// Convert string IDs to MessageID type
	ids := make([]types.MessageID, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = types.MessageID(id)
	}

	return client.MarkRead(ctx, ids, time.Now(), chatJID, senderJID)
}

// SendPresence sends presence (online/offline)
func (c *Client) SendPresence(ctx context.Context, available bool) error {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	presence := types.PresenceUnavailable
	if available {
		presence = types.PresenceAvailable
	}

	return client.SendPresence(ctx, presence)
}

// SendChatPresence sends chat presence (typing/recording)
func (c *Client) SendChatPresence(ctx context.Context, chatJID types.JID, state ChatPresenceState) error {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return ErrClientNotReady
	}

	var presence types.ChatPresence
	var media types.ChatPresenceMedia

	switch state {
	case ChatPresenceComposing:
		presence = types.ChatPresenceComposing
	case ChatPresenceRecording:
		presence = types.ChatPresenceComposing
		media = types.ChatPresenceMediaAudio
	default:
		presence = types.ChatPresencePaused
	}

	return client.SendChatPresence(ctx, chatJID, presence, media)
}

// DownloadMedia downloads media from a message
func (c *Client) DownloadMedia(ctx context.Context, msg any) ([]byte, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	downloadable, ok := msg.(whatsmeow.DownloadableMessage)
	if !ok {
		return nil, fmt.Errorf("message is not downloadable")
	}

	return client.Download(ctx, downloadable)
}

// GetAllContacts returns the address book stored on the paired device.
//
// This is the agent's own WhatsApp contact list — the numbers they saved by
// hand, which is exactly where the useful phone tends to live when the CRM is
// out of date.
//
// Two properties matter to whoever consumes this:
//
// Phone is resolved, not assumed. WhatsApp is migrating identities to opaque
// LID JIDs that carry no phone number. For those we go through ResolvePN; when
// the mapping is unknown the entry still comes back, with Phone empty. An empty
// phone is a distinct outcome from "not a customer", and callers are expected
// to keep them apart — collapsing the two hides a technical failure behind what
// looks like a data one.
//
// Groups and broadcast lists are excluded. They are not people, and a group JID
// would never match a customer record anyway.
func (c *Client) GetAllContacts(ctx context.Context) ([]ContactInfo, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, ErrClientNotReady
	}

	stored, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("read device contacts: %w", err)
	}

	contacts := make([]ContactInfo, 0, len(stored))
	for jid, info := range stored {
		if entry, keep := contactEntry(jid, info, c.ResolvePN(ctx, jid)); keep {
			contacts = append(contacts, entry)
		}
	}

	return contacts, nil
}

// contactEntry decides what a single stored contact becomes.
//
// Split out from GetAllContacts so the two rules that carry consequence — which
// JIDs are people, and when a phone number is actually known — are testable
// without a paired device.
func contactEntry(jid types.JID, info types.ContactInfo, resolved types.JID) (ContactInfo, bool) {
	// Groups and broadcast lists are not people. A group JID would never match
	// a customer record, and letting them through would inflate the "unmatched"
	// pile with entries nobody can act on.
	if jid.Server == types.GroupServer || jid.Server == types.BroadcastServer {
		return ContactInfo{}, false
	}

	entry := ContactInfo{
		JID:          jid,
		FullName:     info.FullName,
		PushName:     info.PushName,
		BusinessName: info.BusinessName,
	}

	// For a plain phone JID the user part IS the number. For a LID it is an
	// opaque id, so Phone is only filled when the resolution actually gave back
	// a phone-number JID — otherwise we would be publishing an internal
	// identifier as if it were a phone.
	if !isLID(resolved) {
		entry.Phone = resolved.User
	}

	return entry, true
}

// GetContactInfo gets information about a contact
func (c *Client) GetContactInfo(ctx context.Context, jid types.JID) (*ContactInfo, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, ErrClientNotReady
	}

	info := &ContactInfo{
		JID: jid,
	}

	// Try to get contact from store
	contact, err := client.Store.Contacts.GetContact(ctx, jid)
	if err == nil {
		info.FullName = contact.FullName
		info.PushName = contact.PushName
		info.BusinessName = contact.BusinessName
	}

	return info, nil
}

// GetProfilePicture gets a contact's profile picture
func (c *Client) GetProfilePicture(ctx context.Context, jid types.JID) (string, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return "", ErrClientNotReady
	}

	pic, err := client.GetProfilePictureInfo(ctx, jid, nil)
	if err != nil {
		return "", err
	}
	if pic == nil {
		return "", nil
	}

	return pic.URL, nil
}

// GetGroupInfo gets information about a group
func (c *Client) GetGroupInfo(ctx context.Context, jid types.JID) (*GroupInfo, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}

	info, err := client.GetGroupInfo(ctx, jid)
	if err != nil {
		return nil, err
	}

	participants := make([]GroupParticipant, len(info.Participants))
	for i, p := range info.Participants {
		participants[i] = GroupParticipant{
			JID:          p.JID,
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
		}
	}

	return &GroupInfo{
		JID:          info.JID,
		Name:         info.Name,
		Topic:        info.Topic,
		Participants: participants,
		CreatedAt:    info.GroupCreated,
		CreatedBy:    info.OwnerJID,
	}, nil
}

// GetEventChannel returns the event channel
func (c *Client) GetEventChannel() <-chan any {
	return c.eventCh
}

// Close closes the client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.stopCh)

	if c.client != nil {
		c.client.Disconnect()
	}

	if c.store != nil {
		return c.store.Close()
	}

	return nil
}

// getOrCreateDevice gets existing device or creates a new one
func (c *Client) getOrCreateDevice(ctx context.Context) (*store.Device, error) {
	// Try to get first device
	device, err := c.store.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// If no device exists, create a new one
	if device == nil {
		device = c.store.NewDevice()
	}

	return device, nil
}

// GetRawClient returns the underlying whatsmeow client
func (c *Client) GetRawClient() *whatsmeow.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}
