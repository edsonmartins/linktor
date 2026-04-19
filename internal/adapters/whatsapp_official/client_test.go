package whatsapp_official

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport rewrites request URLs to point to the test server,
// keeping the original path and query string intact.
type rewriteTransport struct {
	baseURL string
	rt      http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace scheme + host with the test server, preserve path + query.
	parsed := strings.TrimPrefix(t.baseURL, "http://")
	req.URL.Scheme = "http"
	req.URL.Host = parsed
	return t.rt.RoundTrip(req)
}

// setupTestClient creates a Client whose HTTP requests are redirected to the
// given httptest handler. The returned server must be closed by the caller.
func setupTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v21.0",
	}
	client := NewClient(config)
	client.httpClient = &http.Client{
		Transport: &rewriteTransport{
			baseURL: server.URL,
			rt:      http.DefaultTransport,
		},
	}
	return client, server
}

// sendMessageSuccessBody returns a standard successful send-message JSON response.
func sendMessageSuccessBody() []byte {
	resp := SendMessageResponse{
		MessagingProduct: "whatsapp",
		Contacts: []SendMessageContact{
			{Input: "5511999999999", WaID: "5511999999999"},
		},
		Messages: []SendMessageResult{
			{ID: "wamid.test123"},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// ---------------------------------------------------------------------------
// TestClient_SendMessage
// ---------------------------------------------------------------------------

func TestClient_SendMessage_Success(t *testing.T) {
	var capturedBody map[string]interface{}
	var capturedAuth string
	var capturedPath string

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	req := &SendMessageRequest{
		To:   "5511999999999",
		Type: MessageTypeText,
		Text: &TextContent{Body: "Hello"},
	}

	resp, err := client.SendMessage(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "whatsapp", resp.MessagingProduct)
	assert.Len(t, resp.Messages, 1)
	assert.Equal(t, "wamid.test123", resp.Messages[0].ID)

	// Verify request body fields
	assert.Equal(t, "whatsapp", capturedBody["messaging_product"])
	assert.Equal(t, "individual", capturedBody["recipient_type"])
	assert.Equal(t, "5511999999999", capturedBody["to"])
	assert.Equal(t, "text", capturedBody["type"])

	// Verify auth header
	assert.Equal(t, "Bearer test-token", capturedAuth)

	// Verify URL path
	assert.Equal(t, "/v21.0/12345/messages", capturedPath)
}

func TestClient_SendMessage_RateLimitError(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		errResp := ErrorResponse{
			Error: APIError{
				Message:      "Rate limit hit",
				Type:         "OAuthException",
				Code:         80007,
				ErrorSubcode: 2446079,
			},
		}
		json.NewEncoder(w).Encode(errResp)
	})
	defer server.Close()

	req := &SendMessageRequest{
		To:   "5511999999999",
		Type: MessageTypeText,
		Text: &TextContent{Body: "Hello"},
	}

	// Use a context with cancel to avoid waiting through all retries
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The rate-limit error IS retryable, so it will retry up to maxRetries.
	// We cancel after the first attempt to avoid long waits.
	// Instead, let's just let it run and check the final error wraps correctly.
	// Actually, rate limit is retryable so it will keep retrying. Let's cancel quickly.
	go func() {
		// Cancel after the first couple of retries
		<-ctx.Done()
	}()

	_, err := client.SendMessage(ctx, req)
	require.Error(t, err)

	// The error could be a context error (if cancelled) or max retries exceeded wrapping APIRequestError.
	// Let's verify the underlying API error is present in the chain.
	assert.Contains(t, err.Error(), "80007")
}

func TestClient_SendMessage_ServerError_Retried(t *testing.T) {
	var requestCount int32

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			errResp := ErrorResponse{
				Error: APIError{
					Message: "Internal server error",
					Type:    "OAuthException",
					Code:    1,
				},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}
		// Third attempt succeeds
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	req := &SendMessageRequest{
		To:   "5511999999999",
		Type: MessageTypeText,
		Text: &TextContent{Body: "Hello"},
	}

	resp, err := client.SendMessage(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "wamid.test123", resp.Messages[0].ID)

	// Verify there were multiple requests (retried)
	assert.True(t, atomic.LoadInt32(&requestCount) >= 3, "expected at least 3 requests due to retries, got %d", requestCount)
}

func TestClient_SendMessage_AuthError_NotRetried(t *testing.T) {
	var requestCount int32

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusUnauthorized)
		errResp := ErrorResponse{
			Error: APIError{
				Message: "Invalid OAuth access token",
				Type:    "OAuthException",
				Code:    190,
			},
		}
		json.NewEncoder(w).Encode(errResp)
	})
	defer server.Close()

	req := &SendMessageRequest{
		To:   "5511999999999",
		Type: MessageTypeText,
		Text: &TextContent{Body: "Hello"},
	}

	_, err := client.SendMessage(context.Background(), req)
	require.Error(t, err)

	// Auth errors should NOT be retried
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "auth error should not be retried")

	var apiErr *APIRequestError
	require.ErrorAs(t, err, &apiErr)
	assert.True(t, apiErr.IsAuthError())
	assert.False(t, apiErr.IsRateLimitError())
}

