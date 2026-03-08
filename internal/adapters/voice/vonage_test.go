package voice

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestRSAKeyPEM generates a PEM-encoded RSA private key for testing.
func generateTestRSAKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}
	return string(pem.EncodeToMemory(pemBlock))
}

func TestNewVonageProvider_Name(t *testing.T) {
	p := NewVonageProvider()
	assert.Equal(t, "vonage", p.Name())
}

func TestVonageProvider_Initialize_Success(t *testing.T) {
	keyPEM := generateTestRSAKeyPEM(t)

	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		PhoneNumber: "+5511999999999",
		Credentials: map[string]string{
			"application_id": "app-123",
			"private_key":    keyPEM,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "app-123", p.applicationID)
	assert.NotNil(t, p.privateKey)
	assert.Equal(t, "+5511999999999", p.phoneNumber)
}

func TestVonageProvider_Initialize_MissingApplicationID(t *testing.T) {
	keyPEM := generateTestRSAKeyPEM(t)

	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"private_key": keyPEM,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "application_id")
}

func TestVonageProvider_Initialize_EmptyApplicationID(t *testing.T) {
	keyPEM := generateTestRSAKeyPEM(t)

	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "",
			"private_key":    keyPEM,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "application_id")
}

func TestVonageProvider_Initialize_MissingPrivateKey(t *testing.T) {
	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "app-123",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private_key")
}

func TestVonageProvider_Initialize_EmptyPrivateKey(t *testing.T) {
	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "app-123",
			"private_key":    "",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private_key")
}

func TestVonageProvider_Initialize_InvalidPEM(t *testing.T) {
	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "app-123",
			"private_key":    "not-a-valid-pem",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private key PEM")
}

func TestVonageProvider_Initialize_InvalidKeyBytes(t *testing.T) {
	// Valid PEM block but invalid key bytes
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: []byte("not valid key bytes"),
	}
	badPEM := string(pem.EncodeToMemory(pemBlock))

	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "app-123",
			"private_key":    badPEM,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestVonageProvider_Capabilities(t *testing.T) {
	p := NewVonageProvider()
	caps := p.Capabilities()

	assert.True(t, caps.OutboundCalls)
	assert.True(t, caps.InboundCalls)
	assert.True(t, caps.Recording)
	assert.True(t, caps.Transcription)
	assert.True(t, caps.TextToSpeech)
	assert.True(t, caps.SpeechRecognition)
	assert.True(t, caps.DTMF)
	assert.True(t, caps.Conferencing)
	assert.False(t, caps.CallQueues)
	assert.True(t, caps.SIP)
	assert.True(t, caps.WebRTC)
}

// --- GenerateIVRResponse tests (NCCO format) ---

func TestVonageProvider_GenerateIVRResponse_Say(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRSay{Text: "Hello world", Language: "en-US", Loop: 2},
	})
	require.NoError(t, err)

	ncco, ok := resp.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, ncco, 1)

	assert.Equal(t, "talk", ncco[0]["action"])
	assert.Equal(t, "Hello world", ncco[0]["text"])
	assert.Equal(t, "en-US", ncco[0]["language"])
	assert.Equal(t, 2, ncco[0]["loop"])
}

func TestVonageProvider_GenerateIVRResponse_SayWithVoice(t *testing.T) {
	p := NewVonageProvider()

	tests := []struct {
		voice    string
		expected int
	}{
		{"man", 0},
		{"woman", 1},
		{"custom", 0}, // default
	}

	for _, tt := range tests {
		t.Run(tt.voice, func(t *testing.T) {
			resp, err := p.GenerateIVRResponse([]IVRAction{
				IVRSay{Text: "test", Language: "en-US", Voice: tt.voice},
			})
			require.NoError(t, err)

			ncco := resp.([]map[string]interface{})
			assert.Equal(t, tt.expected, ncco[0]["style"])
		})
	}
}

func TestVonageProvider_GenerateIVRResponse_Play(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRPlay{URL: "https://example.com/audio.mp3", Loop: 3},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)

	assert.Equal(t, "stream", ncco[0]["action"])
	streamURL := ncco[0]["streamUrl"].([]string)
	assert.Equal(t, "https://example.com/audio.mp3", streamURL[0])
	assert.Equal(t, 3, ncco[0]["loop"])
}

