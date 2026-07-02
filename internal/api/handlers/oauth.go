package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// OAuthHandler handles OAuth flows for Facebook and Instagram
type OAuthHandler struct {
	channelRepo repository.ChannelRepository
	baseURL     string // Base URL for callbacks (e.g., https://api.linktor.com)
	stateSecret []byte

	// pending holds app credentials between login and callback, keyed by the
	// signed state, so the browser never has to retain the app secret.
	pendingMu sync.Mutex
	pending   map[string]pendingOAuth
}

type pendingOAuth struct {
	appID     string
	appSecret string
	expiresAt time.Time
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(channelRepo repository.ChannelRepository, baseURL string) *OAuthHandler {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		sum := sha256.Sum256([]byte(strings.TrimSuffix(baseURL, "/") + ":oauth-state"))
		secret = sum[:]
	}

	return &OAuthHandler{
		channelRepo: channelRepo,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		stateSecret: secret,
		pending:     make(map[string]pendingOAuth),
	}
}

// storePendingOAuth remembers the app credentials for a state token until the
// callback consumes them. Expired entries are pruned on each insert.
func (h *OAuthHandler) storePendingOAuth(state, appID, appSecret string) {
	h.pendingMu.Lock()
	defer h.pendingMu.Unlock()
	now := time.Now()
	for key, entry := range h.pending {
		if now.After(entry.expiresAt) {
			delete(h.pending, key)
		}
	}
	h.pending[state] = pendingOAuth{
		appID:     appID,
		appSecret: appSecret,
		expiresAt: now.Add(10 * time.Minute),
	}
}

// takePendingOAuth consumes the app credentials stored for a state token.
func (h *OAuthHandler) takePendingOAuth(state string) (appID, appSecret string, ok bool) {
	h.pendingMu.Lock()
	defer h.pendingMu.Unlock()
	entry, found := h.pending[state]
	if !found || time.Now().After(entry.expiresAt) {
		delete(h.pending, state)
		return "", "", false
	}
	delete(h.pending, state)
	return entry.appID, entry.appSecret, true
}

// OAuthState stores OAuth state information
type OAuthState struct {
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	ChannelType string `json:"channel_type"`
	RedirectURL string `json:"redirect_url"`
	Timestamp   int64  `json:"timestamp"`
}

// generateState creates a secure random state token
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// encodeState encodes OAuth state as a JSON string
func encodeState(state *OAuthState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// decodeState decodes OAuth state from a hex-encoded JSON string
func decodeState(encoded string) (*OAuthState, error) {
	data, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var state OAuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (h *OAuthHandler) encodeSignedState(state *OAuthState) (string, error) {
	encoded, err := encodeState(state)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write([]byte(encoded))
	signature := hex.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func (h *OAuthHandler) decodeSignedState(encoded string) (*OAuthState, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid signed state")
	}

	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return nil, fmt.Errorf("invalid state signature")
	}

	return decodeState(parts[0])
}

// FacebookLoginRequest represents a request to initiate Facebook OAuth
type FacebookLoginRequest struct {
	AppID       string   `json:"app_id" binding:"required"`
	AppSecret   string   `json:"app_secret" binding:"required"`
	RedirectURL string   `json:"redirect_url"` // Frontend callback URL
	Scopes      []string `json:"scopes"`
}

// FacebookCallbackRequest represents the OAuth callback data. App credentials
// are optional: when omitted, the ones stored at login time (keyed by state)
// are used, so the browser does not need to retain the app secret.
type FacebookCallbackRequest struct {
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// FacebookPagesResponse represents the response containing user's pages
type FacebookPagesResponse struct {
	Pages []PageInfo `json:"pages"`
}

// PageInfo represents a Facebook Page
type PageInfo struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	AccessToken string         `json:"access_token"`
	Category    string         `json:"category"`
	PictureURL  string         `json:"picture_url,omitempty"`
	Instagram   *InstagramInfo `json:"instagram,omitempty"`
}

// InstagramInfo represents connected Instagram account info
type InstagramInfo struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture_url,omitempty"`
	FollowersCount int    `json:"followers_count"`
}

