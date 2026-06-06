package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for email (SMTP/SendGrid/
// Mailgun/SES/Postmark).
type SenderFactory struct{}

// NewSenderFactory creates an email sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "email" }

// New builds a Sender from a channel's email credentials.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := ConfigFromMap(creds)
	if cfg.FromEmail == "" {
		return nil, fmt.Errorf("from_email is required")
	}
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &emailSender{client: client}, nil
}

type emailSender struct {
	client *Client
}

func (s *emailSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient email is required")
	}

	subject, body := emailSubjectBody(msg)
	if body == "" {
		return nil, outbound.Permanentf("empty email body")
	}

	res, err := s.client.SendText(ctx, msg.To, subject, body)
	if err != nil {
		return nil, err
	}
	if res == nil || !res.Success {
		reason := "email send failed"
		if res != nil && res.Error != "" {
			reason = res.Error
		}
		return nil, fmt.Errorf("%s", reason)
	}
	id := res.MessageID
	if id == "" {
		id = res.ExternalID
	}
	return &outbound.Receipt{ProviderMessageID: id}, nil
}

// emailSubjectBody derives a subject and text body from the content. The
// outbound model carries no subject, so a generic default is used; richer
// subject handling can be added if the typed model grows a field for it.
func emailSubjectBody(msg *outbound.Message) (subject, body string) {
	subject = "Message"

	switch c := msg.Content.(type) {
	case outbound.Text:
		body = c.Body
	case outbound.Template:
		body = strings.TrimSpace(strings.Join(c.BodyParams, " "))
	case outbound.Media:
		body = strings.TrimSpace(c.Caption + "\n" + c.URL)
	}
	return subject, body
}
