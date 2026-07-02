package slack

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/outbound"
	"github.com/msgfy/linktor/pkg/logger"
)

// SenderFactory builds stateless per-channel Senders for Slack.
type SenderFactory struct{}

// NewSenderFactory creates a Slack sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return ChannelType }

// New builds a Sender from a channel's credentials (requires bot_token).
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	botToken := creds[CredBotToken]
	if botToken == "" {
		return nil, fmt.Errorf("slack: bot_token is required")
	}
	return &slackSender{client: NewClient(botToken)}, nil
}

type slackSender struct {
	client *Client
}

// Send posts a message to a Slack channel. msg.To is the Slack channel id (for
// DMs, the stable IM channel id captured on inbound).
func (s *slackSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	channel := strings.TrimSpace(msg.To)
	if channel == "" {
		return nil, outbound.Permanentf("slack: empty channel id")
	}

	// Native upload for media: fetch the (Linktor-hosted) bytes and share them as
	// a real file. Falls back to posting the link on any failure so delivery is
	// never lost.
	if media, ok := msg.Content.(outbound.Media); ok && media.URL != "" {
		if fileID, err := s.sendMedia(ctx, channel, media); err == nil {
			return &outbound.Receipt{ProviderMessageID: fileID}, nil
		} else {
			logger.Warn("slack: native media upload failed, falling back to link: " + err.Error())
		}
	}

	text, err := renderText(msg.Content)
	if err != nil {
		return nil, err
	}

	ts, err := s.client.PostMessage(ctx, channel, text)
	if err != nil {
		return nil, classifyError(err)
	}
	return &outbound.Receipt{ProviderMessageID: ts}, nil
}

// sendMedia shares the media as a native file upload, with the caption as the
// initial comment, returning the new file id.
func (s *slackSender) sendMedia(ctx context.Context, channel string, media outbound.Media) (string, error) {
	data, filename, err := outbound.FetchMedia(ctx, media.URL, media.Filename)
	if err != nil {
		return "", err
	}
	return s.client.UploadFile(ctx, channel, filename, media.Caption, data)
}

// renderText flattens typed content to Slack text. Media is sent as a link
// (caption + url); native file uploads are a follow-up.
func renderText(content outbound.Content) (string, error) {
	switch c := content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return "", outbound.Permanentf("slack: text body is required")
		}
		return c.Body, nil

	case outbound.Template:
		text := strings.TrimSpace(strings.Join(c.BodyParams, " "))
		if text == "" {
			text = c.Name
		}
		if text == "" {
			return "", outbound.Permanentf("slack: empty template payload")
		}
		return text, nil

	case outbound.Media:
		if c.URL == "" {
			return "", outbound.Permanentf("slack: media requires a URL")
		}
		if c.Caption != "" {
			return c.Caption + "\n" + c.URL, nil
		}
		return c.URL, nil

	default:
		return "", outbound.Permanentf("slack: unsupported content kind %q", content.Kind())
	}
}

// permanentSlackErrors are Slack API error codes that must not be retried.
var permanentSlackErrors = map[string]bool{
	"channel_not_found": true,
	"not_in_channel":    true,
	"is_archived":       true,
	"msg_too_long":      true,
	"no_text":           true,
	"invalid_auth":      true,
	"token_revoked":     true,
	"account_inactive":  true,
	"not_authed":        true,
	"restricted_action": true,
}

// classifyError marks unrecoverable Slack failures as permanent; rate limits
// and 5xx stay transient for NATS/DLQ retry.
func classifyError(err error) error {
	if outbound.IsPermanent(err) {
		return err
	}
	var ae *apiError
	if errors.As(err, &ae) {
		if ae.Err == "ratelimited" {
			return err // transient
		}
		if ae.Code == 429 || ae.Code >= 500 {
			return err // transient
		}
		if permanentSlackErrors[ae.Err] {
			return outbound.Permanent(err)
		}
		if ae.Code >= 400 && ae.Code < 500 {
			return outbound.Permanent(err)
		}
	}
	return err // network/unknown → transient
}
