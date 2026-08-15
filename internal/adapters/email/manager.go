package email

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
)

const (
	// Piso do intervalo: a caixa é consultada por conexão nova a cada rodada, e
	// provedor de e-mail costuma limitar conexões por minuto.
	intervaloMinimo = 15 * time.Second
	intervaloPadrao = 30 * time.Second

	// Teto por rodada. Uma caixa com histórico não lido viraria centenas de
	// conversas de uma vez; o excedente vem nas rodadas seguintes.
	maxPorRodada = 25
)

// Manager mantém o recebimento por IMAP de cada canal de e-mail configurado.
//
// Antes disto, o lado de recebimento não existia em produção: o adaptador de
// e-mail era registrado no boot mas nunca inicializado por canal, e
// ChannelService.Connect só instancia adaptador para WhatsApp — para os demais
// tipos ele apenas grava connection_status=connected. O canal aparecia
// "conectado", enviava normalmente, e toda resposta era descartada em silêncio.
type Manager struct {
	channelRepo repository.ChannelRepository
	producer    nats.Publisher

	mu      sync.Mutex
	baseCtx context.Context
	cancels map[string]context.CancelFunc
}

func NewManager(channelRepo repository.ChannelRepository, producer nats.Publisher) *Manager {
	return &Manager{
		channelRepo: channelRepo,
		producer:    producer,
		cancels:     map[string]context.CancelFunc{},
	}
}

// Start amarra o gerenciador ao ctx e liga o polling de todo canal de e-mail
// habilitado que tenha IMAP configurado.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	m.baseCtx = ctx
	m.mu.Unlock()

	canais, err := m.channelRepo.FindByTypes(ctx, []entity.ChannelType{entity.ChannelTypeEmail})
	if err != nil {
		return err
	}

	ligados := 0
	for _, canal := range canais {
		if m.StartChannel(canal) {
			ligados++
		}
	}
	if ligados > 0 {
		logger.Info("email: recebimento por IMAP ativo", zap.Int("canais", ligados))
	}
	return nil
}

