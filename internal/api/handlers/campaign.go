package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// CampaignHandler exposes bulk campaign management.
type CampaignHandler struct {
	svc   *service.CampaignService
	audit *service.AuditService
}

// NewCampaignHandler creates a new CampaignHandler.
func NewCampaignHandler(svc *service.CampaignService, audit *service.AuditService) *CampaignHandler {
	return &CampaignHandler{svc: svc, audit: audit}
}

type campaignRecipientRequest struct {
	Phone     string   `json:"phone" binding:"required"`
	ContactID string   `json:"contact_id"`
	Params    []string `json:"params"`
}

type createCampaignRequest struct {
	Name             string                     `json:"name" binding:"required"`
	ChannelID        string                     `json:"channel_id" binding:"required"`
	TemplateName     string                     `json:"template_name" binding:"required"`
	TemplateLanguage string                     `json:"template_language"`
	Recipients       []campaignRecipientRequest `json:"recipients" binding:"required"`
}

func (h *CampaignHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req createCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	recipients := make([]service.RecipientInput, 0, len(req.Recipients))
	for _, r := range req.Recipients {
		recipients = append(recipients, service.RecipientInput{Phone: r.Phone, ContactID: r.ContactID, Params: r.Params})
	}

	campaign, err := h.svc.Create(c.Request.Context(), tenantID, middleware.GetUserID(c), &service.CampaignInput{
		Name:             req.Name,
		ChannelID:        req.ChannelID,
		TemplateName:     req.TemplateName,
		TemplateLanguage: req.TemplateLanguage,
		Recipients:       recipients,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "campaign.create", "campaign", campaign.ID, map[string]interface{}{"name": campaign.Name, "recipients": campaign.TotalRecipients})
	RespondCreated(c, campaign)
}

func (h *CampaignHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	params := repository.NewListParams()
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		params.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && pageSize > 0 {
		params.PageSize = pageSize
	}
	campaigns, total, err := h.svc.List(c.Request.Context(), tenantID, params)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondWithMeta(c, campaigns, &MetaResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: total,
		TotalPages: int((total + int64(params.PageSize) - 1) / int64(params.PageSize)),
		HasNext:    int64(params.Page*params.PageSize) < total,
		HasPrev:    params.Page > 1,
	})
}

func (h *CampaignHandler) Get(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	campaign, err := h.svc.Get(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, campaign)
}

func (h *CampaignHandler) ListRecipients(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	params := repository.NewListParams()
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		params.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "100")); err == nil && pageSize > 0 {
		params.PageSize = pageSize
	}
	recipients, total, err := h.svc.ListRecipients(c.Request.Context(), tenantID, c.Param("id"), params)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondWithMeta(c, recipients, &MetaResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: total,
		TotalPages: int((total + int64(params.PageSize) - 1) / int64(params.PageSize)),
		HasNext:    int64(params.Page*params.PageSize) < total,
		HasPrev:    params.Page > 1,
	})
}

func (h *CampaignHandler) Start(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	id := c.Param("id")
	campaign, err := h.svc.Start(c.Request.Context(), tenantID, id)
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "campaign.start", "campaign", id, nil)
	RespondSuccess(c, campaign)
}

func (h *CampaignHandler) Retry(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	id := c.Param("id")
	count, err := h.svc.RetryFailed(c.Request.Context(), tenantID, id)
	if err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "campaign.retry", "campaign", id, map[string]interface{}{"requeued": count})
	RespondSuccess(c, gin.H{"requeued": count})
}

func (h *CampaignHandler) Cancel(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	id := c.Param("id")
	if err := h.svc.Cancel(c.Request.Context(), tenantID, id); err != nil {
		RespondError(c, err)
		return
	}
	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c), "campaign.cancel", "campaign", id, nil)
	RespondNoContent(c)
}
