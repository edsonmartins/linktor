package ctwa

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Client handles Click-to-WhatsApp Ads tracking and attribution
type Client struct {
	mu              sync.RWMutex
	referrals       map[string]*Referral       // key: referral_id
	conversions     map[string]*AdConversion   // key: conversion_id
	freeWindows     map[string]*FreeMessagingWindow // key: customer_phone
	attributionCfg  *AttributionWindow
}

// ClientConfig represents configuration for the CTWA client
type ClientConfig struct {
	AttributionWindow *AttributionWindow
}

// NewClient creates a new CTWA tracking client
func NewClient(config *ClientConfig) *Client {
	attrWindow := config.AttributionWindow
	if attrWindow == nil {
		attrWindow = DefaultAttributionWindow()
	}

	return &Client{
		referrals:      make(map[string]*Referral),
		conversions:    make(map[string]*AdConversion),
		freeWindows:    make(map[string]*FreeMessagingWindow),
		attributionCfg: attrWindow,
	}
}

// =============================================================================
// Referral Tracking
// =============================================================================

// ProcessReferral processes an incoming referral from a webhook
func (c *Client) ProcessReferral(channelID string, message *ReferralMessage) (*Referral, error) {
	if message.Referral == nil {
		return nil, fmt.Errorf("no referral data in message")
	}

	payload := message.Referral
	now := time.Now()
	freeWindowExpiry := now.Add(c.attributionCfg.FreeMessageWindow)

	// Determine referral source
	source := ReferralSourceUnknown
	switch payload.SourceType {
	case "ad":
		source = ReferralSourceAd
	case "post":
		source = ReferralSourcePost
	case "story":
		source = ReferralSourceStory
	}

	// Determine ad type
	adType := AdTypeFacebook // Default
	// Could be determined from source_url or other metadata

	referral := &Referral{
		ID:               generateID("ref"),
		ChannelID:        channelID,
		CustomerPhone:    message.From,
		Source:           source,
		SourceID:         payload.SourceID,
		SourceURL:        payload.SourceURL,
		Headline:         payload.Headline,
		Body:             payload.Body,
		MediaType:        payload.MediaType,
		MediaURL:         payload.MediaURL,
		CTWAClid:         payload.CTWAClid,
		AdID:             payload.AdID,
		AdName:           payload.AdTitle,
		AdSetID:          payload.AdSetID,
		AdSetName:        payload.AdSetName,
		CampaignID:       payload.CampaignID,
		CampaignName:     payload.CampaignName,
		AdType:           adType,
		MessageID:        message.ID,
		FreeWindowExpiry: &freeWindowExpiry,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	c.mu.Lock()
	c.referrals[referral.ID] = referral

	// Create free messaging window
	c.freeWindows[message.From] = &FreeMessagingWindow{
		ReferralID:    referral.ID,
		CustomerPhone: message.From,
		StartsAt:      now,
		ExpiresAt:     freeWindowExpiry,
		IsActive:      true,
		MessagesCount: 0,
	}
	c.mu.Unlock()

	return referral, nil
}

// GetReferral retrieves a referral by ID
func (c *Client) GetReferral(referralID string) (*Referral, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	referral, ok := c.referrals[referralID]
	return referral, ok
}

// GetReferralByPhone retrieves the most recent referral for a phone number
func (c *Client) GetReferralByPhone(phone string) (*Referral, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var latestReferral *Referral
	for _, ref := range c.referrals {
		if ref.CustomerPhone == phone {
			if latestReferral == nil || ref.CreatedAt.After(latestReferral.CreatedAt) {
				latestReferral = ref
			}
		}
	}

	return latestReferral, latestReferral != nil
}

// GetReferralsByCampaign retrieves all referrals for a campaign
func (c *Client) GetReferralsByCampaign(campaignID string) []*Referral {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*Referral
	for _, ref := range c.referrals {
		if ref.CampaignID == campaignID {
			result = append(result, ref)
		}
	}
	return result
}

// =============================================================================
// Conversion Tracking
// =============================================================================

// TrackConversion tracks a conversion from a CTWA referral
func (c *Client) TrackConversion(referralID, conversionType string, value float64, currency string) (*AdConversion, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	referral, ok := c.referrals[referralID]
	if !ok {
		return nil, fmt.Errorf("referral not found: %s", referralID)
	}

	now := time.Now()
	conversion := &AdConversion{
		ID:             generateID("conv"),
		ReferralID:     referralID,
		OrganizationID: referral.OrganizationID,
		ChannelID:      referral.ChannelID,
		CustomerPhone:  referral.CustomerPhone,
		ConversionType: conversionType,
		Status:         ConversionStatusConverted,
		Value:          value,
		Currency:       currency,
		AdID:           referral.AdID,
		CampaignID:     referral.CampaignID,
		AttributedAt:   &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	c.conversions[conversion.ID] = conversion

	return conversion, nil
}

// GetConversion retrieves a conversion by ID
func (c *Client) GetConversion(conversionID string) (*AdConversion, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conversion, ok := c.conversions[conversionID]
	return conversion, ok
}

// GetConversionsByReferral retrieves all conversions for a referral
func (c *Client) GetConversionsByReferral(referralID string) []*AdConversion {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*AdConversion
	for _, conv := range c.conversions {
		if conv.ReferralID == referralID {
			result = append(result, conv)
		}
	}
	return result
}

// =============================================================================
// Free Messaging Window
// =============================================================================

// GetFreeWindow retrieves the free messaging window for a customer
func (c *Client) GetFreeWindow(customerPhone string) (*FreeMessagingWindow, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	window, ok := c.freeWindows[customerPhone]
	if !ok {
		return nil, false
	}

	// Check if still valid
	if !window.IsValid() {
		return nil, false
	}

	return window, true
}

// IncrementFreeWindowMessages increments the message count for a free window
func (c *Client) IncrementFreeWindowMessages(customerPhone string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if window, ok := c.freeWindows[customerPhone]; ok {
		window.MessagesCount++
	}
}

// CleanupExpiredWindows removes expired free messaging windows
func (c *Client) CleanupExpiredWindows() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for phone, window := range c.freeWindows {
		if now.After(window.ExpiresAt) {
			window.IsActive = false
			delete(c.freeWindows, phone)
		}
	}
}

// =============================================================================
// Statistics
// =============================================================================

// GetStats returns CTWA statistics
func (c *Client) GetStats() *CTWAStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &CTWAStats{
		ByCampaign: make(map[string]*CampaignStats),
		ByAdSet:    make(map[string]*AdSetStats),
		ByAd:       make(map[string]*AdStats),
		BySource:   make(map[ReferralSource]int),
	}

	// Count referrals
	for _, ref := range c.referrals {
		stats.TotalReferrals++
		stats.BySource[ref.Source]++

		// Campaign stats
		if ref.CampaignID != "" {
			if _, ok := stats.ByCampaign[ref.CampaignID]; !ok {
				stats.ByCampaign[ref.CampaignID] = &CampaignStats{
					CampaignID:   ref.CampaignID,
					CampaignName: ref.CampaignName,
				}
			}
			stats.ByCampaign[ref.CampaignID].Referrals++
		}

		// AdSet stats
		if ref.AdSetID != "" {
			if _, ok := stats.ByAdSet[ref.AdSetID]; !ok {
				stats.ByAdSet[ref.AdSetID] = &AdSetStats{
					AdSetID:    ref.AdSetID,
					AdSetName:  ref.AdSetName,
					CampaignID: ref.CampaignID,
				}
			}
			stats.ByAdSet[ref.AdSetID].Referrals++
		}

		// Ad stats
		if ref.AdID != "" {
			if _, ok := stats.ByAd[ref.AdID]; !ok {
				stats.ByAd[ref.AdID] = &AdStats{
					AdID:       ref.AdID,
					AdName:     ref.AdName,
					AdSetID:    ref.AdSetID,
					CampaignID: ref.CampaignID,
				}
			}
			stats.ByAd[ref.AdID].Referrals++
		}
	}

	// Count conversions
	for _, conv := range c.conversions {
		stats.TotalConversions++
		stats.TotalValue += conv.Value
		if stats.Currency == "" && conv.Currency != "" {
			stats.Currency = conv.Currency
		}

		// Update campaign stats
		if conv.CampaignID != "" {
			if camp, ok := stats.ByCampaign[conv.CampaignID]; ok {
				camp.Conversions++
				camp.TotalValue += conv.Value
			}
		}

		// Update ad stats
		if conv.AdID != "" {
			if ad, ok := stats.ByAd[conv.AdID]; ok {
				ad.Conversions++
				ad.TotalValue += conv.Value
			}
		}
	}

	// Calculate rates
	if stats.TotalReferrals > 0 {
		stats.ConversionRate = float64(stats.TotalConversions) / float64(stats.TotalReferrals) * 100
	}
	if stats.TotalConversions > 0 {
		stats.AverageValue = stats.TotalValue / float64(stats.TotalConversions)
	}

	// Calculate campaign conversion rates
	for _, camp := range stats.ByCampaign {
		if camp.Referrals > 0 {
			camp.ConversionRate = float64(camp.Conversions) / float64(camp.Referrals) * 100
		}
	}

	// Calculate ad conversion rates
	for _, ad := range stats.ByAd {
		if ad.Referrals > 0 {
			ad.ConversionRate = float64(ad.Conversions) / float64(ad.Referrals) * 100
		}
	}

	return stats
}

