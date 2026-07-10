package officialcalls

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// CallPhase is a call lifecycle phase surfaced to the host.
type CallPhase string

const (
	PhaseRinging   CallPhase = "ringing"   // inbound offer received
	PhaseConnected CallPhase = "connected" // media connected
	PhaseEnded     CallPhase = "ended"     // call terminated
)

// GatewayEvent is a call lifecycle notification.
type GatewayEvent struct {
	CallID        string    `json:"call_id"`
	Phase         CallPhase `json:"phase"`
	From          string    `json:"from"`
	PhoneNumberID string    `json:"phone_number_id,omitempty"`
	RecordingPath string    `json:"recording_path,omitempty"`
}

// GatewayHandler receives call lifecycle events. Must not block.
type GatewayHandler func(ctx context.Context, evt GatewayEvent)

// GatewayConfig configures a Gateway.
type GatewayConfig struct {
	Signaling  *Client
	Session    SessionConfig
	AutoAnswer bool // answer inbound calls automatically (typical for a bot/IVR)
	Handler    GatewayHandler
	Logger     *slog.Logger
}

// Gateway ties the Graph signaling to the WebRTC media engine: it answers
// inbound calls (webhook connect → SDP answer → accept) and drives call
// lifecycle, keeping one media session per call.
type Gateway struct {
	signaling  *Client
	sessCfg    SessionConfig
	autoAnswer bool
	handler    GatewayHandler
	log        *slog.Logger

	mu       sync.Mutex
	sessions map[string]*CallSession
	pending  map[string]pendingCall
	meta     map[string]callMeta // from/phone-number-id per call, for events
	ended    map[string]struct{} // calls that already reached a terminal state
}

type pendingCall struct {
	offer string
}

type callMeta struct {
	from          string
	phoneNumberID string
}

// NewGateway builds a call gateway.
func NewGateway(cfg GatewayConfig) *Gateway {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Gateway{
		signaling:  cfg.Signaling,
		sessCfg:    cfg.Session,
		autoAnswer: cfg.AutoAnswer,
		handler:    cfg.Handler,
		log:        log,
		sessions:   make(map[string]*CallSession),
		pending:    make(map[string]pendingCall),
		meta:       make(map[string]callMeta),
		ended:      make(map[string]struct{}),
	}
}

// HandleWebhook parses a Cloud API webhook body and routes any call events. Safe
// to call on every inbound webhook; non-call payloads are ignored.
func (g *Gateway) HandleWebhook(ctx context.Context, body []byte) error {
	calls, err := ParseWebhookCalls(body)
	if err != nil {
		return err
	}
	for _, c := range calls {
		switch {
		case c.Event.IsConnect():
			g.onConnect(ctx, c.PhoneNumberID, c.Event)
		case c.Event.IsTerminate():
			g.onTerminate(ctx, c.PhoneNumberID, c.Event)
		}
	}
	return nil
}

func (g *Gateway) onConnect(ctx context.Context, phoneNumberID string, ev WebhookCallEvent) {
	if ev.Session == nil || ev.Session.SDP == "" {
		g.log.Warn("call connect without SDP offer", "call_id", ev.ID)
		return
	}
	g.mu.Lock()
	g.meta[ev.ID] = callMeta{from: ev.From, phoneNumberID: phoneNumberID}
	g.mu.Unlock()
	g.emit(ctx, GatewayEvent{CallID: ev.ID, Phase: PhaseRinging, From: ev.From, PhoneNumberID: phoneNumberID})

	if !g.autoAnswer {
		g.mu.Lock()
		g.pending[ev.ID] = pendingCall{offer: ev.Session.SDP}
		g.mu.Unlock()
		return
	}
	if err := g.answer(ctx, ev.ID, ev.Session.SDP); err != nil {
		g.log.Error("failed to answer inbound call", "call_id", ev.ID, "err", err)
		_ = g.signaling.Reject(ctx, ev.ID)
		g.finishCall(ctx, ev.ID) // never leave the host stuck in "ringing"
	}
}

// AcceptCall answers a ringing inbound call that was not auto-answered.
func (g *Gateway) AcceptCall(ctx context.Context, callID string) error {
	g.mu.Lock()
	p, ok := g.pending[callID]
	delete(g.pending, callID)
	g.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending call %s", callID)
	}
	if err := g.answer(ctx, callID, p.offer); err != nil {
		_ = g.signaling.Reject(ctx, callID)
		g.finishCall(ctx, callID)
		return err
	}
	return nil
}

