package ctwa

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ClientTestSuite tests the CTWA client
type ClientTestSuite struct {
	suite.Suite
	client *Client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	config := &ClientConfig{
		AttributionWindow: DefaultAttributionWindow(),
	}
	suite.client = NewClient(config)
}

// NewClient tests
func (suite *ClientTestSuite) TestNewClient_WithValidConfig() {
	config := &ClientConfig{}

	client := NewClient(config)

	assert.NotNil(suite.T(), client)
	assert.NotNil(suite.T(), client.referrals)
	assert.NotNil(suite.T(), client.conversions)
	assert.NotNil(suite.T(), client.freeWindows)
}

func (suite *ClientTestSuite) TestNewClient_DefaultsAttributionWindow() {
	config := &ClientConfig{}

	client := NewClient(config)

	assert.Equal(suite.T(), 72*time.Hour, client.attributionCfg.FreeMessageWindow)
}

// ProcessReferral tests
func (suite *ClientTestSuite) TestProcessReferral_Valid() {
	message := &ReferralMessage{
		From: "5511999999999",
		ID:   "msg-123",
		Referral: &ReferralPayload{
			SourceID:   "ad-123",
			SourceType: "ad",
			SourceURL:  "https://fb.com/ad/123",
			AdID:       "ad-123",
			AdTitle:    "Test Ad",
			CampaignID: "camp-123",
		},
	}

	referral, err := suite.client.ProcessReferral("channel-1", message)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), referral)
	assert.Equal(suite.T(), "5511999999999", referral.CustomerPhone)
	assert.Equal(suite.T(), ReferralSourceAd, referral.Source)
	assert.NotNil(suite.T(), referral.FreeWindowExpiry)
}

func (suite *ClientTestSuite) TestProcessReferral_NoReferralData() {
	message := &ReferralMessage{
		From:     "5511999999999",
		ID:       "msg-123",
		Referral: nil,
	}

	_, err := suite.client.ProcessReferral("channel-1", message)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "no referral data")
}

func (suite *ClientTestSuite) TestProcessReferral_CreatesWindow() {
	message := &ReferralMessage{
		From: "5511888888888",
		ID:   "msg-456",
		Referral: &ReferralPayload{
			SourceID:   "ad-456",
			SourceType: "ad",
		},
	}

	_, err := suite.client.ProcessReferral("channel-1", message)
	assert.NoError(suite.T(), err)

	window, found := suite.client.GetFreeWindow("5511888888888")
	assert.True(suite.T(), found)
	assert.True(suite.T(), window.IsActive)
	assert.True(suite.T(), window.IsValid())
}

// GetReferral tests
func (suite *ClientTestSuite) TestGetReferral_NotFound() {
	_, found := suite.client.GetReferral("non-existent")

	assert.False(suite.T(), found)
}

func (suite *ClientTestSuite) TestGetReferralByPhone() {
	// Add two referrals for same phone
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.referrals["ref-1"] = &Referral{
		ID:            "ref-1",
		CustomerPhone: "5511999999999",
		CreatedAt:     now.Add(-1 * time.Hour),
	}
	suite.client.referrals["ref-2"] = &Referral{
		ID:            "ref-2",
		CustomerPhone: "5511999999999",
		CreatedAt:     now, // More recent
	}
	suite.client.mu.Unlock()

	referral, found := suite.client.GetReferralByPhone("5511999999999")

	assert.True(suite.T(), found)
	assert.Equal(suite.T(), "ref-2", referral.ID) // Should return most recent
}

func (suite *ClientTestSuite) TestGetReferralsByCampaign() {
	suite.client.mu.Lock()
	suite.client.referrals["ref-1"] = &Referral{
		ID:         "ref-1",
		CampaignID: "camp-123",
	}
	suite.client.referrals["ref-2"] = &Referral{
		ID:         "ref-2",
		CampaignID: "camp-456",
	}
	suite.client.referrals["ref-3"] = &Referral{
		ID:         "ref-3",
		CampaignID: "camp-123",
	}
	suite.client.mu.Unlock()

	referrals := suite.client.GetReferralsByCampaign("camp-123")

	assert.Len(suite.T(), referrals, 2)
}

// TrackConversion tests
func (suite *ClientTestSuite) TestTrackConversion_Valid() {
	suite.client.mu.Lock()
	suite.client.referrals["ref-conv"] = &Referral{
		ID:            "ref-conv",
		CustomerPhone: "5511999999999",
		AdID:          "ad-123",
		CampaignID:    "camp-123",
	}
	suite.client.mu.Unlock()

	conversion, err := suite.client.TrackConversion("ref-conv", "purchase", 99.90, "BRL")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), conversion)
	assert.Equal(suite.T(), "purchase", conversion.ConversionType)
	assert.Equal(suite.T(), 99.90, conversion.Value)
	assert.Equal(suite.T(), "BRL", conversion.Currency)
}

