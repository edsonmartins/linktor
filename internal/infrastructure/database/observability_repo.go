package database

import (
	"context"
	"encoding/json"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// ObservabilityRepository implements repository.ObservabilityRepository with PostgreSQL
type ObservabilityRepository struct {
	db *PostgresDB
}

// NewObservabilityRepository creates a new observability repository
func NewObservabilityRepository(db *PostgresDB) repository.ObservabilityRepository {
	return &ObservabilityRepository{db: db}
}

// CreateLog creates a new log entry
func (r *ObservabilityRepository) CreateLog(ctx context.Context, log *entity.MessageLog) error {
	metadata, err := json.Marshal(log.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	query := `
		INSERT INTO message_logs (tenant_id, channel_id, level, source, message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		log.TenantID,
		log.ChannelID,
		log.Level,
		log.Source,
		log.Message,
		metadata,
	).Scan(&log.ID, &log.CreatedAt)
}

// GetLogs retrieves logs with filtering and pagination
func (r *ObservabilityRepository) GetLogs(ctx context.Context, filter entity.MessageLogFilter) (*entity.LogsResponse, error) {
	// Build query with optional filters
	baseQuery := `
		FROM message_logs ml
		LEFT JOIN channels ch ON ch.id = ml.channel_id
		WHERE ml.tenant_id = $1
		  AND ml.created_at >= $2
		  AND ml.created_at <= $3
	`
	args := []interface{}{filter.TenantID, filter.StartDate, filter.EndDate}
	argIdx := 4

	if filter.ChannelID != "" {
		baseQuery += ` AND ml.channel_id = $` + string(rune('0'+argIdx))
		args = append(args, filter.ChannelID)
		argIdx++
	}

	if filter.Level != "" {
		baseQuery += ` AND ml.level = $` + string(rune('0'+argIdx))
		args = append(args, filter.Level)
		argIdx++
	}

	if filter.Source != "" {
		baseQuery += ` AND ml.source = $` + string(rune('0'+argIdx))
		args = append(args, filter.Source)
		argIdx++
	}

	// Get total count
	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Get logs with pagination
	selectQuery := `
		SELECT ml.id, ml.tenant_id, ml.channel_id, ch.name as channel_name,
			   ml.level, ml.source, ml.message, ml.metadata, ml.created_at
	` + baseQuery + `
		ORDER BY ml.created_at DESC
		LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []entity.MessageLog
	for rows.Next() {
		var log entity.MessageLog
		var channelName *string
		var metadataJSON []byte

		if err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.ChannelID,
			&channelName,
			&log.Level,
			&log.Source,
			&log.Message,
			&metadataJSON,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}

		if channelName != nil {
			log.ChannelName = *channelName
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &log.Metadata)
		}

		logs = append(logs, log)
	}

	return &entity.LogsResponse{
		Logs:    logs,
		Total:   total,
		HasMore: int64(filter.Offset+filter.Limit) < total,
	}, rows.Err()
}

