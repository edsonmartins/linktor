package whatsapp_official

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage: error paths for endpoints that previously only had a
// success-path test. See the test coverage audit for context.

func TestClient_UpdateBusinessProfile_Success(t *testing.T) {
	var capturedPath, capturedMethod string
	var body []byte

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	require.NoError(t, client.UpdateBusinessProfile(context.Background(), &BusinessProfile{
		About:       "We sell great things",
		Address:     "Rua Exemplo 123",
		Description: "Ecommerce brasileira",
		Email:       "ops@linktor.dev",
		Vertical:    "RETAIL",
	}))
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Contains(t, capturedPath, "/12345/whatsapp_business_profile")
	assert.Contains(t, string(body), "messaging_product")
	assert.Contains(t, string(body), "whatsapp")
}

func TestClient_UpdateBusinessProfile_APIError(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid vertical","code":100}}`))
	})
	defer server.Close()

	err := client.UpdateBusinessProfile(context.Background(), &BusinessProfile{Vertical: "NONSENSE"})
	require.Error(t, err)
}

func TestClient_GetPhoneNumberInfo_Success(t *testing.T) {
	var capturedPath string
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"12345","display_phone_number":"+55 11 99999-9999","verified_name":"Linktor","quality_rating":"GREEN"}`))
	})
	defer server.Close()

	info, err := client.GetPhoneNumberInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/v21.0/12345", capturedPath)
	assert.Equal(t, "Linktor", info.VerifiedName)
	assert.Equal(t, "GREEN", info.QualityRating)
}

func TestClient_GetPhoneNumberInfo_APIError(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"token expired","code":190}}`))
	})
	defer server.Close()

	_, err := client.GetPhoneNumberInfo(context.Background())
	require.Error(t, err)
}

func TestClient_GetHealth_Success(t *testing.T) {
	var capturedPath string
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"67890","health_status":{"can_send_message":"AVAILABLE"}}`))
	})
	defer server.Close()

	_, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/v21.0/67890", capturedPath)
}

func TestClient_GetHealth_APIError(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"service unavailable"}}`))
	})
	defer server.Close()

	_, err := client.GetHealth(context.Background())
	require.Error(t, err)
}

func TestClient_DeleteMedia_APIError(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"media not found","code":100}}`))
	})
	defer server.Close()

	err := client.DeleteMedia(context.Background(), "missing-id")
	require.Error(t, err)
}
