# Plano de Implementação - Linktor - Fase 5: Chatbot/AI

## Status Atual do Projeto
- **Fase 1 (Foundation)**: DONE - Database, API Gateway, JWT Auth
- **Fase 2 (Core Messaging)**: DONE - NATS, Plugin System, WebChat adapter
- **Fase 3 (Admin Panel)**: DONE - React 19, Next.js 15, WebSocket real-time
- **Fase 4 (WhatsApp Official)**: DONE - Meta Cloud API integration
- **Fase 5 (Chatbot/AI)**: NEXT

---

## Objetivo
Implementar sistema de chatbot inteligente com suporte a múltiplos provedores de AI (OpenAI, Anthropic, modelos locais), fluxos automatizados, detecção de intenção e handoff para agentes humanos.

---

## Arquitetura

### Fluxo de Mensagens com AI

```
INBOUND:
Channel → Webhook → NATS(inbound) → ReceiveMessageUseCase → DB
                                           ↓
                                    EventMessageReceived
                                           ↓
                                    AIAnalyzerConsumer
                                           ↓
                              ┌────────────┴────────────┐
                              ↓                         ↓
                        Bot Handles              Escalate to Agent
                              ↓                         ↓
                    AIResponseUseCase           AssignmentService
                              ↓                         ↓
                    NATS(outbound)              Notify Agent
                              ↓
                    Channel Adapter → External
```

### Estrutura de Diretórios

```
internal/
├── adapters/
│   └── ai/
│       ├── openai/
│       │   ├── client.go       # OpenAI API client
│       │   └── provider.go     # AIProvider implementation
│       ├── anthropic/
│       │   ├── client.go       # Claude API client
│       │   └── provider.go     # AIProvider implementation
│       └── ollama/
│           ├── client.go       # Local Ollama client
│           └── provider.go     # AIProvider implementation
├── application/
│   ├── service/
│   │   ├── bot.go              # Bot service (routing, config)
│   │   ├── ai_provider.go      # AI provider interface & factory
│   │   ├── intent.go           # Intent classification
│   │   ├── knowledge.go        # Knowledge base/RAG
│   │   └── conversation_context.go  # Context management
│   └── usecase/
│       ├── generate_ai_response.go  # AI response generation
│       ├── analyze_message.go       # Message analysis/intent
│       └── escalate_conversation.go # Handoff to human
├── domain/
│   └── entity/
│       ├── bot.go              # Bot entity
│       ├── intent.go           # Intent entity
│       └── ai_response.go      # AI response entity
└── infrastructure/
    ├── database/
    │   ├── bot_repo.go         # Bot repository
    │   └── knowledge_repo.go   # Knowledge base repository
    └── nats/
        └── subjects.go         # Add AI subjects
```

---

## Entidades de Domínio

### Bot
```go
type Bot struct {
    ID          string            `json:"id"`
    TenantID    string            `json:"tenant_id"`
    Name        string            `json:"name"`
    Type        BotType           `json:"type"`        // ai, rule_based, hybrid
    Provider    AIProvider        `json:"provider"`    // openai, anthropic, ollama
    Model       string            `json:"model"`       // gpt-4, claude-3, llama3
    Config      BotConfig         `json:"config"`
    Status      BotStatus         `json:"status"`      // active, inactive, training
    Channels    []string          `json:"channels"`    // channel IDs assigned
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type BotConfig struct {
    SystemPrompt      string            `json:"system_prompt"`
    Temperature       float64           `json:"temperature"`
    MaxTokens         int               `json:"max_tokens"`
    ConfidenceThreshold float64         `json:"confidence_threshold"` // Min for auto-response
    EscalationRules   []EscalationRule  `json:"escalation_rules"`
    KnowledgeBaseID   *string           `json:"knowledge_base_id"`
    WelcomeMessage    *string           `json:"welcome_message"`
    FallbackMessage   string            `json:"fallback_message"`
    WorkingHours      *WorkingHours     `json:"working_hours"`
}

type EscalationRule struct {
    Condition  string   `json:"condition"`  // low_confidence, keyword, sentiment
    Value      string   `json:"value"`      // 0.5, "urgent", "negative"
    Action     string   `json:"action"`     // escalate, notify, tag
    Priority   string   `json:"priority"`   // high, urgent
}
```

