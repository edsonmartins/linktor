package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newConnPair creates a connected client/server websocket pair in-process.
// The server side is handed to the caller (to wrap in a Bridge); the client
// side is returned for speaking the protocol from the "device".
func newConnPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	var serverConn *websocket.Conn
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConn = conn
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = clientConn.Close() })

	deadline := time.Now().Add(2 * time.Second)
	for serverConn == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if serverConn == nil {
		t.Fatal("server connection never established")
	}
	return clientConn, serverConn
}

func unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func writeJSON(conn *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

func waitForWelcome(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	defer conn.SetReadDeadline(time.Time{})
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	var f Frame
	if err := unmarshal(data, &f); err != nil {
		t.Fatalf("parse welcome: %v", err)
	}
	if f.Type != "welcome" {
		t.Fatalf("expected welcome, got %q", f.Type)
	}
}
