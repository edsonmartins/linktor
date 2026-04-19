package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rewriteTransport struct {
	baseURL string
	rt      http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.TrimPrefix(t.baseURL, "http://")
	req.URL.Scheme = "http"
	req.URL.Host = host
	return t.rt.RoundTrip(req)
}

func newHTTPTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	c := NewClient(&ClientConfig{
		AccessToken:   "test-token",
		BusinessID:    "biz-77",
		PhoneNumberID: "phone-55",
		APIVersion:    "v23.0",
	})
	c.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return c, server
}

// -----------------------------------------------------------------------------
// GetConversationAnalytics
// -----------------------------------------------------------------------------

func TestClient_GetConversationAnalytics_Success(t *testing.T) {
	var capturedPath, capturedQuery, capturedAuth string

	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		capturedAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"conversation_analytics":{"data":[{"start":1700000000,"end":1700086400,"conversation":10,"cost":0.05}]}}`))
	})
	defer server.Close()

	req := &AnalyticsRequest{
		PhoneNumberID: "phone-55",
		StartDate:     time.Unix(1700000000, 0),
		EndDate:       time.Unix(1700086400, 0),
		Granularity:   "DAILY",
	}

	analytics, err := client.GetConversationAnalytics(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, analytics)

	assert.Equal(t, "/v23.0/phone-55", capturedPath)
	assert.Contains(t, capturedQuery, "conversation_analytics.start=1700000000")
	assert.Contains(t, capturedQuery, "conversation_analytics.end=1700086400")
	assert.Contains(t, capturedQuery, "conversation_analytics.granularity=DAILY")
	assert.Contains(t, capturedQuery, "conversation_analytics.dimensions=")
	assert.Equal(t, "Bearer test-token", capturedAuth)
}

func TestClient_GetConversationAnalytics_APIError(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid token","code":190}}`))
	})
	defer server.Close()

	_, err := client.GetConversationAnalytics(context.Background(), &AnalyticsRequest{
		PhoneNumberID: "phone-55",
		StartDate:     time.Now().Add(-24 * time.Hour),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestClient_GetConversationAnalytics_ServesFromCache(t *testing.T) {
	var callCount int
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_, _ = w.Write([]byte(`{"conversation_analytics":{"data":[]}}`))
	})
	defer server.Close()

	req := &AnalyticsRequest{
		PhoneNumberID: "phone-55",
		StartDate:     time.Unix(1700000000, 0),
		EndDate:       time.Unix(1700086400, 0),
		Granularity:   "DAILY",
	}

	_, err := client.GetConversationAnalytics(context.Background(), req)
	require.NoError(t, err)
	_, err = client.GetConversationAnalytics(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, 1, callCount, "second call should hit cache, not the network")
}

// -----------------------------------------------------------------------------
// GetPhoneNumberAnalytics
// -----------------------------------------------------------------------------

func TestClient_GetPhoneNumberAnalytics_Success(t *testing.T) {
	var capturedPath, capturedQuery string
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
			"id":"phone-55",
			"display_phone_number":"+1 555-123-4567",
			"quality_rating":"GREEN",
			"messaging_limit_tier":"TIER_1K",
			"throughput":{"level":80},
			"status":"CONNECTED",
			"name_status":"APPROVED",
			"new_name_status":"NONE"
		}`))
	})
	defer server.Close()

	info, err := client.GetPhoneNumberAnalytics(context.Background(), "phone-55")
	require.NoError(t, err)

	assert.Equal(t, "/v23.0/phone-55", capturedPath)
	assert.Contains(t, capturedQuery, "fields=")
	assert.Contains(t, capturedQuery, "quality_rating")
	assert.Equal(t, "GREEN", info.QualityRating)
	assert.Equal(t, "TIER_1K", info.MessagingLimit)
	assert.Equal(t, 80, info.CurrentThroughput)
	assert.Equal(t, "CONNECTED", info.Status)
}

func TestClient_GetPhoneNumberAnalytics_APIError(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"phone number not found","code":100}}`))
	})
	defer server.Close()

	_, err := client.GetPhoneNumberAnalytics(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// -----------------------------------------------------------------------------
// GetTemplateAnalytics
// -----------------------------------------------------------------------------

func TestClient_GetTemplateAnalytics_Success(t *testing.T) {
	var capturedPath, capturedQuery string
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
			"id":"tpl-1",
			"name":"welcome",
			"category":"MARKETING",
			"language":"pt_BR",
			"daily_stats":[
				{"date":"2026-04-01","sent":100,"delivered":95,"read":70,"clicked":10},
				{"date":"2026-04-02","sent":120,"delivered":110,"read":80,"clicked":15}
			],
			"quality_score":{"score":"GREEN","reasons":[]}
		}`))
	})
	defer server.Close()

	start := time.Unix(1700000000, 0)
	end := time.Unix(1700086400, 0)
	analytics, err := client.GetTemplateAnalytics(context.Background(), "tpl-1", start, end)
	require.NoError(t, err)

	assert.Equal(t, "/v23.0/tpl-1", capturedPath)
	assert.Contains(t, capturedQuery, "daily_stats.start=1700000000")
	assert.Contains(t, capturedQuery, "daily_stats.end=1700086400")
	assert.Equal(t, "welcome", analytics.TemplateName)
	assert.Equal(t, "MARKETING", analytics.Category)
	assert.Len(t, analytics.DailyStats, 2)
}

func TestClient_GetTemplateAnalytics_APIError(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	_, err := client.GetTemplateAnalytics(context.Background(), "tpl-1", time.Now(), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
