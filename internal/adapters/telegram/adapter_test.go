package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter()
	require.NotNil(t, adapter)

	// Check channel type
	assert.Equal(t, plugin.ChannelTypeTelegram, adapter.GetChannelType())

	// Check info
	info := adapter.GetChannelInfo()
	require.NotNil(t, info)
	assert.Equal(t, "Telegram", info.Name)
	assert.Equal(t, "1.0.0", info.Version)

	// Check capabilities
	caps := adapter.GetCapabilities()
	require.NotNil(t, caps)

	assert.Equal(t, 4096, caps.MaxMessageLength)
	assert.Equal(t, int64(50*1024*1024), caps.MaxMediaSize)
	assert.Equal(t, 1, caps.MaxAttachments)
	assert.True(t, caps.SupportsMedia)
	assert.True(t, caps.SupportsLocation)
	assert.False(t, caps.SupportsTemplates)
	assert.True(t, caps.SupportsInteractive)
	assert.False(t, caps.SupportsReadReceipts)
	assert.True(t, caps.SupportsTypingIndicator)
	assert.False(t, caps.SupportsReactions)
	assert.True(t, caps.SupportsReplies)
	assert.False(t, caps.SupportsForwarding)

	// Check supported content types
	expectedTypes := []plugin.ContentType{
		plugin.ContentTypeText,
		plugin.ContentTypeImage,
		plugin.ContentTypeVideo,
		plugin.ContentTypeAudio,
		plugin.ContentTypeDocument,
		plugin.ContentTypeLocation,
		plugin.ContentTypeContact,
	}
	assert.Equal(t, expectedTypes, caps.SupportedContentTypes)

	// Not connected initially
	assert.False(t, adapter.IsConnected())
}

func TestInitialize_ValidConfig(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{
		"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
		"bot_name":  "testbot",
	})
	assert.NoError(t, err)

	cfg := adapter.GetConfig()
	assert.Equal(t, "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", cfg.BotToken)
	assert.Equal(t, "testbot", cfg.BotName)
}

func TestInitialize_MissingBotToken(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{
		"bot_name": "testbot",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bot_token is required")
}

func TestInitialize_EmptyBotToken(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{
		"bot_token": "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bot_token is required")
}

func TestSendMessage_NotConnected(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()

	result, err := adapter.SendMessage(ctx, &plugin.OutboundMessage{
		RecipientID: "12345",
		ContentType: plugin.ContentTypeText,
		Content:     "hello",
		Metadata:    map[string]string{},
	})

	assert.NoError(t, err) // Returns result, not error
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, plugin.MessageStatusFailed, result.Status)
	assert.Equal(t, "adapter not connected", result.Error)
}

func TestValidateWebhook_AlwaysTrue(t *testing.T) {
	adapter := NewAdapter()

	// ValidateWebhook always returns true
	assert.True(t, adapter.ValidateWebhook(nil, nil))
	assert.True(t, adapter.ValidateWebhook(map[string]string{"x": "y"}, []byte("body")))
	assert.True(t, adapter.ValidateWebhook(map[string]string{}, []byte{}))
}

func TestHandleWebhook_NoHandler(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()

	err := adapter.HandleWebhook(ctx, []byte("{}"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no message handler configured")
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()

	// Set a handler so we get past that check
	adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		return nil
	})

	err := adapter.HandleWebhook(ctx, []byte("not json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse webhook")
}

func TestHandleWebhook_NonPrivateChat(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()

	handlerCalled := false
	adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		handlerCalled = true
		return nil
	})

	// Group chat message - should be ignored (ExtractIncomingMessage returns nil)
	payload := map[string]interface{}{
		"update_id": 1,
		"message": map[string]interface{}{
			"message_id": 1,
			"from":       map[string]interface{}{"id": 1, "first_name": "U"},
			"chat":       map[string]interface{}{"id": 10, "type": "group"},
			"date":       time.Now().Unix(),
			"text":       "group msg",
		},
	}
	body, _ := json.Marshal(payload)

	err := adapter.HandleWebhook(ctx, body)
	assert.NoError(t, err) // Returns nil, not an error
	assert.False(t, handlerCalled)
}

