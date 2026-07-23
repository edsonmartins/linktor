package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSandboxMessageTest builds a MessageService over a sandbox WhatsApp
// channel with the synchronous allowlist check wired.
func setupSandboxMessageTest(env entity.ChannelEnvironment) (*MessageService, *testutil.MockMessageRepository, *testutil.MockSandboxAllowlistRepository) {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	allowlist := testutil.NewMockSandboxAllowlistRepository()

	contactRepo.Contacts["contact1"] = &entity.Contact{
		ID: "contact1", TenantID: "tenant1", Phone: "5511999999999",
		Identities: []*entity.ContactIdentity{{ChannelType: "whatsapp_official", Identifier: "5511999999999"}},
	}
	channelRepo.Channels["channel1"] = &entity.Channel{
		ID: "channel1", TenantID: "tenant1",
		Type: entity.ChannelTypeWhatsAppOfficial, Environment: env,
	}
	convRepo.Conversations["conv1"] = &entity.Conversation{
		ID: "conv1", TenantID: "tenant1", ContactID: "contact1", ChannelID: "channel1",
		Status: entity.ConversationStatusOpen,
	}

	svc := NewMessageService(msgRepo, convRepo, channelRepo, contactRepo, nil)
	svc.SetSandboxAllowlist(allowlist)
	return svc, msgRepo, allowlist
}

func sendInput() *SendMessageInput {
	return &SendMessageInput{
		TenantID:       "tenant1",
		ConversationID: "conv1",
		SenderType:     "user",
		SenderID:       "user1",
		Content:        "hello",
	}
}

func TestMessageService_Send_SandboxUnlistedRecipientFailsFast(t *testing.T) {
	svc, msgRepo, _ := setupSandboxMessageTest(entity.ChannelEnvironmentSandbox)

	_, err := svc.Send(context.Background(), sendInput())

	require.Error(t, err, "API must reject synchronously for an unlisted recipient")
	assert.True(t, errors.IsValidation(err))
	assert.NotContains(t, err.Error(), "5511999999999", "error must not leak the full recipient")
	assert.Empty(t, msgRepo.Messages, "no message row may be created for a fail-fast rejection")
}

func TestMessageService_Send_SandboxAllowlistedRecipientProceeds(t *testing.T) {
	svc, msgRepo, allowlist := setupSandboxMessageTest(entity.ChannelEnvironmentSandbox)
	allowlist.Entries["e1"] = &entity.SandboxAllowlistEntry{
		ID: "e1", TenantID: "tenant1", Recipient: "+5511999999999",
	}

	msg, err := svc.Send(context.Background(), sendInput())

	require.NoError(t, err)
	assert.Len(t, msgRepo.Messages, 1)
	assert.Equal(t, entity.MessageStatusPending, msg.Status)
}

func TestMessageService_Send_ProductionChannelUnaffected(t *testing.T) {
	// Production channel with an EMPTY allowlist: behavior must be identical to
	// before this feature — the check simply does not apply.
	svc, msgRepo, _ := setupSandboxMessageTest(entity.ChannelEnvironmentProduction)

	_, err := svc.Send(context.Background(), sendInput())

	require.NoError(t, err)
	assert.Len(t, msgRepo.Messages, 1)
}
