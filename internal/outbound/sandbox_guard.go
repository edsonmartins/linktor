package outbound

import (
	"context"
	"fmt"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/metrics"
)

// AllowlistChecker answers whether a recipient is authorized for a tenant's
// sandbox channel. Implemented by database.SandboxAllowlistRepository. The
// guard calls it on EVERY send (INV-017): the allowlist is never captured at
// sender construction nor cached here, so removing a recipient takes effect on
// the very next send — the Resolver's sender cache only reuses this guard's
// wiring, not any allowlist state.
type AllowlistChecker interface {
	IsAllowed(ctx context.Context, tenantID, channelID, recipient string) (bool, error)
}

// sandboxGuardedTypes are the channel types whose sandbox delivery semantics
// (E.164 recipient + allowlist) are defined in this phase. A sandbox channel of
// any OTHER type fails closed: silently delivering unguarded synthetic traffic
// is the one unacceptable failure mode, so an undefined case blocks.
var sandboxGuardedTypes = map[string]bool{
	string(entity.ChannelTypeWhatsApp):           true,
	string(entity.ChannelTypeWhatsAppOfficial):   true,
	string(entity.ChannelTypeWhatsAppUnofficial): true,
}

// sandboxGuard is a Sender decorator applied by the Resolver to every sandbox
// channel. It sits in the single delivery funnel (worker → Resolver → Sender),
// so API sends, campaigns, bots and NATS retries are all covered; production
// channels never receive it and keep their exact current behavior.
type sandboxGuard struct {
	inner       Sender
	allowlist   AllowlistChecker
	tenantID    string
	channelID   string
	channelType string
}

func newSandboxGuard(inner Sender, allowlist AllowlistChecker, tenantID, channelID, channelType string) *sandboxGuard {
	return &sandboxGuard{
		inner:       inner,
		allowlist:   allowlist,
		tenantID:    tenantID,
		channelID:   channelID,
		channelType: channelType,
	}
}

// Send blocks before any network call to the provider unless the recipient is
// explicitly allowlisted. Blocks are permanent errors (no retry); an allowlist
// lookup failure is transient — the send is NOT attempted (fail closed) and
// NATS redelivers it.
func (g *sandboxGuard) Send(ctx context.Context, msg *Message) (*Receipt, error) {
	if !sandboxGuardedTypes[g.channelType] {
		metrics.RecordGuardBlocked(g.channelType, metrics.BlockReasonUnsupportedSandbox, "enforce")
		return nil, Blocked(metrics.BlockReasonUnsupportedSandbox,
			"sandbox guard: channel type %q has no sandbox delivery semantics yet; refusing to send", g.channelType)
	}

	recipient, ok := entity.NormalizeE164(msg.To)
	if !ok {
		metrics.RecordGuardBlocked(g.channelType, metrics.BlockReasonInvalidRecipient, "enforce")
		return nil, Blocked(metrics.BlockReasonInvalidRecipient,
			"sandbox guard: recipient %s is not a valid E.164 number", entity.MaskRecipient(msg.To))
	}

	allowed, err := g.allowlist.IsAllowed(ctx, g.tenantID, g.channelID, recipient)
	if err != nil {
		// Transient: never fall through to the provider on a failed check.
		return nil, fmt.Errorf("sandbox guard: allowlist check failed (send withheld): %w", err)
	}
	if !allowed {
		metrics.RecordGuardBlocked(g.channelType, metrics.BlockReasonAllowlist, "enforce")
		return nil, Blocked(metrics.BlockReasonAllowlist,
			"sandbox guard: recipient %s is not in the tenant's sandbox allowlist", entity.MaskRecipient(recipient))
	}

	return g.inner.Send(ctx, msg)
}
