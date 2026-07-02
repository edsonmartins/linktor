-- +goose Up
-- WhatsApp Flows created on Meta return a Meta-side flow ID that every
-- subsequent lifecycle call (update/publish/delete/preview) must address by.
-- The baseline stored it only inside the free-text description, so those calls
-- were sending our local UUID to Meta and failing. Persist it as a first-class,
-- queryable column.
ALTER TABLE flows ADD COLUMN IF NOT EXISTS meta_flow_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_flows_meta_flow_id ON flows(meta_flow_id) WHERE meta_flow_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_flows_meta_flow_id;
ALTER TABLE flows DROP COLUMN IF EXISTS meta_flow_id;
