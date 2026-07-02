package teams

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for Teams. Each channel
// (bot) gets its own Sender holding that bot's Bot Connector client.
type SenderFactory struct{}

// NewSenderFactory creates a Teams sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return ChannelType }

// New builds a Sender from a channel's credentials (requires app_id/app_password).
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := configFromCreds(creds)
	if cfg.AppID == "" || cfg.AppPassword == "" {
		return nil, fmt.Errorf("teams: app_id and app_password are required")
	}
	return &teamsSender{client: NewClient(cfg)}, nil
}

type teamsSender struct {
	client *Client
}

// Send delivers a message to a Teams conversation. msg.To is the Bot Framework
// conversation id (the stable per-conversation address captured on inbound);
// the service URL comes from the channel config or the message metadata.
func (s *teamsSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	conversationID := strings.TrimSpace(msg.To)
	if conversationID == "" {
		return nil, outbound.Permanentf("teams: empty conversation id")
	}

	activity, err := buildOutboundActivity(msg.Content)
	if err != nil {
		return nil, err
	}

	serviceURL := msg.Meta("service_url")
	providerID, err := s.client.SendActivity(ctx, serviceURL, conversationID, activity)
	if err != nil {
		return nil, classifyError(err)
	}
	return &outbound.Receipt{ProviderMessageID: providerID}, nil
}

// buildOutboundActivity converts typed outbound content into a Teams Activity.
// Rich content (Adaptive Cards) is out of scope for v1: text + media URLs first.
func buildOutboundActivity(content outbound.Content) (*Activity, error) {
	activity := &Activity{Type: "message"}

	switch c := content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return nil, outbound.Permanentf("teams: text body is required")
		}
		activity.Text = c.Body

	case outbound.Template:
		text := strings.TrimSpace(strings.Join(c.BodyParams, " "))
		if text == "" {
			text = c.Name
		}
		if text == "" {
			return nil, outbound.Permanentf("teams: empty template payload")
		}
		activity.Text = text

	case outbound.Media:
		if c.URL == "" {
			return nil, outbound.Permanentf("teams: media requires a URL")
		}
		activity.Text = c.Caption
		activity.Attachments = []Attachment{{
			ContentType: mediaContentType(c.Type),
			ContentURL:  c.URL,
			Name:        c.Filename,
		}}

	default:
		return nil, outbound.Permanentf("teams: unsupported content kind %q", content.Kind())
	}

	return activity, nil
}

func mediaContentType(t outbound.MediaType) string {
	switch t {
	case outbound.MediaImage:
		return "image/*"
	case outbound.MediaVideo:
		return "video/*"
	case outbound.MediaAudio:
		return "audio/*"
	default:
		return "application/octet-stream"
	}
}

// classifyError marks 4xx (non-429) Bot Connector failures as permanent so the
// worker does not retry them; 429 and 5xx stay transient for NATS/DLQ retry.
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
