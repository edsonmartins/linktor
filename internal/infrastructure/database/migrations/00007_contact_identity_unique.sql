-- +goose Up
-- Prevent duplicate contacts for the same person on the same channel within a
-- tenant. `contact_identities` previously only enforced UNIQUE(contact_id,
-- channel_type, identifier), which is scoped by contact_id and therefore does
-- NOT stop two *different* contacts sharing the same (channel_type, identifier)
-- for one tenant. Under concurrent inbound webhooks from a brand-new contact,
-- two goroutines could both miss the find and both create a contact, producing
-- two rows for one person. We add a tenant-scoped partial UNIQUE index so the
-- database becomes the arbiter of the race; the loser reuses the winner's row.

-- 1. Add the tenant_id column (nullable first so the backfill can run).
ALTER TABLE contact_identities ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- 2. Backfill tenant_id from the owning contact.
UPDATE contact_identities ci
SET tenant_id = c.tenant_id
FROM contacts c
WHERE ci.contact_id = c.id AND ci.tenant_id IS NULL;

-- 3. Defensive de-dup: if a real database already holds duplicate
--    (tenant_id, channel_type, identifier) rows, the unique index creation
--    below would fail. Keep the earliest row per group and delete the rest.
DELETE FROM contact_identities ci
USING contact_identities keep
WHERE ci.tenant_id IS NOT NULL
  AND ci.tenant_id = keep.tenant_id
  AND ci.channel_type = keep.channel_type
  AND ci.identifier = keep.identifier
  AND (
        ci.created_at > keep.created_at
        OR (ci.created_at = keep.created_at AND ci.id > keep.id)
      );

-- 4. Tenant-scoped uniqueness. Partial index skips legacy rows that could not be
--    backfilled (orphaned identities with no matching contact), so it never
--    blocks on NULL tenant_id.
CREATE UNIQUE INDEX IF NOT EXISTS uq_contact_identities_tenant_channel_identifier
    ON contact_identities (tenant_id, channel_type, identifier)
    WHERE tenant_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_contact_identities_tenant_channel_identifier;
ALTER TABLE contact_identities DROP COLUMN IF EXISTS tenant_id;
