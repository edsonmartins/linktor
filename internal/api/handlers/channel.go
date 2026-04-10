package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/rcs"
	"github.com/msgfy/linktor/internal/adapters/sms"
	"github.com/msgfy/linktor/internal/adapters/telegram"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// ChannelHandler handles channel endpoints
type ChannelHandler struct {
	channelService *service.ChannelService
	producer       nats.Publisher
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(channelService *service.ChannelService, producer nats.Publisher) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		producer:       producer,
	}
}

// CreateChannelRequest represents a create channel request
type CreateChannelRequest struct {
	Type        string            `json:"type" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	Identifier  string            `json:"identifier"`
	Config      map[string]string `json:"config"`
	Credentials map[string]string `json:"credentials"`
}

type TestChannelRequest struct {
	Type        string            `json:"type"`
	Config      map[string]string `json:"config"`
	Credentials map[string]string `json:"credentials"`
}

// List godoc
// @Summary      List channels
// @Description  Returns all channels for the current tenant
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} Response{data=[]entity.Channel}
// @Failure      401 {object} Response
// @Router       /channels [get]
func (h *ChannelHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	channels, err := h.channelService.List(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channels)
}

// Create godoc
// @Summary      Create channel
// @Description  Create a new channel for the current tenant
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateChannelRequest true "Channel data"
// @Success      201 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /channels [post]
func (h *ChannelHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateChannelInput{
		TenantID:    tenantID,
		Type:        req.Type,
		Name:        req.Name,
		Identifier:  req.Identifier,
		Config:      req.Config,
		Credentials: req.Credentials,
	}

	channel, err := h.channelService.Create(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, channel)
}

// TestConnection validates channel credentials without creating a channel.
func (h *ChannelHandler) TestConnection(c *gin.Context) {
	h.testConnection(c, "")
}

func (h *ChannelHandler) TestWhatsAppConnection(c *gin.Context) {
	h.testConnection(c, "whatsapp_official")
}

func (h *ChannelHandler) TestTelegramConnection(c *gin.Context) {
	h.testConnection(c, "telegram")
}

func (h *ChannelHandler) TestTwilioConnection(c *gin.Context) {
	h.testConnection(c, "sms")
}

func (h *ChannelHandler) TestFacebookConnection(c *gin.Context) {
	h.testConnection(c, "facebook")
}

func (h *ChannelHandler) TestInstagramConnection(c *gin.Context) {
	h.testConnection(c, "instagram")
}

func (h *ChannelHandler) testConnection(c *gin.Context, forcedType string) {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	channelType := forcedType
	if channelType == "" {
		channelType = stringFromMap(raw, "type")
	}
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	if channelType == "" {
		RespondValidationError(c, "type is required", nil)
		return
	}

	config := flattenChannelTestConfig(raw)
	if err := validateChannelTestConfig(channelType, config); err != nil {
		RespondValidationError(c, err.Error(), nil)
		return
	}

	RespondSuccess(c, gin.H{
		"status":  "ok",
		"type":    channelType,
		"valid":   true,
		"message": "configuration accepted",
	})
}

// Get godoc
// @Summary      Get channel
// @Description  Returns a channel by ID
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id} [get]
func (h *ChannelHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	channel, err := h.channelService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channel)
}

// Update godoc
// @Summary      Update channel
// @Description  Update a channel's configuration
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body CreateChannelRequest true "Channel update data"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id} [put]
func (h *ChannelHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateChannelInput{
		Name:        &req.Name,
		Identifier:  &req.Identifier,
		Config:      req.Config,
		Credentials: req.Credentials,
	}

	channel, err := h.channelService.Update(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channel)
}

// Delete godoc
// @Summary      Delete channel
// @Description  Delete a channel by ID
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      204 "No Content"
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id} [delete]
func (h *ChannelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	if err := h.channelService.Delete(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// Connect godoc
// @Summary      Connect channel
// @Description  Connect a channel to start receiving messages
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      200 {object} Response{data=object}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/connect [post]
func (h *ChannelHandler) Connect(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	result, err := h.channelService.Connect(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

// Disconnect godoc
// @Summary      Disconnect channel
// @Description  Disconnect a channel to stop receiving messages
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Success      200 {object} Response{data=object{message=string}}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/disconnect [post]
func (h *ChannelHandler) Disconnect(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	if err := h.channelService.Disconnect(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Channel disconnected"})
}

// UpdateStatusRequest represents a request to update channel status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// UpdateStatus godoc
// @Summary      Update channel status
// @Description  Update a channel's status (active/inactive)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body UpdateStatusRequest true "New status"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/status [put]
func (h *ChannelHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid status. Must be 'active' or 'inactive'", nil)
		return
	}

	channel, err := h.channelService.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channel)
}

// UpdateEnabled godoc
// @Summary      Update channel enabled state
// @Description  Enable or disable a channel (system-level, independent of connection status)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body UpdateEnabledRequest true "Enabled state"
// @Success      200 {object} Response{data=entity.Channel}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/enabled [put]
func (h *ChannelHandler) UpdateEnabled(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req UpdateEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	channel, err := h.channelService.UpdateEnabled(c.Request.Context(), id, req.Enabled)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, channel)
}

// UpdateEnabledRequest represents a request to enable/disable a channel
type UpdateEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// PairCodeRequest represents a request for WhatsApp pair code
type PairCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

// RequestPairCode godoc
// @Summary      Request pair code
// @Description  Request a pair code for WhatsApp authentication (alternative to QR code)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        request body PairCodeRequest true "Phone number"
// @Success      200 {object} Response{data=service.ConnectResult}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/pair [post]
func (h *ChannelHandler) RequestPairCode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	var req PairCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "phone_number is required", nil)
		return
	}

	result, err := h.channelService.RequestPairCode(c.Request.Context(), id, req.PhoneNumber)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

func flattenChannelTestConfig(raw map[string]interface{}) map[string]string {
	config := make(map[string]string)

	for key, value := range raw {
		switch key {
		case "type", "config", "credentials":
			continue
		default:
			if str, ok := value.(string); ok {
				config[normalizeConfigKey(key)] = str
			}
		}
	}

	mergeStringMap(config, raw["config"])
	mergeStringMap(config, raw["credentials"])

	return config
}

func mergeStringMap(dst map[string]string, value interface{}) {
	values, ok := value.(map[string]interface{})
	if !ok {
		return
	}

	for key, rawValue := range values {
		if str, ok := rawValue.(string); ok {
			dst[normalizeConfigKey(key)] = str
		}
	}
}

func normalizeConfigKey(key string) string {
	replacements := map[string]string{
		"accessToken":         "access_token",
		"phoneNumberId":       "phone_number_id",
		"phoneNumberID":       "phone_number_id",
		"apiVersion":          "api_version",
		"botToken":            "bot_token",
		"accountSid":          "account_sid",
		"accountSID":          "account_sid",
		"authToken":           "auth_token",
		"apiKeySid":           "api_key_sid",
		"apiKeySID":           "api_key_sid",
		"apiKeySecret":        "api_key_secret",
		"messagingServiceSid": "messaging_service_sid",
		"messagingServiceSID": "messaging_service_sid",
		"pageId":              "page_id",
		"pageID":              "page_id",
		"pageAccessToken":     "page_access_token",
		"instagramId":         "instagram_id",
		"instagramID":         "instagram_id",
		"appId":               "app_id",
		"appID":               "app_id",
		"appSecret":           "app_secret",
		"verifyToken":         "verify_token",
		"agentId":             "agent_id",
		"agentID":             "agent_id",
		"apiKey":              "api_key",
		"apiSecret":           "api_secret",
		"baseUrl":             "base_url",
		"baseURL":             "base_url",
		"senderId":            "sender_id",
		"senderID":            "sender_id",
		"webhookUrl":          "webhook_url",
		"webhookURL":          "webhook_url",
		"webhookSecret":       "webhook_secret",
		"statusCallbackUrl":   "status_callback_url",
		"statusCallbackURL":   "status_callback_url",
		"userAccessToken":     "user_access_token",
	}
	if replacement, ok := replacements[key]; ok {
		return replacement
	}
	return key
}

func stringFromMap(values map[string]interface{}, key string) string {
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return value
}

func validateChannelTestConfig(channelType string, config map[string]string) error {
	switch channelType {
	case "whatsapp", "whatsapp_official":
		if strings.TrimSpace(config["access_token"]) == "" {
			return fmt.Errorf("access_token is required")
		}
		if strings.TrimSpace(config["phone_number_id"]) == "" {
			return fmt.Errorf("phone_number_id is required")
		}
		return nil
	case "telegram":
		return telegram.NewAdapter().Initialize(config)
	case "sms", "twilio":
		return sms.NewAdapter().Initialize(config)
	case "facebook":
		cfg := &facebook.FacebookConfig{
			PageID:          config["page_id"],
			PageAccessToken: config["page_access_token"],
		}
		return cfg.Validate()
	case "instagram":
		cfg := &instagram.InstagramConfig{
			InstagramID:     config["instagram_id"],
			AccessToken:     config["access_token"],
			PageAccessToken: config["page_access_token"],
		}
		return cfg.Validate()
	case "rcs":
		cfg := &rcs.Config{
			Provider: rcs.Provider(config["provider"]),
			AgentID:  config["agent_id"],
			APIKey:   config["api_key"],
		}
		if cfg.Provider == "" {
			cfg.Provider = rcs.ProviderZenvia
		}
		return cfg.Validate()
	default:
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// WhatsAppWebhook handles WhatsApp webhooks
func (h *ChannelHandler) WhatsAppWebhook(c *gin.Context) {
	channelID := c.Param("channelId")
	// TODO: Implement WhatsApp webhook handling
	RespondSuccess(c, gin.H{"channel_id": channelID, "status": "received"})
}

// WhatsAppVerify handles WhatsApp webhook verification
func (h *ChannelHandler) WhatsAppVerify(c *gin.Context) {
	// TODO: Implement WhatsApp webhook verification
	challenge := c.Query("hub.challenge")
	c.String(200, challenge)
}

// TelegramWebhook handles Telegram webhooks
func (h *ChannelHandler) TelegramWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelService.GetByID(c.Request.Context(), channelID)
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

	// Parse webhook
	update, err := telegram.ParseWebhook(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Extract message
	incoming := telegram.ExtractIncomingMessage(update)
	if incoming == nil {
		// Not a message we handle (e.g., channel post, group message)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Build metadata
	metadata := map[string]string{
		"from_user_id": fmt.Sprintf("%d", incoming.FromUserID),
		"username":     incoming.FromUsername,
		"first_name":   incoming.FromFirstName,
		"last_name":    incoming.FromLastName,
		"chat_id":      fmt.Sprintf("%d", incoming.ChatID),
	}

	// Determine content type
	contentType := "text"
	content := incoming.Text
	var attachments []nats.AttachmentData

	switch incoming.MessageType {
	case telegram.MessageTypePhoto:
		contentType = "image"
		content = incoming.Caption
		if incoming.MediaFileID != "" {
			attachments = append(attachments, nats.AttachmentData{
				Type: "image",
				URL:  incoming.MediaFileID,
				Metadata: map[string]string{
					"file_id": incoming.MediaFileID,
				},
			})
		}
	case telegram.MessageTypeVideo:
		contentType = "video"
		content = incoming.Caption
		if incoming.MediaFileID != "" {
			attachments = append(attachments, nats.AttachmentData{
				Type:     "video",
				URL:      incoming.MediaFileID,
				MimeType: incoming.MediaMimeType,
				Metadata: map[string]string{
					"file_id": incoming.MediaFileID,
				},
			})
		}
	case telegram.MessageTypeAudio, telegram.MessageTypeVoice:
		contentType = "audio"
		if incoming.MediaFileID != "" {
			attachments = append(attachments, nats.AttachmentData{
				Type:     "audio",
				URL:      incoming.MediaFileID,
				MimeType: incoming.MediaMimeType,
				Metadata: map[string]string{
					"file_id": incoming.MediaFileID,
				},
			})
		}
	case telegram.MessageTypeDocument:
		contentType = "document"
		content = incoming.Caption
		if incoming.MediaFileID != "" {
			attachments = append(attachments, nats.AttachmentData{
				Type:     "document",
				URL:      incoming.MediaFileID,
				Filename: incoming.MediaFileName,
				MimeType: incoming.MediaMimeType,
				Metadata: map[string]string{
					"file_id": incoming.MediaFileID,
				},
			})
		}
	case telegram.MessageTypeLocation:
		contentType = "location"
		if incoming.Location != nil {
			content = fmt.Sprintf("%f,%f", incoming.Location.Latitude, incoming.Location.Longitude)
			metadata["latitude"] = fmt.Sprintf("%f", incoming.Location.Latitude)
			metadata["longitude"] = fmt.Sprintf("%f", incoming.Location.Longitude)
		}
	case telegram.MessageTypeContact:
		contentType = "contact"
		if incoming.Contact != nil {
			contactData, _ := json.Marshal(incoming.Contact)
			content = string(contactData)
		}
	}

	// Handle reply
	if incoming.ReplyToMsgID != nil {
		metadata["reply_to_id"] = fmt.Sprintf("%d", *incoming.ReplyToMsgID)
	}

	// Create sender name
	senderName := incoming.FromFirstName
	if incoming.FromLastName != "" {
		senderName += " " + incoming.FromLastName
	}

	// Publish to NATS
	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: "telegram",
		ExternalID:  fmt.Sprintf("%d", incoming.MessageID),
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}
	inbound.Metadata["sender_id"] = fmt.Sprintf("%d", incoming.ChatID)
	inbound.Metadata["sender_name"] = senderName

	if h.producer != nil {
		if err := h.producer.PublishInbound(c.Request.Context(), inbound); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process message"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TwilioWebhook handles Twilio SMS/MMS webhooks
func (h *ChannelHandler) TwilioWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	// Get channel
	channel, err := h.channelService.GetByID(c.Request.Context(), channelID)
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

		if h.producer != nil {
			if err := h.producer.PublishInbound(c.Request.Context(), inbound); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process message"})
				return
			}
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
		if h.producer != nil {
			statusUpdate := &nats.StatusUpdate{
				ExternalID:   payload.MessageSID,
				ChannelType:  "sms",
				Status:       status,
				ErrorMessage: payload.ErrorMessage,
				Timestamp:    time.Now(),
			}
			h.producer.PublishStatusUpdate(c.Request.Context(), statusUpdate)
		}

		c.Header("Content-Type", "text/xml")
		c.String(http.StatusOK, sms.EmptyTwiMLResponse())

	default:
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
