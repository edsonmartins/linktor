package handlers

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// smtpFalso responde o mínimo de SMTP para chegar até o AUTH e registra as
// credenciais recebidas. É a única forma de provar que o teste de conexão
// autenticou de fato — do lado de fora, "conectou" e "autenticou" parecem
// iguais, e essa diferença é justamente o defeito que se está corrigindo.
type smtpFalso struct {
	ln          net.Listener
	autenticado chan string
}

func novoSMTPFalso(t *testing.T) *smtpFalso {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &smtpFalso{ln: ln, autenticado: make(chan string, 1)}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.atender(conn)
		}
	}()
	return s
}

func (s *smtpFalso) atender(conn net.Conn) {
	defer conn.Close()
	leitor := bufio.NewReader(conn)
	fmt.Fprint(conn, "220 falso ESMTP\r\n")

	for {
		linha, err := leitor.ReadString('\n')
		if err != nil {
			return
		}
		linha = strings.TrimRight(linha, "\r\n")

		switch {
		case strings.HasPrefix(linha, "EHLO"), strings.HasPrefix(linha, "HELO"):
			fmt.Fprint(conn, "250-falso\r\n250 AUTH PLAIN\r\n")
		case strings.HasPrefix(linha, "AUTH PLAIN "):
			bruto, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(linha, "AUTH PLAIN "))
			// O payload do PLAIN é "\x00usuário\x00senha".
			partes := strings.Split(string(bruto), "\x00")
			if len(partes) == 3 {
				select {
				case s.autenticado <- partes[1] + "/" + partes[2]:
				default:
				}
			}
			fmt.Fprint(conn, "235 2.7.0 Accepted\r\n")
		case strings.HasPrefix(linha, "QUIT"):
			fmt.Fprint(conn, "221 Bye\r\n")
			return
		default:
			fmt.Fprint(conn, "250 OK\r\n")
		}
	}
}

func (s *smtpFalso) hostPorta(t *testing.T) (string, string) {
	t.Helper()
	host, porta, err := net.SplitHostPort(s.ln.Addr().String())
	if err != nil {
		t.Fatalf("addr: %v", err)
	}
	if _, err := strconv.Atoi(porta); err != nil {
		t.Fatalf("porta inválida: %v", err)
	}
	return host, porta
}

// Reproduz o caminho real: o formulário de edição não reexibe a senha guardada,
// então ele envia o campo em branco. Antes, o backend testava só o que chegava —
// e como provedores só autenticam quando há usuário E senha, o teste passava sem
// nunca ter autenticado, aprovando um canal que poderia não enviar nada.
func TestTestEmailConnection_CompletaSegredoGuardado(t *testing.T) {
	servidor := novoSMTPFalso(t)
	host, porta := servidor.hostPorta(t)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-1", "tenant-1", "Alçada", entity.ChannelTypeEmail)
	canal.Config = map[string]string{
		"provider":        "smtp",
		"from_email":      "alcada@exemplo.org",
		"smtp_host":       host,
		"smtp_port":       porta,
		"smtp_encryption": "none",
	}
	canal.Credentials = map[string]string{
		"smtp_username": "conta@gmail.com",
		"smtp_password": "senha-guardada",
	}

	corpo, _ := json.Marshal(map[string]interface{}{
		"channel_id": "ch-1",
		"type":       "email",
		"config": map[string]string{
			"provider":        "smtp",
			"from_email":      "alcada@exemplo.org",
			"smtp_host":       host,
			"smtp_port":       porta,
			"smtp_encryption": "none",
		},
		// Como no formulário reaberto: usuário visível, senha em branco.
		"credentials": map[string]string{
			"smtp_username": "conta@gmail.com",
			"smtp_password": "",
		},
	})

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-email", corpo)
	handler.TestEmailConnection(c)

	if w.Code != 200 {
		t.Fatalf("esperado 200, veio %d: %s", w.Code, w.Body.String())
	}

	select {
	case usado := <-servidor.autenticado:
		if usado != "conta@gmail.com/senha-guardada" {
			t.Fatalf("autenticou com %q; esperado a credencial guardada", usado)
		}
	default:
		t.Fatal("o teste passou sem autenticar — é o falso positivo que se quer eliminar")
	}
}

