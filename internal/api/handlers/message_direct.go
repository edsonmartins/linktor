package handlers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// DirectSendRequest is the body of POST /api/v1/messages/send: a channel +
// recipient send that does not require the caller to know a conversation.
type DirectSendRequest struct {
	ChannelID   string `json:"channel_id"`
	To          string `json:"to"`
	ContentType string `json:"content_type"`
	Text        string `json:"text"`
	// Content is accepted as an alias of Text so a caller can use the same field
	// name as POST /conversations/{id}/messages. Text wins when both are set.
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}

// body returns the message text, preferring "text" over the "content" alias.
func (r *DirectSendRequest) body() string {
	if strings.TrimSpace(r.Text) != "" {
		return r.Text
	}
	return r.Content
}

// DirectSendResponse is the canonical Linktor answer to a direct send.
type DirectSendResponse struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	ChannelID      string `json:"channel_id"`
	Status         string `json:"status"`
}

// reservedMetadataKeys are metadata fields Linktor itself writes: reaction
// targets, campaign bookkeeping, and the routing facts inbound events carry. A
// client that could set them would forge routing and campaign state, so a
// request naming any of them is rejected outright rather than silently stripped
// — silent stripping would make a caller believe their value survived.
//
// Deliberately NOT reserved: "reply_to_id" and "quoted_text". They look
// internal, but they are the only way a client asks for a quoted reply — the
// WhatsApp, WhatsApp Cloud and Telegram senders all read them off the outbound
// metadata (e.g. whatsapp/adapter.go SendTextMessageWithReply), and
// SendMessageRequest has no dedicated reply field. Reserving them would leave
// no route in the API capable of replying with a quote.
var reservedMetadataKeys = map[string]bool{
	"is_reaction":                 true,
	"reaction":                    true,
	"reaction_emoji":              true,
	"reaction_message_id":         true,
	"reaction_target_external_id": true,
	"reaction_target_message_id":  true,
	"campaign_id":                 true,
	"campaign_recipient_id":       true,
	"sender_id":                   true,
	"sender_name":                 true,
	"sender_type":                 true,
	"is_group":                    true,
	"chat_jid":                    true,
	"mentions":                    true,
	"external_id":                 true,
	"message_id":                  true,
	"conversation_id":             true,
	"contact_id":                  true,
	"channel_id":                  true,
	"channel_type":                true,
	"tenant_id":                   true,
	"environment":                 true,
}

// validateClientMetadata rejects metadata that would overwrite Linktor-internal
// fields. Returns the offending keys (sorted, so the error is deterministic).
func validateClientMetadata(metadata map[string]string) []string {
	var offending []string
	for k := range metadata {
		if reservedMetadataKeys[strings.ToLower(strings.TrimSpace(k))] {
			offending = append(offending, k)
		}
	}
	sort.Strings(offending)
	return offending
}

// persistableSenderID returns the acting user id only when it can actually be
// stored: messages.sender_id is a UUID column, but an API key with no bound user
// authenticates as the synthetic "apikey:<id>" (see middleware.apiKeyUserID).
// Writing that would fail the insert with `invalid input syntax for type uuid`
// and 500 the send — for exactly the server-to-server caller these routes
// target. Returning "" binds NULL instead; the sender is still identified by
// sender_type plus the audit log.
func persistableSenderID(userID string) string {
	if _, err := uuid.Parse(userID); err != nil {
		return ""
	}
	return userID
}

// respondReservedMetadata answers a request that tried to set internal fields.
func respondReservedMetadata(c *gin.Context, offending []string) {
	RespondValidationError(c, "metadata cannot overwrite Linktor internal fields", map[string]string{
		"reserved_keys": strings.Join(offending, ","),
	})
}

// SendDirect godoc
// @Summary      Send a message directly to a recipient
// @Description  Sends a message on a channel to a recipient, resolving (or creating) the contact identity, contact and conversation inside the tenant. Requires the messages:send scope for API-key callers.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request body DirectSendRequest true "Direct send"
// @Success      202 {object} Response{data=DirectSendResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      403 {object} Response
// @Failure      404 {object} Response
// @Router       /messages/send [post]
func (h *MessageHandler) SendDirect(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req DirectSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	if offending := validateClientMetadata(req.Metadata); len(offending) > 0 {
		respondReservedMetadata(c, offending)
		return
	}

	result, err := h.messageService.SendDirect(c.Request.Context(), &service.DirectSendInput{
		TenantID:    tenantID,
		ChannelID:   req.ChannelID,
		To:          req.To,
		ContentType: req.ContentType,
		Content:     req.body(),
		SenderID:    persistableSenderID(c.GetString(middleware.UserIDKey)),
		Metadata:    req.Metadata,
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	// Echo the message to the agents watching this tenant, exactly as the
	// conversation route does — otherwise a server-to-server send only appears in
	// an open conversation after a manual refetch. Skipped for an idempotent
	// repeat, which created nothing to echo.
	if !result.Idempotent && result.Message != nil {
		BroadcastNewMessage(tenantID, result.ConversationID, result.Message)
	}

	// The message is persisted and enqueued, not yet delivered: 202 with a
	// "queued" status is the honest answer. A repeated idempotency key gets the
	// same 202 carrying the original message — idempotent success, not a
	// conflict.
	c.JSON(http.StatusAccepted, Response{
		Success: true,
		Data: DirectSendResponse{
			ID:             result.MessageID,
			ConversationID: result.ConversationID,
			ChannelID:      result.ChannelID,
			Status:         directSendStatus(result.Status),
		},
	})
}

// directSendStatus renders the message status for the wire. A freshly accepted
// send is always pending, which the contract calls "queued". An idempotent
// repeat can land on an already-delivered (or already-failed) message, and
// reporting that as "queued" would be a lie the caller acts on.
func directSendStatus(status entity.MessageStatus) string {
	if status == "" || status == entity.MessageStatusPending {
		return "queued"
	}
	return string(status)
}
