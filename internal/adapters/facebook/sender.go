package facebook

import (
	"context"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for Facebook Messenger.
type SenderFactory struct{}

// NewSenderFactory creates a Facebook Messenger sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "facebook" }

// New builds a Sender from a channel's Facebook page credentials.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := &FacebookConfig{
		AppID:           creds["app_id"],
		AppSecret:       creds["app_secret"],
		PageID:          creds["page_id"],
		PageAccessToken: creds["page_access_token"],
		UserAccessToken: creds["user_access_token"],
		VerifyToken:     creds["verify_token"],
		InstagramID:     creds["instagram_id"],
	}
	if cfg.PageAccessToken == "" {
		return nil, fmt.Errorf("page_access_token is required")
	}
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &fbSender{client: client}, nil
}

type fbSender struct {
	client *Client
}

func (s *fbSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient PSID is required")
	}

	switch c := msg.Content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return nil, outbound.Permanentf("text body is required")
		}
		resp, err := s.client.SendTextMessage(ctx, msg.To, c.Body)
		if err != nil {
			return nil, err
		}
		return &outbound.Receipt{ProviderMessageID: resp.MessageID}, nil

	case outbound.Template:
		text := strings.TrimSpace(strings.Join(c.BodyParams, " "))
		if text == "" {
			return nil, outbound.Permanentf("empty template payload")
		}
		resp, err := s.client.SendTextMessage(ctx, msg.To, text)
		if err != nil {
			return nil, err
		}
		return &outbound.Receipt{ProviderMessageID: resp.MessageID}, nil

	case outbound.Media:
		if c.URL == "" {
			return nil, outbound.Permanentf("media requires a URL")
		}
		switch c.Type {
		case outbound.MediaImage:
			return receiptOrErr(s.client.SendImage(ctx, msg.To, c.URL))
		case outbound.MediaVideo:
			return receiptOrErr(s.client.SendVideo(ctx, msg.To, c.URL))
		case outbound.MediaAudio:
			return receiptOrErr(s.client.SendAudio(ctx, msg.To, c.URL))
		default:
			return receiptOrErr(s.client.SendFile(ctx, msg.To, c.URL))
		}

	default:
		return nil, outbound.Permanentf("unsupported content kind %q", msg.Content.Kind())
	}
}

func receiptOrErr(resp *meta.SendMessageResponse, err error) (*outbound.Receipt, error) {
	if err != nil {
		return nil, err
	}
	return &outbound.Receipt{ProviderMessageID: resp.MessageID}, nil
}
