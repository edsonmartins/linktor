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

	email := buildOutboundEmail(ctx, msg)
	if email.TextBody == "" && email.HTMLBody == "" && !hasContentAttachment(email.Attachments) {
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
func buildOutboundEmail(ctx context.Context, msg *outbound.Message) *OutboundEmail {
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

	// A media reply becomes a real attachment. Email providers only send the
	// attachment BYTES (Content) — none of them fetch a bare URL — so download the
	// media and attach it. If the fetch fails, fall back to putting the link in
	// the body so the media is never silently lost.
	if m, ok := msg.Content.(outbound.Media); ok && m.URL != "" {
		data, name, err := outbound.FetchMedia(ctx, m.URL, m.Filename)
		if err == nil && len(data) > 0 {
			email.Attachments = append(email.Attachments, &Attachment{
				Filename:    name,
				ContentType: msg.Meta("mime_type"),
				Content:     data,
			})
		} else {
			if email.TextBody != "" {
				email.TextBody += "\n"
			}
			email.TextBody += m.URL
		}
	}
	return email
}

// hasContentAttachment reports whether any attachment carries actual bytes, so
// the empty-body guard is not satisfied by a content-less placeholder.
func hasContentAttachment(atts []*Attachment) bool {
	for _, a := range atts {
		if len(a.Content) > 0 {
			return true
		}
	}
	return false
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
