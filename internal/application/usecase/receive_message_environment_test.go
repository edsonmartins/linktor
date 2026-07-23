package usecase

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReceiveMessage_SandboxConversationCarriesEnvironment verifies INV-018:
// a conversation born from an inbound message on a sandbox channel is marked
// sandbox, and the message.received outbox event payload carries the marking.
func TestReceiveMessage_SandboxConversationCarriesEnvironment(t *testing.T) {
	f := newReceiveMessageFixture()
	ch := makeChannel("channel-sb", "tenant1")
	ch.Environment = entity.ChannelEnvironmentSandbox
	f.channelRepo.Channels[ch.ID] = ch

	inbound := makeInbound(ch.ID, "tenant1")
	out, err := f.uc.Execute(context.Background(), inbound)
	require.NoError(t, err)

	conv, err := f.conversationRepo.FindByID(context.Background(), out.Conversation.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ChannelEnvironmentSandbox, conv.Environment,
		"conversation created on a sandbox channel must be marked sandbox")

	require.NotEmpty(t, f.messageRepo.OutboxEvents, "message.received outbox event must exist")
	payload := string(f.messageRepo.OutboxEvents[len(f.messageRepo.OutboxEvents)-1].Payload)
	assert.Contains(t, payload, `"environment":"sandbox"`,
		"event payload must carry the environment marking")
}

// TestReceiveMessage_ProductionConversationDefaults confirms production (and
// legacy, unmarked) channels keep producing production-marked conversations.
func TestReceiveMessage_ProductionConversationDefaults(t *testing.T) {
	f := newReceiveMessageFixture()
	ch := makeChannel("channel-prod", "tenant1") // no Environment set (legacy)
	f.channelRepo.Channels[ch.ID] = ch

	inbound := makeInbound(ch.ID, "tenant1")
	out, err := f.uc.Execute(context.Background(), inbound)
	require.NoError(t, err)

	conv, err := f.conversationRepo.FindByID(context.Background(), out.Conversation.ID)
	require.NoError(t, err)
	assert.NotEqual(t, entity.ChannelEnvironmentSandbox, conv.Environment)
}
