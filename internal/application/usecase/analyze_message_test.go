package usecase

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestFormatEscalationReason(t *testing.T) {
	tests := []struct {
		condition entity.EscalationCondition
		value     string
		expected  string
	}{
		{entity.EscalationConditionLowConfidence, "", "Low confidence in AI response"},
		{entity.EscalationConditionSentiment, "", "Negative sentiment detected"},
		{entity.EscalationConditionKeyword, "help", "Escalation keyword detected: help"},
		{entity.EscalationConditionIntent, "cancel", "Escalation intent detected: cancel"},
		{entity.EscalationConditionUserRequest, "", "User requested human assistance"},
		{"unknown", "", "Escalation rule triggered"},
	}

	for _, tt := range tests {
		t.Run(string(tt.condition), func(t *testing.T) {
			rule := &entity.EscalationRule{
				Condition: tt.condition,
				Value:     tt.value,
			}
			assert.Equal(t, tt.expected, formatEscalationReason(rule))
		})
	}
}

func TestCheckKeywordEscalation(t *testing.T) {
	uc := &AnalyzeMessageUseCase{}

	rules := []entity.EscalationRule{
		{Condition: entity.EscalationConditionKeyword, Value: "help"},
		{Condition: entity.EscalationConditionKeyword, Value: "urgent"},
		{Condition: entity.EscalationConditionSentiment, Value: "negative"}, // non-keyword rule
	}

	t.Run("matching keyword", func(t *testing.T) {
		shouldEscalate, rule := uc.checkKeywordEscalation([]string{"help", "question"}, rules)
		assert.True(t, shouldEscalate)
		assert.NotNil(t, rule)
		assert.Equal(t, "help", rule.Value)
	})

	t.Run("no matching keyword", func(t *testing.T) {
		shouldEscalate, rule := uc.checkKeywordEscalation([]string{"hello", "world"}, rules)
		assert.False(t, shouldEscalate)
		assert.Nil(t, rule)
	})

	t.Run("empty keywords", func(t *testing.T) {
		shouldEscalate, rule := uc.checkKeywordEscalation([]string{}, rules)
		assert.False(t, shouldEscalate)
		assert.Nil(t, rule)
	})

	t.Run("empty rules", func(t *testing.T) {
		shouldEscalate, rule := uc.checkKeywordEscalation([]string{"help"}, []entity.EscalationRule{})
		assert.False(t, shouldEscalate)
		assert.Nil(t, rule)
	})
}

func TestAnalyzeMessageOutput_Defaults(t *testing.T) {
	output := &AnalyzeMessageOutput{
		Sentiment: entity.SentimentNeutral,
	}
	assert.Equal(t, entity.SentimentNeutral, output.Sentiment)
	assert.False(t, output.ShouldEscalate)
	assert.Empty(t, output.EscalateReason)
	assert.Nil(t, output.Bot)
	assert.Nil(t, output.Intent)
}
