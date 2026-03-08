package service

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock ObservabilityRepository
// ============================================================================

type mockObservabilityRepository struct {
	logs        []entity.MessageLog
	cleanupDays int
	ReturnError error
}

func newMockObservabilityRepository() *mockObservabilityRepository {
	return &mockObservabilityRepository{
		logs: make([]entity.MessageLog, 0),
	}
}

func (m *mockObservabilityRepository) CreateLog(ctx context.Context, log *entity.MessageLog) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockObservabilityRepository) GetLogs(ctx context.Context, filter entity.MessageLogFilter) (*entity.LogsResponse, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return &entity.LogsResponse{
		Logs:    m.logs,
		Total:   int64(len(m.logs)),
		HasMore: false,
	}, nil
}

func (m *mockObservabilityRepository) GetLogsByChannel(ctx context.Context, tenantID, channelID string, limit, offset int) ([]entity.MessageLog, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return m.logs, nil
}

func (m *mockObservabilityRepository) GetSystemStats(ctx context.Context, filter entity.StatsFilter) (*entity.SystemStats, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return &entity.SystemStats{}, nil
}

func (m *mockObservabilityRepository) GetMessageStats(ctx context.Context, tenantID string, period entity.StatsPeriod) (*entity.MessageStats, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return &entity.MessageStats{}, nil
}

func (m *mockObservabilityRepository) GetErrorStats(ctx context.Context, tenantID string) (*entity.ErrorStats, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return &entity.ErrorStats{}, nil
}

func (m *mockObservabilityRepository) CleanupOldLogs(ctx context.Context, tenantID string, retentionDays int) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	m.cleanupDays = retentionDays
	return int64(len(m.logs)), nil
}

// ============================================================================
// Tests
// ============================================================================

func TestNewObservabilityService(t *testing.T) {
	repo := newMockObservabilityRepository()

	svc := NewObservabilityService(repo, nil)
	require.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Nil(t, svc.natsMonitor)
}

func TestObservabilityService_GetDateRange(t *testing.T) {
	repo := newMockObservabilityRepository()
	svc := NewObservabilityService(repo, nil)

	tests := []struct {
		name           string
		period         entity.StatsPeriod
		expectedMinAge time.Duration
		expectedMaxAge time.Duration
	}{
		{
			name:           "hour period",
			period:         entity.StatsPeriodHour,
			expectedMinAge: 59 * time.Minute,
			expectedMaxAge: 61 * time.Minute,
		},
		{
			name:           "day period",
			period:         entity.StatsPeriodDay,
			expectedMinAge: 23 * time.Hour,
			expectedMaxAge: 25 * time.Hour,
		},
		{
			name:           "week period",
			period:         entity.StatsPeriodWeek,
			expectedMinAge: 6*24*time.Hour + 23*time.Hour,
			expectedMaxAge: 7*24*time.Hour + time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDate, endDate := svc.GetDateRange(tt.period)

			// endDate should be approximately now
			assert.WithinDuration(t, time.Now().UTC(), endDate, 2*time.Second)

			// startDate should be the expected duration before endDate
			diff := endDate.Sub(startDate)
			assert.GreaterOrEqual(t, diff, tt.expectedMinAge)
			assert.LessOrEqual(t, diff, tt.expectedMaxAge)
		})
	}

	t.Run("default is day", func(t *testing.T) {
		startDate, endDate := svc.GetDateRange(entity.StatsPeriod("invalid"))
		diff := endDate.Sub(startDate)
		// Should default to 24 hours
		assert.InDelta(t, 24*time.Hour, diff, float64(2*time.Second))
	})
}

func TestObservabilityService_CleanupOldLogs_DefaultRetention(t *testing.T) {
	repo := newMockObservabilityRepository()
	svc := NewObservabilityService(repo, nil)
	ctx := context.Background()

	t.Run("default retention is 30 days", func(t *testing.T) {
		_, err := svc.CleanupOldLogs(ctx, "tenant-1", 0)
		require.NoError(t, err)
		assert.Equal(t, 30, repo.cleanupDays)
	})

	t.Run("negative retention uses default", func(t *testing.T) {
		_, err := svc.CleanupOldLogs(ctx, "tenant-1", -5)
		require.NoError(t, err)
		assert.Equal(t, 30, repo.cleanupDays)
	})

	t.Run("custom retention is respected", func(t *testing.T) {
		_, err := svc.CleanupOldLogs(ctx, "tenant-1", 7)
		require.NoError(t, err)
		assert.Equal(t, 7, repo.cleanupDays)
	})
}
