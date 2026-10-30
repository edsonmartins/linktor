package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Chamada pelo bridge. O WhatsApp só sinaliza (oferta, aceite, recusa,
// término); o áudio toca e é atendido no aparelho pareado, nunca passa por
// aqui. O Linktor registra a chamada na conversa do contato como mensagem de
// tipo call, acompanha o estado enquanto ela dura e pode recusá-la, por
// pedido do atendente ou pela configuração do canal.

// Estados de uma chamada (metadata call_state da mensagem).
const (
	CallStateRinging  = "ringing"
	CallStateAnswered = "answered"
	CallStateMissed   = "missed"
	CallStateRejected = "rejected"
	CallStateEnded    = "ended"
)

// Chaves de metadata da mensagem de chamada.
const (
	MetaCallID         = "call_id"
	MetaCallState      = "call_state"
	MetaCallVideo      = "call_video"
	MetaCallDuration   = "call_duration_seconds"
	MetaCallRejectedBy = "call_rejected_by"
)

// Quem recusou a chamada.
const (
	RejectedByAuto     = "auto"     // configuração do canal
	RejectedByOperator = "operator" // botão no admin
	RejectedByDevice   = "device"   // aparelho pareado ou o próprio WhatsApp
)

// defaultRingTimeout cobre o caso em que o término nunca chega (bridge caiu
// no meio do toque): o WhatsApp toca por volta de um minuto.
const defaultRingTimeout = 90 * time.Second

var (
	// ErrCallNotFound diz que não há chamada com esse ID em andamento.
	ErrCallNotFound = errors.New("call not found")
	// ErrCallNotRinging diz que a chamada já foi atendida, recusada ou acabou.
	ErrCallNotRinging = errors.New("call is not ringing")
)

// CallUpdate é uma mudança de estado depois da oferta. A oferta em si vira
// mensagem pelo pipeline normal (PublishInbound); o resto chega por aqui.
type CallUpdate struct {
	ChannelID  string
	CallID     string
	From       string // endereço com que a chamada foi sinalizada
	State      string
	RejectedBy string        // só em rejected
	Duration   time.Duration // só em ended
	At         time.Time
}

// CallSink recebe as mudanças de estado; roda fora do loop de leitura do
// bridge, então pode ir ao banco.
type CallSink func(ctx context.Context, update CallUpdate)

// SetCallSink registra quem grava o estado da chamada na mensagem.
func (h *Handler) SetCallSink(fn CallSink) {
	h.onCall = fn
}

type activeCall struct {
	channelID string
	// from é o endereço com que o WhatsApp sinalizou a chamada (o LID quando
	// veio por LID): a recusa tem de usar o mesmo call-creator.
	from       string
	state      string
	answeredAt time.Time
	ringTimer  *time.Timer
}

// callRegistry guarda as chamadas em andamento de todos os bridges, por
// call_id (o WhatsApp o gera único).
type callRegistry struct {
	mu          sync.Mutex
	calls       map[string]*activeCall
	ringTimeout time.Duration
}

func newCallRegistry() *callRegistry {
	return &callRegistry{calls: make(map[string]*activeCall), ringTimeout: defaultRingTimeout}
}

// add registra a chamada tocando; devolve false se o ID já existe (oferta
// repetida por reconexão do bridge).
func (r *callRegistry) add(callID string, call *activeCall, onTimeout func()) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.calls[callID]; exists {
		return false
	}
	call.state = CallStateRinging
	call.ringTimer = time.AfterFunc(r.ringTimeout, onTimeout)
	r.calls[callID] = call
	return true
}

func (r *callRegistry) get(callID string) (activeCall, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call, ok := r.calls[callID]
	if !ok {
		return activeCall{}, false
	}
	return *call, true
}

func (r *callRegistry) remove(callID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dropLocked(callID)
}

func (r *callRegistry) dropLocked(callID string) {
	if call, ok := r.calls[callID]; ok {
		if call.ringTimer != nil {
			call.ringTimer.Stop()
		}
		delete(r.calls, callID)
	}
}

