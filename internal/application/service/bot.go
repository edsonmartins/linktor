package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// CreateBotInput represents input for creating a bot
type CreateBotInput struct {
	TenantID     string
	Name         string
	Type         entity.BotType
	Provider     entity.AIProviderType
	Model        string
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
}

// UpdateBotInput represents input for updating a bot
type UpdateBotInput struct {
	Name            *string
	Model           *string
	SystemPrompt    *string
	Temperature     *float64
	MaxTokens       *int
	WelcomeMessage  *string
	FallbackMessage *string
}

// BotServiceImpl implements BotService interface
type BotServiceImpl struct {
	botRepo        repository.BotRepository
	channelRepo    repository.ChannelRepository
	contextRepo    repository.ConversationContextRepository
	contextService *ConversationContextService
	aiFactory      *AIProviderFactory
	flowEngine     *FlowEngineService
	vreService     *VREService // VRE for visual responses
}

// NewBotService creates a new bot service
func NewBotService(
	botRepo repository.BotRepository,
	channelRepo repository.ChannelRepository,
	contextRepo repository.ConversationContextRepository,
	contextService *ConversationContextService,
	aiFactory *AIProviderFactory,
	flowEngine *FlowEngineService,
) *BotServiceImpl {
	return &BotServiceImpl{
		botRepo:        botRepo,
		channelRepo:    channelRepo,
		contextRepo:    contextRepo,
		contextService: contextService,
		aiFactory:      aiFactory,
		flowEngine:     flowEngine,
	}
}

// SetVREService sets the VRE service for visual responses
func (s *BotServiceImpl) SetVREService(vreService *VREService) {
	s.vreService = vreService
}

// Create creates a new bot
func (s *BotServiceImpl) Create(ctx context.Context, input *CreateBotInput) (*entity.Bot, error) {
	// Validate provider is available
	if _, err := s.aiFactory.Get(input.Provider); err != nil {
		return nil, errors.New(errors.ErrCodeBadRequest, "AI provider not available: "+string(input.Provider))
	}

	bot := entity.NewBot(input.TenantID, input.Name, input.Type, input.Provider, input.Model)
	bot.ID = uuid.New().String()

	// Set optional config
	if input.SystemPrompt != "" {
		bot.Config.SystemPrompt = input.SystemPrompt
	}
	if input.Temperature > 0 {
		bot.Config.Temperature = input.Temperature
	}
	if input.MaxTokens > 0 {
		bot.Config.MaxTokens = input.MaxTokens
	}

	if err := s.botRepo.Create(ctx, bot); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create bot")
	}

	return bot, nil
}

// GetByID gets a bot by ID
func (s *BotServiceImpl) GetByID(ctx context.Context, id string) (*entity.Bot, error) {
	return s.botRepo.FindByID(ctx, id)
}

// List lists bots for a tenant
func (s *BotServiceImpl) List(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Bot, int64, error) {
	return s.botRepo.FindByTenant(ctx, tenantID, params)
}

// Update updates a bot
func (s *BotServiceImpl) Update(ctx context.Context, id string, input *UpdateBotInput) (*entity.Bot, error) {
	bot, err := s.botRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		bot.Name = *input.Name
	}
	if input.Model != nil {
		bot.Model = *input.Model
	}
	if input.SystemPrompt != nil {
		bot.Config.SystemPrompt = *input.SystemPrompt
	}
	if input.Temperature != nil {
		bot.Config.Temperature = *input.Temperature
	}
	if input.MaxTokens != nil {
		bot.Config.MaxTokens = *input.MaxTokens
	}
	if input.WelcomeMessage != nil {
		bot.Config.WelcomeMessage = input.WelcomeMessage
	}
	if input.FallbackMessage != nil {
		bot.Config.FallbackMessage = *input.FallbackMessage
	}

	bot.UpdatedAt = time.Now()

	if err := s.botRepo.Update(ctx, bot); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update bot")
	}

	return bot, nil
}

