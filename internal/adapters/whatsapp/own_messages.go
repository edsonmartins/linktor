package whatsapp

import (
	"sync"
	"time"
)

// sentTTL is how long a message id stays remembered as "we sent this".
// The echo, when it comes, arrives within seconds of the send; ten minutes is
// slack for a reconnect replaying a recent batch, and short enough that the
// registry stays small on a busy channel.
const sentTTL = 10 * time.Minute

// sentRegistrySoftMax triggers a prune. It is not a cap: entries are dropped by
// age, never by count, because evicting a young id would turn a real echo back
// into a message the app stores twice.
const sentRegistrySoftMax = 2048

// sentRegistry remembers the ids of messages this client itself sent.
//
// It exists to tell two very different things apart, because WhatsApp marks
// both with IsFromMe: the echo of a message Linktor sent (already stored by the
// application — replaying it would duplicate) and a message the operator typed
// on the paired phone (which the application has never seen, and which is half
// of every conversation when the phone is still in use).
//
// Only the first is ours to drop, and only ids we handed out identify it.
type sentRegistry struct {
	mu  sync.Mutex
	ids map[string]time.Time
}

func newSentRegistry() *sentRegistry {
	return &sentRegistry{ids: make(map[string]time.Time)}
}

// remember records an id as sent by us, at the given moment.
//
// now is a parameter so the caller — and the tests — decide what time it is
// rather than the registry reading the clock behind their back.
func (r *sentRegistry) remember(id string, now time.Time) {
	if r == nil || id == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids[id] = now
	if len(r.ids) > sentRegistrySoftMax {
		r.pruneLocked(now)
	}
}

// isOurs reports whether the id was sent by this client and is still within the
// retention window.
//
// An id older than sentTTL answers false: past that point we cannot claim to
// know, and the safe answer is to let the message through. A duplicate is
// visible and fixable; a silently dropped message is neither.
func (r *sentRegistry) isOurs(id string, now time.Time) bool {
	if r == nil || id == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	at, ok := r.ids[id]
	return ok && now.Sub(at) <= sentTTL
}

func (r *sentRegistry) pruneLocked(now time.Time) {
	for id, at := range r.ids {
		if now.Sub(at) > sentTTL {
			delete(r.ids, id)
		}
	}
}