// ---------------------------------------------------------------------------
// TestClient_SendTextMessage
// ---------------------------------------------------------------------------

func TestClient_SendTextMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	resp, err := client.SendTextMessage(context.Background(), "5511999999999", "Hello world", true)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "wamid.test123", resp.Messages[0].ID)

	assert.Equal(t, "text", capturedBody["type"])
	textObj, ok := capturedBody["text"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Hello world", textObj["body"])
	assert.Equal(t, true, textObj["preview_url"])
}

// ---------------------------------------------------------------------------
// TestClient_SendImageMessage
// ---------------------------------------------------------------------------

func TestClient_SendImageMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	media := &MediaObject{
		Link:    "https://example.com/image.jpg",
		Caption: "Test image",
	}

	resp, err := client.SendImageMessage(context.Background(), "5511999999999", media)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "wamid.test123", resp.Messages[0].ID)

	assert.Equal(t, "image", capturedBody["type"])
	imgObj, ok := capturedBody["image"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.com/image.jpg", imgObj["link"])
	assert.Equal(t, "Test image", imgObj["caption"])
}

// ---------------------------------------------------------------------------
// TestClient_SendReactionMessage
// ---------------------------------------------------------------------------

func TestClient_SendReactionMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	resp, err := client.SendReactionMessage(context.Background(), "5511999999999", "wamid.original123", "\U0001F44D")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "reaction", capturedBody["type"])
	reactionObj, ok := capturedBody["reaction"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wamid.original123", reactionObj["message_id"])
	assert.Equal(t, "\U0001F44D", reactionObj["emoji"])
}

func TestClient_RemoveReaction(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	resp, err := client.RemoveReaction(context.Background(), "5511999999999", "wamid.original123")
	require.NoError(t, err)
	require.NotNil(t, resp)

	reactionObj, ok := capturedBody["reaction"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "", reactionObj["emoji"])
}

// ---------------------------------------------------------------------------
// TestClient_MarkAsRead
// ---------------------------------------------------------------------------

func TestClient_MarkAsRead(t *testing.T) {
	var capturedBody map[string]interface{}
	var capturedMethod string

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})
	defer server.Close()

	err := client.MarkAsRead(context.Background(), "wamid.msg123")
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "whatsapp", capturedBody["messaging_product"])
	assert.Equal(t, "read", capturedBody["status"])
	assert.Equal(t, "wamid.msg123", capturedBody["message_id"])
}

// ---------------------------------------------------------------------------
// TestClient_GetMediaInfo
// ---------------------------------------------------------------------------

func TestClient_GetMediaInfo(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v21.0/media-id-123", r.URL.Path)

		resp := MediaInfoResponse{
			ID:       "media-id-123",
			URL:      "https://lookaside.fbsbx.com/whatsapp_business/attachments/...",
			MimeType: "image/jpeg",
			SHA256:   "abc123sha256",
			FileSize: 12345,
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	info, err := client.GetMediaInfo(context.Background(), "media-id-123")
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, "media-id-123", info.ID)
	assert.Equal(t, "https://lookaside.fbsbx.com/whatsapp_business/attachments/...", info.URL)
	assert.Equal(t, "image/jpeg", info.MimeType)
	assert.Equal(t, "abc123sha256", info.SHA256)
	assert.Equal(t, int64(12345), info.FileSize)
}

