package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/msgfy/linktor/internal/voip/call"
	"github.com/msgfy/linktor/internal/voip/core"
	"github.com/msgfy/linktor/internal/voip/wasocket"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Native WhatsApp calling over the unofficial multi-device protocol. This
// bridges whatsmeow's low-level call signaling events to the ported VoIP engine
// (CallManager + mlow codec under internal/voip). It is deliberately isolated
// from the messaging event loop so the heavier call machinery can evolve —
// and its whatsmeow version requirements be managed — independently.

// CallEventType enumerates the call lifecycle transitions surfaced to the host.
type CallEventType string

const (
	CallEventIncoming CallEventType = "incoming"
	CallEventState    CallEventType = "state"
	CallEventEnded    CallEventType = "ended"
)

// CallEvent is a call lifecycle notification delivered to the registered handler.
type CallEvent struct {
	Type    CallEventType `json:"type"`
	CallID  string        `json:"call_id"`
	PeerID  string        `json:"peer_id"`  // phone-number user part when resolvable
	PeerJID string        `json:"peer_jid"` // raw peer JID
	IsVideo bool          `json:"is_video"`
	State   string        `json:"state,omitempty"`
	Reason  string        `json:"reason,omitempty"`

	// RecordingPath is set on the ended event when call recording is enabled and
	// a WAV file was written for this call.
	RecordingPath string `json:"recording_path,omitempty"`
}

// CallHandler receives call lifecycle events. Handlers must not block.
type CallHandler func(ctx context.Context, evt CallEvent)

// PeerAudioSink receives decoded 16 kHz mono PCM from the remote peer during a
// call — the hook to plug in recording and/or speech-to-text.
type PeerAudioSink func(callID string, pcm []float32)

// CallGateway maintains one CallManager per active call and routes whatsmeow
// call-signaling events into the VoIP engine.
type CallGateway struct {
	client  *whatsmeow.Client
	socket  core.VoipSocket
	log     *slog.Logger
	handler CallHandler

	// AudioSink, when set, receives the remote peer's decoded PCM for every
	// active call (recording / transcription). Optional.
	AudioSink PeerAudioSink

	// Recorder, when set, captures each call's peer audio to a WAV file, flushed
	// when the call ends (the ended CallEvent carries RecordingPath). Optional.
	Recorder *CallRecorder

	mu    sync.Mutex
	calls map[string]*call.CallManager
}

// NewCallGateway wires a gateway onto a connected whatsmeow client.
func NewCallGateway(client *whatsmeow.Client, log *slog.Logger, handler CallHandler) *CallGateway {
	if log == nil {
		log = slog.Default()
	}
	return &CallGateway{
		client:  client,
		socket:  wasocket.NewSocket(client),
		log:     log,
		handler: handler,
		calls:   make(map[string]*call.CallManager),
	}
}

// HandleWhatsmeowEvent routes a raw whatsmeow event to the VoIP engine. Register
// it with client.AddEventHandler; it ignores non-call events.
func (g *CallGateway) HandleWhatsmeowEvent(rawEvt any) {
	ctx := context.Background()
	switch evt := rawEvt.(type) {
	case *events.CallOffer:
		g.onOffer(ctx, evt)
	case *events.CallAccept:
		if m := g.get(evt.CallID); m != nil {
			m.HandleCallAccept(ctx, wrapCall(evt.From, evt.Data), evt.From)
		}
	case *events.CallPreAccept:
		if m := g.get(evt.CallID); m != nil {
			m.HandleCallPreAccept(ctx, wrapCall(evt.From, evt.Data), evt.From)
		}
	case *events.CallTransport:
		if m := g.get(evt.CallID); m != nil {
			m.HandleCallTransport(ctx, wrapCall(evt.From, evt.Data), evt.From)
		}
	case *events.CallTerminate:
		if m := g.get(evt.CallID); m != nil {
			m.HandleCallTerminate(wrapCall(evt.From, evt.Data))
			g.remove(evt.CallID)
		}
	case *events.CallReject:
		if m := g.get(evt.CallID); m != nil {
			m.HandleCallTerminate(wrapCall(evt.From, evt.Data))
			g.remove(evt.CallID)
		}
	}
}

// PlaceCall starts an outgoing call and returns its call id.
func (g *CallGateway) PlaceCall(ctx context.Context, to string, isVideo bool) (string, error) {
	if g.client == nil {
		return "", ErrClientNotReady
	}
	jid, err := types.ParseJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}
	callID := string(g.client.GenerateMessageID())
	m := g.newManager(callID)
	m.StartCall(ctx, callID, jid, isVideo)
	return callID, nil
}

