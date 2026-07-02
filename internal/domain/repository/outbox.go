package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// OutboxRepository reads and updates the transactional outbox. The enqueue side
// is transactional with the aggregate write and lives on the owning repository
// (e.g. MessageRepository.CreateWithOutboxEvent), so it is not part of this
// interface — this covers only the relay's read/mark path.
type OutboxRepository interface {
	// FetchUnpublished returns up to limit events that have not been published,
	// oldest first.
	FetchUnpublished(ctx context.Context, limit int) ([]*entity.OutboxEvent, error)

	// MarkPublished stamps an event as published so the relay stops retrying it.
	MarkPublished(ctx context.Context, id string) error

	// RecordFailure increments the attempt counter and stores the last error for
	// an event that failed to publish, so it is retried on the next relay pass.
	RecordFailure(ctx context.Context, id, lastErr string) error
}
