package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
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
	handler, msgRepo, _, _, _, _ := setupMessageHandler()

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
	assert.Equal(t, int64(2), resp.Meta.TotalItems)

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

// ---------------------------------------------------------------------------
// Send
// ---------------------------------------------------------------------------

func TestMessageSend_ValidRequest_Returns201(t *testing.T) {
	handler, _, convRepo, channelRepo, contactRepo, _ := setupMessageHandler()

	// Seed conversation, channel, and contact for the Send flow
	now := time.Now()
	convRepo.Conversations["conv-1"] = &entity.Conversation{
		ID:        "conv-1",
		TenantID:  "tenant-1",
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Status:    entity.ConversationStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
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

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestMessageGet_ValidID_Returns200(t *testing.T) {
	handler, msgRepo, _, _, _, _ := setupMessageHandler()

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

// ---------------------------------------------------------------------------
// SendReaction
// ---------------------------------------------------------------------------

func TestMessageSendReaction_Valid_Returns200(t *testing.T) {
	handler, msgRepo, convRepo, _, _, _ := setupMessageHandler()

	now := time.Now()
	convRepo.Conversations["conv-1"] = &entity.Conversation{
		ID:        "conv-1",
		TenantID:  "tenant-1",
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Status:    entity.ConversationStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
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

	now := time.Now()
	convRepo.Conversations["conv-1"] = &entity.Conversation{
		ID:        "conv-1",
		TenantID:  "tenant-1",
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Status:    entity.ConversationStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
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
