// Package direto implements the VendaX Direto channel connector for Linktor.
//
// Direto is a chat app (Tinode-backed) that the Direto runtime exposes as a
// WhatsApp-flavored channel API (RFC-009): a REST send endpoint and an inbound
// webhook. This package is the OUTBOUND side (Linktor → Direto): a stateless
// per-channel Sender that POSTs to the Direto send API. The INBOUND side lives
// in the central webhook handler (DiretoWebhook).
//
// Not to be confused with internal/integration/vendax (Linktor → VendaX Core
// axis): that forwards inbound to the Core; this makes Direto a channel.
package direto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds one channel's resolved Direto credentials.
type Config struct {
	// SendURL is the Direto runtime base (e.g. https://direto.vendax.ai).
	SendURL string
	// InstanceID is the channel instance id in the send path.
	InstanceID string
	// APIToken is the Bearer presented to the Direto send endpoint.
	APIToken string
}

// Client is a thin HTTP client for the Direto send API.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient builds a Direto API client.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// sendRequest mirrors the Direto RFC-009 send body.
type sendRequest struct {
	To       string `json:"to"`
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MediaURL string `json:"mediaUrl,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

// sendResponse mirrors the Direto RFC-009 send response {messages:[{id}]}.
type sendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// APIError is a non-2xx response from the Direto send API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("direto send API returned %d: %s", e.StatusCode, e.Body)
}

// Permanent reports whether the error is a 4xx that must not be retried
// (unknown recipient, bad token) — distinct from 5xx/network (transient).
func (e *APIError) Permanent() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// SendText sends a plain text message; returns the Direto messageId.
func (c *Client) SendText(ctx context.Context, to, text string) (string, error) {
	return c.send(ctx, sendRequest{To: to, Type: "text", Text: text})
}

// SendMedia sends a media message (image|audio|video|file) by URL; returns the messageId.
func (c *Client) SendMedia(ctx context.Context, to, mediaType, url, caption string) (string, error) {
	return c.send(ctx, sendRequest{To: to, Type: mediaType, MediaURL: url, Caption: caption})
}

func (c *Client) send(ctx context.Context, body sendRequest) (string, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(c.cfg.SendURL, "/") + "/channel/v1/" + c.cfg.InstanceID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	var parsed sendResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Messages) == 0 {
		return "", fmt.Errorf("direto send API: no message id in response")
	}
	return parsed.Messages[0].ID, nil
}
