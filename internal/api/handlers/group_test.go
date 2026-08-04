package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupHandler(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	require.NotNil(t, h)
}

// A send to a group JID publishes an outbound with RecipientID = the group JID,
// so the whatsmeow adapter delivers it to the group without contact resolution.
func TestGroupSendMessage_PublishesToGroupJID(t *testing.T) {
	channelRepo := testutil.NewMockChannelRepository()
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID:       "ch-1",
		TenantID: "tenant-1",
		Type:     entity.ChannelTypeWhatsApp,
	}
	producer := testutil.NewMockProducer()
	channelSvc := service.NewChannelService(channelRepo, plugin.GetGlobalRegistry(), producer)
	h := NewGroupHandler(producer, channelSvc)

	body, _ := json.Marshal(GroupSendRequest{Text: "aviso do bot"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}, {Key: "groupId", Value: "120363000000000000@g.us"}}
	c.Request = httptest.NewRequest(http.MethodPost,
		"/channels/ch-1/groups/120363000000000000@g.us/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.SendMessage(c)

	assert.Equal(t, http.StatusAccepted, w.Code)
	require.Len(t, producer.OutboundMessages, 1)
	out := producer.OutboundMessages[0]
	assert.Equal(t, "120363000000000000@g.us", out.RecipientID, "endereça o grupo pelo JID")
	assert.Equal(t, "ch-1", out.ChannelID)
	assert.Equal(t, string(entity.ChannelTypeWhatsApp), out.ChannelType)
	assert.Equal(t, "aviso do bot", out.Content)
}

// A send for a channel that does not belong to the tenant must not publish.
func TestGroupSendMessage_UnknownChannel(t *testing.T) {
	channelRepo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	channelSvc := service.NewChannelService(channelRepo, plugin.GetGlobalRegistry(), producer)
	h := NewGroupHandler(producer, channelSvc)

	body, _ := json.Marshal(GroupSendRequest{Text: "x"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nope"}, {Key: "groupId", Value: "1@g.us"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/channels/nope/groups/1@g.us/messages",
		bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.SendMessage(c)

	assert.NotEqual(t, http.StatusAccepted, w.Code)
	assert.Empty(t, producer.OutboundMessages, "canal desconhecido não publica")
}

func TestGroupHandler_List(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodGet, "/api/v1/groups", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	// Data should be an empty array
	data, ok := resp.Data.([]interface{})
	assert.True(t, ok)
	assert.Empty(t, data)
}

func TestGroupHandler_Get(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodGet, "/api/v1/groups/group-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "group-1"}}

	h.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "group-1", data["id"])
	assert.Equal(t, "Placeholder Group", data["name"])
}

func TestGroupHandler_Create_ValidRequest(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodPost, "/api/v1/groups", GroupCreateRequest{
		ChannelID:    "channel-1",
		Name:         "Test Group",
		Description:  "A test group",
		Participants: []string{"user1", "user2"},
	})

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Group created", data["message"])

	group, ok := data["group"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "channel-1", group["channel_id"])
	assert.Equal(t, "Test Group", group["name"])
	assert.NotEmpty(t, group["id"])
}

func TestGroupHandler_Create_MissingFields(t *testing.T) {
	h := NewGroupHandler(nil, nil)

	// Missing name
	w, c := newTestContext(http.MethodPost, "/api/v1/groups", map[string]string{
		"channel_id": "channel-1",
	})
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Missing channel_id
	w2, c2 := newTestContext(http.MethodPost, "/api/v1/groups", map[string]string{
		"name": "Test Group",
	})
	h.Create(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestGroupHandler_UpdateParticipants_Valid(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodPost, "/api/v1/groups/group-1/participants", GroupParticipantRequest{
		Participants: []string{"user3"},
		Action:       "add",
	})
	c.Params = gin.Params{{Key: "id", Value: "group-1"}}

	h.UpdateParticipants(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Participants updated", data["message"])
	assert.Equal(t, "add", data["action"])
}

func TestGroupHandler_UpdateParticipants_MissingAction(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodPost, "/api/v1/groups/group-1/participants", map[string]interface{}{
		"participants": []string{"user3"},
	})
	c.Params = gin.Params{{Key: "id", Value: "group-1"}}

	h.UpdateParticipants(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGroupHandler_GetInviteLink(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodGet, "/api/v1/groups/group-1/invite", nil)
	c.Params = gin.Params{{Key: "id", Value: "group-1"}}

	h.GetInviteLink(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "group-1", data["group_id"])
	assert.Contains(t, data["invite_link"], "group-1")
}

func TestGroupHandler_Leave(t *testing.T) {
	h := NewGroupHandler(nil, nil)
	w, c := newTestContext(http.MethodPost, "/api/v1/groups/group-1/leave", nil)
	c.Params = gin.Params{{Key: "id", Value: "group-1"}}

	h.Leave(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Left group successfully", data["message"])
	assert.Equal(t, "group-1", data["group_id"])
}
