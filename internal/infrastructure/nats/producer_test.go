package nats

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Constructor & Interface Tests
// ============================================================================

func TestNewProducer(t *testing.T) {
	producer := NewProducer(nil)
	assert.NotNil(t, producer)
	assert.Nil(t, producer.client)
}

func TestProducer_ImplementsPublisher(t *testing.T) {
	// This is already asserted in producer.go via var _ Publisher = (*Producer)(nil)
	// but we verify it compiles here too
	var _ Publisher = (*Producer)(nil)
}

// ============================================================================
// InboundMessage JSON Serialization
// ============================================================================

func TestInboundMessage_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	msg := &InboundMessage{
		ID:             "msg-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ChannelType:    "whatsapp",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
		ExternalID:     "ext-1",
		ContentType:    "text",
		Content:        "Hello World",
		Metadata:       map[string]string{"key": "value"},
		Attachments: []AttachmentData{
			{
				Type:     "image",
				URL:      "https://example.com/img.png",
				Filename: "img.png",
				MimeType: "image/png",
			},
		},
		Timestamp: ts,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded InboundMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.ID, decoded.ID)
	assert.Equal(t, msg.TenantID, decoded.TenantID)
	assert.Equal(t, msg.ChannelID, decoded.ChannelID)
	assert.Equal(t, msg.ChannelType, decoded.ChannelType)
	assert.Equal(t, msg.ContactID, decoded.ContactID)
	assert.Equal(t, msg.ConversationID, decoded.ConversationID)
	assert.Equal(t, msg.ExternalID, decoded.ExternalID)
	assert.Equal(t, msg.ContentType, decoded.ContentType)
	assert.Equal(t, msg.Content, decoded.Content)
	assert.Equal(t, msg.Metadata, decoded.Metadata)
	assert.Len(t, decoded.Attachments, 1)
	assert.Equal(t, "image", decoded.Attachments[0].Type)
	assert.True(t, msg.Timestamp.Equal(decoded.Timestamp))
}

func TestInboundMessage_JSONOmitEmpty(t *testing.T) {
	msg := &InboundMessage{
		ID:          "msg-1",
		TenantID:    "tenant-1",
		ChannelID:   "channel-1",
		ChannelType: "whatsapp",
		ExternalID:  "ext-1",
		ContentType: "text",
		Content:     "Hello",
		Timestamp:   time.Now(),
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// omitempty fields should be absent
	_, hasContactID := raw["contact_id"]
	_, hasConvID := raw["conversation_id"]
	_, hasMeta := raw["metadata"]
	_, hasAttach := raw["attachments"]
	assert.False(t, hasContactID)
	assert.False(t, hasConvID)
	assert.False(t, hasMeta)
	assert.False(t, hasAttach)
}

// ============================================================================
// OutboundMessage JSON Serialization
// ============================================================================

func TestOutboundMessage_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	msg := &OutboundMessage{
		ID:             "out-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ChannelType:    "whatsapp",
		ConversationID: "conv-1",
		ContactID:      "contact-1",
		RecipientID:    "+5511999999999",
		ContentType:    "text",
		Content:        "Reply",
		Metadata:       map[string]string{"source": "bot"},
		Timestamp:      ts,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded OutboundMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.ID, decoded.ID)
	assert.Equal(t, msg.RecipientID, decoded.RecipientID)
	assert.Equal(t, msg.Content, decoded.Content)
	assert.Equal(t, msg.Metadata, decoded.Metadata)
	assert.True(t, msg.Timestamp.Equal(decoded.Timestamp))
}

