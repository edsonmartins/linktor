package voice

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- IVRAction ActionType() tests ---

func TestIVRSay_ActionType(t *testing.T) {
	a := IVRSay{Text: "hello", Voice: "alice", Language: "en-US", Loop: 2}
	assert.Equal(t, "say", a.ActionType())
}

func TestIVRPlay_ActionType(t *testing.T) {
	a := IVRPlay{URL: "https://example.com/audio.mp3", Loop: 1, Digits: "1234"}
	assert.Equal(t, "play", a.ActionType())
}

func TestIVRGather_ActionType(t *testing.T) {
	a := IVRGather{Input: []string{"dtmf"}, Timeout: 5}
	assert.Equal(t, "gather", a.ActionType())
}

func TestIVRRecord_ActionType(t *testing.T) {
	a := IVRRecord{MaxLength: 60, Transcribe: true}
	assert.Equal(t, "record", a.ActionType())
}

func TestIVRDial_ActionType(t *testing.T) {
	a := IVRDial{Number: "+5511999999999"}
	assert.Equal(t, "dial", a.ActionType())
}

func TestIVRHangup_ActionType(t *testing.T) {
	a := IVRHangup{Reason: "completed"}
	assert.Equal(t, "hangup", a.ActionType())
}

func TestIVRPause_ActionType(t *testing.T) {
	a := IVRPause{Length: 3}
	assert.Equal(t, "pause", a.ActionType())
}

func TestIVRRedirect_ActionType(t *testing.T) {
	a := IVRRedirect{URL: "https://example.com/next", Method: "POST"}
	assert.Equal(t, "redirect", a.ActionType())
}

func TestIVRQueue_ActionType(t *testing.T) {
	a := IVRQueue{Name: "support", WaitURL: "https://example.com/wait"}
	assert.Equal(t, "queue", a.ActionType())
}

func TestIVRConference_ActionType(t *testing.T) {
	a := IVRConference{Name: "room1", Muted: true, StartOnEnter: true}
	assert.Equal(t, "conference", a.ActionType())
}

// --- IVRAction interface satisfaction ---

func TestAllIVRActions_SatisfyInterface(t *testing.T) {
	actions := []IVRAction{
		IVRSay{},
		IVRPlay{},
		IVRGather{},
		IVRRecord{},
		IVRDial{},
		IVRHangup{},
		IVRPause{},
		IVRRedirect{},
		IVRQueue{},
		IVRConference{},
	}

	expectedTypes := []string{
		"say", "play", "gather", "record", "dial",
		"hangup", "pause", "redirect", "queue", "conference",
	}

	for i, action := range actions {
		assert.Equal(t, expectedTypes[i], action.ActionType(), "Action index %d", i)
	}
}

// --- Call struct JSON round-trip ---

func TestCall_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	answered := now.Add(5 * time.Second)
	ended := now.Add(65 * time.Second)

	original := Call{
		ID:            "call-123",
		ExternalID:    "ext-456",
		ChannelID:     "ch-789",
		Direction:     CallDirectionInbound,
		Status:        CallStatusInProgress,
		From:          "+5511999999999",
		To:            "+5511888888888",
		CallerName:    "John Doe",
		Duration:      60,
		RecordingURL:  "https://example.com/rec.mp3",
		Transcription: "Hello world",
		Metadata:      map[string]string{"key": "value"},
		StartedAt:     now,
		AnsweredAt:    &answered,
		EndedAt:       &ended,
		CreatedAt:     now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Call
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.ExternalID, decoded.ExternalID)
	assert.Equal(t, original.ChannelID, decoded.ChannelID)
	assert.Equal(t, original.Direction, decoded.Direction)
	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.From, decoded.From)
	assert.Equal(t, original.To, decoded.To)
	assert.Equal(t, original.CallerName, decoded.CallerName)
	assert.Equal(t, original.Duration, decoded.Duration)
	assert.Equal(t, original.RecordingURL, decoded.RecordingURL)
	assert.Equal(t, original.Transcription, decoded.Transcription)
	assert.Equal(t, original.Metadata, decoded.Metadata)
	assert.NotNil(t, decoded.AnsweredAt)
	assert.NotNil(t, decoded.EndedAt)
}

// --- MakeCallInput JSON marshaling ---

func TestMakeCallInput_JSONMarshal(t *testing.T) {
	input := MakeCallInput{
		To:               "+5511999999999",
		From:             "+5511888888888",
		CallbackURL:      "https://example.com/callback",
		StatusURL:        "https://example.com/status",
		Record:           true,
		Transcribe:       true,
		Timeout:          30,
		MachineDetection: true,
		TwiML:            "<Response><Say>Hi</Say></Response>",
		Metadata:         map[string]string{"tenant": "abc"},
	}

	data, err := json.Marshal(input)
	require.NoError(t, err)

	var decoded MakeCallInput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, input.To, decoded.To)
	assert.Equal(t, input.From, decoded.From)
	assert.Equal(t, input.CallbackURL, decoded.CallbackURL)
	assert.Equal(t, input.StatusURL, decoded.StatusURL)
	assert.Equal(t, input.Record, decoded.Record)
	assert.Equal(t, input.Transcribe, decoded.Transcribe)
	assert.Equal(t, input.Timeout, decoded.Timeout)
	assert.Equal(t, input.MachineDetection, decoded.MachineDetection)
	assert.Equal(t, input.TwiML, decoded.TwiML)
	assert.Equal(t, input.Metadata, decoded.Metadata)
}

