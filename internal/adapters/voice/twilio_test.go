package voice

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTwilioProvider_Name(t *testing.T) {
	p := NewTwilioProvider()
	assert.Equal(t, "twilio", p.Name())
}

func TestTwilioProvider_Initialize_Success(t *testing.T) {
	p := NewTwilioProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		PhoneNumber: "+5511999999999",
		Credentials: map[string]string{
			"account_sid": "ACtest123",
			"auth_token":  "token123",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "ACtest123", p.accountSID)
	assert.Equal(t, "token123", p.authToken)
	assert.Equal(t, "+5511999999999", p.phoneNumber)
}

func TestTwilioProvider_Initialize_MissingAccountSID(t *testing.T) {
	p := NewTwilioProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"auth_token": "token123",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account_sid")
}

func TestTwilioProvider_Initialize_EmptyAccountSID(t *testing.T) {
	p := NewTwilioProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"account_sid": "",
			"auth_token":  "token123",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account_sid")
}

func TestTwilioProvider_Initialize_MissingAuthToken(t *testing.T) {
	p := NewTwilioProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"account_sid": "ACtest123",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_token")
}

func TestTwilioProvider_Initialize_EmptyAuthToken(t *testing.T) {
	p := NewTwilioProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"account_sid": "ACtest123",
			"auth_token":  "",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_token")
}

func TestTwilioProvider_Initialize_NoCreds(t *testing.T) {
	p := NewTwilioProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{},
	})
	require.Error(t, err)
}

func TestTwilioProvider_Capabilities(t *testing.T) {
	p := NewTwilioProvider()
	caps := p.Capabilities()

	assert.True(t, caps.OutboundCalls)
	assert.True(t, caps.InboundCalls)
	assert.True(t, caps.Recording)
	assert.True(t, caps.Transcription)
	assert.True(t, caps.TextToSpeech)
	assert.True(t, caps.SpeechRecognition)
	assert.True(t, caps.DTMF)
	assert.True(t, caps.Conferencing)
	assert.True(t, caps.CallQueues)
	assert.True(t, caps.SIP)
	assert.True(t, caps.WebRTC)
}

// --- GenerateIVRResponse tests ---

func TestTwilioProvider_GenerateIVRResponse_Say(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRSay{Text: "Hello world", Language: "en-US", Voice: "alice", Loop: 2},
	})
	require.NoError(t, err)

	twiml, ok := resp.(string)
	require.True(t, ok)

	assert.Contains(t, twiml, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, twiml, `<Response>`)
	assert.Contains(t, twiml, `</Response>`)
	assert.Contains(t, twiml, `<Say language="en-US" voice="alice" loop="2">Hello world</Say>`)
}

func TestTwilioProvider_GenerateIVRResponse_SayNoVoice(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRSay{Text: "Ola", Language: "pt-BR"},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Say language="pt-BR">Ola</Say>`)
	assert.NotContains(t, twiml, `voice=`)
}

func TestTwilioProvider_GenerateIVRResponse_GatherWithNestedSay(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRGather{
			Input:     []string{"dtmf", "speech"},
			Timeout:   5,
			NumDigits: 1,
			Nested: []IVRAction{
				IVRSay{Text: "Press 1 for sales", Language: "en-US"},
			},
		},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Gather input="dtmf speech" timeout="5" numDigits="1">`)
	assert.Contains(t, twiml, `<Say language="en-US">Press 1 for sales</Say>`)
	assert.Contains(t, twiml, `</Gather>`)
}

func TestTwilioProvider_GenerateIVRResponse_GatherWithNestedPlay(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRGather{
			Input:   []string{"dtmf"},
			Timeout: 10,
			Nested: []IVRAction{
				IVRPlay{URL: "https://example.com/prompt.mp3"},
			},
		},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Play>https://example.com/prompt.mp3</Play>`)
}

