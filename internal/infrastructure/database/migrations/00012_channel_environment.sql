-- +goose Up
-- Channel environment (production | sandbox). Policy attribute consulted by the
-- sandbox delivery guard, marking propagation and future retention/purge jobs;
-- it never participates in send routing. Immutable after creation (enforced at
-- the domain edge, ChannelService). Preexisting channels are production.
ALTER TABLE channels ADD COLUMN IF NOT EXISTS environment VARCHAR(20) NOT NULL DEFAULT 'production';

CREATE INDEX IF NOT EXISTS idx_channels_environment ON channels(environment);

-- +goose Down
DROP INDEX IF EXISTS idx_channels_environment;
ALTER TABLE channels DROP COLUMN IF EXISTS environment;
