package officialcalls

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c, err := NewClient(Config{
		BaseURL:       srv.URL,
		APIVersion:    "v23.0",
		AccessToken:   "TOKEN",
		PhoneNumberID: "phone-55",
		HTTPClient:    srv.Client(),
	})
	require.NoError(t, err)
	return c, srv
}

func TestNewClient_Validation(t *testing.T) {
	_, err := NewClient(Config{PhoneNumberID: "p"})
	assert.Error(t, err)
	_, err = NewClient(Config{AccessToken: "t"})
	assert.Error(t, err)
}

func TestEnableCalling(t *testing.T) {
	var gotPath, gotAuth string
	var body settingsRequest
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer srv.Close()

	require.NoError(t, c.EnableCalling(context.Background()))
	assert.Equal(t, "/v23.0/phone-55/settings", gotPath)
	assert.Equal(t, "Bearer TOKEN", gotAuth)
	require.NotNil(t, body.Calling)
	assert.Equal(t, CallingEnabled, body.Calling.Status)
}

func TestConnect_SendsOfferAndReturnsID(t *testing.T) {
	var req callActionRequest
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v23.0/phone-55/calls", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &req)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","calls":[{"id":"call-xyz"}]}`))
	})
	defer srv.Close()

	id, err := c.Connect(context.Background(), "5511999999999", "v=0 offer-sdp")
	require.NoError(t, err)
	assert.Equal(t, "call-xyz", id)
	assert.Equal(t, ActionConnect, req.Action)
	assert.Equal(t, "5511999999999", req.To)
	require.NotNil(t, req.Session)
	assert.Equal(t, SDPOffer, req.Session.SDPType)
	assert.Equal(t, "v=0 offer-sdp", req.Session.SDP)
}

func TestAccept_SendsAnswer(t *testing.T) {
	var req callActionRequest
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &req)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp"}`))
	})
	defer srv.Close()

	require.NoError(t, c.Accept(context.Background(), "call-1", "v=0 answer-sdp"))
	assert.Equal(t, ActionAccept, req.Action)
	assert.Equal(t, "call-1", req.CallID)
	require.NotNil(t, req.Session)
	assert.Equal(t, SDPAnswer, req.Session.SDPType)
}

func TestRejectAndTerminate(t *testing.T) {
	var actions []CallAction
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req callActionRequest
		_ = json.Unmarshal(raw, &req)
		actions = append(actions, req.Action)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	require.NoError(t, c.Reject(context.Background(), "c1"))
	require.NoError(t, c.Terminate(context.Background(), "c1"))
	assert.Equal(t, []CallAction{ActionReject, ActionTerminate}, actions)
}

func TestGetCallPermission(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v23.0/phone-55/call_permissions", r.URL.Path)
		assert.Equal(t, "5511999999999", r.URL.Query().Get("user_wa_id"))
		_, _ = w.Write([]byte(`{"data":[{"status":"temporary"}]}`))
	})
	defer srv.Close()

	perm, err := c.GetCallPermission(context.Background(), "5511999999999")
	require.NoError(t, err)
	assert.Equal(t, "temporary", perm.Status)
	assert.True(t, perm.CanPlaceCall)
}

func TestGraphErrorMapping(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"calling not enabled","code":131049}}`))
	})
	defer srv.Close()

	err := c.Terminate(context.Background(), "c1")
	require.Error(t, err)
	var ge *GraphError
	require.ErrorAs(t, err, &ge)
	assert.Equal(t, 131049, ge.Code)
	assert.Equal(t, http.StatusBadRequest, ge.HTTPStatus)
}
