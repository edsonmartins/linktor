package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/testutil"
)

// setupSendMessageTest creates common test fixtures for SendMessageUseCase tests.
func setupSendMessageTest() (
	*testutil.MockMessageRepository,
	*testutil.MockConversationRepository,
	*testutil.MockChannelRepository,
	*testutil.MockContactRepository,
	*testutil.MockProducer,
	*SendMessageUseCase,
) {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	chRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	producer := testutil.NewMockProducer()

	uc := NewSendMessageUseCase(msgRepo, convRepo, chRepo, contactRepo, producer)
	return msgRepo, convRepo, chRepo, contactRepo, producer, uc
}

// activeWhatsAppChannel returns an active WhatsApp channel for testing.
func activeWhatsAppChannel(tenantID, channelID string) *entity.Channel {
	return &entity.Channel{
		ID:               channelID,
		TenantID:         tenantID,
		Type:             entity.ChannelTypeWhatsApp,
		Name:             "Test WhatsApp",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
	}
}

// activeWebChatChannel returns an active WebChat channel for testing.
func activeWebChatChannel(tenantID, channelID string) *entity.Channel {
	return &entity.Channel{
		ID:               channelID,
		TenantID:         tenantID,
		Type:             entity.ChannelTypeWebChat,
		Name:             "Test WebChat",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
	}
}

func TestSendMessageUseCase_Validation(t *testing.T) {
	t.Run("empty ConversationID returns validation error", func(t *testing.T) {
		_, _, _, _, _, uc := setupSendMessageTest()

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "",
			Content:        "hello",
		})
		require.Error(t, err)
		appErr, ok := err.(*errors.AppError)
		require.True(t, ok)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
	})

	t.Run("empty Content and no Attachments returns validation error", func(t *testing.T) {
		_, _, _, _, _, uc := setupSendMessageTest()

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			Content:        "",
			Attachments:    nil,
		})
		require.Error(t, err)
		appErr, ok := err.(*errors.AppError)
		require.True(t, ok)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
	})

	t.Run("Content present is valid", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeUser,
			ContentType:    entity.ContentTypeText,
			Content:        "hello",
		})
		assert.NoError(t, err)
	})

	t.Run("empty Content but Attachments present is valid", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeUser,
			ContentType:    entity.ContentTypeImage,
			Content:        "",
			Attachments: []*AttachmentInput{
				{Type: "image", URL: "https://example.com/img.png", Filename: "img.png", MimeType: "image/png", SizeBytes: 1024},
			},
		})
		assert.NoError(t, err)
	})
}

func TestSendMessageUseCase_HappyPath(t *testing.T) {
	msgRepo, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

	conv := &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	convRepo.Conversations["conv1"] = conv
	chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Name: "John", Phone: "5511999"}
	contactRepo.Identities["c1"] = []*entity.ContactIdentity{
		{ID: "id1", ContactID: "c1", ChannelType: "whatsapp", Identifier: "5511999888"},
	}

	input := &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderID:       "user1",
		SenderType:     entity.SenderTypeUser,
		ContentType:    entity.ContentTypeText,
		Content:        "Hello, world!",
	}

	output, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotNil(t, output.Message)
	require.NotNil(t, output.Conversation)

	// Verify message saved
	assert.Len(t, msgRepo.Messages, 1)
	msg := output.Message
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "conv1", msg.ConversationID)
	assert.Equal(t, entity.SenderTypeUser, msg.SenderType)
	assert.Equal(t, "user1", msg.SenderID)
	assert.Equal(t, entity.ContentTypeText, msg.ContentType)
	assert.Equal(t, "Hello, world!", msg.Content)
	assert.Equal(t, entity.MessageStatusPending, msg.Status)

	// Verify outbound published
	require.Len(t, producer.OutboundMessages, 1)
	outbound := producer.OutboundMessages[0]
	assert.Equal(t, msg.ID, outbound.ID)
	assert.Equal(t, "t1", outbound.TenantID)
	assert.Equal(t, "ch1", outbound.ChannelID)
	assert.Equal(t, "whatsapp", outbound.ChannelType)
	assert.Equal(t, "conv1", outbound.ConversationID)
	assert.Equal(t, "c1", outbound.ContactID)
	assert.Equal(t, "5511999888", outbound.RecipientID)
	assert.Equal(t, "text", outbound.ContentType)
	assert.Equal(t, "Hello, world!", outbound.Content)
}

