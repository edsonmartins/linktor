package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// RoleRepository defines persistence for RBAC roles and their permissions.
type RoleRepository interface {
	Create(ctx context.Context, role *entity.Role) error
	FindByID(ctx context.Context, id string) (*entity.Role, error)
	FindByName(ctx context.Context, tenantID, name string) (*entity.Role, error)
	FindByTenant(ctx context.Context, tenantID string) ([]*entity.Role, error)
	Update(ctx context.Context, role *entity.Role) error
	Delete(ctx context.Context, id string) error
}