func TestTwilioProvider_GenerateIVRResponse_MultipleActions(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRSay{Text: "Welcome", Language: "en-US"},
		IVRPause{Length: 2},
		IVRGather{
			Input:     []string{"dtmf"},
			Timeout:   5,
			NumDigits: 1,
			Nested: []IVRAction{
				IVRSay{Text: "Press a key", Language: "en-US"},
			},
		},
		IVRSay{Text: "Goodbye", Language: "en-US"},
		IVRHangup{},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Say language="en-US">Welcome</Say>`)
	assert.Contains(t, twiml, `<Pause length="2"/>`)
	assert.Contains(t, twiml, `<Gather`)
	assert.Contains(t, twiml, `<Say language="en-US">Goodbye</Say>`)
	assert.Contains(t, twiml, `<Hangup/>`)
}

func TestTwilioProvider_GenerateIVRResponse_Play(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRPlay{URL: "https://example.com/music.mp3", Loop: 3, Digits: "1234"},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Play loop="3" digits="1234">https://example.com/music.mp3</Play>`)
}

func TestTwilioProvider_GenerateIVRResponse_Record(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRRecord{
			MaxLength:   60,
			Timeout:     5,
			FinishOnKey: "#",
			Transcribe:  true,
			PlayBeep:    true,
			ActionURL:   "https://example.com/record",
		},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Record`)
	assert.Contains(t, twiml, `maxLength="60"`)
	assert.Contains(t, twiml, `timeout="5"`)
	assert.Contains(t, twiml, `finishOnKey="#"`)
	assert.Contains(t, twiml, `transcribe="true"`)
	assert.Contains(t, twiml, `playBeep="true"`)
	assert.Contains(t, twiml, `action="https://example.com/record"`)
}

func TestTwilioProvider_GenerateIVRResponse_Dial(t *testing.T) {
	p := NewTwilioProvider()

	tests := []struct {
		name     string
		dial     IVRDial
		contains []string
	}{
		{
			name:     "dial number",
			dial:     IVRDial{Number: "+5511999999999", Timeout: 30, CallerID: "+5511888888888"},
			contains: []string{`<Dial`, `timeout="30"`, `callerId="+5511888888888"`, "+5511999999999", `</Dial>`},
		},
		{
			name:     "dial SIP",
			dial:     IVRDial{SIPEndpoint: "sip:user@example.com"},
			contains: []string{`<Sip>sip:user@example.com</Sip>`},
		},
		{
			name:     "dial queue",
			dial:     IVRDial{Queue: "support"},
			contains: []string{`<Queue>support</Queue>`},
		},
		{
			name:     "dial with record",
			dial:     IVRDial{Number: "+5511999999999", Record: true},
			contains: []string{`record="record-from-answer"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.GenerateIVRResponse([]IVRAction{tt.dial})
			require.NoError(t, err)
			twiml := resp.(string)
			for _, c := range tt.contains {
				assert.Contains(t, twiml, c)
			}
		})
	}
}

func TestTwilioProvider_GenerateIVRResponse_Redirect(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRRedirect{URL: "https://example.com/next", Method: "POST"},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Redirect method="POST">https://example.com/next</Redirect>`)
}

func TestTwilioProvider_GenerateIVRResponse_Queue(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRQueue{Name: "support", WaitURL: "https://example.com/wait", ActionURL: "https://example.com/action"},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Enqueue waitUrl="https://example.com/wait" action="https://example.com/action">support</Enqueue>`)
}

func TestTwilioProvider_GenerateIVRResponse_Conference(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRConference{
			Name:            "room1",
			Muted:           true,
			StartOnEnter:    true,
			EndOnExit:       true,
			WaitURL:         "https://example.com/wait",
			MaxParticipants: 10,
			Record:          true,
		},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Dial><Conference`)
	assert.Contains(t, twiml, `muted="true"`)
	assert.Contains(t, twiml, `startConferenceOnEnter="true"`)
	assert.Contains(t, twiml, `endConferenceOnExit="true"`)
	assert.Contains(t, twiml, `maxParticipants="10"`)
	assert.Contains(t, twiml, `record="record-from-start"`)
	assert.Contains(t, twiml, `>room1</Conference></Dial>`)
}

func TestTwilioProvider_GenerateIVRResponse_EmptyActions(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse(nil)
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `<Response>`)
	assert.Contains(t, twiml, `</Response>`)
}

