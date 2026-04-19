package payments

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
// sendPaymentMessage (private — tested directly from the same package)
// -----------------------------------------------------------------------------

func TestClient_SendPaymentMessage_Success(t *testing.T) {
	var capturedPath, capturedMethod, capturedAuth string
	var captured map[string]interface{}

	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.pay-1"}]}`))
	})
	defer server.Close()

	req := &PaymentRequest{
		To:          "+15551234567",
		ReferenceID: "ref-42",
		Amount:      9999,
		Currency:    "USD",
		Description: "Premium plan",
		Items: []PaymentItem{
			{Name: "Subscription", Quantity: 1, TotalPrice: 9999, Currency: "USD"},
		},
	}
	resp := &PaymentResponse{PaymentID: "pay-1"}

	id, err := client.sendPaymentMessage(context.Background(), req, resp)
	require.NoError(t, err)
	assert.Equal(t, "wamid.pay-1", id)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)

	assert.Equal(t, "whatsapp", captured["messaging_product"])
	assert.Equal(t, "interactive", captured["type"])
	inter := captured["interactive"].(map[string]interface{})
	assert.Equal(t, "order_details", inter["type"])
	action := inter["action"].(map[string]interface{})
	assert.Equal(t, "review_and_pay", action["name"])
	params := action["parameters"].(map[string]interface{})
	assert.Equal(t, "ref-42", params["reference_id"])
	assert.Equal(t, "pending", params["payment_status"])

	order := params["order"].(map[string]interface{})
	items := order["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, "Subscription", item["name"])

	subtotal := order["subtotal"].(map[string]interface{})
	assert.Equal(t, float64(9999), subtotal["value"])
	assert.Equal(t, float64(100), subtotal["offset"])
}

func TestClient_SendPaymentMessage_APIError(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid reference","code":131051}}`))
	})
	defer server.Close()

	_, err := client.sendPaymentMessage(context.Background(),
		&PaymentRequest{To: "+15551234567", ReferenceID: "r", Amount: 100, Currency: "USD"},
		&PaymentResponse{PaymentID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestClient_SendPaymentMessage_NoMessageID(t *testing.T) {
	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[]}`))
	})
	defer server.Close()

	_, err := client.sendPaymentMessage(context.Background(),
		&PaymentRequest{To: "+15551234567", ReferenceID: "r", Amount: 100, Currency: "USD"},
		&PaymentResponse{PaymentID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no message ID")
}

// -----------------------------------------------------------------------------
// sendRefundNotification (fire-and-forget — verify it hits the right URL)
// -----------------------------------------------------------------------------

func TestClient_SendRefundNotification_HitsMessagesEndpoint(t *testing.T) {
	var capturedPath string
	var captured map[string]interface{}
	hit := make(chan struct{}, 1)

	client, server := newHTTPTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.refund-1"}]}`))
		hit <- struct{}{}
	})
	defer server.Close()

	payment := &Payment{
		ID:            "pay-1",
		ReferenceID:   "ref-42",
		CustomerPhone: "+15551234567",
		Amount:        5000,
		Currency:      "USD",
	}
	refund := &Refund{
		ID:       "refund-1",
		Amount:   5000,
		Currency: "USD",
	}

	client.sendRefundNotification(context.Background(), payment, refund)

	<-hit
	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)
	assert.Equal(t, "whatsapp", captured["messaging_product"])
	assert.Equal(t, "text", captured["type"])
	text := captured["text"].(map[string]interface{})["body"].(string)
	assert.Contains(t, text, "refund")
	assert.Contains(t, text, "ref-42")
}

// -----------------------------------------------------------------------------
// CreatePayment — full integration: gateway + WhatsApp message + store
// -----------------------------------------------------------------------------

func TestClient_CreatePayment_SendsMessage(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.create-1"}]}`))
	}))
	defer server.Close()

	store := newMockPaymentStore()
	client := NewClient(&ClientConfig{
		AccessToken:    "test-token",
		PhoneNumberID:  "phone-55",
		APIVersion:     "v23.0",
		Store:          store,
		OrganizationID: "org-1",
		ChannelID:      "chan-1",
		GatewayConfig: &GatewayConfig{
			// Type left zero-valued to fall through to NewMockGateway
		},
	})
	client.httpClient = &http.Client{Transport: &rewriteTransport{baseURL: server.URL, rt: http.DefaultTransport}}

	resp, err := client.CreatePayment(context.Background(), &PaymentRequest{
		To:          "+15551234567",
		ReferenceID: "ref-integration",
		Type:        PaymentTypeOrder,
		Amount:      5000,
		Currency:    "USD",
		Description: "Test payment",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "/v23.0/phone-55/messages", capturedPath)
	assert.Equal(t, "wamid.create-1", resp.MessageID)

	// The payment should be stored with the message ID
	stored, err := store.GetByReference(context.Background(), "ref-integration")
	require.NoError(t, err)
	assert.Equal(t, "wamid.create-1", stored.MessageID)
}
