package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset" // registra ISO-8859-1 e demais charsets
	"github.com/emersion/go-message/mail"

	"github.com/msgfy/linktor/pkg/logger"
)

// maxAnexoBytes limita o que se lê de cada anexo. É o teto de envio do Gmail;
// acima disso só nome e tipo seguem, e o arquivo continua na caixa.
const maxAnexoBytes = 25 << 20

// Fetcher lê mensagens novas de uma caixa por IMAP.
//
// Substitui o cliente artesanal deste pacote (imap.go), que montava comandos e
// parseava respostas na mão. Aquele caminho estava desligado por padrão com a
// observação de que "retorna zero mensagens silenciosamente" — e o motivo é
// estrutural: IMAP tem literais, continuação de linha e nomes de pasta em UTF-7
// modificado, e corpo de e-mail tem multipart, transfer-encoding e charsets que
// não são ASCII. Cada um desses é uma forma silenciosa de não receber nada.
//
// A biblioteca resolve os dois lados: go-imap fala o protocolo, go-message
// decodifica o MIME.
type Fetcher struct {
	host     string
	porta    int
	usuario  string
	senha    string
	pasta    string
	timeout  time.Duration
	inseguro bool // só para teste: aceita certificado autoassinado
}

// NewFetcher monta o leitor a partir da configuração do canal.
func NewFetcher(cfg *Config) (*Fetcher, error) {
	if cfg.IMAPHost == "" {
		return nil, fmt.Errorf("imap_host is required")
	}
	porta := cfg.IMAPPort
	if porta == 0 {
		porta = 993
	}
	pasta := cfg.IMAPFolder
	if pasta == "" {
		pasta = "INBOX"
	}
	return &Fetcher{
		host:    cfg.IMAPHost,
		porta:   porta,
		usuario: cfg.IMAPUsername,
		senha:   cfg.IMAPPassword,
		pasta:   pasta,
		timeout: 30 * time.Second,
	}, nil
}

func (f *Fetcher) conectar() (*client.Client, error) {
	endereco := fmt.Sprintf("%s:%d", f.host, f.porta)

	// 993 é TLS desde o primeiro byte. Não há STARTTLS aqui de propósito: todo
	// provedor relevante atende 993, e aceitar 143 abriria a porta para a senha
	// trafegar em claro se a configuração estiver errada.
	c, err := client.DialTLS(endereco, &tls.Config{
		ServerName:         f.host,
		InsecureSkipVerify: f.inseguro, //nolint:gosec // só habilitado em teste
	})
	if err != nil {
		return nil, fmt.Errorf("conexão IMAP em %s: %w", endereco, err)
	}
	c.Timeout = f.timeout

	if err := c.Login(f.usuario, f.senha); err != nil {
		c.Logout()
		return nil, fmt.Errorf("login IMAP de %s: %w", f.usuario, err)
	}
	return c, nil
}

// Verificar confirma que dá para conectar, autenticar e abrir a pasta.
// Devolve a quantidade de mensagens não lidas encontradas.
func (f *Fetcher) Verificar(ctx context.Context) (int, error) {
	c, err := f.conectar()
	if err != nil {
		return 0, err
	}
	defer c.Logout()

	if _, err := c.Select(f.pasta, true); err != nil {
		return 0, fmt.Errorf("abrir pasta %q: %w", f.pasta, err)
	}

	criterio := imap.NewSearchCriteria()
	criterio.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criterio)
	if err != nil {
		return 0, fmt.Errorf("buscar não lidas em %q: %w", f.pasta, err)
	}
	return len(ids), nil
}

