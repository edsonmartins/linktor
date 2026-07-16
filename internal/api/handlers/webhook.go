package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/email"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/rcs"
	"github.com/msgfy/linktor/internal/adapters/slack"
	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/adapters/teams"
	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	appservice "github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/errors"
)

// WebhookHandler handles incoming webhooks from external channels
type WebhookHandler struct {
	channelRepo           repository.ChannelRepository
	producer              nats.Publisher
	templateSvc           *appservice.TemplateService
	requireWebhookSecrets bool
	teamsValidator        *teams.Validator
	// mediaStore re-hosts provider media that sits behind authenticated URLs
	// (e.g. Slack url_private) so external consumers can fetch it. May be nil,
	// in which case the original provider URL is passed through unchanged.
	mediaStore storage.Client
}

// NewWebhookHandler creates a new webhook handler. mediaStore may be nil to
// disable inbound media re-hosting.
func NewWebhookHandler(channelRepo repository.ChannelRepository, producer nats.Publisher, templateSvc *appservice.TemplateService, requireWebhookSecrets bool, mediaStore storage.Client) *WebhookHandler {
	return &WebhookHandler{
		channelRepo:           channelRepo,
		producer:              producer,
		templateSvc:           templateSvc,
		requireWebhookSecrets: requireWebhookSecrets,
		teamsValidator:        teams.NewValidator(nil),
		mediaStore:            mediaStore,
	}
}

func (h *WebhookHandler) publishInbound(ctx context.Context, inbound *nats.InboundMessage) error {
	if h.producer == nil {
		return fmt.Errorf("message queue unavailable")
	}
	return h.producer.PublishInbound(ctx, inbound)
}

func (h *WebhookHandler) publishStatusUpdate(ctx context.Context, status *nats.StatusUpdate) error {
	if h.producer == nil {
		return fmt.Errorf("message queue unavailable")
	}
	return h.producer.PublishStatusUpdate(ctx, status)
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

	var rawBody []byte

	// Verify signature if secret is configured
	if secret, ok := channel.Credentials["webhook_secret"]; ok && secret != "" {
		rawBody, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

		if !h.verifyWhatsAppSignature(rawBody, c.GetHeader("X-Hub-Signature-256"), secret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	} else {
		if h.requireWebhookSecrets {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
			return
		}
		rawBody, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))
	}

	// Parse webhook payload
	var payload WhatsAppWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var officialPayload whatsappofficial.WebhookPayload
	if err := json.Unmarshal(rawBody, &officialPayload); err == nil {
		h.processWhatsAppTemplateWebhooks(c.Request.Context(), &officialPayload)
		h.processWhatsAppChannelWebhooks(c.Request.Context(), channel, &officialPayload)
	}

	// Process messages
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field == "message_echoes" && channel.IsCoexistenceChannel() {
				h.updateCoexistenceLastEcho(c.Request.Context(), channel)
			}

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

func (h *WebhookHandler) updateCoexistenceLastEcho(ctx context.Context, channel *entity.Channel) {
	channel.UpdateLastEchoAt()
	if channel.CoexistenceStatus == entity.CoexistenceStatusWarning ||
		channel.CoexistenceStatus == entity.CoexistenceStatusDisconnected ||
		channel.CoexistenceStatus == entity.CoexistenceStatusPending {
		channel.CoexistenceStatus = entity.CoexistenceStatusActive
	}

	_ = h.channelRepo.Update(ctx, channel)
}