// --- MakeCallResult JSON ---

func TestMakeCallResult_JSONRoundTrip(t *testing.T) {
	result := MakeCallResult{
		CallID:     "call-001",
		ExternalID: "ext-001",
		Status:     CallStatusInitiated,
		Message:    "Call queued",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded MakeCallResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result, decoded)
}

// --- VoiceConfig struct fields ---

func TestVoiceConfig_JSONRoundTrip(t *testing.T) {
	config := VoiceConfig{
		Provider:        "twilio",
		PhoneNumber:     "+5511999999999",
		WebhookURL:      "https://example.com/webhook",
		StatusURL:       "https://example.com/status",
		RecordCalls:     true,
		TranscribeCalls: true,
		DefaultVoice:    "alice",
		DefaultLanguage: "pt-BR",
		Credentials: map[string]string{
			"account_sid": "AC123",
			"auth_token":  "token123",
		},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var decoded VoiceConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, config.Provider, decoded.Provider)
	assert.Equal(t, config.PhoneNumber, decoded.PhoneNumber)
	assert.Equal(t, config.WebhookURL, decoded.WebhookURL)
	assert.Equal(t, config.StatusURL, decoded.StatusURL)
	assert.Equal(t, config.RecordCalls, decoded.RecordCalls)
	assert.Equal(t, config.TranscribeCalls, decoded.TranscribeCalls)
	assert.Equal(t, config.DefaultVoice, decoded.DefaultVoice)
	assert.Equal(t, config.DefaultLanguage, decoded.DefaultLanguage)
	assert.Equal(t, config.Credentials, decoded.Credentials)
}

// --- ProviderCapabilities ---

func TestProviderCapabilities_JSONRoundTrip(t *testing.T) {
	caps := ProviderCapabilities{
		OutboundCalls:     true,
		InboundCalls:      true,
		Recording:         true,
		Transcription:     false,
		TextToSpeech:      true,
		SpeechRecognition: false,
		DTMF:              true,
		Conferencing:      true,
		CallQueues:        false,
		SIP:               true,
		WebRTC:            false,
	}

	data, err := json.Marshal(caps)
	require.NoError(t, err)

	var decoded ProviderCapabilities
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, caps, decoded)
}

func TestProviderCapabilities_ZeroValue(t *testing.T) {
	var caps ProviderCapabilities
	assert.False(t, caps.OutboundCalls)
	assert.False(t, caps.InboundCalls)
	assert.False(t, caps.Recording)
	assert.False(t, caps.Transcription)
	assert.False(t, caps.TextToSpeech)
	assert.False(t, caps.SpeechRecognition)
	assert.False(t, caps.DTMF)
	assert.False(t, caps.Conferencing)
	assert.False(t, caps.CallQueues)
	assert.False(t, caps.SIP)
	assert.False(t, caps.WebRTC)
}

// --- CallDirection/CallStatus constants ---

func TestCallDirection_Constants(t *testing.T) {
	assert.Equal(t, CallDirection("inbound"), CallDirectionInbound)
	assert.Equal(t, CallDirection("outbound"), CallDirectionOutbound)
}

func TestCallStatus_Constants(t *testing.T) {
	assert.Equal(t, CallStatus("initiated"), CallStatusInitiated)
	assert.Equal(t, CallStatus("ringing"), CallStatusRinging)
	assert.Equal(t, CallStatus("answered"), CallStatusAnswered)
	assert.Equal(t, CallStatus("in-progress"), CallStatusInProgress)
	assert.Equal(t, CallStatus("completed"), CallStatusCompleted)
	assert.Equal(t, CallStatus("busy"), CallStatusBusy)
	assert.Equal(t, CallStatus("no-answer"), CallStatusNoAnswer)
	assert.Equal(t, CallStatus("failed"), CallStatusFailed)
	assert.Equal(t, CallStatus("canceled"), CallStatusCanceled)
}

// --- Recording JSON ---

func TestRecording_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	rec := Recording{
		ID:            "rec-001",
		CallID:        "call-001",
		URL:           "https://example.com/rec.mp3",
		Duration:      120,
		Size:          1024000,
		Format:        "mp3",
		Transcription: "Hello world",
		CreatedAt:     now,
	}

	data, err := json.Marshal(rec)
	require.NoError(t, err)

	var decoded Recording
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, rec.ID, decoded.ID)
	assert.Equal(t, rec.CallID, decoded.CallID)
	assert.Equal(t, rec.URL, decoded.URL)
	assert.Equal(t, rec.Duration, decoded.Duration)
	assert.Equal(t, rec.Size, decoded.Size)
	assert.Equal(t, rec.Format, decoded.Format)
	assert.Equal(t, rec.Transcription, decoded.Transcription)
}

