package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ChannelRepository implements repository.ChannelRepository with PostgreSQL
type ChannelRepository struct {
	db *PostgresDB
}

// NewChannelRepository creates a new PostgreSQL channel repository
func NewChannelRepository(db *PostgresDB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// Create creates a new channel
func (r *ChannelRepository) Create(ctx context.Context, channel *entity.Channel) error {
	credentials, err := json.Marshal(channel.Credentials)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal credentials")
	}

	config, err := json.Marshal(channel.Config)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal config")
	}

	query := `
		INSERT INTO channels (
			id, tenant_id, name, type, enabled, connection_status, credentials, config,
			webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		channel.ID,
		channel.TenantID,
		channel.Name,
		string(channel.Type),
		channel.Enabled,
		string(channel.ConnectionStatus),
		credentials,
		config,
		nullString(channel.WebhookURL),
		channel.IsCoexistence,
		nullString(channel.WABAID),
		channel.LastEchoAt,
		string(channel.CoexistenceStatus),
		channel.CreatedAt,
		channel.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create channel")
	}

	return nil
}

// FindByID finds a channel by ID
func (r *ChannelRepository) FindByID(ctx context.Context, id string) (*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE id = $1
	`

	channel, err := r.scanChannel(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find channel")
	}

	return channel, nil
}

// FindByTenant finds channels for a tenant with pagination
func (r *ChannelRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Channel, int64, error) {
	// Use default params if nil
	if params == nil {
		params = repository.NewListParams()
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM channels WHERE tenant_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count channels")
	}

	// Get channels
	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE tenant_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeChannelColumn(params.SortBy), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, tenantID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query channels")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		channels = append(channels, channel)
	}

	return channels, total, nil
}

// FindByType finds channels of a specific type for a tenant
func (r *ChannelRepository) FindByType(ctx context.Context, tenantID string, channelType entity.ChannelType) ([]*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE tenant_id = $1 AND type = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, string(channelType))
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query channels")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// FindEnabledByTenant finds enabled channels for a tenant
func (r *ChannelRepository) FindEnabledByTenant(ctx context.Context, tenantID string) ([]*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE tenant_id = $1 AND enabled = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query channels")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// FindActiveByTenant finds channels that are both enabled AND connected
func (r *ChannelRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE tenant_id = $1 AND enabled = true AND connection_status = 'connected'
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query channels")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// Update updates a channel
func (r *ChannelRepository) Update(ctx context.Context, channel *entity.Channel) error {
	channel.UpdatedAt = time.Now()

	credentials, err := json.Marshal(channel.Credentials)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal credentials")
	}

	config, err := json.Marshal(channel.Config)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal config")
	}

	query := `
		UPDATE channels SET
			name = $1,
			enabled = $2,
			connection_status = $3,
			credentials = $4,
			config = $5,
			webhook_url = $6,
			is_coexistence = $7,
			waba_id = $8,
			last_echo_at = $9,
			coexistence_status = $10,
			updated_at = $11
		WHERE id = $12
	`

	result, err := r.db.Pool.Exec(ctx, query,
		channel.Name,
		channel.Enabled,
		string(channel.ConnectionStatus),
		credentials,
		config,
		nullString(channel.WebhookURL),
		channel.IsCoexistence,
		nullString(channel.WABAID),
		channel.LastEchoAt,
		nullString(string(channel.CoexistenceStatus)),
		channel.UpdatedAt,
		channel.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update channel")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	return nil
}

// UpdateEnabled updates only the channel enabled state
func (r *ChannelRepository) UpdateEnabled(ctx context.Context, id string, enabled bool) error {
	query := `UPDATE channels SET enabled = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, enabled, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update channel enabled state")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	return nil
}

// UpdateConnectionStatus updates only the channel connection status
func (r *ChannelRepository) UpdateConnectionStatus(ctx context.Context, id string, status entity.ConnectionStatus) error {
	query := `UPDATE channels SET connection_status = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, string(status), time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update channel connection status")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	return nil
}

// UpdateStatus updates the channel status (deprecated, use UpdateEnabled or UpdateConnectionStatus)
func (r *ChannelRepository) UpdateStatus(ctx context.Context, id string, status entity.ChannelStatus) error {
	// For backwards compatibility, map old status to new fields
	return r.UpdateConnectionStatus(ctx, id, entity.ConnectionStatus(status))
}

// Delete deletes a channel
func (r *ChannelRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM channels WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete channel")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	return nil
}

// CountByTenant counts channels for a tenant
func (r *ChannelRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM channels WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count channels")
	}

	return count, nil
}

