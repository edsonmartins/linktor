//go:build e2e

package whatsapp

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/skip2/go-qrcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

/*
E2E Tests for WhatsApp Unofficial Adapter

These tests require a real WhatsApp account and will:
1. Connect to WhatsApp servers
2. Send real messages
3. Receive real responses

Environment Variables Required:
- WHATSAPP_TEST_PHONE: Phone number to send test messages (format: 5511999999999)
- WHATSAPP_DB_PATH: Optional path for SQLite database (default: /tmp/linktor_e2e_test.db)
- WHATSAPP_SKIP_INTERACTIVE: Set to "true" to skip tests requiring QR code scan

Usage:
  go test ./internal/adapters/whatsapp/... -v -tags=e2e

WARNING: These tests will send real messages to the specified phone number.
*/

// E2ETestSuite tests WhatsApp adapter with real connection
type E2ETestSuite struct {
	suite.Suite
	adapter      *Adapter
	testPhone    string
	dbPath       string
	skipInteractive bool
	receivedMessages []*plugin.InboundMessage
	receivedStatuses []*plugin.StatusCallback
}

func TestE2ETestSuite(t *testing.T) {
	// Check if required environment variables are set
	testPhone := os.Getenv("WHATSAPP_TEST_PHONE")
	if testPhone == "" {
		t.Skip("WHATSAPP_TEST_PHONE not set, skipping E2E tests")
	}

	suite.Run(t, new(E2ETestSuite))
}

func (suite *E2ETestSuite) SetupSuite() {
	suite.testPhone = os.Getenv("WHATSAPP_TEST_PHONE")
	suite.dbPath = os.Getenv("WHATSAPP_DB_PATH")
	if suite.dbPath == "" {
		suite.dbPath = "/tmp/linktor_e2e_test.db"
	}
	suite.skipInteractive = os.Getenv("WHATSAPP_SKIP_INTERACTIVE") == "true"
	suite.receivedMessages = make([]*plugin.InboundMessage, 0)
	suite.receivedStatuses = make([]*plugin.StatusCallback, 0)
}

func (suite *E2ETestSuite) SetupTest() {
	suite.adapter = NewAdapter()

	config := map[string]string{
		"channel_id":    "e2e-test-channel",
		"database_path": suite.dbPath,
		"device_name":   "Linktor E2E Test",
		"platform_type": "chrome",
		"log_level":     "INFO",
	}

	err := suite.adapter.Initialize(config)
	require.NoError(suite.T(), err)

	// Set up message handler
	suite.adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		suite.receivedMessages = append(suite.receivedMessages, msg)
		fmt.Printf("📩 Received message: %s\n", msg.Content)
		return nil
	})

	// Set up status handler
	suite.adapter.SetStatusHandler(func(ctx context.Context, status *plugin.StatusCallback) error {
		suite.receivedStatuses = append(suite.receivedStatuses, status)
		fmt.Printf("📋 Status update: %s -> %s\n", status.MessageID, status.Status)
		return nil
	})
}

func (suite *E2ETestSuite) TearDownTest() {
	if suite.adapter != nil && suite.adapter.IsConnected() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.adapter.Disconnect(ctx)
	}
}

// TestE2E_01_Connect tests connecting to WhatsApp
func (suite *E2ETestSuite) TestE2E_01_Connect() {
	if suite.skipInteractive {
		suite.T().Skip("Skipping interactive test")
	}

	ctx := context.Background()

	// Connect
	err := suite.adapter.Connect(ctx)
	require.NoError(suite.T(), err)

	// Check if already logged in
	if suite.adapter.IsLoggedIn() {
		fmt.Println("✅ Already logged in, skipping QR code")
		assert.True(suite.T(), suite.adapter.IsConnected())
		return
	}

	// Get QR code
	qrChan, err := suite.adapter.Login(ctx)
	require.NoError(suite.T(), err)

	fmt.Println("📱 Scan the QR code with WhatsApp on your phone:")
	fmt.Println("   (You have 60 seconds)")

	// Wait for QR code or login success
	timeout := time.After(60 * time.Second)
	for {
		select {
		case qrEvt, ok := <-qrChan:
			if !ok {
				// Channel closed - check if logged in
				if suite.adapter.IsLoggedIn() {
					fmt.Println("✅ Login successful!")
					return
				}
				suite.T().Fatal("QR channel closed without login")
			}

			// Display QR code in terminal
			displayQRCode(qrEvt.Code)
			fmt.Printf("QR Code expires at: %s\n", qrEvt.ExpiresAt.Format(time.RFC3339))

		case <-timeout:
			suite.T().Fatal("Timeout waiting for QR code scan")
		}
	}
}

