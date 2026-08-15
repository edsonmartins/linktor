package email

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
)

// servidorIMAP sobe um servidor IMAP de verdade (go-imap + backend em memória)
// sobre TLS com certificado gerado na hora.
//
// Testar contra servidor real é o ponto: o caminho anterior tinha testes de
// unidade passando enquanto, em produção, "retornava zero mensagens
// silenciosamente". Só o protocolo completo — LOGIN, SELECT, SEARCH, FETCH,
// STORE — mostra esse tipo de falha.
func servidorIMAP(t *testing.T) (host string, porta int) {
	t.Helper()

	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("chave: %v", err)
	}
	modelo := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &modelo, &modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("certificado: %v", err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: chave}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := server.New(memory.New())
	s.AllowInsecureAuth = true // a conexão já é TLS; isto libera LOGIN simples
	go s.Serve(ln)             //nolint:errcheck // encerra junto com o listener
	t.Cleanup(func() { ln.Close() })

	endereco := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", endereco.Port
}

func leitorDeTeste(host string, porta int, pasta string) *Fetcher {
	f, _ := NewFetcher(&Config{
		IMAPHost:     host,
		IMAPPort:     porta,
		IMAPUsername: "username",
		IMAPPassword: "password",
		IMAPFolder:   pasta,
	})
	f.inseguro = true
	return f
}

// depositar entrega uma mensagem na caixa, sem flags — ou seja, não lida.
func depositar(t *testing.T, host string, porta int, bruto string) {
	t.Helper()
	c, err := client.DialTLS(host+":"+strconv.Itoa(porta), &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	if err != nil {
		t.Fatalf("conectar para depositar: %v", err)
	}
	defer c.Logout()
	if err := c.Login("username", "password"); err != nil {
		t.Fatalf("login para depositar: %v", err)
	}
	if err := c.Append("INBOX", nil, time.Now(), bytes.NewBufferString(bruto)); err != nil {
		t.Fatalf("append: %v", err)
	}
}

const mensagemMultipart = "From: \"Edson Martins\" <edson@squadx.dev>\r\n" +
	"To: alcada@squadx.dev\r\n" +
	"Subject: =?UTF-8?Q?Confirma=C3=A7=C3=A3o_do_or=C3=A7amento?=\r\n" +
	"Message-ID: <resposta-123@squadx.dev>\r\n" +
	"In-Reply-To: <original-456@squadx.dev>\r\n" +
	"Date: Fri, 15 Aug 2026 10:00:00 +0000\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/alternative; boundary=\"limite\"\r\n" +
	"\r\n" +
	"--limite\r\n" +
	"Content-Type: text/plain; charset=UTF-8\r\n" +
	"Content-Transfer-Encoding: quoted-printable\r\n" +
	"\r\n" +
	"Pode seguir com a proposta. Aten=C3=A7=C3=A3o ao prazo.\r\n" +
	"--limite\r\n" +
	"Content-Type: text/html; charset=UTF-8\r\n" +
	"\r\n" +
	"<p>Pode seguir com a proposta.</p>\r\n" +
	"--limite--\r\n"

func TestFetcherLeMultipartComAcentoECabecalhoCodificado(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemMultipart)

	leitor := leitorDeTeste(host, porta, "INBOX")
	mensagens, err := leitor.Buscar(context.Background(), 10)
	if err != nil {
		t.Fatalf("Buscar: %v", err)
	}
	if len(mensagens) != 1 {
		t.Fatalf("esperava 1 mensagem não lida, veio %d", len(mensagens))
	}

	m := mensagens[0]
	if m.From != "edson@squadx.dev" {
		t.Errorf("From = %q", m.From)
	}
	if m.FromName != "Edson Martins" {
		t.Errorf("FromName = %q", m.FromName)
	}
	// Assunto em =?UTF-8?Q?...?= tem de chegar legível; era exatamente o tipo de
	// coisa que o parser artesanal entregava cru ou descartava.
	if m.Subject != "Confirmação do orçamento" {
		t.Errorf("Subject = %q", m.Subject)
	}
	if !strings.Contains(m.TextBody, "Atenção ao prazo") {
		t.Errorf("TextBody sem o texto decodificado: %q", m.TextBody)
	}
	if !strings.Contains(m.HTMLBody, "<p>") {
		t.Errorf("HTMLBody = %q", m.HTMLBody)
	}
	if m.MessageID != "resposta-123@squadx.dev" {
		t.Errorf("MessageID = %q", m.MessageID)
	}
	// É o que liga a resposta à mensagem original na conversa.
	if m.InReplyTo != "original-456@squadx.dev" {
		t.Errorf("InReplyTo = %q", m.InReplyTo)
	}
}

// A mesma mensagem não pode virar duas conversas a cada 30 segundos.
func TestFetcherNaoReentregaMensagemJaLida(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemMultipart)

	leitor := leitorDeTeste(host, porta, "INBOX")

	primeira, err := leitor.Buscar(context.Background(), 10)
	if err != nil || len(primeira) != 1 {
		t.Fatalf("primeira rodada: %d mensagens, err=%v", len(primeira), err)
	}

	segunda, err := leitor.Buscar(context.Background(), 10)
	if err != nil {
		t.Fatalf("segunda rodada: %v", err)
	}
	if len(segunda) != 0 {
		t.Fatalf("a mesma mensagem voltou na segunda rodada (%d)", len(segunda))
	}
}

func TestFetcherRespeitaOLimitePorRodada(t *testing.T) {
	host, porta := servidorIMAP(t)
	for i := 0; i < 5; i++ {
		depositar(t, host, porta, mensagemMultipart)
	}

	leitor := leitorDeTeste(host, porta, "INBOX")
	mensagens, err := leitor.Buscar(context.Background(), 2)
	if err != nil {
		t.Fatalf("Buscar: %v", err)
	}
	if len(mensagens) != 2 {
		t.Fatalf("limite de 2 não respeitado: veio %d", len(mensagens))
	}

	// O restante continua não lido e vem na rodada seguinte — nada se perde.
	resto, err := leitor.Buscar(context.Background(), 10)
	if err != nil {
		t.Fatalf("segunda rodada: %v", err)
	}
	if len(resto) != 3 {
		t.Fatalf("esperava as 3 restantes, veio %d", len(resto))
	}
}

// Pasta inexistente é o erro mais comum depois da credencial (marcador do Gmail
// digitado errado, por exemplo) e precisa aparecer como erro, não como silêncio.
func TestFetcherAcusaPastaInexistente(t *testing.T) {
	host, porta := servidorIMAP(t)
	leitor := leitorDeTeste(host, porta, "Nao/Existe")

	if _, err := leitor.Verificar(context.Background()); err == nil {
		t.Fatal("pasta inexistente deveria falhar")
	} else if !strings.Contains(err.Error(), "abrir pasta") {
		t.Fatalf("erro pouco específico: %v", err)
	}
}

func TestFetcherAcusaSenhaErrada(t *testing.T) {
	host, porta := servidorIMAP(t)
	leitor := leitorDeTeste(host, porta, "INBOX")
	leitor.senha = "errada"

	if _, err := leitor.Verificar(context.Background()); err == nil {
		t.Fatal("senha errada deveria falhar")
	} else if !strings.Contains(err.Error(), "login IMAP") {
		t.Fatalf("erro pouco específico: %v", err)
	}
}
