package repository

import (
	"context"
	"time"
)

// MessageIdempotencyRepository guards the logical uniqueness of an outbound
// send inside a tenant. The key is supplied by the caller
// (metadata.idempotency_key on POST /messages/send) and is scoped to
// (tenant_id, idempotency_key) — never global.
//
// The contract is reserve-then-write: the caller allocates the message id
// first, reserves the key, and only creates the message when the reservation
// was won. That ordering is what makes two concurrent requests with the same
// key collapse into one message instead of racing past a read-modify-write
// check.
type MessageIdempotencyRepository interface {
	// Reserve atomically claims (tenantID, key) for messageID. When the key was
	// already claimed it makes no change and returns the message id of the
	// original claim; on a fresh claim it returns "".
	Reserve(ctx context.Context, tenantID, key, messageID string) (existingMessageID string, err error)

	// ReclaimIfStale re-points an abandoned reservation at messageID, but only
	// when it was made before staleBefore. A reservation is abandoned when the
	// process died between reserving and creating the message: the row then names
	// a message that will never exist, and without reclaim every later retry of
	// that key would be answered forever with a message that was never sent.
	// Returns true when the caller now owns the key.
	ReclaimIfStale(ctx context.Context, tenantID, key, messageID string, staleBefore time.Time) (bool, error)

	// Release drops a reservation this caller owns — messageID scopes the delete
	// so a failing request cannot drop a reservation another in-flight request
	// is relying on. Callers use it when the send failed after reserving, so a
	// retry with the same key is not permanently answered with a message that
	// was never created.
	Release(ctx context.Context, tenantID, key, messageID string) error
}
