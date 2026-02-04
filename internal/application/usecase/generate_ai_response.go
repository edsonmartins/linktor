package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// GenerateAIResponseInput represents input for AI response generation
type GenerateAIResponseInput struct {
	MessageID      string
	ConversationID string
	TenantID       string
	ChannelID      string
	Content        string
	Bot            *entity.Bot
}

// GenerateAIResponseOutput represents the result of AI response generation
type GenerateAIResponseOutput struct {
	Response       string               `json:"response"`
	Confidence     float64              `json:"confidence"`
	TokensUsed     int                  `json:"tokens_used"`
	LatencyMs      int64                `json:"latency_ms"`
	Model          string               `json:"model"`
	ShouldEscalate bool                 `json:"should_escalate"`
	EscalateReason string               `json:"escalate_reason,omitempty"`
	QuickReplies   []entity.QuickReply  `json:"quick_replies,omitempty"` // Interactive buttons
	FlowID         string               `json:"flow_id,omitempty"`       // Active flow if any
	FlowEnded      bool                 `json:"flow_ended,omitempty"`    // True if flow just ended
}

// KnowledgeSearchService interface for knowledge base search (optional)
type KnowledgeSearchService interface {
	Search(ctx context.Context, knowledgeBaseID, query string, limit int) ([]entity.SearchResult, error)
}

// GenerateAIResponseUseCase handles AI response generation
type GenerateAIResponseUseCase struct {
	aiFactory        *service.AIProviderFactory
	botRepo          repository.BotRepository
	aiResponseRepo   repository.AIResponseRepository
	contextService   *service.ConversationContextService
	knowledgeService KnowledgeSearchService
	producer         *nats.Producer
}

// NewGenerateAIResponseUseCase creates a new generate AI response use case
func NewGenerateAIResponseUseCase(
	aiFactory *service.AIProviderFactory,
	botRepo repository.BotRepository,
	aiResponseRepo repository.AIResponseRepository,
	contextService *service.ConversationContextService,
	knowledgeService KnowledgeSearchService,
	producer *nats.Producer,
) *GenerateAIResponseUseCase {
	return &GenerateAIResponseUseCase{
		aiFactory:        aiFactory,
		botRepo:          botRepo,
		aiResponseRepo:   aiResponseRepo,
		contextService:   contextService,
		knowledgeService: knowledgeService,
		producer:         producer,
	}
}

