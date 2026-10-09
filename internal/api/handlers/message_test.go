package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMessageHandler creates a MessageHandler backed by mock repos and a mock producer.
func setupMessageHandler() (
	*MessageHandler,
	*testutil.MockMessageRepository,
	*testutil.MockConversationRepository,
	*testutil.MockChannelRepository,
	*testutil.MockContactRepository,
	*testutil.MockProducer,
) {
	msgRepo := testutil.NewMockMessageRepository()
	convRepo := testutil.NewMockConversationRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()
	producer := testutil.NewMockProducer()

	svc := service.NewMessageService(msgRepo, convRepo, channelRepo, contactRepo, producer)
	handler := NewMessageHandler(svc)

	return handler, msgRepo, convRepo, channelRepo, contactRepo, producer
}

// newMessageAuthContext creates a gin context with tenant_id and user_id set.
func newMessageAuthContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// seedMessage adds a message to the mock repo and returns it.
func seedMessage(repo *testutil.MockMessageRepository, id, conversationID string) *entity.Message {
	now := time.Now()
	msg := &entity.Message{
		ID:             id,
		ConversationID: conversationID,
		SenderType:     entity.SenderTypeUser,
		SenderID:       "user-1",
		ContentType:    entity.ContentTypeText,
		Content:        "Hello world",
		Status:         entity.MessageStatusSent,
		Metadata:       make(map[string]string),
		Attachments:    make([]*entity.MessageAttachment, 0),
		CreatedAt:      now,
	}
	repo.Messages[id] = msg
	return msg
}

// parseMessageResponse unmarshals the recorder body into a Response struct.
func parseMessageResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body")
	return resp
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestMessageList_ValidConversationID_Returns200(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-1")
	seedMessage(msgRepo, "msg-2", "conv-1")
	seedMessage(msgRepo, "msg-3", "conv-other") // different conversation

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations/conv-1/messages", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.True(t, resp.Success)
	require.NotNil(t, resp.Meta)
	assert.Equal(t, 2, resp.Meta.PageSize)
	assert.False(t, resp.Meta.HasNext)
	assert.Empty(t, resp.Meta.NextCursor, "sem mais nada atrás, não se emite cursor")

	dataSlice, ok := resp.Data.([]interface{})
	require.True(t, ok, "expected data to be a slice")
	assert.Len(t, dataSlice, 2)
}

func TestMessageList_EmptyConversationID_Returns400(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	c, w := newMessageAuthContext()
	c.Params = gin.Params{}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations//messages", nil)

	handler.List(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestMessageList_OtherTenantConversation_ReturnsError(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-2", "tenant-2", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-2")

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-2"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations/conv-2/messages", nil)

	handler.List(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}

// ---------------------------------------------------------------------------
// Send
// ---------------------------------------------------------------------------

func TestMessageSend_ValidRequest_Returns201(t *testing.T) {
	handler, _, convRepo, channelRepo, contactRepo, _ := setupMessageHandler()

	// Seed conversation, channel, and contact for the Send flow
	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID:       "channel-1",
		TenantID: "tenant-1",
		Type:     entity.ChannelTypeWhatsApp,
	}
	contactRepo.Contacts["contact-1"] = &entity.Contact{
		ID:       "contact-1",
		TenantID: "tenant-1",
		Name:     "Test Contact",
		Phone:    "+5511999999999",
	}

	payload := SendMessageRequest{
		ContentType: "text",
		Content:     "Hello from test",
	}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Send(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	resp := parseMessageResponse(t, w)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "expected data to be an object")
	assert.Equal(t, "conv-1", data["conversation_id"])
	assert.Equal(t, "Hello from test", data["content"])
}

func TestMessageSend_EmptyConversationID_Returns400(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	payload := SendMessageRequest{
		ContentType: "text",
		Content:     "Hello",
	}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations//messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Send(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
}