func (suite *ClientTestSuite) TestTrackConversion_ReferralNotFound() {
	_, err := suite.client.TrackConversion("non-existent", "purchase", 99.90, "BRL")

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "referral not found")
}

// GetConversion tests
func (suite *ClientTestSuite) TestGetConversion_NotFound() {
	_, found := suite.client.GetConversion("non-existent")

	assert.False(suite.T(), found)
}

func (suite *ClientTestSuite) TestGetConversionsByReferral() {
	suite.client.mu.Lock()
	suite.client.conversions["conv-1"] = &AdConversion{
		ID:         "conv-1",
		ReferralID: "ref-123",
	}
	suite.client.conversions["conv-2"] = &AdConversion{
		ID:         "conv-2",
		ReferralID: "ref-456",
	}
	suite.client.conversions["conv-3"] = &AdConversion{
		ID:         "conv-3",
		ReferralID: "ref-123",
	}
	suite.client.mu.Unlock()

	conversions := suite.client.GetConversionsByReferral("ref-123")

	assert.Len(suite.T(), conversions, 2)
}

// Free window tests
func (suite *ClientTestSuite) TestGetFreeWindow_NotFound() {
	_, found := suite.client.GetFreeWindow("non-existent")

	assert.False(suite.T(), found)
}

func (suite *ClientTestSuite) TestGetFreeWindow_Expired() {
	suite.client.mu.Lock()
	suite.client.freeWindows["expired-phone"] = &FreeMessagingWindow{
		CustomerPhone: "expired-phone",
		IsActive:      true,
		ExpiresAt:     time.Now().Add(-1 * time.Hour), // Expired
	}
	suite.client.mu.Unlock()

	_, found := suite.client.GetFreeWindow("expired-phone")

	assert.False(suite.T(), found)
}

func (suite *ClientTestSuite) TestIncrementFreeWindowMessages() {
	suite.client.mu.Lock()
	suite.client.freeWindows["test-phone"] = &FreeMessagingWindow{
		CustomerPhone: "test-phone",
		IsActive:      true,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		MessagesCount: 5,
	}
	suite.client.mu.Unlock()

	suite.client.IncrementFreeWindowMessages("test-phone")

	window, _ := suite.client.GetFreeWindow("test-phone")
	assert.Equal(suite.T(), 6, window.MessagesCount)
}

func (suite *ClientTestSuite) TestCleanupExpiredWindows() {
	suite.client.mu.Lock()
	suite.client.freeWindows["active"] = &FreeMessagingWindow{
		CustomerPhone: "active",
		IsActive:      true,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
	}
	suite.client.freeWindows["expired"] = &FreeMessagingWindow{
		CustomerPhone: "expired",
		IsActive:      true,
		ExpiresAt:     time.Now().Add(-1 * time.Hour),
	}
	suite.client.mu.Unlock()

	suite.client.CleanupExpiredWindows()

	_, foundActive := suite.client.GetFreeWindow("active")
	_, foundExpired := suite.client.GetFreeWindow("expired")

	assert.True(suite.T(), foundActive)
	assert.False(suite.T(), foundExpired)
}

// Statistics tests
func (suite *ClientTestSuite) TestGetStats_Empty() {
	stats := suite.client.GetStats()

	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), 0, stats.TotalReferrals)
	assert.Equal(suite.T(), 0, stats.TotalConversions)
}

func (suite *ClientTestSuite) TestGetStats_WithData() {
	suite.client.mu.Lock()
	suite.client.referrals["ref-1"] = &Referral{
		ID:           "ref-1",
		Source:       ReferralSourceAd,
		CampaignID:   "camp-1",
		CampaignName: "Campaign 1",
		AdID:         "ad-1",
		AdName:       "Ad 1",
	}
	suite.client.referrals["ref-2"] = &Referral{
		ID:           "ref-2",
		Source:       ReferralSourceAd,
		CampaignID:   "camp-1",
		CampaignName: "Campaign 1",
		AdID:         "ad-2",
		AdName:       "Ad 2",
	}
	suite.client.conversions["conv-1"] = &AdConversion{
		ID:         "conv-1",
		CampaignID: "camp-1",
		AdID:       "ad-1",
		Value:      100.0,
		Currency:   "BRL",
	}
	suite.client.mu.Unlock()

	stats := suite.client.GetStats()

	assert.Equal(suite.T(), 2, stats.TotalReferrals)
	assert.Equal(suite.T(), 1, stats.TotalConversions)
	assert.Equal(suite.T(), 100.0, stats.TotalValue)
	assert.Equal(suite.T(), 50.0, stats.ConversionRate) // 1/2 * 100
}

