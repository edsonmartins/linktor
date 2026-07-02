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

	email := buildOutboundEmail(msg)
	if email.TextBody == "" && email.HTMLBody == "" && len(email.Attachments) == 0 {
		return nil, outbound.Permanentf("empty email body")
	}

	res, err := s.client.Send(ctx, email)
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

// buildOutboundEmail maps the unified outbound message onto a full OutboundEmail
// so the production send path preserves threading (In-Reply-To/References), the
// HTML alternative, Reply-To, CC/BCC and media attachments — all previously
// dropped by the text-only SendText path, which broke reply threading and made
// every agent reply start a new thread in the recipient's inbox.
func buildOutboundEmail(msg *outbound.Message) *OutboundEmail {
	subject, body := emailSubjectBody(msg)

	email := &OutboundEmail{
		To:         []string{msg.To},
		Subject:    subject,
		TextBody:   body,
		HTMLBody:   msg.Meta("html_body"),
		ReplyTo:    msg.Meta("reply_to"),
		InReplyTo:  msg.Meta("in_reply_to"),
		References: msg.Meta("references"),
	}
	if cc := msg.Meta("cc"); cc != "" {
		email.CC = splitAddresses(cc)
	}
	if bcc := msg.Meta("bcc"); bcc != "" {
		email.BCC = splitAddresses(bcc)
	}

	// A media reply becomes an attachment (URL-hosted); the caption is the body.
	if m, ok := msg.Content.(outbound.Media); ok && m.URL != "" {
		email.Attachments = append(email.Attachments, &Attachment{
			Filename:    m.Filename,
			ContentType: msg.Meta("mime_type"),
			URL:         m.URL,
		})
	}
	return email
}

// splitAddresses splits a comma-separated address list, trimming whitespace and
// dropping empties.
func splitAddresses(list string) []string {
	parts := strings.Split(list, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// emailSubjectBody derives the subject (from the "subject"/"email_subject"
// metadata, defaulting to "Message") and a text body from the content.
func emailSubjectBody(msg *outbound.Message) (subject, body string) {
	subject = msg.Meta("subject")
	if subject == "" {
		subject = msg.Meta("email_subject")
	}
	if subject == "" {
		subject = "Message"
	}

	switch c := msg.Content.(type) {
	case outbound.Text:
		body = c.Body
	case outbound.Template:
		body = strings.TrimSpace(strings.Join(c.BodyParams, " "))
	case outbound.Media:
		body = strings.TrimSpace(c.Caption)
	}
	return subject, body
}
