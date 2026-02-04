package service

import (
	"context"
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// Common intents that bots typically handle
var CommonIntents = []string{
	"greeting",
	"goodbye",
	"thanks",
	"help",
	"complaint",
	"inquiry",
	"purchase",
	"refund",
	"support",
	"feedback",
	"escalate",
	"unknown",
}

// IntentServiceConfig holds configuration for intent classification
type IntentServiceConfig struct {
	DefaultIntents      []string
	ConfidenceThreshold float64
	UseKeywordFallback  bool
}

// DefaultIntentConfig returns default intent configuration
func DefaultIntentConfig() *IntentServiceConfig {
	return &IntentServiceConfig{
		DefaultIntents:      CommonIntents,
		ConfidenceThreshold: 0.6,
		UseKeywordFallback:  true,
	}
}

// IntentService handles intent classification and analysis
type IntentService struct {
	aiFactory *AIProviderFactory
	config    *IntentServiceConfig
}

// NewIntentService creates a new intent service
func NewIntentService(aiFactory *AIProviderFactory, config *IntentServiceConfig) *IntentService {
	if config == nil {
		config = DefaultIntentConfig()
	}
	return &IntentService{
		aiFactory: aiFactory,
		config:    config,
	}
}

// ClassifyIntent classifies the intent of a message
func (s *IntentService) ClassifyIntent(ctx context.Context, message string, providerType entity.AIProviderType, intents []string) (*entity.IntentResult, error) {
	// Use default intents if none provided
	if len(intents) == 0 {
		intents = s.config.DefaultIntents
	}

	// Get AI provider
	provider, err := s.aiFactory.Get(providerType)
	if err != nil {
		// Fall back to keyword-based classification if AI not available
		if s.config.UseKeywordFallback {
			return s.classifyByKeywords(message, intents), nil
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get AI provider")
	}

	// Classify using AI
	req := &IntentClassificationRequest{
		Message: message,
		Intents: intents,
	}

	result, err := provider.ClassifyIntent(ctx, req)
	if err != nil {
		// Fall back to keyword-based classification on error
		if s.config.UseKeywordFallback {
			return s.classifyByKeywords(message, intents), nil
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to classify intent")
	}

	return result, nil
}

// AnalyzeSentiment analyzes the sentiment of a message
func (s *IntentService) AnalyzeSentiment(ctx context.Context, message string, providerType entity.AIProviderType) (*entity.SentimentResult, error) {
	// Get AI provider
	provider, err := s.aiFactory.Get(providerType)
	if err != nil {
		// Fall back to keyword-based sentiment analysis
		return s.analyzeSentimentByKeywords(message), nil
	}

	// Analyze using AI
	req := &SentimentAnalysisRequest{
		Message: message,
	}

	result, err := provider.AnalyzeSentiment(ctx, req)
	if err != nil {
		// Fall back to keyword-based analysis on error
		return s.analyzeSentimentByKeywords(message), nil
	}

	return result, nil
}

// AnalyzeMessage performs full analysis including intent and sentiment
func (s *IntentService) AnalyzeMessage(ctx context.Context, message string, providerType entity.AIProviderType, intents []string) (*MessageAnalysis, error) {
	// Classify intent
	intentResult, err := s.ClassifyIntent(ctx, message, providerType, intents)
	if err != nil {
		return nil, err
	}

	// Analyze sentiment
	sentimentResult, err := s.AnalyzeSentiment(ctx, message, providerType)
	if err != nil {
		return nil, err
	}

	return &MessageAnalysis{
		Intent:    intentResult,
		Sentiment: sentimentResult,
	}, nil
}

// MessageAnalysis holds the complete analysis of a message
type MessageAnalysis struct {
	Intent    *entity.IntentResult    `json:"intent"`
	Sentiment *entity.SentimentResult `json:"sentiment"`
}

// ShouldEscalate determines if a conversation should be escalated based on analysis
func (s *IntentService) ShouldEscalate(analysis *MessageAnalysis, rules []entity.EscalationRule) (bool, *entity.EscalationRule) {
	if analysis == nil {
		return false, nil
	}

	for _, rule := range rules {
		switch rule.Condition {
		case entity.EscalationConditionLowConfidence:
			if analysis.Intent != nil && analysis.Intent.Intent != nil {
				if analysis.Intent.Intent.Confidence < s.config.ConfidenceThreshold {
					return true, &rule
				}
			}

		case entity.EscalationConditionSentiment:
			if analysis.Sentiment != nil {
				if string(analysis.Sentiment.Sentiment) == rule.Value {
					return true, &rule
				}
			}

		case entity.EscalationConditionIntent:
			if analysis.Intent != nil && analysis.Intent.Intent != nil {
				if analysis.Intent.Intent.Name == rule.Value {
					return true, &rule
				}
			}

		case entity.EscalationConditionKeyword:
			// Keyword check would need the original message
			// This is handled in the bot service
		}
	}

	return false, nil
}

// Keyword-based fallback methods

var greetingKeywords = []string{"hi", "hello", "hey", "good morning", "good afternoon", "good evening", "ola", "oi"}
var goodbyeKeywords = []string{"bye", "goodbye", "see you", "tchau", "adeus", "até"}
var thanksKeywords = []string{"thank", "thanks", "obrigado", "obrigada", "valeu", "agradeço"}
var helpKeywords = []string{"help", "ajuda", "socorro", "support", "suporte", "assist"}
var complaintKeywords = []string{"complaint", "reclamação", "problema", "issue", "bug", "error", "erro"}
var escalateKeywords = []string{"human", "agent", "atendente", "pessoa", "humano", "falar com alguém", "talk to someone"}

func (s *IntentService) classifyByKeywords(message string, intents []string) *entity.IntentResult {
	lower := strings.ToLower(message)

	// Check each intent category
	intentChecks := map[string][]string{
		"greeting":  greetingKeywords,
		"goodbye":   goodbyeKeywords,
		"thanks":    thanksKeywords,
		"help":      helpKeywords,
		"complaint": complaintKeywords,
		"escalate":  escalateKeywords,
	}

	for intent, keywords := range intentChecks {
		// Only check if this intent is in the allowed list
		if !containsIntent(intents, intent) {
			continue
		}

		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return &entity.IntentResult{
					Intent: entity.NewIntent(intent, 0.7),
				}
			}
		}
	}

	// Default to unknown with low confidence
	return &entity.IntentResult{
		Intent: entity.NewIntent("unknown", 0.3),
	}
}

var positiveKeywords = []string{"great", "awesome", "excellent", "good", "love", "happy", "thanks", "perfect", "amazing", "wonderful", "ótimo", "excelente", "maravilhoso", "feliz"}
var negativeKeywords = []string{"bad", "terrible", "awful", "hate", "angry", "frustrated", "disappointed", "horrible", "worst", "ruim", "péssimo", "horrível", "frustrado", "decepcionado", "raiva"}

func (s *IntentService) analyzeSentimentByKeywords(message string) *entity.SentimentResult {
	lower := strings.ToLower(message)

	positiveScore := 0
	negativeScore := 0

	for _, kw := range positiveKeywords {
		if strings.Contains(lower, kw) {
			positiveScore++
		}
	}

	for _, kw := range negativeKeywords {
		if strings.Contains(lower, kw) {
			negativeScore++
		}
	}

	if positiveScore > negativeScore {
		return &entity.SentimentResult{
			Sentiment:  entity.SentimentPositive,
			Score:      float64(positiveScore) / float64(positiveScore+negativeScore+1),
			Confidence: 0.6,
		}
	} else if negativeScore > positiveScore {
		return &entity.SentimentResult{
			Sentiment:  entity.SentimentNegative,
			Score:      -float64(negativeScore) / float64(positiveScore+negativeScore+1),
			Confidence: 0.6,
		}
	}

	return &entity.SentimentResult{
		Sentiment:  entity.SentimentNeutral,
		Score:      0,
		Confidence: 0.5,
	}
}

func containsIntent(intents []string, target string) bool {
	for _, i := range intents {
		if i == target {
			return true
		}
	}
	return false
}

// ExtractKeywords extracts keywords that might trigger escalation
func ExtractKeywords(message string) []string {
	lower := strings.ToLower(message)
	var found []string

	allKeywords := append(append(append(escalateKeywords, complaintKeywords...), negativeKeywords...), helpKeywords...)

	for _, kw := range allKeywords {
		if strings.Contains(lower, kw) {
			found = append(found, kw)
		}
	}

	return found
}
