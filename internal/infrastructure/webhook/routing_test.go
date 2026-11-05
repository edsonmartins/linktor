package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P5 (VCR-36): o inbound de um contato vai ao dono da conversa aberta; sem
// regra, ao consumidor padrão, exatamente como antes; na última tentativa ao
// dono, ao padrão com routing.fallback=true. Nada é descartado (ADR-025 §6).

type fakeOwners struct {
	rule *entity.ConversationOwnerRule
	err  error
	keys []string
}

func (f *fakeOwners) Find(_ context.Context, _, _, key string) (*entity.ConversationOwnerRule, error) {
	f.keys = append(f.keys, key)
	return f.rule, f.err
}

type fakeIdentities struct{ ids []*entity.ContactIdentity }

func (f *fakeIdentities) FindIdentitiesByContact(_ context.Context, _ string) ([]*entity.ContactIdentity, error) {
	return f.ids, nil
}

var agoraFixo = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

func canalComCopilot() *entity.Channel {
	return &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		WebhookURL:  "https://crm.exemplo.test/webhook",
		Credentials: map[string]string{credWebhookSecret: "segredo-padrao", "consumers.copilot.webhook_secret": "segredo-copilot"},
		Config:      map[string]string{"consumers.copilot.webhook_url": "https://copilot.exemplo.test/webhook"},
	}
}

func eventoInbound() *nats.Event {
	return &nats.Event{Type: nats.EventMessageReceived, TenantID: "tenant-1", Payload: map[string]interface{}{
		"channel_id": "ch-1", "message_id": "m-1", "conversation_id": "c-1", "contact_id": "ct-1",
		"content_type": "text", "content": "oi", "sender_id": "5511987654321",
	}}
}

func regraDe(consumidor string) *entity.ConversationOwnerRule {
	return &entity.ConversationOwnerRule{TenantID: "tenant-1", ChannelID: "ch-1", ContactIdentifier: "+5511987654321",
		Consumer: consumidor, ConversaRef: "conv-1", Desde: agoraFixo.Add(-time.Hour)}
}

func despachar(t *testing.T, d *Dispatcher, ev *nats.Event) (*nats.WebhookDelivery, map[string]json.RawMessage) {
	t.Helper()
	pub := d.deliveries.(*recordingPublisher)
	pub.deliveries = nil
	require.NoError(t, d.handle(context.Background(), ev))
	if len(pub.deliveries) == 0 {
		return nil, nil
	}
	require.Len(t, pub.deliveries, 1)
	var env map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(pub.deliveries[0].Body, &env))
	return pub.deliveries[0], env
}

func roteamento(t *testing.T, env map[string]json.RawMessage) *Routing {
	t.Helper()
	raw, ok := env["routing"]
	if !ok {
		return nil
	}
	var r Routing
	require.NoError(t, json.Unmarshal(raw, &r))
	return &r
}

func novoDispatcher(ch *entity.Channel) *Dispatcher {
	d := NewDispatcher(&recordingPublisher{}, &fakeChannels{channel: ch})
	d.now = func() time.Time { return agoraFixo }
	d.newID = func() string { return "fixo" }
	return d
}

func TestRouting_CanalSemConsumidorExtraComoHoje(t *testing.T) {
	for _, ev := range []*nats.Event{eventoInbound(), {Type: nats.EventConversationAssigned, TenantID: "tenant-1",
		Payload: map[string]interface{}{"channel_id": "ch-1", "conversation_id": "c-1", "contact_id": "ct-1", "status": "open"}}} {
		antes, _ := despachar(t, novoDispatcher(newTestChannel()), ev)
		comRegras := novoDispatcher(newTestChannel()).WithOwners(&fakeOwners{}, &fakeIdentities{})
		depois, env := despachar(t, comRegras, ev)
		require.NotNil(t, antes)
		assert.Equal(t, string(antes.Body), string(depois.Body), "%s: o mesmo envelope, byte a byte", ev.Type)
		assert.Equal(t, antes.URL, depois.URL)
		assert.Empty(t, depois.Consumer)
		assert.Nil(t, roteamento(t, env), "sem regra, sem bloco routing")
	}
}