func TestSendMessageUseCase_TenantMismatch(t *testing.T) {
	_, convRepo, _, _, _, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}

	_, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t2",
		ConversationID: "conv1",
		Content:        "hello",
	})
	require.Error(t, err)
	appErr, ok := err.(*errors.AppError)
	require.True(t, ok)
	assert.Equal(t, errors.ErrCodeForbidden, appErr.Code)
}

func TestSendMessageUseCase_ChannelNotActive(t *testing.T) {
	t.Run("channel disabled", func(t *testing.T) {
		_, convRepo, chRepo, _, _, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		}
		chRepo.Channels["ch1"] = &entity.Channel{
			ID: "ch1", TenantID: "t1", Type: entity.ChannelTypeWhatsApp,
			Enabled: false, ConnectionStatus: entity.ConnectionStatusConnected,
		}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID: "t1", ConversationID: "conv1", Content: "hello",
		})
		require.Error(t, err)
		appErr, ok := err.(*errors.AppError)
		require.True(t, ok)
		assert.Equal(t, errors.ErrCodeChannelDisconnected, appErr.Code)
	})

	t.Run("channel disconnected", func(t *testing.T) {
		_, convRepo, chRepo, _, _, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		}
		chRepo.Channels["ch1"] = &entity.Channel{
			ID: "ch1", TenantID: "t1", Type: entity.ChannelTypeWhatsApp,
			Enabled: true, ConnectionStatus: entity.ConnectionStatusDisconnected,
		}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID: "t1", ConversationID: "conv1", Content: "hello",
		})
		require.Error(t, err)
		appErr, ok := err.(*errors.AppError)
		require.True(t, ok)
		assert.Equal(t, errors.ErrCodeChannelDisconnected, appErr.Code)
	})
}

func TestSendMessageUseCase_RecipientResolution(t *testing.T) {
	t.Run("by identity", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{
			ID: "c1", TenantID: "t1", Phone: "5511888", Email: "test@test.com",
		}
		contactRepo.Identities["c1"] = []*entity.ContactIdentity{
			{ID: "id1", ContactID: "c1", ChannelType: "whatsapp", Identifier: "5511999"},
		}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID: "t1", ConversationID: "conv1", Content: "hi",
			SenderType: entity.SenderTypeUser, ContentType: entity.ContentTypeText,
		})
		require.NoError(t, err)
		require.Len(t, producer.OutboundMessages, 1)
		assert.Equal(t, "5511999", producer.OutboundMessages[0].RecipientID)
	})

	t.Run("fallback to phone", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{
			ID: "c1", TenantID: "t1", Phone: "5511888", Email: "test@test.com",
		}
		// No identities

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID: "t1", ConversationID: "conv1", Content: "hi",
			SenderType: entity.SenderTypeUser, ContentType: entity.ContentTypeText,
		})
		require.NoError(t, err)
		require.Len(t, producer.OutboundMessages, 1)
		assert.Equal(t, "5511888", producer.OutboundMessages[0].RecipientID)
	})

	t.Run("fallback to email", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{
			ID: "c1", TenantID: "t1", Email: "test@test.com",
		}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID: "t1", ConversationID: "conv1", Content: "hi",
			SenderType: entity.SenderTypeUser, ContentType: entity.ContentTypeText,
		})
		require.NoError(t, err)
		require.Len(t, producer.OutboundMessages, 1)
		assert.Equal(t, "test@test.com", producer.OutboundMessages[0].RecipientID)
	})

	t.Run("empty when no identity phone or email", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{
			ID: "c1", TenantID: "t1",
		}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID: "t1", ConversationID: "conv1", Content: "hi",
			SenderType: entity.SenderTypeUser, ContentType: entity.ContentTypeText,
		})
		require.NoError(t, err)
		require.Len(t, producer.OutboundMessages, 1)
		assert.Equal(t, "", producer.OutboundMessages[0].RecipientID)
	})
}

