package whatsapp_official

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
)

// Adapter implements the WhatsApp Business Cloud API channel adapter
type Adapter struct {
	*plugin.BaseAdapter

	mu               sync.RWMutex
	client           *Client
	webhookProcessor *WebhookProcessor
	messageHandler   plugin.MessageHandler
	statusHandler    plugin.StatusHandler
	config           *Config
	sessions         map[string]*SessionInfo // phone -> session info
}

// NewAdapter creates a new WhatsApp Official adapter
func NewAdapter() *Adapter {
	info := &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeWhatsAppOfficial,
		Name:        "WhatsApp Business Cloud API",
		Description: "Official WhatsApp Business Cloud API integration (Meta)",
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
				plugin.ContentTypeTemplate,
				plugin.ContentTypeInteractive,
			},
			SupportsMedia:           true,
			SupportsLocation:        true,
			SupportsTemplates:       true,
			SupportsInteractive:     true,
			SupportsReadReceipts:    true,
			SupportsTypingIndicator: false, // WhatsApp doesn't support typing indicators via API
			SupportsReactions:       true,
			SupportsReplies:         true,
			SupportsForwarding:      false,
			MaxMessageLength:        4096,
			MaxMediaSize:            16 * 1024 * 1024, // 16MB for documents
			MaxAttachments:          1,                // WhatsApp sends one media per message
			SupportedMediaTypes: []string{
				// Images
				"image/jpeg", "image/png",
				// Audio
				"audio/aac", "audio/mp4", "audio/mpeg", "audio/amr", "audio/ogg",
				// Video
				"video/mp4", "video/3gp",
				// Documents
				"application/pdf",
				"application/vnd.ms-powerpoint",
				"application/msword",
				"application/vnd.ms-excel",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.openxmlformats-officedocument.presentationml.presentation",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"text/plain",
			},
		},
	}

	return &Adapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeWhatsAppOfficial, info),
		config:      &Config{},
		sessions:    make(map[string]*SessionInfo),
	}
}

// Initialize configures the adapter with credentials
func (a *Adapter) Initialize(config map[string]string) error {
	if err := a.BaseAdapter.Initialize(config); err != nil {
		return err
	}

	// Parse configuration
	a.config = &Config{
		AccessToken:   config["access_token"],
		PhoneNumberID: config["phone_number_id"],
		BusinessID:    config["business_id"],
		VerifyToken:   config["verify_token"],
		WebhookSecret: config["webhook_secret"],
		APIVersion:    getOrDefault(config, "api_version", DefaultAPIVersion),
	}

	// Validate required fields
	if a.config.AccessToken == "" {
		return fmt.Errorf("access_token is required")
	}
	if a.config.PhoneNumberID == "" {
		return fmt.Errorf("phone_number_id is required")
	}
	if a.config.VerifyToken == "" {
		return fmt.Errorf("verify_token is required")
	}

	return nil
}

// Connect establishes connection to the WhatsApp API
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create API client
	a.client = NewClient(a.config)

	// Create webhook processor
	a.webhookProcessor = NewWebhookProcessor(a.config)

	// Verify connection by getting phone number info
	phoneInfo, err := a.client.GetPhoneNumberInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify WhatsApp connection: %w", err)
	}

	// Log connection info
	a.SetConnected(true)

	// Store phone info in connection status metadata
	_ = phoneInfo // Can be used for logging or status

	return nil
}

// Disconnect closes the WhatsApp connection
func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.client = nil
	a.webhookProcessor = nil
	a.SetConnected(false)

	return nil
}

