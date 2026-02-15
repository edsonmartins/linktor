package analytics

import (
	"time"
)

// =============================================================================
// Conversation Analytics Types
// =============================================================================

// ConversationAnalytics represents conversation analytics data
type ConversationAnalytics struct {
	ID                 string                    `json:"id"`
	OrganizationID     string                    `json:"organization_id"`
	PhoneNumberID      string                    `json:"phone_number_id"`
	Period             AnalyticsPeriod           `json:"period"`
	TotalConversations int                       `json:"total_conversations"`
	TotalCost          float64                   `json:"total_cost"`
	Currency           string                    `json:"currency"`
	ByType             map[ConversationType]int  `json:"by_type"`
	ByDirection        map[Direction]int         `json:"by_direction"`
	ByCountry          map[string]int            `json:"by_country"`
	ByCategory         map[ConversationCategory]ConversationCategoryCost `json:"by_category"`
	Timeline           []DailyAnalytics          `json:"timeline"`
	FetchedAt          time.Time                 `json:"fetched_at"`
}

// AnalyticsPeriod represents a time period for analytics
type AnalyticsPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ConversationType represents the type of conversation
type ConversationType string

const (
	ConversationTypeRegular       ConversationType = "REGULAR"
	ConversationTypeFreeEntry     ConversationType = "FREE_ENTRY"
	ConversationTypeFreeEntryTier ConversationType = "FREE_TIER"
)

// Direction represents message direction
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// ConversationCategory represents pricing categories
type ConversationCategory string

const (
	CategoryAuthentication ConversationCategory = "AUTHENTICATION"
	CategoryMarketing      ConversationCategory = "MARKETING"
	CategoryUtility        ConversationCategory = "UTILITY"
	CategoryService        ConversationCategory = "SERVICE"
	CategoryReferralConversion ConversationCategory = "REFERRAL_CONVERSION"
)

// ConversationCategoryCost represents cost breakdown by category
type ConversationCategoryCost struct {
	Count int     `json:"count"`
	Cost  float64 `json:"cost"`
}

// DailyAnalytics represents daily conversation data
type DailyAnalytics struct {
	Date          string  `json:"date"` // YYYY-MM-DD format
	Conversations int     `json:"conversations"`
	Cost          float64 `json:"cost"`
	ByCategory    map[ConversationCategory]int `json:"by_category,omitempty"`
}

// =============================================================================
// Message Analytics Types
// =============================================================================

// MessageAnalytics represents message-level analytics
type MessageAnalytics struct {
	TotalSent      int                      `json:"total_sent"`
	TotalDelivered int                      `json:"total_delivered"`
	TotalRead      int                      `json:"total_read"`
	TotalFailed    int                      `json:"total_failed"`
	DeliveryRate   float64                  `json:"delivery_rate"`
	ReadRate       float64                  `json:"read_rate"`
	ByType         map[string]MessageStats  `json:"by_type"`
	Timeline       []DailyMessageStats      `json:"timeline"`
}

// MessageStats represents message statistics
type MessageStats struct {
	Sent      int     `json:"sent"`
	Delivered int     `json:"delivered"`
	Read      int     `json:"read"`
	Failed    int     `json:"failed"`
}

// DailyMessageStats represents daily message statistics
type DailyMessageStats struct {
	Date      string `json:"date"`
	Sent      int    `json:"sent"`
	Delivered int    `json:"delivered"`
	Read      int    `json:"read"`
	Failed    int    `json:"failed"`
}

// =============================================================================
// Template Analytics Types
// =============================================================================

// TemplateAnalytics represents template performance analytics
type TemplateAnalytics struct {
	TemplateID   string                 `json:"template_id"`
	TemplateName string                 `json:"template_name"`
	Category     string                 `json:"category"`
	Language     string                 `json:"language"`
	Stats        TemplateStats          `json:"stats"`
	DailyStats   []DailyTemplateStats   `json:"daily_stats"`
	QualityScore *TemplateQualityScore  `json:"quality_score,omitempty"`
}

// TemplateStats represents overall template statistics
type TemplateStats struct {
	Sent           int     `json:"sent"`
	Delivered      int     `json:"delivered"`
	Read           int     `json:"read"`
	Clicked        int     `json:"clicked"`
	Replied        int     `json:"replied"`
	DeliveryRate   float64 `json:"delivery_rate"`
	ReadRate       float64 `json:"read_rate"`
	ClickRate      float64 `json:"click_rate"`
	ResponseRate   float64 `json:"response_rate"`
}

