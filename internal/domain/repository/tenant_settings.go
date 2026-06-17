package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// TenantSettingsRepository persists per-tenant operational settings.
type TenantSettingsRepository interface {
	// Get returns the settings for a tenant, or default settings if none exist.
	Get(ctx context.Context, tenantID string) (*entity.TenantSettings, error)
	// Upsert creates or updates the settings for a tenant.
	Upsert(ctx context.Context, settings *entity.TenantSettings) error
	// ListWithSLA returns settings for all tenants that have any SLA/auto-close
	// rule enabled. Used by the SLA monitor to limit its scan.
	ListWithSLA(ctx context.Context) ([]*entity.TenantSettings, error)
}
