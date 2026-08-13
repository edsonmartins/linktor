package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/pkg/errors"
)

// MessageIdempotencyRepository implements repository.MessageIdempotencyRepository
// on top of the message_idempotency_keys table (see migration 00017).
type MessageIdempotencyRepository struct {
	db *PostgresDB
}

// NewMessageIdempotencyRepository creates a PostgreSQL-backed idempotency store.
func NewMessageIdempotencyRepository(db *PostgresDB) *MessageIdempotencyRepository {
	return &MessageIdempotencyRepository{db: db}
}

// reserveAttempts bounds the insert/read retry below. More than a couple of
// laps means a pathological release-under-contention loop, not a real race.
const reserveAttempts = 3

// Reserve claims (tenantID, key) for messageID. The claim is the INSERT itself:
// two concurrent requests with the same key contend on the primary key, so
// exactly one wins and the loser reads the winner's message id back.
func (r *MessageIdempotencyRepository) Reserve(ctx context.Context, tenantID, key, messageID string) (string, error) {
	for attempt := 0; attempt < reserveAttempts; attempt++ {
		tag, err := r.db.Pool.Exec(ctx, `
			INSERT INTO message_idempotency_keys (tenant_id, idempotency_key, message_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
		`, tenantID, key, messageID)
		if err != nil {
			return "", errors.Wrap(err, errors.ErrCodeInternal, "failed to reserve idempotency key")
		}
		if tag.RowsAffected() == 1 {
			return "", nil
		}

		var existing string
		err = r.db.Pool.QueryRow(ctx, `
			SELECT message_id FROM message_idempotency_keys
			WHERE tenant_id = $1 AND idempotency_key = $2
		`, tenantID, key).Scan(&existing)
		if err == nil {
			return existing, nil
		}
		if err != pgx.ErrNoRows {
			return "", errors.Wrap(err, errors.ErrCodeInternal, "failed to read idempotency key")
		}
		// The winner released between our INSERT and this read, so the key is
		// free again. Retrying the INSERT is the only correct move: reporting
		// "fresh claim" here would hand back a claim we never made, and the next
		// duplicate would not be collapsed — sending the message twice, the exact
		// thing the key promises to prevent.
	}
	return "", errors.New(errors.ErrCodeConflict,
		"idempotency key is contended; retry the request")
}

// ReclaimIfStale re-points an abandoned reservation at messageID.
func (r *MessageIdempotencyRepository) ReclaimIfStale(
	ctx context.Context, tenantID, key, messageID string, staleBefore time.Time,
) (bool, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		UPDATE message_idempotency_keys
		SET message_id = $3, created_at = NOW()
		WHERE tenant_id = $1 AND idempotency_key = $2 AND created_at < $4
	`, tenantID, key, messageID, staleBefore)
	if err != nil {
		return false, errors.Wrap(err, errors.ErrCodeInternal, "failed to reclaim idempotency key")
	}
	return tag.RowsAffected() == 1, nil
}

// Release drops the reservation so a later retry with the same key can proceed.
// Scoped to messageID: without it, a request that failed after its own
// reservation was reclaimed would delete the reservation now owned by someone
// else.
func (r *MessageIdempotencyRepository) Release(ctx context.Context, tenantID, key, messageID string) error {
	_, err := r.db.Pool.Exec(ctx, `
		DELETE FROM message_idempotency_keys
		WHERE tenant_id = $1 AND idempotency_key = $2 AND message_id = $3
	`, tenantID, key, messageID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to release idempotency key")
	}
	return nil
}
