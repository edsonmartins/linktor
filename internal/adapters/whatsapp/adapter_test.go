package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
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
	// Native-flow buttons/lists over the unofficial protocol (interactive.go).
	assert.True(suite.T(), caps.SupportsInteractive)
	// Text forward via metadata is_forwarded (edit.go).
	assert.True(suite.T(), caps.SupportsForwarding)

	// Unofficial API limitations
	assert.False(suite.T(), caps.SupportsTemplates)

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

func (suite *AdapterTestSuite) TestShouldForwardInbound_IgnoreGroups() {
	on := &Adapter{config: &Config{IgnoreGroups: true}}
	off := &Adapter{config: &Config{IgnoreGroups: false}}
	group := &IncomingMessage{IsGroup: true}
	direct := &IncomingMessage{IsGroup: false}
	mine := &IncomingMessage{IsGroup: false, IsFromMe: true}

	assert.False(suite.T(), on.shouldForwardInbound(group), "ignore_groups → grupo não é encaminhado")
	assert.True(suite.T(), on.shouldForwardInbound(direct), "1:1 flui mesmo com ignore_groups")
	assert.True(suite.T(), off.shouldForwardInbound(group), "sem ignore_groups, grupo flui (padrão)")
	assert.False(suite.T(), off.shouldForwardInbound(mine), "eco próprio nunca é encaminhado")
}

func (suite *AdapterTestSuite) TestShouldForwardInbound_IgnoreStatus() {
	on := &Adapter{config: &Config{IgnoreStatus: true}}
	off := &Adapter{config: &Config{IgnoreStatus: false}}
	status := &IncomingMessage{ChatJID: types.NewJID("status", types.BroadcastServer)}
	direct := &IncomingMessage{ChatJID: types.NewJID("5511999999999", types.DefaultUserServer)}

	assert.False(suite.T(), on.shouldForwardInbound(status), "ignore_status → story/status não é encaminhado")
	assert.True(suite.T(), on.shouldForwardInbound(direct), "1:1 flui com ignore_status")
	assert.True(suite.T(), off.shouldForwardInbound(status), "sem ignore_status, status flui (padrão)")
}

func (suite *AdapterTestSuite) TestAtoiOr() {
	assert.Equal(suite.T(), 5, atoiOr("5", 0))
	assert.Equal(suite.T(), 0, atoiOr("", 0))
	assert.Equal(suite.T(), 3, atoiOr("nope", 3))
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_Mentions() {
	msg := suite.fixtures.SampleIncomingMessage("msg-men", "5511999999999", "@gestor decide pf", true)
	msg.Mentions = []string{"5511777777777@s.whatsapp.net", "5512999999999@s.whatsapp.net"}

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(),
		"5511777777777@s.whatsapp.net,5512999999999@s.whatsapp.net",
		result.Metadata["mentions"])
}

func (suite *AdapterTestSuite) TestConvertToInboundMessage_NoMentions() {
	msg := suite.fixtures.SampleIncomingMessage("msg-nomen", "5511999999999", "bom dia", false)

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	_, has := result.Metadata["mentions"]
	assert.False(suite.T(), has, "message without mention must not set the mentions metadata")
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

func (suite *AdapterTestSuite) TestConvertToInboundMessage_LocationCoordinates() {
	msg := suite.fixtures.SampleIncomingMessage("msg-loc", "5511999999999", "São Paulo", false)
	msg.MessageType = "location"
	msg.Attachments = []Attachment{
		{
			Type:      "location",
			Latitude:  -23.5505,
			Longitude: -46.6333,
			Filename:  "São Paulo",
			Caption:   "São Paulo, Brazil",
		},
	}

	result := convertToInboundMessage(msg)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), plugin.ContentTypeLocation, result.ContentType)
	assert.Equal(suite.T(), "-23.5505", result.Metadata["latitude"])
	assert.Equal(suite.T(), "-46.6333", result.Metadata["longitude"])
	assert.Equal(suite.T(), "São Paulo", result.Metadata["name"])
	assert.Equal(suite.T(), "São Paulo, Brazil", result.Metadata["address"])
	// Coordinates are also mirrored onto the attachment metadata.
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "-23.5505", result.Attachments[0].Metadata["latitude"])
	assert.Equal(suite.T(), "-46.6333", result.Attachments[0].Metadata["longitude"])
}

// fakeMediaDownloader is a test seam for enrichInboundMedia that returns fixed
// bytes without a live whatsmeow connection.
type fakeMediaDownloader struct {
	data   []byte
	err    error
	called int
}

func (f *fakeMediaDownloader) DownloadMedia(_ context.Context, _ any) ([]byte, error) {
	f.called++
	return f.data, f.err
}

// enrichInboundMedia tests
func (suite *AdapterTestSuite) TestEnrichInboundMedia_DownloadsDecryptedBytes() {
	// An inbound media message with a downloadable reference.
	src := &IncomingMessage{
		MessageType: "image",
		Attachments: []Attachment{
			{Type: "image", URL: "https://mmg.whatsapp.net/encrypted-blob", MimeType: "image/jpeg", download: &waE2E.ImageMessage{}},
		},
	}
	inbound := &plugin.InboundMessage{
		Attachments: []*plugin.Attachment{
			{Type: "image", URL: "https://mmg.whatsapp.net/encrypted-blob", MimeType: "image/jpeg"},
		},
	}
	dl := &fakeMediaDownloader{data: []byte("decrypted-image-bytes")}

	enrichInboundMedia(context.Background(), src, inbound, dl)

	assert.Equal(suite.T(), 1, dl.called)
	att := inbound.Attachments[0]
	// Encrypted URL dropped, decrypted bytes attached as base64.
	assert.Empty(suite.T(), att.URL)
	assert.Equal(suite.T(), "base64", att.Metadata["data_encoding"])
	decoded, err := base64.StdEncoding.DecodeString(att.Metadata["data"])
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []byte("decrypted-image-bytes"), decoded)
	assert.Equal(suite.T(), int64(len("decrypted-image-bytes")), att.SizeBytes)
}

