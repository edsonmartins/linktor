package outbound

import (
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/logger"
)

// PolicyMode controls the rollout stage of a delivery policy guard (INV-015).
// The two-stage rollout is deliberate: policies change production behavior, so
// they must run observably (dry_run) before they block (enforce), and enforce
// must be revertible to dry_run by configuration alone — no deploy.
type PolicyMode string

const (
	// PolicyModeOff disables the policy entirely.
	PolicyModeOff PolicyMode = "off"
	// PolicyModeDryRun evaluates the policy, records metric+log for what WOULD
	// be blocked, and lets the send proceed unchanged.
	PolicyModeDryRun PolicyMode = "dry_run"
	// PolicyModeEnforce blocks violating sends before the provider call.
	PolicyModeEnforce PolicyMode = "enforce"
)

// ParsePolicyMode maps a configuration string to a PolicyMode. Empty and
// unknown values fall back to dry_run — the stage-1 default that never changes
// send behavior — and unknown values are logged so a typo cannot silently
// flip enforcement on or off.
func ParsePolicyMode(v string) PolicyMode {
	switch PolicyMode(v) {
	case PolicyModeOff, PolicyModeDryRun, PolicyModeEnforce:
		return PolicyMode(v)
	case "":
		return PolicyModeDryRun
	default:
		logger.Warn("outbound: unknown policy mode " + v + ", falling back to dry_run")
		return PolicyModeDryRun
	}
}

// PolicyDecorator wraps a channel's Sender with a delivery policy. Decorators
// registered via Resolver.AddPolicy are applied in registration order; the
// sandbox delivery guard is always applied last (outermost), so security
// screening happens before any policy evaluation.
type PolicyDecorator func(channel *entity.Channel, s Sender) Sender