func TestRouting_DonoConfiguradoRecebe(t *testing.T) {
	owners := &fakeOwners{rule: regraDe("copilot")}
	ids := &fakeIdentities{ids: []*entity.ContactIdentity{
		{ChannelType: "telegram", Identifier: "outro"},
		{ChannelType: string(entity.ChannelTypeWhatsAppOfficial), Identifier: "55 11 98765-4321"},
	}}
	d := novoDispatcher(canalComCopilot()).WithOwners(owners, ids)

	wh, env := despachar(t, d, eventoInbound())
	require.NotNil(t, wh)
	assert.Equal(t, "copilot", wh.Consumer)
	assert.Equal(t, "https://copilot.exemplo.test/webhook", wh.URL)
	assert.Equal(t, &Routing{Consumer: "copilot"}, roteamento(t, env))
	assert.Equal(t, []string{"+5511987654321"}, owners.keys, "chave pela identidade do tipo do canal, normalizada")

	// conversation.* também segue o dono (o evento só traz contact_id).
	wh, _ = despachar(t, d, &nats.Event{Type: nats.EventConversationResolved, TenantID: "tenant-1",
		Payload: map[string]interface{}{"channel_id": "ch-1", "conversation_id": "c-1", "contact_id": "ct-1", "status": "resolved"}})
	require.NotNil(t, wh)
	assert.Equal(t, "copilot", wh.Consumer)
}

func TestRouting_DonoSemConfiguracaoVaiAoPadraoMarcado(t *testing.T) {
	ch := newTestChannel()
	d := novoDispatcher(ch).WithOwners(&fakeOwners{rule: regraDe("copilot")}, &fakeIdentities{})
	wh, env := despachar(t, d, eventoInbound())
	require.NotNil(t, wh)
	assert.Empty(t, wh.Consumer)
	assert.Equal(t, ch.WebhookURL, wh.URL)
	assert.Equal(t, &Routing{Consumer: "default", Dono: "copilot", Fallback: true, Motivo: MotivoConsumidorNaoConfigurado}, roteamento(t, env))
}

func TestRouting_DonoEhOPadraoNomeado(t *testing.T) {
	ch := canalComCopilot()
	ch.Config[cfgDefaultConsumerName] = "crm"
	d := novoDispatcher(ch).WithOwners(&fakeOwners{rule: regraDe("crm")}, &fakeIdentities{})
	wh, env := despachar(t, d, eventoInbound())
	require.NotNil(t, wh)
	assert.Empty(t, wh.Consumer)
	assert.Equal(t, ch.WebhookURL, wh.URL)
	assert.Equal(t, &Routing{Consumer: "crm"}, roteamento(t, env), "dono = o padrão: sem fallback")
}

func TestRouting_RegraExpiradaOuFalhaDaConsulta(t *testing.T) {
	expirada := regraDe("copilot")
	fim := agoraFixo.Add(-time.Minute)
	expirada.ExpiraEm = &fim
	wh, env := despachar(t, novoDispatcher(canalComCopilot()).WithOwners(&fakeOwners{rule: expirada}, &fakeIdentities{}), eventoInbound())
	assert.Empty(t, wh.Consumer)
	assert.Nil(t, roteamento(t, env), "regra expirada conta como ausente")

	wh, env = despachar(t, novoDispatcher(canalComCopilot()).WithOwners(&fakeOwners{err: errors.New("banco fora")}, &fakeIdentities{}), eventoInbound())
	assert.Empty(t, wh.Consumer, "fail-closed: vai ao padrão")
	assert.Equal(t, &Routing{Consumer: "default", Fallback: true, Motivo: MotivoRegraIndisponivel}, roteamento(t, env))
}

