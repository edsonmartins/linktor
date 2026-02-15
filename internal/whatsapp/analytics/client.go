package analytics

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Client handles WhatsApp Analytics API interactions
type Client struct {
	httpClient    *http.Client
	accessToken   string
	businessID    string
	phoneNumberID string
	apiVersion    string
	baseURL       string

	mu    sync.RWMutex
	cache map[string]*cachedAnalytics
}

// cachedAnalytics represents cached analytics data
type cachedAnalytics struct {
	data      *ConversationAnalytics
	expiresAt time.Time
}

// ClientConfig represents configuration for the analytics client
type ClientConfig struct {
	AccessToken   string
	BusinessID    string
	PhoneNumberID string
	APIVersion    string
}

// NewClient creates a new analytics client
func NewClient(config *ClientConfig) *Client {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		accessToken:   config.AccessToken,
		businessID:    config.BusinessID,
		phoneNumberID: config.PhoneNumberID,
		apiVersion:    apiVersion,
		baseURL:       "https://graph.facebook.com",
		cache:         make(map[string]*cachedAnalytics),
	}
}

// buildURL builds the API URL
func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", c.baseURL, c.apiVersion, path)
}

// doRequest executes an HTTP request
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
	}

	return respBody, nil
}

// =============================================================================
// Conversation Analytics
// =============================================================================

