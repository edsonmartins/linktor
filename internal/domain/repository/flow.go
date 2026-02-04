package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// FlowRepository defines the interface for flow persistence
type FlowRepository interface {
	// Create creates a new flow
	Create(ctx context.Context, flow *entity.Flow) error

	// FindByID finds a flow by ID
	FindByID(ctx context.Context, id string) (*entity.Flow, error)

	// FindByTenant finds flows for a tenant with pagination
	FindByTenant(ctx context.Context, tenantID string, filter *entity.FlowFilter, params *ListParams) ([]*entity.Flow, int64, error)

	// FindByBot finds flows assigned to a specific bot
	FindByBot(ctx context.Context, botID string) ([]*entity.Flow, error)

	// FindActiveByTenant finds all active flows for a tenant
	FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Flow, error)

	// FindByTrigger finds flows that match a trigger type and value
	FindByTrigger(ctx context.Context, tenantID string, trigger entity.FlowTriggerType, triggerValue string) ([]*entity.Flow, error)

	// Update updates a flow
	Update(ctx context.Context, flow *entity.Flow) error

	// UpdateStatus activates or deactivates a flow
	UpdateStatus(ctx context.Context, id string, isActive bool) error

	// Delete deletes a flow
	Delete(ctx context.Context, id string) error

	// CountByTenant counts flows for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// CountByBot counts flows for a bot
	CountByBot(ctx context.Context, botID string) (int64, error)
}
