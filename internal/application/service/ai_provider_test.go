package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAIProvider implements AIProvider for testing
type testAIProvider struct {
	name      entity.AIProviderType
	available bool
	models    []string
}

func (m *testAIProvider) Name() entity.AIProviderType { return m.name }
func (m *testAIProvider) Models() []string            { return m.models }
func (m *testAIProvider) DefaultModel() string {
	if len(m.models) > 0 {
		return m.models[0]
	}
	return "default"
}
func (m *testAIProvider) IsAvailable() bool { return m.available }
func (m *testAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{Content: "test response", Model: m.models[0]}, nil
}
func (m *testAIProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return &EmbeddingResponse{Embedding: []float64{0.1, 0.2}}, nil
}
func (m *testAIProvider) ClassifyIntent(ctx context.Context, req *IntentClassificationRequest) (*entity.IntentResult, error) {
	return &entity.IntentResult{}, nil
}
func (m *testAIProvider) AnalyzeSentiment(ctx context.Context, req *SentimentAnalysisRequest) (*entity.SentimentResult, error) {
	return &entity.SentimentResult{}, nil
}

func TestNewAIProviderFactory(t *testing.T) {
	factory := NewAIProviderFactory()
	assert.NotNil(t, factory)
	assert.Empty(t, factory.List())
}

func TestAIProviderFactory_Register(t *testing.T) {
	factory := NewAIProviderFactory()

	factory.Register(&testAIProvider{
		name:      entity.AIProviderOpenAI,
		available: true,
		models:    []string{"gpt-4"},
	})

	assert.Len(t, factory.List(), 1)
	assert.Contains(t, factory.List(), entity.AIProviderOpenAI)
}

func TestAIProviderFactory_Get(t *testing.T) {
	factory := NewAIProviderFactory()

	factory.Register(&testAIProvider{
		name:      entity.AIProviderOpenAI,
		available: true,
		models:    []string{"gpt-4"},
	})

	t.Run("existing provider", func(t *testing.T) {
		provider, err := factory.Get(entity.AIProviderOpenAI)
		require.NoError(t, err)
		assert.Equal(t, entity.AIProviderOpenAI, provider.Name())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := factory.Get(entity.AIProviderAnthropic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("not available", func(t *testing.T) {
		factory.Register(&testAIProvider{
			name:      entity.AIProviderOllama,
			available: false,
			models:    []string{"llama3"},
		})
		_, err := factory.Get(entity.AIProviderOllama)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
	})
}

func TestAIProviderFactory_GetForBot(t *testing.T) {
	factory := NewAIProviderFactory()
	factory.Register(&testAIProvider{
		name:      entity.AIProviderOpenAI,
		available: true,
		models:    []string{"gpt-4"},
	})

	bot := &entity.Bot{Provider: entity.AIProviderOpenAI}
	provider, err := factory.GetForBot(bot)
	require.NoError(t, err)
	assert.Equal(t, entity.AIProviderOpenAI, provider.Name())
}

func TestAIProviderFactory_ListAvailable(t *testing.T) {
	factory := NewAIProviderFactory()

	factory.Register(&testAIProvider{name: entity.AIProviderOpenAI, available: true, models: []string{"gpt-4"}})
	factory.Register(&testAIProvider{name: entity.AIProviderAnthropic, available: true, models: []string{"claude-3"}})
	factory.Register(&testAIProvider{name: entity.AIProviderOllama, available: false, models: []string{"llama3"}})

	available := factory.ListAvailable()
	assert.Len(t, available, 2)
	assert.NotContains(t, available, entity.AIProviderOllama)
}

func TestAIProviderFactory_ListProviders(t *testing.T) {
	factory := NewAIProviderFactory()

	factory.Register(&testAIProvider{name: entity.AIProviderOpenAI, available: true, models: []string{"gpt-4", "gpt-3.5"}})
	factory.Register(&testAIProvider{name: entity.AIProviderOllama, available: false, models: []string{"llama3"}})

	infos := factory.ListProviders()
	assert.Len(t, infos, 2)

	for _, info := range infos {
		if info.Name == entity.AIProviderOpenAI {
			assert.True(t, info.Available)
			assert.Equal(t, "gpt-4", info.DefaultModel)
			assert.Len(t, info.Models, 2)
		}
		if info.Name == entity.AIProviderOllama {
			assert.False(t, info.Available)
		}
	}
}

func TestBuildPromptFromContext(t *testing.T) {
	t.Run("with system prompt and context", func(t *testing.T) {
		convCtx := &entity.ConversationContext{
			ContextWindow: []entity.ContextMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
		}

		messages := BuildPromptFromContext("You are helpful.", convCtx, "How are you?")
		require.Len(t, messages, 4)
		assert.Equal(t, "system", messages[0].Role)
		assert.Equal(t, "You are helpful.", messages[0].Content)
		assert.Equal(t, "user", messages[1].Role)
		assert.Equal(t, "Hello", messages[1].Content)
		assert.Equal(t, "assistant", messages[2].Role)
		assert.Equal(t, "user", messages[3].Role)
		assert.Equal(t, "How are you?", messages[3].Content)
	})

	t.Run("without system prompt", func(t *testing.T) {
		convCtx := &entity.ConversationContext{
			ContextWindow: []entity.ContextMessage{},
		}

		messages := BuildPromptFromContext("", convCtx, "Hi")
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Hi", messages[0].Content)
	})
}

func TestCalculateConfidence(t *testing.T) {
	t.Run("stop finish reason with intent", func(t *testing.T) {
		resp := &CompletionResponse{FinishReason: "stop"}
		intent := &entity.Intent{Confidence: 0.9}
		conf := CalculateConfidence(resp, intent)
		assert.InDelta(t, 0.9, conf, 0.01)
	})

	t.Run("length finish reason", func(t *testing.T) {
		resp := &CompletionResponse{FinishReason: "length"}
		intent := &entity.Intent{Confidence: 0.9}
		conf := CalculateConfidence(resp, intent)
		assert.InDelta(t, 0.72, conf, 0.01) // 0.9 * 0.8
	})

	t.Run("content_filter finish reason", func(t *testing.T) {
		resp := &CompletionResponse{FinishReason: "content_filter"}
		intent := &entity.Intent{Confidence: 0.9}
		conf := CalculateConfidence(resp, intent)
		assert.InDelta(t, 0.45, conf, 0.01) // 0.9 * 0.5
	})

	t.Run("no intent uses default", func(t *testing.T) {
		resp := &CompletionResponse{FinishReason: "stop"}
		conf := CalculateConfidence(resp, nil)
		assert.InDelta(t, 0.7, conf, 0.01)
	})
}
