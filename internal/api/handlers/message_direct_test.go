package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// directSendDeps bundles the mocks a direct-send test asserts against.
type directSendDeps struct {
	handler     *MessageHandler
	messages    *testutil.MockMessageRepository
	convos      *testutil.MockConversationRepository
	channels    *testutil.MockChannelRepository
	contacts    *testutil.MockContactRepository
	producer    *testutil.MockProducer
	sandbox     *testutil.MockSandboxAllowlistRepository
	idempotency *testutil.MockMessageIdempotencyRepository
}

func setupDirectSend() *directSendDeps {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	producer := testutil.NewMockProducer()
	sandbox := testutil.NewMockSandboxAllowlistRepository()
	idem := testutil.NewMockMessageIdempotencyRepository()

	svc := service.NewMessageService(msgRepo, convRepo, channelRepo, contactRepo, producer)
	svc.SetSandboxAllowlist(sandbox)
	svc.SetIdempotencyStore(idem)

	return &directSendDeps{
		handler:     NewMessageHandler(svc),
		messages:    msgRepo,
		convos:      convRepo,
		channels:    channelRepo,
		contacts:    contactRepo,
		producer:    producer,
		sandbox:     sandbox,
		idempotency: idem,
	}
}

// seedConnectedChannel adds an enabled+connected channel to the mock repo.
func seedConnectedChannel(repo *testutil.MockChannelRepository, id, tenantID string, chType entity.ChannelType) *entity.Channel {
	now := time.Now()
	ch := &entity.Channel{
		ID:               id,
		TenantID:         tenantID,
		Type:             chType,
		Name:             string(chType) + "-" + id,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Environment:      entity.ChannelEnvironmentProduction,
		Config:           map[string]string{},
		Credentials:      map[string]string{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	repo.Channels[id] = ch
	return ch
}

// postDirectSend runs the handler against a JSON body with tenant-1 authenticated.
func postDirectSend(d *directSendDeps, body map[string]interface{}) (*httptest.ResponseRecorder, Response) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "apikey:key-1")
	raw, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")

	d.handler.SendDirect(c)

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w, resp
}

// directSendData reads the canonical data block out of a 202 response.
func directSendData(t *testing.T, resp Response) map[string]interface{} {
	t.Helper()
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "response data is not an object: %#v", resp.Data)
	return data
}

// ---------------------------------------------------------------------------
// Tenant isolation and validation
// ---------------------------------------------------------------------------

func TestDirectSend_ChannelOfAnotherTenant_Returns404(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-other", "tenant-2", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-other",
		"to":         "+5511999999999",
		"text":       "oi",
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "CHANNEL_NOT_FOUND", resp.Error.Code)
	assert.Empty(t, d.producer.OutboundMessages)
}

func TestDirectSend_UnknownChannel_Returns404(t *testing.T) {
	d := setupDirectSend()

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "does-not-exist",
		"to":         "+5511999999999",
		"text":       "oi",
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "CHANNEL_NOT_FOUND", resp.Error.Code)
}

func TestDirectSend_DisconnectedChannel_IsRejected(t *testing.T) {
	d := setupDirectSend()
	ch := seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	ch.ConnectionStatus = entity.ConnectionStatusDisconnected

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "oi",
	})

	assert.NotEqual(t, http.StatusAccepted, w.Code)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "CHANNEL_DISCONNECTED", resp.Error.Code)
	assert.Empty(t, d.producer.OutboundMessages)
}

