-- Migration: 010_whatsapp_coexistence.sql
-- Description: Add WhatsApp Coexistence support for simultaneous Business App + Cloud API usage
-- Date: 2026-02-15

-- =====================================================
-- PHASE 1: Channel Coexistence Fields
-- =====================================================

-- Add coexistence tracking fields to channels table
ALTER TABLE channels ADD COLUMN IF NOT EXISTS is_coexistence BOOLEAN DEFAULT FALSE;
ALTER TABLE channels ADD COLUMN IF NOT EXISTS waba_id VARCHAR(64);
ALTER TABLE channels ADD COLUMN IF NOT EXISTS last_echo_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE channels ADD COLUMN IF NOT EXISTS coexistence_status VARCHAR(32) DEFAULT 'inactive';
-- coexistence_status values: 'inactive', 'pending', 'active', 'warning', 'disconnected'

-- Index for efficient coexistence channel queries
CREATE INDEX IF NOT EXISTS idx_channels_coexistence ON channels(is_coexistence) WHERE is_coexistence = TRUE;
CREATE INDEX IF NOT EXISTS idx_channels_last_echo ON channels(last_echo_at) WHERE is_coexistence = TRUE;
CREATE INDEX IF NOT EXISTS idx_channels_coex_status ON channels(coexistence_status) WHERE is_coexistence = TRUE;

-- =====================================================
-- PHASE 2: Message Source Tracking
-- =====================================================

-- Add source tracking to messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS source VARCHAR(32) DEFAULT 'api';
-- source values: 'api', 'business_app', 'imported'

ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_imported BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS imported_at TIMESTAMP WITH TIME ZONE;

-- Indexes for message source queries
CREATE INDEX IF NOT EXISTS idx_messages_source ON messages(source);
CREATE INDEX IF NOT EXISTS idx_messages_imported ON messages(is_imported) WHERE is_imported = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_business_app ON messages(channel_id, source) WHERE source = 'business_app';

-- =====================================================
-- PHASE 3: Chat History Import Tracking
-- =====================================================

CREATE TABLE IF NOT EXISTS whatsapp_history_imports (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tenant_id VARCHAR(64) NOT NULL,

    -- Import status
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    -- status values: 'pending', 'in_progress', 'completed', 'failed', 'cancelled'

    -- Progress tracking
    total_conversations INTEGER DEFAULT 0,
    imported_conversations INTEGER DEFAULT 0,
    total_messages INTEGER DEFAULT 0,
    imported_messages INTEGER DEFAULT 0,
    total_contacts INTEGER DEFAULT 0,
    imported_contacts INTEGER DEFAULT 0,

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Error handling
    error_message TEXT,
    error_details JSONB,

    -- Import configuration
    import_since TIMESTAMP WITH TIME ZONE, -- How far back to import (max 6 months)

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for history imports
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_channel ON whatsapp_history_imports(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_tenant ON whatsapp_history_imports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_status ON whatsapp_history_imports(status);
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_created ON whatsapp_history_imports(created_at DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_whatsapp_history_imports_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_whatsapp_history_imports_updated_at ON whatsapp_history_imports;
CREATE TRIGGER trigger_whatsapp_history_imports_updated_at
    BEFORE UPDATE ON whatsapp_history_imports
    FOR EACH ROW
    EXECUTE FUNCTION update_whatsapp_history_imports_updated_at();

-- =====================================================
-- PHASE 4: Coexistence Activity Log
-- =====================================================

CREATE TABLE IF NOT EXISTS whatsapp_coexistence_activity (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tenant_id VARCHAR(64) NOT NULL,

    -- Activity type
    activity_type VARCHAR(32) NOT NULL,
    -- activity_type values: 'echo_received', 'status_changed', 'warning_sent', 'disconnected'

    -- Details
    details JSONB,

    -- Timing
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for activity queries
CREATE INDEX IF NOT EXISTS idx_wa_coex_activity_channel ON whatsapp_coexistence_activity(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_coex_activity_type ON whatsapp_coexistence_activity(activity_type);
CREATE INDEX IF NOT EXISTS idx_wa_coex_activity_created ON whatsapp_coexistence_activity(created_at DESC);

-- =====================================================
-- PHASE 5: Coexistence Notifications
-- =====================================================

CREATE TABLE IF NOT EXISTS whatsapp_coexistence_notifications (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tenant_id VARCHAR(64) NOT NULL,

    -- Notification type
    notification_type VARCHAR(32) NOT NULL,
    -- notification_type values: 'warning_10_days', 'warning_12_days', 'disconnected'

    -- Delivery status
    sent_at TIMESTAMP WITH TIME ZONE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,

    -- Notification content
    title VARCHAR(255),
    message TEXT,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for notification queries
CREATE INDEX IF NOT EXISTS idx_wa_coex_notif_channel ON whatsapp_coexistence_notifications(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_coex_notif_type ON whatsapp_coexistence_notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_wa_coex_notif_pending ON whatsapp_coexistence_notifications(channel_id)
    WHERE sent_at IS NULL;

-- =====================================================
-- COMMENTS
-- =====================================================

COMMENT ON COLUMN channels.is_coexistence IS 'Whether this channel uses WhatsApp Coexistence (Business App + Cloud API)';
COMMENT ON COLUMN channels.waba_id IS 'WhatsApp Business Account ID for this channel';
COMMENT ON COLUMN channels.last_echo_at IS 'Last time a message echo was received from Business App';
COMMENT ON COLUMN channels.coexistence_status IS 'Current coexistence status: inactive, pending, active, warning, disconnected';

COMMENT ON COLUMN messages.source IS 'Source of the message: api, business_app, or imported';
COMMENT ON COLUMN messages.is_imported IS 'Whether this message was imported from chat history';
COMMENT ON COLUMN messages.imported_at IS 'When this message was imported';

COMMENT ON TABLE whatsapp_history_imports IS 'Tracks chat history import jobs for coexistence channels';
COMMENT ON TABLE whatsapp_coexistence_activity IS 'Activity log for coexistence events';
COMMENT ON TABLE whatsapp_coexistence_notifications IS 'Notifications sent for coexistence warnings';
