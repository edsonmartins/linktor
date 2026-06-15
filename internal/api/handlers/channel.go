package handlers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
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

// Reencrypt re-encrypts all of the tenant's channel credentials with the
// current primary key (used after a crypto key rotation).
func (h *ChannelHandler) Reencrypt(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	count, err := h.channelService.ReencryptCredentials(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, gin.H{"reencrypted": count})
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
