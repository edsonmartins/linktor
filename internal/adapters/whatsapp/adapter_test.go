package whatsapp

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AdapterTestSuite tests the WhatsApp adapter
type AdapterTestSuite struct {
	suite.Suite
	fixtures *TestFixtures
}

func TestAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(AdapterTestSuite))
}

func (suite *AdapterTestSuite) SetupTest() {
	suite.fixtures = NewTestFixtures()
}

// NewAdapter tests
func (suite *AdapterTestSuite) TestNewAdapter_CreatesValidAdapter() {
	adapter := NewAdapter()

	assert.NotNil(suite.T(), adapter)
	assert.Equal(suite.T(), plugin.ChannelTypeWhatsApp, adapter.GetChannelType())
}

func (suite *AdapterTestSuite) TestNewAdapter_HasCorrectInfo() {
	adapter := NewAdapter()
	info := adapter.GetChannelInfo()

	assert.NotNil(suite.T(), info)
	assert.Equal(suite.T(), plugin.ChannelTypeWhatsApp, info.Type)
	assert.Equal(suite.T(), "WhatsApp (Unofficial)", info.Name)
	assert.Contains(suite.T(), info.Description, "whatsmeow")
	assert.Equal(suite.T(), "1.0.0", info.Version)
}

func (suite *AdapterTestSuite) TestNewAdapter_HasCorrectCapabilities() {
	adapter := NewAdapter()
	caps := adapter.GetCapabilities()

	assert.NotNil(suite.T(), caps)

	// Check supported content types
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeText)
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeImage)
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeVideo)
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeAudio)
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeDocument)
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeLocation)
	assert.Contains(suite.T(), caps.SupportedContentTypes, plugin.ContentTypeContact)

	// Check feature support
	assert.True(suite.T(), caps.SupportsMedia)
	assert.True(suite.T(), caps.SupportsLocation)
	assert.True(suite.T(), caps.SupportsReadReceipts)
	assert.True(suite.T(), caps.SupportsTypingIndicator)
	assert.True(suite.T(), caps.SupportsReactions)
	assert.True(suite.T(), caps.SupportsReplies)

	// Unofficial API limitations
	assert.False(suite.T(), caps.SupportsTemplates)
	assert.False(suite.T(), caps.SupportsInteractive)
	assert.False(suite.T(), caps.SupportsForwarding)

	// Check limits
	assert.Equal(suite.T(), 65536, caps.MaxMessageLength)
	assert.Equal(suite.T(), int64(100*1024*1024), caps.MaxMediaSize)
	assert.Equal(suite.T(), 1, caps.MaxAttachments)
}

// Initialize tests
func (suite *AdapterTestSuite) TestInitialize_WithValidConfig() {
	adapter := NewAdapter()
	config := suite.fixtures.ValidConfig()

	err := adapter.Initialize(config)

	assert.NoError(suite.T(), err)
}

func (suite *AdapterTestSuite) TestInitialize_WithMinimalConfig() {
	adapter := NewAdapter()
	config := suite.fixtures.MinimalConfig()

	err := adapter.Initialize(config)

	assert.NoError(suite.T(), err)
}

func (suite *AdapterTestSuite) TestInitialize_SetsConfigValues() {
	adapter := NewAdapter()
	config := suite.fixtures.ValidConfig()

	err := adapter.Initialize(config)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-channel-123", adapter.config.ChannelID)
	assert.Equal(suite.T(), "/tmp/test_whatsapp.db", adapter.config.DatabasePath)
	assert.Equal(suite.T(), "TestDevice", adapter.config.DeviceName)
}

// Connection status tests
func (suite *AdapterTestSuite) TestGetConnectionStatus_WhenDisconnected() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	status := adapter.GetConnectionStatus()

	assert.NotNil(suite.T(), status)
	assert.False(suite.T(), status.Connected)
	assert.Equal(suite.T(), "disconnected", status.Status)
}

