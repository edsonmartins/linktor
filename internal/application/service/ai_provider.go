package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Message represents a message in the completion request
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// CompletionRequest represents a request for AI completion
type CompletionRequest struct {
	Messages     []Message `json:"messages"`
	Model        string    `json:"model"`
	MaxTokens    int       `json:"max_tokens"`
	Temperature  float64   `json:"temperature"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
}

// CompletionResponse represents a response from AI completion
type CompletionResponse struct {
	Content      string  `json:"content"`
	Model        string  `json:"model"`
	TokensUsed   int     `json:"tokens_used"`
	PromptTokens int     `json:"prompt_tokens"`
	CompTokens   int     `json:"completion_tokens"`
	FinishReason string  `json:"finish_reason"`
	Confidence   float64 `json:"confidence,omitempty"`
	LatencyMs    int64   `json:"latency_ms"`
}

// EmbeddingRequest represents a request for embedding generation
type EmbeddingRequest struct {
	Text  string `json:"text"`
	Model string `json:"model,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Embedding  []float64 `json:"embedding"`
	Model      string    `json:"model"`
	TokensUsed int       `json:"tokens_used"`
}

// IntentClassificationRequest represents an intent classification request
type IntentClassificationRequest struct {
	Message string   `json:"message"`
	Intents []string `json:"intents"`
	Model   string   `json:"model,omitempty"`
}

// SentimentAnalysisRequest represents a sentiment analysis request
type SentimentAnalysisRequest struct {
	Message string `json:"message"`
	Model   string `json:"model,omitempty"`
}

// AIProvider defines the interface for AI providers
type AIProvider interface {
	// Name returns the provider name (openai, anthropic, ollama)
	Name() entity.AIProviderType

	// Models returns available models for this provider
	Models() []string

	// DefaultModel returns the default model for this provider
	DefaultModel() string

	// Complete generates a completion from messages
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Embed generates embeddings for text (for RAG)
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// ClassifyIntent classifies message intent
	ClassifyIntent(ctx context.Context, req *IntentClassificationRequest) (*entity.IntentResult, error)

	// AnalyzeSentiment analyzes message sentiment
	AnalyzeSentiment(ctx context.Context, req *SentimentAnalysisRequest) (*entity.SentimentResult, error)

	// IsAvailable checks if the provider is properly configured and available
	IsAvailable() bool
}

// AIProviderConfig holds common configuration for AI providers
type AIProviderConfig struct {
	APIKey        string  `json:"api_key"`
	BaseURL       string  `json:"base_url,omitempty"`
	OrgID         string  `json:"org_id,omitempty"`
	DefaultModel  string  `json:"default_model,omitempty"`
	MaxRetries    int     `json:"max_retries,omitempty"`
	TimeoutSecs   int     `json:"timeout_secs,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	MaxTokens     int     `json:"max_tokens,omitempty"`
}

// AIProviderFactory creates AI provider instances
type AIProviderFactory struct {
	mu        sync.RWMutex
	providers map[entity.AIProviderType]AIProvider
}

// NewAIProviderFactory creates a new AI provider factory
func NewAIProviderFactory() *AIProviderFactory {
	return &AIProviderFactory{
		providers: make(map[entity.AIProviderType]AIProvider),
	}
}

// Register registers an AI provider
func (f *AIProviderFactory) Register(provider AIProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[provider.Name()] = provider
}

// Get returns an AI provider by type
func (f *AIProviderFactory) Get(providerType entity.AIProviderType) (AIProvider, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	provider, ok := f.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("AI provider not found: %s", providerType)
	}

	if !provider.IsAvailable() {
		return nil, fmt.Errorf("AI provider not available: %s", providerType)
	}

	return provider, nil
}

// GetForBot returns the appropriate AI provider for a bot
func (f *AIProviderFactory) GetForBot(bot *entity.Bot) (AIProvider, error) {
	return f.Get(bot.Provider)
}

// List returns all registered providers
func (f *AIProviderFactory) List() []entity.AIProviderType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]entity.AIProviderType, 0, len(f.providers))
	for t := range f.providers {
		types = append(types, t)
	}
	return types
}

// ListAvailable returns only available providers
func (f *AIProviderFactory) ListAvailable() []entity.AIProviderType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]entity.AIProviderType, 0, len(f.providers))
	for t, p := range f.providers {
		if p.IsAvailable() {
			types = append(types, t)
		}
	}
	return types
}

// ProviderInfo contains information about an AI provider
type ProviderInfo struct {
	Name         entity.AIProviderType `json:"name"`
	Available    bool                  `json:"available"`
	DefaultModel string                `json:"default_model"`
	Models       []string              `json:"models"`
}

// ListProviders returns information about all registered providers
func (f *AIProviderFactory) ListProviders() []ProviderInfo {
	f.mu.RLock()
	defer f.mu.RUnlock()

	infos := make([]ProviderInfo, 0, len(f.providers))
	for _, p := range f.providers {
		infos = append(infos, ProviderInfo{
			Name:         p.Name(),
			Available:    p.IsAvailable(),
			DefaultModel: p.DefaultModel(),
			Models:       p.Models(),
		})
	}
	return infos
}

// BotResponse represents a response from the bot
type BotResponse struct {
	Content        string                 `json:"content"`
	Confidence     float64                `json:"confidence"`
	Intent         *entity.Intent         `json:"intent,omitempty"`
	Sentiment      entity.Sentiment       `json:"sentiment,omitempty"`
	Actions        []BotAction            `json:"actions,omitempty"`
	QuickReplies   []entity.QuickReply    `json:"quick_replies,omitempty"` // Interactive buttons
	ShouldEscalate bool                   `json:"should_escalate"`
	EscalateReason string                 `json:"escalate_reason,omitempty"`
	TokensUsed     int                    `json:"tokens_used"`
	LatencyMs      int64                  `json:"latency_ms"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	FlowID         string                 `json:"flow_id,omitempty"`     // Active flow if any
	FlowEnded      bool                   `json:"flow_ended,omitempty"`  // True if flow just ended
}

