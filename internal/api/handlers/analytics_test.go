package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// mockAnalyticsRepository
// ============================================================================

type mockAnalyticsRepository struct {
	ReturnError error
}

func (m *mockAnalyticsRepository) GetOverview(ctx context.Context, filter entity.AnalyticsFilter) (*entity.OverviewAnalytics, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return &entity.OverviewAnalytics{}, nil
}

func (m *mockAnalyticsRepository) GetConversationsByDay(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.ConversationAnalytics, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return []entity.ConversationAnalytics{}, nil
}

func (m *mockAnalyticsRepository) GetFlowAnalytics(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.FlowAnalytics, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return []entity.FlowAnalytics{}, nil
}

func (m *mockAnalyticsRepository) GetEscalationsByReason(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.EscalationAnalytics, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return []entity.EscalationAnalytics{}, nil
}

func (m *mockAnalyticsRepository) GetChannelAnalytics(ctx context.Context, filter entity.AnalyticsFilter) ([]entity.ChannelAnalytics, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return []entity.ChannelAnalytics{}, nil
}

// ============================================================================
// Setup helper
// ============================================================================

func setupAnalyticsTest(t *testing.T) *AnalyticsHandler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &mockAnalyticsRepository{}
	svc := service.NewAnalyticsService(repo, nil)
	return NewAnalyticsHandler(svc)
}

// ============================================================================
// Tests
// ============================================================================

func TestAnalyticsHandler_parseAnalyticsParams(t *testing.T) {
	handler := setupAnalyticsTest(t)

	t.Run("default period", func(t *testing.T) {
		w, c := newTestContext(http.MethodGet, "/analytics/overview", nil)
		_ = w

		period, startDate, endDate := handler.parseAnalyticsParams(c)

		assert.Equal(t, "weekly", string(period))
		assert.True(t, endDate.After(startDate))
	})

	t.Run("daily period", func(t *testing.T) {
		w, c := newTestContext(http.MethodGet, "/analytics/overview?period=daily", nil)
		_ = w

		period, startDate, endDate := handler.parseAnalyticsParams(c)

		assert.Equal(t, "daily", string(period))
		assert.True(t, endDate.After(startDate))
	})

	t.Run("custom dates", func(t *testing.T) {
		w, c := newTestContext(http.MethodGet, "/analytics/overview?start_date=2025-01-01&end_date=2025-01-31", nil)
		_ = w

		_, startDate, endDate := handler.parseAnalyticsParams(c)

		assert.Equal(t, 2025, startDate.Year())
		assert.Equal(t, 1, int(startDate.Month()))
		assert.Equal(t, 1, startDate.Day())
		assert.Equal(t, 2025, endDate.Year())
		assert.Equal(t, 1, int(endDate.Month()))
		assert.Equal(t, 31, endDate.Day())
	})
}

func TestAnalyticsHandler_GetOverview_NoTenantID(t *testing.T) {
	handler := setupAnalyticsTest(t)

	w, c := newTestContext(http.MethodGet, "/analytics/overview", nil)
	// Do not set tenant_id - GetTenantID returns "" but doesn't abort

	handler.GetOverview(c)

	// With mock repo, service returns empty overview with tenantID=""
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAnalyticsHandler_GetConversations(t *testing.T) {
	handler := setupAnalyticsTest(t)

	w, c := newTestContext(http.MethodGet, "/analytics/conversations", nil)
	c.Set("tenant_id", "tenant-1")

	handler.GetConversations(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["data"])
}

func TestAnalyticsHandler_GetFlows(t *testing.T) {
	handler := setupAnalyticsTest(t)

	w, c := newTestContext(http.MethodGet, "/analytics/flows", nil)
	c.Set("tenant_id", "tenant-1")

	handler.GetFlows(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["data"])
}

func TestAnalyticsHandler_GetEscalations(t *testing.T) {
	handler := setupAnalyticsTest(t)

	w, c := newTestContext(http.MethodGet, "/analytics/escalations", nil)
	c.Set("tenant_id", "tenant-1")

	handler.GetEscalations(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["data"])
}

func TestAnalyticsHandler_GetChannels(t *testing.T) {
	handler := setupAnalyticsTest(t)

	w, c := newTestContext(http.MethodGet, "/analytics/channels", nil)
	c.Set("tenant_id", "tenant-1")

	handler.GetChannels(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["data"])
}
