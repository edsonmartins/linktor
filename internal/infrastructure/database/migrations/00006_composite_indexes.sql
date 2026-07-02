-- +goose Up
-- Composite indexes matching the hot read paths. The baseline shipped only
-- single-column indexes, forcing the planner to combine/sort for the two most
-- frequent queries: paginated message history and the filtered conversation
-- list. These are additive (IF NOT EXISTS) and supersede the single-column
-- indexes for these specific queries.

-- MessageRepository.FindByConversation: WHERE conversation_id = $1 ORDER BY created_at.
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created
    ON messages (conversation_id, created_at DESC);

-- ConversationRepository list/count filtered by tenant + status.
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_status
    ON conversations (tenant_id, status);

-- ConversationRepository default tenant-scoped list ordered by created_at DESC.
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_created
    ON conversations (tenant_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_conversations_tenant_created;
DROP INDEX IF EXISTS idx_conversations_tenant_status;
DROP INDEX IF EXISTS idx_messages_conversation_created;
