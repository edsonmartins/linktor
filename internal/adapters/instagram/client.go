package instagram

import (
	"context"
	"fmt"
	"time"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// Client wraps the Instagram Graph API for DMs
type Client struct {
	api    *meta.Client
	config *InstagramConfig
}

// NewClient creates a new Instagram DM client
func NewClient(config *InstagramConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Use Instagram Graph API base URL
	api := meta.NewInstagramClient(config.GetEffectiveAccessToken(), config.AppSecret)

	return &Client{
		api:    api,
		config: config,
	}, nil
}

// GetAccountInfo retrieves information about the Instagram account
func (c *Client) GetAccountInfo(ctx context.Context) (*meta.InstagramAccount, error) {
	profile, err := c.api.GetUserProfile(ctx, c.config.InstagramID, []string{"id", "username", "name"})
	if err != nil {
		return nil, err
	}

	return &meta.InstagramAccount{
		ID:       profile.ID,
		Username: profile.Name, // Instagram returns username in name field for business accounts
		Name:     profile.FirstName,
	}, nil
}

// SendTextMessage sends a text message
func (c *Client) SendTextMessage(ctx context.Context, recipientID, text string) (*meta.SendMessageResponse, error) {
	msg := &meta.OutboundMessage{
		Recipient: meta.MessageRecipient{ID: recipientID},
		Message: meta.MessageContent{
			Text: text,
		},
	}

	return c.api.SendInstagramMessage(ctx, c.config.InstagramID, msg)
}

// SendAttachment sends a media attachment
func (c *Client) SendAttachment(ctx context.Context, recipientID, attachmentType, url string) (*meta.SendMessageResponse, error) {
	msg := &meta.OutboundMessage{
		Recipient: meta.MessageRecipient{ID: recipientID},
		Message: meta.MessageContent{
			Attachment: &meta.MessageAttachment{
				Type: attachmentType,
				Payload: meta.MessageAttachmentPayload{
					URL: url,
				},
			},
		},
	}

	return c.api.SendInstagramMessage(ctx, c.config.InstagramID, msg)
}

// SendImage sends an image attachment
func (c *Client) SendImage(ctx context.Context, recipientID, imageURL string) (*meta.SendMessageResponse, error) {
	return c.SendAttachment(ctx, recipientID, "image", imageURL)
}

// SendVideo sends a video attachment
func (c *Client) SendVideo(ctx context.Context, recipientID, videoURL string) (*meta.SendMessageResponse, error) {
	return c.SendAttachment(ctx, recipientID, "video", videoURL)
}

// SendAudio sends an audio attachment
func (c *Client) SendAudio(ctx context.Context, recipientID, audioURL string) (*meta.SendMessageResponse, error) {
	return c.SendAttachment(ctx, recipientID, "audio", audioURL)
}

// SendFile sends a file attachment
func (c *Client) SendFile(ctx context.Context, recipientID, fileURL string) (*meta.SendMessageResponse, error) {
	return c.SendAttachment(ctx, recipientID, "file", fileURL)
}

// SubscribeToWebhooks subscribes the account to webhook events
func (c *Client) SubscribeToWebhooks(ctx context.Context) error {
	fields := []string{
		"messages",
		"message_reactions",
		"messaging_seen",
	}
	return c.api.SubscribeToWebhook(ctx, c.config.InstagramID, fields)
}

// UnsubscribeFromWebhooks unsubscribes from webhook events
func (c *Client) UnsubscribeFromWebhooks(ctx context.Context) error {
	return c.api.UnsubscribeFromWebhook(ctx, c.config.InstagramID)
}

// ValidateWebhookSignature validates the webhook signature
func (c *Client) ValidateWebhookSignature(payload []byte, signature string) bool {
	if c.config.AppSecret == "" {
		return true // Skip validation if no app secret
	}
	return meta.ValidateWebhookSignature(c.config.AppSecret, payload, signature)
}

// ConvertIncomingMessage converts a meta.MessagingEvent to IncomingMessage
func ConvertIncomingMessage(event *meta.MessagingEvent, instagramID string) *IncomingMessage {
	if event.Message == nil {
		return nil
	}

	msg := &IncomingMessage{
		ExternalID:  event.Message.MID,
		SenderID:    event.Sender.ID,
		RecipientID: event.Recipient.ID,
		InstagramID: instagramID,
		Text:        event.Message.Text,
		IsEcho:      event.Message.IsEcho,
		IsDeleted:   event.Message.IsDeleted,
	}

	// Convert timestamp
	if event.Timestamp > 0 {
		msg.Timestamp = time.UnixMilli(event.Timestamp)
	}

	// Convert attachments
	for _, att := range event.Message.Attachments {
		attachment := Attachment{
			Type: att.Type,
			URL:  att.Payload.URL,
		}
		msg.Attachments = append(msg.Attachments, attachment)
	}

	return msg
}

// GetAttachmentType returns the normalized attachment type
func GetAttachmentType(igType string) string {
	switch igType {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "file":
		return "document"
	case "share":
		return "link"
	case "story_mention":
		return "story_mention"
	default:
		return "file"
	}
}

// OAuthHelper provides OAuth-related functionality for Instagram
type OAuthHelper struct {
	AppID     string
	AppSecret string
}

// NewOAuthHelper creates a new OAuth helper
func NewOAuthHelper(appID, appSecret string) *OAuthHelper {
	return &OAuthHelper{
		AppID:     appID,
		AppSecret: appSecret,
	}
}

// GetLoginURL generates the Instagram Login URL
// Instagram uses Facebook OAuth for business accounts
func (h *OAuthHelper) GetLoginURL(redirectURI, state string, scopes []string) string {
	scopeStr := "instagram_basic,instagram_manage_messages,pages_manage_metadata,pages_read_engagement,pages_show_list"
	if len(scopes) > 0 {
		scopeStr = ""
		for i, s := range scopes {
			if i > 0 {
				scopeStr += ","
			}
			scopeStr += s
		}
	}

	return fmt.Sprintf(
		"https://www.facebook.com/%s/dialog/oauth?client_id=%s&redirect_uri=%s&state=%s&scope=%s",
		meta.DefaultAPIVersion,
		h.AppID,
		redirectURI,
		state,
		scopeStr,
	)
}

// GetInstagramLoginURL generates the Instagram Direct Login URL (for IG direct integration)
func (h *OAuthHelper) GetInstagramLoginURL(redirectURI, state string) string {
	scopes := "instagram_business_basic,instagram_business_manage_messages"

	return fmt.Sprintf(
		"https://www.instagram.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		h.AppID,
		redirectURI,
		scopes,
		state,
	)
}

// ExchangeCodeForToken exchanges an OAuth code for tokens
func (h *OAuthHelper) ExchangeCodeForToken(ctx context.Context, redirectURI, code string) (*meta.OAuthTokenResponse, error) {
	client := meta.NewClient("", h.AppSecret)
	return client.ExchangeCodeForToken(ctx, h.AppID, h.AppSecret, redirectURI, code)
}

// GetLongLivedToken exchanges a short-lived token for a long-lived one
func (h *OAuthHelper) GetLongLivedToken(ctx context.Context, shortLivedToken string) (*meta.LongLivedTokenResponse, error) {
	client := meta.NewClient("", h.AppSecret)
	return client.GetLongLivedToken(ctx, h.AppID, h.AppSecret, shortLivedToken)
}

// GetInstagramAccountFromPage retrieves the Instagram account connected to a Facebook page
func (h *OAuthHelper) GetInstagramAccountFromPage(ctx context.Context, pageAccessToken, pageID string) (*meta.InstagramAccount, error) {
	client := meta.NewClient(pageAccessToken, h.AppSecret)
	return client.GetInstagramAccount(ctx, pageID)
}

// GetUserPages retrieves pages the user has access to (for IG via FB flow)
func (h *OAuthHelper) GetUserPages(ctx context.Context, userAccessToken string) (*meta.PagesResponse, error) {
	client := meta.NewClient(userAccessToken, h.AppSecret)
	return client.GetMyPages(ctx)
}

// InstagramAccountInfo represents Instagram account information for OAuth response
type InstagramAccountInfo struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	ProfilePictureURL string `json:"profile_picture_url,omitempty"`
	FollowersCount    int    `json:"followers_count"`
	PageID            string `json:"page_id"`
	PageName          string `json:"page_name"`
	PageAccessToken   string `json:"page_access_token"`
}

// GetInstagramAccounts retrieves Instagram accounts connected to user's Facebook pages
func (h *OAuthHelper) GetInstagramAccounts(ctx context.Context, userAccessToken string) ([]InstagramAccountInfo, error) {
	// Get user's pages first
	pages, err := h.GetUserPages(ctx, userAccessToken)
	if err != nil {
		return nil, err
	}

	var accounts []InstagramAccountInfo

	// For each page, check if it has a connected Instagram account
	for _, page := range pages.Data {
		if page.AccessToken == "" {
			continue
		}

		igAccount, err := h.GetInstagramAccountFromPage(ctx, page.AccessToken, page.ID)
		if err != nil {
			// Page doesn't have Instagram connected, skip
			continue
		}

		if igAccount != nil {
			accounts = append(accounts, InstagramAccountInfo{
				ID:                igAccount.ID,
				Username:          igAccount.Username,
				Name:              igAccount.Name,
				ProfilePictureURL: igAccount.ProfilePic,
				FollowersCount:    igAccount.FollowersCount,
				PageID:            page.ID,
				PageName:          page.Name,
				PageAccessToken:   page.AccessToken,
			})
		}
	}

	return accounts, nil
}
