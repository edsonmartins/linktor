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
// It is designed to never interfere with the live call: the audio callback runs
// on the RTP decode loop, so write() does only a cheap in-memory append (which
// also copies the samples, so the decoder's buffer is never aliased across
// goroutines). The single disk write happens in finish(), at call teardown —
// off the audio path entirely.
//
// The recording covers ONLY the remote peer's leg. Capturing the local/outgoing
// leg as well (full-duplex) would require tapping the engine's playback path.
type CallRecorder struct {
	dir        string
	sampleRate int
	maxSamples int

	mu   sync.Mutex
	bufs map[string][]float32
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
		// ~30 min at 16 kHz mono: a memory guard so a stuck/never-ending call
		// cannot grow the buffer without bound.
		maxSamples: 30 * 60 * sampleRate,
		bufs:       make(map[string][]float32),
	}
}

// write appends peer PCM for a call. Cheap and non-blocking — safe to call from
// the audio callback.
func (r *CallRecorder) write(callID string, pcm []float32) {
	if r == nil || callID == "" || len(pcm) == 0 {
		return
	}
	r.mu.Lock()
	buf := r.bufs[callID]
	room := r.maxSamples - len(buf)
	if room <= 0 {
		r.mu.Unlock()
		return
	}
	if len(pcm) > room {
		pcm = pcm[:room]
	}
	r.bufs[callID] = append(buf, pcm...)
	r.mu.Unlock()
}

// finish encodes the buffered audio to a WAV file and returns its path. It
// clears the call's buffer regardless of outcome. Recordings shorter than half
// a second are discarded as noise.
func (r *CallRecorder) finish(callID string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("nil recorder")
	}
	r.mu.Lock()
	pcm := r.bufs[callID]
	delete(r.bufs, callID)
	rate := r.sampleRate
	r.mu.Unlock()

	if len(pcm) < rate/2 {
		return "", fmt.Errorf("recording too short")
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("call_%s_%d.wav", sanitizeCallID(callID), time.Now().UnixMilli())
	full := filepath.Join(r.dir, name)
	if err := os.WriteFile(full, encodeWavMonoPCM16(pcm, rate), 0o644); err != nil {
		return "", err
	}
	return full, nil
}

// discard drops a call's buffer without writing (e.g. rejected calls).
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

// encodeWavMonoPCM16 renders float32 samples in [-1,1] as a 16-bit mono WAV.
func encodeWavMonoPCM16(pcm []float32, rate int) []byte {
	dataLen := len(pcm) * 2
	buf := bytes.NewBuffer(make([]byte, 0, 44+dataLen))
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))      // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))      // channels
	_ = binary.Write(buf, binary.LittleEndian, uint32(rate))   // sample rate
	_ = binary.Write(buf, binary.LittleEndian, uint32(rate*2)) // byte rate
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))      // block align
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))     // bits/sample
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataLen))
	for _, s := range pcm {
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		_ = binary.Write(buf, binary.LittleEndian, int16(s*32767))
	}
	return buf.Bytes()
}
