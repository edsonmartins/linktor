package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

const (
	// OAuthStateMaxAge is the maximum age for OAuth state (10 minutes)
	OAuthStateMaxAge = 600
	// HTTPClientTimeout is the timeout for HTTP requests to Meta Graph API
	HTTPClientTimeout = 30 * time.Second
)

// WhatsAppEmbeddedSignupHandler handles WhatsApp Embedded Signup for Coexistence
type WhatsAppEmbeddedSignupHandler struct {
	channelRepo repository.ChannelRepository
	baseURL     string
	graphAPIURL string
	stateSecret []byte
	httpClient  *http.Client
}

// NewWhatsAppEmbeddedSignupHandler creates a new handler
func NewWhatsAppEmbeddedSignupHandler(channelRepo repository.ChannelRepository, baseURL string) *WhatsAppEmbeddedSignupHandler {
	// Validate HTTPS in production (allow localhost for development)
	normalizedURL := strings.TrimSuffix(baseURL, "/")
	if !strings.HasPrefix(normalizedURL, "https://") {
		isLocalhost := strings.Contains(normalizedURL, "localhost") ||
			strings.Contains(normalizedURL, "127.0.0.1") ||
			strings.Contains(normalizedURL, "0.0.0.0")
		if !isLocalhost {
			// Log warning but continue - let deployment handle SSL termination
			fmt.Printf("WARNING: WhatsApp Embedded Signup baseURL should use HTTPS in production: %s\n", baseURL)
		}
	}

	// Generate random secret for state signing
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		// Fallback to a derived secret from baseURL
		h := sha256.Sum256([]byte(baseURL + "whatsapp-embedded-signup"))
		secret = h[:]
	}

	return &WhatsAppEmbeddedSignupHandler{
		channelRepo: channelRepo,
		baseURL:     normalizedURL,
		graphAPIURL: "https://graph.facebook.com/v21.0",
		stateSecret: secret,
		httpClient: &http.Client{
			Timeout: HTTPClientTimeout,
		},
	}
}

// EmbeddedSignupState stores state for embedded signup flow
type EmbeddedSignupState struct {
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	RedirectURL string `json:"redirect_url"`
	Timestamp   int64  `json:"timestamp"`
	Nonce       string `json:"nonce"`
}

// EmbeddedSignupStartRequest represents request to start embedded signup
type EmbeddedSignupStartRequest struct {
	AppID       string `json:"app_id" binding:"required"`
	ConfigID    string `json:"config_id"` // Optional: WhatsApp Embedded Signup Config ID
	RedirectURL string `json:"redirect_url"`
}

// EmbeddedSignupStartResponse represents response with login URL
type EmbeddedSignupStartResponse struct {
	LoginURL string `json:"login_url"`
	State    string `json:"state"`
}

// EmbeddedSignupCallbackRequest represents the callback data
type EmbeddedSignupCallbackRequest struct {
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	AppID     string `json:"app_id" binding:"required"`
	AppSecret string `json:"app_secret" binding:"required"`
}

// EmbeddedSignupCallbackResponse represents the callback response
type EmbeddedSignupCallbackResponse struct {
	AccessToken     string            `json:"access_token"`
	WABAID          string            `json:"waba_id"`
	PhoneNumberID   string            `json:"phone_number_id"`
	PhoneNumber     string            `json:"phone_number"`
	IsCoexistence   bool              `json:"is_coexistence"`
	CoexStatus      string            `json:"coexistence_status"`
	BusinessName    string            `json:"business_name,omitempty"`
	QualityRating   string            `json:"quality_rating,omitempty"`
	VerifyToken     string            `json:"verify_token"`
	WebhookURL      string            `json:"webhook_url"`
	SubscribedFields []string         `json:"subscribed_fields"`
}

// CoexistenceStatusResponse represents coexistence status
type CoexistenceStatusResponse struct {
	Enabled             bool       `json:"enabled"`
	Status              string     `json:"status"`
	LastEchoAt          *time.Time `json:"last_echo_at,omitempty"`
	DaysSinceLastEcho   int        `json:"days_since_last_echo"`
	DaysUntilDisconnect int        `json:"days_until_disconnect"`
	Recommendation      string     `json:"recommendation,omitempty"`
}

