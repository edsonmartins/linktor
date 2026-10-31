package whatsapp

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/callsignal"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/require"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	callerLID = types.NewJID("252415295144120", types.HiddenUserServer)
	callerPN  = types.NewJID("554499806761", types.DefaultUserServer)
)

// Os eventos de chamada do whatsmeow viram sinais; no aparelho conectado a
// oferta chega como aviso (CallOfferNotice) e vale igual.
func TestToCallSignal(t *testing.T) {
	meta := types.BasicCallMeta{From: callerLID, CallCreatorAlt: callerPN, CallID: "C1", Timestamp: time.Unix(5, 0)}
	cases := []struct {
		evt    any
		signal string
		video  bool
	}{
		{&events.CallOfferNotice{BasicCallMeta: meta, Media: "video"}, "offer", true},
		{&events.CallOffer{BasicCallMeta: meta, Data: &waBinary.Node{Tag: "offer", Content: []waBinary.Node{{Tag: "audio"}}}}, "offer", false},
		{&events.CallOffer{BasicCallMeta: meta, Data: &waBinary.Node{Tag: "offer", Content: []waBinary.Node{{Tag: "video"}}}}, "offer", true},
		{&events.CallAccept{BasicCallMeta: meta}, callsignal.SignalAccept, false},
		{&events.CallReject{BasicCallMeta: meta}, callsignal.SignalReject, false},
		{&events.CallTerminate{BasicCallMeta: meta}, callsignal.SignalTerminate, false},
	}
	for _, tc := range cases {
		sig := toCallSignal(tc.evt)
		require.NotNil(t, sig)
		require.Equal(t, tc.signal, sig.Signal)
		require.Equal(t, tc.video, sig.Video)
		require.Equal(t, "C1", sig.CallID)
		require.Equal(t, callerLID, sig.From)
	}
	require.Nil(t, toCallSignal(&events.CallRelayLatency{BasicCallMeta: meta}))
	require.Nil(t, toCallSignal(&events.Connected{}))
}

// A oferta vira mensagem com a identidade das mensagens: o número como
// remetente, o LID em metadata e a recusa endereçada ao LID.
func TestHandleCallSignalOfferIdentity(t *testing.T) {
	a := NewAdapter()
	var got []CallSignalEvent
	a.SetCallSignalHandler(func(_ context.Context, evt CallSignalEvent) { got = append(got, evt) })

	a.handleCallSignal(context.Background(), nil, &callSignal{Signal: "offer", CallID: "C1", From: callerLID, CreatorAlt: callerPN, Video: true, Time: time.Unix(7, 0)})
	a.handleCallSignal(context.Background(), nil, &callSignal{Signal: callsignal.SignalAccept, CallID: "C1", From: types.NewJID("554497239232", types.DefaultUserServer), Time: time.Unix(9, 0)})

	require.Len(t, got, 2)
	offer := got[0]
	require.Equal(t, "C1", offer.CallID)
	require.Equal(t, callerLID.String(), offer.SignaledBy)
	require.Equal(t, int64(7), offer.At.Unix())
	msg := offer.Message
	require.NotNil(t, msg)
	require.Equal(t, plugin.ContentTypeCall, msg.ContentType)
	require.Equal(t, "C1", msg.ExternalID)
	require.Equal(t, "554499806761", msg.SenderID)
	require.Equal(t, "Chamada de vídeo", msg.Content)
	require.Equal(t, callerPN.String(), msg.Metadata["chat_jid"])
	require.Equal(t, callerPN.String(), msg.Metadata["sender_pn"])
	require.Equal(t, "554499806761", msg.Metadata["phone"])
	require.Equal(t, callerLID.String(), msg.Metadata["lid"])
	require.Equal(t, callsignal.StateRinging, msg.Metadata[callsignal.MetaState])
	require.Equal(t, "true", msg.Metadata[callsignal.MetaVideo])

	accept := got[1]
	require.Equal(t, callsignal.SignalAccept, accept.Signal)
	require.Nil(t, accept.Message)
}

// LID sem número alternativo e sem sessão para resolver segue com o LID
// inteiro, sem sender_pn: melhor um contato pelo LID do que um número errado.
func TestHandleCallSignalUnresolvedLID(t *testing.T) {
	a := NewAdapter()
	var got CallSignalEvent
	a.SetCallSignalHandler(func(_ context.Context, evt CallSignalEvent) { got = evt })
	a.handleCallSignal(context.Background(), nil, &callSignal{Signal: "offer", CallID: "C2", From: callerLID})
	require.NotNil(t, got.Message)
	require.Equal(t, callerLID.User, got.Message.SenderID)
	require.Empty(t, got.Message.Metadata["sender_pn"])
	require.Empty(t, got.Message.Metadata["phone"])
	require.False(t, got.At.IsZero())
}

// Sem handler registrado, e para sinal sem ID, nada acontece.
func TestHandleCallSignalWithoutHandler(t *testing.T) {
	a := NewAdapter()
	a.handleCallSignal(context.Background(), nil, &callSignal{Signal: "offer", CallID: "C1", From: callerPN})
	var calls int
	a.SetCallSignalHandler(func(context.Context, CallSignalEvent) { calls++ })
	a.handleCallSignal(context.Background(), nil, &callSignal{Signal: "offer", From: callerPN})
	require.Zero(t, calls)
}

// Recusar sem sessão ligada é ErrClientNotReady.
func TestRejectCallFromWithoutClient(t *testing.T) {
	require.ErrorIs(t, NewAdapter().RejectCallFrom(context.Background(), callerLID.String(), "C1"), ErrClientNotReady)
}
