package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// AuditHandler exposes the audit trail (read-only).
type AuditHandler struct {
	auditService *service.AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// CurrentActor builds an audit Actor from the authenticated request context.
// Exported so other handlers can record audited actions consistently.
func CurrentActor(c *gin.Context) service.Actor {
	return service.Actor{
		ID:    middleware.GetUserID(c),
		Email: middleware.GetUserEmail(c),
		IP:    c.ClientIP(),
		Agent: c.Request.UserAgent(),
	}
}

// List returns audit entries for the current tenant.
func (h *AuditHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	params := repository.NewListParams()
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		params.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "50")); err == nil && pageSize > 0 {
		params.PageSize = pageSize
	}

	filters := &repository.AuditLogFilters{
		ActorID:      strings.TrimSpace(c.Query("actor_id")),
		Action:       strings.TrimSpace(c.Query("action")),
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		ResourceID:   strings.TrimSpace(c.Query("resource_id")),
	}

	logs, total, err := h.auditService.List(c.Request.Context(), tenantID, filters, params)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, logs, &MetaResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: total,
		TotalPages: int((total + int64(params.PageSize) - 1) / int64(params.PageSize)),
		HasNext:    int64(params.Page*params.PageSize) < total,
		HasPrev:    params.Page > 1,
	})
}
