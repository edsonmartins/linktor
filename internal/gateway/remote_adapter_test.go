package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/outbound"
	"github.com/msgfy/linktor/pkg/plugin"
)

// TestRemoteAdapterSendMessage verifies that the adapter the outbound worker
// resolves through the registry forwards a text message to the live bridge and
// returns the provider message ID as the receipt.
func TestRemoteAdapterSendMessage(t *testing.T) {
	hub := NewHub()
	// Simulate a live bridge by registering one with a stub connection pair.
	clientConn, serverConn := newConnPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	bridge := NewBridge("wh-abc", serverConn)
	go hub.ServeConnection(bridge)
	waitForWelcome(t, clientConn)

	adapter := NewRemoteAdapter(hub, "wh-abc")

	type res struct {
		send *plugin.SendResult
		err  error
	}
	resCh := make(chan res, 1)
	go func() {
		send, err := adapter.SendMessage(context.Background(), &plugin.OutboundMessage{
			ID:          "m_1",
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeText,
			Content:     "olá bridge",
		})
		resCh <- res{send, err}
	}()

	// Read the outbound frame, ack it.
	_, data, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read outbound: %v", err)
	}
	var f Frame
	if err := unmarshal(data, &f); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f.CorrelationID != "m_1" {
		t.Fatalf("correlation = %q, want m_1", f.CorrelationID)
	}
	writeJSON(clientConn, Frame{Type: "ack", CorrelationID: "m_1", OK: true, MessageID: "WAMID-1"})

	select {
	case r := <-resCh:
		if r.err != nil {
			t.Fatalf("SendMessage: %v", r.err)
		}
		if !r.send.Success || r.send.ExternalID != "WAMID-1" {
			t.Fatalf("unexpected result: %+v", r.send)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for send result")
	}
}

// TestRemoteAdapterUnsupportedContent verifies media over the bridge is a
// permanent (non-retryable) failure.
func TestRemoteAdapterUnsupportedContent(t *testing.T) {
	hub := NewHub()
	adapter := NewRemoteAdapter(hub, "wh-xyz")
	_, err := adapter.SendMessage(context.Background(), &plugin.OutboundMessage{
		RecipientID: "5511999999999",
		ContentType: plugin.ContentTypeImage,
		Content:     "x",
	})
	if err == nil || !outbound.IsPermanent(err) {
		t.Fatalf("expected permanent error for media, got %v", err)
	}
}

// TestRemoteAdapterNoBridge verifies a send against a channel without a live
// bridge is transient (the worker retries it).
func TestRemoteAdapterNoBridge(t *testing.T) {
	hub := NewHub()
	adapter := NewRemoteAdapter(hub, "wh-xyz")
	_, err := adapter.SendMessage(context.Background(), &plugin.OutboundMessage{
		RecipientID: "5511999999999",
		ContentType: plugin.ContentTypeText,
		Content:     "oi",
	})
	if err == nil || outbound.IsPermanent(err) {
		t.Fatalf("expected transient error, got %v", err)
	}
}
