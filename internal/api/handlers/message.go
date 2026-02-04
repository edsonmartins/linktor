package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
)

// MessageHandler handles message endpoints
type MessageHandler struct {
	messageService *service.MessageService
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// SendMessageRequest represents a send message request
type SendMessageRequest struct {
	ContentType string            `json:"content_type" binding:"required"`
	Content     string            `json:"content" binding:"required"`
	Metadata    map[string]string `json:"metadata"`
}

// List returns all messages for a conversation
func (h *MessageHandler) List(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	messages, total, err := h.messageService.ListByConversation(c.Request.Context(), conversationID, nil)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondWithMeta(c, messages, &MetaResponse{
		Page:       1,
		PageSize:   50,
		TotalItems: total,
	})
}

// Send sends a new message
func (h *MessageHandler) Send(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	userID := middleware.MustGetUserID(c)
	if userID == "" {
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.SendMessageInput{
		ConversationID: conversationID,
		SenderID:       userID,
		SenderType:     "user",
		ContentType:    req.ContentType,
		Content:        req.Content,
		Metadata:       req.Metadata,
	}

	message, err := h.messageService.Send(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, message)
}

// Get returns a message by ID
func (h *MessageHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Message ID is required", nil)
		return
	}

	message, err := h.messageService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, message)
}
