-- +goose Up
-- Align tables with the columns their repositories actually read/write.
-- The baseline (v1) shipped conversation_contexts/ai_responses/templates with a
-- shape that diverged from bot_repo.go / analytics_repo.go / template_repo.go,
-- so those code paths failed with "column does not exist" against a fresh DB.

-- conversation_contexts: columns used by ConversationContextRepository (bot_repo.go)
-- and by AnalyticsRepository flow metrics (current_flow_id, metadata).
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS bot_id UUID REFERENCES bots(id) ON DELETE SET NULL;
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS intent_name VARCHAR(255);
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS intent_confidence DOUBLE PRECISION;
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS entities JSONB DEFAULT '{}';
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS context_window JSONB DEFAULT '[]';
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS state JSONB DEFAULT '{}';
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS last_analysis_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS current_flow_id UUID;
ALTER TABLE conversation_contexts ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_conversation_contexts_bot_id ON conversation_contexts(bot_id);
CREATE INDEX IF NOT EXISTS idx_conversation_contexts_current_flow_id ON conversation_contexts(current_flow_id);

-- ai_responses: the repository inserts message_id and omits conversation_id.
ALTER TABLE ai_responses ADD COLUMN IF NOT EXISTS message_id UUID;
ALTER TABLE ai_responses ALTER COLUMN conversation_id DROP NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_responses_message_id ON ai_responses(message_id);

-- conversations.metadata: read by AnalyticsRepository.GetEscalationsByReason
-- (metadata->>'escalation_reason'). escalated_at already added in the baseline.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- templates.sub_category: filtered by TemplateRepository sort/filter allowlist.
ALTER TABLE templates ADD COLUMN IF NOT EXISTS sub_category VARCHAR(100);

-- +goose Down
ALTER TABLE templates DROP COLUMN IF EXISTS sub_category;
ALTER TABLE conversations DROP COLUMN IF EXISTS metadata;
DROP INDEX IF EXISTS idx_ai_responses_message_id;
ALTER TABLE ai_responses DROP COLUMN IF EXISTS message_id;
DROP INDEX IF EXISTS idx_conversation_contexts_current_flow_id;
DROP INDEX IF EXISTS idx_conversation_contexts_bot_id;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS metadata;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS current_flow_id;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS last_analysis_at;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS state;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS context_window;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS entities;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS intent_confidence;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS intent_name;
ALTER TABLE conversation_contexts DROP COLUMN IF EXISTS bot_id;
