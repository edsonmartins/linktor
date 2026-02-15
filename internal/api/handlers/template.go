package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// TemplateHandler handles template endpoints
type TemplateHandler struct {
	templateService *service.TemplateService
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(templateService *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
	}
}

// CreateTemplateRequest represents a request to create a template
type CreateTemplateRequest struct {
	ChannelID  string                      `json:"channel_id" binding:"required"`
	Name       string                      `json:"name" binding:"required"`
	Language   string                      `json:"language" binding:"required"`
	Category   string                      `json:"category" binding:"required,oneof=AUTHENTICATION MARKETING UTILITY"`
	Components []entity.TemplateComponent  `json:"components" binding:"required"`
}

// List godoc
// @Summary      List templates
// @Description  Returns all templates for the current tenant
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channel_id query string false "Filter by channel ID"
// @Param        status query string false "Filter by status"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(50)
// @Success      200 {object} Response{data=[]entity.Template,meta=MetaResponse}
// @Failure      401 {object} Response
// @Router       /templates [get]
func (h *TemplateHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	channelID := c.Query("channel_id")

	var templates []*entity.Template
	var total int64
	var err error

	if channelID != "" {
		templates, total, err = h.templateService.ListByChannel(c.Request.Context(), channelID, nil)
	} else {
		templates, total, err = h.templateService.List(c.Request.Context(), tenantID, nil)
	}

	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, templates, &MetaResponse{
		Page:       1,
		PageSize:   50,
		TotalItems: total,
	})
}

// Create godoc
// @Summary      Create template
// @Description  Create a new message template
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateTemplateRequest true "Template data"
// @Success      201 {object} Response{data=entity.Template}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /templates [post]
func (h *TemplateHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateTemplateInput{
		TenantID:   tenantID,
		ChannelID:  req.ChannelID,
		Name:       req.Name,
		Language:   req.Language,
		Category:   entity.TemplateCategory(req.Category),
		Components: req.Components,
	}

	template, err := h.templateService.Create(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, template)
}

// Get godoc
// @Summary      Get template
// @Description  Returns a template by ID
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template ID"
// @Success      200 {object} Response{data=entity.Template}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /templates/{id} [get]
func (h *TemplateHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Template ID is required", nil)
		return
	}

	template, err := h.templateService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, template)
}

// Delete godoc
// @Summary      Delete template
// @Description  Deletes a template
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template ID"
// @Success      204
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /templates/{id} [delete]
func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Template ID is required", nil)
		return
	}

	if err := h.templateService.Delete(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// Sync godoc
// @Summary      Sync templates
// @Description  Synchronizes templates with Meta for a channel
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channel_id path string true "Channel ID"
// @Success      200 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{channel_id}/templates/sync [post]
func (h *TemplateHandler) Sync(c *gin.Context) {
	channelID := c.Param("channel_id")
	if channelID == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	if err := h.templateService.SyncFromMeta(c.Request.Context(), channelID); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, map[string]string{
		"message": "Templates synced successfully",
	})
}