### ConversationContext
```go
type ConversationContext struct {
    ID              string                 `json:"id"`
    ConversationID  string                 `json:"conversation_id"`
    BotID           *string                `json:"bot_id"`
    Intent          *Intent                `json:"intent"`
    Entities        map[string]interface{} `json:"entities"`
    Sentiment       Sentiment              `json:"sentiment"`
    ContextWindow   []ContextMessage       `json:"context_window"`
    State           map[string]interface{} `json:"state"`         // Flow state
    LastAnalysisAt  time.Time              `json:"last_analysis_at"`
    CreatedAt       time.Time              `json:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at"`
}

type Intent struct {
    Name       string  `json:"name"`
    Confidence float64 `json:"confidence"`
    Entities   map[string]string `json:"entities"`
}

type ContextMessage struct {
    Role      string    `json:"role"`      // user, assistant, system
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

### KnowledgeBase
```go
type KnowledgeBase struct {
    ID          string            `json:"id"`
    TenantID    string            `json:"tenant_id"`
    Name        string            `json:"name"`
    Type        KnowledgeType     `json:"type"`  // faq, documents, website
    Config      KnowledgeConfig   `json:"config"`
    Status      string            `json:"status"`
    ItemCount   int               `json:"item_count"`
    LastSyncAt  *time.Time        `json:"last_sync_at"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type KnowledgeItem struct {
    ID              string    `json:"id"`
    KnowledgeBaseID string    `json:"knowledge_base_id"`
    Question        string    `json:"question"`
    Answer          string    `json:"answer"`
    Keywords        []string  `json:"keywords"`
    Embedding       []float64 `json:"embedding"`  // Vector for RAG
    Source          string    `json:"source"`
    CreatedAt       time.Time `json:"created_at"`
}
```

---

## Tarefas de Implementação

### Sprint 9 - AI Foundation

| ID | Arquivo | Descrição |
|----|---------|-----------|
| 5.1.1 | `internal/domain/entity/bot.go` | Entity Bot, BotConfig, EscalationRule |
| 5.1.2 | `internal/domain/entity/intent.go` | Entity Intent, ConversationContext |
| 5.1.3 | `internal/domain/entity/knowledge.go` | Entity KnowledgeBase, KnowledgeItem |
| 5.1.4 | `internal/application/service/ai_provider.go` | Interface AIProvider + Factory |
| 5.1.5 | `internal/adapters/ai/openai/client.go` | OpenAI API client |
| 5.1.6 | `internal/adapters/ai/openai/provider.go` | OpenAI provider implementation |
| 5.1.7 | `internal/infrastructure/database/bot_repo.go` | Bot repository |
| 5.1.8 | `deploy/docker/migrations/005_ai_tables.sql` | Migrations para AI |

### Sprint 10 - AI Processing Pipeline

| ID | Arquivo | Descrição |
|----|---------|-----------|
| 5.2.1 | `internal/application/service/conversation_context.go` | Context window management |
| 5.2.2 | `internal/application/service/intent.go` | Intent classification service |
| 5.2.3 | `internal/application/usecase/analyze_message.go` | Message analysis use case |
| 5.2.4 | `internal/application/usecase/generate_ai_response.go` | AI response generation |
| 5.2.5 | `internal/infrastructure/nats/subjects.go` | Add AI subjects |
| 5.2.6 | `internal/infrastructure/nats/ai_consumer.go` | AI message consumer |
| 5.2.7 | `cmd/server/main.go` | Register AI consumers |

### Sprint 11 - Bot Management & Escalation

| ID | Arquivo | Descrição |
|----|---------|-----------|
| 5.3.1 | `internal/application/service/bot.go` | Bot routing & management |
| 5.3.2 | `internal/application/usecase/escalate_conversation.go` | Escalation use case |
| 5.3.3 | `internal/api/handlers/bot.go` | Bot CRUD endpoints |
| 5.3.4 | `internal/api/handlers/ai.go` | AI testing endpoints |
| 5.3.5 | `internal/adapters/ai/anthropic/provider.go` | Anthropic Claude provider |
| 5.3.6 | `internal/adapters/ai/ollama/provider.go` | Ollama local provider |

### Sprint 12 - Knowledge Base & RAG

| ID | Arquivo | Descrição |
|----|---------|-----------|
| 5.4.1 | `internal/application/service/knowledge.go` | Knowledge base service |
| 5.4.2 | `internal/application/service/embedding.go` | Embedding service |
| 5.4.3 | `internal/infrastructure/database/knowledge_repo.go` | Knowledge repository |
| 5.4.4 | `internal/api/handlers/knowledge.go` | Knowledge base endpoints |

### Sprint 13 - Admin UI

| ID | Arquivo | Descrição |
|----|---------|-----------|
| 5.5.1 | `web/admin/src/app/(dashboard)/bots/page.tsx` | Bot management page |
| 5.5.2 | `web/admin/src/app/(dashboard)/bots/[id]/page.tsx` | Bot config page |
| 5.5.3 | `web/admin/src/app/(dashboard)/bots/[id]/test.tsx` | Bot testing playground |
| 5.5.4 | `web/admin/src/app/(dashboard)/knowledge/page.tsx` | Knowledge base management |
| 5.5.5 | `web/admin/src/components/bot/prompt-editor.tsx` | System prompt editor |
| 5.5.6 | `web/admin/src/components/bot/flow-builder.tsx` | Visual flow builder (future) |

---

## Interfaces Principais

### AIProvider Interface
```go
type AIProvider interface {
    // Name returns the provider name (openai, anthropic, ollama)
    Name() string

    // Models returns available models
    Models() []string

    // Complete generates a completion from messages
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

    // Embed generates embeddings for text (for RAG)
    Embed(ctx context.Context, text string) ([]float64, error)

    // ClassifyIntent classifies message intent
    ClassifyIntent(ctx context.Context, message string, intents []string) (*IntentResult, error)

    // AnalyzeSentiment analyzes message sentiment
    AnalyzeSentiment(ctx context.Context, message string) (*SentimentResult, error)
}

type CompletionRequest struct {
    Messages    []Message  `json:"messages"`
    Model       string     `json:"model"`
    MaxTokens   int        `json:"max_tokens"`
    Temperature float64    `json:"temperature"`
    SystemPrompt string    `json:"system_prompt,omitempty"`
    Context     []ContextMessage `json:"context,omitempty"`
}

type CompletionResponse struct {
    Content     string  `json:"content"`
    Model       string  `json:"model"`
    TokensUsed  int     `json:"tokens_used"`
    FinishReason string `json:"finish_reason"`
    Confidence  float64 `json:"confidence,omitempty"`
}
```

### BotService Interface
```go
type BotService interface {
    // GetBotForChannel returns the bot assigned to a channel
    GetBotForChannel(ctx context.Context, channelID string) (*entity.Bot, error)

    // ShouldBotHandle determines if bot should handle this message
    ShouldBotHandle(ctx context.Context, conversation *entity.Conversation) (bool, error)

    // ProcessMessage processes a message through the bot
    ProcessMessage(ctx context.Context, message *entity.Message, conversation *entity.Conversation) (*BotResponse, error)

    // ShouldEscalate checks if conversation should be escalated
    ShouldEscalate(ctx context.Context, context *entity.ConversationContext) (bool, *EscalationReason, error)
}

type BotResponse struct {
    Content     string                 `json:"content"`
    Confidence  float64                `json:"confidence"`
    Intent      *entity.Intent         `json:"intent,omitempty"`
    Actions     []BotAction            `json:"actions,omitempty"`
    ShouldEscalate bool                `json:"should_escalate"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

---

## Database Migrations

### 005_ai_tables.sql
```sql
-- Bots table
CREATE TABLE bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'ai',
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(255) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    channels UUID[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bots_tenant_id ON bots(tenant_id);
CREATE INDEX idx_bots_status ON bots(status);
CREATE INDEX idx_bots_channels ON bots USING GIN(channels);

-- Conversation contexts table
CREATE TABLE conversation_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    intent_name VARCHAR(100),
    intent_confidence DECIMAL(3,2),
    entities JSONB DEFAULT '{}',
    sentiment VARCHAR(20),
    context_window JSONB DEFAULT '[]',
    state JSONB DEFAULT '{}',
    last_analysis_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_conversation_contexts_conversation ON conversation_contexts(conversation_id);
CREATE INDEX idx_conversation_contexts_bot_id ON conversation_contexts(bot_id);
CREATE INDEX idx_conversation_contexts_intent ON conversation_contexts(intent_name);

-- Knowledge bases table
CREATE TABLE knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'faq',
    config JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    item_count INT DEFAULT 0,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_knowledge_bases_tenant_id ON knowledge_bases(tenant_id);

-- Knowledge items table
CREATE TABLE knowledge_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_base_id UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    keywords TEXT[] DEFAULT '{}',
    embedding vector(1536),  -- OpenAI ada-002 dimensions
    source VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_knowledge_items_kb_id ON knowledge_items(knowledge_base_id);
CREATE INDEX idx_knowledge_items_keywords ON knowledge_items USING GIN(keywords);

-- AI responses audit table
CREATE TABLE ai_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    prompt JSONB NOT NULL,
    response TEXT NOT NULL,
    confidence DECIMAL(3,2),
    tokens_used INT,
    latency_ms INT,
    model VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ai_responses_message_id ON ai_responses(message_id);
CREATE INDEX idx_ai_responses_bot_id ON ai_responses(bot_id);
CREATE INDEX idx_ai_responses_created_at ON ai_responses(created_at);

-- Extend conversations table
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS bot_id UUID REFERENCES bots(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_bot_response_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS escalation_reason VARCHAR(500);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS ai_handled BOOLEAN DEFAULT FALSE;

-- Extend messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_confidence DECIMAL(3,2);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_intent VARCHAR(100);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_entities JSONB;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS ai_model VARCHAR(100);

-- Add 'bot' to sender_type enum (if using enum, otherwise just document convention)
COMMENT ON COLUMN messages.sender_type IS 'Allowed values: contact, user, system, bot';

-- Enable pgvector extension for embeddings (if not enabled)
CREATE EXTENSION IF NOT EXISTS vector;
```

---

## NATS Subjects

### Adicionar em subjects.go
```go
// AI/Bot subjects
const (
    // Bot message analysis
    SubjectBotAnalyze    = "linktor.bot.analyze"
    SubjectBotAnalyzeAll = "linktor.bot.analyze.>"

    // Bot responses
    SubjectBotResponse   = "linktor.bot.response"
    SubjectBotResponseTenant = "linktor.bot.response.%s" // tenant_id

    // Escalation
    SubjectBotEscalate   = "linktor.bot.escalate"
    SubjectBotEscalateTenant = "linktor.bot.escalate.%s"

    // Intent classification
    SubjectBotIntent     = "linktor.bot.intent"

    // Context updates
    SubjectBotContext    = "linktor.bot.context"
)

// Helper functions
func SubjectBotAnalyzeTenant(tenantID string) string {
    return fmt.Sprintf("linktor.bot.analyze.%s", tenantID)
}

func SubjectBotResponseForTenant(tenantID string) string {
    return fmt.Sprintf(SubjectBotResponseTenant, tenantID)
}

func SubjectBotEscalateForTenant(tenantID string) string {
    return fmt.Sprintf(SubjectBotEscalateTenant, tenantID)
}
```

---

## Fluxo de Processamento

### 1. Nova Mensagem Recebida
```
1. ReceiveMessageUseCase publica EventMessageReceived
2. AIAnalyzerConsumer recebe evento
3. Verifica se canal tem bot ativo
4. Se sim:
   a. Carrega ConversationContext
   b. Atualiza context_window com nova mensagem
   c. Classifica intent (se necessário)
   d. Verifica regras de escalação
   e. Se não escalar:
      - Gera resposta via AIProvider
      - Publica para SubjectBotResponse
   f. Se escalar:
      - Publica para SubjectBotEscalate
      - Atualiza conversation status
5. BotResponseConsumer envia resposta via SendMessageUseCase
```

### 2. Geração de Resposta
```
1. GenerateAIResponseUseCase recebe request
2. Carrega bot config (system prompt, temperature, etc)
3. Busca knowledge base (se configurado)
4. Faz busca semântica por contexto relevante (RAG)
5. Monta prompt com:
   - System prompt do bot
   - Context window (últimas N mensagens)
   - Knowledge relevante
   - Mensagem atual
6. Chama AIProvider.Complete()
7. Salva resposta em ai_responses (audit)
8. Retorna BotResponse
```

### 3. Escalação
```
1. EscalateConversationUseCase recebe request
2. Atualiza conversation:
   - status = "pending"
   - escalation_reason = motivo
   - ai_handled = false
3. Remove bot_id da conversation
4. Publica evento de escalação
5. AssignmentService pode auto-atribuir agente
6. Notifica agentes disponíveis via WebSocket
```

---

## Configuração de Ambiente

### Variables de Ambiente
```env
# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_ORG_ID=org-...
OPENAI_DEFAULT_MODEL=gpt-4-turbo-preview

# Anthropic
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_DEFAULT_MODEL=claude-3-sonnet-20240229

# Ollama (local)
OLLAMA_HOST=http://localhost:11434
OLLAMA_DEFAULT_MODEL=llama3

# AI Processing
AI_CONTEXT_WINDOW_SIZE=10
AI_DEFAULT_TEMPERATURE=0.7
AI_MAX_TOKENS=1024
AI_CONFIDENCE_THRESHOLD=0.7

# Vector Store (for RAG)
PGVECTOR_ENABLED=true
EMBEDDING_MODEL=text-embedding-ada-002
```

---

## API Endpoints

### Bot Management
```
POST   /api/v1/bots                    # Create bot
GET    /api/v1/bots                    # List bots
GET    /api/v1/bots/:id                # Get bot
PUT    /api/v1/bots/:id                # Update bot
DELETE /api/v1/bots/:id                # Delete bot
POST   /api/v1/bots/:id/test           # Test bot with message
POST   /api/v1/bots/:id/channels       # Assign channels
DELETE /api/v1/bots/:id/channels/:cid  # Unassign channel
```

### Knowledge Base
```
POST   /api/v1/knowledge-bases         # Create KB
GET    /api/v1/knowledge-bases         # List KBs
GET    /api/v1/knowledge-bases/:id     # Get KB
PUT    /api/v1/knowledge-bases/:id     # Update KB
DELETE /api/v1/knowledge-bases/:id     # Delete KB
POST   /api/v1/knowledge-bases/:id/items      # Add item
PUT    /api/v1/knowledge-bases/:id/items/:iid # Update item
DELETE /api/v1/knowledge-bases/:id/items/:iid # Delete item
POST   /api/v1/knowledge-bases/:id/sync       # Sync from source
POST   /api/v1/knowledge-bases/:id/search     # Semantic search
```

### AI Testing
```
POST   /api/v1/ai/complete             # Direct completion test
POST   /api/v1/ai/classify-intent      # Intent classification test
POST   /api/v1/ai/analyze-sentiment    # Sentiment analysis test
POST   /api/v1/ai/embed                # Generate embedding
```

---

## Verificação

1. **Bot CRUD**: Criar, listar, atualizar, deletar bots
2. **Bot Assignment**: Atribuir bot a canal, verificar routing
3. **Auto-Response**: Enviar mensagem → receber resposta automática
4. **Context Window**: Verificar que bot mantém contexto
5. **Intent Classification**: Verificar detecção de intent
6. **Escalation**: Testar escalação para agente humano
7. **Knowledge Base**: Criar KB, adicionar itens, buscar
8. **RAG**: Verificar que bot usa KB para contextualizar

---

## Ordem de Implementação

```
Semana 1: Entities + Migrations + OpenAI Provider
Semana 2: AI Consumer + Context Service + GenerateAIResponseUseCase
Semana 3: Bot Service + Escalation + API Endpoints
Semana 4: Knowledge Base + RAG + Embedding
Semana 5: Admin UI - Bot Management
Semana 6: Admin UI - Knowledge Base + Testing
Semana 7: Anthropic + Ollama providers
Semana 8: Testes + Documentação + Otimizações
```

---

## Considerações de Segurança

1. **API Keys**: Armazenar criptografadas no banco
2. **Rate Limiting**: Limitar chamadas por tenant
3. **Token Limits**: Monitorar e limitar uso de tokens
4. **Audit Trail**: Registrar todas as interações AI
5. **PII Detection**: Evitar vazar dados sensíveis para AI
6. **Model Selection**: Permitir apenas modelos aprovados

---

## Métricas e Monitoramento

1. **Latência de Resposta**: Tempo médio de geração
2. **Taxa de Escalação**: % de conversas escaladas
3. **Confidence Score**: Distribuição de confiança
4. **Token Usage**: Consumo por tenant/bot
5. **Intent Distribution**: Intents mais comuns
6. **Sentiment Trends**: Sentimento ao longo do tempo
7. **Resolution Rate**: % resolvido pelo bot vs humano