// generateNonce creates a secure random nonce
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// encodeEmbeddedState encodes state for embedded signup with HMAC signature
func (h *WhatsAppEmbeddedSignupHandler) encodeEmbeddedState(state *EmbeddedSignupState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}

	// Create HMAC signature
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write(data)
	signature := mac.Sum(nil)

	// Combine data + signature
	combined := append(data, signature...)
	return hex.EncodeToString(combined), nil
}

// decodeEmbeddedState decodes and validates state from embedded signup
func (h *WhatsAppEmbeddedSignupHandler) decodeEmbeddedState(encoded string) (*EmbeddedSignupState, error) {
	combined, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid state encoding: %w", err)
	}

	// Signature is 32 bytes (SHA256)
	if len(combined) < 32 {
		return nil, fmt.Errorf("state too short")
	}

	data := combined[:len(combined)-32]
	signature := combined[len(combined)-32:]

	// Verify HMAC signature
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write(data)
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return nil, fmt.Errorf("invalid state signature")
	}

	var state EmbeddedSignupState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("invalid state data: %w", err)
	}
	return &state, nil
}

// StartEmbeddedSignup initiates the WhatsApp Embedded Signup flow
// POST /api/v1/oauth/whatsapp/embedded-signup/start
func (h *WhatsAppEmbeddedSignupHandler) StartEmbeddedSignup(c *gin.Context) {
	var req EmbeddedSignupStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	tenantID := c.GetString(middleware.TenantIDKey)
	userID := c.GetString(middleware.UserIDKey)

	// Generate nonce
	nonce, err := generateNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}

	// Create state
	state := &EmbeddedSignupState{
		TenantID:    tenantID,
		UserID:      userID,
		RedirectURL: req.RedirectURL,
		Timestamp:   time.Now().Unix(),
		Nonce:       nonce,
	}

	stateStr, err := h.encodeEmbeddedState(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create state"})
		return
	}

	// Build Facebook Login URL with WhatsApp Embedded Signup permissions
	redirectURI := h.baseURL + "/api/v1/oauth/whatsapp/embedded-signup/callback"

	// Required scopes for WhatsApp Embedded Signup
	scopes := []string{
		"whatsapp_business_management",
		"whatsapp_business_messaging",
		"business_management",
	}

	params := url.Values{}
	params.Set("client_id", req.AppID)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", stateStr)
	params.Set("scope", strings.Join(scopes, ","))
	params.Set("response_type", "code")

	// Add config_id if provided (for Embedded Signup flow)
	if req.ConfigID != "" {
		params.Set("config_id", req.ConfigID)
	}

	// Add extras for embedded signup
	params.Set("extras", `{"feature":"whatsapp_embedded_signup","sessionInfoVersion":2}`)

	loginURL := "https://www.facebook.com/v21.0/dialog/oauth?" + params.Encode()

	c.JSON(http.StatusOK, EmbeddedSignupStartResponse{
		LoginURL: loginURL,
		State:    stateStr,
	})
}

