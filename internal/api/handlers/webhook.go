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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
