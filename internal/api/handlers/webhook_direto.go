package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// diretoWebhookPayload is the inbound envelope the VendaX Direto runtime POSTs
// to Linktor (RFC-009). Signed with HMAC-SHA256 in X-Direto-Signature-256
// (same "sha256=<hex>" format as Meta's X-Hub-Signature-256).
type diretoWebhookPayload struct {
	InstanceID string                 `json:"instanceId"`
	Channel    string                 `json:"channel"`
	Messages   []diretoWebhookMessage `json:"messages"`
}

type diretoWebhookMessage struct {
	ID      string `json:"id"`
	From    string `json:"from"` // phone E.164 of the Direto client
	Type    string `json:"type"`
	Text    *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
	Media *struct {
		Mime string `json:"mime"`
		Name string `json:"name"`
		Ref  string `json:"ref"`
	} `json:"media,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// DiretoWebhook receives inbound messages from the VendaX Direto channel and
// publishes them onto the inbound stream (RFC-009). Mirrors WhatsAppWebhook:
// resolve channel by :channelId, verify HMAC, parse, publish nats.InboundMessage.
func (h *WebhookHandler) DiretoWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	// Verify HMAC if a secret is configured (X-Direto-Signature-256).
	if secret, ok := channel.Credentials["webhook_secret"]; ok && secret != "" {
		if !h.verifyWhatsAppSignature(rawBody, c.GetHeader("X-Direto-Signature-256"), secret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	} else if h.requireWebhookSecrets {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook secret not configured"})
		return
	}

	var payload diretoWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	for _, msg := range payload.Messages {
		if err := h.processDiretoMessage(c, channel.ID, channel.TenantID, msg); err != nil {
			// Log and continue — don't fail the whole batch on one message.
			continue
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) processDiretoMessage(c *gin.Context, channelID, tenantID string, msg diretoWebhookMessage) error {
	contentType := "text"
	content := ""
	metadata := map[string]string{"phone": msg.From, "sender_id": msg.From}
	var attachments []nats.AttachmentData

	if msg.Media != nil {
		contentType = msg.Type
		if contentType == "" {
			contentType = "document"
		}
		content = msg.Caption
		attachments = append(attachments, nats.AttachmentData{
			Type:     msg.Type,
			URL:      msg.Media.Ref, // Tinode ref — fetchable URL resolution is a Direto follow-up
			Filename: msg.Media.Name,
			MimeType: msg.Media.Mime,
			Metadata: map[string]string{"tinode_ref": msg.Media.Ref},
		})
	} else {
		contentType = "text"
		if msg.Text != nil {
			content = msg.Text.Body
		}
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		ChannelType: string(entity.ChannelTypeDireto),
		ExternalID:  msg.ID,
		ContentType: contentType,
		Content:     content,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}
	return h.publishInbound(c.Request.Context(), inbound)
}
