package repository

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// AuditLogFilters narrows an audit log listing.
type AuditLogFilters struct {
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	// Actor matches actor_email/actor_name/actor_id (case-insensitive
	// substring) so a human can filter by who acted without knowing the UUID.
	Actor string
	// StartTime/EndTime bound created_at (inclusive). Zero value = unbounded.
	StartTime time.Time
	EndTime   time.Time
}

// AuditLogRepository defines persistence for the audit trail.
type AuditLogRepository interface {
	// Create appends an audit entry.
	Create(ctx context.Context, log *entity.AuditLog) error

	// FindByTenant lists audit entries for a tenant, most recent first.
	FindByTenant(ctx context.Context, tenantID string, filters *AuditLogFilters, params *ListParams) ([]*entity.AuditLog, int64, error)
}
