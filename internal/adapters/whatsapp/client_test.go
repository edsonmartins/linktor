package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ClientTestSuite tests the WhatsApp client
type ClientTestSuite struct {
	suite.Suite
	fixtures *TestFixtures
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	suite.fixtures = NewTestFixtures()
}

// Config validation tests
func (suite *ClientTestSuite) TestNewClient_WithValidConfig() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_client.db",
		DeviceName:   "TestDevice",
	}

	client, err := NewClient(config)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), client)
	assert.Equal(suite.T(), DeviceStateDisconnected, client.GetState())
}

func (suite *ClientTestSuite) TestNewClient_WithInvalidConfig() {
	config := &Config{
		ChannelID: "", // Empty - invalid
	}

	client, err := NewClient(config)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), client)
	assert.Contains(suite.T(), err.Error(), "channel_id is required")
}

func (suite *ClientTestSuite) TestNewClient_SetsDefaults() {
	config := &Config{
		ChannelID: "test-channel",
	}

	client, err := NewClient(config)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), client)
	// Verify defaults were set
	assert.Equal(suite.T(), "storages/whatsapp_test-channel.db", client.config.DatabasePath)
	assert.Equal(suite.T(), "Linktor", client.config.DeviceName)
	assert.True(suite.T(), client.config.AutoReconnect)
}

// State tests
func (suite *ClientTestSuite) TestGetState_ReturnsCurrentState() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_state.db",
	}

	client, _ := NewClient(config)

	assert.Equal(suite.T(), DeviceStateDisconnected, client.GetState())
}

func (suite *ClientTestSuite) TestIsLoggedIn_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_login.db",
	}

	client, _ := NewClient(config)

	assert.False(suite.T(), client.IsLoggedIn())
}

func (suite *ClientTestSuite) TestIsConnected_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_conn.db",
	}

	client, _ := NewClient(config)

	assert.False(suite.T(), client.IsConnected())
}

// GetDeviceInfo tests
func (suite *ClientTestSuite) TestGetDeviceInfo_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_info.db",
		DeviceName:   "MyDevice",
	}

	client, _ := NewClient(config)
	info := client.GetDeviceInfo()

	assert.NotNil(suite.T(), info)
	assert.Equal(suite.T(), DeviceStateDisconnected, info.State)
	assert.Empty(suite.T(), info.ID)
	assert.Empty(suite.T(), info.JID)
}

// GetEventChannel tests
func (suite *ClientTestSuite) TestGetEventChannel_ReturnsChannel() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_events.db",
	}

	client, _ := NewClient(config)
	eventCh := client.GetEventChannel()

	assert.NotNil(suite.T(), eventCh)
}

// Close tests
func (suite *ClientTestSuite) TestClose_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_close.db",
	}

	client, _ := NewClient(config)
	err := client.Close()

	// Should not error when closing an unconnected client
	assert.NoError(suite.T(), err)
}

// GetRawClient tests
func (suite *ClientTestSuite) TestGetRawClient_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_raw.db",
	}

	client, _ := NewClient(config)
	rawClient := client.GetRawClient()

	// Raw client should be nil when not connected
	assert.Nil(suite.T(), rawClient)
}

// Disconnect tests
func (suite *ClientTestSuite) TestDisconnect_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_disc.db",
	}

	client, _ := NewClient(config)

	// Should not panic
	client.Disconnect()

	assert.Equal(suite.T(), DeviceStateDisconnected, client.GetState())
}

// Connection error tests
func (suite *ClientTestSuite) TestConnect_RequiresValidClient() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_connect.db",
	}

	client, err := NewClient(config)
	assert.NoError(suite.T(), err)

	// Note: Full connection testing requires E2E tests
	// Here we just verify the client was created
	assert.NotNil(suite.T(), client)
}

// Login tests (partial - full tests in E2E)
func (suite *ClientTestSuite) TestLogin_WhenClientNotReady() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_login_ready.db",
	}

	client, _ := NewClient(config)
	// Don't call Connect() so client.client is nil

	_, err := client.Login(nil)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestLoginWithPairCode_WhenClientNotReady() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_pair_ready.db",
	}

	client, _ := NewClient(config)
	// Don't call Connect() so client.client is nil

	_, err := client.LoginWithPairCode(nil, "5511999999999")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestLogout_WhenClientNotReady() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_logout_ready.db",
	}

	client, _ := NewClient(config)

	err := client.Logout(nil)

	// Should not error - gracefully handles nil client
	assert.NoError(suite.T(), err)
}

// Message sending tests (when not connected)
func (suite *ClientTestSuite) TestSendTextMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_text.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendTextMessage(nil, "5511999999999", "Hello")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendImageMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_img.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendImageMessage(nil, "5511999999999", []byte("fake"), "image/jpeg", "caption")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendVideoMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_vid.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendVideoMessage(nil, "5511999999999", []byte("fake"), "video/mp4", "caption")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendAudioMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_aud.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendAudioMessage(nil, "5511999999999", []byte("fake"), "audio/ogg", false)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendDocumentMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_doc.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendDocumentMessage(nil, "5511999999999", []byte("fake"), "application/pdf", "doc.pdf", "caption")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendStickerMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_stk.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendStickerMessage(nil, "5511999999999", []byte("fake"))

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendLocationMessage_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_loc.db",
	}

	client, _ := NewClient(config)

	_, err := client.SendLocationMessage(nil, "5511999999999", -23.55, -46.63, "São Paulo", "Brazil")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendReaction_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_send_react.db",
	}

	client, _ := NewClient(config)

	err := client.SendReaction(nil, "5511999999999", "msg-123", "👍")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestMarkAsRead_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_mark_read.db",
	}

	client, _ := NewClient(config)
	jid := suite.fixtures.SampleJID("5511999999999")

	err := client.MarkAsRead(nil, []string{"msg-1"}, jid, jid)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendPresence_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_presence.db",
	}

	client, _ := NewClient(config)

	err := client.SendPresence(nil, true)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestSendChatPresence_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_chat_presence.db",
	}

	client, _ := NewClient(config)
	jid := suite.fixtures.SampleJID("5511999999999")

	err := client.SendChatPresence(nil, jid, ChatPresenceComposing)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestDownloadMedia_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_download.db",
	}

	client, _ := NewClient(config)

	_, err := client.DownloadMedia(nil, nil)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestGetContactInfo_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_contact.db",
	}

	client, _ := NewClient(config)
	jid := suite.fixtures.SampleJID("5511999999999")

	_, err := client.GetContactInfo(nil, jid)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestGetProfilePicture_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_profile.db",
	}

	client, _ := NewClient(config)
	jid := suite.fixtures.SampleJID("5511999999999")

	_, err := client.GetProfilePicture(nil, jid)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}

func (suite *ClientTestSuite) TestGetGroupInfo_WhenNotConnected() {
	config := &Config{
		ChannelID:    "test-channel",
		DatabasePath: "/tmp/test_group.db",
	}

	client, _ := NewClient(config)
	jid := suite.fixtures.SampleGroupJID("123456789")

	_, err := client.GetGroupInfo(nil, jid)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrClientNotReady, err)
}