// FindByTypes finds all channels of specific types across all tenants
func (r *ChannelRepository) FindByTypes(ctx context.Context, types []entity.ChannelType) ([]*entity.Channel, error) {
	if len(types) == 0 {
		return nil, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(types))
	args := make([]interface{}, len(types))
	for i, t := range types {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = string(t)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE type IN (%s)
		ORDER BY created_at DESC
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query channels by types")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// FindByConfigValue finds channels that have a specific config key-value pair
// Useful for finding WhatsApp channels by phone_number_id
func (r *ChannelRepository) FindByConfigValue(ctx context.Context, key, value string) ([]*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE config->>$1 = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, key, value)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query channels by config")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// FindWhatsAppByPhoneNumberID finds a WhatsApp channel by its phone number ID
func (r *ChannelRepository) FindWhatsAppByPhoneNumberID(ctx context.Context, phoneNumberID string) (*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE (type = 'whatsapp' OR type = 'whatsapp_official')
		  AND config->>'phone_number_id' = $1
		LIMIT 1
	`

	channel, err := r.scanChannel(r.db.Pool.QueryRow(ctx, query, phoneNumberID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeChannelNotFound, "WhatsApp channel not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find WhatsApp channel")
	}

	return channel, nil
}

// Helper methods

func (r *ChannelRepository) scanChannel(row pgx.Row) (*entity.Channel, error) {
	var c entity.Channel
	var channelType, connectionStatus string
	var credentials, config []byte
	var webhookURL, wabaID, coexistenceStatus *string

	err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &channelType, &c.Enabled, &connectionStatus,
		&credentials, &config, &webhookURL, &c.IsCoexistence, &wabaID,
		&c.LastEchoAt, &coexistenceStatus, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.Type = entity.ChannelType(channelType)
	c.ConnectionStatus = entity.ConnectionStatus(connectionStatus)

	if webhookURL != nil {
		c.WebhookURL = *webhookURL
	}
	if wabaID != nil {
		c.WABAID = *wabaID
	}
	if coexistenceStatus != nil {
		c.CoexistenceStatus = entity.CoexistenceStatus(*coexistenceStatus)
	}

	if err := json.Unmarshal(credentials, &c.Credentials); err != nil {
		c.Credentials = make(map[string]string)
	}

	if err := json.Unmarshal(config, &c.Config); err != nil {
		c.Config = make(map[string]string)
	}

	return &c, nil
}

func (r *ChannelRepository) scanChannelFromRows(rows pgx.Rows) (*entity.Channel, error) {
	var c entity.Channel
	var channelType, connectionStatus string
	var credentials, config []byte
	var webhookURL, wabaID, coexistenceStatus *string

	err := rows.Scan(
		&c.ID, &c.TenantID, &c.Name, &channelType, &c.Enabled, &connectionStatus,
		&credentials, &config, &webhookURL, &c.IsCoexistence, &wabaID,
		&c.LastEchoAt, &coexistenceStatus, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan channel")
	}

	c.Type = entity.ChannelType(channelType)
	c.ConnectionStatus = entity.ConnectionStatus(connectionStatus)

	if webhookURL != nil {
		c.WebhookURL = *webhookURL
	}
	if wabaID != nil {
		c.WABAID = *wabaID
	}
	if coexistenceStatus != nil {
		c.CoexistenceStatus = entity.CoexistenceStatus(*coexistenceStatus)
	}

	if err := json.Unmarshal(credentials, &c.Credentials); err != nil {
		c.Credentials = make(map[string]string)
	}

	if err := json.Unmarshal(config, &c.Config); err != nil {
		c.Config = make(map[string]string)
	}

	return &c, nil
}

// FindCoexistenceChannels finds all channels with coexistence enabled
func (r *ChannelRepository) FindCoexistenceChannels(ctx context.Context) ([]*entity.Channel, error) {
	query := `
		SELECT id, tenant_id, name, type, enabled, connection_status, credentials, config,
		       webhook_url, is_coexistence, waba_id, last_echo_at, coexistence_status,
		       created_at, updated_at
		FROM channels
		WHERE is_coexistence = true
		ORDER BY last_echo_at ASC NULLS FIRST
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query coexistence channels")
	}
	defer rows.Close()

	var channels []*entity.Channel
	for rows.Next() {
		channel, err := r.scanChannelFromRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

func sanitizeChannelColumn(col string) string {
	allowed := map[string]bool{
		"created_at":        true,
		"updated_at":        true,
		"name":              true,
		"type":              true,
		"enabled":           true,
		"connection_status": true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}
