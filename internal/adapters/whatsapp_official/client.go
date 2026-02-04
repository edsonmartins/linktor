package whatsapp_official

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client is the HTTP client for Meta Graph API
type Client struct {
	httpClient    *http.Client
	config        *Config
	rateLimiter   *rate.Limiter
	mu            sync.RWMutex
	rateLimitInfo *RateLimitInfo
}

// NewClient creates a new WhatsApp API client
func NewClient(config *Config) *Client {
	if config.APIVersion == "" {
		config.APIVersion = DefaultAPIVersion
	}

	// Business tier: 80 messages/second
	limiter := rate.NewLimiter(rate.Limit(80), 100)

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config:      config,
		rateLimiter: limiter,
		rateLimitInfo: &RateLimitInfo{
			Limit:     80,
			Remaining: 80,
			ResetAt:   time.Now().Add(time.Second),
		},
	}
}

// SendMessage sends a message via the WhatsApp Cloud API
func (c *Client) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	req.MessagingProduct = "whatsapp"
	if req.RecipientType == "" {
		req.RecipientType = "individual"
	}

	endpoint := c.buildURL(fmt.Sprintf("/%s/messages", c.config.PhoneNumberID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, endpoint, body, nil)
	if err != nil {
		return nil, err
	}

	var response SendMessageResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// SendTextMessage is a convenience method to send a text message
func (c *Client) SendTextMessage(ctx context.Context, to, text string, previewURL bool) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: MessageTypeText,
		Text: &TextContent{
			Body:       text,
			PreviewURL: previewURL,
		},
	}
	return c.SendMessage(ctx, req)
}

// SendImageMessage sends an image message
func (c *Client) SendImageMessage(ctx context.Context, to string, media *MediaObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:    to,
		Type:  MessageTypeImage,
		Image: media,
	}
	return c.SendMessage(ctx, req)
}

// SendDocumentMessage sends a document message
func (c *Client) SendDocumentMessage(ctx context.Context, to string, doc *DocumentObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:       to,
		Type:     MessageTypeDocument,
		Document: doc,
	}
	return c.SendMessage(ctx, req)
}

// SendVideoMessage sends a video message
func (c *Client) SendVideoMessage(ctx context.Context, to string, media *MediaObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:    to,
		Type:  MessageTypeVideo,
		Video: media,
	}
	return c.SendMessage(ctx, req)
}

// SendAudioMessage sends an audio message
func (c *Client) SendAudioMessage(ctx context.Context, to string, media *MediaObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:    to,
		Type:  MessageTypeAudio,
		Audio: media,
	}
	return c.SendMessage(ctx, req)
}

// SendLocationMessage sends a location message
func (c *Client) SendLocationMessage(ctx context.Context, to string, location *LocationObject) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:       to,
		Type:     MessageTypeLocation,
		Location: location,
	}
	return c.SendMessage(ctx, req)
}

// SendContactsMessage sends contacts message
func (c *Client) SendContactsMessage(ctx context.Context, to string, contacts []ContactContent) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:       to,
		Type:     MessageTypeContacts,
		Contacts: contacts,
	}
	return c.SendMessage(ctx, req)
}

// SendReactionMessage sends a reaction to a message
func (c *Client) SendReactionMessage(ctx context.Context, to, messageID, emoji string) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: MessageTypeReaction,
		Reaction: &ReactionObject{
			MessageID: messageID,
			Emoji:     emoji,
		},
	}
	return c.SendMessage(ctx, req)
}

