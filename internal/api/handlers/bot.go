package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// BotHandler handles bot endpoints
type BotHandler struct {
	botService *service.BotServiceImpl
}

// NewBotHandler creates a new bot handler
func NewBotHandler(botService *service.BotServiceImpl) *BotHandler {
	return &BotHandler{
		botService: botService,
	}
}

// CreateBotRequest represents a create bot request
type CreateBotRequest struct {
	Name         string  `json:"name" binding:"required"`
	Type         string  `json:"type" binding:"required"` // customer_service, sales, faq
	Provider     string  `json:"provider" binding:"required"` // openai, anthropic, ollama
	Model        string  `json:"model" binding:"required"`
	SystemPrompt string  `json:"system_prompt"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
}

// UpdateBotRequest represents an update bot request
type UpdateBotRequest struct {
	Name            *string  `json:"name"`
	Model           *string  `json:"model"`
	SystemPrompt    *string  `json:"system_prompt"`
	Temperature     *float64 `json:"temperature"`
	MaxTokens       *int     `json:"max_tokens"`
	WelcomeMessage  *string  `json:"welcome_message"`
	FallbackMessage *string  `json:"fallback_message"`
}

// UpdateBotConfigRequest represents an update bot config request
type UpdateBotConfigRequest struct {
	SystemPrompt        *string                    `json:"system_prompt"`
	Temperature         *float64                   `json:"temperature"`
	MaxTokens           *int                       `json:"max_tokens"`
	ContextWindowSize   *int                       `json:"context_window_size"`
	WelcomeMessage      *string                    `json:"welcome_message"`
	FallbackMessage     *string                    `json:"fallback_message"`
	ConfidenceThreshold *float64                   `json:"confidence_threshold"`
	EscalationRules     []entity.EscalationRule    `json:"escalation_rules"`
	WorkingHours        *entity.WorkingHours       `json:"working_hours"`
	KnowledgeBaseID     *string                    `json:"knowledge_base_id"`
}

// AssignChannelRequest represents a channel assignment request
type AssignChannelRequest struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

// AddEscalationRuleRequest represents an escalation rule request
type AddEscalationRuleRequest struct {
	Condition string `json:"condition" binding:"required"` // low_confidence, sentiment, keyword, intent, user_request
	Value     string `json:"value"`
	Priority  string `json:"priority"` // high, urgent, normal
}

// TestBotRequest represents a test bot request
type TestBotRequest struct {
	Message string `json:"message" binding:"required"`
}

// List returns all bots for the tenant
func (h *BotHandler) List(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := &repository.ListParams{
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.DefaultQuery("sort_by", "created_at"),
		SortDir:  c.DefaultQuery("sort_dir", "desc"),
	}

	bots, total, err := h.botService.List(c.Request.Context(), tenantID, params)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondPaginated(c, bots, total, params.Page, params.PageSize)
}

// Create creates a new bot
func (h *BotHandler) Create(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.CreateBotInput{
		TenantID:     tenantID,
		Name:         req.Name,
		Type:         entity.BotType(req.Type),
		Provider:     entity.AIProviderType(req.Provider),
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
	}

	bot, err := h.botService.Create(c.Request.Context(), input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondCreated(c, bot)
}

// Get returns a bot by ID
func (h *BotHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	bot, err := h.botService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, bot)
}

// Update updates a bot
func (h *BotHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	var req UpdateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	input := &service.UpdateBotInput{
		Name:            req.Name,
		Model:           req.Model,
		SystemPrompt:    req.SystemPrompt,
		Temperature:     req.Temperature,
		MaxTokens:       req.MaxTokens,
		WelcomeMessage:  req.WelcomeMessage,
		FallbackMessage: req.FallbackMessage,
	}

	bot, err := h.botService.Update(c.Request.Context(), id, input)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, bot)
}

// Delete deletes a bot
func (h *BotHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	if err := h.botService.Delete(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondNoContent(c)
}

// Activate activates a bot
func (h *BotHandler) Activate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	if err := h.botService.Activate(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Bot activated"})
}

// Deactivate deactivates a bot
func (h *BotHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	if err := h.botService.Deactivate(c.Request.Context(), id); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Bot deactivated"})
}

// AssignChannel assigns a channel to a bot
func (h *BotHandler) AssignChannel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	var req AssignChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	if err := h.botService.AssignChannel(c.Request.Context(), id, req.ChannelID); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Channel assigned to bot"})
}

// UnassignChannel unassigns a channel from a bot
func (h *BotHandler) UnassignChannel(c *gin.Context) {
	botID := c.Param("id")
	channelID := c.Param("channelId")

	if botID == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}
	if channelID == "" {
		RespondValidationError(c, "Channel ID is required", nil)
		return
	}

	if err := h.botService.UnassignChannel(c.Request.Context(), botID, channelID); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Channel unassigned from bot"})
}

// UpdateConfig updates bot configuration
func (h *BotHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	var req UpdateBotConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	// Get existing bot to merge config
	bot, err := h.botService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	// Update config fields
	config := bot.Config
	if req.SystemPrompt != nil {
		config.SystemPrompt = *req.SystemPrompt
	}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.ContextWindowSize != nil {
		config.ContextWindowSize = *req.ContextWindowSize
	}
	if req.WelcomeMessage != nil {
		config.WelcomeMessage = req.WelcomeMessage
	}
	if req.FallbackMessage != nil {
		config.FallbackMessage = *req.FallbackMessage
	}
	if req.ConfidenceThreshold != nil {
		config.ConfidenceThreshold = *req.ConfidenceThreshold
	}
	if req.EscalationRules != nil {
		config.EscalationRules = req.EscalationRules
	}
	if req.WorkingHours != nil {
		config.WorkingHours = req.WorkingHours
	}
	if req.KnowledgeBaseID != nil {
		config.KnowledgeBaseID = req.KnowledgeBaseID
	}

	if err := h.botService.UpdateConfig(c.Request.Context(), id, config); err != nil {
		RespondError(c, err)
		return
	}

	// Return updated bot
	updatedBot, err := h.botService.GetByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, updatedBot)
}

// AddEscalationRule adds an escalation rule to a bot
func (h *BotHandler) AddEscalationRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	var req AddEscalationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	rule := entity.EscalationRule{
		Condition: entity.EscalationCondition(req.Condition),
		Value:     req.Value,
		Priority:  req.Priority,
	}

	if err := h.botService.AddEscalationRule(c.Request.Context(), id, rule); err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, gin.H{"message": "Escalation rule added"})
}

// Test tests a bot with a message
func (h *BotHandler) Test(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondValidationError(c, "Bot ID is required", nil)
		return
	}

	var req TestBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}

	response, err := h.botService.TestBot(c.Request.Context(), id, req.Message)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, response)
}

// RespondPaginated responds with paginated data
func RespondPaginated(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"pagination": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}
