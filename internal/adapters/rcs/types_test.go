package rcs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== Config.Validate() ==========

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		Provider: ProviderZenvia,
		AgentID:  "agent-123",
		APIKey:   "key-abc",
	}
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_EmptyProvider(t *testing.T) {
	cfg := &Config{
		Provider: "",
		AgentID:  "agent-123",
		APIKey:   "key-abc",
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is required")
}

func TestConfig_Validate_EmptyAgentID(t *testing.T) {
	cfg := &Config{
		Provider: ProviderZenvia,
		AgentID:  "",
		APIKey:   "key-abc",
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent_id is required")
}

func TestConfig_Validate_EmptyAPIKey(t *testing.T) {
	cfg := &Config{
		Provider: ProviderZenvia,
		AgentID:  "agent-123",
		APIKey:   "",
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

// ========== Config.GetBaseURL() ==========

func TestConfig_GetBaseURL_Zenvia(t *testing.T) {
	cfg := &Config{Provider: ProviderZenvia}
	assert.Equal(t, "https://api.zenvia.com/v2", cfg.GetBaseURL())
}

func TestConfig_GetBaseURL_Infobip(t *testing.T) {
	cfg := &Config{Provider: ProviderInfobip}
	assert.Equal(t, "https://api.infobip.com", cfg.GetBaseURL())
}

func TestConfig_GetBaseURL_Pontaltech(t *testing.T) {
	cfg := &Config{Provider: ProviderPontaltech}
	assert.Equal(t, "https://api.pontaltech.com.br/v1", cfg.GetBaseURL())
}

func TestConfig_GetBaseURL_Google(t *testing.T) {
	cfg := &Config{Provider: ProviderGoogle}
	assert.Equal(t, "https://rcsbusinessmessaging.googleapis.com/v1", cfg.GetBaseURL())
}

func TestConfig_GetBaseURL_CustomBaseURL(t *testing.T) {
	cfg := &Config{
		Provider: ProviderZenvia,
		BaseURL:  "https://custom.api.example.com",
	}
	assert.Equal(t, "https://custom.api.example.com", cfg.GetBaseURL())
}

func TestConfig_GetBaseURL_UnknownProvider(t *testing.T) {
	cfg := &Config{Provider: Provider("unknown")}
	assert.Equal(t, "", cfg.GetBaseURL())
}

// ========== Provider Constants ==========

func TestProviderConstants(t *testing.T) {
	assert.Equal(t, Provider("zenvia"), ProviderZenvia)
	assert.Equal(t, Provider("infobip"), ProviderInfobip)
	assert.Equal(t, Provider("pontaltech"), ProviderPontaltech)
	assert.Equal(t, Provider("google"), ProviderGoogle)
}

// ========== SuggestionType Constants ==========

func TestSuggestionTypeConstants(t *testing.T) {
	assert.Equal(t, SuggestionType("reply"), SuggestionTypeReply)
	assert.Equal(t, SuggestionType("action"), SuggestionTypeAction)
	assert.Equal(t, SuggestionType("dial"), SuggestionTypeDial)
	assert.Equal(t, SuggestionType("url"), SuggestionTypeURL)
	assert.Equal(t, SuggestionType("location"), SuggestionTypeLocation)
	assert.Equal(t, SuggestionType("calendar"), SuggestionTypeCalendar)
	assert.Equal(t, SuggestionType("share"), SuggestionTypeShare)
}

// ========== DeliveryStatus Constants ==========

func TestDeliveryStatusConstants(t *testing.T) {
	assert.Equal(t, DeliveryStatus("pending"), StatusPending)
	assert.Equal(t, DeliveryStatus("sent"), StatusSent)
	assert.Equal(t, DeliveryStatus("delivered"), StatusDelivered)
	assert.Equal(t, DeliveryStatus("read"), StatusRead)
	assert.Equal(t, DeliveryStatus("failed"), StatusFailed)
}

// ========== Error Variables ==========

func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrInvalidProvider)
	assert.NotNil(t, ErrMissingCredentials)
	assert.NotNil(t, ErrProviderUnavailable)
	assert.NotNil(t, ErrMessageTooLong)
	assert.NotNil(t, ErrUnsupportedMedia)

	assert.Equal(t, "invalid RCS provider", ErrInvalidProvider.Error())
	assert.Equal(t, "missing RCS credentials", ErrMissingCredentials.Error())
	assert.Equal(t, "RCS provider unavailable", ErrProviderUnavailable.Error())
	assert.Equal(t, "message exceeds maximum length", ErrMessageTooLong.Error())
	assert.Equal(t, "unsupported media type", ErrUnsupportedMedia.Error())
}

// ========== JSON Round-Trip Tests ==========

func TestOutboundMessage_JSONRoundTrip(t *testing.T) {
	msg := &OutboundMessage{
		To:        "+5511999999999",
		Text:      "Hello from RCS",
		MediaURL:  "https://example.com/image.jpg",
		MediaType: "image/jpeg",
		Metadata:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded OutboundMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.To, decoded.To)
	assert.Equal(t, msg.Text, decoded.Text)
	assert.Equal(t, msg.MediaURL, decoded.MediaURL)
	assert.Equal(t, msg.MediaType, decoded.MediaType)
	assert.Equal(t, msg.Metadata["key"], decoded.Metadata["key"])
}

func TestRichCard_JSONRoundTrip(t *testing.T) {
	card := &RichCard{
		Title:       "Product",
		Description: "A great product",
		MediaURL:    "https://example.com/product.jpg",
		MediaType:   "image/jpeg",
		Suggestions: []Suggestion{
			{Type: SuggestionTypeReply, Text: "Buy", PostbackData: "buy_product"},
		},
	}

	data, err := json.Marshal(card)
	require.NoError(t, err)

	var decoded RichCard
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, card.Title, decoded.Title)
	assert.Equal(t, card.Description, decoded.Description)
	assert.Equal(t, card.MediaURL, decoded.MediaURL)
	assert.Equal(t, card.MediaType, decoded.MediaType)
	require.Len(t, decoded.Suggestions, 1)
	assert.Equal(t, SuggestionTypeReply, decoded.Suggestions[0].Type)
	assert.Equal(t, "Buy", decoded.Suggestions[0].Text)
	assert.Equal(t, "buy_product", decoded.Suggestions[0].PostbackData)
}

func TestCarousel_JSONRoundTrip(t *testing.T) {
	carousel := &Carousel{
		Width: "MEDIUM",
		Cards: []RichCard{
			{Title: "Card 1", Description: "First card"},
			{Title: "Card 2", Description: "Second card"},
		},
	}

	data, err := json.Marshal(carousel)
	require.NoError(t, err)

	var decoded Carousel
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "MEDIUM", decoded.Width)
	require.Len(t, decoded.Cards, 2)
	assert.Equal(t, "Card 1", decoded.Cards[0].Title)
	assert.Equal(t, "Card 2", decoded.Cards[1].Title)
}

