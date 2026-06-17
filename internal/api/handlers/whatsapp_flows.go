package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// WhatsAppFlowsHandler exposes WhatsApp Flows management endpoints.
type WhatsAppFlowsHandler struct {
	service *service.WhatsAppFlowsService
}

// NewWhatsAppFlowsHandler creates a new WhatsAppFlowsHandler.
func NewWhatsAppFlowsHandler(s *service.WhatsAppFlowsService) *WhatsAppFlowsHandler {
	return &WhatsAppFlowsHandler{service: s}
}

type createWhatsAppFlowRequest struct {
	ChannelID   string   `json:"channel_id" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Categories  []string `json:"categories"`
	EndpointURI string   `json:"endpoint_uri"`
}

type updateWhatsAppFlowRequest struct {
	ChannelID   string   `json:"channel_id" binding:"required"`
	Name        string   `json:"name"`
	Categories  []string `json:"categories"`
	EndpointURI string   `json:"endpoint_uri"`
}

type channelOnlyRequest struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

// CreateFlow godoc
// @Summary  Create a WhatsApp Flow on Meta and persist it locally
// @Tags     whatsapp-flows
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    request body createWhatsAppFlowRequest true "Flow creation parameters"
// @Success  201 {object} entity.Flow
// @Router   /whatsapp/flows [post]
func (h *WhatsAppFlowsHandler) CreateFlow(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req createWhatsAppFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	flow, err := h.service.CreateFlow(c.Request.Context(), tenantID, req.ChannelID, req.Name, req.Categories, req.EndpointURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, flow)
}

// UpdateFlow godoc
// @Summary  Update a WhatsApp Flow on Meta
// @Tags     whatsapp-flows
// @Security BearerAuth
// @Router   /whatsapp/flows/{id} [put]
func (h *WhatsAppFlowsHandler) UpdateFlow(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	flowID := c.Param("id")
	var req updateWhatsAppFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	flow, err := h.service.UpdateFlow(c.Request.Context(), tenantID, req.ChannelID, flowID, req.Name, req.Categories, req.EndpointURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flow)
}

// DeleteFlow godoc
// @Summary  Delete a WhatsApp Flow
// @Tags     whatsapp-flows
// @Security BearerAuth
// @Router   /whatsapp/flows/{id} [delete]
func (h *WhatsAppFlowsHandler) DeleteFlow(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	flowID := c.Param("id")
	channelID := c.Query("channel_id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id query parameter is required"})
		return
	}
	if err := h.service.DeleteFlow(c.Request.Context(), tenantID, channelID, flowID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// PublishFlow godoc
// @Summary  Publish a WhatsApp Flow on Meta
// @Tags     whatsapp-flows
// @Security BearerAuth
// @Router   /whatsapp/flows/{id}/publish [post]
func (h *WhatsAppFlowsHandler) PublishFlow(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	flowID := c.Param("id")
	var req channelOnlyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.PublishFlow(c.Request.Context(), tenantID, req.ChannelID, flowID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

// GetFlowPreview godoc
// @Summary  Get the preview URL for a WhatsApp Flow
// @Tags     whatsapp-flows
// @Security BearerAuth
// @Router   /whatsapp/flows/{id}/preview [get]
func (h *WhatsAppFlowsHandler) GetFlowPreview(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	flowID := c.Param("id")
	channelID := c.Query("channel_id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_id query parameter is required"})
		return
	}
	url, err := h.service.GetFlowPreview(c.Request.Context(), tenantID, channelID, flowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"preview_url": url})
}

// SyncFlows godoc
// @Summary  Sync WhatsApp Flows from Meta
// @Tags     whatsapp-flows
// @Security BearerAuth
// @Router   /whatsapp/flows/sync [post]
func (h *WhatsAppFlowsHandler) SyncFlows(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req channelOnlyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	flows, err := h.service.SyncFlowsFromMeta(c.Request.Context(), tenantID, req.ChannelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flows": flows, "count": len(flows)})
}
