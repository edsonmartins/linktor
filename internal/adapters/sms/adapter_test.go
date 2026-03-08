package sms

import (
	"testing"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()
	require.NotNil(t, a)

	info := a.GetChannelInfo()
	assert.Equal(t, plugin.ChannelTypeSMS, info.Type)
	assert.Equal(t, "SMS (Twilio)", info.Name)

	caps := info.Capabilities
	require.NotNil(t, caps)
	assert.True(t, caps.SupportsMedia) // MMS
	assert.False(t, caps.SupportsLocation)
	assert.False(t, caps.SupportsTemplates)
	assert.False(t, caps.SupportsInteractive)
	assert.False(t, caps.SupportsReadReceipts)
	assert.False(t, caps.SupportsTypingIndicator)
	assert.Equal(t, 1600, caps.MaxMessageLength)
	assert.Equal(t, int64(5*1024*1024), caps.MaxMediaSize)
	assert.Equal(t, 10, caps.MaxAttachments)
}

func TestAdapter_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]string
		wantErr string
	}{
		{
			name: "valid with auth_token and phone",
			config: map[string]string{
				"account_sid":  "AC123",
				"auth_token":   "token123",
				"phone_number": "+15551234567",
			},
		},
		{
			name: "valid with api_key and messaging_service",
			config: map[string]string{
				"account_sid":          "AC123",
				"api_key_sid":          "SK123",
				"api_key_secret":       "secret",
				"messaging_service_sid": "MG123",
			},
		},
		{
			name:    "missing account_sid",
			config:  map[string]string{"auth_token": "t", "phone_number": "+1555"},
			wantErr: "account_sid is required",
		},
		{
			name:    "missing auth credentials",
			config:  map[string]string{"account_sid": "AC123", "phone_number": "+1555"},
			wantErr: "auth_token or api_key",
		},
		{
			name:    "missing sender",
			config:  map[string]string{"account_sid": "AC123", "auth_token": "token"},
			wantErr: "phone_number or messaging_service_sid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAdapter()
			err := a.Initialize(tt.config)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseMessageStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected MessageStatus
	}{
		{"queued", StatusQueued},
		{"sending", StatusSending},
		{"sent", StatusSent},
		{"delivered", StatusDelivered},
		{"undelivered", StatusUndelivered},
		{"failed", StatusFailed},
		{"received", StatusReceived},
		{"accepted", StatusAccepted},
		{"scheduled", StatusScheduled},
		{"read", StatusRead},
		{"canceled", StatusCanceled},
		{"unknown", MessageStatus("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseMessageStatus(tt.input))
		})
	}
}

func TestMessageStatus_IsTerminalStatus(t *testing.T) {
	assert.True(t, StatusDelivered.IsTerminalStatus())
	assert.True(t, StatusUndelivered.IsTerminalStatus())
	assert.True(t, StatusFailed.IsTerminalStatus())
	assert.True(t, StatusCanceled.IsTerminalStatus())
	assert.False(t, StatusQueued.IsTerminalStatus())
	assert.False(t, StatusSending.IsTerminalStatus())
	assert.False(t, StatusSent.IsTerminalStatus())
}

func TestMessageStatus_IsSuccessStatus(t *testing.T) {
	assert.True(t, StatusDelivered.IsSuccessStatus())
	assert.True(t, StatusRead.IsSuccessStatus())
	assert.False(t, StatusSent.IsSuccessStatus())
	assert.False(t, StatusFailed.IsSuccessStatus())
}

func TestMessageStatus_IsFailureStatus(t *testing.T) {
	assert.True(t, StatusFailed.IsFailureStatus())
	assert.True(t, StatusUndelivered.IsFailureStatus())
	assert.False(t, StatusDelivered.IsFailureStatus())
	assert.False(t, StatusQueued.IsFailureStatus())
}
