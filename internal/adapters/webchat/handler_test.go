package webchat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

// TestHandleClientMessage_SameSessionKeepsOneConversation reproduz o fluxo real: o handler do
// widget publica o inbound e o ReceiveMessageUseCase o persiste. Cada mensagem do widget tem um
// id novo, e o use case identificava o contato por esse id, então cada mensagem abria um
// contato e uma conversa novos e a resposta do atendente não achava a sessão do visitante.
func TestHandleClientMessage_SameSessionKeepsOneConversation(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	channelRepo := testutil.NewMockChannelRepository()
	conversationRepo := testutil.NewMockConversationRepository()
	contactRepo := testutil.NewMockContactRepository()
	messageRepo := testutil.NewMockMessageRepository()
	producer := testutil.NewMockProducer()

	channel := &entity.Channel{
		ID:               "ch-webchat",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeWebChat,
		Name:             "Website Chat",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config:           map[string]string{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	channelRepo.Channels[channel.ID] = channel

	h := NewHandler(nil, channelRepo, conversationRepo, contactRepo, producer, nil)
	uc := usecase.NewReceiveMessageUseCase(messageRepo, conversationRepo, channelRepo, contactRepo,
		producer, service.NewMessageNormalizer())

	client := NewClient(nil, nil, "session-abc")
	client.Metadata["name"] = "Maria Souza"
	client.Metadata["email"] = "maria.souza@exemplo.com"

	for i, msgID := range []string{"msg-1", "msg-2", "msg-3"} {
		require.NoError(t, h.handleClientMessage(ctx, client, channel, &MessagePayload{
			ID: msgID, ContentType: "text", Content: "oi",
		}))
		inbound := producer.InboundMessages[i]
		assert.Equal(t, "session-abc", inbound.Metadata["sender_id"])

		out, err := uc.Execute(ctx, inbound)
		require.NoError(t, err)
		assert.Equal(t, client.ConversationID, out.Conversation.ID,
			"a mensagem %d caiu fora da conversa da sessão", i+1)
		assert.Equal(t, client.ContactID, out.Contact.ID)
	}

	assert.Len(t, conversationRepo.Conversations, 1)
	assert.Len(t, contactRepo.Contacts, 1)

	// A resposta do atendente é endereçada pela identidade webchat do contato: tem de ser a sessão.
	contact := contactRepo.Contacts[client.ContactID]
	require.NotNil(t, contact.GetIdentityByChannel("webchat"))
	assert.Equal(t, "session-abc", contact.GetIdentityByChannel("webchat").Identifier)

	// Um áudio gravado no widget vem sem texto e com um anexo. Tem de cair na mesma
	// conversa e virar uma mensagem de áudio, não de texto.
	require.NoError(t, h.handleClientMessage(ctx, client, channel, &MessagePayload{
		ID: "msg-4", ContentType: "audio", Content: "",
		Attachments: []AttachmentPayload{{
			Type: "audio", URL: "http://media.test/api/v1/media/webchat/ch-webchat/a.webm",
			Filename: "audio-1.webm", MimeType: "audio/webm", SizeBytes: 1234,
		}},
	}))
	out, err := uc.Execute(ctx, producer.InboundMessages[3])
	require.NoError(t, err)
	assert.Equal(t, client.ConversationID, out.Conversation.ID)
	assert.Equal(t, entity.ContentTypeAudio, out.Message.ContentType)
	require.Len(t, out.Message.Attachments, 1)
	assert.Equal(t, "audio", out.Message.Attachments[0].Type)
	assert.Equal(t, "audio/webm", out.Message.Attachments[0].MimeType)
	assert.Equal(t, int64(1234), out.Message.Attachments[0].SizeBytes)
	assert.Len(t, conversationRepo.Conversations, 1)

	// A visitante recarregou a página: sessão nova, mesmo e-mail. Continua o mesmo
	// contato e a mesma conversa, e a resposta do atendente passa a ser endereçada
	// à sessão nova, que é a conectada.
	reconnected := NewClient(nil, nil, "session-def")
	reconnected.Metadata["name"] = "Maria Souza"
	reconnected.Metadata["email"] = "maria.souza@exemplo.com"
	require.NoError(t, h.handleClientMessage(ctx, reconnected, channel, &MessagePayload{
		ID: "msg-5", ContentType: "text", Content: "voltei",
	}))
	assert.Equal(t, client.ContactID, reconnected.ContactID)
	assert.Equal(t, client.ConversationID, reconnected.ConversationID)
	out, err = uc.Execute(ctx, producer.InboundMessages[4])
	require.NoError(t, err)
	assert.Equal(t, client.ConversationID, out.Conversation.ID)
	assert.Len(t, contactRepo.Contacts, 1)
	assert.Equal(t, "session-def", contact.GetIdentityByChannel("webchat").Identifier)
}