func TestHandleWebhook_PrivateMessage(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()

	var receivedMsg *plugin.InboundMessage
	adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		receivedMsg = msg
		return nil
	})

	payload := map[string]interface{}{
		"update_id": 1,
		"message": map[string]interface{}{
			"message_id": 42,
			"from": map[string]interface{}{
				"id":         100,
				"first_name": "Test",
				"last_name":  "User",
				"username":   "testuser",
			},
			"chat": map[string]interface{}{
				"id":   200,
				"type": "private",
			},
			"date": time.Now().Unix(),
			"text": "hello from webhook",
		},
	}
	body, _ := json.Marshal(payload)

	err := adapter.HandleWebhook(ctx, body)
	assert.NoError(t, err)
	require.NotNil(t, receivedMsg)
	assert.Equal(t, "42", receivedMsg.ExternalID)
	assert.Equal(t, "200", receivedMsg.SenderID)
	assert.Equal(t, "hello from webhook", receivedMsg.Content)
	assert.Equal(t, plugin.ContentTypeText, receivedMsg.ContentType)
	assert.Equal(t, "Test User", receivedMsg.SenderName)
	assert.Equal(t, "testuser", receivedMsg.Metadata["username"])
	assert.Equal(t, "100", receivedMsg.Metadata["from_user_id"])
}

func TestHandleWebhook_HandlerReturnsError(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()

	adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		return fmt.Errorf("handler error")
	})

	payload := map[string]interface{}{
		"update_id": 1,
		"message": map[string]interface{}{
			"message_id": 1,
			"from":       map[string]interface{}{"id": 1, "first_name": "U"},
			"chat":       map[string]interface{}{"id": 2, "type": "private"},
			"date":       time.Now().Unix(),
			"text":       "hello",
		},
	}
	body, _ := json.Marshal(payload)

	err := adapter.HandleWebhook(ctx, body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
}

func TestToInboundMessage_TextMessage(t *testing.T) {
	adapter := NewAdapter()
	now := time.Now()

	incoming := &IncomingMessage{
		MessageID:     100,
		ChatID:        200,
		FromUserID:    300,
		FromUsername:   "jdoe",
		FromFirstName: "John",
		FromLastName:  "Doe",
		Text:          "test message",
		MessageType:   MessageTypeText,
		Timestamp:     now,
	}

	inbound := adapter.toInboundMessage(incoming)
	require.NotNil(t, inbound)
	assert.NotEmpty(t, inbound.ID) // UUID
	assert.Equal(t, "100", inbound.ExternalID)
	assert.Equal(t, "200", inbound.SenderID)
	assert.Equal(t, "John Doe", inbound.SenderName)
	assert.Equal(t, "test message", inbound.Content)
	assert.Equal(t, plugin.ContentTypeText, inbound.ContentType)
	assert.Equal(t, now, inbound.Timestamp)
	assert.Equal(t, "300", inbound.Metadata["from_user_id"])
	assert.Equal(t, "jdoe", inbound.Metadata["username"])
	assert.Equal(t, "John", inbound.Metadata["first_name"])
	assert.Equal(t, "Doe", inbound.Metadata["last_name"])
	assert.Equal(t, "200", inbound.Metadata["chat_id"])
}

func TestToInboundMessage_FirstNameOnly(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "Alice",
		FromLastName:  "",
		Text:          "hi",
		MessageType:   MessageTypeText,
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, "Alice", inbound.SenderName) // No last name, no space appended
}

func TestToInboundMessage_PhotoMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		MessageType:   MessageTypePhoto,
		MediaFileID:   "photo-file-id",
		Caption:       "a nice photo",
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeImage, inbound.ContentType)
	assert.Equal(t, "a nice photo", inbound.Content)
	require.Len(t, inbound.Attachments, 1)
	assert.Equal(t, "image", inbound.Attachments[0].Type)
	assert.Equal(t, "photo-file-id", inbound.Attachments[0].Metadata["file_id"])
}

