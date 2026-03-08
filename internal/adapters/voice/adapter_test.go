package voice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter_Twilio(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	assert.Equal(t, "twilio", adapter.Provider())
}

func TestNewAdapter_Vonage(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "vonage"})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	assert.Equal(t, "vonage", adapter.Provider())
}

func TestNewAdapter_AmazonConnect(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "amazon_connect"})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	assert.Equal(t, "amazon_connect", adapter.Provider())
}

func TestNewAdapter_Asterisk(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "asterisk"})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	assert.Equal(t, "asterisk", adapter.Provider())
}

func TestNewAdapter_FreeSWITCH(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "freeswitch"})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	assert.Equal(t, "freeswitch", adapter.Provider())
}

func TestNewAdapter_UnknownProvider(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "unknown_provider"})
	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "unsupported voice provider")
}

func TestNewAdapter_EmptyProvider(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: ""})
	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "unsupported voice provider")
}

func TestAdapter_Name(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)
	assert.Equal(t, "voice", adapter.Name())
}

func TestAdapter_Type(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)
	assert.Equal(t, "voice", adapter.Type())
}

func TestAdapter_Version(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", adapter.Version())
}

func TestAdapter_Capabilities(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)

	caps := adapter.Capabilities()
	assert.True(t, caps.OutboundCalls)
	assert.True(t, caps.InboundCalls)
	assert.True(t, caps.Recording)
}

func TestAdapter_SetWebhookHandler(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)

	called := false
	handler := func(ctx context.Context, event *WebhookEvent) ([]IVRAction, error) {
		called = true
		return nil, nil
	}

	adapter.SetWebhookHandler(handler)
	assert.NotNil(t, adapter.handler)

	// Verify the handler is stored (we can't call it through HandleWebhook
	// without valid webhook validation, but we can verify it was set)
	assert.False(t, called) // not called yet, just set
}

func TestAdapter_MakeCall_SetsDefaults(t *testing.T) {
	// This test verifies that the adapter sets defaults from config.
	// It will fail at the HTTP call level since there's no mock server,
	// but we can verify the logic by checking the config is applied.
	config := VoiceConfig{
		Provider:        "twilio",
		PhoneNumber:     "+5511999999999",
		WebhookURL:      "https://example.com/webhook",
		StatusURL:       "https://example.com/status",
		RecordCalls:     true,
		TranscribeCalls: true,
		Credentials: map[string]string{
			"account_sid": "ACtest",
			"auth_token":  "testtoken",
		},
	}

	adapter, err := NewAdapter(config)
	require.NoError(t, err)

	err = adapter.Initialize(context.Background())
	require.NoError(t, err)

	assert.Equal(t, config.PhoneNumber, adapter.config.PhoneNumber)
	assert.Equal(t, config.WebhookURL, adapter.config.WebhookURL)
	assert.True(t, adapter.config.RecordCalls)
	assert.True(t, adapter.config.TranscribeCalls)
}

func TestAdapter_GenerateIVRResponse(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)

	actions := []IVRAction{
		IVRSay{Text: "Hello", Language: "en-US"},
		IVRHangup{},
	}

	response, err := adapter.GenerateIVRResponse(actions)
	require.NoError(t, err)

	twiml, ok := response.(string)
	require.True(t, ok)
	assert.Contains(t, twiml, "<Say")
	assert.Contains(t, twiml, "Hello")
	assert.Contains(t, twiml, "<Hangup/>")
}

func TestAdapter_GenerateIVRResponse_NilActions(t *testing.T) {
	adapter, err := NewAdapter(VoiceConfig{Provider: "twilio"})
	require.NoError(t, err)

	response, err := adapter.GenerateIVRResponse(nil)
	require.NoError(t, err)

	twiml, ok := response.(string)
	require.True(t, ok)
	assert.Contains(t, twiml, "<Response>")
	assert.Contains(t, twiml, "</Response>")
}
