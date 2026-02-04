package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// Provider implements the AIProvider interface for Anthropic Claude
type Provider struct {
	client       *Client
	defaultModel string
}

// ProviderConfig holds configuration for the Anthropic provider
type ProviderConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
	Timeout      time.Duration
	RateLimit    int
}

// NewProvider creates a new Anthropic provider
func NewProvider(config *ProviderConfig) *Provider {
	clientConfig := &ClientConfig{
		APIKey:    config.APIKey,
		BaseURL:   config.BaseURL,
		Timeout:   config.Timeout,
		RateLimit: config.RateLimit,
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = "claude-3-5-sonnet-20241022"
	}

	return &Provider{
		client:       NewClient(clientConfig),
		defaultModel: defaultModel,
	}
}

// Name returns the provider name
func (p *Provider) Name() entity.AIProviderType {
	return entity.AIProviderAnthropic
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

	// Build messages for Anthropic
	messages := make([]Message, 0, len(req.Messages))
	var systemPrompt string

	// Extract system prompt and convert messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		// Anthropic only accepts "user" and "assistant" roles
		role := msg.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		messages = append(messages, NewTextMessage(role, msg.Content))
	}

	// Override with explicit system prompt if provided
	if req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt
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
	msgReq := &MessageRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		System:      systemPrompt,
	}

	// Call Anthropic API
	resp, err := p.client.CreateMessage(ctx, msgReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic completion failed: %w", err)
	}

	// Extract text content
	content := GetTextContent(resp)
	if content == "" {
		return nil, fmt.Errorf("no text content in Anthropic response")
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// Map stop reason to finish reason
	finishReason := "stop"
	if resp.StopReason == "max_tokens" {
		finishReason = "length"
	} else if resp.StopReason == "stop_sequence" {
		finishReason = "stop"
	}

	return &service.CompletionResponse{
		Content:      content,
		Model:        resp.Model,
		TokensUsed:   resp.Usage.InputTokens + resp.Usage.OutputTokens,
		PromptTokens: resp.Usage.InputTokens,
		CompTokens:   resp.Usage.OutputTokens,
		FinishReason: finishReason,
		LatencyMs:    latencyMs,
	}, nil
}

// Embed generates embeddings for text
// Note: Anthropic doesn't have a native embedding API, so we return an error
func (p *Provider) Embed(ctx context.Context, req *service.EmbeddingRequest) (*service.EmbeddingResponse, error) {
	return nil, fmt.Errorf("Anthropic does not support embeddings; use OpenAI or another provider for embeddings")
}

// ClassifyIntent classifies message intent using Claude
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

	msgReq := &MessageRequest{
		Model:       model,
		Messages:    []Message{NewTextMessage("user", req.Message)},
		MaxTokens:   256,
		Temperature: 0.3, // Lower temperature for classification
		System:      systemPrompt,
	}

	resp, err := p.client.CreateMessage(ctx, msgReq)
	if err != nil {
		return nil, fmt.Errorf("intent classification failed: %w", err)
	}

	content := GetTextContent(resp)
	if content == "" {
		return nil, fmt.Errorf("no content in response")
	}

	// Parse the response
	var result struct {
		Intent     string            `json:"intent"`
		Confidence float64           `json:"confidence"`
		Entities   map[string]string `json:"entities"`
	}

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

// AnalyzeSentiment analyzes message sentiment using Claude
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

	msgReq := &MessageRequest{
		Model:       model,
		Messages:    []Message{NewTextMessage("user", req.Message)},
		MaxTokens:   128,
		Temperature: 0.3,
		System:      systemPrompt,
	}

	resp, err := p.client.CreateMessage(ctx, msgReq)
	if err != nil {
		return nil, fmt.Errorf("sentiment analysis failed: %w", err)
	}

	content := GetTextContent(resp)
	if content == "" {
		return nil, fmt.Errorf("no content in response")
	}

	var result struct {
		Sentiment  string  `json:"sentiment"`
		Score      float64 `json:"score"`
		Confidence float64 `json:"confidence"`
	}

	// Try to extract JSON from the response
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
