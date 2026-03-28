package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Default backoff delays for retry attempts
var defaultBackoffDelays = []time.Duration{
	0,               // immediate
	5 * time.Second, // 5s
	30 * time.Second, // 30s
	2 * time.Minute,  // 2m
	10 * time.Minute, // 10m
}

// EndpointConfig configures a webhook delivery endpoint
type EndpointConfig struct {
	URL              string
	Headers          map[string]string // custom headers (e.g., X-Signature)
	SubscribedEvents []string          // filter events; empty = all
	MaxRetries       int               // default 5
	TimeoutSeconds   int               // per-request timeout; default 30
}

// DeliveryResult tracks a webhook delivery attempt
type DeliveryResult struct {
	WebhookID    string    `json:"webhook_id"`
	URL          string    `json:"url"`
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `json:"response_body,omitempty"`
	Attempt      int       `json:"attempt"`
	Error        string    `json:"error,omitempty"`
	DeliveredAt  time.Time `json:"delivered_at"`
}

// WebhookProducer delivers webhook events to external URLs with retry
type WebhookProducer struct {
	httpClient    *http.Client
	backoffDelays []time.Duration
}

// Option configures the WebhookProducer
type Option func(*WebhookProducer)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(p *WebhookProducer) {
		p.httpClient = client
	}
}

// WithBackoffDelays sets custom backoff delays (useful for testing)
func WithBackoffDelays(delays []time.Duration) Option {
	return func(p *WebhookProducer) {
		p.backoffDelays = delays
	}
}

// NewWebhookProducer creates a new producer with configurable options
func NewWebhookProducer(opts ...Option) *WebhookProducer {
	p := &WebhookProducer{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		backoffDelays: defaultBackoffDelays,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ShouldDeliver checks if the endpoint is subscribed to this event type.
// An empty SubscribedEvents list means deliver all events.
func (p *WebhookProducer) ShouldDeliver(endpoint EndpointConfig, eventType string) bool {
	if len(endpoint.SubscribedEvents) == 0 {
		return true
	}
	for _, e := range endpoint.SubscribedEvents {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}

// Deliver sends a webhook payload with exponential backoff retry
func (p *WebhookProducer) Deliver(ctx context.Context, endpoint EndpointConfig, eventType string, payload []byte) (*DeliveryResult, error) {
	maxRetries := endpoint.MaxRetries
	if maxRetries <= 0 {
		maxRetries = len(p.backoffDelays)
	}

	timeout := time.Duration(endpoint.TimeoutSeconds) * time.Second
	if endpoint.TimeoutSeconds <= 0 {
		timeout = 30 * time.Second
	}

	var lastResult *DeliveryResult

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Apply backoff delay (skip for first attempt)
		if attempt > 0 && attempt-1 < len(p.backoffDelays) {
			delay := p.backoffDelays[attempt]
			if delay > 0 {
				select {
				case <-ctx.Done():
					return lastResult, ctx.Err()
				case <-time.After(delay):
				}
			}
		} else if attempt > 0 && attempt-1 >= len(p.backoffDelays) {
			// Use the last backoff delay for subsequent attempts
			delay := p.backoffDelays[len(p.backoffDelays)-1]
			select {
			case <-ctx.Done():
				return lastResult, ctx.Err()
			case <-time.After(delay):
			}
		}

		result := p.doRequest(ctx, endpoint, payload, timeout, attempt+1)
		lastResult = result

		if result.Error == "" && result.StatusCode >= 200 && result.StatusCode < 300 {
			return result, nil
		}
	}

	if lastResult != nil && lastResult.Error != "" {
		return lastResult, fmt.Errorf("webhook delivery failed after %d attempts: %s", maxRetries, lastResult.Error)
	}
	return lastResult, fmt.Errorf("webhook delivery failed after %d attempts: status %d", maxRetries, lastResult.StatusCode)
}

func (p *WebhookProducer) doRequest(ctx context.Context, endpoint EndpointConfig, payload []byte, timeout time.Duration, attempt int) *DeliveryResult {
	result := &DeliveryResult{
		URL:         endpoint.URL,
		Attempt:     attempt,
		DeliveredAt: time.Now(),
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint.URL, bytes.NewReader(payload))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range endpoint.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	result.StatusCode = resp.StatusCode
	result.ResponseBody = string(body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("non-2xx response: %d", resp.StatusCode)
	}

	return result
}
