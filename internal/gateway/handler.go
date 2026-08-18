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
}

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
	Text       string `json:"text"`
	Timestamp  int64  `json:"ts"`
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
	if evt.ID == "" || evt.Text == "" {
		return nil // ignore receipts/connection frames forwarded without text
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
		senderID = evt.From
		chatJID  = evt.Chat
	)
	if from, err := jidUserOrBare(evt.From); err == nil && from != "" {
		senderID = from
	}
	if chatJID == "" {
		chatJID = evt.From
	}

	metadata := map[string]string{
		"sender_id":  senderID,
		"sender_jid": evt.From,
		"chat_jid":   chatJID,
	}
	if evt.SenderName != "" {
		metadata["sender_name"] = evt.SenderName
	}
	inbound := &nats.InboundMessage{
		ID:          evt.ID,
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: string(channel.Type),
		ExternalID:  evt.ID,
		ContentType: "text",
		Content:     evt.Text,
		Metadata:    metadata,
		Timestamp:   time.Unix(evt.Timestamp, 0),
	}
	return h.producer.PublishInbound(context.Background(), inbound)
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
