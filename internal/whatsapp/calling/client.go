package calling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client handles WhatsApp Business Calling API interactions
type Client struct {
	httpClient    *http.Client
	accessToken   string
	phoneNumberID string
	apiVersion    string
	baseURL       string

	mu    sync.RWMutex
	calls map[string]*Call // In-memory storage, should be replaced with DB
}

// ClientConfig represents configuration for the calling client
type ClientConfig struct {
	AccessToken   string
	PhoneNumberID string
	APIVersion    string
}

// NewClient creates a new calling client
func NewClient(config *ClientConfig) *Client {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken:   config.AccessToken,
		phoneNumberID: config.PhoneNumberID,
		apiVersion:    apiVersion,
		baseURL:       "https://graph.facebook.com",
		calls:         make(map[string]*Call),
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
// Call Operations
// =============================================================================

// InitiateCall initiates an outbound call
func (c *Client) InitiateCall(ctx context.Context, req *InitiateCallRequest) (*InitiateCallResponse, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s/calls", c.phoneNumberID))

	// Build request body
	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                req.To,
		"type":              req.Type,
	}

	if req.Timeout > 0 {
		body["timeout"] = req.Timeout
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Calls []struct {
			ID string `json:"id"`
		} `json:"calls"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Calls) == 0 {
		return nil, fmt.Errorf("no call ID in response")
	}

	callID := result.Calls[0].ID
	now := time.Now()

	// Store call
	call := &Call{
		ID:            callID,
		PhoneNumberID: c.phoneNumberID,
		From:          c.phoneNumberID,
		To:            req.To,
		Direction:     CallDirectionOutbound,
		Type:          req.Type,
		Status:        CallStatusInitiated,
		StartedAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	c.mu.Lock()
	c.calls[callID] = call
	c.mu.Unlock()

	return &InitiateCallResponse{
		CallID:    callID,
		Status:    CallStatusInitiated,
		CreatedAt: now,
	}, nil
}

// EndCall ends an active call
func (c *Client) EndCall(ctx context.Context, callID string) error {
	apiURL := c.buildURL(fmt.Sprintf("/%s/calls/%s", c.phoneNumberID, callID))

	body := map[string]interface{}{
		"action": "end",
	}

	_, err := c.doRequest(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return err
	}

	// Update call status
	c.mu.Lock()
	if call, ok := c.calls[callID]; ok {
		now := time.Now()
		call.Status = CallStatusCompleted
		call.EndedAt = &now
		call.UpdatedAt = now
		if call.ConnectedAt != nil {
			call.Duration = int(now.Sub(*call.ConnectedAt).Seconds())
		}
	}
	c.mu.Unlock()

	return nil
}

// GetCall retrieves a call by ID
func (c *Client) GetCall(callID string) (*Call, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	call, ok := c.calls[callID]
	return call, ok
}

// GetCallsByPhone retrieves calls for a phone number
func (c *Client) GetCallsByPhone(phone string) []*Call {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*Call
	for _, call := range c.calls {
		if call.From == phone || call.To == phone {
			result = append(result, call)
		}
	}
	return result
}

// =============================================================================
// Webhook Handling
// =============================================================================

// ProcessWebhook processes a call webhook event
func (c *Client) ProcessWebhook(payload *CallWebhookPayload) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find or create call
	call, ok := c.calls[payload.CallID]
	if !ok {
		// Inbound call - create new record
		now := time.Now()
		call = &Call{
			ID:            payload.CallID,
			PhoneNumberID: c.phoneNumberID,
			From:          payload.From,
			To:            payload.To,
			Direction:     payload.Direction,
			Type:          CallTypeVoice, // Default to voice
			Status:        payload.Status,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		c.calls[payload.CallID] = call
	}

	// Update call status
	call.Status = payload.Status
	call.UpdatedAt = time.Now()

	now := time.Now()
	switch payload.Status {
	case CallStatusRinging:
		if call.StartedAt == nil {
			call.StartedAt = &now
		}
	case CallStatusConnected:
		call.ConnectedAt = &now
	case CallStatusCompleted:
		call.EndedAt = &now
		call.Duration = payload.Duration
	case CallStatusFailed:
		call.EndedAt = &now
		call.FailureReason = payload.FailureReason
	case CallStatusNoAnswer, CallStatusBusy, CallStatusRejected:
		call.EndedAt = &now
	}

	return nil
}

// =============================================================================
// Statistics
// =============================================================================

// GetCallStats returns call statistics
func (c *Client) GetCallStats() *CallStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &CallStats{
		ByStatus:    make(map[CallStatus]int),
		ByDirection: make(map[CallDirection]int),
	}

	var totalDuration int

	for _, call := range c.calls {
		stats.TotalCalls++
		stats.ByStatus[call.Status]++
		stats.ByDirection[call.Direction]++

		switch call.Direction {
		case CallDirectionInbound:
			stats.InboundCalls++
		case CallDirectionOutbound:
			stats.OutboundCalls++
		}

		switch call.Status {
		case CallStatusCompleted:
			stats.CompletedCalls++
			totalDuration += call.Duration
		case CallStatusNoAnswer, CallStatusBusy:
			stats.MissedCalls++
		case CallStatusFailed:
			stats.FailedCalls++
		}
	}

	stats.TotalDuration = totalDuration
	if stats.CompletedCalls > 0 {
		stats.AverageDuration = float64(totalDuration) / float64(stats.CompletedCalls)
	}

	return stats
}

// GetCallStatsByPeriod returns call statistics for a period
func (c *Client) GetCallStatsByPeriod(startDate, endDate time.Time) *CallStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &CallStats{
		ByStatus:    make(map[CallStatus]int),
		ByDirection: make(map[CallDirection]int),
		DailyStats:  make([]DailyCallStats, 0),
	}

	dailyData := make(map[string]*DailyCallStats)
	var totalDuration int

	for _, call := range c.calls {
		// Check if call is within period
		if call.CreatedAt.Before(startDate) || call.CreatedAt.After(endDate) {
			continue
		}

		stats.TotalCalls++
		stats.ByStatus[call.Status]++
		stats.ByDirection[call.Direction]++

		switch call.Direction {
		case CallDirectionInbound:
			stats.InboundCalls++
		case CallDirectionOutbound:
			stats.OutboundCalls++
		}

		switch call.Status {
		case CallStatusCompleted:
			stats.CompletedCalls++
			totalDuration += call.Duration
		case CallStatusNoAnswer, CallStatusBusy:
			stats.MissedCalls++
		case CallStatusFailed:
			stats.FailedCalls++
		}

		// Daily aggregation
		date := call.CreatedAt.Format("2006-01-02")
		if daily, ok := dailyData[date]; ok {
			daily.TotalCalls++
			if call.Direction == CallDirectionInbound {
				daily.InboundCalls++
			} else {
				daily.OutboundCalls++
			}
			if call.Status == CallStatusCompleted {
				daily.CompletedCalls++
				daily.TotalDuration += call.Duration
			}
			if call.Status == CallStatusNoAnswer || call.Status == CallStatusBusy {
				daily.MissedCalls++
			}
		} else {
			daily := &DailyCallStats{
				Date:       date,
				TotalCalls: 1,
			}
			if call.Direction == CallDirectionInbound {
				daily.InboundCalls = 1
			} else {
				daily.OutboundCalls = 1
			}
			if call.Status == CallStatusCompleted {
				daily.CompletedCalls = 1
				daily.TotalDuration = call.Duration
			}
			if call.Status == CallStatusNoAnswer || call.Status == CallStatusBusy {
				daily.MissedCalls = 1
			}
			dailyData[date] = daily
		}
	}

	stats.TotalDuration = totalDuration
	if stats.CompletedCalls > 0 {
		stats.AverageDuration = float64(totalDuration) / float64(stats.CompletedCalls)
	}

	// Convert daily data to slice
	for _, daily := range dailyData {
		if daily.CompletedCalls > 0 {
			daily.AverageDuration = float64(daily.TotalDuration) / float64(daily.CompletedCalls)
		}
		stats.DailyStats = append(stats.DailyStats, *daily)
	}

	return stats
}

// =============================================================================
// Call Quality
// =============================================================================

// GetCallQuality retrieves quality metrics for a call
func (c *Client) GetCallQuality(ctx context.Context, callID string) (*CallQuality, error) {
	c.mu.RLock()
	call, ok := c.calls[callID]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("call not found: %s", callID)
	}

	return &CallQuality{
		CallID:     callID,
		Score:      call.Metadata.QualityScore,
		PacketLoss: call.Metadata.PacketLoss,
		Jitter:     call.Metadata.Jitter,
		Latency:    call.Metadata.Latency,
		MeasuredAt: time.Now(),
	}, nil
}

// UpdateCallQuality updates quality metrics for a call
func (c *Client) UpdateCallQuality(callID string, quality *CallQuality) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	call, ok := c.calls[callID]
	if !ok {
		return fmt.Errorf("call not found: %s", callID)
	}

	call.Metadata.QualityScore = quality.Score
	call.Metadata.PacketLoss = quality.PacketLoss
	call.Metadata.Jitter = quality.Jitter
	call.Metadata.Latency = quality.Latency
	call.UpdatedAt = time.Now()

	return nil
}

// =============================================================================
// Call Recording
// =============================================================================

// GetCallRecording retrieves the recording URL for a call
func (c *Client) GetCallRecording(ctx context.Context, callID string) (string, error) {
	c.mu.RLock()
	call, ok := c.calls[callID]
	c.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("call not found: %s", callID)
	}

	if call.RecordingURL == "" {
		return "", fmt.Errorf("no recording available for call: %s", callID)
	}

	return call.RecordingURL, nil
}

// SetCallRecording sets the recording URL for a call
func (c *Client) SetCallRecording(callID string, recordingURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	call, ok := c.calls[callID]
	if !ok {
		return fmt.Errorf("call not found: %s", callID)
	}

	call.RecordingURL = recordingURL
	call.UpdatedAt = time.Now()

	return nil
}

// =============================================================================
// Recent Calls
// =============================================================================

// GetRecentCalls returns recent calls with pagination
func (c *Client) GetRecentCalls(limit, offset int) []*Call {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Convert map to slice
	calls := make([]*Call, 0, len(c.calls))
	for _, call := range c.calls {
		calls = append(calls, call)
	}

	// Sort by created_at descending (most recent first)
	for i := 0; i < len(calls)-1; i++ {
		for j := i + 1; j < len(calls); j++ {
			if calls[j].CreatedAt.After(calls[i].CreatedAt) {
				calls[i], calls[j] = calls[j], calls[i]
			}
		}
	}

	// Apply pagination
	if offset >= len(calls) {
		return []*Call{}
	}

	end := offset + limit
	if end > len(calls) {
		end = len(calls)
	}

	return calls[offset:end]
}
