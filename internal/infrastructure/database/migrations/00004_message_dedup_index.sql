-- +goose Up
-- Atomic inbound message de-duplication. Redelivered provider messages (same
-- external_id arriving again on the same conversation, e.g. a WhatsApp/Telegram
-- webhook retry) must not create duplicate rows. A partial unique index scopes
-- uniqueness to inbound messages that carry a provider id — outbound rows insert
-- external_id as NULL (it is assigned only after the send ack) and are excluded,
-- so this never blocks legitimate outbound inserts. The repository INSERT uses
-- ON CONFLICT with this exact predicate to drop the duplicate atomically.
CREATE UNIQUE INDEX IF NOT EXISTS uq_messages_conversation_external_id
    ON messages (conversation_id, external_id)
    WHERE external_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_messages_conversation_external_id;
