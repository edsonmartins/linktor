package service

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// SettingsService reads and writes per-tenant operational settings.
type SettingsService struct {
	repo repository.TenantSettingsRepository
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(repo repository.TenantSettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

// SettingsInput carries updatable settings fields.
type SettingsInput struct {
	AssignmentStrategy      string
	AutoAssignEnabled       bool
	SLAFirstResponseMinutes int
	SLAResolutionMinutes    int
	AutoCloseAfterMinutes   int
}

// Get returns the tenant's settings (defaults if unset).
func (s *SettingsService) Get(ctx context.Context, tenantID string) (*entity.TenantSettings, error) {
	return s.repo.Get(ctx, tenantID)
}

// Update validates and persists the tenant's settings.
func (s *SettingsService) Update(ctx context.Context, tenantID string, in *SettingsInput) (*entity.TenantSettings, error) {
	strategy := entity.AssignmentStrategy(in.AssignmentStrategy)
	switch strategy {
	case entity.AssignmentManual, entity.AssignmentRoundRobin, entity.AssignmentLoadBalanced:
	case "":
		strategy = entity.AssignmentManual
	default:
		return nil, errors.Validation("invalid assignment_strategy")
	}
	if in.SLAFirstResponseMinutes < 0 || in.SLAResolutionMinutes < 0 || in.AutoCloseAfterMinutes < 0 {
		return nil, errors.Validation("SLA minutes must be non-negative")
	}

	settings := entity.DefaultTenantSettings(tenantID)
	settings.AssignmentStrategy = strategy
	settings.AutoAssignEnabled = in.AutoAssignEnabled
	settings.SLAFirstResponseMinutes = in.SLAFirstResponseMinutes
	settings.SLAResolutionMinutes = in.SLAResolutionMinutes
	settings.AutoCloseAfterMinutes = in.AutoCloseAfterMinutes

	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, tenantID)
}
