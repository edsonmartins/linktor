package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/application/usecase"
)

// ConversationHandler handles conversation endpoints
type ConversationHandler struct {
	conversationService *service.ConversationService
	escalateUC          *usecase.EscalateConversationUseCase
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(
	conversationService *service.ConversationService,
	escalateUC *usecase.EscalateConversationUseCase,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		escalateUC:          escalateUC,
	}
}

// CreateConversationRequest represents a create conversation request
type CreateConversationRequest struct {
	ContactID string   `json:"contact_id" binding:"required"`
	ChannelID string   `json:"channel_id" binding:"required"`
	Subject   string   `json:"subject"`
	Priority  string   `json:"priority"`
	Tags      []string `json:"tags"`
}

// List returns all conversations for the tenant
func (h *ConversationHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Parse query parameters
	status := c.Query("status")
	assignedTo := c.Query("assigned_to")
	channelID := c.Query("channel_id")

	filters := &service.ConversationFilters{
		Status:     status,
		AssignedTo: assignedTo,
		ChannelID:  channelID,
	}

	conversations, total, err := h.conversationService.List(c.Request.Context(), tenantID, filters, nil)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, conversations, &MetaResponse{
		Page:       1,
		PageSize:   20,
		TotalItems: total,
	})
}

// Create creates a new conversation
func (h *ConversationHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateConversationInput{
		TenantID:  tenantID,
		ContactID: req.ContactID,
		ChannelID: req.ChannelID,
		Subject:   req.Subject,
		Priority:  req.Priority,
		Tags:      req.Tags,
	}

	conversation, err := h.conversationService.Create(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, conversation)
}

// Get returns a conversation by ID
func (h *ConversationHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	conversation, err := h.conversationService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, conversation)
}

// UpdateConversationRequest represents an update conversation request
type UpdateConversationRequest struct {
	Subject  *string  `json:"subject"`
	Priority *string  `json:"priority"`
	Status   *string  `json:"status"`
	Tags     []string `json:"tags"`
}

// Update updates a conversation
func (h *ConversationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateConversationInput{
		Subject:  req.Subject,
		Priority: req.Priority,
		Status:   req.Status,
		Tags:     req.Tags,
	}

	conversation, err := h.conversationService.Update(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, conversation)
}

// AssignRequest represents an assign conversation request
type AssignRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// Assign assigns a conversation to a user
func (h *ConversationHandler) Assign(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	conversation, err := h.conversationService.Assign(c.Request.Context(), id, req.UserID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, conversation)
}

// Resolve marks a conversation as resolved
func (h *ConversationHandler) Resolve(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	conversation, err := h.conversationService.Resolve(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, conversation)
}

// Reopen reopens a resolved conversation
func (h *ConversationHandler) Reopen(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	conversation, err := h.conversationService.Reopen(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, conversation)
}

// GetEscalationContext returns the escalation context for a conversation
// This provides human agents with full context when taking over from a bot
func (h *ConversationHandler) GetEscalationContext(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	if h.escalateUC == nil {
		RespondError(c, nil)
		return
	}

	escCtx, err := h.escalateUC.GetEscalationContext(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, escCtx)
}

// ConversationEscalateRequest represents an escalate conversation request
type ConversationEscalateRequest struct {
	Reason       string  `json:"reason"`
	Priority     string  `json:"priority"` // low, normal, high, urgent
	AssignTo     *string `json:"assign_to"`
}

// Escalate escalates a conversation to human agents
func (h *ConversationHandler) Escalate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req ConversationEscalateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Get conversation to get channel and contact IDs
	conversation, err := h.conversationService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	if h.escalateUC == nil {
		RespondError(c, nil)
		return
	}

	input := &usecase.EscalateConversationInput{
		ConversationID: id,
		TenantID:       tenantID,
		ChannelID:      conversation.ChannelID,
		ContactID:      conversation.ContactID,
		Reason:         req.Reason,
		Priority:       req.Priority,
		RequestedBy:    "user",
	}

	output, err := h.escalateUC.Execute(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, output)
}
