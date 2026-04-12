package whatsapp_official

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers to build test payloads
// ---------------------------------------------------------------------------

func testConfig() *Config {
	return &Config{
		AccessToken:   "test-access-token",
		PhoneNumberID: "123456789",
		BusinessID:    "987654321",
		VerifyToken:   "my-verify-token",
		WebhookSecret: "my-webhook-secret",
		APIVersion:    "v21.0",
	}
}

func testProcessor() *WebhookProcessor {
	return NewWebhookProcessor(testConfig())
}

// buildMessagePayload creates a WebhookPayload containing a single message.
func buildMessagePayload(msg IncomingMessage, contacts []ContactInfo) *WebhookPayload {
	return &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{
				ID: "ENTRY1",
				Changes: []WebhookChange{
					{
						Field: "messages",
						Value: WebhookChangeValue{
							MessagingProduct: "whatsapp",
							Metadata: WebhookMetadata{
								DisplayPhoneNumber: "+1555000111",
								PhoneNumberID:      "100200300",
							},
							Contacts: contacts,
							Messages: []IncomingMessage{msg},
						},
					},
				},
			},
		},
	}
}

// buildStatusPayload creates a WebhookPayload containing a single status update.
func buildStatusPayload(status StatusUpdate) *WebhookPayload {
	return &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{
				ID: "ENTRY1",
				Changes: []WebhookChange{
					{
						Field: "messages",
						Value: WebhookChangeValue{
							MessagingProduct: "whatsapp",
							Metadata: WebhookMetadata{
								DisplayPhoneNumber: "+1555000111",
								PhoneNumberID:      "100200300",
							},
							Statuses: []StatusUpdate{status},
						},
					},
				},
			},
		},
	}
}

func defaultContacts() []ContactInfo {
	return []ContactInfo{
		{
			WaID: "5511999998888",
			Profile: ContactProfile{
				Name: "Test User",
			},
		},
	}
}

