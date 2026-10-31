package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/callsignal"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/require"
)

const caller = "5511999999999@s.whatsapp.net"

// newCallHandler monta o handler com o registro de chamadas real e um sink
// que entrega cada desfecho num canal.
func newCallHandler(t *testing.T, cfg map[string]string) (*Handler, *testutil.MockProducer, *callsignal.Tracker, chan callsignal.Update) {
	t.Helper()
	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-1"] = &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: entity.ChannelType(plugin.ChannelTypeWhatsAppRemote), Config: cfg}
	producer := testutil.NewMockProducer()
	h := NewHandler(NewHub(), channels, producer, nil, nil)
	tracker := callsignal.NewTracker(channels)
	updates := make(chan callsignal.Update, 8)
	tracker.SetSink(func(_ context.Context, u callsignal.Update) { updates <- u })
	h.SetCallTracker(tracker)
	return h, producer, tracker, updates
}

func callFrame(t *testing.T, evt InboundEvent) json.RawMessage {
	t.Helper()
	evt.Type = "call"
	raw, err := json.Marshal(evt)
	require.NoError(t, err)
	return raw
}

func nextUpdate(t *testing.T, updates chan callsignal.Update) callsignal.Update {
	t.Helper()
	select {
	case u := <-updates:
		return u
	case <-time.After(2 * time.Second):
		t.Fatal("sink não recebeu o desfecho")
		return callsignal.Update{}
	}
}

// connectBridge liga um bridge falso ao hub do handler.
func connectBridge(t *testing.T, h *Handler) *websocket.Conn {
	t.Helper()
	clientConn, serverConn := newConnPair(t)
	go h.hub.ServeConnection(NewBridge("ch-1", serverConn))
	waitForWelcome(t, clientConn)
	return clientConn
}

// readCallCommand lê o próximo frame do bridge falso e o devolve.
func readCallCommand(t *testing.T, conn *websocket.Conn) Frame {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var cmd Frame
	require.NoError(t, json.Unmarshal(data, &cmd))
	require.Equal(t, "call_command", cmd.Type)
	require.Equal(t, "reject", cmd.Text)
	return cmd
}

// A oferta vira mensagem de tipo call na conversa do contato, com a mesma
// identidade das mensagens (o número, com o LID ao lado); repetida não
// duplica.
func TestCall_OfferBecomesMessage(t *testing.T) {
	h, producer, tracker, _ := newCallHandler(t, nil)
	raw := callFrame(t, InboundEvent{ID: "CALL1", From: caller, SenderLID: "252415295144120@lid", SenderName: "Ale", Timestamp: 1700000000, Summary: "offer", Video: true})
	require.NoError(t, h.handleCall("ch-1", raw))
	require.NoError(t, h.handleCall("ch-1", raw))

	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	require.Equal(t, "call", msg.ContentType)
	require.Equal(t, "CALL1", msg.ExternalID)
	require.Equal(t, "Chamada de vídeo", msg.Content)
	require.Equal(t, "5511999999999", msg.Metadata["sender_id"])
	require.Equal(t, "5511999999999", msg.Metadata["phone"])
	require.Equal(t, "252415295144120@lid", msg.Metadata["sender_lid"])
	require.Equal(t, "Ale", msg.Metadata["sender_name"])
	require.Equal(t, callsignal.StateRinging, msg.Metadata[callsignal.MetaState])
	require.Equal(t, "true", msg.Metadata[callsignal.MetaVideo])
	require.Equal(t, int64(1700000000), msg.Timestamp.Unix())
	require.True(t, tracker.Ringing("CALL1"))
}

// Sequência real de um aparelho principal que atende, pelo WebSocket como o
// bridge manda: aceite (do próprio número) e término no mesmo segundo viram
// só "atendida".
func TestCall_AnsweredOnPhoneThroughHub(t *testing.T) {
	h, _, _, updates := newCallHandler(t, nil)
	conn := connectBridge(t, h)
	for _, evt := range []InboundEvent{
		{ID: "X", From: "554499806761@s.whatsapp.net", SenderLID: "252415295144120@lid", Timestamp: 1792956583, Summary: "offer"},
		{ID: "X", From: "554497239232@s.whatsapp.net", SenderLID: "258570671104121@lid", Timestamp: 1792956588, Summary: "accept"},
		{ID: "X", From: "554499806761@s.whatsapp.net", SenderLID: "252415295144120@lid", Timestamp: 1792956588, Summary: "terminate"},
	} {
		require.NoError(t, writeJSON(conn, Frame{Type: "call_event", ChannelID: "ch-1", Message: callFrame(t, evt)}))
	}
	u := nextUpdate(t, updates)
	require.Equal(t, callsignal.StateAnswered, u.State)
	require.Equal(t, int64(1792956588), u.At.Unix())
}