func (suite *AdapterTestSuite) TestEnrichInboundMedia_RecordsDownloadError() {
	src := &IncomingMessage{
		Attachments: []Attachment{
			{Type: "audio", URL: "https://mmg.whatsapp.net/blob", download: &waE2E.AudioMessage{}},
		},
	}
	inbound := &plugin.InboundMessage{
		Attachments: []*plugin.Attachment{
			{Type: "audio", URL: "https://mmg.whatsapp.net/blob"},
		},
	}
	dl := &fakeMediaDownloader{err: errors.New("boom")}

	enrichInboundMedia(context.Background(), src, inbound, dl)

	att := inbound.Attachments[0]
	assert.Equal(suite.T(), "boom", att.Metadata["download_error"])
	// On failure the reference URL is preserved.
	assert.Equal(suite.T(), "https://mmg.whatsapp.net/blob", att.URL)
	assert.Empty(suite.T(), att.Metadata["data"])
}

func (suite *AdapterTestSuite) TestEnrichInboundMedia_NoDownloaderIsNoOp() {
	src := &IncomingMessage{
		Attachments: []Attachment{{Type: "image", download: &waE2E.ImageMessage{}}},
	}
	inbound := &plugin.InboundMessage{
		Attachments: []*plugin.Attachment{{Type: "image", URL: "https://x/y"}},
	}

	assert.NotPanics(suite.T(), func() {
		enrichInboundMedia(context.Background(), src, inbound, nil)
	})
	assert.Equal(suite.T(), "https://x/y", inbound.Attachments[0].URL)
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

// SendMessage media tests — these exercise the previously panicking path where a
// media attachment could not be resolved (and, before the fix, the shadowed err
// left resp nil, panicking on resp.MessageID). Now they must return a failed
// SendResult without panicking.
func (suite *AdapterTestSuite) TestSendMessage_ImageWithoutAttachment() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	msg := &plugin.OutboundMessage{
		RecipientID: "5511999999999",
		ContentType: plugin.ContentTypeImage,
		Metadata:    make(map[string]string),
	}

	result, err := adapter.SendMessage(context.Background(), msg)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.Success)
	assert.Equal(suite.T(), plugin.MessageStatusFailed, result.Status)
}

func (suite *AdapterTestSuite) TestSendMessage_DocumentWithoutAttachment() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	msg := &plugin.OutboundMessage{
		RecipientID: "5511999999999",
		ContentType: plugin.ContentTypeDocument,
		Metadata:    make(map[string]string),
	}

	result, err := adapter.SendMessage(context.Background(), msg)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.Success)
	assert.Equal(suite.T(), plugin.MessageStatusFailed, result.Status)
}

// getMediaData with an attachment that has neither data nor URL must error, not panic.
func (suite *AdapterTestSuite) TestGetMediaData_NoDataNoURL() {
	att := &plugin.Attachment{Type: "image", MimeType: "image/jpeg"}

	data, err := getMediaData(context.Background(), att)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), data)
}

// Double Disconnect must not panic (previously closed an already-closed stopCh).
func (suite *AdapterTestSuite) TestDisconnect_TwiceDoesNotPanic() {
	adapter := NewAdapter()
	adapter.Initialize(suite.fixtures.MinimalConfig())

	ctx := context.Background()

	assert.NotPanics(suite.T(), func() {
		_ = adapter.Disconnect(ctx)
		_ = adapter.Disconnect(ctx)
	})
}

// SSRF protection tests for fetchMediaFromURL.
func (suite *AdapterTestSuite) TestFetchMediaFromURL_BlocksLoopback() {
	_, err := fetchMediaFromURL(context.Background(), "http://127.0.0.1/secret")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "disallowed")
}

func (suite *AdapterTestSuite) TestFetchMediaFromURL_BlocksLocalhost() {
	_, err := fetchMediaFromURL(context.Background(), "http://localhost:8080/secret")
	assert.Error(suite.T(), err)
}

func (suite *AdapterTestSuite) TestFetchMediaFromURL_BlocksCloudMetadata() {
	_, err := fetchMediaFromURL(context.Background(), "http://169.254.169.254/latest/meta-data/")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "disallowed")
}

func (suite *AdapterTestSuite) TestFetchMediaFromURL_BlocksPrivateIP() {
	_, err := fetchMediaFromURL(context.Background(), "http://10.0.0.5/x")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "disallowed")
}

func (suite *AdapterTestSuite) TestFetchMediaFromURL_RejectsNonHTTPScheme() {
	_, err := fetchMediaFromURL(context.Background(), "file:///etc/passwd")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "scheme")
}

func (suite *AdapterTestSuite) TestIsBlockedIP() {
	assert.True(suite.T(), isBlockedIP(net.ParseIP("127.0.0.1")))
	assert.True(suite.T(), isBlockedIP(net.ParseIP("10.0.0.1")))
	assert.True(suite.T(), isBlockedIP(net.ParseIP("192.168.1.1")))
	assert.True(suite.T(), isBlockedIP(net.ParseIP("169.254.169.254")))
	assert.True(suite.T(), isBlockedIP(net.ParseIP("::1")))
	assert.False(suite.T(), isBlockedIP(net.ParseIP("8.8.8.8")))
	assert.False(suite.T(), isBlockedIP(net.ParseIP("1.1.1.1")))
}
