package calling

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rewriteTransport struct {
	baseURL string
	rt      http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.TrimPrefix(t.baseURL, "http://")
	req.URL.Scheme = "http"
	req.URL.Host = host
	return t.rt.RoundTrip(req)
}

func newHTTPTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	c := NewClient(&ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "phone-55",
		APIVersion:    "v23.0",
	})
	c.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}
	return c, server
}

// -----------------------------------------------------------------------------
// InitiateCall
// -----------------------------------------------------------------------------

func TestClient_InitiateCall_Success(t *testing.T) {
	var capturedPath, capturedMethod, capturedAuth string
	var captured map[string]interface{}

	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"calls":[{"id":"call-123"}]}`))
	})
	defer server.Close()

	resp, err := client.InitiateCall(context.Background(), &InitiateCallRequest{
		To:      "+15551234567",
		Type:    CallTypeVoice,
		Timeout: 30,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "call-123", resp.CallID)
	assert.Equal(t, CallStatusInitiated, resp.Status)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/phone-55/calls", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)

	assert.Equal(t, "whatsapp", captured["messaging_product"])
	assert.Equal(t, "+15551234567", captured["to"])
	assert.Equal(t, "voice", captured["type"])
	assert.Equal(t, float64(30), captured["timeout"])

	// Call should be stored internally
	call, ok := client.GetCall("call-123")
	require.True(t, ok)
	assert.Equal(t, CallDirectionOutbound, call.Direction)
}

func TestClient_InitiateCall_APIError(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"calling not enabled","code":131049}}`))
	})
	defer server.Close()

	_, err := client.InitiateCall(context.Background(), &InitiateCallRequest{
		To:   "+15551234567",
		Type: CallTypeVoice,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestClient_InitiateCall_NoCallIDInResponse(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"calls":[]}`))
	})
	defer server.Close()

	_, err := client.InitiateCall(context.Background(), &InitiateCallRequest{
		To: "+15551234567", Type: CallTypeVoice,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no call ID")
}

// -----------------------------------------------------------------------------
// EndCall
// -----------------------------------------------------------------------------

func TestClient_EndCall_Success(t *testing.T) {
	var capturedPath, capturedMethod string
	var captured map[string]interface{}

	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	// Seed an active call
	client.calls["call-xyz"] = &Call{ID: "call-xyz", Status: CallStatusConnected}

	require.NoError(t, client.EndCall(context.Background(), "call-xyz"))
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/phone-55/calls/call-xyz", capturedPath)
	assert.Equal(t, "end", captured["action"])

	call, ok := client.GetCall("call-xyz")
	require.True(t, ok)
	assert.Equal(t, CallStatusCompleted, call.Status)
}

func TestClient_EndCall_APIError(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"call already ended"}}`))
	})
	defer server.Close()

	err := client.EndCall(context.Background(), "call-xyz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}
