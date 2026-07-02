package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAIProvider is a minimal service.AIProvider used to observe what the
// handler forwards to the provider (e.g. the capped max_tokens).
type fakeAIProvider struct {
	name    entity.AIProviderType
	models  []string
	lastReq *service.CompletionRequest
}

func (f *fakeAIProvider) Name() entity.AIProviderType { return f.name }
func (f *fakeAIProvider) Models() []string            { return f.models }
func (f *fakeAIProvider) DefaultModel() string        { return "" }
func (f *fakeAIProvider) IsAvailable() bool            { return true }

func (f *fakeAIProvider) Complete(_ context.Context, req *service.CompletionRequest) (*service.CompletionResponse, error) {
	f.lastReq = req
	return &service.CompletionResponse{Content: "ok", Model: req.Model}, nil
}

func (f *fakeAIProvider) Embed(context.Context, *service.EmbeddingRequest) (*service.EmbeddingResponse, error) {
	return &service.EmbeddingResponse{}, nil
}

func (f *fakeAIProvider) ClassifyIntent(context.Context, *service.IntentClassificationRequest) (*entity.IntentResult, error) {
	return &entity.IntentResult{}, nil
}

func (f *fakeAIProvider) AnalyzeSentiment(context.Context, *service.SentimentAnalysisRequest) (*entity.SentimentResult, error) {
	return &entity.SentimentResult{}, nil
}

func newAITestHandler(provider *fakeAIProvider) *AIHandler {
	factory := service.NewAIProviderFactory()
	if provider != nil {
		factory.Register(provider)
	}
	return &AIHandler{aiFactory: factory}
}

func doComplete(h *AIHandler, body CompletionRequest) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	raw, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/complete", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Complete(c)
	return w
}

func TestComplete_CapsMaxTokens(t *testing.T) {
	provider := &fakeAIProvider{name: entity.AIProviderOpenAI, models: []string{"gpt-4"}}
	h := newAITestHandler(provider)

	w := doComplete(h, CompletionRequest{
		Provider:  string(entity.AIProviderOpenAI),
		Model:     "gpt-4",
		Messages:  []MessageRequest{{Role: "user", Content: "hi"}},
		MaxTokens: 1_000_000, // absurd
	})

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, provider.lastReq)
	assert.Equal(t, maxCompletionTokens, provider.lastReq.MaxTokens, "max_tokens must be capped")
}

func TestComplete_RejectsDisallowedProvider(t *testing.T) {
	h := newAITestHandler(&fakeAIProvider{name: entity.AIProviderOpenAI, models: []string{"gpt-4"}})

	w := doComplete(h, CompletionRequest{
		Provider: "evil-provider",
		Model:    "gpt-4",
		Messages: []MessageRequest{{Role: "user", Content: "hi"}},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestComplete_RejectsUnknownModel(t *testing.T) {
	h := newAITestHandler(&fakeAIProvider{name: entity.AIProviderOpenAI, models: []string{"gpt-4"}})

	w := doComplete(h, CompletionRequest{
		Provider: string(entity.AIProviderOpenAI),
		Model:    "gpt-99-superleak",
		Messages: []MessageRequest{{Role: "user", Content: "hi"}},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestComplete_AllowsValidRequest(t *testing.T) {
	provider := &fakeAIProvider{name: entity.AIProviderOpenAI, models: []string{"gpt-4"}}
	h := newAITestHandler(provider)

	w := doComplete(h, CompletionRequest{
		Provider:  string(entity.AIProviderOpenAI),
		Model:     "gpt-4",
		Messages:  []MessageRequest{{Role: "user", Content: "hi"}},
		MaxTokens: 512,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 512, provider.lastReq.MaxTokens)
}
