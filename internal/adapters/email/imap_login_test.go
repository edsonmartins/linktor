package email

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// servidorFalso responde a UM comando IMAP e devolve a linha recebida.
//
// net.Pipe é síncrono: o Write do cliente só retorna quando este lado lê, o
// que dispensa sincronização extra no teste.
func servidorFalso(t *testing.T, cli *IMAPClient, resposta string) <-chan string {
	t.Helper()

	servidor, cliente := net.Pipe()
	cli.conn = cliente
	t.Cleanup(func() { servidor.Close(); cliente.Close() })

	recebido := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		_ = servidor.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := servidor.Read(buf)
		if err != nil {
			recebido <- ""
			return
		}
		linha := string(buf[:n])
		recebido <- linha

		campos := strings.Fields(linha)
		if len(campos) == 0 {
			return
		}
		_, _ = servidor.Write([]byte(campos[0] + " OK " + resposta + "\r\n"))
	}()

	return recebido
}

// O caso real: o Google exibe a senha de app em quatro blocos separados por
// espaço. Sem aspas, o servidor lê "abcd" como senha e o resto como argumentos
// a mais — login recusado, sem pista do motivo.
func TestLoginCitaCredenciaisComEspaco(t *testing.T) {
	cli, err := NewIMAPClient(&Config{
		IMAPHost:     "imap.gmail.com",
		IMAPUsername: "canal@gmail.com",
		IMAPPassword: "abcd efgh ijkl mnop",
	})
	require.NoError(t, err)

	recebido := servidorFalso(t, cli, "LOGIN completed")
	require.NoError(t, cli.login())

	assert.Contains(t, <-recebido, `LOGIN "canal@gmail.com" "abcd efgh ijkl mnop"`)
}

// Pastas do Gmail têm espaço e colchete; sem aspas o SELECT seleciona outra
// coisa (ou falha), e o canal fica sem receber nada em silêncio.
func TestSelectFolderCitaPastaComEspaco(t *testing.T) {
	cli, err := NewIMAPClient(&Config{IMAPHost: "imap.gmail.com"})
	require.NoError(t, err)

	recebido := servidorFalso(t, cli, "[READ-WRITE] SELECT completed")
	require.NoError(t, cli.selectFolder("[Gmail]/Todos os e-mails"))

	assert.Contains(t, <-recebido, `SELECT "[Gmail]/Todos os e-mails"`)
}

func TestQuoteIMAPEscapaEspeciais(t *testing.T) {
	assert.Equal(t, `"INBOX"`, quoteIMAP("INBOX"))
	assert.Equal(t, `"a\"b"`, quoteIMAP(`a"b`))
	assert.Equal(t, `"a\\b"`, quoteIMAP(`a\b`))
}

// Aspas não protegem contra CR/LF: eles encerrariam a linha e o resto viraria
// um comando novo para o servidor.
func TestLoginRecusaQuebraDeLinha(t *testing.T) {
	cli, err := NewIMAPClient(&Config{
		IMAPHost:     "imap.gmail.com",
		IMAPUsername: "canal@gmail.com",
		IMAPPassword: "senha\r\nA1 DELETE INBOX",
	})
	require.NoError(t, err)

	servidor, cliente := net.Pipe()
	defer servidor.Close()
	defer cliente.Close()
	cli.conn = cliente

	// Nada deve ser escrito na conexão: o erro vem antes do primeiro Write.
	// (net.Pipe bloqueia no Write sem leitor, então um teste que passa aqui já
	// prova que nenhum comando foi enviado.)
	err = cli.login()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quebra de linha")
}
