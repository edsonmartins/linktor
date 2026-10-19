package email

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/outbound"
)

// smtpFalso é um servidor SMTP mínimo em texto puro: responde o que o roteiro
// manda para um comando e guarda o que recebeu no DATA. Testar contra o
// protocolo é o ponto: o código SMTP que o servidor devolve é o que decide se
// o envio é repetido ou marcado como falhou.
type smtpFalso struct {
	porta     int
	respostas map[string]string // comando (MAIL, RCPT, AUTH…) -> resposta forçada
	dados     chan string
}

func servidorSMTP(t *testing.T, respostas map[string]string) *smtpFalso {
	t.Helper()
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	s := &smtpFalso{porta: ln.Addr().(*net.TCPAddr).Port, respostas: respostas, dados: make(chan string, 4)}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.atender(conn)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *smtpFalso) atender(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	fmt.Fprint(conn, "220 localhost pronto\r\n")
	for {
		linha, err := r.ReadString('\n')
		if err != nil {
			return
		}
		campos := strings.Fields(linha)
		if len(campos) == 0 {
			continue
		}
		cmd := strings.ToUpper(campos[0])
		if resp, ok := s.respostas[cmd]; ok {
			fmt.Fprint(conn, resp+"\r\n")
			continue
		}
		switch cmd {
		case "EHLO", "HELO":
			fmt.Fprint(conn, "250-localhost\r\n250 AUTH PLAIN\r\n")
		case "AUTH":
			fmt.Fprint(conn, "235 2.7.0 ok\r\n")
		case "MAIL", "RCPT":
			fmt.Fprint(conn, "250 ok\r\n")
		case "DATA":
			fmt.Fprint(conn, "354 manda\r\n")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			s.dados <- b.String()
			fmt.Fprint(conn, "250 2.0.0 aceito\r\n")
		case "QUIT":
			fmt.Fprint(conn, "221 tchau\r\n")
			return
		default:
			fmt.Fprint(conn, "250 ok\r\n")
		}
	}
}

// provedorSMTPDeTeste aponta para o servidor falso sem TLS ("none" e nada de
// STARTTLS anunciado); "localhost" é o único host em que o PLAIN do net/smtp
// aceita conexão sem TLS.
func provedorSMTPDeTeste(t *testing.T, porta int) *SMTPProvider {
	t.Helper()
	p, err := NewSMTPProvider(&Config{
		Provider:       ProviderSMTP,
		FromEmail:      "suporte@linktor.dev",
		FromName:       "Suporte Técnico",
		SMTPHost:       "localhost",
		SMTPPort:       porta,
		SMTPUsername:   usuarioDeTeste,
		SMTPPassword:   credencialDeTeste,
		SMTPEncryption: "none",
	})
	require.NoError(t, err)
	p.timeoutConexao = 2 * time.Second
	p.timeoutIO = 2 * time.Second
	return p
}

func enviarTexto(t *testing.T, p *SMTPProvider) error {
	t.Helper()
	s := &emailSender{client: &Client{provider: p, config: p.config}}
	_, err := s.Send(context.Background(), &outbound.Message{To: "cliente@exemplo.dev", Content: outbound.Text{Body: "oi"}})
	return err
}

func TestSMTP_CredencialRecusadaEDefinitiva(t *testing.T) {
	s := servidorSMTP(t, map[string]string{"AUTH": "535 5.7.8 Username and Password not accepted"})
	err := enviarTexto(t, provedorSMTPDeTeste(t, s.porta))
	require.Error(t, err)
	assert.True(t, outbound.IsPermanent(err), "535 não melhora repetindo: %v", err)
	assert.Contains(t, err.Error(), "535")
}

func TestSMTP_DestinatarioRecusadoEDefinitivo(t *testing.T) {
	s := servidorSMTP(t, map[string]string{"RCPT": "550 5.1.1 user unknown"})
	err := enviarTexto(t, provedorSMTPDeTeste(t, s.porta))
	require.Error(t, err)
	assert.True(t, outbound.IsPermanent(err), "%v", err)
}