func TestOutboundMessage_JSONOmitEmpty(t *testing.T) {
	msg := &OutboundMessage{
		ID:             "out-1",
		TenantID:       "tenant-1",
		ChannelID:      "channel-1",
		ChannelType:    "sms",
		ConversationID: "conv-1",
		ContactID:      "contact-1",
		RecipientID:    "+5511999999999",
		ContentType:    "text",
		Content:        "Hi",
		Timestamp:      time.Now(),
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasMeta := raw["metadata"]
	_, hasAttach := raw["attachments"]
	assert.False(t, hasMeta)
	assert.False(t, hasAttach)
}

// ============================================================================
// StatusUpdate JSON Serialization
// ============================================================================

func TestStatusUpdate_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	status := &StatusUpdate{
		MessageID:    "msg-1",
		ExternalID:   "ext-1",
		ChannelType:  "whatsapp",
		Status:       "delivered",
		ErrorMessage: "",
		Timestamp:    ts,
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded StatusUpdate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, status.MessageID, decoded.MessageID)
	assert.Equal(t, status.ExternalID, decoded.ExternalID)
	assert.Equal(t, status.Status, decoded.Status)
	assert.True(t, status.Timestamp.Equal(decoded.Timestamp))
}

func TestStatusUpdate_WithError(t *testing.T) {
	status := &StatusUpdate{
		MessageID:    "msg-1",
		ChannelType:  "sms",
		Status:       "failed",
		ErrorMessage: "recipient unreachable",
		Timestamp:    time.Now(),
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded StatusUpdate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "failed", decoded.Status)
	assert.Equal(t, "recipient unreachable", decoded.ErrorMessage)
}

func TestStatusUpdate_OmitEmpty(t *testing.T) {
	status := &StatusUpdate{
		MessageID:   "msg-1",
		ChannelType: "whatsapp",
		Status:      "sent",
		Timestamp:   time.Now(),
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasExtID := raw["external_id"]
	_, hasErr := raw["error_message"]
	assert.False(t, hasExtID)
	assert.False(t, hasErr)
}

// ============================================================================
// Event JSON Serialization
// ============================================================================

func TestEvent_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	event := &Event{
		Type:     "conversation.created",
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"conversation_id": "conv-1",
			"channel_id":      "channel-1",
		},
		Timestamp: ts,
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Type, decoded.Type)
	assert.Equal(t, event.TenantID, decoded.TenantID)
	assert.Equal(t, "conv-1", decoded.Payload["conversation_id"])
	assert.True(t, event.Timestamp.Equal(decoded.Timestamp))
}

func TestEvent_EmptyPayload(t *testing.T) {
	event := &Event{
		Type:      "system.health",
		TenantID:  "tenant-1",
		Payload:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Empty(t, decoded.Payload)
}

// ============================================================================
// WebhookDelivery JSON Serialization
// ============================================================================

func TestWebhookDelivery_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	webhook := &WebhookDelivery{
		ID:        "wh-1",
		TenantID:  "tenant-1",
		URL:       "https://example.com/webhook",
		EventType: "message.received",
		Payload: map[string]interface{}{
			"message_id": "msg-1",
		},
		Headers: map[string]string{
			"X-Signature": "abc123",
		},
		RetryCount: 0,
		MaxRetries: 3,
		Timestamp:  ts,
	}

	data, err := json.Marshal(webhook)
	require.NoError(t, err)

	var decoded WebhookDelivery
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, webhook.ID, decoded.ID)
	assert.Equal(t, webhook.URL, decoded.URL)
	assert.Equal(t, webhook.EventType, decoded.EventType)
	assert.Equal(t, "msg-1", decoded.Payload["message_id"])
	assert.Equal(t, "abc123", decoded.Headers["X-Signature"])
	assert.Equal(t, 0, decoded.RetryCount)
	assert.Equal(t, 3, decoded.MaxRetries)
	assert.True(t, webhook.Timestamp.Equal(decoded.Timestamp))
}

func TestWebhookDelivery_OmitEmpty(t *testing.T) {
	webhook := &WebhookDelivery{
		ID:         "wh-1",
		TenantID:   "tenant-1",
		URL:        "https://example.com/webhook",
		EventType:  "message.sent",
		Payload:    map[string]interface{}{},
		RetryCount: 0,
		MaxRetries: 3,
		Timestamp:  time.Now(),
	}

	data, err := json.Marshal(webhook)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasHeaders := raw["headers"]
	assert.False(t, hasHeaders)
}