// CompleteEmbeddedSignup handles the OAuth callback and completes signup
// POST /api/v1/oauth/whatsapp/embedded-signup/callback
func (h *WhatsAppEmbeddedSignupHandler) CompleteEmbeddedSignup(c *gin.Context) {
	var req EmbeddedSignupCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Decode and validate state with HMAC verification
	state, err := h.decodeEmbeddedState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state: " + err.Error()})
		return
	}

	// Check state age (max 10 minutes)
	if time.Now().Unix()-state.Timestamp > OAuthStateMaxAge {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state expired"})
		return
	}

	// Validate tenant matches the authenticated user
	tenantID := c.GetString(middleware.TenantIDKey)
	if state.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant mismatch"})
		return
	}

	ctx := c.Request.Context()

	// 1. Exchange code for access token
	accessToken, err := h.exchangeCodeForToken(ctx, req.AppID, req.AppSecret, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange code: " + err.Error()})
		return
	}

	// 2. Debug token to get WABA ID and Phone Number ID
	wabaID, phoneNumberID, err := h.getWABAInfo(ctx, accessToken, req.AppID, req.AppSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get WABA info: " + err.Error()})
		return
	}

	// 3. Get phone number details
	phoneDetails, err := h.getPhoneNumberDetails(ctx, phoneNumberID, accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get phone details: " + err.Error()})
		return
	}

	// 4. Check if coexistence is enabled
	isCoex, err := h.checkCoexistence(ctx, phoneNumberID, accessToken)
	if err != nil {
		// Non-fatal: assume not coexistence if check fails
		isCoex = false
	}

	// 5. Subscribe to webhooks including message_echoes
	verifyToken, _ := generateNonce()
	subscribedFields, err := h.subscribeToWebhooks(ctx, wabaID, accessToken, isCoex)
	if err != nil {
		// Non-fatal: log but continue
		subscribedFields = []string{"messages"}
	}

	// Build webhook URL
	webhookURL := h.baseURL + "/api/v1/webhooks/whatsapp_official/{channel_id}"

	// Determine coexistence status
	coexStatus := "inactive"
	if isCoex {
		coexStatus = "pending" // Will become 'active' after first echo
	}

	c.JSON(http.StatusOK, EmbeddedSignupCallbackResponse{
		AccessToken:      accessToken,
		WABAID:           wabaID,
		PhoneNumberID:    phoneNumberID,
		PhoneNumber:      phoneDetails.DisplayPhoneNumber,
		IsCoexistence:    isCoex,
		CoexStatus:       coexStatus,
		BusinessName:     phoneDetails.VerifiedName,
		QualityRating:    phoneDetails.QualityRating,
		VerifyToken:      verifyToken,
		WebhookURL:       webhookURL,
		SubscribedFields: subscribedFields,
	})
}

