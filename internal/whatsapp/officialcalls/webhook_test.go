package officialcalls

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const connectWebhook = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "WABA_ID",
    "changes": [{
      "field": "calls",
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"display_phone_number": "5511...", "phone_number_id": "phone-55"},
        "calls": [{
          "id": "call-abc",
          "from": "5511999999999",
          "to": "5511000000000",
          "event": "connect",
          "timestamp": "1700000000",
          "direction": "USER_INITIATED",
          "session": {"sdp_type": "offer", "sdp": "v=0 the-offer"}
        }]
      }
    }]
  }]
}`

const terminateWebhook = `{
  "object": "whatsapp_business_account",
  "entry": [{"changes": [{"field": "calls", "value": {
    "metadata": {"phone_number_id": "phone-55"},
    "calls": [{"id": "call-abc", "event": "terminate", "status": "COMPLETED", "duration": 42}]
  }}]}]
}`

const messageOnlyWebhook = `{
  "object": "whatsapp_business_account",
  "entry": [{"changes": [{"field": "messages", "value": {
    "metadata": {"phone_number_id": "phone-55"},
    "messages": [{"id": "wamid.x", "type": "text"}]
  }}]}]
}`

func TestParseWebhookCalls_Connect(t *testing.T) {
	calls, err := ParseWebhookCalls([]byte(connectWebhook))
	require.NoError(t, err)
	require.Len(t, calls, 1)

	c := calls[0]
	assert.Equal(t, "phone-55", c.PhoneNumberID)
	assert.True(t, c.Event.IsConnect())
	assert.Equal(t, "call-abc", c.Event.ID)
	assert.Equal(t, "5511999999999", c.Event.From)
	assert.Equal(t, DirectionUserInitiated, c.Event.Direction)
	require.NotNil(t, c.Event.Session)
	assert.Equal(t, SDPOffer, c.Event.Session.SDPType)
	assert.Equal(t, "v=0 the-offer", c.Event.Session.SDP)
}

func TestParseWebhookCalls_Terminate(t *testing.T) {
	calls, err := ParseWebhookCalls([]byte(terminateWebhook))
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.True(t, calls[0].Event.IsTerminate())
	assert.Equal(t, 42, calls[0].Event.Duration)
	assert.Equal(t, "COMPLETED", calls[0].Event.Status)
}

func TestParseWebhookCalls_IgnoresNonCalls(t *testing.T) {
	calls, err := ParseWebhookCalls([]byte(messageOnlyWebhook))
	require.NoError(t, err)
	assert.Empty(t, calls)
}

// A change whose field is not "calls" must be ignored even if it carries a
// "calls" array — routing keys off the field name, not the array's presence.
func TestParseWebhookCalls_IgnoresCallsArrayUnderNonCallsField(t *testing.T) {
	const body = `{
      "object": "whatsapp_business_account",
      "entry": [{"changes": [{"field": "messages", "value": {
        "metadata": {"phone_number_id": "phone-55"},
        "calls": [{"id": "call-x", "event": "connect", "session": {"sdp_type": "offer", "sdp": "v=0"}}]
      }}]}]
    }`
	calls, err := ParseWebhookCalls([]byte(body))
	require.NoError(t, err)
	assert.Empty(t, calls)
}
