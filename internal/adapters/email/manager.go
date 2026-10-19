package email

import (
	"context"
	"encoding/base64"
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

	// maxInlineBytes é quanto anexo em base64 ainda cabe dentro de uma
	// mensagem NATS (o broker aceita 1 MB por padrão). O que não foi
	// re-hospedado antes de publicar e não cabe perde os bytes: chega ao chat
	// só com nome e tipo, em vez de prender a caixa numa publicação que nunca
	// passa.
	maxInlineBytes = 600 << 10
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
	rehost      func(context.Context, *nats.InboundMessage)

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

// SetRehost registra o passo que move anexos inline (base64) para o object
// store antes da publicação. Sem ele o anexo viaja inteiro dentro da mensagem
// NATS, e o limite do broker derruba um email com um PDF comum.
func (m *Manager) SetRehost(fn func(context.Context, *nats.InboundMessage)) {
	m.rehost = fn
}

// Start amarra o gerenciador ao ctx e liga o polling de todo canal de e-mail
// apto (habilitado, conectado e com IMAP configurado).
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
// só envia, e o recebimento é por webhook do provedor. Canal desabilitado ou
// desconectado para: o recebimento rejeitaria tudo que ele lesse, e a leitura
// já teria marcado o email como lido na caixa.
func (m *Manager) StartChannel(canal *entity.Channel) bool {
	if canal == nil || canal.Type != entity.ChannelTypeEmail {
		return false
	}

	cfg := ConfigFromMap(mesclar(canal.Config, canal.Credentials))
	if !canal.Enabled || !canal.IsConnected() || cfg.IMAPHost == "" {
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
	_, err := leitor.Buscar(ctx, maxPorRodada, func(msg *IncomingEmail) error {
		return m.publicar(ctx, canal, msg)
	})
	if err != nil {
		// Falha aqui é ruidosa de propósito. A versão anterior engolia o erro
		// ("Log error but continue" sem log nenhum), e o resultado era um canal
		// que parecia saudável sem nunca receber nada.
		logger.Warn("email: falha ao ler a caixa do canal " + canal.ID + ": " + err.Error())
	}
}

// publicar entrega a mensagem ao mesmo pipeline dos demais canais — daí em
// diante contato, conversa e o evento message.received seguem iguais.
func (m *Manager) publicar(ctx context.Context, canal *entity.Channel, msg *IncomingEmail) error {
	if m.producer == nil {
		return nil
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

	entrada := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    canal.TenantID,
		ChannelID:   canal.ID,
		ChannelType: string(entity.ChannelTypeEmail),
		ExternalID:  externo,
		ContentType: "text",
		Content:     msg.Texto(),
		Environment: string(canal.Environment),
		Metadata:    metadados,
		Attachments: AnexosParaNATS(msg.Attachments),
		Timestamp:   recebido,
	}
	if m.rehost != nil {
		m.rehost(ctx, entrada)
	}
	descartarInlineExcedente(entrada.Attachments, maxInlineBytes)

	return m.producer.PublishInbound(ctx, entrada)
}

// AnexosParaNATS converte anexos de email no formato do pipeline. O tipo sai
// do MIME, e os bytes vão em base64 no metadata, que é o contrato que o passo
// de re-hospedagem já entende; sem URL nem bytes o anexo chegava ao chat vazio.
func AnexosParaNATS(anexos []*Attachment) []nats.AttachmentData {
	if len(anexos) == 0 {
		return nil
	}
	saida := make([]nats.AttachmentData, 0, len(anexos))
	for _, a := range anexos {
		ad := nats.AttachmentData{
			Type:      tipoDoAnexo(a.ContentType),
			URL:       a.URL,
			Filename:  a.Filename,
			MimeType:  a.ContentType,
			SizeBytes: a.Size,
		}
		if ad.URL == "" && len(a.Content) > 0 {
			ad.Metadata = map[string]string{
				"data":          base64.StdEncoding.EncodeToString(a.Content),
				"data_encoding": "base64",
			}
			if ad.SizeBytes == 0 {
				ad.SizeBytes = int64(len(a.Content))
			}
		}
		saida = append(saida, ad)
	}
	return saida
}

func tipoDoAnexo(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	}
	return "document"
}

// descartarInlineExcedente tira os bytes inline dos anexos que não cabem no
// orçamento, na ordem em que vieram. Quem perde os bytes mantém nome e tipo.
func descartarInlineExcedente(anexos []nats.AttachmentData, orcamento int) {
	restante := orcamento
	for i := range anexos {
		a := &anexos[i]
		dados := a.Metadata["data"]
		if dados == "" {
			continue
		}
		if len(dados) <= restante {
			restante -= len(dados)
			continue
		}
		logger.Warn("email: anexo " + a.Filename + " sem object store e grande demais para a fila; segue só o registro")
		delete(a.Metadata, "data")
		delete(a.Metadata, "data_encoding")
	}
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
