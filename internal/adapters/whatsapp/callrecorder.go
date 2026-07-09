package whatsapp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CallRecorder captures per-call PCM and writes a WAV file when the call ends.
//
// It is designed to never interfere with the live call: both audio callbacks
// (peer via the RTP decode loop, local via FeedCapturedPCM) do only a cheap
// in-memory append — which also copies the samples, so the engine's buffers are
// never aliased across goroutines. The single disk write happens in finish(),
// at call teardown, off the audio path entirely.
//
// Full-duplex: the remote peer leg and the local/outgoing leg are buffered
// separately and rendered as a stereo WAV (left = peer, right = local). When no
// local audio was fed (e.g. a listen-only call), a mono peer-only WAV is
// written instead. The two legs are aligned from call start on a per-sample
// index basis (each leg is 16 kHz mono); this is an approximation, not a
// timestamp-accurate sync.
type CallRecorder struct {
	dir        string
	sampleRate int
	maxSamples int

	mu   sync.Mutex
	bufs map[string]*callLegs
}

// callLegs holds the two directional buffers for one call.
type callLegs struct {
	peer []float32
	self []float32
}

// NewCallRecorder returns a recorder writing WAV files under dir (default
// "media/recordings"). sampleRate defaults to 16 kHz (the engine's PCM rate).
func NewCallRecorder(dir string, sampleRate int) *CallRecorder {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if strings.TrimSpace(dir) == "" {
		dir = filepath.Join("media", "recordings")
	}
	return &CallRecorder{
		dir:        dir,
		sampleRate: sampleRate,
		// ~30 min at 16 kHz mono per leg: a memory guard so a stuck/never-ending
		// call cannot grow a buffer without bound.
		maxSamples: 30 * 60 * sampleRate,
		bufs:       make(map[string]*callLegs),
	}
}

// writePeer appends remote-peer PCM. Cheap and non-blocking.
func (r *CallRecorder) writePeer(callID string, pcm []float32) {
	r.appendLeg(callID, pcm, true)
}

// writeSelf appends local/outgoing PCM. Cheap and non-blocking.
func (r *CallRecorder) writeSelf(callID string, pcm []float32) {
	r.appendLeg(callID, pcm, false)
}

func (r *CallRecorder) appendLeg(callID string, pcm []float32, peer bool) {
	if r == nil || callID == "" || len(pcm) == 0 {
		return
	}
	r.mu.Lock()
	legs := r.bufs[callID]
	if legs == nil {
		legs = &callLegs{}
		r.bufs[callID] = legs
	}
	dst := &legs.self
	if peer {
		dst = &legs.peer
	}
	room := r.maxSamples - len(*dst)
	if room <= 0 {
		r.mu.Unlock()
		return
	}
	if len(pcm) > room {
		pcm = pcm[:room]
	}
	*dst = append(*dst, pcm...)
	r.mu.Unlock()
}

// finish encodes the buffered audio to a WAV file and returns its path. It
// clears the call's buffers regardless of outcome. Recordings with less than
// half a second on both legs are discarded as noise.
func (r *CallRecorder) finish(callID string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("nil recorder")
	}
	r.mu.Lock()
	legs := r.bufs[callID]
	delete(r.bufs, callID)
	rate := r.sampleRate
	r.mu.Unlock()

	if legs == nil {
		return "", fmt.Errorf("no recording for call %s", callID)
	}
	if len(legs.peer) < rate/2 && len(legs.self) < rate/2 {
		return "", fmt.Errorf("recording too short")
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return "", err
	}

	var data []byte
	if len(legs.self) == 0 {
		// Listen-only / one-way: a mono peer file is the faithful rendering.
		data = encodeWavMonoPCM16(legs.peer, rate)
	} else {
		data = encodeWavStereoPCM16(legs.peer, legs.self, rate)
	}

	name := fmt.Sprintf("call_%s_%d.wav", sanitizeCallID(callID), time.Now().UnixMilli())
	full := filepath.Join(r.dir, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return full, nil
}

// discard drops a call's buffers without writing (e.g. rejected calls).
func (r *CallRecorder) discard(callID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.bufs, callID)
	r.mu.Unlock()
}

// sanitizeCallID keeps only filename-safe characters from a call id.
func sanitizeCallID(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, id)
}

// clampSample folds a float32 into the int16 PCM range.
func clampSample(s float32) int16 {
	if s > 1 {
		s = 1
	} else if s < -1 {
		s = -1
	}
	return int16(s * 32767)
}

// encodeWavMonoPCM16 renders float32 samples in [-1,1] as a 16-bit mono WAV.
func encodeWavMonoPCM16(pcm []float32, rate int) []byte {
	dataLen := len(pcm) * 2
	buf := bytes.NewBuffer(make([]byte, 0, 44+dataLen))
	writeWavHeader(buf, rate, 1, dataLen)
	for _, s := range pcm {
		_ = binary.Write(buf, binary.LittleEndian, clampSample(s))
	}
	return buf.Bytes()
}

// encodeWavStereoPCM16 renders two mono legs as an interleaved 16-bit stereo WAV
// (left = peer, right = local), padding the shorter leg with silence.
func encodeWavStereoPCM16(left, right []float32, rate int) []byte {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	dataLen := n * 2 * 2 // frames * channels * 2 bytes
	buf := bytes.NewBuffer(make([]byte, 0, 44+dataLen))
	writeWavHeader(buf, rate, 2, dataLen)
	for i := 0; i < n; i++ {
		var l, r float32
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		_ = binary.Write(buf, binary.LittleEndian, clampSample(l))
		_ = binary.Write(buf, binary.LittleEndian, clampSample(r))
	}
	return buf.Bytes()
}

// writeWavHeader writes a 44-byte canonical PCM WAV header.
func writeWavHeader(buf *bytes.Buffer, rate, channels, dataLen int) {
	byteRate := rate * channels * 2
	blockAlign := channels * 2
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(rate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16)) // bits/sample
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataLen))
}
