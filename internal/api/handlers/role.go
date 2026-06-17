package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// RoleHandler exposes RBAC role management.
type RoleHandler struct {
	svc   *service.RoleService
	audit *service.AuditService
}

// NewRoleHandler creates a new RoleHandler.
func NewRoleHandler(svc *service.RoleService, audit *service.AuditService) *RoleHandler {
	return &RoleHandler{svc: svc, audit: audit}
}

type roleRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Permissions []entity.Permission `json:"permissions"`
}

// Catalog returns the available resources and actions for building roles in UI.
func (h *RoleHandler) Catalog(c *gin.Context) {
	RespondSuccess(c, gin.H{
		"resources": []string{
			entity.ResourceUsers, entity.ResourceRoles, entity.ResourceChannels,
			entity.ResourceContacts, entity.ResourceConversations, entity.ResourceMessages,
			entity.ResourceBots, entity.ResourceFlows, entity.ResourceTemplates,
			entity.ResourceCampaigns, entity.ResourceCanned, entity.ResourceAnalytics,
			entity.ResourceKnowledge, entity.ResourceAudit, entity.ResourceSettings,
		},
		"actions": []string{
			entity.ActionRead, entity.ActionCreate, entity.ActionUpdate,
			entity.ActionDelete, entity.ActionManage,
		},
	})
}

func (h *RoleHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	roles, err := h.svc.List(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, roles)
}

func (h *RoleHandler) Get(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	role, err := h.svc.Get(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, role)
}

func (h *RoleHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}
	role, err := h.svc.Create(c.Request.Context(), tenantID, req.Name, req.Description, req.Permissions)
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "role.create", "role", role.ID, map[string]interface{}{"name": role.Name})
	RespondCreated(c, role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}
	role, err := h.svc.Update(c.Request.Context(), tenantID, c.Param("id"), req.Name, req.Description, req.Permissions)
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "role.update", "role", role.ID, map[string]interface{}{"name": role.Name})
	RespondSuccess(c, role)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), tenantID, id); err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "role.delete", "role", id, nil)
	RespondNoContent(c)
}