func payloadJSON(p *WebhookPayload) []byte {
	b, _ := json.Marshal(p)
	return b
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// ---------------------------------------------------------------------------
// 1. ValidateSignature
// ---------------------------------------------------------------------------

func TestValidateSignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	secret := "test-secret"
	validSig := computeSignature(secret, body)

	tests := []struct {
		name      string
		secret    string
		body      []byte
		signature string
		want      bool
	}{
		{"valid signature", secret, body, validSig, true},
		{"invalid signature", secret, body, "sha256=badbadbadbad", false},
		{"empty secret", "", body, validSig, false},
		{"empty signature", secret, body, "", false},
		{"both empty", "", body, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateSignature(tt.secret, tt.body, tt.signature)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// 2. VerifyChallenge
// ---------------------------------------------------------------------------

func TestVerifyChallenge(t *testing.T) {
	tests := []struct {
		name        string
		verifyToken string
		mode        string
		token       string
		challenge   string
		wantResp    string
		wantOK      bool
	}{
		{"subscribe correct token", "tok", "subscribe", "tok", "ch123", "ch123", true},
		{"subscribe wrong token", "tok", "subscribe", "wrong", "ch123", "", false},
		{"wrong mode", "tok", "unsubscribe", "tok", "ch123", "", false},
		{"empty mode", "tok", "", "tok", "ch123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, ok := VerifyChallenge(tt.verifyToken, tt.mode, tt.token, tt.challenge)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantResp, resp)
		})
	}
}

// ---------------------------------------------------------------------------
// 3. ParseWebhook
// ---------------------------------------------------------------------------

func TestParseWebhook(t *testing.T) {
	proc := testProcessor()

	t.Run("valid payload", func(t *testing.T) {
		body := payloadJSON(&WebhookPayload{
			Object: "whatsapp_business_account",
			Entry:  []WebhookEntry{{ID: "E1"}},
		})
		payload, err := proc.ParseWebhook(body)
		require.NoError(t, err)
		assert.Equal(t, "whatsapp_business_account", payload.Object)
		assert.Len(t, payload.Entry, 1)
	})

	t.Run("wrong object", func(t *testing.T) {
		body := payloadJSON(&WebhookPayload{Object: "instagram"})
		_, err := proc.ParseWebhook(body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook object")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := proc.ParseWebhook([]byte(`{not valid json`))
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// 4. ExtractMessages - ALL message types
// ---------------------------------------------------------------------------

func TestExtractMessages_Text(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-text-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "Hello world"},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, "msg-text-1", m.ExternalID)
	assert.Equal(t, "5511999998888", m.From)
	assert.Equal(t, "Test User", m.SenderName)
	assert.Equal(t, plugin.ContentTypeText, m.ContentType)
	assert.Equal(t, "Hello world", m.Content)
	assert.Equal(t, "100200300", m.PhoneNumberID)
	assert.Empty(t, m.Attachments)
}

func TestExtractMessages_Image(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-img-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeImage,
			Image: &MediaContent{
				ID:       "media-img-1",
				Caption:  "Photo caption",
				MimeType: "image/jpeg",
				SHA256:   "abc123sha",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeImage, m.ContentType)
	assert.Equal(t, "Photo caption", m.Content)
	require.Len(t, m.Attachments, 1)
	att := m.Attachments[0]
	assert.Equal(t, "image", att.Type)
	assert.Equal(t, "media-img-1", att.URL)
	assert.Equal(t, "image/jpeg", att.MimeType)
	assert.Equal(t, "media-img-1", att.Metadata["media_id"])
	assert.Equal(t, "abc123sha", att.Metadata["sha256"])
}

func TestExtractMessages_Video(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-vid-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeVideo,
			Video: &MediaContent{
				ID:       "media-vid-1",
				Caption:  "Video caption",
				MimeType: "video/mp4",
				SHA256:   "vid-sha",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeVideo, m.ContentType)
	assert.Equal(t, "Video caption", m.Content)
	require.Len(t, m.Attachments, 1)
	assert.Equal(t, "video", m.Attachments[0].Type)
	assert.Equal(t, "media-vid-1", m.Attachments[0].Metadata["media_id"])
}

func TestExtractMessages_Audio(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-aud-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeAudio,
			Audio: &MediaContent{
				ID:       "media-aud-1",
				MimeType: "audio/ogg",
				SHA256:   "aud-sha",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeAudio, m.ContentType)
	assert.Equal(t, "", m.Content)
	require.Len(t, m.Attachments, 1)
	assert.Equal(t, "audio", m.Attachments[0].Type)
	assert.Equal(t, "audio/ogg", m.Attachments[0].MimeType)
}

func TestExtractMessages_Document(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-doc-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeDocument,
			Document: &DocumentContent{
				ID:       "media-doc-1",
				Caption:  "Doc caption",
				Filename: "report.pdf",
				MimeType: "application/pdf",
				SHA256:   "doc-sha",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeDocument, m.ContentType)
	assert.Equal(t, "Doc caption", m.Content)
	require.Len(t, m.Attachments, 1)
	att := m.Attachments[0]
	assert.Equal(t, "document", att.Type)
	assert.Equal(t, "report.pdf", att.Filename)
	assert.Equal(t, "application/pdf", att.MimeType)
	assert.Equal(t, "media-doc-1", att.Metadata["media_id"])
	assert.Equal(t, "doc-sha", att.Metadata["sha256"])
}

func TestExtractMessages_Sticker(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-stk-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeSticker,
			Sticker: &StickerContent{
				ID:       "media-stk-1",
				MimeType: "image/webp",
				SHA256:   "stk-sha",
				Animated: true,
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeImage, m.ContentType)
	assert.Equal(t, "true", m.Metadata["is_sticker"])
	require.Len(t, m.Attachments, 1)
	att := m.Attachments[0]
	assert.Equal(t, "sticker", att.Type)
	assert.Equal(t, "image/webp", att.MimeType)
	assert.Equal(t, "true", att.Metadata["animated"])
}

func TestExtractMessages_Location(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-loc-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeLocation,
			Location: &LocationContent{
				Latitude:  -23.5505,
				Longitude: -46.6333,
				Name:      "Sao Paulo",
				Address:   "Paulista Ave",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeLocation, m.ContentType)
	// Content should be JSON of the location
	assert.Contains(t, m.Content, "-23.5505")
	assert.Contains(t, m.Metadata["latitude"], "-23.55")
	assert.Contains(t, m.Metadata["longitude"], "-46.63")
	assert.Equal(t, "Sao Paulo", m.Metadata["location_name"])
	assert.Equal(t, "Paulista Ave", m.Metadata["location_address"])
}

func TestExtractMessages_Contacts(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-cnt-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeContacts,
			Contacts: []ContactContent{
				{
					Name: ContactName{
						FormattedName: "John Doe",
						FirstName:     "John",
						LastName:      "Doe",
					},
					Phones: []ContactPhone{
						{Phone: "+1234567890", Type: "CELL"},
					},
				},
				{
					Name: ContactName{FormattedName: "Jane Doe"},
				},
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeContact, m.ContentType)
	assert.Contains(t, m.Content, "John Doe")
	assert.Equal(t, "2", m.Metadata["contact_count"])
}

func TestExtractMessages_Interactive_ButtonReply(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ibr-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeInteractive,
			Interactive: &InteractiveResponse{
				Type: "button_reply",
				ButtonReply: &ButtonReplyData{
					ID:    "btn-1",
					Title: "Yes",
				},
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeInteractive, m.ContentType)
	assert.Equal(t, "Yes", m.Content)
	assert.Equal(t, "btn-1", m.Metadata["button_id"])
	assert.Equal(t, "Yes", m.Metadata["button_title"])
	assert.Equal(t, "button_reply", m.Metadata["interactive_type"])
}

func TestExtractMessages_Interactive_ListReply(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ilr-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeInteractive,
			Interactive: &InteractiveResponse{
				Type: "list_reply",
				ListReply: &ListReplyData{
					ID:          "list-1",
					Title:       "Option A",
					Description: "First option",
				},
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeInteractive, m.ContentType)
	assert.Equal(t, "Option A", m.Content)
	assert.Equal(t, "list-1", m.Metadata["list_id"])
	assert.Equal(t, "Option A", m.Metadata["list_title"])
	assert.Equal(t, "First option", m.Metadata["list_description"])
}

func TestExtractMessages_Interactive_NfmReply(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-nfm-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeInteractive,
			Interactive: &InteractiveResponse{
				Type: "nfm_reply",
				NfmReply: &NfmReplyData{
					Name:         "my_flow",
					Body:         "Flow body text",
					ResponseJSON: `{"screen":"SUCCESS","data":{"key":"value"}}`,
					FlowToken:    "tok-123",
				},
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeInteractive, m.ContentType)
	assert.Equal(t, `{"screen":"SUCCESS","data":{"key":"value"}}`, m.Content)
	assert.Equal(t, "true", m.Metadata["is_flow_response"])
	assert.Equal(t, "my_flow", m.Metadata["flow_name"])
	assert.Equal(t, "Flow body text", m.Metadata["flow_body"])
	assert.Equal(t, `{"screen":"SUCCESS","data":{"key":"value"}}`, m.Metadata["flow_response_json"])
	assert.Equal(t, "tok-123", m.Metadata["flow_token"])
}

func TestExtractMessages_Order(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ord-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeOrder,
			Order: &OrderContent{
				CatalogID: "catalog-99",
				Text:      "I want these items",
				ProductItems: []OrderItem{
					{ProductRetailerID: "SKU-A", Quantity: 2, ItemPrice: 19.99, Currency: "USD"},
					{ProductRetailerID: "SKU-B", Quantity: 1, ItemPrice: 5.50, Currency: "USD"},
				},
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeInteractive, m.ContentType)
	assert.Equal(t, "true", m.Metadata["is_order"])
	assert.Equal(t, "catalog-99", m.Metadata["catalog_id"])
	assert.Equal(t, "I want these items", m.Content)
	assert.Contains(t, m.Metadata["order_items"], "SKU-A")
	assert.Equal(t, "2", m.Metadata["order_item_count"])
}

func TestExtractMessages_Button(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-btn-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeButton,
			Button: &ButtonResponse{
				Text:    "Buy now",
				Payload: "buy_payload_123",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeInteractive, m.ContentType)
	assert.Equal(t, "Buy now", m.Content)
	assert.Equal(t, "buy_payload_123", m.Metadata["button_payload"])
}

func TestExtractMessages_Reaction(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-react-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeReaction,
			Reaction: &ReactionContent{
				MessageID: "original-msg-1",
				Emoji:     "\U0001F44D",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, plugin.ContentTypeText, m.ContentType)
	assert.Equal(t, "true", m.Metadata["is_reaction"])
	assert.Equal(t, "original-msg-1", m.Metadata["reaction_message_id"])
	assert.Equal(t, "\U0001F44D", m.Metadata["reaction_emoji"])
	assert.Equal(t, "\U0001F44D", m.Content)
}

// ---------------------------------------------------------------------------
// 5. Context / reply-to
// ---------------------------------------------------------------------------

func TestExtractMessages_Context(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ctx-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "reply text"},
			Context: &MessageContext{
				MessageID: "original-msg-999",
				From:      "5511888887777",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, "original-msg-999", m.ReplyToID)
	assert.Equal(t, "original-msg-999", m.Metadata["reply_to_id"])
	assert.Equal(t, "5511888887777", m.Metadata["reply_to_from"])
}

func TestExtractMessages_ContextForwarded(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-fwd-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "forwarded text"},
			Context: &MessageContext{
				MessageID:           "orig-1",
				From:                "5511888887777",
				Forwarded:           true,
				FrequentlyForwarded: true,
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, "true", m.Metadata["forwarded"])
	assert.Equal(t, "true", m.Metadata["frequently_forwarded"])
}

// ---------------------------------------------------------------------------
// 6. Referral
// ---------------------------------------------------------------------------

func TestExtractMessages_Referral(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ref-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "came from ad"},
			Referral: &ReferralContent{
				SourceType: "ad",
				SourceID:   "ad-12345",
				Headline:   "Summer Sale",
				Body:       "50% off",
			},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, "ad", m.Metadata["referral_source_type"])
	assert.Equal(t, "ad-12345", m.Metadata["referral_source_id"])
	assert.Equal(t, "Summer Sale", m.Metadata["referral_headline"])
	assert.Equal(t, "50% off", m.Metadata["referral_body"])
}