// RejectCall declines a ringing inbound call.
func (g *Gateway) RejectCall(ctx context.Context, callID string) error {
	g.mu.Lock()
	delete(g.pending, callID)
	g.mu.Unlock()
	err := g.signaling.Reject(ctx, callID)
	g.finishCall(ctx, callID) // emit a terminal event even on manual reject
	return err
}

// EndCall terminates an active call and tears down its media.
func (g *Gateway) EndCall(ctx context.Context, callID string) error {
	err := g.signaling.Terminate(ctx, callID)
	g.finishCall(ctx, callID)
	return err
}

// answer builds the media session, negotiates the SDP answer and accepts. The
// per-call from/phone-number-id come from meta captured at connect time.
func (g *Gateway) answer(ctx context.Context, callID, offer string) error {
	g.mu.Lock()
	meta := g.meta[callID]
	g.mu.Unlock()

	sess, err := NewCallSession(callID, g.sessCfg)
	if err != nil {
		return fmt.Errorf("media session: %w", err)
	}
	// Wire handlers before negotiation so a media failure is never lost: OnClosed
	// finalizes recording, removes the session and emits PhaseEnded even when no
	// terminate webhook ever arrives (e.g. ICE failure).
	sess.SetHandlers(
		func() {
			g.emit(context.Background(), GatewayEvent{CallID: callID, Phase: PhaseConnected, From: meta.from, PhoneNumberID: meta.phoneNumberID})
		},
		func() { g.finishCall(context.Background(), callID) },
	)

	answerSDP, err := sess.AnswerOffer(ctx, offer)
	if err != nil {
		_ = sess.Close()
		return fmt.Errorf("answer offer: %w", err)
	}
	if err := g.signaling.Accept(ctx, callID, answerSDP); err != nil {
		_ = sess.Close()
		return fmt.Errorf("accept: %w", err)
	}

	g.mu.Lock()
	// If a terminate already arrived while we were negotiating, don't register a
	// doomed session — tear it down instead of leaking it.
	if _, done := g.ended[callID]; done {
		g.mu.Unlock()
		_ = sess.Close()
		return nil
	}
	g.sessions[callID] = sess
	g.mu.Unlock()
	return nil
}

func (g *Gateway) onTerminate(ctx context.Context, _ string, ev WebhookCallEvent) {
	g.finishCall(ctx, ev.ID)
}

// finishCall marks a call terminal exactly once: it tears down any media
// session, finalizes the recording and emits PhaseEnded. Subsequent calls
// (duplicate terminate webhook, OnClosed after an explicit End, etc.) are
// no-ops, so the host receives a single terminal event per call.
func (g *Gateway) finishCall(ctx context.Context, callID string) {
	g.mu.Lock()
	if _, done := g.ended[callID]; done {
		g.mu.Unlock()
		return
	}
	g.ended[callID] = struct{}{}
	sess := g.sessions[callID]
	meta := g.meta[callID]
	delete(g.sessions, callID)
	delete(g.pending, callID)
	delete(g.meta, callID)
	g.mu.Unlock()

	rec := ""
	if sess != nil {
		rec = sess.RecordingPath()
		_ = sess.Close()
	}
	g.emit(ctx, GatewayEvent{
		CallID: callID, Phase: PhaseEnded, From: meta.from,
		PhoneNumberID: meta.phoneNumberID, RecordingPath: rec,
	})
}

// Session returns the active media session for a call (nil if none), e.g. to
// feed outgoing audio via WriteAudio.
func (g *Gateway) Session(callID string) *CallSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sessions[callID]
}

// Shutdown closes every active session.
func (g *Gateway) Shutdown() {
	g.mu.Lock()
	sessions := make([]*CallSession, 0, len(g.sessions))
	for _, s := range g.sessions {
		sessions = append(sessions, s)
	}
	g.sessions = make(map[string]*CallSession)
	g.pending = make(map[string]pendingCall)
	g.meta = make(map[string]callMeta)
	g.ended = make(map[string]struct{})
	g.mu.Unlock()

	for _, s := range sessions {
		_ = s.Close()
	}
}

func (g *Gateway) emit(ctx context.Context, evt GatewayEvent) {
	if g.handler != nil {
		g.handler(ctx, evt)
	}
}