// transition aplica uma sinalização e devolve a mudança resultante, se houve:
//   - answered: só enquanto toca;
//   - rejected: enquanto toca ou já atendida;
//   - terminate: tocando vira missed, atendida vira ended (com duração),
//     recusada só some.
//
// Estados finais saem do registro.
func (r *callRegistry) transition(callID, state, rejectedBy string, at time.Time) (CallUpdate, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call, ok := r.calls[callID]
	if !ok {
		return CallUpdate{}, false
	}
	update := CallUpdate{ChannelID: call.channelID, CallID: callID, From: call.from, At: at}
	switch state {
	case CallStateAnswered:
		if call.state != CallStateRinging {
			return CallUpdate{}, false
		}
		call.state = CallStateAnswered
		call.answeredAt = at
		if call.ringTimer != nil {
			call.ringTimer.Stop()
		}
	case CallStateRejected:
		if call.state == CallStateRejected {
			return CallUpdate{}, false
		}
		call.state = CallStateRejected
		update.RejectedBy = rejectedBy
		r.dropLocked(callID)
	case CallStateEnded: // término
		switch call.state {
		case CallStateRinging:
			state = CallStateMissed
		case CallStateAnswered:
			update.Duration = at.Sub(call.answeredAt)
		default:
			r.dropLocked(callID)
			return CallUpdate{}, false
		}
		r.dropLocked(callID)
	default:
		return CallUpdate{}, false
	}
	update.State = state
	return update, true
}

// handleCall recebe a sinalização de chamada do bridge (frame call_event).
func (h *Handler) handleCall(channelID string, raw json.RawMessage) error {
	if len(raw) == 0 {
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
	switch evt.Summary {
	case "offer":
		return h.callOffered(channelID, evt, at)
	case "accept":
		h.applyCall(evt.ID, CallStateAnswered, "", at)
	case "reject":
		h.applyCall(evt.ID, CallStateRejected, RejectedByDevice, at)
	case "terminate":
		h.applyCall(evt.ID, CallStateEnded, "", at)
	}
	return nil
}

// callOffered registra a chamada e a publica como mensagem da conversa do
// contato, com a mesma identidade de remetente das mensagens. Se o canal
// recusa chamadas automaticamente, a recusa sai em seguida.
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
	call := &activeCall{channelID: channelID, from: signaledBy}
	if !h.calls.add(evt.ID, call, func() { h.applyCall(evt.ID, CallStateEnded, "", time.Now()) }) {
		return nil
	}

	metadata := senderMetadata(evt)
	metadata[MetaCallID] = evt.ID
	metadata[MetaCallState] = CallStateRinging
	if evt.Video {
		metadata[MetaCallVideo] = "true"
	}
	inbound := &nats.InboundMessage{
		ID:          evt.ID,
		TenantID:    channel.TenantID,
		ChannelID:   channel.ID,
		ChannelType: string(channel.Type),
		ExternalID:  evt.ID,
		ContentType: string(plugin.ContentTypeCall),
		Content:     callContent(evt.Video),
		Metadata:    metadata,
		Timestamp:   at,
	}
	if err := h.producer.PublishInbound(ctx, inbound); err != nil {
		h.calls.remove(evt.ID)
		return fmt.Errorf("publish call: %w", err)
	}

	if settings := channel.GetAdvancedSettings(); settings != nil && settings.RejectCall {
		go func() {
			if err := h.RejectCall(context.Background(), channelID, evt.ID, RejectedByAuto); err != nil {
				log.Printf("gateway: auto-reject of call %s on %s failed: %v", evt.ID, channelID, err)
			}
		}()
	}
	return nil
}

// RejectCall manda o bridge recusar uma chamada que está tocando e registra
// a recusa. by diz quem pediu (RejectedByOperator ou RejectedByAuto).
func (h *Handler) RejectCall(ctx context.Context, channelID, callID, by string) error {
	call, ok := h.calls.get(callID)
	if !ok || call.channelID != channelID {
		return ErrCallNotFound
	}
	if call.state != CallStateRinging {
		return ErrCallNotRinging
	}
	ack, err := h.hub.RejectCallOnChannel(ctx, channelID, "call_"+callID, call.from, callID)
	if err != nil {
		return err
	}
	if !ack.OK {
		return fmt.Errorf("bridge could not reject call: %s", ack.Error)
	}
	h.applyCall(callID, CallStateRejected, by, time.Now())
	return nil
}

// applyCall registra a transição e avisa o sink fora do loop de leitura.
func (h *Handler) applyCall(callID, state, rejectedBy string, at time.Time) {
	update, ok := h.calls.transition(callID, state, rejectedBy, at)
	if !ok || h.onCall == nil {
		return
	}
	go h.onCall(context.Background(), update)
}

// callContent é o texto da mensagem de chamada: o que a lista de conversas e
// os consumidores de webhook veem; o admin desenha pelo metadata.
func callContent(video bool) string {
	if video {
		return "Chamada de vídeo"
	}
	return "Chamada de voz"
}

// eventTime converte o timestamp unix do bridge; zero vale agora.
func eventTime(ts int64) time.Time {
	if ts == 0 {
		return time.Now()
	}
	return time.Unix(ts, 0)
}
