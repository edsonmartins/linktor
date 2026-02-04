package service

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// UpdateTenantInput represents input for updating a tenant
type UpdateTenantInput struct {
	Name     *string
	Plan     *entity.Plan
	Settings map[string]string
}

// TenantUsage represents tenant usage statistics
type TenantUsage struct {
	Users            int64 `json:"users"`
	Channels         int64 `json:"channels"`
	Contacts         int64 `json:"contacts"`
	MessagesThisMonth int64 `json:"messages_this_month"`
	Limits           *entity.TenantLimits `json:"limits"`
}

// TenantService handles tenant operations
type TenantService struct {
	tenantRepo repository.TenantRepository
	userRepo   repository.UserRepository
}

// NewTenantService creates a new tenant service
func NewTenantService(tenantRepo repository.TenantRepository, userRepo repository.UserRepository) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
	}
}

// GetByID returns a tenant by ID
func (s *TenantService) GetByID(ctx context.Context, id string) (*entity.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeTenantNotFound, "Tenant not found")
	}
	return tenant, nil
}

// Update updates a tenant
func (s *TenantService) Update(ctx context.Context, id string, input *UpdateTenantInput) (*entity.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeTenantNotFound, "Tenant not found")
	}

	if input.Name != nil {
		tenant.Name = *input.Name
	}
	if input.Plan != nil {
		tenant.Plan = *input.Plan
		tenant.Limits = entity.GetPlanLimits(*input.Plan)
	}
	if input.Settings != nil {
		for k, v := range input.Settings {
			tenant.Settings[k] = v
		}
	}

	tenant.UpdatedAt = time.Now()

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to update tenant")
	}

	return tenant, nil
}

// GetUsage returns tenant usage statistics
func (s *TenantService) GetUsage(ctx context.Context, tenantID string) (*TenantUsage, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeTenantNotFound, "Tenant not found")
	}

	userCount, err := s.userRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to count users")
	}

	// TODO: Count channels, contacts, and messages

	return &TenantUsage{
		Users:             userCount,
		Channels:         0, // TODO
		Contacts:         0, // TODO
		MessagesThisMonth: 0, // TODO
		Limits:           tenant.Limits,
	}, nil
}
