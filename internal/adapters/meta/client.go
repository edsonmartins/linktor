package meta

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client wraps the Meta Graph API
type Client struct {
	httpClient  *http.Client
	accessToken string
	appSecret   string
	apiVersion  string
	baseURL     string
}

// NewClient creates a new Meta Graph API client
func NewClient(accessToken, appSecret string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken: accessToken,
		appSecret:   appSecret,
		apiVersion:  DefaultAPIVersion,
		baseURL:     GraphAPIBaseURL,
	}
}

// NewInstagramClient creates a client for Instagram Graph API
func NewInstagramClient(accessToken, appSecret string) *Client {
	c := NewClient(accessToken, appSecret)
	c.baseURL = InstagramAPIBaseURL
	return c
}

// SetAPIVersion changes the API version
func (c *Client) SetAPIVersion(version string) {
	c.apiVersion = version
}

// SetAccessToken updates the access token
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// buildURL constructs the API URL
func (c *Client) buildURL(endpoint string) string {
	return fmt.Sprintf("%s/%s/%s", c.baseURL, c.apiVersion, endpoint)
}

// addAuth adds authentication to the request
func (c *Client) addAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
}

// addAppSecretProof adds app secret proof for server-to-server calls
func (c *Client) addAppSecretProof(params url.Values) {
	if c.appSecret != "" {
		h := hmac.New(sha256.New, []byte(c.appSecret))
		h.Write([]byte(c.accessToken))
		proof := hex.EncodeToString(h.Sum(nil))
		params.Set("appsecret_proof", proof)
	}
}

// doRequest executes an HTTP request
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(endpoint), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error *APIError `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != nil {
			return nil, apiErr.Error
		}
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	return respBody, nil
}

// doRequestWithQuery executes an HTTP request with query parameters
func (c *Client) doRequestWithQuery(ctx context.Context, method, endpoint string, params url.Values, body interface{}) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", c.accessToken)
	c.addAppSecretProof(params)

	fullURL := c.buildURL(endpoint) + "?" + params.Encode()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error *APIError `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != nil {
			return nil, apiErr.Error
		}
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	return respBody, nil
}

