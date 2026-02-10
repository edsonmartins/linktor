package whatsapp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// EventsTestSuite tests event conversion functions
type EventsTestSuite struct {
	suite.Suite
}

func TestEventsTestSuite(t *testing.T) {
	suite.Run(t, new(EventsTestSuite))
}

// Helper to create a basic message info
func createMessageInfo(id string, senderPhone string, isGroup bool, isFromMe bool) types.MessageInfo {
	senderJID := types.NewJID(senderPhone, types.DefaultUserServer)
	chatJID := senderJID
	if isGroup {
		chatJID = types.NewJID("123456789", types.GroupServer)
	}

	return types.MessageInfo{
		MessageSource: types.MessageSource{
			Sender:   senderJID,
			Chat:     chatJID,
			IsFromMe: isFromMe,
			IsGroup:  isGroup,
		},
		ID:        types.MessageID(id),
		PushName:  "Test User",
		Timestamp: time.Now(),
	}
}

// convertMessage() tests
func (suite *EventsTestSuite) TestConvertMessage_NilEvent() {
	result := convertMessage(nil)
	assert.Nil(suite.T(), result)
}

func (suite *EventsTestSuite) TestConvertMessage_NilMessage() {
	evt := &events.Message{
		Info:    createMessageInfo("msg-1", "5511999999999", false, false),
		Message: nil,
	}

	result := convertMessage(evt)
	assert.Nil(suite.T(), result)
}

func (suite *EventsTestSuite) TestConvertMessage_SimpleText() {
	evt := &events.Message{
		Info: createMessageInfo("msg-1", "5511999999999", false, false),
		Message: &waE2E.Message{
			Conversation: proto.String("Hello, World!"),
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "msg-1", result.ExternalID)
	assert.Equal(suite.T(), "Hello, World!", result.Text)
	assert.Equal(suite.T(), "text", result.MessageType)
	assert.False(suite.T(), result.IsFromMe)
	assert.False(suite.T(), result.IsGroup)
	assert.Equal(suite.T(), "Test User", result.SenderName)
}

func (suite *EventsTestSuite) TestConvertMessage_ExtendedText() {
	evt := &events.Message{
		Info: createMessageInfo("msg-2", "5511999999999", false, false),
		Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("Extended text message"),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "Extended text message", result.Text)
	assert.Equal(suite.T(), "text", result.MessageType)
}

func (suite *EventsTestSuite) TestConvertMessage_ExtendedTextWithReply() {
	quotedMsgID := "quoted-msg-123"
	quotedParticipant := "5511888888888@s.whatsapp.net"

	evt := &events.Message{
		Info: createMessageInfo("msg-3", "5511999999999", false, false),
		Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("This is a reply"),
				ContextInfo: &waE2E.ContextInfo{
					StanzaID:    proto.String(quotedMsgID),
					Participant: proto.String(quotedParticipant),
					QuotedMessage: &waE2E.Message{
						Conversation: proto.String("Original message"),
					},
				},
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "This is a reply", result.Text)
	assert.Equal(suite.T(), quotedMsgID, result.QuotedID)
	assert.NotNil(suite.T(), result.ReplyTo)
	assert.Equal(suite.T(), quotedMsgID, result.ReplyTo.MessageID)
	assert.Equal(suite.T(), "Original message", result.ReplyTo.Text)
}

func (suite *EventsTestSuite) TestConvertMessage_ExtendedTextWithMentions() {
	mentions := []string{"5511111111111@s.whatsapp.net", "5522222222222@s.whatsapp.net"}

	evt := &events.Message{
		Info: createMessageInfo("msg-4", "5511999999999", true, false),
		Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("Hey @user1 and @user2"),
				ContextInfo: &waE2E.ContextInfo{
					MentionedJID: mentions,
				},
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), mentions, result.Mentions)
	assert.True(suite.T(), result.IsGroup)
}

func (suite *EventsTestSuite) TestConvertMessage_ForwardedMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-5", "5511999999999", false, false),
		Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("Forwarded message"),
				ContextInfo: &waE2E.ContextInfo{
					IsForwarded: proto.Bool(true),
				},
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.IsForwarded)
}