// Buscar lê as mensagens não lidas da pasta e passa cada uma a entregar.
//
// Só a mensagem que entregar aceitou é marcada como lida; a que falhou fica
// não lida e volta na rodada seguinte. A leitura usa BODY.PEEK: sem isso o
// próprio FETCH marcava tudo como lido no servidor antes de qualquer entrega,
// e uma falha de publicação perdia o email.
//
// Mensagem que não dá para parsear também é marcada: com PEEK ela voltaria
// toda rodada e, com o teto por rodada, trancaria a fila atrás de si.
//
// Marcar como lida não é infalível — uma queda entre entregar e marcar traria
// a mensagem de novo — e por isso o Message-ID vai como ExternalID: o
// repositório tem índice único por (conversation_id, external_id) e descarta a
// repetição. Devolve quantas foram entregues.
func (f *Fetcher) Buscar(ctx context.Context, limite int, entregar func(*IncomingEmail) error) (int, error) {
	c, err := f.conectar()
	if err != nil {
		return 0, err
	}
	defer c.Logout()

	if _, err := c.Select(f.pasta, false); err != nil {
		return 0, fmt.Errorf("abrir pasta %q: %w", f.pasta, err)
	}

	criterio := imap.NewSearchCriteria()
	criterio.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criterio)
	if err != nil {
		return 0, fmt.Errorf("buscar não lidas em %q: %w", f.pasta, err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	if limite > 0 && len(ids) > limite {
		// Uma caixa com histórico acumulado não pode virar uma enxurrada de
		// conversas de uma vez; o resto vem na rodada seguinte.
		ids = ids[:limite]
	}

	conjunto := new(imap.SeqSet)
	conjunto.AddNum(ids...)

	secao := &imap.BodySectionName{Peek: true}
	mensagens := make(chan *imap.Message, len(ids))
	erroBusca := make(chan error, 1)
	go func() {
		erroBusca <- c.Fetch(conjunto, []imap.FetchItem{secao.FetchItem(), imap.FetchEnvelope}, mensagens)
	}()

	entregues := 0
	var processados []uint32
	for msg := range mensagens {
		if msg == nil {
			continue
		}
		corpo := msg.GetBody(secao)
		if corpo == nil {
			continue
		}
		email, err := parsearMensagem(corpo)
		if err != nil {
			logger.Warn("email: mensagem malformada em " + f.pasta + " marcada como lida sem entrega: " + err.Error())
			processados = append(processados, msg.SeqNum)
			continue
		}
		if email.MessageID == "" && msg.Envelope != nil {
			email.MessageID = msg.Envelope.MessageId
		}
		if err := entregar(email); err != nil {
			logger.Warn("email: mensagem " + email.MessageID + " fica não lida para nova tentativa: " + err.Error())
			continue
		}
		entregues++
		processados = append(processados, msg.SeqNum)
	}
	if err := <-erroBusca; err != nil {
		return entregues, fmt.Errorf("ler mensagens de %q: %w", f.pasta, err)
	}

	if len(processados) > 0 {
		marcar := new(imap.SeqSet)
		marcar.AddNum(processados...)
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		if err := c.Store(marcar, item, []interface{}{imap.SeenFlag}, nil); err != nil {
			// Entregar duas vezes é melhor que perder: o índice único por
			// external_id descarta a repetição lá na frente.
			return entregues, fmt.Errorf("marcar como lidas em %q: %w", f.pasta, err)
		}
	}

	return entregues, nil
}

// parsearMensagem converte o RFC 5322 bruto no formato interno, decodificando
// cabeçalhos codificados (=?UTF-8?...?=), multipart e transfer-encoding.
func parsearMensagem(r io.Reader) (*IncomingEmail, error) {
	leitor, err := mail.CreateReader(r)
	if err != nil {
		return nil, err
	}
	defer leitor.Close()

	cabecalho := leitor.Header
	email := &IncomingEmail{
		MessageID:  strings.Trim(primeiro(cabecalho.Get("Message-Id"), cabecalho.Get("Message-ID")), "<>"),
		Subject:    primeiro(textoOuVazio(cabecalho.Subject())),
		InReplyTo:  strings.Trim(cabecalho.Get("In-Reply-To"), "<>"),
		References: cabecalho.Get("References"),
	}
	if data, err := cabecalho.Date(); err == nil {
		email.ReceivedAt = data
	} else {
		email.ReceivedAt = time.Now()
	}

	if remetentes, err := cabecalho.AddressList("From"); err == nil && len(remetentes) > 0 {
		email.From = remetentes[0].Address
		email.FromName = remetentes[0].Name
	}
	email.To = enderecos(cabecalho, "To")
	email.CC = enderecos(cabecalho, "Cc")

	for {
		parte, err := leitor.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch cabecalhoParte := parte.Header.(type) {
		case *mail.InlineHeader:
			// go-message classifica pela disposição, não pelo tipo: uma imagem
			// inline (assinatura, Content-ID) chega aqui como InlineHeader. Só
			// texto vira corpo; o resto é anexo — antes os bytes da imagem
			// viravam o "texto" da mensagem.
			tipo, _, _ := cabecalhoParte.ContentType()
			switch {
			case strings.EqualFold(tipo, "text/html"):
				if email.HTMLBody == "" {
					email.HTMLBody = lerTexto(parte.Body)
				}
			case strings.HasPrefix(strings.ToLower(tipo), "text/"):
				if email.TextBody == "" {
					email.TextBody = lerTexto(parte.Body)
				}
			default:
				email.Attachments = append(email.Attachments,
					lerAnexo(parte.Body, nomeDaParte(cabecalhoParte.Header), tipo, cabecalhoParte.Get("Content-ID")))
			}
		case *mail.AttachmentHeader:
			nome, _ := cabecalhoParte.Filename()
			tipo, _, _ := cabecalhoParte.ContentType()
			email.Attachments = append(email.Attachments,
				lerAnexo(parte.Body, nome, tipo, cabecalhoParte.Get("Content-ID")))
		}
	}

	return email, nil
}

func lerTexto(r io.Reader) string {
	conteudo, err := io.ReadAll(io.LimitReader(r, maxAnexoBytes))
	if err != nil {
		return ""
	}
	return string(conteudo)
}

// lerAnexo lê os bytes do anexo. Acima do teto fica só o registro (nome e
// tipo), para o chat mostrar que o arquivo existe sem carregá-lo.
func lerAnexo(r io.Reader, nome, tipo, contentID string) *Attachment {
	anexo := &Attachment{
		Filename:    nome,
		ContentType: tipo,
		ContentID:   strings.Trim(contentID, "<>"),
	}
	dados, err := io.ReadAll(io.LimitReader(r, maxAnexoBytes+1))
	if err != nil || len(dados) > maxAnexoBytes {
		logger.Warn("email: anexo " + nome + " não carregado (erro de leitura ou acima de 25 MB)")
		return anexo
	}
	anexo.Content = dados
	anexo.Size = int64(len(dados))
	return anexo
}

// nomeDaParte tira o nome de uma parte inline do Content-Disposition ou, na
// falta dele, do parâmetro name do Content-Type.
func nomeDaParte(h message.Header) string {
	if _, params, err := h.ContentDisposition(); err == nil && params["filename"] != "" {
		return params["filename"]
	}
	if _, params, err := h.ContentType(); err == nil {
		return params["name"]
	}
	return ""
}

func enderecos(h mail.Header, campo string) []string {
	lista, err := h.AddressList(campo)
	if err != nil {
		return nil
	}
	var saida []string
	for _, e := range lista {
		saida = append(saida, e.Address)
	}
	return saida
}

func textoOuVazio(s string, err error) string {
	if err != nil {
		return ""
	}
	return s
}

func primeiro(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}
