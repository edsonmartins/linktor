package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestHubSendTextAck verifies the correlation machinery: an outbound send waits
// for the ack and returns the provider message ID, and an unrelated ack does not
// resolve it.
func TestHubSendTextAck(t *testing.T) {
	hub := NewHub()

	// Server-side: accept a WS connection and speak the protocol like a device.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		bridge := NewBridge("wh-abc", conn)
		go hub.ServeConnection(bridge)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Wait for welcome.
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	var welcome Frame
	if err := json.Unmarshal(data, &welcome); err != nil || welcome.Type != "welcome" {
		t.Fatalf("expected welcome, got %s (%v)", welcome.Type, err)
	}

	// Issue an outbound send in a goroutine (it blocks until ack).
	type sendResult struct {
		ack AckResult
		err error
	}
	resultCh := make(chan sendResult, 1)
	go func() {
		ack, err := hub.SendToChannel(context.Background(), "wh-abc", "c_1", "5511999999999", "olá", "")
		resultCh <- sendResult{ack, err}
	}()

	// The bridge should receive an outbound frame.
	_, data, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read outbound: %v", err)
	}
	var outbound Frame
	if err := json.Unmarshal(data, &outbound); err != nil {
		t.Fatalf("parse outbound: %v", err)
	}
	if outbound.Type != "outbound" || outbound.CorrelationID != "c_1" || outbound.To != "5511999999999" || outbound.Text != "olá" {
		t.Fatalf("unexpected outbound frame: %+v", outbound)
	}

	// Reply with an ack carrying the provider message ID.
	ackPayload, _ := json.Marshal(Frame{
		Type: "ack", CorrelationID: "c_1", OK: true, MessageID: "3EB0", ChannelID: "wh-abc",
	})
	if err := conn.WriteMessage(websocket.TextMessage, ackPayload); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("SendToChannel: %v", res.err)
		}
		if !res.ack.OK || res.ack.MessageID != "3EB0" {
			t.Fatalf("unexpected ack: %+v", res.ack)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ack")
	}
}

// TestHubSendToChannelNoBridge verifies a transient error is returned when the
// channel has no live bridge (the outbound worker will retry).
func TestHubSendToChannelNoBridge(t *testing.T) {
	hub := NewHub()
	_, err := hub.SendToChannel(context.Background(), "wh-xyz", "c_2", "5511999999999", "oi", "")
	if err == nil || !strings.Contains(err.Error(), "no live bridge") {
		t.Fatalf("expected no-live-bridge error, got %v", err)
	}
}

// TestHubInboundSink verifies inbound frames are routed to the configured sink.
func TestHubInboundSink(t *testing.T) {
	hub := NewHub()
	var got json.RawMessage
	var gotChannel string
	hub.SetInboundHandler(func(channelID string, raw json.RawMessage) error {
		gotChannel = channelID
		got = raw
		return nil
	})

	bridge := NewBridge("wh-abc", nil) // no real connection; dispatch directly
	bridge.owner = hub
	payload := json.RawMessage(`{"type":"message","id":"M1","from":"5511999999999@s.whatsapp.net","text":"oi","ts":1}`)
	bridge.handleFrame(Frame{Type: "inbound", ChannelID: "wh-abc", Message: payload})

	if gotChannel != "wh-abc" {
		t.Fatalf("inbound routed to %q, want wh-abc", gotChannel)
	}
	if !strings.Contains(string(got), `"id":"M1"`) {
		t.Fatalf("inbound payload not forwarded: %s", got)
	}
}

// TestHubHeartbeatPing registers a bridge with a live socket pair, sends an
// application heartbeat (ping) and verifies: the pong is returned, the status
// is captured on the bridge, and the health snapshot reports non-stale liveness.
func TestHubHeartbeatPing(t *testing.T) {
	hub := NewHub()
	clientConn, serverConn := newConnPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	bridge := NewBridge("wh-abc", serverConn)
	go hub.ServeConnection(bridge)
	waitForWelcome(t, clientConn)

	status := json.RawMessage(`{"paired":true,"connected":true,"number":"5511999999999"}`)
	if err := writeJSON(clientConn, Frame{
		Type: "ping", ChannelID: "wh-abc", TS: time.Now().Unix(), Status: status,
	}); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	defer clientConn.SetReadDeadline(time.Time{})
	_, data, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read pong: %v", err)
	}
	var pong Frame
	if err := json.Unmarshal(data, &pong); err != nil {
		t.Fatalf("parse pong: %v", err)
	}
	if pong.Type != "pong" {
		t.Fatalf("expected pong, got %q", pong.Type)
	}

	health := hub.Health("wh-abc")
	if !health.Online {
		t.Fatal("bridge should be online")
	}
	if health.Stale {
		t.Fatal("bridge marked stale right after a heartbeat")
	}
	if !strings.Contains(string(health.LastSession), `"connected":true`) {
		t.Fatalf("last session status not captured: %s", health.LastSession)
	}
	if health.LastSeenPing.IsZero() {
		t.Fatal("last_seen_ping not recorded")
	}
}

// TestHubHealthOffline verifies an offline channel reports online=false.
func TestHubHealthOffline(t *testing.T) {
	hub := NewHub()
	health := hub.Health("wh-xyz")
	if health.Online {
		t.Fatal("expected offline health for a channel with no bridge")
	}
}
