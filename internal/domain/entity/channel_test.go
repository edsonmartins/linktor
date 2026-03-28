package entity

import (
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