// Delete deletes a bot
func (s *BotServiceImpl) Delete(ctx context.Context, id string) error {
	return s.botRepo.Delete(ctx, id)
}

// Activate activates a bot
func (s *BotServiceImpl) Activate(ctx context.Context, id string) error {
	return s.botRepo.UpdateStatus(ctx, id, entity.BotStatusActive)
}

// Deactivate deactivates a bot
func (s *BotServiceImpl) Deactivate(ctx context.Context, id string) error {
	return s.botRepo.UpdateStatus(ctx, id, entity.BotStatusInactive)
}

// AssignChannel assigns a channel to a bot
func (s *BotServiceImpl) AssignChannel(ctx context.Context, botID, channelID string) error {
	// Verify bot exists
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return err
	}

	// Verify channel exists
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return err
	}

	// Verify same tenant
	if bot.TenantID != channel.TenantID {
		return errors.New(errors.ErrCodeForbidden, "bot and channel must belong to same tenant")
	}

	// Check if channel is already assigned to another bot
	existingBot, err := s.botRepo.FindByChannel(ctx, channelID)
	if err == nil && existingBot != nil && existingBot.ID != botID {
		// Unassign from existing bot first
		if err := s.botRepo.UnassignChannel(ctx, existingBot.ID, channelID); err != nil {
			return errors.Wrap(err, errors.ErrCodeInternal, "failed to unassign channel from existing bot")
		}
	}

	return s.botRepo.AssignChannel(ctx, botID, channelID)
}

// UnassignChannel unassigns a channel from a bot
func (s *BotServiceImpl) UnassignChannel(ctx context.Context, botID, channelID string) error {
	return s.botRepo.UnassignChannel(ctx, botID, channelID)
}

// GetBotForChannel returns the active bot assigned to a channel
func (s *BotServiceImpl) GetBotForChannel(ctx context.Context, channelID string) (*entity.Bot, error) {
	return s.botRepo.FindByChannel(ctx, channelID)
}

// ShouldBotHandle determines if a bot should handle a conversation
func (s *BotServiceImpl) ShouldBotHandle(ctx context.Context, conversation *entity.Conversation, bot *entity.Bot) (bool, error) {
	// Bot must be active
	if !bot.IsActive() {
		return false, nil
	}

	// Bot must be assigned to the channel
	if !bot.HasChannel(conversation.ChannelID) {
		return false, nil
	}

	// Check if conversation is already assigned to a human
	if conversation.AssignedUserID != nil {
		return false, nil
	}

	// Check working hours if configured
	if bot.Config.WorkingHours != nil && bot.Config.WorkingHours.Enabled {
		if !s.isWithinWorkingHours(bot.Config.WorkingHours) {
			return false, nil
		}
	}

	return true, nil
}

// ProcessMessage processes a message through the bot
func (s *BotServiceImpl) ProcessMessage(ctx context.Context, message *entity.Message, conversation *entity.Conversation, bot *entity.Bot) (*BotResponse, error) {
	// Get conversation context
	convContext, err := s.contextService.GetOrCreate(ctx, conversation.ID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get conversation context")
	}

	// Check if there's an active flow
	if s.flowEngine != nil && s.flowEngine.HasActiveFlow(convContext) {
		return s.processFlowMessage(ctx, message, conversation, bot, convContext)
	}

	// Check if a flow should be triggered
	if s.flowEngine != nil {
		if flow, triggered := s.flowEngine.CheckTrigger(ctx, conversation.TenantID, message.Content, convContext); triggered {
			return s.startFlow(ctx, flow, message, conversation, bot, convContext)
		}
	}

	// Process with AI
	return s.processAIMessage(ctx, message, conversation, bot, convContext)
}