// ---------------------------------------------------------------------------
// TestClient_DownloadMedia
// ---------------------------------------------------------------------------

func TestClient_DownloadMedia_Success(t *testing.T) {
	expectedData := []byte("fake-image-binary-data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedData)
	}))
	defer server.Close()

	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v21.0",
	}
	client := NewClient(config)

	data, contentType, err := client.DownloadMedia(context.Background(), server.URL+"/media/download")
	require.NoError(t, err)
	assert.Equal(t, expectedData, data)
	assert.Equal(t, "image/jpeg", contentType)
}

func TestClient_DownloadMedia_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v21.0",
	}
	client := NewClient(config)

	data, contentType, err := client.DownloadMedia(context.Background(), server.URL+"/media/download")
	require.Error(t, err)
	assert.Nil(t, data)
	assert.Empty(t, contentType)
	assert.Contains(t, err.Error(), "media download failed")
	assert.Contains(t, err.Error(), "404")
}

// ---------------------------------------------------------------------------
// TestClient_UploadMedia
// ---------------------------------------------------------------------------

func TestClient_UploadMedia(t *testing.T) {
	var capturedContentType string
	var capturedMethod string
	var capturedPath string

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedContentType = r.Header.Get("Content-Type")

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		assert.Equal(t, "whatsapp", r.FormValue("messaging_product"))
		assert.Equal(t, "image/jpeg", r.FormValue("type"))

		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "test.jpg", header.Filename)
		fileData, _ := io.ReadAll(file)
		assert.Equal(t, []byte("fake-image-data"), fileData)

		resp := MediaUploadResponse{ID: "uploaded-media-id-456"}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.UploadMedia(context.Background(), "test.jpg", "image/jpeg", []byte("fake-image-data"))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "uploaded-media-id-456", result.ID)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v21.0/12345/media", capturedPath)
	assert.True(t, strings.HasPrefix(capturedContentType, "multipart/form-data"))
}

// ---------------------------------------------------------------------------
// TestClient_DeleteMedia
// ---------------------------------------------------------------------------

func TestClient_DeleteMedia(t *testing.T) {
	var capturedMethod string
	var capturedPath string

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})
	defer server.Close()

	err := client.DeleteMedia(context.Background(), "media-to-delete-789")
	require.NoError(t, err)

	assert.Equal(t, http.MethodDelete, capturedMethod)
	assert.Equal(t, "/v21.0/media-to-delete-789", capturedPath)
}

// ---------------------------------------------------------------------------
// TestClient_GetBusinessProfile
// ---------------------------------------------------------------------------

func TestClient_GetBusinessProfile(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/v21.0/12345/whatsapp_business_profile")
		assert.Contains(t, r.URL.RawQuery, "fields=")

		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"about":              "Test business",
					"address":            "123 Main St",
					"description":        "A test business profile",
					"email":              "test@example.com",
					"profile_picture_url": "https://example.com/pic.jpg",
					"websites":           []string{"https://example.com"},
					"vertical":           "OTHER",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	profile, err := client.GetBusinessProfile(context.Background())
	require.NoError(t, err)
	require.NotNil(t, profile)

	assert.Equal(t, "Test business", profile.About)
	assert.Equal(t, "123 Main St", profile.Address)
	assert.Equal(t, "A test business profile", profile.Description)
	assert.Equal(t, "test@example.com", profile.Email)
	assert.Equal(t, "https://example.com/pic.jpg", profile.ProfilePictureURL)
	assert.Equal(t, "OTHER", profile.Vertical)
}

func TestClient_GetBusinessProfile_Empty(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	profile, err := client.GetBusinessProfile(context.Background())
	require.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "no business profile found")
}

// ---------------------------------------------------------------------------
// TestClient_UpdateBusinessProfile
// ---------------------------------------------------------------------------

func TestClient_UpdateBusinessProfile(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})
	defer server.Close()

	profile := &BusinessProfile{
		About:       "Updated about",
		Description: "Updated description",
	}

	err := client.UpdateBusinessProfile(context.Background(), profile)
	require.NoError(t, err)

	assert.Equal(t, "whatsapp", capturedBody["messaging_product"])
	assert.Equal(t, "Updated about", capturedBody["about"])
	assert.Equal(t, "Updated description", capturedBody["description"])
}

