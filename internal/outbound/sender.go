package outbound

import (
	"context"
	"errors"
	"fmt"
)

// Receipt is the result of a successful send.
type Receipt struct {
	// ProviderMessageID is the message ID assigned by the provider (e.g. Meta's
	// wamid), used later to correlate delivery/read status webhooks.
	ProviderMessageID string
}

// Sender delivers a single message on one channel. Implementations are
// stateless w.r.t. application state and safe for concurrent use: a Sender
// holds one channel's resolved credentials/client and nothing per-message.
type Sender interface {
	Send(ctx context.Context, msg *Message) (*Receipt, error)
}

// Factory builds a Sender for a channel type from that channel's credentials.
// One Factory is registered per ChannelType at composition time; the Resolver
// calls New once per channel and caches the result.
type Factory interface {
	// ChannelType returns the channel type this factory serves (e.g. "whatsapp_official").
	ChannelType() string
	// New builds a Sender from a channel's merged credentials+config map.
	New(credentials map[string]string) (Sender, error)
}

// PermanentError marks a failure that must NOT be retried (invalid template,
// bad recipient, auth rejected). The worker acks these after recording failure.
// Any other (transient) error is retried via NATS redelivery.
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return "permanent: " + e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// Permanent wraps err as a non-retryable failure.
func Permanent(err error) error { return &PermanentError{Err: err} }

// Permanentf wraps a formatted message as a non-retryable failure.
func Permanentf(format string, args ...interface{}) error {
	return &PermanentError{Err: fmt.Errorf(format, args...)}
}

// IsPermanent reports whether err (or anything it wraps) is a PermanentError.
func IsPermanent(err error) bool {
	var p *PermanentError
	return errors.As(err, &p)
}
