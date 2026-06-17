package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// AssignmentRepository implements repository.AssignmentRepository.
type AssignmentRepository struct {
	db *PostgresDB
}

// NewAssignmentRepository creates a new PostgreSQL assignment repository.
func NewAssignmentRepository(db *PostgresDB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

// FindAssignableAgent picks an eligible agent according to the strategy.
func (r *AssignmentRepository) FindAssignableAgent(ctx context.Context, tenantID string, strategy entity.AssignmentStrategy) (string, error) {
	var query string
	switch strategy {
	case entity.AssignmentLoadBalanced:
		// Fewest open/pending conversations, then least-recently assigned.
		query = `
			SELECT u.id
			FROM users u
			LEFT JOIN (
				SELECT assignee_id, COUNT(*) AS open_count
				FROM conversations
				WHERE status IN ('open', 'pending') AND assignee_id IS NOT NULL
				GROUP BY assignee_id
			) oc ON oc.assignee_id = u.id
			WHERE u.tenant_id = $1
			  AND u.status = 'active'
			  AND u.is_available = true
			  AND u.role IN ('agent', 'supervisor', 'admin', 'owner')
			ORDER BY COALESCE(oc.open_count, 0) ASC, u.last_assigned_at ASC NULLS FIRST
			LIMIT 1`
	default: // round_robin
		query = `
			SELECT id
			FROM users
			WHERE tenant_id = $1
			  AND status = 'active'
			  AND is_available = true
			  AND role IN ('agent', 'supervisor', 'admin', 'owner')
			ORDER BY last_assigned_at ASC NULLS FIRST
			LIMIT 1`
	}

	var userID string
	err := r.db.Pool.QueryRow(ctx, query, tenantID).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, errors.ErrCodeInternal, "failed to find assignable agent")
	}
	return userID, nil
}

// MarkAssigned updates the agent's round-robin cursor.
func (r *AssignmentRepository) MarkAssigned(ctx context.Context, userID string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE users SET last_assigned_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to mark agent assigned")
	}
	return nil
}