// SendMessage sends a message via the Send API
func (c *Client) SendMessage(ctx context.Context, pageID string, msg *OutboundMessage) (*SendMessageResponse, error) {
	endpoint := fmt.Sprintf("%s/messages", pageID)
	if pageID == "" || pageID == "me" {
		endpoint = "me/messages"
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, endpoint, msg)
	if err != nil {
		return nil, err
	}

	var resp SendMessageResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// SendInstagramMessage sends a message via Instagram Messaging API
func (c *Client) SendInstagramMessage(ctx context.Context, igUserID string, msg *OutboundMessage) (*SendMessageResponse, error) {
	endpoint := fmt.Sprintf("%s/messages", igUserID)
	if igUserID == "" {
		endpoint = "me/messages"
	}

	params := url.Values{}
	respBody, err := c.doRequestWithQuery(ctx, http.MethodPost, endpoint, params, msg)
	if err != nil {
		return nil, err
	}

	var resp SendMessageResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// SendSenderAction sends a sender action (typing indicator, mark seen)
func (c *Client) SendSenderAction(ctx context.Context, pageID string, action *SenderAction) error {
	endpoint := fmt.Sprintf("%s/messages", pageID)
	if pageID == "" || pageID == "me" {
		endpoint = "me/messages"
	}

	_, err := c.doRequest(ctx, http.MethodPost, endpoint, action)
	return err
}

// GetUserProfile retrieves a user's profile information
func (c *Client) GetUserProfile(ctx context.Context, userID string, fields []string) (*UserProfile, error) {
	params := url.Values{}
	if len(fields) > 0 {
		fieldsStr := ""
		for i, f := range fields {
			if i > 0 {
				fieldsStr += ","
			}
			fieldsStr += f
		}
		params.Set("fields", fieldsStr)
	} else {
		params.Set("fields", "id,name,first_name,last_name,profile_pic")
	}

	respBody, err := c.doRequestWithQuery(ctx, http.MethodGet, userID, params, nil)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &profile, nil
}

// GetMyPages retrieves the pages the user has access to
func (c *Client) GetMyPages(ctx context.Context) (*PagesResponse, error) {
	params := url.Values{}
	params.Set("fields", "id,name,access_token,category,picture")

	respBody, err := c.doRequestWithQuery(ctx, http.MethodGet, "me/accounts", params, nil)
	if err != nil {
		return nil, err
	}

	var pages PagesResponse
	if err := json.Unmarshal(respBody, &pages); err != nil {
		return nil, fmt.Errorf("failed to parse pages: %w", err)
	}

	return &pages, nil
}

// GetPageInfo retrieves information about a specific page
func (c *Client) GetPageInfo(ctx context.Context, pageID string) (*PageInfo, error) {
	params := url.Values{}
	params.Set("fields", "id,name,access_token,category,picture")

	respBody, err := c.doRequestWithQuery(ctx, http.MethodGet, pageID, params, nil)
	if err != nil {
		return nil, err
	}

	var page PageInfo
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("failed to parse page info: %w", err)
	}

	return &page, nil
}

// GetInstagramAccount retrieves the connected Instagram account for a page
func (c *Client) GetInstagramAccount(ctx context.Context, pageID string) (*InstagramAccount, error) {
	params := url.Values{}
	params.Set("fields", "instagram_business_account{id,username,name,profile_picture_url,followers_count}")

	respBody, err := c.doRequestWithQuery(ctx, http.MethodGet, pageID, params, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		InstagramBusinessAccount *InstagramAccount `json:"instagram_business_account"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse Instagram account: %w", err)
	}

	if resp.InstagramBusinessAccount == nil {
		return nil, fmt.Errorf("no Instagram business account connected to this page")
	}

	return resp.InstagramBusinessAccount, nil
}

// SubscribeToWebhook subscribes a page/account to webhook events
func (c *Client) SubscribeToWebhook(ctx context.Context, pageID string, fields []string) error {
	endpoint := fmt.Sprintf("%s/subscribed_apps", pageID)

	params := url.Values{}
	if len(fields) > 0 {
		fieldsStr := ""
		for i, f := range fields {
			if i > 0 {
				fieldsStr += ","
			}
			fieldsStr += f
		}
		params.Set("subscribed_fields", fieldsStr)
	}

	respBody, err := c.doRequestWithQuery(ctx, http.MethodPost, endpoint, params, nil)
	if err != nil {
		return err
	}

	var resp SubscribedAppsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to subscribe to webhooks")
	}

	return nil
}

// UnsubscribeFromWebhook unsubscribes from webhook events
func (c *Client) UnsubscribeFromWebhook(ctx context.Context, pageID string) error {
	endpoint := fmt.Sprintf("%s/subscribed_apps", pageID)

	_, err := c.doRequestWithQuery(ctx, http.MethodDelete, endpoint, nil, nil)
	return err
}

// ExchangeCodeForToken exchanges an OAuth code for an access token
func (c *Client) ExchangeCodeForToken(ctx context.Context, appID, appSecret, redirectURI, code string) (*OAuthTokenResponse, error) {
	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("redirect_uri", redirectURI)
	params.Set("code", code)

	// Use the standard client for OAuth (no auth header needed)
	fullURL := c.buildURL("oauth/access_token") + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tokenResp, nil
}

// GetLongLivedToken exchanges a short-lived token for a long-lived one
func (c *Client) GetLongLivedToken(ctx context.Context, appID, appSecret, shortLivedToken string) (*LongLivedTokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "fb_exchange_token")
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("fb_exchange_token", shortLivedToken)

	fullURL := c.buildURL("oauth/access_token") + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp LongLivedTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tokenResp, nil
}

// ValidateWebhookSignature validates the X-Hub-Signature-256 header
func ValidateWebhookSignature(appSecret string, payload []byte, signature string) bool {
	if len(signature) < 8 || signature[:7] != "sha256=" {
		return false
	}

	expectedSig := signature[7:]

	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write(payload)
	actualSig := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(actualSig))
}

// ParseWebhookPayload parses the webhook payload
func ParseWebhookPayload(body []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}
	return &payload, nil
}

// ExtractMessagingEvents extracts messaging events from webhook entries
func ExtractMessagingEvents(entries []WebhookEntry) []MessagingEvent {
	var events []MessagingEvent
	for _, entry := range entries {
		events = append(events, entry.Messaging...)
		events = append(events, entry.Standby...)
	}
	return events
}

// ParseInboundMessage converts a MessagingEvent to ParsedInboundMessage
func ParseInboundMessage(event *MessagingEvent) *ParsedInboundMessage {
	if event.Message == nil {
		return nil
	}

	msg := &ParsedInboundMessage{
		ExternalID:  event.Message.MID,
		SenderID:    event.Sender.ID,
		RecipientID: event.Recipient.ID,
		Text:        event.Message.Text,
		IsEcho:      event.Message.IsEcho,
		IsDeleted:   event.Message.IsDeleted,
		Timestamp:   time.Unix(event.Timestamp/1000, 0),
	}

	if event.Message.QuickReply != nil {
		msg.QuickReply = event.Message.QuickReply.Payload
	}

	if event.Message.ReplyTo != nil {
		msg.ReplyToMID = event.Message.ReplyTo.MID
	}

	for _, att := range event.Message.Attachments {
		parsed := ParsedAttachment{
			Type:  att.Type,
			URL:   att.Payload.URL,
			Title: att.Payload.Title,
		}
		if att.Payload.Coordinates != nil {
			parsed.Lat = att.Payload.Coordinates.Lat
			parsed.Long = att.Payload.Coordinates.Long
		}
		msg.Attachments = append(msg.Attachments, parsed)
	}

	return msg
}