// O valor digitado tem de vencer o guardado: é assim que se testa uma senha
// nova antes de salvar.
func TestTestEmailConnection_DigitadoVenceGuardado(t *testing.T) {
	servidor := novoSMTPFalso(t)
	host, porta := servidor.hostPorta(t)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-1", "tenant-1", "Alçada", entity.ChannelTypeEmail)
	canal.Config = map[string]string{
		"provider": "smtp", "from_email": "alcada@exemplo.org",
		"smtp_host": host, "smtp_port": porta, "smtp_encryption": "none",
	}
	canal.Credentials = map[string]string{
		"smtp_username": "conta@gmail.com",
		"smtp_password": "senha-antiga",
	}

	corpo, _ := json.Marshal(map[string]interface{}{
		"channel_id": "ch-1",
		"type":       "email",
		"config": map[string]string{
			"provider": "smtp", "from_email": "alcada@exemplo.org",
			"smtp_host": host, "smtp_port": porta, "smtp_encryption": "none",
		},
		"credentials": map[string]string{
			"smtp_username": "conta@gmail.com",
			"smtp_password": "senha-nova",
		},
	})

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-email", corpo)
	handler.TestEmailConnection(c)

	if w.Code != 200 {
		t.Fatalf("esperado 200, veio %d: %s", w.Code, w.Body.String())
	}
	if usado := <-servidor.autenticado; usado != "conta@gmail.com/senha-nova" {
		t.Fatalf("autenticou com %q; esperado a senha digitada", usado)
	}
}

// GetByID não recebe tenant: sem a comparação explícita no handler, um
// channel_id de outra conta viraria um oráculo de teste sobre credenciais
// alheias.
func TestTestEmailConnection_RecusaCanalDeOutroTenant(t *testing.T) {
	servidor := novoSMTPFalso(t)
	host, porta := servidor.hostPorta(t)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-alheio", "tenant-2", "De outro", entity.ChannelTypeEmail)
	canal.Config = map[string]string{
		"provider": "smtp", "from_email": "outro@exemplo.org",
		"smtp_host": host, "smtp_port": porta, "smtp_encryption": "none",
	}
	canal.Credentials = map[string]string{
		"smtp_username": "alheio@gmail.com",
		"smtp_password": "segredo-alheio",
	}

	corpo, _ := json.Marshal(map[string]interface{}{
		"channel_id": "ch-alheio",
		"type":       "email",
		"config": map[string]string{
			"provider": "smtp", "from_email": "outro@exemplo.org",
			"smtp_host": host, "smtp_port": porta, "smtp_encryption": "none",
		},
		"credentials": map[string]string{},
	})

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-email", corpo)
	handler.TestEmailConnection(c)

	if w.Code != 404 {
		t.Fatalf("esperado 404 para canal de outro tenant, veio %d: %s", w.Code, w.Body.String())
	}
	select {
	case usado := <-servidor.autenticado:
		t.Fatalf("credencial alheia foi usada: %q", usado)
	default:
	}
}

// O usuário identifica a conta e não é segredo; sem devolvê-lo, o formulário de
// edição abre em branco e não há como conferir com que conta o canal ficou
// gravado — foi o que fez parecer que alterar o usuário "não salvava".
func TestChannelMarshalJSON_ExpoeUsuarioMasNaoSenha(t *testing.T) {
	canal := &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeEmail, Name: "Alçada",
		Config: map[string]string{"smtp_host": "smtp.gmail.com"},
		Credentials: map[string]string{
			"smtp_username": "conta@gmail.com",
			"smtp_password": "senha-secreta",
			"imap_username": "conta@gmail.com",
			"imap_password": "senha-secreta",
		},
	}

	bruto, err := json.Marshal(canal)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	saida := string(bruto)

	for _, esperado := range []string{`"public_credentials"`, `"smtp_username":"conta@gmail.com"`, `"imap_username":"conta@gmail.com"`} {
		if !strings.Contains(saida, esperado) {
			t.Errorf("faltou %s em: %s", esperado, saida)
		}
	}
	if strings.Contains(saida, "senha-secreta") || strings.Contains(saida, "smtp_password") {
		t.Fatalf("senha vazou na serialização: %s", saida)
	}
}
