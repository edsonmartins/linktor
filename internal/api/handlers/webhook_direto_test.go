package handlers

import (
	"encoding/json"
	"testing"
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
