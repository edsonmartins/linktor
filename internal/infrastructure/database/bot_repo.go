package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// BotRepository implements repository.BotRepository with PostgreSQL
type BotRepository struct {
	db *PostgresDB
}

// NewBotRepository creates a new PostgreSQL bot repository
func NewBotRepository(db *PostgresDB) *BotRepository {
	return &BotRepository{db: db}
}

// Create creates a new bot
func (r *BotRepository) Create(ctx context.Context, bot *entity.Bot) error {
	config, err := json.Marshal(bot.Config)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal config")
	}

	query := `
		INSERT INTO bots (
			id, tenant_id, name, type, provider, model, config,
			status, channels, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		bot.ID,
		bot.TenantID,
		bot.Name,
		string(bot.Type),
		string(bot.Provider),
		bot.Model,
		config,
		string(bot.Status),
		pq.Array(bot.Channels),
		bot.CreatedAt,
		bot.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create bot")
	}

	return nil
}

// FindByID finds a bot by ID
func (r *BotRepository) FindByID(ctx context.Context, id string) (*entity.Bot, error) {
	query := `
		SELECT id, tenant_id, name, type, provider, model, config,
		       status, channels, created_at, updated_at
		FROM bots
		WHERE id = $1
	`

	bot, err := r.scanBot(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "bot not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find bot")
	}

	return bot, nil
}

// FindByTenant finds bots for a tenant with pagination
func (r *BotRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Bot, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM bots WHERE tenant_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count bots")
	}

	// Get bots
	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, type, provider, model, config,
		       status, channels, created_at, updated_at
		FROM bots
		WHERE tenant_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeBotColumn(params.SortBy), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, tenantID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query bots")
	}
	defer rows.Close()

	var bots []*entity.Bot
	for rows.Next() {
		bot, err := r.scanBotFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		bots = append(bots, bot)
	}

	return bots, total, nil
}

// FindByChannel finds the bot assigned to a channel
func (r *BotRepository) FindByChannel(ctx context.Context, channelID string) (*entity.Bot, error) {
	query := `
		SELECT id, tenant_id, name, type, provider, model, config,
		       status, channels, created_at, updated_at
		FROM bots
		WHERE $1 = ANY(channels) AND status = 'active'
		LIMIT 1
	`

	bot, err := r.scanBot(r.db.Pool.QueryRow(ctx, query, channelID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "no bot found for channel")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find bot for channel")
	}

	return bot, nil
}

// FindActiveByTenant finds active bots for a tenant
func (r *BotRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Bot, error) {
	query := `
		SELECT id, tenant_id, name, type, provider, model, config,
		       status, channels, created_at, updated_at
		FROM bots
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query active bots")
	}
	defer rows.Close()

	var bots []*entity.Bot
	for rows.Next() {
		bot, err := r.scanBotFromRows(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}

	return bots, nil
}

// Update updates a bot
func (r *BotRepository) Update(ctx context.Context, bot *entity.Bot) error {
	bot.UpdatedAt = time.Now()

	config, err := json.Marshal(bot.Config)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal config")
	}

	query := `
		UPDATE bots SET
			name = $1,
			type = $2,
			provider = $3,
			model = $4,
			config = $5,
			status = $6,
			channels = $7,
			updated_at = $8
		WHERE id = $9
	`

	result, err := r.db.Pool.Exec(ctx, query,
		bot.Name,
		string(bot.Type),
		string(bot.Provider),
		bot.Model,
		config,
		string(bot.Status),
		pq.Array(bot.Channels),
		bot.UpdatedAt,
		bot.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update bot")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "bot not found")
	}

	return nil
}

// UpdateStatus updates only the bot status
func (r *BotRepository) UpdateStatus(ctx context.Context, id string, status entity.BotStatus) error {
	query := `UPDATE bots SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, string(status), time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update bot status")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "bot not found")
	}

	return nil
}

// Delete deletes a bot
func (r *BotRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM bots WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete bot")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "bot not found")
	}

	return nil
}

// CountByTenant counts bots for a tenant
func (r *BotRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM bots WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count bots")
	}

	return count, nil
}

// AssignChannel assigns a channel to a bot
func (r *BotRepository) AssignChannel(ctx context.Context, botID, channelID string) error {
	query := `
		UPDATE bots
		SET channels = array_append(channels, $1), updated_at = $2
		WHERE id = $3 AND NOT ($1 = ANY(channels))
	`

	_, err := r.db.Pool.Exec(ctx, query, channelID, time.Now(), botID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to assign channel to bot")
	}

	return nil
}

// UnassignChannel removes a channel from a bot
func (r *BotRepository) UnassignChannel(ctx context.Context, botID, channelID string) error {
	query := `
		UPDATE bots
		SET channels = array_remove(channels, $1), updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Pool.Exec(ctx, query, channelID, time.Now(), botID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to unassign channel from bot")
	}

	return nil
}

