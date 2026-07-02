package entity

import "time"

// OutboxEvent is a domain event queued for durable publication via the
// transactional-outbox pattern. It is written to the outbox table in the SAME
// database transaction as the aggregate change (e.g. an inbound message), so the
// event exists if and only if the change committed — no dual-write inconsistency.
// A relay later publishes it to NATS and stamps PublishedAt. Publication carries
// IdempotencyKey as the JetStream message id, so a relay retry after a partial
// failure (published but not yet marked) is deduplicated broker-side.
type OutboxEvent struct {
	ID             string
	EventType      string // NATS event type (e.g. "message.received")
	TenantID       string
	AggregateType  string // e.g. "message"
	AggregateID    string // e.g. the message id
	Payload        []byte // JSON payload delivered verbatim as the event payload
	IdempotencyKey string
	Attempts       int
	LastError      string
	CreatedAt      time.Time
	PublishedAt    *time.Time
}
