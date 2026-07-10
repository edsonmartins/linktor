package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func imagePayload(caption string) *waE2E.Message {
	return &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
		Caption:  proto.String(caption),
		Mimetype: proto.String("image/jpeg"),
		URL:      proto.String("https://cdn.example/enc"),
		MediaKey: []byte("k"),
	}}
}

func TestUnwrapEnvelopes_Ephemeral(t *testing.T) {
	m := &waE2E.Message{EphemeralMessage: &waE2E.FutureProofMessage{
		Message: &waE2E.Message{Conversation: proto.String("hi")},
	}}
	got, edited := unwrapEnvelopes(m)
	assert.False(t, edited)
	assert.Equal(t, "hi", got.GetConversation())
}

func TestUnwrapEnvelopes_ViewOnceV2Image(t *testing.T) {
	m := &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: imagePayload("secret")}}
	got, edited := unwrapEnvelopes(m)
	assert.False(t, edited)
	assert.NotNil(t, got.GetImageMessage())
	assert.Equal(t, "secret", got.GetImageMessage().GetCaption())
}

func TestUnwrapEnvelopes_DeviceSent(t *testing.T) {
	m := &waE2E.Message{DeviceSentMessage: &waE2E.DeviceSentMessage{
		Message: &waE2E.Message{Conversation: proto.String("from other device")},
	}}
	got, _ := unwrapEnvelopes(m)
	assert.Equal(t, "from other device", got.GetConversation())
}

func TestUnwrapEnvelopes_Edited(t *testing.T) {
	m := &waE2E.Message{EditedMessage: &waE2E.FutureProofMessage{
		Message: &waE2E.Message{Conversation: proto.String("edited text")},
	}}
	got, edited := unwrapEnvelopes(m)
	assert.True(t, edited)
	assert.Equal(t, "edited text", got.GetConversation())
}

func TestUnwrapEnvelopes_ProtocolEdited(t *testing.T) {
	m := &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{
		EditedMessage: &waE2E.Message{Conversation: proto.String("v2")},
	}}
	got, edited := unwrapEnvelopes(m)
	assert.True(t, edited)
	assert.Equal(t, "v2", got.GetConversation())
}

func TestUnwrapEnvelopes_Nested(t *testing.T) {
	// ephemeral(viewonce(image)) unwraps fully to the image.
	m := &waE2E.Message{EphemeralMessage: &waE2E.FutureProofMessage{
		Message: &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: imagePayload("deep")}},
	}}
	got, _ := unwrapEnvelopes(m)
	assert.NotNil(t, got.GetImageMessage())
	assert.Equal(t, "deep", got.GetImageMessage().GetCaption())
}

func TestConvertMessage_UnwrapsViewOnceImage(t *testing.T) {
	evt := &events.Message{
		Info:    createMessageInfo("MID1", "5511988887777", false, false),
		Message: &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: imagePayload("cap")}},
	}
	msg := convertMessage(evt)
	assert.NotNil(t, msg)
	assert.Equal(t, "image", msg.MessageType)
	assert.Equal(t, "cap", msg.Text)
	assert.Len(t, msg.Attachments, 1)
	assert.Equal(t, "image", msg.Attachments[0].Type)
	assert.NotNil(t, msg.Attachments[0].download)
}

func TestConvertMessage_EditedFlag(t *testing.T) {
	evt := &events.Message{
		Info: createMessageInfo("MID2", "5511988887777", false, false),
		Message: &waE2E.Message{EditedMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{Conversation: proto.String("new body")},
		}},
	}
	msg := convertMessage(evt)
	assert.NotNil(t, msg)
	assert.True(t, msg.IsEdit)
	assert.Equal(t, "new body", msg.Text)
}

func TestConvertMessage_CTWAMatchedTextFallback(t *testing.T) {
	evt := &events.Message{
		Info: createMessageInfo("MID3", "5511988887777", false, false),
		Message: &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			MatchedText: proto.String("https://ad.example"),
		}},
	}
	msg := convertMessage(evt)
	assert.NotNil(t, msg)
	assert.Equal(t, "text", msg.MessageType)
	assert.Equal(t, "https://ad.example", msg.Text)
}