// ---------------------------------------------------------------------------
// TestClient_GetPhoneNumberInfo
// ---------------------------------------------------------------------------

func TestClient_GetPhoneNumberInfo(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v21.0/12345", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "fields=")

		resp := PhoneNumberInfo{
			ID:                 "12345",
			VerifiedName:       "Test Business",
			DisplayPhoneNumber: "+55 11 99999-9999",
			QualityRating:      "GREEN",
			Status:             "CONNECTED",
			NameStatus:         "APPROVED",
			MessagingLimitTier: "TIER_10K",
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	info, err := client.GetPhoneNumberInfo(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, "12345", info.ID)
	assert.Equal(t, "Test Business", info.VerifiedName)
	assert.Equal(t, "+55 11 99999-9999", info.DisplayPhoneNumber)
	assert.Equal(t, "GREEN", info.QualityRating)
	assert.Equal(t, "CONNECTED", info.Status)
	assert.Equal(t, "APPROVED", info.NameStatus)
	assert.Equal(t, "TIER_10K", info.MessagingLimitTier)
}

// ---------------------------------------------------------------------------
// TestClient_GetHealth
// ---------------------------------------------------------------------------

func TestClient_GetHealth(t *testing.T) {
	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v21.0/67890", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "fields=health_status")

		resp := map[string]interface{}{
			"health_status": map[string]interface{}{
				"health": []map[string]interface{}{
					{
						"entity_type":      "APP",
						"can_send_message": "AVAILABLE",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	health, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	require.NotNil(t, health)
	require.Len(t, health.Health, 1)
	assert.Equal(t, "APP", health.Health[0].EntityType)
	assert.Equal(t, "AVAILABLE", health.Health[0].CanSendMessage)
}

// ---------------------------------------------------------------------------
// TestClient_buildURL
// ---------------------------------------------------------------------------

func TestClient_buildURL(t *testing.T) {
	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v21.0",
	}
	client := NewClient(config)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "messages endpoint",
			path:     "/12345/messages",
			expected: "https://graph.facebook.com/v21.0/12345/messages",
		},
		{
			name:     "media endpoint",
			path:     "/12345/media",
			expected: "https://graph.facebook.com/v21.0/12345/media",
		},
		{
			name:     "media ID endpoint",
			path:     "/media-id-123",
			expected: "https://graph.facebook.com/v21.0/media-id-123",
		},
		{
			name:     "business profile endpoint",
			path:     "/12345/whatsapp_business_profile",
			expected: "https://graph.facebook.com/v21.0/12345/whatsapp_business_profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.buildURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAPIRequestError
// ---------------------------------------------------------------------------

func TestAPIRequestError_Error(t *testing.T) {
	err := &APIRequestError{
		StatusCode: 400,
		APIError: APIError{
			Message:      "Invalid parameter",
			Type:         "OAuthException",
			Code:         100,
			ErrorSubcode: 2018001,
		},
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "100")
	assert.Contains(t, errStr, "2018001")
	assert.Contains(t, errStr, "Invalid parameter")
	assert.Equal(t, fmt.Sprintf("WhatsApp API error (code %d, subcode %d): %s", 100, 2018001, "Invalid parameter"), errStr)
}

func TestAPIRequestError_IsRateLimitError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       int
		expected   bool
	}{
		{
			name:       "429 status code",
			statusCode: 429,
			code:       0,
			expected:   true,
		},
		{
			name:       "code 80007",
			statusCode: 200,
			code:       80007,
			expected:   true,
		},
		{
			name:       "429 with code 80007",
			statusCode: 429,
			code:       80007,
			expected:   true,
		},
		{
			name:       "400 with other code",
			statusCode: 400,
			code:       100,
			expected:   false,
		},
		{
			name:       "500 server error",
			statusCode: 500,
			code:       1,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIRequestError{
				StatusCode: tt.statusCode,
				APIError:   APIError{Code: tt.code},
			}
			assert.Equal(t, tt.expected, err.IsRateLimitError())
		})
	}
}

func TestAPIRequestError_IsAuthError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       int
		expected   bool
	}{
		{
			name:       "401 status code",
			statusCode: 401,
			code:       0,
			expected:   true,
		},
		{
			name:       "code 190",
			statusCode: 200,
			code:       190,
			expected:   true,
		},
		{
			name:       "401 with code 190",
			statusCode: 401,
			code:       190,
			expected:   true,
		},
		{
			name:       "400 with other code",
			statusCode: 400,
			code:       100,
			expected:   false,
		},
		{
			name:       "403 forbidden is not auth error",
			statusCode: 403,
			code:       10,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIRequestError{
				StatusCode: tt.statusCode,
				APIError:   APIError{Code: tt.code},
			}
			assert.Equal(t, tt.expected, err.IsAuthError())
		})
	}
}

