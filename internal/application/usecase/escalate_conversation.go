package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// EscalateConversationInput represents input for escalating a conversation
type EscalateConversationInput struct {
	ConversationID string
	TenantID       string
	ChannelID      string
	ContactID      string
	BotID          string
	Reason         string
	Priority       string // low, normal, high, urgent
	RequestedBy    string // bot, user, system
}

// EscalateConversationOutput represents the result of escalation
type EscalateConversationOutput struct {
	ConversationID string    `json:"conversation_id"`
	Status         string    `json:"status"`
	QueuePosition  int       `json:"queue_position,omitempty"`
	EstimatedWait  int       `json:"estimated_wait_seconds,omitempty"`
	AssignedUserID string    `json:"assigned_user_id,omitempty"`
	EscalatedAt    time.Time `json:"escalated_at"`
}

// EscalateConversationUseCase handles conversation escalation from bot to human
type EscalateConversationUseCase struct {
	conversationRepo repository.ConversationRepository
	messageRepo      repository.MessageRepository
	contactRepo      repository.ContactRepository
	channelRepo      repository.ChannelRepository
	botRepo          repository.BotRepository
	userRepo         repository.UserRepository
	contextRepo      repository.ConversationContextRepository
	aiFactory        *service.AIProviderFactory
	producer         *nats.Producer
}

// NewEscalateConversationUseCase creates a new escalate conversation use case
func NewEscalateConversationUseCase(
	conversationRepo repository.ConversationRepository,
	messageRepo repository.MessageRepository,
	contactRepo repository.ContactRepository,
	channelRepo repository.ChannelRepository,
	botRepo repository.BotRepository,
	userRepo repository.UserRepository,
	contextRepo repository.ConversationContextRepository,
	aiFactory *service.AIProviderFactory,
	producer *nats.Producer,
) *EscalateConversationUseCase {
	return &EscalateConversationUseCase{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		contactRepo:      contactRepo,
		channelRepo:      channelRepo,
		botRepo:          botRepo,
		userRepo:         userRepo,
		contextRepo:      contextRepo,
		aiFactory:        aiFactory,
		producer:         producer,
	}
}

// Execute escalates a conversation from bot to human agent
func (uc *EscalateConversationUseCase) Execute(ctx context.Context, input *EscalateConversationInput) (*EscalateConversationOutput, error) {
	// Get conversation
	conversation, err := uc.conversationRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "conversation not found")
	}

	// Verify tenant match
	if conversation.TenantID != input.TenantID {
		return nil, errors.New(errors.ErrCodeForbidden, "conversation does not belong to tenant")
	}

	// Check if already escalated/assigned
	if conversation.AssignedUserID != nil {
		return &EscalateConversationOutput{
			ConversationID: conversation.ID,
			Status:         "already_assigned",
			AssignedUserID: *conversation.AssignedUserID,
			EscalatedAt:    time.Now(),
		}, nil
	}

	// Map priority to conversation priority
	priority := mapPriority(input.Priority)

	// Update conversation status to pending (waiting for human)
	conversation.Status = entity.ConversationStatusPending
	conversation.Priority = priority
	conversation.UpdatedAt = time.Now()

	// Add escalation metadata
	if conversation.Metadata == nil {
		conversation.Metadata = make(map[string]string)
	}
	conversation.Metadata["escalation_reason"] = input.Reason
	conversation.Metadata["escalated_by"] = input.RequestedBy
	conversation.Metadata["escalated_at"] = time.Now().Format(time.RFC3339)
	if input.BotID != "" {
		conversation.Metadata["escalated_from_bot"] = input.BotID
	}

	// Try to auto-assign to available agent
	assignedUserID, queuePosition := uc.tryAutoAssign(ctx, conversation)

	if assignedUserID != "" {
		conversation.AssignedUserID = &assignedUserID
		conversation.Status = entity.ConversationStatusOpen
	}

	// Save conversation
	if err := uc.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update conversation")
	}

	// Publish escalation event
	uc.publishEscalationEvent(ctx, input, conversation, assignedUserID)

	output := &EscalateConversationOutput{
		ConversationID: conversation.ID,
		Status:         string(conversation.Status),
		QueuePosition:  queuePosition,
		EscalatedAt:    time.Now(),
	}

	if assignedUserID != "" {
		output.AssignedUserID = assignedUserID
		output.Status = "assigned"
	} else {
		output.Status = "queued"
		// Estimate wait time based on queue position (simple heuristic)
		output.EstimatedWait = queuePosition * 120 // 2 min per conversation
	}

	return output, nil
}