// ---------------------------------------------------------------------------
// 7. Timestamp parsing
// ---------------------------------------------------------------------------

func TestExtractMessages_TimestampValid(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ts-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "ts test"},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	assert.Equal(t, time.Unix(1700000000, 0), msgs[0].Timestamp)
}

func TestExtractMessages_TimestampInvalid(t *testing.T) {
	proc := testProcessor()
	before := time.Now().Add(-2 * time.Second)
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-ts-bad",
			From:      "5511999998888",
			Timestamp: "not-a-number",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "bad ts"},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	after := time.Now().Add(2 * time.Second)
	assert.True(t, msgs[0].Timestamp.After(before), "timestamp should be approximately now")
	assert.True(t, msgs[0].Timestamp.Before(after), "timestamp should be approximately now")
}

// ---------------------------------------------------------------------------
// 8. ExtractStatuses
// ---------------------------------------------------------------------------

func TestExtractStatuses(t *testing.T) {
	tests := []struct {
		name       string
		waStatus   MessageStatus
		wantStatus plugin.MessageStatus
		errors     []WebhookError
		wantErrMsg string
	}{
		{"sent", StatusSent, plugin.MessageStatusSent, nil, ""},
		{"delivered", StatusDelivered, plugin.MessageStatusDelivered, nil, ""},
		{"read", StatusRead, plugin.MessageStatusRead, nil, ""},
		{"failed with error", StatusFailed, plugin.MessageStatusFailed,
			[]WebhookError{{Code: 131047, Title: "Re-engagement message", Message: "Outside 24h window"}},
			"Outside 24h window"},
		{"unknown status", MessageStatus("something_else"), plugin.MessageStatusPending, nil, ""},
	}

	proc := testProcessor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := buildStatusPayload(StatusUpdate{
				ID:          "wamid-status-1",
				RecipientID: "5511888887777",
				Status:      tt.waStatus,
				Timestamp:   "1700000000",
				Errors:      tt.errors,
			})

			statuses := proc.ExtractStatuses(payload)
			require.Len(t, statuses, 1)
			s := statuses[0]
			assert.Equal(t, "wamid-status-1", s.MessageID)
			assert.Equal(t, "5511888887777", s.RecipientID)
			assert.Equal(t, tt.wantStatus, s.Status)
			assert.Equal(t, tt.wantErrMsg, s.ErrorMessage)
			assert.Equal(t, time.Unix(1700000000, 0), s.Timestamp)
		})
	}
}

