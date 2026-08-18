// Package gateway implements the Linktor server side of the Bridge Protocol
// (see bridges/docs/PROTOCOL.md). It owns live bridge WebSocket connections per
// channel, forwards outbound send requests to the connected bridge and waits
// for the ack, and feeds inbound/status events into the platform.
package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/outbound"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Frame mirrors the JSON envelope of the Bridge Protocol. Only the fields the
// server consumes/emits are modeled here; unknown fields are ignored.
type Frame struct {
	Type          string          `json:"type"`
	ChannelID     string          `json:"channel_id"`
	Platform      string          `json:"platform"`
	Version       string          `json:"version"`
	OK            bool            `json:"ok"`
	Error         string          `json:"error"`
	Reason        string          `json:"reason"`
	CorrelationID string          `json:"correlation_id"`
	To            string          `json:"to"`
	Text          string          `json:"text"`
	MessageID     string          `json:"message_id"`
	Message       json.RawMessage `json:"message"`
	Status        json.RawMessage `json:"status"`
	TS            int64           `json:"ts,omitempty"`
}

// AckResult is the outcome of an outbound send, delivered to the waiter keyed
// by correlation ID.
type AckResult struct {
	OK        bool
	MessageID string
	Error     string
}

// StatusEvent is a sanitized session status received from a bridge.
type StatusEvent struct {
	Paired    bool   `json:"paired"`
	Connected bool   `json:"connected"`
	Number    string `json:"number,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

const (
	writeTimeout = 10 * time.Second
	ackTimeout   = 60 * time.Second
)

// Hub tracks connected bridges and routes outbound sends through them.
// One hub serves all channels; a channel can have at most one live bridge.
type Hub struct {
	mu      sync.RWMutex
	bridges map[string]*Bridge // channelID -> bridge
	// onInbound, when set, is called with the channelID and the raw inbound
	// message JSON so a service can publish it into the platform.
	onInbound func(channelID string, raw json.RawMessage) error
	onStatus  func(channelID string, raw json.RawMessage) error
}

// NewHub creates a Hub.
func NewHub() *Hub {
	return &Hub{bridges: make(map[string]*Bridge)}
}

// SetInboundHandler wires the inbound event sink. Overwrites any previous.
func (h *Hub) SetInboundHandler(fn func(channelID string, raw json.RawMessage) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onInbound = fn
}

// SetStatusHandler wires the status event sink.
func (h *Hub) SetStatusHandler(fn func(channelID string, raw json.RawMessage) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onStatus = fn
}

// Register binds a live bridge to a channel. If the channel already has a
// bridge, the previous one is replaced (its connection is closed).
func (h *Hub) Register(b *Bridge) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.bridges[b.ChannelID]; ok && old != b {
		closeConn(old.conn, closeRestart, "replaced by new bridge")
	}
	b.owner = h
	h.bridges[b.ChannelID] = b
}

// Unregister removes a bridge binding if it is still the registered one
// for the channel (avoiding tearing down a replacement).
func (h *Hub) Unregister(b *Bridge) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cur, ok := h.bridges[b.ChannelID]; ok && cur == b {
		delete(h.bridges, b.ChannelID)
	}
}

// IsOnline reports whether a channel currently has a live bridge.
func (h *Hub) IsOnline(channelID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.bridges[channelID]
	return ok
}

// OnlineChannels lists channel IDs with a live bridge.
func (h *Hub) OnlineChannels() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.bridges))
	for id := range h.bridges {
		out = append(out, id)
	}
	return out
}

// closeConn sends a WebSocket close handshake (best-effort) and drops the
// connection. gorilla v1.5.1 has no Close(code, reason): the close payload is
// carried as a CloseMessage frame.
func closeConn(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	_ = conn.Close()
}

const (
	closeNormal  = websocket.CloseNormalClosure
	closeRestart = 1012 // service restart
)

// Bridge holds one live WebSocket connection to a device bridge and its
// per-connection send/ack machinery.
type Bridge struct {
	ChannelID string
	Platform  string
	Version   string

	owner *Hub

	conn      *websocket.Conn
	writeMu   sync.Mutex
	pending   map[string]chan AckResult
	pendingMu sync.Mutex
	closeOnce sync.Once

	statusMu    sync.Mutex
	lastStatus  json.RawMessage
	connectedAt time.Time
	lastSeen    time.Time
}

// NewBridge creates a Bridge wrapper for a websocket connection.
func NewBridge(channelID string, conn *websocket.Conn) *Bridge {
	return &Bridge{
		ChannelID: channelID,
		conn:      conn,
		pending:   make(map[string]chan AckResult),
	}
}

// setConnectedAt marks when this bridge connected (used for health reporting).
func (b *Bridge) markConnected() {
	b.statusMu.Lock()
	defer b.statusMu.Unlock()
	if b.connectedAt.IsZero() {
		b.connectedAt = time.Now()
	}
	b.lastStatus = nil
}

// Close closes the underlying connection once.
func (b *Bridge) Close(code int, reason string) {
	b.closeOnce.Do(func() {
		closeConn(b.conn, code, reason)
	})
}

// resolveAck delivers an ack to the waiting outbound send, if any.
func (b *Bridge) resolveAck(correlationID string, ack AckResult) {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()
	if ch, ok := b.pending[correlationID]; ok {
		ch <- ack
	}
}

// SendText sends an outbound command to the bridge and waits for its ack.
// Returns the WhatsApp message ID on success. messageID may be empty (the
// bridge generates one) or a caller-persisted ID reused for idempotent retry.
func (b *Bridge) SendText(ctx context.Context, correlationID, to, text, messageID string) (AckResult, error) {
	ackCh := make(chan AckResult, 1)
	b.pendingMu.Lock()
	b.pending[correlationID] = ackCh
	b.pendingMu.Unlock()
	defer func() {
		b.pendingMu.Lock()
		delete(b.pending, correlationID)
		b.pendingMu.Unlock()
	}()

	f := Frame{
		Type:          "outbound",
		ChannelID:     b.ChannelID,
		CorrelationID: correlationID,
		To:            to,
		Text:          text,
		MessageID:     messageID,
	}
	if err := b.write(f); err != nil {
		return AckResult{}, fmt.Errorf("write outbound to bridge: %w", err)
	}

	select {
	case ack := <-ackCh:
		return ack, nil
	case <-ctx.Done():
		return AckResult{}, ctx.Err()
	case <-time.After(ackTimeout):
		return AckResult{}, errors.New("bridge did not acknowledge in time")
	}
}

// write sends a frame, serializing concurrent writers.
func (b *Bridge) write(f Frame) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	_ = b.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return b.conn.WriteMessage(websocket.TextMessage, data)
}

// handleFrame processes one frame received from the bridge.
func (b *Bridge) handleFrame(f Frame) {
	// Any traffic proves liveness; the app-level heartbeat is the primary signal.
	b.statusMu.Lock()
	b.lastSeen = time.Now()
	b.statusMu.Unlock()

	switch f.Type {
	case "ping":
		b.statusMu.Lock()
		if len(f.Status) > 0 {
			b.lastStatus = append(json.RawMessage(nil), f.Status...)
		}
		b.statusMu.Unlock()
		_ = b.write(Frame{Type: "pong", ChannelID: b.ChannelID, TS: time.Now().Unix()})
	case "ack":
		b.resolveAck(f.CorrelationID, AckResult{OK: f.OK, MessageID: f.MessageID, Error: f.Error})
	case "inbound":
		if b.owner != nil && b.owner.onInbound != nil {
			if err := b.owner.onInbound(b.ChannelID, f.Message); err != nil {
				log.Printf("gateway: inbound handler failed for %s: %v", b.ChannelID, err)
			}
		}
	case "status":
		b.statusMu.Lock()
		b.lastStatus = append(json.RawMessage(nil), f.Status...)
		b.statusMu.Unlock()
		if b.owner != nil && b.owner.onStatus != nil {
			if err := b.owner.onStatus(b.ChannelID, f.Status); err != nil {
				log.Printf("gateway: status handler failed for %s: %v", b.ChannelID, err)
			}
		}
	case "hello":
		if f.Platform != "" {
			b.Platform = f.Platform
		}
		if f.Version != "" {
			b.Version = f.Version
		}
	default:
		// welcome/bye/close handled by the connection loop owner.
	}
}

// ServeConnection runs the read loop for a bridge connection until it closes.
// It registers the bridge in the hub, emits welcome once and resets pending
// ack state on disconnect.
func (h *Hub) ServeConnection(b *Bridge) {
	h.Register(b)
	defer h.Unregister(b)

	b.markConnected()
	_ = b.write(Frame{Type: "welcome", OK: true})

	for {
		_, data, err := b.conn.ReadMessage()
		if err != nil {
			break
		}
		var f Frame
		if err := json.Unmarshal(data, &f); err != nil {
			log.Printf("gateway: invalid frame from %s: %v", b.ChannelID, err)
			continue
		}
		if f.ChannelID != "" && f.ChannelID != b.ChannelID {
			_ = b.write(Frame{Type: "close", Reason: "channel_mismatch"})
			break
		}
		switch f.Type {
		case "bye", "close":
			closeConn(b.conn, closeNormal, "bye")
			return
		default:
			b.handleFrame(f)
		}
	}
}

// SendToChannel is the exported entry point the RemoteSender uses: it resolves
// the channel's live bridge and performs the send, waiting for the ack.
func (h *Hub) SendToChannel(ctx context.Context, channelID, correlationID, to, text, messageID string) (AckResult, error) {
	h.mu.RLock()
	b, ok := h.bridges[channelID]
	h.mu.RUnlock()
	if !ok {
		return AckResult{}, fmt.Errorf("no live bridge for channel %s", channelID)
	}
	return b.SendText(ctx, correlationID, to, text, messageID)
}

// HealthInfo is the sanitized health snapshot of a bridge, for the admin API/UI.
type HealthInfo struct {
	ChannelID    string          `json:"channel_id"`
	Online       bool            `json:"online"`
	Platform     string          `json:"platform,omitempty"`
	Version      string          `json:"version,omitempty"`
	ConnectedAt  time.Time       `json:"connected_at,omitempty"`
	LastSession  json.RawMessage `json:"last_session,omitempty"` // raw wacore.Status from the device
	LastSeenPing time.Time       `json:"last_seen_ping,omitempty"`
	// Stale is true when the socket is up but no application heartbeat arrived
	// within heartbeatStaleAfter — the bridge is likely hung, not just offline.
	Stale bool `json:"stale"`
}

// heartbeatStaleAfter bounds how long we wait for an app-level heartbeat before
// flagging a bridge as stale. The device sends one every 20s.
const heartbeatStaleAfter = 90 * time.Second

// Health returns the health snapshot for a channel, or an offline entry when no
// bridge is connected.
func (h *Hub) Health(channelID string) HealthInfo {
	h.mu.RLock()
	b, ok := h.bridges[channelID]
	h.mu.RUnlock()
	if !ok {
		return HealthInfo{ChannelID: channelID, Online: false}
	}
	b.statusMu.Lock()
	info := HealthInfo{
		ChannelID:    channelID,
		Online:       true,
		Platform:     b.Platform,
		Version:      b.Version,
		ConnectedAt:  b.connectedAt,
		LastSession:  append(json.RawMessage(nil), b.lastStatus...),
		LastSeenPing: b.lastSeen,
	}
	if b.lastSeen.IsZero() {
		info.LastSeenPing = b.connectedAt
	}
	info.Stale = !info.LastSeenPing.IsZero() && time.Since(info.LastSeenPing) > heartbeatStaleAfter
	b.statusMu.Unlock()
	return info
}

// HealthAll returns health snapshots for every channel with a live bridge.
func (h *Hub) HealthAll() []HealthInfo {
	out := make([]HealthInfo, 0, 8)
	for _, channelID := range h.OnlineChannels() {
		out = append(out, h.Health(channelID))
	}
	return out
}

// RemoteAdapter bridges the plugin.ChannelAdapter contract to a live bridge.
// It is registered per-channel in the global plugin registry while a bridge is
// online so the unified outbound worker resolves it exactly like an embedded
// adapter.
type RemoteAdapter struct {
	hub       *Hub
	channelID string
}

// NewRemoteAdapter creates a RemoteAdapter for a channel.
func NewRemoteAdapter(hub *Hub, channelID string) *RemoteAdapter {
	return &RemoteAdapter{hub: hub, channelID: channelID}
}

var _ plugin.ChannelAdapter = (*RemoteAdapter)(nil)
var _ plugin.ReactionSender = (*RemoteAdapter)(nil)

// Initialize implements plugin.ChannelAdapter.
func (a *RemoteAdapter) Initialize(config map[string]string) error { return nil }

// Connect implements plugin.ChannelAdapter. The bridge owns connectivity.
func (a *RemoteAdapter) Connect(ctx context.Context) error { return nil }

// Disconnect implements plugin.ChannelAdapter. The bridge owns connectivity.
func (a *RemoteAdapter) Disconnect(ctx context.Context) error { return nil }

// IsConnected implements plugin.ChannelAdapter.
func (a *RemoteAdapter) IsConnected() bool { return a.hub.IsOnline(a.channelID) }

// GetConnectionStatus implements plugin.ChannelAdapter.
func (a *RemoteAdapter) GetConnectionStatus() *plugin.ConnectionStatus {
	connected := a.hub.IsOnline(a.channelID)
	status := "disconnected"
	if connected {
		status = "connected"
	}
	return &plugin.ConnectionStatus{Connected: connected, Status: status}
}

// SendMessage implements plugin.ChannelAdapter. Only text is supported for now;
// media/interactive/template over the bridge is a follow-up milestone.
func (a *RemoteAdapter) SendMessage(ctx context.Context, msg *plugin.OutboundMessage) (*plugin.SendResult, error) {
	if msg == nil || msg.RecipientID == "" {
		return nil, outbound.Permanentf("recipient is required")
	}
	if msg.ContentType != plugin.ContentTypeText {
		return nil, outbound.Permanentf("bridge currently supports text outbound only (got %s)", msg.ContentType)
	}

	ack, err := a.hub.SendToChannel(ctx, a.channelID, msg.ID, msg.RecipientID, msg.Content, "")
	if err != nil {
		return nil, err // transient: worker retries
	}
	if !ack.OK {
		// A rejection from the device is permanent: retrying the same payload
		// against the same session won't succeed.
		return nil, outbound.Permanentf("bridge rejected send: %s", ack.Error)
	}
	return &plugin.SendResult{
		Success:    true,
		ExternalID: ack.MessageID,
		Status:     plugin.MessageStatusSent,
		Timestamp:  time.Now(),
	}, nil
}

// SendReaction implements plugin.ReactionSender. Not yet transported over the
// bridge protocol; override to forward when supported.
func (a *RemoteAdapter) SendReaction(ctx context.Context, reaction *plugin.OutboundReaction) error {
	return outbound.Permanentf("reactions over bridge are not supported yet")
}

// SendTypingIndicator implements plugin.ChannelAdapter (no-op for now).
func (a *RemoteAdapter) SendTypingIndicator(ctx context.Context, indicator *plugin.TypingIndicator) error {
	return nil
}

// SendReadReceipt implements plugin.ChannelAdapter (no-op for now).
func (a *RemoteAdapter) SendReadReceipt(ctx context.Context, receipt *plugin.ReadReceipt) error {
	return nil
}

// UploadMedia implements plugin.ChannelAdapter (not supported yet).
func (a *RemoteAdapter) UploadMedia(ctx context.Context, media *plugin.Media) (*plugin.MediaUpload, error) {
	return nil, outbound.Permanentf("media upload over bridge is not supported yet")
}

// DownloadMedia implements plugin.ChannelAdapter (not supported yet).
func (a *RemoteAdapter) DownloadMedia(ctx context.Context, mediaID string) (*plugin.Media, error) {
	return nil, outbound.Permanentf("media download over bridge is not supported yet")
}

// GetChannelType implements plugin.ChannelAdapter.
func (a *RemoteAdapter) GetChannelType() plugin.ChannelType { return plugin.ChannelTypeWhatsAppRemote }

// GetChannelInfo implements plugin.ChannelAdapter.
func (a *RemoteAdapter) GetChannelInfo() *plugin.ChannelInfo {
	return &plugin.ChannelInfo{
		Type:        plugin.ChannelTypeWhatsAppRemote,
		Name:        "WhatsApp (Linktor Bridge)",
		Description: "WhatsApp Multi-Device session hosted on the user's device",
		Version:     "0.1.0",
		Capabilities: &plugin.ChannelCapabilities{
			SupportedContentTypes: []plugin.ContentType{plugin.ContentTypeText},
			SupportsMedia:         false,
			MaxMessageLength:      65536,
		},
	}
}

// GetCapabilities implements plugin.ChannelAdapter.
func (a *RemoteAdapter) GetCapabilities() *plugin.ChannelCapabilities {
	return a.GetChannelInfo().Capabilities
}
