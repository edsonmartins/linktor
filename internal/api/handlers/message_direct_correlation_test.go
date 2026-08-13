package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	infrawebhook "github.com/msgfy/linktor/internal/infrastructure/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oneShotSubscriber feeds the dispatcher a single event through its public
// Start entry point, so the test drives it exactly as the runtime does.
type oneShotSubscriber struct{ event *nats.Event }

func (s *oneShotSubscriber) SubscribeEvents(ctx context.Context, handler nats.EventHandler) error {
	return handler(ctx, s.event)
}

// capturingDeliveries records the webhook bodies the dispatcher enqueues.
type capturingDeliveries struct{ bodies [][]byte }

func (c *capturingDeliveries) PublishWebhookDelivery(_ context.Context, wh *nats.WebhookDelivery) error {
	c.bodies = append(c.bodies, wh.Body)
	return nil
}

// TestDirectSendThenQuotedReplyRoundTripsTheCorrelation walks the whole loop the
// integration exists for: send with an opaque correlation token, the provider
// acknowledges with its own message id, the contact replies quoting it, and the
// outbound webhook hands the same token back.
//
// The two halves are otherwise only tested apart — this pins the seam between
// them: the outbound metadata Linktor persists, and the (conversation, external
// id) lookup that finds it again.
func TestDirectSendThenQuotedReplyRoundTripsTheCorrelation(t *testing.T) {
	d := setupDirectSend()
	channel := seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	channel.WebhookURL = "https://consumidor.example/webhook"

	// 1. Direct send carrying the integrator's correlation.
	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Aprova o despacho?",
		"metadata": map[string]string{
			"source":             "alcada",
			"alcada_correlation": "token-opaco",
			"idempotency_key":    "despacho-1",
		},
	})
	require.Equal(t, http.StatusAccepted, w.Code)
	data := directSendData(t, resp)
	conversationID := data["conversation_id"].(string)

	// 2. The provider acknowledges: the status pipeline stamps its message id on
	// our row. That id is the only handle the contact's reply will carry.
	sent := d.messages.Messages[data["id"].(string)]
	require.NotNil(t, sent)
	sent.ExternalID = "wamid.OUT-1"

	// 3. The contact replies quoting it, and the durable event carries the quoted
	// provider id (see buildMessageReceivedOutboxEvent).
	deliveries := &capturingDeliveries{}
	dispatcher := infrawebhook.NewDispatcher(deliveries, d.channels).WithMessages(d.messages)
	inbound := &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":      "msg_in",
			"channel_id":      "channel-1",
			"conversation_id": conversationID,
			"contact_id":      "contact_1",
			"content_type":    "text",
			"content":         "Pode seguir",
			"sender_id":       "+5511999999999",
			"reply_to_id":     "wamid.OUT-1",
		},
	}
	require.NoError(t, dispatcher.Start(context.Background(), &oneShotSubscriber{event: inbound}))

	// 4. The webhook hands the same correlation back — and nothing else.
	require.Len(t, deliveries.bodies, 1)
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			Context map[string]string `json:"context"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(deliveries.bodies[0], &envelope))
	assert.Equal(t, "message.received", envelope.Type)
	assert.Equal(t, "token-opaco", envelope.Data.Context["alcada_correlation"])
	assert.Equal(t, "alcada", envelope.Data.Context["source"])
	assert.NotContains(t, envelope.Data.Context, "idempotency_key")
}
