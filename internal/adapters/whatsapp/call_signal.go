package whatsapp

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/callsignal"
	"github.com/msgfy/linktor/pkg/plugin"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Sinalização de chamada para o registro de chamadas do Linktor
// (callsignal). A sessão embutida é um aparelho conectado: recebe só o
// aviso de que a chamada toca (CallOfferNotice) e os sinais seguintes, nunca
// o áudio. Isto é independente do motor de voz (calling.go), que só age
// sobre a oferta completa.

// callSignal é um sinal de chamada do whatsmeow, na fila de eventos do
// cliente para manter a ordem oferta → aceite/término.
type callSignal struct {
	Signal     string // offer | accept | reject | terminate
	CallID     string
	From       types.JID // quem sinalizou (o LID quando veio por LID)
	CreatorAlt types.JID // o número de quem ligou, quando o WhatsApp mandou
	Video      bool
	Time       time.Time
}

// CallSignalEvent é o sinal entregue ao host. Message só vem na oferta: a
// chamada como mensagem de tipo call, com a mesma identidade das mensagens.
type CallSignalEvent struct {
	CallID string
	Signal string
	At     time.Time
	// SignaledBy é o endereço com que a chamada foi sinalizada; a recusa
	// (RejectCallFrom) tem de usar o mesmo.
	SignaledBy string
	Message    *plugin.InboundMessage
}

// CallSignalHandler recebe os sinais de chamada no loop de eventos.
type CallSignalHandler func(ctx context.Context, evt CallSignalEvent)

// SetCallSignalHandler registra quem recebe os sinais de chamada.
func (a *Adapter) SetCallSignalHandler(handler CallSignalHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.callSignalHandler = handler
}

// RejectCallFrom recusa a chamada callID sinalizada por signaledBy. Não
// depende do motor de voz: um aparelho conectado nunca recebe a oferta
// completa que ele precisaria.
func (a *Adapter) RejectCallFrom(ctx context.Context, signaledBy, callID string) error {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()
	if client == nil {
		return ErrClientNotReady
	}
	raw := client.GetRawClient()
	if raw == nil || !raw.IsConnected() {
		return ErrClientNotReady
	}
	from, err := types.ParseJID(signaledBy)
	if err != nil {
		return fmt.Errorf("invalid caller JID: %w", err)
	}
	return raw.RejectCall(ctx, from, callID)
}

// toCallSignal traduz os eventos de chamada do whatsmeow; nil para o resto.
func toCallSignal(evt any) *callSignal {
	switch v := evt.(type) {
	case *events.CallOffer:
		return newCallSignal(callsignalOffer, v.BasicCallMeta, isVideoOffer(v.Data))
	case *events.CallOfferNotice:
		return newCallSignal(callsignalOffer, v.BasicCallMeta, v.Media == "video")
	case *events.CallAccept:
		return newCallSignal(callsignal.SignalAccept, v.BasicCallMeta, false)
	case *events.CallReject:
		return newCallSignal(callsignal.SignalReject, v.BasicCallMeta, false)
	case *events.CallTerminate:
		return newCallSignal(callsignal.SignalTerminate, v.BasicCallMeta, false)
	}
	return nil
}

const callsignalOffer = "offer"

func newCallSignal(signal string, meta types.BasicCallMeta, video bool) *callSignal {
	return &callSignal{Signal: signal, CallID: meta.CallID, From: meta.From, CreatorAlt: meta.CallCreatorAlt, Video: video, Time: meta.Timestamp}
}

// isVideoOffer lê no nó da oferta se há faixa de vídeo.
func isVideoOffer(offer *waBinary.Node) bool {
	return offer != nil && len(offer.GetChildrenByTag("video")) > 0
}

// handleCallSignal resolve a identidade da oferta como nas mensagens (o
// número, com o LID em metadata) e entrega o sinal ao host.
func (a *Adapter) handleCallSignal(ctx context.Context, client *Client, sig *callSignal) {
	a.mu.RLock()
	handler := a.callSignalHandler
	a.mu.RUnlock()
	if handler == nil || sig.CallID == "" {
		return
	}
	evt := CallSignalEvent{CallID: sig.CallID, Signal: sig.Signal, At: sig.Time, SignaledBy: sig.From.ToNonAD().String()}
	if evt.At.IsZero() {
		evt.At = time.Now()
	}
	if sig.Signal == callsignalOffer {
		pn := sig.From.ToNonAD()
		if isLID(pn) {
			if !sig.CreatorAlt.IsEmpty() && !isLID(sig.CreatorAlt) {
				pn = sig.CreatorAlt.ToNonAD()
			} else if client != nil {
				if resolved := client.ResolvePN(ctx, pn); resolved.User != "" {
					pn = resolved.ToNonAD()
				}
			}
		}
		evt.Message = callMessage(sig, pn, evt.At)
		evt.Message.SenderName = callerName(ctx, client, pn)
	}
	handler(ctx, evt)
}

// callerName é o nome de quem ligou na agenda da sessão. A chamada, ao
// contrário da mensagem, não traz o pushName; sem isto quem liga antes de
// mandar mensagem vira "Contato Desconhecido".
func callerName(ctx context.Context, client *Client, pn types.JID) string {
	if client == nil || isLID(pn) {
		return ""
	}
	info, err := client.GetContactInfo(ctx, pn)
	if err != nil || info == nil {
		return ""
	}
	for _, name := range []string{info.PushName, info.FullName, info.BusinessName} {
		if name != "" {
			return name
		}
	}
	return ""
}

// callMessage monta a chamada como mensagem, com as chaves de identidade de
// convertToInboundMessage.
func callMessage(sig *callSignal, pn types.JID, at time.Time) *plugin.InboundMessage {
	signaledBy := sig.From.ToNonAD()
	metadata := callsignal.MessageMetadata(sig.CallID, sig.Video)
	metadata["chat_jid"] = pn.String()
	metadata["sender_jid"] = signaledBy.String()
	metadata["is_group"] = "false"
	if !isLID(pn) {
		metadata["sender_pn"] = pn.String()
		// Mesmo contrato do bridge e do webhook oficial: com o número em
		// "phone" a chamada casa com o contato que já existe com esse número
		// em outro canal, em vez de abrir um contato novo.
		metadata["phone"] = pn.User
		if isLID(signaledBy) {
			metadata["lid"] = signaledBy.String()
		}
	}
	return &plugin.InboundMessage{
		ID:          uuid.New().String(),
		ExternalID:  sig.CallID,
		SenderID:    pn.User,
		ContentType: plugin.ContentTypeCall,
		Content:     callsignal.Content(sig.Video),
		Timestamp:   at,
		Metadata:    metadata,
	}
}
