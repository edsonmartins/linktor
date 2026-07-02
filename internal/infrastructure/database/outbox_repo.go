package database

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// OutboxRepository implements repository.OutboxRepository with PostgreSQL.
type OutboxRepository struct {
	db *PostgresDB
}

// NewOutboxRepository creates a new PostgreSQL outbox repository.
func NewOutboxRepository(db *PostgresDB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// FetchUnpublished returns up to limit unpublished events, oldest first.
func (r *OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, event_type, tenant_id, aggregate_type, aggregate_id,
		       payload, idempotency_key, attempts, last_error, created_at, published_at
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to fetch outbox events")
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		var (
			e             entity.OutboxEvent
			tenantID      *string
			aggregateType *string
			aggregateID   *string
			idempotency   *string
			lastError     *string
		)
		if err := rows.Scan(&e.ID, &e.EventType, &tenantID, &aggregateType, &aggregateID,
			&e.Payload, &idempotency, &e.Attempts, &lastError, &e.CreatedAt, &e.PublishedAt); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan outbox event")
		}
		if tenantID != nil {
			e.TenantID = *tenantID
		}
		if aggregateType != nil {
			e.AggregateType = *aggregateType
		}
		if aggregateID != nil {
			e.AggregateID = *aggregateID
		}
		if idempotency != nil {
			e.IdempotencyKey = *idempotency
		}
		if lastError != nil {
			e.LastError = *lastError
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed iterating outbox events")
	}
	return events, nil
}

// MarkPublished stamps an event as published.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE outbox_events SET published_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to mark outbox event published")
	}
	return nil
}

// RecordFailure increments the attempt counter and stores the last error.
func (r *OutboxRepository) RecordFailure(ctx context.Context, id, lastErr string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE outbox_events SET attempts = attempts + 1, last_error = $2 WHERE id = $1`, id, lastErr)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to record outbox failure")
	}
	return nil
}
