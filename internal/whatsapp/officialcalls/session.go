package officialcalls

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

// SessionConfig configures the WebRTC media for a call.
type SessionConfig struct {
	// ICEServers are the STUN/TURN servers. If nil, a public STUN server is used;
	// pass a non-nil empty slice for host-candidate-only (e.g. tests).
	ICEServers []webrtc.ICEServer
	// RecordDir, when set, records the remote audio to an Ogg/Opus file there.
	// Recording writes the received OPUS RTP directly (no decode), so it is
	// pure-Go and adds no codec dependency.
	RecordDir string
}

// CallSession is one call's WebRTC media leg — standard ICE + DTLS + SRTP with
// OPUS, handled by pion. Unlike the unofficial whatsmeow path it needs no
// proprietary codec or relay.
type CallSession struct {
	callID string
	pc     *webrtc.PeerConnection
	local  *webrtc.TrackLocalStaticSample

	mu         sync.Mutex
	ogg        *oggwriter.OggWriter
	recPath    string
	recStarted bool // true once the Ogg file has actually been created
	closed     bool

	// Handlers are guarded by mu (set via SetHandlers, read when firing) and each
	// fires at most once: onConnected when media connects, onClosed at teardown.
	onConnected    func()
	onClosed       func()
	connectedFired bool
	closedFired    bool
}

// SetHandlers registers the connect/close callbacks. onConnected fires at most
// once when media connects; onClosed fires at most once when the call is torn
// down (peer failure or Close). Both are guarded so pion's state-change
// goroutine never races the caller setting them.
func (s *CallSession) SetHandlers(onConnected, onClosed func()) {
	s.mu.Lock()
	s.onConnected = onConnected
	s.onClosed = onClosed
	s.mu.Unlock()
}

// NewCallSession builds a media session for callID.
func NewCallSession(callID string, cfg SessionConfig) (*CallSession, error) {
	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("register codecs: %w", err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me))

	iceServers := cfg.ICEServers
	if iceServers == nil {
		iceServers = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}
	}
	pc, err := api.NewPeerConnection(webrtc.Configuration{ICEServers: iceServers})
	if err != nil {
		return nil, fmt.Errorf("new peer connection: %w", err)
	}

	s := &CallSession{callID: callID, pc: pc}

	// Outgoing OPUS track so the business can send audio (agent/TTS).
	local, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "linktor-"+sanitizeID(callID))
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("new local track: %w", err)
	}
	if _, err := pc.AddTrack(local); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add track: %w", err)
	}
	s.local = local

	if cfg.RecordDir != "" {
		if err := os.MkdirAll(cfg.RecordDir, 0o755); err == nil {
			s.recPath = filepath.Join(cfg.RecordDir, "call_"+sanitizeID(callID)+".ogg")
		}
	}

	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			s.recordTrack(track)
		}
	})
	pc.OnConnectionStateChange(func(st webrtc.PeerConnectionState) {
		switch st {
		case webrtc.PeerConnectionStateConnected:
			s.fireConnected()
		// Disconnected is transient (ICE may recover), so it is not terminal;
		// only Failed/Closed end the call.
		case webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			s.fireClosed()
		}
	})

	return s, nil
}

func (s *CallSession) fireConnected() {
	s.mu.Lock()
	if s.connectedFired {
		s.mu.Unlock()
		return
	}
	s.connectedFired = true
	cb := s.onConnected
	s.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func (s *CallSession) fireClosed() {
	s.mu.Lock()
	if s.closedFired {
		s.mu.Unlock()
		return
	}
	s.closedFired = true
	cb := s.onClosed
	s.mu.Unlock()
	if cb != nil {
		cb()
	}
}

// AnswerOffer sets the remote SDP offer, creates our answer, waits for ICE
// gathering (Meta uses non-trickle), and returns the local answer SDP.
func (s *CallSession) AnswerOffer(ctx context.Context, offerSDP string) (string, error) {
	if err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer, SDP: offerSDP,
	}); err != nil {
		return "", fmt.Errorf("set remote offer: %w", err)
	}
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("create answer: %w", err)
	}
	gather := webrtc.GatheringCompletePromise(s.pc)
	if err := s.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("set local answer: %w", err)
	}
	select {
	case <-gather:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return s.pc.LocalDescription().SDP, nil
}

// CreateOffer produces a local SDP offer (business-initiated call), waiting for
// ICE gathering.
func (s *CallSession) CreateOffer(ctx context.Context) (string, error) {
	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("create offer: %w", err)
	}
	gather := webrtc.GatheringCompletePromise(s.pc)
	if err := s.pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("set local offer: %w", err)
	}
	select {
	case <-gather:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return s.pc.LocalDescription().SDP, nil
}

// AcceptAnswer applies the remote SDP answer (for the offer/business-initiated flow).
func (s *CallSession) AcceptAnswer(answerSDP string) error {
	return s.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer, SDP: answerSDP,
	})
}

// WriteAudio sends an OPUS-encoded sample to the peer.
func (s *CallSession) WriteAudio(sample media.Sample) error {
	return s.local.WriteSample(sample)
}

// RecordingPath returns the Ogg file path, or empty if recording is off or no
// audio ever arrived (so callers never get a path to a file that was never
// created).
func (s *CallSession) RecordingPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.recStarted {
		return ""
	}
	return s.recPath
}

// Close finalizes the recording and tears down the peer connection. Idempotent.
func (s *CallSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	// Finalize the writer under the lock so it can never race a concurrent
	// WriteRTP in recordTrack (oggwriter is not safe for concurrent use).
	if s.ogg != nil {
		_ = s.ogg.Close()
		s.ogg = nil
	}
	s.mu.Unlock()
	return s.pc.Close()
}

// recordTrack streams the remote OPUS RTP into an Ogg file. It copies packets
// straight through (no decode), so it never blocks on codec work. Every write
// happens under the session lock, serialized against Close.
func (s *CallSession) recordTrack(track *webrtc.TrackRemote) {
	s.mu.Lock()
	if s.recPath == "" || s.closed {
		s.mu.Unlock()
		return
	}
	if s.ogg == nil {
		channels := track.Codec().Channels
		if channels == 0 {
			channels = 2
		}
		ogg, err := oggwriter.New(s.recPath, 48000, channels)
		if err != nil {
			s.mu.Unlock()
			return
		}
		s.ogg = ogg
		s.recStarted = true
	}
	s.mu.Unlock()

	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.closed || s.ogg == nil {
			s.mu.Unlock()
			return
		}
		_ = s.ogg.WriteRTP(pkt)
		s.mu.Unlock()
	}
}

// sanitizeID keeps only filename-safe characters.
func sanitizeID(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, id)
}
