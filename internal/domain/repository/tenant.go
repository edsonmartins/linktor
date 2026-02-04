package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// TenantRepository defines the interface for tenant persistence
type TenantRepository interface {
	// Create creates a new tenant
	Create(ctx context.Context, tenant *entity.Tenant) error

	// FindByID finds a tenant by ID
	FindByID(ctx context.Context, id string) (*entity.Tenant, error)

	// FindBySlug finds a tenant by slug
	FindBySlug(ctx context.Context, slug string) (*entity.Tenant, error)

	// Update updates a tenant
	Update(ctx context.Context, tenant *entity.Tenant) error

	// Delete deletes a tenant
	Delete(ctx context.Context, id string) error
}
