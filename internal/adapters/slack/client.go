package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultAPIBase is the Slack Web API root.
const defaultAPIBase = "https://slack.com/api"

// maxInboundMediaBytes caps how much of a provider attachment we buffer into
// memory when re-hosting, so a hostile or oversized file can't OOM the process.
const maxInboundMediaBytes = 64 << 20 // 64 MiB

// Client calls the Slack Web API for one channel (bot), holding that bot token.
type Client struct {
	botToken   string
	apiBase    string
	httpClient *http.Client
}

// NewClient builds a Slack Web API client.
func NewClient(botToken string) *Client {
	return &Client{
		botToken:   botToken,
		apiBase:    defaultAPIBase,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// apiError carries a Slack API-level error so the sender can classify it.
// Slack returns HTTP 200 with {"ok":false,"error":"..."} on logical failures.
type apiError struct {
	Code int    // HTTP status (0 for API-level)
	Err  string // Slack error code (e.g. "channel_not_found")
}

func (e *apiError) Error() string {
	if e.Err != "" {
		return "slack: api error: " + e.Err
	}
	return fmt.Sprintf("slack: http %d", e.Code)
}

// DownloadFile fetches a Slack file by its url_private. Slack file URLs are NOT
// public: they require the bot token as a Bearer credential. Returns the raw
// bytes and the response content type.
func (c *Client) DownloadFile(ctx context.Context, urlPrivate string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPrivate, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("slack: file download returned %d", resp.StatusCode)
	}
	// Slack returns an HTML login page (200) when the token can't read the file;
	// guard against silently storing that as "media".
	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/html") {
		return nil, "", fmt.Errorf("slack: file not accessible with this token")
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxInboundMediaBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxInboundMediaBytes {
		return nil, "", fmt.Errorf("slack: file exceeds %d bytes", maxInboundMediaBytes)
	}
	return data, contentType, nil
}

// AuthTest verifies the bot token by calling auth.test, returning the team and
// bot user it resolves to. Used by the "test connection" endpoint.
func (c *Client) AuthTest(ctx context.Context) (team, user string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/auth.test", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", &apiError{Code: resp.StatusCode}
	}

	var result struct {
		OK    bool   `json:"ok"`
		Team  string `json:"team"`
		User  string `json:"user"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if !result.OK {
		return "", "", &apiError{Err: result.Error}
	}
	return result.Team, result.User, nil
}

// PostMessage posts text to a Slack channel and returns the message ts (id).
func (c *Client) PostMessage(ctx context.Context, channel, text string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"channel": channel,
		"text":    text,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/chat.postMessage", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &apiError{Code: resp.StatusCode}
	}

	var result struct {
		OK    bool   `json:"ok"`
		TS    string `json:"ts"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", &apiError{Err: result.Error}
	}
	return result.TS, nil
}

// UploadFile shares a file to a channel using Slack's external upload flow
// (getUploadURLExternal → POST bytes → completeUploadExternal). initialComment
// becomes the message text. Returns the new file id.
func (c *Client) UploadFile(ctx context.Context, channelID, filename, initialComment string, data []byte) (string, error) {
	uploadURL, fileID, err := c.getUploadURLExternal(ctx, filename, len(data))
	if err != nil {
		return "", err
	}
	if err := c.postBytes(ctx, uploadURL, filename, data); err != nil {
		return "", err
	}
	if err := c.completeUploadExternal(ctx, channelID, fileID, filename, initialComment); err != nil {
		return "", err
	}
	return fileID, nil
}

// getUploadURLExternal reserves an upload URL and file id for a file of length n.
func (c *Client) getUploadURLExternal(ctx context.Context, filename string, n int) (uploadURL, fileID string, err error) {
	form := url.Values{}
	form.Set("filename", filename)
	form.Set("length", strconv.Itoa(n))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/files.getUploadURLExternal", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", &apiError{Code: resp.StatusCode}
	}

	var result struct {
		OK        bool   `json:"ok"`
		UploadURL string `json:"upload_url"`
		FileID    string `json:"file_id"`
		Error     string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if !result.OK {
		return "", "", &apiError{Err: result.Error}
	}
	return result.UploadURL, result.FileID, nil
}

// postBytes uploads the raw file content to the reserved upload URL.
func (c *Client) postBytes(ctx context.Context, uploadURL, filename string, data []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{Code: resp.StatusCode}
	}
	return nil
}

// completeUploadExternal finalizes the upload and shares the file to a channel.
func (c *Client) completeUploadExternal(ctx context.Context, channelID, fileID, title, initialComment string) error {
	payload := map[string]interface{}{
		"files":      []map[string]string{{"id": fileID, "title": title}},
		"channel_id": channelID,
	}
	if initialComment != "" {
		payload["initial_comment"] = initialComment
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/files.completeUploadExternal", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{Code: resp.StatusCode}
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return &apiError{Err: result.Error}
	}
	return nil
}
