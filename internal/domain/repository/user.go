package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *entity.User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id string) (*entity.User, error)

	// FindByEmail finds a user by email (across all tenants)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByTenantAndEmail finds a user by tenant ID and email
	FindByTenantAndEmail(ctx context.Context, tenantID, email string) (*entity.User, error)

	// FindByTenant finds all users for a tenant
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.User, int64, error)

	// Update updates a user
	Update(ctx context.Context, user *entity.User) error

	// Delete deletes a user
	Delete(ctx context.Context, id string) error

	// CountByTenant counts users for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// FindAvailableAgents finds available agents for a channel
	FindAvailableAgents(ctx context.Context, tenantID, channelID string) ([]*entity.User, error)
}

// ListParams represents pagination and filtering parameters
type ListParams struct {
	Page     int
	PageSize int
	SortBy   string
	SortDir  string
	Filters  map[string]interface{}
}

// NewListParams creates default list parameters
func NewListParams() *ListParams {
	return &ListParams{
		Page:     1,
		PageSize: 20,
		SortBy:   "created_at",
		SortDir:  "desc",
		Filters:  make(map[string]interface{}),
	}
}

// Offset returns the offset for pagination
func (p *ListParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the limit for pagination
func (p *ListParams) Limit() int {
	if p.PageSize <= 0 {
		return 20
	}
	if p.PageSize > 100 {
		return 100
	}
	return p.PageSize
}
