// Package callsignal registra chamadas de WhatsApp vistas por um aparelho
// conectado: o Linktor Bridge ou a sessão whatsmeow embutida no servidor.
//
// O WhatsApp só sinaliza a chamada a um aparelho conectado (oferta, aceite,
// recusa, término); o áudio toca e é atendido no aparelho principal, nunca
// passa por aqui. Quem sinaliza (gateway do bridge ou adaptador embutido)
// publica a chamada como mensagem de tipo call na conversa do contato e
// entrega os sinais ao Tracker, que guarda o desfecho e pode recusá-la, por
// pedido do atendente ou pela configuração do canal.
package callsignal

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Estados de uma chamada (metadata call_state da mensagem). Tocando é o
// único estado intermediário; os outros três são o desfecho, e cada chamada
// tem um só. Um aparelho conectado não vê o fim da conversa: quando o
// aparelho principal atende, o WhatsApp encerra a participação dele no mesmo
// instante, então "atendida" é final e a duração não é observável.
const (
	StateRinging  = "ringing"
	StateAnswered = "answered"
	StateMissed   = "missed"
	StateRejected = "rejected"
)

// Chaves de metadata da mensagem de chamada.
const (
	MetaCallID     = "call_id"
	MetaState      = "call_state"
	MetaVideo      = "call_video"
	MetaRejectedBy = "call_rejected_by"
)

// Quem recusou a chamada.
const (
	RejectedByAuto     = "auto"     // configuração do canal
	RejectedByOperator = "operator" // botão no admin
	RejectedByDevice   = "device"   // aparelho principal ou o próprio WhatsApp
)

// Sinais que chegam depois da oferta.
const (
	SignalAccept    = "accept"
	SignalReject    = "reject"
	SignalTerminate = "terminate"
)

// defaultRingTimeout cobre o caso em que nenhum desfecho chega (aparelho caiu
// no meio do toque): o WhatsApp toca por volta de um minuto.
const defaultRingTimeout = 90 * time.Second

var (
	// ErrCallNotFound diz que não há chamada tocando com esse ID nesse canal.
	ErrCallNotFound = errors.New("call not found")
	// ErrDeviceOffline é o que Offer.Reject devolve (embrulhado) quando o
	// aparelho que sinalizou a chamada não está ligado agora.
	ErrDeviceOffline = errors.New("device offline")
)

// Update é o desfecho de uma chamada; chega ao Sink uma vez por chamada.
type Update struct {
	ChannelID  string
	CallID     string
	State      string // answered | missed | rejected
	RejectedBy string // só em rejected
	At         time.Time
}

// Sink recebe os desfechos fora do caminho de quem sinalizou, então pode ir
// ao banco.
type Sink func(ctx context.Context, update Update)

// ChannelFinder é o pedaço do repositório de canais que o Tracker usa para
// ler a recusa automática.
type ChannelFinder interface {
	FindByID(ctx context.Context, id string) (*entity.Channel, error)
}

// Offer é uma chamada que começou a tocar.
type Offer struct {
	ChannelID string
	CallID    string
	// Publish grava a chamada na conversa (mensagem de tipo call, ver
	// MessageMetadata e Content). Só roda se a oferta for nova.
	Publish func(ctx context.Context) error
	// Reject pede ao aparelho que recuse a chamada; cada origem sabe como
	// (frame call_command no bridge, whatsmeow no adaptador embutido).
	Reject func(ctx context.Context) error
}

type ringing struct {
	channelID string
	reject    func(ctx context.Context) error
	timer     *time.Timer
}

// Tracker guarda as chamadas tocando de todos os canais, por call_id (o
// WhatsApp o gera único). Estar no registro é estar tocando.
type Tracker struct {
	channels    ChannelFinder
	ringTimeout time.Duration

	mu    sync.Mutex
	calls map[string]*ringing
	sink  Sink
}

// NewTracker cria o registro.
func NewTracker(channels ChannelFinder) *Tracker {
	return &Tracker{channels: channels, ringTimeout: defaultRingTimeout, calls: make(map[string]*ringing)}
}

// SetSink registra quem grava o desfecho na mensagem da chamada.
func (t *Tracker) SetSink(fn Sink) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sink = fn
}

