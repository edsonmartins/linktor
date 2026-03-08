package telegram

import (
	"encoding/json"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWebhook_ValidMessage(t *testing.T) {
	payload := map[string]interface{}{
		"update_id": 12345,
		"message": map[string]interface{}{
			"message_id": 100,
			"from": map[string]interface{}{
				"id":         456,
				"is_bot":     false,
				"first_name": "Test",
				"username":   "testuser",
			},
			"chat": map[string]interface{}{
				"id":   789,
				"type": "private",
			},
			"date": time.Now().Unix(),
			"text": "hello",
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	update, err := ParseWebhook(body)
	require.NoError(t, err)
	require.NotNil(t, update)
	require.NotNil(t, update.Message)
	assert.Equal(t, 100, update.Message.MessageID)
	assert.Equal(t, "hello", update.Message.Text)
	assert.Equal(t, int64(789), update.Message.Chat.ID)
}

func TestParseWebhook_InvalidJSON(t *testing.T) {
	update, err := ParseWebhook([]byte("not json"))
	assert.Error(t, err)
	assert.Nil(t, update)
}

func TestParseWebhook_EmptyBody(t *testing.T) {
	update, err := ParseWebhook([]byte("{}"))
	require.NoError(t, err)
	require.NotNil(t, update)
	assert.Nil(t, update.Message)
	assert.Nil(t, update.CallbackQuery)
}

func TestParseWebhook_CallbackQuery(t *testing.T) {
	payload := map[string]interface{}{
		"update_id": 99999,
		"callback_query": map[string]interface{}{
			"id": "cb-123",
			"from": map[string]interface{}{
				"id":         10,
				"first_name": "Cb",
				"username":   "cbuser",
			},
			"data": "action_1",
			"message": map[string]interface{}{
				"message_id": 500,
				"chat": map[string]interface{}{
					"id":   20,
					"type": "private",
				},
				"date": time.Now().Unix(),
				"text": "original",
				"from": map[string]interface{}{
					"id":         10,
					"first_name": "Cb",
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	update, err := ParseWebhook(body)
	require.NoError(t, err)
	require.NotNil(t, update.CallbackQuery)
	assert.Equal(t, "action_1", update.CallbackQuery.Data)
	assert.Equal(t, "cb-123", update.CallbackQuery.ID)
}

func TestIsCallbackQuery(t *testing.T) {
	tests := []struct {
		name     string
		update   *tgbotapi.Update
		expected bool
	}{
		{
			name:     "with callback query",
			update:   &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{ID: "1"}},
			expected: true,
		},
		{
			name:     "without callback query",
			update:   &tgbotapi.Update{},
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsCallbackQuery(tc.update))
		})
	}
}

func TestIsMessage(t *testing.T) {
	tests := []struct {
		name     string
		update   *tgbotapi.Update
		expected bool
	}{
		{
			name:     "with message",
			update:   &tgbotapi.Update{Message: &tgbotapi.Message{MessageID: 1}},
			expected: true,
		},
		{
			name:     "without message",
			update:   &tgbotapi.Update{},
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsMessage(tc.update))
		})
	}
}

func TestIsEditedMessage(t *testing.T) {
	tests := []struct {
		name     string
		update   *tgbotapi.Update
		expected bool
	}{
		{
			name:     "with edited message",
			update:   &tgbotapi.Update{EditedMessage: &tgbotapi.Message{MessageID: 1}},
			expected: true,
		},
		{
			name:     "without edited message",
			update:   &tgbotapi.Update{},
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsEditedMessage(tc.update))
		})
	}
}

func TestIsCommand(t *testing.T) {
	tests := []struct {
		name     string
		update   *tgbotapi.Update
		expected bool
	}{
		{
			name: "with command",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					Text:      "/start",
					Entities: []tgbotapi.MessageEntity{
						{Type: "bot_command", Offset: 0, Length: 6},
					},
				},
			},
			expected: true,
		},
		{
			name: "with regular text",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{MessageID: 1, Text: "hello"},
			},
			expected: false,
		},
		{
			name:     "nil message",
			update:   &tgbotapi.Update{},
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsCommand(tc.update))
		})
	}
}

