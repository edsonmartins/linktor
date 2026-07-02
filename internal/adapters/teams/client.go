package teams

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

// maxInboundMediaBytes caps how much of a Bot Connector attachment we buffer
// into memory when re-hosting, so a hostile or oversized file can't OOM us.
const maxInboundMediaBytes = 64 << 20 // 64 MiB

// Client posts outbound Activities to the Bot Connector for one channel (bot).
// It holds the channel's config and an AAD token source, and is safe for
// concurrent use.
type Client struct {
	cfg        Config
	tokens     *tokenSource
	httpClient *http.Client
}

// NewClient builds a Teams Bot Connector client from resolved config.
func NewClient(cfg Config) *Client {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	return &Client{
		cfg:        cfg,
		tokens:     newTokenSource(cfg, httpClient),
		httpClient: httpClient,
	}
}

// NewClientFromCredentials builds a client straight from a channel credentials map.
func NewClientFromCredentials(creds map[string]string) *Client {
	return NewClient(configFromCreds(creds))
}

// Ping verifies the channel's credentials by acquiring an AAD app token
// (client-credentials). A successful token means app_id/app_password/tenant_id
// are valid. Used by the "test connection" endpoint.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.tokens.Token(ctx)
	return err
}

// DownloadAttachment fetches an inbound attachment's bytes. Bot Connector
// attachment URLs require the bot's AAD bearer token; pre-authenticated URLs
// (e.g. Teams file download.info SharePoint links) ignore the header harmlessly.
// Returns the bytes and the response content type.
func (c *Client) DownloadAttachment(ctx context.Context, contentURL string) ([]byte, string, error) {
	if contentURL == "" {
		return nil, "", fmt.Errorf("teams: empty content url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, contentURL, nil)
	if err != nil {
		return nil, "", err
	}
	if token, err := c.tokens.Token(ctx); err == nil && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &httpError{StatusCode: resp.StatusCode}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxInboundMediaBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxInboundMediaBytes {
		return nil, "", fmt.Errorf("teams: attachment exceeds %d bytes", maxInboundMediaBytes)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// httpError carries the provider status code so the sender can classify
// permanent vs transient failures.
type httpError struct {
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("teams: bot connector returned %d: %s", e.StatusCode, e.Body)
}

// SendActivity posts an Activity to a conversation and returns the resulting
// activity id. serviceURL defaults to the channel's configured ServiceURL when
// empty.
func (c *Client) SendActivity(ctx context.Context, serviceURL, conversationID string, activity *Activity) (string, error) {
	if serviceURL == "" {
		serviceURL = c.cfg.ServiceURL
	}
	if serviceURL == "" {
		return "", fmt.Errorf("teams: no service_url configured for channel")
	}
	if conversationID == "" {
		return "", fmt.Errorf("teams: conversation id is required")
	}

	token, err := c.tokens.Token(ctx)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/v3/conversations/%s/activities",
		strings.TrimRight(serviceURL, "/"), conversationID)

	payload, err := json.Marshal(activity)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var result struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &result)
	return result.ID, nil
}