// ---------------------------------------------------------------------------
// Test isRetryableError
// ---------------------------------------------------------------------------

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "rate limit error is retryable",
			err: &APIRequestError{
				StatusCode: 429,
				APIError:   APIError{Code: 80007, Message: "Rate limit"},
			},
			expected: true,
		},
		{
			name: "500 server error is retryable",
			err: &APIRequestError{
				StatusCode: 500,
				APIError:   APIError{Code: 1, Message: "Internal error"},
			},
			expected: true,
		},
		{
			name: "502 server error is retryable",
			err: &APIRequestError{
				StatusCode: 502,
				APIError:   APIError{Code: 2, Message: "Bad gateway"},
			},
			expected: true,
		},
		{
			name: "400 client error is not retryable",
			err: &APIRequestError{
				StatusCode: 400,
				APIError:   APIError{Code: 100, Message: "Invalid parameter"},
			},
			expected: false,
		},
		{
			name: "401 auth error is not retryable",
			err: &APIRequestError{
				StatusCode: 401,
				APIError:   APIError{Code: 190, Message: "Invalid token"},
			},
			expected: false,
		},
		{
			name:     "non-APIRequestError is not retryable",
			err:      fmt.Errorf("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isRetryableError(tt.err))
		})
	}
}

// ---------------------------------------------------------------------------
// TestClient_SendRawRequest
// ---------------------------------------------------------------------------

func TestClient_SendRawRequest(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	rawReq := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                "5511999999999",
		"type":              "template",
		"template": map[string]interface{}{
			"name": "hello_world",
			"language": map[string]interface{}{
				"code": "en_US",
			},
		},
	}

	resp, err := client.SendRawRequest(context.Background(), rawReq)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "wamid.test123", resp.Messages[0].ID)

	assert.Equal(t, "whatsapp", capturedBody["messaging_product"])
	assert.Equal(t, "template", capturedBody["type"])
	tmpl, ok := capturedBody["template"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "hello_world", tmpl["name"])
}

// ---------------------------------------------------------------------------
// TestClient_SendDocumentMessage
// ---------------------------------------------------------------------------

func TestClient_SendDocumentMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	doc := &DocumentObject{
		Link:     "https://example.com/doc.pdf",
		Caption:  "Important document",
		Filename: "report.pdf",
	}

	resp, err := client.SendDocumentMessage(context.Background(), "5511999999999", doc)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "document", capturedBody["type"])
	docObj, ok := capturedBody["document"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.com/doc.pdf", docObj["link"])
	assert.Equal(t, "Important document", docObj["caption"])
	assert.Equal(t, "report.pdf", docObj["filename"])
}

// ---------------------------------------------------------------------------
// TestClient_SendVideoMessage
// ---------------------------------------------------------------------------

func TestClient_SendVideoMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	media := &MediaObject{
		Link:    "https://example.com/video.mp4",
		Caption: "Check this video",
	}

	resp, err := client.SendVideoMessage(context.Background(), "5511999999999", media)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "video", capturedBody["type"])
	videoObj, ok := capturedBody["video"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.com/video.mp4", videoObj["link"])
}

// ---------------------------------------------------------------------------
// TestClient_SendAudioMessage
// ---------------------------------------------------------------------------

func TestClient_SendAudioMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	media := &MediaObject{
		Link: "https://example.com/audio.ogg",
	}

	resp, err := client.SendAudioMessage(context.Background(), "5511999999999", media)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "audio", capturedBody["type"])
	audioObj, ok := capturedBody["audio"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.com/audio.ogg", audioObj["link"])
}