func TestDirectSend_InvalidBody_Returns400(t *testing.T) {
	seedChannelFor := func(d *directSendDeps) {
		seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	}

	cases := []struct {
		name string
		body map[string]interface{}
	}{
		{"sem channel_id", map[string]interface{}{"to": "+5511999999999", "text": "oi"}},
		{"sem destinatário", map[string]interface{}{"channel_id": "channel-1", "text": "oi"}},
		{"destinatário em branco", map[string]interface{}{"channel_id": "channel-1", "to": "   ", "text": "oi"}},
		{"sem texto", map[string]interface{}{"channel_id": "channel-1", "to": "+5511999999999"}},
		{"texto em branco", map[string]interface{}{"channel_id": "channel-1", "to": "+5511999999999", "text": "  "}},
		{"content_type não suportado", map[string]interface{}{
			"channel_id": "channel-1", "to": "+5511999999999", "text": "oi", "content_type": "location",
		}},
		// Mídia declarada sem nada para entregar: sem esta recusa a mensagem sai
		// anunciada como imagem carregando vazio, e o envio ainda reporta sucesso.
		{"mídia sem anexo", map[string]interface{}{
			"channel_id": "channel-1", "to": "+5511999999999", "text": "olha isso", "content_type": "image",
		}},
		{"anexo sem url", map[string]interface{}{
			"channel_id": "channel-1", "to": "+5511999999999", "content_type": "image",
			"attachments": []map[string]interface{}{{"type": "image", "mime_type": "image/png"}},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := setupDirectSend()
			seedChannelFor(d)

			w, resp := postDirectSend(d, tc.body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
			assert.Empty(t, d.producer.OutboundMessages)
		})
	}
}

func TestDirectSend_MalformedJSON_Returns400(t *testing.T) {
	d := setupDirectSend()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", bytes.NewReader([]byte("{not json")))
	c.Request.Header.Set("Content-Type", "application/json")

	d.handler.SendDirect(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDirectSend_ReservedMetadata_Returns400(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "oi",
		"metadata": map[string]string{
			"source":      "alcada",
			"sender_id":   "outro-usuario",
			"campaign_id": "campanha-forjada",
		},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "campaign_id,sender_id", resp.Error.Details["reserved_keys"])
	assert.Empty(t, d.producer.OutboundMessages)
	assert.Empty(t, d.messages.Messages)
}

// ---------------------------------------------------------------------------
// Happy path: identity, contact, conversation, message, NATS
// ---------------------------------------------------------------------------

func TestDirectSend_CreatesContactConversationAndPublishes(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
		"metadata": map[string]string{
			"source":             "alcada",
			"idempotency_key":    "chave-logica",
			"alcada_correlation": "token-opaco",
		},
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, resp.Success)

	data := directSendData(t, resp)
	assert.NotEmpty(t, data["id"])
	assert.NotEmpty(t, data["conversation_id"])
	assert.Equal(t, "channel-1", data["channel_id"])
	assert.Equal(t, "queued", data["status"])

	// Contact + identity created inside the tenant, stored in the canonical
	// digits-only form the WhatsApp adapters use.
	require.Len(t, d.contacts.Contacts, 1)
	var contact *entity.Contact
	for _, c := range d.contacts.Contacts {
		contact = c
	}
	assert.Equal(t, "tenant-1", contact.TenantID)
	assert.Equal(t, "5511999999999", contact.Phone)
	require.Len(t, contact.Identities, 1)
	assert.Equal(t, "whatsapp", contact.Identities[0].ChannelType)
	assert.Equal(t, "5511999999999", contact.Identities[0].Identifier)

	// Conversation opened on the channel.
	require.Len(t, d.convos.Conversations, 1)
	var conversation *entity.Conversation
	for _, c := range d.convos.Conversations {
		conversation = c
	}
	assert.Equal(t, "channel-1", conversation.ChannelID)
	assert.Equal(t, contact.ID, conversation.ContactID)
	assert.Equal(t, data["conversation_id"], conversation.ID)

	// Message persisted with the caller's metadata intact.
	require.Len(t, d.messages.Messages, 1)
	message := d.messages.Messages[data["id"].(string)]
	require.NotNil(t, message)
	assert.Equal(t, "Mensagem", message.Content)
	assert.Equal(t, entity.ContentTypeText, message.ContentType)
	assert.Equal(t, "alcada", message.Metadata["source"])
	assert.Equal(t, "token-opaco", message.Metadata["alcada_correlation"])
	assert.Equal(t, "chave-logica", message.Metadata["idempotency_key"])

	// Published on the outbound stream (not straight to the adapter), carrying
	// the same metadata.
	require.Len(t, d.producer.OutboundMessages, 1)
	out := d.producer.OutboundMessages[0]
	assert.Equal(t, message.ID, out.ID)
	assert.Equal(t, "tenant-1", out.TenantID)
	assert.Equal(t, "channel-1", out.ChannelID)
	assert.Equal(t, "whatsapp", out.ChannelType)
	assert.Equal(t, conversation.ID, out.ConversationID)
	assert.Equal(t, "5511999999999", out.RecipientID)
	assert.Equal(t, "Mensagem", out.Content)
	assert.Equal(t, "alcada", out.Metadata["source"])
	assert.Equal(t, "token-opaco", out.Metadata["alcada_correlation"])
}

func TestDirectSend_ContentAliasIsAccepted(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id":   "channel-1",
		"to":           "+5511999999999",
		"content_type": "text",
		"content":      "via alias",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	data := directSendData(t, resp)
	assert.Equal(t, "via alias", d.messages.Messages[data["id"].(string)].Content)
}

func TestDirectSend_MediaWithAttachmentGoesOut(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id":   "channel-1",
		"to":           "+5511999999999",
		"content_type": "image",
		"text":         "segue o comprovante",
		"attachments": []map[string]interface{}{{
			"url":        "https://arquivos.exemplo/comprovante.png",
			"type":       "image",
			"filename":   "comprovante.png",
			"mime_type":  "image/png",
			"size_bytes": 2048,
		}},
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	data := directSendData(t, resp)

	message := d.messages.Messages[data["id"].(string)]
	require.NotNil(t, message)
	assert.Equal(t, entity.ContentTypeImage, message.ContentType)
	// A legenda acompanha a mídia; não vira mensagem separada.
	assert.Equal(t, "segue o comprovante", message.Content)
	require.Len(t, message.Attachments, 1)
	assert.Equal(t, "https://arquivos.exemplo/comprovante.png", message.Attachments[0].URL)
	assert.Equal(t, "image/png", message.Attachments[0].MimeType)

	require.Len(t, d.producer.OutboundMessages, 1)
	out := d.producer.OutboundMessages[0]
	assert.Equal(t, "image", out.ContentType)
	require.Len(t, out.Attachments, 1)
	assert.Equal(t, "https://arquivos.exemplo/comprovante.png", out.Attachments[0].URL)
}

// Anexo sem content_type vira "document": é o único tipo que todo transporte
// carrega, então errar aqui piora a apresentação, nunca a entrega.
func TestDirectSend_AttachmentWithoutContentTypeBecomesDocument(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"attachments": []map[string]interface{}{{
			"url":      "https://arquivos.exemplo/boleto.pdf",
			"filename": "boleto.pdf",
		}},
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	data := directSendData(t, resp)
	assert.Equal(t, entity.ContentTypeDocument, d.messages.Messages[data["id"].(string)].ContentType)
}

func TestDirectSend_ReusesExistingContactAndConversation(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	// A contact the inbound path would have created: identity stored digits-only.
	contact := seedContact(d.contacts, "contact-1", "tenant-1", "Cliente", "", "5511999999999")
	identity := &entity.ContactIdentity{
		ID:          "identity-1",
		ContactID:   contact.ID,
		TenantID:    "tenant-1",
		ChannelType: "whatsapp",
		Identifier:  "5511999999999",
	}
	contact.Identities = []*entity.ContactIdentity{identity}
	d.contacts.Identities[contact.ID] = []*entity.ContactIdentity{identity}

	existing := &entity.Conversation{
		ID:        "conv-1",
		TenantID:  "tenant-1",
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Status:    entity.ConversationStatusOpen,
		CreatedAt: time.Now(),
	}
	d.convos.Conversations[existing.ID] = existing

	// The caller writes the number in E.164; the stored identity has no "+".
	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+55 11 99999-9999",
		"text":       "oi de novo",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	data := directSendData(t, resp)
	assert.Equal(t, "conv-1", data["conversation_id"])
	assert.Len(t, d.contacts.Contacts, 1, "não deve duplicar o contato")
	assert.Len(t, d.convos.Conversations, 1, "não deve abrir outra conversa")
}

func TestDirectSend_EmailChannelUsesTheSameRoute(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-email", "tenant-1", entity.ChannelTypeEmail)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id":   "channel-email",
		"to":           "Gestor@Empresa.com",
		"content_type": "text",
		"text":         "Seu despacho por exceção...",
		"metadata": map[string]string{
			"source":          "alcada",
			"idempotency_key": "gestor:resumo:INICIO:2026-08-13",
			"subject":         "Alçada — despacho por exceção",
		},
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	data := directSendData(t, resp)

	require.Len(t, d.contacts.Contacts, 1)
	var contact *entity.Contact
	for _, c := range d.contacts.Contacts {
		contact = c
	}
	assert.Equal(t, "gestor@empresa.com", contact.Email)
	require.Len(t, contact.Identities, 1)
	assert.Equal(t, "email", contact.Identities[0].ChannelType)
	assert.Equal(t, "gestor@empresa.com", contact.Identities[0].Identifier)

	require.Len(t, d.producer.OutboundMessages, 1)
	out := d.producer.OutboundMessages[0]
	assert.Equal(t, "email", out.ChannelType)
	assert.Equal(t, "gestor@empresa.com", out.RecipientID)
	// The email sender reads metadata.subject; it must survive the direct send.
	assert.Equal(t, "Alçada — despacho por exceção", out.Metadata["subject"])
	assert.Equal(t, "alcada", d.messages.Messages[data["id"].(string)].Metadata["source"])
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

func TestDirectSend_SameIdempotencyKeyDoesNotDuplicate(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	body := map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
		"metadata":   map[string]string{"idempotency_key": "chave-logica"},
	}

	w1, resp1 := postDirectSend(d, body)
	require.Equal(t, http.StatusAccepted, w1.Code)
	first := directSendData(t, resp1)

	w2, resp2 := postDirectSend(d, body)
	require.Equal(t, http.StatusAccepted, w2.Code, "repetição é sucesso idempotente, não conflito")
	second := directSendData(t, resp2)

	assert.Equal(t, first["id"], second["id"], "deve devolver a mensagem original")
	assert.Equal(t, first["conversation_id"], second["conversation_id"])
	assert.Len(t, d.messages.Messages, 1, "não deve criar uma segunda mensagem")
	assert.Len(t, d.producer.OutboundMessages, 1, "não deve publicar duas vezes")
}

func TestDirectSend_IdempotencyIsScopedToTheTenant(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	_, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
		"metadata":   map[string]string{"idempotency_key": "chave-logica"},
	})
	firstID := directSendData(t, resp)["id"]

	// The same key on another tenant is a different key.
	assert.Contains(t, d.idempotency.Reservations, "tenant-1\x00chave-logica")
	existing, err := d.idempotency.Reserve(t.Context(), "tenant-2", "chave-logica", "outra-mensagem")
	require.NoError(t, err)
	assert.Empty(t, existing, "a chave de outro tenant não deve colidir")
	assert.NotEqual(t, "outra-mensagem", firstID)
}

