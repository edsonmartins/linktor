package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for Telegram. Each channel
// (bot) gets its own Sender holding only that bot's API client.
type SenderFactory struct{}

// NewSenderFactory creates a Telegram sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "telegram" }

// New builds a Sender from a channel's credentials (requires bot_token).
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	botToken := creds["bot_token"]
	if botToken == "" {
		return nil, fmt.Errorf("bot_token is required")
	}
	client, err := NewClient(botToken)
	if err != nil {
		return nil, err
	}
	return &telegramSender{client: client}, nil
}

type telegramSender struct {
	client *Client
}

// Send translates a typed outbound.Message into a Telegram Bot API call.
func (s *telegramSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	chatID, err := strconv.ParseInt(strings.TrimSpace(msg.To), 10, 64)
	if err != nil {
		return nil, outbound.Permanentf("invalid telegram chat_id %q", msg.To)
	}

	providerID, err := s.dispatch(chatID, msg.Content)
	if err != nil {
		return nil, classifyError(err)
	}
	return &outbound.Receipt{ProviderMessageID: providerID}, nil
}

// classifyError marks Telegram failures as permanent when the Bot API returned
// a 4xx that is not a rate limit (bad chat, blocked bot, bad payload). 429 and
// 5xx stay transient for NATS/DLQ retry. Already-permanent errors pass through.
func classifyError(err error) error {
	if outbound.IsPermanent(err) {
		return err
	}
	var tgErr *tgbotapi.Error
	if errors.As(err, &tgErr) {
		if tgErr.Code == 429 || tgErr.Code >= 500 {
			return err // transient
		}
		if tgErr.Code >= 400 {
			return outbound.Permanent(err) // 4xx (non-rate-limit) → permanent
		}
	}
	return err // network/unknown → transient
}

func (s *telegramSender) dispatch(chatID int64, content outbound.Content) (string, error) {
	switch c := content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return "", outbound.Permanentf("text body is required")
		}
		m, err := s.client.SendMessage(chatID, c.Body, "", 0)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(m.MessageID), nil

	case outbound.Template:
		// Telegram has no template concept; fall back to a plain-text rendering
		// of the body parameters (best effort).
		text := strings.TrimSpace(strings.Join(c.BodyParams, " "))
		if text == "" {
			text = c.Name
		}
		if text == "" {
			return "", outbound.Permanentf("empty telegram template payload")
		}
		m, err := s.client.SendMessage(chatID, text, "", 0)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(m.MessageID), nil

	case outbound.Media:
		return s.sendMedia(chatID, c)

	default:
		return "", outbound.Permanentf("unsupported content kind %q", content.Kind())
	}
}

func (s *telegramSender) sendMedia(chatID int64, m outbound.Media) (string, error) {
	url := m.URL
	if url == "" {
		return "", outbound.Permanentf("telegram media requires a URL")
	}

	switch m.Type {
	case outbound.MediaImage:
		res, e := s.client.SendPhoto(chatID, url, m.Caption, 0)
		return strconv.Itoa(res.MessageID), e
	case outbound.MediaVideo:
		res, e := s.client.SendVideo(chatID, url, m.Caption, 0)
		return strconv.Itoa(res.MessageID), e
	case outbound.MediaAudio:
		res, e := s.client.SendAudio(chatID, url, m.Caption, 0)
		return strconv.Itoa(res.MessageID), e
	case outbound.MediaDocument:
		res, e := s.client.SendDocument(chatID, url, m.Caption, 0)
		return strconv.Itoa(res.MessageID), e
	default:
		return "", outbound.Permanentf("unsupported media type %q", m.Type)
	}
}