func (h *WebhookHandler) processWhatsAppChannelWebhooks(ctx context.Context, channel *entity.Channel, payload *whatsappofficial.WebhookPayload) {
	if channel == nil || channel.Type != entity.ChannelTypeWhatsAppOfficial {
		return
	}

	processor := whatsappofficial.NewWebhookProcessor(nil)
	qualityEvents := processor.ExtractPhoneQualityUpdates(payload)
	if len(qualityEvents) == 0 {
		return
	}

	if channel.Config == nil {
		channel.Config = make(map[string]string)
	}

	for _, event := range qualityEvents {
		if event == nil {
			continue
		}

		channel.Config["quality_rating_event"] = event.Event
		channel.Config["messaging_limit_tier"] = event.CurrentLimit
		if event.PhoneNumber != "" {
			channel.Config["phone_number"] = event.PhoneNumber
		}

		switch event.Event {
		case "FLAGGED":
			channel.Config["quality_rating"] = "RED"
		case "UNFLAGGED":
			if channel.Config["quality_rating"] == "" || channel.Config["quality_rating"] == "RED" {
				channel.Config["quality_rating"] = "GREEN"
			}
		}
	}

	_ = h.channelRepo.Update(ctx, channel)
}

// TelegramWebhook handles Telegram Bot API webhooks
func (h *WebhookHandler) TelegramWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	secretToken := channel.Credentials["secret_token"]
	if secretToken == "" {
		secretToken = channel.Credentials["webhook_secret"]
	}
	if secretToken != "" {
		if c.GetHeader("X-Telegram-Bot-Api-Secret-Token") != secretToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid secret token"})
			return
		}
	} else if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return
	}

	var update TelegramUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	switch {
	case update.Message != nil:
		if err := h.processTelegramMessage(c.Request.Context(), channel, update.Message); err != nil {
			// Log error but continue
		}
	case update.EditedMessage != nil:
		// Treat an edited message like a fresh inbound so downstream sees the
		// updated content; flag it so consumers can dedupe/annotate if needed.
		if err := h.processTelegramMessage(c.Request.Context(), channel, update.EditedMessage); err != nil {
			// Log error but continue
		}
	case update.CallbackQuery != nil:
		// Inline-keyboard button taps arrive as callback queries, not messages;
		// route them through the same inbound pipeline carrying callback_data.
		if err := h.processTelegramCallback(c.Request.Context(), channel, update.CallbackQuery); err != nil {
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

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}
	if !h.verifySharedSecretWebhook(c, body, channel.Credentials["webhook_secret"]) {
		return
	}

	var payload GenericWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if payload.Metadata == nil {
		payload.Metadata = make(map[string]string)
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

	if err := h.publishInbound(c.Request.Context(), inbound); err != nil {
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

	if !h.twilioSignatureOK(c, channel, body) {
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

		if err := h.publishInbound(c.Request.Context(), inbound); err != nil {
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
		h.publishStatusUpdate(c.Request.Context(), statusUpdate)

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
	if appSecret == "" && h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return
	}

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

	// Button-click postbacks (Get Started, persistent menu, template buttons) are
	// delivered as their own messaging events, not as messages, so ExtractMessages
	// does not surface them. Ingest them as inbound events so the flow engine can
	// react to the click.
	for _, pb := range facebook.ExtractPostbacks(payload) {
		if err := h.processFacebookPostback(c.Request.Context(), channel, pb); err != nil {
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
	if appSecret == "" && h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return
	}

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

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}
	if !h.verifySharedSecretWebhook(c, body, channel.Credentials["webhook_secret"]) {
		return
	}

	var payload StatusCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
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

	if err := h.publishStatusUpdate(c.Request.Context(), status); err != nil {
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

func (h *WebhookHandler) verifyWhatsAppSignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

func (h *WebhookHandler) verifySharedSecretWebhook(c *gin.Context, body []byte, secret string) bool {
	if secret == "" {
		if h.requireWebhookSecrets {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
			return false
		}
		return true
	}

	signature := c.GetHeader("X-Linktor-Signature")
	if signature == "" {
		signature = c.GetHeader("X-Hub-Signature-256")
	}
	if !h.verifyWhatsAppSignature(body, signature, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return false
	}
	return true
}

func requestURL(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	return scheme + "://" + c.Request.Host + c.Request.URL.RequestURI()
}

func firstValues(values url.Values) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if len(value) > 0 {
			result[key] = value[0]
		}
	}
	return result
}

func (h *WebhookHandler) verifyEmailWebhook(c *gin.Context, provider email.Provider, body []byte, headers map[string]string, credentials map[string]string) bool {
	switch provider {
	case email.ProviderMailgun:
		apiKey := firstNonEmpty(credentials["mailgun_webhook_signing_key"], credentials["mailgun_api_key"], credentials["api_key"], credentials["webhook_secret"])
		if apiKey == "" {
			return h.allowMissingWebhookSecret(c)
		}

		token, timestamp, signature := mailgunSignatureParts(body, headers)
		if token == "" || timestamp == "" || signature == "" || !email.ValidateMailgunWebhook(apiKey, token, timestamp, signature) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return false
		}
		return true

	case email.ProviderPostmark:
		password := firstNonEmpty(credentials["postmark_webhook_password"], credentials["webhook_password"], credentials["webhook_secret"])
		if password == "" {
			return h.allowMissingWebhookSecret(c)
		}
		if !validatePostmarkAuthorization(password, headers["Authorization"]) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization"})
			return false
		}
		return true

	case email.ProviderSendGrid, email.ProviderSES:
		secret := credentials["webhook_secret"]
		if secret == "" {
			return h.allowMissingWebhookSecret(c)
		}
		return h.verifySharedSecretWebhook(c, body, secret)

	default:
		return h.allowMissingWebhookSecret(c)
	}
}

func (h *WebhookHandler) allowMissingWebhookSecret(c *gin.Context) bool {
	if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return false
	}
	return true
}

