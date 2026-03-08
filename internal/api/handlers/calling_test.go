package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCallingHandler(t *testing.T) {
	h := NewCallingHandler()
	require.NotNil(t, h)
	assert.NotNil(t, h.clients)
}

func TestCallingHandler_InitiateCall_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/calls", map[string]string{"to": "+5511999999999"})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.InitiateCall(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not found")
}

func TestCallingHandler_InitiateCall_InvalidBody(t *testing.T) {
	// Even without a client, the no-client check happens first, so this also returns 404
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/calls", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.InitiateCall(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_GetCall_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/calls/call-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "callId", Value: "call-1"}}

	h.GetCall(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not found")
}

func TestCallingHandler_EndCall_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/calls/call-1/end", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "callId", Value: "call-1"}}

	h.EndCall(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_GetCallStats_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/calls/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetCallStats(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_GetRecentCalls_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/calls", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetRecentCalls(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_GetCallsByPhone_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/calls/phone/+5511999999999", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "phone", Value: "+5511999999999"}}

	h.GetCallsByPhone(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_GetCallQuality_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/calls/call-1/quality", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "callId", Value: "call-1"}}

	h.GetCallQuality(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_GetCallRecording_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/calls/call-1/recording", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "callId", Value: "call-1"}}

	h.GetCallRecording(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCallingHandler_HandleWebhook_NoClient(t *testing.T) {
	h := NewCallingHandler()
	w, c := newTestContext(http.MethodPost, "/webhooks/calls/channel-1", map[string]string{"event": "call.ended"})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.HandleWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
