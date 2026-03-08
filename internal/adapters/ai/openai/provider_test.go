package openai

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
		APIKey: "sk-test",
	})
	require.NotNil(t, p)
	assert.NotNil(t, p.client)
	assert.Equal(t, "gpt-4-turbo-preview", p.defaultModel)
	assert.Equal(t, "text-embedding-ada-002", p.embeddingModel)
}

func TestNewProvider_CustomModels(t *testing.T) {
	p := NewProvider(&ProviderConfig{
		APIKey:         "sk-test",
		DefaultModel:   "gpt-4",
		EmbeddingModel: "text-embedding-3-large",
	})
	assert.Equal(t, "gpt-4", p.defaultModel)
	assert.Equal(t, "text-embedding-3-large", p.embeddingModel)
}

func TestProvider_Name(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})
	assert.Equal(t, entity.AIProviderOpenAI, p.Name())
}

func TestProvider_Models(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})
	models := p.Models()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "gpt-4-turbo-preview")
}

func TestProvider_DefaultModel(t *testing.T) {
	p := NewProvider(&ProviderConfig{APIKey: "test"})
	assert.Equal(t, "gpt-4-turbo-preview", p.DefaultModel())
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
	mockResp := ChatCompletionResponse{
		ID:      "chatcmpl-prov",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-4-turbo-preview",
		Choices: []Choice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "The answer is 42.",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     15,
			CompletionTokens: 10,
			TotalTokens:      25,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "sk-test",
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
	assert.Equal(t, "gpt-4-turbo-preview", resp.Model)
	assert.Equal(t, 25, resp.TokensUsed)
	assert.Equal(t, 15, resp.PromptTokens)
	assert.Equal(t, 10, resp.CompTokens)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Greater(t, resp.LatencyMs, int64(-1))
}

func TestProvider_Complete_WithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Should have system message prepended
		require.NotEmpty(t, req.Messages)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "You are helpful.", req.Messages[0].Content)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			ID:      "chatcmpl-sys",
			Object:  "chat.completion",
			Model:   "gpt-4-turbo-preview",
			Choices: []Choice{{Index: 0, Message: ChatMessage{Role: "assistant", Content: "Sure!"}, FinishReason: "stop"}},
			Usage:   Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages:     []service.Message{{Role: "user", Content: "Hi"}},
		SystemPrompt: "You are helpful.",
	})

	require.NoError(t, err)
	assert.Equal(t, "Sure!", resp.Content)
}

func TestProvider_Complete_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(APIError{
			Error: struct {
				Message string      `json:"message"`
				Type    string      `json:"type"`
				Param   interface{} `json:"param"`
				Code    interface{} `json:"code"`
			}{
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages: []service.Message{{Role: "user", Content: "Hello"}},
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OpenAI completion failed")
}

func TestProvider_Embed(t *testing.T) {
	mockResp := EmbeddingResponse{
		Object: "list",
		Data: []EmbeddingData{
			{
				Object:    "embedding",
				Embedding: []float64{0.1, 0.2, 0.3},
				Index:     0,
			},
		},
		Model: "text-embedding-ada-002",
		Usage: Usage{
			PromptTokens: 3,
			TotalTokens:  3,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/embeddings", r.URL.Path)

		var req EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "test embedding text", req.Input)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	resp, err := p.Embed(context.Background(), &service.EmbeddingRequest{
		Text: "test embedding text",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Embedding, 3)
	assert.Equal(t, "text-embedding-ada-002", resp.Model)
	assert.Equal(t, 3, resp.TokensUsed)
}

func TestProvider_Embed_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			Error: struct {
				Message string      `json:"message"`
				Type    string      `json:"type"`
				Param   interface{} `json:"param"`
				Code    interface{} `json:"code"`
			}{
				Message: "Invalid input",
				Type:    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	resp, err := p.Embed(context.Background(), &service.EmbeddingRequest{
		Text: "",
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OpenAI embedding failed")
}