// CreateCoexistenceChannel creates a channel with coexistence enabled
// POST /api/v1/oauth/whatsapp/embedded-signup/create-channel
func (h *WhatsAppEmbeddedSignupHandler) CreateCoexistenceChannel(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		AccessToken   string `json:"access_token" binding:"required"`
		WABAID        string `json:"waba_id" binding:"required"`
		PhoneNumberID string `json:"phone_number_id" binding:"required"`
		PhoneNumber   string `json:"phone_number" binding:"required"`
		AppID         string `json:"app_id" binding:"required"`
		AppSecret     string `json:"app_secret" binding:"required"`
		VerifyToken   string `json:"verify_token"`
		IsCoexistence bool   `json:"is_coexistence"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	tenantID := c.GetString(middleware.TenantIDKey)

	// Generate verify token if not provided
	verifyToken := req.VerifyToken
	if verifyToken == "" {
		verifyToken, _ = generateNonce()
	}

	// Determine coexistence status
	coexStatus := entity.CoexistenceStatusInactive
	if req.IsCoexistence {
		coexStatus = entity.CoexistenceStatusPending
	}

	// Create channel
	channel := &entity.Channel{
		TenantID:         tenantID,
		Name:             req.Name,
		Type:             entity.ChannelTypeWhatsAppOfficial,
		Identifier:       req.PhoneNumber,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials: map[string]string{
			"access_token":    req.AccessToken,
			"app_id":          req.AppID,
			"app_secret":      req.AppSecret,
			"verify_token":    verifyToken,
			"phone_number_id": req.PhoneNumberID,
			"waba_id":         req.WABAID,
		},
		Config: map[string]string{
			"phone_number":    req.PhoneNumber,
			"phone_number_id": req.PhoneNumberID,
			"waba_id":         req.WABAID,
		},
		IsCoexistence:     req.IsCoexistence,
		WABAID:            req.WABAID,
		CoexistenceStatus: coexStatus,
	}

	// Save channel
	if err := h.channelRepo.Create(c.Request.Context(), channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel: " + err.Error()})
		return
	}

	// Build webhook URL with actual channel ID
	webhookURL := h.baseURL + "/api/v1/webhooks/whatsapp_official/" + channel.ID

	c.JSON(http.StatusCreated, gin.H{
		"channel":      channel,
		"webhook_url":  webhookURL,
		"verify_token": verifyToken,
	})
}

// GetCoexistenceStatus returns the coexistence status for a channel
// GET /api/v1/channels/:id/coexistence-status
func (h *WhatsAppEmbeddedSignupHandler) GetCoexistenceStatus(c *gin.Context) {
	channelID := c.Param("id")
	tenantID := c.GetString(middleware.TenantIDKey)

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Validate tenant ownership
	if channel.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if !channel.IsCoexistence {
		c.JSON(http.StatusOK, CoexistenceStatusResponse{
			Enabled: false,
			Status:  "inactive",
		})
		return
	}

	daysSinceEcho := channel.DaysSinceLastEcho()
	daysUntilDisconnect := 0
	if daysSinceEcho >= 0 && daysSinceEcho < 14 {
		daysUntilDisconnect = 14 - daysSinceEcho
	}

	status := channel.CheckCoexistenceStatus()

	// Generate recommendation based on status
	var recommendation string
	switch status {
	case entity.CoexistenceStatusWarning:
		recommendation = fmt.Sprintf("Open WhatsApp Business App within the next %d days to maintain coexistence.", daysUntilDisconnect)
	case entity.CoexistenceStatusDisconnected:
		recommendation = "Coexistence has been disconnected. Open WhatsApp Business App to reconnect."
	case entity.CoexistenceStatusPending:
		recommendation = "Open WhatsApp Business App to activate coexistence."
	}

	c.JSON(http.StatusOK, CoexistenceStatusResponse{
		Enabled:             true,
		Status:              string(status),
		LastEchoAt:          channel.LastEchoAt,
		DaysSinceLastEcho:   daysSinceEcho,
		DaysUntilDisconnect: daysUntilDisconnect,
		Recommendation:      recommendation,
	})
}

// SubscribeMessageEchoes subscribes a channel to message_echoes webhook
// POST /api/v1/channels/:id/subscribe-echoes
func (h *WhatsAppEmbeddedSignupHandler) SubscribeMessageEchoes(c *gin.Context) {
	channelID := c.Param("id")
	tenantID := c.GetString(middleware.TenantIDKey)

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Validate tenant ownership
	if channel.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if channel.Type != entity.ChannelTypeWhatsAppOfficial {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel is not WhatsApp Official"})
		return
	}

	accessToken := channel.Credentials["access_token"]
	wabaID := channel.WABAID
	if wabaID == "" {
		wabaID = channel.Credentials["waba_id"]
	}

	if accessToken == "" || wabaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing credentials"})
		return
	}

	// Subscribe to message_echoes
	fields, err := h.subscribeToWebhooks(c.Request.Context(), wabaID, accessToken, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to subscribe: " + err.Error()})
		return
	}

	// Update channel as coexistence
	channel.IsCoexistence = true
	channel.CoexistenceStatus = entity.CoexistenceStatusPending
	if err := h.channelRepo.Update(c.Request.Context(), channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"subscribed_fields": fields,
	})
}

// exchangeCodeForToken exchanges OAuth code for access token
func (h *WhatsAppEmbeddedSignupHandler) exchangeCodeForToken(ctx context.Context, appID, appSecret, code string) (string, error) {
	tokenURL := h.graphAPIURL + "/oauth/access_token"

	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}

// getWABAInfo gets WABA ID and Phone Number ID from debug token
func (h *WhatsAppEmbeddedSignupHandler) getWABAInfo(ctx context.Context, accessToken, appID, appSecret string) (string, string, error) {
	// First, get the debug token info
	debugURL := h.graphAPIURL + "/debug_token"
	params := url.Values{}
	params.Set("input_token", accessToken)
	params.Set("access_token", appID+"|"+appSecret)

	req, err := http.NewRequestWithContext(ctx, "GET", debugURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("debug request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read debug response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("debug token request failed: %s", string(body))
	}

	var debugResp struct {
		Data struct {
			GranularScopes []struct {
				Scope      string   `json:"scope"`
				TargetIDs  []string `json:"target_ids"`
			} `json:"granular_scopes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &debugResp); err != nil {
		return "", "", fmt.Errorf("failed to parse debug response: %w", err)
	}

	var wabaID, phoneNumberID string

	// Extract WABA ID and Phone Number ID from scopes
	for _, scope := range debugResp.Data.GranularScopes {
		if scope.Scope == "whatsapp_business_management" && len(scope.TargetIDs) > 0 {
			wabaID = scope.TargetIDs[0]
		}
		if scope.Scope == "whatsapp_business_messaging" && len(scope.TargetIDs) > 0 {
			phoneNumberID = scope.TargetIDs[0]
		}
	}

	// If phone number ID not found in scopes, try to get it from WABA
	if wabaID != "" && phoneNumberID == "" {
		phoneNumberID, _ = h.getPhoneNumberIDFromWABA(ctx, wabaID, accessToken)
	}

	if wabaID == "" {
		return "", "", fmt.Errorf("WABA ID not found in token scopes")
	}

	return wabaID, phoneNumberID, nil
}

