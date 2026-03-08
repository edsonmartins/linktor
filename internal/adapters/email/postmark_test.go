package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostmarkProvider(t *testing.T) {
	cfg := &Config{
		Provider:            ProviderPostmark,
		FromEmail:           "test@example.com",
		PostmarkServerToken: "token123",
	}
	provider, err := NewPostmarkProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotNil(t, provider.httpClient)
}

func TestPostmarkProvider_ValidateConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p, _ := NewPostmarkProvider(&Config{
			Provider:            ProviderPostmark,
			FromEmail:           "test@example.com",
			PostmarkServerToken: "token123",
		})
		assert.NoError(t, p.ValidateConfig())
	})

	t.Run("missing server_token", func(t *testing.T) {
		p, _ := NewPostmarkProvider(&Config{
			Provider:  ProviderPostmark,
			FromEmail: "test@example.com",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postmark_server_token is required")
	})

	t.Run("missing from_email", func(t *testing.T) {
		p, _ := NewPostmarkProvider(&Config{
			Provider:            ProviderPostmark,
			PostmarkServerToken: "token123",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from_email is required")
	})
}

func TestPostmarkProvider_GetProviderName(t *testing.T) {
	p, _ := NewPostmarkProvider(&Config{})
	assert.Equal(t, "postmark", p.GetProviderName())
}

func TestPostmarkProvider_TestConnection(t *testing.T) {
	t.Run("200 success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "token123", r.Header.Get("X-Postmark-Server-Token"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"Name":"Test Server"}`))
		}))
		defer server.Close()

		p, _ := NewPostmarkProvider(&Config{
			Provider:            ProviderPostmark,
			FromEmail:           "test@example.com",
			PostmarkServerToken: "token123",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		}

		err := p.TestConnection(context.Background())
		assert.NoError(t, err)
	})

	t.Run("401 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		p, _ := NewPostmarkProvider(&Config{
			Provider:            ProviderPostmark,
			FromEmail:           "test@example.com",
			PostmarkServerToken: "badtoken",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		}

		err := p.TestConnection(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server token")
	})
}

func TestPostmarkProvider_Send(t *testing.T) {
	t.Run("200 success with MessageID and SubmittedAt", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "token123", r.Header.Get("X-Postmark-Server-Token"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var payload map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&payload)
			assert.NoError(t, err)
			assert.Equal(t, "Test Subject", payload["Subject"])

			resp := map[string]interface{}{
				"To":          "to@example.com",
				"SubmittedAt": "2021-01-01T00:00:00Z",
				"MessageID":   "pm-msg-123",
				"ErrorCode":   0,
				"Message":     "OK",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p, _ := NewPostmarkProvider(&Config{
			Provider:            ProviderPostmark,
			FromEmail:           "test@example.com",
			PostmarkServerToken: "token123",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		}

		result, err := p.Send(context.Background(), &OutboundEmail{
			To:       []string{"to@example.com"},
			Subject:  "Test Subject",
			TextBody: "Hello",
		})
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "pm-msg-123", result.ExternalID)
		assert.Equal(t, "pm-msg-123", result.MessageID)
	})

	t.Run("API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"ErrorCode":300,"Message":"Invalid 'From' address"}`))
		}))
		defer server.Close()

		p, _ := NewPostmarkProvider(&Config{
			Provider:            ProviderPostmark,
			FromEmail:           "test@example.com",
			PostmarkServerToken: "token123",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		}

		result, err := p.Send(context.Background(), &OutboundEmail{
			To:       []string{"to@example.com"},
			Subject:  "Test",
			TextBody: "Hello",
		})
		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "Postmark API error")
	})
}
