-- ============================================
-- LINKTOR PHASE 6: OBSERVABILITY TABLES
-- ============================================
-- This migration adds tables for system observability
-- including message logs, system events, and audit trails

-- ============================================
-- MESSAGE LOGS TABLE
-- ============================================
-- Stores logs from channels, queue processing, and system events
CREATE TABLE IF NOT EXISTS message_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    level VARCHAR(10) NOT NULL DEFAULT 'info', -- info, warn, error
    source VARCHAR(50) NOT NULL, -- channel, queue, system, webhook
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_message_logs_tenant_created ON message_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_logs_channel ON message_logs(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_logs_level ON message_logs(tenant_id, level, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_logs_source ON message_logs(tenant_id, source, created_at DESC);

COMMENT ON TABLE message_logs IS 'System observability logs for channels, queues, and events';
COMMENT ON COLUMN message_logs.level IS 'Log level: info, warn, error';
COMMENT ON COLUMN message_logs.source IS 'Log source: channel, queue, system, webhook';
COMMENT ON COLUMN message_logs.metadata IS 'Additional structured data (error codes, stack traces, etc.)';

-- ============================================
-- SYSTEM METRICS TABLE
-- ============================================
-- Stores aggregated system metrics for statistics
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(20, 4) NOT NULL,
    dimensions JSONB DEFAULT '{}', -- channel_id, bot_id, etc.
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_tenant_name ON system_metrics(tenant_id, metric_name, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_metrics_recorded ON system_metrics(recorded_at DESC);

COMMENT ON TABLE system_metrics IS 'Aggregated system metrics for observability dashboards';
COMMENT ON COLUMN system_metrics.metric_name IS 'Metric name: messages_per_hour, avg_response_time_ms, etc.';
COMMENT ON COLUMN system_metrics.dimensions IS 'Metric dimensions for filtering (channel_id, bot_id, etc.)';

-- ============================================
-- LOG RETENTION FUNCTION
-- ============================================
-- Function to clean up old logs (for scheduled job)
CREATE OR REPLACE FUNCTION cleanup_old_message_logs(retention_days INT DEFAULT 30)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM message_logs
    WHERE created_at < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_old_message_logs IS 'Cleans up message logs older than the specified retention period';

-- ============================================
-- LOG SUMMARY VIEW
-- ============================================
-- View for quick log summaries by level
CREATE OR REPLACE VIEW log_summary_by_hour AS
SELECT
    tenant_id,
    channel_id,
    level,
    source,
    DATE_TRUNC('hour', created_at) AS hour,
    COUNT(*) AS log_count
FROM message_logs
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY tenant_id, channel_id, level, source, DATE_TRUNC('hour', created_at)
ORDER BY hour DESC;

COMMENT ON VIEW log_summary_by_hour IS 'Hourly summary of logs by level and source for the last 24 hours';

-- ============================================
-- SYSTEM HEALTH VIEW
-- ============================================
-- View for system health overview
CREATE OR REPLACE VIEW system_health AS
SELECT
    ml.tenant_id,
    COUNT(*) FILTER (WHERE ml.level = 'error' AND ml.created_at >= NOW() - INTERVAL '1 hour') AS errors_last_hour,
    COUNT(*) FILTER (WHERE ml.level = 'warn' AND ml.created_at >= NOW() - INTERVAL '1 hour') AS warnings_last_hour,
    COUNT(*) FILTER (WHERE ml.created_at >= NOW() - INTERVAL '1 hour') AS total_logs_last_hour,
    (SELECT COUNT(*) FROM channels c WHERE c.tenant_id = ml.tenant_id AND c.status = 'connected') AS connected_channels,
    (SELECT COUNT(*) FROM channels c WHERE c.tenant_id = ml.tenant_id) AS total_channels,
    (SELECT COUNT(*) FROM conversations cv WHERE cv.tenant_id = ml.tenant_id AND cv.status = 'open') AS open_conversations
FROM message_logs ml
WHERE ml.created_at >= NOW() - INTERVAL '24 hours'
GROUP BY ml.tenant_id;

COMMENT ON VIEW system_health IS 'System health overview per tenant';