func TestMessageSend_InvalidJSON_Returns400(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages", bytes.NewReader([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Send(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestMessageSend_NoUserID_Returns401(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	payload := SendMessageRequest{
		ContentType: "text",
		Content:     "Hello",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No user_id set
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Send(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMessageSend_OtherTenantConversation_ReturnsError(t *testing.T) {
	handler, _, convRepo, channelRepo, contactRepo, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-2", "tenant-2", entity.ConversationStatusOpen)
	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID:       "channel-1",
		TenantID: "tenant-2",
		Type:     entity.ChannelTypeWhatsApp,
	}
	contactRepo.Contacts["contact-1"] = &entity.Contact{
		ID:       "contact-1",
		TenantID: "tenant-2",
		Name:     "Other Tenant Contact",
		Phone:    "+5511888888888",
	}

	payload := SendMessageRequest{
		ContentType: "text",
		Content:     "cross-tenant attempt",
	}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-2"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-2/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Send(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestMessageGet_ValidID_Returns200(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-1")

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "msg-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/messages/msg-1", nil)

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "expected data to be an object")
	assert.Equal(t, "msg-1", data["id"])
	assert.Equal(t, "Hello world", data["content"])
}

func TestMessageGet_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	c, w := newMessageAuthContext()
	c.Params = gin.Params{}
	c.Request = httptest.NewRequest(http.MethodGet, "/messages/", nil)

	handler.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestMessageGet_NotFound_ReturnsError(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/messages/nonexistent", nil)

	handler.Get(c)

	// The service wraps the not-found error with an app error code, so it should not be 200
	assert.NotEqual(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}

func TestMessageGet_OtherTenantMessage_ReturnsError(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-2", "tenant-2", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-2")

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "msg-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/messages/msg-1", nil)

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}

// ---------------------------------------------------------------------------
// SendReaction
// ---------------------------------------------------------------------------