// ---------------------------------------------------------------------------
// 9. ToInboundMessage (ParsedMessage)
// ---------------------------------------------------------------------------

func TestParsedMessage_ToInboundMessage(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	pm := &ParsedMessage{
		ExternalID:    "ext-1",
		From:          "5511999998888",
		SenderName:    "Alice",
		ContentType:   plugin.ContentTypeText,
		Content:       "hello",
		Attachments:   nil,
		Metadata:      map[string]string{"key": "val"},
		ReplyToID:     "reply-1",
		Timestamp:     ts,
		PhoneNumberID: "pn-1",
	}

	inb := pm.ToInboundMessage()
	assert.Equal(t, "ext-1", inb.ExternalID)
	assert.Equal(t, "5511999998888", inb.SenderID)
	assert.Equal(t, "Alice", inb.SenderName)
	assert.Equal(t, plugin.ContentTypeText, inb.ContentType)
	assert.Equal(t, "hello", inb.Content)
	assert.Equal(t, "val", inb.Metadata["key"])
	assert.Equal(t, ts, inb.Timestamp)
}

// ---------------------------------------------------------------------------
// 10. ToStatusCallback (ParsedStatus)
// ---------------------------------------------------------------------------

func TestParsedStatus_ToStatusCallback(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	ps := &ParsedStatus{
		MessageID:    "msg-1",
		RecipientID:  "55119999",
		Status:       plugin.MessageStatusDelivered,
		ErrorMessage: "",
		Timestamp:    ts,
	}

	cb := ps.ToStatusCallback()
	assert.Equal(t, "msg-1", cb.ExternalID)
	assert.Equal(t, plugin.MessageStatusDelivered, cb.Status)
	assert.Equal(t, "", cb.ErrorMessage)
	assert.Equal(t, ts, cb.Timestamp)
}

// ---------------------------------------------------------------------------
// 11. Helper functions: IsMessageWebhook, IsStatusWebhook, GetWebhookPhoneNumberID
// ---------------------------------------------------------------------------

func TestIsMessageWebhook(t *testing.T) {
	t.Run("has messages", func(t *testing.T) {
		payload := buildMessagePayload(
			IncomingMessage{ID: "m1", From: "5511999998888", Timestamp: "1700000000", Type: MessageTypeText, Text: &TextContent{Body: "hi"}},
			defaultContacts(),
		)
		assert.True(t, IsMessageWebhook(payload))
	})

	t.Run("no messages", func(t *testing.T) {
		payload := buildStatusPayload(StatusUpdate{ID: "s1", RecipientID: "5511", Status: StatusSent, Timestamp: "1700000000"})
		assert.False(t, IsMessageWebhook(payload))
	})
}

