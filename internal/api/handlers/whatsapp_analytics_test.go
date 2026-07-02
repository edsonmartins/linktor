package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/whatsapp/analytics"
	"github.com/msgfy/linktor/pkg/testutil"
)

// TestWhatsAppAnalyticsHandler_CrossTenantRejected proves the tenant-isolation
// guard: tenant-a cannot read analytics for a channel owned by tenant-b. A client
// is registered for the channel, so if the guard were missing the handler would
// dereference it and panic — a clean 404 proves the guard rejects first.
func TestWhatsAppAnalyticsHandler_CrossTenantRejected(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	repo.Channels["channel-b"] = &entity.Channel{ID: "channel-b", TenantID: "tenant-b"}

	h := NewWhatsAppAnalyticsHandler(repo)
	h.RegisterClient("channel-b", (*analytics.Client)(nil))

	w, c := newTestContext(http.MethodGet, "/channels/channel-b/analytics/conversations", nil)
	c.Set(middleware.TenantIDKey, "tenant-a")
	c.Params = gin.Params{{Key: "id", Value: "channel-b"}}

	h.GetConversationAnalytics(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotContains(t, resp["error"], "not configured")
}

func TestNewWhatsAppAnalyticsHandler(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	require.NotNil(t, h)
	assert.NotNil(t, h.clients)
}

func TestWhatsAppAnalyticsHandler_GetConversationAnalytics_NoClient(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/analytics/conversations", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetConversationAnalytics(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not found")
}

func TestWhatsAppAnalyticsHandler_GetPhoneNumberAnalytics_NoClient(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/analytics/phone", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetPhoneNumberAnalytics(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWhatsAppAnalyticsHandler_GetTemplateAnalytics_NoClient(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/analytics/templates/tmpl-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "templateId", Value: "tmpl-1"}}

	h.GetTemplateAnalytics(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWhatsAppAnalyticsHandler_GetAggregatedStats_NoClient(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/analytics/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetAggregatedStats(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWhatsAppAnalyticsHandler_ExportAnalytics_NoClient(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/analytics/export", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.ExportAnalytics(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWhatsAppAnalyticsHandler_GetDashboardData_NoClient(t *testing.T) {
	h := NewWhatsAppAnalyticsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/analytics/dashboard", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetDashboardData(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestParseDate_Valid(t *testing.T) {
	result, err := parseDate("2024-01-15")
	require.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, 1, int(result.Month()))
	assert.Equal(t, 15, result.Day())
}

func TestParseDate_Invalid(t *testing.T) {
	_, err := parseDate("invalid-date")
	assert.Error(t, err)
}

func TestParseDate_Empty(t *testing.T) {
	_, err := parseDate("")
	assert.Error(t, err)
}
