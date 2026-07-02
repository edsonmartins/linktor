package repository

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// APIKeyRepository defines persistence for tenant-scoped API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *entity.APIKey) error
	ListByTenant(ctx context.Context, tenantID string) ([]*entity.APIKey, error)
	Delete(ctx context.Context, tenantID, id string) error
	// FindActiveByPrefix returns non-expired keys whose stored prefix matches,
	// including the key hash so callers can verify the presented secret. Used by
	// X-API-Key authentication (server-to-server / DeskLenz outbound).
	FindActiveByPrefix(ctx context.Context, prefix string) ([]*entity.APIKey, error)
	// TouchLastUsed records that a key authenticated a request (best effort).
	TouchLastUsed(ctx context.Context, id string, at time.Time) error
}