func TestRouting_FiltroDeEventosDoDono(t *testing.T) {
	ch := canalComCopilot()
	ch.Config["consumers.copilot.webhook_events"] = "message.received"
	d := novoDispatcher(ch).WithOwners(&fakeOwners{rule: regraDe("copilot")}, &fakeIdentities{ids: []*entity.ContactIdentity{
		{ChannelType: string(entity.ChannelTypeWhatsAppOfficial), Identifier: "5511987654321"}}})
	wh, _ := despachar(t, d, eventoInbound())
	require.NotNil(t, wh)
	wh, _ = despachar(t, d, &nats.Event{Type: nats.EventConversationAssigned, TenantID: "tenant-1",
		Payload: map[string]interface{}{"channel_id": "ch-1", "conversation_id": "c-1", "contact_id": "ct-1"}})
	assert.Nil(t, wh, "o dono não assina conversation.assigned")
}

// ---- worker ----

type destino struct {
	mu     sync.Mutex
	status int
	corpos [][]byte
	sigs   []string
	tss    []string
}

func (d *destino) servidor() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		d.mu.Lock()
		d.corpos = append(d.corpos, b)
		d.sigs = append(d.sigs, r.Header.Get(SignatureHeader))
		d.tss = append(d.tss, r.Header.Get(TimestampHeader))
		st := d.status
		d.mu.Unlock()
		w.WriteHeader(st)
	}))
}

func canalComServidores(padrao, copilot *httptest.Server) *entity.Channel {
	ch := canalComCopilot()
	ch.WebhookURL = padrao.URL
	ch.Config["consumers.copilot.webhook_url"] = copilot.URL
	return ch
}

func entregaAoCopilot() *nats.WebhookDelivery {
	return &nats.WebhookDelivery{ID: "evt_1", ChannelID: "ch-1", Consumer: "copilot", EventType: TypeMessageReceived,
		Body: []byte(`{"id":"evt_1","type":"message.received","tenantId":"tenant-1","routing":{"consumer":"copilot","fallback":false},"data":{}}`)}
}

func TestRouting_WorkerFalhaAteAUltimaEChegaAoPadrao(t *testing.T) {
	padrao, copilot := &destino{status: http.StatusOK}, &destino{status: http.StatusInternalServerError}
	sp, sc := padrao.servidor(), copilot.servidor()
	defer sp.Close()
	defer sc.Close()
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: canalComServidores(sp, sc)})

	const maxDeliver = 10
	for n := uint64(1); n < maxDeliver; n++ {
		err := worker.Handle(nats.WithDeliveryAttempt(context.Background(), n, maxDeliver), entregaAoCopilot())
		require.Error(t, err, "tentativa %d ao dono falhou: devolve erro para o NATS reentregar", n)
	}
	assert.Len(t, copilot.corpos, maxDeliver-1)
	assert.Empty(t, padrao.corpos, "antes da última, o padrão não recebe")

	require.NoError(t, worker.Handle(nats.WithDeliveryAttempt(context.Background(), maxDeliver, maxDeliver), entregaAoCopilot()))
	assert.Len(t, copilot.corpos, maxDeliver-1, "a última não vai ao dono")
	require.Len(t, padrao.corpos, 1, "a mensagem chega ao padrão")

	var env map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(padrao.corpos[0], &env))
	assert.Equal(t, &Routing{Consumer: "default", Dono: "copilot", Fallback: true, Motivo: MotivoTentativasEsgotadas}, roteamento(t, env))
	assert.JSONEq(t, `{}`, string(env["data"]), "o resto do envelope é o mesmo")
	assert.Equal(t, ComputeTimestampedSignature(padrao.tss[0], padrao.corpos[0], "segredo-padrao"), padrao.sigs[0],
		"assinado com o segredo do padrão, sobre o corpo enviado")
}

func TestRouting_WorkerUltimaTentativaComPadraoFalhandoNaoDescarta(t *testing.T) {
	padrao, copilot := &destino{status: http.StatusBadGateway}, &destino{status: http.StatusInternalServerError}
	sp, sc := padrao.servidor(), copilot.servidor()
	defer sp.Close()
	defer sc.Close()
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: canalComServidores(sp, sc)})
	err := worker.Handle(nats.WithDeliveryAttempt(context.Background(), 10, 10), entregaAoCopilot())
	assert.Error(t, err, "o erro sobe: o consumidor do NATS manda à DLQ, nada se perde")
	assert.Len(t, padrao.corpos, 1)
}

