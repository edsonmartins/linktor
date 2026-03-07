package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func setupMessageTest() *MessageService {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()

	// Add fixtures
	contactRepo.Contacts["contact1"] = &entity.Contact{
		ID: "contact1", TenantID: "tenant1", Phone: "5511999999999",
		Identities: []*entity.ContactIdentity{{ChannelType: "whatsapp", Identifier: "5511999999999"}},
	}
	channelRepo.Channels["channel1"] = &entity.Channel{ID: "channel1", TenantID: "tenant1", Type: entity.ChannelTypeWhatsApp}
	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "tenant1", ContactID: "contact1", ChannelID: "channel1",
		Status: entity.ConversationStatusOpen,
	}

	return NewMessageService(msgRepo, convRepo, channelRepo, contactRepo, nil) // nil producer for unit tests
}

func TestMessageService_ListByConversation(t *testing.T) {
	svc := setupMessageTest()

	messages, count, err := svc.ListByConversation(context.Background(), "conv1", nil)
	assert.NoError(t, err)
	assert.Empty(t, messages)
	assert.Equal(t, int64(0), count)
}

func TestMessageService_Send(t *testing.T) {
	svc := setupMessageTest()

	msg, err := svc.Send(context.Background(), &SendMessageInput{
		ConversationID: "conv1",
		SenderType:     "user",
		SenderID:       "user1",
		ContentType:    "text",
		Content:        "Hello!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, "Hello!", msg.Content)
	assert.Equal(t, entity.MessageStatusPending, msg.Status)
}

func TestMessageService_Send_MissingConversation(t *testing.T) {
	svc := setupMessageTest()

	_, err := svc.Send(context.Background(), &SendMessageInput{
		ConversationID: "",
		Content:        "Hello!",
	})

	assert.Error(t, err)
}

func TestMessageService_Send_ConversationNotFound(t *testing.T) {
	svc := setupMessageTest()

	_, err := svc.Send(context.Background(), &SendMessageInput{
		ConversationID: "nonexistent",
		SenderType:     "user",
		Content:        "Hello!",
	})

	assert.Error(t, err)
}
