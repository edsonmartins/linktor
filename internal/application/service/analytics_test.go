package service

import (
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestAnalyticsService_GetDateRange(t *testing.T) {
	svc := NewAnalyticsService(nil, nil)

	t.Run("daily period", func(t *testing.T) {
		start, end := svc.GetDateRange(entity.PeriodDaily, nil, nil)
		assert.WithinDuration(t, time.Now().UTC().Add(-24*time.Hour), start, 2*time.Second)
		assert.WithinDuration(t, time.Now().UTC(), end, 2*time.Second)
	})

	t.Run("weekly period", func(t *testing.T) {
		start, end := svc.GetDateRange(entity.PeriodWeekly, nil, nil)
		assert.WithinDuration(t, time.Now().UTC().Add(-7*24*time.Hour), start, 2*time.Second)
		assert.WithinDuration(t, time.Now().UTC(), end, 2*time.Second)
	})

	t.Run("monthly period", func(t *testing.T) {
		start, end := svc.GetDateRange(entity.PeriodMonthly, nil, nil)
		assert.WithinDuration(t, time.Now().UTC().Add(-30*24*time.Hour), start, 2*time.Second)
		assert.WithinDuration(t, time.Now().UTC(), end, 2*time.Second)
	})

	t.Run("custom date range", func(t *testing.T) {
		customStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		customEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
		start, end := svc.GetDateRange(entity.PeriodDaily, &customStart, &customEnd)
		assert.Equal(t, customStart, start)
		assert.Equal(t, customEnd, end)
	})

	t.Run("default period", func(t *testing.T) {
		start, end := svc.GetDateRange("unknown", nil, nil)
		assert.WithinDuration(t, time.Now().UTC().Add(-7*24*time.Hour), start, 2*time.Second)
		assert.WithinDuration(t, time.Now().UTC(), end, 2*time.Second)
	})
}

func TestAnalyticsService_GetWhatsAppConversationAnalytics_NotConfigured(t *testing.T) {
	svc := NewAnalyticsService(nil, nil)

	_, err := svc.GetWhatsAppConversationAnalytics(nil, "phone-1", time.Now(), time.Now(), "DAILY")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}
