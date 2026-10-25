package email

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The status folds the provider's event (open/click → read, bounce/spam/failure → failed); the
// original rides along in ProviderEvent, with BounceType and URL when the provider tells.

func statusOf(t *testing.T, provider Provider, body []byte, headers map[string]string) *StatusCallback {
	t.Helper()
	payload, err := ParseWebhook(provider, body, headers)
	require.NoError(t, err)
	require.Equal(t, "status", payload.Type)
	require.NotNil(t, payload.StatusCallback)
	return payload.StatusCallback
}

func TestSendGridProviderEvent(t *testing.T) {
	json := map[string]string{"Content-Type": "application/json"}
	cases := []struct {
		body, event, bounce, url string
	}{
		{`[{"event":"open","sg_message_id":"m","timestamp":1}]`, ProviderEventOpened, "", ""},
		{`[{"event":"click","sg_message_id":"m","timestamp":1,"url":"https://loja.exemplo.test/oferta"}]`, ProviderEventClicked, "", "https://loja.exemplo.test/oferta"},
		{`[{"event":"bounce","type":"bounce","sg_message_id":"m","timestamp":1,"reason":"550 user unknown"}]`, ProviderEventBounced, BounceHard, ""},
		{`[{"event":"bounce","type":"blocked","sg_message_id":"m","timestamp":1}]`, ProviderEventBounced, BounceSoft, ""},
		{`[{"event":"spamreport","sg_message_id":"m","timestamp":1}]`, ProviderEventSpamReport, "", ""},
		{`[{"event":"group_unsubscribe","sg_message_id":"m","timestamp":1}]`, ProviderEventUnsubscribed, "", ""},
		{`[{"event":"dropped","sg_message_id":"m","timestamp":1}]`, ProviderEventFailed, "", ""},
		{`[{"event":"delivered","sg_message_id":"m","timestamp":1}]`, ProviderEventDelivered, "", ""},
		{`[{"event":"processed","sg_message_id":"m","timestamp":1}]`, "", "", ""},
	}
	for _, c := range cases {
		s := statusOf(t, ProviderSendGrid, []byte(c.body), json)
		assert.Equal(t, c.event, s.ProviderEvent, c.body)
		assert.Equal(t, c.bounce, s.BounceType, c.body)
		assert.Equal(t, c.url, s.URL, c.body)
	}
}

func TestMailgunProviderEvent(t *testing.T) {
	ev := func(event, severity, url string) []byte {
		return []byte(`{"event-data":{"event":"` + event + `","id":"e1","timestamp":1,"severity":"` + severity + `","url":"` + url +
			`","message":{"headers":{"message-id":"<m@exemplo.test>"}},"recipient":"ana@exemplo.test"}}`)
	}
	s := statusOf(t, ProviderMailgun, ev("failed", "permanent", ""), nil)
	assert.Equal(t, StatusFailed, s.Status, "status unchanged")
	assert.Equal(t, ProviderEventBounced, s.ProviderEvent)
	assert.Equal(t, BounceHard, s.BounceType)

	s = statusOf(t, ProviderMailgun, ev("clicked", "", "https://loja.exemplo.test/x"), nil)
	assert.Equal(t, ProviderEventClicked, s.ProviderEvent)
	assert.Equal(t, "https://loja.exemplo.test/x", s.URL)

	s = statusOf(t, ProviderMailgun, ev("complained", "", ""), nil)
	assert.Equal(t, ProviderEventSpamReport, s.ProviderEvent)
	assert.Empty(t, s.BounceType)

	s = statusOf(t, ProviderMailgun, ev("opened", "", ""), nil)
	assert.Equal(t, ProviderEventOpened, s.ProviderEvent)
}

func sesBody(t *testing.T, msg SESMessage) []byte {
	t.Helper()
	m, _ := json.Marshal(msg)
	b, _ := json.Marshal(SESNotification{Type: "Notification", Message: string(m)})
	return b
}

func TestSESProviderEvent(t *testing.T) {
	mail := &SESMail{MessageId: "ses-1"}
	s := statusOf(t, ProviderSES, sesBody(t, SESMessage{NotificationType: "Bounce", Mail: mail,
		Bounce: &SESBounce{BounceType: "Permanent", BouncedRecipients: []SESBouncedRecipient{{EmailAddress: "a@exemplo.test"}}}}), nil)
	assert.Equal(t, ProviderEventBounced, s.ProviderEvent)
	assert.Equal(t, BounceHard, s.BounceType)

	s = statusOf(t, ProviderSES, sesBody(t, SESMessage{NotificationType: "Bounce", Mail: mail, Bounce: &SESBounce{BounceType: "Transient"}}), nil)
	assert.Equal(t, BounceSoft, s.BounceType)

	s = statusOf(t, ProviderSES, sesBody(t, SESMessage{NotificationType: "Bounce", Mail: mail, Bounce: &SESBounce{BounceType: "Undetermined"}}), nil)
	assert.Empty(t, s.BounceType, "the provider didn't say")

	s = statusOf(t, ProviderSES, sesBody(t, SESMessage{NotificationType: "Complaint", Mail: mail}), nil)
	assert.Equal(t, ProviderEventSpamReport, s.ProviderEvent)
}

func TestPostmarkProviderEvent(t *testing.T) {
	cases := []struct {
		body, event, bounce, url string
	}{
		{`{"RecordType":"Bounce","MessageID":"p","Type":"HardBounce"}`, ProviderEventBounced, BounceHard, ""},
		{`{"RecordType":"Bounce","MessageID":"p","Type":"SoftBounce"}`, ProviderEventBounced, BounceSoft, ""},
		{`{"RecordType":"Bounce","MessageID":"p","Type":"AutoResponder"}`, ProviderEventBounced, "", ""},
		{`{"RecordType":"SpamComplaint","MessageID":"p"}`, ProviderEventSpamReport, "", ""},
		{`{"RecordType":"Click","MessageID":"p","OriginalLink":"https://loja.exemplo.test/y"}`, ProviderEventClicked, "", "https://loja.exemplo.test/y"},
		{`{"RecordType":"Open","MessageID":"p"}`, ProviderEventOpened, "", ""},
		{`{"RecordType":"SubscriptionChange","MessageID":"p"}`, ProviderEventUnsubscribed, "", ""},
	}
	for _, c := range cases {
		s := statusOf(t, ProviderPostmark, []byte(c.body), nil)
		assert.Equal(t, c.event, s.ProviderEvent, c.body)
		assert.Equal(t, c.bounce, s.BounceType, c.body)
		assert.Equal(t, c.url, s.URL, c.body)
	}
}
