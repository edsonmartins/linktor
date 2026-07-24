package outbound

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/metrics"
	"github.com/msgfy/linktor/pkg/logger"
)

// TemplateStatusProvider looks up the locally-synced template registry.
// Implemented by database.TemplateRepository. The guard queries it on every
// template send — never a long-lived cache — because Meta recategorizes and
// pauses templates on its own schedule; the local status is kept fresh by the
// template webhook (ProcessTemplateStatusWebhook) and periodic sync.
type TemplateStatusProvider interface {
	FindByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error)
}

// blockedTemplateStatuses are statuses Meta is known to reject at send time.
// The set is deliberately a blocklist, not an approval allowlist: anything not
// listed (APPROVED, REINSTATED, PENDING, unknown) passes through and Meta stays
// the final validator — a stale or unsynced local status must not block a
// template Meta would accept.
var blockedTemplateStatuses = map[entity.TemplateStatus]bool{
	entity.TemplateStatusRejected:        true,
	entity.TemplateStatusInAppeal:        true, // rejected until the appeal succeeds
	entity.TemplateStatusPaused:          true,
	entity.TemplateStatusDisabled:        true,
	entity.TemplateStatusDeleted:         true,
	entity.TemplateStatusPendingDeletion: true,
	entity.TemplateStatusArchived:        true,
	entity.TemplateStatusLimitExceeded:   true,
}

// NewTemplateStatusPolicy returns a PolicyDecorator that blocks (or observes,
// per mode) template sends whose locally-synced status is known-unusable, on
// whatsapp_official channels. Other channel types are returned unwrapped.
func NewTemplateStatusPolicy(mode PolicyMode, provider TemplateStatusProvider) PolicyDecorator {
	return func(channel *entity.Channel, s Sender) Sender {
		if mode == PolicyModeOff || channel.Type != entity.ChannelTypeWhatsAppOfficial {
			return s
		}
		return &templateStatusGuard{
			inner:     s,
			provider:  provider,
			mode:      mode,
			tenantID:  channel.TenantID,
			channelID: channel.ID,
		}
	}
}

type templateStatusGuard struct {
	inner     Sender
	provider  TemplateStatusProvider
	mode      PolicyMode
	tenantID  string
	channelID string
}

// Send consults the template status at send time. Unknown status — template
// not found locally or lookup failure — FAILS OPEN with a log line: the local
// registry may simply not have synced yet, and blocking an approved template
// over sync lag is a worse failure than letting Meta reject the send.
func (g *templateStatusGuard) Send(ctx context.Context, msg *Message) (*Receipt, error) {
	tpl, ok := msg.Content.(Template)
	if !ok {
		return g.inner.Send(ctx, msg)
	}

	record, err := g.provider.FindByName(ctx, g.tenantID, g.channelID, tpl.Name, tpl.Language)
	if err != nil || record == nil {
		logger.Warn("outbound: template status unknown for " + tpl.Name + " (" + tpl.Language + "), allowing send; Meta remains the validator")
		return g.inner.Send(ctx, msg)
	}
	if !blockedTemplateStatuses[record.Status] {
		return g.inner.Send(ctx, msg)
	}

	if g.mode == PolicyModeDryRun {
		metrics.RecordGuardBlocked(string(entity.ChannelTypeWhatsAppOfficial), metrics.BlockReasonTemplateRejected, "dry_run")
		logger.Warn("outbound: [dry-run] template " + tpl.Name + " has status " + string(record.Status) + " and would be blocked under enforcement (message " + msg.ID + ")")
		return g.inner.Send(ctx, msg)
	}

	metrics.RecordGuardBlocked(string(entity.ChannelTypeWhatsAppOfficial), metrics.BlockReasonTemplateRejected, "enforce")
	return nil, Blocked(metrics.BlockReasonTemplateRejected,
		"template %s (%s) is not usable: status %s", tpl.Name, tpl.Language, record.Status)
}
