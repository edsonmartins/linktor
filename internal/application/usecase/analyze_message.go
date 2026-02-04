package usecase

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// AnalyzeMessageInput represents input for message analysis
type AnalyzeMessageInput struct {
	MessageID      string
	ConversationID string
	TenantID       string
	Content        string
	ChannelID      string
}

// AnalyzeMessageOutput represents the result of message analysis
type AnalyzeMessageOutput struct {
	Intent         *entity.Intent         `json:"intent,omitempty"`
	Sentiment      entity.Sentiment       `json:"sentiment"`
	ShouldEscalate bool                   `json:"should_escalate"`
	EscalateReason string                 `json:"escalate_reason,omitempty"`
	Bot            *entity.Bot            `json:"bot,omitempty"`
	Keywords       []string               `json:"keywords,omitempty"`
}

// AnalyzeMessageUseCase handles message analysis for AI processing
type AnalyzeMessageUseCase struct {
	botRepo        repository.BotRepository
	contextService *service.ConversationContextService
	intentService  *service.IntentService
	producer       *nats.Producer
}

// NewAnalyzeMessageUseCase creates a new analyze message use case
func NewAnalyzeMessageUseCase(
	botRepo repository.BotRepository,
	contextService *service.ConversationContextService,
	intentService *service.IntentService,
	producer *nats.Producer,
) *AnalyzeMessageUseCase {
	return &AnalyzeMessageUseCase{
		botRepo:        botRepo,
		contextService: contextService,
		intentService:  intentService,
		producer:       producer,
	}
}

// Execute analyzes an incoming message and determines how to handle it
func (uc *AnalyzeMessageUseCase) Execute(ctx context.Context, input *AnalyzeMessageInput) (*AnalyzeMessageOutput, error) {
	output := &AnalyzeMessageOutput{
		Sentiment: entity.SentimentNeutral,
	}

	// Find bot for this channel
	bot, err := uc.botRepo.FindByChannel(ctx, input.ChannelID)
	if err != nil {
		if errors.IsNotFound(err) {
			// No bot assigned to this channel
			return output, nil
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find bot for channel")
	}

	if !bot.IsActive() {
		// Bot is not active
		return output, nil
	}

	output.Bot = bot

	// Get or create conversation context
	convContext, err := uc.contextService.GetOrCreate(ctx, input.ConversationID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get conversation context")
	}

	// Set bot in context if not already set
	if convContext.BotID == nil {
		if err := uc.contextService.SetBot(ctx, input.ConversationID, bot.ID); err != nil {
			// Log but continue
		}
	}

	// Add user message to context window
	if err := uc.contextService.AddUserMessage(ctx, input.ConversationID, input.Content, input.MessageID); err != nil {
		// Log but continue
	}

	// Analyze message
	analysis, err := uc.intentService.AnalyzeMessage(ctx, input.Content, bot.Provider, bot.Config.EnabledIntents)
	if err != nil {
		// Log error but continue with default values
	} else {
		if analysis.Intent != nil && analysis.Intent.Intent != nil {
			output.Intent = analysis.Intent.Intent

			// Update context with intent
			if err := uc.contextService.UpdateIntent(ctx, input.ConversationID, output.Intent); err != nil {
				// Log but continue
			}
		}

		if analysis.Sentiment != nil {
			output.Sentiment = analysis.Sentiment.Sentiment

			// Update context with sentiment
			if err := uc.contextService.UpdateSentiment(ctx, input.ConversationID, output.Sentiment); err != nil {
				// Log but continue
			}
		}
	}

	// Extract keywords for escalation check
	output.Keywords = service.ExtractKeywords(input.Content)

	// Check if should escalate
	shouldEscalate, rule := uc.intentService.ShouldEscalate(analysis, bot.Config.EscalationRules)
	if !shouldEscalate && len(bot.Config.EscalationRules) > 0 {
		// Also check keyword-based escalation
		shouldEscalate, rule = uc.checkKeywordEscalation(output.Keywords, bot.Config.EscalationRules)
	}

	// Check if user explicitly requested escalation
	if !shouldEscalate && output.Intent != nil && output.Intent.Name == "escalate" {
		shouldEscalate = true
		output.EscalateReason = "User requested to talk to a human"
	}

	if shouldEscalate {
		output.ShouldEscalate = true
		if rule != nil && output.EscalateReason == "" {
			output.EscalateReason = formatEscalationReason(rule)
		}
	}

	// Publish analysis event
	uc.publishAnalysisEvent(ctx, input, output)

	return output, nil
}

// checkKeywordEscalation checks if any keywords trigger escalation
func (uc *AnalyzeMessageUseCase) checkKeywordEscalation(keywords []string, rules []entity.EscalationRule) (bool, *entity.EscalationRule) {
	for _, rule := range rules {
		if rule.Condition == entity.EscalationConditionKeyword {
			for _, kw := range keywords {
				if kw == rule.Value {
					return true, &rule
				}
			}
		}
	}
	return false, nil
}

func formatEscalationReason(rule *entity.EscalationRule) string {
	switch rule.Condition {
	case entity.EscalationConditionLowConfidence:
		return "Low confidence in AI response"
	case entity.EscalationConditionSentiment:
		return "Negative sentiment detected"
	case entity.EscalationConditionKeyword:
		return "Escalation keyword detected: " + rule.Value
	case entity.EscalationConditionIntent:
		return "Escalation intent detected: " + rule.Value
	case entity.EscalationConditionUserRequest:
		return "User requested human assistance"
	default:
		return "Escalation rule triggered"
	}
}

func (uc *AnalyzeMessageUseCase) publishAnalysisEvent(ctx context.Context, input *AnalyzeMessageInput, output *AnalyzeMessageOutput) {
	payload := map[string]interface{}{
		"message_id":      input.MessageID,
		"conversation_id": input.ConversationID,
		"sentiment":       string(output.Sentiment),
		"should_escalate": output.ShouldEscalate,
	}

	if output.Intent != nil {
		payload["intent"] = output.Intent.Name
		payload["confidence"] = output.Intent.Confidence
	}

	if output.Bot != nil {
		payload["bot_id"] = output.Bot.ID
	}

	if output.EscalateReason != "" {
		payload["escalate_reason"] = output.EscalateReason
	}

	event := &nats.Event{
		Type:      nats.EventMessageAnalyzed,
		TenantID:  input.TenantID,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	uc.producer.PublishEvent(ctx, event)
}