// processFlowMessage processes a message through an active flow
func (s *BotServiceImpl) processFlowMessage(ctx context.Context, message *entity.Message, conversation *entity.Conversation, bot *entity.Bot, convContext *entity.ConversationContext) (*BotResponse, error) {
	result, err := s.flowEngine.ContinueFlow(ctx, conversation.TenantID, message.Content, convContext)
	if err != nil {
		// Flow error, fall back to AI
		return s.processAIMessage(ctx, message, conversation, bot, convContext)
	}

	// Update context
	if err := s.contextRepo.Update(ctx, convContext); err != nil {
		// Log but don't fail
	}

	// Convert flow result to bot response
	response := &BotResponse{
		Content:      result.Message,
		QuickReplies: result.QuickReplies,
		Confidence:   1.0, // Flows are deterministic
		FlowID:       s.flowEngine.GetActiveFlowID(convContext),
		FlowEnded:    result.FlowEnded,
	}

	// Process flow actions
	if len(result.Actions) > 0 {
		for _, action := range result.Actions {
			switch action.Type {
			case entity.FlowActionEscalate:
				response.ShouldEscalate = true
				if priority, ok := action.Config["priority"].(string); ok {
					response.EscalateReason = "Flow escalation: " + priority
				} else {
					response.EscalateReason = "Flow triggered escalation"
				}
			case entity.FlowActionTag:
				// Add tag action to response
				response.Actions = append(response.Actions, BotAction{
					Type:       "add_tag",
					Parameters: action.Config,
				})
			case entity.FlowActionAssign:
				response.Actions = append(response.Actions, BotAction{
					Type:       "assign",
					Parameters: action.Config,
				})
			}
		}
	}

	return response, nil
}

// startFlow starts a new flow execution
func (s *BotServiceImpl) startFlow(ctx context.Context, flow *entity.Flow, message *entity.Message, conversation *entity.Conversation, bot *entity.Bot, convContext *entity.ConversationContext) (*BotResponse, error) {
	result, err := s.flowEngine.StartFlow(ctx, flow, convContext)
	if err != nil {
		// Flow start error, fall back to AI
		return s.processAIMessage(ctx, message, conversation, bot, convContext)
	}

	// Update context with flow state
	if err := s.contextRepo.Update(ctx, convContext); err != nil {
		// Log but don't fail
	}

	// Convert flow result to bot response
	response := &BotResponse{
		Content:      result.Message,
		QuickReplies: result.QuickReplies,
		Confidence:   1.0,
		FlowID:       flow.ID,
		FlowEnded:    result.FlowEnded,
	}

	return response, nil
}

// processAIMessage processes a message through the AI
func (s *BotServiceImpl) processAIMessage(ctx context.Context, message *entity.Message, conversation *entity.Conversation, bot *entity.Bot, convContext *entity.ConversationContext) (*BotResponse, error) {
	// Get AI provider
	provider, err := s.aiFactory.Get(bot.Provider)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get AI provider")
	}

	// Build messages for completion
	messages, err := s.contextService.BuildMessagesForAI(
		ctx,
		conversation.ID,
		bot.Config.SystemPrompt,
		message.Content,
		bot.Config.ContextWindowSize,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to build messages")
	}

	// Get tools for the bot
	tools := bot.GetTools()

	// Generate completion
	startTime := time.Now()
	completion, err := provider.Complete(ctx, &CompletionRequest{
		Messages:    messages,
		Model:       bot.Model,
		MaxTokens:   bot.Config.MaxTokens,
		Temperature: bot.Config.Temperature,
		Tools:       tools,
		ToolChoice:  bot.Config.ToolChoice,
	})
	if err != nil {
		// Use fallback message
		return &BotResponse{
			Content:        bot.Config.FallbackMessage,
			Confidence:     0,
			ShouldEscalate: true,
			EscalateReason: "AI generation failed: " + err.Error(),
		}, nil
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// Check if AI made tool calls
	if len(completion.ToolCalls) > 0 {
		return s.handleToolCalls(ctx, completion, conversation, bot, convContext, latencyMs)
	}

	// Calculate confidence
	confidence := CalculateConfidence(completion, convContext.Intent)

	// Check escalation rules
	shouldEscalate, rule := bot.ShouldEscalate(confidence, string(convContext.Sentiment), ExtractKeywords(message.Content))

	response := &BotResponse{
		Content:    completion.Content,
		Confidence: confidence,
		TokensUsed: completion.TokensUsed,
		LatencyMs:  latencyMs,
	}

	if convContext.Intent != nil {
		response.Intent = convContext.Intent
	}
	response.Sentiment = convContext.Sentiment

	if shouldEscalate && rule != nil {
		response.ShouldEscalate = true
		response.EscalateReason = formatEscalationReason(rule)
	}

	// Check confidence threshold
	if confidence < bot.Config.ConfidenceThreshold {
		response.ShouldEscalate = true
		if response.EscalateReason == "" {
			response.EscalateReason = "Low confidence response"
		}
	}

	return response, nil
}