// tryAutoAssign attempts to auto-assign the conversation to an available agent
func (uc *EscalateConversationUseCase) tryAutoAssign(ctx context.Context, conversation *entity.Conversation) (string, int) {
	// Find available agents for this channel
	agents, err := uc.userRepo.FindAvailableAgents(ctx, conversation.TenantID, conversation.ChannelID)
	if err != nil || len(agents) == 0 {
		// No available agents, calculate queue position
		queuePosition := uc.calculateQueuePosition(ctx, conversation)
		return "", queuePosition
	}

	// Find agent with lowest workload
	var bestAgent *entity.User
	lowestWorkload := int64(999999)

	for _, agent := range agents {
		workload, err := uc.conversationRepo.CountActiveByUser(ctx, agent.ID)
		if err != nil {
			continue
		}
		if workload < lowestWorkload {
			lowestWorkload = workload
			bestAgent = agent
		}
	}

	if bestAgent != nil {
		return bestAgent.ID, 0
	}

	queuePosition := uc.calculateQueuePosition(ctx, conversation)
	return "", queuePosition
}

// calculateQueuePosition calculates the queue position for a conversation
func (uc *EscalateConversationUseCase) calculateQueuePosition(ctx context.Context, conversation *entity.Conversation) int {
	// Count waiting conversations with same or higher priority
	count, err := uc.conversationRepo.CountWaiting(ctx, conversation.TenantID, conversation.Priority)
	if err != nil {
		return 1
	}
	return int(count) + 1
}

