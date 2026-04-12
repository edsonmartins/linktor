package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/msgfy/linktor/internal/infrastructure/config"
)

// PostgresDB wraps a PostgreSQL connection pool
type PostgresDB struct {
	Pool *pgxpool.Pool
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = time.Duration(cfg.MaxLifetime) * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *PostgresDB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Ping checks the database connection
func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// RunMigrations runs database migrations
func (db *PostgresDB) RunMigrations(ctx context.Context) error {
	migrations := []string{
		createExtensions,
		createUpdatedAtFunction,
		createTenantsTable,
		createUsersTable,
		createAPIKeysTable,
		createChannelsTable,
		createContactsTable,
		createContactIdentitiesTable,
		createConversationsTable,
		createMessagesTable,
		createMessageAttachmentsTable,
		createBotsTable,
		createBotChannelsTable,
		createFlowsTable,
		createConversationContextsTable,
		createKnowledgeBasesTable,
		createKnowledgeItemsTable,
		alignConversationAnalyticsSchema,
		alignKnowledgeItemsSchema,
		alignChannelCoexistenceStatusSchema,
		createAIResponsesTable,
		createIndexes,
		createNewIndexes,
		addLimitsColumn,
		addBotsChannelsColumn,
		refactorChannelStatus,
		addMessageCoexistenceColumns,
		addChannelCoexistenceColumns,
		createTemplatesTable,
		createMessageLogsTable,
		createSystemMetricsTable,
		createWhatsAppPaymentsTables,
		createWhatsAppHistoryImportsTable,
		createWhatsAppCoexistenceTables,
	}

	for _, migration := range migrations {
		if _, err := db.Pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

const createExtensions = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
`

const createUpdatedAtFunction = `
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
`

const addLimitsColumn = `
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tenants' AND column_name='limits') THEN
        ALTER TABLE tenants ADD COLUMN limits JSONB DEFAULT '{}';
    END IF;
END $$;
`

const addBotsChannelsColumn = `
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='bots' AND column_name='channels') THEN
        ALTER TABLE bots ADD COLUMN channels UUID[] DEFAULT '{}';
    END IF;
END $$;
`

const addMessageCoexistenceColumns = `
ALTER TABLE messages ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'api';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_imported BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS imported_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_messages_source ON messages(source);
CREATE INDEX IF NOT EXISTS idx_messages_is_imported ON messages(is_imported);
CREATE INDEX IF NOT EXISTS idx_messages_imported_at ON messages(imported_at);
`

const addChannelCoexistenceColumns = `
ALTER TABLE channels ADD COLUMN IF NOT EXISTS is_coexistence BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE channels ADD COLUMN IF NOT EXISTS waba_id VARCHAR(64);
ALTER TABLE channels ADD COLUMN IF NOT EXISTS last_echo_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE channels ADD COLUMN IF NOT EXISTS coexistence_status VARCHAR(32) NOT NULL DEFAULT 'inactive';

CREATE INDEX IF NOT EXISTS idx_channels_is_coexistence ON channels(is_coexistence);
CREATE INDEX IF NOT EXISTS idx_channels_waba_id ON channels(waba_id);
CREATE INDEX IF NOT EXISTS idx_channels_coexistence_status ON channels(coexistence_status);
`

// Migration to separate enabled (boolean) from connection_status
const refactorChannelStatus = `
DO $$
BEGIN
    -- Add enabled column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='channels' AND column_name='enabled') THEN
        ALTER TABLE channels ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
    END IF;

    -- Add connection_status column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='channels' AND column_name='connection_status') THEN
        ALTER TABLE channels ADD COLUMN connection_status VARCHAR(50) NOT NULL DEFAULT 'disconnected';
    END IF;

    -- Migrate data from status to new columns if status column exists and connection_status is empty
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='channels' AND column_name='status') THEN
        -- Set enabled based on old status (active = enabled, inactive = disabled)
        UPDATE channels SET enabled = (status IN ('active', 'connecting', 'connected')) WHERE enabled = true;

        -- Map old status values to new connection_status
        UPDATE channels SET connection_status = CASE
            WHEN status = 'active' THEN 'connected'
            WHEN status = 'inactive' THEN 'disconnected'
            WHEN status = 'connecting' THEN 'connecting'
            WHEN status = 'disconnected' THEN 'disconnected'
            WHEN status = 'error' THEN 'error'
            ELSE 'disconnected'
        END WHERE connection_status = 'disconnected';
    END IF;

    -- Create index on connection_status
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_channels_connection_status') THEN
        CREATE INDEX idx_channels_connection_status ON channels(connection_status);
    END IF;

    -- Create index on enabled
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_channels_enabled') THEN
        CREATE INDEX idx_channels_enabled ON channels(enabled);
    END IF;