// ============================================================================
// AttachmentData JSON Serialization
// ============================================================================

func TestAttachmentData_JSONRoundTrip(t *testing.T) {
	att := AttachmentData{
		Type:         "image",
		URL:          "https://cdn.example.com/photo.jpg",
		Filename:     "photo.jpg",
		MimeType:     "image/jpeg",
		SizeBytes:    1024000,
		ThumbnailURL: "https://cdn.example.com/photo_thumb.jpg",
		Metadata:     map[string]string{"width": "800", "height": "600"},
	}

	data, err := json.Marshal(att)
	require.NoError(t, err)

	var decoded AttachmentData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, att.Type, decoded.Type)
	assert.Equal(t, att.URL, decoded.URL)
	assert.Equal(t, att.Filename, decoded.Filename)
	assert.Equal(t, att.MimeType, decoded.MimeType)
	assert.Equal(t, att.SizeBytes, decoded.SizeBytes)
	assert.Equal(t, att.ThumbnailURL, decoded.ThumbnailURL)
	assert.Equal(t, att.Metadata, decoded.Metadata)
}

func TestAttachmentData_MinimalFields(t *testing.T) {
	att := AttachmentData{
		Type: "document",
		URL:  "https://cdn.example.com/file.pdf",
	}

	data, err := json.Marshal(att)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// omitempty fields
	_, hasFilename := raw["filename"]
	_, hasMime := raw["mime_type"]
	_, hasSize := raw["size_bytes"]
	_, hasThumb := raw["thumbnail_url"]
	_, hasMeta := raw["metadata"]
	assert.False(t, hasFilename)
	assert.False(t, hasMime)
	assert.False(t, hasSize)
	assert.False(t, hasThumb)
	assert.False(t, hasMeta)
}

// ============================================================================
// Cross-type JSON Keys Verification
// ============================================================================

func TestJSONFieldNames(t *testing.T) {
	t.Run("InboundMessage fields", func(t *testing.T) {
		msg := &InboundMessage{
			ID:          "id",
			TenantID:    "tid",
			ChannelID:   "cid",
			ChannelType: "ct",
			ExternalID:  "eid",
			ContentType: "text",
			Content:     "c",
			Timestamp:   time.Now(),
		}
		data, _ := json.Marshal(msg)
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		assert.Contains(t, raw, "id")
		assert.Contains(t, raw, "tenant_id")
		assert.Contains(t, raw, "channel_id")
		assert.Contains(t, raw, "channel_type")
		assert.Contains(t, raw, "external_id")
		assert.Contains(t, raw, "content_type")
		assert.Contains(t, raw, "content")
		assert.Contains(t, raw, "timestamp")
	})

	t.Run("OutboundMessage has recipient_id", func(t *testing.T) {
		msg := &OutboundMessage{
			ID:             "id",
			TenantID:       "tid",
			ChannelID:      "cid",
			ChannelType:    "ct",
			ConversationID: "conv",
			ContactID:      "contact",
			RecipientID:    "recipient",
			ContentType:    "text",
			Content:        "c",
			Timestamp:      time.Now(),
		}
		data, _ := json.Marshal(msg)
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		assert.Contains(t, raw, "recipient_id")
		assert.Equal(t, "recipient", raw["recipient_id"])
	})

	t.Run("WebhookDelivery fields", func(t *testing.T) {
		wh := &WebhookDelivery{
			ID:         "id",
			TenantID:   "tid",
			URL:        "url",
			EventType:  "et",
			Payload:    map[string]interface{}{},
			RetryCount: 1,
			MaxRetries: 5,
			Timestamp:  time.Now(),
		}
		data, _ := json.Marshal(wh)
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		assert.Contains(t, raw, "event_type")
		assert.Contains(t, raw, "retry_count")
		assert.Contains(t, raw, "max_retries")
	})
}