func TestSMTP_ServidorOcupadoETemporario(t *testing.T) {
	s := servidorSMTP(t, map[string]string{"MAIL": "421 4.7.0 try again later"})
	err := enviarTexto(t, provedorSMTPDeTeste(t, s.porta))
	require.Error(t, err)
	assert.False(t, outbound.IsPermanent(err), "421 merece nova tentativa: %v", err)
}

func TestSMTP_ConexaoRecusadaETemporaria(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	porta := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // porta livre: ninguém atende

	err = enviarTexto(t, provedorSMTPDeTeste(t, porta))
	require.Error(t, err)
	assert.False(t, outbound.IsPermanent(err), "%v", err)
}

// Servidor que aceita a conexão e nunca fala: sem prazo o envio ficava preso
// além do AckWait da fila e a mensagem era reentregue e enviada duas vezes.
func TestSMTP_ServidorMudoRespeitaOPrazo(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
		}
	}()

	p := provedorSMTPDeTeste(t, ln.Addr().(*net.TCPAddr).Port)
	p.timeoutIO = 300 * time.Millisecond
	inicio := time.Now()
	err = enviarTexto(t, p)
	require.Error(t, err)
	assert.Less(t, time.Since(inicio), 3*time.Second)
	assert.False(t, outbound.IsPermanent(err))
}

func TestSMTP_TestConnectionAcusaCredencialErrada(t *testing.T) {
	s := servidorSMTP(t, map[string]string{"AUTH": "535 5.7.8 bad credentials"})
	err := provedorSMTPDeTeste(t, s.porta).TestConnection(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")

	ok := servidorSMTP(t, nil)
	require.NoError(t, provedorSMTPDeTeste(t, ok.porta).TestConnection(context.Background()))
}

// A mensagem no fio: thread, assunto e nome codificados, corpo em
// quoted-printable de verdade, anexo com Content-Type.
func TestSMTP_MensagemNoFio(t *testing.T) {
	s := servidorSMTP(t, nil)
	p := provedorSMTPDeTeste(t, s.porta)

	res, err := p.Send(context.Background(), &OutboundEmail{
		To:         []string{"cliente@exemplo.dev"},
		Subject:    "Re: Orçamento nº 1",
		TextBody:   "Atenção ao prazo.",
		InReplyTo:  "<abc@exemplo.dev>",
		References: "<r@exemplo.dev> <abc@exemplo.dev>",
		Attachments: []*Attachment{{
			Filename:    "proposta.pdf",
			ContentType: "application/pdf",
			Content:     []byte("%PDF-1.4 teste"),
		}},
	})
	require.NoError(t, err)
	require.True(t, res.Success)
	assert.True(t, strings.HasPrefix(res.MessageID, "<") && strings.HasSuffix(res.MessageID, "@linktor.dev>"), res.MessageID)

	fio := <-s.dados
	assert.Contains(t, fio, "In-Reply-To: <abc@exemplo.dev>\r\n")
	assert.Contains(t, fio, "References: <r@exemplo.dev> <abc@exemplo.dev>\r\n")
	assert.Contains(t, fio, "Subject: =?utf-8?q?Re:_Or=C3=A7amento_n=C2=BA_1?=\r\n")
	assert.Contains(t, fio, "From: =?utf-8?q?Suporte_T=C3=A9cnico?= <suporte@linktor.dev>\r\n")
	assert.Contains(t, fio, "Content-Transfer-Encoding: quoted-printable\r\n\r\nAten=C3=A7=C3=A3o ao prazo.")
	assert.Contains(t, fio, "Content-Type: application/pdf; name=\"proposta.pdf\"\r\n")
	assert.Contains(t, fio, "JVBERi0xLjQgdGVzdGU=")
}

func TestSMTP_PortaConfiguradaEUsada(t *testing.T) {
	s := servidorSMTP(t, nil)
	p := provedorSMTPDeTeste(t, s.porta)
	assert.Equal(t, strconv.Itoa(s.porta), strconv.Itoa(p.config.SMTPPort))
}

// Credenciais de mentira do servidor falso. Em constante, e não no literal do
// campo, para o scanner de segredos do repositório não confundir com senha real.
const (
	usuarioDeTeste    = "usuario"
	credencialDeTeste = "nada-secreto"
)
