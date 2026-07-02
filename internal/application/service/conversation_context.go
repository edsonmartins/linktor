package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

const (
	// defaultContextCacheTTL bounds how long a cached context entry lives before
	// it is eligible for eviction. It also caps memory growth over time.
	defaultContextCacheTTL = 5 * time.Minute
	// defaultContextCacheSize caps the number of cached contexts to prevent the
	// in-memory cache from growing without bound (memory leak).
	defaultContextCacheSize = 2048
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

// cacheEntry wraps a cached context with its expiry for TTL-based eviction.
type cacheEntry struct {
	ctx       *entity.ConversationContext
	expiresAt time.Time
}

// ConversationContextService manages AI context for conversations
type ConversationContextService struct {
	repo         repository.ConversationContextRepository
	config       *ConversationContextConfig
	mu           sync.Mutex
	cache        map[string]*cacheEntry // In-memory cache by conversation ID
	cacheTTL     time.Duration
	maxCacheSize int
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
		repo:         repo,
		config:       config,
		cache:        make(map[string]*cacheEntry),
		cacheTTL:     defaultContextCacheTTL,
		maxCacheSize: defaultContextCacheSize,
	}
}

// GetOrCreate gets existing context or creates a new one for a conversation.
//
// The repository is treated as the source of truth on read: flow state is
// persisted out-of-band by callers (bot.go persists via the repository, not
// through this service), so serving a stale cached object would lose flow
// state. The returned value is always an independent deep copy (copy-on-read)
// so that concurrent handlers of the same conversation never share and mutate
// the same State map, which previously caused fatal concurrent map writes.
func (s *ConversationContextService) GetOrCreate(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
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

	s.putCache(convContext)
	return cloneConversationContext(convContext), nil
}

// Get retrieves context by conversation ID. The returned value is an
// independent deep copy (see GetOrCreate for rationale).
func (s *ConversationContextService) Get(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	convContext, err := s.repo.FindByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	s.putCache(convContext)
	return cloneConversationContext(convContext), nil
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
	s.cache = make(map[string]*cacheEntry)
	s.mu.Unlock()
}

// Helper methods

// putCache stores a private copy of convContext in the cache with a TTL,
// evicting expired and, if necessary, the soonest-to-expire entries so the
// cache size stays bounded.
func (s *ConversationContextService) putCache(convContext *entity.ConversationContext) {
	if convContext == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked()
	s.cache[convContext.ConversationID] = &cacheEntry{
		ctx:       cloneConversationContext(convContext),
		expiresAt: time.Now().Add(s.cacheTTL),
	}
}

// evictLocked drops expired entries and enforces maxCacheSize. Caller must hold s.mu.
func (s *ConversationContextService) evictLocked() {
	now := time.Now()
	for key, entry := range s.cache {
		if now.After(entry.expiresAt) {
			delete(s.cache, key)
		}
	}
	// Enforce max size by evicting the entries that expire soonest.
	for len(s.cache) >= s.maxCacheSize {
		var oldestKey string
		var oldestAt time.Time
		first := true
		for key, entry := range s.cache {
			if first || entry.expiresAt.Before(oldestAt) {
				oldestKey, oldestAt, first = key, entry.expiresAt, false
			}
		}
		if oldestKey == "" {
			break
		}
		delete(s.cache, oldestKey)
	}
}

func (s *ConversationContextService) save(ctx context.Context, convContext *entity.ConversationContext) error {
	// Update in database
	if err := s.repo.Update(ctx, convContext); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to save conversation context")
	}

	// Update cache with a private copy
	s.putCache(convContext)

	return nil
}

// cloneConversationContext returns a deep copy of a conversation context. It is
// used for copy-on-read so that a context handed to one goroutine cannot be
// mutated concurrently through a shared State/Entities map by another.
func cloneConversationContext(c *entity.ConversationContext) *entity.ConversationContext {
	if c == nil {
		return nil
	}
	clone := *c // copy value fields
	clone.Entities = cloneInterfaceMap(c.Entities)
	clone.State = cloneInterfaceMap(c.State)
	if c.ContextWindow != nil {
		cw := make([]entity.ContextMessage, len(c.ContextWindow))
		copy(cw, c.ContextWindow)
		clone.ContextWindow = cw
	}
	if c.BotID != nil {
		b := *c.BotID
		clone.BotID = &b
	}
	if c.Intent != nil {
		intent := *c.Intent
		clone.Intent = &intent
	}
	if c.LastAnalysisAt != nil {
		t := *c.LastAnalysisAt
		clone.LastAnalysisAt = &t
	}
	return &clone
}

// cloneInterfaceMap deep-copies a map[string]interface{}, recursively copying
// nested maps commonly stored in flow State (map[string]interface{} and
// map[string]string) and slices so no inner map is shared between copies.
func cloneInterfaceMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = cloneInterfaceValue(v)
	}
	return out
}

func cloneInterfaceValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return cloneInterfaceMap(val)
	case map[string]string:
		out := make(map[string]string, len(val))
		for k, s := range val {
			out[k] = s
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, e := range val {
			out[i] = cloneInterfaceValue(e)
		}
		return out
	default:
		return v
	}
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
