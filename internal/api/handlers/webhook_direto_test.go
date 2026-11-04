package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// Teste de CONTRATO do conector Direto (RFC-009): trava o formato + HMAC contra o MESMO vetor
// "golden" verificado no lado Direto (core-cliente ChannelContractTest.java). Se qualquer ponta
// mudar o formato/HMAC, um dos dois testes quebra.
//
//	SECRET = e2e-secret-01
//	BODY   = {"instanceId":"inst-rq-01","channel":"direto","messages":[{"id":"e2e-1",
//	          "from":"+5511999990001","type":"text","text":{"body":"oi de volta"}}]}
//	SIG    = sha256=009d8348a21a8660e134613c5b3e1de7bee1e90a6949556faea16eddfdd085aa
const (
	goldenSecret = "e2e-secret-01"
	goldenBody   = `{"instanceId":"inst-rq-01","channel":"direto","messages":[{"id":"e2e-1","from":"+5511999990001","type":"text","text":{"body":"oi de volta"}}]}`
	goldenSig    = "sha256=009d8348a21a8660e134613c5b3e1de7bee1e90a6949556faea16eddfdd085aa"
)

// O Linktor DEVE aceitar a assinatura que o Direto produz (mesmo formato sha256=<hex> do WhatsApp).
func TestDiretoWebhookAcceptsGoldenSignature(t *testing.T) {
	h := &WebhookHandler{}
	if !h.verifyWhatsAppSignature([]byte(goldenBody), goldenSig, goldenSecret) {
		t.Fatal("Linktor rejeitou a assinatura golden do Direto — HMAC incompatível")
	}
	// sanity: um secret errado NÃO deve validar.
	if h.verifyWhatsAppSignature([]byte(goldenBody), goldenSig, "outro-secret") {
		t.Fatal("assinatura validou com secret errado")
	}
}

// O Linktor DEVE parsear o payload que o Direto envia, nos campos que usa no inbound.
func TestDiretoWebhookParsesGoldenPayload(t *testing.T) {
	var p diretoWebhookPayload
	if err := json.Unmarshal([]byte(goldenBody), &p); err != nil {
		t.Fatalf("Linktor não parseou o payload golden: %v", err)
	}
	if p.InstanceID != "inst-rq-01" {
		t.Errorf("instanceId: got %q", p.InstanceID)
	}
	if len(p.Messages) != 1 {
		t.Fatalf("messages: got %d", len(p.Messages))
	}
	m := p.Messages[0]
	if m.ID != "e2e-1" || m.From != "+5511999990001" || m.Type != "text" {
		t.Errorf("message meta: %+v", m)
	}
	if m.Text == nil || m.Text.Body != "oi de volta" {
		t.Errorf("text body: %+v", m.Text)
	}
}

// Contrato dos EVENTOS efêmeros (RFC-009): o Linktor DEVE parsear o payload de typing/presença que
// o Direto envia (ChannelSignalDispatcher.buildPayload → {instanceId,channel,events:[{type,from,state}]}).
func TestDiretoWebhookParsesEvents(t *testing.T) {
	const body = `{"instanceId":"inst-rq-01","channel":"direto","events":[{"type":"typing","from":"+5511999990001","state":"on"}]}`
	var p diretoWebhookPayload
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("não parseou o payload de eventos: %v", err)
	}
	if len(p.Events) != 1 {
		t.Fatalf("events: got %d", len(p.Events))
	}
	e := p.Events[0]
	if e.Type != "typing" || e.From != "+5511999990001" || e.State != "on" {
		t.Errorf("event: %+v", e)
	}
}

// Sem as deps de typing (SetTypingDeps não chamado), processDiretoEvent é no-op seguro (não panica).
func TestProcessDiretoEventNoopWithoutDeps(t *testing.T) {
	h := &WebhookHandler{}
	h.processDiretoEvent(context.Background(), &entity.Channel{TenantID: "t"},
		diretoWebhookEvent{Type: "typing", From: "+5511999990001", State: "on"})
}

