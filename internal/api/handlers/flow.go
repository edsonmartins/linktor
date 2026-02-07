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

// List godoc
// @Summary      List flows
// @Description  Returns all conversation flows for the current tenant with pagination and filters
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Param        bot_id query string false "Filter by bot ID"
// @Param        is_active query bool false "Filter by active status"
// @Param        trigger query string false "Filter by trigger type"
// @Success      200 {object} Response{data=[]entity.Flow}
// @Failure      401 {object} Response
// @Router       /flows [get]
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

// Create godoc
// @Summary      Create flow
// @Description  Create a new conversation flow for automation
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateFlowRequest true "Flow data"
// @Success      201 {object} Response{data=entity.Flow}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /flows [post]
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

// Get godoc
// @Summary      Get flow
// @Description  Returns a flow by ID with all its nodes
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Flow ID"
// @Success      200 {object} Response{data=entity.Flow}
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /flows/{id} [get]
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

// Update godoc
// @Summary      Update flow
// @Description  Update a flow's definition and nodes
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Flow ID"
// @Param        request body UpdateFlowRequest true "Flow update data"
// @Success      200 {object} Response{data=entity.Flow}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /flows/{id} [put]
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

// Delete godoc
// @Summary      Delete flow
// @Description  Delete a flow by ID
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Flow ID"
// @Success      200 {object} Response{data=object{message=string}}
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /flows/{id} [delete]
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

// Activate godoc
// @Summary      Activate flow
// @Description  Activate a flow so it starts processing messages
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Flow ID"
// @Success      200 {object} Response{data=object{message=string}}
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /flows/{id}/activate [post]
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

// Deactivate godoc
// @Summary      Deactivate flow
// @Description  Deactivate a flow so it stops processing messages
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Flow ID"
// @Success      200 {object} Response{data=object{message=string}}
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /flows/{id}/deactivate [post]
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

// Test godoc
// @Summary      Test flow
// @Description  Test a flow with simulated message inputs
// @Tags         flows
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Flow ID"
// @Param        request body TestFlowRequest true "Test inputs"
// @Success      200 {object} Response{data=object{flow_id=string,results=[]object}}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /flows/{id}/test [post]
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
