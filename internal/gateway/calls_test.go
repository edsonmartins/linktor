package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/require"
)

const caller = "5511999999999@s.whatsapp.net"

// newCallHandler monta o handler com um sink que entrega cada CallUpdate num
// canal, para o teste esperar a transição sem dormir.
func newCallHandler(t *testing.T, cfg map[string]string) (*Handler, *testutil.MockProducer, chan CallUpdate) {
	t.Helper()
	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-1"] = &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: entity.ChannelType(plugin.ChannelTypeWhatsAppRemote), Config: cfg}
	producer := testutil.NewMockProducer()
	h := NewHandler(NewHub(), channels, producer, nil, nil)
	updates := make(chan CallUpdate, 8)
	h.SetCallSink(func(_ context.Context, u CallUpdate) { updates <- u })
	return h, producer, updates
}

func callRaw(t *testing.T, id, kind string, ts int64) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(InboundEvent{Type: "call", ID: id, From: caller, Timestamp: ts, Summary: kind})
	require.NoError(t, err)
	return raw
}

func nextUpdate(t *testing.T, updates chan CallUpdate) CallUpdate {
	t.Helper()
	select {
	case u := <-updates:
		return u
	case <-time.After(2 * time.Second):
		t.Fatal("sink não recebeu a transição")
		return CallUpdate{}
	}
}

func noUpdate(t *testing.T, updates chan CallUpdate) {
	t.Helper()
	select {
	case u := <-updates:
		t.Fatalf("transição inesperada: %+v", u)
	case <-time.After(50 * time.Millisecond):
	}
}

// A oferta vira mensagem de tipo call na conversa do contato, com a mesma
// identidade das mensagens e o estado tocando; oferta repetida não duplica.
func TestCall_OfferBecomesMessage(t *testing.T) {
	h, producer, updates := newCallHandler(t, nil)
	raw, err := json.Marshal(InboundEvent{Type: "call", ID: "CALL1", From: caller, SenderName: "Ale", Timestamp: 1700000000, Summary: "offer", Video: true})
	require.NoError(t, err)
	require.NoError(t, h.handleCall("ch-1", raw))
	require.NoError(t, h.handleCall("ch-1", raw))

	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	require.Equal(t, "call", msg.ContentType)
	require.Equal(t, "CALL1", msg.ExternalID)
	require.Equal(t, "Chamada de vídeo", msg.Content)
	require.Equal(t, "5511999999999", msg.Metadata["sender_id"])
	require.Equal(t, "5511999999999", msg.Metadata["phone"])
	require.Equal(t, "Ale", msg.Metadata["sender_name"])
	require.Equal(t, "CALL1", msg.Metadata[MetaCallID])
	require.Equal(t, CallStateRinging, msg.Metadata[MetaCallState])
	require.Equal(t, "true", msg.Metadata[MetaCallVideo])
	require.Equal(t, int64(1700000000), msg.Timestamp.Unix())
	noUpdate(t, updates)
}

// Aceite e término viram atendida e encerrada com a duração; término sem
// aceite é perdida; recusa pelo aparelho é recusada por device.
func TestCall_Transitions(t *testing.T) {
	h, _, updates := newCallHandler(t, nil)

	require.NoError(t, h.handleCall("ch-1", callRaw(t, "A", "offer", 100)))
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "A", "accept", 105)))
	u := nextUpdate(t, updates)
	require.Equal(t, CallStateAnswered, u.State)
	require.Equal(t, caller, u.From)
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "A", "terminate", 165)))
	u = nextUpdate(t, updates)
	require.Equal(t, CallStateEnded, u.State)
	require.Equal(t, 60*time.Second, u.Duration)

	require.NoError(t, h.handleCall("ch-1", callRaw(t, "B", "offer", 200)))
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "B", "terminate", 230)))
	require.Equal(t, CallStateMissed, nextUpdate(t, updates).State)

	require.NoError(t, h.handleCall("ch-1", callRaw(t, "C", "offer", 300)))
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "C", "reject", 310)))
	u = nextUpdate(t, updates)
	require.Equal(t, CallStateRejected, u.State)
	require.Equal(t, RejectedByDevice, u.RejectedBy)
	// o término que segue a recusa não gera nada
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "C", "terminate", 311)))
	noUpdate(t, updates)

	// sinalização de chamada desconhecida (ou já encerrada) é ignorada
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "Z", "accept", 400)))
	noUpdate(t, updates)
	_, ok := h.calls.get("A")
	require.False(t, ok, "chamada encerrada deve sair do registro")
}

// Sem término (bridge caiu no meio do toque) a chamada vira perdida pelo
// tempo limite, em vez de ficar tocando para sempre.
func TestCall_RingTimeout(t *testing.T) {
	h, _, updates := newCallHandler(t, nil)
	h.calls.ringTimeout = 20 * time.Millisecond
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "T", "offer", 0)))
	require.Equal(t, CallStateMissed, nextUpdate(t, updates).State)
}