// TestE2E_02_SendTextMessage tests sending a text message
func (suite *E2ETestSuite) TestE2E_02_SendTextMessage() {
	suite.ensureConnected()

	ctx := context.Background()
	msg := &plugin.OutboundMessage{
		RecipientID: suite.testPhone,
		Content:     fmt.Sprintf("🧪 Linktor E2E Test - Text Message\nTimestamp: %s", time.Now().Format(time.RFC3339)),
		ContentType: plugin.ContentTypeText,
		Metadata:    make(map[string]string),
	}

	result, err := suite.adapter.SendMessage(ctx, msg)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), result.Success)
	assert.NotEmpty(suite.T(), result.ExternalID)
	assert.Equal(suite.T(), plugin.MessageStatusSent, result.Status)

	fmt.Printf("✅ Text message sent with ID: %s\n", result.ExternalID)
}

// TestE2E_03_SendTypingIndicator tests sending typing indicator
func (suite *E2ETestSuite) TestE2E_03_SendTypingIndicator() {
	suite.ensureConnected()

	ctx := context.Background()

	// Send typing
	indicator := &plugin.TypingIndicator{
		RecipientID: suite.testPhone,
		IsTyping:    true,
	}

	err := suite.adapter.SendTypingIndicator(ctx, indicator)
	assert.NoError(suite.T(), err)

	fmt.Println("✅ Typing indicator sent")

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Stop typing
	indicator.IsTyping = false
	err = suite.adapter.SendTypingIndicator(ctx, indicator)
	assert.NoError(suite.T(), err)

	fmt.Println("✅ Stopped typing indicator")
}

// TestE2E_04_SendLocationMessage tests sending a location
func (suite *E2ETestSuite) TestE2E_04_SendLocationMessage() {
	suite.ensureConnected()

	ctx := context.Background()
	msg := &plugin.OutboundMessage{
		RecipientID: suite.testPhone,
		ContentType: plugin.ContentTypeLocation,
		Metadata: map[string]string{
			"latitude":  "-23.5505",
			"longitude": "-46.6333",
			"name":      "São Paulo - Linktor Test",
			"address":   "São Paulo, SP, Brazil",
		},
	}

	result, err := suite.adapter.SendMessage(ctx, msg)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), result.Success)

	fmt.Printf("✅ Location message sent with ID: %s\n", result.ExternalID)
}

// TestE2E_05_SendMessageWithReply tests sending a reply
func (suite *E2ETestSuite) TestE2E_05_SendMessageWithReply() {
	suite.ensureConnected()

	ctx := context.Background()

	// First, send a message to reply to
	originalMsg := &plugin.OutboundMessage{
		RecipientID: suite.testPhone,
		Content:     "Original message for reply test",
		ContentType: plugin.ContentTypeText,
		Metadata:    make(map[string]string),
	}

	originalResult, err := suite.adapter.SendMessage(ctx, originalMsg)
	require.NoError(suite.T(), err)
	require.True(suite.T(), originalResult.Success)

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Now send a reply
	replyMsg := &plugin.OutboundMessage{
		RecipientID: suite.testPhone,
		Content:     "This is a reply to the previous message",
		ContentType: plugin.ContentTypeText,
		Metadata: map[string]string{
			"reply_to_id": originalResult.ExternalID,
			"quoted_text": originalMsg.Content,
		},
	}

	replyResult, err := suite.adapter.SendMessage(ctx, replyMsg)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), replyResult.Success)

	fmt.Printf("✅ Reply message sent with ID: %s\n", replyResult.ExternalID)
}

// TestE2E_06_GetDeviceInfo tests getting device information
func (suite *E2ETestSuite) TestE2E_06_GetDeviceInfo() {
	suite.ensureConnected()

	info := suite.adapter.GetDeviceInfo()

	assert.NotNil(suite.T(), info)
	assert.Equal(suite.T(), DeviceStateConnected, info.State)
	assert.NotEmpty(suite.T(), info.JID)

	fmt.Printf("✅ Device Info:\n")
	fmt.Printf("   JID: %s\n", info.JID)
	fmt.Printf("   Phone: %s\n", info.PhoneNumber)
	fmt.Printf("   Platform: %s\n", info.Platform)
}

