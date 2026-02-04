package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// MessageRepository implements repository.MessageRepository with PostgreSQL
type MessageRepository struct {
	db *PostgresDB
}

// NewMessageRepository creates a new PostgreSQL message repository
func NewMessageRepository(db *PostgresDB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
func (r *MessageRepository) Create(ctx context.Context, message *entity.Message) error {
	metadata, err := json.Marshal(message.Metadata)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal metadata")
	}

	query := `
		INSERT INTO messages (
			id, conversation_id, sender_type, sender_id, content_type, content,
			metadata, status, external_id, error_message, sent_at, delivered_at,
			read_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	var senderID *string
	if message.SenderID != "" {
		senderID = &message.SenderID
	}

	_, err = r.db.Pool.Exec(ctx, query,
		message.ID,
		message.ConversationID,
		string(message.SenderType),
		senderID,
		string(message.ContentType),
		message.Content,
		metadata,
		string(message.Status),
		nullString(message.ExternalID),
		nullString(message.ErrorMessage),
		message.SentAt,
		message.DeliveredAt,
		message.ReadAt,
		message.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create message")
	}

	return nil
}

// FindByID finds a message by ID
func (r *MessageRepository) FindByID(ctx context.Context, id string) (*entity.Message, error) {
	query := `
		SELECT id, conversation_id, sender_type, sender_id, content_type, content,
		       metadata, status, external_id, error_message, sent_at, delivered_at,
		       read_at, created_at
		FROM messages
		WHERE id = $1
	`

	message, err := r.scanMessage(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeMessageNotFound, "message not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find message")
	}

	// Load attachments
	attachments, err := r.FindAttachmentsByMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	message.Attachments = attachments

	return message, nil
}

// FindByExternalID finds a message by external ID
func (r *MessageRepository) FindByExternalID(ctx context.Context, externalID string) (*entity.Message, error) {
	query := `
		SELECT id, conversation_id, sender_type, sender_id, content_type, content,
		       metadata, status, external_id, error_message, sent_at, delivered_at,
		       read_at, created_at
		FROM messages
		WHERE external_id = $1
	`

	message, err := r.scanMessage(r.db.Pool.QueryRow(ctx, query, externalID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeMessageNotFound, "message not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find message")
	}

	return message, nil
}

// FindByConversation finds messages for a conversation with pagination
func (r *MessageRepository) FindByConversation(ctx context.Context, conversationID string, params *repository.ListParams) ([]*entity.Message, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, conversationID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count messages")
	}

	// Get messages
	query := fmt.Sprintf(`
		SELECT id, conversation_id, sender_type, sender_id, content_type, content,
		       metadata, status, external_id, error_message, sent_at, delivered_at,
		       read_at, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeColumn(params.SortBy, "created_at"), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, conversationID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query messages")
	}
	defer rows.Close()

	var messages []*entity.Message
	for rows.Next() {
		message, err := r.scanMessageFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		messages = append(messages, message)
	}

	return messages, total, nil
}

// Update updates a message
func (r *MessageRepository) Update(ctx context.Context, message *entity.Message) error {
	metadata, err := json.Marshal(message.Metadata)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal metadata")
	}

	query := `
		UPDATE messages SET
			content_type = $1,
			content = $2,
			metadata = $3,
			status = $4,
			external_id = $5,
			error_message = $6,
			sent_at = $7,
			delivered_at = $8,
			read_at = $9
		WHERE id = $10
	`

	result, err := r.db.Pool.Exec(ctx, query,
		string(message.ContentType),
		message.Content,
		metadata,
		string(message.Status),
		nullString(message.ExternalID),
		nullString(message.ErrorMessage),
		message.SentAt,
		message.DeliveredAt,
		message.ReadAt,
		message.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update message")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeMessageNotFound, "message not found")
	}

	return nil
}

// UpdateStatus updates only the message status
func (r *MessageRepository) UpdateStatus(ctx context.Context, id string, status entity.MessageStatus, errorMessage string) error {
	var query string
	var args []interface{}

	now := time.Now()

	switch status {
	case entity.MessageStatusSent:
		query = `UPDATE messages SET status = $1, sent_at = $2 WHERE id = $3`
		args = []interface{}{string(status), now, id}
	case entity.MessageStatusDelivered:
		query = `UPDATE messages SET status = $1, delivered_at = $2 WHERE id = $3`
		args = []interface{}{string(status), now, id}
	case entity.MessageStatusRead:
		query = `UPDATE messages SET status = $1, read_at = $2 WHERE id = $3`
		args = []interface{}{string(status), now, id}
	case entity.MessageStatusFailed:
		query = `UPDATE messages SET status = $1, error_message = $2 WHERE id = $3`
		args = []interface{}{string(status), errorMessage, id}
	default:
		query = `UPDATE messages SET status = $1 WHERE id = $2`
		args = []interface{}{string(status), id}
	}

	result, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update message status")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeMessageNotFound, "message not found")
	}

	return nil
}

// Delete deletes a message
func (r *MessageRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM messages WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete message")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeMessageNotFound, "message not found")
	}

	return nil
}

// CountByConversation counts messages in a conversation
func (r *MessageRepository) CountByConversation(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM messages WHERE conversation_id = $1",
		conversationID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count messages")
	}

	return count, nil
}

