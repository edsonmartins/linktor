package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
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
	ChannelID             string                     `json:"channel_id" binding:"required"`
	Name                  string                     `json:"name" binding:"required"`
	Language              string                     `json:"language" binding:"required"`
	Category              string                     `json:"category" binding:"required,oneof=AUTHENTICATION MARKETING UTILITY"`
	SubCategory           string                     `json:"sub_category,omitempty"`
	ParameterFormat       string                     `json:"parameter_format,omitempty" binding:"omitempty,oneof=POSITIONAL NAMED"`
	MessageSendTTLSeconds int                        `json:"message_send_ttl_seconds,omitempty"`
	AllowCategoryChange   bool                       `json:"allow_category_change,omitempty"`
	Components            []entity.TemplateComponent `json:"components" binding:"required"`
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
	params := templateListParamsFromQuery(c)

	var templates []*entity.Template
	var total int64
	var err error

	if channelID != "" {
		templates, total, err = h.templateService.ListByChannel(c.Request.Context(), channelID, params)
	} else {
		templates, total, err = h.templateService.List(c.Request.Context(), tenantID, params)
	}

	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, templates, &MetaResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: total,
	})
}

// templateListParamsFromQuery builds ListParams from the request's query string.
// The filters map is populated with the values the template repo understands
// (category, sub_category, language, status, quality_score, name, content,
// since, until). Page/page_size are parsed with sensible defaults.
func templateListParamsFromQuery(c *gin.Context) *repository.ListParams {
	params := repository.NewListParams()
	params.PageSize = 50

	if s := c.Query("page"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			params.Page = n
		}
	}
	if s := c.Query("page_size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 250 {
			params.PageSize = n
		}
	}

	// Pass string filters straight through; the repo validates shapes.
	for _, key := range []string{"category", "sub_category", "language", "status", "quality_score", "name", "content"} {
		if v := c.Query(key); v != "" {
			params.Filters[key] = v
		}
	}

	// since/until accept unix seconds. Silently ignore malformed values —
	// no filter beats rejecting the request for a typo.
	if s := c.Query("since"); s != "" {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			params.Filters["since"] = ts
		}
	}
	if s := c.Query("until"); s != "" {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			params.Filters["until"] = ts
		}
	}
	return params
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
		TenantID:              tenantID,
		ChannelID:             req.ChannelID,
		Name:                  req.Name,
		Language:              req.Language,
		Category:              entity.TemplateCategory(req.Category),
		SubCategory:           req.SubCategory,
		ParameterFormat:       entity.TemplateParameterFormat(req.ParameterFormat),
		MessageSendTTLSeconds: req.MessageSendTTLSeconds,
		AllowCategoryChange:   req.AllowCategoryChange,
		Components:            req.Components,
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

// EditTemplateRequest is the payload accepted by PATCH /templates/{id}.
// Category and components are the two fields Meta accepts on edit;
// message_send_ttl_seconds piggybacks because it's also editable.
type EditTemplateRequest struct {
	Category              string                     `json:"category,omitempty" binding:"omitempty,oneof=AUTHENTICATION MARKETING UTILITY"`
	Components            []entity.TemplateComponent `json:"components,omitempty"`
	MessageSendTTLSeconds int                        `json:"message_send_ttl_seconds,omitempty"`
}

// Edit godoc
// @Summary      Edit template
// @Description  Updates an existing template on Meta (status resets to PENDING)
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template ID"
// @Param        request body EditTemplateRequest true "Template edits"
// @Success      200 {object} Response{data=entity.Template}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /templates/{id} [patch]
func (h *TemplateHandler) Edit(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Template ID is required", nil)
		return
	}

	var req EditTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	template, err := h.templateService.Edit(c.Request.Context(), &service.EditTemplateInput{
		ID:                    id,
		Category:              entity.TemplateCategory(req.Category),
		Components:            req.Components,
		MessageSendTTLSeconds: req.MessageSendTTLSeconds,
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, template)
}

// Refresh godoc
// @Summary      Refresh template from Meta
// @Description  Re-fetches a single template from Meta and updates the local copy
// @Tags         templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template ID"
// @Success      200 {object} Response{data=entity.Template}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /templates/{id}/refresh [post]
func (h *TemplateHandler) Refresh(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Template ID is required", nil)
		return
	}

	template, err := h.templateService.RefreshFromMeta(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, template)
}