func TestIsStatusWebhook(t *testing.T) {
	t.Run("has statuses", func(t *testing.T) {
		payload := buildStatusPayload(StatusUpdate{ID: "s1", RecipientID: "5511", Status: StatusSent, Timestamp: "1700000000"})
		assert.True(t, IsStatusWebhook(payload))
	})

	t.Run("no statuses", func(t *testing.T) {
		payload := buildMessagePayload(
			IncomingMessage{ID: "m1", From: "5511999998888", Timestamp: "1700000000", Type: MessageTypeText, Text: &TextContent{Body: "hi"}},
			defaultContacts(),
		)
		assert.False(t, IsStatusWebhook(payload))
	})
}

func TestGetWebhookPhoneNumberID(t *testing.T) {
	payload := buildMessagePayload(
		IncomingMessage{ID: "m1", From: "5511999998888", Timestamp: "1700000000", Type: MessageTypeText, Text: &TextContent{Body: "hi"}},
		defaultContacts(),
	)
	assert.Equal(t, "100200300", GetWebhookPhoneNumberID(payload))
}

func TestGetWebhookPhoneNumberID_Empty(t *testing.T) {
	payload := &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry:  []WebhookEntry{},
	}
	assert.Equal(t, "", GetWebhookPhoneNumberID(payload))
}

// ---------------------------------------------------------------------------
// 12. Extended webhook field functions
// ---------------------------------------------------------------------------

func buildFieldPayload(field string, value WebhookChangeValue) *WebhookPayload {
	return &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{
				ID: "ENTRY-EXT",
				Changes: []WebhookChange{
					{Field: field, Value: value},
				},
			},
		},
	}
}

func TestExtractTemplateStatusUpdates(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("message_template_status_update", WebhookChangeValue{
		Event:                   "APPROVED",
		MessageTemplateID:       42,
		MessageTemplateName:     "hello_world",
		MessageTemplateLanguage: "en_US",
		Reason:                  "",
	})

	events := proc.ExtractTemplateStatusUpdates(payload)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, int64(42), e.TemplateID)
	assert.Equal(t, "hello_world", e.TemplateName)
	assert.Equal(t, "en_US", e.Language)
	assert.Equal(t, "APPROVED", e.Event)
}

func TestExtractTemplateQualityUpdates(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("message_template_quality_update", WebhookChangeValue{
		MessageTemplateID:       43,
		MessageTemplateName:     "promo",
		MessageTemplateLanguage: "pt_BR",
		PreviousQualityScore:    "GREEN",
		NewQualityScore:         "YELLOW",
	})

	events := proc.ExtractTemplateQualityUpdates(payload)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, int64(43), e.TemplateID)
	assert.Equal(t, "GREEN", e.PreviousQuality)
	assert.Equal(t, "YELLOW", e.NewQuality)
}

func TestExtractTemplateCategoryUpdates(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("template_category_update", WebhookChangeValue{
		MessageTemplateID:       44,
		MessageTemplateName:     "promo",
		MessageTemplateLanguage: "pt_BR",
		PreviousCategory:        "UTILITY",
		NewCategory:             "MARKETING",
	})

	events := proc.ExtractTemplateCategoryUpdates(payload)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, int64(44), e.TemplateID)
	assert.Equal(t, "UTILITY", e.PreviousCategory)
	assert.Equal(t, "MARKETING", e.NewCategory)
}

func TestExtractAccountAlerts(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("account_alerts", WebhookChangeValue{
		Title:   "Rate limit approaching",
		Message: "You are near your messaging limit",
	})

	events := proc.ExtractAccountAlerts(payload)
	require.Len(t, events, 1)
	assert.Equal(t, "Rate limit approaching", events[0].Title)
	assert.Equal(t, "You are near your messaging limit", events[0].Message)
}

func TestExtractPhoneQualityUpdates(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("phone_number_quality_update", WebhookChangeValue{
		DisplayPhoneNumber: "+1555000222",
		Event:              "FLAGGED",
		CurrentLimit:       "TIER_1K",
	})

	events := proc.ExtractPhoneQualityUpdates(payload)
	require.Len(t, events, 1)
	assert.Equal(t, "+1555000222", events[0].PhoneNumber)
	assert.Equal(t, "FLAGGED", events[0].Event)
	assert.Equal(t, "TIER_1K", events[0].CurrentLimit)
}

func TestExtractFlowEvents(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("flows", WebhookChangeValue{
		FlowID:       "flow-1",
		FlowName:     "survey_flow",
		Event:        "FLOW_STATUS_CHANGE",
		OldStatus:    "DRAFT",
		NewStatus:    "PUBLISHED",
		ErrorType:    "",
		ErrorMessage: "",
	})

	events := proc.ExtractFlowEvents(payload)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, "flow-1", e.FlowID)
	assert.Equal(t, "survey_flow", e.FlowName)
	assert.Equal(t, "FLOW_STATUS_CHANGE", e.Event)
	assert.Equal(t, "DRAFT", e.OldStatus)
	assert.Equal(t, "PUBLISHED", e.NewStatus)
}

