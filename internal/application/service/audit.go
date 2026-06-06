package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/logger"
)

// Actor identifies who performed an audited action.
type Actor struct {
	ID    string
	Email string
	Name  string
	IP    string
	Agent string
}

// AuditService records and queries the audit trail. Recording failures are
// logged but never propagated, so auditing can never break a business action.
type AuditService struct {
	repo repository.AuditLogRepository
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Record persists an audit entry. It is best-effort: errors are logged, not returned.
func (s *AuditService) Record(ctx context.Context, tenantID string, actor Actor, action, resourceType, resourceID string, changes map[string]interface{}) {
	if s == nil || s.repo == nil {
		return
	}

	log := entity.NewAuditLog(tenantID, action)
	log.ID = uuid.New().String()
	log.ActorID = actor.ID
	log.ActorEmail = actor.Email
	log.ActorName = actor.Name
	log.IPAddress = actor.IP
	log.UserAgent = actor.Agent
	log.ResourceType = resourceType
	log.ResourceID = resourceID
	if changes != nil {
		log.Changes = changes
	}

	if err := s.repo.Create(ctx, log); err != nil {
		logger.Error("failed to record audit log: " + err.Error())
	}
}

// List returns audit entries for a tenant.
func (s *AuditService) List(ctx context.Context, tenantID string, filters *repository.AuditLogFilters, params *repository.ListParams) ([]*entity.AuditLog, int64, error) {
	return s.repo.FindByTenant(ctx, tenantID, filters, params)
}