func TestToInboundMessage_VideoMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		MessageType:   MessageTypeVideo,
		MediaFileID:   "vid-id",
		MediaMimeType: "video/mp4",
		Caption:       "video caption",
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeVideo, inbound.ContentType)
	assert.Equal(t, "video caption", inbound.Content)
	require.Len(t, inbound.Attachments, 1)
	assert.Equal(t, "video", inbound.Attachments[0].Type)
	assert.Equal(t, "video/mp4", inbound.Attachments[0].MimeType)
}

func TestToInboundMessage_AudioMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:      1,
		ChatID:         2,
		FromUserID:     3,
		FromFirstName:  "U",
		MessageType:    MessageTypeAudio,
		MediaFileID:    "aud-id",
		MediaMimeType:  "audio/mpeg",
		MediaFileName:  "song.mp3",
		MediaFileSize:  300000,
		Caption:        "audio caption",
		Timestamp:      time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeAudio, inbound.ContentType)
	assert.Equal(t, "audio caption", inbound.Content)
	require.Len(t, inbound.Attachments, 1)
	att := inbound.Attachments[0]
	assert.Equal(t, "audio", att.Type)
	assert.Equal(t, "audio/mpeg", att.MimeType)
	assert.Equal(t, "song.mp3", att.Filename)
	assert.Equal(t, int64(300000), att.SizeBytes)
}

func TestToInboundMessage_VoiceMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		MessageType:   MessageTypeVoice,
		MediaFileID:   "voice-id",
		MediaMimeType: "audio/ogg",
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeAudio, inbound.ContentType)
	require.Len(t, inbound.Attachments, 1)
	assert.Equal(t, "voice", inbound.Attachments[0].Type)
}

func TestToInboundMessage_DocumentMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:      1,
		ChatID:         2,
		FromUserID:     3,
		FromFirstName:  "U",
		MessageType:    MessageTypeDocument,
		MediaFileID:    "doc-id",
		MediaMimeType:  "application/pdf",
		MediaFileName:  "report.pdf",
		MediaFileSize:  100000,
		Caption:        "doc caption",
		Timestamp:      time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeDocument, inbound.ContentType)
	assert.Equal(t, "doc caption", inbound.Content)
	require.Len(t, inbound.Attachments, 1)
	att := inbound.Attachments[0]
	assert.Equal(t, "document", att.Type)
	assert.Equal(t, "application/pdf", att.MimeType)
	assert.Equal(t, "report.pdf", att.Filename)
}

func TestToInboundMessage_LocationMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		MessageType:   MessageTypeLocation,
		Location: &Location{
			Latitude:  -23.5505,
			Longitude: -46.6333,
		},
		Timestamp: time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeLocation, inbound.ContentType)
	assert.Contains(t, inbound.Content, "-23.550500")
	assert.Contains(t, inbound.Content, "-46.633300")
	assert.NotEmpty(t, inbound.Metadata["latitude"])
	assert.NotEmpty(t, inbound.Metadata["longitude"])
}

func TestToInboundMessage_ContactMessage(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		MessageType:   MessageTypeContact,
		Contact: &Contact{
			PhoneNumber: "+5511999999999",
			FirstName:   "Jane",
			LastName:    "Smith",
		},
		Timestamp: time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeContact, inbound.ContentType)
	assert.Equal(t, "+5511999999999", inbound.Content)
	assert.Equal(t, "+5511999999999", inbound.Metadata["contact_phone"])
	assert.Equal(t, "Jane", inbound.Metadata["contact_first_name"])
	assert.Equal(t, "Smith", inbound.Metadata["contact_last_name"])
}

func TestToInboundMessage_EditedFlag(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		Text:          "edited",
		MessageType:   MessageTypeText,
		IsEdited:      true,
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, "true", inbound.Metadata["is_edited"])
}

func TestToInboundMessage_ReplyToID(t *testing.T) {
	adapter := NewAdapter()
	replyID := int64(42)
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		Text:          "replying",
		MessageType:   MessageTypeText,
		ReplyToMsgID:  &replyID,
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, "42", inbound.Metadata["reply_to_id"])
}

