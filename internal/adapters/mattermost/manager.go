package mattermost

import (
	"context"
	"sync"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/logger"
)

// Manager owns the inbound WebSocket listeners for connected Mattermost
// channels. It is started once at boot (one reconnecting listener per
// already-connected channel) and also reacts to channel connect/disconnect at
// runtime via StartChannel/StopChannel, so newly connected channels begin
// streaming immediately without a restart.
type Manager struct {
	channelRepo repository.ChannelRepository
	producer    nats.Publisher
	store       storage.Client // re-hosts inbound file attachments; may be nil

	mu      sync.Mutex
	baseCtx context.Context               // lifetime ceiling for all listeners
	cancels map[string]context.CancelFunc // channelID -> listener cancel
}

// NewManager builds a Mattermost inbound manager. store may be nil to disable
// inbound file re-hosting (text still flows through).
func NewManager(channelRepo repository.ChannelRepository, producer nats.Publisher, store storage.Client) *Manager {
	return &Manager{
		channelRepo: channelRepo,
		producer:    producer,
		store:       store,
		cancels:     map[string]context.CancelFunc{},
	}
}

// Start binds the manager's lifetime to ctx and launches a listener for every
// enabled+connected Mattermost channel. Listeners run until ctx is cancelled or
// the channel is explicitly stopped.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	m.baseCtx = ctx
	m.mu.Unlock()

	channels, err := m.channelRepo.FindByTypes(ctx, []entity.ChannelType{entity.ChannelTypeMattermost})
	if err != nil {
		return err
	}

	started := 0
	for _, channel := range channels {
		if !channel.IsConnected() {
			continue
		}
		if m.StartChannel(channel) {
			started++
		}
	}

	if started > 0 {
		logger.Info("mattermost: started inbound listeners for connected channels")
	}
	return nil
}

// StartChannel launches (or restarts) the inbound listener for one channel.
// It is safe to call before Start (the manager has no base context yet, so it
// no-ops) and idempotent: an existing listener for the same channel is stopped
// first. Returns true when a listener was started.
func (m *Manager) StartChannel(channel *entity.Channel) bool {
	if channel == nil {
		return false
	}
	if channel.Credentials[CredBaseURL] == "" || channel.Credentials[CredBotToken] == "" {
		logger.Warn("mattermost: channel " + channel.ID + " missing base_url/bot_token; skipping")
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.baseCtx == nil {
		// Manager not started yet; boot Start will pick this channel up.
		return false
	}
	m.stopLocked(channel.ID)

	ctx, cancel := context.WithCancel(m.baseCtx)
	m.cancels[channel.ID] = cancel

	// The listener loops until ctx is cancelled (by StopChannel/restart or by
	// baseCtx on shutdown); whoever cancels also removes the map entry, so no
	// self-cleanup is needed here.
	go newListener(channel, m.producer, m.store).run(ctx)
	return true
}

// StopChannel stops the inbound listener for one channel, if running.
func (m *Manager) StopChannel(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked(channelID)
}

// stopLocked cancels and forgets a channel's listener. Caller holds m.mu.
func (m *Manager) stopLocked(channelID string) {
	if cancel, ok := m.cancels[channelID]; ok {
		cancel()
		delete(m.cancels, channelID)
	}
}