END $$;
`

const createTenantsTable = `
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    limits JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'agent',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    avatar_url VARCHAR(500),
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);
`

const createAPIKeysTable = `
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
`

const createChannelsTable = `
CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    credentials JSONB DEFAULT '{}',
    config JSONB DEFAULT '{}',
    webhook_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createContactsTable = `
CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    avatar_url VARCHAR(500),
    custom_fields JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createContactIdentitiesTable = `
CREATE TABLE IF NOT EXISTS contact_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel_type VARCHAR(50) NOT NULL,
    identifier VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(contact_id, channel_type, identifier)
);
`

const createConversationsTable = `
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    priority VARCHAR(50) NOT NULL DEFAULT 'normal',
    subject VARCHAR(500),
    unread_count INT DEFAULT 0,
    first_reply_at TIMESTAMP WITH TIME ZONE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createMessagesTable = `
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_type VARCHAR(50) NOT NULL,
    sender_id UUID,
    content_type VARCHAR(50) NOT NULL DEFAULT 'text',
    content TEXT,
    metadata JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    external_id VARCHAR(255),
    error_message TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createMessageAttachmentsTable = `
CREATE TABLE IF NOT EXISTS message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    filename VARCHAR(255),
    mime_type VARCHAR(100),
    size_bytes BIGINT,
    url VARCHAR(500) NOT NULL,
    thumbnail_url VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createBotsTable = `
CREATE TABLE IF NOT EXISTS bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'customer_service',
    provider VARCHAR(50) NOT NULL DEFAULT 'openai',
    model VARCHAR(100) NOT NULL DEFAULT 'gpt-4',
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    config JSONB DEFAULT '{}',
    channels UUID[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createBotChannelsTable = `
CREATE TABLE IF NOT EXISTS bot_channels (
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (bot_id, channel_id)
);
`

const createFlowsTable = `
CREATE TABLE IF NOT EXISTS flows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger VARCHAR(50) NOT NULL DEFAULT 'keyword',
    trigger_value VARCHAR(255),
    start_node_id VARCHAR(100),
    nodes JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT false,
    priority INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createConversationContextsTable = `
CREATE TABLE IF NOT EXISTS conversation_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    context_data JSONB DEFAULT '{}',
    flow_state JSONB DEFAULT '{}',
    intent VARCHAR(100),
    sentiment VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(conversation_id)
);
`

const createKnowledgeBasesTable = `
CREATE TABLE IF NOT EXISTS knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'document',
    config JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    item_count INT DEFAULT 0,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createKnowledgeItemsTable = `
CREATE TABLE IF NOT EXISTS knowledge_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    keywords TEXT[] DEFAULT '{}',
    embedding TEXT,
    source VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const alignConversationAnalyticsSchema = `
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_conversations_escalated_at ON conversations(escalated_at);
`

const alignKnowledgeItemsSchema = `
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='question') THEN
        ALTER TABLE knowledge_items ADD COLUMN question TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='answer') THEN
        ALTER TABLE knowledge_items ADD COLUMN answer TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='keywords') THEN
        ALTER TABLE knowledge_items ADD COLUMN keywords TEXT[] DEFAULT '{}';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='embedding') THEN
        ALTER TABLE knowledge_items ADD COLUMN embedding TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='source') THEN
        ALTER TABLE knowledge_items ADD COLUMN source VARCHAR(255);
    END IF;

	IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='title') THEN
		UPDATE knowledge_items SET question = COALESCE(question, title) WHERE question IS NULL;
		ALTER TABLE knowledge_items ALTER COLUMN title DROP NOT NULL;
	END IF;

	IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='knowledge_items' AND column_name='content') THEN
		UPDATE knowledge_items SET answer = COALESCE(answer, content) WHERE answer IS NULL;
		ALTER TABLE knowledge_items ALTER COLUMN content DROP NOT NULL;
	END IF;

    UPDATE knowledge_items SET question = '' WHERE question IS NULL;
    UPDATE knowledge_items SET answer = '' WHERE answer IS NULL;

    ALTER TABLE knowledge_items ALTER COLUMN question SET NOT NULL;
    ALTER TABLE knowledge_items ALTER COLUMN answer SET NOT NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_knowledge_items_keywords ON knowledge_items USING GIN(keywords);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_source ON knowledge_items(source);
`