func TestDirectSend_FailedSendReleasesTheKey(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	d.producer.ReturnError = assert.AnError

	body := map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
		"metadata":   map[string]string{"idempotency_key": "chave-logica"},
	}

	w, _ := postDirectSend(d, body)
	require.NotEqual(t, http.StatusAccepted, w.Code)
	assert.Empty(t, d.idempotency.Reservations, "uma chave que não protege nada deve ser liberada")

	// A retry with the same key now goes through.
	d.producer.ReturnError = nil
	w2, resp2 := postDirectSend(d, body)
	require.Equal(t, http.StatusAccepted, w2.Code)
	assert.NotEmpty(t, directSendData(t, resp2)["id"])
}

// ---------------------------------------------------------------------------
// Sandbox
// ---------------------------------------------------------------------------

func TestDirectSend_SandboxGuardStillApplies(t *testing.T) {
	d := setupDirectSend()
	ch := seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	ch.Environment = entity.ChannelEnvironmentSandbox

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "sandbox")
	assert.Empty(t, d.producer.OutboundMessages)
}

func TestDirectSend_SandboxAllowlistedRecipientGoesThrough(t *testing.T) {
	d := setupDirectSend()
	ch := seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	ch.Environment = entity.ChannelEnvironmentSandbox
	d.sandbox.Entries["entry-1"] = &entity.SandboxAllowlistEntry{
		ID:        "entry-1",
		TenantID:  "tenant-1",
		Recipient: "+5511999999999",
	}

	w, _ := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Len(t, d.producer.OutboundMessages, 1)
}

