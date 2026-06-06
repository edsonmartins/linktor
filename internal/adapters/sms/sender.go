package sms

import (
	"context"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/outbound"
)

// SenderFactory builds stateless per-channel Senders for Twilio SMS.
type SenderFactory struct{}

// NewSenderFactory creates a Twilio SMS sender factory.
func NewSenderFactory() *SenderFactory { return &SenderFactory{} }

// ChannelType implements outbound.Factory.
func (SenderFactory) ChannelType() string { return "sms" }

// New builds a Sender from a channel's Twilio credentials.
func (SenderFactory) New(creds map[string]string) (outbound.Sender, error) {
	cfg := &TwilioConfig{
		AccountSID:          creds["account_sid"],
		AuthToken:           creds["auth_token"],
		APIKeySID:           creds["api_key_sid"],
		APIKeySecret:        creds["api_key_secret"],
		PhoneNumber:         creds["phone_number"],
		MessagingServiceSID: creds["messaging_service_sid"],
		StatusCallbackURL:   creds["status_callback_url"],
	}
	if cfg.AccountSID == "" {
		return nil, fmt.Errorf("account_sid is required")
	}
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &smsSender{client: client}, nil
}

type smsSender struct {
	client *Client
}

func (s *smsSender) Send(ctx context.Context, msg *outbound.Message) (*outbound.Receipt, error) {
	if msg.To == "" {
		return nil, outbound.Permanentf("recipient phone is required")
	}

	body, mediaURLs := smsBody(msg.Content)
	if body == "" && len(mediaURLs) == 0 {
		return nil, outbound.Permanentf("empty sms payload")
	}

	res, err := s.client.SendMessage(msg.To, body, mediaURLs)
	if err != nil {
		return nil, err
	}
	if res == nil || !res.Success {
		reason := "sms send failed"
		if res != nil && res.Error != "" {
			reason = res.Error
		}
		return nil, fmt.Errorf("%s", reason)
	}
	return &outbound.Receipt{ProviderMessageID: res.MessageSID}, nil
}

// smsBody flattens typed content into an SMS body + optional media URLs.
func smsBody(content outbound.Content) (string, []string) {
	switch c := content.(type) {
	case outbound.Text:
		return c.Body, nil
	case outbound.Template:
		return strings.TrimSpace(strings.Join(c.BodyParams, " ")), nil
	case outbound.Media:
		if c.URL != "" {
			return c.Caption, []string{c.URL}
		}
		return c.Caption, nil
	default:
		return "", nil
	}
}