// GetConversationAnalytics retrieves conversation analytics for a period
func (c *Client) GetConversationAnalytics(ctx context.Context, req *AnalyticsRequest) (*ConversationAnalytics, error) {
	// Check cache first
	cacheKey := c.getCacheKey(req)
	if cached := c.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	// Build API URL
	apiURL := c.buildURL(fmt.Sprintf("/%s", req.PhoneNumberID))

	params := url.Values{}
	params.Set("fields", "conversation_analytics")
	params.Set("conversation_analytics.start", fmt.Sprintf("%d", req.StartDate.Unix()))
	params.Set("conversation_analytics.end", fmt.Sprintf("%d", req.EndDate.Unix()))
	params.Set("conversation_analytics.granularity", req.Granularity)
	params.Set("conversation_analytics.dimensions", "[\"CONVERSATION_TYPE\",\"CONVERSATION_DIRECTION\",\"COUNTRY\",\"CONVERSATION_CATEGORY\"]")
	apiURL += "?" + params.Encode()

	respBody, err := c.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result struct {
		ConversationAnalytics struct {
			Data []ConversationDataPoint `json:"data"`
		} `json:"conversation_analytics"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Aggregate the data
	analytics := c.aggregateConversationData(req, result.ConversationAnalytics.Data)

	// Cache the result
	c.setInCache(cacheKey, analytics, 15*time.Minute)

	return analytics, nil
}

// aggregateConversationData aggregates raw data points into analytics
func (c *Client) aggregateConversationData(req *AnalyticsRequest, dataPoints []ConversationDataPoint) *ConversationAnalytics {
	analytics := &ConversationAnalytics{
		PhoneNumberID: req.PhoneNumberID,
		Period: AnalyticsPeriod{
			Start: req.StartDate,
			End:   req.EndDate,
		},
		Currency:    "USD", // Default, should be from config
		ByType:      make(map[ConversationType]int),
		ByDirection: make(map[Direction]int),
		ByCountry:   make(map[string]int),
		ByCategory:  make(map[ConversationCategory]ConversationCategoryCost),
		FetchedAt:   time.Now(),
	}

	// Track daily data
	dailyData := make(map[string]*DailyAnalytics)

	for _, dp := range dataPoints {
		analytics.TotalConversations += dp.Sent
		analytics.TotalCost += dp.Cost

		// By type
		convType := ConversationType(dp.ConversationType)
		analytics.ByType[convType] += dp.Sent

		// By direction
		dir := Direction(dp.ConversationDirection)
		analytics.ByDirection[dir] += dp.Sent

		// By country
		if dp.Country != "" {
			analytics.ByCountry[dp.Country] += dp.Sent
		}

		// By category
		if dp.ConversationCategory != "" {
			category := ConversationCategory(dp.ConversationCategory)
			existing := analytics.ByCategory[category]
			existing.Count += dp.Sent
			existing.Cost += dp.Cost
			analytics.ByCategory[category] = existing
		}

		// Daily aggregation
		date := time.Unix(dp.Start, 0).Format("2006-01-02")
		if daily, ok := dailyData[date]; ok {
			daily.Conversations += dp.Sent
			daily.Cost += dp.Cost
		} else {
			dailyData[date] = &DailyAnalytics{
				Date:          date,
				Conversations: dp.Sent,
				Cost:          dp.Cost,
				ByCategory:    make(map[ConversationCategory]int),
			}
		}
		if dp.ConversationCategory != "" {
			dailyData[date].ByCategory[ConversationCategory(dp.ConversationCategory)] += dp.Sent
		}
	}

	// Convert daily data to timeline
	for _, daily := range dailyData {
		analytics.Timeline = append(analytics.Timeline, *daily)
	}

	// Sort timeline by date
	sort.Slice(analytics.Timeline, func(i, j int) bool {
		return analytics.Timeline[i].Date < analytics.Timeline[j].Date
	})

	return analytics
}

// =============================================================================
// Phone Number Analytics
// =============================================================================

// GetPhoneNumberAnalytics retrieves analytics for a phone number
func (c *Client) GetPhoneNumberAnalytics(ctx context.Context, phoneNumberID string) (*PhoneNumberAnalytics, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s", phoneNumberID))
	params := url.Values{}
	params.Set("fields", "display_phone_number,quality_rating,messaging_limit_tier,throughput,status,name_status,new_name_status")
	apiURL += "?" + params.Encode()

	respBody, err := c.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID                 string `json:"id"`
		DisplayPhoneNumber string `json:"display_phone_number"`
		QualityRating      string `json:"quality_rating"`
		MessagingLimitTier string `json:"messaging_limit_tier"`
		Throughput         struct {
			Level int `json:"level"`
		} `json:"throughput"`
		Status        string `json:"status"`
		NameStatus    string `json:"name_status"`
		NewNameStatus string `json:"new_name_status"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &PhoneNumberAnalytics{
		PhoneNumberID:     result.ID,
		DisplayNumber:     result.DisplayPhoneNumber,
		QualityRating:     result.QualityRating,
		MessagingLimit:    result.MessagingLimitTier,
		CurrentThroughput: result.Throughput.Level,
		Status:            result.Status,
		NameStatus:        result.NameStatus,
		NewNameStatus:     result.NewNameStatus,
	}, nil
}

// =============================================================================
// Template Analytics
// =============================================================================

// GetTemplateAnalytics retrieves analytics for a template
func (c *Client) GetTemplateAnalytics(ctx context.Context, templateID string, startDate, endDate time.Time) (*TemplateAnalytics, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s", templateID))
	params := url.Values{}
	params.Set("fields", "name,category,language,message_template_status_update,daily_stats")
	params.Set("daily_stats.start", fmt.Sprintf("%d", startDate.Unix()))
	params.Set("daily_stats.end", fmt.Sprintf("%d", endDate.Unix()))
	apiURL += "?" + params.Encode()

	respBody, err := c.doRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
		Language string `json:"language"`
		DailyStats []struct {
			Date      string `json:"date"`
			Sent      int    `json:"sent"`
			Delivered int    `json:"delivered"`
			Read      int    `json:"read"`
			Clicked   int    `json:"clicked"`
		} `json:"daily_stats"`
		QualityScore struct {
			Score   string   `json:"score"`
			Reasons []string `json:"reasons"`
		} `json:"quality_score"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	analytics := &TemplateAnalytics{
		TemplateID:   result.ID,
		TemplateName: result.Name,
		Category:     result.Category,
		Language:     result.Language,
		DailyStats:   make([]DailyTemplateStats, 0, len(result.DailyStats)),
	}

	// Calculate totals
	var totalSent, totalDelivered, totalRead, totalClicked int
	for _, daily := range result.DailyStats {
		totalSent += daily.Sent
		totalDelivered += daily.Delivered
		totalRead += daily.Read
		totalClicked += daily.Clicked

		analytics.DailyStats = append(analytics.DailyStats, DailyTemplateStats{
			Date:      daily.Date,
			Sent:      daily.Sent,
			Delivered: daily.Delivered,
			Read:      daily.Read,
			Clicked:   daily.Clicked,
		})
	}

	analytics.Stats = TemplateStats{
		Sent:      totalSent,
		Delivered: totalDelivered,
		Read:      totalRead,
		Clicked:   totalClicked,
	}

	// Calculate rates
	if totalSent > 0 {
		analytics.Stats.DeliveryRate = float64(totalDelivered) / float64(totalSent) * 100
		analytics.Stats.ReadRate = float64(totalRead) / float64(totalSent) * 100
		analytics.Stats.ClickRate = float64(totalClicked) / float64(totalSent) * 100
	}

	// Quality score
	if result.QualityScore.Score != "" {
		analytics.QualityScore = &TemplateQualityScore{
			Score:   result.QualityScore.Score,
			Reasons: result.QualityScore.Reasons,
		}
	}

	return analytics, nil
}

// =============================================================================
// Aggregated Statistics
// =============================================================================

// GetAggregatedStats retrieves aggregated statistics
func (c *Client) GetAggregatedStats(ctx context.Context, req *AnalyticsRequest) (*AggregatedStats, error) {
	// Get conversation analytics
	convAnalytics, err := c.GetConversationAnalytics(ctx, req)
	if err != nil {
		return nil, err
	}

	stats := &AggregatedStats{
		Period:             convAnalytics.Period,
		TotalConversations: convAnalytics.TotalConversations,
		TotalCost:          convAnalytics.TotalCost,
		Currency:           convAnalytics.Currency,
		TopCountries:       make([]CountryStat, 0),
	}

	// Calculate average daily cost
	days := convAnalytics.Period.End.Sub(convAnalytics.Period.Start).Hours() / 24
	if days > 0 {
		stats.AverageDailyCost = convAnalytics.TotalCost / days
	}

	// Top countries
	type countryCount struct {
		country string
		count   int
	}
	countries := make([]countryCount, 0, len(convAnalytics.ByCountry))
	for country, count := range convAnalytics.ByCountry {
		countries = append(countries, countryCount{country, count})
	}
	sort.Slice(countries, func(i, j int) bool {
		return countries[i].count > countries[j].count
	})

	// Take top 10
	limit := 10
	if len(countries) < limit {
		limit = len(countries)
	}
	for i := 0; i < limit; i++ {
		var percentage float64
		if convAnalytics.TotalConversations > 0 {
			percentage = float64(countries[i].count) / float64(convAnalytics.TotalConversations) * 100
		}
		stats.TopCountries = append(stats.TopCountries, CountryStat{
			Country:       countries[i].country,
			CountryCode:   countries[i].country,
			Conversations: countries[i].count,
			Percentage:    percentage,
		})
	}

	return stats, nil
}

// =============================================================================
// Export Functions
// =============================================================================

// ExportToCSV exports analytics to CSV format
func (c *Client) ExportToCSV(analytics *ConversationAnalytics) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header
	header := []string{"Date", "Conversations", "Cost", "Authentication", "Marketing", "Utility", "Service"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Data rows
	for _, daily := range analytics.Timeline {
		row := []string{
			daily.Date,
			fmt.Sprintf("%d", daily.Conversations),
			fmt.Sprintf("%.4f", daily.Cost),
			fmt.Sprintf("%d", daily.ByCategory[CategoryAuthentication]),
			fmt.Sprintf("%d", daily.ByCategory[CategoryMarketing]),
			fmt.Sprintf("%d", daily.ByCategory[CategoryUtility]),
			fmt.Sprintf("%d", daily.ByCategory[CategoryService]),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ExportToJSON exports analytics to JSON format
func (c *Client) ExportToJSON(analytics *ConversationAnalytics) ([]byte, error) {
	return json.MarshalIndent(analytics, "", "  ")
}

// =============================================================================
// Cache Management
// =============================================================================

func (c *Client) getCacheKey(req *AnalyticsRequest) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		req.PhoneNumberID,
		req.StartDate.Format("2006-01-02"),
		req.EndDate.Format("2006-01-02"),
		req.Granularity,
	)
}

func (c *Client) getFromCache(key string) *ConversationAnalytics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.cache[key]; ok {
		if time.Now().Before(cached.expiresAt) {
			return cached.data
		}
	}
	return nil
}

func (c *Client) setInCache(key string, data *ConversationAnalytics, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cachedAnalytics{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

// ClearCache clears the analytics cache
func (c *Client) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cachedAnalytics)
}

// CleanupExpiredCache removes expired cache entries
func (c *Client) CleanupExpiredCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cached := range c.cache {
		if now.After(cached.expiresAt) {
			delete(c.cache, key)
		}
	}
}