// --- TranscriptionResult and Word ---

func TestTranscriptionResult_JSONRoundTrip(t *testing.T) {
	result := TranscriptionResult{
		CallID:      "call-001",
		RecordingID: "rec-001",
		Text:        "Hello world",
		Confidence:  0.95,
		Language:    "en-US",
		Words: []Word{
			{Word: "Hello", Start: 0.0, End: 0.5, Confidence: 0.98},
			{Word: "world", Start: 0.6, End: 1.0, Confidence: 0.92},
		},
		CreatedAt: time.Now().Truncate(time.Millisecond),
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded TranscriptionResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.CallID, decoded.CallID)
	assert.Equal(t, result.Text, decoded.Text)
	assert.Equal(t, result.Confidence, decoded.Confidence)
	assert.Len(t, decoded.Words, 2)
	assert.Equal(t, "Hello", decoded.Words[0].Word)
	assert.Equal(t, 0.98, decoded.Words[0].Confidence)
}

// --- WebhookEvent JSON ---

func TestWebhookEvent_JSONRoundTrip(t *testing.T) {
	event := WebhookEvent{
		Type:         "status",
		CallID:       "call-001",
		ExternalID:   "ext-001",
		Status:       CallStatusInProgress,
		Direction:    CallDirectionInbound,
		From:         "+5511999999999",
		To:           "+5511888888888",
		Duration:     30,
		Digits:       "1",
		SpeechResult: "yes",
		RecordingURL: "https://example.com/rec.mp3",
		Error:        "",
		Timestamp:    time.Now().Truncate(time.Millisecond),
		RawPayload:   map[string]interface{}{"key": "value"},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded WebhookEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Type, decoded.Type)
	assert.Equal(t, event.CallID, decoded.CallID)
	assert.Equal(t, event.Status, decoded.Status)
	assert.Equal(t, event.Direction, decoded.Direction)
	assert.Equal(t, event.From, decoded.From)
	assert.Equal(t, event.Digits, decoded.Digits)
}

// --- IVRGather with nested actions ---

func TestIVRGather_NestedActions(t *testing.T) {
	gather := IVRGather{
		Input:       []string{"dtmf", "speech"},
		Timeout:     10,
		NumDigits:   1,
		FinishOnKey: "#",
		ActionURL:   "https://example.com/action",
		Language:    "pt-BR",
		Hints:       []string{"sim", "nao"},
		Nested: []IVRAction{
			IVRSay{Text: "Press 1 for sales", Language: "en-US"},
			IVRPlay{URL: "https://example.com/beep.mp3"},
		},
	}

	assert.Equal(t, "gather", gather.ActionType())
	assert.Len(t, gather.Nested, 2)
	assert.Equal(t, "say", gather.Nested[0].ActionType())
	assert.Equal(t, "play", gather.Nested[1].ActionType())
}

// --- Helper function tests ---

func TestBuildIVRMenu(t *testing.T) {
	options := map[string]string{
		"1": "vendas",
		"2": "suporte",
	}

	actions := BuildIVRMenu("Bem-vindo!", options, 5)

	require.Len(t, actions, 3)
	assert.Equal(t, "say", actions[0].ActionType())
	assert.Equal(t, "gather", actions[1].ActionType())
	assert.Equal(t, "say", actions[2].ActionType())

	say := actions[0].(IVRSay)
	assert.Equal(t, "Bem-vindo!", say.Text)
	assert.Equal(t, "pt-BR", say.Language)

	gather := actions[1].(IVRGather)
	assert.Equal(t, []string{"dtmf", "speech"}, gather.Input)
	assert.Equal(t, 5, gather.Timeout)
	assert.Equal(t, 1, gather.NumDigits)
	assert.Len(t, gather.Nested, 1)
}

func TestBuildCallbackRequest(t *testing.T) {
	actions := BuildCallbackRequest()

	require.Len(t, actions, 4)
	assert.Equal(t, "say", actions[0].ActionType())
	assert.Equal(t, "gather", actions[1].ActionType())
	assert.Equal(t, "say", actions[2].ActionType())
	assert.Equal(t, "hangup", actions[3].ActionType())

	gather := actions[1].(IVRGather)
	assert.Equal(t, 11, gather.NumDigits)
	assert.Equal(t, "#", gather.FinishOnKey)
}

func TestBuildHoldMusic(t *testing.T) {
	actions := BuildHoldMusic("https://example.com/music.mp3", "Please wait", 30)

	require.Len(t, actions, 2)
	assert.Equal(t, "say", actions[0].ActionType())
	assert.Equal(t, "play", actions[1].ActionType())

	play := actions[1].(IVRPlay)
	assert.Equal(t, "https://example.com/music.mp3", play.URL)
	assert.Equal(t, 10, play.Loop)
}