func (suite *EventsTestSuite) TestConvertMessage_ImageMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-6", "5511999999999", false, false),
		Message: &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				URL:        proto.String("https://example.com/image.jpg"),
				Mimetype:   proto.String("image/jpeg"),
				Caption:    proto.String("Check this out!"),
				FileLength: proto.Uint64(12345),
				Width:      proto.Uint32(800),
				Height:     proto.Uint32(600),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "image", result.MessageType)
	assert.Equal(suite.T(), "Check this out!", result.Text)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "image", result.Attachments[0].Type)
	assert.Equal(suite.T(), "image/jpeg", result.Attachments[0].MimeType)
	assert.Equal(suite.T(), uint64(12345), result.Attachments[0].FileSize)
	assert.Equal(suite.T(), uint32(800), result.Attachments[0].Width)
	assert.Equal(suite.T(), uint32(600), result.Attachments[0].Height)
}

func (suite *EventsTestSuite) TestConvertMessage_VideoMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-7", "5511999999999", false, false),
		Message: &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				URL:        proto.String("https://example.com/video.mp4"),
				Mimetype:   proto.String("video/mp4"),
				Caption:    proto.String("Video caption"),
				FileLength: proto.Uint64(1000000),
				Width:      proto.Uint32(1920),
				Height:     proto.Uint32(1080),
				Seconds:    proto.Uint32(60),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "video", result.MessageType)
	assert.Equal(suite.T(), "Video caption", result.Text)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "video", result.Attachments[0].Type)
	assert.Equal(suite.T(), uint32(60), result.Attachments[0].Duration)
}

func (suite *EventsTestSuite) TestConvertMessage_AudioMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-8", "5511999999999", false, false),
		Message: &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:        proto.String("https://example.com/audio.ogg"),
				Mimetype:   proto.String("audio/ogg"),
				FileLength: proto.Uint64(50000),
				Seconds:    proto.Uint32(30),
				PTT:        proto.Bool(false),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "audio", result.MessageType)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "audio", result.Attachments[0].Type)
}

func (suite *EventsTestSuite) TestConvertMessage_PTTMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-9", "5511999999999", false, false),
		Message: &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:        proto.String("https://example.com/voice.ogg"),
				Mimetype:   proto.String("audio/ogg; codecs=opus"),
				FileLength: proto.Uint64(25000),
				Seconds:    proto.Uint32(15),
				PTT:        proto.Bool(true),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "ptt", result.MessageType)
}

func (suite *EventsTestSuite) TestConvertMessage_DocumentMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-10", "5511999999999", false, false),
		Message: &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				URL:        proto.String("https://example.com/doc.pdf"),
				Mimetype:   proto.String("application/pdf"),
				FileName:   proto.String("document.pdf"),
				Caption:    proto.String("Important document"),
				FileLength: proto.Uint64(500000),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "document", result.MessageType)
	assert.Equal(suite.T(), "Important document", result.Text)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "document", result.Attachments[0].Type)
	assert.Equal(suite.T(), "document.pdf", result.Attachments[0].Filename)
}

func (suite *EventsTestSuite) TestConvertMessage_StickerMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-11", "5511999999999", false, false),
		Message: &waE2E.Message{
			StickerMessage: &waE2E.StickerMessage{
				URL:        proto.String("https://example.com/sticker.webp"),
				Mimetype:   proto.String("image/webp"),
				FileLength: proto.Uint64(10000),
				Width:      proto.Uint32(512),
				Height:     proto.Uint32(512),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "sticker", result.MessageType)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "sticker", result.Attachments[0].Type)
}

func (suite *EventsTestSuite) TestConvertMessage_LocationMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-12", "5511999999999", false, false),
		Message: &waE2E.Message{
			LocationMessage: &waE2E.LocationMessage{
				DegreesLatitude:  proto.Float64(-23.5505),
				DegreesLongitude: proto.Float64(-46.6333),
				Name:             proto.String("São Paulo"),
				Address:          proto.String("São Paulo, Brazil"),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "location", result.MessageType)
	assert.Equal(suite.T(), "São Paulo", result.Text)
	assert.Len(suite.T(), result.Attachments, 1)
	assert.Equal(suite.T(), "location", result.Attachments[0].Type)
}