// GetStatsByPeriod returns CTWA statistics for a period
func (c *Client) GetStatsByPeriod(startDate, endDate time.Time) *CTWAStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &CTWAStats{
		ByCampaign: make(map[string]*CampaignStats),
		ByAdSet:    make(map[string]*AdSetStats),
		ByAd:       make(map[string]*AdStats),
		BySource:   make(map[ReferralSource]int),
		DailyStats: make([]DailyCTWAStats, 0),
	}

	dailyData := make(map[string]*DailyCTWAStats)

	// Filter referrals by period
	for _, ref := range c.referrals {
		if ref.CreatedAt.Before(startDate) || ref.CreatedAt.After(endDate) {
			continue
		}

		stats.TotalReferrals++
		stats.BySource[ref.Source]++

		// Daily stats
		date := ref.CreatedAt.Format("2006-01-02")
		if _, ok := dailyData[date]; !ok {
			dailyData[date] = &DailyCTWAStats{Date: date}
		}
		dailyData[date].Referrals++

		// Campaign stats
		if ref.CampaignID != "" {
			if _, ok := stats.ByCampaign[ref.CampaignID]; !ok {
				stats.ByCampaign[ref.CampaignID] = &CampaignStats{
					CampaignID:   ref.CampaignID,
					CampaignName: ref.CampaignName,
				}
			}
			stats.ByCampaign[ref.CampaignID].Referrals++
		}

		// Ad stats
		if ref.AdID != "" {
			if _, ok := stats.ByAd[ref.AdID]; !ok {
				stats.ByAd[ref.AdID] = &AdStats{
					AdID:       ref.AdID,
					AdName:     ref.AdName,
					AdSetID:    ref.AdSetID,
					CampaignID: ref.CampaignID,
				}
			}
			stats.ByAd[ref.AdID].Referrals++
		}
	}

	// Filter conversions by period
	for _, conv := range c.conversions {
		if conv.CreatedAt.Before(startDate) || conv.CreatedAt.After(endDate) {
			continue
		}

		stats.TotalConversions++
		stats.TotalValue += conv.Value
		if stats.Currency == "" && conv.Currency != "" {
			stats.Currency = conv.Currency
		}

		// Daily stats
		date := conv.CreatedAt.Format("2006-01-02")
		if daily, ok := dailyData[date]; ok {
			daily.Conversions++
			daily.TotalValue += conv.Value
		}

		// Update campaign stats
		if conv.CampaignID != "" {
			if camp, ok := stats.ByCampaign[conv.CampaignID]; ok {
				camp.Conversions++
				camp.TotalValue += conv.Value
			}
		}

		// Update ad stats
		if conv.AdID != "" {
			if ad, ok := stats.ByAd[conv.AdID]; ok {
				ad.Conversions++
				ad.TotalValue += conv.Value
			}
		}
	}

	// Calculate rates
	if stats.TotalReferrals > 0 {
		stats.ConversionRate = float64(stats.TotalConversions) / float64(stats.TotalReferrals) * 100
	}
	if stats.TotalConversions > 0 {
		stats.AverageValue = stats.TotalValue / float64(stats.TotalConversions)
	}

	// Calculate campaign rates
	for _, camp := range stats.ByCampaign {
		if camp.Referrals > 0 {
			camp.ConversionRate = float64(camp.Conversions) / float64(camp.Referrals) * 100
		}
	}

	// Calculate ad rates
	for _, ad := range stats.ByAd {
		if ad.Referrals > 0 {
			ad.ConversionRate = float64(ad.Conversions) / float64(ad.Referrals) * 100
		}
	}

	// Convert daily data to slice and calculate rates
	for _, daily := range dailyData {
		if daily.Referrals > 0 {
			daily.ConversionRate = float64(daily.Conversions) / float64(daily.Referrals) * 100
		}
		stats.DailyStats = append(stats.DailyStats, *daily)
	}

	return stats
}

