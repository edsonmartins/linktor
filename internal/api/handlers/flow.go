package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// FlowHandler handles flow endpoints
type FlowHandler struct {
	flowService *service.FlowService
}

// NewFlowHandler creates a new flow handler
func NewFlowHandler(flowService *service.FlowService) *FlowHandler {
	return &FlowHandler{
		flowService: flowService,
	}
}

// CreateFlowRequest represents a create flow request
type CreateFlowRequest struct {
	Name         string            `json:"name" binding:"required"`
	Description  string            `json:"description"`
	BotID        *string           `json:"bot_id"`
	Trigger      string            `json:"trigger" binding:"required"` // intent, keyword, manual, welcome
	TriggerValue string            `json:"trigger_value"`
	StartNodeID  string            `json:"start_node_id" binding:"required"`
	Nodes        []entity.FlowNode `json:"nodes" binding:"required,min=1"`
	Priority     int               `json:"priority"`
}

// UpdateFlowRequest represents an update flow request
type UpdateFlowRequest struct {
	Name         *string            `json:"name"`
	Description  *string            `json:"description"`
	TriggerValue *string            `json:"trigger_value"`
	StartNodeID  *string            `json:"start_node_id"`
	Nodes        *[]entity.FlowNode `json:"nodes"`
	Priority     *int               `json:"priority"`
}

// TestFlowRequest represents a test flow request
type TestFlowRequest struct {
	Inputs []string `json:"inputs" binding:"required"`
}

// List returns all flows for the tenant
func (h *FlowHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := &repository.ListParams{
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.DefaultQuery("sort_by", "created_at"),
		SortDir:  c.DefaultQuery("sort_dir", "desc"),
	}

	// Parse filters
	var filter *entity.FlowFilter
	botID := c.Query("bot_id")
	isActiveStr := c.Query("is_active")
	trigger := c.Query("trigger")

	if botID != "" || isActiveStr != "" || trigger != "" {
		filter = &entity.FlowFilter{}
		if botID != "" {
			filter.BotID = &botID
		}
		if isActiveStr != "" {
			isActive := isActiveStr == "true"
			filter.IsActive = &isActive
		}
		if trigger != "" {
			filter.Trigger = &trigger
		}
	}

	flows, total, err := h.flowService.List(c.Request.Context(), tenantID, filter, params)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondPaginated(c, flows, total, params.Page, params.PageSize)
}

// Create creates a new flow
func (h *FlowHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &entity.CreateFlowInput{
		Name:         req.Name,
		Description:  req.Description,
		BotID:        req.BotID,
		Trigger:      entity.FlowTriggerType(req.Trigger),
		TriggerValue: req.TriggerValue,
		StartNodeID:  req.StartNodeID,
		Nodes:        req.Nodes,
		Priority:     req.Priority,
	}

	flow, err := h.flowService.Create(c.Request.Context(), tenantID, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, flow)
}

// Get returns a flow by ID
func (h *FlowHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Flow ID is required", nil)
		return
	}

	flow, err := h.flowService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	// Verify tenant ownership
	tenantID := middleware.MustGetTenantID(c)
	if flow.TenantID != tenantID {
		RespondForbidden(c, "Flow does not belong to tenant")
		return
	}

	c.JSON(http.StatusOK, flow)
}

// Update updates a flow
func (h *FlowHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Flow ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Verify ownership
	existing, err := h.flowService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}
	if existing.TenantID != tenantID {
		RespondForbidden(c, "Flow does not belong to tenant")
		return
	}

	var req UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &entity.UpdateFlowInput{
		Name:         req.Name,
		Description:  req.Description,
		TriggerValue: req.TriggerValue,
		StartNodeID:  req.StartNodeID,
		Nodes:        req.Nodes,
		Priority:     req.Priority,
	}

	flow, err := h.flowService.Update(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, flow)
}

// Delete deletes a flow
func (h *FlowHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Flow ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Verify ownership
	existing, err := h.flowService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}
	if existing.TenantID != tenantID {
		RespondForbidden(c, "Flow does not belong to tenant")
		return
	}

	if err := h.flowService.Delete(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flow deleted successfully"})
}

// Activate activates a flow
func (h *FlowHandler) Activate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Flow ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Verify ownership
	existing, err := h.flowService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}
	if existing.TenantID != tenantID {
		RespondForbidden(c, "Flow does not belong to tenant")
		return
	}

	if err := h.flowService.Activate(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flow activated successfully"})
}

// Deactivate deactivates a flow
func (h *FlowHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Flow ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Verify ownership
	existing, err := h.flowService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}
	if existing.TenantID != tenantID {
		RespondForbidden(c, "Flow does not belong to tenant")
		return
	}

	if err := h.flowService.Deactivate(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flow deactivated successfully"})
}

// Test tests a flow with simulated inputs
func (h *FlowHandler) Test(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Flow ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Verify ownership
	existing, err := h.flowService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}
	if existing.TenantID != tenantID {
		RespondForbidden(c, "Flow does not belong to tenant")
		return
	}

	var req TestFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	results, err := h.flowService.TestFlow(c.Request.Context(), id, req.Inputs)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flow_id": id,
		"results": results,
	})
}