// publishEscalationEvent publishes the escalation event
func (uc *EscalateConversationUseCase) publishEscalationEvent(
	ctx context.Context,
	input *EscalateConversationInput,
	conversation *entity.Conversation,
	assignedUserID string,
) {
	payload := map[string]interface{}{
		"conversation_id": conversation.ID,
		"channel_id":      conversation.ChannelID,
		"contact_id":      conversation.ContactID,
		"reason":          input.Reason,
		"priority":        input.Priority,
		"requested_by":    input.RequestedBy,
		"status":          string(conversation.Status),
	}

	if input.BotID != "" {
		payload["bot_id"] = input.BotID
	}

	if assignedUserID != "" {
		payload["assigned_user_id"] = assignedUserID
	}

	event := &nats.Event{
		Type:      nats.EventConversationEscalated,
		TenantID:  input.TenantID,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	uc.producer.PublishEvent(ctx, event)
}

// mapPriority maps string priority to entity priority
func mapPriority(priority string) entity.ConversationPriority {
	switch priority {
	case "urgent":
		return entity.ConversationPriorityUrgent
	case "high":
		return entity.ConversationPriorityHigh
	case "low":
		return entity.ConversationPriorityLow
	default:
		return entity.ConversationPriorityNormal
	}
}

// EscalateFromBot is a convenience method for bot-initiated escalation
func (uc *EscalateConversationUseCase) EscalateFromBot(
	ctx context.Context,
	conversationID, tenantID, channelID, contactID, botID, reason string,
) (*EscalateConversationOutput, error) {
	// Determine priority based on reason
	priority := "normal"
	if containsUrgentKeywords(reason) {
		priority = "high"
	}

	return uc.Execute(ctx, &EscalateConversationInput{
		ConversationID: conversationID,
		TenantID:       tenantID,
		ChannelID:      channelID,
		ContactID:      contactID,
		BotID:          botID,
		Reason:         reason,
		Priority:       priority,
		RequestedBy:    "bot",
	})
}

// EscalateFromUser is a convenience method for user-requested escalation
func (uc *EscalateConversationUseCase) EscalateFromUser(
	ctx context.Context,
	conversationID, tenantID, channelID, contactID string,
) (*EscalateConversationOutput, error) {
	return uc.Execute(ctx, &EscalateConversationInput{
		ConversationID: conversationID,
		TenantID:       tenantID,
		ChannelID:      channelID,
		ContactID:      contactID,
		Reason:         "User requested human assistance",
		Priority:       "normal",
		RequestedBy:    "user",
	})
}

// containsUrgentKeywords checks if reason contains urgent keywords
func containsUrgentKeywords(reason string) bool {
	urgentKeywords := []string{
		"urgent", "urgente", "emergency", "emergência",
		"complaint", "reclamação", "angry", "raiva",
		"critical", "crítico",
	}

	lowerReason := reason
	for _, kw := range urgentKeywords {
		if containsString(lowerReason, kw) {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if len(s) >= len(substr) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetEscalationContext generates a complete escalation context for human agents
func (uc *EscalateConversationUseCase) GetEscalationContext(ctx context.Context, conversationID, tenantID string) (*entity.EscalationContext, error) {
	// Get conversation
	conversation, err := uc.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "conversation not found")
	}

	// Verify tenant
	if conversation.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeForbidden, "conversation does not belong to tenant")
	}

	// Create escalation context
	reason := entity.EscalationReasonManual
	if conversation.Metadata != nil {
		if r, ok := conversation.Metadata["escalation_reason"]; ok {
			reason = mapReasonString(r)
		}
	}

	escCtx := entity.NewEscalationContext(conversationID, tenantID, reason)
	escCtx.ConversationStartedAt = conversation.CreatedAt

	// Set priority from conversation
	escCtx.Priority = mapEntityPriorityToEscalation(conversation.Priority)

	// Get last messages
	messages, _, err := uc.messageRepo.FindByConversation(ctx, conversationID, &repository.ListParams{
		Page:     1,
		PageSize: 20,
		SortBy:   "created_at",
		SortDir:  "desc",
	})
	if err == nil {
		escCtx.MessageCount = len(messages)
		// Reverse to get chronological order
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			escMsg := entity.EscalationMessage{
				ID:         msg.ID,
				SenderType: string(msg.SenderType),
				Content:    msg.Content,
				Timestamp:  msg.CreatedAt,
				IsBot:      msg.SenderType == entity.SenderTypeBot,
			}
			escCtx.AddMessage(escMsg)
		}
	}

	// Get contact info
	if contact, err := uc.contactRepo.FindByID(ctx, conversation.ContactID); err == nil {
		escCtx.Customer = &entity.EscalationCustomer{
			ID:    contact.ID,
			Name:  contact.Name,
			Email: contact.Email,
			Phone: contact.Phone,
		}
		if contact.CustomFields != nil {
			escCtx.Customer.CustomFields = contact.CustomFields
		}
	}

	// Get channel info
	if channel, err := uc.channelRepo.FindByID(ctx, conversation.ChannelID); err == nil {
		escCtx.ChannelType = string(channel.Type)
		escCtx.ChannelName = channel.Name
	}

	// Get conversation context (intent, sentiment, entities, flow state)
	if convContext, err := uc.contextRepo.FindByConversation(ctx, conversationID); err == nil {
		if convContext.Intent != nil {
			escCtx.DetectedIntent = convContext.Intent.Name
		}
		escCtx.Sentiment = string(convContext.Sentiment)

		// Extract entities
		if convContext.Entities != nil {
			for k, v := range convContext.Entities {
				if strVal, ok := v.(string); ok {
					escCtx.AddEntity(k, strVal)
				}
			}
		}

		// Extract flow state
		if convContext.State != nil {
			if flowID, ok := convContext.State["active_flow_id"].(string); ok && flowID != "" {
				escCtx.ActiveFlowID = flowID
			}
			if nodeID, ok := convContext.State["current_node_id"].(string); ok {
				escCtx.FlowNodeID = nodeID
			}
			if collected, ok := convContext.State["collected_data"].(map[string]string); ok {
				escCtx.FlowData = collected
			}
		}

		// Get bot info
		if convContext.BotID != nil && *convContext.BotID != "" {
			escCtx.BotID = *convContext.BotID
			if bot, err := uc.botRepo.FindByID(ctx, *convContext.BotID); err == nil {
				escCtx.BotName = bot.Name
			}
		}
	}

	// Add conversation tags
	escCtx.Tags = conversation.Tags

	// Calculate wait time
	escCtx.WaitTimeSeconds = escCtx.CalculateWaitTime()

	// Generate AI summary
	if len(escCtx.LastMessages) > 0 && uc.aiFactory != nil {
		summary, err := uc.generateSummary(ctx, escCtx)
		if err == nil {
			escCtx.Summary = summary
		}
	}

	return escCtx, nil
}

