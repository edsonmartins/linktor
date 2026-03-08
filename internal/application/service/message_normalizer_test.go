package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

func TestMessageNormalizer_NormalizeInbound(t *testing.T) {
	normalizer := NewMessageNormalizer()

	t.Run("basic text message", func(t *testing.T) {
		ts := time.Now()
		msg := &nats.InboundMessage{
			ID:          "msg-1",
			TenantID:    "tenant-1",
			ChannelID:   "ch-1",
			ChannelType: "whatsapp",
			ExternalID:  "ext-1",
			ContentType: "text",
			Content:     "Hello World",
			Metadata:    map[string]string{"phone": "5511999"},
			Timestamp:   ts,
		}

		result := normalizer.NormalizeInbound(msg)

		assert.Equal(t, "msg-1", result.ID)
		assert.Equal(t, "tenant-1", result.TenantID)
		assert.Equal(t, "ch-1", result.ChannelID)
		assert.Equal(t, "whatsapp", result.ChannelType)
		assert.Equal(t, "ext-1", result.ExternalID)
		assert.Equal(t, entity.ContentTypeText, result.ContentType)
		assert.Equal(t, "Hello World", result.Content)
		assert.Equal(t, entity.SenderTypeContact, result.SenderType)
		assert.Equal(t, ts, result.Timestamp)
		assert.NotNil(t, result.Metadata)
	})

	t.Run("nil metadata initialized", func(t *testing.T) {
		msg := &nats.InboundMessage{
			ContentType: "text",
			Content:     "test",
		}

		result := normalizer.NormalizeInbound(msg)
		assert.NotNil(t, result.Metadata)
	})

	t.Run("with attachments", func(t *testing.T) {
		msg := &nats.InboundMessage{
			ContentType: "image",
			Content:     "caption",
			Attachments: []nats.AttachmentData{
				{
					Type:     "image",
					URL:      "https://example.com/image.jpg",
					Filename: "photo.jpg",
					MimeType: "image/jpeg",
				},
			},
		}

		result := normalizer.NormalizeInbound(msg)

		assert.Equal(t, entity.ContentTypeImage, result.ContentType)
		require.Len(t, result.Attachments, 1)
		assert.Equal(t, "image", result.Attachments[0].Type)
		assert.Equal(t, "https://example.com/image.jpg", result.Attachments[0].URL)
		assert.Equal(t, "photo.jpg", result.Attachments[0].Filename)
	})

	t.Run("content trimming", func(t *testing.T) {
		msg := &nats.InboundMessage{
			ContentType: "text",
			Content:     "  hello  ",
		}

		result := normalizer.NormalizeInbound(msg)
		assert.Equal(t, "hello", result.Content)
	})

	t.Run("line break normalization", func(t *testing.T) {
		msg := &nats.InboundMessage{
			ContentType: "text",
			Content:     "line1\r\nline2\rline3",
		}

		result := normalizer.NormalizeInbound(msg)
		assert.Equal(t, "line1\nline2\nline3", result.Content)
	})
}