// DailyTemplateStats represents daily template statistics
type DailyTemplateStats struct {
	Date      string `json:"date"`
	Sent      int    `json:"sent"`
	Delivered int    `json:"delivered"`
	Read      int    `json:"read"`
	Clicked   int    `json:"clicked"`
}

// TemplateQualityScore represents template quality information
type TemplateQualityScore struct {
	Score   string `json:"score"` // GREEN, YELLOW, RED, UNKNOWN
	Reasons []string `json:"reasons,omitempty"`
}

// =============================================================================
// Phone Number Analytics Types
// =============================================================================

// PhoneNumberAnalytics represents phone number quality analytics
type PhoneNumberAnalytics struct {
	PhoneNumberID    string              `json:"phone_number_id"`
	DisplayNumber    string              `json:"display_phone_number"`
	QualityRating    string              `json:"quality_rating"` // GREEN, YELLOW, RED
	MessagingLimit   string              `json:"messaging_limit"` // TIER_1K, TIER_10K, TIER_100K, TIER_UNLIMITED
	CurrentThroughput int                `json:"current_throughput"`
	Status           string              `json:"status"`
	NameStatus       string              `json:"name_status"`
	NewNameStatus    string              `json:"new_name_status,omitempty"`
}

// =============================================================================
// API Request/Response Types
// =============================================================================

// AnalyticsRequest represents a request for analytics data
type AnalyticsRequest struct {
	PhoneNumberID string    `json:"phone_number_id"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	Granularity   string    `json:"granularity"` // HALF_HOUR, DAILY, MONTHLY
}

// ConversationAnalyticsResponse represents the API response
type ConversationAnalyticsResponse struct {
	Data   []ConversationDataPoint `json:"data"`
	Paging *Paging                 `json:"paging,omitempty"`
}

// ConversationDataPoint represents a single data point
type ConversationDataPoint struct {
	DataPointID     string `json:"data_point_id"`
	Start           int64  `json:"start"` // Unix timestamp
	End             int64  `json:"end"`   // Unix timestamp
	Sent            int    `json:"sent"`
	Delivered       int    `json:"delivered"`
	ConversationType string `json:"conversation_type"`
	ConversationDirection string `json:"conversation_direction"`
	ConversationCategory string `json:"conversation_category,omitempty"`
	Cost            float64 `json:"cost"`
	Country         string  `json:"country,omitempty"`
}

// Paging represents pagination info
type Paging struct {
	Cursors struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"cursors"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

// =============================================================================
// Export Types
// =============================================================================

// ExportFormat represents the export format
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
	ExportFormatXLSX ExportFormat = "xlsx"
)

// ExportRequest represents an export request
type ExportRequest struct {
	Format      ExportFormat  `json:"format"`
	StartDate   time.Time     `json:"start_date"`
	EndDate     time.Time     `json:"end_date"`
	IncludeRaw  bool          `json:"include_raw"`
}

// =============================================================================
// Aggregated Stats Types
// =============================================================================

// AggregatedStats represents aggregated analytics
type AggregatedStats struct {
	Period              AnalyticsPeriod `json:"period"`
	TotalConversations  int             `json:"total_conversations"`
	TotalMessages       int             `json:"total_messages"`
	TotalCost           float64         `json:"total_cost"`
	Currency            string          `json:"currency"`
	AverageDailyCost    float64         `json:"average_daily_cost"`
	AverageResponseTime float64         `json:"average_response_time_seconds"`
	TopCountries        []CountryStat   `json:"top_countries"`
	TopTemplates        []TemplateStat  `json:"top_templates"`
	Trends              TrendData       `json:"trends"`
}

// CountryStat represents country-level statistics
type CountryStat struct {
	Country       string  `json:"country"`
	CountryCode   string  `json:"country_code"`
	Conversations int     `json:"conversations"`
	Cost          float64 `json:"cost"`
	Percentage    float64 `json:"percentage"`
}

// TemplateStat represents template-level statistics
type TemplateStat struct {
	TemplateID   string  `json:"template_id"`
	TemplateName string  `json:"template_name"`
	Sent         int     `json:"sent"`
	DeliveryRate float64 `json:"delivery_rate"`
	ReadRate     float64 `json:"read_rate"`
}

// TrendData represents trend analysis
type TrendData struct {
	ConversationTrend float64 `json:"conversation_trend"` // Percentage change
	CostTrend         float64 `json:"cost_trend"`
	MessageTrend      float64 `json:"message_trend"`
	Period            string  `json:"comparison_period"` // e.g., "vs_last_week"
}