func TestVonageProvider_GenerateIVRResponse_Gather(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRGather{
			Input:       []string{"dtmf", "speech"},
			Timeout:     10,
			NumDigits:   4,
			FinishOnKey: "#",
			ActionURL:   "https://example.com/input",
			Language:    "pt-BR",
			Hints:       []string{"sim", "nao"},
		},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)

	assert.Equal(t, "input", ncco[0]["action"])

	dtmf := ncco[0]["dtmf"].(map[string]interface{})
	assert.Equal(t, 10, dtmf["timeOut"])
	assert.Equal(t, 4, dtmf["maxDigits"])
	assert.Equal(t, true, dtmf["submitOnHash"])

	speech := ncco[0]["speech"].(map[string]interface{})
	assert.Equal(t, "pt-BR", speech["language"])
	assert.Equal(t, []string{"sim", "nao"}, speech["context"])

	eventURL := ncco[0]["eventUrl"].([]string)
	assert.Equal(t, "https://example.com/input", eventURL[0])
}

func TestVonageProvider_GenerateIVRResponse_GatherWithNestedSay(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRGather{
			Input:   []string{"dtmf"},
			Timeout: 5,
			Nested: []IVRAction{
				IVRSay{Text: "Press 1 for sales", Language: "en-US"},
			},
		},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	// Nested say becomes a separate "talk" action before "input"
	require.Len(t, ncco, 2)

	assert.Equal(t, "talk", ncco[0]["action"])
	assert.Equal(t, "Press 1 for sales", ncco[0]["text"])
	assert.Equal(t, "input", ncco[1]["action"])
}

func TestVonageProvider_GenerateIVRResponse_Record(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRRecord{
			MaxLength:   60,
			FinishOnKey: "#",
			PlayBeep:    true,
			Transcribe:  true,
			ActionURL:   "https://example.com/record",
		},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)

	assert.Equal(t, "record", ncco[0]["action"])
	assert.Equal(t, "mp3", ncco[0]["format"])
	assert.Equal(t, 60, ncco[0]["timeOut"])
	assert.Equal(t, "#", ncco[0]["endOnKey"])
	assert.Equal(t, true, ncco[0]["beepStart"])
	assert.NotNil(t, ncco[0]["transcription"])

	eventURL := ncco[0]["eventUrl"].([]string)
	assert.Equal(t, "https://example.com/record", eventURL[0])
}

func TestVonageProvider_GenerateIVRResponse_DialNumber(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRDial{
			Number:   "+5511999999999",
			Timeout:  30,
			CallerID: "+5511888888888",
			ActionURL: "https://example.com/dial",
		},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)

	assert.Equal(t, "connect", ncco[0]["action"])
	assert.Equal(t, 30, ncco[0]["timeout"])
	assert.Equal(t, "5511888888888", ncco[0]["from"])

	endpoints := ncco[0]["endpoint"].([]map[string]interface{})
	require.Len(t, endpoints, 1)
	assert.Equal(t, "phone", endpoints[0]["type"])
	assert.Equal(t, "5511999999999", endpoints[0]["number"])
}

func TestVonageProvider_GenerateIVRResponse_DialSIP(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRDial{SIPEndpoint: "sip:user@example.com"},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)

	endpoints := ncco[0]["endpoint"].([]map[string]interface{})
	assert.Equal(t, "sip", endpoints[0]["type"])
	assert.Equal(t, "sip:user@example.com", endpoints[0]["uri"])
}

func TestVonageProvider_GenerateIVRResponse_Conference(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRConference{
			Name:         "room1",
			Muted:        true,
			StartOnEnter: true,
			EndOnExit:    true,
			Record:       true,
			WaitURL:      "https://example.com/hold.mp3",
		},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)

	assert.Equal(t, "conversation", ncco[0]["action"])
	assert.Equal(t, "room1", ncco[0]["name"])
	assert.Equal(t, true, ncco[0]["mute"])
	assert.Equal(t, true, ncco[0]["startOnEnter"])
	assert.Equal(t, true, ncco[0]["endOnExit"])
	assert.Equal(t, true, ncco[0]["record"])

	musicURL := ncco[0]["musicOnHoldUrl"].([]string)
	assert.Equal(t, "https://example.com/hold.mp3", musicURL[0])
}

func TestVonageProvider_GenerateIVRResponse_Pause(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRPause{Length: 3},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 1)
	assert.Equal(t, "stream", ncco[0]["action"])
}

func TestVonageProvider_GenerateIVRResponse_Hangup(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRHangup{},
	})
	require.NoError(t, err)

	// Vonage has no explicit hangup in NCCO
	ncco := resp.([]map[string]interface{})
	assert.Nil(t, ncco) // no actions generated for hangup
}

