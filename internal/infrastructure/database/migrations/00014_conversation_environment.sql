-- +goose Up
-- Denormalized channel environment on conversations (INV-018). A conversation
-- is born bound to one channel and the channel's environment is immutable, so
-- this never drifts. Makes sandbox purge and analytics-export blocking
-- indexable without joining channels. Preexisting rows are production.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS environment VARCHAR(20) NOT NULL DEFAULT 'production';

-- Partial index: sandbox rows are the minority and the only ones purge/export
-- filters ever target.
CREATE INDEX IF NOT EXISTS idx_conversations_sandbox
    ON conversations(tenant_id)
    WHERE environment = 'sandbox';

-- +goose Down
DROP INDEX IF EXISTS idx_conversations_sandbox;
ALTER TABLE conversations DROP COLUMN IF EXISTS environment;
