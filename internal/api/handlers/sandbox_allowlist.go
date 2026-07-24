package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// SandboxAllowlistHandler exposes the tenant sandbox recipient allowlist
// (INV-017). Routes are registered admin-only (see main.go) and every change
// is audited (INV-023).
type SandboxAllowlistHandler struct {
	service *service.SandboxAllowlistService
	audit   *service.AuditService
}

// NewSandboxAllowlistHandler creates a new SandboxAllowlistHandler.
func NewSandboxAllowlistHandler(svc *service.SandboxAllowlistService, audit *service.AuditService) *SandboxAllowlistHandler {
	return &SandboxAllowlistHandler{service: svc, audit: audit}
}

// AddSandboxAllowlistRequest is the payload for adding an allowlist entry.
type AddSandboxAllowlistRequest struct {
	Recipient string `json:"recipient" binding:"required"`
	ChannelID string `json:"channel_id"`
	Note      string `json:"note"`
}

// List godoc
// @Summary      List sandbox allowlist
// @Description  Returns the tenant's sandbox recipient allowlist
// @Tags         sandbox
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} Response{data=[]entity.SandboxAllowlistEntry}
// @Router       /sandbox/allowlist [get]
func (h *SandboxAllowlistHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	entries, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, entries)
}

// Add godoc
// @Summary      Add sandbox allowlist entry
// @Description  Authorizes a recipient (E.164) for the tenant's sandbox channels
// @Tags         sandbox
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body AddSandboxAllowlistRequest true "Entry data"
// @Success      201 {object} Response{data=entity.SandboxAllowlistEntry}
// @Failure      400 {object} Response
// @Router       /sandbox/allowlist [post]
func (h *SandboxAllowlistHandler) Add(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req AddSandboxAllowlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	entry, err := h.service.Add(c.Request.Context(), &service.AddSandboxAllowlistInput{
		TenantID:  tenantID,
		ChannelID: req.ChannelID,
		Recipient: req.Recipient,
		Note:      req.Note,
		CreatedBy: middleware.GetUserID(c),
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
		"sandbox_allowlist.add", "sandbox_allowlist", entry.ID,
		map[string]interface{}{"recipient": entry.Recipient, "channel_id": entry.ChannelID})
	RespondCreated(c, entry)
}

// Remove godoc
// @Summary      Remove sandbox allowlist entry
// @Description  Removes a recipient authorization; takes effect on the next send
// @Tags         sandbox
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Entry ID"
// @Success      204 "No Content"
// @Failure      404 {object} Response
// @Router       /sandbox/allowlist/{id} [delete]
func (h *SandboxAllowlistHandler) Remove(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Entry ID is required", nil)
		return
	}

	entry, err := h.service.Remove(c.Request.Context(), tenantID, id)
	if err != nil {
		RespondError(c, err)
		return
	}

	h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
		"sandbox_allowlist.remove", "sandbox_allowlist", entry.ID,
		map[string]interface{}{"recipient": entry.Recipient, "channel_id": entry.ChannelID})
	RespondNoContent(c)
}
