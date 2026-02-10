package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TypesTestSuite tests pure functions in types.go
type TypesTestSuite struct {
	suite.Suite
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

// Config.Validate() tests
func (suite *TypesTestSuite) TestConfig_Validate_EmptyChannelID() {
	config := &Config{
		ChannelID: "",
	}

	err := config.Validate()

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "channel_id is required")
}

func (suite *TypesTestSuite) TestConfig_Validate_ValidChannelID() {
	config := &Config{
		ChannelID: "test-channel-123",
	}

	err := config.Validate()

	assert.NoError(suite.T(), err)
}

// Config.SetDefaults() tests
func (suite *TypesTestSuite) TestConfig_SetDefaults_SetsAllDefaults() {
	config := &Config{
		ChannelID: "test-channel",
	}

	config.SetDefaults()

	assert.Equal(suite.T(), "storages/whatsapp_test-channel.db", config.DatabasePath)
	assert.Equal(suite.T(), "Linktor", config.DeviceName)
	assert.Equal(suite.T(), "chrome", config.PlatformType)
	assert.True(suite.T(), config.AutoReconnect)
	assert.True(suite.T(), config.AutoTrustIdentity)
}

func (suite *TypesTestSuite) TestConfig_SetDefaults_DoesNotOverwriteExisting() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/custom/path/db.sqlite",
		DeviceName:   "CustomDevice",
		PlatformType: "firefox",
	}

	config.SetDefaults()

	// Should not overwrite existing values
	assert.Equal(suite.T(), "/custom/path/db.sqlite", config.DatabasePath)
	assert.Equal(suite.T(), "CustomDevice", config.DeviceName)
	assert.Equal(suite.T(), "firefox", config.PlatformType)
	// These are always set
	assert.True(suite.T(), config.AutoReconnect)
	assert.True(suite.T(), config.AutoTrustIdentity)
}

// AttachmentTypeFromMIME() tests
func (suite *TypesTestSuite) TestAttachmentTypeFromMIME() {
	tests := []struct {
		name     string
		mimeType string
		expected string
	}{
		// Images
		{"JPEG image", "image/jpeg", "image"},
		{"PNG image", "image/png", "image"},
		{"GIF image", "image/gif", "image"},

		// Sticker (special case)
		{"WebP sticker", "image/webp", "sticker"},

		// Videos
		{"MP4 video", "video/mp4", "video"},
		{"3GPP video", "video/3gpp", "video"},
		{"WebM video", "video/webm", "video"},

		// Audio
		{"MP3 audio", "audio/mpeg", "audio"},
		{"OGG audio", "audio/ogg", "audio"},
		{"AAC audio", "audio/aac", "audio"},
		{"WAV audio", "audio/wav", "audio"},

		// Documents
		{"PDF document", "application/pdf", "document"},
		{"Word document", "application/msword", "document"},
		{"Excel document", "application/vnd.ms-excel", "document"},
		{"ZIP archive", "application/zip", "document"},
		{"JSON file", "application/json", "document"},

		// Edge cases
		{"Empty MIME type", "", "document"},
		{"Unknown MIME type", "unknown/type", "document"},
		{"Text plain", "text/plain", "document"},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := AttachmentTypeFromMIME(tt.mimeType)
			assert.Equal(suite.T(), tt.expected, result)
		})
	}
}

// DeviceState constants tests
func (suite *TypesTestSuite) TestDeviceState_Constants() {
	assert.Equal(suite.T(), DeviceState("disconnected"), DeviceStateDisconnected)
	assert.Equal(suite.T(), DeviceState("connecting"), DeviceStateConnecting)
	assert.Equal(suite.T(), DeviceState("connected"), DeviceStateConnected)
	assert.Equal(suite.T(), DeviceState("logged_out"), DeviceStateLoggedOut)
	assert.Equal(suite.T(), DeviceState("qr_pending"), DeviceStateQRPending)
}

// ReceiptType constants tests
func (suite *TypesTestSuite) TestReceiptType_Constants() {
	assert.Equal(suite.T(), ReceiptType("delivered"), ReceiptTypeDelivered)
	assert.Equal(suite.T(), ReceiptType("read"), ReceiptTypeRead)
	assert.Equal(suite.T(), ReceiptType("played"), ReceiptTypePlayed)
}

// ChatPresenceState constants tests
func (suite *TypesTestSuite) TestChatPresenceState_Constants() {
	assert.Equal(suite.T(), ChatPresenceState("composing"), ChatPresenceComposing)
	assert.Equal(suite.T(), ChatPresenceState("recording"), ChatPresenceRecording)
	assert.Equal(suite.T(), ChatPresenceState("paused"), ChatPresencePaused)
}

// MediaType constants tests
func (suite *TypesTestSuite) TestMediaType_Constants() {
	assert.Equal(suite.T(), MediaType("image"), MediaTypeImage)
	assert.Equal(suite.T(), MediaType("video"), MediaTypeVideo)
	assert.Equal(suite.T(), MediaType("audio"), MediaTypeAudio)
	assert.Equal(suite.T(), MediaType("document"), MediaTypeDocument)
	assert.Equal(suite.T(), MediaType("sticker"), MediaTypeSticker)
}
