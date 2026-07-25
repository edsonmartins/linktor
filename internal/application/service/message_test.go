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

// A entrega da reação precisa do endereço do contato NO CANAL, que mora nas identidades — o
// contato sozinho pode ter o telefone vazio. Sem carregá-las o worker recusava com
// "recipient is required" e a reação nunca chegava ao cliente.
func TestMessageService_SendReaction_ResolvesRecipientFromIdentities(t *testing.T) {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	producer := testutil.NewMockProducer()

	// Contato SEM telefone no registro: o endereço só existe na identidade do canal.
	contactRepo.Contacts["contact1"] = &entity.Contact{ID: "contact1", TenantID: "tenant1"}
	contactRepo.Identities["contact1"] = []*entity.ContactIdentity{
		{ContactID: "contact1", ChannelType: "whatsapp", Identifier: "5511999999999"},
	}
	channelRepo.Channels["channel1"] = &entity.Channel{
		ID: "channel1", TenantID: "tenant1", Type: entity.ChannelTypeWhatsApp,
	}
	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "tenant1", ContactID: "contact1", ChannelID: "channel1",
		Status: entity.ConversationStatusOpen,
	}
	msgRepo.Messages["msg1"] = &entity.Message{
		ID: "msg1", ConversationID: "conv1", ExternalID: "WA-EXTERNAL-1",
	}

	svc := NewMessageService(msgRepo, convRepo, channelRepo, contactRepo, producer)

	err := svc.SendReaction(context.Background(), "conv1", "msg1", "👍", "user1")
	assert.NoError(t, err)

	if assert.Len(t, producer.OutboundMessages, 1) {
		out := producer.OutboundMessages[0]
		assert.Equal(t, "5511999999999", out.RecipientID, "sem destinatário o worker recusa a entrega")
		assert.Equal(t, "reaction", out.ContentType)
		assert.Equal(t, "👍", out.Content)
		assert.Equal(t, "WA-EXTERNAL-1", out.Metadata["reaction_target_external_id"])
	}
}

// Mensagem que nunca saiu ao canal não tem o que reagir do lado do provedor: nada é enfileirado,
// mas a reação segue persistida localmente.
func TestMessageService_SendReaction_SkipsDeliveryWithoutExternalID(t *testing.T) {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	producer := testutil.NewMockProducer()

	contactRepo.Contacts["contact1"] = &entity.Contact{ID: "contact1", TenantID: "tenant1"}
	channelRepo.Channels["channel1"] = &entity.Channel{
		ID: "channel1", TenantID: "tenant1", Type: entity.ChannelTypeWhatsApp,
	}
	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "tenant1", ContactID: "contact1", ChannelID: "channel1",
	}
	msgRepo.Messages["msg1"] = &entity.Message{ID: "msg1", ConversationID: "conv1"}

	svc := NewMessageService(msgRepo, convRepo, channelRepo, contactRepo, producer)

	assert.NoError(t, svc.SendReaction(context.Background(), "conv1", "msg1", "👍", "user1"))
	assert.Empty(t, producer.OutboundMessages)
}