const alignChannelCoexistenceStatusSchema = `
UPDATE channels
SET coexistence_status = 'inactive'
WHERE coexistence_status IS NULL OR coexistence_status = '';
`

const createAIResponsesTable = `
CREATE TABLE IF NOT EXISTS ai_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    prompt TEXT,
    response TEXT,
    model VARCHAR(100),
    tokens_used INT,
    latency_ms INT,
    confidence FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
`

const createIndexes = `
-- Tenant indexes
CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- Channel indexes
CREATE INDEX IF NOT EXISTS idx_channels_tenant_id ON channels(tenant_id);
CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type);
CREATE INDEX IF NOT EXISTS idx_channels_status ON channels(status);

-- Contact indexes
CREATE INDEX IF NOT EXISTS idx_contacts_tenant_id ON contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_phone ON contacts(phone);

-- Contact identity indexes
CREATE INDEX IF NOT EXISTS idx_contact_identities_contact_id ON contact_identities(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_identities_channel_type ON contact_identities(channel_type);
CREATE INDEX IF NOT EXISTS idx_contact_identities_identifier ON contact_identities(identifier);

-- Conversation indexes
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_id ON conversations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversations_channel_id ON conversations(channel_id);
CREATE INDEX IF NOT EXISTS idx_conversations_contact_id ON conversations(contact_id);
CREATE INDEX IF NOT EXISTS idx_conversations_assignee_id ON conversations(assignee_id);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(status);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC);

-- Message indexes
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_type ON messages(sender_type);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_external_id ON messages(external_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);

-- Attachment indexes
CREATE INDEX IF NOT EXISTS idx_message_attachments_message_id ON message_attachments(message_id);
`

const createNewIndexes = `
-- Bot indexes
CREATE INDEX IF NOT EXISTS idx_bots_tenant_id ON bots(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bots_status ON bots(status);
CREATE INDEX IF NOT EXISTS idx_bots_provider ON bots(provider);

-- Bot channels indexes
CREATE INDEX IF NOT EXISTS idx_bot_channels_bot_id ON bot_channels(bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_channels_channel_id ON bot_channels(channel_id);

-- Flow indexes
CREATE INDEX IF NOT EXISTS idx_flows_tenant_id ON flows(tenant_id);
CREATE INDEX IF NOT EXISTS idx_flows_is_active ON flows(is_active);
CREATE INDEX IF NOT EXISTS idx_flows_trigger ON flows(trigger);
CREATE INDEX IF NOT EXISTS idx_flows_bot_id ON flows(bot_id);

-- Conversation context indexes
CREATE INDEX IF NOT EXISTS idx_conversation_contexts_conversation_id ON conversation_contexts(conversation_id);

-- Knowledge base indexes
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_tenant_id ON knowledge_bases(tenant_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_status ON knowledge_bases(status);

-- Knowledge item indexes
CREATE INDEX IF NOT EXISTS idx_knowledge_items_knowledge_base_id ON knowledge_items(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_source ON knowledge_items(source);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_keywords ON knowledge_items USING GIN(keywords);

-- AI response indexes
CREATE INDEX IF NOT EXISTS idx_ai_responses_conversation_id ON ai_responses(conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_responses_bot_id ON ai_responses(bot_id);
CREATE INDEX IF NOT EXISTS idx_ai_responses_created_at ON ai_responses(created_at DESC);
`

