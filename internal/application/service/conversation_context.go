package service

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ConversationContextConfig holds configuration for context management
type ConversationContextConfig struct {
	MaxContextWindowSize int // Maximum number of messages to keep in context
	TrimToSize           int // Size to trim to when max is exceeded
}

// DefaultContextConfig returns default context configuration
func DefaultContextConfig() *ConversationContextConfig {
	return &ConversationContextConfig{
		MaxContextWindowSize: 20,
		TrimToSize:           10,
	}
}

// ConversationContextService manages AI context for conversations
type ConversationContextService struct {
	repo   repository.ConversationContextRepository
	config *ConversationContextConfig
	mu     sync.RWMutex
	cache  map[string]*entity.ConversationContext // In-memory cache by conversation ID
}

// NewConversationContextService creates a new conversation context service
func NewConversationContextService(
	repo repository.ConversationContextRepository,
	config *ConversationContextConfig,
) *ConversationContextService {
	if config == nil {
		config = DefaultContextConfig()
	}
	return &ConversationContextService{
		repo:   repo,
		config: config,
		cache:  make(map[string]*entity.ConversationContext),
	}
}

// GetOrCreate gets existing context or creates a new one for a conversation
func (s *ConversationContextService) GetOrCreate(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	// Check cache first
	s.mu.RLock()
	if cached, ok := s.cache[conversationID]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	// Try to find in database
	convContext, err := s.repo.FindByConversation(ctx, conversationID)
	if err != nil {
		// If not found, create new
		if errors.IsNotFound(err) {
			convContext = entity.NewConversationContext(conversationID)
			convContext.ID = uuid.New().String()

			if err := s.repo.Create(ctx, convContext); err != nil {
				return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create conversation context")
			}
		} else {
			return nil, err
		}
	}

	// Cache it
	s.mu.Lock()
	s.cache[conversationID] = convContext
	s.mu.Unlock()

	return convContext, nil
}

// Get retrieves context by conversation ID
func (s *ConversationContextService) Get(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	// Check cache first
	s.mu.RLock()
	if cached, ok := s.cache[conversationID]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	convContext, err := s.repo.FindByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Cache it
	s.mu.Lock()
	s.cache[conversationID] = convContext
	s.mu.Unlock()

	return convContext, nil
}

// AddUserMessage adds a user message to the context window
func (s *ConversationContextService) AddUserMessage(ctx context.Context, conversationID, content, messageID string) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.AddUserMessage(content, messageID)
	s.trimContextWindowIfNeeded(convContext)

	return s.save(ctx, convContext)
}

// AddAssistantMessage adds an assistant (bot) message to the context window
func (s *ConversationContextService) AddAssistantMessage(ctx context.Context, conversationID, content, messageID string) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.AddAssistantMessage(content, messageID)
	s.trimContextWindowIfNeeded(convContext)

	return s.save(ctx, convContext)
}

// AddSystemMessage adds a system message to the context window
func (s *ConversationContextService) AddSystemMessage(ctx context.Context, conversationID, content string) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.AddSystemMessage(content)
	s.trimContextWindowIfNeeded(convContext)

	return s.save(ctx, convContext)
}

// SetBot assigns a bot to the conversation context
func (s *ConversationContextService) SetBot(ctx context.Context, conversationID, botID string) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.SetBot(botID)
	return s.save(ctx, convContext)
}

// ClearBot removes bot assignment from context
func (s *ConversationContextService) ClearBot(ctx context.Context, conversationID string) error {
	convContext, err := s.Get(ctx, conversationID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	convContext.ClearBot()
	return s.save(ctx, convContext)
}

// UpdateIntent updates the detected intent
func (s *ConversationContextService) UpdateIntent(ctx context.Context, conversationID string, intent *entity.Intent) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.SetIntent(intent)
	return s.save(ctx, convContext)
}

// UpdateSentiment updates the detected sentiment
func (s *ConversationContextService) UpdateSentiment(ctx context.Context, conversationID string, sentiment entity.Sentiment) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.SetSentiment(sentiment)
	return s.save(ctx, convContext)
}

// SetEntity sets an entity value in the context
func (s *ConversationContextService) SetEntity(ctx context.Context, conversationID, key string, value interface{}) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.SetEntity(key, value)
	return s.save(ctx, convContext)
}

// SetStateValue sets a state variable in the context (for flows)
func (s *ConversationContextService) SetStateValue(ctx context.Context, conversationID, key string, value interface{}) error {
	convContext, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	convContext.SetStateValue(key, value)
	return s.save(ctx, convContext)
}

// ClearState clears all state variables (e.g., when starting a new flow)
func (s *ConversationContextService) ClearState(ctx context.Context, conversationID string) error {
	convContext, err := s.Get(ctx, conversationID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	convContext.ClearState()
	return s.save(ctx, convContext)
}

// GetContextWindow returns the context window for a conversation
func (s *ConversationContextService) GetContextWindow(ctx context.Context, conversationID string, maxMessages int) ([]entity.ContextMessage, error) {
	convContext, err := s.Get(ctx, conversationID)
	if err != nil {
		if errors.IsNotFound(err) {
			return []entity.ContextMessage{}, nil
		}
		return nil, err
	}

	messages := convContext.GetContextMessages()

	// Limit if requested
	if maxMessages > 0 && len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}

	return messages, nil
}

// Delete removes a conversation context
func (s *ConversationContextService) Delete(ctx context.Context, conversationID string) error {
	// Remove from cache
	s.mu.Lock()
	delete(s.cache, conversationID)
	s.mu.Unlock()

	// Get context to find ID
	convContext, err := s.repo.FindByConversation(ctx, conversationID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return s.repo.Delete(ctx, convContext.ID)
}

// InvalidateCache removes a conversation from the cache
func (s *ConversationContextService) InvalidateCache(conversationID string) {
	s.mu.Lock()
	delete(s.cache, conversationID)
	s.mu.Unlock()
}

// ClearCache clears the entire cache
func (s *ConversationContextService) ClearCache() {
	s.mu.Lock()
	s.cache = make(map[string]*entity.ConversationContext)
	s.mu.Unlock()
}

// Helper methods

func (s *ConversationContextService) save(ctx context.Context, convContext *entity.ConversationContext) error {
	// Update in database
	if err := s.repo.Update(ctx, convContext); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to save conversation context")
	}

	// Update cache
	s.mu.Lock()
	s.cache[convContext.ConversationID] = convContext
	s.mu.Unlock()

	return nil
}

func (s *ConversationContextService) trimContextWindowIfNeeded(convContext *entity.ConversationContext) {
	if len(convContext.ContextWindow) > s.config.MaxContextWindowSize {
		convContext.TrimContextWindow(s.config.TrimToSize)
	}
}

// BuildMessagesForAI builds the messages array for AI completion
func (s *ConversationContextService) BuildMessagesForAI(ctx context.Context, conversationID, systemPrompt, currentMessage string, maxContext int) ([]Message, error) {
	contextWindow, err := s.GetContextWindow(ctx, conversationID, maxContext)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(contextWindow)+2)

	// Add system prompt
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add context window
	for _, msg := range contextWindow {
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current message
	messages = append(messages, Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages, nil
}
