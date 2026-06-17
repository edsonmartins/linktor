package rcs

import (
	"context"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for RCS Business Messaging.
type SenderFactory struct{}

// NewSenderFactory creates an RCS sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "rcs" }

// New builds a Sender from a channel's RCS credentials.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := &Config{
		Provider:      Provider(creds["provider"]),
		AgentID:       creds["agent_id"],
		APIKey:        creds["api_key"],
		APISecret:     creds["api_secret"],
		WebhookURL:    creds["webhook_url"],
		WebhookSecret: creds["webhook_secret"],
		BaseURL:       creds["base_url"],
		SenderID:      creds["sender_id"],
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &rcsSender{client: client}, nil
}

type rcsSender struct {
	client *Client
}

func (s *rcsSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient phone is required")
	}

	out := &OutboundMessage{To: msg.To}
	switch c := msg.Content.(type) {
	case outbound.Text:
		out.Text = c.Body
	case outbound.Template:
		out.Text = strings.TrimSpace(strings.Join(c.BodyParams, " "))
	case outbound.Media:
		out.MediaURL = c.URL
		out.MediaType = string(c.Type)
		out.Text = c.Caption
	default:
		return nil, outbound.Permanentf("unsupported content kind %q", msg.Content.Kind())
	}
	if out.Text == "" && out.MediaURL == "" {
		return nil, outbound.Permanentf("empty rcs payload")
	}

	res, err := s.client.SendMessage(ctx, out)
	if err != nil {
		return nil, err
	}
	if res == nil || !res.Success {
		reason := "rcs send failed"
		if res != nil && res.Error != "" {
			reason = res.Error
		}
		return nil, fmt.Errorf("%s", reason)
	}
	return &outbound.Receipt{ProviderMessageID: res.MessageID}, nil
}