// RejectCall manda call_command reject ao bridge com o JID de quem ligou e,
// com o call_ack ok, registra a recusa; chamada desconhecida ou já atendida
// é erro antes de falar com o bridge.
func TestCall_RejectThroughBridge(t *testing.T) {
	h, _, updates := newCallHandler(t, nil)
	clientConn, serverConn := newConnPair(t)
	go h.hub.ServeConnection(NewBridge("ch-1", serverConn))
	waitForWelcome(t, clientConn)

	require.ErrorIs(t, h.RejectCall(context.Background(), "ch-1", "NOPE", RejectedByOperator), ErrCallNotFound)

	require.NoError(t, h.handleCall("ch-1", callRaw(t, "R", "offer", 0)))
	require.ErrorIs(t, h.RejectCall(context.Background(), "ch-other", "R", RejectedByOperator), ErrCallNotFound)

	done := make(chan error, 1)
	go func() { done <- h.RejectCall(context.Background(), "ch-1", "R", RejectedByOperator) }()

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := clientConn.ReadMessage()
	require.NoError(t, err)
	var cmd Frame
	require.NoError(t, json.Unmarshal(data, &cmd))
	require.Equal(t, "call_command", cmd.Type)
	require.Equal(t, "reject", cmd.Text)
	require.Equal(t, "R", cmd.CallID)
	require.Equal(t, caller, cmd.From)
	require.NoError(t, writeJSON(clientConn, Frame{Type: "call_ack", ChannelID: "ch-1", CorrelationID: cmd.CorrelationID, OK: true}))

	require.NoError(t, <-done)
	u := nextUpdate(t, updates)
	require.Equal(t, CallStateRejected, u.State)
	require.Equal(t, RejectedByOperator, u.RejectedBy)
	require.ErrorIs(t, h.RejectCall(context.Background(), "ch-1", "R", RejectedByOperator), ErrCallNotFound)
}

// Sem bridge ligado a recusa falha com ErrNoBridge e a chamada continua
// tocando; o bridge negando o comando também não muda o estado.
func TestCall_RejectWithoutBridgeOrRefused(t *testing.T) {
	h, _, updates := newCallHandler(t, nil)
	require.NoError(t, h.handleCall("ch-1", callRaw(t, "N", "offer", 0)))
	err := h.RejectCall(context.Background(), "ch-1", "N", RejectedByOperator)
	require.True(t, errors.Is(err, ErrNoBridge), "erro = %v", err)
	noUpdate(t, updates)

	clientConn, serverConn := newConnPair(t)
	go h.hub.ServeConnection(NewBridge("ch-1", serverConn))
	waitForWelcome(t, clientConn)
	done := make(chan error, 1)
	go func() { done <- h.RejectCall(context.Background(), "ch-1", "N", RejectedByOperator) }()
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := clientConn.ReadMessage()
	require.NoError(t, err)
	var cmd Frame
	require.NoError(t, json.Unmarshal(data, &cmd))
	require.NoError(t, writeJSON(clientConn, Frame{Type: "call_ack", ChannelID: "ch-1", CorrelationID: cmd.CorrelationID, OK: false, Error: "not ringing"}))
	require.ErrorContains(t, <-done, "not ringing")
	noUpdate(t, updates)
	call, ok := h.calls.get("N")
	require.True(t, ok)
	require.Equal(t, CallStateRinging, call.state)
}

// Canal com reject_call ligado recusa sozinho: a oferta ainda vira mensagem,
// o bridge recebe o comando e o sink vê a recusa por auto.
func TestCall_AutoReject(t *testing.T) {
	h, producer, updates := newCallHandler(t, map[string]string{"reject_call": "true"})
	clientConn, serverConn := newConnPair(t)
	go h.hub.ServeConnection(NewBridge("ch-1", serverConn))
	waitForWelcome(t, clientConn)

	require.NoError(t, h.handleCall("ch-1", callRaw(t, "AUTO", "offer", 0)))
	require.Len(t, producer.InboundMessages, 1)

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := clientConn.ReadMessage()
	require.NoError(t, err)
	var cmd Frame
	require.NoError(t, json.Unmarshal(data, &cmd))
	require.Equal(t, "call_command", cmd.Type)
	require.Equal(t, "AUTO", cmd.CallID)
	require.NoError(t, clientConn.WriteMessage(websocket.TextMessage, mustJSON(t, Frame{Type: "call_ack", ChannelID: "ch-1", CorrelationID: cmd.CorrelationID, OK: true})))

	u := nextUpdate(t, updates)
	require.Equal(t, CallStateRejected, u.State)
	require.Equal(t, RejectedByAuto, u.RejectedBy)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
