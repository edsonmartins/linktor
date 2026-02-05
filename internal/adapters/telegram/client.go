package telegram

import (
	"fmt"
	"io"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Client wraps the Telegram Bot API client
type Client struct {
	api      *tgbotapi.BotAPI
	botToken string
}

// NewClient creates a new Telegram client with the provided bot token
func NewClient(botToken string) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &Client{
		api:      bot,
		botToken: botToken,
	}, nil
}

// GetMe returns information about the bot
func (c *Client) GetMe() (tgbotapi.User, error) {
	return c.api.GetMe()
}

// SetWebhook configures the webhook URL for receiving updates
func (c *Client) SetWebhook(webhookURL string) error {
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to create webhook config: %w", err)
	}

	_, err = c.api.Request(wh)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	return nil
}

// DeleteWebhook removes the current webhook
func (c *Client) DeleteWebhook() error {
	dw := tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: false,
	}
	_, err := c.api.Request(dw)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	return nil
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(chatID int64, text string, parseMode string, replyToMsgID int64) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)

	if parseMode != "" {
		msg.ParseMode = parseMode
	}

	if replyToMsgID > 0 {
		msg.ReplyToMessageID = int(replyToMsgID)
	}

	return c.api.Send(msg)
}

// SendMessageWithKeyboard sends a message with an inline keyboard
func (c *Client) SendMessageWithKeyboard(chatID int64, text string, parseMode string, keyboard *InlineKeyboard, replyToMsgID int64) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)

	if parseMode != "" {
		msg.ParseMode = parseMode
	}

	if replyToMsgID > 0 {
		msg.ReplyToMessageID = int(replyToMsgID)
	}

	if keyboard != nil && len(keyboard.Buttons) > 0 {
		// Convert our keyboard format to tgbotapi format
		var rows [][]tgbotapi.InlineKeyboardButton
		for _, row := range keyboard.Buttons {
			var buttons []tgbotapi.InlineKeyboardButton
			for _, btn := range row {
				if btn.URL != "" {
					buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonURL(btn.Text, btn.URL))
				} else {
					buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.CallbackData))
				}
			}
			rows = append(rows, buttons)
		}
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	}

	return c.api.Send(msg)
}

// SendPhoto sends a photo to a chat
func (c *Client) SendPhoto(chatID int64, fileURL string, caption string, replyToMsgID int64) (tgbotapi.Message, error) {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(fileURL))
	photo.Caption = caption
	if replyToMsgID > 0 {
		photo.ReplyToMessageID = int(replyToMsgID)
	}
	return c.api.Send(photo)
}

// SendDocument sends a document/file to a chat
func (c *Client) SendDocument(chatID int64, fileURL string, caption string, replyToMsgID int64) (tgbotapi.Message, error) {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileURL(fileURL))
	doc.Caption = caption
	if replyToMsgID > 0 {
		doc.ReplyToMessageID = int(replyToMsgID)
	}
	return c.api.Send(doc)
}

// SendVideo sends a video to a chat
func (c *Client) SendVideo(chatID int64, fileURL string, caption string, replyToMsgID int64) (tgbotapi.Message, error) {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(fileURL))
	video.Caption = caption
	if replyToMsgID > 0 {
		video.ReplyToMessageID = int(replyToMsgID)
	}
	return c.api.Send(video)
}

// SendAudio sends an audio file to a chat
func (c *Client) SendAudio(chatID int64, fileURL string, caption string, replyToMsgID int64) (tgbotapi.Message, error) {
	audio := tgbotapi.NewAudio(chatID, tgbotapi.FileURL(fileURL))
	audio.Caption = caption
	if replyToMsgID > 0 {
		audio.ReplyToMessageID = int(replyToMsgID)
	}
	return c.api.Send(audio)
}

// SendChatAction sends a chat action (typing indicator, etc.)
func (c *Client) SendChatAction(chatID int64, action string) error {
	chatAction := tgbotapi.NewChatAction(chatID, action)
	_, err := c.api.Request(chatAction)
	return err
}

// SendTyping sends a "typing" indicator to the chat
func (c *Client) SendTyping(chatID int64) error {
	return c.SendChatAction(chatID, tgbotapi.ChatTyping)
}

// GetFile retrieves file information for downloading
func (c *Client) GetFile(fileID string) (tgbotapi.File, error) {
	return c.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
}

// GetFileURL constructs the download URL for a file
func (c *Client) GetFileURL(filePath string) string {
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", c.botToken, filePath)
}

// DownloadFile downloads a file from Telegram servers
func (c *Client) DownloadFile(fileID string) ([]byte, string, error) {
	file, err := c.GetFile(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileURL := c.GetFileURL(file.FilePath)

	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download file: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file data: %w", err)
	}

	return data, file.FilePath, nil
}

// GetUserProfilePhotos retrieves a user's profile photos
func (c *Client) GetUserProfilePhotos(userID int64) (tgbotapi.UserProfilePhotos, error) {
	config := tgbotapi.NewUserProfilePhotos(userID)
	config.Limit = 1 // We only need the latest photo
	return c.api.GetUserProfilePhotos(config)
}
