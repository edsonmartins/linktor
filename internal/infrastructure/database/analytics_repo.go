package database

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// AnalyticsRepository implements repository.AnalyticsRepository with PostgreSQL
type AnalyticsRepository struct {
	db *PostgresDB
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *PostgresDB) repository.AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// GetOverview returns high-level analytics metrics
func (r *AnalyticsRepository) GetOverview(ctx context.Context, filter entity.AnalyticsFilter) (*entity.OverviewAnalytics, error) {
	// Current period stats
	query := `
		WITH conv_stats AS (
			SELECT
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'resolved' AND escalated_at IS NULL) as resolved_by_bot,
				COUNT(*) FILTER (WHERE escalated_at IS NOT NULL) as escalated,
				AVG(EXTRACT(EPOCH FROM (first_reply_at - created_at)) * 1000)
					FILTER (WHERE first_reply_at IS NOT NULL) as avg_first_response,
				AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) * 1000)
					FILTER (WHERE resolved_at IS NOT NULL) as avg_resolution_time
			FROM conversations
			WHERE tenant_id = $1
			  AND created_at >= $2
			  AND created_at < $3
		),
		bot_stats AS (
			SELECT
				COUNT(*) as total_messages,
				AVG((metadata->>'confidence')::float) as avg_confidence
			FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			WHERE c.tenant_id = $1
			  AND m.sender_type = 'bot'
			  AND m.created_at >= $2
			  AND m.created_at < $3
		),
		prev_conv_stats AS (
			SELECT
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'resolved' AND escalated_at IS NULL) as resolved_by_bot
			FROM conversations
			WHERE tenant_id = $1
			  AND created_at >= $2 - ($3 - $2)
			  AND created_at < $2
		)
		SELECT
			COALESCE(cs.total, 0),
			COALESCE(cs.resolved_by_bot, 0),
			COALESCE(cs.escalated, 0),
			COALESCE(cs.avg_first_response, 0),
			COALESCE(cs.avg_resolution_time, 0),
			COALESCE(bs.total_messages, 0),
			COALESCE(bs.avg_confidence, 0),
			COALESCE(pcs.total, 0),
			COALESCE(pcs.resolved_by_bot, 0)
		FROM conv_stats cs, bot_stats bs, prev_conv_stats pcs
	`

	var total, resolvedByBot, escalated, prevTotal, prevResolved int64
	var avgFirstResponse, avgResolutionTime float64
	var totalBotMessages int64
	var avgConfidence float64

	err := r.db.Pool.QueryRow(ctx, query, filter.TenantID, filter.StartDate, filter.EndDate).Scan(
		&total,
		&resolvedByBot,
		&escalated,
		&avgFirstResponse,
		&avgResolutionTime,
		&totalBotMessages,
		&avgConfidence,
		&prevTotal,
		&prevResolved,
	)
	if err != nil {
		return nil, err
	}

	// Calculate resolution rate
	var resolutionRate float64
	if total > 0 {
		resolutionRate = float64(resolvedByBot) / float64(total) * 100
	}

	// Calculate trends
	var conversationsTrend, resolutionTrend float64
	if prevTotal > 0 {
		conversationsTrend = float64(total-prevTotal) / float64(prevTotal) * 100
	}
	if prevResolved > 0 && prevTotal > 0 {
		prevRate := float64(prevResolved) / float64(prevTotal) * 100
		if prevRate > 0 {
			resolutionTrend = (resolutionRate - prevRate) / prevRate * 100
		}
	}

	return &entity.OverviewAnalytics{
		Period:              filter.Period,
		StartDate:           filter.StartDate,
		EndDate:             filter.EndDate,
		TotalConversations:  total,
		ResolvedByBot:       resolvedByBot,
		EscalatedToHuman:    escalated,
		ResolutionRate:      resolutionRate,
		AvgFirstResponseMs:  int64(avgFirstResponse),
		AvgResolutionTimeMs: int64(avgResolutionTime),
		TotalBotMessages:    totalBotMessages,
		AvgConfidence:       avgConfidence,
		ConversationsTrend:  conversationsTrend,
		ResolutionTrend:     resolutionTrend,
	}, nil
}

