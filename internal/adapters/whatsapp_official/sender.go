package whatsapp_official

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for the WhatsApp Cloud API.
// Unlike the inbound Adapter (a shared, connection-stateful instance), each
// channel gets its own Sender holding only that channel's HTTP client.
type SenderFactory struct{}

// NewSenderFactory creates a WhatsApp Cloud API sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "whatsapp_official" }

// New builds a Sender from a channel's credentials. Returns an error if the
// mandatory Cloud API credentials are absent.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := &Config{
		AccessToken:   creds["access_token"],
		PhoneNumberID: creds["phone_number_id"],
		BusinessID:    creds["business_id"],
		VerifyToken:   creds["verify_token"],
		WebhookSecret: creds["webhook_secret"],
		APIVersion:    getOrDefault(creds, "api_version", DefaultAPIVersion),
	}
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("access_token is required")
	}
	if cfg.PhoneNumberID == "" {
		return nil, fmt.Errorf("phone_number_id is required")
	}

	client := NewClient(cfg)
	return &cloudSender{
		client:   client,
		template: NewTemplateSender(client),
	}, nil
}

// cloudSender implements outbound.Sender for WhatsApp Cloud API.
type cloudSender struct {
	client   *Client
	template *TemplateSender
}

// Send translates a typed outbound.Message into the appropriate Cloud API call.
func (s *cloudSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient address is required")
	}

	resp, err := s.dispatch(ctx, msg)
	if err != nil {
		return nil, classifyError(err)
	}
	if resp == nil || len(resp.Messages) == 0 {
		return nil, outbound.Permanentf("provider returned no message ID")
	}
	return &outbound.Receipt{ProviderMessageID: resp.Messages[0].ID}, nil
}

// classifyError marks Cloud API failures as permanent (do not retry) when Meta
// returned a 4xx that is not a rate limit — invalid template, bad recipient,
// auth. Rate limits and 5xx stay transient so NATS/DLQ retry handles them.
// Errors already tagged permanent (payload validation) pass through unchanged.
func classifyError(err error) error {
	if outbound.IsPermanent(err) {
		return err
	}
	var apiErr *APIRequestError
	if errors.As(err, &apiErr) {
		if apiErr.IsRateLimitError() || apiErr.StatusCode >= 500 {
			return err // transient
		}
		return outbound.Permanent(err) // 4xx (non-rate-limit) → permanent
	}
	return err // network/unknown → transient
}

func (s *cloudSender) dispatch(ctx context.Context, msg *outbound.Message) (*SendMessageResponse, error) {
	switch c := msg.Content.(type) {
	case outbound.Text:
		if c.Body == "" {
			return nil, outbound.Permanentf("text body is required")
		}
		return s.client.SendTextMessage(ctx, msg.To, c.Body, c.PreviewURL)

	case outbound.Template:
		return s.sendTemplate(ctx, msg.To, c)

	case outbound.Media:
		return s.sendMedia(ctx, msg.To, c)

	default:
		return nil, outbound.Permanentf("unsupported content kind %q", msg.Content.Kind())
	}
}

func (s *cloudSender) sendTemplate(ctx context.Context, to string, t outbound.Template) (*SendMessageResponse, error) {
	if t.Name == "" {
		return nil, outbound.Permanentf("template name is required")
	}
	lang := t.Language
	if lang == "" {
		lang = "en"
	}

	// A provider-native components payload (header/buttons/carousel) takes
	// precedence; otherwise fall back to positional body params.
	if t.ComponentsJSON != "" {
		var components []TemplateComponent
		if err := json.Unmarshal([]byte(t.ComponentsJSON), &components); err != nil {
			return nil, outbound.Permanentf("invalid template components: %v", err)
		}
		tmpl := &TemplateObject{
			Name:       t.Name,
			Language:   &TemplateLanguage{Code: lang},
			Components: components,
		}
		return s.template.SendTemplate(ctx, to, tmpl)
	}

	return s.template.SendTemplateWithBodyParams(ctx, to, t.Name, lang, t.BodyParams...)
}

func (s *cloudSender) sendMedia(ctx context.Context, to string, m outbound.Media) (*SendMessageResponse, error) {
	if m.URL == "" && m.MediaID == "" {
		return nil, outbound.Permanentf("media requires a URL or media ID")
	}
	switch m.Type {
	case outbound.MediaImage:
		return s.client.SendImageMessage(ctx, to, &MediaObject{ID: m.MediaID, Link: m.URL, Caption: m.Caption})
	case outbound.MediaVideo:
		return s.client.SendVideoMessage(ctx, to, &MediaObject{ID: m.MediaID, Link: m.URL, Caption: m.Caption})
	case outbound.MediaAudio:
		return s.client.SendAudioMessage(ctx, to, &MediaObject{ID: m.MediaID, Link: m.URL})
	case outbound.MediaDocument:
		return s.client.SendDocumentMessage(ctx, to, &DocumentObject{ID: m.MediaID, Link: m.URL, Caption: m.Caption, Filename: m.Filename})
	default:
		return nil, outbound.Permanentf("unsupported media type %q", m.Type)
	}
}
