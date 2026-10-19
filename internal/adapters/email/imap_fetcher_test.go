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
	"errors"
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
	mensagens := buscarTudo(t, leitor, 10)
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

	if primeira := buscarTudo(t, leitor, 10); len(primeira) != 1 {
		t.Fatalf("primeira rodada: %d mensagens", len(primeira))
	}

	segunda := buscarTudo(t, leitor, 10)
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
	mensagens := buscarTudo(t, leitor, 2)
	if len(mensagens) != 2 {
		t.Fatalf("limite de 2 não respeitado: veio %d", len(mensagens))
	}

	// O restante continua não lido e vem na rodada seguinte — nada se perde.
	resto := buscarTudo(t, leitor, 10)
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

// buscarTudo roda Buscar aceitando tudo e devolve o que foi entregue.
func buscarTudo(t *testing.T, leitor *Fetcher, limite int) []*IncomingEmail {
	t.Helper()
	var saida []*IncomingEmail
	if _, err := leitor.Buscar(context.Background(), limite, func(m *IncomingEmail) error {
		saida = append(saida, m)
		return nil
	}); err != nil {
		t.Fatalf("Buscar: %v", err)
	}
	return saida
}

const mensagemComAnexo = "From: \"Edson Martins\" <edson@squadx.dev>\r\n" +
	"To: alcada@squadx.dev\r\n" +
	"Subject: Proposta\r\n" +
	"Message-ID: <anexo-1@squadx.dev>\r\n" +
	"Date: Fri, 15 Aug 2026 10:00:00 +0000\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/mixed; boundary=\"mix\"\r\n" +
	"\r\n" +
	"--mix\r\n" +
	"Content-Type: text/plain; charset=UTF-8\r\n" +
	"\r\n" +
	"Segue o PDF.\r\n" +
	"--mix\r\n" +
	"Content-Type: application/pdf; name=\"proposta.pdf\"\r\n" +
	"Content-Disposition: attachment; filename=\"proposta.pdf\"\r\n" +
	"Content-Transfer-Encoding: base64\r\n" +
	"\r\n" +
	"JVBERi0xLjQgdGVzdGU=\r\n" +
	"--mix--\r\n"

const mensagemSoHTML = "From: \"Edson Martins\" <edson@squadx.dev>\r\n" +
	"To: alcada@squadx.dev\r\n" +
	"Subject: Newsletter\r\n" +
	"Message-ID: <html-1@squadx.dev>\r\n" +
	"Date: Fri, 15 Aug 2026 10:00:00 +0000\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: text/html; charset=UTF-8\r\n" +
	"\r\n" +
	"<html><head><style>p{margin:0}</style></head><body><p>Olá, <b>Alexandre</b>!</p>" +
	"<p>Segue o <a href=\"https://exemplo.com\">link</a>.</p></body></html>\r\n"

const mensagemComImagemInline = "From: \"Edson Martins\" <edson@squadx.dev>\r\n" +
	"To: alcada@squadx.dev\r\n" +
	"Subject: Assinatura\r\n" +
	"Message-ID: <inline-1@squadx.dev>\r\n" +
	"Date: Fri, 15 Aug 2026 10:00:00 +0000\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/related; boundary=\"rel\"\r\n" +
	"\r\n" +
	"--rel\r\n" +
	"Content-Type: text/html; charset=UTF-8\r\n" +
	"\r\n" +
	"<p>Assinatura:</p><img src=\"cid:logo\">\r\n" +
	"--rel\r\n" +
	"Content-Type: image/png; name=\"logo.png\"\r\n" +
	"Content-Disposition: inline; filename=\"logo.png\"\r\n" +
	"Content-ID: <logo>\r\n" +
	"Content-Transfer-Encoding: base64\r\n" +
	"\r\n" +
	"UE5HREFUQQ==\r\n" +
	"--rel--\r\n"

// Transfer-encoding desconhecida: o parser recusa a mensagem inteira.
const mensagemMalformada = "From: alguem@squadx.dev\r\n" +
	"To: alcada@squadx.dev\r\n" +
	"Subject: Quebrada\r\n" +
	"Message-ID: <quebrada-1@squadx.dev>\r\n" +
	"Content-Type: text/plain; charset=UTF-8\r\n" +
	"Content-Transfer-Encoding: nope\r\n" +
	"\r\n" +
	"corpo\r\n"