func TestTwilioProvider_GenerateIVRResponse_GatherWithAllOptions(t *testing.T) {
	p := NewTwilioProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRGather{
			Input:       []string{"dtmf", "speech"},
			Timeout:     10,
			NumDigits:   4,
			FinishOnKey: "#",
			ActionURL:   "https://example.com/gather",
			Method:      "POST",
			Language:    "pt-BR",
			Hints:       []string{"sim", "nao", "talvez"},
		},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.Contains(t, twiml, `input="dtmf speech"`)
	assert.Contains(t, twiml, `timeout="10"`)
	assert.Contains(t, twiml, `numDigits="4"`)
	assert.Contains(t, twiml, `finishOnKey="#"`)
	assert.Contains(t, twiml, `action="https://example.com/gather"`)
	assert.Contains(t, twiml, `language="pt-BR"`)
	assert.Contains(t, twiml, `hints="sim,nao,talvez"`)
}

// --- ParseWebhook tests ---

func TestTwilioProvider_ParseWebhook_StatusEvent(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":    {"CA123"},
		"CallStatus": {"in-progress"},
		"From":       {"+5511999999999"},
		"To":         {"+5511888888888"},
		"Direction":  {"inbound"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "status", event.Type)
	assert.Equal(t, "CA123", event.CallID)
	assert.Equal(t, "CA123", event.ExternalID)
	assert.Equal(t, CallStatusInProgress, event.Status)
	assert.Equal(t, "+5511999999999", event.From)
	assert.Equal(t, "+5511888888888", event.To)
	assert.Equal(t, CallDirectionInbound, event.Direction)
}

func TestTwilioProvider_ParseWebhook_DTMFEvent(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":    {"CA123"},
		"CallStatus": {"in-progress"},
		"From":       {"+5511999999999"},
		"To":         {"+5511888888888"},
		"Digits":     {"1"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "dtmf", event.Type)
	assert.Equal(t, "1", event.Digits)
}

func TestTwilioProvider_ParseWebhook_SpeechEvent(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":      {"CA123"},
		"CallStatus":   {"in-progress"},
		"SpeechResult": {"yes please"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "speech", event.Type)
	assert.Equal(t, "yes please", event.SpeechResult)
}

func TestTwilioProvider_ParseWebhook_RecordingEvent(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":      {"CA123"},
		"CallStatus":   {"completed"},
		"RecordingUrl": {"https://api.twilio.com/recording.mp3"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "recording", event.Type)
	assert.Equal(t, "https://api.twilio.com/recording.mp3", event.RecordingURL)
}

func TestTwilioProvider_ParseWebhook_TranscriptionEvent(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":           {"CA123"},
		"CallStatus":        {"completed"},
		"TranscriptionText": {"Hello world"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "transcription", event.Type)
	assert.Equal(t, "Hello world", event.Transcription)
}

func TestTwilioProvider_ParseWebhook_WithDuration(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":      {"CA123"},
		"CallStatus":   {"completed"},
		"CallDuration": {"120"},
		"Direction":    {"outbound-api"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, 120, event.Duration)
	assert.Equal(t, CallDirectionOutbound, event.Direction)
}

func TestTwilioProvider_ParseWebhook_RawPayload(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":    {"CA123"},
		"CallStatus": {"ringing"},
		"CustomKey":  {"custom_value"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "custom_value", event.RawPayload["CustomKey"])
	assert.Equal(t, "CA123", event.RawPayload["CallSid"])
}

func TestTwilioProvider_ParseWebhook_InvalidBody(t *testing.T) {
	p := NewTwilioProvider()
	// url.ParseQuery is very lenient, so we test with valid-ish data
	// that produces empty results
	event, err := p.ParseWebhook(context.Background(), nil, []byte(""))
	require.NoError(t, err)
	assert.Equal(t, "status", event.Type)
	assert.Empty(t, event.CallID)
}

// --- ValidateWebhook tests ---

