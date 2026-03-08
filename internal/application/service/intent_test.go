package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIntentService(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		factory := NewAIProviderFactory()
		svc := NewIntentService(factory, nil)
		require.NotNil(t, svc)
		assert.Equal(t, 0.6, svc.config.ConfidenceThreshold)
		assert.True(t, svc.config.UseKeywordFallback)
		assert.Equal(t, CommonIntents, svc.config.DefaultIntents)
	})

	t.Run("custom config", func(t *testing.T) {
		factory := NewAIProviderFactory()
		cfg := &IntentServiceConfig{
			DefaultIntents:      []string{"greeting", "help"},
			ConfidenceThreshold: 0.8,
			UseKeywordFallback:  false,
		}
		svc := NewIntentService(factory, cfg)
		require.NotNil(t, svc)
		assert.Equal(t, 0.8, svc.config.ConfidenceThreshold)
		assert.False(t, svc.config.UseKeywordFallback)
		assert.Equal(t, []string{"greeting", "help"}, svc.config.DefaultIntents)
	})
}

func TestDefaultIntentConfig(t *testing.T) {
	cfg := DefaultIntentConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 0.6, cfg.ConfidenceThreshold)
	assert.True(t, cfg.UseKeywordFallback)
	assert.Equal(t, CommonIntents, cfg.DefaultIntents)
	assert.Contains(t, cfg.DefaultIntents, "greeting")
	assert.Contains(t, cfg.DefaultIntents, "unknown")
}

func TestIntentService_ClassifyByKeywords(t *testing.T) {
	factory := NewAIProviderFactory()
	svc := NewIntentService(factory, nil)
	ctx := context.Background()

	tests := []struct {
		name           string
		message        string
		expectedIntent string
	}{
		{"greeting - hello", "Hello there!", "greeting"},
		{"greeting - oi", "Oi, tudo bem?", "greeting"},
		{"goodbye - bye", "Bye, see you later", "goodbye"},
		{"goodbye - tchau", "Tchau!", "goodbye"},
		{"thanks - thank you", "Thank you so much!", "thanks"},
		{"thanks - obrigado", "Muito obrigado!", "thanks"},
		{"help - help me", "I need help with my order", "help"},
		{"help - ajuda", "Preciso de ajuda", "help"},
		{"complaint - issue", "I have an issue with my account", "complaint"},
		{"complaint - erro", "There is an error in my bill", "complaint"},
		{"escalate - human", "I want to talk to a human", "escalate"},
		{"escalate - atendente", "Quero falar com um atendente", "escalate"},
		{"unknown - random", "The weather is nice today", "unknown"},
		{"unknown - empty-ish", "xyz abc 123", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ClassifyIntent with empty provider type triggers keyword fallback
			result, err := svc.ClassifyIntent(ctx, tt.message, entity.AIProviderType("nonexistent"), nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Intent)
			assert.Equal(t, tt.expectedIntent, result.Intent.Name)

			if tt.expectedIntent == "unknown" {
				assert.Equal(t, 0.3, result.Intent.Confidence)
			} else {
				assert.Equal(t, 0.7, result.Intent.Confidence)
			}
		})
	}
}

func TestIntentService_AnalyzeSentimentByKeywords(t *testing.T) {
	factory := NewAIProviderFactory()
	svc := NewIntentService(factory, nil)
	ctx := context.Background()

	tests := []struct {
		name              string
		message           string
		expectedSentiment entity.Sentiment
	}{
		{"positive - great", "This is great, I love it!", entity.SentimentPositive},
		{"positive - excellent", "Excellent service, amazing!", entity.SentimentPositive},
		{"negative - terrible", "This is terrible and horrible", entity.SentimentNegative},
		{"negative - angry", "I hate this, I am angry", entity.SentimentNegative},
		{"neutral - plain", "I would like to check my order", entity.SentimentNeutral},
		{"neutral - no keywords", "Please send me the document", entity.SentimentNeutral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.AnalyzeSentiment(ctx, tt.message, entity.AIProviderType("nonexistent"))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedSentiment, result.Sentiment)
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		minCount int // minimum number of keywords expected
	}{
		{"escalation message", "I want to talk to a human agent", 2},
		{"complaint message", "I have a problem and an error", 2},
		{"help message", "I need help and support", 2},
		{"negative message", "This is terrible and awful", 2},
		{"no keywords", "The weather is nice today", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := ExtractKeywords(tt.message)
			assert.GreaterOrEqual(t, len(keywords), tt.minCount)
		})
	}
}