func TestSendMessageUseCase_QuickReplies_Buttons(t *testing.T) {
	msgRepo, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

	input := &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderType:     entity.SenderTypeBot,
		ContentType:    entity.ContentTypeText,
		Content:        "Choose an option",
		QuickReplies: []entity.QuickReply{
			{ID: "opt1", Title: "Option 1", Value: "opt1"},
			{ID: "opt2", Title: "Option 2", Value: "opt2"},
		},
	}

	output, err := uc.Execute(context.Background(), input)
	require.NoError(t, err)

	msg := output.Message
	assert.Equal(t, entity.ContentTypeInteractive, msg.ContentType)
	assert.Equal(t, "button", msg.Metadata["interactive_type"])

	// Verify interactive JSON
	var interactive map[string]interface{}
	err = json.Unmarshal([]byte(msg.Metadata["interactive"]), &interactive)
	require.NoError(t, err)
	assert.Equal(t, "button", interactive["type"])

	body, ok := interactive["body"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Choose an option", body["text"])

	action, ok := interactive["action"].(map[string]interface{})
	require.True(t, ok)
	buttons, ok := action["buttons"].([]interface{})
	require.True(t, ok)
	assert.Len(t, buttons, 2)

	// Verify message saved in repo
	assert.Len(t, msgRepo.Messages, 1)
}

func TestSendMessageUseCase_QuickReplies_List(t *testing.T) {
	_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

	qrs := make([]entity.QuickReply, 5)
	for i := 0; i < 5; i++ {
		qrs[i] = entity.QuickReply{
			ID:    fmt.Sprintf("opt%d", i+1),
			Title: fmt.Sprintf("Option %d", i+1),
		}
	}

	output, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderType:     entity.SenderTypeBot,
		ContentType:    entity.ContentTypeText,
		Content:        "Pick one",
		QuickReplies:   qrs,
	})
	require.NoError(t, err)

	msg := output.Message
	assert.Equal(t, entity.ContentTypeInteractive, msg.ContentType)
	assert.Equal(t, "list", msg.Metadata["interactive_type"])

	var interactive map[string]interface{}
	err = json.Unmarshal([]byte(msg.Metadata["interactive"]), &interactive)
	require.NoError(t, err)
	assert.Equal(t, "list", interactive["type"])

	action, ok := interactive["action"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Options", action["button"])

	sections, ok := action["sections"].([]interface{})
	require.True(t, ok)
	require.Len(t, sections, 1)

	section, ok := sections[0].(map[string]interface{})
	require.True(t, ok)
	rows, ok := section["rows"].([]interface{})
	require.True(t, ok)
	assert.Len(t, rows, 5)
}

func TestSendMessageUseCase_QuickReplies_TitleTruncation(t *testing.T) {
	t.Run("button title truncated to 20 chars", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		longTitle := "This title is way too long for a button"
		output, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeBot,
			ContentType:    entity.ContentTypeText,
			Content:        "Choose",
			QuickReplies: []entity.QuickReply{
				{ID: "opt1", Title: longTitle},
			},
		})
		require.NoError(t, err)

		var interactive map[string]interface{}
		err = json.Unmarshal([]byte(output.Message.Metadata["interactive"]), &interactive)
		require.NoError(t, err)

		action := interactive["action"].(map[string]interface{})
		buttons := action["buttons"].([]interface{})
		btn := buttons[0].(map[string]interface{})
		reply := btn["reply"].(map[string]interface{})
		assert.Equal(t, longTitle[:20], reply["title"])
		assert.Len(t, reply["title"], 20)
	})

	t.Run("list item title truncated to 24 chars", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		convRepo.Conversations["conv1"] = &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status: entity.ConversationStatusOpen,
		}
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		longTitle := "This is a very long title for a list item option"
		qrs := make([]entity.QuickReply, 4)
		for i := 0; i < 4; i++ {
			qrs[i] = entity.QuickReply{ID: fmt.Sprintf("opt%d", i+1), Title: longTitle}
		}

		output, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeBot,
			ContentType:    entity.ContentTypeText,
			Content:        "Choose",
			QuickReplies:   qrs,
		})
		require.NoError(t, err)

		var interactive map[string]interface{}
		err = json.Unmarshal([]byte(output.Message.Metadata["interactive"]), &interactive)
		require.NoError(t, err)

		action := interactive["action"].(map[string]interface{})
		sections := action["sections"].([]interface{})
		section := sections[0].(map[string]interface{})
		rows := section["rows"].([]interface{})
		row := rows[0].(map[string]interface{})
		assert.Equal(t, longTitle[:24], row["title"])
		assert.Len(t, row["title"], 24)
	})
}

