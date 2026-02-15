package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ClientTestSuite tests the analytics client
type ClientTestSuite struct {
	suite.Suite
	client *Client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		BusinessID:    "test-business",
		PhoneNumberID: "test-phone",
		APIVersion:    "v21.0",
	}
	suite.client = NewClient(config)
}

// NewClient tests
func (suite *ClientTestSuite) TestNewClient_WithValidConfig() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		BusinessID:    "test-business",
		PhoneNumberID: "test-phone",
	}

	client := NewClient(config)

	assert.NotNil(suite.T(), client)
	assert.NotNil(suite.T(), client.httpClient)
	assert.NotNil(suite.T(), client.cache)
}

func (suite *ClientTestSuite) TestNewClient_DefaultsAPIVersion() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		BusinessID:    "test-business",
		PhoneNumberID: "test-phone",
		APIVersion:    "", // Empty, should default
	}

	client := NewClient(config)

	assert.Equal(suite.T(), "v21.0", client.apiVersion)
}

// Cache tests
func (suite *ClientTestSuite) TestCache_SetAndGet() {
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 100,
		TotalCost:          50.0,
	}

	req := &AnalyticsRequest{
		PhoneNumberID: "test-phone",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	}

	key := suite.client.getCacheKey(req)
	suite.client.setInCache(key, analytics, 15*time.Minute)

	cached := suite.client.getFromCache(key)

	assert.NotNil(suite.T(), cached)
	assert.Equal(suite.T(), 100, cached.TotalConversations)
	assert.Equal(suite.T(), 50.0, cached.TotalCost)
}

func (suite *ClientTestSuite) TestCache_ExpiredEntry() {
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 100,
	}

	req := &AnalyticsRequest{
		PhoneNumberID: "test-phone",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	}

	key := suite.client.getCacheKey(req)
	// Set with expired TTL
	suite.client.setInCache(key, analytics, -1*time.Second)

	cached := suite.client.getFromCache(key)

	assert.Nil(suite.T(), cached)
}

func (suite *ClientTestSuite) TestClearCache() {
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 100,
	}

	req := &AnalyticsRequest{
		PhoneNumberID: "test-phone",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	}

	key := suite.client.getCacheKey(req)
	suite.client.setInCache(key, analytics, 15*time.Minute)

	suite.client.ClearCache()

	cached := suite.client.getFromCache(key)
	assert.Nil(suite.T(), cached)
}

func (suite *ClientTestSuite) TestCleanupExpiredCache() {
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 100,
	}

	// Add expired entry
	suite.client.mu.Lock()
	suite.client.cache["expired-key"] = &cachedAnalytics{
		data:      analytics,
		expiresAt: time.Now().Add(-1 * time.Hour),
	}
	suite.client.mu.Unlock()

	suite.client.CleanupExpiredCache()

	suite.client.mu.RLock()
	_, exists := suite.client.cache["expired-key"]
	suite.client.mu.RUnlock()

	assert.False(suite.T(), exists)
}

// AggregateConversationData tests
func (suite *ClientTestSuite) TestAggregateConversationData_EmptyData() {
	req := &AnalyticsRequest{
		PhoneNumberID: "test-phone",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	}

	result := suite.client.aggregateConversationData(req, []ConversationDataPoint{})

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0, result.TotalConversations)
	assert.Equal(suite.T(), 0.0, result.TotalCost)
	assert.Empty(suite.T(), result.Timeline)
}

func (suite *ClientTestSuite) TestAggregateConversationData_WithData() {
	req := &AnalyticsRequest{
		PhoneNumberID: "test-phone",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	}

	now := time.Now().Unix()
	dataPoints := []ConversationDataPoint{
		{
			Start:                 now,
			Sent:                  10,
			Cost:                  5.0,
			ConversationType:      "REGULAR",
			ConversationDirection: "user_initiated",
			Country:               "BR",
			ConversationCategory:  "MARKETING",
		},
		{
			Start:                 now,
			Sent:                  5,
			Cost:                  2.5,
			ConversationType:      "REGULAR",
			ConversationDirection: "business_initiated",
			Country:               "BR",
			ConversationCategory:  "UTILITY",
		},
	}

	result := suite.client.aggregateConversationData(req, dataPoints)

	assert.Equal(suite.T(), 15, result.TotalConversations)
	assert.Equal(suite.T(), 7.5, result.TotalCost)
	assert.Equal(suite.T(), 15, result.ByCountry["BR"])
}

// ExportToCSV tests
func (suite *ClientTestSuite) TestExportToCSV() {
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 100,
		Timeline: []DailyAnalytics{
			{
				Date:          "2024-01-01",
				Conversations: 50,
				Cost:          25.0,
				ByCategory: map[ConversationCategory]int{
					CategoryMarketing: 30,
					CategoryUtility:   20,
				},
			},
		},
	}

	csv, err := suite.client.ExportToCSV(analytics)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), csv)
	assert.Contains(suite.T(), string(csv), "Date,Conversations,Cost")
	assert.Contains(suite.T(), string(csv), "2024-01-01")
}

// ExportToJSON tests
func (suite *ClientTestSuite) TestExportToJSON() {
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 100,
	}

	json, err := suite.client.ExportToJSON(analytics)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), json)
	assert.Contains(suite.T(), string(json), "test-phone")
	assert.Contains(suite.T(), string(json), "100")
}

// GetAggregatedStats tests
func (suite *ClientTestSuite) TestGetAggregatedStats_ZeroConversations() {
	// This test verifies the fix for division by zero
	ctx := context.Background()
	req := &AnalyticsRequest{
		PhoneNumberID: "test-phone",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		Granularity:   "DAILY",
	}

	// Pre-cache empty data to avoid HTTP call
	analytics := &ConversationAnalytics{
		PhoneNumberID:      "test-phone",
		TotalConversations: 0,
		ByCountry:          make(map[string]int),
	}
	key := suite.client.getCacheKey(req)
	suite.client.setInCache(key, analytics, 15*time.Minute)

	stats, err := suite.client.GetAggregatedStats(ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), 0, stats.TotalConversations)
}