// SendMessage sends a message via WhatsApp
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

	// Build WhatsApp message request
	req, err := a.buildSendRequest(msg)
	if err != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	// Send message
	resp, err := client.SendMessage(ctx, req)
	if err != nil {
		return &plugin.SendResult{
			Success:   false,
			Status:    plugin.MessageStatusFailed,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	// Get message ID from response
	externalID := ""
	if len(resp.Messages) > 0 {
		externalID = resp.Messages[0].ID
	}

	return &plugin.SendResult{
		Success:    true,
		ExternalID: externalID,
		Status:     plugin.MessageStatusSent,
		Timestamp:  time.Now(),
	}, nil
}

// buildSendRequest builds a WhatsApp send message request from plugin message
func (a *Adapter) buildSendRequest(msg *plugin.OutboundMessage) (*SendMessageRequest, error) {
	req := &SendMessageRequest{
		To: msg.RecipientID,
	}

	// Handle reply-to context
	if replyTo, ok := msg.Metadata["reply_to_id"]; ok && replyTo != "" {
		req.Context = &ContextObject{
			MessageID: replyTo,
		}
	}

	switch msg.ContentType {
	case plugin.ContentTypeText:
		req.Type = MessageTypeText
		previewURL := false
		if msg.Metadata["preview_url"] == "true" {
			previewURL = true
		}
		req.Text = &TextContent{
			Body:       msg.Content,
			PreviewURL: previewURL,
		}

	case plugin.ContentTypeImage:
		req.Type = MessageTypeImage
		media, err := a.buildMediaObject(msg)
		if err != nil {
			return nil, err
		}
		req.Image = media

	case plugin.ContentTypeVideo:
		req.Type = MessageTypeVideo
		media, err := a.buildMediaObject(msg)
		if err != nil {
			return nil, err
		}
		req.Video = media

	case plugin.ContentTypeAudio:
		req.Type = MessageTypeAudio
		media, err := a.buildMediaObject(msg)
		if err != nil {
			return nil, err
		}
		req.Audio = media

	case plugin.ContentTypeDocument:
		req.Type = MessageTypeDocument
		doc, err := a.buildDocumentObject(msg)
		if err != nil {
			return nil, err
		}
		req.Document = doc

	case plugin.ContentTypeLocation:
		req.Type = MessageTypeLocation
		loc, err := a.buildLocationObject(msg)
		if err != nil {
			return nil, err
		}
		req.Location = loc

	case plugin.ContentTypeContact:
		req.Type = MessageTypeContacts
		contacts, err := a.buildContactsObject(msg)
		if err != nil {
			return nil, err
		}
		req.Contacts = contacts

	case plugin.ContentTypeTemplate:
		// Templates are handled separately via template.go
		template, err := a.buildTemplateFromMetadata(msg)
		if err != nil {
			return nil, err
		}
		req.Type = MessageType("template")
		req.Template = template

	case plugin.ContentTypeInteractive:
		// Interactive messages are handled separately via interactive.go
		interactive, err := a.buildInteractiveFromMetadata(msg)
		if err != nil {
			return nil, err
		}
		req.Type = MessageTypeInteractive
		req.Interactive = interactive

	default:
		// Default to text
		req.Type = MessageTypeText
		req.Text = &TextContent{
			Body: msg.Content,
		}
	}

	return req, nil
}

// buildMediaObject builds a media object from attachments
func (a *Adapter) buildMediaObject(msg *plugin.OutboundMessage) (*MediaObject, error) {
	media := &MediaObject{
		Caption: msg.Content,
	}

	if len(msg.Attachments) > 0 {
		att := msg.Attachments[0]
		// Check if we have a media ID or URL
		if mediaID, ok := att.Metadata["media_id"]; ok && mediaID != "" {
			media.ID = mediaID
		} else if att.URL != "" {
			media.Link = att.URL
		} else {
			return nil, fmt.Errorf("media attachment must have either media_id or URL")
		}
	} else if msg.Metadata["media_id"] != "" {
		media.ID = msg.Metadata["media_id"]
	} else if msg.Metadata["media_url"] != "" {
		media.Link = msg.Metadata["media_url"]
	} else {
		return nil, fmt.Errorf("no media provided")
	}

	return media, nil
}

// buildDocumentObject builds a document object from attachments
func (a *Adapter) buildDocumentObject(msg *plugin.OutboundMessage) (*DocumentObject, error) {
	doc := &DocumentObject{
		Caption: msg.Content,
	}

	if len(msg.Attachments) > 0 {
		att := msg.Attachments[0]
		doc.Filename = att.Filename
		if mediaID, ok := att.Metadata["media_id"]; ok && mediaID != "" {
			doc.ID = mediaID
		} else if att.URL != "" {
			doc.Link = att.URL
		} else {
			return nil, fmt.Errorf("document attachment must have either media_id or URL")
		}
	} else if msg.Metadata["media_id"] != "" {
		doc.ID = msg.Metadata["media_id"]
		doc.Filename = msg.Metadata["filename"]
	} else if msg.Metadata["media_url"] != "" {
		doc.Link = msg.Metadata["media_url"]
		doc.Filename = msg.Metadata["filename"]
	} else {
		return nil, fmt.Errorf("no document provided")
	}

	if doc.Filename == "" {
		doc.Filename = "document"
	}

	return doc, nil
}

// buildLocationObject builds a location object from metadata
func (a *Adapter) buildLocationObject(msg *plugin.OutboundMessage) (*LocationObject, error) {
	// Try to parse location from content as JSON
	var loc LocationObject
	if err := json.Unmarshal([]byte(msg.Content), &loc); err == nil {
		return &loc, nil
	}

	// Try to build from metadata
	if lat, ok := msg.Metadata["latitude"]; ok {
		var latf, lonf float64
		fmt.Sscanf(lat, "%f", &latf)
		if lon, ok := msg.Metadata["longitude"]; ok {
			fmt.Sscanf(lon, "%f", &lonf)
		}
		loc.Latitude = latf
		loc.Longitude = lonf
		loc.Name = msg.Metadata["name"]
		loc.Address = msg.Metadata["address"]
		return &loc, nil
	}

	return nil, fmt.Errorf("location data not provided in content or metadata")
}

// buildContactsObject builds contacts from metadata
func (a *Adapter) buildContactsObject(msg *plugin.OutboundMessage) ([]ContactContent, error) {
	var contacts []ContactContent
	if err := json.Unmarshal([]byte(msg.Content), &contacts); err != nil {
		return nil, fmt.Errorf("failed to parse contacts: %w", err)
	}
	return contacts, nil
}

// buildTemplateFromMetadata builds a template object from metadata
// This is a stub - full implementation in template.go
func (a *Adapter) buildTemplateFromMetadata(msg *plugin.OutboundMessage) (*TemplateObject, error) {
	templateName := msg.Metadata["template_name"]
	if templateName == "" {
		return nil, fmt.Errorf("template_name is required in metadata")
	}

	languageCode := msg.Metadata["template_language"]
	if languageCode == "" {
		languageCode = "en"
	}

	template := &TemplateObject{
		Name: templateName,
		Language: &TemplateLanguage{
			Code: languageCode,
		},
	}

	// Parse components from metadata if present
	if componentsJSON, ok := msg.Metadata["template_components"]; ok && componentsJSON != "" {
		var components []TemplateComponent
		if err := json.Unmarshal([]byte(componentsJSON), &components); err != nil {
			return nil, fmt.Errorf("failed to parse template components: %w", err)
		}
		template.Components = components
	}

	return template, nil
}

// buildInteractiveFromMetadata builds an interactive object from metadata
// This is a stub - full implementation in interactive.go
func (a *Adapter) buildInteractiveFromMetadata(msg *plugin.OutboundMessage) (*InteractiveObject, error) {
	interactiveType := msg.Metadata["interactive_type"]
	if interactiveType == "" {
		return nil, fmt.Errorf("interactive_type is required in metadata")
	}

	// Parse the full interactive object from content
	var interactive InteractiveObject
	if err := json.Unmarshal([]byte(msg.Content), &interactive); err != nil {
		return nil, fmt.Errorf("failed to parse interactive message: %w", err)
	}

	return &interactive, nil
}

// SendTypingIndicator - WhatsApp API doesn't support typing indicators
func (a *Adapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	// WhatsApp Cloud API doesn't support typing indicators
	return nil
}

// SendReadReceipt marks a message as read
func (a *Adapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("adapter not connected")
	}

	return client.MarkAsRead(ctx, receipt.MessageID)
}

