package service

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/adapters/whatsapp"
	"github.com/msgfy/linktor/internal/callsignal"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// A sessão embutida entrega a oferta ao registro compartilhado, que publica a
// chamada pelo mesmo caminho das mensagens (com sender_id); os sinais
// seguintes resolvem o desfecho.
func TestChannelService_WhatsAppCallSignal(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	channel := &entity.Channel{ID: "ch-wa", TenantID: "t-1", Type: entity.ChannelTypeWhatsApp}
	repo.Channels[channel.ID] = channel
	producer := testutil.NewMockProducer()
	svc := NewChannelService(repo, nil, producer)

	adapter := whatsapp.NewAdapter()
	offer := whatsapp.CallSignalEvent{
		CallID: "C1", Signal: "offer", At: time.Unix(7, 0), SignaledBy: "252415295144120@lid",
		Message: &plugin.InboundMessage{
			ID: "m-1", ExternalID: "C1", SenderID: "554499806761", ContentType: plugin.ContentTypeCall,
			Content: "Chamada de voz", Timestamp: time.Unix(7, 0),
			Metadata: callsignal.MessageMetadata("C1", false),
		},
	}

	// sem registro configurado a chamada é ignorada
	svc.handleWhatsAppCallSignal(context.Background(), adapter, channel, offer)
	require.Empty(t, producer.InboundMessages)

	tracker := callsignal.NewTracker(repo)
	updates := make(chan callsignal.Update, 4)
	tracker.SetSink(func(_ context.Context, u callsignal.Update) { updates <- u })
	svc.SetCallTracker(tracker)

	svc.handleWhatsAppCallSignal(context.Background(), adapter, channel, offer)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	require.Equal(t, "call", msg.ContentType)
	require.Equal(t, "C1", msg.ExternalID)
	require.Equal(t, "ch-wa", msg.ChannelID)
	require.Equal(t, "554499806761", msg.Metadata["sender_id"])
	require.True(t, tracker.Ringing("C1"))

	// a recusa vai pelo adaptador; sem sessão ligada é aparelho offline
	err := tracker.Reject(context.Background(), "ch-wa", "C1", callsignal.RejectedByOperator)
	require.ErrorIs(t, err, callsignal.ErrDeviceOffline)

	svc.handleWhatsAppCallSignal(context.Background(), adapter, channel, whatsapp.CallSignalEvent{CallID: "C1", Signal: callsignal.SignalAccept, At: time.Unix(9, 0)})
	select {
	case u := <-updates:
		require.Equal(t, callsignal.StateAnswered, u.State)
		require.Equal(t, "ch-wa", u.ChannelID)
	case <-time.After(2 * time.Second):
		t.Fatal("desfecho não chegou")
	}
}
