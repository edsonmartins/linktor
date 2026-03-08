package telegram

import (
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPrivateChat(chatID int64) *tgbotapi.Chat {
	return &tgbotapi.Chat{ID: chatID, Type: "private"}
}

func newUser(id int64, username, firstName, lastName string) *tgbotapi.User {
	return &tgbotapi.User{
		ID:        id,
		UserName:  username,
		FirstName: firstName,
		LastName:  lastName,
	}
}

func TestExtractIncomingMessage_TextMessage(t *testing.T) {
	now := time.Now()
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 100,
			From:      newUser(456, "testuser", "Test", "User"),
			Chat:      newPrivateChat(789),
			Date:      int(now.Unix()),
			Text:      "hello world",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, int64(100), msg.MessageID)
	assert.Equal(t, int64(789), msg.ChatID)
	assert.Equal(t, int64(456), msg.FromUserID)
	assert.Equal(t, "testuser", msg.FromUsername)
	assert.Equal(t, "Test", msg.FromFirstName)
	assert.Equal(t, "User", msg.FromLastName)
	assert.Equal(t, "hello world", msg.Text)
	assert.Equal(t, MessageTypeText, msg.MessageType)
	assert.False(t, msg.IsEdited)
	assert.Nil(t, msg.ReplyToMsgID)
	assert.Equal(t, time.Unix(int64(update.Message.Date), 0), msg.Timestamp)
}

func TestExtractIncomingMessage_EditedMessage(t *testing.T) {
	update := &tgbotapi.Update{
		EditedMessage: &tgbotapi.Message{
			MessageID: 200,
			From:      newUser(1, "editor", "Ed", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Text:      "edited text",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.True(t, msg.IsEdited)
	assert.Equal(t, "edited text", msg.Text)
	assert.Equal(t, MessageTypeText, msg.MessageType)
}

func TestExtractIncomingMessage_CallbackQuery(t *testing.T) {
	update := &tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-1",
			From: newUser(10, "cbuser", "Cb", ""),
			Message: &tgbotapi.Message{
				MessageID: 300,
				From:      newUser(10, "cbuser", "Cb", ""),
				Chat:      newPrivateChat(20),
				Date:      int(time.Now().Unix()),
				Text:      "original text",
			},
			Data: "button_pressed",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	// CallbackQuery overrides text and type
	assert.Equal(t, "button_pressed", msg.Text)
	assert.Equal(t, MessageTypeText, msg.MessageType)
	assert.Equal(t, int64(300), msg.MessageID)
	assert.Equal(t, int64(20), msg.ChatID)
}

func TestExtractIncomingMessage_NilUpdate(t *testing.T) {
	// No Message, EditedMessage, or CallbackQuery
	update := &tgbotapi.Update{}
	msg := ExtractIncomingMessage(update)
	assert.Nil(t, msg)
}

func TestExtractIncomingMessage_CallbackQueryNilMessage(t *testing.T) {
	update := &tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb-2",
			From:    newUser(1, "u", "U", ""),
			Message: nil,
			Data:    "data",
		},
	}
	msg := ExtractIncomingMessage(update)
	assert.Nil(t, msg)
}

func TestExtractIncomingMessage_GroupChatIgnored(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      newUser(1, "u", "U", ""),
			Chat:      &tgbotapi.Chat{ID: 10, Type: "group"},
			Date:      int(time.Now().Unix()),
			Text:      "group message",
		},
	}
	msg := ExtractIncomingMessage(update)
	assert.Nil(t, msg)
}

func TestExtractIncomingMessage_SupergroupChatIgnored(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      newUser(1, "u", "U", ""),
			Chat:      &tgbotapi.Chat{ID: 10, Type: "supergroup"},
			Date:      int(time.Now().Unix()),
			Text:      "supergroup message",
		},
	}
	msg := ExtractIncomingMessage(update)
	assert.Nil(t, msg)
}

func TestExtractIncomingMessage_ChannelChatIgnored(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      newUser(1, "u", "U", ""),
			Chat:      &tgbotapi.Chat{ID: 10, Type: "channel"},
			Date:      int(time.Now().Unix()),
			Text:      "channel message",
		},
	}
	msg := ExtractIncomingMessage(update)
	assert.Nil(t, msg)
}

func TestExtractIncomingMessage_Photo(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 10,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Photo: []tgbotapi.PhotoSize{
				{FileID: "small", FileUniqueID: "s1", Width: 100, Height: 100, FileSize: 1000},
				{FileID: "medium", FileUniqueID: "m1", Width: 320, Height: 320, FileSize: 5000},
				{FileID: "large", FileUniqueID: "l1", Width: 800, Height: 800, FileSize: 20000},
			},
			Caption: "my photo",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypePhoto, msg.MessageType)
	assert.Equal(t, "large", msg.MediaFileID) // Largest photo
	assert.Equal(t, int64(20000), msg.MediaFileSize)
	assert.Equal(t, "my photo", msg.Caption)
}

