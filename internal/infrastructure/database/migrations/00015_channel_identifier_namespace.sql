-- +goose Up
-- Persist two orphan Channel fields (G-005): Identifier (channel address, e.g.
-- phone/page id, previously lost on every restart) and
-- MessageTemplateNamespace (WABA-level namespace fetched lazily from Meta,
-- previously re-fetched forever because it never survived a reload).
-- Caught by the reflexive round-trip test (roundtrip_reflect_test.go).
ALTER TABLE channels ADD COLUMN IF NOT EXISTS identifier VARCHAR(255);
ALTER TABLE channels ADD COLUMN IF NOT EXISTS message_template_namespace VARCHAR(255);

-- +goose Down
ALTER TABLE channels DROP COLUMN IF EXISTS message_template_namespace;
ALTER TABLE channels DROP COLUMN IF EXISTS identifier;
