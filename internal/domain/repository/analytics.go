package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// AnalyticsRepository defines the interface for analytics data access
type AnalyticsRepository interface {
	// GetOverview returns high-level analytics metrics
	GetOverview(ctx context.Context, filter entity.AnalyticsFilter) (*entity.OverviewAnalytics, error)

	// GetConversationsByDay returns conversation metrics grouped by day
	GetConversationsByDay(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.ConversationAnalytics, error)

	// GetFlowAnalytics returns metrics for all flows
	GetFlowAnalytics(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.FlowAnalytics, error)

	// GetEscalationsByReason returns escalation metrics grouped by reason
	GetEscalationsByReason(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.EscalationAnalytics, error)

	// GetChannelAnalytics returns metrics grouped by channel
	GetChannelAnalytics(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.ChannelAnalytics, error)
}
