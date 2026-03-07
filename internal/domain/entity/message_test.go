package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageStatus_Constants(t *testing.T) {
	assert.Equal(t, MessageStatus("pending"), MessageStatusPending)
	assert.Equal(t, MessageStatus("sent"), MessageStatusSent)
	assert.Equal(t, MessageStatus("delivered"), MessageStatusDelivered)
	assert.Equal(t, MessageStatus("read"), MessageStatusRead)
	assert.Equal(t, MessageStatus("failed"), MessageStatusFailed)
}

func TestContentType_Constants(t *testing.T) {
	assert.Equal(t, ContentType("text"), ContentTypeText)
	assert.Equal(t, ContentType("image"), ContentTypeImage)
	assert.Equal(t, ContentType("video"), ContentTypeVideo)
	assert.Equal(t, ContentType("audio"), ContentTypeAudio)
	assert.Equal(t, ContentType("document"), ContentTypeDocument)
}

func TestSenderType_Constants(t *testing.T) {
	assert.Equal(t, SenderType("contact"), SenderTypeContact)
	assert.Equal(t, SenderType("user"), SenderTypeUser)
	assert.Equal(t, SenderType("system"), SenderTypeSystem)
	assert.Equal(t, SenderType("bot"), SenderTypeBot)
}
