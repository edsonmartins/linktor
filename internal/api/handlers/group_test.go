package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupHandler(t *testing.T) {
	h := NewGroupHandler()
	require.NotNil(t, h)
}

func TestGroupHandler_List(t *testing.T) {
	h := NewGroupHandler()
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
	h := NewGroupHandler()
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
	h := NewGroupHandler()
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
	h := NewGroupHandler()

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
	h := NewGroupHandler()
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
	h := NewGroupHandler()
	w, c := newTestContext(http.MethodPost, "/api/v1/groups/group-1/participants", map[string]interface{}{
		"participants": []string{"user3"},
	})
	c.Params = gin.Params{{Key: "id", Value: "group-1"}}

	h.UpdateParticipants(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGroupHandler_GetInviteLink(t *testing.T) {
	h := NewGroupHandler()
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
	h := NewGroupHandler()
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
