package whatsapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/msgfy/linktor/internal/voip/call"
	"github.com/msgfy/linktor/internal/voip/core"
	"go.mau.fi/whatsmeow/types"
)

func TestCallEvent_Mapping(t *testing.T) {
	ci := &call.CallInfo{
		CallID:    "C1",
		PeerJid:   "5511999999999@s.whatsapp.net",
		CallerPn:  "5511999999999",
		MediaType: core.CallMediaTypeVideo,
	}
	ci.StateData.State = core.CallStateInitiating

	evt := callEvent(CallEventIncoming, ci)
	assert.Equal(t, CallEventIncoming, evt.Type)
	assert.Equal(t, "C1", evt.CallID)
	assert.Equal(t, "5511999999999", evt.PeerID)
	assert.True(t, evt.IsVideo)
}

func TestCallEvent_PeerIDFromJIDWhenNoPn(t *testing.T) {
	ci := &call.CallInfo{
		CallID:    "C2",
		PeerJid:   "5511888887777@s.whatsapp.net",
		MediaType: core.CallMediaTypeAudio,
	}
	evt := callEvent(CallEventState, ci)
	assert.Equal(t, "5511888887777", evt.PeerID)
	assert.False(t, evt.IsVideo)
}

func TestCallEvent_NilInfo(t *testing.T) {
	evt := callEvent(CallEventEnded, nil)
	assert.Equal(t, CallEventEnded, evt.Type)
	assert.Empty(t, evt.CallID)
}

func TestWrapCall_Structure(t *testing.T) {
	from := types.NewJID("5511999999999", types.DefaultUserServer)
	node := wrapCall(from, nil)
	assert.Equal(t, "call", node.Tag)
	assert.Equal(t, from, node.Attrs["from"])
	assert.Empty(t, node.GetChildren())
}

func TestCallGateway_GetRemove(t *testing.T) {
	g := &CallGateway{calls: map[string]*call.CallManager{}}
	assert.Nil(t, g.get("x"))

	m := call.NewCallManager(nil, nil)
	g.mu.Lock()
	g.calls["x"] = m
	g.mu.Unlock()
	assert.NotNil(t, g.get("x"))

	g.remove("x")
	assert.Nil(t, g.get("x"))
}

func TestAdapterCall_NotConnected(t *testing.T) {
	a := NewAdapter()
	ctx := context.Background()

	_, err := a.PlaceCall(ctx, "5511999999999", false)
	assert.ErrorIs(t, err, ErrClientNotReady)
	assert.ErrorIs(t, a.AcceptCall(ctx, "C1"), ErrClientNotReady)
	assert.ErrorIs(t, a.RejectCall(ctx, "C1"), ErrClientNotReady)
	assert.ErrorIs(t, a.EndCall(ctx, "C1"), ErrClientNotReady)
}

func TestInitialize_RecordCallsParam(t *testing.T) {
	a := NewAdapter()
	err := a.Initialize(map[string]string{
		"channel_id":     "test-channel",
		"record_calls":   "true",
		"recordings_dir": "/tmp/rec",
	})
	assert.NoError(t, err)
	assert.True(t, a.config.RecordCalls)
	assert.Equal(t, "/tmp/rec", a.config.RecordingsDir)
	assert.NotNil(t, a.callRecorder, "recorder built from config")

	// Default: recording off when the param is absent/false.
	b := NewAdapter()
	_ = b.Initialize(map[string]string{"channel_id": "c2"})
	assert.False(t, b.config.RecordCalls)
	assert.Nil(t, b.callRecorder)
}
