package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// HistoryImportRepository implements repository.HistoryImportRepository with PostgreSQL
type HistoryImportRepository struct {
	db *PostgresDB
}

// NewHistoryImportRepository creates a new PostgreSQL history import repository
func NewHistoryImportRepository(db *PostgresDB) *HistoryImportRepository {
	return &HistoryImportRepository{db: db}
}

// Create creates a new history import job
func (r *HistoryImportRepository) Create(ctx context.Context, importJob *entity.HistoryImport) error {
	errorDetails, err := json.Marshal(importJob.ErrorDetails)
	if err != nil {
		errorDetails = []byte("{}")
	}

	query := `
		INSERT INTO whatsapp_history_imports (
			id, channel_id, tenant_id, status,
			total_conversations, imported_conversations,
			total_messages, imported_messages,
			total_contacts, imported_contacts,
			started_at, completed_at,
			error_message, error_details,
			import_since, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		importJob.ID,
		importJob.ChannelID,
		importJob.TenantID,
		string(importJob.Status),
		importJob.TotalConversations,
		importJob.ImportedConversations,
		importJob.TotalMessages,
		importJob.ImportedMessages,
		importJob.TotalContacts,
		importJob.ImportedContacts,
		importJob.StartedAt,
		importJob.CompletedAt,
		nullString(importJob.ErrorMessage),
		errorDetails,
		importJob.ImportSince,
		importJob.CreatedAt,
		importJob.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create history import")
	}

	return nil
}

// FindByID finds a history import by ID
func (r *HistoryImportRepository) FindByID(ctx context.Context, id string) (*entity.HistoryImport, error) {
	query := `
		SELECT id, channel_id, tenant_id, status,
			total_conversations, imported_conversations,
			total_messages, imported_messages,
			total_contacts, imported_contacts,
			started_at, completed_at,
			error_message, error_details,
			import_since, created_at, updated_at
		FROM whatsapp_history_imports
		WHERE id = $1
	`

	row := r.db.Pool.QueryRow(ctx, query, id)
	return r.scanHistoryImport(row)
}

// FindByChannelID finds all imports for a channel
func (r *HistoryImportRepository) FindByChannelID(ctx context.Context, channelID string) ([]*entity.HistoryImport, error) {
	query := `
		SELECT id, channel_id, tenant_id, status,
			total_conversations, imported_conversations,
			total_messages, imported_messages,
			total_contacts, imported_contacts,
			started_at, completed_at,
			error_message, error_details,
			import_since, created_at, updated_at
		FROM whatsapp_history_imports
		WHERE channel_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, channelID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query history imports")
	}
	defer rows.Close()

	return r.scanHistoryImports(rows)
}

// FindByTenantID finds all imports for a tenant with pagination
func (r *HistoryImportRepository) FindByTenantID(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.HistoryImport, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM whatsapp_history_imports WHERE tenant_id = $1`
	var total int64
	err := r.db.Pool.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count history imports")
	}

	// Get paginated results
	query := `
		SELECT id, channel_id, tenant_id, status,
			total_conversations, imported_conversations,
			total_messages, imported_messages,
			total_contacts, imported_contacts,
			started_at, completed_at,
			error_message, error_details,
			import_since, created_at, updated_at
		FROM whatsapp_history_imports
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	limit := 20
	offset := 0
	if params != nil {
		limit = params.Limit()
		offset = params.Offset()
	}

	rows, err := r.db.Pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query history imports")
	}
	defer rows.Close()

	imports, err := r.scanHistoryImports(rows)
	if err != nil {
		return nil, 0, err
	}

	return imports, total, nil
}