func mailgunSignatureParts(body []byte, headers map[string]string) (token, timestamp, signature string) {
	values, _ := url.ParseQuery(string(body))
	token = values.Get("token")
	timestamp = values.Get("timestamp")
	signature = values.Get("signature")
	if token != "" || timestamp != "" || signature != "" {
		return token, timestamp, signature
	}

	var payload struct {
		Signature struct {
			Token     string `json:"token"`
			Timestamp string `json:"timestamp"`
			Signature string `json:"signature"`
		} `json:"signature"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		token = payload.Signature.Token
		timestamp = payload.Signature.Timestamp
		signature = payload.Signature.Signature
	}
	if token == "" {
		token = firstNonEmpty(headers["X-Mailgun-Token"], headers["X-Mailgun-Webhook-Token"])
	}
	if timestamp == "" {
		timestamp = firstNonEmpty(headers["X-Mailgun-Timestamp"], headers["X-Mailgun-Webhook-Timestamp"])
	}
	if signature == "" {
		signature = firstNonEmpty(headers["X-Mailgun-Signature"], headers["X-Mailgun-Webhook-Signature"])
	}
	return token, timestamp, signature
}

func validatePostmarkAuthorization(password, authorization string) bool {
	if authorization == "" {
		return false
	}
	if authorization == password {
		return true
	}
	if strings.HasPrefix(authorization, "Bearer ") {
		return strings.TrimPrefix(authorization, "Bearer ") == password
	}
	if strings.HasPrefix(authorization, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorization, "Basic "))
		if err != nil {
			return false
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return false
		}
		return parts[1] == password
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (h *WebhookHandler) processWhatsAppTemplateWebhooks(ctx context.Context, payload *whatsappofficial.WebhookPayload) {
	if h.templateSvc == nil || payload == nil {
		return
	}

	processor := whatsappofficial.NewWebhookProcessor(&whatsappofficial.Config{})

	for _, event := range processor.ExtractTemplateStatusUpdates(payload) {
		_ = h.templateSvc.ProcessTemplateStatusWebhook(ctx, &appservice.TemplateStatusEvent{
			TemplateID:   event.TemplateID,
			TemplateName: event.TemplateName,
			Language:     event.Language,
			Event:        event.Event,
			Reason:       event.Reason,
		})
	}

	for _, event := range processor.ExtractTemplateQualityUpdates(payload) {
		_ = h.templateSvc.ProcessTemplateQualityWebhook(ctx, &appservice.TemplateQualityEvent{
			TemplateID:      event.TemplateID,
			TemplateName:    event.TemplateName,
			Language:        event.Language,
			PreviousQuality: event.PreviousQuality,
			NewQuality:      event.NewQuality,
		})
	}

	for _, event := range processor.ExtractTemplateCategoryUpdates(payload) {
		_ = h.templateSvc.ProcessTemplateCategoryWebhook(ctx, &appservice.TemplateCategoryEvent{
			TemplateID:       event.TemplateID,
			TemplateName:     event.TemplateName,
			Language:         event.Language,
			PreviousCategory: event.PreviousCategory,
			NewCategory:      event.NewCategory,
		})
	}
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

	// WhatsApp Cloud API delivers media as an opaque media_id, not a fetchable
	// URL. Resolve+download each with the channel token and re-host to the media
	// store so the chat can render it (mirrors the Teams/Slack re-host path).
	for i := range attachments {
		if mid := attachments[i].Metadata["media_id"]; mid != "" {
			if url, ok := h.rehostWhatsAppMedia(ctx, channel, mid, attachments[i].MimeType); ok {
				attachments[i].URL = url
			}
		}
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

	return h.publishInbound(ctx, inbound)
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

	h.publishStatusUpdate(ctx, update)
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
		senderID = strconv.FormatInt(msg.From.ID, 10)
		senderName = msg.From.FirstName
		if msg.From.LastName != "" {
			senderName += " " + msg.From.LastName
		}
	}

	metadata := map[string]string{
		"sender_id":   senderID,
		"sender_name": senderName,
		"chat_id":     strconv.FormatInt(msg.Chat.ID, 10),
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "telegram",
		ExternalID:  strconv.FormatInt(int64(msg.MessageID), 10),
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.publishInbound(ctx, inbound)
}

// processTelegramCallback normalizes an inline-keyboard button tap
// (update.callback_query) into an inbound event carrying the button's
// callback_data. Without this, taps on interactive keyboards would never reach
// the system. Sender/chat IDs are formatted as decimal strings (see the int64
// handling in processTelegramMessage) so contact/status correlation stays intact.
func (h *WebhookHandler) processTelegramCallback(ctx context.Context, channel *entity.Channel, cb *TelegramCallbackQuery) error {
	senderID := ""
	senderName := ""
	if cb.From != nil {
		senderID = strconv.FormatInt(cb.From.ID, 10)
		senderName = cb.From.FirstName
		if cb.From.LastName != "" {
			senderName += " " + cb.From.LastName
		}
	}

	metadata := map[string]string{
		"sender_id":         senderID,
		"sender_name":       senderName,
		"callback_data":     cb.Data,
		"callback_query_id": cb.ID,
		"is_callback":       "true",
	}
	// The originating message carries the chat the keyboard lives in.
	if cb.Message != nil {
		metadata["chat_id"] = strconv.FormatInt(cb.Message.Chat.ID, 10)
		metadata["callback_message_id"] = strconv.FormatInt(int64(cb.Message.MessageID), 10)
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "telegram",
		ExternalID:  cb.ID,
		ContentType: "interactive",
		Content:     cb.Data,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}

	return h.publishInbound(ctx, inbound)
}

// TeamsWebhook handles Microsoft Teams Bot Framework Activity webhooks. It
// validates the Bot Connector JWT, normalizes message Activities to the inbound
// pipeline, and persists the conversation reference (service_url) needed for
// proactive outbound replies.
func (h *WebhookHandler) TeamsWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Validate the Bot Connector JWT. The audience is the channel's app_id; the
	// same app may serve many tenants (multi-tenant ownership model).
	appID := channel.Credentials[teams.CredAppID]
	if appID != "" {
		if err := h.teamsValidator.Validate(c.Request.Context(), c.GetHeader("Authorization"), appID); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid bot connector token"})
			return
		}
	} else if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "teams app_id not configured"})
		return
	}

	var activity teams.Activity
	if err := json.Unmarshal(rawBody, &activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity payload"})
		return
	}

	if activity.Type == "message" {
		if err := h.processTeamsActivity(c.Request.Context(), channel, &activity); err != nil {
			// Log and continue; respond 200 so the Bot Connector does not retry storm.
			_ = err
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TeamsWebhookShared handles the multi-tenant Teams ownership model: one Azure
// bot app (one Bot Framework messaging endpoint) serves many tenants. There is
// no channelId in the URL, so the channel is resolved from the request itself —
// the token's audience (the bot app id) plus the Activity's AAD tenant — and the
// token is then fully validated against the resolved channel's app id.
func (h *WebhookHandler) TeamsWebhookShared(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var activity teams.Activity
	if err := json.Unmarshal(rawBody, &activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity payload"})
		return
	}

	// Read the audience UNVERIFIED only to find the channel; the token is fully
	// validated (JWKS) against that channel's app id below before we trust it.
	appID := teams.UnverifiedAudience(c.GetHeader("Authorization"))
	if appID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bot connector token"})
		return
	}

	candidates, err := h.channelRepo.FindAllByType(c.Request.Context(), teams.ChannelType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "channel lookup failed"})
		return
	}
	// SECURITY (trust boundary of the shared-app model): the Bot Connector JWT
	// only proves issuer + signature + audience (the shared app_id); it does NOT
	// cryptographically bind the AAD org tenant, which is read from the request
	// body (channelData.tenant.id). A holder of any valid token for the shared
	// app can therefore claim another tenant's id. This is inherent to running a
	// single multi-tenant bot registration across tenants — tenants that share
	// one app registration are within ONE trust boundary. For strict per-tenant
	// isolation, use the per-channel endpoint (/webhooks/teams/:channelId) with a
	// distinct app registration per tenant. The serviceUrl allowlist in
	// processTeamsActivity prevents this from escalating into bearer-token
	// exfiltration regardless.
	channel := teams.MatchSharedChannel(candidates, appID, activity.AADTenant())
	if channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no teams channel for this bot/tenant"})
		return
	}

	if err := h.teamsValidator.Validate(c.Request.Context(), c.GetHeader("Authorization"), channel.Credentials[teams.CredAppID]); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid bot connector token"})
		return
	}

	if activity.Type == "message" {
		if err := h.processTeamsActivity(c.Request.Context(), channel, &activity); err != nil {
			_ = err // log-and-continue; ack so the Bot Connector does not retry storm
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) processTeamsActivity(ctx context.Context, channel *entity.Channel, activity *teams.Activity) error {
	// Persist the service URL so proactive (out-of-turn) outbound replies can
	// reach the Bot Connector for this channel. serviceUrl is the destination for
	// outbound calls that carry the bot's AAD bearer token, so we only trust (and
	// persist) values on known Bot Framework / Teams connector hosts — never an
	// arbitrary URL from the request body (which would exfiltrate the token).
	trustedServiceURL := teams.IsTrustedServiceURL(activity.ServiceURL)
	if trustedServiceURL && channel.Credentials[teams.CredServiceURL] != activity.ServiceURL {
		if channel.Credentials == nil {
			channel.Credentials = map[string]string{}
		}
		channel.Credentials[teams.CredServiceURL] = activity.ServiceURL
		_ = h.channelRepo.Update(ctx, channel)
	}

	contentType := "text"
	content := activity.Text
	var attachments []nats.AttachmentData
	for _, att := range activity.Attachments {
		if att.ContentURL == "" {
			continue
		}
		attType := teamsAttachmentType(att.ContentType)
		if attType != "text" {
			contentType = attType
		}
		// Bot Connector attachment URLs require the bot's bearer token, so an
		// external consumer can't fetch them directly. Re-host to our media store
		// when configured; fall back to the provider URL otherwise.
		url := att.ContentURL
		if rehosted, ok := h.rehostTeamsAttachment(ctx, channel, att); ok {
			url = rehosted
		}
		attachments = append(attachments, nats.AttachmentData{
			Type:     attType,
			URL:      url,
			Filename: att.Name,
			MimeType: att.ContentType,
		})
	}

	// The Bot Framework conversation id is the stable per-conversation address
	// reused for outbound; it doubles as the contact surrogate key for Teams 1:1.
	metadata := map[string]string{
		"sender_id":   activity.Conversation.ID,
		"sender_name": activity.From.Name,
		"aad_user_id": activity.From.AadObjectID,
	}
	// Only surface a serviceUrl we've validated as a trusted connector host.
	if trustedServiceURL {
		metadata["service_url"] = activity.ServiceURL
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: teams.ChannelType,
		ExternalID:  activity.ID,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.publishInbound(ctx, inbound)
}

// rehostTeamsAttachment downloads a Teams attachment (authenticated with the
// channel's bot AAD token) and stores it in the media store, returning the
// public URL. Best effort: returns ("", false) when no store is configured or
// any step fails, so the caller falls back to the original provider URL.
func (h *WebhookHandler) rehostTeamsAttachment(ctx context.Context, channel *entity.Channel, att teams.Attachment) (string, bool) {
	if h.mediaStore == nil || att.ContentURL == "" {
		return "", false
	}

	dlCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	data, ctype, err := teams.NewClientFromCredentials(channel.Credentials).DownloadAttachment(dlCtx, att.ContentURL)
	if err != nil {
		return "", false
	}
	if ctype == "" {
		ctype = att.ContentType
	}

	url, err := h.mediaStore.Upload(dlCtx, inboundMediaKey(teams.ChannelType, channel.ID, att.Name), data, ctype)
	if err != nil {
		return "", false
	}
	return url, true
}

// rehostWhatsAppMedia resolves a WhatsApp Cloud API media_id to its temporary
// Graph URL, downloads the bytes with the channel access token, and stores them
// in the media store, returning a durable public URL. Best effort: returns
// ("", false) when no store/token is configured or any step fails, so the
// caller keeps the original (unusable) media_id reference.
func (h *WebhookHandler) rehostWhatsAppMedia(ctx context.Context, channel *entity.Channel, mediaID, mimeType string) (string, bool) {
	if h.mediaStore == nil || mediaID == "" {
		return "", false
	}
	token := channel.Credentials["access_token"]
	if token == "" {
		token = channel.Config["access_token"]
	}
	if token == "" {
		return "", false
	}

	dlCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	client := whatsappofficial.NewClient(&whatsappofficial.Config{AccessToken: token})
	info, err := client.GetMediaInfo(dlCtx, mediaID)
	if err != nil || info == nil || info.URL == "" {
		return "", false
	}
	data, ctype, err := client.DownloadMedia(dlCtx, info.URL)
	if err != nil {
		return "", false
	}
	if ctype == "" {
		ctype = mimeType
	}
	if ctype == "" {
		ctype = info.MimeType
	}

	url, err := h.mediaStore.Upload(dlCtx, inboundMediaKey(string(channel.Type), channel.ID, mediaID), data, ctype)
	if err != nil {
		return "", false
	}
	return url, true
}

// teamsAttachmentType maps a Bot Framework attachment content type to the
// canonical inbound content type.
func teamsAttachmentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case contentType == "":
		return "text"
	default:
		return "document"
	}
}

// SlackWebhook handles Slack Events API webhooks: the url_verification
// handshake, X-Slack-Signature verification, and normalizing inbound message
// events (filtering the bot's own echoes).
func (h *WebhookHandler) SlackWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Verify the request signature over the raw body.
	signingSecret := channel.Credentials[slack.CredSigningSecret]
	if signingSecret != "" {
		if !slack.VerifySignature(
			signingSecret,
			c.GetHeader("X-Slack-Signature"),
			c.GetHeader("X-Slack-Request-Timestamp"),
			rawBody,
			time.Now(),
		) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	} else if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "slack signing_secret not configured"})
		return
	}

	var env slack.EventEnvelope
	if err := json.Unmarshal(rawBody, &env); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Respond to the one-time URL verification challenge.
	if env.Type == "url_verification" {
		c.String(http.StatusOK, env.Challenge)
		return
	}

	if env.Type == "event_callback" && env.Event != nil && env.Event.Type == "message" {
		if !env.Event.IsBotEcho(channel.Credentials[slack.CredBotUserID]) {
			if err := h.processSlackMessage(c.Request.Context(), channel, env.Event); err != nil {
				_ = err // log-and-continue; ack so Slack does not retry storm
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) processSlackMessage(ctx context.Context, channel *entity.Channel, event *slack.InnerEvent) error {
	contentType := "text"
	content := event.Text
	var attachments []nats.AttachmentData
	for _, f := range event.Files {
		if f.URLPrivate == "" {
			continue
		}
		attType := slackFileType(f.Mimetype)
		if attType != "text" {
			contentType = attType
		}
		// Slack url_private requires the bot token to fetch, so external
		// consumers can't use it directly. Re-host to our media store when
		// configured; fall back to the provider URL otherwise.
		url := f.URLPrivate
		if rehosted, ok := h.rehostSlackFile(ctx, channel, f); ok {
			url = rehosted
		}
		attachments = append(attachments, nats.AttachmentData{
			Type:      attType,
			URL:       url,
			Filename:  f.Name,
			MimeType:  f.Mimetype,
			SizeBytes: f.Size,
		})
	}

	// The Slack channel id (a stable IM channel for DMs) is the outbound reply
	// address and doubles as the contact surrogate key.
	metadata := map[string]string{
		"sender_id":       event.Channel,
		"slack_user_id":   event.User,
		"slack_channel":   event.Channel,
		"slack_thread_ts": event.ThreadTS,
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: slack.ChannelType,
		ExternalID:  event.TS,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.publishInbound(ctx, inbound)
}

// rehostSlackFile downloads a Slack file (authenticated with the channel's bot
// token) and stores it in the media store, returning the public URL. Best
// effort: returns ("", false) when no store is configured or any step fails, so
// the caller falls back to the original provider URL.
func (h *WebhookHandler) rehostSlackFile(ctx context.Context, channel *entity.Channel, f slack.File) (string, bool) {
	if h.mediaStore == nil {
		return "", false
	}
	botToken := channel.Credentials[slack.CredBotToken]
	if botToken == "" {
		return "", false
	}

	dlCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	data, ctype, err := slack.NewClient(botToken).DownloadFile(dlCtx, f.URLPrivate)
	if err != nil {
		return "", false
	}
	if ctype == "" {
		ctype = f.Mimetype
	}

	url, err := h.mediaStore.Upload(dlCtx, inboundMediaKey(slack.ChannelType, channel.ID, f.Name), data, ctype)
	if err != nil {
		return "", false
	}
	return url, true
}

// inboundMediaKey builds a stable, collision-free storage key for re-hosted
// inbound media.
func inboundMediaKey(channelType, channelID, filename string) string {
	name := filename
	if name == "" {
		name = "file"
	}
	return fmt.Sprintf("inbound/%s/%s/%s-%s", channelType, channelID, uuid.New().String(), name)
}

func slackFileType(mimetype string) string {
	switch {
	case strings.HasPrefix(mimetype, "image/"):
		return "image"
	case strings.HasPrefix(mimetype, "video/"):
		return "video"
	case strings.HasPrefix(mimetype, "audio/"):
		return "audio"
	case mimetype == "":
		return "text"
	default:
		return "document"
	}
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

	return h.publishInbound(ctx, inbound)
}

// processFacebookPostback turns a button-click postback into an inbound message
// whose content is the postback payload, so bots/flows react to the click. The
// HTTP-level webhook dedup middleware guards against Meta retries, so no stable
// external id is needed.
func (h *WebhookHandler) processFacebookPostback(ctx context.Context, channel *entity.Channel, pb *facebook.Postback) error {
	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "facebook",
		ContentType: "text",
		Content:     pb.Payload,
		Metadata: map[string]string{
			"sender_id":        pb.SenderID,
			"page_id":          pb.PageID,
			"recipient_id":     pb.RecipientID,
			"event_type":       "postback",
			"postback_payload": pb.Payload,
			"postback_title":   pb.Title,
		},
		Timestamp: pb.Timestamp,
	}
	return h.publishInbound(ctx, inbound)
}

func (h *WebhookHandler) processFacebookDeliveryStatus(ctx context.Context, channel *entity.Channel, status *facebook.DeliveryStatus) {
	for _, msgID := range status.MessageIDs {
		update := &nats.StatusUpdate{
			ExternalID:  msgID,
			ChannelType: "facebook",
			Status:      "delivered",
			Timestamp:   status.Watermark,
		}
		h.publishStatusUpdate(ctx, update)
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

	return h.publishInbound(ctx, inbound)
}

// Payload types

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

	// Create RCS client for validation. Default the provider to Zenvia when the
	// channel omits it — mirrors Adapter.Initialize. Without this, an empty
	// provider fails Config.Validate and every inbound webhook returns HTTP 500.
	provider := rcs.Provider(channel.Config["provider"])
	if provider == "" {
		provider = rcs.ProviderZenvia
	}
	rcsConfig := &rcs.Config{
		Provider:      provider,
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
	} else if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return
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
			// Surface a publish failure as 500 so Zenvia retries the delivery,
			// rather than acking and silently losing the inbound message.
			if err := h.processRCSMessage(c.Request.Context(), channel, payload.Message); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process message"})
				return
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

		return h.publishInbound(ctx, inboundMsg)
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
		h.publishStatusUpdate(ctx, &nats.StatusUpdate{
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
	if !h.verifyEmailWebhook(c, provider, body, headers, channel.Credentials) {
		return
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

		ad := nats.AttachmentData{
			Type:      attType,
			URL:       att.URL,
			Filename:  att.Filename,
			MimeType:  att.ContentType,
			SizeBytes: att.Size,
		}
		// Inbound mail carries the file inline (no URL). Hand the bytes to the
		// generic re-host step as base64 so it gets a durable object-store URL;
		// without this the attachment reaches the chat with an empty URL.
		if ad.URL == "" && len(att.Content) > 0 {
			ad.Metadata = map[string]string{
				"data":          base64.StdEncoding.EncodeToString(att.Content),
				"data_encoding": "base64",
			}
			if ad.SizeBytes == 0 {
				ad.SizeBytes = int64(len(att.Content))
			}
		}
		attachments = append(attachments, ad)
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

		return h.publishInbound(ctx, inboundMsg)
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
		h.publishStatusUpdate(ctx, &nats.StatusUpdate{
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
	Contacts    []interface{} `json:"contacts,omitempty"`
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
	UpdateID      int                    `json:"update_id"`
	Message       *TelegramMessage       `json:"message,omitempty"`
	EditedMessage *TelegramMessage       `json:"edited_message,omitempty"`
	CallbackQuery *TelegramCallbackQuery `json:"callback_query,omitempty"`
}

// TelegramCallbackQuery is an inline-keyboard button tap. Data holds the
// button's callback_data; Message is the message the keyboard was attached to.
type TelegramCallbackQuery struct {
	ID   string `json:"id"`
	From *struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	} `json:"from,omitempty"`
	Message *TelegramMessage `json:"message,omitempty"`
	Data    string           `json:"data,omitempty"`
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
	Text    string `json:"text,omitempty"`
	Caption string `json:"caption,omitempty"`
	Photo   []struct {
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
