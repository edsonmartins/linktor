package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// AuditLogRepository implements repository.AuditLogRepository with PostgreSQL.
type AuditLogRepository struct {
	db *PostgresDB
}

// NewAuditLogRepository creates a new PostgreSQL audit log repository.
func NewAuditLogRepository(db *PostgresDB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create appends an audit entry.
func (r *AuditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	changes, err := json.Marshal(log.Changes)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal audit changes")
	}

	query := `
		INSERT INTO audit_logs (
			id, tenant_id, actor_id, actor_email, actor_name, action,
			resource_type, resource_id, changes, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		log.ID,
		log.TenantID,
		nullString(log.ActorID),
		nullString(log.ActorEmail),
		nullString(log.ActorName),
		log.Action,
		nullString(log.ResourceType),
		nullString(log.ResourceID),
		changes,
		nullString(log.IPAddress),
		nullString(log.UserAgent),
		log.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create audit log")
	}

	return nil
}

// FindByTenant lists audit entries for a tenant, most recent first.
func (r *AuditLogRepository) FindByTenant(ctx context.Context, tenantID string, filters *repository.AuditLogFilters, params *repository.ListParams) ([]*entity.AuditLog, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}
	if filters == nil {
		filters = &repository.AuditLogFilters{}
	}

	// Build dynamic WHERE clause.
	where := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	idx := 2
	if filters.ActorID != "" {
		where += " AND actor_id = $" + itoa(idx)
		args = append(args, filters.ActorID)
		idx++
	}
	if filters.Action != "" {
		where += " AND action = $" + itoa(idx)
		args = append(args, filters.Action)
		idx++
	}
	if filters.ResourceType != "" {
		where += " AND resource_type = $" + itoa(idx)
		args = append(args, filters.ResourceType)
		idx++
	}
	if filters.ResourceID != "" {
		where += " AND resource_id = $" + itoa(idx)
		args = append(args, filters.ResourceID)
		idx++
	}
	if filters.Actor != "" {
		where += " AND (actor_email ILIKE $" + itoa(idx) + " OR actor_name ILIKE $" + itoa(idx) + " OR actor_id::text = $" + itoa(idx+1) + ")"
		args = append(args, "%"+filters.Actor+"%", filters.Actor)
		idx += 2
	}
	if !filters.StartTime.IsZero() {
		where += " AND created_at >= $" + itoa(idx)
		args = append(args, filters.StartTime)
		idx++
	}
	if !filters.EndTime.IsZero() {
		where += " AND created_at <= $" + itoa(idx)
		args = append(args, filters.EndTime)
		idx++
	}

	var total int64
	if err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count audit logs")
	}

	query := "SELECT id, tenant_id, actor_id, actor_email, actor_name, action, resource_type, resource_id, changes, ip_address, user_agent, created_at FROM audit_logs " +
		where + " ORDER BY created_at DESC LIMIT $" + itoa(idx) + " OFFSET $" + itoa(idx+1)
	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query audit logs")
	}
	defer rows.Close()

	var logs []*entity.AuditLog
	for rows.Next() {
		log, err := scanAuditLog(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

func scanAuditLog(row pgx.Row) (*entity.AuditLog, error) {
	var a entity.AuditLog
	var actorID, actorEmail, actorName, resourceType, resourceID, ipAddress, userAgent *string
	var changes []byte

	err := row.Scan(
		&a.ID, &a.TenantID, &actorID, &actorEmail, &actorName, &a.Action,
		&resourceType, &resourceID, &changes, &ipAddress, &userAgent, &a.CreatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan audit log")
	}

	a.ActorID = derefString(actorID)
	a.ActorEmail = derefString(actorEmail)
	a.ActorName = derefString(actorName)
	a.ResourceType = derefString(resourceType)
	a.ResourceID = derefString(resourceID)
	a.IPAddress = derefString(ipAddress)
	a.UserAgent = derefString(userAgent)

	if len(changes) > 0 {
		_ = json.Unmarshal(changes, &a.Changes)
	}
	if a.Changes == nil {
		a.Changes = make(map[string]interface{})
	}

	return &a, nil
}
