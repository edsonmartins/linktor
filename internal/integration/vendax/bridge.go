// Package vendax implementa o bridge (L0) entre o Linktor e o VendaX Core, integrados por NATS.
//
// Fronteira: no Linktor a conversa é sempre vendedor <-> cliente. O Agente de IA do VendaX NÃO é
// participante do canal — ele assiste/sugere ao vendedor apenas dentro do VendaX. Logo o vendedor é
// o usuário atribuído à conversa (Conversation.AssignedUserID) e mapeia para o vendorId do Core.
// Ver docs/vendax-integration/PLANO-integracao-linktor-vendax.md.
//
// L0 (spike de-risk): só texto, 1 canal, 1 vendedor por canal.
//
//	Inbound  (Linktor->Core): consome o evento interno linktor.events.message.received (JetStream
//	         durable) e publica tenant.{id}.linktor.inbound (NATS core pub/sub) como LinktorEnvelope.
//	Outbound (Core->Linktor): assina tenant.*.core.outbound (NATS core sub) e entrega no canal via
//	         SendMessageUseCase (reusa as validações de tenant/canal/destinatário do Linktor).
//
// O Core usa NATS core pub/sub (plain), não JetStream; por isso o inbound é publicado com core
// publish e o outbound é consumido com core subscribe. O consumo do evento interno do Linktor usa
// JetStream durável (não perde eventos). Habilitado por LINKTOR_VENDAX_BRIDGE_ENABLED.
package vendax

import (
	"context"
	"fmt"
	"sync"

	natsgo "github.com/nats-io/nats.go"

	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/repository"
	linktornats "github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
)

const (
	// durableInbound é o consumer JetStream próprio do bridge no stream LINKTOR_EVENTS.
	durableInbound = "vendax-bridge-inbound"
)

// Bridge liga o Linktor ao VendaX Core por NATS. Ver o doc do pacote.
type Bridge struct {
	nats             *linktornats.Client
	sendMessageUC    *usecase.SendMessageUseCase
	conversationRepo repository.ConversationRepository
	contactRepo      repository.ContactRepository
	channelRepo      repository.ChannelRepository

	outboundSub *natsgo.Subscription
	channelSub  *natsgo.Subscription
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// appliedVersion guarda a última versão de channel.config aplicada por tenant (idempotência).
	mu             sync.Mutex
	appliedVersion map[string]int
}

// NewBridge monta o bridge com a conexão NATS, o usecase de envio e os repositórios necessários
// para traduzir identidade (conversa/contato/canal) entre os dois lados.
func NewBridge(
	nats *linktornats.Client,
	sendMessageUC *usecase.SendMessageUseCase,
	conversationRepo repository.ConversationRepository,
	contactRepo repository.ContactRepository,
	channelRepo repository.ChannelRepository,
) *Bridge {
	return &Bridge{
		nats:             nats,
		sendMessageUC:    sendMessageUC,
		conversationRepo: conversationRepo,
		contactRepo:      contactRepo,
		channelRepo:      channelRepo,
		appliedVersion:   make(map[string]int),
	}
}

// Start liga os dois fluxos (inbound e outbound). Retorna erro se o wiring do NATS falhar.
// O ctx recebido controla o ciclo de vida; Stop() também cancela.
func (b *Bridge) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)
	if err := b.startInbound(ctx); err != nil {
		return fmt.Errorf("bridge inbound: %w", err)
	}
	if err := b.startOutbound(ctx); err != nil {
		return fmt.Errorf("bridge outbound: %w", err)
	}
	if err := b.startChannelConfig(ctx); err != nil {
		return fmt.Errorf("bridge channel.config: %w", err)
	}
	logger.Info("VendaX bridge iniciado (L0 texto + L1 channel.config)")
	return nil
}

// Stop encerra as subscrições e aguarda os loops terminarem.
func (b *Bridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	if b.outboundSub != nil {
		_ = b.outboundSub.Unsubscribe()
	}
	if b.channelSub != nil {
		_ = b.channelSub.Unsubscribe()
	}
	b.wg.Wait()
	logger.Info("VendaX bridge parado")
}