func TestExtractSecurityEvents(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("security", WebhookChangeValue{
		Event:              "TWO_STEP_VERIFICATION_DISABLED",
		DisplayPhoneNumber: "+1555000333",
	})

	events := proc.ExtractSecurityEvents(payload)
	require.Len(t, events, 1)
	assert.Equal(t, "TWO_STEP_VERIFICATION_DISABLED", events[0].Event)
	assert.Equal(t, "+1555000333", events[0].PhoneNumber)
}

func TestGetWebhookFields(t *testing.T) {
	proc := testProcessor()
	payload := &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{
				ID: "E1",
				Changes: []WebhookChange{
					{Field: "messages", Value: WebhookChangeValue{}},
					{Field: "flows", Value: WebhookChangeValue{}},
				},
			},
		},
	}

	fields := proc.GetWebhookFields(payload)
	assert.Len(t, fields, 2)
	fieldStrs := make(map[WebhookFieldType]bool)
	for _, f := range fields {
		fieldStrs[f] = true
	}
	assert.True(t, fieldStrs[FieldMessages])
	assert.True(t, fieldStrs[FieldFlows])
}

// ---------------------------------------------------------------------------
// Is*Webhook helpers
// ---------------------------------------------------------------------------

func TestIsTemplateStatusWebhook(t *testing.T) {
	payload := buildFieldPayload("message_template_status_update", WebhookChangeValue{})
	assert.True(t, IsTemplateStatusWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsTemplateStatusWebhook(payloadOther))
}

func TestIsTemplateQualityWebhook(t *testing.T) {
	payload := buildFieldPayload("message_template_quality_update", WebhookChangeValue{})
	assert.True(t, IsTemplateQualityWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsTemplateQualityWebhook(payloadOther))
}

func TestIsFlowWebhook(t *testing.T) {
	payload := buildFieldPayload("flows", WebhookChangeValue{})
	assert.True(t, IsFlowWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsFlowWebhook(payloadOther))
}

func TestIsTemplateCategoryWebhook(t *testing.T) {
	payload := buildFieldPayload("template_category_update", WebhookChangeValue{})
	assert.True(t, IsTemplateCategoryWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsTemplateCategoryWebhook(payloadOther))
}

func TestIsAccountAlertWebhook(t *testing.T) {
	payload := buildFieldPayload("account_alerts", WebhookChangeValue{})
	assert.True(t, IsAccountAlertWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsAccountAlertWebhook(payloadOther))
}

func TestIsPhoneQualityWebhook(t *testing.T) {
	payload := buildFieldPayload("phone_number_quality_update", WebhookChangeValue{})
	assert.True(t, IsPhoneQualityWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsPhoneQualityWebhook(payloadOther))
}

func TestIsSecurityWebhook(t *testing.T) {
	payload := buildFieldPayload("security", WebhookChangeValue{})
	assert.True(t, IsSecurityWebhook(payload))

	payloadOther := buildFieldPayload("messages", WebhookChangeValue{})
	assert.False(t, IsSecurityWebhook(payloadOther))
}

// ---------------------------------------------------------------------------
// 13. Message Echoes (coexistence)
// ---------------------------------------------------------------------------

func buildEchoPayload(msg IncomingMessage, contacts []ContactInfo) *WebhookPayload {
	return &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{
				ID: "ENTRY-ECHO",
				Changes: []WebhookChange{
					{
						Field: "message_echoes",
						Value: WebhookChangeValue{
							MessagingProduct: "whatsapp",
							Metadata: WebhookMetadata{
								DisplayPhoneNumber: "+1555000111",
								PhoneNumberID:      "100200300",
							},
							Contacts: contacts,
							Messages: []IncomingMessage{msg},
						},
					},
				},
			},
		},
	}
}

func TestExtractMessageEchoes_TextEcho(t *testing.T) {
	proc := testProcessor()
	payload := buildEchoPayload(
		IncomingMessage{
			ID:        "echo-msg-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "Hello from business app"},
		},
		[]ContactInfo{
			{WaID: "5511999998888", Profile: ContactProfile{Name: "Customer"}},
		},
	)

	echoes := proc.ExtractMessageEchoes(payload)
	require.Len(t, echoes, 1)
	e := echoes[0]

	assert.Equal(t, "echo-msg-1", e.ExternalID)
	assert.Equal(t, "5511999998888", e.To)
	assert.Equal(t, "Customer", e.RecipientName)
	assert.Equal(t, plugin.ContentTypeText, e.ContentType)
	assert.Equal(t, "Hello from business app", e.Content)
	assert.Equal(t, "100200300", e.PhoneNumberID)
	assert.Equal(t, "+1555000111", e.SenderPhone)

	// Echo metadata
	assert.Equal(t, "business_app", e.Metadata["source"])
	assert.Equal(t, "true", e.Metadata["is_echo"])
	assert.Equal(t, "5511999998888", e.Metadata["recipient_phone"])
	assert.Equal(t, "+1555000111", e.Metadata["sender_phone"])
	assert.Equal(t, "100200300", e.Metadata["phone_number_id"])
}