// Helper methods

func (r *BotRepository) scanBot(row pgx.Row) (*entity.Bot, error) {
	var b entity.Bot
	var botType, provider, status string
	var config []byte
	var channels []string

	err := row.Scan(
		&b.ID, &b.TenantID, &b.Name, &botType, &provider, &b.Model,
		&config, &status, pq.Array(&channels), &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	b.Type = entity.BotType(botType)
	b.Provider = entity.AIProviderType(provider)
	b.Status = entity.BotStatus(status)
	b.Channels = channels

	if err := json.Unmarshal(config, &b.Config); err != nil {
		b.Config = entity.BotConfig{}
	}

	return &b, nil
}

func (r *BotRepository) scanBotFromRows(rows pgx.Rows) (*entity.Bot, error) {
	var b entity.Bot
	var botType, provider, status string
	var config []byte
	var channels []string

	err := rows.Scan(
		&b.ID, &b.TenantID, &b.Name, &botType, &provider, &b.Model,
		&config, &status, pq.Array(&channels), &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan bot")
	}

	b.Type = entity.BotType(botType)
	b.Provider = entity.AIProviderType(provider)
	b.Status = entity.BotStatus(status)
	b.Channels = channels

	if err := json.Unmarshal(config, &b.Config); err != nil {
		b.Config = entity.BotConfig{}
	}

	return &b, nil
}

func sanitizeBotColumn(col string) string {
	allowed := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"name":       true,
		"type":       true,
		"status":     true,
		"provider":   true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}

// ConversationContextRepository implements repository.ConversationContextRepository
type ConversationContextRepository struct {
	db *PostgresDB
}

// NewConversationContextRepository creates a new PostgreSQL conversation context repository
func NewConversationContextRepository(db *PostgresDB) *ConversationContextRepository {
	return &ConversationContextRepository{db: db}
}

// Create creates a new conversation context
func (r *ConversationContextRepository) Create(ctx context.Context, convContext *entity.ConversationContext) error {
	entities, err := json.Marshal(convContext.Entities)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal entities")
	}

	contextWindow, err := json.Marshal(convContext.ContextWindow)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal context window")
	}

	state, err := json.Marshal(convContext.State)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal state")
	}

	var intentName *string
	var intentConfidence *float64
	if convContext.Intent != nil {
		intentName = &convContext.Intent.Name
		intentConfidence = &convContext.Intent.Confidence
	}

	query := `
		INSERT INTO conversation_contexts (
			id, conversation_id, bot_id, intent_name, intent_confidence,
			entities, sentiment, context_window, state, last_analysis_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		convContext.ID,
		convContext.ConversationID,
		convContext.BotID,
		intentName,
		intentConfidence,
		entities,
		string(convContext.Sentiment),
		contextWindow,
		state,
		convContext.LastAnalysisAt,
		convContext.CreatedAt,
		convContext.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create conversation context")
	}

	return nil
}

// FindByID finds a conversation context by ID
func (r *ConversationContextRepository) FindByID(ctx context.Context, id string) (*entity.ConversationContext, error) {
	query := `
		SELECT id, conversation_id, bot_id, intent_name, intent_confidence,
		       entities, sentiment, context_window, state, last_analysis_at,
		       created_at, updated_at
		FROM conversation_contexts
		WHERE id = $1
	`

	convContext, err := r.scanContext(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "conversation context not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find conversation context")
	}

	return convContext, nil
}

// FindByConversation finds the context for a conversation
func (r *ConversationContextRepository) FindByConversation(ctx context.Context, conversationID string) (*entity.ConversationContext, error) {
	query := `
		SELECT id, conversation_id, bot_id, intent_name, intent_confidence,
		       entities, sentiment, context_window, state, last_analysis_at,
		       created_at, updated_at
		FROM conversation_contexts
		WHERE conversation_id = $1
	`

	convContext, err := r.scanContext(r.db.Pool.QueryRow(ctx, query, conversationID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "conversation context not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find conversation context")
	}

	return convContext, nil
}

// Update updates a conversation context
func (r *ConversationContextRepository) Update(ctx context.Context, convContext *entity.ConversationContext) error {
	convContext.UpdatedAt = time.Now()

	entities, err := json.Marshal(convContext.Entities)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal entities")
	}

	contextWindow, err := json.Marshal(convContext.ContextWindow)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal context window")
	}

	state, err := json.Marshal(convContext.State)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal state")
	}

	var intentName *string
	var intentConfidence *float64
	if convContext.Intent != nil {
		intentName = &convContext.Intent.Name
		intentConfidence = &convContext.Intent.Confidence
	}

	query := `
		UPDATE conversation_contexts SET
			bot_id = $1,
			intent_name = $2,
			intent_confidence = $3,
			entities = $4,
			sentiment = $5,
			context_window = $6,
			state = $7,
			last_analysis_at = $8,
			updated_at = $9
		WHERE id = $10
	`

	result, err := r.db.Pool.Exec(ctx, query,
		convContext.BotID,
		intentName,
		intentConfidence,
		entities,
		string(convContext.Sentiment),
		contextWindow,
		state,
		convContext.LastAnalysisAt,
		convContext.UpdatedAt,
		convContext.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update conversation context")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "conversation context not found")
	}

	return nil
}

// UpdateIntent updates only the intent
func (r *ConversationContextRepository) UpdateIntent(ctx context.Context, id string, intent *entity.Intent) error {
	var intentName *string
	var intentConfidence *float64
	if intent != nil {
		intentName = &intent.Name
		intentConfidence = &intent.Confidence
	}

	query := `
		UPDATE conversation_contexts
		SET intent_name = $1, intent_confidence = $2, last_analysis_at = $3, updated_at = $4
		WHERE id = $5
	`

	now := time.Now()
	result, err := r.db.Pool.Exec(ctx, query, intentName, intentConfidence, now, now, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update intent")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "conversation context not found")
	}

	return nil
}

// UpdateSentiment updates only the sentiment
func (r *ConversationContextRepository) UpdateSentiment(ctx context.Context, id string, sentiment entity.Sentiment) error {
	query := `UPDATE conversation_contexts SET sentiment = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, string(sentiment), time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update sentiment")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "conversation context not found")
	}

	return nil
}