// BotAction represents an action the bot wants to perform
type BotAction struct {
	Type       string                 `json:"type"`   // send_message, assign_tag, set_field, etc.
	Parameters map[string]interface{} `json:"parameters"`
}

// BotService defines the interface for bot operations
type BotService interface {
	// GetBotForChannel returns the bot assigned to a channel
	GetBotForChannel(ctx context.Context, channelID string) (*entity.Bot, error)

	// ShouldBotHandle determines if bot should handle this message
	ShouldBotHandle(ctx context.Context, conversation *entity.Conversation, bot *entity.Bot) (bool, error)

	// ProcessMessage processes a message through the bot
	ProcessMessage(ctx context.Context, message *entity.Message, conversation *entity.Conversation, bot *entity.Bot) (*BotResponse, error)

	// ShouldEscalate checks if conversation should be escalated
	ShouldEscalate(ctx context.Context, botCtx *entity.ConversationContext, bot *entity.Bot, response *BotResponse) (bool, string, error)

	// GetOrCreateContext gets or creates conversation context
	GetOrCreateContext(ctx context.Context, conversationID string) (*entity.ConversationContext, error)
}

// Global factory instance
var globalAIFactory *AIProviderFactory
var factoryOnce sync.Once

// GetAIProviderFactory returns the global AI provider factory
func GetAIProviderFactory() *AIProviderFactory {
	factoryOnce.Do(func() {
		globalAIFactory = NewAIProviderFactory()
	})
	return globalAIFactory
}

// BuildPromptFromContext builds the messages array from conversation context
func BuildPromptFromContext(systemPrompt string, context *entity.ConversationContext, currentMessage string) []Message {
	messages := make([]Message, 0, len(context.ContextWindow)+2)

	// Add system prompt
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add context window messages
	for _, msg := range context.ContextWindow {
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current message
	messages = append(messages, Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}

// CalculateConfidence calculates a confidence score based on various factors
func CalculateConfidence(response *CompletionResponse, intent *entity.Intent) float64 {
	// Base confidence from intent if available
	confidence := 0.7 // default
	if intent != nil {
		confidence = intent.Confidence
	}

	// Adjust based on finish reason
	switch response.FinishReason {
	case "stop":
		// Normal completion, no adjustment
	case "length":
		// Response was cut off, lower confidence
		confidence *= 0.8
	case "content_filter":
		// Content was filtered, lower confidence
		confidence *= 0.5
	}

	return confidence
}
