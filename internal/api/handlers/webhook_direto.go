package handlers

import (
	"bytes"
	"context"
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
	// Eventos EFÊMEROS (typing/presença) — RFC-009. Não são mensagens.
	Events []diretoWebhookEvent `json:"events,omitempty"`
}

// diretoWebhookEvent é um sinal efêmero: type="typing"|"presence"; from=telefone E.164 do cliente;
// state="on" (typing) | "active"|"away" (presença).
type diretoWebhookEvent struct {
	Type  string `json:"type"`
	From  string `json:"from"`
	State string `json:"state"`
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

	// Sinais efêmeros (typing/presença) — RFC-009. Best-effort: não afetam o status da resposta.
	for _, ev := range payload.Events {
		h.processDiretoEvent(c.Request.Context(), channel, ev)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// processDiretoEvent mostra o "digitando" do cliente na UI do agente. Resolve telefone→contato→
// conversa aberta e dispara o typing indicator. No-op se as deps de typing não estiverem ligadas
// (SetTypingDeps) ou não houver contato/conversa. Presença (active/away) é recebida mas ainda não
// exibida (follow-up) — só typing por ora.
func (h *WebhookHandler) processDiretoEvent(ctx context.Context, channel *entity.Channel, ev diretoWebhookEvent) {
	if h.typingSvc == nil || h.contactRepo == nil || h.conversationRepo == nil {
		return
	}
	if ev.Type != "typing" || ev.From == "" {
		return
	}
	contact, err := h.contactRepo.FindByPhone(ctx, channel.TenantID, ev.From)
	if err != nil || contact == nil {
		return // cliente desconhecido: nada a mostrar
	}
	conv, err := h.conversationRepo.FindOpenByContactAndChannel(ctx, contact.ID, channel.ID)
	if err != nil || conv == nil {
		return // sem conversa aberta: nada a mostrar
	}
	_ = h.typingSvc.SendTypingIndicatorForTenant(ctx, channel.TenantID, conv.ID, ev.State == "on")
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