// FacebookLogin initiates the Facebook OAuth flow
// POST /api/v1/oauth/facebook/login
func (h *OAuthHandler) FacebookLogin(c *gin.Context) {
	var req FacebookLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tenantID := c.GetString(middleware.TenantIDKey)
	userID := c.GetString(middleware.UserIDKey)

	// Create state with tenant info
	state := &OAuthState{
		TenantID:    tenantID,
		UserID:      userID,
		ChannelType: "facebook",
		RedirectURL: req.RedirectURL,
		Timestamp:   time.Now().Unix(),
	}

	stateStr, err := h.encodeSignedState(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create state"})
		return
	}

	// Default scopes for Facebook Messenger
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"pages_messaging",
			"pages_read_engagement",
			"pages_manage_metadata",
			"pages_show_list",
		}
	}

	// Keep the app credentials server-side for the callback.
	h.storePendingOAuth(stateStr, req.AppID, req.AppSecret)

	// Generate login URL
	helper := facebook.NewOAuthHelper(req.AppID, req.AppSecret)
	redirectURI := h.baseURL + "/api/v1/oauth/facebook/callback"
	loginURL := helper.GetLoginURL(redirectURI, stateStr, scopes)

	c.JSON(http.StatusOK, gin.H{
		"login_url": loginURL,
		"state":     stateStr,
	})
}

// FacebookCallback handles the OAuth callback from Facebook
// POST /api/v1/oauth/facebook/callback
func (h *OAuthHandler) FacebookCallback(c *gin.Context) {
	var req FacebookCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Decode and validate state
	state, err := h.decodeSignedState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	// Check state age (max 10 minutes)
	if time.Now().Unix()-state.Timestamp > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state expired"})
		return
	}
	if state.ChannelType != "facebook" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	// Prefer the credentials stored at login time over client-supplied ones.
	if appID, appSecret, ok := h.takePendingOAuth(req.State); ok {
		req.AppID = appID
		req.AppSecret = appSecret
	}
	if req.AppID == "" || req.AppSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth session expired, restart the flow"})
		return
	}

	// Exchange code for token
	helper := facebook.NewOAuthHelper(req.AppID, req.AppSecret)
	redirectURI := h.baseURL + "/api/v1/oauth/facebook/callback"

	tokenResp, err := helper.ExchangeCodeForToken(c.Request.Context(), redirectURI, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange code: " + err.Error()})
		return
	}

	// Get long-lived token
	longLivedResp, err := helper.GetLongLivedToken(c.Request.Context(), tokenResp.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get long-lived token: " + err.Error()})
		return
	}

	// Get user's pages
	pagesResp, err := helper.GetUserPages(c.Request.Context(), longLivedResp.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pages: " + err.Error()})
		return
	}

	// Convert to response format
	pages := make([]PageInfo, 0, len(pagesResp.Data))
	for _, page := range pagesResp.Data {
		pageInfo := PageInfo{
			ID:          page.ID,
			Name:        page.Name,
			AccessToken: page.AccessToken,
			Category:    page.Category,
		}
		if page.Picture != nil {
			pageInfo.PictureURL = page.Picture.Data.URL
		}

		// Try to get connected Instagram account
		if page.AccessToken != "" {
			igClient := meta.NewClient(page.AccessToken, req.AppSecret)
			igAccount, err := igClient.GetInstagramAccount(c.Request.Context(), page.ID)
			if err == nil && igAccount != nil {
				pageInfo.Instagram = &InstagramInfo{
					ID:             igAccount.ID,
					Username:       igAccount.Username,
					Name:           igAccount.Name,
					ProfilePicture: igAccount.ProfilePic,
					FollowersCount: igAccount.FollowersCount,
				}
			}
		}

		pages = append(pages, pageInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"user_access_token": longLivedResp.AccessToken,
		"token_type":        longLivedResp.TokenType,
		"expires_in":        longLivedResp.ExpiresIn,
		"pages":             pages,
		"state":             state,
	})
}