// handleToolCalls processes tool calls from the AI response
func (s *BotServiceImpl) handleToolCalls(ctx context.Context, completion *CompletionResponse, conversation *entity.Conversation, bot *entity.Bot, convContext *entity.ConversationContext, latencyMs int64) (*BotResponse, error) {
	// Process the first tool call (typically there's only one for visual tools)
	toolCall := completion.ToolCalls[0]

	// Find the tool definition
	tool := bot.GetToolByName(toolCall.Name)
	if tool == nil {
		// Tool not found, return text content if available
		return &BotResponse{
			Content:    completion.Content,
			Confidence: 0.5,
			TokensUsed: completion.TokensUsed,
			LatencyMs:  latencyMs,
		}, nil
	}

	// Check if it's a visual (VRE) tool
	if tool.IsVisual() && s.vreService != nil {
		return s.handleVisualToolCall(ctx, toolCall, tool, conversation, bot, completion.TokensUsed, latencyMs)
	}

	// For non-visual tools, return text content
	// Future: implement other tool types (data, text)
	return &BotResponse{
		Content:    completion.Content,
		Confidence: 0.9,
		TokensUsed: completion.TokensUsed,
		LatencyMs:  latencyMs,
	}, nil
}

// handleVisualToolCall processes a visual tool call using VRE
func (s *BotServiceImpl) handleVisualToolCall(ctx context.Context, toolCall *entity.ToolCall, tool *entity.Tool, conversation *entity.Conversation, bot *entity.Bot, tokensUsed int, latencyMs int64) (*BotResponse, error) {
	// Determine channel type from conversation
	channel := entity.VREChannelWhatsApp // default
	// TODO: get actual channel type from conversation.ChannelID

	// Build render request
	renderReq := &entity.RenderRequest{
		TenantID:   conversation.TenantID,
		TemplateID: tool.GetLinktorTemplate(),
		Data:       toolCall.Arguments,
		Channel:    channel,
	}

	// Render the template
	renderResp, err := s.vreService.Render(ctx, renderReq)
	if err != nil {
		// Failed to render, return error message
		return &BotResponse{
			Content:        "Desculpe, não consegui gerar a visualização. Por favor, tente novamente.",
			Confidence:     0.5,
			ShouldEscalate: false,
			TokensUsed:     tokensUsed,
			LatencyMs:      latencyMs,
		}, nil
	}

	// Build visual response
	return &BotResponse{
		Content:      "", // No text content for visual responses
		IsVisual:     true,
		ImageBase64:  renderResp.ImageBase64,
		ImageURL:     renderResp.ImageURL,
		Caption:      renderResp.Caption,
		FollowUpText: renderResp.FollowUpText,
		TemplateID:   renderReq.TemplateID,
		Confidence:   1.0, // Visual responses are deterministic
		TokensUsed:   tokensUsed,
		LatencyMs:    latencyMs + int64(renderResp.RenderTime.Milliseconds()),
	}, nil
}