func TestTwilioProvider_ValidateWebhook_CorrectSignature(t *testing.T) {
	p := NewTwilioProvider()
	p.authToken = "test_auth_token"

	requestURL := "https://example.com/voice/webhook"
	body := url.Values{
		"CallSid":    {"CA123"},
		"CallStatus": {"ringing"},
		"From":       {"+15551234567"},
	}

	// Build validation string the same way the provider does
	validationString := requestURL
	var keys []string
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		validationString += k + body.Get(k)
	}

	mac := hmac.New(sha1.New, []byte(p.authToken))
	mac.Write([]byte(validationString))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string]string{
		"X-Twilio-Signature": signature,
		"X-Forwarded-Proto":  "https",
		"Host":               "example.com",
		"X-Original-URI":     "/voice/webhook",
	}

	result := p.ValidateWebhook(context.Background(), headers, []byte(body.Encode()))
	assert.True(t, result)
}

func TestTwilioProvider_ValidateWebhook_WrongSignature(t *testing.T) {
	p := NewTwilioProvider()
	p.authToken = "test_auth_token"

	body := url.Values{
		"CallSid":    {"CA123"},
		"CallStatus": {"ringing"},
	}

	headers := map[string]string{
		"X-Twilio-Signature": "wrong_signature_here",
		"X-Forwarded-Proto":  "https",
		"Host":               "example.com",
		"X-Original-URI":     "/voice/webhook",
	}

	result := p.ValidateWebhook(context.Background(), headers, []byte(body.Encode()))
	assert.False(t, result)
}

func TestTwilioProvider_ValidateWebhook_MissingSignature(t *testing.T) {
	p := NewTwilioProvider()
	p.authToken = "test_auth_token"

	headers := map[string]string{
		"X-Forwarded-Proto": "https",
		"Host":              "example.com",
		"X-Original-URI":    "/voice/webhook",
	}

	result := p.ValidateWebhook(context.Background(), headers, []byte("CallSid=CA123"))
	assert.False(t, result)
}

func TestTwilioProvider_ValidateWebhook_LowercaseHeader(t *testing.T) {
	p := NewTwilioProvider()
	p.authToken = "test_auth_token"

	requestURL := "https://example.com/voice/webhook"
	body := url.Values{"CallSid": {"CA123"}}

	validationString := requestURL + "CallSid" + "CA123"
	mac := hmac.New(sha1.New, []byte(p.authToken))
	mac.Write([]byte(validationString))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string]string{
		"x-twilio-signature": signature,
		"X-Forwarded-Proto":  "https",
		"Host":               "example.com",
		"X-Original-URI":     "/voice/webhook",
	}

	result := p.ValidateWebhook(context.Background(), headers, []byte(body.Encode()))
	assert.True(t, result)
}

func TestTwilioProvider_ValidateWebhook_DevFallback(t *testing.T) {
	// When URL headers are missing, it falls back to "://" and skips validation
	p := NewTwilioProvider()
	p.authToken = "test_auth_token"

	headers := map[string]string{
		"X-Twilio-Signature": "any_signature",
	}

	result := p.ValidateWebhook(context.Background(), headers, []byte("CallSid=CA123"))
	assert.True(t, result) // development fallback
}

// --- MakeCall with httptest ---

func TestTwilioProvider_MakeCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/Accounts/ACtest/Calls.json")
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		// Verify basic auth
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "ACtest", user)
		assert.Equal(t, "testtoken", pass)

		// Parse form values
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "+5511888888888", r.FormValue("To"))
		assert.Equal(t, "+5511999999999", r.FormValue("From"))
		assert.Equal(t, "true", r.FormValue("Record"))

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"sid":    "CA_test_sid",
			"status": "queued",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	result, err := p.MakeCall(context.Background(), MakeCallInput{
		To:     "+5511888888888",
		From:   "+5511999999999",
		Record: true,
	})
	require.NoError(t, err)

	assert.Equal(t, "CA_test_sid", result.CallID)
	assert.Equal(t, "CA_test_sid", result.ExternalID)
	assert.Equal(t, CallStatusInitiated, result.Status)
}