func TestIntentService_ShouldEscalate(t *testing.T) {
	factory := NewAIProviderFactory()
	svc := NewIntentService(factory, nil) // threshold = 0.6

	tests := []struct {
		name           string
		analysis       *MessageAnalysis
		rules          []entity.EscalationRule
		shouldEscalate bool
	}{
		{
			name:           "nil analysis",
			analysis:       nil,
			rules:          []entity.EscalationRule{{Condition: entity.EscalationConditionLowConfidence}},
			shouldEscalate: false,
		},
		{
			name: "low confidence triggers escalation",
			analysis: &MessageAnalysis{
				Intent: &entity.IntentResult{
					Intent: entity.NewIntent("unknown", 0.3),
				},
			},
			rules: []entity.EscalationRule{
				{Condition: entity.EscalationConditionLowConfidence, Action: entity.EscalationActionEscalate},
			},
			shouldEscalate: true,
		},
		{
			name: "high confidence does not escalate",
			analysis: &MessageAnalysis{
				Intent: &entity.IntentResult{
					Intent: entity.NewIntent("greeting", 0.9),
				},
			},
			rules: []entity.EscalationRule{
				{Condition: entity.EscalationConditionLowConfidence, Action: entity.EscalationActionEscalate},
			},
			shouldEscalate: false,
		},
		{
			name: "negative sentiment triggers escalation",
			analysis: &MessageAnalysis{
				Sentiment: &entity.SentimentResult{
					Sentiment: entity.SentimentNegative,
				},
			},
			rules: []entity.EscalationRule{
				{Condition: entity.EscalationConditionSentiment, Value: "negative", Action: entity.EscalationActionEscalate},
			},
			shouldEscalate: true,
		},
		{
			name: "positive sentiment does not match negative rule",
			analysis: &MessageAnalysis{
				Sentiment: &entity.SentimentResult{
					Sentiment: entity.SentimentPositive,
				},
			},
			rules: []entity.EscalationRule{
				{Condition: entity.EscalationConditionSentiment, Value: "negative", Action: entity.EscalationActionEscalate},
			},
			shouldEscalate: false,
		},
		{
			name: "intent match triggers escalation",
			analysis: &MessageAnalysis{
				Intent: &entity.IntentResult{
					Intent: entity.NewIntent("escalate", 0.8),
				},
			},
			rules: []entity.EscalationRule{
				{Condition: entity.EscalationConditionIntent, Value: "escalate", Action: entity.EscalationActionEscalate},
			},
			shouldEscalate: true,
		},
		{
			name:     "no rules",
			analysis: &MessageAnalysis{},
			rules:    []entity.EscalationRule{},
			shouldEscalate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldEscalate, rule := svc.ShouldEscalate(tt.analysis, tt.rules)
			assert.Equal(t, tt.shouldEscalate, shouldEscalate)
			if tt.shouldEscalate {
				assert.NotNil(t, rule)
			} else {
				assert.Nil(t, rule)
			}
		})
	}
}

func TestIntentService_ClassifyIntent_NoProvider(t *testing.T) {
	factory := NewAIProviderFactory()
	svc := NewIntentService(factory, nil)
	ctx := context.Background()

	result, err := svc.ClassifyIntent(ctx, "Hello", entity.AIProviderOpenAI, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Intent)
	assert.Equal(t, "greeting", result.Intent.Name)
}

func TestIntentService_AnalyzeSentiment_NoProvider(t *testing.T) {
	factory := NewAIProviderFactory()
	svc := NewIntentService(factory, nil)
	ctx := context.Background()

	result, err := svc.AnalyzeSentiment(ctx, "This is great!", entity.AIProviderOpenAI)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, entity.SentimentPositive, result.Sentiment)
}

func TestContainsIntent(t *testing.T) {
	intents := []string{"greeting", "help", "escalate"}

	tests := []struct {
		name   string
		target string
		found  bool
	}{
		{"found - greeting", "greeting", true},
		{"found - help", "help", true},
		{"found - escalate", "escalate", true},
		{"not found - unknown", "unknown", false},
		{"not found - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.found, containsIntent(intents, tt.target))
		})
	}
}