// StartChannel liga (ou religa) o recebimento de um canal. É idempotente: um
// polling anterior do mesmo canal é encerrado antes.
//
// Devolve true quando ficou rodando. Canal sem IMAP configurado não é erro —
// só envia, e o recebimento é por webhook do provedor.
func (m *Manager) StartChannel(canal *entity.Channel) bool {
	if canal == nil || canal.Type != entity.ChannelTypeEmail {
		return false
	}

	cfg := ConfigFromMap(mesclar(canal.Config, canal.Credentials))
	if !canal.Enabled || cfg.IMAPHost == "" {
		m.StopChannel(canal.ID)
		return false
	}
	if cfg.IMAPUsername == "" || cfg.IMAPPassword == "" {
		logger.Warn("email: canal " + canal.ID + " tem imap_host sem usuário/senha; recebimento desligado")
		m.StopChannel(canal.ID)
		return false
	}

	leitor, err := NewFetcher(cfg)
	if err != nil {
		logger.Warn("email: canal " + canal.ID + " com IMAP inválido: " + err.Error())
		return false
	}

	intervalo := time.Duration(cfg.IMAPPollInterval) * time.Second
	if intervalo < intervaloMinimo {
		intervalo = intervaloPadrao
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.baseCtx == nil {
		return false // ainda não subiu; o Start do boot pega este canal
	}
	m.pararTravado(canal.ID)

	ctx, cancel := context.WithCancel(m.baseCtx)
	m.cancels[canal.ID] = cancel
	go m.rodar(ctx, canal, leitor, intervalo)
	return true
}

// StopChannel encerra o recebimento de um canal, se estiver rodando.
func (m *Manager) StopChannel(canalID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pararTravado(canalID)
}

func (m *Manager) pararTravado(canalID string) {
	if cancel, ok := m.cancels[canalID]; ok {
		cancel()
		delete(m.cancels, canalID)
	}
}

// rodar consulta a caixa a cada intervalo até o ctx ser cancelado.
func (m *Manager) rodar(ctx context.Context, canal *entity.Channel, leitor *Fetcher, intervalo time.Duration) {
	logger.Info("email: recebimento iniciado para o canal " + canal.ID + " em " + leitor.host + "/" + leitor.pasta)

	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()

	m.rodada(ctx, canal, leitor)
	for {
		select {
		case <-ctx.Done():
			logger.Info("email: recebimento encerrado para o canal " + canal.ID)
			return
		case <-ticker.C:
			m.rodada(ctx, canal, leitor)
		}
	}
}

func (m *Manager) rodada(ctx context.Context, canal *entity.Channel, leitor *Fetcher) {
	mensagens, err := leitor.Buscar(ctx, maxPorRodada)
	if err != nil {
		// Falha aqui é ruidosa de propósito. A versão anterior engolia o erro
		// ("Log error but continue" sem log nenhum), e o resultado era um canal
		// que parecia saudável sem nunca receber nada.
		logger.Warn("email: falha ao ler a caixa do canal " + canal.ID + ": " + err.Error())
		if len(mensagens) == 0 {
			return
		}
	}

	for _, msg := range mensagens {
		if err := m.publicar(ctx, canal, msg); err != nil {
			logger.Warn("email: falha ao publicar mensagem do canal " + canal.ID + ": " + err.Error())
		}
	}
}

// publicar entrega a mensagem ao mesmo pipeline dos demais canais — daí em
// diante contato, conversa e o evento message.received seguem iguais.
func (m *Manager) publicar(ctx context.Context, canal *entity.Channel, msg *IncomingEmail) error {
	if m.producer == nil {
		return nil
	}

	conteudo := msg.TextBody
	tipo := "text"
	if strings.TrimSpace(conteudo) == "" && msg.HTMLBody != "" {
		conteudo = msg.HTMLBody
		tipo = "html"
	}

	metadados := map[string]string{
		"sender_id":   msg.From,
		"sender_name": primeiro(msg.FromName, msg.From),
		"subject":     msg.Subject,
	}
	if len(msg.To) > 0 {
		metadados["to"] = strings.Join(msg.To, ",")
	}
	if len(msg.CC) > 0 {
		metadados["cc"] = strings.Join(msg.CC, ",")
	}
	// Guardados para correlacionar a resposta com a mensagem original.
	if msg.InReplyTo != "" {
		metadados["in_reply_to"] = msg.InReplyTo
	}
	if msg.References != "" {
		metadados["references"] = msg.References
	}

	var anexos []nats.AttachmentData
	for _, a := range msg.Attachments {
		anexos = append(anexos, nats.AttachmentData{
			Type:     "file",
			Filename: a.Filename,
			MimeType: a.ContentType,
		})
	}

	recebido := msg.ReceivedAt
	if recebido.IsZero() {
		recebido = time.Now()
	}

	// ExternalID = Message-ID: o repositório tem índice único por
	// (conversation_id, external_id), então uma reentrega vira descarte em vez
	// de mensagem duplicada na conversa.
	externo := msg.MessageID
	if externo == "" {
		externo = uuid.New().String()
	}

	return m.producer.PublishInbound(ctx, &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    canal.TenantID,
		ChannelID:   canal.ID,
		ChannelType: string(entity.ChannelTypeEmail),
		ExternalID:  externo,
		ContentType: tipo,
		Content:     conteudo,
		Environment: string(canal.Environment),
		Metadata:    metadados,
		Attachments: anexos,
		Timestamp:   recebido,
	})
}

// mesclar junta config e credenciais num mapa só, que é o formato que
// ConfigFromMap espera. Credencial vence em caso de chave repetida.
func mesclar(config, credenciais map[string]string) map[string]string {
	junto := make(map[string]string, len(config)+len(credenciais))
	for k, v := range config {
		junto[k] = v
	}
	for k, v := range credenciais {
		junto[k] = v
	}
	return junto
}