func TestSendMessageUseCase_QuickReplies_NonInteractiveChannel(t *testing.T) {
	_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = activeWebChatChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

	output, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderType:     entity.SenderTypeBot,
		ContentType:    entity.ContentTypeText,
		Content:        "Choose an option",
		QuickReplies: []entity.QuickReply{
			{ID: "opt1", Title: "Option 1"},
			{ID: "opt2", Title: "Option 2"},
		},
	})
	require.NoError(t, err)

	msg := output.Message
	assert.Equal(t, entity.ContentTypeText, msg.ContentType)
	assert.Empty(t, msg.Metadata["interactive"])
	assert.Empty(t, msg.Metadata["interactive_type"])
}

func TestSendMessageUseCase_Attachments(t *testing.T) {
	msgRepo, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

	output, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderType:     entity.SenderTypeUser,
		ContentType:    entity.ContentTypeImage,
		Content:        "check this",
		Attachments: []*AttachmentInput{
			{
				Type: "image", URL: "https://example.com/img.png",
				Filename: "img.png", MimeType: "image/png",
				SizeBytes: 2048, ThumbnailURL: "https://example.com/thumb.png",
			},
		},
	})
	require.NoError(t, err)

	msg := output.Message
	require.Len(t, msg.Attachments, 1)
	att := msg.Attachments[0]
	assert.Equal(t, msg.ID, att.MessageID)
	assert.Equal(t, "image", att.Type)
	assert.Equal(t, "https://example.com/img.png", att.URL)
	assert.Equal(t, "img.png", att.Filename)
	assert.Equal(t, "image/png", att.MimeType)
	assert.Equal(t, int64(2048), att.SizeBytes)
	assert.Equal(t, "https://example.com/thumb.png", att.ThumbnailURL)

	// Verify attachment stored in repo
	storedAtts := msgRepo.Attachments[msg.ID]
	require.Len(t, storedAtts, 1)
	assert.Equal(t, msg.ID, storedAtts[0].MessageID)
}

func TestSendMessageUseCase_PublishFailure(t *testing.T) {
	msgRepo, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}
	producer.ReturnError = fmt.Errorf("nats connection lost")

	_, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderType:     entity.SenderTypeUser,
		ContentType:    entity.ContentTypeText,
		Content:        "hello",
	})
	require.Error(t, err)

	// Verify message status updated to failed
	require.Len(t, msgRepo.Messages, 1)
	for _, msg := range msgRepo.Messages {
		assert.Equal(t, entity.MessageStatusFailed, msg.Status)
	}
}

func TestSendMessageUseCase_FirstReplyTracking(t *testing.T) {
	t.Run("SenderType User sets FirstReplyAt when nil", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		conv := &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status:       entity.ConversationStatusOpen,
			FirstReplyAt: nil,
		}
		convRepo.Conversations["conv1"] = conv
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeUser,
			ContentType:    entity.ContentTypeText,
			Content:        "hello",
		})
		require.NoError(t, err)
		assert.NotNil(t, conv.FirstReplyAt)
	})

	t.Run("SenderType Bot does not set FirstReplyAt", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		conv := &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status:       entity.ConversationStatusOpen,
			FirstReplyAt: nil,
		}
		convRepo.Conversations["conv1"] = conv
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeBot,
			ContentType:    entity.ContentTypeText,
			Content:        "hello",
		})
		require.NoError(t, err)
		assert.Nil(t, conv.FirstReplyAt)
	})

	t.Run("FirstReplyAt already set stays unchanged", func(t *testing.T) {
		_, convRepo, chRepo, contactRepo, _, uc := setupSendMessageTest()

		existingTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		conv := &entity.Conversation{
			ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
			Status:       entity.ConversationStatusOpen,
			FirstReplyAt: &existingTime,
		}
		convRepo.Conversations["conv1"] = conv
		chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
		contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

		_, err := uc.Execute(context.Background(), &SendMessageInput{
			TenantID:       "t1",
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeUser,
			ContentType:    entity.ContentTypeText,
			Content:        "hello again",
		})
		require.NoError(t, err)
		assert.Equal(t, existingTime, *conv.FirstReplyAt)
	})
}

