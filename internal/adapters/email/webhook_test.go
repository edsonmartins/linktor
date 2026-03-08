package email

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSendGridWebhook(t *testing.T) {
	t.Run("inbound email with multipart content-type", func(t *testing.T) {
		form := url.Values{}
		form.Set("from", "sender@example.com")
		form.Set("to", "recipient@example.com")
		form.Set("subject", "Test Subject")
		form.Set("text", "Hello World")
		form.Set("html", "<p>Hello World</p>")
		form.Set("spam_score", "1.5")
		form.Set("SPF", "pass")
		form.Set("dkim", "pass")
		form.Set("attachments", "0")

		body := []byte(form.Encode())
		headers := map[string]string{
			"Content-Type": "multipart/form-data",
		}

		payload, err := ParseWebhook(ProviderSendGrid, body, headers)
		require.NoError(t, err)
		assert.Equal(t, ProviderSendGrid, payload.Provider)
		assert.Equal(t, "inbound", payload.Type)
		require.NotNil(t, payload.IncomingEmail)
		assert.Equal(t, "sender@example.com", payload.IncomingEmail.From)
		assert.Equal(t, "Test Subject", payload.IncomingEmail.Subject)
		assert.Equal(t, "Hello World", payload.IncomingEmail.TextBody)
		assert.Equal(t, "<p>Hello World</p>", payload.IncomingEmail.HTMLBody)
		assert.Equal(t, 1.5, payload.IncomingEmail.SpamScore)
		assert.Equal(t, "pass", payload.IncomingEmail.Metadata["spf"])
		assert.Equal(t, "pass", payload.IncomingEmail.Metadata["dkim"])
	})

	t.Run("event webhook with JSON", func(t *testing.T) {
		events := []SendGridEventWebhook{
			{
				Email:     "recipient@example.com",
				Event:     "delivered",
				MessageID: "sg-msg-123",
				Timestamp: 1609459200,
				Response:  "250 OK",
			},
		}
		body, _ := json.Marshal(events)
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		payload, err := ParseWebhook(ProviderSendGrid, body, headers)
		require.NoError(t, err)
		assert.Equal(t, "status", payload.Type)
		require.NotNil(t, payload.StatusCallback)
		assert.Equal(t, StatusDelivered, payload.StatusCallback.Status)
		assert.Equal(t, "sg-msg-123", payload.StatusCallback.ExternalID)
		assert.Equal(t, "recipient@example.com", payload.StatusCallback.Recipient)
		assert.Equal(t, "250 OK", payload.StatusCallback.Metadata["response"])
	})
}

func TestParseMailgunWebhook(t *testing.T) {
	t.Run("inbound email with form content-type", func(t *testing.T) {
		form := url.Values{}
		form.Set("recipient", "to@example.com")
		form.Set("sender", "sender@example.com")
		form.Set("from", "Sender Name <sender@example.com>")
		form.Set("subject", "Test Subject")
		form.Set("body-plain", "Hello Body")
		form.Set("body-html", "<p>Hello Body</p>")
		form.Set("Message-Id", "<msg123@example.com>")
		form.Set("timestamp", "1609459200")

		body := []byte(form.Encode())
		headers := map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}

		payload, err := ParseWebhook(ProviderMailgun, body, headers)
		require.NoError(t, err)
		assert.Equal(t, ProviderMailgun, payload.Provider)
		assert.Equal(t, "inbound", payload.Type)
		require.NotNil(t, payload.IncomingEmail)
		assert.Equal(t, "Sender Name <sender@example.com>", payload.IncomingEmail.From)
		assert.Equal(t, []string{"to@example.com"}, payload.IncomingEmail.To)
		assert.Equal(t, "Test Subject", payload.IncomingEmail.Subject)
		assert.Equal(t, "Hello Body", payload.IncomingEmail.TextBody)
		assert.Equal(t, "<msg123@example.com>", payload.IncomingEmail.MessageID)
	})

	t.Run("event webhook with JSON", func(t *testing.T) {
		event := MailgunEventWebhook{}
		event.Signature.Token = "token123"
		event.Signature.Timestamp = "1609459200"
		event.Signature.Signature = "sig123"
		event.EventData.Event = "delivered"
		event.EventData.ID = "event-id-123"
		event.EventData.Timestamp = 1609459200
		event.EventData.Message.Headers.MessageID = "<msg@example.com>"
		event.EventData.Recipient = "user@example.com"
		event.EventData.Severity = ""

		body, _ := json.Marshal(event)
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		payload, err := ParseWebhook(ProviderMailgun, body, headers)
		require.NoError(t, err)
		assert.Equal(t, "status", payload.Type)
		require.NotNil(t, payload.StatusCallback)
		assert.Equal(t, StatusDelivered, payload.StatusCallback.Status)
		assert.Equal(t, "event-id-123", payload.StatusCallback.ExternalID)
		assert.Equal(t, "<msg@example.com>", payload.StatusCallback.MessageID)
		assert.Equal(t, "user@example.com", payload.StatusCallback.Recipient)
	})
}