// CountUnreadByConversation counts unread messages in a conversation
func (r *MessageRepository) CountUnreadByConversation(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM messages WHERE conversation_id = $1 AND status != 'read' AND sender_type = 'contact'",
		conversationID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count unread messages")
	}

	return count, nil
}

// MarkAsRead marks messages as read up to a given message ID
func (r *MessageRepository) MarkAsRead(ctx context.Context, conversationID string, upToMessageID string) error {
	now := time.Now()
	query := `
		UPDATE messages
		SET status = 'read', read_at = $1
		WHERE conversation_id = $2
		  AND status != 'read'
		  AND created_at <= (SELECT created_at FROM messages WHERE id = $3)
	`

	_, err := r.db.Pool.Exec(ctx, query, now, conversationID, upToMessageID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to mark messages as read")
	}

	return nil
}

// CreateAttachment creates a message attachment
func (r *MessageRepository) CreateAttachment(ctx context.Context, attachment *entity.MessageAttachment) error {
	metadata, err := json.Marshal(attachment.Metadata)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal metadata")
	}

	query := `
		INSERT INTO message_attachments (
			id, message_id, type, filename, mime_type, size_bytes,
			url, thumbnail_url, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		attachment.ID,
		attachment.MessageID,
		attachment.Type,
		nullString(attachment.Filename),
		nullString(attachment.MimeType),
		attachment.SizeBytes,
		attachment.URL,
		nullString(attachment.ThumbnailURL),
		metadata,
		attachment.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create attachment")
	}

	return nil
}

// FindAttachmentsByMessage finds attachments for a message
func (r *MessageRepository) FindAttachmentsByMessage(ctx context.Context, messageID string) ([]*entity.MessageAttachment, error) {
	query := `
		SELECT id, message_id, type, filename, mime_type, size_bytes,
		       url, thumbnail_url, metadata, created_at
		FROM message_attachments
		WHERE message_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.Pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query attachments")
	}
	defer rows.Close()

	var attachments []*entity.MessageAttachment
	for rows.Next() {
		var a entity.MessageAttachment
		var metadata []byte
		var filename, mimeType, thumbnailURL *string

		err := rows.Scan(
			&a.ID, &a.MessageID, &a.Type, &filename, &mimeType,
			&a.SizeBytes, &a.URL, &thumbnailURL, &metadata, &a.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan attachment")
		}

		if filename != nil {
			a.Filename = *filename
		}
		if mimeType != nil {
			a.MimeType = *mimeType
		}
		if thumbnailURL != nil {
			a.ThumbnailURL = *thumbnailURL
		}

		if err := json.Unmarshal(metadata, &a.Metadata); err != nil {
			a.Metadata = make(map[string]string)
		}

		attachments = append(attachments, &a)
	}

	return attachments, nil
}

// scanMessage scans a single message row
func (r *MessageRepository) scanMessage(row pgx.Row) (*entity.Message, error) {
	var m entity.Message
	var senderID, externalID, errorMessage *string
	var metadata []byte
	var senderType, contentType, status string

	err := row.Scan(
		&m.ID, &m.ConversationID, &senderType, &senderID, &contentType, &m.Content,
		&metadata, &status, &externalID, &errorMessage, &m.SentAt, &m.DeliveredAt,
		&m.ReadAt, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	m.SenderType = entity.SenderType(senderType)
	m.ContentType = entity.ContentType(contentType)
	m.Status = entity.MessageStatus(status)

	if senderID != nil {
		m.SenderID = *senderID
	}
	if externalID != nil {
		m.ExternalID = *externalID
	}
	if errorMessage != nil {
		m.ErrorMessage = *errorMessage
	}

	if err := json.Unmarshal(metadata, &m.Metadata); err != nil {
		m.Metadata = make(map[string]string)
	}

	return &m, nil
}

// scanMessageFromRows scans a message from rows
func (r *MessageRepository) scanMessageFromRows(rows pgx.Rows) (*entity.Message, error) {
	var m entity.Message
	var senderID, externalID, errorMessage *string
	var metadata []byte
	var senderType, contentType, status string

	err := rows.Scan(
		&m.ID, &m.ConversationID, &senderType, &senderID, &contentType, &m.Content,
		&metadata, &status, &externalID, &errorMessage, &m.SentAt, &m.DeliveredAt,
		&m.ReadAt, &m.CreatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan message")
	}

	m.SenderType = entity.SenderType(senderType)
	m.ContentType = entity.ContentType(contentType)
	m.Status = entity.MessageStatus(status)

	if senderID != nil {
		m.SenderID = *senderID
	}
	if externalID != nil {
		m.ExternalID = *externalID
	}
	if errorMessage != nil {
		m.ErrorMessage = *errorMessage
	}

	if err := json.Unmarshal(metadata, &m.Metadata); err != nil {
		m.Metadata = make(map[string]string)
	}

	return &m, nil
}

// Helper functions

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func sanitizeColumn(col, defaultCol string) string {
	allowed := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"sent_at":      true,
		"delivered_at": true,
		"read_at":      true,
		"status":       true,
	}
	if allowed[col] {
		return col
	}
	return defaultCol
}

func sanitizeDirection(dir string) string {
	if dir == "asc" || dir == "ASC" {
		return "ASC"
	}
	return "DESC"
}
