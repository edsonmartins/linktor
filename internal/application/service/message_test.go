package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
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

func TestMessageService_ListPage_EmptyConversation(t *testing.T) {
	svc := setupMessageTest()

	page, err := svc.ListPageByConversationForTenant(context.Background(), "tenant1", "conv1", nil, 0)
	assert.NoError(t, err)
	assert.Empty(t, page.Messages)
	assert.False(t, page.HasMore)
}

// A página pede uma mensagem além do limite para saber se há mais — e essa
// sobra não pode vazar para quem chamou.
func TestMessageService_ListPage_TrimsTheProbeRow(t *testing.T) {
	svc := setupMessageTest()
	msgRepo := svc.messageRepo.(*testutil.MockMessageRepository)

	base := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("m-%d", i)
		msgRepo.Messages[id] = &entity.Message{
			ID:             id,
			ConversationID: "conv1",
			SenderType:     entity.SenderTypeContact,
			ContentType:    entity.ContentTypeText,
			Content:        "oi",
			CreatedAt:      base.Add(time.Duration(i) * time.Minute),
		}
	}

	page, err := svc.ListPageByConversationForTenant(context.Background(), "tenant1", "conv1", nil, 2)
	assert.NoError(t, err)
	assert.Len(t, page.Messages, 2)
	assert.True(t, page.HasMore)
	assert.Equal(t, "m-4", page.Messages[0].ID, "a página começa pela mais nova")

	ultima, err := svc.ListPageByConversationForTenant(context.Background(), "tenant1", "conv1",
		&repository.MessageCursor{CreatedAt: page.Messages[1].CreatedAt, ID: page.Messages[1].ID}, 10)
	assert.NoError(t, err)
	assert.Len(t, ultima.Messages, 3)
	assert.False(t, ultima.HasMore)
}

// Conversa de outro tenant não entrega página nenhuma.
func TestMessageService_ListPage_OtherTenant(t *testing.T) {
	svc := setupMessageTest()

	_, err := svc.ListPageByConversationForTenant(context.Background(), "tenant2", "conv1", nil, 0)
	assert.Error(t, err)
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
