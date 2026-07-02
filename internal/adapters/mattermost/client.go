package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// maxInboundMediaBytes caps how much of a Mattermost file we buffer into memory
// when re-hosting (server uploads can be very large), so a hostile or oversized
// attachment can't OOM the listener.
const maxInboundMediaBytes = 64 << 20 // 64 MiB

// Client calls the Mattermost REST API for one channel (bot).
type Client struct {
	baseURL    string
	botToken   string
	httpClient *http.Client
}

// NewClient builds a Mattermost REST client.
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		botToken:   cfg.BotToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// httpError carries the provider status so the sender can classify failures.
type httpError struct {
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("mattermost: api returned %d: %s", e.StatusCode, e.Body)
}

// CreatePost posts a text message to a Mattermost channel and returns the post id.
func (c *Client) CreatePost(ctx context.Context, channelID, message string) (string, error) {
	return c.CreatePostWithFiles(ctx, channelID, message, nil)
}

// CreatePostWithFiles posts a message optionally attaching previously-uploaded
// files (by id) and returns the post id.
func (c *Client) CreatePostWithFiles(ctx context.Context, channelID, message string, fileIDs []string) (string, error) {
	if channelID == "" {
		return "", fmt.Errorf("mattermost: channel id is required")
	}

	post := map[string]interface{}{
		"channel_id": channelID,
		"message":    message,
	}
	if len(fileIDs) > 0 {
		post["file_ids"] = fileIDs
	}
	payload, err := json.Marshal(post)
	if err != nil {
		return "", err
	}

	endpoint := c.baseURL + "/api/v4/posts"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

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

// UploadFile uploads raw bytes to a channel and returns the new file id, to be
// attached to a post via CreatePostWithFiles. Uses POST /api/v4/files (multipart).
func (c *Client) UploadFile(ctx context.Context, channelID, filename string, data []byte) (string, error) {
	if channelID == "" {
		return "", fmt.Errorf("mattermost: channel id is required")
	}
	if filename == "" {
		filename = "file"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("channel_id", channelID); err != nil {
		return "", err
	}
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	endpoint := c.baseURL + "/api/v4/files"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &httpError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result struct {
		FileInfos []struct {
			ID string `json:"id"`
		} `json:"file_infos"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if len(result.FileInfos) == 0 {
		return "", fmt.Errorf("mattermost: upload returned no file info")
	}
	return result.FileInfos[0].ID, nil
}

// Me verifies the bot PAT and server reachability by fetching the bot's own
// user, returning its id and username. Used by the "test connection" endpoint
// and to auto-resolve bot_user_id for anti-echo filtering.
func (c *Client) Me(ctx context.Context) (id, username string, err error) {
	endpoint := c.baseURL + "/api/v4/users/me"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", "", &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var u struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", "", err
	}
	return u.ID, u.Username, nil
}

// GetFileInfo fetches metadata for a file attached to a post.
func (c *Client) GetFileInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	endpoint := fmt.Sprintf("%s/api/v4/files/%s/info", c.baseURL, fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var info FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DownloadFile fetches the raw bytes of a file by id (authenticated with the PAT).
func (c *Client) DownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/api/v4/files/%s", c.baseURL, fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &httpError{StatusCode: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxInboundMediaBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxInboundMediaBytes {
		return nil, fmt.Errorf("mattermost: file exceeds %d bytes", maxInboundMediaBytes)
	}
	return data, nil
}
