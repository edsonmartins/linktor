package entity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChannel(t *testing.T) {
	ch := NewChannel("tenant1", ChannelTypeWhatsApp, "My Channel", "+5511999999999")
	assert.Equal(t, "tenant1", ch.TenantID)
	assert.Equal(t, ChannelTypeWhatsApp, ch.Type)
	assert.Equal(t, "My Channel", ch.Name)
	assert.True(t, ch.Enabled)
	assert.Equal(t, ConnectionStatusDisconnected, ch.ConnectionStatus)
	assert.NotNil(t, ch.Config)
}

func TestDefaultAdvancedSettings(t *testing.T) {
	s := DefaultAdvancedSettings()
	assert.Equal(t, 5, s.QRCodeMaxCount)
	assert.False(t, s.AlwaysOnline)
	assert.False(t, s.RejectCall)
	assert.Empty(t, s.RejectCallMsg)
	assert.False(t, s.AutoReadMessages)
	assert.False(t, s.IgnoreGroups)
	assert.False(t, s.IgnoreStatus)
	assert.Empty(t, s.ProxyHost)
	assert.Zero(t, s.ProxyPort)
	assert.Empty(t, s.ProxyUser)
	assert.Empty(t, s.ProxyPass)
}

func TestChannel_GetAdvancedSettings_Empty(t *testing.T) {
	ch := &Channel{}
	s := ch.GetAdvancedSettings()
	assert.Equal(t, 5, s.QRCodeMaxCount)
	assert.False(t, s.AlwaysOnline)
	assert.False(t, s.RejectCall)
}

func TestChannel_SetAndGetAdvancedSettings_RoundTrip(t *testing.T) {
	ch := NewChannel("t1", ChannelTypeWhatsApp, "test", "123")

	input := &AdvancedSettings{
		AlwaysOnline:     true,
		RejectCall:       true,
		RejectCallMsg:    "Cannot take calls right now",
		AutoReadMessages: true,
		IgnoreGroups:     false,
		IgnoreStatus:     true,
		QRCodeMaxCount:   10,
		ProxyHost:        "proxy.example.com",
		ProxyPort:        1080,
		ProxyUser:        "user",
		ProxyPass:        "pass",
	}

	ch.SetAdvancedSettings(input)
	output := ch.GetAdvancedSettings()

	assert.Equal(t, input.AlwaysOnline, output.AlwaysOnline)
	assert.Equal(t, input.RejectCall, output.RejectCall)
	assert.Equal(t, input.RejectCallMsg, output.RejectCallMsg)
	assert.Equal(t, input.AutoReadMessages, output.AutoReadMessages)
	assert.Equal(t, input.IgnoreGroups, output.IgnoreGroups)
	assert.Equal(t, input.IgnoreStatus, output.IgnoreStatus)
	assert.Equal(t, input.QRCodeMaxCount, output.QRCodeMaxCount)
	assert.Equal(t, input.ProxyHost, output.ProxyHost)
	assert.Equal(t, input.ProxyPort, output.ProxyPort)
	assert.Equal(t, input.ProxyUser, output.ProxyUser)
	assert.Equal(t, input.ProxyPass, output.ProxyPass)
}

func TestAdvancedSettings_HasProxy_True(t *testing.T) {
	s := &AdvancedSettings{
		ProxyHost: "proxy.example.com",
		ProxyPort: 1080,
	}
	assert.True(t, s.HasProxy())
}

func TestAdvancedSettings_HasProxy_False(t *testing.T) {
	s := &AdvancedSettings{}
	assert.False(t, s.HasProxy())
}

func TestChannel_SetAdvancedSettings_AllFields(t *testing.T) {
	ch := NewChannel("t1", ChannelTypeWhatsApp, "test", "123")

	settings := &AdvancedSettings{
		AlwaysOnline:     true,
		RejectCall:       true,
		RejectCallMsg:    "No calls please",
		AutoReadMessages: true,
		IgnoreGroups:     true,
		IgnoreStatus:     true,
		QRCodeMaxCount:   3,
		ProxyHost:        "socks.example.com",
		ProxyPort:        9050,
		ProxyUser:        "admin",
		ProxyPass:        "secret",
	}

	ch.SetAdvancedSettings(settings)

	assert.Equal(t, "true", ch.Config["always_online"])
	assert.Equal(t, "true", ch.Config["reject_call"])
	assert.Equal(t, "No calls please", ch.Config["reject_call_msg"])
	assert.Equal(t, "true", ch.Config["auto_read_messages"])
	assert.Equal(t, "true", ch.Config["ignore_groups"])
	assert.Equal(t, "true", ch.Config["ignore_status"])
	assert.Equal(t, "3", ch.Config["qrcode_max_count"])
	assert.Equal(t, "socks.example.com", ch.Config["proxy_host"])
	assert.Equal(t, "9050", ch.Config["proxy_port"])
	assert.Equal(t, "admin", ch.Config["proxy_user"])
	assert.Equal(t, "secret", ch.Config["proxy_pass"])
}

func TestChannel_MarshalJSON_RedactsSecrets(t *testing.T) {
	ch := NewChannel("tenant1", ChannelTypeWhatsAppOfficial, "WA", "+55")
	ch.Config["access_token"] = "EAAG-super-secret"
	ch.Config["app_secret"] = "app-secret-value"
	ch.Config["widget_secret"] = "widget-secret-value"
	ch.Config["phone_number_id"] = "123456" // non-secret, must stay
	ch.Config["empty_secret_token"] = ""    // empty sensitive-ish key stays as-is
	ch.Credentials["api_key"] = "must-not-leak"

	raw, err := json.Marshal(ch)
	assert.NoError(t, err)
	s := string(raw)

	// Secrets are masked, real values never appear.
	assert.NotContains(t, s, "EAAG-super-secret")
	assert.NotContains(t, s, "app-secret-value")
	assert.NotContains(t, s, "widget-secret-value")
	assert.Contains(t, s, RedactedSecret)
	// Non-secret config is preserved and credentials never serialize.
	assert.Contains(t, s, "123456")
	assert.NotContains(t, s, "must-not-leak")

	// The in-memory struct is untouched (redaction happens on a copy).
	assert.Equal(t, "EAAG-super-secret", ch.Config["access_token"])
}

func TestNewChannel_DefaultsToProductionEnvironment(t *testing.T) {
	ch := NewChannel("tenant1", ChannelTypeWhatsApp, "My Channel", "+5511999999999")
	assert.Equal(t, ChannelEnvironmentProduction, ch.Environment)
	assert.False(t, ch.IsSandbox())
}

func TestParseChannelEnvironment(t *testing.T) {
	tests := []struct {
		input string
		want  ChannelEnvironment
		ok    bool
	}{
		{"", ChannelEnvironmentProduction, true},
		{"production", ChannelEnvironmentProduction, true},
		{"sandbox", ChannelEnvironmentSandbox, true},
		{"staging", "", false},
		{"SANDBOX", "", false},
		{"prod", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseChannelEnvironment(tt.input)
		assert.Equal(t, tt.ok, ok, "input %q", tt.input)
		assert.Equal(t, tt.want, got, "input %q", tt.input)
	}
}

func TestChannel_IsSandbox(t *testing.T) {
	ch := NewChannel("tenant1", ChannelTypeWhatsAppOfficial, "Sandbox", "+5511999999999")
	ch.Environment = ChannelEnvironmentSandbox
	assert.True(t, ch.IsSandbox())

	// Zero-value / legacy struct without environment is production.
	assert.False(t, (&Channel{}).IsSandbox())
}
