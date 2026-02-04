package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// BotRepository defines the interface for bot persistence
type BotRepository interface {
	// Create creates a new bot
	Create(ctx context.Context, bot *entity.Bot) error

	// FindByID finds a bot by ID
	FindByID(ctx context.Context, id string) (*entity.Bot, error)

	// FindByTenant finds bots for a tenant with pagination
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.Bot, int64, error)

	// FindByChannel finds the bot assigned to a channel
	FindByChannel(ctx context.Context, channelID string) (*entity.Bot, error)

	// FindActiveByTenant finds active bots for a tenant
	FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Bot, error)

	// Update updates a bot
	Update(ctx context.Context, bot *entity.Bot) error

	// UpdateStatus updates only the bot status
	UpdateStatus(ctx context.Context, id string, status entity.BotStatus) error

	// Delete deletes a bot
	Delete(ctx context.Context, id string) error

	// CountByTenant counts bots for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)

	// AssignChannel assigns a channel to a bot
	AssignChannel(ctx context.Context, botID, channelID string) error

	// UnassignChannel removes a channel from a bot
	UnassignChannel(ctx context.Context, botID, channelID string) error
}

// ConversationContextRepository defines the interface for conversation context persistence
type ConversationContextRepository interface {
	// Create creates a new conversation context
	Create(ctx context.Context, convContext *entity.ConversationContext) error

	// FindByID finds a conversation context by ID
	FindByID(ctx context.Context, id string) (*entity.ConversationContext, error)

	// FindByConversation finds the context for a conversation
	FindByConversation(ctx context.Context, conversationID string) (*entity.ConversationContext, error)

	// Update updates a conversation context
	Update(ctx context.Context, convContext *entity.ConversationContext) error

	// UpdateIntent updates only the intent
	UpdateIntent(ctx context.Context, id string, intent *entity.Intent) error

	// UpdateSentiment updates only the sentiment
	UpdateSentiment(ctx context.Context, id string, sentiment entity.Sentiment) error

	// UpdateContextWindow updates the context window
	UpdateContextWindow(ctx context.Context, id string, window []entity.ContextMessage) error

	// Delete deletes a conversation context
	Delete(ctx context.Context, id string) error
}

// AIResponseRepository defines the interface for AI response audit persistence
type AIResponseRepository interface {
	// Create creates a new AI response record
	Create(ctx context.Context, response *entity.AIResponse) error

	// FindByID finds an AI response by ID
	FindByID(ctx context.Context, id string) (*entity.AIResponse, error)

	// FindByMessage finds AI responses for a message
	FindByMessage(ctx context.Context, messageID string) ([]*entity.AIResponse, error)

	// FindByBot finds AI responses by bot with pagination
	FindByBot(ctx context.Context, botID string, params *ListParams) ([]*entity.AIResponse, int64, error)

	// CountByBot counts AI responses for a bot
	CountByBot(ctx context.Context, botID string) (int64, error)

	// GetAverageLatency gets average latency for a bot
	GetAverageLatency(ctx context.Context, botID string) (float64, error)

	// GetTotalTokensUsed gets total tokens used by a bot
	GetTotalTokensUsed(ctx context.Context, botID string) (int64, error)
}

// KnowledgeBaseRepository defines the interface for knowledge base persistence
type KnowledgeBaseRepository interface {
	// Create creates a new knowledge base
	Create(ctx context.Context, kb *entity.KnowledgeBase) error

	// FindByID finds a knowledge base by ID
	FindByID(ctx context.Context, id string) (*entity.KnowledgeBase, error)

	// FindByTenant finds knowledge bases for a tenant with pagination
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.KnowledgeBase, int64, error)

	// Update updates a knowledge base
	Update(ctx context.Context, kb *entity.KnowledgeBase) error

	// Delete deletes a knowledge base
	Delete(ctx context.Context, id string) error

	// CountByTenant counts knowledge bases for a tenant
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
}

// KnowledgeItemRepository defines the interface for knowledge item persistence
type KnowledgeItemRepository interface {
	// Create creates a new knowledge item
	Create(ctx context.Context, item *entity.KnowledgeItem) error

	// FindByID finds a knowledge item by ID
	FindByID(ctx context.Context, id string) (*entity.KnowledgeItem, error)

	// FindByKnowledgeBase finds items for a knowledge base with pagination
	FindByKnowledgeBase(ctx context.Context, kbID string, params *ListParams) ([]*entity.KnowledgeItem, int64, error)

	// Update updates a knowledge item
	Update(ctx context.Context, item *entity.KnowledgeItem) error

	// UpdateEmbedding updates only the embedding
	UpdateEmbedding(ctx context.Context, id string, embedding []float64) error

	// Delete deletes a knowledge item
	Delete(ctx context.Context, id string) error

	// CountByKnowledgeBase counts items in a knowledge base
	CountByKnowledgeBase(ctx context.Context, kbID string) (int64, error)

	// SearchByKeywords searches items by keywords
	SearchByKeywords(ctx context.Context, kbID string, keywords []string, limit int) ([]*entity.KnowledgeItem, error)

	// SearchByEmbedding searches items by vector similarity (RAG)
	SearchByEmbedding(ctx context.Context, kbID string, embedding []float64, limit int, minScore float64) ([]*entity.SearchResult, error)

	// DeleteByKnowledgeBase deletes all items in a knowledge base
	DeleteByKnowledgeBase(ctx context.Context, kbID string) error
}