func TestTwilioProvider_MakeCall_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "https://example.com/callback", r.FormValue("Url"))
		assert.Equal(t, "https://example.com/status", r.FormValue("StatusCallback"))
		assert.Equal(t, "<Response><Say>Hi</Say></Response>", r.FormValue("Twiml"))
		assert.Equal(t, "30", r.FormValue("Timeout"))
		assert.Equal(t, "Enable", r.FormValue("MachineDetection"))

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"sid":    "CA_test_sid2",
			"status": "queued",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	result, err := p.MakeCall(context.Background(), MakeCallInput{
		To:               "+5511888888888",
		From:             "+5511999999999",
		CallbackURL:      "https://example.com/callback",
		StatusURL:        "https://example.com/status",
		TwiML:            "<Response><Say>Hi</Say></Response>",
		Timeout:          30,
		MachineDetection: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "CA_test_sid2", result.CallID)
}

func TestTwilioProvider_MakeCall_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    21211,
			"message": "Invalid 'To' phone number",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	result, err := p.MakeCall(context.Background(), MakeCallInput{
		To:   "invalid",
		From: "+5511999999999",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "twilio error")
}

// --- GetCall with httptest ---

func TestTwilioProvider_GetCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/Calls/CA_test_sid.json")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"sid":          "CA_test_sid",
			"status":       "in-progress",
			"direction":    "inbound",
			"from":         "+5511999999999",
			"to":           "+5511888888888",
			"duration":     "60",
			"date_created": "Mon, 01 Jan 2024 12:00:00 +0000",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	call, err := p.GetCall(context.Background(), "CA_test_sid")
	require.NoError(t, err)

	assert.Equal(t, "CA_test_sid", call.ID)
	assert.Equal(t, "CA_test_sid", call.ExternalID)
	assert.Equal(t, CallStatusInProgress, call.Status)
	assert.Equal(t, CallDirectionInbound, call.Direction)
	assert.Equal(t, "+5511999999999", call.From)
	assert.Equal(t, "+5511888888888", call.To)
	assert.Equal(t, 60, call.Duration)
}

func TestTwilioProvider_GetCall_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	call, err := p.GetCall(context.Background(), "CA_nonexistent")
	assert.Error(t, err)
	assert.Nil(t, call)
	assert.Contains(t, err.Error(), "call not found")
}

// --- EndCall with httptest ---

func TestTwilioProvider_EndCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/Calls/CA_test_sid.json")

		require.NoError(t, r.ParseForm())
		assert.Equal(t, "completed", r.FormValue("Status"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"sid":    "CA_test_sid",
			"status": "completed",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	err := p.EndCall(context.Background(), "CA_test_sid")
	assert.NoError(t, err)
}

func TestTwilioProvider_EndCall_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	err := p.EndCall(context.Background(), "CA_test_sid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to end call")
}

// --- TransferCall with httptest ---

func TestTwilioProvider_TransferCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		require.NoError(t, r.ParseForm())

		twiml := r.FormValue("Twiml")
		assert.Contains(t, twiml, "<Dial>+5511777777777</Dial>")

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	err := p.TransferCall(context.Background(), "CA_test_sid", "+5511777777777")
	assert.NoError(t, err)
}

// --- GetRecording with httptest ---

func TestTwilioProvider_GetRecording_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/Recordings/RE_test.json")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"sid":          "RE_test",
			"call_sid":     "CA_test",
			"duration":     "30",
			"date_created": "Mon, 01 Jan 2024 12:00:00 +0000",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	rec, err := p.GetRecording(context.Background(), "RE_test")
	require.NoError(t, err)

	assert.Equal(t, "RE_test", rec.ID)
	assert.Equal(t, "CA_test", rec.CallID)
	assert.Equal(t, 30, rec.Duration)
	assert.Equal(t, "mp3", rec.Format)
	assert.Contains(t, rec.URL, "RE_test.mp3")
}

func TestTwilioProvider_GetRecording_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	rec, err := p.GetRecording(context.Background(), "RE_nonexistent")
	assert.Error(t, err)
	assert.Nil(t, rec)
	assert.Contains(t, err.Error(), "recording not found")
}

// --- DeleteRecording with httptest ---

func TestTwilioProvider_DeleteRecording_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/Recordings/RE_test.json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	err := p.DeleteRecording(context.Background(), "RE_test")
	assert.NoError(t, err)
}