// Offer registra a chamada tocando e a publica na conversa. Oferta repetida
// (reconexão do aparelho) não publica de novo. Se o canal recusa chamadas
// automaticamente, a recusa sai em seguida.
func (t *Tracker) Offer(ctx context.Context, o Offer) error {
	t.mu.Lock()
	if _, exists := t.calls[o.CallID]; exists {
		t.mu.Unlock()
		return nil
	}
	callID := o.CallID
	t.calls[callID] = &ringing{
		channelID: o.ChannelID,
		reject:    o.Reject,
		timer:     time.AfterFunc(t.ringTimeout, func() { t.Signal(callID, SignalTerminate, time.Now()) }),
	}
	t.mu.Unlock()

	if err := o.Publish(ctx); err != nil {
		t.drop(callID)
		return err
	}

	if t.autoReject(ctx, o.ChannelID) {
		go func() {
			if err := t.Reject(context.Background(), o.ChannelID, callID, RejectedByAuto); err != nil {
				log.Printf("callsignal: auto-reject of call %s on %s failed: %v", callID, o.ChannelID, err)
			}
		}()
	}
	return nil
}

// Signal aplica o primeiro sinal depois da oferta: aceite é atendida, recusa
// pelo aparelho é recusada, término com a chamada tocando é não atendida.
// Sinais de chamada desconhecida ou já resolvida são ignorados: é o término
// que segue um aceite ou uma recusa.
func (t *Tracker) Signal(callID, signal string, at time.Time) {
	switch signal {
	case SignalAccept:
		t.finish(callID, StateAnswered, "", at)
	case SignalReject:
		t.finish(callID, StateRejected, RejectedByDevice, at)
	case SignalTerminate:
		t.finish(callID, StateMissed, "", at)
	}
}

// Reject pede ao aparelho que recuse uma chamada que está tocando e registra
// a recusa. by diz quem pediu (RejectedByOperator ou RejectedByAuto). Se o
// aparelho falhar, a chamada continua tocando.
func (t *Tracker) Reject(ctx context.Context, channelID, callID, by string) error {
	t.mu.Lock()
	call, ok := t.calls[callID]
	t.mu.Unlock()
	if !ok || call.channelID != channelID {
		return ErrCallNotFound
	}
	if err := call.reject(ctx); err != nil {
		return err
	}
	t.finish(callID, StateRejected, by, time.Now())
	return nil
}

// Ringing diz se a chamada está tocando (para testes e diagnóstico).
func (t *Tracker) Ringing(callID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.calls[callID]
	return ok
}

func (t *Tracker) finish(callID, state, rejectedBy string, at time.Time) {
	t.mu.Lock()
	call, ok := t.calls[callID]
	if ok {
		call.timer.Stop()
		delete(t.calls, callID)
	}
	sink := t.sink
	t.mu.Unlock()
	if !ok || sink == nil {
		return
	}
	go sink(context.Background(), Update{ChannelID: call.channelID, CallID: callID, State: state, RejectedBy: rejectedBy, At: at})
}

func (t *Tracker) drop(callID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if call, ok := t.calls[callID]; ok {
		call.timer.Stop()
		delete(t.calls, callID)
	}
}

func (t *Tracker) autoReject(ctx context.Context, channelID string) bool {
	if t.channels == nil {
		return false
	}
	channel, err := t.channels.FindByID(ctx, channelID)
	if err != nil || channel == nil {
		return false
	}
	settings := channel.GetAdvancedSettings()
	return settings != nil && settings.RejectCall
}

// MessageMetadata são as chaves de chamada que a mensagem leva além da
// identidade do remetente.
func MessageMetadata(callID string, video bool) map[string]string {
	meta := map[string]string{MetaCallID: callID, MetaState: StateRinging}
	if video {
		meta[MetaVideo] = "true"
	}
	return meta
}

// Content é o texto da mensagem de chamada: o que a lista de conversas e os
// consumidores de webhook veem; o admin desenha pelo metadata.
func Content(video bool) string {
	if video {
		return "Chamada de vídeo"
	}
	return "Chamada de voz"
}