// generateSummary uses AI to generate a conversation summary for agents
func (uc *EscalateConversationUseCase) generateSummary(ctx context.Context, escCtx *entity.EscalationContext) (string, error) {
	// Try to get any available AI provider
	var provider service.AIProvider
	var err error

	// Try OpenAI first, then others
	for _, providerType := range []entity.AIProviderType{entity.AIProviderOpenAI, entity.AIProviderAnthropic, entity.AIProviderOllama} {
		provider, err = uc.aiFactory.Get(providerType)
		if err == nil {
			break
		}
	}

	if provider == nil {
		return "", errors.New(errors.ErrCodeInternal, "no AI provider available")
	}

	// Build conversation text
	var conversationText strings.Builder
	for _, msg := range escCtx.LastMessages {
		role := "Customer"
		if msg.IsBot {
			role = "Bot"
		} else if msg.SenderType == "user" {
			role = "Agent"
		}
		conversationText.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	// Build summary prompt
	prompt := fmt.Sprintf(`Resuma esta conversa em 2-3 frases para um agente humano que vai assumir o atendimento.
Inclua:
- Qual é o problema ou solicitação do cliente
- O que o bot/sistema já tentou fazer
- Qual parece ser o sentimento do cliente

Conversa:
%s

Resumo:`, conversationText.String())

	// Request completion
	req := &service.CompletionRequest{
		Messages: []service.Message{
			{Role: "system", Content: "Você é um assistente que gera resumos concisos de conversas para agentes de suporte."},
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o-mini", // Use a smaller model for summaries
		MaxTokens:   200,
		Temperature: 0.3,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeInternal, "failed to generate summary")
	}

	return strings.TrimSpace(resp.Content), nil
}

// mapReasonString maps a reason string to EscalationReason
func mapReasonString(reason string) entity.EscalationReason {
	lowerReason := strings.ToLower(reason)
	switch {
	case strings.Contains(lowerReason, "confidence"):
		return entity.EscalationReasonLowConfidence
	case strings.Contains(lowerReason, "sentiment") || strings.Contains(lowerReason, "negative"):
		return entity.EscalationReasonNegativeSentiment
	case strings.Contains(lowerReason, "keyword"):
		return entity.EscalationReasonKeywordTrigger
	case strings.Contains(lowerReason, "user") || strings.Contains(lowerReason, "human"):
		return entity.EscalationReasonUserRequest
	case strings.Contains(lowerReason, "flow"):
		return entity.EscalationReasonFlowAction
	case strings.Contains(lowerReason, "fail") || strings.Contains(lowerReason, "error"):
		return entity.EscalationReasonBotFailure
	default:
		return entity.EscalationReasonManual
	}
}

// mapEntityPriorityToEscalation maps conversation priority to escalation priority
func mapEntityPriorityToEscalation(p entity.ConversationPriority) entity.EscalationPriority {
	switch p {
	case entity.ConversationPriorityUrgent:
		return entity.EscalationPriorityUrgent
	case entity.ConversationPriorityHigh:
		return entity.EscalationPriorityHigh
	case entity.ConversationPriorityLow:
		return entity.EscalationPriorityLow
	default:
		return entity.EscalationPriorityNormal
	}
}