// A recusa vai ao bridge como call_command endereçado ao LID com que a
// chamada foi sinalizada e só conta com o call_ack ok.
func TestCall_RejectThroughBridgeByLID(t *testing.T) {
	h, _, tracker, updates := newCallHandler(t, nil)
	conn := connectBridge(t, h)
	require.NoError(t, h.handleCall("ch-1", callFrame(t, InboundEvent{ID: "R", From: caller, SenderLID: "252415295144120@lid", Summary: "offer"})))

	done := make(chan error, 1)
	go func() { done <- tracker.Reject(context.Background(), "ch-1", "R", callsignal.RejectedByOperator) }()
	cmd := readCallCommand(t, conn)
	require.Equal(t, "R", cmd.CallID)
	require.Equal(t, "252415295144120@lid", cmd.From)
	require.NoError(t, writeJSON(conn, Frame{Type: "call_ack", ChannelID: "ch-1", CorrelationID: cmd.CorrelationID, OK: true}))
	require.NoError(t, <-done)
	require.Equal(t, callsignal.RejectedByOperator, nextUpdate(t, updates).RejectedBy)
}

// Sem bridge ligado a recusa falha com ErrNoBridge; o bridge negando também
// falha. Nos dois casos a chamada continua tocando.
func TestCall_RejectWithoutBridgeOrRefused(t *testing.T) {
	h, _, tracker, _ := newCallHandler(t, nil)
	require.NoError(t, h.handleCall("ch-1", callFrame(t, InboundEvent{ID: "N", From: caller, Summary: "offer"})))
	err := tracker.Reject(context.Background(), "ch-1", "N", callsignal.RejectedByOperator)
	require.True(t, errors.Is(err, callsignal.ErrDeviceOffline), "erro = %v", err)
	require.True(t, tracker.Ringing("N"))

	conn := connectBridge(t, h)
	done := make(chan error, 1)
	go func() { done <- tracker.Reject(context.Background(), "ch-1", "N", callsignal.RejectedByOperator) }()
	cmd := readCallCommand(t, conn)
	require.Equal(t, caller, cmd.From)
	require.NoError(t, writeJSON(conn, Frame{Type: "call_ack", ChannelID: "ch-1", CorrelationID: cmd.CorrelationID, OK: false, Error: "not ringing"}))
	require.ErrorContains(t, <-done, "not ringing")
	require.True(t, tracker.Ringing("N"))
}

// Canal com reject_call recusa sozinho pelo bridge depois de publicar.
func TestCall_AutoRejectThroughBridge(t *testing.T) {
	h, producer, _, updates := newCallHandler(t, map[string]string{"reject_call": "true"})
	conn := connectBridge(t, h)
	require.NoError(t, h.handleCall("ch-1", callFrame(t, InboundEvent{ID: "AUTO", From: caller, Summary: "offer"})))
	require.Len(t, producer.InboundMessages, 1)

	cmd := readCallCommand(t, conn)
	require.Equal(t, "AUTO", cmd.CallID)
	require.NoError(t, writeJSON(conn, Frame{Type: "call_ack", ChannelID: "ch-1", CorrelationID: cmd.CorrelationID, OK: true}))
	require.Equal(t, callsignal.RejectedByAuto, nextUpdate(t, updates).RejectedBy)
}

// Sem registro configurado os frames de chamada são ignorados.
func TestCall_NoTrackerIgnores(t *testing.T) {
	channels := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	h := NewHandler(NewHub(), channels, producer, nil, nil)
	require.NoError(t, h.handleCall("ch-1", callFrame(t, InboundEvent{ID: "Q", From: caller, Summary: "offer"})))
	require.Empty(t, producer.InboundMessages)
}