// FacebookWebhookCallback handles the GET verification for Facebook webhooks
// GET /api/v1/oauth/facebook/webhook-callback
func (h *OAuthHandler) FacebookWebhookCallback(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	// For webhook verification, we need the verify_token from the request
	// The frontend should pass the expected verify_token
	expectedToken := c.Query("expected_token")

	if mode == "subscribe" && token == expectedToken {
		c.String(http.StatusOK, challenge)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "verification failed"})
}

// InstagramLoginRequest represents a request to initiate Instagram OAuth
type InstagramLoginRequest struct {
	AppID       string   `json:"app_id" binding:"required"`
	AppSecret   string   `json:"app_secret" binding:"required"`
	RedirectURL string   `json:"redirect_url"`
	Scopes      []string `json:"scopes"`
}

// InstagramLogin initiates the Instagram OAuth flow
// POST /api/v1/oauth/instagram/login
func (h *OAuthHandler) InstagramLogin(c *gin.Context) {
	var req InstagramLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tenantID := c.GetString(middleware.TenantIDKey)
	userID := c.GetString(middleware.UserIDKey)

	// Create state
	state := &OAuthState{
		TenantID:    tenantID,
		UserID:      userID,
		ChannelType: "instagram",
		RedirectURL: req.RedirectURL,
		Timestamp:   time.Now().Unix(),
	}

	stateStr, err := h.encodeSignedState(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create state"})
		return
	}

	// Default scopes for Instagram
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"instagram_basic",
			"instagram_manage_messages",
			"pages_manage_metadata",
			"pages_read_engagement",
			"pages_show_list",
		}
	}

	// Keep the app credentials server-side for the callback.
	h.storePendingOAuth(stateStr, req.AppID, req.AppSecret)

	// Generate login URL (Instagram uses Facebook Login)
	helper := instagram.NewOAuthHelper(req.AppID, req.AppSecret)
	redirectURI := h.baseURL + "/api/v1/oauth/instagram/callback"
	loginURL := helper.GetLoginURL(redirectURI, stateStr, scopes)

	c.JSON(http.StatusOK, gin.H{
		"login_url": loginURL,
		"state":     stateStr,
	})
}

// InstagramCallbackRequest represents the OAuth callback data for Instagram.
// App credentials are optional: when omitted, the ones stored at login time
// (keyed by state) are used.
type InstagramCallbackRequest struct {
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// InstagramCallback handles the OAuth callback from Instagram
// POST /api/v1/oauth/instagram/callback
func (h *OAuthHandler) InstagramCallback(c *gin.Context) {
	var req InstagramCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Decode and validate state
	state, err := h.decodeSignedState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	// Check state age
	if time.Now().Unix()-state.Timestamp > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state expired"})
		return
	}
	if state.ChannelType != "instagram" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	// Prefer the credentials stored at login time over client-supplied ones.
	if appID, appSecret, ok := h.takePendingOAuth(req.State); ok {
		req.AppID = appID
		req.AppSecret = appSecret
	}
	if req.AppID == "" || req.AppSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth session expired, restart the flow"})
		return
	}

	// Exchange code for token
	helper := instagram.NewOAuthHelper(req.AppID, req.AppSecret)
	redirectURI := h.baseURL + "/api/v1/oauth/instagram/callback"

	tokenResp, err := helper.ExchangeCodeForToken(c.Request.Context(), redirectURI, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange code: " + err.Error()})
		return
	}

	// Get long-lived token
	longLivedResp, err := helper.GetLongLivedToken(c.Request.Context(), tokenResp.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get long-lived token: " + err.Error()})
		return
	}

	// Get Instagram accounts via pages
	accounts, err := helper.GetInstagramAccounts(c.Request.Context(), longLivedResp.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get Instagram accounts: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_access_token": longLivedResp.AccessToken,
		"token_type":        longLivedResp.TokenType,
		"expires_in":        longLivedResp.ExpiresIn,
		"accounts":          accounts,
		"state":             state,
	})
}

