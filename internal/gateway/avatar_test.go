package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// liveBridge sobe um bridge registrado no hub e devolve o lado do "aparelho":
// answer lê o próximo pedido e responde com reply, no correlation_id dele.
func liveBridge(t *testing.T, hub *Hub, channelID string) (answer func(reply Frame) Frame) {
	t.Helper()
	clientConn, serverConn := newConnPair(t)
	go hub.ServeConnection(NewBridge(channelID, serverConn))
	waitForWelcome(t, clientConn)
	t.Cleanup(func() { _ = clientConn.Close() })
	return func(reply Frame) Frame {
		t.Helper()
		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := clientConn.ReadMessage()
		require.NoError(t, err)
		var f Frame
		require.NoError(t, unmarshal(data, &f))
		reply.Type = "avatar_result"
		reply.CorrelationID = f.CorrelationID
		require.NoError(t, writeJSON(clientConn, reply))
		return f
	}
}

func TestRemoteAdapterGetContactAvatar_IdaEVolta(t *testing.T) {
	hub := NewHub()
	answer := liveBridge(t, hub, "wh-1")
	adapter := NewRemoteAdapter(hub, "wh-1")

	cases := []struct {
		name  string
		reply Frame
		check func(*plugin.ContactAvatar, error)
	}{
		{"foto", Frame{OK: true, PictureID: "p2", URL: "https://pps.whatsapp.net/x"}, func(a *plugin.ContactAvatar, err error) {
			require.NoError(t, err)
			assert.Equal(t, &plugin.ContactAvatar{PictureID: "p2", URL: "https://pps.whatsapp.net/x"}, a)
		}},
		{"não mudou", Frame{OK: true, PictureID: "p1", Unchanged: true}, func(a *plugin.ContactAvatar, err error) {
			require.NoError(t, err)
			assert.True(t, a.Unchanged)
		}},
		{"sem foto", Frame{Reason: "not_set"}, func(_ *plugin.ContactAvatar, err error) {
			assert.ErrorIs(t, err, plugin.ErrAvatarNotSet)
		}},
		{"oculta", Frame{Reason: "hidden"}, func(_ *plugin.ContactAvatar, err error) {
			assert.ErrorIs(t, err, plugin.ErrAvatarHidden)
		}},
		{"sessão do aparelho caída", Frame{Reason: "not_connected", Error: "client is not connected"}, func(_ *plugin.ContactAvatar, err error) {
			assert.ErrorIs(t, err, plugin.ErrAvatarSessionUnavailable)
		}},
		{"outra falha", Frame{Error: "iq timeout"}, func(_ *plugin.ContactAvatar, err error) {
			require.Error(t, err)
			assert.NotErrorIs(t, err, plugin.ErrAvatarSessionUnavailable)
			assert.NotErrorIs(t, err, plugin.ErrAvatarNotSet)
		}},
	}
	for _, tc := range cases {
		type res struct {
			a   *plugin.ContactAvatar
			err error
		}
		done := make(chan res, 1)
		go func() {
			a, err := adapter.GetContactAvatar(context.Background(), "5544999990000", "p1")
			done <- res{a, err}
		}()
		req := answer(tc.reply)
		assert.Equal(t, "avatar_request", req.Type, tc.name)
		assert.Equal(t, "5544999990000", req.To, tc.name)
		assert.Equal(t, "p1", req.PictureID, tc.name)
		assert.NotEmpty(t, req.CorrelationID, tc.name)
		select {
		case r := <-done:
			tc.check(r.a, r.err)
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: sem resposta", tc.name)
		}
	}
}

func TestRemoteAdapterGetContactAvatar_SemBridge(t *testing.T) {
	adapter := NewRemoteAdapter(NewHub(), "wh-off")
	_, err := adapter.GetContactAvatar(context.Background(), "5544999990000", "")
	assert.ErrorIs(t, err, plugin.ErrAvatarSessionUnavailable)
}

// Bridge antigo, que não conhece avatar_request, não responde: a consulta
// desiste e vira "sem sessão", não um minuto pendurado.
func TestRemoteAdapterGetContactAvatar_BridgeQueNaoResponde(t *testing.T) {
	hub := NewHub()
	clientConn, serverConn := newConnPair(t)
	go hub.ServeConnection(NewBridge("wh-old", serverConn))
	waitForWelcome(t, clientConn)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := NewRemoteAdapter(hub, "wh-old").GetContactAvatar(ctx, "5544999990000", "")
	assert.ErrorIs(t, err, plugin.ErrAvatarSessionUnavailable)
}
