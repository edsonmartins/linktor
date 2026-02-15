package ctwa

import (
	"time"
)

// =============================================================================
// CTWA (Click-to-WhatsApp Ads) Types
// =============================================================================

// ReferralSource represents the source of a CTWA referral
type ReferralSource string

const (
	ReferralSourceAd       ReferralSource = "ad"
	ReferralSourcePost     ReferralSource = "post"
	ReferralSourceStory    ReferralSource = "story"
	ReferralSourceUnknown  ReferralSource = "unknown"
)

// AdType represents the type of ad
type AdType string

const (
	AdTypeFacebook  AdType = "facebook"
	AdTypeInstagram AdType = "instagram"
)

// ConversionStatus represents the status of a conversion
type ConversionStatus string

const (
	ConversionStatusPending    ConversionStatus = "pending"
	ConversionStatusConverted  ConversionStatus = "converted"
	ConversionStatusExpired    ConversionStatus = "expired"
)

// =============================================================================
// Referral Entity
// =============================================================================

// Referral represents a CTWA referral from an ad click
type Referral struct {
	ID               string         `json:"id"`
	OrganizationID   string         `json:"organization_id"`
	ChannelID        string         `json:"channel_id"`
	CustomerPhone    string         `json:"customer_phone"`
	Source           ReferralSource `json:"source"`
	SourceID         string         `json:"source_id"` // Ad ID, Post ID, etc.
	SourceURL        string         `json:"source_url,omitempty"`
	Headline         string         `json:"headline,omitempty"`
	Body             string         `json:"body,omitempty"`
	MediaType        string         `json:"media_type,omitempty"` // image, video
	MediaURL         string         `json:"media_url,omitempty"`
	CTWAClid         string         `json:"ctwa_clid,omitempty"` // Click ID for attribution
	AdID             string         `json:"ad_id,omitempty"`
	AdName           string         `json:"ad_name,omitempty"`
	AdSetID          string         `json:"adset_id,omitempty"`
	AdSetName        string         `json:"adset_name,omitempty"`
	CampaignID       string         `json:"campaign_id,omitempty"`
	CampaignName     string         `json:"campaign_name,omitempty"`
	AdType           AdType         `json:"ad_type,omitempty"`
	MessageID        string         `json:"message_id,omitempty"` // First message from user
	ConversationID   string         `json:"conversation_id,omitempty"`
	FreeWindowExpiry *time.Time     `json:"free_window_expiry,omitempty"` // 72h free messaging
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// =============================================================================
// Ad Conversion Entity
// =============================================================================

// AdConversion represents a conversion from a CTWA ad
type AdConversion struct {
	ID             string           `json:"id"`
	ReferralID     string           `json:"referral_id"`
	OrganizationID string           `json:"organization_id"`
	ChannelID      string           `json:"channel_id"`
	CustomerPhone  string           `json:"customer_phone"`
	ConversionType string           `json:"conversion_type"` // message, purchase, signup, etc.
	Status         ConversionStatus `json:"status"`
	Value          float64          `json:"value,omitempty"`
	Currency       string           `json:"currency,omitempty"`
	AdID           string           `json:"ad_id"`
	CampaignID     string           `json:"campaign_id"`
	AttributedAt   *time.Time       `json:"attributed_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// =============================================================================
// CTWA Message Types
// =============================================================================

// ReferralMessage represents an incoming message with referral data
type ReferralMessage struct {
	From     string           `json:"from"`
	ID       string           `json:"id"`
	Text     string           `json:"text,omitempty"`
	Referral *ReferralPayload `json:"referral,omitempty"`
}

// ReferralPayload represents the referral data in a webhook message
type ReferralPayload struct {
	SourceID    string `json:"source_id"`
	SourceType  string `json:"source_type"` // ad, post
	SourceURL   string `json:"source_url,omitempty"`
	Headline    string `json:"headline,omitempty"`
	Body        string `json:"body,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
	MediaURL    string `json:"media_url,omitempty"`
	CTWAClid    string `json:"ctwa_clid,omitempty"`
	// Ad details (from extended referral data)
	AdID         string `json:"ad_id,omitempty"`
	AdTitle      string `json:"ad_title,omitempty"`
	AdSetID      string `json:"adset_id,omitempty"`
	AdSetName    string `json:"adset_name,omitempty"`
	CampaignID   string `json:"campaign_id,omitempty"`
	CampaignName string `json:"campaign_name,omitempty"`
}

// =============================================================================
// CTWA Statistics Types
// =============================================================================

// CTWAStats represents CTWA statistics
type CTWAStats struct {
	TotalReferrals      int                        `json:"total_referrals"`
	TotalConversions    int                        `json:"total_conversions"`
	ConversionRate      float64                    `json:"conversion_rate"`
	TotalValue          float64                    `json:"total_value"`
	Currency            string                     `json:"currency"`
	AverageValue        float64                    `json:"average_value"`
	ByCampaign          map[string]*CampaignStats  `json:"by_campaign"`
	ByAdSet             map[string]*AdSetStats     `json:"by_adset"`
	ByAd                map[string]*AdStats        `json:"by_ad"`
	BySource            map[ReferralSource]int     `json:"by_source"`
	DailyStats          []DailyCTWAStats           `json:"daily_stats,omitempty"`
	TopPerformingAds    []AdPerformance            `json:"top_performing_ads,omitempty"`
}

// CampaignStats represents statistics for a campaign
type CampaignStats struct {
	CampaignID     string  `json:"campaign_id"`
	CampaignName   string  `json:"campaign_name"`
	Referrals      int     `json:"referrals"`
	Conversions    int     `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	TotalValue     float64 `json:"total_value"`
	ROI            float64 `json:"roi,omitempty"`
}

// AdSetStats represents statistics for an ad set
type AdSetStats struct {
	AdSetID        string  `json:"adset_id"`
	AdSetName      string  `json:"adset_name"`
	CampaignID     string  `json:"campaign_id"`
	Referrals      int     `json:"referrals"`
	Conversions    int     `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	TotalValue     float64 `json:"total_value"`
}

// AdStats represents statistics for an ad
type AdStats struct {
	AdID           string  `json:"ad_id"`
	AdName         string  `json:"ad_name"`
	AdSetID        string  `json:"adset_id"`
	CampaignID     string  `json:"campaign_id"`
	Referrals      int     `json:"referrals"`
	Conversions    int     `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	TotalValue     float64 `json:"total_value"`
}

// DailyCTWAStats represents daily CTWA statistics
type DailyCTWAStats struct {
	Date           string  `json:"date"`
	Referrals      int     `json:"referrals"`
	Conversions    int     `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	TotalValue     float64 `json:"total_value"`
}

// AdPerformance represents ad performance metrics
type AdPerformance struct {
	AdID           string  `json:"ad_id"`
	AdName         string  `json:"ad_name"`
	CampaignName   string  `json:"campaign_name"`
	Referrals      int     `json:"referrals"`
	Conversions    int     `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	TotalValue     float64 `json:"total_value"`
	CostPerResult  float64 `json:"cost_per_result,omitempty"`
	ROAS           float64 `json:"roas,omitempty"` // Return on Ad Spend
}

// =============================================================================
// Attribution Types
// =============================================================================

// AttributionWindow represents the attribution window configuration
type AttributionWindow struct {
	ClickWindow      time.Duration `json:"click_window"`       // Default: 7 days
	ViewWindow       time.Duration `json:"view_window"`        // Default: 1 day
	FreeMessageWindow time.Duration `json:"free_message_window"` // 72 hours
}

// DefaultAttributionWindow returns the default attribution window
func DefaultAttributionWindow() *AttributionWindow {
	return &AttributionWindow{
		ClickWindow:       7 * 24 * time.Hour,
		ViewWindow:        24 * time.Hour,
		FreeMessageWindow: 72 * time.Hour,
	}
}

// =============================================================================
// Free Messaging Window
// =============================================================================

// FreeMessagingWindow represents the 72-hour free messaging window from CTWA
type FreeMessagingWindow struct {
	ReferralID    string    `json:"referral_id"`
	CustomerPhone string    `json:"customer_phone"`
	StartsAt      time.Time `json:"starts_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	IsActive      bool      `json:"is_active"`
	MessagesCount int       `json:"messages_count"`
}

// IsValid checks if the free messaging window is still valid
func (f *FreeMessagingWindow) IsValid() bool {
	return f.IsActive && time.Now().Before(f.ExpiresAt)
}

// TimeRemaining returns the time remaining in the free window
func (f *FreeMessagingWindow) TimeRemaining() time.Duration {
	if !f.IsValid() {
		return 0
	}
	return time.Until(f.ExpiresAt)
}

// =============================================================================
// CTWA Event Types
// =============================================================================

// CTWAEvent represents a CTWA event for pub/sub
type CTWAEvent struct {
	Type      string     `json:"type"`
	Referral  *Referral  `json:"referral,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// CTWA event type constants
const (
	CTWAEventReferralReceived   = "ctwa.referral.received"
	CTWAEventConversionTracked  = "ctwa.conversion.tracked"
	CTWAEventFreeWindowExpired  = "ctwa.free_window.expired"
)

// NewCTWAEvent creates a new CTWA event
func NewCTWAEvent(eventType string, referral *Referral) *CTWAEvent {
	return &CTWAEvent{
		Type:      eventType,
		Referral:  referral,
		Timestamp: time.Now(),
	}
}

// =============================================================================
// Report Types
// =============================================================================

// CTWAReport represents a CTWA performance report
type CTWAReport struct {
	Period        ReportPeriod     `json:"period"`
	Summary       CTWAStats        `json:"summary"`
	Campaigns     []CampaignStats  `json:"campaigns"`
	TopAds        []AdPerformance  `json:"top_ads"`
	Trends        []TrendData      `json:"trends"`
	Insights      []ReportInsight  `json:"insights,omitempty"`
	GeneratedAt   time.Time        `json:"generated_at"`
}

// ReportPeriod represents the period for a report
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TrendData represents trend data for charts
type TrendData struct {
	Date        string  `json:"date"`
	Referrals   int     `json:"referrals"`
	Conversions int     `json:"conversions"`
	Value       float64 `json:"value"`
}

// ReportInsight represents an insight from the report
type ReportInsight struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // positive, negative, neutral
}