// ---------------------------------------------------------------------------
// Scope enforcement (route level)
// ---------------------------------------------------------------------------

// directSendRouter mounts the real route with the messages:send gate, so the
// scope check is exercised exactly as main.go wires it.
func directSendRouter(d *directSendDeps, scopes []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	auth := middleware.NewAuthMiddleware(nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Set("user_id", "apikey:key-1")
		c.Set(middleware.ScopesKey, scopes)
		c.Next()
	})
	r.POST("/api/v1/messages/send", auth.RequireScope(middleware.ScopeMessagesSend), d.handler.SendDirect)
	return r
}

func TestDirectSend_KeyWithoutMessagesSendScope_Returns403(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	raw, _ := json.Marshal(map[string]interface{}{
		"channel_id": "channel-1", "to": "+5511999999999", "text": "oi",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	directSendRouter(d, []string{"channels:write", "contacts:read"}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, d.producer.OutboundMessages)
}

func TestDirectSend_KeyWithMessagesSendScope_IsAccepted(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	raw, _ := json.Marshal(map[string]interface{}{
		"channel_id": "channel-1", "to": "+5511999999999", "text": "oi",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	directSendRouter(d, []string{middleware.ScopeMessagesSend}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Len(t, d.producer.OutboundMessages, 1)
}

// An idempotent repeat reports the message's real status. Answering "queued"
// for something that already failed would be a lie the caller acts on.
func TestDirectSend_IdempotentRepeatReportsTheRealStatus(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	body := map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
		"metadata":   map[string]string{"idempotency_key": "chave-logica"},
	}

	_, resp := postDirectSend(d, body)
	first := directSendData(t, resp)
	require.Equal(t, "queued", first["status"])

	// The provider rejected it: the status pipeline marked the message failed.
	d.messages.Messages[first["id"].(string)].Status = entity.MessageStatusFailed

	w, resp2 := postDirectSend(d, body)
	require.Equal(t, http.StatusAccepted, w.Code)
	second := directSendData(t, resp2)
	assert.Equal(t, first["id"], second["id"])
	assert.Equal(t, "failed", second["status"])
}

// reply_to_id / quoted_text look internal but are the client-facing way to ask
// for a quoted reply (the WhatsApp and Telegram senders read them off the
// outbound metadata). Reserving them would leave no route in the API able to
// reply with a quote, so they must pass through to the adapter untouched.
func TestDirectSend_QuotedReplyMetadataIsAccepted(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, _ := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "respondendo",
		"metadata": map[string]string{
			"reply_to_id": "wamid.ORIGINAL",
			"quoted_text": "mensagem citada",
		},
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Len(t, d.producer.OutboundMessages, 1)
	out := d.producer.OutboundMessages[0]
	assert.Equal(t, "wamid.ORIGINAL", out.Metadata["reply_to_id"])
	assert.Equal(t, "mensagem citada", out.Metadata["quoted_text"])
}

// An API key with no bound user authenticates as "apikey:<id>", which is not a
// UUID. messages.sender_id is a UUID column, so persisting it verbatim would
// 500 the send — exactly on the server-to-server path this route exists for.
func TestDirectSend_APIKeyWithoutUserDoesNotPersistASyntheticSenderID(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "oi",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	message := d.messages.Messages[directSendData(t, resp)["id"].(string)]
	require.NotNil(t, message)
	assert.Empty(t, message.SenderID, "sender_id sintético não pode chegar a uma coluna UUID")
	assert.Equal(t, entity.SenderTypeSystem, message.SenderType)
}

// The caller named an explicit recipient; a contact holding several identities
// of the same channel type must not divert the delivery. The identity query has
// no ORDER BY, so re-deriving the address is a coin flip.
func TestDirectSend_DeliversToTheRequestedAddress(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	// Known contact whose first whatsapp identity is a DIFFERENT number.
	contact := seedContact(d.contacts, "contact-1", "tenant-1", "Cliente", "", "5511888888888")
	stale := &entity.ContactIdentity{
		ID: "identity-stale", ContactID: contact.ID, TenantID: "tenant-1",
		ChannelType: "whatsapp", Identifier: "5511888888888",
	}
	wanted := &entity.ContactIdentity{
		ID: "identity-wanted", ContactID: contact.ID, TenantID: "tenant-1",
		ChannelType: "whatsapp", Identifier: "5511999999999",
	}
	contact.Identities = []*entity.ContactIdentity{stale, wanted}
	d.contacts.Identities[contact.ID] = contact.Identities

	w, _ := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "oi",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Len(t, d.producer.OutboundMessages, 1)
	assert.Equal(t, "5511999999999", d.producer.OutboundMessages[0].RecipientID)
}

// A blocked sandbox send must not leave a contact, a conversation and their
// "created" webhooks behind on every attempt.
func TestDirectSend_SandboxBlockWritesNothing(t *testing.T) {
	d := setupDirectSend()
	ch := seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)
	ch.Environment = entity.ChannelEnvironmentSandbox

	w, _ := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
	})

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, d.contacts.Contacts, "não deve criar contato para um envio bloqueado")
	assert.Empty(t, d.convos.Conversations, "não deve abrir conversa para um envio bloqueado")
	assert.Empty(t, d.contacts.OutboxEvents, "não deve emitir contact.created")
	assert.Empty(t, d.convos.OutboxEvents, "não deve emitir conversation.created")
}

