package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// Provider implements the AIProvider interface for Ollama
type Provider struct {
	client         *Client
	defaultModel   string
	embeddingModel string
}

// ProviderConfig holds configuration for the Ollama provider
type ProviderConfig struct {
	BaseURL        string
	DefaultModel   string
	EmbeddingModel string
	Timeout        time.Duration
	RateLimit      int
}

// NewProvider creates a new Ollama provider
func NewProvider(config *ProviderConfig) *Provider {
	clientConfig := &ClientConfig{
		BaseURL:   config.BaseURL,
		Timeout:   config.Timeout,
		RateLimit: config.RateLimit,
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "llama3.1:8b"
	}

	embeddingModel := config.EmbeddingModel
	if embeddingModel == "" {
		embeddingModel = "nomic-embed-text"
	}

	return &Provider{
		client:         NewClient(clientConfig),
		defaultModel:   defaultModel,
		embeddingModel: embeddingModel,
	}
}

// Name returns the provider name
func (p *Provider) Name() entity.AIProviderType {
	return entity.AIProviderOllama
}

// Models returns available models
func (p *Provider) Models() []string {
	// Try to get actual models from Ollama
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := p.client.ListModels(ctx)
	if err != nil {
		// Return default models if we can't reach Ollama
		return p.client.DefaultModels()
	}

	modelNames := make([]string, len(models))
	for i, m := range models {
		modelNames[i] = m.Name
	}
	return modelNames
}

// DefaultModel returns the default model
func (p *Provider) DefaultModel() string {
	return p.defaultModel
}

// IsAvailable checks if the provider is properly configured
func (p *Provider) IsAvailable() bool {
	// Refresh availability
	p.client.RefreshAvailability()
	return p.client.IsConfigured()
}

// Complete generates a completion from messages
func (p *Provider) Complete(ctx context.Context, req *service.CompletionRequest) (*service.CompletionResponse, error) {
	startTime := time.Now()

	// Build messages for Ollama
	messages := make([]ChatMessage, 0, len(req.Messages)+1)

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		messages = append(messages, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Determine model to use
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	// Set defaults
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// Build request
	chatReq := &ChatRequest{
		Model:    model,
		Messages: messages,
		Options: &Options{
			Temperature: temperature,
			NumPredict:  maxTokens,
		},
	}

	// Call Ollama API
	resp, err := p.client.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama completion failed: %w", err)
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// Determine finish reason
	finishReason := "stop"
	if resp.EvalCount >= maxTokens {
		finishReason = "length"
	}

	return &service.CompletionResponse{
		Content:      resp.Message.Content,
		Model:        resp.Model,
		TokensUsed:   resp.PromptEvalCount + resp.EvalCount,
		PromptTokens: resp.PromptEvalCount,
		CompTokens:   resp.EvalCount,
		FinishReason: finishReason,
		LatencyMs:    latencyMs,
	}, nil
}

// Embed generates embeddings for text
func (p *Provider) Embed(ctx context.Context, req *service.EmbeddingRequest) (*service.EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = p.embeddingModel
	}

	embedReq := &EmbeddingRequest{
		Model:  model,
		Prompt: req.Text,
	}

	resp, err := p.client.CreateEmbedding(ctx, embedReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama embedding failed: %w", err)
	}

	return &service.EmbeddingResponse{
		Embedding:  resp.Embedding,
		Model:      model,
		TokensUsed: 0, // Ollama doesn't report token usage for embeddings
	}, nil
}

// ClassifyIntent classifies message intent using the LLM
func (p *Provider) ClassifyIntent(ctx context.Context, req *service.IntentClassificationRequest) (*entity.IntentResult, error) {
	// Build prompt for intent classification
	intentsJSON, _ := json.Marshal(req.Intents)
	systemPrompt := fmt.Sprintf(`You are an intent classifier. Given a user message, classify it into one of these intents: %s

Respond with a JSON object containing:
- "intent": the classified intent name (must be one from the list)
- "confidence": a number between 0 and 1 indicating confidence
- "entities": an object with any extracted entities

If the message doesn't clearly match any intent, use "unknown" as the intent.
Respond ONLY with the JSON object, no other text.`, string(intentsJSON))

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	chatReq := &ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Message},
		},
		Options: &Options{
			Temperature: 0.3, // Lower temperature for classification
			NumPredict:  256,
		},
		Format: "json",
	}

	resp, err := p.client.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("intent classification failed: %w", err)
	}

	// Parse the response
	var result struct {
		Intent     string            `json:"intent"`
		Confidence float64           `json:"confidence"`
		Entities   map[string]string `json:"entities"`
	}

	content := resp.Message.Content
	// Try to extract JSON from the response
	jsonContent := extractJSON(content)
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		// Fallback: try to extract intent from text
		contentLower := strings.ToLower(content)
		for _, intent := range req.Intents {
			if strings.Contains(contentLower, strings.ToLower(intent)) {
				result.Intent = intent
				result.Confidence = 0.6
				break
			}
		}
		if result.Intent == "" {
			result.Intent = "unknown"
			result.Confidence = 0.3
		}
	}

	intent := entity.NewIntent(result.Intent, result.Confidence)
	if result.Entities != nil {
		for k, v := range result.Entities {
			intent.AddEntity(k, v)
		}
	}

	return &entity.IntentResult{
		Intent: intent,
	}, nil
}

// AnalyzeSentiment analyzes message sentiment
func (p *Provider) AnalyzeSentiment(ctx context.Context, req *service.SentimentAnalysisRequest) (*entity.SentimentResult, error) {
	systemPrompt := `You are a sentiment analyzer. Given a message, analyze its sentiment.

Respond with a JSON object containing:
- "sentiment": one of "positive", "neutral", or "negative"
- "score": a number from -1 (very negative) to 1 (very positive)
- "confidence": a number between 0 and 1 indicating confidence

Respond ONLY with the JSON object, no other text.`

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	chatReq := &ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Message},
		},
		Options: &Options{
			Temperature: 0.3,
			NumPredict:  128,
		},
		Format: "json",
	}

	resp, err := p.client.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("sentiment analysis failed: %w", err)
	}

	var result struct {
		Sentiment  string  `json:"sentiment"`
		Score      float64 `json:"score"`
		Confidence float64 `json:"confidence"`
	}

	content := resp.Message.Content
	jsonContent := extractJSON(content)
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		// Default to neutral if parsing fails
		result.Sentiment = "neutral"
		result.Score = 0
		result.Confidence = 0.5
	}

	// Map string sentiment to entity type
	var sentiment entity.Sentiment
	switch strings.ToLower(result.Sentiment) {
	case "positive":
		sentiment = entity.SentimentPositive
	case "negative":
		sentiment = entity.SentimentNegative
	default:
		sentiment = entity.SentimentNeutral
	}

	return &entity.SentimentResult{
		Sentiment:  sentiment,
		Score:      result.Score,
		Confidence: result.Confidence,
	}, nil
}

// extractJSON tries to extract JSON from a string that may contain extra text
func extractJSON(s string) string {
	// Find the first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

// Ensure Provider implements AIProvider interface
var _ service.AIProvider = (*Provider)(nil)
