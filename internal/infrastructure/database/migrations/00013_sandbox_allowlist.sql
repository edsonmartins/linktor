-- +goose Up
-- Sandbox recipient allowlist (INV-017): a sandbox channel may only deliver to
-- recipients listed here for its tenant. channel_id NULL means the entry is
-- valid for every sandbox channel of the tenant; non-NULL narrows it to one
-- channel. recipient is stored normalized in E.164 ("+" + digits).
CREATE TABLE IF NOT EXISTS sandbox_allowlist_entries (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    recipient VARCHAR(20) NOT NULL,
    note VARCHAR(255),
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_sandbox_allowlist_tenant_recipient
    ON sandbox_allowlist_entries(tenant_id, recipient)
    WHERE channel_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_sandbox_allowlist_channel_recipient
    ON sandbox_allowlist_entries(tenant_id, channel_id, recipient)
    WHERE channel_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sandbox_allowlist_tenant
    ON sandbox_allowlist_entries(tenant_id, recipient);

-- +goose Down
DROP TABLE IF EXISTS sandbox_allowlist_entries;
