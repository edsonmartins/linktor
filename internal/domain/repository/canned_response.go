package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// CannedResponseRepository defines persistence for quick replies.
type CannedResponseRepository interface {
	Create(ctx context.Context, cr *entity.CannedResponse) error
	FindByID(ctx context.Context, id string) (*entity.CannedResponse, error)
	FindByShortcut(ctx context.Context, tenantID, shortcut string) (*entity.CannedResponse, error)
	FindByTenant(ctx context.Context, tenantID, search string, params *ListParams) ([]*entity.CannedResponse, int64, error)
	Update(ctx context.Context, cr *entity.CannedResponse) error
	Delete(ctx context.Context, id string) error
	IncrementUsage(ctx context.Context, id string) error
}
