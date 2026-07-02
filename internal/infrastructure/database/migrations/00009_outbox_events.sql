-- +goose Up
-- Transactional outbox for durable event publication. An event row is written in
-- the same transaction as its aggregate change (see MessageRepository
-- .CreateWithOutboxEvent), and a relay publishes unpublished rows to NATS and
-- stamps published_at. This removes the inbound dual-write window: the
-- "message.received" event is published iff the message row committed.
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    tenant_id UUID,
    aggregate_type VARCHAR(50),
    aggregate_id UUID,
    payload JSONB NOT NULL DEFAULT '{}',
    idempotency_key VARCHAR(255),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE
);

-- The relay only ever scans unpublished rows, oldest first.
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_unpublished;
DROP TABLE IF EXISTS outbox_events;
