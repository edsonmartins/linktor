-- +goose Up
-- Backfill contacts that were persisted without a real name — the legacy
-- hardcoded "Unknown" placeholder or an empty name — using the sender name the
-- channel already stored on the contact's identity metadata (WhatsApp
-- profile.name / pushName, saved as metadata->>'sender_name'; an explicit
-- metadata->>'name' takes precedence, mirroring inboundName() in
-- receive_message.go). Contacts with a real, human-curated name are left alone.
--
-- Going forward, new inbound messages backfill the name live; this one-shot
-- migration repairs the rows created before that fix.
UPDATE contacts c
SET name = sub.resolved_name,
    updated_at = NOW()
FROM (
    SELECT DISTINCT ON (ci.contact_id)
        ci.contact_id,
        COALESCE(NULLIF(ci.metadata->>'name', ''), NULLIF(ci.metadata->>'sender_name', '')) AS resolved_name
    FROM contact_identities ci
    WHERE COALESCE(NULLIF(ci.metadata->>'name', ''), NULLIF(ci.metadata->>'sender_name', '')) IS NOT NULL
    ORDER BY ci.contact_id, ci.created_at DESC
) sub
WHERE c.id = sub.contact_id
  AND sub.resolved_name IS NOT NULL
  AND (c.name IS NULL OR c.name = '' OR c.name = 'Unknown');

-- +goose Down
-- Irreversible data backfill: the previous empty/"Unknown" placeholder carried
-- no information, so there is nothing to restore and no schema change to revert.
SELECT 1;
