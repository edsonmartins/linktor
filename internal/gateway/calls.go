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
// término); o áudio toca e é atendido no aparelho principal do WhatsApp,
// nunca passa por aqui. O Linktor registra a chamada na conversa do contato
// como mensagem de tipo call, guarda o desfecho e pode recusá-la, por pedido
// do atendente ou pela configuração do canal.

// Estados de uma chamada (metadata call_state da mensagem). Tocando é o
// único estado intermediário; os outros três são o desfecho, e cada chamada
// tem um só. O bridge é um aparelho pareado e não vê o fim da conversa:
// quando o aparelho principal atende, o WhatsApp encerra a participação do
// bridge no mesmo instante, então "atendida" é final e a duração não é
// observável.
const (
	CallStateRinging  = "ringing"
	CallStateAnswered = "answered"
	CallStateMissed   = "missed"
	CallStateRejected = "rejected"
)

// Chaves de metadata da mensagem de chamada.
const (
	MetaCallID         = "call_id"
	MetaCallState      = "call_state"
	MetaCallVideo      = "call_video"
	MetaCallRejectedBy = "call_rejected_by"
)

// Quem recusou a chamada.
const (
	RejectedByAuto     = "auto"     // configuração do canal
	RejectedByOperator = "operator" // botão no admin
	RejectedByDevice   = "device"   // aparelho pareado ou o próprio WhatsApp
)

// defaultRingTimeout cobre o caso em que nenhum desfecho chega (bridge caiu
// no meio do toque): o WhatsApp toca por volta de um minuto.
const defaultRingTimeout = 90 * time.Second

// ErrCallNotFound diz que não há chamada tocando com esse ID.
var ErrCallNotFound = errors.New("call not found")

// CallUpdate é o desfecho de uma chamada. A oferta em si vira mensagem pelo
// pipeline normal (PublishInbound); o desfecho chega por aqui, uma vez.
type CallUpdate struct {
	ChannelID  string
	CallID     string
	From       string // endereço com que a chamada foi sinalizada
	State      string // answered | missed | rejected
	RejectedBy string // só em rejected
	At         time.Time
}

// CallSink recebe os desfechos; roda fora do loop de leitura do bridge,
// então pode ir ao banco.
type CallSink func(ctx context.Context, update CallUpdate)

// SetCallSink registra quem grava o desfecho da chamada na mensagem.
func (h *Handler) SetCallSink(fn CallSink) {
	h.onCall = fn
}

type activeCall struct {
	channelID string
	// from é o endereço com que o WhatsApp sinalizou a chamada (o LID quando
	// veio por LID): a recusa tem de usar o mesmo call-creator.
	from      string
	ringTimer *time.Timer
}

// callRegistry guarda as chamadas tocando em todos os bridges, por call_id
// (o WhatsApp o gera único). Estar no registro é estar tocando.
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
		call.ringTimer.Stop()
		delete(r.calls, callID)
	}
}

// finish tira a chamada do registro com o desfecho dado. Só a primeira
// sinalização conta: o término que segue um aceite ou uma recusa encontra a
// chamada já resolvida e não gera nada.
func (r *callRegistry) finish(callID, state, rejectedBy string, at time.Time) (CallUpdate, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call, ok := r.calls[callID]
	if !ok {
		return CallUpdate{}, false
	}
	r.dropLocked(callID)
	return CallUpdate{
		ChannelID:  call.channelID,
		CallID:     callID,
		From:       call.from,
		State:      state,
		RejectedBy: rejectedBy,
		At:         at,
	}, true
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
		h.finishCall(evt.ID, CallStateAnswered, "", at)
	case "reject":
		h.finishCall(evt.ID, CallStateRejected, RejectedByDevice, at)
	case "terminate":
		// Término com a chamada ainda tocando: quem ligou desistiu.
		h.finishCall(evt.ID, CallStateMissed, "", at)
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
	if !h.calls.add(evt.ID, call, func() { h.finishCall(evt.ID, CallStateMissed, "", time.Now()) }) {
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
	ack, err := h.hub.RejectCallOnChannel(ctx, channelID, "call_"+callID, call.from, callID)
	if err != nil {
		return err
	}
	if !ack.OK {
		return fmt.Errorf("bridge could not reject call: %s", ack.Error)
	}
	h.finishCall(callID, CallStateRejected, by, time.Now())
	return nil
}

// finishCall registra o desfecho e avisa o sink fora do loop de leitura.
func (h *Handler) finishCall(callID, state, rejectedBy string, at time.Time) {
	update, ok := h.calls.finish(callID, state, rejectedBy, at)
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
