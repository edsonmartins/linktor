package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// SettingsHandler exposes per-tenant operational settings and assignment actions.
type SettingsHandler struct {
	settings   *service.SettingsService
	assignment *service.AssignmentService
	audit      *service.AuditService
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(settings *service.SettingsService, assignment *service.AssignmentService, audit *service.AuditService) *SettingsHandler {
	return &SettingsHandler{settings: settings, assignment: assignment, audit: audit}
}

type settingsRequest struct {
	AssignmentStrategy      string `json:"assignment_strategy"`
	AutoAssignEnabled       bool   `json:"auto_assign_enabled"`
	SLAFirstResponseMinutes int    `json:"sla_first_response_minutes"`
	SLAResolutionMinutes    int    `json:"sla_resolution_minutes"`
	AutoCloseAfterMinutes   int    `json:"auto_close_after_minutes"`
}

// Get returns the current tenant settings.
func (h *SettingsHandler) Get(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	settings, err := h.settings.Get(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, settings)
}

// Update persists tenant settings (admin only via route guard).
func (h *SettingsHandler) Update(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req settingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}
	settings, err := h.settings.Update(c.Request.Context(), tenantID, &service.SettingsInput{
		AssignmentStrategy:      req.AssignmentStrategy,
		AutoAssignEnabled:       req.AutoAssignEnabled,
		SLAFirstResponseMinutes: req.SLAFirstResponseMinutes,
		SLAResolutionMinutes:    req.SLAResolutionMinutes,
		AutoCloseAfterMinutes:   req.AutoCloseAfterMinutes,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "settings.update", "settings", tenantID, nil)
	RespondSuccess(c, settings)
}

// AutoAssign routes a specific conversation to an available agent now.
func (h *SettingsHandler) AutoAssign(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	conversationID := c.Param("id")
	agentID, err := h.assignment.AutoAssignByID(c.Request.Context(), tenantID, conversationID)
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "conversation.auto_assign", "conversation", conversationID, map[string]interface{}{"assigned_to": agentID})
	RespondSuccess(c, gin.H{"conversation_id": conversationID, "assigned_to": agentID})
}
