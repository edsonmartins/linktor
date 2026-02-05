package email

import (
	"fmt"
	"time"
)

// Provider represents the email service provider
type Provider string

const (
	ProviderSMTP     Provider = "smtp"
	ProviderSendGrid Provider = "sendgrid"
	ProviderMailgun  Provider = "mailgun"
	ProviderSES      Provider = "ses"
	ProviderPostmark Provider = "postmark"
)

// Config holds the email adapter configuration
type Config struct {
	Provider  Provider `json:"provider"`
	FromEmail string   `json:"from_email"`
	FromName  string   `json:"from_name"`
	ReplyTo   string   `json:"reply_to,omitempty"`

	// SMTP Config
	SMTPHost       string `json:"smtp_host,omitempty"`
	SMTPPort       int    `json:"smtp_port,omitempty"`
	SMTPUsername   string `json:"smtp_username,omitempty"`
	SMTPPassword   string `json:"smtp_password,omitempty"`
	SMTPEncryption string `json:"smtp_encryption,omitempty"` // "tls", "starttls", "none"

	// IMAP Config (for receiving via SMTP provider)
	IMAPHost         string `json:"imap_host,omitempty"`
	IMAPPort         int    `json:"imap_port,omitempty"`
	IMAPUsername     string `json:"imap_username,omitempty"`
	IMAPPassword     string `json:"imap_password,omitempty"`
	IMAPFolder       string `json:"imap_folder,omitempty"`        // Default: "INBOX"
	IMAPPollInterval int    `json:"imap_poll_interval,omitempty"` // Seconds between polls

	// SendGrid
	SendGridAPIKey string `json:"sendgrid_api_key,omitempty"`

	// Mailgun
	MailgunDomain string `json:"mailgun_domain,omitempty"`
	MailgunAPIKey string `json:"mailgun_api_key,omitempty"`
	MailgunRegion string `json:"mailgun_region,omitempty"` // "us" or "eu"

	// AWS SES
	SESRegion      string `json:"ses_region,omitempty"`
	SESAccessKeyID string `json:"ses_access_key_id,omitempty"`
	SESSecretKey   string `json:"ses_secret_key,omitempty"`

	// Postmark
	PostmarkServerToken string `json:"postmark_server_token,omitempty"`

	// Webhook
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if c.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}

	switch c.Provider {
	case ProviderSMTP:
		if c.SMTPHost == "" {
			return fmt.Errorf("smtp_host is required for SMTP provider")
		}
		if c.SMTPPort == 0 {
			return fmt.Errorf("smtp_port is required for SMTP provider")
		}

	case ProviderSendGrid:
		if c.SendGridAPIKey == "" {
			return fmt.Errorf("sendgrid_api_key is required for SendGrid provider")
		}

	case ProviderMailgun:
		if c.MailgunDomain == "" {
			return fmt.Errorf("mailgun_domain is required for Mailgun provider")
		}
		if c.MailgunAPIKey == "" {
			return fmt.Errorf("mailgun_api_key is required for Mailgun provider")
		}

	case ProviderSES:
		if c.SESRegion == "" {
			return fmt.Errorf("ses_region is required for SES provider")
		}
		if c.SESAccessKeyID == "" {
			return fmt.Errorf("ses_access_key_id is required for SES provider")
		}
		if c.SESSecretKey == "" {
			return fmt.Errorf("ses_secret_key is required for SES provider")
		}

	case ProviderPostmark:
		if c.PostmarkServerToken == "" {
			return fmt.Errorf("postmark_server_token is required for Postmark provider")
		}

	default:
		return fmt.Errorf("unsupported provider: %s", c.Provider)
	}

	return nil
}