func TestRouting_WorkerEntregaAoDonoComOSegredoDele(t *testing.T) {
	padrao, copilot := &destino{status: http.StatusOK}, &destino{status: http.StatusOK}
	sp, sc := padrao.servidor(), copilot.servidor()
	defer sp.Close()
	defer sc.Close()
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: canalComServidores(sp, sc)})
	wh := entregaAoCopilot()
	require.NoError(t, worker.Handle(nats.WithDeliveryAttempt(context.Background(), 1, 10), wh))
	require.Len(t, copilot.corpos, 1)
	assert.Empty(t, padrao.corpos)
	assert.Equal(t, string(wh.Body), string(copilot.corpos[0]), "corpo intacto")
	assert.Equal(t, ComputeTimestampedSignature(copilot.tss[0], copilot.corpos[0], "segredo-copilot"), copilot.sigs[0])
}

func TestRouting_WorkerConsumidorSemConfiguracaoVaiAoPadraoNaHora(t *testing.T) {
	padrao := &destino{status: http.StatusOK}
	sp := padrao.servidor()
	defer sp.Close()
	ch := newTestChannel()
	ch.WebhookURL = sp.URL
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: ch})
	require.NoError(t, worker.Handle(nats.WithDeliveryAttempt(context.Background(), 1, 10), entregaAoCopilot()))
	require.Len(t, padrao.corpos, 1)
	var env map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(padrao.corpos[0], &env))
	assert.Equal(t, &Routing{Consumer: "default", Dono: "copilot", Fallback: true, Motivo: MotivoConsumidorNaoConfigurado}, roteamento(t, env))
}

func TestRouting_WorkerSemConsumidorComoHoje(t *testing.T) {
	padrao := &destino{status: http.StatusOK}
	sp := padrao.servidor()
	defer sp.Close()
	ch := newTestChannel()
	ch.WebhookURL = sp.URL
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: ch})
	body := []byte(`{"id":"evt_9","type":"message.received","data":{}}`)
	// Mesmo na última tentativa, sem consumidor nomeado nada muda: corpo intacto, sem routing.
	require.NoError(t, worker.Handle(nats.WithDeliveryAttempt(context.Background(), 10, 10),
		&nats.WebhookDelivery{ChannelID: "ch-1", EventType: TypeMessageReceived, Body: body}))
	require.Len(t, padrao.corpos, 1)
	assert.Equal(t, string(body), string(padrao.corpos[0]))
}

// ---- etapa 3: status para quem enviou; conversation.owner_changed ----

type fakeSent struct {
	msgs map[string]*entity.Message
	err  error
}

func (f *fakeSent) FindByID(_ context.Context, id string) (*entity.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.msgs[id], nil
}

func eventoStatus(tipo, messageID string) *nats.Event {
	return &nats.Event{Type: tipo, TenantID: "tenant-1", Payload: map[string]interface{}{
		"channel_id": "ch-1", "message_id": messageID, "status": "delivered"}}
}

func canalComChaves() *entity.Channel {
	ch := canalComCopilot()
	ch.Config["consumers.copilot.api_key_id"] = "key-copilot"
	return ch
}

func enviadas() *fakeSent {
	return &fakeSent{msgs: map[string]*entity.Message{
		"m-copilot": {ID: "m-copilot", Metadata: map[string]string{entity.MetadataSentByAPIKey: "key-copilot"}},
		"m-crm":     {ID: "m-crm", Metadata: map[string]string{entity.MetadataSentByAPIKey: "key-crm"}},
		"m-antiga":  {ID: "m-antiga", Metadata: map[string]string{}},
	}}
}

