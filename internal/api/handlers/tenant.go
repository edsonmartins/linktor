package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// TenantHandler handles tenant endpoints
type TenantHandler struct {
	tenantService *service.TenantService
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// Get returns the current tenant
func (h *TenantHandler) Get(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	tenant, err := h.tenantService.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, tenant)
}

// UpdateTenantRequest represents an update tenant request
type UpdateTenantRequest struct {
	Name     *string           `json:"name"`
	Settings map[string]string `json:"settings"`
}

// Update updates the current tenant
func (h *TenantHandler) Update(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateTenantInput{
		Name:     req.Name,
		Settings: req.Settings,
	}

	tenant, err := h.tenantService.Update(c.Request.Context(), tenantID, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, tenant)
}

// GetUsage returns the current tenant's usage statistics
func (h *TenantHandler) GetUsage(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	usage, err := h.tenantService.GetUsage(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, usage)
}
