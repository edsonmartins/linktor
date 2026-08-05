package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
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