// OutboundEmail represents an email to be sent
type OutboundEmail struct {
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	TextBody    string            `json:"text_body,omitempty"`
	HTMLBody    string            `json:"html_body,omitempty"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []*Attachment     `json:"attachments,omitempty"`
	InReplyTo   string            `json:"in_reply_to,omitempty"` // Message-ID for threading
	References  string            `json:"references,omitempty"`  // Thread references
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// IncomingEmail represents a received email
type IncomingEmail struct {
	MessageID   string            `json:"message_id"`
	From        string            `json:"from"`
	FromName    string            `json:"from_name,omitempty"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	Subject     string            `json:"subject"`
	TextBody    string            `json:"text_body,omitempty"`
	HTMLBody    string            `json:"html_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []*Attachment     `json:"attachments,omitempty"`
	InReplyTo   string            `json:"in_reply_to,omitempty"`
	References  string            `json:"references,omitempty"`
	ReceivedAt  time.Time         `json:"received_at"`
	SpamScore   float64           `json:"spam_score,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Content     []byte `json:"content,omitempty"`
	ContentID   string `json:"content_id,omitempty"` // For inline images
	URL         string `json:"url,omitempty"`        // If hosted externally
}

// SendResult represents the result of sending an email
type SendResult struct {
	Success    bool      `json:"success"`
	MessageID  string    `json:"message_id,omitempty"`
	ExternalID string    `json:"external_id,omitempty"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// EmailStatus represents the status of an email
type EmailStatus string

const (
	StatusQueued       EmailStatus = "queued"
	StatusSent         EmailStatus = "sent"
	StatusDelivered    EmailStatus = "delivered"
	StatusOpened       EmailStatus = "opened"
	StatusClicked      EmailStatus = "clicked"
	StatusBounced      EmailStatus = "bounced"
	StatusSpam         EmailStatus = "spam"
	StatusUnsubscribed EmailStatus = "unsubscribed"
	StatusFailed       EmailStatus = "failed"
)

// StatusCallback represents an email status update from a provider webhook
type StatusCallback struct {
	MessageID    string            `json:"message_id,omitempty"`
	ExternalID   string            `json:"external_id,omitempty"`
	Status       EmailStatus       `json:"status"`
	Recipient    string            `json:"recipient,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// WebhookPayload represents a generic webhook payload from providers
type WebhookPayload struct {
	Provider       Provider         `json:"provider"`
	Type           string           `json:"type"` // "inbound" or "status"
	IncomingEmail  *IncomingEmail   `json:"incoming_email,omitempty"`
	StatusCallback *StatusCallback  `json:"status_callback,omitempty"`
	RawPayload     []byte           `json:"raw_payload,omitempty"`
}

// SendGrid webhook types
type SendGridInboundWebhook struct {
	Headers     string `form:"headers"`
	To          string `form:"to"`
	From        string `form:"from"`
	Subject     string `form:"subject"`
	Text        string `form:"text"`
	HTML        string `form:"html"`
	Attachments int    `form:"attachments"`
	SPF         string `form:"SPF"`
	DKIM        string `form:"dkim"`
	SpamScore   string `form:"spam_score"`
	Envelope    string `form:"envelope"`
}

type SendGridEventWebhook struct {
	Email     string `json:"email"`
	Event     string `json:"event"` // delivered, bounce, open, click, dropped, deferred, spam_report
	MessageID string `json:"sg_message_id"`
	Timestamp int64  `json:"timestamp"`
	Reason    string `json:"reason,omitempty"`
	Response  string `json:"response,omitempty"`
	URL       string `json:"url,omitempty"` // for click events
}

// Mailgun webhook types
type MailgunInboundWebhook struct {
	Recipient   string `form:"recipient"`
	Sender      string `form:"sender"`
	From        string `form:"from"`
	Subject     string `form:"subject"`
	BodyPlain   string `form:"body-plain"`
	BodyHTML    string `form:"body-html"`
	MessageID   string `form:"Message-Id"`
	Timestamp   string `form:"timestamp"`
	Token       string `form:"token"`
	Signature   string `form:"signature"`
	ContentType string `form:"Content-Type"`
}

type MailgunEventWebhook struct {
	Signature struct {
		Token     string `json:"token"`
		Timestamp string `json:"timestamp"`
		Signature string `json:"signature"`
	} `json:"signature"`
	EventData struct {
		Event     string `json:"event"` // delivered, failed, opened, clicked, unsubscribed, complained
		ID        string `json:"id"`
		Timestamp float64 `json:"timestamp"`
		Message   struct {
			Headers struct {
				MessageID string `json:"message-id"`
			} `json:"headers"`
		} `json:"message"`
		Recipient string `json:"recipient"`
		Severity  string `json:"severity,omitempty"` // for failed events: permanent, temporary
	} `json:"event-data"`
}

// AWS SES webhook types (via SNS)
type SESNotification struct {
	Type         string `json:"Type"`
	Message      string `json:"Message"` // JSON encoded SES notification
	MessageId    string `json:"MessageId"`
	TopicArn     string `json:"TopicArn"`
	Timestamp    string `json:"Timestamp"`
	SubscribeURL string `json:"SubscribeURL,omitempty"` // For subscription confirmation
}

type SESMessage struct {
	NotificationType string       `json:"notificationType"` // Bounce, Complaint, Delivery
	Bounce           *SESBounce   `json:"bounce,omitempty"`
	Complaint        *SESComplaint `json:"complaint,omitempty"`
	Delivery         *SESDelivery  `json:"delivery,omitempty"`
	Mail             *SESMail     `json:"mail"`
}

type SESBounce struct {
	BounceType        string              `json:"bounceType"` // Permanent, Transient
	BounceSubType     string              `json:"bounceSubType"`
	BouncedRecipients []SESBouncedRecipient `json:"bouncedRecipients"`
	Timestamp         string              `json:"timestamp"`
}

type SESBouncedRecipient struct {
	EmailAddress   string `json:"emailAddress"`
	Action         string `json:"action,omitempty"`
	Status         string `json:"status,omitempty"`
	DiagnosticCode string `json:"diagnosticCode,omitempty"`
}

type SESComplaint struct {
	ComplainedRecipients []SESComplainedRecipient `json:"complainedRecipients"`
	ComplaintFeedbackType string                   `json:"complaintFeedbackType,omitempty"`
	Timestamp            string                   `json:"timestamp"`
}

type SESComplainedRecipient struct {
	EmailAddress string `json:"emailAddress"`
}

type SESDelivery struct {
	Timestamp            string   `json:"timestamp"`
	ProcessingTimeMillis int64    `json:"processingTimeMillis"`
	Recipients           []string `json:"recipients"`
	SMTPResponse         string   `json:"smtpResponse"`
}

type SESMail struct {
	Timestamp        string   `json:"timestamp"`
	MessageId        string   `json:"messageId"`
	Source           string   `json:"source"`
	Destination      []string `json:"destination"`
	CommonHeaders    map[string]interface{} `json:"commonHeaders,omitempty"`
}

// Postmark webhook types
type PostmarkInboundWebhook struct {
	From          string                 `json:"From"`
	FromName      string                 `json:"FromName"`
	To            string                 `json:"To"`
	CC            string                 `json:"Cc"`
	Subject       string                 `json:"Subject"`
	TextBody      string                 `json:"TextBody"`
	HtmlBody      string                 `json:"HtmlBody"`
	MessageID     string                 `json:"MessageID"`
	ReplyTo       string                 `json:"ReplyTo"`
	OriginalRecipient string             `json:"OriginalRecipient"`
	Attachments   []PostmarkAttachment   `json:"Attachments"`
	Headers       []PostmarkHeader       `json:"Headers"`
}

type PostmarkAttachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"` // Base64 encoded
	ContentType string `json:"ContentType"`
	ContentID   string `json:"ContentID,omitempty"`
	ContentLength int64 `json:"ContentLength"`
}

type PostmarkHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type PostmarkEventWebhook struct {
	RecordType  string `json:"RecordType"` // Delivery, Bounce, SpamComplaint, Open, Click
	MessageID   string `json:"MessageID"`
	Recipient   string `json:"Recipient"`
	DeliveredAt string `json:"DeliveredAt,omitempty"`
	BouncedAt   string `json:"BouncedAt,omitempty"`
	Type        string `json:"Type,omitempty"`     // Bounce type
	TypeCode    int    `json:"TypeCode,omitempty"` // Bounce type code
	Description string `json:"Description,omitempty"`
}
