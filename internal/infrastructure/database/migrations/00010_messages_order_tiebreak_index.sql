-- +goose Up
-- MessageRepository.FindByConversation orders by (created_at, id) so tied
-- created_at values (inbound messages share a second-precision provider
-- timestamp) get a stable, deterministic order instead of shuffling on every
-- realtime refetch. Extend the covering index with id DESC so that ordering
-- stays index-covered and the tiebreaker adds no sort cost.
DROP INDEX IF EXISTS idx_messages_conversation_created;
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created_id
    ON messages (conversation_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_conversation_created_id;
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created
    ON messages (conversation_id, created_at DESC);
