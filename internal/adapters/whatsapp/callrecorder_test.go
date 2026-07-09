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

func TestCallRecorder_WriteFinishProducesWav(t *testing.T) {
	dir := t.TempDir()
	r := NewCallRecorder(dir, 16000)

	// 1 second of audio across two writes.
	r.write("CALL-1", tone(8000))
	r.write("CALL-1", tone(8000))

	path, err := r.finish("CALL-1")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// RIFF/WAVE header sanity.
	assert.Equal(t, "RIFF", string(data[0:4]))
	assert.Equal(t, "WAVE", string(data[8:12]))
	assert.Equal(t, "data", string(data[36:40]))

	// data chunk length == samples * 2 bytes.
	dataLen := binary.LittleEndian.Uint32(data[40:44])
	assert.Equal(t, uint32(16000*2), dataLen)
	assert.Equal(t, 44+int(dataLen), len(data))

	// Sample rate in the fmt chunk.
	assert.Equal(t, uint32(16000), binary.LittleEndian.Uint32(data[24:28]))

	// Buffer is cleared after finish.
	_, err = r.finish("CALL-1")
	assert.Error(t, err)
}

func TestCallRecorder_TooShortDiscarded(t *testing.T) {
	r := NewCallRecorder(t.TempDir(), 16000)
	r.write("C", tone(100)) // << rate/2
	_, err := r.finish("C")
	assert.Error(t, err)
}

func TestCallRecorder_MemoryCap(t *testing.T) {
	r := NewCallRecorder(t.TempDir(), 8000)
	// maxSamples = 30*60*8000. Write beyond it and confirm it stops growing.
	r.write("C", make([]float32, r.maxSamples))
	r.write("C", tone(1000))
	r.mu.Lock()
	got := len(r.bufs["C"])
	r.mu.Unlock()
	assert.Equal(t, r.maxSamples, got)
}

func TestCallRecorder_Discard(t *testing.T) {
	r := NewCallRecorder(t.TempDir(), 16000)
	r.write("C", tone(16000))
	r.discard("C")
	_, err := r.finish("C")
	assert.Error(t, err, "discarded call has no buffer to flush")
}

func TestEncodeWavClipsAndSizes(t *testing.T) {
	wav := encodeWavMonoPCM16([]float32{2.0, -2.0, 0}, 16000)
	assert.Len(t, wav, 44+3*2)
	// Clipped: +2.0 -> +32767, -2.0 -> -32767.
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
		r.write("C", tone(10))
		r.discard("C")
	})
}
