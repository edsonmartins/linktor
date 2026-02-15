-- Migration: Create tables for WhatsApp advanced features
-- Version: 009
-- Features: Analytics cache, Payments, Calling, CTWA (Click-to-WhatsApp Ads)

-- =============================================================================
-- WhatsApp Analytics Cache
-- =============================================================================

CREATE TABLE IF NOT EXISTS whatsapp_analytics_cache (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    phone_number_id VARCHAR(64) NOT NULL,
    cache_key VARCHAR(255) NOT NULL,
    data JSONB NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_analytics_cache_channel ON whatsapp_analytics_cache(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_analytics_cache_key ON whatsapp_analytics_cache(cache_key);
CREATE INDEX IF NOT EXISTS idx_wa_analytics_cache_expires ON whatsapp_analytics_cache(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_analytics_cache_unique ON whatsapp_analytics_cache(channel_id, cache_key);

-- =============================================================================
-- Payments Tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS whatsapp_payments (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    order_id VARCHAR(64) REFERENCES orders(id) ON DELETE SET NULL,
    reference_id VARCHAR(128) NOT NULL,
    customer_phone VARCHAR(32) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    type VARCHAR(32) NOT NULL DEFAULT 'order',
    method VARCHAR(32),
    gateway_payment_id VARCHAR(128),
    gateway_order_id VARCHAR(128),
    description TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    message_id VARCHAR(128),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_payments_organization ON whatsapp_payments(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_payments_channel ON whatsapp_payments(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_payments_customer ON whatsapp_payments(customer_phone);
CREATE INDEX IF NOT EXISTS idx_wa_payments_reference ON whatsapp_payments(reference_id);
CREATE INDEX IF NOT EXISTS idx_wa_payments_status ON whatsapp_payments(status);
CREATE INDEX IF NOT EXISTS idx_wa_payments_created ON whatsapp_payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_payments_gateway ON whatsapp_payments(gateway_payment_id);

-- Payment Refunds
CREATE TABLE IF NOT EXISTS whatsapp_payment_refunds (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    payment_id VARCHAR(64) NOT NULL REFERENCES whatsapp_payments(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    reason TEXT,
    gateway_refund_id VARCHAR(128),
    failure_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_wa_refunds_payment ON whatsapp_payment_refunds(payment_id);
CREATE INDEX IF NOT EXISTS idx_wa_refunds_status ON whatsapp_payment_refunds(status);

-- Payment Gateway Config (encrypted)
CREATE TABLE IF NOT EXISTS whatsapp_payment_gateways (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    gateway_type VARCHAR(32) NOT NULL, -- razorpay, pagseguro, mercadopago, stripe
    api_key_encrypted BYTEA NOT NULL,
    api_secret_encrypted BYTEA,
    merchant_id VARCHAR(128),
    webhook_secret_encrypted BYTEA,
    sandbox_mode BOOLEAN NOT NULL DEFAULT TRUE,
    webhook_url TEXT,
    extra_config JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_gateways_organization ON whatsapp_payment_gateways(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_gateways_channel ON whatsapp_payment_gateways(channel_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_gateways_unique ON whatsapp_payment_gateways(channel_id, gateway_type);

-- =============================================================================
-- Calling Tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS whatsapp_calls (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    phone_number_id VARCHAR(64) NOT NULL,
    from_number VARCHAR(32) NOT NULL,
    to_number VARCHAR(32) NOT NULL,
    direction VARCHAR(16) NOT NULL, -- inbound, outbound
    call_type VARCHAR(16) NOT NULL DEFAULT 'voice', -- voice, video
    status VARCHAR(32) NOT NULL DEFAULT 'initiated',
    duration INTEGER NOT NULL DEFAULT 0, -- in seconds
    started_at TIMESTAMP WITH TIME ZONE,
    connected_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    recording_url TEXT,
    quality_score INTEGER,
    packet_loss DECIMAL(5,2),
    jitter DECIMAL(5,2),
    latency INTEGER,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_calls_organization ON whatsapp_calls(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_calls_channel ON whatsapp_calls(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_calls_from ON whatsapp_calls(from_number);
CREATE INDEX IF NOT EXISTS idx_wa_calls_to ON whatsapp_calls(to_number);
CREATE INDEX IF NOT EXISTS idx_wa_calls_status ON whatsapp_calls(status);
CREATE INDEX IF NOT EXISTS idx_wa_calls_direction ON whatsapp_calls(direction);
CREATE INDEX IF NOT EXISTS idx_wa_calls_created ON whatsapp_calls(created_at DESC);

-- =============================================================================
-- CTWA (Click-to-WhatsApp Ads) Tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS whatsapp_referrals (
    id VARCHAR(64) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    customer_phone VARCHAR(32) NOT NULL,
    source VARCHAR(32) NOT NULL, -- ad, post, story
    source_id VARCHAR(128),
    source_url TEXT,
    headline TEXT,
    body TEXT,
    media_type VARCHAR(32),
    media_url TEXT,
    ctwa_clid VARCHAR(255), -- Click ID for attribution
    ad_id VARCHAR(64),
    ad_name VARCHAR(255),
    adset_id VARCHAR(64),
    adset_name VARCHAR(255),
    campaign_id VARCHAR(64),
    campaign_name VARCHAR(255),
    ad_type VARCHAR(32), -- facebook, instagram
    message_id VARCHAR(128),
    conversation_id VARCHAR(64) REFERENCES conversations(id) ON DELETE SET NULL,
    free_window_expiry TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_referrals_organization ON whatsapp_referrals(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_channel ON whatsapp_referrals(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_customer ON whatsapp_referrals(customer_phone);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_campaign ON whatsapp_referrals(campaign_id);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_ad ON whatsapp_referrals(ad_id);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_source ON whatsapp_referrals(source);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_created ON whatsapp_referrals(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_referrals_free_window ON whatsapp_referrals(free_window_expiry) WHERE free_window_expiry > NOW();

-- Ad Conversions
CREATE TABLE IF NOT EXISTS whatsapp_ad_conversions (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    referral_id VARCHAR(64) NOT NULL REFERENCES whatsapp_referrals(id) ON DELETE CASCADE,
    organization_id VARCHAR(64) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    customer_phone VARCHAR(32) NOT NULL,
    conversion_type VARCHAR(64) NOT NULL, -- message, purchase, signup, etc.
    status VARCHAR(32) NOT NULL DEFAULT 'converted',
    value DECIMAL(12,2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'BRL',
    ad_id VARCHAR(64),
    campaign_id VARCHAR(64),
    attributed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_conversions_referral ON whatsapp_ad_conversions(referral_id);
CREATE INDEX IF NOT EXISTS idx_wa_conversions_organization ON whatsapp_ad_conversions(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_conversions_campaign ON whatsapp_ad_conversions(campaign_id);
CREATE INDEX IF NOT EXISTS idx_wa_conversions_ad ON whatsapp_ad_conversions(ad_id);
CREATE INDEX IF NOT EXISTS idx_wa_conversions_type ON whatsapp_ad_conversions(conversion_type);
CREATE INDEX IF NOT EXISTS idx_wa_conversions_created ON whatsapp_ad_conversions(created_at DESC);

-- Free Messaging Windows
CREATE TABLE IF NOT EXISTS whatsapp_free_windows (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    referral_id VARCHAR(64) NOT NULL REFERENCES whatsapp_referrals(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    customer_phone VARCHAR(32) NOT NULL,
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    messages_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_free_windows_channel ON whatsapp_free_windows(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_free_windows_customer ON whatsapp_free_windows(customer_phone);
CREATE INDEX IF NOT EXISTS idx_wa_free_windows_active ON whatsapp_free_windows(is_active, expires_at) WHERE is_active = TRUE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_free_windows_unique ON whatsapp_free_windows(channel_id, customer_phone) WHERE is_active = TRUE;

-- =============================================================================
-- Triggers for updated_at
-- =============================================================================

CREATE TRIGGER update_whatsapp_payments_updated_at
    BEFORE UPDATE ON whatsapp_payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_whatsapp_payment_gateways_updated_at
    BEFORE UPDATE ON whatsapp_payment_gateways
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_whatsapp_calls_updated_at
    BEFORE UPDATE ON whatsapp_calls
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_whatsapp_referrals_updated_at
    BEFORE UPDATE ON whatsapp_referrals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_whatsapp_ad_conversions_updated_at
    BEFORE UPDATE ON whatsapp_ad_conversions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
