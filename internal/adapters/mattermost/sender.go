package mattermost

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/outbound"
	"github.com/msgfy/linktor/pkg/logger"
)

// SenderFactory builds stateless per-channel Senders for Mattermost.
type SenderFactory struct{}

// NewSenderFactory creates a Mattermost sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return ChannelType }

// New builds a Sender from a channel's credentials (requires base_url + bot_token).
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := configFromCreds(creds)
	if cfg.BaseURL == "" || cfg.BotToken == "" {
		return nil, fmt.Errorf("mattermost: base_url and bot_token are required")
	}
	return &mattermostSender{client: NewClient(cfg)}, nil
}

type mattermostSender struct {
	client *Client
}

// Send posts a message to a Mattermost channel. msg.To is the channel id (the
// stable per-conversation address captured on inbound).
func (s *mattermostSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	channelID := strings.TrimSpace(msg.To)
	if channelID == "" {
		return nil, outbound.Permanentf("mattermost: empty channel id")
	}

	// Native upload for media: fetch the (Linktor-hosted) bytes and attach them as
	// a real file. Falls back to posting the link on any failure so delivery is
	// never lost.
	if media, ok := msg.Content.(outbound.Media); ok && media.URL != "" {
		if postID, err := s.sendMedia(ctx, channelID, media); err == nil {
			return &outbound.Receipt{ProviderMessageID: postID}, nil
		} else {
			logger.Warn("mattermost: native media upload failed, falling back to link: " + err.Error())
		}
	}

	text, err := renderText(msg.Content)
	if err != nil {
		return nil, err
	}

	postID, err := s.client.CreatePost(ctx, channelID, text)
	if err != nil {
		return nil, classifyError(err)
	}
	return &outbound.Receipt{ProviderMessageID: postID}, nil
}

// sendMedia uploads the media as a native file attachment with the caption as
// the post body, returning the post id.
func (s *mattermostSender) sendMedia(ctx context.Context, channelID string, media outbound.Media) (string, error) {
	data, filename, err := outbound.FetchMedia(ctx, media.URL, media.Filename)
	if err != nil {
		return "", err
	}
	fileID, err := s.client.UploadFile(ctx, channelID, filename, data)
	if err != nil {
		return "", err
	}
	return s.client.CreatePostWithFiles(ctx, channelID, media.Caption, []string{fileID})
}

// renderText flattens typed content to a Mattermost message. Media is sent as a
// link (caption + url); native file uploads via /api/v4/files are a follow-up.
func renderText(content outbound.Content) (string, error) {
	switch c := content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return "", outbound.Permanentf("mattermost: text body is required")
		}
		return c.Body, nil

	case outbound.Template:
		text := strings.TrimSpace(strings.Join(c.BodyParams, " "))
		if text == "" {
			text = c.Name
		}
		if text == "" {
			return "", outbound.Permanentf("mattermost: empty template payload")
		}
		return text, nil

	case outbound.Media:
		if c.URL == "" {
			return "", outbound.Permanentf("mattermost: media requires a URL")
		}
		if c.Caption != "" {
			return c.Caption + "\n" + c.URL, nil
		}
		return c.URL, nil

	default:
		return "", outbound.Permanentf("mattermost: unsupported content kind %q", content.Kind())
	}
}

// classifyError marks 4xx (non-429) failures as permanent; 429 and 5xx stay
// transient for NATS/DLQ retry.
func classifyError(err error) error {
	if outbound.IsPermanent(err) {
		return err
	}
	var he *httpError
	if errors.As(err, &he) {
		if he.StatusCode == 429 || he.StatusCode >= 500 {
			return err // transient
		}
		if he.StatusCode >= 400 {
			return outbound.Permanent(err)
		}
	}
	return err // network/unknown → transient
}
