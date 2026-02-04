package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// Provider implements the AIProvider interface for OpenAI
type Provider struct {
	client       *Client
	defaultModel string
	embeddingModel string
}

// ProviderConfig holds configuration for the OpenAI provider
type ProviderConfig struct {
	APIKey         string
	OrgID          string
	BaseURL        string
	DefaultModel   string
	EmbeddingModel string
	Timeout        time.Duration
	RateLimit      int
}

// NewProvider creates a new OpenAI provider
func NewProvider(config *ProviderConfig) *Provider {
	clientConfig := &ClientConfig{
		APIKey:    config.APIKey,
		OrgID:     config.OrgID,
		BaseURL:   config.BaseURL,
		Timeout:   config.Timeout,
		RateLimit: config.RateLimit,
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "gpt-4-turbo-preview"
	}

	embeddingModel := config.EmbeddingModel
	if embeddingModel == "" {
		embeddingModel = "text-embedding-ada-002"
	}

	return &Provider{
		client:         NewClient(clientConfig),
		defaultModel:   defaultModel,
		embeddingModel: embeddingModel,
	}
}

// Name returns the provider name
func (p *Provider) Name() entity.AIProviderType {
	return entity.AIProviderOpenAI
}

// Models returns available models
func (p *Provider) Models() []string {
	return p.client.Models()
}

// DefaultModel returns the default model
func (p *Provider) DefaultModel() string {
	return p.defaultModel
}

// IsAvailable checks if the provider is properly configured
func (p *Provider) IsAvailable() bool {
	return p.client.IsConfigured()
}

// Complete generates a completion from messages
func (p *Provider) Complete(ctx context.Context, req *service.CompletionRequest) (*service.CompletionResponse, error) {
	startTime := time.Now()

	// Build messages for OpenAI
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
	chatReq := &ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Call OpenAI API
	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI completion failed: %w", err)
	}

	// Build response
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	latencyMs := time.Since(startTime).Milliseconds()

	return &service.CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		TokensUsed:   resp.Usage.TotalTokens,
		PromptTokens: resp.Usage.PromptTokens,
		CompTokens:   resp.Usage.CompletionTokens,
		FinishReason: resp.Choices[0].FinishReason,
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
		Model: model,
		Input: req.Text,
	}

	resp, err := p.client.CreateEmbedding(ctx, embedReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return &service.EmbeddingResponse{
		Embedding:  resp.Data[0].Embedding,
		Model:      resp.Model,
		TokensUsed: resp.Usage.TotalTokens,
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

	chatReq := &ChatCompletionRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Message},
		},
		MaxTokens:      256,
		Temperature:    0.3, // Lower temperature for classification
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("intent classification failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Parse the response
	var result struct {
		Intent     string            `json:"intent"`
		Confidence float64           `json:"confidence"`
		Entities   map[string]string `json:"entities"`
	}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		// Fallback: try to extract intent from text
		content := strings.ToLower(resp.Choices[0].Message.Content)
		for _, intent := range req.Intents {
			if strings.Contains(content, strings.ToLower(intent)) {
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

	chatReq := &ChatCompletionRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Message},
		},
		MaxTokens:      128,
		Temperature:    0.3,
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("sentiment analysis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var result struct {
		Sentiment  string  `json:"sentiment"`
		Score      float64 `json:"score"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
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

// Ensure Provider implements AIProvider interface
var _ service.AIProvider = (*Provider)(nil)