// GetLogsByChannel retrieves logs for a specific channel
func (r *ObservabilityRepository) GetLogsByChannel(ctx context.Context, tenantID, channelID string, limit, offset int) ([]entity.MessageLog, error) {
	query := `
		SELECT ml.id, ml.tenant_id, ml.channel_id, ch.name as channel_name,
			   ml.level, ml.source, ml.message, ml.metadata, ml.created_at
		FROM message_logs ml
		LEFT JOIN channels ch ON ch.id = ml.channel_id
		WHERE ml.tenant_id = $1 AND ml.channel_id = $2
		ORDER BY ml.created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, channelID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []entity.MessageLog
	for rows.Next() {
		var log entity.MessageLog
		var channelName *string
		var metadataJSON []byte

		if err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.ChannelID,
			&channelName,
			&log.Level,
			&log.Source,
			&log.Message,
			&metadataJSON,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}

		if channelName != nil {
			log.ChannelName = *channelName
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &log.Metadata)
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetSystemStats retrieves system statistics for a tenant
func (r *ObservabilityRepository) GetSystemStats(ctx context.Context, filter entity.StatsFilter) (*entity.SystemStats, error) {
	messageStats, err := r.GetMessageStats(ctx, filter.TenantID, filter.Period)
	if err != nil {
		return nil, err
	}

	errorStats, err := r.GetErrorStats(ctx, filter.TenantID)
	if err != nil {
		return nil, err
	}

	// Get channel stats
	channelQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'connected') as connected,
			COUNT(*) FILTER (WHERE status != 'connected') as disconnected
		FROM channels
		WHERE tenant_id = $1
	`
	var channelStats entity.ChannelStats
	if err := r.db.Pool.QueryRow(ctx, channelQuery, filter.TenantID).Scan(
		&channelStats.Total,
		&channelStats.Connected,
		&channelStats.Disconnected,
	); err != nil {
		return nil, err
	}

	// Get conversation stats
	convQuery := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'open') as active,
			COUNT(*) FILTER (WHERE status = 'resolved' AND resolved_at >= NOW() - INTERVAL '24 hours') as resolved_today,
			COUNT(*) FILTER (WHERE status = 'pending') as pending
		FROM conversations
		WHERE tenant_id = $1
	`
	var convStats entity.ConversationStats
	if err := r.db.Pool.QueryRow(ctx, convQuery, filter.TenantID).Scan(
		&convStats.Active,
		&convStats.ResolvedToday,
		&convStats.Pending,
	); err != nil {
		return nil, err
	}

	// Get response time stats
	rtQuery := `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (first_reply_at - created_at)) * 1000), 0) as avg_ms,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (first_reply_at - created_at)) * 1000), 0) as p95_ms,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (first_reply_at - created_at)) * 1000), 0) as p99_ms
		FROM conversations
		WHERE tenant_id = $1
		  AND first_reply_at IS NOT NULL
		  AND created_at >= NOW() - INTERVAL '24 hours'
	`
	var rtStats entity.ResponseTimeStats
	if err := r.db.Pool.QueryRow(ctx, rtQuery, filter.TenantID).Scan(
		&rtStats.AvgMs,
		&rtStats.P95Ms,
		&rtStats.P99Ms,
	); err != nil {
		// Ignore errors for percentile calculation if no data
		rtStats = entity.ResponseTimeStats{}
	}

	return &entity.SystemStats{
		Messages:      *messageStats,
		ResponseTime:  rtStats,
		Channels:      channelStats,
		Conversations: convStats,
		Errors:        *errorStats,
	}, nil
}

// GetMessageStats retrieves message statistics
func (r *ObservabilityRepository) GetMessageStats(ctx context.Context, tenantID string, period entity.StatsPeriod) (*entity.MessageStats, error) {
	// Determine interval based on period
	var interval string
	switch period {
	case entity.StatsPeriodHour:
		interval = "1 hour"
	case entity.StatsPeriodWeek:
		interval = "7 days"
	default:
		interval = "24 hours"
	}

	// Get total messages
	totalQuery := `
		SELECT COUNT(*)
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE c.tenant_id = $1
		  AND m.created_at >= NOW() - $2::INTERVAL
	`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, totalQuery, tenantID, interval).Scan(&total); err != nil {
		return nil, err
	}

	// Get messages per hour (last 24 hours)
	perHourQuery := `
		SELECT
			DATE_TRUNC('hour', m.created_at) as hour,
			COUNT(*) as count
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE c.tenant_id = $1
		  AND m.created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY DATE_TRUNC('hour', m.created_at)
		ORDER BY hour
	`
	rows, err := r.db.Pool.Query(ctx, perHourQuery, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perHour []entity.HourlyCount
	for rows.Next() {
		var hc entity.HourlyCount
		if err := rows.Scan(&hc.Hour, &hc.Count); err != nil {
			return nil, err
		}
		perHour = append(perHour, hc)
	}

	// Get messages by channel
	byChannelQuery := `
		SELECT
			ch.id as channel_id,
			ch.name as channel_name,
			ch.type as channel_type,
			COUNT(m.id) as count
		FROM channels ch
		LEFT JOIN conversations c ON c.channel_id = ch.id
		LEFT JOIN messages m ON m.conversation_id = c.id
			AND m.created_at >= NOW() - $2::INTERVAL
		WHERE ch.tenant_id = $1
		GROUP BY ch.id, ch.name, ch.type
		ORDER BY count DESC
	`
	channelRows, err := r.db.Pool.Query(ctx, byChannelQuery, tenantID, interval)
	if err != nil {
		return nil, err
	}
	defer channelRows.Close()

	var byChannel []entity.ChannelCount
	for channelRows.Next() {
		var cc entity.ChannelCount
		if err := channelRows.Scan(&cc.ChannelID, &cc.ChannelName, &cc.ChannelType, &cc.Count); err != nil {
			return nil, err
		}
		byChannel = append(byChannel, cc)
	}

	return &entity.MessageStats{
		Total:     total,
		PerHour:   perHour,
		ByChannel: byChannel,
	}, nil
}

// GetErrorStats retrieves error statistics
func (r *ObservabilityRepository) GetErrorStats(ctx context.Context, tenantID string) (*entity.ErrorStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 hour') as last_hour,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') as last_24h
		FROM message_logs
		WHERE tenant_id = $1 AND level = 'error'
	`
	var stats entity.ErrorStats
	if err := r.db.Pool.QueryRow(ctx, query, tenantID).Scan(&stats.LastHour, &stats.Last24h); err != nil {
		return nil, err
	}

	// Get errors by source
	bySourceQuery := `
		SELECT source, COUNT(*) as count
		FROM message_logs
		WHERE tenant_id = $1
		  AND level = 'error'
		  AND created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY source
		ORDER BY count DESC
	`
	rows, err := r.db.Pool.Query(ctx, bySourceQuery, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sec entity.SourceErrorCount
		if err := rows.Scan(&sec.Source, &sec.Count); err != nil {
			return nil, err
		}
		stats.BySource = append(stats.BySource, sec)
	}

	return &stats, rows.Err()
}

// CleanupOldLogs removes logs older than the retention period
func (r *ObservabilityRepository) CleanupOldLogs(ctx context.Context, tenantID string, retentionDays int) (int64, error) {
	query := `
		DELETE FROM message_logs
		WHERE tenant_id = $1
		  AND created_at < NOW() - ($2 || ' days')::INTERVAL
	`
	result, err := r.db.Pool.Exec(ctx, query, tenantID, retentionDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