// UpdateContextWindow updates the context window
func (r *ConversationContextRepository) UpdateContextWindow(ctx context.Context, id string, window []entity.ContextMessage) error {
	contextWindow, err := json.Marshal(window)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal context window")
	}

	query := `UPDATE conversation_contexts SET context_window = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, contextWindow, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update context window")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "conversation context not found")
	}

	return nil
}

// Delete deletes a conversation context
func (r *ConversationContextRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM conversation_contexts WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete conversation context")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "conversation context not found")
	}

	return nil
}

func (r *ConversationContextRepository) scanContext(row pgx.Row) (*entity.ConversationContext, error) {
	var c entity.ConversationContext
	var intentName *string
	var intentConfidence *float64
	var sentiment string
	var entities, contextWindow, state []byte

	err := row.Scan(
		&c.ID, &c.ConversationID, &c.BotID, &intentName, &intentConfidence,
		&entities, &sentiment, &contextWindow, &state, &c.LastAnalysisAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.Sentiment = entity.Sentiment(sentiment)

	if intentName != nil && intentConfidence != nil {
		c.Intent = &entity.Intent{
			Name:       *intentName,
			Confidence: *intentConfidence,
		}
	}

	if err := json.Unmarshal(entities, &c.Entities); err != nil {
		c.Entities = make(map[string]interface{})
	}

	if err := json.Unmarshal(contextWindow, &c.ContextWindow); err != nil {
		c.ContextWindow = []entity.ContextMessage{}
	}

	if err := json.Unmarshal(state, &c.State); err != nil {
		c.State = make(map[string]interface{})
	}

	return &c, nil
}

// AIResponseRepository implements repository.AIResponseRepository
type AIResponseRepository struct {
	db *PostgresDB
}

// NewAIResponseRepository creates a new PostgreSQL AI response repository
func NewAIResponseRepository(db *PostgresDB) *AIResponseRepository {
	return &AIResponseRepository{db: db}
}

// Create creates a new AI response record
func (r *AIResponseRepository) Create(ctx context.Context, response *entity.AIResponse) error {
	prompt, err := json.Marshal(response.Prompt)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal prompt")
	}

	query := `
		INSERT INTO ai_responses (
			id, message_id, bot_id, prompt, response, confidence,
			tokens_used, latency_ms, model, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		response.ID,
		response.MessageID,
		response.BotID,
		prompt,
		response.Response,
		response.Confidence,
		response.TokensUsed,
		response.LatencyMs,
		response.Model,
		response.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create AI response")
	}

	return nil
}