// MarkAsRead marks a message as read
func (c *Client) MarkAsRead(ctx context.Context, messageID string) error {
	endpoint := c.buildURL(fmt.Sprintf("/%s/messages", c.config.PhoneNumberID))

	body, err := json.Marshal(map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = c.doRequest(ctx, http.MethodPost, endpoint, body, nil)
	return err
}

// GetMediaInfo retrieves media information (including download URL)
func (c *Client) GetMediaInfo(ctx context.Context, mediaID string) (*MediaInfoResponse, error) {
	endpoint := c.buildURL(fmt.Sprintf("/%s", mediaID))

	respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}

	var response MediaInfoResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// DownloadMedia downloads media content
func (c *Client) DownloadMedia(ctx context.Context, mediaURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("media download failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read media body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// UploadMedia uploads media to WhatsApp servers
func (c *Client) UploadMedia(ctx context.Context, filename, mimeType string, data []byte) (*MediaUploadResponse, error) {
	endpoint := c.buildURL(fmt.Sprintf("/%s/media", c.config.PhoneNumberID))

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add messaging product field
	if err := writer.WriteField("messaging_product", "whatsapp"); err != nil {
		return nil, fmt.Errorf("failed to write messaging_product field: %w", err)
	}

	// Add type field
	if err := writer.WriteField("type", mimeType); err != nil {
		return nil, fmt.Errorf("failed to write type field: %w", err)
	}

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, endpoint, buf.Bytes(), headers)
	if err != nil {
		return nil, err
	}

	var response MediaUploadResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// DeleteMedia deletes media from WhatsApp servers
func (c *Client) DeleteMedia(ctx context.Context, mediaID string) error {
	endpoint := c.buildURL(fmt.Sprintf("/%s", mediaID))
	_, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil, nil)
	return err
}

// GetBusinessProfile retrieves the business profile
func (c *Client) GetBusinessProfile(ctx context.Context) (*BusinessProfile, error) {
	endpoint := c.buildURL(fmt.Sprintf("/%s/whatsapp_business_profile", c.config.PhoneNumberID))
	endpoint += "?fields=about,address,description,email,profile_picture_url,websites,vertical"

	respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Data []BusinessProfile `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("no business profile found")
	}

	return &wrapper.Data[0], nil
}

// UpdateBusinessProfile updates the business profile
func (c *Client) UpdateBusinessProfile(ctx context.Context, profile *BusinessProfile) error {
	endpoint := c.buildURL(fmt.Sprintf("/%s/whatsapp_business_profile", c.config.PhoneNumberID))

	profile.MessagingProduct = "whatsapp"
	body, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = c.doRequest(ctx, http.MethodPost, endpoint, body, nil)
	return err
}

// GetPhoneNumberInfo retrieves phone number information
func (c *Client) GetPhoneNumberInfo(ctx context.Context) (*PhoneNumberInfo, error) {
	endpoint := c.buildURL(fmt.Sprintf("/%s", c.config.PhoneNumberID))
	endpoint += "?fields=verified_name,display_phone_number,quality_rating,status,name_status,messaging_limit_tier"

	respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}

	var response PhoneNumberInfo
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// GetHealth retrieves the health status
func (c *Client) GetHealth(ctx context.Context) (*HealthStatus, error) {
	endpoint := c.buildURL(fmt.Sprintf("/%s", c.config.BusinessID))
	endpoint += "?fields=health_status"

	respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		HealthStatus HealthStatus `json:"health_status"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.HealthStatus, nil
}

// buildURL builds the API URL
func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", BaseURL, c.config.APIVersion, path)
}

// doRequest executes an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	var lastErr error
	maxRetries := 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		respBody, err := c.executeRequest(ctx, method, url, body, headers)
		if err == nil {
			return respBody, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// executeRequest executes a single HTTP request
func (c *Client) executeRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	if body != nil && headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
		}
		return nil, &APIRequestError{
			StatusCode: resp.StatusCode,
			APIError:   errResp.Error,
		}
	}

	return respBody, nil
}

// APIRequestError represents an API request error
type APIRequestError struct {
	StatusCode int
	APIError   APIError
}

func (e *APIRequestError) Error() string {
	return fmt.Sprintf("WhatsApp API error (code %d, subcode %d): %s",
		e.APIError.Code, e.APIError.ErrorSubcode, e.APIError.Message)
}

// IsRateLimitError checks if the error is a rate limit error
func (e *APIRequestError) IsRateLimitError() bool {
	return e.StatusCode == 429 || e.APIError.Code == 80007
}

// IsAuthError checks if the error is an authentication error
func (e *APIRequestError) IsAuthError() bool {
	return e.StatusCode == 401 || e.APIError.Code == 190
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if apiErr, ok := err.(*APIRequestError); ok {
		// Retry on rate limit or server errors
		return apiErr.IsRateLimitError() || apiErr.StatusCode >= 500
	}
	return false
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

// UpdateConfig updates the client configuration
func (c *Client) UpdateConfig(config *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
}
