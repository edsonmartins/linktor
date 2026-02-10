package whatsapp

import (
	"context"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"go.mau.fi/whatsmeow/types"
)

// MockMessageHandler captures message handler calls for testing
type MockMessageHandler struct {
	Calls    []*plugin.InboundMessage
	ReturnErr error
}

func (m *MockMessageHandler) Handle(ctx context.Context, msg *plugin.InboundMessage) error {
	m.Calls = append(m.Calls, msg)
	return m.ReturnErr
}

func (m *MockMessageHandler) Handler() plugin.MessageHandler {
	return func(ctx context.Context, msg *plugin.InboundMessage) error {
		return m.Handle(ctx, msg)
	}
}

// MockStatusHandler captures status handler calls for testing
type MockStatusHandler struct {
	Calls     []*plugin.StatusCallback
	ReturnErr error
}

func (m *MockStatusHandler) Handle(ctx context.Context, status *plugin.StatusCallback) error {
	m.Calls = append(m.Calls, status)
	return m.ReturnErr
}

func (m *MockStatusHandler) Handler() plugin.StatusHandler {
	return func(ctx context.Context, status *plugin.StatusCallback) error {
		return m.Handle(ctx, status)
	}
}

// TestFixtures provides common test data
type TestFixtures struct{}

// NewTestFixtures creates a new TestFixtures instance
func NewTestFixtures() *TestFixtures {
	return &TestFixtures{}
}

// ValidConfig returns a valid adapter configuration
func (f *TestFixtures) ValidConfig() map[string]string {
	return map[string]string{
		"channel_id":    "test-channel-123",
		"database_path": "/tmp/test_whatsapp.db",
		"device_name":   "TestDevice",
		"platform_type": "chrome",
		"log_level":     "WARN",
	}
}

// MinimalConfig returns minimal required configuration
func (f *TestFixtures) MinimalConfig() map[string]string {
	return map[string]string{
		"channel_id": "test-channel-minimal",
	}
}

// InvalidConfig returns an invalid configuration (missing channel_id)
func (f *TestFixtures) InvalidConfig() map[string]string {
	return map[string]string{
		"device_name": "TestDevice",
	}
}

// SampleJID returns a sample WhatsApp JID
func (f *TestFixtures) SampleJID(phone string) types.JID {
	return types.NewJID(phone, types.DefaultUserServer)
}

// SampleGroupJID returns a sample group JID
func (f *TestFixtures) SampleGroupJID(id string) types.JID {
	return types.NewJID(id, types.GroupServer)
}

// SampleTextOutbound returns a sample outbound text message
func (f *TestFixtures) SampleTextOutbound(to, content string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     content,
		ContentType: plugin.ContentTypeText,
		Metadata:    make(map[string]string),
	}
}

// SampleTextOutboundWithReply returns a sample outbound text message with reply
func (f *TestFixtures) SampleTextOutboundWithReply(to, content, replyToID, quotedText string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     content,
		ContentType: plugin.ContentTypeText,
		Metadata: map[string]string{
			"reply_to_id": replyToID,
			"quoted_text": quotedText,
		},
	}
}

// SampleImageOutbound returns a sample outbound image message
func (f *TestFixtures) SampleImageOutbound(to, caption string, imageData []byte) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		Content:     caption,
		ContentType: plugin.ContentTypeImage,
		Attachments: []*plugin.Attachment{
			{
				Type:     "image",
				MimeType: "image/jpeg",
				Metadata: map[string]string{
					"data": string(imageData),
				},
			},
		},
		Metadata: make(map[string]string),
	}
}

// SampleLocationOutbound returns a sample outbound location message
func (f *TestFixtures) SampleLocationOutbound(to string, lat, lon float64, name, address string) *plugin.OutboundMessage {
	return &plugin.OutboundMessage{
		RecipientID: to,
		ContentType: plugin.ContentTypeLocation,
		Metadata: map[string]string{
			"latitude":  formatFloat(lat),
			"longitude": formatFloat(lon),
			"name":      name,
			"address":   address,
		},
	}
}

// SampleTypingIndicator returns a sample typing indicator
func (f *TestFixtures) SampleTypingIndicator(to string, isTyping bool) *plugin.TypingIndicator {
	return &plugin.TypingIndicator{
		RecipientID: to,
		IsTyping:    isTyping,
	}
}

// SampleReadReceipt returns a sample read receipt
func (f *TestFixtures) SampleReadReceipt(to, messageID string) *plugin.ReadReceipt {
	return &plugin.ReadReceipt{
		RecipientID: to,
		MessageID:   messageID,
	}
}

// SampleIncomingMessage returns a sample incoming message struct
func (f *TestFixtures) SampleIncomingMessage(id, senderPhone, text string, isGroup bool) *IncomingMessage {
	senderJID := f.SampleJID(senderPhone)
	chatJID := senderJID
	if isGroup {
		chatJID = f.SampleGroupJID("123456789")
	}

	return &IncomingMessage{
		ExternalID:  id,
		SenderJID:   senderJID,
		ChatJID:     chatJID,
		SenderName:  "Test Sender",
		Text:        text,
		Timestamp:   time.Now(),
		IsFromMe:    false,
		IsGroup:     isGroup,
		MessageType: "text",
	}
}

// SampleReceipt returns a sample receipt
func (f *TestFixtures) SampleReceipt(messageIDs []string, receiptType ReceiptType) *Receipt {
	return &Receipt{
		MessageIDs: messageIDs,
		SenderJID:  f.SampleJID("5511999999999"),
		ChatJID:    f.SampleJID("5511999999999"),
		Type:       receiptType,
		Timestamp:  time.Now(),
	}
}

// Helper function to format float as string
func formatFloat(f float64) string {
	return string(rune(int(f*1000000))) // Simple conversion for testing
}

// AssertEventually waits for a condition to be true
func AssertEventually(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
