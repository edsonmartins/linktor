package repository

import "context"

// SLARepository runs the time-based queries the SLA monitor needs over the
// conversations table.
type SLARepository interface {
	// FindForAutoClose returns IDs of open/pending conversations whose last
	// message is older than idleMinutes.
	FindForAutoClose(ctx context.Context, tenantID string, idleMinutes int) ([]string, error)
	// FindFirstResponseBreaches returns IDs of open/pending conversations with no
	// agent reply that are older than minutes and not yet flagged as breached.
	FindFirstResponseBreaches(ctx context.Context, tenantID string, minutes int) ([]string, error)
	// MarkBreached flags a conversation as having breached its SLA.
	MarkBreached(ctx context.Context, id string) error
	// Close resolves a conversation (status=resolved, resolved_at=now).
	Close(ctx context.Context, id string) error
}
