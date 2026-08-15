package email

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

func canalDeEmail(host string, porta int) *entity.Channel {
	return &entity.Channel{
		ID:               "ch-email",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeEmail,
		Name:             "Alçada",
		Enabled:          true,
		Environment:      entity.ChannelEnvironmentProduction,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config: map[string]string{
			"provider":           "smtp",
			"from_email":         "alcada@squadx.dev",
			"smtp_host":          "smtpout.secureserver.net",
			"smtp_port":          "465",
			"smtp_encryption":    "tls",
			"imap_host":          host,
			"imap_port":          strconv.Itoa(porta),
			"imap_folder":        "INBOX",
			"imap_poll_interval": "30",
		},
		Credentials: map[string]string{
			"imap_username": "username",
			"imap_password": "password",
		},
	}
}

// Percorre o caminho inteiro: mensagem na caixa IMAP -> leitura -> publicação no
// mesmo formato que os outros canais usam. É daí para frente que contato,
// conversa e o evento message.received acontecem.
func TestManagerPublicaMensagemRecebida(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemMultipart)

	produtor := testutil.NewMockProducer()
	m := NewManager(testutil.NewMockChannelRepository(), produtor)
	canal := canalDeEmail(host, porta)

	m.rodada(context.Background(), canal, leitorDeTeste(host, porta, "INBOX"))

	if len(produtor.InboundMessages) != 1 {
		t.Fatalf("esperava 1 mensagem publicada, veio %d", len(produtor.InboundMessages))
	}
	pub := produtor.InboundMessages[0]

	if pub.ChannelID != canal.ID || pub.TenantID != canal.TenantID {
		t.Errorf("canal/tenant errados: %s / %s", pub.ChannelID, pub.TenantID)
	}
	if pub.ChannelType != string(entity.ChannelTypeEmail) {
		t.Errorf("channel_type = %q", pub.ChannelType)
	}
	// O remetente é o que identifica o contato lá na frente.
	if pub.Metadata["sender_id"] != "edson@squadx.dev" {
		t.Errorf("sender_id = %q", pub.Metadata["sender_id"])
	}
	if pub.Metadata["sender_name"] != "Edson Martins" {
		t.Errorf("sender_name = %q", pub.Metadata["sender_name"])
	}
	if pub.Metadata["subject"] != "Confirmação do orçamento" {
		t.Errorf("subject = %q", pub.Metadata["subject"])
	}
	// Message-ID como ExternalID é o que faz a reentrega virar descarte no
	// índice único (conversation_id, external_id).
	if pub.ExternalID != "resposta-123@squadx.dev" {
		t.Errorf("external_id = %q", pub.ExternalID)
	}
	if pub.Content == "" {
		t.Error("conteúdo vazio")
	}
}

func TestManagerIgnoraCanalSemIMAP(t *testing.T) {
	m := NewManager(testutil.NewMockChannelRepository(), testutil.NewMockProducer())
	m.baseCtx = context.Background()

	canal := canalDeEmail("127.0.0.1", 1)
	delete(canal.Config, "imap_host")

	if m.StartChannel(canal) {
		t.Fatal("canal sem imap_host não deveria iniciar recebimento")
	}
}

func TestManagerIgnoraCanalDesabilitado(t *testing.T) {
	host, porta := servidorIMAP(t)
	m := NewManager(testutil.NewMockChannelRepository(), testutil.NewMockProducer())
	m.baseCtx = context.Background()

	canal := canalDeEmail(host, porta)
	canal.Enabled = false

	if m.StartChannel(canal) {
		t.Fatal("canal desabilitado não deveria iniciar recebimento")
	}
}

func TestManagerIgnoraCanalSemCredencial(t *testing.T) {
	host, porta := servidorIMAP(t)
	m := NewManager(testutil.NewMockChannelRepository(), testutil.NewMockProducer())
	m.baseCtx = context.Background()

	canal := canalDeEmail(host, porta)
	delete(canal.Credentials, "imap_password")

	if m.StartChannel(canal) {
		t.Fatal("canal sem senha de IMAP não deveria iniciar recebimento")
	}
}

// Alterar a configuração do canal religa o polling em vez de acumular dois:
// era o cenário de trocar host/pasta/senha e o Linktor continuar lendo a caixa
// antiga até alguém reiniciar o backend.
func TestManagerReligaSemDuplicar(t *testing.T) {
	host, porta := servidorIMAP(t)
	m := NewManager(testutil.NewMockChannelRepository(), testutil.NewMockProducer())
	m.baseCtx = context.Background()
	canal := canalDeEmail(host, porta)

	if !m.StartChannel(canal) {
		t.Fatal("não iniciou")
	}
	if !m.StartChannel(canal) {
		t.Fatal("não religou")
	}

	m.mu.Lock()
	ativos := len(m.cancels)
	m.mu.Unlock()
	if ativos != 1 {
		t.Fatalf("esperava 1 polling ativo, há %d", ativos)
	}

	m.StopChannel(canal.ID)
	m.mu.Lock()
	ativos = len(m.cancels)
	m.mu.Unlock()
	if ativos != 0 {
		t.Fatalf("StopChannel não encerrou: %d ativos", ativos)
	}
}

// Start no boot só liga o que está apto — é o que evita um canal meio
// configurado derrubar a subida do servidor.
func TestManagerStartSelecionaCanaisAptos(t *testing.T) {
	host, porta := servidorIMAP(t)
	repo := testutil.NewMockChannelRepository()

	apto := canalDeEmail(host, porta)
	repo.Channels[apto.ID] = apto

	semIMAP := canalDeEmail(host, porta)
	semIMAP.ID = "ch-sem-imap"
	delete(semIMAP.Config, "imap_host")
	repo.Channels[semIMAP.ID] = semIMAP

	m := NewManager(repo, testutil.NewMockProducer())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	m.mu.Lock()
	ativos := len(m.cancels)
	m.mu.Unlock()
	if ativos != 1 {
		t.Fatalf("esperava 1 canal ativo, há %d", ativos)
	}

	// O cancelamento do contexto encerra tudo.
	cancel()
	time.Sleep(50 * time.Millisecond)
}
