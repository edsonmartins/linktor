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

// TestRemoteAdapterUnsupportedContent verifies that what the protocol does not
// carry (interactive, template) is a permanent failure, and that media without
// a URL (nothing for the bridge to download) is permanent too.
func TestRemoteAdapterUnsupportedContent(t *testing.T) {
	hub := NewHub()
	adapter := NewRemoteAdapter(hub, "wh-xyz")
	for _, msg := range []*plugin.OutboundMessage{
		{RecipientID: "5511999999999", ContentType: plugin.ContentTypeInteractive, Content: "x"},
		{RecipientID: "5511999999999", ContentType: plugin.ContentTypeImage, Content: "sem url"},
	} {
		_, err := adapter.SendMessage(context.Background(), msg)
		if err == nil || !outbound.IsPermanent(err) {
			t.Fatalf("%s: expected permanent error, got %v", msg.ContentType, err)
		}
	}
}

// TestRemoteAdapterSendMedia verifies an image with a signed URL goes over the
// bridge as an outbound frame carrying media (caption in text) and the ack is
// returned as the receipt.
func TestRemoteAdapterSendMedia(t *testing.T) {
	hub := NewHub()
	clientConn, serverConn := newConnPair(t)
	defer clientConn.Close()
	defer serverConn.Close()
	go hub.ServeConnection(NewBridge("wh-abc", serverConn))
	waitForWelcome(t, clientConn)

	adapter := NewRemoteAdapter(hub, "wh-abc")
	errCh := make(chan error, 1)
	go func() {
		_, err := adapter.SendMessage(context.Background(), &plugin.OutboundMessage{
			ID:          "m_2",
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeImage,
			Content:     "a foto",
			Metadata:    map[string]string{"ptt": "false"},
			Attachments: []*plugin.Attachment{{
				Type: "image", URL: "https://linktor.test/api/v1/media/k.jpg?exp=1&sig=x",
				Filename: "foto.jpg", MimeType: "image/jpeg",
			}},
		})
		errCh <- err
	}()

	_, data, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read outbound: %v", err)
	}
	var f Frame
	if err := unmarshal(data, &f); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f.Type != "outbound" || f.Text != "a foto" || f.Media == nil {
		t.Fatalf("unexpected frame: %+v", f)
	}
	if f.Media.Type != "image" || f.Media.MimeType != "image/jpeg" || f.Media.Filename != "foto.jpg" || f.Media.URL == "" || f.Media.PTT {
		t.Fatalf("unexpected media: %+v", f.Media)
	}
	writeJSON(clientConn, Frame{Type: "ack", CorrelationID: "m_2", OK: true, MessageID: "WAMID-2"})
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for send result")
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