func TestLocation_JSONRoundTrip(t *testing.T) {
	loc := &Location{
		Latitude:  -23.5505,
		Longitude: -46.6333,
		Label:     "Sao Paulo",
		Query:     "Sao Paulo, Brazil",
	}

	data, err := json.Marshal(loc)
	require.NoError(t, err)

	var decoded Location
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.InDelta(t, loc.Latitude, decoded.Latitude, 0.0001)
	assert.InDelta(t, loc.Longitude, decoded.Longitude, 0.0001)
	assert.Equal(t, loc.Label, decoded.Label)
	assert.Equal(t, loc.Query, decoded.Query)
}

func TestCalendarEvent_JSONRoundTrip(t *testing.T) {
	event := &CalendarEvent{
		Title:       "Meeting",
		Description: "Project sync",
		StartTime:   time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 3, 10, 15, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded CalendarEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Title, decoded.Title)
	assert.Equal(t, event.Description, decoded.Description)
	assert.True(t, event.StartTime.Equal(decoded.StartTime))
	assert.True(t, event.EndTime.Equal(decoded.EndTime))
}

func TestSendResult_JSONRoundTrip(t *testing.T) {
	result := &SendResult{
		Success:   true,
		MessageID: "msg-123",
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded SendResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Success, decoded.Success)
	assert.Equal(t, result.MessageID, decoded.MessageID)
}

func TestWebhookPayload_JSONRoundTrip(t *testing.T) {
	payload := &WebhookPayload{
		Provider: ProviderZenvia,
		Type:     "message",
		Message: &IncomingMessage{
			ExternalID:  "ext-123",
			SenderPhone: "+5511999999999",
			Text:        "Hello",
			Timestamp:   time.Now().UTC().Truncate(time.Second),
			AgentID:     "agent-1",
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded WebhookPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ProviderZenvia, decoded.Provider)
	assert.Equal(t, "message", decoded.Type)
	require.NotNil(t, decoded.Message)
	assert.Equal(t, "ext-123", decoded.Message.ExternalID)
	assert.Equal(t, "+5511999999999", decoded.Message.SenderPhone)
}

func TestDeliveryReport_JSONRoundTrip(t *testing.T) {
	report := &DeliveryReport{
		MessageID: "msg-456",
		Status:    StatusDelivered,
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var decoded DeliveryReport
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "msg-456", decoded.MessageID)
	assert.Equal(t, StatusDelivered, decoded.Status)
}

func TestSuggestion_JSONRoundTrip(t *testing.T) {
	suggestion := &Suggestion{
		Type:         SuggestionTypeDial,
		Text:         "Call us",
		PostbackData: "call_action",
		PhoneNumber:  "+5511999999999",
	}

	data, err := json.Marshal(suggestion)
	require.NoError(t, err)

	var decoded Suggestion
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, SuggestionTypeDial, decoded.Type)
	assert.Equal(t, "Call us", decoded.Text)
	assert.Equal(t, "+5511999999999", decoded.PhoneNumber)
}

func TestOutboundMessage_WithCardAndCarousel(t *testing.T) {
	msg := &OutboundMessage{
		To:   "+5511999999999",
		Text: "",
		Card: &RichCard{
			Title:       "Title",
			Description: "Desc",
		},
		Carousel: &Carousel{
			Width: "SMALL",
			Cards: []RichCard{
				{Title: "A"},
				{Title: "B"},
			},
		},
		Suggestions: []Suggestion{
			{Type: SuggestionTypeReply, Text: "Yes"},
			{Type: SuggestionTypeURL, Text: "Visit", URL: "https://example.com"},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded OutboundMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.NotNil(t, decoded.Card)
	assert.Equal(t, "Title", decoded.Card.Title)
	require.NotNil(t, decoded.Carousel)
	assert.Equal(t, "SMALL", decoded.Carousel.Width)
	require.Len(t, decoded.Carousel.Cards, 2)
	require.Len(t, decoded.Suggestions, 2)
	assert.Equal(t, "https://example.com", decoded.Suggestions[1].URL)
}