// GetConversationsByDay returns conversation metrics grouped by day
func (r *AnalyticsRepository) GetConversationsByDay(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.ConversationAnalytics, error) {
	query := `
		SELECT
			TO_CHAR(created_at, 'YYYY-MM-DD') as date,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'resolved' AND escalated_at IS NULL) as resolved_by_bot,
			COUNT(*) FILTER (WHERE escalated_at IS NOT NULL) as escalated,
			COUNT(*) FILTER (WHERE status = 'pending' OR status = 'open') as pending
		FROM conversations
		WHERE tenant_id = $1
		  AND created_at >= $2
		  AND created_at < $3
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD')
		ORDER BY date
	`

	rows, err := r.db.Pool.Query(ctx, query, filter.TenantID, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entity.ConversationAnalytics
	for rows.Next() {
		var ca entity.ConversationAnalytics
		if err := rows.Scan(&ca.Date, &ca.TotalConversations, &ca.ResolvedByBot, &ca.Escalated, &ca.Pending); err != nil {
			return nil, err
		}
		result = append(result, ca)
	}

	return result, rows.Err()
}

// GetFlowAnalytics returns metrics for all flows
func (r *AnalyticsRepository) GetFlowAnalytics(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.FlowAnalytics, error) {
	query := `
		SELECT
			f.id,
			f.name,
			COUNT(DISTINCT cc.id) FILTER (WHERE cc.current_flow_id = f.id OR cc.metadata->>'triggered_flow_id' = f.id::text) as times_triggered,
			COUNT(DISTINCT cc.id) FILTER (WHERE cc.metadata->>'flow_completed' = 'true' AND (cc.current_flow_id = f.id OR cc.metadata->>'triggered_flow_id' = f.id::text)) as times_completed,
			COALESCE(AVG((cc.metadata->>'flow_steps')::int) FILTER (WHERE cc.current_flow_id = f.id), 0) as avg_steps
		FROM flows f
		LEFT JOIN conversation_contexts cc ON cc.tenant_id = f.tenant_id
		LEFT JOIN conversations c ON c.id = cc.conversation_id
		WHERE f.tenant_id = $1
		  AND (c.created_at IS NULL OR (c.created_at >= $2 AND c.created_at < $3))
		GROUP BY f.id, f.name
		ORDER BY times_triggered DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, filter.TenantID, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entity.FlowAnalytics
	for rows.Next() {
		var fa entity.FlowAnalytics
		if err := rows.Scan(&fa.FlowID, &fa.FlowName, &fa.TimesTriggered, &fa.TimesCompleted, &fa.AvgStepsToEnd); err != nil {
			return nil, err
		}
		if fa.TimesTriggered > 0 {
			fa.CompletionRate = float64(fa.TimesCompleted) / float64(fa.TimesTriggered) * 100
		}
		result = append(result, fa)
	}

	return result, rows.Err()
}

// GetEscalationsByReason returns escalation metrics grouped by reason
func (r *AnalyticsRepository) GetEscalationsByReason(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.EscalationAnalytics, error) {
	query := `
		WITH escalation_data AS (
			SELECT
				COALESCE(metadata->>'escalation_reason', 'unknown') as reason,
				COUNT(*) as count,
				AVG(EXTRACT(EPOCH FROM (escalated_at - created_at)) * 1000) as avg_time
			FROM conversations
			WHERE tenant_id = $1
			  AND escalated_at IS NOT NULL
			  AND created_at >= $2
			  AND created_at < $3
			GROUP BY reason
		),
		total AS (
			SELECT SUM(count) as total_count FROM escalation_data
		)
		SELECT
			ed.reason,
			ed.count,
			CASE WHEN t.total_count > 0 THEN ed.count::float / t.total_count * 100 ELSE 0 END as percentage,
			COALESCE(ed.avg_time, 0) as avg_time
		FROM escalation_data ed, total t
		ORDER BY ed.count DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, filter.TenantID, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entity.EscalationAnalytics
	for rows.Next() {
		var ea entity.EscalationAnalytics
		var avgTime float64
		if err := rows.Scan(&ea.Reason, &ea.Count, &ea.Percentage, &avgTime); err != nil {
			return nil, err
		}
		ea.AvgTimeToEscMs = int64(avgTime)
		result = append(result, ea)
	}

	return result, rows.Err()
}

// GetChannelAnalytics returns metrics grouped by channel
func (r *AnalyticsRepository) GetChannelAnalytics(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.ChannelAnalytics, error) {
	query := `
		SELECT
			ch.id,
			ch.name,
			ch.type,
			COUNT(c.id) as total,
			COUNT(c.id) FILTER (WHERE c.status = 'resolved' AND c.escalated_at IS NULL) as resolved_by_bot
		FROM channels ch
		LEFT JOIN conversations c ON c.channel_id = ch.id
			AND c.created_at >= $2
			AND c.created_at < $3
		WHERE ch.tenant_id = $1
		GROUP BY ch.id, ch.name, ch.type
		ORDER BY total DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, filter.TenantID, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entity.ChannelAnalytics
	for rows.Next() {
		var ca entity.ChannelAnalytics
		if err := rows.Scan(&ca.ChannelID, &ca.ChannelName, &ca.ChannelType, &ca.TotalConversations, &ca.ResolvedByBot); err != nil {
			return nil, err
		}
		if ca.TotalConversations > 0 {
			ca.ResolutionRate = float64(ca.ResolvedByBot) / float64(ca.TotalConversations) * 100
		}
		result = append(result, ca)
	}

	return result, rows.Err()
}
