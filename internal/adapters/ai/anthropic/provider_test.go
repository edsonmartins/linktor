package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider(&ProviderConfig{
		APIKey: "test-key",
	})
	require.NotNil(t, p)
	assert.NotNil(t, p.client)
	assert.Equal(t, "claude-3-5-sonnet-20241022", p.defaultModel)
}

func TestNewProvider_CustomModel(t *testing.T) {
	p := NewProvider(&ProviderConfig{
		APIKey:       "test-key",
		DefaultModel: "claude-3-opus-20240229",
	})
	assert.Equal(t, "claude-3-opus-20240229", p.defaultModel)
}

func TestProvider_Name(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})
	assert.Equal(t, entity.AIProviderAnthropic, p.Name())
}

func TestProvider_Models(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})
	models := p.Models()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "claude-3-5-sonnet-20241022")
}

func TestProvider_DefaultModel(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})
	assert.Equal(t, "claude-3-5-sonnet-20241022", p.DefaultModel())
}

func TestProvider_IsAvailable(t *testing.T) {
	t.Run("available with key", func(t *testing.T) {
		p := NewProvider(&ProviderConfig{APIKey: "sk-test"})
		assert.True(t, p.IsAvailable())
	})

	t.Run("unavailable without key", func(t *testing.T) {
		p := NewProvider(&ProviderConfig{APIKey: ""})
		assert.False(t, p.IsAvailable())
	})
}

func TestProvider_Complete(t *testing.T) {
	mockResp := MessageResponse{
		ID:         "msg_abc",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-3-5-sonnet-20241022",
		StopReason: "end_turn",
		Content: []ContentBlock{
			{Type: "text", Text: "The answer is 42."},
		},
		Usage: Usage{
			InputTokens:  15,
			OutputTokens: 10,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages: []service.Message{
			{Role: "user", Content: "What is the meaning of life?"},
		},
		MaxTokens:   100,
		Temperature: 0.5,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "The answer is 42.", resp.Content)
	assert.Equal(t, "claude-3-5-sonnet-20241022", resp.Model)
	assert.Equal(t, 25, resp.TokensUsed)
	assert.Equal(t, 15, resp.PromptTokens)
	assert.Equal(t, 10, resp.CompTokens)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Greater(t, resp.LatencyMs, int64(-1))
}

func TestProvider_Complete_WithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req MessageRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "You are a helpful assistant.", req.System)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MessageResponse{
			ID:         "msg_sys",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Content:    []ContentBlock{{Type: "text", Text: "Sure!"}},
			Usage:      Usage{InputTokens: 5, OutputTokens: 3},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages:     []service.Message{{Role: "user", Content: "Hi"}},
		SystemPrompt: "You are a helpful assistant.",
	})

	require.NoError(t, err)
	assert.Equal(t, "Sure!", resp.Content)
}

func TestProvider_Complete_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{
			Type: "error",
			Error: ErrorDetail{
				Type:    "api_error",
				Message: "internal server error",
			},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages: []service.Message{{Role: "user", Content: "Hello"}},
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Anthropic completion failed")
}

func TestProvider_Complete_MaxTokensFinishReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MessageResponse{
			ID:         "msg_len",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "max_tokens",
			Content:    []ContentBlock{{Type: "text", Text: "Partial response..."}},
			Usage:      Usage{InputTokens: 10, OutputTokens: 100},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages: []service.Message{{Role: "user", Content: "Tell me a long story"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "length", resp.FinishReason)
}

func TestProvider_Embed(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})

	resp, err := p.Embed(context.Background(), &service.EmbeddingRequest{
		Text: "test text",
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support embeddings")
}
