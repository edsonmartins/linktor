package officialcalls

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hostOnly keeps ICE local (no STUN/network) for fast, offline tests.
var hostOnly = SessionConfig{ICEServers: []webrtc.ICEServer{}}

// newOfferer builds a plain pion peer with an outgoing OPUS track that produces
// an SDP offer — standing in for the WhatsApp user's client.
func newOfferer(t *testing.T) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticSample) {
	t.Helper()
	me := &webrtc.MediaEngine{}
	require.NoError(t, me.RegisterDefaultCodecs())
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "user")
	require.NoError(t, err)
	_, err = pc.AddTrack(track)
	require.NoError(t, err)
	return pc, track
}

func offerSDP(t *testing.T, pc *webrtc.PeerConnection) string {
	t.Helper()
	offer, err := pc.CreateOffer(nil)
	require.NoError(t, err)
	gather := webrtc.GatheringCompletePromise(pc)
	require.NoError(t, pc.SetLocalDescription(offer))
	<-gather
	return pc.LocalDescription().SDP
}

func TestAnswerOffer_ProducesValidAnswer(t *testing.T) {
	offerer, _ := newOfferer(t)
	defer offerer.Close()
	offer := offerSDP(t, offerer)

	s, err := NewCallSession("call-1", hostOnly)
	require.NoError(t, err)
	defer s.Close()

	answer, err := s.AnswerOffer(context.Background(), offer)
	require.NoError(t, err)
	assert.Contains(t, answer, "m=audio")
	assert.Contains(t, answer, "a=setup:")
	assert.True(t, strings.Contains(answer, "opus") || strings.Contains(answer, "OPUS"))
}

func TestCreateOffer_ProducesAudioOffer(t *testing.T) {
	s, err := NewCallSession("call-2", hostOnly)
	require.NoError(t, err)
	defer s.Close()

	offer, err := s.CreateOffer(context.Background())
	require.NoError(t, err)
	assert.Contains(t, offer, "m=audio")
}

func TestRecordingPathAndCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, err := NewCallSession("call-3", SessionConfig{ICEServers: []webrtc.ICEServer{}, RecordDir: dir})
	require.NoError(t, err)
	// No audio arrived yet, so there is no recording file — RecordingPath must be
	// empty rather than point at a file that was never created.
	assert.Empty(t, s.RecordingPath())
	assert.NoError(t, s.Close())
	assert.NoError(t, s.Close(), "Close is idempotent")

	noRec, err := NewCallSession("c", hostOnly)
	require.NoError(t, err)
	defer noRec.Close()
	assert.Empty(t, noRec.RecordingPath())
}

// End-to-end over loopback: the offerer connects to the CallSession and streams
// OPUS; the session must reach a connected state and start recording to Ogg.
func TestSession_LoopbackConnectAndRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping media loopback test in -short mode")
	}
	dir := t.TempDir()
	offerer, track := newOfferer(t)
	defer offerer.Close()

	answerer, err := NewCallSession("loop-1", SessionConfig{ICEServers: []webrtc.ICEServer{}, RecordDir: dir})
	require.NoError(t, err)
	defer answerer.Close()

	connected := make(chan struct{}, 1)
	answerer.SetHandlers(func() {
		select {
		case connected <- struct{}{}:
		default:
		}
	}, nil)

	// SDP exchange.
	offer := offerSDP(t, offerer)
	answer, err := answerer.AnswerOffer(context.Background(), offer)
	require.NoError(t, err)
	require.NoError(t, offerer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer, SDP: answer,
	}))

	// Stream a valid Opus silence frame repeatedly once connected.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		silence := []byte{0xf8, 0xff, 0xfe} // minimal Opus frame
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = track.WriteSample(media.Sample{Data: silence, Duration: 20 * time.Millisecond})
			}
		}
	}()

	select {
	case <-connected:
	case <-time.After(20 * time.Second):
		t.Fatal("media did not connect over loopback")
	}

	// Give a few frames time to be recorded, then confirm the Ogg file grew.
	require.Eventually(t, func() bool {
		info, err := os.Stat(answerer.RecordingPath())
		return err == nil && info.Size() > 0
	}, 5*time.Second, 100*time.Millisecond, "recording file should be created and non-empty")
}
