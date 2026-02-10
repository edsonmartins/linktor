package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// VREHandler handles VRE endpoints
type VREHandler struct {
	vreService *service.VREService
}

// NewVREHandler creates a new VRE handler
func NewVREHandler(vreService *service.VREService) *VREHandler {
	return &VREHandler{
		vreService: vreService,
	}
}

// RenderRequest represents the API request for rendering
type RenderRequest struct {
	TenantID     string                 `json:"tenant_id"`
	TemplateID   string                 `json:"template_id,omitempty"`
	HTML         string                 `json:"html,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Channel      string                 `json:"channel,omitempty"`
	Caption      string                 `json:"caption,omitempty"`
	FollowUpText string                 `json:"follow_up_text,omitempty"`
	SendTo       string                 `json:"send_to,omitempty"`
	Width        int                    `json:"width,omitempty"`
	Format       string                 `json:"format,omitempty"`
	Quality      int                    `json:"quality,omitempty"`
	Scale        float64                `json:"scale,omitempty"`
}

// Render handles POST /api/v1/vre/render
// @Summary Render HTML to image
// @Description Renders HTML content or a predefined template to an image
// @Tags VRE
// @Accept json
// @Produce json
// @Param request body RenderRequest true "Render request"
// @Success 200 {object} entity.RenderResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/render [post]
func (h *VREHandler) Render(c *gin.Context) {
	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get tenant ID from context if not provided
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = middleware.MustGetTenantID(c)
	}

	// Build entity request
	renderReq := &entity.RenderRequest{
		TenantID:     tenantID,
		TemplateID:   req.TemplateID,
		HTML:         req.HTML,
		Data:         req.Data,
		Channel:      entity.VREChannelType(req.Channel),
		Caption:      req.Caption,
		FollowUpText: req.FollowUpText,
		SendTo:       req.SendTo,
		Width:        req.Width,
		Quality:      req.Quality,
		Scale:        req.Scale,
	}

	if req.Format != "" {
		renderReq.Format = entity.OutputFormat(req.Format)
	}

	// Render
	response, err := h.vreService.Render(c.Request.Context(), renderReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RenderAndSend handles POST /api/v1/vre/render-and-send
// @Summary Render and send to channel
// @Description Renders HTML and sends the image directly to a recipient
// @Tags VRE
// @Accept json
// @Produce json
// @Param request body RenderRequest true "Render request with send_to"
// @Success 200 {object} entity.RenderResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/render-and-send [post]
func (h *VREHandler) RenderAndSend(c *gin.Context) {
	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.SendTo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "send_to is required"})
		return
	}

	// Get tenant ID from context if not provided
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = middleware.MustGetTenantID(c)
	}

	// Build entity request
	renderReq := &entity.RenderRequest{
		TenantID:     tenantID,
		TemplateID:   req.TemplateID,
		HTML:         req.HTML,
		Data:         req.Data,
		Channel:      entity.VREChannelType(req.Channel),
		Caption:      req.Caption,
		FollowUpText: req.FollowUpText,
		SendTo:       req.SendTo,
		Width:        req.Width,
		Quality:      req.Quality,
		Scale:        req.Scale,
	}

	if req.Format != "" {
		renderReq.Format = entity.OutputFormat(req.Format)
	}

	// Render
	response, err := h.vreService.Render(c.Request.Context(), renderReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: Send via channel adapter
	// For now, just return the rendered response
	response.Delivered = false // Will be true when channel sending is implemented

	c.JSON(http.StatusOK, response)
}

// ListTemplates handles GET /api/v1/vre/templates
// @Summary List available templates
// @Description Returns list of available template IDs for the tenant
// @Tags VRE
// @Produce json
// @Success 200 {object} map[string][]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/templates [get]
func (h *VREHandler) ListTemplates(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)

	templates, err := h.vreService.ListTemplates(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// PreviewTemplate handles GET /api/v1/vre/templates/:id/preview
// @Summary Preview a template
// @Description Renders a template with sample data for preview
// @Tags VRE
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} entity.RenderResponse
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/templates/{id}/preview [get]
func (h *VREHandler) PreviewTemplate(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	templateID := c.Param("id")

	response, err := h.vreService.PreviewTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetBrandConfig handles GET /api/v1/vre/config
// @Summary Get brand configuration
// @Description Returns the brand configuration for the tenant
// @Tags VRE
// @Produce json
// @Success 200 {object} entity.TenantBrandConfig
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/config [get]
func (h *VREHandler) GetBrandConfig(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)

	config, err := h.vreService.GetBrandConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateBrandConfig handles PUT /api/v1/vre/config
// @Summary Update brand configuration
// @Description Updates the brand configuration for the tenant
// @Tags VRE
// @Accept json
// @Produce json
// @Param config body entity.TenantBrandConfig true "Brand configuration"
// @Success 200 {object} entity.TenantBrandConfig
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/config [put]
func (h *VREHandler) UpdateBrandConfig(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)

	var config entity.TenantBrandConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	config.TenantID = tenantID

	if err := h.vreService.SaveBrandConfig(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UploadTemplate handles POST /api/v1/vre/templates/:id
// @Summary Upload custom template
// @Description Uploads a custom HTML template for the tenant
// @Tags VRE
// @Accept text/html
// @Produce json
// @Param id path string true "Template ID"
// @Param template body string true "HTML template content"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/templates/{id} [post]
func (h *VREHandler) UploadTemplate(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	templateID := c.Param("id")

	// Read body as HTML
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body: " + err.Error()})
		return
	}

	if err := h.vreService.SaveTemplate(c.Request.Context(), tenantID, templateID, string(body)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template saved successfully", "template_id": templateID})
}

// InvalidateCache handles DELETE /api/v1/vre/cache
// @Summary Invalidate cache
// @Description Invalidates all cached renders for the tenant
// @Tags VRE
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/vre/cache [delete]
func (h *VREHandler) InvalidateCache(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)

	if err := h.vreService.InvalidateCache(c.Request.Context(), tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache invalidated"})
}
