-- +goose Up
-- Persist emoji reactions on messages. Previously SendReaction only published a
-- message.reaction NATS event and never stored the reaction, so the admin UI
-- lost reactions on refetch. Store them as a JSONB array of {user_id, emoji,
-- timestamp} objects, defaulting to an empty array so existing rows are valid.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reactions JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE messages DROP COLUMN IF EXISTS reactions;