func (suite *ClientTestSuite) TestGetStatsByPeriod() {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -7)

	suite.client.mu.Lock()
	suite.client.referrals["ref-today"] = &Referral{
		ID:        "ref-today",
		Source:    ReferralSourceAd,
		CreatedAt: now,
	}
	suite.client.referrals["ref-old"] = &Referral{
		ID:        "ref-old",
		Source:    ReferralSourceAd,
		CreatedAt: lastWeek.AddDate(0, 0, -1), // Before period
	}
	suite.client.mu.Unlock()

	stats := suite.client.GetStatsByPeriod(yesterday, now.Add(time.Hour))

	assert.Equal(suite.T(), 1, stats.TotalReferrals)
}

// Top performing ads tests
func (suite *ClientTestSuite) TestGetTopPerformingAds() {
	suite.client.mu.Lock()
	suite.client.referrals["ref-1"] = &Referral{
		ID:           "ref-1",
		AdID:         "ad-1",
		AdName:       "Ad 1",
		CampaignID:   "camp-1",
		CampaignName: "Campaign 1",
	}
	suite.client.referrals["ref-2"] = &Referral{
		ID:           "ref-2",
		AdID:         "ad-1",
		AdName:       "Ad 1",
		CampaignID:   "camp-1",
		CampaignName: "Campaign 1",
	}
	suite.client.referrals["ref-3"] = &Referral{
		ID:           "ref-3",
		AdID:         "ad-2",
		AdName:       "Ad 2",
		CampaignID:   "camp-1",
		CampaignName: "Campaign 1",
	}
	suite.client.conversions["conv-1"] = &AdConversion{
		ID:   "conv-1",
		AdID: "ad-1",
	}
	suite.client.conversions["conv-2"] = &AdConversion{
		ID:   "conv-2",
		AdID: "ad-1",
	}
	suite.client.mu.Unlock()

	topAds := suite.client.GetTopPerformingAds(10)

	assert.Len(suite.T(), topAds, 2)
	assert.Equal(suite.T(), "ad-1", topAds[0].AdID) // Most conversions first
	assert.Equal(suite.T(), 2, topAds[0].Conversions)
}

// Report tests
func (suite *ClientTestSuite) TestGenerateReport() {
	suite.client.mu.Lock()
	suite.client.referrals["ref-1"] = &Referral{
		ID:           "ref-1",
		Source:       ReferralSourceAd,
		CampaignID:   "camp-1",
		CampaignName: "Campaign 1",
		CreatedAt:    time.Now(),
	}
	suite.client.mu.Unlock()

	report := suite.client.GenerateReport(
		time.Now().AddDate(0, 0, -7),
		time.Now().Add(time.Hour),
	)

	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotNil(suite.T(), report.GeneratedAt)
}

// FreeMessagingWindow tests
func TestFreeMessagingWindow_IsValid(t *testing.T) {
	validWindow := &FreeMessagingWindow{
		IsActive:  true,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	expiredWindow := &FreeMessagingWindow{
		IsActive:  true,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	inactiveWindow := &FreeMessagingWindow{
		IsActive:  false,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	assert.True(t, validWindow.IsValid())
	assert.False(t, expiredWindow.IsValid())
	assert.False(t, inactiveWindow.IsValid())
}

func TestFreeMessagingWindow_TimeRemaining(t *testing.T) {
	window := &FreeMessagingWindow{
		IsActive:  true,
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	expiredWindow := &FreeMessagingWindow{
		IsActive:  true,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	assert.True(t, window.TimeRemaining() > 0)
	assert.Equal(t, time.Duration(0), expiredWindow.TimeRemaining())
}

// DefaultAttributionWindow test
func TestDefaultAttributionWindow(t *testing.T) {
	window := DefaultAttributionWindow()

	assert.NotNil(t, window)
	assert.Equal(t, 7*24*time.Hour, window.ClickWindow)
	assert.Equal(t, 24*time.Hour, window.ViewWindow)
	assert.Equal(t, 72*time.Hour, window.FreeMessageWindow)
}

// NewCTWAEvent test
func TestNewCTWAEvent(t *testing.T) {
	referral := &Referral{
		ID:            "ref-123",
		CustomerPhone: "5511999999999",
	}

	event := NewCTWAEvent(CTWAEventReferralReceived, referral)

	assert.Equal(t, CTWAEventReferralReceived, event.Type)
	assert.NotNil(t, event.Referral)
	assert.NotNil(t, event.Timestamp)
}
