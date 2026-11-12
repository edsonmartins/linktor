package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pricingChannels struct{ ch *entity.Channel }

func (p pricingChannels) FindByID(context.Context, string) (*entity.Channel, error) { return p.ch, nil }

type pricingDeliveries struct{ got []*nats.WebhookDelivery }

func (p *pricingDeliveries) PublishWebhookDelivery(_ context.Context, d *nats.WebhookDelivery) error {
	p.got = append(p.got, d)
	return nil
}

type pricingSubscriber struct{ handler nats.EventHandler }

func (s *pricingSubscriber) SubscribeEvents(_ context.Context, h nats.EventHandler) error {
	s.handler = h
	return nil
}

// appBody leva o StatusUpdate pelo caminho até o webhook do app: o payload do
// evento, o JSON do NATS no meio (que vira map[string]interface{}) e o
// dispatcher. Devolve o `data` do envelope e o id da entrega.
func appBody(t *testing.T, deliveries *pricingDeliveries, sub *pricingSubscriber, eventType string, status *nats.StatusUpdate) (map[string]interface{}, string) {
	t.Helper()
	raw, err := json.Marshal(&nats.Event{Type: eventType, TenantID: "t-1", Payload: statusEventPayload("msg-1", "ch-1", status)})
	require.NoError(t, err)
	var event nats.Event
	require.NoError(t, json.Unmarshal(raw, &event))

	require.NoError(t, sub.handler(context.Background(), &event))
	last := deliveries.got[len(deliveries.got)-1]
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(last.Body, &env))
	return env.Data, last.ID
}

func newPricingDispatcher(t *testing.T) (*pricingDeliveries, *pricingSubscriber) {
	t.Helper()
	deliveries := &pricingDeliveries{}
	sub := &pricingSubscriber{}
	ch := &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: entity.ChannelTypeWhatsAppOfficial, WebhookURL: "https://core.example/webhook"}
	require.NoError(t, webhook.NewDispatcher(deliveries, pricingChannels{ch}).Start(context.Background(), sub))
	return deliveries, sub
}

// O pricing da Meta chega ao app em data.pricing, com os nomes do contrato.
func TestStatusPricingReachesAppWebhook(t *testing.T) {
	deliveries, sub := newPricingDispatcher(t)

	data, _ := appBody(t, deliveries, sub, nats.EventMessageDelivered, &nats.StatusUpdate{
		ExternalID: "wamid.1", Status: "delivered",
		Pricing: &nats.StatusPricing{Billable: true, PricingModel: "PMP", Category: "utility"},
	})
	assert.Equal(t, "msg-1", data["messageId"])
	assert.Equal(t, "delivered", data["status"])
	assert.Equal(t, "whatsapp_official", data["channelType"])
	assert.Equal(t, map[string]interface{}{"billable": true, "pricingModel": "PMP", "category": "utility"}, data["pricing"])

	// billable=false não some do JSON
	data, _ = appBody(t, deliveries, sub, nats.EventMessageRead, &nats.StatusUpdate{
		ExternalID: "wamid.1", Status: "read",
		Pricing: &nats.StatusPricing{Billable: false, PricingModel: "PMP", Category: "service"},
	})
	assert.Equal(t, map[string]interface{}{"billable": false, "pricingModel": "PMP", "category": "service"}, data["pricing"])
}

// Sem pricing da Meta, sem o campo — nem null, nem objeto vazio.
func TestStatusWithoutPricingHasNoField(t *testing.T) {
	deliveries, sub := newPricingDispatcher(t)

	data, _ := appBody(t, deliveries, sub, nats.EventMessageDelivered, &nats.StatusUpdate{ExternalID: "wamid.1", Status: "delivered"})
	_, present := data["pricing"]
	assert.False(t, present, "data: %v", data)
	assert.NotContains(t, string(deliveries.got[0].Body), "pricing")
}

// O "sent" do ack do envio e o "sent" da Meta com pricing não podem ter o mesmo
// id de entrega, ou o segundo morre na deduplicação do stream com o pricing
// junto. A reentrega do mesmo status com pricing continua com o mesmo id.
func TestStatusPricingHasOwnDeliveryID(t *testing.T) {
	deliveries, sub := newPricingDispatcher(t)

	_, ackID := appBody(t, deliveries, sub, nats.EventMessageSent, &nats.StatusUpdate{ExternalID: "wamid.1", Status: "sent"})
	pricing := &nats.StatusUpdate{ExternalID: "wamid.1", Status: "sent",
		Pricing: &nats.StatusPricing{Billable: true, PricingModel: "PMP", Category: "marketing"}}
	_, metaID := appBody(t, deliveries, sub, nats.EventMessageSent, pricing)
	_, redeliveredID := appBody(t, deliveries, sub, nats.EventMessageSent, pricing)

	assert.NotEqual(t, ackID, metaID)
	assert.Equal(t, metaID, redeliveredID)
}
