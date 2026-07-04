package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// An inbound Twilio voice call is logged as a conversation message and answered
// with a greeting + record TwiML.
func TestWebhookVoice_InboundCallBecomesMessageAndGreets(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-voice", entity.ChannelTypeVoice, map[string]string{"auth_token": "voice-token"})

	form := url.Values{
		"CallSid":    {"CA_call_1"},
		"CallStatus": {"ringing"},
		"Direction":  {"inbound"},
		"From":       {"+15551112222"},
		"To":         {"+15553334444"},
	}

	w, c := postTwilioForm("ch-voice", "voice-token", form, "")
	handler.VoiceWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "voice", msg.ChannelType)
	assert.Equal(t, "CA_call_1", msg.ExternalID)
	assert.Equal(t, "call_started", msg.Metadata["event_type"])
	assert.Equal(t, "+15551112222", msg.Metadata["sender_id"])

	// The initial inbound call is answered with a greeting + record.
	body := w.Body.String()
	assert.Contains(t, body, "<Say>")
	assert.Contains(t, body, "<Record")
}

func TestWebhookVoice_InvalidSignatureRejected(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-voice", entity.ChannelTypeVoice, map[string]string{"auth_token": "voice-token"})

	form := url.Values{"CallSid": {"CA_x"}, "CallStatus": {"ringing"}, "Direction": {"inbound"}}
	w, c := postTwilioForm("ch-voice", "voice-token", form, "bad-signature")
	handler.VoiceWebhook(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, producer.InboundMessages)
}

// A recording callback becomes an audio message with the recording as an
// attachment, and is acked with empty TwiML (not a greeting).
func TestWebhookVoice_RecordingBecomesAudioAttachment(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-voice", entity.ChannelTypeVoice, map[string]string{"auth_token": "voice-token"})

	form := url.Values{
		"CallSid":      {"CA_call_1"},
		"CallStatus":   {"completed"},
		"Direction":    {"inbound"},
		"From":         {"+15551112222"},
		"RecordingUrl": {"https://api.twilio.com/rec/RE1.mp3"},
	}

	w, c := postTwilioForm("ch-voice", "voice-token", form, "")
	handler.VoiceWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "audio", msg.ContentType)
	assert.Equal(t, "recording", msg.Metadata["event_type"])
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "https://api.twilio.com/rec/RE1.mp3", msg.Attachments[0].URL)

	// A recording callback is not the initial call answer → empty TwiML.
	assert.NotContains(t, w.Body.String(), "<Record")
	assert.True(t, strings.Contains(w.Body.String(), "<Response></Response>"))
}
