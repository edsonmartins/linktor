package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// ObservabilityRepository defines the interface for observability data access
type ObservabilityRepository interface {
	// CreateLog creates a new log entry
	CreateLog(ctx context.Context, log *entity.MessageLog) error

	// GetLogs retrieves logs with filtering and pagination
	GetLogs(ctx context.Context, filter entity.MessageLogFilter) (*entity.LogsResponse, error)

	// GetLogsByChannel retrieves logs for a specific channel
	GetLogsByChannel(ctx context.Context, tenantID, channelID string, limit, offset int) ([]entity.MessageLog, error)

	// GetSystemStats retrieves system statistics for a tenant
	GetSystemStats(ctx context.Context, filter entity.StatsFilter) (*entity.SystemStats, error)

	// GetMessageStats retrieves message statistics
	GetMessageStats(ctx context.Context, tenantID string, period entity.StatsPeriod) (*entity.MessageStats, error)

	// GetErrorStats retrieves error statistics
	GetErrorStats(ctx context.Context, tenantID string) (*entity.ErrorStats, error)

	// CleanupOldLogs removes logs older than the retention period
	CleanupOldLogs(ctx context.Context, tenantID string, retentionDays int) (int64, error)
}
