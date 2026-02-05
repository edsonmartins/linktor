package telegram

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramConfig holds the configuration for a Telegram channel
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	BotName  string `json:"bot_name,omitempty"`
}

// WebhookPayload represents the incoming webhook update from Telegram
// We use the library's Update type directly but define this for documentation
type WebhookPayload = tgbotapi.Update

// MessageType represents the type of Telegram message
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypePhoto    MessageType = "photo"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeVoice    MessageType = "voice"
	MessageTypeDocument MessageType = "document"
	MessageTypeSticker  MessageType = "sticker"
	MessageTypeLocation MessageType = "location"
	MessageTypeContact  MessageType = "contact"
	MessageTypeVideoNote MessageType = "video_note"
)

// IncomingMessage represents a normalized incoming message from Telegram
type IncomingMessage struct {
	MessageID      int64
	ChatID         int64
	FromUserID     int64
	FromUsername   string
	FromFirstName  string
	FromLastName   string
	Text           string
	MessageType    MessageType
	Timestamp      time.Time
	ReplyToMsgID   *int64
	MediaFileID    string
	MediaMimeType  string
	MediaFileName  string
	MediaFileSize  int64
	Location       *Location
	Contact        *Contact
	Caption        string
	IsEdited       bool
}

// Location represents geographic coordinates
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Contact represents a contact shared via Telegram
type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
}

// OutgoingMessage represents a message to be sent to Telegram
type OutgoingMessage struct {
	ChatID         int64
	Text           string
	ParseMode      string // "HTML" or "Markdown" or ""
	ReplyToMsgID   int64
	ReplyMarkup    interface{} // For inline keyboards
	MediaURL       string
	MediaType      MessageType
	MediaCaption   string
}

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboard represents an inline keyboard with buttons
type InlineKeyboard struct {
	Buttons [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// SendResult contains the result of sending a message
type SendResult struct {
	MessageID int64
	Success   bool
	Error     string
}

// ExtractIncomingMessage converts a Telegram Update to our normalized IncomingMessage
func ExtractIncomingMessage(update *tgbotapi.Update) *IncomingMessage {
	var msg *tgbotapi.Message
	isEdited := false

	// Handle different update types
	if update.Message != nil {
		msg = update.Message
	} else if update.EditedMessage != nil {
		msg = update.EditedMessage
		isEdited = true
	} else if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		// Handle callback query from inline keyboard
		msg = update.CallbackQuery.Message
	} else {
		return nil
	}

	// Only process private chats (not groups or channels)
	if msg.Chat.Type != "private" {
		return nil
	}

	incoming := &IncomingMessage{
		MessageID:     int64(msg.MessageID),
		ChatID:        msg.Chat.ID,
		FromUserID:    msg.From.ID,
		FromUsername:  msg.From.UserName,
		FromFirstName: msg.From.FirstName,
		FromLastName:  msg.From.LastName,
		Timestamp:     time.Unix(int64(msg.Date), 0),
		IsEdited:      isEdited,
	}

	// Handle reply
	if msg.ReplyToMessage != nil {
		replyID := int64(msg.ReplyToMessage.MessageID)
		incoming.ReplyToMsgID = &replyID
	}

	// Determine message type and extract content
	switch {
	case msg.Text != "":
		incoming.MessageType = MessageTypeText
		incoming.Text = msg.Text

	case len(msg.Photo) > 0:
		incoming.MessageType = MessageTypePhoto
		// Get the largest photo (last in array)
		photo := msg.Photo[len(msg.Photo)-1]
		incoming.MediaFileID = photo.FileID
		incoming.MediaFileSize = int64(photo.FileSize)
		incoming.Caption = msg.Caption

	case msg.Video != nil:
		incoming.MessageType = MessageTypeVideo
		incoming.MediaFileID = msg.Video.FileID
		incoming.MediaMimeType = msg.Video.MimeType
		incoming.MediaFileSize = int64(msg.Video.FileSize)
		incoming.Caption = msg.Caption

	case msg.Audio != nil:
		incoming.MessageType = MessageTypeAudio
		incoming.MediaFileID = msg.Audio.FileID
		incoming.MediaMimeType = msg.Audio.MimeType
		incoming.MediaFileName = msg.Audio.FileName
		incoming.MediaFileSize = int64(msg.Audio.FileSize)
		incoming.Caption = msg.Caption

	case msg.Voice != nil:
		incoming.MessageType = MessageTypeVoice
		incoming.MediaFileID = msg.Voice.FileID
		incoming.MediaMimeType = msg.Voice.MimeType
		incoming.MediaFileSize = int64(msg.Voice.FileSize)

	case msg.Document != nil:
		incoming.MessageType = MessageTypeDocument
		incoming.MediaFileID = msg.Document.FileID
		incoming.MediaMimeType = msg.Document.MimeType
		incoming.MediaFileName = msg.Document.FileName
		incoming.MediaFileSize = int64(msg.Document.FileSize)
		incoming.Caption = msg.Caption

	case msg.Sticker != nil:
		incoming.MessageType = MessageTypeSticker
		incoming.MediaFileID = msg.Sticker.FileID
		incoming.MediaFileSize = int64(msg.Sticker.FileSize)

	case msg.Location != nil:
		incoming.MessageType = MessageTypeLocation
		incoming.Location = &Location{
			Latitude:  msg.Location.Latitude,
			Longitude: msg.Location.Longitude,
		}

	case msg.Contact != nil:
		incoming.MessageType = MessageTypeContact
		incoming.Contact = &Contact{
			PhoneNumber: msg.Contact.PhoneNumber,
			FirstName:   msg.Contact.FirstName,
			LastName:    msg.Contact.LastName,
			UserID:      msg.Contact.UserID,
		}

	case msg.VideoNote != nil:
		incoming.MessageType = MessageTypeVideoNote
		incoming.MediaFileID = msg.VideoNote.FileID
		incoming.MediaFileSize = int64(msg.VideoNote.FileSize)

	default:
		// Unknown message type, treat as text with empty content
		incoming.MessageType = MessageTypeText
		incoming.Text = ""
	}

	// Handle callback query data
	if update.CallbackQuery != nil {
		incoming.Text = update.CallbackQuery.Data
		incoming.MessageType = MessageTypeText
	}

	return incoming
}
