-- +goose Up
-- conversations.tags: written by ConversationService.Update (conversation.Tags)
-- and read back by ConversationRepository. Without this column, tagging a
-- conversation silently dropped the tags at the persistence layer.
-- conversations.metadata already exists from 00002_align_repo_schema.sql.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- +goose Down
ALTER TABLE conversations DROP COLUMN IF EXISTS tags;
