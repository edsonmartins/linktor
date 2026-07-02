package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// defaultMaxDownloadBytes caps how much data DownloadFile will read from a
// Telegram file to protect against unbounded memory usage. Kept consistent
// with the Slack/Teams adapters (25 MB).
const defaultMaxDownloadBytes int64 = 25 * 1024 * 1024

// Client wraps the Telegram Bot API client
type Client struct {
	api         *tgbotapi.BotAPI
	botToken    string
	secretToken string
	httpClient  *http.Client
	// apiBaseURL is the Bot API base (no trailing slash). Defaults to the
	// public Telegram endpoint; overridable in tests.
	apiBaseURL string
	// maxDownloadBytes caps DownloadFile reads. Zero means the default cap.
	maxDownloadBytes int64
}

// NewClient creates a new Telegram client with the provided bot token
func NewClient(botToken string) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &Client{
		api:              bot,
		botToken:         botToken,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		apiBaseURL:       "https://api.telegram.org",
		maxDownloadBytes: defaultMaxDownloadBytes,
	}, nil
}

// SetSecretToken configures the secret_token registered with Telegram's
// setWebhook. Telegram echoes it back on every update via the
// X-Telegram-Bot-Api-Secret-Token header so the inbound handler can validate
// requests. It MUST be registered here or legitimate updates get rejected.
func (c *Client) SetSecretToken(token string) {
	c.secretToken = token
}

// GetMe returns information about the bot
func (c *Client) GetMe() (tgbotapi.User, error) {
	return c.api.GetMe()
}

// SetWebhook configures the webhook URL for receiving updates. When a secret
// token is configured it is registered as secret_token so Telegram sends the
// X-Telegram-Bot-Api-Secret-Token header on every update.
//
// NOTE: the tgbotapi WebhookConfig (v5.5.1) does not expose a secret_token
// field, so we call the setWebhook endpoint directly to be able to register it.
func (c *Client) SetWebhook(webhookURL string) error {
	return c.SetWebhookContext(context.Background(), webhookURL)
}

// SetWebhookContext is SetWebhook with an explicit context.
func (c *Client) SetWebhookContext(ctx context.Context, webhookURL string) error {
	req, err := c.buildSetWebhookRequest(ctx, webhookURL)
	if err != nil {
		return fmt.Errorf("failed to build setWebhook request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var apiResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to set webhook: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if !apiResp.OK {
		return fmt.Errorf("failed to set webhook: %s", apiResp.Description)
	}

	return nil
}

// buildSetWebhookRequest builds the setWebhook POST request, including the
// secret_token when configured.
func (c *Client) buildSetWebhookRequest(ctx context.Context, webhookURL string) (*http.Request, error) {
	form := url.Values{}
	form.Set("url", webhookURL)
	if c.secretToken != "" {
		form.Set("secret_token", c.secretToken)
	}

	endpoint := fmt.Sprintf("%s/bot%s/setWebhook", c.apiBaseURL, c.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
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
	return c.DownloadFileContext(context.Background(), fileID)
}

// DownloadFileContext downloads a file from Telegram servers using the given
// context, a bounded HTTP client (timeout) and a size cap to avoid unbounded
// memory usage.
func (c *Client) DownloadFileContext(ctx context.Context, fileID string) ([]byte, string, error) {
	file, err := c.GetFile(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileURL := c.GetFileURL(file.FilePath)

	data, err := c.downloadURL(ctx, fileURL)
	if err != nil {
		return nil, "", err
	}

	return data, file.FilePath, nil
}

// downloadURL performs the bounded HTTP GET used by DownloadFile.
func (c *Client) downloadURL(ctx context.Context, fileURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: status %d", resp.StatusCode)
	}

	limit := c.maxDownloadBytes
	if limit <= 0 {
		limit = defaultMaxDownloadBytes
	}

	// Read up to limit+1 so we can detect an over-limit body.
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds maximum download size of %d bytes", limit)
	}

	return data, nil
}

// GetUserProfilePhotos retrieves a user's profile photos
func (c *Client) GetUserProfilePhotos(userID int64) (tgbotapi.UserProfilePhotos, error) {
	config := tgbotapi.NewUserProfilePhotos(userID)
	config.Limit = 1 // We only need the latest photo
	return c.api.GetUserProfilePhotos(config)
}
