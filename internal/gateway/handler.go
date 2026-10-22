package gateway

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/plugin"
)

// StatusUpdater is the subset of the channel service the gateway needs to
// reflect bridge connection state onto the channel (DB status + notifications).
type StatusUpdater interface {
	UpdateBridgeConnectionStatus(ctx context.Context, channelID string, status entity.ConnectionStatus) error
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// The bridge authenticates with token in the query string; origin checks are
	// not meaningful for a device connecting out. Enforce transport security at
	// the ingress layer.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler serves bridge WebSocket connections under /api/v1/gateways/ws.
type Handler struct {
	hub      *Hub
	channels repository.ChannelRepository
	producer nats.Publisher
	status   StatusUpdater
	registry *plugin.Registry
	// rehost move os anexos inline (base64) para o object store antes de
	// publicar, como o e-mail faz: o NATS limita o payload a 1 MB.
	rehost func(context.Context, *nats.InboundMessage)
}

// SetRehost registra o passo que move anexos inline para o object store antes
// da publicação no NATS.
func (h *Handler) SetRehost(fn func(context.Context, *nats.InboundMessage)) {
	h.rehost = fn
}

const (
	// maxInboundAttachmentBytes é o maior anexo aceito do bridge, em bytes
	// decodificados. Fotos do WhatsApp têm no máximo 16 MB; vídeos e documentos
	// maiores chegam só com nome, tipo e tamanho.
	maxInboundAttachmentBytes = 32 << 20
	// maxInlineBytes é quanto base64 ainda pode seguir no NATS quando não há
	// object store configurado (mesmo orçamento do e-mail).
	maxInlineBytes = 600 << 10
)

// HealthSummary returns the live bridge health snapshots for admin visibility.
func (h *Handler) HealthSummary() []HealthInfo {
	return h.hub.HealthAll()
}

// HealthForChannel returns the health snapshot of one channel's bridge.
func (h *Handler) HealthForChannel(channelID string) HealthInfo {
	return h.hub.Health(channelID)
}

// NewHandler builds the gateway WS handler.
func NewHandler(hub *Hub, channels repository.ChannelRepository, producer nats.Publisher, status StatusUpdater, registry *plugin.Registry) *Handler {
	dh := &Handler{hub: hub, channels: channels, producer: producer, status: status, registry: registry}
	// Wire the platform sinks once; idempotent across handler instances since
	// they all delegate to the same service layer.
	hub.SetInboundHandler(dh.handleInbound)
	hub.SetStatusHandler(dh.handleStatus)
	return dh
}

// InboundEvent is the sanitized message payload the bridge forwards.
type InboundEvent struct {
	Type       string `json:"type"` // "message" | "receipt" | "connection" | "logout"
	ID         string `json:"id"`
	From       string `json:"from"`
	SenderName string `json:"sender_name"`
	Chat       string `json:"chat"`
	// SenderLID é o LID do remetente quando o bridge traduziu From para o
	// número; vazio quando From já era o número ou quando só há o LID.
	SenderLID string `json:"sender_lid,omitempty"`
	// IsFromMe é mensagem digitada no aparelho pareado: From já é o cliente e
	// OperatorName quem digitou. Vira is_from_me, que o normalizer arquiva como
	// mensagem do atendente.
	IsFromMe     bool   `json:"is_from_me,omitempty"`
	OperatorName string `json:"operator_name,omitempty"`
	Text         string `json:"text"`
	Timestamp    int64  `json:"ts"`
	// ContentType é o tipo da mensagem (text, image, video, audio, document);
	// vazio vale text. MsgType guarda a variante do WhatsApp (ptt, sticker).
	ContentType string              `json:"content_type,omitempty"`
	MsgType     string              `json:"msg_type,omitempty"`
	Attachments []InboundAttachment `json:"attachments,omitempty"`
}

// InboundAttachment é um anexo baixado e decifrado no device. Data traz os
// bytes em base64; Error explica quando o bridge não conseguiu baixar.
type InboundAttachment struct {
	Type      string `json:"type"`
	MimeType  string `json:"mime_type,omitempty"`
	Filename  string `json:"filename,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Data      string `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ServeHTTP is the entry point for GET /api/v1/gateways/ws: authenticate,
// upgrade, run the bridge read loop and clean up on close.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	token := r.URL.Query().Get("token")
	if channelID == "" || token == "" {
		http.Error(w, "channel_id and token are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	channel, err := h.channels.FindByID(ctx, channelID)
	if err != nil {
		http.Error(w, "channel not found", http.StatusUnauthorized)
		return
	}
	expected := channel.Credentials["bridge_token"]
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
		http.Error(w, "invalid bridge token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("gateway: upgrade failed for %s: %v", channelID, err)
		return
	}

	bridge := NewBridge(channelID, conn)
	bridge.Platform = r.URL.Query().Get("platform")
	bridge.Version = r.URL.Query().Get("version")

	// Register the remote adapter so the unified outbound worker resolves this
	// channel to the bridge (moving its delivery out of any embedded whatsmeow).
	if h.registry != nil {
		h.registry.RegisterChannelAdapter(channelID, NewRemoteAdapter(h.hub, channelID))
	}

	// Reflect connection state on the channel.
	if h.status != nil {
		_ = h.status.UpdateBridgeConnectionStatus(ctx, channelID, entity.ConnectionStatusConnected)
	}

	h.hub.ServeConnection(bridge)

	// Connection closed. The RemoteAdapter stays registered: while the bridge is
	// offline it reports IsConnected=false and returns a transient "no live
	// bridge" error, so the outbound worker keeps redelivering instead of
	// silently falling through to an embedded adapter that was never connected.
	// Removing the registry entry here would also delete a previously embedded
	// adapter slot and send subsequent messages down the wrong fallback path.
	if h.status != nil {
		_ = h.status.UpdateBridgeConnectionStatus(context.Background(), channelID, entity.ConnectionStatusDisconnected)
	}
}

// handleInbound converts a bridge inbound payload into a nats.InboundMessage and
// publishes it, reusing the existing inbound pipeline untouched.
func (h *Handler) handleInbound(channelID string, raw json.RawMessage) error {
	if h.producer == nil || len(raw) == 0 {
		return nil
	}
	var evt InboundEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return fmt.Errorf("parse inbound payload: %w", err)
	}
	if evt.ID == "" || (evt.Text == "" && len(evt.Attachments) == 0) {
		return nil // ignore receipts/connection frames forwarded without content
	}

	channel, err := h.channels.FindByID(context.Background(), channelID)
	if err != nil {
		return fmt.Errorf("resolve channel: %w", err)
	}

	// Build the same identity contract as the embedded whatsapp adapter: the
	// stable contact key is the phone-only user part (JID.User), with the full
	// JIDs preserved in metadata for group/1:1 context. Parsing is lenient so a
	// bare number from the bridge still works.
	var (
		senderID    = evt.From
		chatJID     = evt.Chat
		senderPhone string
	)
	// Um remetente que só veio como LID (o bridge não conseguiu traduzir para
	// número) fica com o JID inteiro como chave: sem o sufixo @lid a resposta
	// seria endereçada a um número que não existe.
	if strings.HasSuffix(evt.From, "@lid") {
		senderID = evt.From
	} else if from, err := jidUserOrBare(evt.From); err == nil && from != "" {
		senderID = from
		// Mesmo contrato do webhook oficial: com o número em "phone" o contato
		// nasce com telefone e casa com o mesmo número vindo de outro canal.
		senderPhone = from
	}
	if chatJID == "" {
		chatJID = evt.From
	}

	metadata := map[string]string{
		"sender_id":  senderID,
		"sender_jid": evt.From,
		"chat_jid":   chatJID,
	}
	if senderPhone != "" {
		metadata["phone"] = senderPhone
	}
	if evt.SenderLID != "" {
		metadata["sender_lid"] = evt.SenderLID
	}
	if evt.SenderName != "" {
		metadata["sender_name"] = evt.SenderName
	}
	if evt.IsFromMe {
		metadata["is_from_me"] = "true"
		if evt.OperatorName != "" {
			metadata["operator_name"] = evt.OperatorName
		}
	}
	// Mesmas chaves do adaptador WhatsApp embutido, para o resto do pipeline
	// não distinguir de onde a sessão veio.
	if evt.MsgType != "" {
		metadata["msg_type"] = evt.MsgType
	}
	if evt.MsgType == "sticker" {
		metadata["is_sticker"] = "true"
	}
	contentType := evt.ContentType
	if contentType == "" {
		contentType = string(plugin.ContentTypeText)
	}
	inbound := &nats.InboundMessage{
		ID:          evt.ID,
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: string(channel.Type),
		ExternalID:  evt.ID,
		ContentType: contentType,
		Content:     evt.Text,
		Metadata:    metadata,
		Attachments: inboundAttachments(evt.Attachments),
		Timestamp:   time.Unix(evt.Timestamp, 0),
	}
	ctx := context.Background()
	if h.rehost != nil && len(inbound.Attachments) > 0 {
		h.rehost(ctx, inbound)
	}
	for _, name := range nats.TrimInlineAttachments(inbound.Attachments, maxInlineBytes) {
		log.Printf("gateway: anexo %q do canal %s sem object store e grande demais para a fila; segue só o registro", name, channelID)
	}
	return h.producer.PublishInbound(ctx, inbound)
}

// inboundAttachments converte os anexos do bridge para o contrato inline do
// pipeline (base64 em metadata, que rehostInboundMedia move para o store).
// Anexos acima do limite ou que o bridge não conseguiu baixar seguem só com
// nome, tipo e tamanho, mais o motivo em download_error.
func inboundAttachments(in []InboundAttachment) []nats.AttachmentData {
	if len(in) == 0 {
		return nil
	}
	out := make([]nats.AttachmentData, 0, len(in))
	for _, a := range in {
		att := nats.AttachmentData{
			Type:      a.Type,
			Filename:  a.Filename,
			MimeType:  a.MimeType,
			SizeBytes: a.SizeBytes,
			Metadata:  map[string]string{},
		}
		if att.Type == "" {
			att.Type = storage.AttachmentTypeFromMime(a.MimeType)
		}
		switch {
		case a.Error != "":
			att.Metadata["download_error"] = a.Error
		case a.Data == "":
			att.Metadata["download_error"] = "bridge forwarded no data"
		case base64DecodedLen(a.Data) > maxInboundAttachmentBytes:
			att.Metadata["download_error"] = fmt.Sprintf("attachment exceeds %d MB", maxInboundAttachmentBytes>>20)
		default:
			att.Metadata[nats.AttachmentDataKey] = a.Data
			att.Metadata[nats.AttachmentDataEncodingKey] = nats.AttachmentDataBase64
			if att.SizeBytes == 0 {
				att.SizeBytes = int64(base64DecodedLen(a.Data))
			}
		}
		out = append(out, att)
	}
	return out
}

// base64DecodedLen estima o tamanho decodificado sem decodificar.
func base64DecodedLen(data string) int {
	n := len(data) / 4 * 3
	if strings.HasSuffix(data, "==") {
		return n - 2
	}
	if strings.HasSuffix(data, "=") {
		return n - 1
	}
	return n
}

// jidUserOrBare returns the phone/user part of a JID (everything before @ and
// stripping s.whatsapp.net), or the input unchanged when it is not a JID.
func jidUserOrBare(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty jid")
	}
	if at := strings.LastIndex(value, "@"); at > 0 {
		return value[:at], nil
	}
	return value, nil
}

// handleStatus reflects a bridge status frame onto the channel.
func (h *Handler) handleStatus(channelID string, raw json.RawMessage) error {
	if len(raw) == 0 || h.status == nil {
		return nil
	}
	var st StatusEvent
	if err := json.Unmarshal(raw, &st); err != nil {
		return fmt.Errorf("parse status payload: %w", err)
	}
	status := entity.ConnectionStatusDisconnected
	if st.Connected {
		status = entity.ConnectionStatusConnected
	}
	return h.status.UpdateBridgeConnectionStatus(context.Background(), channelID, status)
}
