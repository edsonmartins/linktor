package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// ChannelHandler handles channel endpoints
type ChannelHandler struct {
	channelService *service.ChannelService
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(channelService *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
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

// List returns all channels for the tenant
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

// Create creates a new channel
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

// Get returns a channel by ID
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

// Update updates a channel
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

// Delete deletes a channel
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

// Connect connects a channel
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

// Disconnect disconnects a channel
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
	// TODO: Implement Telegram webhook handling
	RespondSuccess(c, gin.H{"channel_id": channelID, "status": "received"})
}

// TwilioWebhook handles Twilio webhooks
func (h *ChannelHandler) TwilioWebhook(c *gin.Context) {
	channelID := c.Param("channelId")
	// TODO: Implement Twilio webhook handling
	RespondSuccess(c, gin.H{"channel_id": channelID, "status": "received"})
}