// AcceptCall answers a ringing inbound call.
func (g *CallGateway) AcceptCall(ctx context.Context, callID string) error {
	m := g.get(callID)
	if m == nil {
		return fmt.Errorf("no active call %s", callID)
	}
	m.AcceptCall(ctx, callID)
	return nil
}

// RejectCall declines a ringing inbound call.
func (g *CallGateway) RejectCall(ctx context.Context, callID string) error {
	m := g.get(callID)
	if m == nil {
		return fmt.Errorf("no active call %s", callID)
	}
	m.RejectCall(ctx, callID, core.EndCallReasonDeclined)
	if g.Recorder != nil {
		g.Recorder.discard(callID)
	}
	g.remove(callID)
	return nil
}

// EndCall hangs up an in-progress call.
func (g *CallGateway) EndCall(ctx context.Context, callID string) error {
	m := g.get(callID)
	if m == nil {
		return fmt.Errorf("no active call %s", callID)
	}
	m.EndCall(ctx, core.EndCallReasonUserEnded)
	g.remove(callID)
	return nil
}

// Shutdown ends every active call and clears the registry.
func (g *CallGateway) Shutdown(ctx context.Context) {
	g.mu.Lock()
	managers := make([]*call.CallManager, 0, len(g.calls))
	for _, m := range g.calls {
		managers = append(managers, m)
	}
	g.calls = make(map[string]*call.CallManager)
	g.mu.Unlock()

	for _, m := range managers {
		m.EndCall(ctx, core.EndCallReasonUserEnded)
	}
}

// ── internals ────────────────────────────────────────────────────────

func (g *CallGateway) onOffer(ctx context.Context, evt *events.CallOffer) {
	callID := evt.CallID
	if callID == "" {
		return
	}
	m := g.newManager(callID)
	m.HandleCallOffer(ctx, wrapCall(evt.From, evt.Data), evt.From)
}

func (g *CallGateway) newManager(callID string) *call.CallManager {
	m := call.NewCallManager(g.socket, g.log)

	m.OnIncoming = func(ci *call.CallInfo) { g.emit(callEvent(CallEventIncoming, ci)) }
	m.OnStateChange = func(ci *call.CallInfo) { g.emit(callEvent(CallEventState, ci)) }
	m.OnEnded = func(ci *call.CallInfo) {
		evt := callEvent(CallEventEnded, ci)
		// Flush the recording (if any) off the audio path, at teardown.
		if g.Recorder != nil {
			if path, err := g.Recorder.finish(ci.CallID); err == nil {
				evt.RecordingPath = path
			}
		}
		g.emit(evt)
		g.remove(ci.CallID)
	}
	// Feed the peer PCM to the recorder and/or the generic sink. The callback
	// only copies bytes and returns, so it never blocks the RTP decode loop.
	if g.Recorder != nil || g.AudioSink != nil {
		m.OnPeerAudio = func(pcm []float32) {
			if g.Recorder != nil {
				g.Recorder.writePeer(callID, pcm)
			}
			if g.AudioSink != nil {
				g.AudioSink(callID, pcm)
			}
		}
	}
	// Tap the local/outgoing leg for full-duplex recording.
	if g.Recorder != nil {
		m.OnSelfAudio = func(pcm []float32) { g.Recorder.writeSelf(callID, pcm) }
	}

	g.mu.Lock()
	g.calls[callID] = m
	g.mu.Unlock()
	return m
}

func (g *CallGateway) get(callID string) *call.CallManager {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls[callID]
}

func (g *CallGateway) remove(callID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.calls, callID)
}

func (g *CallGateway) emit(evt CallEvent) {
	if g.handler != nil {
		g.handler(context.Background(), evt)
	}
}

// callEvent maps a VoIP CallInfo to a host-facing CallEvent.
func callEvent(t CallEventType, ci *call.CallInfo) CallEvent {
	if ci == nil {
		return CallEvent{Type: t}
	}
	peerID := ci.CallerPn
	if peerID == "" {
		if jid, err := types.ParseJID(ci.PeerJid); err == nil {
			peerID = jid.User
		}
	}
	return CallEvent{
		Type:    t,
		CallID:  ci.CallID,
		PeerID:  peerID,
		PeerJID: ci.PeerJid,
		IsVideo: ci.MediaType == core.CallMediaTypeVideo,
		State:   string(ci.StateData.State),
		Reason:  string(ci.StateData.EndReason),
	}
}

// wrapCall re-wraps a call event's inner data node in the <call from=...>
// envelope the signaling parser expects.
func wrapCall(from types.JID, inner *waBinary.Node) *waBinary.Node {
	content := []waBinary.Node{}
	if inner != nil {
		content = append(content, *inner)
	}
	return &waBinary.Node{
		Tag:     "call",
		Attrs:   waBinary.Attrs{"from": from},
		Content: content,
	}
}