// An idempotency key longer than the column is a bad request, not a 500 from
// the driver.
func TestDirectSend_OversizedIdempotencyKey_Returns400(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	w, resp := postDirectSend(d, map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "oi",
		"metadata":   map[string]string{"idempotency_key": strings.Repeat("k", service.MaxIdempotencyKeyLength+1)},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	assert.Empty(t, d.producer.OutboundMessages)
}

// A reservation whose message never appeared must not answer "queued" forever.
// While it may still be in flight the caller is told to retry; once it is old
// enough to be an orphan from a crashed process, it is reclaimed and the send
// goes through.
func TestDirectSend_AbandonedReservationIsReclaimed(t *testing.T) {
	d := setupDirectSend()
	seedConnectedChannel(d.channels, "channel-1", "tenant-1", entity.ChannelTypeWhatsApp)

	// A reservation naming a message that does not exist.
	const key = "tenant-1\x00chave-orfa"
	d.idempotency.Reservations[key] = "mensagem-que-nunca-existiu"
	d.idempotency.ReservedAt[key] = time.Now()

	body := map[string]interface{}{
		"channel_id": "channel-1",
		"to":         "+5511999999999",
		"text":       "Mensagem",
		"metadata":   map[string]string{"idempotency_key": "chave-orfa"},
	}

	// Fresh: could still be a concurrent first request → tell the caller to retry.
	w, resp := postDirectSend(d, body)
	assert.Equal(t, http.StatusConflict, w.Code)
	require.NotNil(t, resp.Error)
	assert.Empty(t, d.producer.OutboundMessages, "não inventa uma entrega")

	// Aged past the abandonment window → reclaimed, and the send happens.
	d.idempotency.ReservedAt[key] = time.Now().Add(-time.Hour)
	w2, resp2 := postDirectSend(d, body)
	require.Equal(t, http.StatusAccepted, w2.Code)
	id := directSendData(t, resp2)["id"].(string)
	assert.NotEqual(t, "mensagem-que-nunca-existiu", id)
	assert.Len(t, d.producer.OutboundMessages, 1)
	assert.Equal(t, id, d.idempotency.Reservations[key], "a chave passa a proteger a nova mensagem")
}