// OAuthCreateChannelRequest represents a request to create a channel from OAuth
type OAuthCreateChannelRequest struct {
	Name        string            `json:"name" binding:"required"`
	Type        string            `json:"type" binding:"required"`
	PageID      string            `json:"page_id"`
	AccessToken string            `json:"access_token" binding:"required"`
	AppID       string            `json:"app_id" binding:"required"`
	AppSecret   string            `json:"app_secret" binding:"required"`
	VerifyToken string            `json:"verify_token"`
	InstagramID string            `json:"instagram_id"`
	Config      map[string]string `json:"config"`
}

// CreateChannel creates a new channel from OAuth credentials
// POST /api/v1/oauth/channels
func (h *OAuthHandler) CreateChannel(c *gin.Context) {
	var req OAuthCreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tenantID := c.GetString(middleware.TenantIDKey)

	// Validate channel type
	var channelType entity.ChannelType
	switch req.Type {
	case "facebook":
		channelType = entity.ChannelTypeFacebook
	case "instagram":
		channelType = entity.ChannelTypeInstagram
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel type"})
		return
	}

	// Generate verify token if not provided
	verifyToken := req.VerifyToken
	if verifyToken == "" {
		verifyToken = generateState()
	}

	// Build credentials
	credentials := map[string]string{
		"access_token": req.AccessToken,
		"app_id":       req.AppID,
		"app_secret":   req.AppSecret,
		"verify_token": verifyToken,
	}

	// Build config
	config := req.Config
	if config == nil {
		config = make(map[string]string)
	}

	if req.PageID != "" {
		config["page_id"] = req.PageID
	}
	if req.InstagramID != "" {
		config["instagram_id"] = req.InstagramID
	}

	// Create channel entity
	channel := &entity.Channel{
		TenantID:         tenantID,
		Name:             req.Name,
		Type:             channelType,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials:      credentials,
		Config:           config,
	}

	// Save to database
	if err := h.channelRepo.Create(c.Request.Context(), channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel: " + err.Error()})
		return
	}

	// Subscribe to webhooks
	if channelType == entity.ChannelTypeFacebook && req.PageID != "" {
		client, _ := facebook.NewClient(&facebook.FacebookConfig{
			PageID:          req.PageID,
			PageAccessToken: req.AccessToken,
			AppSecret:       req.AppSecret,
		})
		if client != nil {
			client.SubscribeToWebhooks(c.Request.Context())
		}
	}

	// Build webhook URL
	webhookURL := h.baseURL + "/api/v1/webhooks/" + string(channelType) + "/" + channel.ID

	c.JSON(http.StatusCreated, gin.H{
		"channel":      channel,
		"webhook_url":  webhookURL,
		"verify_token": verifyToken,
	})
}

// RefreshTokenRequest represents a request to refresh an access token. App
// credentials are optional: when omitted, the ones stored on the channel are
// used.
type RefreshTokenRequest struct {
	ChannelID string `json:"channel_id" binding:"required"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// RefreshToken refreshes the access token for a channel
// POST /api/v1/oauth/refresh
func (h *OAuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Request.Context(), req.ChannelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	currentToken := channel.Credentials["access_token"]
	if currentToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no access token found"})
		return
	}

	// Fall back to the app credentials stored on the channel.
	if req.AppID == "" {
		req.AppID = channel.Credentials["app_id"]
	}
	if req.AppSecret == "" {
		req.AppSecret = channel.Credentials["app_secret"]
	}
	if req.AppID == "" || req.AppSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app credentials not found for channel"})
		return
	}

	// Get new long-lived token
	helper := facebook.NewOAuthHelper(req.AppID, req.AppSecret)
	newToken, err := helper.GetLongLivedToken(c.Request.Context(), currentToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token: " + err.Error()})
		return
	}

	// Update channel credentials
	channel.Credentials["access_token"] = newToken.AccessToken
	if err := h.channelRepo.Update(c.Request.Context(), channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newToken.AccessToken,
		"expires_in":   newToken.ExpiresIn,
	})
}
