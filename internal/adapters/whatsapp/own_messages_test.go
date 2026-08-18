package whatsapp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// WhatsApp marks two different things with IsFromMe: the echo of what this
// client sent, and what the person holding the paired phone typed. Only the id
// separates them, and these tests protect that separation — getting it wrong
// either duplicates every outgoing message or silently deletes the operator's
// side of every conversation.

func TestSentRegistry_OnlyIdsWeHandedOutAreOurs(t *testing.T) {
	now := time.Now()
	r := newSentRegistry()
	r.remember("nosso", now)

	assert.True(t, r.isOurs("nosso", now))
	assert.False(t, r.isOurs("do-aparelho", now), "id que não emitimos não é eco nosso")
}

func TestSentRegistry_ForgetsAfterTheWindow(t *testing.T) {
	// Past the window we no longer claim to know. Forwarding then may duplicate,
	// which is visible and fixable; the alternative is dropping a real message,
	// which is neither.
	now := time.Now()
	r := newSentRegistry()
	r.remember("antigo", now)

	assert.False(t, r.isOurs("antigo", now.Add(sentTTL+time.Second)))
	assert.True(t, r.isOurs("antigo", now.Add(sentTTL-time.Second)))
}

func TestSentRegistry_PruneKeepsWhatIsStillRecent(t *testing.T) {
	// The prune runs on size but drops on age only. A young id evicted to make
	// room would come back as an echo we no longer recognize.
	now := time.Now()
	r := newSentRegistry()
	for i := 0; i < sentRegistrySoftMax+1; i++ {
		r.remember(idOf(i), now.Add(-2*sentTTL))
	}
	r.remember("recente", now)

	assert.True(t, r.isOurs("recente", now))
	assert.False(t, r.isOurs(idOf(0), now), "os vencidos saíram")
}

func TestSentRegistry_NilAndEmptyAreSafe(t *testing.T) {
	// An Adapter built without a registry has sent nothing, so nothing is its
	// echo — and an empty id identifies no message at all.
	var r *sentRegistry
	now := time.Now()

	r.remember("x", now)
	assert.False(t, r.isOurs("x", now))
	assert.False(t, newSentRegistry().isOurs("", now))
}

func idOf(i int) string {
	return "msg-" + time.Duration(i).String()
}
