package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// CannedResponseHandler exposes CRUD for quick replies.
type CannedResponseHandler struct {
	svc   *service.CannedResponseService
	audit *service.AuditService
}

// NewCannedResponseHandler creates a new CannedResponseHandler.
func NewCannedResponseHandler(svc *service.CannedResponseService, audit *service.AuditService) *CannedResponseHandler {
	return &CannedResponseHandler{svc: svc, audit: audit}
}

type cannedRequest struct {
	Shortcut string   `json:"shortcut"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
}

func (h *CannedResponseHandler) List(c *gin.Context) {
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
	search := strings.TrimSpace(c.Query("search"))

	items, total, err := h.svc.List(c.Request.Context(), tenantID, search, params)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondWithMeta(c, items, &MetaResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: total,
		TotalPages: int((total + int64(params.PageSize) - 1) / int64(params.PageSize)),
		HasNext:    int64(params.Page*params.PageSize) < total,
		HasPrev:    params.Page > 1,
	})
}

func (h *CannedResponseHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req cannedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}
	cr, err := h.svc.Create(c.Request.Context(), tenantID, middleware.GetUserID(c), &service.CannedResponseInput{
		Shortcut: req.Shortcut, Title: req.Title, Content: req.Content, Tags: req.Tags,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "canned_response.create", "canned_response", cr.ID, nil)
	RespondCreated(c, cr)
}

func (h *CannedResponseHandler) Get(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	cr, err := h.svc.Get(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, cr)
}

func (h *CannedResponseHandler) Update(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req cannedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}
	cr, err := h.svc.Update(c.Request.Context(), tenantID, c.Param("id"), &service.CannedResponseInput{
		Shortcut: req.Shortcut, Title: req.Title, Content: req.Content, Tags: req.Tags,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "canned_response.update", "canned_response", cr.ID, nil)
	RespondSuccess(c, cr)
}

func (h *CannedResponseHandler) Delete(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), tenantID, id); err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "canned_response.delete", "canned_response", id, nil)
	RespondNoContent(c)
}

// Use resolves a shortcut and bumps its usage counter.
func (h *CannedResponseHandler) Use(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	cr, err := h.svc.Use(c.Request.Context(), tenantID, c.Param("shortcut"))
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, cr)
}