// O telefone do Direto (E.164 com "+") vira o formato canônico do Linktor, o
// mesmo do envio direto: sem isso a resposta do cliente abria outro contato.
func TestDiretoPhoneIsCanonical(t *testing.T) {
	for in, want := range map[string]string{
		"+5544997239232": "5544997239232",
		"5544997239232":  "5544997239232",
	} {
		if got := diretoPhone(in); got != want {
			t.Errorf("diretoPhone(%q) = %q, want %q", in, got, want)
		}
	}
}

// postDiretoJSON assina o corpo com X-Direto-Signature-256 e chama o handler.
func postDiretoJSON(t *testing.T, h *WebhookHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/direto/ch-d", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Direto-Signature-256", computeHMACSHA256("sek", []byte(body)))
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-d"}}
	h.DiretoWebhook(c)
	return w
}

func newDiretoWebhookHandler(t *testing.T) (*WebhookHandler, *testutil.MockProducer) {
	t.Helper()
	h, channels, producer, _ := setupWebhookTest()
	channels.Channels["ch-d"] = &entity.Channel{ID: "ch-d", TenantID: "tenant-1", Type: entity.ChannelTypeDireto,
		Credentials: map[string]string{"webhook_secret": "sek"}}
	return h, producer
}

// Mídia: a URL baixável vence o ref do Tinode, o tipo vem do MIME (o "file"
// do Direto vira document) e o tamanho segue; sem URL, cai no ref.
func TestDiretoWebhookMedia(t *testing.T) {
	h, producer := newDiretoWebhookHandler(t)
	body := `{"instanceId":"inst","channel":"direto","messages":[` +
		`{"id":"m1","from":"+5544997239232","type":"file","media":{"mime":"application/pdf","name":"contrato.pdf","url":"https://direto.example/channel/v1/inst/media/x.pdf","ref":"/v0/file/s/x.pdf","size":2048}},` +
		`{"id":"m2","from":"+5544997239232","type":"audio","media":{"mime":"audio/mp4","name":"a.m4a","ref":"/v0/file/s/a.mp4"},"caption":"ouve isso"}]}`
	w := postDiretoJSON(t, h, body)
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 2)

	pdf := producer.InboundMessages[0]
	require.Equal(t, "document", pdf.ContentType)
	require.Equal(t, "5544997239232", pdf.Metadata["sender_id"])
	require.Len(t, pdf.Attachments, 1)
	att := pdf.Attachments[0]
	require.Equal(t, "document", att.Type)
	require.Equal(t, "https://direto.example/channel/v1/inst/media/x.pdf", att.URL)
	require.Equal(t, int64(2048), att.SizeBytes)
	require.Equal(t, "/v0/file/s/x.pdf", att.Metadata["tinode_ref"])

	audio := producer.InboundMessages[1]
	require.Equal(t, "audio", audio.ContentType)
	require.Equal(t, "ouve isso", audio.Content)
	require.Equal(t, "/v0/file/s/a.mp4", audio.Attachments[0].URL)
}

// Recibos viram StatusUpdate pelo external_id; status desconhecido ou sem id
// é ignorado.
func TestDiretoWebhookStatuses(t *testing.T) {
	h, producer := newDiretoWebhookHandler(t)
	body := `{"instanceId":"inst","channel":"direto","statuses":[` +
		`{"id":"out-1","status":"delivered","recipient":"+5544997239232","timestamp":1800000000},` +
		`{"id":"out-2","status":"read"},{"id":"out-3","status":"sent"},{"status":"read"}]}`
	w := postDiretoJSON(t, h, body)
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.StatusUpdates, 2)
	require.Equal(t, "out-1", producer.StatusUpdates[0].ExternalID)
	require.Equal(t, "delivered", producer.StatusUpdates[0].Status)
	require.Equal(t, "direto", producer.StatusUpdates[0].ChannelType)
	require.Equal(t, int64(1800000000), producer.StatusUpdates[0].Timestamp.Unix())
	require.Equal(t, "read", producer.StatusUpdates[1].Status)
	require.False(t, producer.StatusUpdates[1].Timestamp.IsZero())
}