// GetTopPerformingAds returns the top performing ads by conversions
func (c *Client) GetTopPerformingAds(limit int) []AdPerformance {
	stats := c.GetStats()

	ads := make([]AdPerformance, 0, len(stats.ByAd))
	for _, ad := range stats.ByAd {
		campaignName := ""
		if camp, ok := stats.ByCampaign[ad.CampaignID]; ok {
			campaignName = camp.CampaignName
		}
		ads = append(ads, AdPerformance{
			AdID:           ad.AdID,
			AdName:         ad.AdName,
			CampaignName:   campaignName,
			Referrals:      ad.Referrals,
			Conversions:    ad.Conversions,
			ConversionRate: ad.ConversionRate,
			TotalValue:     ad.TotalValue,
		})
	}

	// Sort by conversions descending
	sort.Slice(ads, func(i, j int) bool {
		return ads[i].Conversions > ads[j].Conversions
	})

	if limit > 0 && limit < len(ads) {
		ads = ads[:limit]
	}

	return ads
}

// =============================================================================
// Reports
// =============================================================================

// GenerateReport generates a CTWA performance report
func (c *Client) GenerateReport(startDate, endDate time.Time) *CTWAReport {
	stats := c.GetStatsByPeriod(startDate, endDate)

	// Convert campaign stats to slice
	campaigns := make([]CampaignStats, 0, len(stats.ByCampaign))
	for _, camp := range stats.ByCampaign {
		campaigns = append(campaigns, *camp)
	}

	// Sort campaigns by conversions
	sort.Slice(campaigns, func(i, j int) bool {
		return campaigns[i].Conversions > campaigns[j].Conversions
	})

	// Get top ads
	topAds := c.GetTopPerformingAds(10)

	// Build trends
	trends := make([]TrendData, 0, len(stats.DailyStats))
	for _, daily := range stats.DailyStats {
		trends = append(trends, TrendData{
			Date:        daily.Date,
			Referrals:   daily.Referrals,
			Conversions: daily.Conversions,
			Value:       daily.TotalValue,
		})
	}

	// Sort trends by date
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Date < trends[j].Date
	})

	// Generate insights
	insights := c.generateInsights(stats)

	return &CTWAReport{
		Period: ReportPeriod{
			Start: startDate,
			End:   endDate,
		},
		Summary:     *stats,
		Campaigns:   campaigns,
		TopAds:      topAds,
		Trends:      trends,
		Insights:    insights,
		GeneratedAt: time.Now(),
	}
}

