package anthropic

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
		APIKey: "test-key",
	})

	require.NotNil(t, client)
	assert.Equal(t, defaultBaseURL, client.baseURL)
	assert.Equal(t, "test-key", client.apiKey)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.rateLimiter)
}

func TestNewClient_CustomConfig(t *testing.T) {
	client := NewClient(&ClientConfig{
		APIKey:  "test-key",
		BaseURL: "https://custom.api.com/v1",
	})

	assert.Equal(t, "https://custom.api.com/v1", client.baseURL)
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
	assert.Contains(t, models, "claude-3-5-sonnet-20241022")
}

func TestClient_DefaultModel(t *testing.T) {
	client := NewClient(&ClientConfig{APIKey: "test"})
	model := client.DefaultModel()
	assert.NotEmpty(t, model)
	assert.Equal(t, "claude-3-5-sonnet-20241022", model)
}

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage("user", "Hello")
	assert.Equal(t, "user", msg.Role)
	require.Len(t, msg.Content, 1)
	assert.Equal(t, "text", msg.Content[0].Type)
	assert.Equal(t, "Hello", msg.Content[0].Text)
}

func TestGetTextContent(t *testing.T) {
	resp := &MessageResponse{
		Content: []ContentBlock{
			{Type: "text", Text: "Hello there"},
		},
	}
	assert.Equal(t, "Hello there", GetTextContent(resp))
}

func TestGetTextContent_Empty(t *testing.T) {
	resp := &MessageResponse{
		Content: []ContentBlock{},
	}
	assert.Equal(t, "", GetTextContent(resp))
}

func TestClient_CreateMessage(t *testing.T) {
	mockResp := MessageResponse{
		ID:         "msg_123",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-3-5-sonnet-20241022",
		StopReason: "end_turn",
		Content: []ContentBlock{
			{Type: "text", Text: "Hi! How can I help?"},
		},
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 8,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, apiVersion, r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req MessageRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "claude-3-5-sonnet-20241022", req.Model)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})

	resp, err := client.CreateMessage(context.Background(), &MessageRequest{
		Model:     "claude-3-5-sonnet-20241022",
		Messages:  []Message{NewTextMessage("user", "Hello")},
		MaxTokens: 100,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "msg_123", resp.ID)
	assert.Equal(t, "assistant", resp.Role)
	assert.Equal(t, 10, resp.Usage.InputTokens)
	assert.Equal(t, 8, resp.Usage.OutputTokens)
	assert.Equal(t, "Hi! How can I help?", GetTextContent(resp))
}

func TestClient_CreateMessage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			Type: "error",
			Error: ErrorDetail{
				Type:    "invalid_request_error",
				Message: "model not found",
			},
		})
	}))
	defer server.Close()

	client := NewClient(&ClientConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})

	resp, err := client.CreateMessage(context.Background(), &MessageRequest{
		Model:     "invalid-model",
		Messages:  []Message{NewTextMessage("user", "Hello")},
		MaxTokens: 100,
	})

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}
