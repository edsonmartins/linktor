package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/email"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/rcs"
	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// WebhookHandler handles incoming webhooks from external channels
type WebhookHandler struct {
	channelRepo repository.ChannelRepository
	producer    *nats.Producer
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(channelRepo repository.ChannelRepository, producer *nats.Producer) *WebhookHandler {
	return &WebhookHandler{
		channelRepo: channelRepo,
		producer:    producer,
	}
}

// WhatsAppWebhook handles WhatsApp Cloud API webhooks
func (h *WebhookHandler) WhatsAppWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Handle verification challenge
	if c.Request.Method == http.MethodGet {
		h.handleWhatsAppVerification(c)
		return
	}

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Verify signature if secret is configured
	if secret, ok := channel.Credentials["webhook_secret"]; ok && secret != "" {
		if !h.verifyWhatsAppSignature(c, secret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	}

	// Parse webhook payload
	var payload WhatsAppWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Process messages
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" {
				for _, msg := range change.Value.Messages {
					if err := h.processWhatsAppMessage(c.Request.Context(), channel, msg, change.Value.Contacts); err != nil {
						// Log error but continue processing
					}
				}

				// Process status updates
				for _, status := range change.Value.Statuses {
					h.processWhatsAppStatus(c.Request.Context(), channel, status)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TelegramWebhook handles Telegram Bot API webhooks
func (h *WebhookHandler) TelegramWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var update TelegramUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if update.Message != nil {
		if err := h.processTelegramMessage(c.Request.Context(), channel, update.Message); err != nil {
			// Log error but continue
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GenericWebhook handles webhooks from generic/custom channels
func (h *WebhookHandler) GenericWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var payload GenericWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: string(channel.Type),
		ExternalID:  payload.MessageID,
		ContentType: payload.ContentType,
		Content:     payload.Content,
		Metadata:    payload.Metadata,
		Timestamp:   time.Now(),
	}

	if payload.SenderID != "" {
		inbound.Metadata["sender_id"] = payload.SenderID
	}
	if payload.SenderName != "" {
		inbound.Metadata["sender_name"] = payload.SenderName
	}

	if err := h.producer.PublishInbound(c.Request.Context(), inbound); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message_id": inbound.ID})
}

// TwilioWebhook handles Twilio SMS/MMS webhooks
func (h *WebhookHandler) TwilioWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Read body (form-encoded)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Parse webhook
	payload, webhookType, err := sms.ParseWebhook(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	switch webhookType {
	case sms.WebhookTypeIncoming:
		// Handle incoming SMS/MMS
		contentType := "text"
		content := payload.Body
		var attachments []nats.AttachmentData

		// Check for MMS media
		numMedia := 0
		if payload.NumMedia != "" {
			fmt.Sscanf(payload.NumMedia, "%d", &numMedia)
		}

		if numMedia > 0 {
			contentType = "image"
			// Extract media URLs from form data
			values, _ := url.ParseQuery(string(body))
			for i := 0; i < numMedia; i++ {
				mediaURL := values.Get(fmt.Sprintf("MediaUrl%d", i))
				mediaType := values.Get(fmt.Sprintf("MediaContentType%d", i))
				if mediaURL != "" {
					attachments = append(attachments, nats.AttachmentData{
						Type:     "image",
						URL:      mediaURL,
						MimeType: mediaType,
					})
				}
			}
		}

		// Build metadata
		metadata := map[string]string{
			"sender_id":    payload.From,
			"from":         payload.From,
			"to":           payload.To,
			"account_sid":  payload.AccountSID,
			"from_city":    payload.FromCity,
			"from_state":   payload.FromState,
			"from_zip":     payload.FromZip,
			"from_country": payload.FromCountry,
		}

		// Publish to NATS
		inbound := &nats.InboundMessage{
			ID:          uuid.New().String(),
			TenantID:    channel.TenantID,
			ChannelID:   channel.ID,
			ChannelType: "sms",
			ExternalID:  payload.MessageSID,
			ContentType: contentType,
			Content:     content,
			Metadata:    metadata,
			Attachments: attachments,
			Timestamp:   time.Now(),
		}

		if err := h.producer.PublishInbound(c.Request.Context(), inbound); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process message"})
			return
		}

		// Return TwiML response (empty response)
		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, sms.EmptyTwiMLResponse())

	case sms.WebhookTypeStatus:
		// Handle status callback
		twilioStatus := payload.MessageStatus
		if twilioStatus == "" {
			twilioStatus = payload.SmsStatus
		}

		// Map Twilio status
		var status string
		switch sms.ParseMessageStatus(twilioStatus) {
		case sms.StatusDelivered:
			status = "delivered"
		case sms.StatusRead:
			status = "read"
		case sms.StatusFailed, sms.StatusUndelivered:
			status = "failed"
		case sms.StatusSent:
			status = "sent"
		default:
			status = "pending"
		}

		// Publish status update
		statusUpdate := &nats.StatusUpdate{
			ExternalID:   payload.MessageSID,
			ChannelType:  "sms",
			Status:       status,
			ErrorMessage: payload.ErrorMessage,
			Timestamp:    time.Now(),
		}
		h.producer.PublishStatusUpdate(c.Request.Context(), statusUpdate)

		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, sms.EmptyTwiMLResponse())

	default:
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// FacebookWebhook handles Facebook Messenger webhooks
func (h *WebhookHandler) FacebookWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Handle verification challenge (GET request)
	if c.Request.Method == http.MethodGet {
		h.handleFacebookVerification(c, channel)
		return
	}

	// Get app secret from credentials
	appSecret := channel.Credentials["app_secret"]
	verifyToken := channel.Credentials["verify_token"]

	// Create webhook handler
	webhookHandler := facebook.NewWebhookHandler(appSecret, verifyToken)

	// Parse webhook payload
	payload, err := webhookHandler.ParseWebhook(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if it's a Messenger webhook
	if !facebook.IsMessengerWebhook(payload) {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// Extract and process messages
	messages := facebook.ExtractMessages(payload)
	for _, msg := range messages {
		// Skip echo messages
		if msg.IsEcho {
			continue
		}

		if err := h.processFacebookMessage(c.Request.Context(), channel, msg); err != nil {
			// Log error but continue
		}
	}

	// Process delivery statuses
	deliveryStatuses := facebook.ExtractDeliveryStatuses(payload)
	for _, status := range deliveryStatuses {
		h.processFacebookDeliveryStatus(c.Request.Context(), channel, status)
	}

	// Process read statuses
	readStatuses := facebook.ExtractReadStatuses(payload)
	for _, status := range readStatuses {
		h.processFacebookReadStatus(c.Request.Context(), channel, status)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// InstagramWebhook handles Instagram DM webhooks
func (h *WebhookHandler) InstagramWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Handle verification challenge (GET request)
	if c.Request.Method == http.MethodGet {
		h.handleInstagramVerification(c, channel)
		return
	}

	// Get app secret from credentials
	appSecret := channel.Credentials["app_secret"]
	verifyToken := channel.Credentials["verify_token"]

	// Create webhook handler
	webhookHandler := instagram.NewWebhookHandler(appSecret, verifyToken)

	// Parse webhook payload
	payload, err := webhookHandler.ParseWebhook(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if it's an Instagram webhook
	if !instagram.IsInstagramWebhook(payload) && !instagram.IsInstagramViaPageWebhook(payload) {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// Extract and process messages
	messages := instagram.ExtractMessages(payload)
	for _, msg := range messages {
		// Skip echo messages
		if msg.IsEcho {
			continue
		}

		// Skip deleted messages
		if msg.IsDeleted {
			continue
		}

		if err := h.processInstagramMessage(c.Request.Context(), channel, msg); err != nil {
			// Log error but continue
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// StatusCallback handles message status callbacks
func (h *WebhookHandler) StatusCallback(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var payload StatusCallbackPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	status := &nats.StatusUpdate{
		MessageID:    payload.MessageID,
		ExternalID:   payload.ExternalID,
		ChannelType:  string(channel.Type),
		Status:       payload.Status,
		ErrorMessage: payload.ErrorMessage,
		Timestamp:    time.Now(),
	}

	if err := h.producer.PublishStatusUpdate(c.Request.Context(), status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Helper methods

func (h *WebhookHandler) handleWhatsAppVerification(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	channelID := c.Param("channelId")
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	verifyToken := channel.Credentials["verify_token"]

	if mode == "subscribe" && token == verifyToken {
		c.String(http.StatusOK, challenge)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "verification failed"})
}

func (h *WebhookHandler) verifyWhatsAppSignature(c *gin.Context, secret string) bool {
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}
	// Restore body for further processing
	c.Request.Body = io.NopCloser(io.MultiReader(io.NopCloser(io.LimitReader(c.Request.Body, 0)), io.NopCloser(
		&bodyReader{data: body},
	)))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

func (h *WebhookHandler) processWhatsAppMessage(ctx context.Context, channel *entity.Channel, msg WhatsAppMessage, contacts []WhatsAppContact) error {
	// Find sender info
	senderName := ""
	senderPhone := msg.From
	for _, contact := range contacts {
		if contact.WaID == msg.From {
			senderName = contact.Profile.Name
			break
		}
	}

	// Determine content type and content
	contentType := "text"
	content := ""
	var attachments []nats.AttachmentData
	metadata := map[string]string{
		"phone":       senderPhone,
		"sender_name": senderName,
		"sender_id":   msg.From,
	}

	switch msg.Type {
	case "text":
		contentType = "text"
		content = msg.Text.Body
	case "image":
		contentType = "image"
		content = msg.Image.Caption
		attachments = append(attachments, nats.AttachmentData{
			Type:     "image",
			URL:      msg.Image.ID, // Media ID - will be resolved later
			MimeType: msg.Image.MimeType,
			Metadata: map[string]string{"media_id": msg.Image.ID},
		})
	case "video":
		contentType = "video"
		content = msg.Video.Caption
		attachments = append(attachments, nats.AttachmentData{
			Type:     "video",
			URL:      msg.Video.ID,
			MimeType: msg.Video.MimeType,
			Metadata: map[string]string{"media_id": msg.Video.ID},
		})
	case "audio":
		contentType = "audio"
		attachments = append(attachments, nats.AttachmentData{
			Type:     "audio",
			URL:      msg.Audio.ID,
			MimeType: msg.Audio.MimeType,
			Metadata: map[string]string{"media_id": msg.Audio.ID},
		})
	case "document":
		contentType = "document"
		content = msg.Document.Caption
		attachments = append(attachments, nats.AttachmentData{
			Type:     "document",
			URL:      msg.Document.ID,
			Filename: msg.Document.Filename,
			MimeType: msg.Document.MimeType,
			Metadata: map[string]string{"media_id": msg.Document.ID},
		})
	case "sticker":
		contentType = "image"
		metadata["is_sticker"] = "true"
		attachments = append(attachments, nats.AttachmentData{
			Type:     "sticker",
			URL:      msg.Sticker.ID,
			MimeType: msg.Sticker.MimeType,
			Metadata: map[string]string{"media_id": msg.Sticker.ID, "animated": "false"},
		})
	case "location":
		contentType = "location"
		locationData, _ := json.Marshal(msg.Location)
		content = string(locationData)
		metadata["latitude"] = fmt.Sprintf("%f", msg.Location.Latitude)
		metadata["longitude"] = fmt.Sprintf("%f", msg.Location.Longitude)
		if msg.Location.Name != "" {
			metadata["location_name"] = msg.Location.Name
		}
		if msg.Location.Address != "" {
			metadata["location_address"] = msg.Location.Address
		}
	case "contacts":
		contentType = "contact"
		if len(msg.Contacts) > 0 {
			data, _ := json.Marshal(msg.Contacts)
			content = string(data)
		}
	case "interactive":
		contentType = "interactive"
		if msg.Interactive != nil {
			metadata["interactive_type"] = msg.Interactive.Type
			if msg.Interactive.ButtonReply != nil {
				content = msg.Interactive.ButtonReply.Title
				metadata["button_id"] = msg.Interactive.ButtonReply.ID
			} else if msg.Interactive.ListReply != nil {
				content = msg.Interactive.ListReply.Title
				metadata["list_id"] = msg.Interactive.ListReply.ID
			}
		}
	case "button":
		contentType = "interactive"
		if msg.Button != nil {
			content = msg.Button.Text
			metadata["button_payload"] = msg.Button.Payload
		}
	case "reaction":
		contentType = "text"
		metadata["is_reaction"] = "true"
		if msg.Reaction != nil {
			content = msg.Reaction.Emoji
			metadata["reaction_message_id"] = msg.Reaction.MessageID
		}
	default:
		contentType = "text"
		metadata["original_type"] = msg.Type
	}

	// Handle reply-to context
	if msg.Context != nil && msg.Context.ID != "" {
		metadata["reply_to_id"] = msg.Context.ID
		metadata["reply_to_from"] = msg.Context.From
	}

	// Determine channel type from channel entity
	channelType := string(channel.Type)
	if channelType == "whatsapp" || channelType == "whatsapp_official" {
		channelType = string(channel.Type)
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: channelType,
		ExternalID:  msg.ID,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.producer.PublishInbound(ctx, inbound)
}

func (h *WebhookHandler) processWhatsAppStatus(ctx context.Context, channel *entity.Channel, status WhatsAppStatus) {
	statusMap := map[string]string{
		"sent":      "sent",
		"delivered": "delivered",
		"read":      "read",
		"failed":    "failed",
	}

	mappedStatus, ok := statusMap[status.Status]
	if !ok {
		return
	}

	errorMessage := ""
	if len(status.Errors) > 0 {
		errorMessage = fmt.Sprintf("[%d] %s: %s", status.Errors[0].Code, status.Errors[0].Title, status.Errors[0].Message)
	}

	// Use the channel type from entity
	channelType := string(channel.Type)

	update := &nats.StatusUpdate{
		ExternalID:   status.ID,
		ChannelType:  channelType,
		Status:       mappedStatus,
		ErrorMessage: errorMessage,
		Timestamp:    time.Now(),
	}

	h.producer.PublishStatusUpdate(ctx, update)
}

func (h *WebhookHandler) processTelegramMessage(ctx context.Context, channel *entity.Channel, msg *TelegramMessage) error {
	contentType := "text"
	content := msg.Text
	var attachments []nats.AttachmentData

	if msg.Photo != nil && len(msg.Photo) > 0 {
		contentType = "image"
		content = msg.Caption
		// Get largest photo
		photo := msg.Photo[len(msg.Photo)-1]
		attachments = append(attachments, nats.AttachmentData{
			Type: "image",
			URL:  photo.FileID,
		})
	} else if msg.Document != nil {
		contentType = "document"
		content = msg.Caption
		attachments = append(attachments, nats.AttachmentData{
			Type:     "document",
			URL:      msg.Document.FileID,
			Filename: msg.Document.FileName,
			MimeType: msg.Document.MimeType,
		})
	} else if msg.Voice != nil {
		contentType = "audio"
		attachments = append(attachments, nats.AttachmentData{
			Type:     "audio",
			URL:      msg.Voice.FileID,
			MimeType: msg.Voice.MimeType,
		})
	} else if msg.Video != nil {
		contentType = "video"
		content = msg.Caption
		attachments = append(attachments, nats.AttachmentData{
			Type:     "video",
			URL:      msg.Video.FileID,
			MimeType: msg.Video.MimeType,
		})
	} else if msg.Location != nil {
		contentType = "location"
	}

	senderID := ""
	senderName := ""
	if msg.From != nil {
		senderID = string(rune(msg.From.ID))
		senderName = msg.From.FirstName
		if msg.From.LastName != "" {
			senderName += " " + msg.From.LastName
		}
	}

	metadata := map[string]string{
		"sender_id":   senderID,
		"sender_name": senderName,
		"chat_id":     string(rune(msg.Chat.ID)),
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "telegram",
		ExternalID:  string(rune(msg.MessageID)),
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.producer.PublishInbound(ctx, inbound)
}

func (h *WebhookHandler) handleFacebookVerification(c *gin.Context, channel *entity.Channel) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	verifyToken := channel.Credentials["verify_token"]

	if mode == "subscribe" && token == verifyToken {
		c.String(http.StatusOK, challenge)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "verification failed"})
}

func (h *WebhookHandler) handleInstagramVerification(c *gin.Context, channel *entity.Channel) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	verifyToken := channel.Credentials["verify_token"]

	if mode == "subscribe" && token == verifyToken {
		c.String(http.StatusOK, challenge)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "verification failed"})
}

func (h *WebhookHandler) processFacebookMessage(ctx context.Context, channel *entity.Channel, msg *facebook.IncomingMessage) error {
	contentType := "text"
	content := msg.Text
	var attachments []nats.AttachmentData

	// Handle attachments
	if len(msg.Attachments) > 0 {
		att := msg.Attachments[0]
		contentType = facebook.GetAttachmentType(att.Type)
		attachments = append(attachments, nats.AttachmentData{
			Type: att.Type,
			URL:  att.URL,
		})

		// Handle location
		if att.Type == "location" {
			contentType = "location"
		}
	}

	metadata := map[string]string{
		"sender_id": msg.SenderID,
		"page_id":   msg.PageID,
	}

	if msg.QuickReply != "" {
		metadata["quick_reply"] = msg.QuickReply
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "facebook",
		ExternalID:  msg.ExternalID,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   msg.Timestamp,
	}

	return h.producer.PublishInbound(ctx, inbound)
}

func (h *WebhookHandler) processFacebookDeliveryStatus(ctx context.Context, channel *entity.Channel, status *facebook.DeliveryStatus) {
	for _, msgID := range status.MessageIDs {
		update := &nats.StatusUpdate{
			ExternalID:  msgID,
			ChannelType: "facebook",
			Status:      "delivered",
			Timestamp:   status.Watermark,
		}
		h.producer.PublishStatusUpdate(ctx, update)
	}
}

func (h *WebhookHandler) processFacebookReadStatus(ctx context.Context, channel *entity.Channel, status *facebook.ReadStatus) {
	// Facebook read status doesn't include specific message IDs, just watermark
	// We can't update specific messages, but we can use the watermark for context
}

func (h *WebhookHandler) processInstagramMessage(ctx context.Context, channel *entity.Channel, msg *instagram.IncomingMessage) error {
	contentType := "text"
	content := msg.Text
	var attachments []nats.AttachmentData

	// Handle attachments
	if len(msg.Attachments) > 0 {
		att := msg.Attachments[0]
		contentType = instagram.GetAttachmentType(att.Type)
		attachments = append(attachments, nats.AttachmentData{
			Type: att.Type,
			URL:  att.URL,
		})
	}

	metadata := map[string]string{
		"sender_id":    msg.SenderID,
		"instagram_id": msg.InstagramID,
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "instagram",
		ExternalID:  msg.ExternalID,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   msg.Timestamp,
	}

	return h.producer.PublishInbound(ctx, inbound)
}

// Payload types

type bodyReader struct {
	data []byte
	pos  int
}

func (b *bodyReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

// RCSWebhook handles RCS Business Messaging webhooks
func (h *WebhookHandler) RCSWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Create RCS client for validation
	rcsConfig := &rcs.Config{
		Provider:      rcs.Provider(channel.Config["provider"]),
		AgentID:       channel.Config["agent_id"],
		APIKey:        channel.Credentials["api_key"],
		WebhookSecret: channel.Credentials["webhook_secret"],
	}
	client, err := rcs.NewClient(rcsConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client"})
		return
	}

	// Verify signature if secret is configured
	if rcsConfig.WebhookSecret != "" {
		signature := c.GetHeader("X-Signature")
		if signature == "" {
			signature = c.GetHeader("X-Hub-Signature-256")
		}
		if !client.ValidateWebhook(signature, body) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	}

	// Parse webhook payload
	payload, err := client.ParseWebhook(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Process based on type
	switch payload.Type {
	case "message":
		if payload.Message != nil {
			if err := h.processRCSMessage(c.Request.Context(), channel, payload.Message); err != nil {
				// Log error but continue
			}
		}
	case "status":
		if payload.Status != nil {
			h.processRCSStatus(c.Request.Context(), channel, payload.Status)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// processRCSMessage processes an incoming RCS message
func (h *WebhookHandler) processRCSMessage(ctx context.Context, channel *entity.Channel, msg *rcs.IncomingMessage) error {
	contentType := "text"
	content := msg.Text
	var attachments []nats.AttachmentData

	// Determine content type and handle media
	if msg.MediaURL != "" {
		switch {
		case len(msg.MediaType) >= 6 && msg.MediaType[:6] == "image/":
			contentType = "image"
		case len(msg.MediaType) >= 6 && msg.MediaType[:6] == "video/":
			contentType = "video"
		case len(msg.MediaType) >= 6 && msg.MediaType[:6] == "audio/":
			contentType = "audio"
		default:
			contentType = "document"
		}
		attachments = append(attachments, nats.AttachmentData{
			Type:     contentType,
			URL:      msg.MediaURL,
			MimeType: msg.MediaType,
		})
	} else if msg.Location != nil {
		contentType = "location"
		content = fmt.Sprintf("%s: %.6f,%.6f", msg.Location.Label, msg.Location.Latitude, msg.Location.Longitude)
	}

	// Build metadata
	metadata := map[string]string{
		"sender_phone": msg.SenderPhone,
		"agent_id":     msg.AgentID,
	}

	// Add suggestion/postback data
	if msg.Suggestion != nil {
		metadata["postback_data"] = msg.Suggestion.PostbackData
	}

	// Publish inbound message
	if h.producer != nil {
		inboundMsg := &nats.InboundMessage{
			ID:          uuid.New().String(),
			TenantID:    channel.TenantID,
			ChannelID:   channel.ID,
			ChannelType: "rcs",
			ExternalID:  msg.ExternalID,
			ContentType: contentType,
			Content:     content,
			Metadata:    metadata,
			Attachments: attachments,
			Timestamp:   msg.Timestamp,
		}

		return h.producer.PublishInbound(ctx, inboundMsg)
	}

	return nil
}

// processRCSStatus processes an RCS delivery status update
func (h *WebhookHandler) processRCSStatus(ctx context.Context, channel *entity.Channel, report *rcs.DeliveryReport) {
	var status string
	switch report.Status {
	case rcs.StatusSent:
		status = "sent"
	case rcs.StatusDelivered:
		status = "delivered"
	case rcs.StatusRead:
		status = "read"
	case rcs.StatusFailed:
		status = "failed"
	default:
		status = "pending"
	}

	if h.producer != nil {
		h.producer.PublishStatusUpdate(ctx, &nats.StatusUpdate{
			ExternalID:   report.MessageID,
			ChannelType:  "rcs",
			Status:       status,
			ErrorMessage: report.Error,
			Timestamp:    report.Timestamp,
		})
	}
}

// EmailWebhook handles Email webhooks from various providers (SendGrid, Mailgun, SES, Postmark)
func (h *WebhookHandler) EmailWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Determine provider from config or URL
	provider := email.Provider(channel.Config["provider"])
	if provider == "" {
		// Try to determine from URL path
		path := c.Request.URL.Path
		switch {
		case contains(path, "sendgrid"):
			provider = email.ProviderSendGrid
		case contains(path, "mailgun"):
			provider = email.ProviderMailgun
		case contains(path, "ses"):
			provider = email.ProviderSES
		case contains(path, "postmark"):
			provider = email.ProviderPostmark
		default:
			provider = email.ProviderSMTP
		}
	}

	// Build headers map
	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.Request.Header.Get(key)
	}

	// Parse webhook payload
	payload, err := email.ParseWebhook(provider, body, headers)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: " + err.Error()})
		return
	}

	// Process based on type
	switch payload.Type {
	case "inbound":
		if payload.IncomingEmail != nil {
			if err := h.processEmailMessage(c.Request.Context(), channel, payload.IncomingEmail); err != nil {
				// Log error but continue
			}
		}
	case "status":
		if payload.StatusCallback != nil {
			h.processEmailStatus(c.Request.Context(), channel, payload.StatusCallback)
		}
	case "subscription_confirmation":
		// SES subscription confirmation - return 200 to acknowledge
		c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// processEmailMessage processes an incoming email message
func (h *WebhookHandler) processEmailMessage(ctx context.Context, channel *entity.Channel, msg *email.IncomingEmail) error {
	contentType := "text"
	content := msg.TextBody
	var attachments []nats.AttachmentData

	// Prefer text body, but use HTML if no text
	if content == "" && msg.HTMLBody != "" {
		content = msg.HTMLBody
		contentType = "text" // Still text, but HTML
	}

	// Handle attachments
	for _, att := range msg.Attachments {
		attType := "document"
		if len(att.ContentType) >= 6 {
			switch att.ContentType[:6] {
			case "image/":
				attType = "image"
			case "video/":
				attType = "video"
			case "audio/":
				attType = "audio"
			}
		}

		attachments = append(attachments, nats.AttachmentData{
			Type:      attType,
			URL:       att.URL,
			Filename:  att.Filename,
			MimeType:  att.ContentType,
			SizeBytes: att.Size,
		})
	}

	// Build metadata
	metadata := map[string]string{
		"sender_id":   msg.From,
		"sender_name": msg.FromName,
		"subject":     msg.Subject,
		"message_id":  msg.MessageID,
	}

	if len(msg.To) > 0 {
		metadata["to"] = joinStrings(msg.To, ",")
	}
	if len(msg.CC) > 0 {
		metadata["cc"] = joinStrings(msg.CC, ",")
	}
	if msg.InReplyTo != "" {
		metadata["in_reply_to"] = msg.InReplyTo
	}
	if msg.References != "" {
		metadata["references"] = msg.References
	}
	if msg.SpamScore > 0 {
		metadata["spam_score"] = fmt.Sprintf("%.2f", msg.SpamScore)
	}

	// Publish inbound message
	if h.producer != nil {
		inboundMsg := &nats.InboundMessage{
			ID:          uuid.New().String(),
			TenantID:    channel.TenantID,
			ChannelID:   channel.ID,
			ChannelType: "email",
			ExternalID:  msg.MessageID,
			ContentType: contentType,
			Content:     content,
			Metadata:    metadata,
			Attachments: attachments,
			Timestamp:   msg.ReceivedAt,
		}

		return h.producer.PublishInbound(ctx, inboundMsg)
	}

	return nil
}

// processEmailStatus processes an email delivery status update
func (h *WebhookHandler) processEmailStatus(ctx context.Context, channel *entity.Channel, report *email.StatusCallback) {
	var status string
	switch report.Status {
	case email.StatusSent:
		status = "sent"
	case email.StatusDelivered:
		status = "delivered"
	case email.StatusOpened, email.StatusClicked:
		status = "read"
	case email.StatusBounced, email.StatusFailed, email.StatusSpam:
		status = "failed"
	default:
		status = "pending"
	}

	if h.producer != nil {
		h.producer.PublishStatusUpdate(ctx, &nats.StatusUpdate{
			MessageID:    report.MessageID,
			ExternalID:   report.ExternalID,
			ChannelType:  "email",
			Status:       status,
			ErrorMessage: report.ErrorMessage,
			Timestamp:    report.Timestamp,
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// WhatsApp types
type WhatsAppWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				MessagingProduct string            `json:"messaging_product"`
				Metadata         map[string]string `json:"metadata"`
				Contacts         []WhatsAppContact `json:"contacts"`
				Messages         []WhatsAppMessage `json:"messages"`
				Statuses         []WhatsAppStatus  `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type WhatsAppContact struct {
	WaID    string `json:"wa_id"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
}

type WhatsAppMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
	Image struct {
		ID       string `json:"id"`
		Caption  string `json:"caption"`
		MimeType string `json:"mime_type"`
	} `json:"image,omitempty"`
	Video struct {
		ID       string `json:"id"`
		Caption  string `json:"caption"`
		MimeType string `json:"mime_type"`
	} `json:"video,omitempty"`
	Audio struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
	} `json:"audio,omitempty"`
	Document struct {
		ID       string `json:"id"`
		Caption  string `json:"caption"`
		Filename string `json:"filename"`
		MimeType string `json:"mime_type"`
	} `json:"document,omitempty"`
	Sticker *struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
	} `json:"sticker,omitempty"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name"`
		Address   string  `json:"address"`
	} `json:"location,omitempty"`
	Contacts []interface{} `json:"contacts,omitempty"`
	Interactive *struct {
		Type        string `json:"type"`
		ButtonReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"button_reply,omitempty"`
		ListReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"list_reply,omitempty"`
	} `json:"interactive,omitempty"`
	Button *struct {
		Text    string `json:"text"`
		Payload string `json:"payload"`
	} `json:"button,omitempty"`
	Reaction *struct {
		MessageID string `json:"message_id"`
		Emoji     string `json:"emoji"`
	} `json:"reaction,omitempty"`
	Context *struct {
		ID   string `json:"id"`
		From string `json:"from"`
	} `json:"context,omitempty"`
}

type WhatsAppStatus struct {
	ID          string `json:"id"`
	RecipientID string `json:"recipient_id"`
	Status      string `json:"status"`
	Timestamp   string `json:"timestamp"`
	Errors      []struct {
		Code    int    `json:"code"`
		Title   string `json:"title"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// Telegram types
type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

type TelegramMessage struct {
	MessageID int `json:"message_id"`
	From      *struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	} `json:"from,omitempty"`
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	Text     string `json:"text,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Photo    []struct {
		FileID   string `json:"file_id"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		FileSize int    `json:"file_size"`
	} `json:"photo,omitempty"`
	Document *struct {
		FileID   string `json:"file_id"`
		FileName string `json:"file_name"`
		MimeType string `json:"mime_type"`
		FileSize int    `json:"file_size"`
	} `json:"document,omitempty"`
	Voice *struct {
		FileID   string `json:"file_id"`
		Duration int    `json:"duration"`
		MimeType string `json:"mime_type"`
	} `json:"voice,omitempty"`
	Video *struct {
		FileID   string `json:"file_id"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Duration int    `json:"duration"`
		MimeType string `json:"mime_type"`
	} `json:"video,omitempty"`
	Location *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location,omitempty"`
}

// Generic webhook types
type GenericWebhookPayload struct {
	MessageID   string            `json:"message_id"`
	SenderID    string            `json:"sender_id"`
	SenderName  string            `json:"sender_name,omitempty"`
	ContentType string            `json:"content_type"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type StatusCallbackPayload struct {
	MessageID    string `json:"message_id"`
	ExternalID   string `json:"external_id,omitempty"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Ensure the handler implements error interface
var _ error = (*errors.AppError)(nil)
