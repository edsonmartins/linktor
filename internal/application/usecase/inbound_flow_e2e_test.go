package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// TestInboundFlowE2E_ChainedFromHandlerToPersistence exercises the full inbound
// pipeline downstream of a webhook: a normalized *nats.InboundMessage (built to
// mirror exactly what WebhookHandler.processFacebookMessage publishes) is run
// through a real ReceiveMessageUseCase wired with the testutil mocks.
//
// It locks two homologation guarantees:
//  1. First delivery persists the message and publishes a MessageReceived event
//     (at-least-once intake).
//  2. A redelivery of the SAME inbound (same channel + external id) is detected
//     as a duplicate, returns a CONFLICT, and does NOT persist a second row —
//     the dedup / at-least-once lock.
func TestInboundFlowE2E_ChainedFromHandlerToPersistence(t *testing.T) {
	ctx := context.Background()
	f := newReceiveMessageFixture()

	// A Facebook channel — the inbound below mirrors the handler's Facebook
	// normalization (channel_type "facebook", sender_id/page_id metadata).
	now := time.Now()
	channel := &entity.Channel{
		ID:               "ch-fb",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeFacebook,
		Name:             "FB Page",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config:           make(map[string]string),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	f.channelRepo.Channels[channel.ID] = channel

	// buildInbound mirrors WebhookHandler.processFacebookMessage output so each
	// delivery is an independent value (the usecase mutates the struct in place).
	buildInbound := func() *nats.InboundMessage {
		return &nats.InboundMessage{
			ID:          "inbound-fb-1",
			TenantID:    "tenant-1",
			ChannelID:   "ch-fb",
			ChannelType: "facebook",
			ExternalID:  "m_fb_abc123",
			ContentType: "text",
			Content:     "Hello from Messenger",
			Metadata: map[string]string{
				"sender_id": "user-999",
				"page_id":   "page-123",
			},
			Timestamp: now,
		}
	}

	// --- First delivery: persist + enqueue MessageReceived in the outbox ----
	out, err := f.uc.Execute(ctx, buildInbound())
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Equal(t, "ch-fb", out.Conversation.ChannelID)
	assert.Equal(t, entity.ChannelTypeFacebook, channel.Type)
	assert.Equal(t, "m_fb_abc123", out.Message.ExternalID)
	assert.Equal(t, "Hello from Messenger", out.Message.Content)
	assert.True(t, out.IsNew)

	// Exactly one message persisted.
	require.Len(t, f.messageRepo.Messages, 1)

	// A MessageReceived event was durably enqueued (transactional outbox) for the
	// freshly stored message; the relay publishes it, so it is not in the producer
	// stream here.
	require.Len(t, f.messageRepo.OutboxEvents, 1, "MessageReceived must be queued on first delivery")
	received := f.messageRepo.OutboxEvents[0]
	assert.Equal(t, nats.EventMessageReceived, received.EventType)
	assert.Equal(t, "evt-message-received-"+out.Message.ID, received.IdempotencyKey)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(received.Payload, &payload))
	assert.Equal(t, out.Message.ID, payload["message_id"])

	// --- Redelivery: same inbound → CONFLICT, no double-persist, no dup event -
	dupOut, dupErr := f.uc.Execute(ctx, buildInbound())
	assert.Nil(t, dupOut)
	require.Error(t, dupErr, "duplicate delivery must return an error so the consumer acks")
	assert.Contains(t, dupErr.Error(), "CONFLICT")

	// No second message row and no second outbox event were created.
	assert.Len(t, f.messageRepo.Messages, 1, "duplicate delivery must not persist a second message")
	assert.Len(t, f.messageRepo.OutboxEvents, 1, "duplicate delivery must not enqueue a second event")
}
