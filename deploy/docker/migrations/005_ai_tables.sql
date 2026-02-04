-- ============================================
-- LINKTOR PHASE 5: AI/CHATBOT TABLES
-- ============================================
-- This migration adds tables for AI-powered chatbot functionality
-- including bots, conversation contexts, knowledge bases, and AI responses

-- Enable pgvector extension for embeddings (for RAG)
CREATE EXTENSION IF NOT EXISTS vector;

-- ============================================
-- BOTS TABLE
-- ============================================
-- Stores AI bot configurations
CREATE TABLE IF NOT EXISTS bots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'ai',  -- ai, rule_based, hybrid
    provider VARCHAR(50) NOT NULL,  -- openai, anthropic, ollama
    model VARCHAR(255) NOT NULL,  -- gpt-4, claude-3, llama3
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',  -- active, inactive, training
    channels UUID[] DEFAULT '{}',  -- channel IDs assigned to this bot
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bots_tenant_id ON bots(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bots_status ON bots(status);
CREATE INDEX IF NOT EXISTS idx_bots_channels ON bots USING GIN(channels);

COMMENT ON TABLE bots IS 'AI chatbot configurations';
COMMENT ON COLUMN bots.type IS 'Bot type: ai (LLM-powered), rule_based (flow-based), hybrid (both)';
COMMENT ON COLUMN bots.provider IS 'AI provider: openai, anthropic, ollama';
COMMENT ON COLUMN bots.config IS 'JSON config: system_prompt, temperature, max_tokens, escalation_rules, etc.';
COMMENT ON COLUMN bots.channels IS 'Array of channel IDs this bot is assigned to';

-- ============================================
-- CONVERSATION CONTEXTS TABLE
-- ============================================
-- Stores AI context for each conversation (intent, sentiment, context window)
CREATE TABLE IF NOT EXISTS conversation_contexts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    intent_name VARCHAR(100),
    intent_confidence DECIMAL(5,4),  -- 0.0000 to 1.0000
    entities JSONB DEFAULT '{}',
    sentiment VARCHAR(20),  -- positive, neutral, negative
    context_window JSONB DEFAULT '[]',  -- Array of recent messages for context
    state JSONB DEFAULT '{}',  -- Flow state variables
    last_analysis_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_contexts_conversation ON conversation_contexts(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_contexts_bot_id ON conversation_contexts(bot_id);
CREATE INDEX IF NOT EXISTS idx_conversation_contexts_intent ON conversation_contexts(intent_name);
CREATE INDEX IF NOT EXISTS idx_conversation_contexts_sentiment ON conversation_contexts(sentiment);

COMMENT ON TABLE conversation_contexts IS 'AI context for conversations including intent, sentiment, and message history';
COMMENT ON COLUMN conversation_contexts.context_window IS 'JSON array of recent messages for LLM context';
COMMENT ON COLUMN conversation_contexts.state IS 'Flow state variables for rule-based bots';

-- ============================================
-- KNOWLEDGE BASES TABLE
-- ============================================
-- Stores knowledge bases for RAG (Retrieval Augmented Generation)
CREATE TABLE IF NOT EXISTS knowledge_bases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'faq',  -- faq, documents, website
    config JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',  -- active, inactive, syncing
    item_count INT DEFAULT 0,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_tenant_id ON knowledge_bases(tenant_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_status ON knowledge_bases(status);
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_type ON knowledge_bases(type);

COMMENT ON TABLE knowledge_bases IS 'Knowledge bases for RAG-enhanced AI responses';
COMMENT ON COLUMN knowledge_bases.type IS 'Knowledge type: faq (Q&A pairs), documents (uploaded files), website (crawled content)';

-- ============================================
-- KNOWLEDGE ITEMS TABLE
-- ============================================
-- Stores individual items in knowledge bases with vector embeddings
CREATE TABLE IF NOT EXISTS knowledge_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    keywords TEXT[] DEFAULT '{}',
    embedding vector(1536),  -- OpenAI text-embedding-ada-002 dimensions
    source VARCHAR(500),  -- Original source URL or file
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_items_kb_id ON knowledge_items(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_keywords ON knowledge_items USING GIN(keywords);

-- Create HNSW index for vector similarity search (faster than IVFFlat for <1M vectors)
CREATE INDEX IF NOT EXISTS idx_knowledge_items_embedding ON knowledge_items USING hnsw (embedding vector_cosine_ops);

COMMENT ON TABLE knowledge_items IS 'Individual knowledge items with vector embeddings for semantic search';
COMMENT ON COLUMN knowledge_items.embedding IS 'Vector embedding for semantic similarity search (RAG)';

-- ============================================
-- AI RESPONSES AUDIT TABLE
-- ============================================
-- Stores all AI-generated responses for auditing and analytics
CREATE TABLE IF NOT EXISTS ai_responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    prompt JSONB NOT NULL,  -- Full prompt sent to AI
    response TEXT NOT NULL,  -- AI response
    confidence DECIMAL(5,4),  -- Confidence score
    tokens_used INT,
    latency_ms INT,
    model VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_responses_message_id ON ai_responses(message_id);
CREATE INDEX IF NOT EXISTS idx_ai_responses_bot_id ON ai_responses(bot_id);
CREATE INDEX IF NOT EXISTS idx_ai_responses_created_at ON ai_responses(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_responses_model ON ai_responses(model);

COMMENT ON TABLE ai_responses IS 'Audit log of all AI-generated responses for analytics and debugging';

-- ============================================
-- EXTEND CONVERSATIONS TABLE
-- ============================================
-- Add AI-related columns to conversations
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS bot_id UUID REFERENCES bots(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_bot_response_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS escalation_reason VARCHAR(500);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS ai_handled BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_conversations_bot_id ON conversations(bot_id);
CREATE INDEX IF NOT EXISTS idx_conversations_ai_handled ON conversations(ai_handled);

COMMENT ON COLUMN conversations.bot_id IS 'Bot currently handling this conversation (NULL if handled by human)';
COMMENT ON COLUMN conversations.escalation_reason IS 'Reason for escalation from bot to human';
COMMENT ON COLUMN conversations.ai_handled IS 'Whether this conversation was/is handled by AI';

-- ============================================
-- EXTEND MESSAGES TABLE
-- ============================================
-- Add AI-related columns to messages
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_confidence DECIMAL(5,4);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_intent VARCHAR(100);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_entities JSONB;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_model VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_messages_ai_intent ON messages(ai_intent);

COMMENT ON COLUMN messages.ai_confidence IS 'AI confidence score for this message response';
COMMENT ON COLUMN messages.ai_intent IS 'Detected intent for incoming messages';
COMMENT ON COLUMN messages.ai_entities IS 'Extracted entities from the message';
COMMENT ON COLUMN messages.ai_model IS 'AI model used to generate the response';

-- ============================================
-- UPDATE TRIGGERS
-- ============================================
-- Add update triggers for new tables

CREATE TRIGGER update_bots_updated_at
    BEFORE UPDATE ON bots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conversation_contexts_updated_at
    BEFORE UPDATE ON conversation_contexts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_bases_updated_at
    BEFORE UPDATE ON knowledge_bases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_items_updated_at
    BEFORE UPDATE ON knowledge_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- FUNCTIONS FOR KNOWLEDGE BASE
-- ============================================

-- Function to update knowledge base item count
CREATE OR REPLACE FUNCTION update_knowledge_base_item_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE knowledge_bases SET item_count = item_count + 1, updated_at = NOW()
        WHERE id = NEW.knowledge_base_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE knowledge_bases SET item_count = item_count - 1, updated_at = NOW()
        WHERE id = OLD.knowledge_base_id;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_kb_item_count
    AFTER INSERT OR DELETE ON knowledge_items
    FOR EACH ROW EXECUTE FUNCTION update_knowledge_base_item_count();

-- Function to search knowledge items by vector similarity
CREATE OR REPLACE FUNCTION search_knowledge_items(
    p_knowledge_base_id UUID,
    p_embedding vector(1536),
    p_limit INT DEFAULT 5,
    p_min_score FLOAT DEFAULT 0.7
)
RETURNS TABLE (
    id UUID,
    question TEXT,
    answer TEXT,
    similarity FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ki.id,
        ki.question,
        ki.answer,
        1 - (ki.embedding <=> p_embedding) AS similarity
    FROM knowledge_items ki
    WHERE ki.knowledge_base_id = p_knowledge_base_id
      AND ki.embedding IS NOT NULL
      AND 1 - (ki.embedding <=> p_embedding) >= p_min_score
    ORDER BY ki.embedding <=> p_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION search_knowledge_items IS 'Search knowledge items by vector similarity using cosine distance';

-- ============================================
-- VIEWS FOR ANALYTICS
-- ============================================

-- Bot performance metrics view
CREATE OR REPLACE VIEW bot_performance_metrics AS
SELECT
    b.id AS bot_id,
    b.name AS bot_name,
    b.tenant_id,
    COUNT(ar.id) AS total_responses,
    AVG(ar.confidence)::DECIMAL(5,4) AS avg_confidence,
    AVG(ar.latency_ms)::INT AS avg_latency_ms,
    SUM(ar.tokens_used) AS total_tokens_used,
    COUNT(DISTINCT DATE(ar.created_at)) AS active_days
FROM bots b
LEFT JOIN ai_responses ar ON b.id = ar.bot_id
GROUP BY b.id, b.name, b.tenant_id;

COMMENT ON VIEW bot_performance_metrics IS 'Aggregated performance metrics for each bot';

-- Escalation analytics view
CREATE OR REPLACE VIEW escalation_analytics AS
SELECT
    c.tenant_id,
    c.bot_id,
    b.name AS bot_name,
    DATE(c.updated_at) AS date,
    COUNT(*) AS escalation_count,
    c.escalation_reason
FROM conversations c
LEFT JOIN bots b ON c.bot_id = b.id
WHERE c.escalation_reason IS NOT NULL
GROUP BY c.tenant_id, c.bot_id, b.name, DATE(c.updated_at), c.escalation_reason;

COMMENT ON VIEW escalation_analytics IS 'Analytics on conversation escalations from bot to human';
