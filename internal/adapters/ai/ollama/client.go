package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultBaseURL = "http://localhost:11434/api"
	defaultTimeout = 180 * time.Second // Longer timeout for local inference
)

// Client is an Ollama API client
type Client struct {
	httpClient  *http.Client
	baseURL     string
	rateLimiter *rate.Limiter
	mu          sync.RWMutex
	available   bool
}

// ClientConfig holds configuration for the Ollama client
type ClientConfig struct {
	BaseURL   string
	Timeout   time.Duration
	RateLimit int // requests per minute
}

// NewClient creates a new Ollama client
func NewClient(config *ClientConfig) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	rateLimit := config.RateLimit
	if rateLimit == 0 {
		rateLimit = 30 // Default 30 requests per minute for local
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:     baseURL,
		rateLimiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(rateLimit)), 1),
	}

	// Check if Ollama is available
	go client.checkAvailability()

	return client
}

// ChatRequest represents an Ollama chat request
type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	Stream    bool          `json:"stream"`
	Options   *Options      `json:"options,omitempty"`
	Format    string        `json:"format,omitempty"` // "json" for JSON output
	KeepAlive string        `json:"keep_alive,omitempty"`
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string   `json:"role"` // system, user, assistant
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // Base64 encoded images for multimodal
}

// Options represents model options
type Options struct {
	Temperature   float64 `json:"temperature,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"` // max tokens
	Stop          []string `json:"stop,omitempty"`
	Seed          int     `json:"seed,omitempty"`
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
}

// ChatResponse represents an Ollama chat response
type ChatResponse struct {
	Model              string      `json:"model"`
	CreatedAt          string      `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	TotalDuration      int64       `json:"total_duration"`
	LoadDuration       int64       `json:"load_duration"`
	PromptEvalCount    int         `json:"prompt_eval_count"`
	PromptEvalDuration int64       `json:"prompt_eval_duration"`
	EvalCount          int         `json:"eval_count"`
	EvalDuration       int64       `json:"eval_duration"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// TagsResponse represents the response from /api/tags
type TagsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ModelInfo represents model information
type ModelInfo struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
}

// Chat creates a chat completion
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	// Ensure stream is false for non-streaming
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result ChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// CreateEmbedding creates an embedding
func (c *Client) CreateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// ListModels lists available models
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result TagsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Models, nil
}

// checkAvailability checks if Ollama is available
func (c *Client) checkAvailability() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.ListModels(ctx)
	c.mu.Lock()
	c.available = err == nil
	c.mu.Unlock()
}

// IsConfigured returns true if Ollama is available
func (c *Client) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.available
}

// RefreshAvailability refreshes the availability status
func (c *Client) RefreshAvailability() {
	c.checkAvailability()
}

// DefaultModels returns a list of commonly available models
func (c *Client) DefaultModels() []string {
	return []string{
		"llama3.1:8b",
		"llama3.1:70b",
		"llama3.2:3b",
		"llama3.2:1b",
		"mistral:7b",
		"mixtral:8x7b",
		"codellama:7b",
		"gemma2:9b",
		"qwen2.5:7b",
		"phi3:mini",
	}
}

// DefaultModel returns the default model
func (c *Client) DefaultModel() string {
	return "llama3.1:8b"
}

// EmbeddingModels returns models that support embeddings
func (c *Client) EmbeddingModels() []string {
	return []string{
		"nomic-embed-text",
		"mxbai-embed-large",
		"all-minilm",
	}
}
