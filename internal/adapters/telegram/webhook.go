package telegram

import (
	"encoding/json"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ParseWebhook parses an incoming webhook payload
func ParseWebhook(body []byte) (*tgbotapi.Update, error) {
	var update tgbotapi.Update
	if err := json.Unmarshal(body, &update); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}
	return &update, nil
}

// WebhookHandler handles incoming webhook requests
type WebhookHandler struct {
	adapter *Adapter
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(adapter *Adapter) *WebhookHandler {
	return &WebhookHandler{
		adapter: adapter,
	}
}

// IsCallbackQuery checks if the update is a callback query (inline keyboard button press)
func IsCallbackQuery(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil
}

// IsMessage checks if the update contains a message
func IsMessage(update *tgbotapi.Update) bool {
	return update.Message != nil
}

// IsEditedMessage checks if the update contains an edited message
func IsEditedMessage(update *tgbotapi.Update) bool {
	return update.EditedMessage != nil
}

// IsCommand checks if the message is a bot command
func IsCommand(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}
	return update.Message.IsCommand()
}

// GetCommand returns the command from a message (without the leading /)
func GetCommand(update *tgbotapi.Update) string {
	if update.Message == nil {
		return ""
	}
	return update.Message.Command()
}

// GetCommandArguments returns the arguments after the command
func GetCommandArguments(update *tgbotapi.Update) string {
	if update.Message == nil {
		return ""
	}
	return update.Message.CommandArguments()
}

// CallbackQueryInfo contains information about a callback query
type CallbackQueryInfo struct {
	ID              string
	Data            string
	ChatID          int64
	MessageID       int64
	FromUserID      int64
	FromUsername    string
	FromFirstName   string
	InlineMessageID string
}

// ExtractCallbackQuery extracts callback query information
func ExtractCallbackQuery(update *tgbotapi.Update) *CallbackQueryInfo {
	if update.CallbackQuery == nil {
		return nil
	}

	cq := update.CallbackQuery

	info := &CallbackQueryInfo{
		ID:            cq.ID,
		Data:          cq.Data,
		FromUserID:    cq.From.ID,
		FromUsername:  cq.From.UserName,
		FromFirstName: cq.From.FirstName,
	}

	if cq.Message != nil {
		info.ChatID = cq.Message.Chat.ID
		info.MessageID = int64(cq.Message.MessageID)
	}

	if cq.InlineMessageID != "" {
		info.InlineMessageID = cq.InlineMessageID
	}

	return info
}

// CommandInfo contains information about a bot command
type CommandInfo struct {
	Command   string
	Arguments string
	ChatID    int64
	FromUser  *UserInfo
}

// UserInfo contains information about a Telegram user
type UserInfo struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	IsBot     bool
}

// ExtractCommand extracts command information from an update
func ExtractCommand(update *tgbotapi.Update) *CommandInfo {
	if update.Message == nil || !update.Message.IsCommand() {
		return nil
	}

	msg := update.Message

	return &CommandInfo{
		Command:   msg.Command(),
		Arguments: msg.CommandArguments(),
		ChatID:    msg.Chat.ID,
		FromUser: &UserInfo{
			ID:        msg.From.ID,
			Username:  msg.From.UserName,
			FirstName: msg.From.FirstName,
			LastName:  msg.From.LastName,
			IsBot:     msg.From.IsBot,
		},
	}
}
