package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProducer() *WebhookProducer {
	return NewWebhookProducer(
		WithBackoffDelays([]time.Duration{0, 0, 0, 0, 0}),
	)
}

func TestNewWebhookProducer(t *testing.T) {
	p := NewWebhookProducer()
	assert.NotNil(t, p)
	assert.NotNil(t, p.httpClient)
	assert.Equal(t, defaultBackoffDelays, p.backoffDelays)
}

func TestNewWebhookProducer_WithOptions(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	delays := []time.Duration{0, time.Second}
	p := NewWebhookProducer(WithHTTPClient(client), WithBackoffDelays(delays))
	assert.Equal(t, client, p.httpClient)
	assert.Equal(t, delays, p.backoffDelays)
}

func TestDeliver_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	p := testProducer()
	result, err := p.Deliver(context.Background(), EndpointConfig{
		URL:        server.URL,
		MaxRetries: 3,
	}, "message.inbound", []byte(`{"test":true}`))

	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, 1, result.Attempt)
	assert.Empty(t, result.Error)
	assert.Contains(t, result.ResponseBody, "ok")
}

func TestDeliver_RetryOnServerError(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&count, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := testProducer()
	result, err := p.Deliver(context.Background(), EndpointConfig{
		URL:        server.URL,
		MaxRetries: 5,
	}, "message.inbound", []byte(`{}`))

	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, 3, result.Attempt)
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}

func TestDeliver_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	p := testProducer()
	result, err := p.Deliver(context.Background(), EndpointConfig{
		URL:        server.URL,
		MaxRetries: 3,
	}, "message.inbound", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 attempts")
	assert.Equal(t, 502, result.StatusCode)
	assert.Equal(t, 3, result.Attempt)
}

func TestDeliver_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Use real delays but cancel context quickly
	p := NewWebhookProducer(WithBackoffDelays([]time.Duration{0, 5 * time.Second}))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.Deliver(ctx, EndpointConfig{
		URL:        server.URL,
		MaxRetries: 5,
	}, "test", []byte(`{}`))

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDeliver_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "sha256=abc123", r.Header.Get("X-Signature"))
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := testProducer()
	result, err := p.Deliver(context.Background(), EndpointConfig{
		URL: server.URL,
		Headers: map[string]string{
			"X-Signature":   "sha256=abc123",
			"Authorization": "Bearer token123",
		},
		MaxRetries: 1,
	}, "message.inbound", []byte(`{}`))

	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
}

func TestDeliver_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := testProducer()
	_, err := p.Deliver(context.Background(), EndpointConfig{
		URL:            server.URL,
		MaxRetries:     1,
		TimeoutSeconds: 1,
	}, "test", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 1 attempts")
}

func TestDeliver_InvalidURL(t *testing.T) {
	p := testProducer()
	_, err := p.Deliver(context.Background(), EndpointConfig{
		URL:        "http://invalid-host-that-does-not-exist.local:99999",
		MaxRetries: 1,
	}, "test", []byte(`{}`))

	require.Error(t, err)
}

func TestShouldDeliver_EmptySubscription(t *testing.T) {
	p := testProducer()
	assert.True(t, p.ShouldDeliver(EndpointConfig{}, "message.inbound"))
	assert.True(t, p.ShouldDeliver(EndpointConfig{SubscribedEvents: []string{}}, "anything"))
}

func TestShouldDeliver_MatchingEvent(t *testing.T) {
	p := testProducer()
	endpoint := EndpointConfig{
		SubscribedEvents: []string{"message.inbound", "message.outbound"},
	}
	assert.True(t, p.ShouldDeliver(endpoint, "message.inbound"))
	assert.True(t, p.ShouldDeliver(endpoint, "message.outbound"))
}

func TestShouldDeliver_NonMatchingEvent(t *testing.T) {
	p := testProducer()
	endpoint := EndpointConfig{
		SubscribedEvents: []string{"message.inbound"},
	}
	assert.False(t, p.ShouldDeliver(endpoint, "conversation.created"))
}

func TestShouldDeliver_Wildcard(t *testing.T) {
	p := testProducer()
	endpoint := EndpointConfig{
		SubscribedEvents: []string{"*"},
	}
	assert.True(t, p.ShouldDeliver(endpoint, "anything"))
	assert.True(t, p.ShouldDeliver(endpoint, "message.inbound"))
}

func TestDeliver_DefaultMaxRetries(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := testProducer()
	_, err := p.Deliver(context.Background(), EndpointConfig{
		URL: server.URL,
		// MaxRetries = 0, should use len(backoffDelays) = 5
	}, "test", []byte(`{}`))

	require.Error(t, err)
	assert.Equal(t, int32(5), atomic.LoadInt32(&count))
}

func TestDeliveryResult_Fields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("accepted"))
	}))
	defer server.Close()

	p := testProducer()
	result, err := p.Deliver(context.Background(), EndpointConfig{
		URL:        server.URL,
		MaxRetries: 1,
	}, "test", []byte(`{"data":"value"}`))

	require.NoError(t, err)
	assert.Equal(t, server.URL, result.URL)
	assert.Equal(t, 202, result.StatusCode)
	assert.Equal(t, "accepted", result.ResponseBody)
	assert.Equal(t, 1, result.Attempt)
	assert.False(t, result.DeliveredAt.IsZero())
}