func (suite *AdapterTestSuite) TestIsConnected_WhenNotInitialized() {
	adapter := NewAdapter()

	assert.False(suite.T(), adapter.IsConnected())
}

func (suite *AdapterTestSuite) TestIsLoggedIn_WhenNotInitialized() {
	adapter := NewAdapter()

	assert.False(suite.T(), adapter.IsLoggedIn())
}

// GetDeviceInfo tests
func (suite *AdapterTestSuite) TestGetDeviceInfo_WhenNoClient() {
	adapter := NewAdapter()

	info := adapter.GetDeviceInfo()

	assert.NotNil(suite.T(), info)
	assert.Equal(suite.T(), DeviceStateDisconnected, info.State)
}

// Handler tests
func (suite *AdapterTestSuite) TestSetMessageHandler_SetsHandler() {
	adapter := NewAdapter()
	mockHandler := &MockMessageHandler{}

	adapter.SetMessageHandler(mockHandler.Handler())

	assert.NotNil(suite.T(), adapter.messageHandler)
}

func (suite *AdapterTestSuite) TestSetStatusHandler_SetsHandler() {
	adapter := NewAdapter()
	mockHandler := &MockStatusHandler{}

	adapter.SetStatusHandler(mockHandler.Handler())

	assert.NotNil(suite.T(), adapter.statusHandler)
}

// SendMessage tests (without actual connection)
func (suite *AdapterTestSuite) TestSendMessage_WhenNotConnected() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	msg := suite.fixtures.SampleTextOutbound("5511999999999", "Test message")
	ctx := context.Background()

	result, err := adapter.SendMessage(ctx, msg)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.Success)
	assert.Equal(suite.T(), plugin.MessageStatusFailed, result.Status)
	assert.Contains(suite.T(), result.Error, "not connected")
}

// SendTypingIndicator tests
func (suite *AdapterTestSuite) TestSendTypingIndicator_WhenNotConnected() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	indicator := suite.fixtures.SampleTypingIndicator("5511999999999", true)
	ctx := context.Background()

	err := adapter.SendTypingIndicator(ctx, indicator)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

// SendReadReceipt tests
func (suite *AdapterTestSuite) TestSendReadReceipt_WhenNotConnected() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	receipt := suite.fixtures.SampleReadReceipt("5511999999999", "msg-123")
	ctx := context.Background()

	err := adapter.SendReadReceipt(ctx, receipt)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

// UploadMedia tests
func (suite *AdapterTestSuite) TestUploadMedia_ReturnsTemporaryID() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	media := &plugin.Media{
		Data:     []byte("fake image data"),
		MimeType: "image/jpeg",
	}
	ctx := context.Background()

	result, err := adapter.UploadMedia(ctx, media)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)
	assert.NotEmpty(suite.T(), result.MediaID)
}

// DownloadMedia tests
func (suite *AdapterTestSuite) TestDownloadMedia_ReturnsError() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	ctx := context.Background()

	media, err := adapter.DownloadMedia(ctx, "some-media-id")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), media)
	assert.Contains(suite.T(), err.Error(), "not supported")
}