func TestVonageProvider_GenerateIVRResponse_EmptyActions(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse(nil)
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	assert.Nil(t, ncco)
}

func TestVonageProvider_GenerateIVRResponse_MultipleActions(t *testing.T) {
	p := NewVonageProvider()
	resp, err := p.GenerateIVRResponse([]IVRAction{
		IVRSay{Text: "Welcome", Language: "pt-BR"},
		IVRPlay{URL: "https://example.com/music.mp3"},
		IVRGather{Input: []string{"dtmf"}, Timeout: 5},
	})
	require.NoError(t, err)

	ncco := resp.([]map[string]interface{})
	require.Len(t, ncco, 3)
	assert.Equal(t, "talk", ncco[0]["action"])
	assert.Equal(t, "stream", ncco[1]["action"])
	assert.Equal(t, "input", ncco[2]["action"])
}

// --- ParseWebhook tests ---

func TestVonageProvider_ParseWebhook_StatusEvent(t *testing.T) {
	p := NewVonageProvider()

	body := `{"uuid":"call-123","status":"answered","from":"+5511999999999","to":"+5511888888888","direction":"inbound"}`

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "status", event.Type)
	assert.Equal(t, "call-123", event.CallID)
	assert.Equal(t, CallStatusAnswered, event.Status)
	assert.Equal(t, "+5511999999999", event.From)
	assert.Equal(t, "+5511888888888", event.To)
	assert.Equal(t, CallDirectionInbound, event.Direction)
}

func TestVonageProvider_ParseWebhook_DTMFEvent(t *testing.T) {
	p := NewVonageProvider()

	body := `{"uuid":"call-123","dtmf":{"digits":"42","timed_out":false}}`

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "dtmf", event.Type)
	assert.Equal(t, "42", event.Digits)
}

func TestVonageProvider_ParseWebhook_SpeechEvent(t *testing.T) {
	p := NewVonageProvider()

	body := `{"uuid":"call-123","speech":{"results":[{"text":"yes please","confidence":0.9}]}}`

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "speech", event.Type)
	assert.Equal(t, "yes please", event.SpeechResult)
}

func TestVonageProvider_ParseWebhook_RecordingEvent(t *testing.T) {
	p := NewVonageProvider()

	body := `{"uuid":"call-123","recording_url":"https://api.nexmo.com/v1/files/rec-123"}`

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "recording", event.Type)
	assert.Equal(t, "https://api.nexmo.com/v1/files/rec-123", event.RecordingURL)
}

func TestVonageProvider_ParseWebhook_ConversationUUID(t *testing.T) {
	p := NewVonageProvider()

	body := `{"conversation_uuid":"conv-123","status":"completed"}`

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, "conv-123", event.CallID)
}

func TestVonageProvider_ParseWebhook_WithDuration(t *testing.T) {
	p := NewVonageProvider()

	body := `{"uuid":"call-123","status":"completed","duration":"120","direction":"outbound"}`

	event, err := p.ParseWebhook(context.Background(), nil, []byte(body))
	require.NoError(t, err)

	assert.Equal(t, 120, event.Duration)
	assert.Equal(t, CallDirectionOutbound, event.Direction)
}

func TestVonageProvider_ParseWebhook_InvalidJSON(t *testing.T) {
	p := NewVonageProvider()

	_, err := p.ParseWebhook(context.Background(), nil, []byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse webhook body")
}

// --- ValidateWebhook ---

func TestVonageProvider_ValidateWebhook_AlwaysTrue(t *testing.T) {
	p := NewVonageProvider()

	result := p.ValidateWebhook(context.Background(), nil, nil)
	assert.True(t, result)
}

// --- mapStatus tests ---

func TestVonageProvider_MapStatus(t *testing.T) {
	p := NewVonageProvider()

	tests := []struct {
		input    string
		expected CallStatus
	}{
		{"started", CallStatusInitiated},
		{"ringing", CallStatusRinging},
		{"answered", CallStatusAnswered},
		{"completed", CallStatusCompleted},
		{"busy", CallStatusBusy},
		{"timeout", CallStatusNoAnswer},
		{"failed", CallStatusFailed},
		{"rejected", CallStatusFailed},
		{"cancelled", CallStatusCanceled},
		{"unknown", CallStatus("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, p.mapStatus(tt.input))
		})
	}
}

// --- GetRecording ---

func TestVonageProvider_GetRecording(t *testing.T) {
	keyPEM := generateTestRSAKeyPEM(t)

	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "app-123",
			"private_key":    keyPEM,
		},
	})
	require.NoError(t, err)

	rec, err := p.GetRecording(context.Background(), "rec-123")
	require.NoError(t, err)

	assert.Equal(t, "rec-123", rec.ID)
	assert.Equal(t, "mp3", rec.Format)
	assert.Contains(t, rec.URL, "rec-123")
}

