package calling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ClientTestSuite tests the calling client
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
		PhoneNumberID: "test-phone",
		APIVersion:    "v21.0",
	}
	suite.client = NewClient(config)
}

// NewClient tests
func (suite *ClientTestSuite) TestNewClient_WithValidConfig() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
	}

	client := NewClient(config)

	assert.NotNil(suite.T(), client)
	assert.NotNil(suite.T(), client.httpClient)
	assert.NotNil(suite.T(), client.calls)
}

func (suite *ClientTestSuite) TestNewClient_DefaultsAPIVersion() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
		APIVersion:    "",
	}

	client := NewClient(config)

	assert.Equal(suite.T(), "v21.0", client.apiVersion)
}

// Call storage tests
func (suite *ClientTestSuite) TestGetCall_NotFound() {
	call, found := suite.client.GetCall("non-existent")

	assert.False(suite.T(), found)
	assert.Nil(suite.T(), call)
}

func (suite *ClientTestSuite) TestGetCallsByPhone() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["c1"] = &Call{
		ID:        "c1",
		From:      "5511999999999",
		To:        "5511888888888",
		CreatedAt: now,
	}
	suite.client.calls["c2"] = &Call{
		ID:        "c2",
		From:      "5511777777777",
		To:        "5511999999999",
		CreatedAt: now,
	}
	suite.client.calls["c3"] = &Call{
		ID:        "c3",
		From:      "5511666666666",
		To:        "5511555555555",
		CreatedAt: now,
	}
	suite.client.mu.Unlock()

	calls := suite.client.GetCallsByPhone("5511999999999")

	assert.Len(suite.T(), calls, 2)
}

// ProcessWebhook tests
func (suite *ClientTestSuite) TestProcessWebhook_NewInboundCall() {
	payload := &CallWebhookPayload{
		CallID:    "new-call",
		From:      "5511999999999",
		To:        "5511888888888",
		Direction: CallDirectionInbound,
		Status:    CallStatusRinging,
	}

	err := suite.client.ProcessWebhook(payload)

	assert.NoError(suite.T(), err)

	call, found := suite.client.GetCall("new-call")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), CallStatusRinging, call.Status)
	assert.Equal(suite.T(), CallDirectionInbound, call.Direction)
}

func (suite *ClientTestSuite) TestProcessWebhook_UpdateExistingCall() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["existing-call"] = &Call{
		ID:        "existing-call",
		Status:    CallStatusRinging,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	suite.client.mu.Unlock()

	payload := &CallWebhookPayload{
		CallID: "existing-call",
		Status: CallStatusConnected,
	}

	err := suite.client.ProcessWebhook(payload)

	assert.NoError(suite.T(), err)

	call, found := suite.client.GetCall("existing-call")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), CallStatusConnected, call.Status)
	assert.NotNil(suite.T(), call.ConnectedAt)
}

func (suite *ClientTestSuite) TestProcessWebhook_CallCompleted() {
	now := time.Now()
	connectedAt := now.Add(-5 * time.Minute)
	suite.client.mu.Lock()
	suite.client.calls["completed-call"] = &Call{
		ID:          "completed-call",
		Status:      CallStatusConnected,
		StartedAt:   &now,
		ConnectedAt: &connectedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	suite.client.mu.Unlock()

	payload := &CallWebhookPayload{
		CallID:   "completed-call",
		Status:   CallStatusCompleted,
		Duration: 300, // 5 minutes
	}

	err := suite.client.ProcessWebhook(payload)

	assert.NoError(suite.T(), err)

	call, found := suite.client.GetCall("completed-call")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), CallStatusCompleted, call.Status)
	assert.NotNil(suite.T(), call.EndedAt)
	assert.Equal(suite.T(), 300, call.Duration)
}

func (suite *ClientTestSuite) TestProcessWebhook_CallFailed() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["failed-call"] = &Call{
		ID:        "failed-call",
		Status:    CallStatusRinging,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	suite.client.mu.Unlock()

	payload := &CallWebhookPayload{
		CallID:        "failed-call",
		Status:        CallStatusFailed,
		FailureReason: "Network error",
	}

	err := suite.client.ProcessWebhook(payload)

	assert.NoError(suite.T(), err)

	call, found := suite.client.GetCall("failed-call")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), CallStatusFailed, call.Status)
	assert.Equal(suite.T(), "Network error", call.FailureReason)
}

// Statistics tests
func (suite *ClientTestSuite) TestGetCallStats_Empty() {
	stats := suite.client.GetCallStats()

	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), 0, stats.TotalCalls)
}

func (suite *ClientTestSuite) TestGetCallStats_WithCalls() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["c1"] = &Call{
		ID:        "c1",
		Status:    CallStatusCompleted,
		Direction: CallDirectionInbound,
		Duration:  120,
		CreatedAt: now,
	}
	suite.client.calls["c2"] = &Call{
		ID:        "c2",
		Status:    CallStatusCompleted,
		Direction: CallDirectionOutbound,
		Duration:  180,
		CreatedAt: now,
	}
	suite.client.calls["c3"] = &Call{
		ID:        "c3",
		Status:    CallStatusNoAnswer,
		Direction: CallDirectionOutbound,
		CreatedAt: now,
	}
	suite.client.mu.Unlock()

	stats := suite.client.GetCallStats()

	assert.Equal(suite.T(), 3, stats.TotalCalls)
	assert.Equal(suite.T(), 1, stats.InboundCalls)
	assert.Equal(suite.T(), 2, stats.OutboundCalls)
	assert.Equal(suite.T(), 2, stats.CompletedCalls)
	assert.Equal(suite.T(), 1, stats.MissedCalls)
	assert.Equal(suite.T(), 300, stats.TotalDuration)
	assert.Equal(suite.T(), 150.0, stats.AverageDuration)
}