func TestGetCommand(t *testing.T) {
	tests := []struct {
		name     string
		update   *tgbotapi.Update
		expected string
	}{
		{
			name: "start command",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					Text:      "/start",
					Entities: []tgbotapi.MessageEntity{
						{Type: "bot_command", Offset: 0, Length: 6},
					},
				},
			},
			expected: "start",
		},
		{
			name: "help command with args",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					Text:      "/help topic",
					Entities: []tgbotapi.MessageEntity{
						{Type: "bot_command", Offset: 0, Length: 5},
					},
				},
			},
			expected: "help",
		},
		{
			name:     "nil message",
			update:   &tgbotapi.Update{},
			expected: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, GetCommand(tc.update))
		})
	}
}

func TestGetCommandArguments(t *testing.T) {
	tests := []struct {
		name     string
		update   *tgbotapi.Update
		expected string
	}{
		{
			name: "command with arguments",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					Text:      "/help some arguments here",
					Entities: []tgbotapi.MessageEntity{
						{Type: "bot_command", Offset: 0, Length: 5},
					},
				},
			},
			expected: "some arguments here",
		},
		{
			name: "command without arguments",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: 1,
					Text:      "/start",
					Entities: []tgbotapi.MessageEntity{
						{Type: "bot_command", Offset: 0, Length: 6},
					},
				},
			},
			expected: "",
		},
		{
			name:     "nil message",
			update:   &tgbotapi.Update{},
			expected: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, GetCommandArguments(tc.update))
		})
	}
}

func TestExtractCallbackQuery(t *testing.T) {
	t.Run("valid callback query with message", func(t *testing.T) {
		update := &tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:   "cq-1",
				From: newUser(10, "quser", "Query", "User"),
				Message: &tgbotapi.Message{
					MessageID: 300,
					Chat:      newPrivateChat(20),
				},
				Data: "btn_action",
			},
		}

		info := ExtractCallbackQuery(update)
		require.NotNil(t, info)
		assert.Equal(t, "cq-1", info.ID)
		assert.Equal(t, "btn_action", info.Data)
		assert.Equal(t, int64(20), info.ChatID)
		assert.Equal(t, int64(300), info.MessageID)
		assert.Equal(t, int64(10), info.FromUserID)
		assert.Equal(t, "quser", info.FromUsername)
		assert.Equal(t, "Query", info.FromFirstName)
		assert.Empty(t, info.InlineMessageID)
	})

	t.Run("callback query without message", func(t *testing.T) {
		update := &tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:              "cq-2",
				From:            newUser(10, "u", "U", ""),
				Data:            "data",
				InlineMessageID: "inline-123",
			},
		}

		info := ExtractCallbackQuery(update)
		require.NotNil(t, info)
		assert.Equal(t, int64(0), info.ChatID)
		assert.Equal(t, int64(0), info.MessageID)
		assert.Equal(t, "inline-123", info.InlineMessageID)
	})

	t.Run("nil callback query", func(t *testing.T) {
		update := &tgbotapi.Update{}
		info := ExtractCallbackQuery(update)
		assert.Nil(t, info)
	})
}

func TestExtractCommand(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		update := &tgbotapi.Update{
			Message: &tgbotapi.Message{
				MessageID: 1,
				From:      &tgbotapi.User{ID: 10, UserName: "cmduser", FirstName: "Cmd", LastName: "User", IsBot: false},
				Chat:      newPrivateChat(20),
				Text:      "/settings language",
				Entities: []tgbotapi.MessageEntity{
					{Type: "bot_command", Offset: 0, Length: 9},
				},
			},
		}

		info := ExtractCommand(update)
		require.NotNil(t, info)
		assert.Equal(t, "settings", info.Command)
		assert.Equal(t, "language", info.Arguments)
		assert.Equal(t, int64(20), info.ChatID)
		require.NotNil(t, info.FromUser)
		assert.Equal(t, int64(10), info.FromUser.ID)
		assert.Equal(t, "cmduser", info.FromUser.Username)
		assert.Equal(t, "Cmd", info.FromUser.FirstName)
		assert.Equal(t, "User", info.FromUser.LastName)
		assert.False(t, info.FromUser.IsBot)
	})

	t.Run("not a command", func(t *testing.T) {
		update := &tgbotapi.Update{
			Message: &tgbotapi.Message{
				MessageID: 1,
				From:      newUser(1, "u", "U", ""),
				Chat:      newPrivateChat(2),
				Text:      "just text",
			},
		}
		info := ExtractCommand(update)
		assert.Nil(t, info)
	})

	t.Run("nil message", func(t *testing.T) {
		update := &tgbotapi.Update{}
		info := ExtractCommand(update)
		assert.Nil(t, info)
	})
}
