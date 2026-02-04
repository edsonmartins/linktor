package service

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// AnalyticsService provides analytics operations
type AnalyticsService struct {
	repo repository.AnalyticsRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// GetOverview returns high-level analytics metrics
func (s *AnalyticsService) GetOverview(ctx context.Context, tenantID string, period entity.AnalyticsPeriod, startDate, endDate time.Time) (*entity.OverviewAnalytics, error) {
	filter := entity.AnalyticsFilter{
		TenantID:  tenantID,
		Period:    period,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.repo.GetOverview(ctx, filter)
}

// GetConversationsByDay returns conversation metrics grouped by day
func (s *AnalyticsService) GetConversationsByDay(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]entity.ConversationAnalytics, error) {
	filter := entity.AnalyticsFilter{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.repo.GetConversationsByDay(ctx, filter)
}

// GetFlowAnalytics returns metrics for all flows
func (s *AnalyticsService) GetFlowAnalytics(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]entity.FlowAnalytics, error) {
	filter := entity.AnalyticsFilter{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.repo.GetFlowAnalytics(ctx, filter)
}

// GetEscalationsByReason returns escalation metrics grouped by reason
func (s *AnalyticsService) GetEscalationsByReason(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]entity.EscalationAnalytics, error) {
	filter := entity.AnalyticsFilter{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.repo.GetEscalationsByReason(ctx, filter)
}

// GetChannelAnalytics returns metrics grouped by channel
func (s *AnalyticsService) GetChannelAnalytics(ctx context.Context, tenantID string, startDate, endDate time.Time) ([]entity.ChannelAnalytics, error) {
	filter := entity.AnalyticsFilter{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.repo.GetChannelAnalytics(ctx, filter)
}

// GetDateRange returns the start and end dates based on the period
func (s *AnalyticsService) GetDateRange(period entity.AnalyticsPeriod, customStart, customEnd *time.Time) (time.Time, time.Time) {
	now := time.Now().UTC()
	var startDate, endDate time.Time

	if customStart != nil && customEnd != nil {
		return *customStart, *customEnd
	}

	switch period {
	case entity.PeriodDaily:
		// Last 24 hours
		startDate = now.Add(-24 * time.Hour)
		endDate = now
	case entity.PeriodWeekly:
		// Last 7 days
		startDate = now.Add(-7 * 24 * time.Hour)
		endDate = now
	case entity.PeriodMonthly:
		// Last 30 days
		startDate = now.Add(-30 * 24 * time.Hour)
		endDate = now
	default:
		// Default to last 7 days
		startDate = now.Add(-7 * 24 * time.Hour)
		endDate = now
	}

	return startDate, endDate
}
