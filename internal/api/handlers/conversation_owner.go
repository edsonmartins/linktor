package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// ConversationOwnerHandler exposes the owner of the open conversation per
// (channel, contact) — P5 of the CRM ↔ Sales Copilot agreement (VCR-36).
// Routes: /channels/:id/contacts/:identifier/owner, scope routing:write
// (routing:read for GET), tenant of the credential. Every change is audited.
type ConversationOwnerHandler struct {
	service *service.ConversationOwnerService
	audit   *service.AuditService
}

// NewConversationOwnerHandler creates the handler.
func NewConversationOwnerHandler(svc *service.ConversationOwnerService, audit *service.AuditService) *ConversationOwnerHandler {
	return &ConversationOwnerHandler{service: svc, audit: audit}
}

// DeclareOwnerRequest is the PUT body.
type DeclareOwnerRequest struct {
	Consumidor  string     `json:"consumidor" binding:"required"`
	ConversaRef string     `json:"conversa_ref" binding:"required"`
	Desde       time.Time  `json:"desde" binding:"required"`
	ExpiraEm    *time.Time `json:"expira_em"`
}

// ReleaseOwnerRequest is the optional DELETE body (conversa_ref may also come as a query parameter).
type ReleaseOwnerRequest struct {
	ConversaRef string `json:"conversa_ref"`
}

// DeclareOwnerResponse says whether the declaration applied and which rule is in force.
type DeclareOwnerResponse struct {
	Aplicada bool                          `json:"aplicada"`
	Regra    *entity.ConversationOwnerRule `json:"regra"`
}

// ReleaseOwnerResponse says whether a rule of that conversation was released.
type ReleaseOwnerResponse struct {
	Liberada bool `json:"liberada"`
}

// Put godoc
// @Summary      Declare the owner of the open conversation
// @Description  Idempotent. A declaration of another, older conversation does not overwrite a newer rule (aplicada=false).
// @Tags         routing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        identifier path string true "Contact identifier (E.164 for phone)"
// @Param        request body DeclareOwnerRequest true "Owner"
// @Success      200 {object} Response{data=DeclareOwnerResponse}
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{id}/contacts/{identifier}/owner [put]
func (h *ConversationOwnerHandler) Put(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	var req DeclareOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body: consumidor, conversa_ref and desde (ISO-8601) are required", nil)
		return
	}
	applied, rule, err := h.service.Declare(c.Request.Context(), &service.DeclareOwnerInput{
		TenantID:    tenantID,
		ChannelID:   c.Param("id"),
		Identifier:  c.Param("identifier"),
		Consumer:    req.Consumidor,
		ConversaRef: req.ConversaRef,
		Desde:       req.Desde,
		ExpiraEm:    req.ExpiraEm,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	if applied {
		h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
			"conversation_owner.declare", "channel", rule.ChannelID,
			map[string]interface{}{"contact": rule.ContactIdentifier, "consumer": rule.Consumer, "conversa_ref": rule.ConversaRef})
	}
	RespondSuccess(c, DeclareOwnerResponse{Aplicada: applied, Regra: rule})
}

// Get godoc
// @Summary      Owner rule in force (audit)
// @Tags         routing
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        identifier path string true "Contact identifier"
// @Success      200 {object} Response{data=entity.ConversationOwnerRule}
// @Failure      404 {object} Response
// @Router       /channels/{id}/contacts/{identifier}/owner [get]
func (h *ConversationOwnerHandler) Get(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	rule, err := h.service.Get(c.Request.Context(), tenantID, c.Param("id"), c.Param("identifier"))
	if err != nil {
		RespondError(c, err)
		return
	}
	RespondSuccess(c, rule)
}

// Delete godoc
// @Summary      Release the owner rule of a conversation
// @Description  Releases only if the rule still belongs to conversa_ref; otherwise does nothing (liberada=false). Idempotent.
// @Tags         routing
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Channel ID"
// @Param        identifier path string true "Contact identifier"
// @Param        conversa_ref query string false "Conversation in the owning product (or in the JSON body)"
// @Success      200 {object} Response{data=ReleaseOwnerResponse}
// @Failure      400 {object} Response
// @Router       /channels/{id}/contacts/{identifier}/owner [delete]
func (h *ConversationOwnerHandler) Delete(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	ref := c.Query("conversa_ref")
	if ref == "" && c.Request.ContentLength != 0 {
		var req ReleaseOwnerRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			ref = req.ConversaRef
		}
	}
	released, err := h.service.Release(c.Request.Context(), tenantID, c.Param("id"), c.Param("identifier"), ref)
	if err != nil {
		RespondError(c, err)
		return
	}
	if released {
		h.audit.Record(c.Request.Context(), tenantID, CurrentActor(c),
			"conversation_owner.release", "channel", c.Param("id"),
			map[string]interface{}{"contact": entity.NormalizeContactIdentifier(c.Param("identifier")), "conversa_ref": ref})
	}
	RespondSuccess(c, ReleaseOwnerResponse{Liberada: released})
}
