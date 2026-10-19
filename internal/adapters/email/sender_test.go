package email

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/outbound"
)

// The production send path must carry threading, HTML and Reply-To/CC/BCC so an
// agent reply lands in the recipient's existing thread as HTML — the old
// SendText path dropped all of these.
func TestBuildOutboundEmail_PreservesThreadingHTMLAndAddressing(t *testing.T) {
	msg := &outbound.Message{
		To:      "customer@example.com",
		Content: outbound.Text{Body: "reply text"},
		Metadata: map[string]string{
			"subject":     "Re: your question",
			"html_body":   "<p>reply text</p>",
			"in_reply_to": "<msg-1@mail.example.com>",
			"references":  "<root@mail.example.com> <msg-1@mail.example.com>",
			"reply_to":    "support@us.example.com",
			"cc":          "cc1@example.com, cc2@example.com",
			"bcc":         "audit@us.example.com",
		},
	}

	email := buildOutboundEmail(context.Background(), msg)

	assert.Equal(t, []string{"customer@example.com"}, email.To)
	assert.Equal(t, "Re: your question", email.Subject)
	assert.Equal(t, "reply text", email.TextBody)
	assert.Equal(t, "<p>reply text</p>", email.HTMLBody)
	assert.Equal(t, "<msg-1@mail.example.com>", email.InReplyTo)
	assert.Equal(t, "<root@mail.example.com> <msg-1@mail.example.com>", email.References)
	assert.Equal(t, "support@us.example.com", email.ReplyTo)
	assert.Equal(t, []string{"cc1@example.com", "cc2@example.com"}, email.CC)
	assert.Equal(t, []string{"audit@us.example.com"}, email.BCC)
}

// Media is downloaded and attached as real bytes (email providers only send
// attachment Content, never a bare URL).
func TestBuildOutboundEmail_MediaIsFetchedIntoAttachmentBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 fake bytes"))
	}))
	defer srv.Close()

	msg := &outbound.Message{
		To:       "customer@example.com",
		Content:  outbound.Media{URL: srv.URL + "/invoice.pdf", Filename: "invoice.pdf", Caption: "see attached"},
		Metadata: map[string]string{"mime_type": "application/pdf"},
	}

	email := buildOutboundEmail(context.Background(), msg)

	assert.Equal(t, "see attached", email.TextBody)
	require.Len(t, email.Attachments, 1)
	assert.Equal(t, "application/pdf", email.Attachments[0].ContentType)
	assert.Equal(t, []byte("%PDF-1.4 fake bytes"), email.Attachments[0].Content)
	assert.NotEmpty(t, email.Attachments[0].Filename)
}

// When the media can't be fetched, the link falls back into the body so the
// media is never silently lost, and no empty content-less attachment is created.
func TestBuildOutboundEmail_MediaFetchFailureFallsBackToBodyLink(t *testing.T) {
	msg := &outbound.Message{
		To:       "customer@example.com",
		Content:  outbound.Media{URL: "http://127.0.0.1:1/nope.pdf", Filename: "nope.pdf", Caption: "see attached"},
		Metadata: map[string]string{"mime_type": "application/pdf"},
	}

	email := buildOutboundEmail(context.Background(), msg)

	assert.Empty(t, email.Attachments)
	assert.Contains(t, email.TextBody, "see attached")
	assert.Contains(t, email.TextBody, "http://127.0.0.1:1/nope.pdf")
}

// provedorFixo devolve sempre o mesmo resultado: serve para testar como o
// sender traduz a resposta de um provedor HTTP.
type provedorFixo struct{ res *SendResult }

func (p provedorFixo) Send(context.Context, *OutboundEmail) (*SendResult, error) { return p.res, nil }
func (provedorFixo) ValidateConfig() error                                       { return nil }
func (provedorFixo) GetProviderName() string                                     { return "fixo" }
func (provedorFixo) TestConnection(context.Context) error                        { return nil }

// Recusa definitiva do provedor (token errado, destinatário não autorizado) vira
// erro permanente; antes era repetida até a fila desistir e a mensagem ficava
// "pendente" para sempre.
func TestSend_RecusaDefinitivaDoProvedorEPermanente(t *testing.T) {
	msg := &outbound.Message{To: "c@example.com", Content: outbound.Text{Body: "hi"}}

	cliente := func(res *SendResult) *Client { return &Client{provider: provedorFixo{res}, config: &Config{}} }

	definitiva := &emailSender{client: cliente(&SendResult{Error: "Mailgun API error (status 403)", Permanent: true})}
	_, err := definitiva.Send(context.Background(), msg)
	assert.True(t, outbound.IsPermanent(err), "%v", err)

	passageira := &emailSender{client: cliente(&SendResult{Error: "Mailgun API error (status 503)"})}
	_, err = passageira.Send(context.Background(), msg)
	assert.Error(t, err)
	assert.False(t, outbound.IsPermanent(err))
}

func TestRespostaDefinitiva(t *testing.T) {
	assert.True(t, respostaDefinitiva(401))
	assert.True(t, respostaDefinitiva(403))
	assert.False(t, respostaDefinitiva(429))
	assert.False(t, respostaDefinitiva(500))
	assert.False(t, respostaDefinitiva(202))
}

func TestTipoDoConteudo(t *testing.T) {
	assert.Equal(t, "image/png", tipoDoConteudo("image/png", "", "x.pdf"), "o da mídia vem primeiro")
	assert.Equal(t, "application/pdf", tipoDoConteudo("", "application/pdf", "x.png"))
	assert.Equal(t, "application/pdf", tipoDoConteudo("", "", "proposta.pdf"), "pela extensão")
	assert.Equal(t, "application/octet-stream", tipoDoConteudo("", "", "semextensao"))
}

// A plain text reply with no threading metadata still produces a valid text email
// (subject falls back to "Message").
func TestBuildOutboundEmail_PlainTextDefaults(t *testing.T) {
	msg := &outbound.Message{To: "c@example.com", Content: outbound.Text{Body: "hi"}}

	email := buildOutboundEmail(context.Background(), msg)

	assert.Equal(t, "Message", email.Subject)
	assert.Equal(t, "hi", email.TextBody)
	assert.Empty(t, email.HTMLBody)
	assert.Empty(t, email.InReplyTo)
	assert.Empty(t, email.Attachments)
}