// getPhoneNumberIDFromWABA gets phone number ID from WABA
func (h *WhatsAppEmbeddedSignupHandler) getPhoneNumberIDFromWABA(ctx context.Context, wabaID, accessToken string) (string, error) {
	reqURL := fmt.Sprintf("%s/%s/phone_numbers", h.graphAPIURL, wabaID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get phone numbers: %s (status %d)", string(body), resp.StatusCode)
	}

	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(data.Data) > 0 {
		return data.Data[0].ID, nil
	}

	return "", fmt.Errorf("no phone numbers found")
}

// PhoneNumberDetails holds phone number information
type PhoneNumberDetails struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	VerifiedName       string `json:"verified_name"`
	QualityRating      string `json:"quality_rating"`
}

// getPhoneNumberDetails gets details about a phone number
func (h *WhatsAppEmbeddedSignupHandler) getPhoneNumberDetails(ctx context.Context, phoneNumberID, accessToken string) (*PhoneNumberDetails, error) {
	reqURL := fmt.Sprintf("%s/%s?fields=display_phone_number,verified_name,quality_rating", h.graphAPIURL, phoneNumberID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get phone number details: %s (status %d)", string(body), resp.StatusCode)
	}

	var details PhoneNumberDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to parse phone number details: %w", err)
	}

	return &details, nil
}

// checkCoexistence checks if a phone number has Business App active (coexistence)
func (h *WhatsAppEmbeddedSignupHandler) checkCoexistence(ctx context.Context, phoneNumberID, accessToken string) (bool, error) {
	reqURL := fmt.Sprintf("%s/%s?fields=is_business_app_number_active", h.graphAPIURL, phoneNumberID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("failed to check coexistence: %s (status %d)", string(body), resp.StatusCode)
	}

	var data struct {
		IsBusinessAppNumberActive bool `json:"is_business_app_number_active"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, fmt.Errorf("failed to parse coexistence response: %w", err)
	}

	return data.IsBusinessAppNumberActive, nil
}

// subscribeToWebhooks subscribes WABA to webhook fields
func (h *WhatsAppEmbeddedSignupHandler) subscribeToWebhooks(ctx context.Context, wabaID, accessToken string, includeEchoes bool) ([]string, error) {
	reqURL := fmt.Sprintf("%s/%s/subscribed_apps", h.graphAPIURL, wabaID)

	// Fields to subscribe to
	fields := []string{
		"messages",
		"message_template_status_update",
		"message_template_quality_update",
		"account_alerts",
		"account_update",
		"phone_number_quality_update",
		"security",
	}

	// Add message_echoes for coexistence
	if includeEchoes {
		fields = append(fields, "message_echoes")
	}

	payload := map[string]interface{}{
		"subscribed_fields": fields,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription failed: %s", string(respBody))
	}

	return fields, nil
}
