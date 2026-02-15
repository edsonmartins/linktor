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

// SendReactionRequest represents a send reaction request
type SendReactionRequest struct {
	Emoji string `json:"emoji"` // Empty string to remove reaction
}

// List godoc
// @Summary      List messages
// @Description  Returns all messages for a conversation
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(50)
// @Success      200 {object} Response{data=[]entity.Message,meta=MetaResponse}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /conversations/{id}/messages [get]
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

// Send godoc
// @Summary      Send message
// @Description  Send a new message in a conversation
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        request body SendMessageRequest true "Message data"
// @Success      201 {object} Response{data=entity.Message}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /conversations/{id}/messages [post]
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

// Get godoc
// @Summary      Get message
// @Description  Returns a message by ID
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Message ID"
// @Success      200 {object} Response{data=entity.Message}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /messages/{id} [get]
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

// SendReaction godoc
// @Summary      Send reaction
// @Description  Send a reaction (emoji) to a message. Send empty emoji to remove reaction.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        messageId path string true "Message ID to react to"
// @Param        request body SendReactionRequest true "Reaction data"
// @Success      200 {object} Response{data=map[string]string}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Router       /conversations/{id}/messages/{messageId}/reactions [post]
func (h *MessageHandler) SendReaction(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		RespondValidationError(c, "Conversation ID is required", nil)
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		RespondValidationError(c, "Message ID is required", nil)
		return
	}

	userID := middleware.MustGetUserID(c)
	if userID == "" {
		return
	}

	var req SendReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Send reaction using the message service
	err := h.messageService.SendReaction(c.Request.Context(), conversationID, messageID, req.Emoji, userID)
	if err != nil {
		RespondError(c, err)
		return
	}

	action := "added"
	if req.Emoji == "" {
		action = "removed"
	}

	RespondSuccess(c, map[string]string{
		"message":    "Reaction " + action + " successfully",
		"message_id": messageID,
		"emoji":      req.Emoji,
	})
}
