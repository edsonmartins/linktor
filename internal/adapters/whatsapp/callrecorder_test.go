package whatsapp

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tone builds n samples of a trivial non-zero signal.
func tone(n int) []float32 {
	pcm := make([]float32, n)
	for i := range pcm {
		pcm[i] = 0.5
	}
	return pcm
}

func TestCallRecorder_PeerOnlyMono(t *testing.T) {
	dir := t.TempDir()
	r := NewCallRecorder(dir, 16000)

	r.writePeer("CALL-1", tone(8000))
	r.writePeer("CALL-1", tone(8000))

	path, err := r.finish("CALL-1")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "RIFF", string(data[0:4]))
	assert.Equal(t, "WAVE", string(data[8:12]))
	assert.Equal(t, "data", string(data[36:40]))
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(data[22:24]), "mono when no self leg")

	dataLen := binary.LittleEndian.Uint32(data[40:44])
	assert.Equal(t, uint32(16000*2), dataLen)
	assert.Equal(t, 44+int(dataLen), len(data))

	// Buffers cleared after finish.
	_, err = r.finish("CALL-1")
	assert.Error(t, err)
}

func TestCallRecorder_FullDuplexStereo(t *testing.T) {
	dir := t.TempDir()
	r := NewCallRecorder(dir, 16000)

	r.writePeer("C", tone(16000)) // 1s peer
	r.writeSelf("C", tone(8000))  // 0.5s local (shorter leg padded)

	path, err := r.finish("C")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, uint16(2), binary.LittleEndian.Uint16(data[22:24]), "stereo when both legs present")
	assert.Equal(t, uint32(16000), binary.LittleEndian.Uint32(data[24:28]))
	// block align = channels(2) * 2 bytes = 4
	assert.Equal(t, uint16(4), binary.LittleEndian.Uint16(data[32:34]))

	// frames = max(peer,self) = 16000; interleaved stereo => 16000*4 bytes.
	dataLen := binary.LittleEndian.Uint32(data[40:44])
	assert.Equal(t, uint32(16000*4), dataLen)

	// First frame: L (peer)=0.5 -> ~16383, R (self)=0.5 -> ~16383.
	l := int16(binary.LittleEndian.Uint16(data[44:46]))
	rr := int16(binary.LittleEndian.Uint16(data[46:48]))
	assert.InDelta(t, 16383, int(l), 2)
	assert.InDelta(t, 16383, int(rr), 2)

	// A frame past the self leg: R must be silence, L still peer.
	off := 44 + 10000*4
	l2 := int16(binary.LittleEndian.Uint16(data[off : off+2]))
	r2 := int16(binary.LittleEndian.Uint16(data[off+2 : off+4]))
	assert.InDelta(t, 16383, int(l2), 2)
	assert.Equal(t, int16(0), r2)
}

func TestCallRecorder_TooShortDiscarded(t *testing.T) {
	r := NewCallRecorder(t.TempDir(), 16000)
	r.writePeer("C", tone(100)) // << rate/2 and no self leg
	_, err := r.finish("C")
	assert.Error(t, err)
}

func TestCallRecorder_MemoryCapPerLeg(t *testing.T) {
	r := NewCallRecorder(t.TempDir(), 8000)
	r.writePeer("C", make([]float32, r.maxSamples))
	r.writePeer("C", tone(1000))
	r.mu.Lock()
	got := len(r.bufs["C"].peer)
	r.mu.Unlock()
	assert.Equal(t, r.maxSamples, got)
}

func TestCallRecorder_Discard(t *testing.T) {
	r := NewCallRecorder(t.TempDir(), 16000)
	r.writePeer("C", tone(16000))
	r.discard("C")
	_, err := r.finish("C")
	assert.Error(t, err)
}

func TestEncodeWavClipsAndSizes(t *testing.T) {
	wav := encodeWavMonoPCM16([]float32{2.0, -2.0, 0}, 16000)
	assert.Len(t, wav, 44+3*2)
	first := int16(binary.LittleEndian.Uint16(wav[44:46]))
	second := int16(binary.LittleEndian.Uint16(wav[46:48]))
	assert.Equal(t, int16(32767), first)
	assert.Equal(t, int16(-32767), second)
}

func TestSanitizeCallID(t *testing.T) {
	assert.Equal(t, "abc-123_XY", sanitizeCallID("abc-123_XY"))
	assert.Equal(t, "a_b_c", sanitizeCallID("a/b:c"))
}

func TestNilCallRecorderSafe(t *testing.T) {
	var r *CallRecorder
	assert.NotPanics(t, func() {
		r.writePeer("C", tone(10))
		r.writeSelf("C", tone(10))
		r.discard("C")
	})
}
