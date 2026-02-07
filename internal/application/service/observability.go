package service

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// ObservabilityService provides observability operations
type ObservabilityService struct {
	repo        repository.ObservabilityRepository
	natsMonitor *nats.Monitor
}

// NewObservabilityService creates a new observability service
func NewObservabilityService(repo repository.ObservabilityRepository, natsMonitor *nats.Monitor) *ObservabilityService {
	return &ObservabilityService{
		repo:        repo,
		natsMonitor: natsMonitor,
	}
}

// CreateLog creates a new log entry
func (s *ObservabilityService) CreateLog(ctx context.Context, log *entity.MessageLog) error {
	return s.repo.CreateLog(ctx, log)
}

// LogInfo creates an info-level log entry
func (s *ObservabilityService) LogInfo(ctx context.Context, tenantID string, channelID *string, source entity.LogSource, message string, metadata map[string]interface{}) error {
	log := &entity.MessageLog{
		TenantID:  tenantID,
		ChannelID: channelID,
		Level:     entity.LogLevelInfo,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	}
	return s.repo.CreateLog(ctx, log)
}

// LogWarn creates a warning-level log entry
func (s *ObservabilityService) LogWarn(ctx context.Context, tenantID string, channelID *string, source entity.LogSource, message string, metadata map[string]interface{}) error {
	log := &entity.MessageLog{
		TenantID:  tenantID,
		ChannelID: channelID,
		Level:     entity.LogLevelWarn,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	}
	return s.repo.CreateLog(ctx, log)
}

// LogError creates an error-level log entry
func (s *ObservabilityService) LogError(ctx context.Context, tenantID string, channelID *string, source entity.LogSource, message string, metadata map[string]interface{}) error {
	log := &entity.MessageLog{
		TenantID:  tenantID,
		ChannelID: channelID,
		Level:     entity.LogLevelError,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	}
	return s.repo.CreateLog(ctx, log)
}

// GetLogs retrieves logs with filtering and pagination
func (s *ObservabilityService) GetLogs(ctx context.Context, tenantID string, channelID string, level entity.LogLevel, source entity.LogSource, startDate, endDate time.Time, limit, offset int) (*entity.LogsResponse, error) {
	// Apply defaults
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	filter := entity.MessageLogFilter{
		TenantID:  tenantID,
		ChannelID: channelID,
		Level:     level,
		Source:    source,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Offset:    offset,
	}

	return s.repo.GetLogs(ctx, filter)
}

// GetLogsByChannel retrieves logs for a specific channel
func (s *ObservabilityService) GetLogsByChannel(ctx context.Context, tenantID, channelID string, limit, offset int) ([]entity.MessageLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetLogsByChannel(ctx, tenantID, channelID, limit, offset)
}

// GetQueueStats retrieves NATS queue statistics
func (s *ObservabilityService) GetQueueStats(ctx context.Context) (*entity.QueueStats, error) {
	return s.natsMonitor.GetQueueStats(ctx)
}

// GetStreamInfo retrieves information about a specific stream
func (s *ObservabilityService) GetStreamInfo(ctx context.Context, streamName string) (*entity.StreamInfo, error) {
	return s.natsMonitor.GetStreamInfo(ctx, streamName)
}

// ResetConsumer resets a consumer to reprocess messages
func (s *ObservabilityService) ResetConsumer(ctx context.Context, streamName, consumerName string) error {
	return s.natsMonitor.ResetConsumer(ctx, streamName, consumerName)
}

// GetSystemStats retrieves system statistics
func (s *ObservabilityService) GetSystemStats(ctx context.Context, tenantID string, period entity.StatsPeriod) (*entity.SystemStats, error) {
	filter := entity.StatsFilter{
		TenantID: tenantID,
		Period:   period,
	}
	return s.repo.GetSystemStats(ctx, filter)
}

// GetErrorStats retrieves error statistics
func (s *ObservabilityService) GetErrorStats(ctx context.Context, tenantID string) (*entity.ErrorStats, error) {
	return s.repo.GetErrorStats(ctx, tenantID)
}

// CleanupOldLogs removes logs older than the retention period
func (s *ObservabilityService) CleanupOldLogs(ctx context.Context, tenantID string, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 30 // Default retention
	}
	return s.repo.CleanupOldLogs(ctx, tenantID, retentionDays)
}

// GetDateRange returns the start and end dates based on the period
func (s *ObservabilityService) GetDateRange(period entity.StatsPeriod) (time.Time, time.Time) {
	now := time.Now().UTC()
	var startDate time.Time

	switch period {
	case entity.StatsPeriodHour:
		startDate = now.Add(-1 * time.Hour)
	case entity.StatsPeriodWeek:
		startDate = now.Add(-7 * 24 * time.Hour)
	default: // day
		startDate = now.Add(-24 * time.Hour)
	}

	return startDate, now
}