func TestTwilioProvider_DeleteRecording_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "ACtest"
	p.authToken = "testtoken"

	err := p.DeleteRecording(context.Background(), "RE_test")
	assert.Error(t, err)
}

// --- mapStatus tests ---

func TestTwilioProvider_MapStatus(t *testing.T) {
	p := NewTwilioProvider()

	tests := []struct {
		input    string
		expected CallStatus
	}{
		{"queued", CallStatusInitiated},
		{"initiated", CallStatusInitiated},
		{"ringing", CallStatusRinging},
		{"in-progress", CallStatusInProgress},
		{"completed", CallStatusCompleted},
		{"busy", CallStatusBusy},
		{"no-answer", CallStatusNoAnswer},
		{"failed", CallStatusFailed},
		{"canceled", CallStatusCanceled},
		{"unknown", CallStatus("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, p.mapStatus(tt.input))
		})
	}
}

func TestTwilioProvider_MapStatus_CaseInsensitive(t *testing.T) {
	p := NewTwilioProvider()

	assert.Equal(t, CallStatusCompleted, p.mapStatus("COMPLETED"))
	assert.Equal(t, CallStatusRinging, p.mapStatus("Ringing"))
	assert.Equal(t, CallStatusBusy, p.mapStatus("BUSY"))
}

// --- MakeCall URL construction ---

func TestTwilioProvider_MakeCall_URLConstruction(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"sid":    "CA_test",
			"status": "queued",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider()
	p.baseURL = server.URL
	p.accountSID = "AC12345"
	p.authToken = "testtoken"

	_, err := p.MakeCall(context.Background(), MakeCallInput{
		To:   "+5511999999999",
		From: "+5511888888888",
	})
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("/Accounts/AC12345/Calls.json"), capturedPath)
}

// --- ParseWebhook status mapping ---

func TestTwilioProvider_ParseWebhook_AllStatuses(t *testing.T) {
	p := NewTwilioProvider()

	statuses := map[string]CallStatus{
		"queued":      CallStatusInitiated,
		"ringing":     CallStatusRinging,
		"in-progress": CallStatusInProgress,
		"completed":   CallStatusCompleted,
		"busy":        CallStatusBusy,
		"no-answer":   CallStatusNoAnswer,
		"failed":      CallStatusFailed,
		"canceled":    CallStatusCanceled,
	}

	for twilioStatus, expectedStatus := range statuses {
		t.Run(twilioStatus, func(t *testing.T) {
			body := url.Values{
				"CallSid":    {"CA123"},
				"CallStatus": {twilioStatus},
			}.Encode()

			event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
			require.NoError(t, err)
			assert.Equal(t, expectedStatus, event.Status)
		})
	}
}

// --- Integration-style: ParseWebhook preserves all form fields in RawPayload ---

func TestTwilioProvider_ParseWebhook_CustomFields(t *testing.T) {
	p := NewTwilioProvider()

	body := url.Values{
		"CallSid":     {"CA123"},
		"CallStatus":  {"ringing"},
		"AccountSid":  {"AC_test"},
		"ApiVersion":  {"2010-04-01"},
		"ForwardedFrom": {"+5511777777777"},
	}.Encode()

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	// All fields should be in RawPayload
	assert.Equal(t, "CA123", event.RawPayload["CallSid"])
	assert.Equal(t, "AC_test", event.RawPayload["AccountSid"])
	assert.Equal(t, "2010-04-01", event.RawPayload["ApiVersion"])
	assert.Equal(t, "+5511777777777", event.RawPayload["ForwardedFrom"])
}

// --- Ensure GenerateIVRResponse XML structure is well-formed ---

func TestTwilioProvider_GenerateIVRResponse_XMLStructure(t *testing.T) {
	p := NewTwilioProvider()

	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRSay{Text: "Hello", Language: "en-US"},
	})
	require.NoError(t, err)

	twiml := resp.(string)
	assert.True(t, strings.HasPrefix(twiml, `<?xml version="1.0" encoding="UTF-8"?>`))
	assert.True(t, strings.Contains(twiml, "<Response>"))
	assert.True(t, strings.HasSuffix(twiml, "</Response>"))
}
