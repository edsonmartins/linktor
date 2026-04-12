package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// APIKeyRepository defines persistence for tenant-scoped API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *entity.APIKey) error
	ListByTenant(ctx context.Context, tenantID string) ([]*entity.APIKey, error)
	Delete(ctx context.Context, tenantID, id string) error
}
