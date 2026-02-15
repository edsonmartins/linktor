-- ============================================
-- LINKTOR PHASE 7: WHATSAPP TEMPLATES TABLE
-- ============================================
-- This migration adds the templates table for WhatsApp message templates
-- with support for sync with Meta Cloud API

-- ============================================
-- TEMPLATES TABLE
-- ============================================
-- Stores WhatsApp message templates with Meta sync support
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    external_id VARCHAR(255),  -- Meta's template ID
    name VARCHAR(512) NOT NULL,
    language VARCHAR(10) NOT NULL,  -- e.g., en_US, pt_BR
    category VARCHAR(50) NOT NULL,  -- AUTHENTICATION, MARKETING, UTILITY
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',  -- PENDING, APPROVED, REJECTED, PAUSED, DISABLED, IN_APPEAL, PENDING_DELETION, DELETED, REINSTATED
    quality_score VARCHAR(20) DEFAULT 'UNKNOWN',  -- GREEN, YELLOW, RED, UNKNOWN
    components JSONB NOT NULL DEFAULT '[]',  -- Template components (header, body, footer, buttons)
    rejection_reason TEXT,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT uq_template_name_language UNIQUE (channel_id, name, language)
);

-- Index for external ID lookups (unique when not null)
CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_external_id
    ON templates(external_id)
    WHERE external_id IS NOT NULL;

-- Standard indexes
CREATE INDEX IF NOT EXISTS idx_templates_tenant_id ON templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_templates_channel_id ON templates(channel_id);
CREATE INDEX IF NOT EXISTS idx_templates_status ON templates(status);
CREATE INDEX IF NOT EXISTS idx_templates_category ON templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_quality_score ON templates(quality_score);
CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name);
CREATE INDEX IF NOT EXISTS idx_templates_last_synced ON templates(last_synced_at);

-- Comments
COMMENT ON TABLE templates IS 'WhatsApp message templates synced with Meta Cloud API';
COMMENT ON COLUMN templates.external_id IS 'Template ID from Meta Cloud API';
COMMENT ON COLUMN templates.name IS 'Template name (unique per channel/language)';
COMMENT ON COLUMN templates.language IS 'Template language code (e.g., en_US, pt_BR)';
COMMENT ON COLUMN templates.category IS 'Template category: AUTHENTICATION, MARKETING, UTILITY';
COMMENT ON COLUMN templates.status IS 'Template approval status from Meta';
COMMENT ON COLUMN templates.quality_score IS 'Template quality rating: GREEN, YELLOW, RED, UNKNOWN';
COMMENT ON COLUMN templates.components IS 'JSON array of template components (header, body, footer, buttons)';
COMMENT ON COLUMN templates.rejection_reason IS 'Reason for rejection if status is REJECTED';
COMMENT ON COLUMN templates.last_synced_at IS 'Last time this template was synced with Meta';

-- ============================================
-- UPDATE TRIGGER
-- ============================================
CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- TEMPLATE ANALYTICS VIEW
-- ============================================
CREATE OR REPLACE VIEW template_analytics AS
SELECT
    t.tenant_id,
    t.channel_id,
    c.name AS channel_name,
    t.category,
    t.status,
    t.quality_score,
    COUNT(*) AS template_count
FROM templates t
JOIN channels c ON t.channel_id = c.id
GROUP BY t.tenant_id, t.channel_id, c.name, t.category, t.status, t.quality_score;

COMMENT ON VIEW template_analytics IS 'Template distribution analytics by channel, category, status, and quality';
