package database

import (
	"context"

	"github.com/msgfy/linktor/pkg/errors"
)

// SLARepository implements repository.SLARepository.
type SLARepository struct {
	db *PostgresDB
}

// NewSLARepository creates a new PostgreSQL SLA repository.
func NewSLARepository(db *PostgresDB) *SLARepository {
	return &SLARepository{db: db}
}

// FindForAutoClose returns open/pending conversations idle longer than idleMinutes.
// Idleness is measured from the most recent message, falling back to creation time.
func (r *SLARepository) FindForAutoClose(ctx context.Context, tenantID string, idleMinutes int) ([]string, error) {
	query := `
		SELECT c.id
		FROM conversations c
		WHERE c.tenant_id = $1
		  AND c.status IN ('open', 'pending')
		  AND COALESCE(
		        (SELECT MAX(m.created_at) FROM messages m WHERE m.conversation_id = c.id),
		        c.created_at
		      ) < NOW() - ($2 * INTERVAL '1 minute')`
	return r.queryIDs(ctx, query, tenantID, idleMinutes)
}

// FindFirstResponseBreaches returns conversations awaiting a first agent reply
// past the SLA window that have not yet been flagged.
func (r *SLARepository) FindFirstResponseBreaches(ctx context.Context, tenantID string, minutes int) ([]string, error) {
	query := `
		SELECT c.id
		FROM conversations c
		WHERE c.tenant_id = $1
		  AND c.status IN ('open', 'pending')
		  AND c.first_reply_at IS NULL
		  AND c.sla_breached = false
		  AND c.created_at < NOW() - ($2 * INTERVAL '1 minute')`
	return r.queryIDs(ctx, query, tenantID, minutes)
}

func (r *SLARepository) MarkBreached(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE conversations SET sla_breached = true WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to mark conversation breached")
	}
	return nil
}

func (r *SLARepository) Close(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE conversations SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND status IN ('open', 'pending')`, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to auto-close conversation")
	}
	return nil
}

func (r *SLARepository) queryIDs(ctx context.Context, query string, args ...interface{}) ([]string, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query conversations")
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}
