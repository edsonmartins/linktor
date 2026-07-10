package officialcalls

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectWebhookBody(callID, from, offer string) []byte {
	env := map[string]any{
		"object": "whatsapp_business_account",
		"entry": []any{map[string]any{
			"changes": []any{map[string]any{
				"field": "calls",
				"value": map[string]any{
					"metadata": map[string]any{"phone_number_id": "phone-55"},
					"calls": []any{map[string]any{
						"id": callID, "from": from, "event": "connect",
						"session": map[string]any{"sdp_type": "offer", "sdp": offer},
					}},
				},
			}},
		}},
	}
	b, _ := json.Marshal(env)
	return b
}

func terminateWebhookBody(callID string) []byte {
	env := map[string]any{
		"entry": []any{map[string]any{"changes": []any{map[string]any{
			"field": "calls",
			"value": map[string]any{
				"metadata": map[string]any{"phone_number_id": "phone-55"},
				"calls":    []any{map[string]any{"id": callID, "event": "terminate", "status": "COMPLETED"}},
			},
		}}}},
	}
	b, _ := json.Marshal(env)
	return b
}

type eventCollector struct {
	mu     sync.Mutex
	events []GatewayEvent
}

func (e *eventCollector) handler(_ context.Context, evt GatewayEvent) {
	e.mu.Lock()
	e.events = append(e.events, evt)
	e.mu.Unlock()
}

func (e *eventCollector) phases() []CallPhase {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []CallPhase
	for _, ev := range e.events {
		out = append(out, ev.Phase)
	}
	return out
}

func TestGateway_AutoAnswerInbound(t *testing.T) {
	var mu sync.Mutex
	var accept callActionRequest
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		_ = json.Unmarshal(raw, &accept)
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ec := &eventCollector{}
	g := NewGateway(GatewayConfig{Signaling: c, Session: hostOnly, AutoAnswer: true, Handler: ec.handler})

	offerer, _ := newOfferer(t)
	defer offerer.Close()
	offer := offerSDP(t, offerer)

	require.NoError(t, g.HandleWebhook(context.Background(), connectWebhookBody("call-9", "5511999999999", offer)))

	// The gateway must have accepted with a real SDP answer.
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, ActionAccept, accept.Action)
	assert.Equal(t, "call-9", accept.CallID)
	require.NotNil(t, accept.Session)
	assert.Equal(t, SDPAnswer, accept.Session.SDPType)
	assert.Contains(t, accept.Session.SDP, "m=audio")

	assert.Contains(t, ec.phases(), PhaseRinging)
	assert.NotNil(t, g.Session("call-9"), "media session registered")
}

func TestGateway_ManualRejectDoesNotAnswer(t *testing.T) {
	var mu sync.Mutex
	var actions []CallAction
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req callActionRequest
		_ = json.Unmarshal(raw, &req)
		mu.Lock()
		actions = append(actions, req.Action)
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	g := NewGateway(GatewayConfig{Signaling: c, Session: hostOnly, AutoAnswer: false})

	offerer, _ := newOfferer(t)
	defer offerer.Close()
	offer := offerSDP(t, offerer)

	require.NoError(t, g.HandleWebhook(context.Background(), connectWebhookBody("call-r", "5511", offer)))
	// Not auto-answered: no accept yet, call is pending.
	require.NoError(t, g.RejectCall(context.Background(), "call-r"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []CallAction{ActionReject}, actions)
}

func TestGateway_ManualRejectEmitsEnded(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ec := &eventCollector{}
	g := NewGateway(GatewayConfig{Signaling: c, Session: hostOnly, AutoAnswer: false, Handler: ec.handler})

	offerer, _ := newOfferer(t)
	defer offerer.Close()
	offer := offerSDP(t, offerer)
	require.NoError(t, g.HandleWebhook(context.Background(), connectWebhookBody("call-rj", "5511", offer)))
	require.NoError(t, g.RejectCall(context.Background(), "call-rj"))

	// The host must get a terminal phase, not be left stuck in "ringing".
	assert.Contains(t, ec.phases(), PhaseEnded)
}

func TestGateway_AutoAnswerFailureEmitsEnded(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ec := &eventCollector{}
	g := NewGateway(GatewayConfig{Signaling: c, Session: hostOnly, AutoAnswer: true, Handler: ec.handler})

	// A garbage SDP offer makes AnswerOffer fail, so the auto-answer path errors.
	require.NoError(t, g.HandleWebhook(context.Background(), connectWebhookBody("call-bad", "5511", "not-a-valid-sdp")))

	assert.Contains(t, ec.phases(), PhaseRinging)
	assert.Contains(t, ec.phases(), PhaseEnded)
	assert.Nil(t, g.Session("call-bad"))
}

func TestGateway_TerminateEmitsEnded(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ec := &eventCollector{}
	g := NewGateway(GatewayConfig{Signaling: c, Session: hostOnly, AutoAnswer: true, Handler: ec.handler})

	offerer, _ := newOfferer(t)
	defer offerer.Close()
	offer := offerSDP(t, offerer)
	require.NoError(t, g.HandleWebhook(context.Background(), connectWebhookBody("call-t", "5511", offer)))
	require.NoError(t, g.HandleWebhook(context.Background(), terminateWebhookBody("call-t")))

	assert.Contains(t, ec.phases(), PhaseEnded)
	assert.Nil(t, g.Session("call-t"), "session removed after terminate")
}

func TestGateway_IgnoresNonCallWebhook(t *testing.T) {
	g := NewGateway(GatewayConfig{AutoAnswer: true})
	assert.NoError(t, g.HandleWebhook(context.Background(), []byte(messageOnlyWebhook)))
}
