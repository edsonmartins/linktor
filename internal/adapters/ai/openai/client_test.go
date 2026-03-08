package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient(&ClientConfig{
		APIKey: "sk-test",
	})

	require.NotNil(t, client)
	assert.Equal(t, defaultBaseURL, client.baseURL)
	assert.Equal(t, "sk-test", client.apiKey)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.rateLimiter)
}

func TestNewClient_CustomConfig(t *testing.T) {
	client := NewClient(&ClientConfig{
		APIKey:  "sk-test",
		OrgID:   "org-123",
		BaseURL: "https://custom.openai.com/v1",
	})

	assert.Equal(t, "https://custom.openai.com/v1", client.baseURL)
	assert.Equal(t, "org-123", client.orgID)
}

func TestClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "sk-test", true},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&ClientConfig{APIKey: tt.apiKey})
			assert.Equal(t, tt.want, client.IsConfigured())
		})
	}
}

func TestClient_Models(t *testing.T) {
	client := NewClient(&ClientConfig{APIKey: "test"})
	models := client.Models()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "gpt-4-turbo-preview")
	assert.Contains(t, models, "gpt-3.5-turbo")
}

func TestClient_DefaultModel(t *testing.T) {
	client := NewClient(&ClientConfig{APIKey: "test"})
	assert.Equal(t, "gpt-4-turbo-preview", client.DefaultModel())
}

func TestClient_EmbeddingModels(t *testing.T) {
	client := NewClient(&ClientConfig{APIKey: "test"})
	models := client.EmbeddingModels()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "text-embedding-ada-002")
	assert.Contains(t, models, "text-embedding-3-small")
}

func TestClient_DefaultEmbeddingModel(t *testing.T) {
	client := NewClient(&ClientConfig{APIKey: "test"})
	assert.Equal(t, "text-embedding-ada-002", client.DefaultEmbeddingModel())
}

func TestClient_CreateChatCompletion(t *testing.T) {
	mockResp := ChatCompletionResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-4-turbo-preview",
		Choices: []Choice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Hello! How can I help you?",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     12,
			CompletionTokens: 8,
			TotalTokens:      20,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4-turbo-preview", req.Model)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		APIKey:  "sk-test-key",
		BaseURL: server.URL,
	})

	resp, err := client.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model: "gpt-4-turbo-preview",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "chatcmpl-123", resp.ID)
	require.Len(t, resp.Choices, 1)
	assert.Equal(t, "Hello! How can I help you?", resp.Choices[0].Message.Content)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
	assert.Equal(t, 20, resp.Usage.TotalTokens)
}

func TestClient_CreateChatCompletion_WithOrgHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "org-abc", r.Header.Get("OpenAI-Organization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			ID:      "chatcmpl-org",
			Object:  "chat.completion",
			Model:   "gpt-4",
			Choices: []Choice{{Index: 0, Message: ChatMessage{Role: "assistant", Content: "OK"}, FinishReason: "stop"}},
			Usage:   Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		})
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		APIKey:  "sk-test",
		OrgID:   "org-abc",
		BaseURL: server.URL,
	})

	resp, err := client.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "chatcmpl-org", resp.ID)
}

func TestClient_CreateChatCompletion_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIError{
			Error: struct {
				Message string      `json:"message"`
				Type    string      `json:"type"`
				Param   interface{} `json:"param"`
				Code    interface{} `json:"code"`
			}{
				Message: "Invalid API key",
				Type:    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		APIKey:  "bad-key",
		BaseURL: server.URL,
	})

	resp, err := client.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestClient_CreateEmbedding(t *testing.T) {
	mockResp := EmbeddingResponse{
		Object: "list",
		Data: []EmbeddingData{
			{
				Object:    "embedding",
				Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
				Index:     0,
			},
		},
		Model: "text-embedding-ada-002",
		Usage: Usage{
			PromptTokens: 5,
			TotalTokens:  5,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer sk-embed-key", r.Header.Get("Authorization"))

		var req EmbeddingRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "text-embedding-ada-002", req.Model)
		assert.Equal(t, "Hello world", req.Input)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		APIKey:  "sk-embed-key",
		BaseURL: server.URL,
	})

	resp, err := client.CreateEmbedding(context.Background(), &EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: "Hello world",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "text-embedding-ada-002", resp.Model)
	require.Len(t, resp.Data, 1)
	assert.Len(t, resp.Data[0].Embedding, 5)
	assert.Equal(t, 5, resp.Usage.TotalTokens)
}