const createTemplatesTable = `
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    external_id VARCHAR(255),
    name VARCHAR(512) NOT NULL,
    language VARCHAR(20) NOT NULL,
    category VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    quality_score VARCHAR(50) NOT NULL DEFAULT 'UNKNOWN',
    components JSONB NOT NULL DEFAULT '[]',
    rejection_reason TEXT,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_external_id_unique ON templates(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_templates_tenant_id ON templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_templates_channel_id ON templates(channel_id);
CREATE INDEX IF NOT EXISTS idx_templates_status ON templates(status);
CREATE INDEX IF NOT EXISTS idx_templates_name_language ON templates(tenant_id, channel_id, name, language);
CREATE INDEX IF NOT EXISTS idx_templates_last_synced_at ON templates(last_synced_at);

DROP TRIGGER IF EXISTS update_templates_updated_at ON templates;
CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
`

const createMessageLogsTable = `
CREATE TABLE IF NOT EXISTS message_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    level VARCHAR(20) NOT NULL,
    source VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_logs_tenant_id ON message_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_message_logs_channel_id ON message_logs(channel_id);
CREATE INDEX IF NOT EXISTS idx_message_logs_level ON message_logs(level);
CREATE INDEX IF NOT EXISTS idx_message_logs_source ON message_logs(source);
CREATE INDEX IF NOT EXISTS idx_message_logs_created_at ON message_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_logs_tenant_created_at ON message_logs(tenant_id, created_at DESC);
`

const createSystemMetricsTable = `
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    labels JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_tenant_id ON system_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_system_metrics_name ON system_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_system_metrics_created_at ON system_metrics(created_at DESC);
`

const createWhatsAppPaymentsTables = `
CREATE TABLE IF NOT EXISTS whatsapp_payments (
    id VARCHAR(128) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL,
    channel_id VARCHAR(64) NOT NULL,
    order_id VARCHAR(128),
    reference_id VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    method VARCHAR(50),
    gateway_payment_id VARCHAR(255),
    gateway_order_id VARCHAR(255),
    description TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    message_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_payments_organization ON whatsapp_payments(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_payments_channel ON whatsapp_payments(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_payments_customer ON whatsapp_payments(customer_phone);
CREATE INDEX IF NOT EXISTS idx_wa_payments_reference ON whatsapp_payments(reference_id);
CREATE INDEX IF NOT EXISTS idx_wa_payments_status ON whatsapp_payments(status);
CREATE INDEX IF NOT EXISTS idx_wa_payments_created ON whatsapp_payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_payments_gateway ON whatsapp_payments(gateway_payment_id);

DROP TRIGGER IF EXISTS update_whatsapp_payments_updated_at ON whatsapp_payments;
CREATE TRIGGER update_whatsapp_payments_updated_at
    BEFORE UPDATE ON whatsapp_payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
`

const createWhatsAppHistoryImportsTable = `
CREATE TABLE IF NOT EXISTS whatsapp_history_imports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_conversations INT NOT NULL DEFAULT 0,
    imported_conversations INT NOT NULL DEFAULT 0,
    total_messages INT NOT NULL DEFAULT 0,
    imported_messages INT NOT NULL DEFAULT 0,
    total_contacts INT NOT NULL DEFAULT 0,
    imported_contacts INT NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    error_details JSONB DEFAULT '{}',
    import_since TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_history_imports_channel ON whatsapp_history_imports(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_tenant ON whatsapp_history_imports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_status ON whatsapp_history_imports(status);
CREATE INDEX IF NOT EXISTS idx_wa_history_imports_created ON whatsapp_history_imports(created_at DESC);

DROP TRIGGER IF EXISTS update_whatsapp_history_imports_updated_at ON whatsapp_history_imports;
CREATE TRIGGER update_whatsapp_history_imports_updated_at
    BEFORE UPDATE ON whatsapp_history_imports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
`

const createWhatsAppCoexistenceTables = `
CREATE TABLE IF NOT EXISTS whatsapp_coexistence_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS whatsapp_coexistence_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_coexistence_activity_channel ON whatsapp_coexistence_activity(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_coexistence_activity_tenant ON whatsapp_coexistence_activity(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wa_coexistence_activity_created ON whatsapp_coexistence_activity(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_coexistence_notifications_channel ON whatsapp_coexistence_notifications(channel_id);
CREATE INDEX IF NOT EXISTS idx_wa_coexistence_notifications_tenant ON whatsapp_coexistence_notifications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wa_coexistence_notifications_read_at ON whatsapp_coexistence_notifications(read_at);
`