// Execute generates an AI response for a message
func (uc *GenerateAIResponseUseCase) Execute(ctx context.Context, input *GenerateAIResponseInput) (*GenerateAIResponseOutput, error) {
	output := &GenerateAIResponseOutput{}

	// Get bot if not provided
	bot := input.Bot
	if bot == nil {
		var err error
		bot, err = uc.botRepo.FindByChannel(ctx, input.ChannelID)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeNotFound, "no bot found for channel")
		}
	}

	if !bot.IsActive() {
		return nil, errors.New(errors.ErrCodeBadRequest, "bot is not active")
	}

	// Get AI provider
	provider, err := uc.aiFactory.Get(bot.Provider)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get AI provider")
	}

	// Build system prompt with knowledge base context
	systemPrompt := bot.Config.SystemPrompt
	if bot.Config.KnowledgeBaseID != nil && uc.knowledgeService != nil {
		// Search knowledge base for relevant context
		results, err := uc.knowledgeService.Search(ctx, *bot.Config.KnowledgeBaseID, input.Content, 3)
		if err == nil && len(results) > 0 {
			systemPrompt = uc.buildPromptWithKnowledge(systemPrompt, results)
		}
	}

	// Build messages from context
	contextSize := bot.Config.ContextWindowSize
	if contextSize == 0 {
		contextSize = 10
	}

	messages, err := uc.contextService.BuildMessagesForAI(
		ctx,
		input.ConversationID,
		systemPrompt,
		input.Content,
		contextSize,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to build context messages")
	}

	// Prepare completion request
	maxTokens := bot.Config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	temperature := bot.Config.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	completionReq := &service.CompletionRequest{
		Messages:    messages,
		Model:       bot.Model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Generate completion
	startTime := time.Now()
	completion, err := provider.Complete(ctx, completionReq)
	if err != nil {
		// Use fallback message if AI fails
		output.Response = bot.Config.FallbackMessage
		output.Confidence = 0.0
		output.ShouldEscalate = true
		output.EscalateReason = "AI generation failed: " + err.Error()
		return output, nil
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// Build output
	output.Response = completion.Content
	output.TokensUsed = completion.TokensUsed
	output.LatencyMs = latencyMs
	output.Model = completion.Model

	// Calculate confidence
	output.Confidence = uc.calculateConfidence(completion)

	// Check if confidence is too low
	if output.Confidence < bot.Config.ConfidenceThreshold {
		output.ShouldEscalate = true
		output.EscalateReason = "Low confidence response"
	}

	// Add assistant message to context
	if err := uc.contextService.AddAssistantMessage(ctx, input.ConversationID, output.Response, ""); err != nil {
		// Log but continue
	}

	// Save AI response for audit
	if err := uc.saveAIResponse(ctx, input, output, bot, messages); err != nil {
		// Log but continue
	}

	// Publish response event
	uc.publishResponseEvent(ctx, input, output, bot)

	return output, nil
}

// buildPromptWithKnowledge enhances the system prompt with knowledge base context
func (uc *GenerateAIResponseUseCase) buildPromptWithKnowledge(basePrompt string, results []entity.SearchResult) string {
	if len(results) == 0 {
		return basePrompt
	}

	knowledgeContext := "\n\nRelevant information from the knowledge base:\n"
	for i, result := range results {
		if result.Item != nil {
			knowledgeContext += "\n---\n"
			knowledgeContext += "Q: " + result.Item.Question + "\n"
			knowledgeContext += "A: " + result.Item.Answer + "\n"
			if i >= 2 {
				break
			}
		}
	}
	knowledgeContext += "\n---\n\nUse the above information to help answer the user's question if relevant."

	return basePrompt + knowledgeContext
}

// calculateConfidence calculates a confidence score for the response
func (uc *GenerateAIResponseUseCase) calculateConfidence(completion *service.CompletionResponse) float64 {
	confidence := 0.8 // Base confidence

	// Adjust based on finish reason
	switch completion.FinishReason {
	case "stop":
		// Normal completion
		confidence = 0.85
	case "length":
		// Response was cut off
		confidence = 0.6
	case "content_filter":
		// Content was filtered
		confidence = 0.4
	}

	// Use pre-calculated confidence if available
	if completion.Confidence > 0 {
		confidence = completion.Confidence
	}

	return confidence
}

// saveAIResponse saves the AI response for audit purposes
func (uc *GenerateAIResponseUseCase) saveAIResponse(
	ctx context.Context,
	input *GenerateAIResponseInput,
	output *GenerateAIResponseOutput,
	bot *entity.Bot,
	messages []service.Message,
) error {
	// Convert messages to map for storage
	promptData := make(map[string]interface{})
	promptData["messages"] = messages
	promptData["model"] = bot.Model
	promptData["temperature"] = bot.Config.Temperature
	promptData["max_tokens"] = bot.Config.MaxTokens

	aiResponse := entity.NewAIResponse(
		input.MessageID,
		bot.ID,
		promptData,
		output.Response,
		output.Confidence,
		output.TokensUsed,
		int(output.LatencyMs),
		output.Model,
	)
	aiResponse.ID = uuid.New().String()

	return uc.aiResponseRepo.Create(ctx, aiResponse)
}

// publishResponseEvent publishes the AI response event
func (uc *GenerateAIResponseUseCase) publishResponseEvent(
	ctx context.Context,
	input *GenerateAIResponseInput,
	output *GenerateAIResponseOutput,
	bot *entity.Bot,
) {
	event := &nats.Event{
		Type:     nats.EventBotResponse,
		TenantID: input.TenantID,
		Payload: map[string]interface{}{
			"message_id":      input.MessageID,
			"conversation_id": input.ConversationID,
			"bot_id":          bot.ID,
			"response":        output.Response,
			"confidence":      output.Confidence,
			"tokens_used":     output.TokensUsed,
			"latency_ms":      output.LatencyMs,
			"model":           output.Model,
			"should_escalate": output.ShouldEscalate,
		},
		Timestamp: time.Now(),
	}

	uc.producer.PublishEvent(ctx, event)
}
