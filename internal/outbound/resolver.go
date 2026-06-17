package outbound

import (
	"context"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// senderCacheTTL bounds how long a built Sender is reused before its channel
// credentials are reloaded. Invalidate busts an entry immediately on change.
const senderCacheTTL = 5 * time.Minute

type cachedSender struct {
	sender    Sender
	expiresAt time.Time
}

// Resolver maps a channel ID to a ready Sender, building one per channel from
// its (decrypted) credentials and caching the result. This is the single place
// that answers "give me something that can send on channel X", replacing the
// shared-mutable-adapter approach.
type Resolver struct {
	channelRepo repository.ChannelRepository
	factories   map[string]Factory // channelType -> factory

	mu    sync.Mutex
	cache map[string]cachedSender // channelID -> sender
	now   func() time.Time
}

// NewResolver creates a Resolver. Register factories with Register before use.
func NewResolver(channelRepo repository.ChannelRepository) *Resolver {
	return &Resolver{
		channelRepo: channelRepo,
		factories:   make(map[string]Factory),
		cache:       make(map[string]cachedSender),
		now:         time.Now,
	}
}

// Register adds a per-type Sender factory. Call once per supported channel type
// at composition time.
func (r *Resolver) Register(f Factory) {
	r.factories[f.ChannelType()] = f
}

// Supports reports whether a channel type has a registered factory.
func (r *Resolver) Supports(channelType string) bool {
	_, ok := r.factories[channelType]
	return ok
}

// SupportedTypes lists the channel types with a registered factory.
func (r *Resolver) SupportedTypes() []string {
	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}

// For returns a Sender for the given channel, building and caching it on miss.
func (r *Resolver) For(ctx context.Context, channelID string) (Sender, error) {
	r.mu.Lock()
	if c, ok := r.cache[channelID]; ok && r.now().Before(c.expiresAt) {
		r.mu.Unlock()
		return c.sender, nil
	}
	r.mu.Unlock()

	channel, err := r.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	factory, ok := r.factories[string(channel.Type)]
	if !ok {
		return nil, errors.New(errors.ErrCodeBadRequest, "no outbound sender for channel type "+string(channel.Type))
	}

	// Merge config and credentials; credentials win on key collisions since they
	// are the canonical secret store.
	creds := make(map[string]string, len(channel.Config)+len(channel.Credentials))
	for k, v := range channel.Config {
		creds[k] = v
	}
	for k, v := range channel.Credentials {
		creds[k] = v
	}

	sender, err := factory.New(creds)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache[channelID] = cachedSender{sender: sender, expiresAt: r.now().Add(senderCacheTTL)}
	r.mu.Unlock()

	return sender, nil
}

// Invalidate drops a channel's cached Sender so the next For rebuilds it.
// Call after a channel's credentials change or it disconnects.
func (r *Resolver) Invalidate(channelID string) {
	r.mu.Lock()
	delete(r.cache, channelID)
	r.mu.Unlock()
}
