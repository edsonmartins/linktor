package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultAPIBaseURL is the public Bot API endpoint.
const DefaultAPIBaseURL = "https://api.telegram.org"

// WebhookRegistrar points a bot at our inbound webhook. Without it a Telegram
// channel is "connected" in the database while Telegram has nowhere to deliver
// updates — nothing ever arrives.
type WebhookRegistrar struct {
	APIBaseURL string // no trailing slash; DefaultAPIBaseURL when empty
	HTTPClient *http.Client
}

// NewWebhookRegistrar returns a registrar for the public Bot API.
func NewWebhookRegistrar() *WebhookRegistrar {
	return &WebhookRegistrar{APIBaseURL: DefaultAPIBaseURL, HTTPClient: &http.Client{Timeout: 15 * time.Second}}
}

// NewSecretToken generates a secret_token for setWebhook (Telegram accepts
// 1-256 chars of A-Z, a-z, 0-9, _ and -).
func NewSecretToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Register validates the token (getMe) and registers webhookURL with
// secretToken, which Telegram echoes back in X-Telegram-Bot-Api-Secret-Token on
// every update. Returns the bot's username.
func (r *WebhookRegistrar) Register(ctx context.Context, botToken, secretToken, webhookURL string) (string, error) {
	var me struct {
		Username string `json:"username"`
	}
	if err := r.call(ctx, botToken, "getMe", nil, &me); err != nil {
		return "", fmt.Errorf("invalid bot token: %w", err)
	}
	form := url.Values{}
	form.Set("url", webhookURL)
	form.Set("secret_token", secretToken)
	if err := r.call(ctx, botToken, "setWebhook", form, nil); err != nil {
		return "", fmt.Errorf("setWebhook failed: %w", err)
	}
	return me.Username, nil
}

// Unregister removes the bot's webhook (pending updates are kept).
func (r *WebhookRegistrar) Unregister(ctx context.Context, botToken string) error {
	return r.call(ctx, botToken, "deleteWebhook", url.Values{}, nil)
}

// call invokes a Bot API method (GET without form, POST with it) and decodes
// result into out when non-nil. Errors carry Telegram's description, never the
// token-bearing URL.
func (r *WebhookRegistrar) call(ctx context.Context, botToken, method string, form url.Values, out any) error {
	base := strings.TrimRight(r.APIBaseURL, "/")
	if base == "" {
		base = DefaultAPIBaseURL
	}
	endpoint := base + "/bot" + botToken + "/" + method

	var req *http.Request
	var err error
	if form == nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if req != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return fmt.Errorf("%s: build request", method)
	}

	httpClient := r.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		// *url.Error embeds the URL, which carries the token.
		return fmt.Errorf("%s: request failed", method)
	}
	defer resp.Body.Close()

	var apiResp struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("%s: unexpected response (HTTP %d)", method, resp.StatusCode)
	}
	if !apiResp.OK {
		return fmt.Errorf("%s: %s", method, apiResp.Description)
	}
	if out != nil {
		if err := json.Unmarshal(apiResp.Result, out); err != nil {
			return fmt.Errorf("%s: unexpected result", method)
		}
	}
	return nil
}