// FindByID finds an AI response by ID
func (r *AIResponseRepository) FindByID(ctx context.Context, id string) (*entity.AIResponse, error) {
	query := `
		SELECT id, message_id, bot_id, prompt, response, confidence,
		       tokens_used, latency_ms, model, created_at
		FROM ai_responses
		WHERE id = $1
	`

	response, err := r.scanAIResponse(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "AI response not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find AI response")
	}

	return response, nil
}

// FindByMessage finds AI responses for a message
func (r *AIResponseRepository) FindByMessage(ctx context.Context, messageID string) ([]*entity.AIResponse, error) {
	query := `
		SELECT id, message_id, bot_id, prompt, response, confidence,
		       tokens_used, latency_ms, model, created_at
		FROM ai_responses
		WHERE message_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query AI responses")
	}
	defer rows.Close()

	var responses []*entity.AIResponse
	for rows.Next() {
		response, err := r.scanAIResponseFromRows(rows)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// FindByBot finds AI responses by bot with pagination
func (r *AIResponseRepository) FindByBot(ctx context.Context, botID string, params *repository.ListParams) ([]*entity.AIResponse, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM ai_responses WHERE bot_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, botID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count AI responses")
	}

	query := `
		SELECT id, message_id, bot_id, prompt, response, confidence,
		       tokens_used, latency_ms, model, created_at
		FROM ai_responses
		WHERE bot_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, botID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query AI responses")
	}
	defer rows.Close()

	var responses []*entity.AIResponse
	for rows.Next() {
		response, err := r.scanAIResponseFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, response)
	}

	return responses, total, nil
}

// CountByBot counts AI responses for a bot
func (r *AIResponseRepository) CountByBot(ctx context.Context, botID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM ai_responses WHERE bot_id = $1",
		botID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count AI responses")
	}

	return count, nil
}

// GetAverageLatency gets average latency for a bot
func (r *AIResponseRepository) GetAverageLatency(ctx context.Context, botID string) (float64, error) {
	var avgLatency float64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COALESCE(AVG(latency_ms), 0) FROM ai_responses WHERE bot_id = $1",
		botID,
	).Scan(&avgLatency)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to get average latency")
	}

	return avgLatency, nil
}

// GetTotalTokensUsed gets total tokens used by a bot
func (r *AIResponseRepository) GetTotalTokensUsed(ctx context.Context, botID string) (int64, error) {
	var totalTokens int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COALESCE(SUM(tokens_used), 0) FROM ai_responses WHERE bot_id = $1",
		botID,
	).Scan(&totalTokens)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to get total tokens used")
	}

	return totalTokens, nil
}

func (r *AIResponseRepository) scanAIResponse(row pgx.Row) (*entity.AIResponse, error) {
	var ar entity.AIResponse
	var prompt []byte

	err := row.Scan(
		&ar.ID, &ar.MessageID, &ar.BotID, &prompt, &ar.Response, &ar.Confidence,
		&ar.TokensUsed, &ar.LatencyMs, &ar.Model, &ar.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(prompt, &ar.Prompt); err != nil {
		ar.Prompt = make(map[string]interface{})
	}

	return &ar, nil
}

func (r *AIResponseRepository) scanAIResponseFromRows(rows pgx.Rows) (*entity.AIResponse, error) {
	var ar entity.AIResponse
	var prompt []byte

	err := rows.Scan(
		&ar.ID, &ar.MessageID, &ar.BotID, &prompt, &ar.Response, &ar.Confidence,
		&ar.TokensUsed, &ar.LatencyMs, &ar.Model, &ar.CreatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan AI response")
	}

	if err := json.Unmarshal(prompt, &ar.Prompt); err != nil {
		ar.Prompt = make(map[string]interface{})
	}

	return &ar, nil
}
