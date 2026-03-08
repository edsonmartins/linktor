package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOllamaTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/tags":
			json.NewEncoder(w).Encode(TagsResponse{
				Models: []ModelInfo{
					{Name: "llama3.1:8b", Size: 4000000000},
					{Name: "mistral:7b", Size: 3500000000},
				},
			})

		case "/chat":
			var req ChatRequest
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(ChatResponse{
				Model:     req.Model,
				CreatedAt: "2024-01-01T00:00:00Z",
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Test response from Ollama.",
				},
				Done:            true,
				TotalDuration:   300000000,
				PromptEvalCount: 12,
				EvalCount:       6,
			})

		case "/embeddings":
			json.NewEncoder(w).Encode(EmbeddingResponse{
				Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestNewProvider(t *testing.T) {
	server := newOllamaTestServer(t)
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		BaseURL: server.URL,
	})
	require.NotNil(t, p)
	assert.NotNil(t, p.client)
	assert.Equal(t, "llama3.1:8b", p.defaultModel)
	assert.Equal(t, "nomic-embed-text", p.embeddingModel)
}

func TestNewProvider_CustomModels(t *testing.T) {
	server := newOllamaTestServer(t)
	defer server.Close()

	p := NewProvider(&ProviderConfig{
		BaseURL:        server.URL,
		DefaultModel:   "mistral:7b",
		EmbeddingModel: "mxbai-embed-large",
	})
	assert.Equal(t, "mistral:7b", p.defaultModel)
	assert.Equal(t, "mxbai-embed-large", p.embeddingModel)
}

func TestProvider_Name(t *testing.T) {
	server := newOllamaTestServer(t)
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	assert.Equal(t, entity.AIProviderOllama, p.Name())
}

func TestProvider_DefaultModel(t *testing.T) {
	server := newOllamaTestServer(t)
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	assert.Equal(t, "llama3.1:8b", p.DefaultModel())
}

func TestProvider_IsAvailable(t *testing.T) {
	t.Run("available when server responds at /api/tags", func(t *testing.T) {
		server := newOllamaTestServer(t)
		defer server.Close()

		p := NewProvider(&ProviderConfig{BaseURL: server.URL})
		// Wait for background availability check
		time.Sleep(200 * time.Millisecond)

		// IsAvailable calls RefreshAvailability which calls checkAvailability
		assert.True(t, p.IsAvailable())
	})

	t.Run("unavailable when server unreachable", func(t *testing.T) {
		p := NewProvider(&ProviderConfig{BaseURL: "http://127.0.0.1:1"})
		// IsAvailable refreshes and the unreachable server will fail
		assert.False(t, p.IsAvailable())
	})
}

func TestProvider_Models(t *testing.T) {
	server := newOllamaTestServer(t)
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	// Wait for background check to complete
	time.Sleep(200 * time.Millisecond)

	models := p.Models()
	assert.NotEmpty(t, models)
	// Should return models from the server
	assert.Contains(t, models, "llama3.1:8b")
	assert.Contains(t, models, "mistral:7b")
}

func TestProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/tags":
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
		case "/chat":
			assert.Equal(t, "POST", r.Method)

			var req ChatRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "llama3.1:8b", req.Model)
			assert.False(t, req.Stream)

			json.NewEncoder(w).Encode(ChatResponse{
				Model:     "llama3.1:8b",
				CreatedAt: "2024-01-01T00:00:00Z",
				Message: ChatMessage{
					Role:    "assistant",
					Content: "The answer is 42.",
				},
				Done:            true,
				TotalDuration:   400000000,
				PromptEvalCount: 15,
				EvalCount:       10,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

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
	assert.Equal(t, "llama3.1:8b", resp.Model)
	assert.Equal(t, 25, resp.TokensUsed)
	assert.Equal(t, 15, resp.PromptTokens)
	assert.Equal(t, 10, resp.CompTokens)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Greater(t, resp.LatencyMs, int64(-1))
}

func TestProvider_Complete_WithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/tags" {
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}

		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify system message is included
		require.NotEmpty(t, req.Messages)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "Be concise.", req.Messages[0].Content)

		json.NewEncoder(w).Encode(ChatResponse{
			Model:   "llama3.1:8b",
			Message: ChatMessage{Role: "assistant", Content: "OK!"},
			Done:    true,
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages:     []service.Message{{Role: "user", Content: "Hi"}},
		SystemPrompt: "Be concise.",
	})

	require.NoError(t, err)
	assert.Equal(t, "OK!", resp.Content)
}

func TestProvider_Complete_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"model not loaded"}`))
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	resp, err := p.Complete(context.Background(), &service.CompletionRequest{
		Messages: []service.Message{{Role: "user", Content: "Hello"}},
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ollama completion failed")
}

func TestProvider_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/tags" {
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}

		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/embeddings", r.URL.Path)

		var req EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "nomic-embed-text", req.Model)
		assert.Equal(t, "test embedding", req.Prompt)

		json.NewEncoder(w).Encode(EmbeddingResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		})
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	resp, err := p.Embed(context.Background(), &service.EmbeddingRequest{
		Text: "test embedding",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Embedding, 3)
	assert.Equal(t, "nomic-embed-text", resp.Model)
	assert.Equal(t, 0, resp.TokensUsed) // Ollama doesn't report token usage for embeddings
}

func TestProvider_Embed_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	p := NewProvider(&ProviderConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	resp, err := p.Embed(context.Background(), &service.EmbeddingRequest{
		Text: "test",
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ollama embedding failed")
}