func (suite *ClientTestSuite) TestGetCallStatsByPeriod() {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -7)

	suite.client.mu.Lock()
	suite.client.calls["c1"] = &Call{
		ID:        "c1",
		Status:    CallStatusCompleted,
		Direction: CallDirectionInbound,
		Duration:  120,
		CreatedAt: now,
	}
	suite.client.calls["c2"] = &Call{
		ID:        "c2",
		Status:    CallStatusCompleted,
		Direction: CallDirectionOutbound,
		Duration:  180,
		CreatedAt: lastWeek.AddDate(0, 0, -1), // Before period
	}
	suite.client.mu.Unlock()

	stats := suite.client.GetCallStatsByPeriod(yesterday, now.Add(time.Hour))

	assert.Equal(suite.T(), 1, stats.TotalCalls)
	assert.Equal(suite.T(), 1, stats.CompletedCalls)
}

// Call quality tests
func (suite *ClientTestSuite) TestGetCallQuality_NotFound() {
	_, err := suite.client.GetCallQuality(nil, "non-existent")

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "call not found")
}

func (suite *ClientTestSuite) TestUpdateCallQuality() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["quality-call"] = &Call{
		ID:        "quality-call",
		CreatedAt: now,
		UpdatedAt: now,
	}
	suite.client.mu.Unlock()

	quality := &CallQuality{
		Score:      4,
		PacketLoss: 0.5,
		Jitter:     10.0,
		Latency:    50,
	}

	err := suite.client.UpdateCallQuality("quality-call", quality)

	assert.NoError(suite.T(), err)

	call, _ := suite.client.GetCall("quality-call")
	assert.Equal(suite.T(), 4, call.Metadata.QualityScore)
	assert.Equal(suite.T(), 0.5, call.Metadata.PacketLoss)
}

func (suite *ClientTestSuite) TestUpdateCallQuality_NotFound() {
	quality := &CallQuality{Score: 4}

	err := suite.client.UpdateCallQuality("non-existent", quality)

	assert.Error(suite.T(), err)
}

// Recording tests
func (suite *ClientTestSuite) TestGetCallRecording_NotFound() {
	_, err := suite.client.GetCallRecording(nil, "non-existent")

	assert.Error(suite.T(), err)
}

func (suite *ClientTestSuite) TestGetCallRecording_NoRecording() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["no-recording"] = &Call{
		ID:        "no-recording",
		CreatedAt: now,
	}
	suite.client.mu.Unlock()

	_, err := suite.client.GetCallRecording(nil, "no-recording")

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "no recording")
}

func (suite *ClientTestSuite) TestSetCallRecording() {
	now := time.Now()
	suite.client.mu.Lock()
	suite.client.calls["recording-call"] = &Call{
		ID:        "recording-call",
		CreatedAt: now,
		UpdatedAt: now,
	}
	suite.client.mu.Unlock()

	err := suite.client.SetCallRecording("recording-call", "https://example.com/recording.mp3")

	assert.NoError(suite.T(), err)

	call, _ := suite.client.GetCall("recording-call")
	assert.Equal(suite.T(), "https://example.com/recording.mp3", call.RecordingURL)
}

// Recent calls tests
func (suite *ClientTestSuite) TestGetRecentCalls_Empty() {
	calls := suite.client.GetRecentCalls(10, 0)

	assert.Empty(suite.T(), calls)
}

func (suite *ClientTestSuite) TestGetRecentCalls_WithPagination() {
	now := time.Now()
	for i := 0; i < 15; i++ {
		suite.client.mu.Lock()
		suite.client.calls[string(rune('a'+i))] = &Call{
			ID:        string(rune('a' + i)),
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		suite.client.mu.Unlock()
	}

	// Get first page
	calls := suite.client.GetRecentCalls(10, 0)
	assert.Len(suite.T(), calls, 10)

	// Get second page
	calls = suite.client.GetRecentCalls(10, 10)
	assert.Len(suite.T(), calls, 5)
}

// DefaultQualityThresholds test
func TestDefaultQualityThresholds(t *testing.T) {
	thresholds := DefaultQualityThresholds()

	assert.NotNil(t, thresholds)
	assert.Equal(t, 3.0, thresholds.MaxPacketLoss)
	assert.Equal(t, 30.0, thresholds.MaxJitter)
	assert.Equal(t, 150, thresholds.MaxLatency)
	assert.Equal(t, 64, thresholds.MinBitrate)
}

// NewCallEvent test
func TestNewCallEvent(t *testing.T) {
	call := &Call{
		ID:     "test-call",
		Status: CallStatusCompleted,
	}

	event := NewCallEvent(CallEventCompleted, call)

	assert.Equal(t, CallEventCompleted, event.Type)
	assert.Equal(t, "test-call", event.CallID)
	assert.Equal(t, "completed", event.Status)
	assert.NotNil(t, event.Data)
}