func TestParsedMessageEcho_ToInboundMessage(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	echo := &ParsedMessageEcho{
		ExternalID:    "echo-1",
		To:            "5511999998888",
		RecipientName: "Customer",
		ContentType:   plugin.ContentTypeText,
		Content:       "Hello",
		Metadata:      map[string]string{"source": "business_app", "is_echo": "true"},
		Timestamp:     ts,
		PhoneNumberID: "100200300",
		SenderPhone:   "+1555000111",
	}

	inb := echo.ToInboundMessage()
	assert.Equal(t, "echo-1", inb.ExternalID)
	assert.Equal(t, "+1555000111", inb.SenderID, "SenderID should be the business sender phone")
	assert.Equal(t, "+1555000111", inb.SenderName, "SenderName should be the business sender phone")
	assert.Equal(t, plugin.ContentTypeText, inb.ContentType)
	assert.Equal(t, "Hello", inb.Content)
	assert.Equal(t, "true", inb.Metadata["is_echo"])
	assert.Equal(t, ts, inb.Timestamp)
}

func TestIsMessageEchoWebhook(t *testing.T) {
	t.Run("has echo messages", func(t *testing.T) {
		payload := buildEchoPayload(
			IncomingMessage{ID: "e1", From: "55119999", Timestamp: "1700000000", Type: MessageTypeText, Text: &TextContent{Body: "echo"}},
			defaultContacts(),
		)
		assert.True(t, IsMessageEchoWebhook(payload))
	})

	t.Run("echo field but no messages", func(t *testing.T) {
		payload := &WebhookPayload{
			Object: "whatsapp_business_account",
			Entry: []WebhookEntry{
				{
					ID: "E",
					Changes: []WebhookChange{
						{Field: "message_echoes", Value: WebhookChangeValue{}},
					},
				},
			},
		}
		assert.False(t, IsMessageEchoWebhook(payload))
	})

	t.Run("regular message is not echo", func(t *testing.T) {
		payload := buildMessagePayload(
			IncomingMessage{ID: "m1", From: "55119999", Timestamp: "1700000000", Type: MessageTypeText, Text: &TextContent{Body: "hi"}},
			defaultContacts(),
		)
		assert.False(t, IsMessageEchoWebhook(payload))
	})
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestExtractMessages_NoContacts(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-no-contact",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "no contacts array"},
		},
		nil, // no contacts
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	assert.Equal(t, "", msgs[0].SenderName)
}

func TestExtractMessages_CommonMetadata(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-meta",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "meta check"},
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]

	assert.Equal(t, "5511999998888", m.Metadata["phone"])
	assert.Equal(t, "5511999998888", m.Metadata["sender_id"])
	assert.Equal(t, "Test User", m.Metadata["sender_name"])
	assert.Equal(t, "100200300", m.Metadata["phone_number_id"])
	assert.Equal(t, "+1555000111", m.Metadata["display_phone_number"])
}

func TestExtractStatuses_EmptyPayload(t *testing.T) {
	proc := testProcessor()
	payload := &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry:  []WebhookEntry{},
	}
	statuses := proc.ExtractStatuses(payload)
	assert.Empty(t, statuses)
}

func TestExtractMessages_EmptyPayload(t *testing.T) {
	proc := testProcessor()
	payload := &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry:  []WebhookEntry{},
	}
	msgs := proc.ExtractMessages(payload)
	assert.Empty(t, msgs)
}

func TestExtractMessages_UnknownType(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "msg-unknown",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageType("ephemeral"),
		},
		defaultContacts(),
	)

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 1)
	m := msgs[0]
	assert.Equal(t, plugin.ContentTypeText, m.ContentType)
	assert.Equal(t, "ephemeral", m.Metadata["original_type"])
}

// ---------------------------------------------------------------------------
// ParseWebhook round-trip: build JSON from struct, parse, then extract
// ---------------------------------------------------------------------------

func TestParseWebhookRoundTrip(t *testing.T) {
	proc := testProcessor()
	payload := buildMessagePayload(
		IncomingMessage{
			ID:        "rt-msg-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeText,
			Text:      &TextContent{Body: "round trip"},
		},
		defaultContacts(),
	)

	body := payloadJSON(payload)
	parsed, err := proc.ParseWebhook(body)
	require.NoError(t, err)

	msgs := proc.ExtractMessages(parsed)
	require.Len(t, msgs, 1)
	assert.Equal(t, "rt-msg-1", msgs[0].ExternalID)
	assert.Equal(t, "round trip", msgs[0].Content)
}