// generateInsights generates insights from statistics
func (c *Client) generateInsights(stats *CTWAStats) []ReportInsight {
	var insights []ReportInsight

	// Conversion rate insight
	if stats.ConversionRate > 5 {
		insights = append(insights, ReportInsight{
			Type:        "conversion_rate",
			Title:       "High Conversion Rate",
			Description: fmt.Sprintf("Your conversion rate of %.1f%% is above average", stats.ConversionRate),
			Impact:      "positive",
		})
	} else if stats.ConversionRate < 1 && stats.TotalReferrals > 10 {
		insights = append(insights, ReportInsight{
			Type:        "conversion_rate",
			Title:       "Low Conversion Rate",
			Description: "Consider optimizing your ad targeting or messaging",
			Impact:      "negative",
		})
	}

	// Top performing campaign insight
	var topCampaign *CampaignStats
	for _, camp := range stats.ByCampaign {
		if topCampaign == nil || camp.Conversions > topCampaign.Conversions {
			topCampaign = camp
		}
	}
	if topCampaign != nil && topCampaign.Conversions > 0 {
		insights = append(insights, ReportInsight{
			Type:        "top_campaign",
			Title:       "Top Performing Campaign",
			Description: fmt.Sprintf("%s is your best performing campaign with %d conversions", topCampaign.CampaignName, topCampaign.Conversions),
			Impact:      "positive",
		})
	}

	return insights
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateID generates a unique ID with prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