// UploadMedia uploads media to WhatsApp servers
func (a *Adapter) UploadMedia(ctx context.Context, media *plugin.Media) (*plugin.MediaUpload, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return &plugin.MediaUpload{
			Success: false,
			Error:   "adapter not connected",
		}, nil
	}

	resp, err := client.UploadMedia(ctx, media.Filename, media.MimeType, media.Data)
	if err != nil {
		return &plugin.MediaUpload{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &plugin.MediaUpload{
		Success: true,
		MediaID: resp.ID,
	}, nil
}

// DownloadMedia downloads media from WhatsApp servers
func (a *Adapter) DownloadMedia(ctx context.Context, mediaID string) (*plugin.Media, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("adapter not connected")
	}

	// First, get media info to get the URL
	info, err := client.GetMediaInfo(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media info: %w", err)
	}

	// Download the media content
	data, contentType, err := client.DownloadMedia(ctx, info.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	return &plugin.Media{
		ID:        mediaID,
		URL:       info.URL,
		Data:      data,
		MimeType:  contentType,
		SizeBytes: info.FileSize,
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
	return "/api/v1/webhooks/whatsapp"
}

// ValidateWebhook validates a webhook request signature
func (a *Adapter) ValidateWebhook(headers map[string]string, body []byte) bool {
	signature := headers["X-Hub-Signature-256"]
	if signature == "" {
		signature = headers["x-hub-signature-256"]
	}

	if a.config.WebhookSecret == "" {
		// If no secret configured, skip validation
		return true
	}

	return ValidateSignature(a.config.WebhookSecret, body, signature)
}

// HandleWebhook processes incoming webhook requests
func (a *Adapter) HandleWebhook(ctx context.Context, body []byte) error {
	a.mu.RLock()
	processor := a.webhookProcessor
	msgHandler := a.messageHandler
	statusHandler := a.statusHandler
	a.mu.RUnlock()

	if processor == nil {
		return fmt.Errorf("webhook processor not initialized")
	}

	// Parse webhook
	payload, err := processor.ParseWebhook(body)
	if err != nil {
		return err
	}

	// Process messages
	if msgHandler != nil {
		messages := processor.ExtractMessages(payload)
		for _, msg := range messages {
			// Update session tracking
			a.updateSession(msg.From)

			// Convert to plugin format and call handler
			if err := msgHandler(ctx, msg.ToInboundMessage()); err != nil {
				// Log error but continue processing
				continue
			}
		}
	}

	// Process status updates
	if statusHandler != nil {
		statuses := processor.ExtractStatuses(payload)
		for _, status := range statuses {
			if err := statusHandler(ctx, status.ToStatusCallback()); err != nil {
				// Log error but continue processing
				continue
			}
		}
	}

	return nil
}

// VerifyWebhookChallenge handles Meta's webhook verification challenge
func (a *Adapter) VerifyWebhookChallenge(mode, token, challenge string) (string, bool) {
	return VerifyChallenge(a.config.VerifyToken, mode, token, challenge)
}

// GetClient returns the WhatsApp API client
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

// IsSessionValid checks if the 24-hour messaging window is open for a contact
func (a *Adapter) IsSessionValid(phone string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	session, ok := a.sessions[phone]
	if !ok {
		return false
	}

	return session.IsSessionValid()
}

// updateSession updates the session after receiving a customer message
func (a *Adapter) updateSession(phone string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session, ok := a.sessions[phone]
	if !ok {
		session = &SessionInfo{
			ContactID: phone,
		}
		a.sessions[phone] = session
	}

	session.UpdateSession()
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
		status.Metadata["phone_number_id"] = a.config.PhoneNumberID
		status.Metadata["api_version"] = a.config.APIVersion
	} else {
		status.Status = "disconnected"
	}

	return status
}

// Helper function
func getOrDefault(config map[string]string, key, defaultValue string) string {
	if v, ok := config[key]; ok && v != "" {
		return v
	}
	return defaultValue
}

// Ensure Adapter implements the required interfaces
var _ plugin.ChannelAdapter = (*Adapter)(nil)
var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
