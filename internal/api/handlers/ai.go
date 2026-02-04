package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// AIHandler handles AI-related endpoints
type AIHandler struct {
	aiFactory          *service.AIProviderFactory
	intentService      *service.IntentService
	generateResponseUC *usecase.GenerateAIResponseUseCase
	analyzeMessageUC   *usecase.AnalyzeMessageUseCase
	escalateUC         *usecase.EscalateConversationUseCase
}

// NewAIHandler creates a new AI handler
func NewAIHandler(
	aiFactory *service.AIProviderFactory,
	intentService *service.IntentService,
	generateResponseUC *usecase.GenerateAIResponseUseCase,
	analyzeMessageUC *usecase.AnalyzeMessageUseCase,
	escalateUC *usecase.EscalateConversationUseCase,
) *AIHandler {
	return &AIHandler{
		aiFactory:          aiFactory,
		intentService:      intentService,
		generateResponseUC: generateResponseUC,
		analyzeMessageUC:   analyzeMessageUC,
		escalateUC:         escalateUC,
	}
}

// CompletionRequest represents a completion request
type CompletionRequest struct {
	Provider    string            `json:"provider" binding:"required"` // openai, anthropic, ollama
	Model       string            `json:"model" binding:"required"`
	Messages    []MessageRequest  `json:"messages" binding:"required"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float64           `json:"temperature"`
}

// MessageRequest represents a message in a completion request
type MessageRequest struct {
	Role    string `json:"role" binding:"required"` // system, user, assistant
	Content string `json:"content" binding:"required"`
}

// IntentClassifyRequest represents an intent classification request
type IntentClassifyRequest struct {
	Provider string   `json:"provider"` // openai, anthropic, ollama
	Message  string   `json:"message" binding:"required"`
	Intents  []string `json:"intents"` // optional list of intents to classify against
}

// SentimentAnalyzeRequest represents a sentiment analysis request
type SentimentAnalyzeRequest struct {
	Provider string `json:"provider"` // openai, anthropic, ollama
	Message  string `json:"message" binding:"required"`
}

// GenerateResponseRequest represents a generate response request
type GenerateResponseRequest struct {
	MessageID      string `json:"message_id" binding:"required"`
	ConversationID string `json:"conversation_id" binding:"required"`
	ChannelID      string `json:"channel_id" binding:"required"`
	Content        string `json:"content" binding:"required"`
	BotID          string `json:"bot_id"` // optional, will find from channel if not provided
}

// EscalateRequest represents an escalation request
type EscalateRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	ChannelID      string `json:"channel_id" binding:"required"`
	ContactID      string `json:"contact_id" binding:"required"`
	BotID          string `json:"bot_id"`
	Reason         string `json:"reason"`
	Priority       string `json:"priority"` // low, normal, high, urgent
}

// ListProviders returns all available AI providers
func (h *AIHandler) ListProviders(c *gin.Context) {
	providers := h.aiFactory.ListProviders()
	RespondSuccess(c, gin.H{"providers": providers})
}

// Complete generates a completion from an AI provider
func (h *AIHandler) Complete(c *gin.Context) {
	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Get provider
	provider, err := h.aiFactory.Get(entity.AIProviderType(req.Provider))
	if err != nil {
		RespondValidationError(c, "Provider not available: "+req.Provider, nil)
		return
	}

	// Convert messages
	messages := make([]service.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = service.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	// Set defaults
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// Generate completion
	completionReq := &service.CompletionRequest{
		Messages:    messages,
		Model:       req.Model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	response, err := provider.Complete(c.Request.Context(), completionReq)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, response)
}

// ClassifyIntent classifies the intent of a message
func (h *AIHandler) ClassifyIntent(c *gin.Context) {
	var req IntentClassifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Use default provider if not specified
	providerType := entity.AIProviderType(req.Provider)
	if providerType == "" {
		providerType = entity.AIProviderOpenAI
	}

	// Get available intents (use defaults if none provided)
	intents := req.Intents
	if len(intents) == 0 {
		intents = []string{
			"greeting", "farewell", "question", "complaint",
			"support_request", "sales_inquiry", "feedback",
			"escalation_request", "general",
		}
	}

	result, err := h.intentService.ClassifyIntent(c.Request.Context(), req.Message, providerType, intents)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

// AnalyzeSentiment analyzes the sentiment of a message
func (h *AIHandler) AnalyzeSentiment(c *gin.Context) {
	var req SentimentAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Use default provider if not specified
	providerType := entity.AIProviderType(req.Provider)
	if providerType == "" {
		providerType = entity.AIProviderOpenAI
	}

	result, err := h.intentService.AnalyzeSentiment(c.Request.Context(), req.Message, providerType)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

// GenerateResponse generates an AI response for a message
func (h *AIHandler) GenerateResponse(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req GenerateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &usecase.GenerateAIResponseInput{
		MessageID:      req.MessageID,
		ConversationID: req.ConversationID,
		TenantID:       tenantID,
		ChannelID:      req.ChannelID,
		Content:        req.Content,
	}

	response, err := h.generateResponseUC.Execute(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, response)
}

// AnalyzeMessage analyzes a message for intent and sentiment
func (h *AIHandler) AnalyzeMessage(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req struct {
		ConversationID string `json:"conversation_id" binding:"required"`
		ChannelID      string `json:"channel_id" binding:"required"`
		Content        string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &usecase.AnalyzeMessageInput{
		ConversationID: req.ConversationID,
		TenantID:       tenantID,
		ChannelID:      req.ChannelID,
		Content:        req.Content,
	}

	result, err := h.analyzeMessageUC.Execute(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

// Escalate escalates a conversation to human agents
func (h *AIHandler) Escalate(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req EscalateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Set default reason
	reason := req.Reason
	if reason == "" {
		reason = "Manual escalation requested"
	}

	// Set default priority
	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}

	input := &usecase.EscalateConversationInput{
		ConversationID: req.ConversationID,
		TenantID:       tenantID,
		ChannelID:      req.ChannelID,
		ContactID:      req.ContactID,
		BotID:          req.BotID,
		Reason:         reason,
		Priority:       priority,
		RequestedBy:    "api",
	}

	result, err := h.escalateUC.Execute(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, result)
}

// GetModels returns available models for a provider
func (h *AIHandler) GetModels(c *gin.Context) {
	providerName := c.Param("provider")
	if providerName == "" {
		RespondValidationError(c, "Provider is required", nil)
		return
	}

	provider, err := h.aiFactory.Get(entity.AIProviderType(providerName))
	if err != nil {
		RespondValidationError(c, "Provider not available: "+providerName, nil)
		return
	}

	models := provider.Models()
	RespondSuccess(c, gin.H{
		"provider": providerName,
		"models":   models,
	})
}
