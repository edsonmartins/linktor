package usecase

import (
	"testing"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAIResponseUseCase_CalculateConfidence(t *testing.T) {
	uc := &GenerateAIResponseUseCase{}

	t.Run("stop finish reason", func(t *testing.T) {
		resp := &service.CompletionResponse{FinishReason: "stop"}
		conf := uc.calculateConfidence(resp)
		assert.InDelta(t, 0.85, conf, 0.01)
	})

	t.Run("length finish reason", func(t *testing.T) {
		resp := &service.CompletionResponse{FinishReason: "length"}
		conf := uc.calculateConfidence(resp)
		assert.InDelta(t, 0.6, conf, 0.01)
	})

	t.Run("content_filter finish reason", func(t *testing.T) {
		resp := &service.CompletionResponse{FinishReason: "content_filter"}
		conf := uc.calculateConfidence(resp)
		assert.InDelta(t, 0.4, conf, 0.01)
	})

	t.Run("pre-calculated confidence takes precedence", func(t *testing.T) {
		resp := &service.CompletionResponse{FinishReason: "stop", Confidence: 0.95}
		conf := uc.calculateConfidence(resp)
		assert.InDelta(t, 0.95, conf, 0.01)
	})

	t.Run("unknown finish reason uses base", func(t *testing.T) {
		resp := &service.CompletionResponse{FinishReason: "unknown"}
		conf := uc.calculateConfidence(resp)
		assert.InDelta(t, 0.8, conf, 0.01)
	})
}

func TestGenerateAIResponseUseCase_BuildPromptWithKnowledge(t *testing.T) {
	uc := &GenerateAIResponseUseCase{}

	t.Run("empty results", func(t *testing.T) {
		result := uc.buildPromptWithKnowledge("Base prompt", nil)
		assert.Equal(t, "Base prompt", result)
	})

	t.Run("with results", func(t *testing.T) {
		results := []entity.SearchResult{
			{Item: &entity.KnowledgeItem{Question: "What is X?", Answer: "X is Y."}},
			{Item: &entity.KnowledgeItem{Question: "How to Z?", Answer: "Do A then B."}},
		}
		result := uc.buildPromptWithKnowledge("You are helpful.", results)
		assert.Contains(t, result, "You are helpful.")
		assert.Contains(t, result, "What is X?")
		assert.Contains(t, result, "X is Y.")
		assert.Contains(t, result, "How to Z?")
		assert.Contains(t, result, "knowledge base")
	})
}

func TestGenerateAIResponseOutput_Defaults(t *testing.T) {
	output := &GenerateAIResponseOutput{}
	assert.Empty(t, output.Response)
	assert.Equal(t, float64(0), output.Confidence)
	assert.False(t, output.ShouldEscalate)
}