func TestMessageSendReaction_Valid_Returns200(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-1")

	payload := SendReactionRequest{Emoji: "thumbsup"}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{
		{Key: "id", Value: "conv-1"},
		{Key: "messageId", Value: "msg-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages/msg-1/reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "expected data to be an object")
	assert.Equal(t, "msg-1", data["message_id"])
	assert.Equal(t, "thumbsup", data["emoji"])
	assert.Equal(t, "Reaction added successfully", data["message"])
}

func TestMessageSendReaction_EmptyConversationID_Returns400(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	payload := SendReactionRequest{Emoji: "thumbsup"}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{
		{Key: "messageId", Value: "msg-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations//messages/msg-1/reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestMessageSendReaction_EmptyMessageID_Returns400(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	payload := SendReactionRequest{Emoji: "thumbsup"}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{
		{Key: "id", Value: "conv-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages//reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "Message ID")
}

func TestMessageSendReaction_NoUserID_Returns401(t *testing.T) {
	handler, _, _, _, _, _ := setupMessageHandler()

	payload := SendReactionRequest{Emoji: "thumbsup"}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No user_id set
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{
		{Key: "id", Value: "conv-1"},
		{Key: "messageId", Value: "msg-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages/msg-1/reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMessageSendReaction_RemoveReaction_Returns200(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-1")

	// Empty emoji means remove reaction
	payload := SendReactionRequest{Emoji: ""}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{
		{Key: "id", Value: "conv-1"},
		{Key: "messageId", Value: "msg-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages/msg-1/reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Reaction removed successfully", data["message"])
}

func TestMessageSendReaction_OtherTenantConversation_ReturnsError(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-2", "tenant-2", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-2")

	payload := SendReactionRequest{Emoji: "thumbsup"}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{
		{Key: "id", Value: "conv-2"},
		{Key: "messageId", Value: "msg-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-2/messages/msg-1/reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}

func TestMessageSendReaction_MessageOutsideConversation_ReturnsError(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedConversation(convRepo, "conv-other", "tenant-1", entity.ConversationStatusOpen)
	seedMessage(msgRepo, "msg-1", "conv-other")

	payload := SendReactionRequest{Emoji: "thumbsup"}
	body, _ := json.Marshal(payload)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{
		{Key: "id", Value: "conv-1"},
		{Key: "messageId", Value: "msg-1"},
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages/msg-1/reactions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendReaction(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	resp := parseMessageResponse(t, w)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
}

// ---------------------------------------------------------------------------
// Metadata contract (shared with POST /messages/send)
// ---------------------------------------------------------------------------

// seedConversationSendDeps seeds the conversation/channel/contact trio the Send
// flow needs and returns the message repo for assertions.
func seedConversationSendDeps(convRepo *testutil.MockConversationRepository, channelRepo *testutil.MockChannelRepository, contactRepo *testutil.MockContactRepository) {
	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID: "channel-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsApp,
	}
	contactRepo.Contacts["contact-1"] = &entity.Contact{
		ID: "contact-1", TenantID: "tenant-1", Name: "Test Contact", Phone: "5511999999999",
	}
}

func postConversationMessage(handler *MessageHandler, payload SendMessageRequest) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Send(c)
	return w
}

// The conversation route carries the caller's metadata exactly like the direct
// send: through the message row and on to the outbound stream.
func TestMessageSend_PreservesCallerMetadata(t *testing.T) {
	handler, msgRepo, convRepo, channelRepo, contactRepo, producer := setupMessageHandler()
	seedConversationSendDeps(convRepo, channelRepo, contactRepo)

	w := postConversationMessage(handler, SendMessageRequest{
		ContentType: "text",
		Content:     "Mensagem",
		Metadata: map[string]string{
			"source":             "alcada",
			"idempotency_key":    "chave-logica",
			"alcada_correlation": "token-opaco",
		},
	})
	require.Equal(t, http.StatusCreated, w.Code)

	require.Len(t, msgRepo.Messages, 1)
	for _, msg := range msgRepo.Messages {
		assert.Equal(t, "alcada", msg.Metadata["source"])
		assert.Equal(t, "chave-logica", msg.Metadata["idempotency_key"])
		assert.Equal(t, "token-opaco", msg.Metadata["alcada_correlation"])
	}

	require.Len(t, producer.OutboundMessages, 1)
	out := producer.OutboundMessages[0]
	assert.Equal(t, "alcada", out.Metadata["source"])
	assert.Equal(t, "token-opaco", out.Metadata["alcada_correlation"])
}

func TestMessageSend_ReservedMetadata_Returns400(t *testing.T) {
	handler, msgRepo, convRepo, channelRepo, contactRepo, producer := setupMessageHandler()
	seedConversationSendDeps(convRepo, channelRepo, contactRepo)

	w := postConversationMessage(handler, SendMessageRequest{
		ContentType: "text",
		Content:     "Mensagem",
		Metadata:    map[string]string{"alcada_correlation": "ok", "campaign_id": "campanha-forjada"},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, msgRepo.Messages)
	assert.Empty(t, producer.OutboundMessages)
}

// ---------------------------------------------------------------------------
// List — paginação por cursor
// ---------------------------------------------------------------------------

// seedMessageAt adiciona uma mensagem com created_at explícito, para controlar
// a ordem (e os empates) que a paginação precisa respeitar.
func seedMessageAt(repo *testutil.MockMessageRepository, id, conversationID string, at time.Time) *entity.Message {
	msg := seedMessage(repo, id, conversationID)
	msg.CreatedAt = at
	return msg
}

// listPage chama o handler e devolve os ids da página, na ordem, mais o meta.
func listPage(t *testing.T, handler *MessageHandler, conversationID, before string, limit int) ([]string, *MetaResponse) {
	t.Helper()

	url := "/conversations/" + conversationID + "/messages?limit=" + strconv.Itoa(limit)
	if before != "" {
		url += "&before=" + before
	}

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: conversationID}}
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)

	handler.List(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Meta *MetaResponse `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	ids := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		ids = append(ids, m.ID)
	}
	return ids, resp.Meta
}

// Percorre a conversa inteira de página em página, como a tela faz ao rolar
// para cima: cada página traz as mais novas que restam, sem repetir nem pular.
func TestMessageList_CursorWalksWholeThread(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()
	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	base := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= 5; i++ {
		seedMessageAt(msgRepo, fmt.Sprintf("msg-%d", i), "conv-1", base.Add(time.Duration(i)*time.Minute))
	}

	page1, meta1 := listPage(t, handler, "conv-1", "", 2)
	assert.Equal(t, []string{"msg-5", "msg-4"}, page1, "a primeira página são as mais novas")
	require.NotNil(t, meta1)
	assert.True(t, meta1.HasNext)
	require.NotEmpty(t, meta1.NextCursor)
	assert.False(t, meta1.HasPrev, "sem cursor na entrada, não há página anterior")

	page2, meta2 := listPage(t, handler, "conv-1", meta1.NextCursor, 2)
	assert.Equal(t, []string{"msg-3", "msg-2"}, page2)
	require.NotNil(t, meta2)
	assert.True(t, meta2.HasNext)
	assert.True(t, meta2.HasPrev)

	page3, meta3 := listPage(t, handler, "conv-1", meta2.NextCursor, 2)
	assert.Equal(t, []string{"msg-1"}, page3)
	require.NotNil(t, meta3)
	assert.False(t, meta3.HasNext, "acabou a conversa")
	assert.Empty(t, meta3.NextCursor)

	vistos := append(append(page1, page2...), page3...)
	assert.Len(t, vistos, 5, "nenhuma mensagem repetida ou perdida nas três páginas")
}

// Mensagens que chegam no mesmo segundo compartilham created_at: o id é o que
// impede a fronteira da página de cair no meio do empate, repetindo uma e
// escondendo outra.
func TestMessageList_CursorHandlesTimestampTies(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()
	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	mesmoInstante := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"msg-a", "msg-b", "msg-c", "msg-d"} {
		seedMessageAt(msgRepo, id, "conv-1", mesmoInstante)
	}

	page1, meta1 := listPage(t, handler, "conv-1", "", 2)
	require.NotNil(t, meta1)
	require.True(t, meta1.HasNext)

	page2, _ := listPage(t, handler, "conv-1", meta1.NextCursor, 2)

	todas := append(append([]string{}, page1...), page2...)
	assert.ElementsMatch(t, []string{"msg-a", "msg-b", "msg-c", "msg-d"}, todas)
}

// Um cursor ilegível é erro do cliente. Começar do zero em silêncio faria um
// botão quebrado parecer o fim da conversa.
func TestMessageList_InvalidCursor_Returns400(t *testing.T) {
	handler, _, convRepo, _, _, _ := setupMessageHandler()
	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	c, w := newMessageAuthContext()
	c.Params = gin.Params{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations/conv-1/messages?before=nao-e-cursor", nil)

	handler.List(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// O limite pedido não pode virar um pedido da conversa inteira.
func TestMessageList_LimitIsCapped(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()
	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	base := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 120; i++ {
		seedMessageAt(msgRepo, fmt.Sprintf("msg-%03d", i), "conv-1", base.Add(time.Duration(i)*time.Second))
	}

	ids, meta := listPage(t, handler, "conv-1", "", 500)
	assert.Len(t, ids, 100, "o teto é 100 por página")
	require.NotNil(t, meta)
	assert.True(t, meta.HasNext)
}

func TestMessageCursor_RoundTrip(t *testing.T) {
	original := repository.MessageCursor{
		CreatedAt: time.Date(2026, 10, 9, 12, 34, 56, 789000000, time.UTC),
		ID:        "6f1c0a3e-0000-4000-8000-000000000001",
	}

	decoded, err := decodeMessageCursor(encodeMessageCursor(original))
	require.NoError(t, err)
	require.NotNil(t, decoded)
	assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
	assert.Equal(t, original.ID, decoded.ID)
}

func TestMessageCursor_EmptyMeansNewest(t *testing.T) {
	decoded, err := decodeMessageCursor("")
	require.NoError(t, err)
	assert.Nil(t, decoded, "sem cursor = começar pelas mais novas")
}

func TestMessageCursor_Rejects(t *testing.T) {
	casos := map[string]string{
		"não é base64":    "@@@",
		"sem separador":   base64.RawURLEncoding.EncodeToString([]byte("2026-10-09T12:00:00Z")),
		"sem id":          base64.RawURLEncoding.EncodeToString([]byte("2026-10-09T12:00:00Z|")),
		"data impossível": base64.RawURLEncoding.EncodeToString([]byte("ontem|msg-1")),
	}
	for nome, cursor := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := decodeMessageCursor(cursor)
			assert.Error(t, err)
		})
	}
}