func TestExtractIncomingMessage_Video(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 11,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Video: &tgbotapi.Video{
				FileID:   "vid1",
				MimeType: "video/mp4",
				FileSize: 500000,
			},
			Caption: "video caption",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeVideo, msg.MessageType)
	assert.Equal(t, "vid1", msg.MediaFileID)
	assert.Equal(t, "video/mp4", msg.MediaMimeType)
	assert.Equal(t, int64(500000), msg.MediaFileSize)
	assert.Equal(t, "video caption", msg.Caption)
}

func TestExtractIncomingMessage_Audio(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 12,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Audio: &tgbotapi.Audio{
				FileID:   "aud1",
				MimeType: "audio/mpeg",
				FileName: "song.mp3",
				FileSize: 300000,
			},
			Caption: "audio caption",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeAudio, msg.MessageType)
	assert.Equal(t, "aud1", msg.MediaFileID)
	assert.Equal(t, "audio/mpeg", msg.MediaMimeType)
	assert.Equal(t, "song.mp3", msg.MediaFileName)
	assert.Equal(t, int64(300000), msg.MediaFileSize)
	assert.Equal(t, "audio caption", msg.Caption)
}

func TestExtractIncomingMessage_Voice(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 13,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Voice: &tgbotapi.Voice{
				FileID:   "voice1",
				MimeType: "audio/ogg",
				FileSize: 50000,
			},
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeVoice, msg.MessageType)
	assert.Equal(t, "voice1", msg.MediaFileID)
	assert.Equal(t, "audio/ogg", msg.MediaMimeType)
	assert.Equal(t, int64(50000), msg.MediaFileSize)
}

func TestExtractIncomingMessage_Document(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 14,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Document: &tgbotapi.Document{
				FileID:   "doc1",
				MimeType: "application/pdf",
				FileName: "report.pdf",
				FileSize: 100000,
			},
			Caption: "doc caption",
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeDocument, msg.MessageType)
	assert.Equal(t, "doc1", msg.MediaFileID)
	assert.Equal(t, "application/pdf", msg.MediaMimeType)
	assert.Equal(t, "report.pdf", msg.MediaFileName)
	assert.Equal(t, int64(100000), msg.MediaFileSize)
	assert.Equal(t, "doc caption", msg.Caption)
}

func TestExtractIncomingMessage_Sticker(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 15,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Sticker: &tgbotapi.Sticker{
				FileID:   "stk1",
				FileSize: 25000,
			},
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeSticker, msg.MessageType)
	assert.Equal(t, "stk1", msg.MediaFileID)
	assert.Equal(t, int64(25000), msg.MediaFileSize)
}

func TestExtractIncomingMessage_Location(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 16,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Location: &tgbotapi.Location{
				Latitude:  -23.5505,
				Longitude: -46.6333,
			},
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeLocation, msg.MessageType)
	require.NotNil(t, msg.Location)
	assert.InDelta(t, -23.5505, msg.Location.Latitude, 0.0001)
	assert.InDelta(t, -46.6333, msg.Location.Longitude, 0.0001)
}

func TestExtractIncomingMessage_Contact(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 17,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Contact: &tgbotapi.Contact{
				PhoneNumber: "+5511999999999",
				FirstName:   "Jane",
				LastName:    "Doe",
				UserID:      999,
			},
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeContact, msg.MessageType)
	require.NotNil(t, msg.Contact)
	assert.Equal(t, "+5511999999999", msg.Contact.PhoneNumber)
	assert.Equal(t, "Jane", msg.Contact.FirstName)
	assert.Equal(t, "Doe", msg.Contact.LastName)
	assert.Equal(t, int64(999), msg.Contact.UserID)
}

func TestExtractIncomingMessage_VideoNote(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 18,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			VideoNote: &tgbotapi.VideoNote{
				FileID:   "vn1",
				FileSize: 75000,
			},
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeVideoNote, msg.MessageType)
	assert.Equal(t, "vn1", msg.MediaFileID)
	assert.Equal(t, int64(75000), msg.MediaFileSize)
}

func TestExtractIncomingMessage_ReplyToMessage(t *testing.T) {
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 50,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			Text:      "reply text",
			ReplyToMessage: &tgbotapi.Message{
				MessageID: 40,
			},
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	require.NotNil(t, msg.ReplyToMsgID)
	assert.Equal(t, int64(40), *msg.ReplyToMsgID)
}

func TestExtractIncomingMessage_EmptyTextFallback(t *testing.T) {
	// A message with no text, photo, video, audio, etc. should default to text type
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 99,
			From:      newUser(1, "u", "U", ""),
			Chat:      newPrivateChat(2),
			Date:      int(time.Now().Unix()),
			// No content fields set
		},
	}

	msg := ExtractIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeText, msg.MessageType)
	assert.Equal(t, "", msg.Text)
}