// ---------------------------------------------------------------------------
// TestClient_SendLocationMessage
// ---------------------------------------------------------------------------

func TestClient_SendLocationMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	location := &LocationObject{
		Latitude:  -23.5505,
		Longitude: -46.6333,
		Name:      "Sao Paulo",
		Address:   "Sao Paulo, Brazil",
	}

	resp, err := client.SendLocationMessage(context.Background(), "5511999999999", location)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "location", capturedBody["type"])
	locObj, ok := capturedBody["location"].(map[string]interface{})
	require.True(t, ok)
	assert.InDelta(t, -23.5505, locObj["latitude"], 0.0001)
	assert.InDelta(t, -46.6333, locObj["longitude"], 0.0001)
	assert.Equal(t, "Sao Paulo", locObj["name"])
	assert.Equal(t, "Sao Paulo, Brazil", locObj["address"])
}

// ---------------------------------------------------------------------------
// TestClient_SendContactsMessage
// ---------------------------------------------------------------------------

func TestClient_SendContactsMessage(t *testing.T) {
	var capturedBody map[string]interface{}

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	contacts := []ContactContent{
		{
			Name: ContactName{
				FormattedName: "John Doe",
				FirstName:     "John",
				LastName:      "Doe",
			},
			Phones: []ContactPhone{
				{Phone: "+5511999999999", Type: "CELL"},
			},
		},
	}

	resp, err := client.SendContactsMessage(context.Background(), "5511999999999", contacts)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "contacts", capturedBody["type"])
	contactsArr, ok := capturedBody["contacts"].([]interface{})
	require.True(t, ok)
	require.Len(t, contactsArr, 1)
}

// ---------------------------------------------------------------------------
// TestNewClient_DefaultAPIVersion
// ---------------------------------------------------------------------------

func TestNewClient_DefaultAPIVersion(t *testing.T) {
	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		// APIVersion intentionally left empty
	}
	client := NewClient(config)

	assert.Equal(t, DefaultAPIVersion, client.config.APIVersion)
	assert.Equal(t, "v23.0", client.config.APIVersion)
}

func TestNewClient_CustomAPIVersion(t *testing.T) {
	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v19.0",
	}
	client := NewClient(config)

	assert.Equal(t, "v19.0", client.config.APIVersion)
}

// ---------------------------------------------------------------------------
// TestClient_GetConfig and UpdateConfig
// ---------------------------------------------------------------------------

func TestClient_GetConfig(t *testing.T) {
	config := &Config{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v21.0",
	}
	client := NewClient(config)

	got := client.GetConfig()
	assert.Equal(t, "test-token", got.AccessToken)
	assert.Equal(t, "12345", got.PhoneNumberID)
	assert.Equal(t, "67890", got.BusinessID)
}

func TestClient_UpdateConfig(t *testing.T) {
	config := &Config{
		AccessToken:   "old-token",
		PhoneNumberID: "12345",
		BusinessID:    "67890",
		APIVersion:    "v21.0",
	}
	client := NewClient(config)

	newConfig := &Config{
		AccessToken:   "new-token",
		PhoneNumberID: "99999",
		BusinessID:    "11111",
		APIVersion:    "v22.0",
	}
	client.UpdateConfig(newConfig)

	got := client.GetConfig()
	assert.Equal(t, "new-token", got.AccessToken)
	assert.Equal(t, "99999", got.PhoneNumberID)
}

// ---------------------------------------------------------------------------
// TestClient_AuthorizationHeader
// ---------------------------------------------------------------------------

func TestClient_AuthorizationHeader(t *testing.T) {
	var capturedAuth string

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	_, err := client.SendTextMessage(context.Background(), "5511999999999", "test", false)
	require.NoError(t, err)
	assert.Equal(t, "Bearer test-token", capturedAuth)
}

// ---------------------------------------------------------------------------
// TestClient_ContentTypeHeader
// ---------------------------------------------------------------------------

func TestClient_ContentTypeHeader_JSON(t *testing.T) {
	var capturedContentType string

	client, server := setupTestClient(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write(sendMessageSuccessBody())
	})
	defer server.Close()

	_, err := client.SendTextMessage(context.Background(), "5511999999999", "test", false)
	require.NoError(t, err)
	assert.Equal(t, "application/json", capturedContentType)
}