func TestMessageNormalizer_NormalizeContentType(t *testing.T) {
	normalizer := NewMessageNormalizer()

	tests := []struct {
		input    string
		expected entity.ContentType
	}{
		{"text", entity.ContentTypeText},
		{"text/plain", entity.ContentTypeText},
		{"plain", entity.ContentTypeText},
		{"image", entity.ContentTypeImage},
		{"image/jpeg", entity.ContentTypeImage},
		{"image/png", entity.ContentTypeImage},
		{"image/gif", entity.ContentTypeImage},
		{"image/webp", entity.ContentTypeImage},
		{"video", entity.ContentTypeVideo},
		{"video/mp4", entity.ContentTypeVideo},
		{"video/3gpp", entity.ContentTypeVideo},
		{"audio", entity.ContentTypeAudio},
		{"audio/ogg", entity.ContentTypeAudio},
		{"audio/mpeg", entity.ContentTypeAudio},
		{"audio/mp3", entity.ContentTypeAudio},
		{"ptt", entity.ContentTypeAudio},
		{"voice", entity.ContentTypeAudio},
		{"document", entity.ContentTypeDocument},
		{"file", entity.ContentTypeDocument},
		{"application/pdf", entity.ContentTypeDocument},
		{"location", entity.ContentTypeLocation},
		{"geo", entity.ContentTypeLocation},
		{"contact", entity.ContentTypeContact},
		{"vcard", entity.ContentTypeContact},
		{"contacts", entity.ContentTypeContact},
		{"template", entity.ContentTypeTemplate},
		{"hsm", entity.ContentTypeTemplate},
		{"interactive", entity.ContentTypeInteractive},
		{"button", entity.ContentTypeInteractive},
		{"list", entity.ContentTypeInteractive},
		{"buttons", entity.ContentTypeInteractive},
		{"list_reply", entity.ContentTypeInteractive},
		// Case insensitive
		{"TEXT", entity.ContentTypeText},
		{"Image", entity.ContentTypeImage},
		// Unknown defaults to text
		{"unknown", entity.ContentTypeText},
		{"", entity.ContentTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizer.normalizeContentType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageNormalizer_ToEntity(t *testing.T) {
	normalizer := NewMessageNormalizer()

	t.Run("converts normalized to entity", func(t *testing.T) {
		ts := time.Now()
		normalized := &NormalizedMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			SenderType:     entity.SenderTypeContact,
			SenderID:       "sender-1",
			ContentType:    entity.ContentTypeText,
			Content:        "Hello",
			Metadata:       map[string]string{"key": "val"},
			ExternalID:     "ext-1",
			Timestamp:      ts,
			Attachments: []*entity.MessageAttachment{
				{ID: "att-1", Type: "image", URL: "https://example.com/img.jpg"},
			},
		}

		message := normalizer.ToEntity(normalized)

		assert.Equal(t, "msg-1", message.ID)
		assert.Equal(t, "conv-1", message.ConversationID)
		assert.Equal(t, entity.SenderTypeContact, message.SenderType)
		assert.Equal(t, entity.ContentTypeText, message.ContentType)
		assert.Equal(t, "Hello", message.Content)
		assert.Equal(t, entity.MessageStatusPending, message.Status)
		assert.Equal(t, "ext-1", message.ExternalID)
		assert.Equal(t, ts, message.CreatedAt)
		require.Len(t, message.Attachments, 1)
	})

	t.Run("nil metadata initialized", func(t *testing.T) {
		normalized := &NormalizedMessage{}
		message := normalizer.ToEntity(normalized)
		assert.NotNil(t, message.Metadata)
	})
}

func TestMessageNormalizer_ToOutbound(t *testing.T) {
	normalizer := NewMessageNormalizer()

	msg := &entity.Message{
		ID:             "msg-1",
		ConversationID: "conv-1",
		ContentType:    entity.ContentTypeText,
		Content:        "Hello",
		Metadata:       map[string]string{"key": "val"},
		Attachments: []*entity.MessageAttachment{
			{
				Type:     "image",
				URL:      "https://example.com/img.jpg",
				Filename: "photo.jpg",
				MimeType: "image/jpeg",
			},
		},
	}

	outbound := normalizer.ToOutbound(msg, "whatsapp", "ch-1", "tenant-1", "contact-1", "5511999")

	assert.Equal(t, "msg-1", outbound.ID)
	assert.Equal(t, "tenant-1", outbound.TenantID)
	assert.Equal(t, "ch-1", outbound.ChannelID)
	assert.Equal(t, "whatsapp", outbound.ChannelType)
	assert.Equal(t, "conv-1", outbound.ConversationID)
	assert.Equal(t, "contact-1", outbound.ContactID)
	assert.Equal(t, "5511999", outbound.RecipientID)
	assert.Equal(t, "text", outbound.ContentType)
	assert.Equal(t, "Hello", outbound.Content)
	require.Len(t, outbound.Attachments, 1)
	assert.Equal(t, "image", outbound.Attachments[0].Type)
	assert.Equal(t, "https://example.com/img.jpg", outbound.Attachments[0].URL)
}

func TestMessageNormalizer_DenormalizeForChannel(t *testing.T) {
	normalizer := NewMessageNormalizer()

	t.Run("whatsapp text", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeText,
			Content:     "Hello",
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp")
		assert.Equal(t, "text", result["type"])
		text := result["text"].(map[string]interface{})
		assert.Equal(t, "Hello", text["body"])
	})

	t.Run("whatsapp_official text", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeText,
			Content:     "Hi",
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp_official")
		assert.Equal(t, "text", result["type"])
	})

	t.Run("whatsapp image", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeImage,
			Content:     "caption",
			Attachments: []*entity.MessageAttachment{
				{URL: "https://example.com/img.jpg"},
			},
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp")
		img := result["image"].(map[string]interface{})
		assert.Equal(t, "https://example.com/img.jpg", img["link"])
		assert.Equal(t, "caption", img["caption"])
	})

	t.Run("whatsapp document", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeDocument,
			Content:     "doc caption",
			Attachments: []*entity.MessageAttachment{
				{URL: "https://example.com/doc.pdf", Filename: "report.pdf"},
			},
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp")
		doc := result["document"].(map[string]interface{})
		assert.Equal(t, "report.pdf", doc["filename"])
	})

	t.Run("whatsapp audio", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeAudio,
			Attachments: []*entity.MessageAttachment{
				{URL: "https://example.com/audio.ogg"},
			},
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp")
		audio := result["audio"].(map[string]interface{})
		assert.Equal(t, "https://example.com/audio.ogg", audio["link"])
	})

	t.Run("whatsapp video", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeVideo,
			Content:     "video caption",
			Attachments: []*entity.MessageAttachment{
				{URL: "https://example.com/video.mp4"},
			},
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp")
		video := result["video"].(map[string]interface{})
		assert.Equal(t, "https://example.com/video.mp4", video["link"])
		assert.Equal(t, "video caption", video["caption"])
	})

	t.Run("whatsapp location", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeLocation,
			Content:     "123 Main St",
			Metadata: map[string]string{
				"latitude":  "-23.5505",
				"longitude": "-46.6333",
				"name":      "Office",
			},
		}
		result := normalizer.DenormalizeForChannel(msg, "whatsapp")
		loc := result["location"].(map[string]interface{})
		assert.Equal(t, "-23.5505", loc["latitude"])
		assert.Equal(t, "-46.6333", loc["longitude"])
		assert.Equal(t, "Office", loc["name"])
		assert.Equal(t, "123 Main St", loc["address"])
	})

	t.Run("telegram text", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeText,
			Content:     "Hello Telegram",
		}
		result := normalizer.DenormalizeForChannel(msg, "telegram")
		assert.Equal(t, "Hello Telegram", result["text"])
		assert.Equal(t, "HTML", result["parse_mode"])
	})

	t.Run("telegram image", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeImage,
			Content:     "pic",
			Attachments: []*entity.MessageAttachment{
				{URL: "https://example.com/img.jpg"},
			},
		}
		result := normalizer.DenormalizeForChannel(msg, "telegram")
		assert.Equal(t, "https://example.com/img.jpg", result["photo"])
		assert.Equal(t, "pic", result["caption"])
	})

	t.Run("webchat format", func(t *testing.T) {
		msg := &entity.Message{
			ID:          "msg-1",
			ContentType: entity.ContentTypeText,
			Content:     "Hello",
			SenderType:  entity.SenderTypeBot,
			SenderID:    "bot-1",
			Status:      entity.MessageStatusSent,
			Metadata:    map[string]string{},
			Attachments: []*entity.MessageAttachment{},
			CreatedAt:   time.Now(),
		}
		result := normalizer.DenormalizeForChannel(msg, "webchat")
		assert.Equal(t, "msg-1", result["id"])
		assert.Equal(t, "text", result["contentType"])
		assert.Equal(t, "Hello", result["content"])
		assert.Equal(t, "bot", result["senderType"])
	})

	t.Run("sms truncation", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 200; i++ {
			longContent += "x"
		}
		msg := &entity.Message{
			ContentType: entity.ContentTypeText,
			Content:     longContent,
		}
		result := normalizer.DenormalizeForChannel(msg, "sms")
		body := result["body"].(string)
		assert.Len(t, body, 160)
		assert.True(t, body[len(body)-3:] == "...")
	})

	t.Run("sms short content not truncated", func(t *testing.T) {
		msg := &entity.Message{
			ContentType: entity.ContentTypeText,
			Content:     "Short",
		}
		result := normalizer.DenormalizeForChannel(msg, "sms")
		assert.Equal(t, "Short", result["body"])
	})

	t.Run("generic format", func(t *testing.T) {
		msg := &entity.Message{
			ID:          "msg-1",
			ContentType: entity.ContentTypeText,
			Content:     "Hello",
			Metadata:    map[string]string{"k": "v"},
		}
		result := normalizer.DenormalizeForChannel(msg, "unknown_channel")
		assert.Equal(t, "msg-1", result["id"])
		assert.Equal(t, "text", result["contentType"])
		assert.Equal(t, "Hello", result["content"])
	})
}

