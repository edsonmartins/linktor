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
}

type pendingCall struct {
	from  string
	offer string
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
	g.emit(ctx, GatewayEvent{CallID: ev.ID, Phase: PhaseRinging, From: ev.From, PhoneNumberID: phoneNumberID})

	if !g.autoAnswer {
		g.mu.Lock()
		g.pending[ev.ID] = pendingCall{from: ev.From, offer: ev.Session.SDP}
		g.mu.Unlock()
		return
	}
	if err := g.answer(ctx, ev.ID, ev.From, phoneNumberID, ev.Session.SDP); err != nil {
		g.log.Error("failed to answer inbound call", "call_id", ev.ID, "err", err)
		_ = g.signaling.Reject(ctx, ev.ID)
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
	return g.answer(ctx, callID, p.from, "", p.offer)
}

// RejectCall declines a ringing inbound call.
func (g *Gateway) RejectCall(ctx context.Context, callID string) error {
	g.mu.Lock()
	delete(g.pending, callID)
	g.mu.Unlock()
	return g.signaling.Reject(ctx, callID)
}

// EndCall terminates an active call and tears down its media.
func (g *Gateway) EndCall(ctx context.Context, callID string) error {
	g.mu.Lock()
	sess := g.sessions[callID]
	delete(g.sessions, callID)
	g.mu.Unlock()
	if sess != nil {
		_ = sess.Close()
	}
	return g.signaling.Terminate(ctx, callID)
}

// answer builds the media session, negotiates the SDP answer and accepts.
func (g *Gateway) answer(ctx context.Context, callID, from, phoneNumberID, offer string) error {
	sess, err := NewCallSession(callID, g.sessCfg)
	if err != nil {
		return fmt.Errorf("media session: %w", err)
	}
	sess.OnConnected = func() {
		g.emit(context.Background(), GatewayEvent{CallID: callID, Phase: PhaseConnected, From: from, PhoneNumberID: phoneNumberID})
	}

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
	g.sessions[callID] = sess
	g.mu.Unlock()
	return nil
}

func (g *Gateway) onTerminate(ctx context.Context, phoneNumberID string, ev WebhookCallEvent) {
	g.mu.Lock()
	sess := g.sessions[ev.ID]
	delete(g.sessions, ev.ID)
	delete(g.pending, ev.ID)
	g.mu.Unlock()

	rec := ""
	if sess != nil {
		rec = sess.RecordingPath()
		_ = sess.Close()
	}
	g.emit(ctx, GatewayEvent{
		CallID: ev.ID, Phase: PhaseEnded, From: ev.From,
		PhoneNumberID: phoneNumberID, RecordingPath: rec,
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