// ---------------------------------------------------------------------------
// Validate signature with actual body from ParseWebhook
// ---------------------------------------------------------------------------

func TestValidateSignature_RealBody(t *testing.T) {
	secret := "real-secret"
	body := payloadJSON(&WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{ID: "E1", Changes: []WebhookChange{
				{Field: "messages", Value: WebhookChangeValue{
					Messages: []IncomingMessage{{ID: "m1", From: "5511", Timestamp: "1700000000", Type: MessageTypeText}},
				}},
			}},
		},
	})

	sig := computeSignature(secret, body)
	assert.True(t, ValidateSignature(secret, body, sig))

	// Tamper one byte
	tampered := make([]byte, len(body))
	copy(tampered, body)
	tampered[5] = 'X'
	assert.False(t, ValidateSignature(secret, tampered, sig))
}

// ---------------------------------------------------------------------------
// ExtractMessageEchoes: image echo
// ---------------------------------------------------------------------------

func TestExtractMessageEchoes_ImageEcho(t *testing.T) {
	proc := testProcessor()
	payload := buildEchoPayload(
		IncomingMessage{
			ID:        "echo-img-1",
			From:      "5511999998888",
			Timestamp: "1700000000",
			Type:      MessageTypeImage,
			Image: &MediaContent{
				ID:       "media-echo-img",
				Caption:  "Echo image caption",
				MimeType: "image/jpeg",
				SHA256:   "echo-sha",
			},
		},
		defaultContacts(),
	)

	echoes := proc.ExtractMessageEchoes(payload)
	require.Len(t, echoes, 1)
	e := echoes[0]

	assert.Equal(t, plugin.ContentTypeImage, e.ContentType)
	assert.Equal(t, "Echo image caption", e.Content)
	require.Len(t, e.Attachments, 1)
	assert.Equal(t, "image", e.Attachments[0].Type)
	assert.Equal(t, "media-echo-img", e.Attachments[0].Metadata["media_id"])
	assert.Equal(t, "true", e.Metadata["is_echo"])
}

// ---------------------------------------------------------------------------
// ExtractTemplateStatusUpdates: skips non-matching fields
// ---------------------------------------------------------------------------

func TestExtractTemplateStatusUpdates_SkipsOtherFields(t *testing.T) {
	proc := testProcessor()
	payload := buildFieldPayload("messages", WebhookChangeValue{
		Event:               "APPROVED",
		MessageTemplateID:   99,
		MessageTemplateName: "should_not_match",
	})

	events := proc.ExtractTemplateStatusUpdates(payload)
	assert.Empty(t, events)
}

// ---------------------------------------------------------------------------
// Multiple entries / changes
// ---------------------------------------------------------------------------

func TestExtractMessages_MultipleMessages(t *testing.T) {
	proc := testProcessor()
	payload := &WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []WebhookEntry{
			{
				ID: "E1",
				Changes: []WebhookChange{
					{
						Field: "messages",
						Value: WebhookChangeValue{
							MessagingProduct: "whatsapp",
							Metadata:         WebhookMetadata{PhoneNumberID: "pn1", DisplayPhoneNumber: "+1"},
							Contacts:         defaultContacts(),
							Messages: []IncomingMessage{
								{ID: "m1", From: "5511999998888", Timestamp: "1700000000", Type: MessageTypeText, Text: &TextContent{Body: "first"}},
								{ID: "m2", From: "5511999998888", Timestamp: "1700000001", Type: MessageTypeText, Text: &TextContent{Body: "second"}},
							},
						},
					},
				},
			},
		},
	}

	msgs := proc.ExtractMessages(payload)
	require.Len(t, msgs, 2)
	assert.Equal(t, "m1", msgs[0].ExternalID)
	assert.Equal(t, "m2", msgs[1].ExternalID)
	assert.Equal(t, "first", msgs[0].Content)
	assert.Equal(t, "second", msgs[1].Content)
}

// ---------------------------------------------------------------------------
// Status timestamp parsing - invalid
// ---------------------------------------------------------------------------

func TestExtractStatuses_InvalidTimestamp(t *testing.T) {
	proc := testProcessor()
	before := time.Now().Add(-2 * time.Second)
	payload := buildStatusPayload(StatusUpdate{
		ID:          "s-bad-ts",
		RecipientID: "5511",
		Status:      StatusSent,
		Timestamp:   "not-a-ts",
	})

	statuses := proc.ExtractStatuses(payload)
	require.Len(t, statuses, 1)
	after := time.Now().Add(2 * time.Second)
	assert.True(t, statuses[0].Timestamp.After(before))
	assert.True(t, statuses[0].Timestamp.Before(after))
}

// ---------------------------------------------------------------------------
// Suppress unused import warning for fmt (used via helpers)
// ---------------------------------------------------------------------------

var _ = fmt.Sprintf
