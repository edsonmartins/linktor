package direto

import (
	"context"
	"errors"
	"fmt"

	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for the VendaX Direto channel.
type SenderFactory struct{}

// NewSenderFactory creates a Direto sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "direto" }

// New builds a Sender from a channel's merged credentials+config. Requires
// send_url, instance_id and api_token.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := Config{
		SendURL:    creds["send_url"],
		InstanceID: creds["instance_id"],
		APIToken:   creds["api_token"],
	}
	if cfg.SendURL == "" {
		return nil, fmt.Errorf("send_url is required")
	}
	if cfg.InstanceID == "" {
		return nil, fmt.Errorf("instance_id is required")
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("api_token is required")
	}
	return &diretoSender{client: NewClient(cfg)}, nil
}

// diretoSender implements outbound.Sender for the Direto channel.
type diretoSender struct {
	client *Client
}

// Send translates a typed outbound.Message into a Direto send call. Supports
// text and media; interactive falls back to its body text (Direto has no native
// quick-reply on this channel), so no message is dropped.
func (s *diretoSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient address is required")
	}

	id, err := s.dispatch(ctx, msg)
	if err != nil {
		return nil, classifyError(err)
	}
	return &outbound.Receipt{ProviderMessageID: id}, nil
}

func (s *diretoSender) dispatch(ctx context.Context, msg *outbound.Message) (string, error) {
	switch c := msg.Content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return "", outbound.Permanentf("text body is required")
		}
		return s.client.SendText(ctx, msg.To, c.Body)

	case outbound.Media:
		if c.URL == "" {
			return "", outbound.Permanentf("direto media requires a URL")
		}
		return s.client.SendMedia(ctx, msg.To, string(c.Type), c.URL, c.Caption)

	case outbound.Interactive:
		if c.Body == "" {
			return "", outbound.Permanentf("interactive message requires body")
		}
		return s.client.SendText(ctx, msg.To, c.Body)

	default:
		return "", outbound.Permanentf("unsupported content kind %q for direto", msg.Content.Kind())
	}
}

// classifyError marks Direto API 4xx (bad recipient/token) as permanent; 5xx and
// network stay transient for NATS/DLQ retry.
func classifyError(err error) error {
	if outbound.IsPermanent(err) {
		return err
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Permanent() {
			return outbound.Permanent(err)
		}
	}
	return err
}
