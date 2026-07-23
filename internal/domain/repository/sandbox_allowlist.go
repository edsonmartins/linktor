package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// SandboxAllowlistRepository defines persistence for sandbox recipient
// allowlists (INV-017). All operations are tenant-scoped: lookups by ID take
// the tenant explicitly so a cross-tenant access fails as not-found instead of
// leaking or silently succeeding.
type SandboxAllowlistRepository interface {
	Create(ctx context.Context, entry *entity.SandboxAllowlistEntry) error
	FindByID(ctx context.Context, tenantID, id string) (*entity.SandboxAllowlistEntry, error)
	FindByTenant(ctx context.Context, tenantID string) ([]*entity.SandboxAllowlistEntry, error)
	Delete(ctx context.Context, tenantID, id string) error
	// IsAllowed reports whether the (already normalized) recipient is
	// authorized for the given sandbox channel: either by a tenant-wide entry
	// or by one scoped to that channel. Consulted at send time by the sandbox
	// delivery guard — never cached by the caller.
	IsAllowed(ctx context.Context, tenantID, channelID, recipient string) (bool, error)
}