func TestFetcherLeAnexoComBytes(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemComAnexo)

	mensagens := buscarTudo(t, leitorDeTeste(host, porta, "INBOX"), 10)
	if len(mensagens) != 1 {
		t.Fatalf("esperava 1 mensagem, veio %d", len(mensagens))
	}
	m := mensagens[0]
	if strings.TrimSpace(m.TextBody) != "Segue o PDF." {
		t.Errorf("TextBody = %q", m.TextBody)
	}
	if len(m.Attachments) != 1 {
		t.Fatalf("esperava 1 anexo, veio %d", len(m.Attachments))
	}
	a := m.Attachments[0]
	if a.Filename != "proposta.pdf" || a.ContentType != "application/pdf" {
		t.Errorf("anexo = %q (%s)", a.Filename, a.ContentType)
	}
	// Antes só nome e tipo eram guardados; o arquivo nunca chegava ao chat.
	if string(a.Content) != "%PDF-1.4 teste" {
		t.Errorf("conteúdo do anexo = %q", a.Content)
	}
	if a.Size != int64(len("%PDF-1.4 teste")) {
		t.Errorf("Size = %d", a.Size)
	}
}

// Quem recusou a entrega (fila fora, por exemplo) recebe a mensagem de novo na
// rodada seguinte: ela só é marcada como lida depois de aceita.
func TestFetcherDeixaNaoLidaQuandoEntregaFalha(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemMultipart)
	leitor := leitorDeTeste(host, porta, "INBOX")

	entregues, err := leitor.Buscar(context.Background(), 10, func(*IncomingEmail) error {
		return errors.New("fila fora")
	})
	if err != nil || entregues != 0 {
		t.Fatalf("rodada com falha: entregues=%d err=%v", entregues, err)
	}

	if de_novo := buscarTudo(t, leitor, 10); len(de_novo) != 1 {
		t.Fatalf("a mensagem recusada deveria voltar: veio %d", len(de_novo))
	}
	if sobrou := buscarTudo(t, leitor, 10); len(sobrou) != 0 {
		t.Fatalf("depois de aceita não deveria voltar: veio %d", len(sobrou))
	}
}

// Uma mensagem que o parser não entende é marcada como lida mesmo sem entrega:
// com PEEK ela voltaria toda rodada e, no teto por rodada, travaria a fila.
func TestFetcherMarcaMalformadaComoLida(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemMalformada)
	depositar(t, host, porta, mensagemMultipart)
	leitor := leitorDeTeste(host, porta, "INBOX")

	if boas := buscarTudo(t, leitor, 10); len(boas) != 1 {
		t.Fatalf("esperava só a mensagem válida, veio %d", len(boas))
	}
	if sobrou := buscarTudo(t, leitor, 10); len(sobrou) != 0 {
		t.Fatalf("a malformada continua voltando (%d)", len(sobrou))
	}
}

// Imagem inline (assinatura com Content-ID) não é texto: antes os bytes dela
// viravam o corpo da mensagem.
func TestFetcherParteInlineNaoTextoViraAnexo(t *testing.T) {
	host, porta := servidorIMAP(t)
	depositar(t, host, porta, mensagemComImagemInline)

	mensagens := buscarTudo(t, leitorDeTeste(host, porta, "INBOX"), 10)
	if len(mensagens) != 1 {
		t.Fatalf("esperava 1 mensagem, veio %d", len(mensagens))
	}
	m := mensagens[0]
	if m.TextBody != "" {
		t.Errorf("imagem inline virou texto: %q", m.TextBody)
	}
	if !strings.Contains(m.HTMLBody, "cid:logo") {
		t.Errorf("HTMLBody = %q", m.HTMLBody)
	}
	if len(m.Attachments) != 1 {
		t.Fatalf("esperava 1 anexo, veio %d", len(m.Attachments))
	}
	a := m.Attachments[0]
	if a.ContentType != "image/png" || a.ContentID != "logo" || a.Filename != "logo.png" {
		t.Errorf("anexo = %+v", a)
	}
	if string(a.Content) != "PNGDATA" {
		t.Errorf("conteúdo = %q", a.Content)
	}
}