func TestStatus_VaiParaQuemEnviou(t *testing.T) {
	d := novoDispatcher(canalComChaves()).WithSenders(enviadas())
	for _, tipo := range []string{nats.EventMessageSent, nats.EventMessageDelivered, nats.EventMessageRead, nats.EventMessageFailed} {
		wh, env := despachar(t, d, eventoStatus(tipo, "m-copilot"))
		require.NotNil(t, wh, tipo)
		assert.Equal(t, "copilot", wh.Consumer, tipo)
		assert.Equal(t, "https://copilot.exemplo.test/webhook", wh.URL)
		assert.Equal(t, &Routing{Consumer: "copilot"}, roteamento(t, env))
	}
	// Chave que não é de nenhum consumidor nomeado (a do CRM, o padrão): vai ao padrão.
	wh, env := despachar(t, d, eventoStatus(nats.EventMessageDelivered, "m-crm"))
	assert.Empty(t, wh.Consumer)
	assert.Nil(t, roteamento(t, env))
}

func TestStatus_MensagemAntigaSemChaveVaiAoPadraoComoHoje(t *testing.T) {
	for _, id := range []string{"m-antiga", "m-inexistente"} {
		antes, _ := despachar(t, novoDispatcher(canalComChaves()), eventoStatus(nats.EventMessageDelivered, id))
		depois, env := despachar(t, novoDispatcher(canalComChaves()).WithSenders(enviadas()), eventoStatus(nats.EventMessageDelivered, id))
		require.NotNil(t, depois)
		assert.Empty(t, depois.Consumer, id)
		assert.Equal(t, antes.URL, depois.URL)
		assert.Equal(t, string(antes.Body), string(depois.Body), "%s: o mesmo envelope de hoje", id)
		assert.Nil(t, roteamento(t, env))
	}
	// Falha ao ler a mensagem: o padrão, sem perder o status.
	wh, _ := despachar(t, novoDispatcher(canalComChaves()).WithSenders(&fakeSent{err: errors.New("banco fora")}),
		eventoStatus(nats.EventMessageDelivered, "m-copilot"))
	require.NotNil(t, wh)
	assert.Empty(t, wh.Consumer)
}

func eventoOwnerChanged() *nats.Event {
	return &nats.Event{Type: nats.EventConversationOwnerChanged, TenantID: "tenant-1", Payload: map[string]interface{}{
		"channel_id": "ch-1", "contact_identifier": "+5511987654321", "previous_consumer": "crm", "consumer": "copilot",
		"conversa_ref": "conv-1", "motivo": "declarado"}}
}

func TestOwnerChanged_EntregueAosDoisConsumidoresQueAssinam(t *testing.T) {
	ch := canalComCopilot()
	ch.Config[cfgDefaultConsumerName] = "crm"
	ch.Config[cfgWebhookEvents] = "message.received,conversation.owner_changed"
	ch.Config["consumers.copilot.webhook_events"] = "conversation.owner_changed"
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: ch})
	require.NoError(t, d.handle(context.Background(), eventoOwnerChanged()))
	require.Len(t, pub.deliveries, 2)
	assert.Empty(t, pub.deliveries[0].Consumer)
	assert.Equal(t, "copilot", pub.deliveries[1].Consumer)
	assert.NotEqual(t, pub.deliveries[0].ID, pub.deliveries[1].ID, "ids distintos: o stream deduplica por id")
	var env struct {
		Type string           `json:"type"`
		Data OwnerChangedData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(pub.deliveries[1].Body, &env))
	assert.Equal(t, TypeConversationOwnerChanged, env.Type)
	assert.Equal(t, OwnerChangedData{ChannelID: "ch-1", ChannelType: string(entity.ChannelTypeWhatsAppOfficial),
		ContactIdentifier: "+5511987654321", PreviousConsumer: "crm", Consumer: "copilot", ConversaRef: "conv-1",
		Motivo: "declarado"}, env.Data)
}

func TestOwnerChanged_OptIn(t *testing.T) {
	ch := canalComCopilot()
	ch.Config["consumers.copilot.webhook_events"] = "*"
	pub := &recordingPublisher{}
	require.NoError(t, NewDispatcher(pub, &fakeChannels{channel: ch}).handle(context.Background(), eventoOwnerChanged()))
	assert.Empty(t, pub.deliveries, "filtro vazio ou * não recebe o evento novo: só quem o lista pelo nome")
}