func TestSendMessageUseCase_OutboundMessageFields(t *testing.T) {
	_, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = &entity.Channel{
		ID: "ch1", TenantID: "t1", Type: entity.ChannelTypeTelegram,
		Name: "Telegram", Enabled: true, ConnectionStatus: entity.ConnectionStatusConnected,
	}
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

	output, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderID:       "user1",
		SenderType:     entity.SenderTypeUser,
		ContentType:    entity.ContentTypeText,
		Content:        "test message",
		Metadata:       map[string]string{"key": "value"},
	})
	require.NoError(t, err)
	require.Len(t, producer.OutboundMessages, 1)

	out := producer.OutboundMessages[0]
	assert.Equal(t, output.Message.ID, out.ID)
	assert.Equal(t, "t1", out.TenantID)
	assert.Equal(t, "ch1", out.ChannelID)
	assert.Equal(t, "telegram", out.ChannelType)
	assert.Equal(t, "conv1", out.ConversationID)
	assert.Equal(t, "c1", out.ContactID)
	assert.Equal(t, "5511999", out.RecipientID)
	assert.Equal(t, "text", out.ContentType)
	assert.Equal(t, "test message", out.Content)
	assert.Equal(t, "value", out.Metadata["key"])
	assert.False(t, out.Timestamp.IsZero())
}