func TestMessageNormalizer_NormalizeAttachments(t *testing.T) {
	normalizer := NewMessageNormalizer()

	t.Run("empty attachments", func(t *testing.T) {
		result := normalizer.normalizeAttachments(nil)
		assert.Nil(t, result)
	})

	t.Run("converts attachments", func(t *testing.T) {
		attachments := []nats.AttachmentData{
			{
				Type:         "image",
				URL:          "https://example.com/img.jpg",
				Filename:     "photo.jpg",
				MimeType:     "image/jpeg",
				SizeBytes:    12345,
				ThumbnailURL: "https://example.com/thumb.jpg",
				Metadata:     map[string]string{"key": "val"},
			},
			{
				Type: "document",
				URL:  "https://example.com/doc.pdf",
			},
		}

		result := normalizer.normalizeAttachments(attachments)

		require.Len(t, result, 2)

		assert.Equal(t, "image", result[0].Type)
		assert.Equal(t, "https://example.com/img.jpg", result[0].URL)
		assert.Equal(t, "photo.jpg", result[0].Filename)
		assert.Equal(t, "image/jpeg", result[0].MimeType)
		assert.Equal(t, int64(12345), result[0].SizeBytes)
		assert.Equal(t, "https://example.com/thumb.jpg", result[0].ThumbnailURL)
		assert.NotEmpty(t, result[0].ID)

		assert.Equal(t, "document", result[1].Type)
		assert.NotNil(t, result[1].Metadata) // nil metadata initialized
	})
}