func TestParseSESWebhook(t *testing.T) {
	t.Run("delivery notification", func(t *testing.T) {
		sesMsg := SESMessage{
			NotificationType: "Delivery",
			Delivery: &SESDelivery{
				Timestamp:  "2021-01-01T00:00:00Z",
				Recipients: []string{"user@example.com"},
			},
			Mail: &SESMail{
				MessageId: "ses-msg-123",
			},
		}
		msgJSON, _ := json.Marshal(sesMsg)
		notification := SESNotification{
			Type:    "Notification",
			Message: string(msgJSON),
		}
		body, _ := json.Marshal(notification)

		payload, err := ParseWebhook(ProviderSES, body, nil)
		require.NoError(t, err)
		assert.Equal(t, "status", payload.Type)
		require.NotNil(t, payload.StatusCallback)
		assert.Equal(t, StatusDelivered, payload.StatusCallback.Status)
		assert.Equal(t, "ses-msg-123", payload.StatusCallback.ExternalID)
		assert.Equal(t, "user@example.com", payload.StatusCallback.Recipient)
	})

	t.Run("bounce notification", func(t *testing.T) {
		sesMsg := SESMessage{
			NotificationType: "Bounce",
			Bounce: &SESBounce{
				BounceType:    "Permanent",
				BounceSubType: "General",
				BouncedRecipients: []SESBouncedRecipient{
					{
						EmailAddress:   "bounce@example.com",
						DiagnosticCode: "smtp; 550 5.1.1 User unknown",
					},
				},
			},
			Mail: &SESMail{
				MessageId: "ses-msg-456",
			},
		}
		msgJSON, _ := json.Marshal(sesMsg)
		notification := SESNotification{
			Type:    "Notification",
			Message: string(msgJSON),
		}
		body, _ := json.Marshal(notification)

		payload, err := ParseWebhook(ProviderSES, body, nil)
		require.NoError(t, err)
		assert.Equal(t, StatusBounced, payload.StatusCallback.Status)
		assert.Equal(t, "bounce@example.com", payload.StatusCallback.Recipient)
		assert.Equal(t, "Permanent", payload.StatusCallback.Metadata["bounce_type"])
		assert.Equal(t, "General", payload.StatusCallback.Metadata["bounce_sub_type"])
		assert.Equal(t, "smtp; 550 5.1.1 User unknown", payload.StatusCallback.ErrorMessage)
	})

	t.Run("complaint notification", func(t *testing.T) {
		sesMsg := SESMessage{
			NotificationType: "Complaint",
			Complaint: &SESComplaint{
				ComplainedRecipients: []SESComplainedRecipient{
					{EmailAddress: "complainer@example.com"},
				},
				ComplaintFeedbackType: "abuse",
			},
			Mail: &SESMail{
				MessageId: "ses-msg-789",
			},
		}
		msgJSON, _ := json.Marshal(sesMsg)
		notification := SESNotification{
			Type:    "Notification",
			Message: string(msgJSON),
		}
		body, _ := json.Marshal(notification)

		payload, err := ParseWebhook(ProviderSES, body, nil)
		require.NoError(t, err)
		assert.Equal(t, StatusSpam, payload.StatusCallback.Status)
		assert.Equal(t, "complainer@example.com", payload.StatusCallback.Recipient)
	})

	t.Run("subscription confirmation", func(t *testing.T) {
		notification := SESNotification{
			Type:         "SubscriptionConfirmation",
			SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&...",
		}
		body, _ := json.Marshal(notification)

		payload, err := ParseWebhook(ProviderSES, body, nil)
		require.NoError(t, err)
		assert.Equal(t, "subscription_confirmation", payload.Type)
		require.NotNil(t, payload.StatusCallback)
		assert.Equal(t, "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&...", payload.StatusCallback.Metadata["subscribe_url"])
	})
}