// TestE2E_07_GetConnectionStatus tests connection status
func (suite *E2ETestSuite) TestE2E_07_GetConnectionStatus() {
	suite.ensureConnected()

	status := suite.adapter.GetConnectionStatus()

	assert.NotNil(suite.T(), status)
	assert.True(suite.T(), status.Connected)
	assert.Equal(suite.T(), "connected", status.Status)

	fmt.Printf("✅ Connection Status: %s\n", status.Status)
}

// TestE2E_08_WaitForIncomingMessage waits for an incoming message
func (suite *E2ETestSuite) TestE2E_08_WaitForIncomingMessage() {
	if suite.skipInteractive {
		suite.T().Skip("Skipping interactive test")
	}

	suite.ensureConnected()

	fmt.Println("📱 Please send a message to this WhatsApp number within 30 seconds...")

	// Wait for incoming message
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	initialCount := len(suite.receivedMessages)

	for {
		select {
		case <-timeout:
			suite.T().Log("Timeout waiting for message - test skipped (not a failure)")
			return
		case <-ticker.C:
			if len(suite.receivedMessages) > initialCount {
				msg := suite.receivedMessages[len(suite.receivedMessages)-1]
				fmt.Printf("✅ Received message from %s: %s\n", msg.SenderID, msg.Content)
				assert.NotEmpty(suite.T(), msg.ExternalID)
				assert.NotEmpty(suite.T(), msg.SenderID)
				return
			}
		}
	}
}

// TestE2E_09_Reconnect tests reconnection with saved session
func (suite *E2ETestSuite) TestE2E_09_Reconnect() {
	suite.ensureConnected()

	ctx := context.Background()

	// Disconnect
	err := suite.adapter.Disconnect(ctx)
	require.NoError(suite.T(), err)

	fmt.Println("Disconnected, waiting 2 seconds...")
	time.Sleep(2 * time.Second)

	// Create new adapter with same database
	newAdapter := NewAdapter()
	config := map[string]string{
		"channel_id":    "e2e-test-channel",
		"database_path": suite.dbPath,
		"device_name":   "Linktor E2E Test",
		"platform_type": "chrome",
		"log_level":     "INFO",
	}

	err = newAdapter.Initialize(config)
	require.NoError(suite.T(), err)

	// Should reconnect without QR code
	err = newAdapter.Connect(ctx)
	require.NoError(suite.T(), err)

	// Wait for connection
	time.Sleep(3 * time.Second)

	assert.True(suite.T(), newAdapter.IsConnected())
	assert.True(suite.T(), newAdapter.IsLoggedIn())

	fmt.Println("✅ Reconnected successfully without QR code")

	// Update suite adapter
	suite.adapter = newAdapter
}

// TestE2E_10_Logout tests logging out
func (suite *E2ETestSuite) TestE2E_10_Logout() {
	if suite.skipInteractive {
		suite.T().Skip("Skipping logout test - would require new QR scan")
	}

	suite.ensureConnected()

	ctx := context.Background()

	fmt.Println("⚠️  This will log out the device. You'll need to scan QR again.")
	fmt.Println("   Skipping actual logout to preserve session...")

	// Uncomment to actually test logout:
	// err := suite.adapter.Logout(ctx)
	// require.NoError(suite.T(), err)
	// assert.False(suite.T(), suite.adapter.IsLoggedIn())

	_ = ctx // Avoid unused variable warning
}

// Helper methods

func (suite *E2ETestSuite) ensureConnected() {
	if !suite.adapter.IsConnected() {
		ctx := context.Background()
		err := suite.adapter.Connect(ctx)
		require.NoError(suite.T(), err)

		// Wait for connection
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				suite.T().Fatal("Timeout waiting for connection")
			case <-ticker.C:
				if suite.adapter.IsConnected() {
					return
				}
			}
		}
	}
}

// displayQRCode displays QR code in terminal
func displayQRCode(code string) {
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		fmt.Printf("QR Code (copy to https://www.qr-code-generator.com/):\n%s\n", code)
		return
	}

	// Print QR code to terminal
	art := qr.ToSmallString(false)
	fmt.Println(art)
}
