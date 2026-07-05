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
	// ExternalID is qualified by event type so distinct events of one call each
	// persist and repeated lifecycle callbacks collapse to a single call_started.
	assert.Equal(t, "CA_call_1:call_started", msg.ExternalID)
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

	// Distinct from the call-started event id, so the inbound dedup keeps both.
	assert.Equal(t, "CA_call_1:recording", msg.ExternalID)
}

// Different events of the SAME call must produce DIFFERENT external ids, so the
// (conversation, external_id) dedup persists each instead of dropping all but the
// first (the bug where recording/transcription/ended were lost).
func TestWebhookVoice_EventsOfSameCallGetDistinctExternalIDs(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-voice", entity.ChannelTypeVoice, map[string]string{"auth_token": "voice-token"})

	post := func(form url.Values) {
		w, c := postTwilioForm("ch-voice", "voice-token", form, "")
		handler.VoiceWebhook(c)
		assert.Equal(t, http.StatusOK, w.Code)
	}
	post(url.Values{"CallSid": {"CA9"}, "CallStatus": {"ringing"}, "Direction": {"inbound"}, "From": {"+15551112222"}})
	post(url.Values{"CallSid": {"CA9"}, "CallStatus": {"completed"}, "Direction": {"inbound"}, "From": {"+15551112222"}, "TranscriptionText": {"hi there"}})
	post(url.Values{"CallSid": {"CA9"}, "CallStatus": {"completed"}, "Direction": {"inbound"}, "From": {"+15551112222"}, "CallDuration": {"42"}})

	require.Len(t, producer.InboundMessages, 3)
	ids := map[string]bool{}
	for _, m := range producer.InboundMessages {
		ids[m.ExternalID] = true
	}
	assert.Len(t, ids, 3, "each event of the call must have a unique external_id")
}

// An event without a caller id must not create a junk empty-identifier contact.
func TestWebhookVoice_EmptyFromIsSkipped(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-voice", entity.ChannelTypeVoice, map[string]string{"auth_token": "voice-token"})

	form := url.Values{"CallSid": {"CA_nofrom"}, "CallStatus": {"completed"}, "Direction": {"inbound"}, "CallDuration": {"5"}}
	w, c := postTwilioForm("ch-voice", "voice-token", form, "")
	handler.VoiceWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, producer.InboundMessages, "an event with no From must not be published")
}