// ShouldEscalate checks if conversation should be escalated based on context
func (s *BotServiceImpl) ShouldEscalate(ctx context.Context, botCtx *entity.ConversationContext, bot *entity.Bot, response *BotResponse) (bool, string, error) {
	// Already marked for escalation
	if response != nil && response.ShouldEscalate {
		return true, response.EscalateReason, nil
	}

	// Check sentiment-based escalation
	if botCtx.Sentiment == entity.SentimentNegative {
		for _, rule := range bot.Config.EscalationRules {
			if rule.Condition == entity.EscalationConditionSentiment && rule.Value == "negative" {
				return true, "Negative sentiment detected", nil
			}
		}
	}

	// Check confidence
	if response != nil && response.Confidence < bot.Config.ConfidenceThreshold {
		return true, "Low confidence response", nil
	}

	return false, "", nil
}

// GetOrCreateContext gets or creates conversation context
func (s *BotServiceImpl) GetOrCreateContext(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	return s.contextService.GetOrCreate(ctx, conversationID)
}

// AddEscalationRule adds an escalation rule to a bot
func (s *BotServiceImpl) AddEscalationRule(ctx context.Context, botID string, rule entity.EscalationRule) error {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return err
	}

	bot.AddEscalationRule(rule)
	return s.botRepo.Update(ctx, bot)
}

// UpdateConfig updates bot configuration
func (s *BotServiceImpl) UpdateConfig(ctx context.Context, botID string, config entity.BotConfig) error {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return err
	}

	bot.Config = config
	bot.UpdatedAt = time.Now()
	return s.botRepo.Update(ctx, bot)
}

// TestBot tests a bot with a message
func (s *BotServiceImpl) TestBot(ctx context.Context, botID, message string) (*BotResponse, error) {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}

	// Get AI provider
	provider, err := s.aiFactory.Get(bot.Provider)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get AI provider")
	}

	// Build simple message array
	messages := []Message{
		{Role: "system", Content: bot.Config.SystemPrompt},
		{Role: "user", Content: message},
	}

	// Generate completion
	startTime := time.Now()
	completion, err := provider.Complete(ctx, &CompletionRequest{
		Messages:    messages,
		Model:       bot.Model,
		MaxTokens:   bot.Config.MaxTokens,
		Temperature: bot.Config.Temperature,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "AI generation failed")
	}

	latencyMs := time.Since(startTime).Milliseconds()

	return &BotResponse{
		Content:    completion.Content,
		Confidence: 0.8,
		TokensUsed: completion.TokensUsed,
		LatencyMs:  latencyMs,
	}, nil
}

// Helper methods

func (s *BotServiceImpl) isWithinWorkingHours(wh *entity.WorkingHours) bool {
	if wh == nil || !wh.Enabled {
		return true
	}

	// Get current time in configured timezone
	loc, err := time.LoadLocation(wh.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	weekday := int(now.Weekday())
	currentTime := now.Format("15:04")

	// Find schedule for today
	for _, schedule := range wh.Schedule {
		if schedule.Day == weekday {
			if currentTime >= schedule.StartTime && currentTime <= schedule.EndTime {
				return true
			}
		}
	}

	return false
}

func formatEscalationReason(rule *entity.EscalationRule) string {
	switch rule.Condition {
	case entity.EscalationConditionLowConfidence:
		return "Low confidence in AI response"
	case entity.EscalationConditionSentiment:
		return "Negative sentiment detected"
	case entity.EscalationConditionKeyword:
		return "Escalation keyword detected: " + rule.Value
	case entity.EscalationConditionIntent:
		return "Escalation intent detected: " + rule.Value
	case entity.EscalationConditionUserRequest:
		return "User requested human assistance"
	default:
		return "Escalation rule triggered"
	}
}

// Note: ExtractKeywords is defined in intent.go