// FindRunning finds all currently running imports
func (r *HistoryImportRepository) FindRunning(ctx context.Context) ([]*entity.HistoryImport, error) {
	query := `
		SELECT id, channel_id, tenant_id, status,
			total_conversations, imported_conversations,
			total_messages, imported_messages,
			total_contacts, imported_contacts,
			started_at, completed_at,
			error_message, error_details,
			import_since, created_at, updated_at
		FROM whatsapp_history_imports
		WHERE status = $1
		ORDER BY started_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, string(entity.HistoryImportStatusInProgress))
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query running imports")
	}
	defer rows.Close()

	return r.scanHistoryImports(rows)
}

// Update updates a history import job
func (r *HistoryImportRepository) Update(ctx context.Context, importJob *entity.HistoryImport) error {
	importJob.UpdatedAt = time.Now()

	errorDetails, err := json.Marshal(importJob.ErrorDetails)
	if err != nil {
		errorDetails = []byte("{}")
	}

	query := `
		UPDATE whatsapp_history_imports SET
			status = $1,
			total_conversations = $2,
			imported_conversations = $3,
			total_messages = $4,
			imported_messages = $5,
			total_contacts = $6,
			imported_contacts = $7,
			started_at = $8,
			completed_at = $9,
			error_message = $10,
			error_details = $11,
			updated_at = $12
		WHERE id = $13
	`

	result, err := r.db.Pool.Exec(ctx, query,
		string(importJob.Status),
		importJob.TotalConversations,
		importJob.ImportedConversations,
		importJob.TotalMessages,
		importJob.ImportedMessages,
		importJob.TotalContacts,
		importJob.ImportedContacts,
		importJob.StartedAt,
		importJob.CompletedAt,
		nullString(importJob.ErrorMessage),
		errorDetails,
		importJob.UpdatedAt,
		importJob.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update history import")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "history import not found")
	}

	return nil
}

// Delete deletes a history import job
func (r *HistoryImportRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM whatsapp_history_imports WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete history import")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "history import not found")
	}

	return nil
}

// scanHistoryImport scans a single row into a HistoryImport entity
func (r *HistoryImportRepository) scanHistoryImport(row pgx.Row) (*entity.HistoryImport, error) {
	var importJob entity.HistoryImport
	var status string
	var errorMessage *string
	var errorDetails []byte

	err := row.Scan(
		&importJob.ID,
		&importJob.ChannelID,
		&importJob.TenantID,
		&status,
		&importJob.TotalConversations,
		&importJob.ImportedConversations,
		&importJob.TotalMessages,
		&importJob.ImportedMessages,
		&importJob.TotalContacts,
		&importJob.ImportedContacts,
		&importJob.StartedAt,
		&importJob.CompletedAt,
		&errorMessage,
		&errorDetails,
		&importJob.ImportSince,
		&importJob.CreatedAt,
		&importJob.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "history import not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan history import")
	}

	importJob.Status = entity.HistoryImportStatus(status)
	if errorMessage != nil {
		importJob.ErrorMessage = *errorMessage
	}
	if len(errorDetails) > 0 {
		json.Unmarshal(errorDetails, &importJob.ErrorDetails)
	}

	return &importJob, nil
}

// scanHistoryImports scans multiple rows into HistoryImport entities
func (r *HistoryImportRepository) scanHistoryImports(rows pgx.Rows) ([]*entity.HistoryImport, error) {
	var imports []*entity.HistoryImport

	for rows.Next() {
		var importJob entity.HistoryImport
		var status string
		var errorMessage *string
		var errorDetails []byte

		err := rows.Scan(
			&importJob.ID,
			&importJob.ChannelID,
			&importJob.TenantID,
			&status,
			&importJob.TotalConversations,
			&importJob.ImportedConversations,
			&importJob.TotalMessages,
			&importJob.ImportedMessages,
			&importJob.TotalContacts,
			&importJob.ImportedContacts,
			&importJob.StartedAt,
			&importJob.CompletedAt,
			&errorMessage,
			&errorDetails,
			&importJob.ImportSince,
			&importJob.CreatedAt,
			&importJob.UpdatedAt,
		)

		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan history import")
		}

		importJob.Status = entity.HistoryImportStatus(status)
		if errorMessage != nil {
			importJob.ErrorMessage = *errorMessage
		}
		if len(errorDetails) > 0 {
			json.Unmarshal(errorDetails, &importJob.ErrorDetails)
		}

		imports = append(imports, &importJob)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "error iterating history imports")
	}

	return imports, nil
}