// convertToInboundMessage tests
func (suite *AdapterTestSuite) TestConvertToInboundMessage_TextMessage() {
	msg := suite.fixtures.SampleIncomingMessage("msg-1", "5511999999999", "Hello", false)

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.NotEmpty(suite.T(), result.ID)
	assert.Equal(suite.T(), "msg-1", result.ExternalID)
	assert.Equal(suite.T(), "5511999999999", result.SenderID)
	assert.Equal(suite.T(), "Test Sender", result.SenderName)
	assert.Equal(suite.T(), "Hello", result.Content)
	assert.Equal(suite.T(), plugin.ContentTypeText, result.ContentType)
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_GroupMessage() {
	msg := suite.fixtures.SampleIncomingMessage("msg-2", "5511999999999", "Group hello", true)

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "true", result.Metadata["is_group"])
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_ImageMessage() {
	msg := suite.fixtures.SampleIncomingMessage("msg-3", "5511999999999", "Image caption", false)
	msg.MessageType = "image"
	msg.Attachments = []Attachment{
		{
			Type:     "image",
			URL:      "https://example.com/image.jpg",
			MimeType: "image/jpeg",
			FileSize: 12345,
		},
	}

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), plugin.ContentTypeImage, result.ContentType)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "image/jpeg", result.Attachments[0].MimeType)
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_WithReply() {
	msg := suite.fixtures.SampleIncomingMessage("msg-4", "5511999999999", "Reply text", false)
	msg.ReplyTo = &ReplyInfo{
		MessageID: "original-msg-123",
		Text:      "Original text",
	}

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "original-msg-123", result.Metadata["reply_to_id"])
	assert.Equal(suite.T(), "Original text", result.Metadata["quoted_text"])
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_WithReaction() {
	msg := suite.fixtures.SampleIncomingMessage("msg-5", "5511999999999", "", false)
	msg.Reaction = &Reaction{
		Emoji:     "👍",
		MessageID: "target-msg-123",
		Timestamp: time.Now(),
	}

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "👍", result.Metadata["reaction"])
	assert.Equal(suite.T(), "target-msg-123", result.Metadata["reaction_message_id"])
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_StickerAsImage() {
	msg := suite.fixtures.SampleIncomingMessage("msg-6", "5511999999999", "", false)
	msg.MessageType = "sticker"

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), plugin.ContentTypeImage, result.ContentType)
	assert.Equal(suite.T(), "true", result.Metadata["is_sticker"])
}

// convertToStatusCallback tests
func (suite *AdapterTestSuite) TestConvertToStatusCallback_Delivered() {
	receipt := suite.fixtures.SampleReceipt([]string{"msg-1"}, ReceiptTypeDelivered)

	result := convertToStatusCallback(receipt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "msg-1", result.MessageID)
	assert.Equal(suite.T(), "msg-1", result.ExternalID)
	assert.Equal(suite.T(), plugin.MessageStatusDelivered, result.Status)
}

func (suite *AdapterTestSuite) TestConvertToStatusCallback_Read() {
	receipt := suite.fixtures.SampleReceipt([]string{"msg-2"}, ReceiptTypeRead)

	result := convertToStatusCallback(receipt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), plugin.MessageStatusRead, result.Status)
}

func (suite *AdapterTestSuite) TestConvertToStatusCallback_MultipleMessages() {
	receipt := suite.fixtures.SampleReceipt([]string{"msg-1", "msg-2", "msg-3"}, ReceiptTypeRead)

	result := convertToStatusCallback(receipt)

	assert.NotNil(suite.T(), result)
	// Only first message ID is returned
	assert.Equal(suite.T(), "msg-1", result.MessageID)
}

func (suite *AdapterTestSuite) TestConvertToStatusCallback_EmptyMessages() {
	receipt := suite.fixtures.SampleReceipt([]string{}, ReceiptTypeDelivered)

	result := convertToStatusCallback(receipt)

	assert.NotNil(suite.T(), result)
	assert.Empty(suite.T(), result.MessageID)
}

// Utility function tests
func (suite *AdapterTestSuite) TestIndexOf_Found() {
	result := indexOf("hello world", "world")
	assert.Equal(suite.T(), 6, result)
}

func (suite *AdapterTestSuite) TestIndexOf_NotFound() {
	result := indexOf("hello world", "foo")
	assert.Equal(suite.T(), -1, result)
}

func (suite *AdapterTestSuite) TestIndexOf_EmptySubstring() {
	result := indexOf("hello", "")
	assert.Equal(suite.T(), 0, result)
}

func (suite *AdapterTestSuite) TestIndexOf_LongerSubstring() {
	result := indexOf("hi", "hello")
	assert.Equal(suite.T(), -1, result)
}

// GetClient tests
func (suite *AdapterTestSuite) TestGetClient_WhenNoClient() {
	adapter := NewAdapter()

	client := adapter.GetClient()

	assert.Nil(suite.T(), client)
}
