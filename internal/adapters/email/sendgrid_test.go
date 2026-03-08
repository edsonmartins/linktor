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

func TestNewSendGridProvider(t *testing.T) {
	cfg := &Config{
		Provider:       ProviderSendGrid,
		FromEmail:      "test@example.com",
		SendGridAPIKey: "SG.testkey",
	}
	provider, err := NewSendGridProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotNil(t, provider.httpClient)
}

func TestSendGridProvider_ValidateConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
		})
		assert.NoError(t, p.ValidateConfig())
	})

	t.Run("missing api_key", func(t *testing.T) {
		p, _ := NewSendGridProvider(&Config{
			Provider:  ProviderSendGrid,
			FromEmail: "test@example.com",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sendgrid_api_key is required")
	})

	t.Run("missing from_email", func(t *testing.T) {
		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			SendGridAPIKey: "SG.testkey",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from_email is required")
	})
}

func TestSendGridProvider_GetProviderName(t *testing.T) {
	p, _ := NewSendGridProvider(&Config{})
	assert.Equal(t, "sendgrid", p.GetProviderName())
}

func TestSendGridProvider_TestConnection(t *testing.T) {
	t.Run("200 success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "Bearer SG.testkey", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"scopes":["mail.send"]}`))
		}))
		defer server.Close()

		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
		})
		p.httpClient = server.Client()

		// We need to override the URL used in TestConnection.
		// Since TestConnection hardcodes the URL, we create a custom transport.
		p.httpClient.Transport = &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		}

		err := p.TestConnection(context.Background())
		assert.NoError(t, err)
	})

	t.Run("401 auth error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.badkey",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		}

		err := p.TestConnection(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})
}

func TestSendGridProvider_Send(t *testing.T) {
	t.Run("202 success with X-Message-Id header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "Bearer SG.testkey", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var payload map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&payload)
			assert.NoError(t, err)
			assert.Equal(t, "Test Subject", payload["subject"])

			w.Header().Set("X-Message-Id", "sendgrid-msg-123")
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
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
		assert.Equal(t, "sendgrid-msg-123", result.ExternalID)
	})

	t.Run("text email", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			content := payload["content"].([]interface{})
			assert.Len(t, content, 1)
			first := content[0].(map[string]interface{})
			assert.Equal(t, "text/plain", first["type"])
			assert.Equal(t, "Plain text", first["value"])

			w.Header().Set("X-Message-Id", "msg-text")
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{base: server.Client().Transport, baseURL: server.URL}

		result, err := p.Send(context.Background(), &OutboundEmail{
			To:       []string{"to@example.com"},
			Subject:  "Test",
			TextBody: "Plain text",
		})
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("HTML email", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			content := payload["content"].([]interface{})
			found := false
			for _, c := range content {
				cm := c.(map[string]interface{})
				if cm["type"] == "text/html" {
					assert.Equal(t, "<h1>Hello</h1>", cm["value"])
					found = true
				}
			}
			assert.True(t, found, "HTML content should be present")

			w.Header().Set("X-Message-Id", "msg-html")
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{base: server.Client().Transport, baseURL: server.URL}

		result, err := p.Send(context.Background(), &OutboundEmail{
			To:       []string{"to@example.com"},
			Subject:  "Test",
			HTMLBody: "<h1>Hello</h1>",
		})
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("with attachments", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			attachments := payload["attachments"].([]interface{})
			assert.Len(t, attachments, 1)
			att := attachments[0].(map[string]interface{})
			assert.Equal(t, "test.pdf", att["filename"])
			assert.Equal(t, "application/pdf", att["type"])

			w.Header().Set("X-Message-Id", "msg-att")
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		p, _ := NewSendGridProvider(&Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
		})
		p.httpClient = server.Client()
		p.httpClient.Transport = &rewriteTransport{base: server.Client().Transport, baseURL: server.URL}

		result, err := p.Send(context.Background(), &OutboundEmail{
			To:       []string{"to@example.com"},
			Subject:  "Test",
			TextBody: "See attached",
			Attachments: []*Attachment{
				{
					Filename:    "test.pdf",
					ContentType: "application/pdf",
					Content:     []byte("pdf-content"),
				},
			},
		})
		require.NoError(t, err)
		assert.True(t, result.Success)
	})
}

// rewriteTransport rewrites all request URLs to point to the test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the request URL to the test server, preserving the path
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[len("http://"):]
	if t.base != nil {
		return t.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