// --- DeleteRecording ---

func TestVonageProvider_DeleteRecording(t *testing.T) {
	p := NewVonageProvider()
	err := p.DeleteRecording(context.Background(), "rec-123")
	assert.NoError(t, err) // Vonage doesn't have direct delete
}

// --- MakeCall with httptest ---

func TestVonageProvider_MakeCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/calls", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		toArr := payload["to"].([]interface{})
		toObj := toArr[0].(map[string]interface{})
		assert.Equal(t, "5511888888888", toObj["number"])

		fromObj := payload["from"].(map[string]interface{})
		assert.Equal(t, "5511999999999", fromObj["number"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"uuid":   "vonage-call-123",
			"status": "started",
		})
	}))
	defer server.Close()

	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	p.baseURL = server.URL + "/v1"
	p.applicationID = "app-123"
	// Parse the key
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	p.privateKey = key

	result, err := p.MakeCall(context.Background(), MakeCallInput{
		To:   "+5511888888888",
		From: "+5511999999999",
	})
	require.NoError(t, err)

	assert.Equal(t, "vonage-call-123", result.CallID)
	assert.Equal(t, CallStatusInitiated, result.Status)
}

// --- GetCall with httptest ---

func TestVonageProvider_GetCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/calls/vonage-call-123")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":      "vonage-call-123",
			"status":    "answered",
			"direction": "inbound",
			"from": map[string]string{
				"number": "5511999999999",
			},
			"to": map[string]string{
				"number": "5511888888888",
			},
			"duration": "60",
		})
	}))
	defer server.Close()

	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	p.baseURL = server.URL + "/v1"
	p.applicationID = "app-123"
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	p.privateKey = key

	call, err := p.GetCall(context.Background(), "vonage-call-123")
	require.NoError(t, err)

	assert.Equal(t, "vonage-call-123", call.ID)
	assert.Equal(t, CallStatusAnswered, call.Status)
	assert.Equal(t, CallDirectionInbound, call.Direction)
	assert.Equal(t, "5511999999999", call.From)
	assert.Equal(t, "5511888888888", call.To)
	assert.Equal(t, 60, call.Duration)
}

func TestVonageProvider_GetCall_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	p.baseURL = server.URL + "/v1"
	p.applicationID = "app-123"
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	p.privateKey = key

	call, err := p.GetCall(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, call)
	assert.Contains(t, err.Error(), "call not found")
}

// --- EndCall with httptest ---

func TestVonageProvider_EndCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/calls/vonage-call-123")

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "hangup", payload["action"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	p.baseURL = server.URL + "/v1"
	p.applicationID = "app-123"
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	p.privateKey = key

	err = p.EndCall(context.Background(), "vonage-call-123")
	assert.NoError(t, err)
}

func TestVonageProvider_EndCall_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	p.baseURL = server.URL + "/v1"
	p.applicationID = "app-123"
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	p.privateKey = key

	err = p.EndCall(context.Background(), "vonage-call-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to end call")
}

// --- TransferCall with httptest ---

func TestVonageProvider_TransferCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "transfer", payload["action"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	p.baseURL = server.URL + "/v1"
	p.applicationID = "app-123"
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	p.privateKey = key

	err = p.TransferCall(context.Background(), "vonage-call-123", "+5511777777777")
	assert.NoError(t, err)
}

// --- generateJWT test ---

func TestVonageProvider_GenerateJWT(t *testing.T) {
	keyPEM := generateTestRSAKeyPEM(t)
	p := NewVonageProvider()
	err := p.Initialize(context.Background(), VoiceConfig{
		Credentials: map[string]string{
			"application_id": "app-123",
			"private_key":    keyPEM,
		},
	})
	require.NoError(t, err)

	token, err := p.generateJWT()
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// Just check it looks like a JWT (3 dot-separated parts)
	assert.Equal(t, 2, countDots(token))
}

func countDots(s string) int {
	count := 0
	for _, c := range s {
		if c == '.' {
			count++
		}
	}
	return count
}