func TestSendMessageUseCase_OutboundAttachments(t *testing.T) {
	_, convRepo, chRepo, contactRepo, producer, uc := setupSendMessageTest()

	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "t1", ChannelID: "ch1", ContactID: "c1",
		Status: entity.ConversationStatusOpen,
	}
	chRepo.Channels["ch1"] = activeWhatsAppChannel("t1", "ch1")
	contactRepo.Contacts["c1"] = &entity.Contact{ID: "c1", TenantID: "t1", Phone: "5511999"}

	_, err := uc.Execute(context.Background(), &SendMessageInput{
		TenantID:       "t1",
		ConversationID: "conv1",
		SenderType:     entity.SenderTypeUser,
		ContentType:    entity.ContentTypeImage,
		Content:        "file",
		Attachments: []*AttachmentInput{
			{
				Type: "document", URL: "https://example.com/doc.pdf",
				Filename: "doc.pdf", MimeType: "application/pdf",
				SizeBytes: 4096, ThumbnailURL: "https://example.com/thumb.png",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, producer.OutboundMessages, 1)

	outAtts := producer.OutboundMessages[0].Attachments
	require.Len(t, outAtts, 1)
	assert.Equal(t, "document", outAtts[0].Type)
	assert.Equal(t, "https://example.com/doc.pdf", outAtts[0].URL)
	assert.Equal(t, "doc.pdf", outAtts[0].Filename)
	assert.Equal(t, "application/pdf", outAtts[0].MimeType)
	assert.Equal(t, int64(4096), outAtts[0].SizeBytes)
	assert.Equal(t, "https://example.com/thumb.png", outAtts[0].ThumbnailURL)
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestChannelSupportsInteractive(t *testing.T) {
	tests := []struct {
		channelType entity.ChannelType
		expected    bool
	}{
		{entity.ChannelTypeWhatsApp, true},
		{entity.ChannelTypeWhatsAppOfficial, true},
		{entity.ChannelTypeTelegram, true},
		{entity.ChannelTypeWebChat, false},
		{entity.ChannelTypeSMS, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.channelType), func(t *testing.T) {
			assert.Equal(t, tt.expected, channelSupportsInteractive(tt.channelType))
		})
	}
}

func TestGetInteractiveType(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{1, "button"},
		{2, "button"},
		{3, "button"},
		{4, "list"},
		{10, "list"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("count_%d", tt.count), func(t *testing.T) {
			assert.Equal(t, tt.expected, getInteractiveType(tt.count))
		})
	}
}

func TestBuildInteractiveFromQuickReplies(t *testing.T) {
	t.Run("empty quick replies returns empty string", func(t *testing.T) {
		result := buildInteractiveFromQuickReplies("body", nil)
		assert.Equal(t, "", result)

		result = buildInteractiveFromQuickReplies("body", []entity.QuickReply{})
		assert.Equal(t, "", result)
	})

	t.Run("2 items produces button format", func(t *testing.T) {
		qrs := []entity.QuickReply{
			{ID: "id1", Title: "Yes"},
			{ID: "id2", Title: "No"},
		}
		result := buildInteractiveFromQuickReplies("Confirm?", qrs)
		require.NotEmpty(t, result)

		var data map[string]interface{}
		err := json.Unmarshal([]byte(result), &data)
		require.NoError(t, err)

		assert.Equal(t, "button", data["type"])
		body := data["body"].(map[string]interface{})
		assert.Equal(t, "Confirm?", body["text"])

		action := data["action"].(map[string]interface{})
		buttons := action["buttons"].([]interface{})
		require.Len(t, buttons, 2)

		btn0 := buttons[0].(map[string]interface{})
		assert.Equal(t, "reply", btn0["type"])
		reply0 := btn0["reply"].(map[string]interface{})
		assert.Equal(t, "id1", reply0["id"])
		assert.Equal(t, "Yes", reply0["title"])

		btn1 := buttons[1].(map[string]interface{})
		reply1 := btn1["reply"].(map[string]interface{})
		assert.Equal(t, "id2", reply1["id"])
		assert.Equal(t, "No", reply1["title"])
	})

	t.Run("5 items produces list format", func(t *testing.T) {
		qrs := make([]entity.QuickReply, 5)
		for i := 0; i < 5; i++ {
			qrs[i] = entity.QuickReply{
				ID:    fmt.Sprintf("id%d", i+1),
				Title: fmt.Sprintf("Item %d", i+1),
			}
		}
		result := buildInteractiveFromQuickReplies("Pick one", qrs)
		require.NotEmpty(t, result)

		var data map[string]interface{}
		err := json.Unmarshal([]byte(result), &data)
		require.NoError(t, err)

		assert.Equal(t, "list", data["type"])
		body := data["body"].(map[string]interface{})
		assert.Equal(t, "Pick one", body["text"])

		action := data["action"].(map[string]interface{})
		assert.Equal(t, "Options", action["button"])
		sections := action["sections"].([]interface{})
		require.Len(t, sections, 1)

		section := sections[0].(map[string]interface{})
		rows := section["rows"].([]interface{})
		require.Len(t, rows, 5)

		row0 := rows[0].(map[string]interface{})
		assert.Equal(t, "id1", row0["id"])
		assert.Equal(t, "Item 1", row0["title"])
	})

	t.Run("button title truncated to 20 chars", func(t *testing.T) {
		longTitle := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		qrs := []entity.QuickReply{{ID: "id1", Title: longTitle}}
		result := buildInteractiveFromQuickReplies("body", qrs)

		var data map[string]interface{}
		err := json.Unmarshal([]byte(result), &data)
		require.NoError(t, err)

		action := data["action"].(map[string]interface{})
		buttons := action["buttons"].([]interface{})
		btn := buttons[0].(map[string]interface{})
		reply := btn["reply"].(map[string]interface{})
		assert.Equal(t, longTitle[:20], reply["title"])
	})

	t.Run("list title truncated to 24 chars and ID to 200", func(t *testing.T) {
		longTitle := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		qrs := make([]entity.QuickReply, 4)
		for i := 0; i < 4; i++ {
			qrs[i] = entity.QuickReply{ID: fmt.Sprintf("id%d", i+1), Title: longTitle}
		}
		result := buildInteractiveFromQuickReplies("body", qrs)

		var data map[string]interface{}
		err := json.Unmarshal([]byte(result), &data)
		require.NoError(t, err)

		action := data["action"].(map[string]interface{})
		sections := action["sections"].([]interface{})
		section := sections[0].(map[string]interface{})
		rows := section["rows"].([]interface{})
		row := rows[0].(map[string]interface{})
		assert.Equal(t, longTitle[:24], row["title"])
	})
}
