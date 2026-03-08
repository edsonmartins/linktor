package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	// Use a server that won't respond to avoid the background goroutine hanging
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		BaseURL: server.URL,
	})

	require.NotNil(t, client)
	assert.Equal(t, server.URL, client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.rateLimiter)
}

func TestNewClient_Defaults(t *testing.T) {
	// Test that defaults are applied (won't connect anywhere)
	client := NewClient(&ClientConfig{
		BaseURL: "http://127.0.0.1:1", // unreachable
	})
	assert.Equal(t, "http://127.0.0.1:1", client.baseURL)
}

func TestClient_DefaultModels(t *testing.T) {
	client := NewClient(&ClientConfig{BaseURL: "http://127.0.0.1:1"})
	models := client.DefaultModels()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "llama3.1:8b")
	assert.Contains(t, models, "mistral:7b")
}

func TestClient_DefaultModel(t *testing.T) {
	client := NewClient(&ClientConfig{BaseURL: "http://127.0.0.1:1"})
	assert.Equal(t, "llama3.1:8b", client.DefaultModel())
}

func TestClient_EmbeddingModels(t *testing.T) {
	client := NewClient(&ClientConfig{BaseURL: "http://127.0.0.1:1"})
	models := client.EmbeddingModels()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "nomic-embed-text")
}

func TestClient_Chat(t *testing.T) {
	mockResp := ChatResponse{
		Model:     "llama3.1:8b",
		CreatedAt: "2024-01-01T00:00:00Z",
		Message: ChatMessage{
			Role:    "assistant",
			Content: "Hello! How can I help?",
		},
		Done:            true,
		TotalDuration:   500000000,
		PromptEvalCount: 10,
		EvalCount:       8,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}

		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req ChatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "llama3.1:8b", req.Model)
		assert.False(t, req.Stream) // stream should be forced to false

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		BaseURL: server.URL,
	})
	// Wait briefly for the background availability check to finish
	time.Sleep(100 * time.Millisecond)

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model: "llama3.1:8b",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "llama3.1:8b", resp.Model)
	assert.Equal(t, "Hello! How can I help?", resp.Message.Content)
	assert.Equal(t, "assistant", resp.Message.Role)
	assert.True(t, resp.Done)
	assert.Equal(t, 10, resp.PromptEvalCount)
	assert.Equal(t, 8, resp.EvalCount)
}

func TestClient_Chat_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "nonexistent",
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestClient_CreateEmbedding(t *testing.T) {
	mockResp := EmbeddingResponse{
		Embedding: []float64{0.1, 0.2, 0.3, 0.4},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{}})
			return
		}

		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/embeddings", r.URL.Path)

		var req EmbeddingRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "nomic-embed-text", req.Model)
		assert.Equal(t, "test text", req.Prompt)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	resp, err := client.CreateEmbedding(context.Background(), &EmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: "test text",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Embedding, 4)
	assert.Equal(t, 0.1, resp.Embedding[0])
}

func TestClient_ListModels(t *testing.T) {
	mockResp := TagsResponse{
		Models: []ModelInfo{
			{Name: "llama3.1:8b", Size: 4000000000},
			{Name: "mistral:7b", Size: 3500000000},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/tags", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	models, err := client.ListModels(context.Background())

	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "llama3.1:8b", models[0].Name)
	assert.Equal(t, "mistral:7b", models[1].Name)
}

func TestClient_ListModels_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{BaseURL: server.URL})
	time.Sleep(100 * time.Millisecond)

	models, err := client.ListModels(context.Background())

	assert.Nil(t, models)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestClient_IsConfigured(t *testing.T) {
	t.Run("available when server responds", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TagsResponse{Models: []ModelInfo{{Name: "llama3.1:8b"}}})
		}))
		defer server.Close()

		client := NewClient(&ClientConfig{BaseURL: server.URL})
		// Wait for background check
		time.Sleep(200 * time.Millisecond)

		assert.True(t, client.IsConfigured())
	})

	t.Run("unavailable when server unreachable", func(t *testing.T) {
		client := NewClient(&ClientConfig{BaseURL: "http://127.0.0.1:1"})
		// Wait briefly - background check should fail quickly or timeout
		time.Sleep(200 * time.Millisecond)

		assert.False(t, client.IsConfigured())
	})
}