func TestToInboundMessage_DefaultMessageType(t *testing.T) {
	adapter := NewAdapter()
	incoming := &IncomingMessage{
		MessageID:     1,
		ChatID:        2,
		FromUserID:    3,
		FromFirstName: "U",
		MessageType:   MessageType("unknown_type"),
		Timestamp:     time.Now(),
	}

	inbound := adapter.toInboundMessage(incoming)
	assert.Equal(t, plugin.ContentTypeText, inbound.ContentType)
}

func TestGetConnectionStatus_Disconnected(t *testing.T) {
	adapter := NewAdapter()

	status := adapter.GetConnectionStatus()
	require.NotNil(t, status)
	assert.False(t, status.Connected)
	assert.Equal(t, "disconnected", status.Status)
	assert.NotNil(t, status.Metadata)
}

func TestGetMediaURL_FromAttachments(t *testing.T) {
	adapter := NewAdapter()
	msg := &plugin.OutboundMessage{
		Attachments: []*plugin.Attachment{
			{URL: "https://example.com/photo.jpg"},
		},
		Metadata: map[string]string{
			"media_url": "https://example.com/fallback.jpg",
		},
	}

	url := adapter.getMediaURL(msg)
	assert.Equal(t, "https://example.com/photo.jpg", url)
}

func TestGetMediaURL_FromMetadata(t *testing.T) {
	adapter := NewAdapter()
	msg := &plugin.OutboundMessage{
		Attachments: nil,
		Metadata: map[string]string{
			"media_url": "https://example.com/fallback.jpg",
		},
	}

	url := adapter.getMediaURL(msg)
	assert.Equal(t, "https://example.com/fallback.jpg", url)
}

func TestGetMediaURL_Empty(t *testing.T) {
	adapter := NewAdapter()
	msg := &plugin.OutboundMessage{
		Metadata: map[string]string{},
	}

	url := adapter.getMediaURL(msg)
	assert.Empty(t, url)
}

func TestBuildKeyboardFromMetadata_NoQuickReplies(t *testing.T) {
	adapter := NewAdapter()
	msg := &plugin.OutboundMessage{
		Metadata: map[string]string{},
	}

	kb := adapter.buildKeyboardFromMetadata(msg)
	assert.Nil(t, kb)
}

func TestBuildKeyboardFromMetadata_EmptyQuickReplies(t *testing.T) {
	adapter := NewAdapter()
	msg := &plugin.OutboundMessage{
		Metadata: map[string]string{
			"quick_replies": "",
		},
	}

	kb := adapter.buildKeyboardFromMetadata(msg)
	assert.Nil(t, kb)
}

func TestBuildKeyboardFromMetadata_WithQuickReplies(t *testing.T) {
	adapter := NewAdapter()
	// Currently buildKeyboardFromMetadata returns nil even with data (placeholder impl)
	msg := &plugin.OutboundMessage{
		Metadata: map[string]string{
			"quick_replies": "text1|data1,text2|data2",
		},
	}

	kb := adapter.buildKeyboardFromMetadata(msg)
	// Current implementation returns nil (placeholder)
	assert.Nil(t, kb)
}

func TestGetWebhookPath(t *testing.T) {
	adapter := NewAdapter()
	assert.Equal(t, "/api/v1/webhooks/telegram", adapter.GetWebhookPath())
}

func TestSetMessageHandler(t *testing.T) {
	adapter := NewAdapter()
	called := false
	handler := func(ctx context.Context, msg *plugin.InboundMessage) error {
		called = true
		return nil
	}
	adapter.SetMessageHandler(handler)

	// Verify handler is set by triggering a webhook
	payload := map[string]interface{}{
		"update_id": 1,
		"message": map[string]interface{}{
			"message_id": 1,
			"from":       map[string]interface{}{"id": 1, "first_name": "U"},
			"chat":       map[string]interface{}{"id": 2, "type": "private"},
			"date":       time.Now().Unix(),
			"text":       "test",
		},
	}
	body, _ := json.Marshal(payload)
	err := adapter.HandleWebhook(context.Background(), body)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestGetClient_NilWhenNotConnected(t *testing.T) {
	adapter := NewAdapter()
	assert.Nil(t, adapter.GetClient())
}