func (suite *EventsTestSuite) TestConvertMessage_ContactMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-13", "5511999999999", false, false),
		Message: &waE2E.Message{
			ContactMessage: &waE2E.ContactMessage{
				DisplayName: proto.String("John Doe"),
				Vcard:       proto.String("BEGIN:VCARD\nVERSION:3.0\nFN:John Doe\nEND:VCARD"),
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "contact", result.MessageType)
	assert.Equal(suite.T(), "John Doe", result.Text)
}

func (suite *EventsTestSuite) TestConvertMessage_ReactionMessage() {
	targetMsgID := "target-msg-123"

	evt := &events.Message{
		Info: createMessageInfo("msg-14", "5511999999999", false, false),
		Message: &waE2E.Message{
			ReactionMessage: &waE2E.ReactionMessage{
				Text: proto.String("👍"),
				Key: &waCommon.MessageKey{
					ID: proto.String(targetMsgID),
				},
			},
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "reaction", result.MessageType)
	assert.NotNil(suite.T(), result.Reaction)
	assert.Equal(suite.T(), "👍", result.Reaction.Emoji)
	assert.Equal(suite.T(), targetMsgID, result.Reaction.MessageID)
}

func (suite *EventsTestSuite) TestConvertMessage_GroupMessage() {
	evt := &events.Message{
		Info: createMessageInfo("msg-15", "5511999999999", true, false),
		Message: &waE2E.Message{
			Conversation: proto.String("Group message"),
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.IsGroup)
}

func (suite *EventsTestSuite) TestConvertMessage_FromMe() {
	evt := &events.Message{
		Info: createMessageInfo("msg-16", "5511999999999", false, true),
		Message: &waE2E.Message{
			Conversation: proto.String("My own message"),
		},
	}

	result := convertMessage(evt)

	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.IsFromMe)
}

// convertReceipt() tests
func (suite *EventsTestSuite) TestConvertReceipt_Nil() {
	result := convertReceipt(nil)
	assert.Nil(suite.T(), result)
}

func (suite *EventsTestSuite) TestConvertReceipt_Delivered() {
	evt := &events.Receipt{
		MessageIDs: []string{"msg-1", "msg-2"},
		MessageSource: types.MessageSource{
			Sender: types.NewJID("5511999999999", types.DefaultUserServer),
			Chat:   types.NewJID("5511999999999", types.DefaultUserServer),
		},
		Type:      events.ReceiptTypeDelivered,
		Timestamp: time.Now(),
	}

	result := convertReceipt(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), []string{"msg-1", "msg-2"}, result.MessageIDs)
	assert.Equal(suite.T(), ReceiptTypeDelivered, result.Type)
}

func (suite *EventsTestSuite) TestConvertReceipt_Read() {
	evt := &events.Receipt{
		MessageIDs: []string{"msg-1"},
		MessageSource: types.MessageSource{
			Sender: types.NewJID("5511999999999", types.DefaultUserServer),
			Chat:   types.NewJID("5511999999999", types.DefaultUserServer),
		},
		Type:      events.ReceiptTypeRead,
		Timestamp: time.Now(),
	}

	result := convertReceipt(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), ReceiptTypeRead, result.Type)
}

func (suite *EventsTestSuite) TestConvertReceipt_Played() {
	evt := &events.Receipt{
		MessageIDs: []string{"msg-1"},
		MessageSource: types.MessageSource{
			Sender: types.NewJID("5511999999999", types.DefaultUserServer),
			Chat:   types.NewJID("5511999999999", types.DefaultUserServer),
		},
		Type:      events.ReceiptTypePlayed,
		Timestamp: time.Now(),
	}

	result := convertReceipt(evt)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), ReceiptTypePlayed, result.Type)
}

func (suite *EventsTestSuite) TestConvertReceipt_UnknownType() {
	evt := &events.Receipt{
		MessageIDs: []string{"msg-1"},
		MessageSource: types.MessageSource{
			Sender: types.NewJID("5511999999999", types.DefaultUserServer),
			Chat:   types.NewJID("5511999999999", types.DefaultUserServer),
		},
		Type:      "unknown",
		Timestamp: time.Now(),
	}

	result := convertReceipt(evt)

	assert.NotNil(suite.T(), result)
	// Unknown types default to delivered
	assert.Equal(suite.T(), ReceiptTypeDelivered, result.Type)
}
