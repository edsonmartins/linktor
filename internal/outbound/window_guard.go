package outbound

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/metrics"
	"github.com/msgfy/linktor/pkg/logger"
)

// sessionWindow is Meta's customer-service window: free-form messages are only
// accepted within 24h of the customer's last inbound message; outside it, only
// approved templates go through. Policy extracted from the orphaned
// whatsapp_official.SessionInfo.IsSessionValid (which fed off in-memory adapter
// state the active flow never populates); the durable source of truth here is
// the messages table via LastInboundProvider.
const sessionWindow = 24 * time.Hour

// LastInboundProvider reports when the contact last messaged in a conversation.
// Implemented by database.MessageRepository.LastInboundAt.
type LastInboundProvider interface {
	LastInboundAt(ctx context.Context, conversationID string) (*time.Time, error)
}

// windowOpen is the extracted, pure 24h-window rule.
func windowOpen(lastInbound *time.Time, now time.Time) bool {
	return lastInbound != nil && now.Sub(*lastInbound) < sessionWindow
}

// NewSessionWindowPolicy returns a PolicyDecorator enforcing (or observing,
// per mode) the WhatsApp Cloud API 24h window on whatsapp_official channels.
// Other channel types are returned unwrapped.
func NewSessionWindowPolicy(mode PolicyMode, provider LastInboundProvider) PolicyDecorator {
	return func(channel *entity.Channel, s Sender) Sender {
		if mode == PolicyModeOff || channel.Type != entity.ChannelTypeWhatsAppOfficial {
			return s
		}
		return &sessionWindowGuard{inner: s, provider: provider, mode: mode, now: time.Now}
	}
}

type sessionWindowGuard struct {
	inner    Sender
	provider LastInboundProvider
	mode     PolicyMode
	now      func() time.Time
}

// Send blocks (enforce) or records (dry_run) free-form sends outside the 24h
// customer-service window. Unlike the sandbox guard, uncertainty FAILS OPEN:
// this is a provider-policy optimization, not a security boundary — a false
// positive blocks a legitimate customer message, which is worse than letting
// Meta reject the occasional out-of-window send.
func (g *sessionWindowGuard) Send(ctx context.Context, msg *Message) (*Receipt, error) {
	// Approved-template sends are exactly what the window rule prescribes.
	if msg.Content != nil && msg.Content.Kind() == KindTemplate {
		return g.inner.Send(ctx, msg)
	}
	// Without a conversation (e.g. direct/VRE sends) the window cannot be
	// evaluated; pass through rather than guess.
	if msg.ConversationID == "" {
		return g.inner.Send(ctx, msg)
	}

	lastInbound, err := g.provider.LastInboundAt(ctx, msg.ConversationID)
	if err != nil {
		// Fail open (G-006) — but observably (INV-024): a systematic lookup
		// failure would otherwise disable enforcement while the block counter
		// only shows an "improvement".
		metrics.RecordGuardFailOpen(string(entity.ChannelTypeWhatsAppOfficial),
			metrics.BlockReasonWindow24h, metrics.FailOpenCauseLookupError, string(g.mode))
		logger.Warn("outbound: 24h-window evaluation FAILED OPEN for message " + msg.ID +
			" (conversation " + msg.ConversationID + "): last-inbound lookup error: " + err.Error())
		return g.inner.Send(ctx, msg)
	}
	if windowOpen(lastInbound, g.now()) {
		return g.inner.Send(ctx, msg)
	}

	if g.mode == PolicyModeDryRun {
		metrics.RecordGuardBlocked(string(entity.ChannelTypeWhatsAppOfficial), metrics.BlockReasonWindow24h, "dry_run")
		logger.Warn("outbound: [dry-run] free-form message " + msg.ID + " is outside the 24h window and would be blocked under enforcement")
		return g.inner.Send(ctx, msg)
	}

	metrics.RecordGuardBlocked(string(entity.ChannelTypeWhatsAppOfficial), metrics.BlockReasonWindow24h, "enforce")
	return nil, Blocked(metrics.BlockReasonWindow24h,
		"24h window: customer-service window expired for this conversation; send an approved template instead")
}
