package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/msgfy/linktor/internal/callsignal"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Chamada pelo bridge: o gateway só traduz os frames call_event para o
// callsignal.Tracker, que guarda o desfecho e a recusa (o mesmo registro da
// sessão whatsmeow embutida no servidor).

// SetCallTracker registra o registro de chamadas. Sem ele os frames de
// chamada são ignorados.
func (h *Handler) SetCallTracker(t *callsignal.Tracker) {
	h.calls = t
}

// handleCall recebe a sinalização de chamada do bridge (frame call_event).
func (h *Handler) handleCall(channelID string, raw json.RawMessage) error {
	if h.calls == nil || len(raw) == 0 {
		return nil
	}
	var evt InboundEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return fmt.Errorf("parse call payload: %w", err)
	}
	if evt.ID == "" {
		return nil
	}
	at := eventTime(evt.Timestamp)
	if evt.Summary == "offer" {
		return h.callOffered(channelID, evt, at)
	}
	h.calls.Signal(evt.ID, evt.Summary, at)
	return nil
}

// callOffered entrega a oferta ao registro: a mensagem vai pela mesma
// identidade de remetente das mensagens, e a recusa vai ao bridge endereçada
// a quem sinalizou (o LID quando veio por LID: o WhatsApp exige o mesmo
// call-creator).
func (h *Handler) callOffered(channelID string, evt InboundEvent, at time.Time) error {
	if h.producer == nil {
		return nil
	}
	ctx := context.Background()
	channel, err := h.channels.FindByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("resolve channel: %w", err)
	}
	signaledBy := evt.From
	if evt.SenderLID != "" {
		signaledBy = evt.SenderLID
	}

	metadata := senderMetadata(evt)
	for k, v := range callsignal.MessageMetadata(evt.ID, evt.Video) {
		metadata[k] = v
	}
	inbound := &nats.InboundMessage{
		ID:          evt.ID,
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: string(channel.Type),
		ExternalID:  evt.ID,
		ContentType: string(plugin.ContentTypeCall),
		Content:     callsignal.Content(evt.Video),
		Metadata:    metadata,
		Timestamp:   at,
	}
	return h.calls.Offer(ctx, callsignal.Offer{
		ChannelID: channelID,
		CallID:    evt.ID,
		Publish: func(ctx context.Context) error {
			return h.producer.PublishInbound(ctx, inbound)
		},
		Reject: func(ctx context.Context) error {
			ack, err := h.hub.RejectCallOnChannel(ctx, channelID, "call_"+evt.ID, signaledBy, evt.ID)
			if errors.Is(err, ErrNoBridge) {
				return fmt.Errorf("%w: %v", callsignal.ErrDeviceOffline, err)
			}
			if err != nil {
				return err
			}
			if !ack.OK {
				return errors.New("bridge could not reject call: " + ack.Error)
			}
			return nil
		},
	})
}

// eventTime converte o timestamp unix do bridge; zero vale agora.
func eventTime(ts int64) time.Time {
	if ts == 0 {
		return time.Now()
	}
	return time.Unix(ts, 0)
}
