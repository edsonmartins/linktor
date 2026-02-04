package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ConversationRepository implements repository.ConversationRepository with PostgreSQL
type ConversationRepository struct {
	db *PostgresDB
}

// NewConversationRepository creates a new PostgreSQL conversation repository
func NewConversationRepository(db *PostgresDB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Create creates a new conversation
func (r *ConversationRepository) Create(ctx context.Context, conversation *entity.Conversation) error {
	query := `
		INSERT INTO conversations (
			id, tenant_id, channel_id, contact_id, assignee_id, status, priority,
			subject, unread_count, first_reply_at, resolved_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		conversation.ID,
		conversation.TenantID,
		conversation.ChannelID,
		conversation.ContactID,
		conversation.AssignedUserID,
		string(conversation.Status),
		string(conversation.Priority),
		nullString(conversation.Subject),
		conversation.UnreadCount,
		conversation.FirstReplyAt,
		conversation.ResolvedAt,
		conversation.CreatedAt,
		conversation.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create conversation")
	}

	return nil
}

// FindByID finds a conversation by ID
func (r *ConversationRepository) FindByID(ctx context.Context, id string) (*entity.Conversation, error) {
	query := `
		SELECT id, tenant_id, channel_id, contact_id, assignee_id, status, priority,
		       subject, unread_count, first_reply_at, resolved_at, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	conversation, err := r.scanConversation(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find conversation")
	}

	return conversation, nil
}

// FindByTenant finds conversations for a tenant with pagination
func (r *ConversationRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	return r.findWithFilter(ctx, "tenant_id = $1", []interface{}{tenantID}, params)
}

// FindByChannel finds conversations for a channel
func (r *ConversationRepository) FindByChannel(ctx context.Context, channelID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	return r.findWithFilter(ctx, "channel_id = $1", []interface{}{channelID}, params)
}

// FindByContact finds conversations for a contact
func (r *ConversationRepository) FindByContact(ctx context.Context, contactID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	return r.findWithFilter(ctx, "contact_id = $1", []interface{}{contactID}, params)
}

// FindByAssignee finds conversations assigned to a user
func (r *ConversationRepository) FindByAssignee(ctx context.Context, assigneeID string, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	return r.findWithFilter(ctx, "assignee_id = $1", []interface{}{assigneeID}, params)
}

// FindOpenByContactAndChannel finds open conversation for a contact on a channel
func (r *ConversationRepository) FindOpenByContactAndChannel(ctx context.Context, contactID, channelID string) (*entity.Conversation, error) {
	query := `
		SELECT id, tenant_id, channel_id, contact_id, assignee_id, status, priority,
		       subject, unread_count, first_reply_at, resolved_at, created_at, updated_at
		FROM conversations
		WHERE contact_id = $1 AND channel_id = $2 AND status IN ('open', 'pending')
		ORDER BY created_at DESC
		LIMIT 1
	`

	conversation, err := r.scanConversation(r.db.Pool.QueryRow(ctx, query, contactID, channelID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No open conversation found
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find conversation")
	}

	return conversation, nil
}

// Update updates a conversation
func (r *ConversationRepository) Update(ctx context.Context, conversation *entity.Conversation) error {
	conversation.UpdatedAt = time.Now()

	query := `
		UPDATE conversations SET
			assignee_id = $1,
			status = $2,
			priority = $3,
			subject = $4,
			unread_count = $5,
			first_reply_at = $6,
			resolved_at = $7,
			updated_at = $8
		WHERE id = $9
	`

	result, err := r.db.Pool.Exec(ctx, query,
		conversation.AssignedUserID,
		string(conversation.Status),
		string(conversation.Priority),
		nullString(conversation.Subject),
		conversation.UnreadCount,
		conversation.FirstReplyAt,
		conversation.ResolvedAt,
		conversation.UpdatedAt,
		conversation.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update conversation")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	return nil
}

// UpdateStatus updates only the conversation status
func (r *ConversationRepository) UpdateStatus(ctx context.Context, id string, status entity.ConversationStatus) error {
	now := time.Now()

	var result interface{ RowsAffected() int64 }
	var err error

	if status == entity.ConversationStatusResolved || status == entity.ConversationStatusClosed {
		query := `UPDATE conversations SET status = $1, resolved_at = $2, updated_at = $3 WHERE id = $4`
		result, err = r.db.Pool.Exec(ctx, query, string(status), now, now, id)
	} else {
		query := `UPDATE conversations SET status = $1, updated_at = $2 WHERE id = $3`
		result, err = r.db.Pool.Exec(ctx, query, string(status), now, id)
	}

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update conversation status")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	return nil
}

// UpdateAssignee updates the conversation assignee
func (r *ConversationRepository) UpdateAssignee(ctx context.Context, id string, assigneeID *string) error {
	query := `UPDATE conversations SET assignee_id = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, assigneeID, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update conversation assignee")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	return nil
}

// IncrementUnreadCount increments the unread message count
func (r *ConversationRepository) IncrementUnreadCount(ctx context.Context, id string) error {
	query := `UPDATE conversations SET unread_count = unread_count + 1, updated_at = $1 WHERE id = $2`

	result, err := r.db.Pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to increment unread count")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	return nil
}

// ResetUnreadCount resets the unread message count to zero
func (r *ConversationRepository) ResetUnreadCount(ctx context.Context, id string) error {
	query := `UPDATE conversations SET unread_count = 0, updated_at = $1 WHERE id = $2`

	result, err := r.db.Pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to reset unread count")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	return nil
}

// Delete deletes a conversation
func (r *ConversationRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM conversations WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete conversation")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	return nil
}

// CountByTenant counts conversations for a tenant
func (r *ConversationRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM conversations WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count conversations")
	}

	return count, nil
}

// CountByStatus counts conversations by status for a tenant
func (r *ConversationRepository) CountByStatus(ctx context.Context, tenantID string, status entity.ConversationStatus) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM conversations WHERE tenant_id = $1 AND status = $2",
		tenantID, string(status),
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count conversations by status")
	}

	return count, nil
}

// Helper methods

func (r *ConversationRepository) findWithFilter(ctx context.Context, whereClause string, args []interface{}, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM conversations WHERE %s", whereClause)
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count conversations")
	}

	// Apply filters
	whereClause, args = applyConversationFilters(whereClause, args, params.Filters)

	// Get conversations
	query := fmt.Sprintf(`
		SELECT id, tenant_id, channel_id, contact_id, assignee_id, status, priority,
		       subject, unread_count, first_reply_at, resolved_at, created_at, updated_at
		FROM conversations
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sanitizeConversationColumn(params.SortBy), sanitizeDirection(params.SortDir), len(args)+1, len(args)+2)

	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query conversations")
	}
	defer rows.Close()

	var conversations []*entity.Conversation
	for rows.Next() {
		conversation, err := r.scanConversationFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		conversations = append(conversations, conversation)
	}

	return conversations, total, nil
}

func (r *ConversationRepository) scanConversation(row pgx.Row) (*entity.Conversation, error) {
	var c entity.Conversation
	var assigneeID, subject *string
	var status, priority string

	err := row.Scan(
		&c.ID, &c.TenantID, &c.ChannelID, &c.ContactID, &assigneeID, &status, &priority,
		&subject, &c.UnreadCount, &c.FirstReplyAt, &c.ResolvedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.Status = entity.ConversationStatus(status)
	c.Priority = entity.ConversationPriority(priority)
	c.AssignedUserID = assigneeID

	if subject != nil {
		c.Subject = *subject
	}

	return &c, nil
}

func (r *ConversationRepository) scanConversationFromRows(rows pgx.Rows) (*entity.Conversation, error) {
	var c entity.Conversation
	var assigneeID, subject *string
	var status, priority string

	err := rows.Scan(
		&c.ID, &c.TenantID, &c.ChannelID, &c.ContactID, &assigneeID, &status, &priority,
		&subject, &c.UnreadCount, &c.FirstReplyAt, &c.ResolvedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan conversation")
	}

	c.Status = entity.ConversationStatus(status)
	c.Priority = entity.ConversationPriority(priority)
	c.AssignedUserID = assigneeID

	if subject != nil {
		c.Subject = *subject
	}

	return &c, nil
}

func sanitizeConversationColumn(col string) string {
	allowed := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"status":        true,
		"priority":      true,
		"unread_count":  true,
		"first_reply_at": true,
	}
	if allowed[col] {
		return col
	}
	return "updated_at"
}

func applyConversationFilters(whereClause string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	if status, ok := filters["status"].(string); ok && status != "" {
		args = append(args, status)
		whereClause += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		args = append(args, priority)
		whereClause += fmt.Sprintf(" AND priority = $%d", len(args))
	}
	if assigneeID, ok := filters["assignee_id"].(string); ok && assigneeID != "" {
		if assigneeID == "unassigned" {
			whereClause += " AND assignee_id IS NULL"
		} else {
			args = append(args, assigneeID)
			whereClause += fmt.Sprintf(" AND assignee_id = $%d", len(args))
		}
	}
	return whereClause, args
}

// CountActiveByUser counts active conversations assigned to a user
func (r *ConversationRepository) CountActiveByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM conversations
		 WHERE assignee_id = $1
		 AND status IN ('open', 'pending')`,
		userID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count active conversations by user")
	}

	return count, nil
}

// CountWaiting counts waiting (pending) conversations with given or higher priority
func (r *ConversationRepository) CountWaiting(ctx context.Context, tenantID string, minPriority entity.ConversationPriority) (int64, error) {
	// Priority order: urgent > high > normal > low
	priorityValues := map[entity.ConversationPriority]int{
		entity.ConversationPriorityUrgent: 4,
		entity.ConversationPriorityHigh:   3,
		entity.ConversationPriorityNormal: 2,
		entity.ConversationPriorityLow:    1,
	}

	minValue := priorityValues[minPriority]
	if minValue == 0 {
		minValue = 2 // default to normal
	}

	// Build list of priorities with higher or equal value
	var priorities []string
	for p, v := range priorityValues {
		if v >= minValue {
			priorities = append(priorities, string(p))
		}
	}

	var count int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM conversations
		 WHERE tenant_id = $1
		 AND status = 'pending'
		 AND assignee_id IS NULL
		 AND priority = ANY($2)`,
		tenantID,
		priorities,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count waiting conversations")
	}

	return count, nil
}