func TestParsePostmarkWebhook(t *testing.T) {
	t.Run("inbound email has From key", func(t *testing.T) {
		inbound := PostmarkInboundWebhook{
			From:      "sender@example.com",
			FromName:  "Sender",
			To:        "to@example.com",
			CC:        "cc@example.com",
			Subject:   "Test Subject",
			TextBody:  "Hello",
			HtmlBody:  "<p>Hello</p>",
			MessageID: "pm-msg-123",
			ReplyTo:   "reply@example.com",
		}
		body, _ := json.Marshal(inbound)

		payload, err := ParseWebhook(ProviderPostmark, body, nil)
		require.NoError(t, err)
		assert.Equal(t, "inbound", payload.Type)
		require.NotNil(t, payload.IncomingEmail)
		assert.Equal(t, "sender@example.com", payload.IncomingEmail.From)
		assert.Equal(t, "Sender", payload.IncomingEmail.FromName)
		assert.Equal(t, "Test Subject", payload.IncomingEmail.Subject)
		assert.Equal(t, "Hello", payload.IncomingEmail.TextBody)
		assert.Equal(t, "<p>Hello</p>", payload.IncomingEmail.HTMLBody)
		assert.Equal(t, "pm-msg-123", payload.IncomingEmail.MessageID)
	})

	t.Run("event webhook has RecordType key", func(t *testing.T) {
		event := PostmarkEventWebhook{
			RecordType:  "Delivery",
			MessageID:   "pm-msg-456",
			Recipient:   "user@example.com",
			DeliveredAt: "2021-01-01T00:00:00Z",
		}
		body, _ := json.Marshal(event)

		payload, err := ParseWebhook(ProviderPostmark, body, nil)
		require.NoError(t, err)
		assert.Equal(t, "status", payload.Type)
		require.NotNil(t, payload.StatusCallback)
		assert.Equal(t, StatusDelivered, payload.StatusCallback.Status)
		assert.Equal(t, "pm-msg-456", payload.StatusCallback.ExternalID)
		assert.Equal(t, "user@example.com", payload.StatusCallback.Recipient)
	})
}

func TestMapSendGridEventToStatus(t *testing.T) {
	tests := []struct {
		event    string
		expected EmailStatus
	}{
		{"processed", StatusQueued},
		{"dropped", StatusFailed},
		{"delivered", StatusDelivered},
		{"deferred", StatusQueued},
		{"bounce", StatusBounced},
		{"open", StatusOpened},
		{"click", StatusClicked},
		{"spam_report", StatusSpam},
		{"unsubscribe", StatusUnsubscribed},
		{"unknown", StatusSent},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapSendGridEventToStatus(tt.event))
		})
	}
}

func TestMapMailgunEventToStatus(t *testing.T) {
	tests := []struct {
		event    string
		expected EmailStatus
	}{
		{"accepted", StatusQueued},
		{"delivered", StatusDelivered},
		{"failed", StatusFailed},
		{"opened", StatusOpened},
		{"clicked", StatusClicked},
		{"unsubscribed", StatusUnsubscribed},
		{"complained", StatusSpam},
		{"unknown", StatusSent},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapMailgunEventToStatus(tt.event))
		})
	}
}

func TestMapPostmarkEventToStatus(t *testing.T) {
	tests := []struct {
		recordType string
		expected   EmailStatus
	}{
		{"Delivery", StatusDelivered},
		{"Bounce", StatusBounced},
		{"SpamComplaint", StatusSpam},
		{"Open", StatusOpened},
		{"Click", StatusClicked},
		{"SubscriptionChange", StatusUnsubscribed},
		{"Unknown", StatusSent},
	}

	for _, tt := range tests {
		t.Run(tt.recordType, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapPostmarkEventToStatus(tt.recordType))
		})
	}
}

func TestValidateMailgunWebhook(t *testing.T) {
	apiKey := "test-api-key"
	timestamp := "1609459200"
	token := "random-token"

	// Compute valid signature
	h := hmac.New(sha256.New, []byte(apiKey))
	h.Write([]byte(timestamp + token))
	validSignature := hex.EncodeToString(h.Sum(nil))

	t.Run("valid signature", func(t *testing.T) {
		assert.True(t, ValidateMailgunWebhook(apiKey, token, timestamp, validSignature))
	})

	t.Run("invalid signature", func(t *testing.T) {
		assert.False(t, ValidateMailgunWebhook(apiKey, token, timestamp, "invalidsignature"))
	})

	t.Run("wrong key", func(t *testing.T) {
		assert.False(t, ValidateMailgunWebhook("wrong-key", token, timestamp, validSignature))
	})
}

func TestValidatePostmarkWebhook(t *testing.T) {
	t.Run("with matching password", func(t *testing.T) {
		assert.True(t, ValidatePostmarkWebhook("secret123", "secret123"))
	})

	t.Run("with wrong password", func(t *testing.T) {
		assert.False(t, ValidatePostmarkWebhook("secret123", "wrong"))
	})

	t.Run("empty password allows all", func(t *testing.T) {
		assert.True(t, ValidatePostmarkWebhook("", "anything"))
		assert.True(t, ValidatePostmarkWebhook("", ""))
	})
}
