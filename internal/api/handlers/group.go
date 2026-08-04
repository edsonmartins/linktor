package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// GroupHandler handles group management endpoints
type GroupHandler struct {
	producer nats.Publisher
	channels *service.ChannelService
}

// Group represents a messaging group
type Group struct {
	ID           string   `json:"id"`
	ChannelID    string   `json:"channel_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Participants []string `json:"participants,omitempty"`
	IsAdmin      bool     `json:"is_admin,omitempty"`
	InviteLink   string   `json:"invite_link,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
}

// GroupCreateRequest represents a request to create a group
type GroupCreateRequest struct {
	ChannelID    string   `json:"channel_id" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	Participants []string `json:"participants"`
}

// GroupParticipantRequest represents a request to manage group participants
type GroupParticipantRequest struct {
	Participants []string `json:"participants" binding:"required"`
	Action       string   `json:"action" binding:"required"` // add, remove, promote, demote
}

// NewGroupHandler creates a new GroupHandler
func NewGroupHandler(producer nats.Publisher, channels *service.ChannelService) *GroupHandler {
	return &GroupHandler{producer: producer, channels: channels}
}

// GroupSendRequest is the body of a send-to-group request.
type GroupSendRequest struct {
	Text     string            `json:"text" binding:"required"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SendMessage publishes a text message straight to a group JID (@g.us), bypassing
// contact resolution — POST /api/v1/channels/:id/groups/:groupId/messages. The
// group JID rides as RecipientID and reaches whatsmeow intact; the outbound worker
// resolves the channel's live adapter and delivers it.
func (h *GroupHandler) SendMessage(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	channelID := c.Param("id")
	groupJID := c.Param("groupId")
	if groupJID == "" {
		RespondValidationError(c, "group id is required", nil)
		return
	}
	var req GroupSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "invalid request body", map[string]string{"details": err.Error()})
		return
	}
	// Validate the channel belongs to the tenant and get its type (routes the subject).
	ch, err := h.channels.GetByTenantAndID(c.Request.Context(), tenantID, channelID)
	if err != nil {
		RespondError(c, err)
		return
	}
	if h.producer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "outbound delivery unavailable"})
		return
	}
	out := &nats.OutboundMessage{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   ch.ID,
		ChannelType: string(ch.Type),
		ContentType: "text",
		Content:     req.Text,
		Metadata:    req.Metadata,
		RecipientID: groupJID,
		Timestamp:   time.Now(),
	}
	if err := h.producer.PublishOutbound(c.Request.Context(), out); err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, Response{
		Success: true,
		Data:    map[string]interface{}{"id": out.ID, "recipient": groupJID, "status": "queued"},
	})
}

// List returns all groups - GET /api/v1/groups
func (h *GroupHandler) List(c *gin.Context) {
	RespondSuccess(c, []Group{})
}

// Get returns a group by ID - GET /api/v1/groups/:id
func (h *GroupHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "group id is required", nil)
		return
	}

	group := Group{
		ID:        id,
		Name:      "Placeholder Group",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	RespondSuccess(c, group)
}

// Create creates a new group - POST /api/v1/groups
func (h *GroupHandler) Create(c *gin.Context) {
	var req GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "invalid request body", map[string]string{
			"details": err.Error(),
		})
		return
	}

	group := Group{
		ID:           uuid.New().String(),
		ChannelID:    req.ChannelID,
		Name:         req.Name,
		Description:  req.Description,
		Participants: req.Participants,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "Group created",
			"group":   group,
		},
	})
}

// UpdateParticipants manages group participants - POST /api/v1/groups/:id/participants
func (h *GroupHandler) UpdateParticipants(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "group id is required", nil)
		return
	}

	var req GroupParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "invalid request body", map[string]string{
			"details": err.Error(),
		})
		return
	}

	validActions := map[string]bool{"add": true, "remove": true, "promote": true, "demote": true}
	if !validActions[req.Action] {
		RespondValidationError(c, "invalid action, must be one of: add, remove, promote, demote", nil)
		return
	}

	RespondSuccess(c, map[string]interface{}{
		"message":      "Participants updated",
		"group_id":     id,
		"action":       req.Action,
		"participants": req.Participants,
	})
}

// GetInviteLink returns the invite link for a group - GET /api/v1/groups/:id/invite
func (h *GroupHandler) GetInviteLink(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "group id is required", nil)
		return
	}

	RespondSuccess(c, map[string]interface{}{
		"group_id":    id,
		"invite_link": "https://chat.example.com/invite/" + id,
	})
}

// Leave leaves a group - POST /api/v1/groups/:id/leave
func (h *GroupHandler) Leave(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "group id is required", nil)
		return
	}

	RespondSuccess(c, map[string]interface{}{
		"message":  "Left group successfully",
		"group_id": id,
	})
}
