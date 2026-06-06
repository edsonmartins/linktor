package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// AuditLogFilters narrows an audit log listing.
type AuditLogFilters struct {
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
}

// AuditLogRepository defines persistence for the audit trail.
type AuditLogRepository interface {
	// Create appends an audit entry.
	Create(ctx context.Context, log *entity.AuditLog) error

	// FindByTenant lists audit entries for a tenant, most recent first.
	FindByTenant(ctx context.Context, tenantID string, filters *AuditLogFilters, params *ListParams) ([]*entity.AuditLog, int64, error)
}
