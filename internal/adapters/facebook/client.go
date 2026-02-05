package facebook

import (
	"context"
	"fmt"
	"time"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// Client wraps the Meta Graph API for Facebook Messenger
type Client struct {
	api    *meta.Client
	config *FacebookConfig
}

// NewClient creates a new Facebook Messenger client
func NewClient(config *FacebookConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		api:    meta.NewClient(config.PageAccessToken, config.AppSecret),
		config: config,
	}, nil
}

// GetPageInfo retrieves information about the configured page
func (c *Client) GetPageInfo(ctx context.Context) (*meta.PageInfo, error) {
	return c.api.GetPageInfo(ctx, c.config.PageID)
}

// GetUserProfile retrieves a user's profile
func (c *Client) GetUserProfile(ctx context.Context, userID string) (*meta.UserProfile, error) {
	return c.api.GetUserProfile(ctx, userID, nil)
}

// SendTextMessage sends a text message
func (c *Client) SendTextMessage(ctx context.Context, recipientID, text string) (*meta.SendMessageResponse, error) {
	msg := &meta.OutboundMessage{
		Recipient: meta.MessageRecipient{ID: recipientID},
		Message: meta.MessageContent{
			Text: text,
		},
	}

	return c.api.SendMessage(ctx, c.config.PageID, msg)
}

// SendMessageWithQuickReplies sends a message with quick reply buttons
func (c *Client) SendMessageWithQuickReplies(ctx context.Context, recipientID, text string, quickReplies []QuickReply) (*meta.SendMessageResponse, error) {
	metaQuickReplies := make([]meta.QuickReply, len(quickReplies))
	for i, qr := range quickReplies {
		metaQuickReplies[i] = meta.QuickReply{
			ContentType: "text",
			Title:       qr.Title,
			Payload:     qr.Payload,
		}
	}

	msg := &meta.OutboundMessage{
		Recipient: meta.MessageRecipient{ID: recipientID},
		Message: meta.MessageContent{
			Text:         text,
			QuickReplies: metaQuickReplies,
		},
	}

	return c.api.SendMessage(ctx, c.config.PageID, msg)
}

// SendAttachment sends a media attachment
func (c *Client) SendAttachment(ctx context.Context, recipientID, attachmentType, url string) (*meta.SendMessageResponse, error) {
	msg := &meta.OutboundMessage{
		Recipient: meta.MessageRecipient{ID: recipientID},
		Message: meta.MessageContent{
			Attachment: &meta.MessageAttachment{
				Type: attachmentType,
				Payload: meta.MessageAttachmentPayload{
					URL:        url,
					IsReusable: true,
				},
			},
		},
	}

	return c.api.SendMessage(ctx, c.config.PageID, msg)
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

// SendTypingOn sends typing indicator on
func (c *Client) SendTypingOn(ctx context.Context, recipientID string) error {
	action := &meta.SenderAction{
		Recipient:    meta.MessageRecipient{ID: recipientID},
		SenderAction: "typing_on",
	}
	return c.api.SendSenderAction(ctx, c.config.PageID, action)
}

// SendTypingOff sends typing indicator off
func (c *Client) SendTypingOff(ctx context.Context, recipientID string) error {
	action := &meta.SenderAction{
		Recipient:    meta.MessageRecipient{ID: recipientID},
		SenderAction: "typing_off",
	}
	return c.api.SendSenderAction(ctx, c.config.PageID, action)
}

// MarkSeen marks messages as seen
func (c *Client) MarkSeen(ctx context.Context, recipientID string) error {
	action := &meta.SenderAction{
		Recipient:    meta.MessageRecipient{ID: recipientID},
		SenderAction: "mark_seen",
	}
	return c.api.SendSenderAction(ctx, c.config.PageID, action)
}

// SubscribeToWebhooks subscribes the page to webhook events
func (c *Client) SubscribeToWebhooks(ctx context.Context) error {
	fields := []string{
		"messages",
		"messaging_postbacks",
		"messaging_optins",
		"messaging_optouts",
		"message_deliveries",
		"message_reads",
		"messaging_handovers",
		"standby",
	}
	return c.api.SubscribeToWebhook(ctx, c.config.PageID, fields)
}

// UnsubscribeFromWebhooks unsubscribes the page from webhook events
func (c *Client) UnsubscribeFromWebhooks(ctx context.Context) error {
	return c.api.UnsubscribeFromWebhook(ctx, c.config.PageID)
}

// ValidateWebhookSignature validates the webhook signature
func (c *Client) ValidateWebhookSignature(payload []byte, signature string) bool {
	if c.config.AppSecret == "" {
		return true // Skip validation if no app secret
	}
	return meta.ValidateWebhookSignature(c.config.AppSecret, payload, signature)
}

// GetConnectedInstagramAccount retrieves the connected Instagram account
func (c *Client) GetConnectedInstagramAccount(ctx context.Context) (*meta.InstagramAccount, error) {
	return c.api.GetInstagramAccount(ctx, c.config.PageID)
}

// ConvertIncomingMessage converts a meta.MessagingEvent to IncomingMessage
func ConvertIncomingMessage(event *meta.MessagingEvent, pageID string) *IncomingMessage {
	if event.Message == nil {
		return nil
	}

	msg := &IncomingMessage{
		ExternalID:  event.Message.MID,
		SenderID:    event.Sender.ID,
		RecipientID: event.Recipient.ID,
		PageID:      pageID,
		Text:        event.Message.Text,
		IsEcho:      event.Message.IsEcho,
	}

	if event.Message.QuickReply != nil {
		msg.QuickReply = event.Message.QuickReply.Payload
	}

	// Convert timestamp
	if event.Timestamp > 0 {
		msg.Timestamp = msToTime(event.Timestamp)
	}

	// Convert attachments
	for _, att := range event.Message.Attachments {
		attachment := Attachment{
			Type:  att.Type,
			URL:   att.Payload.URL,
			Title: att.Payload.Title,
		}
		if att.Payload.Coordinates != nil {
			attachment.Lat = att.Payload.Coordinates.Lat
			attachment.Long = att.Payload.Coordinates.Long
		}
		msg.Attachments = append(msg.Attachments, attachment)
	}

	return msg
}

// ConvertDeliveryStatus converts a meta.MessagingEvent to DeliveryStatus
func ConvertDeliveryStatus(event *meta.MessagingEvent) *DeliveryStatus {
	if event.Delivery == nil {
		return nil
	}

	return &DeliveryStatus{
		MessageIDs: event.Delivery.MIDs,
		Watermark:  msToTime(event.Delivery.Watermark),
	}
}

// ConvertReadStatus converts a meta.MessagingEvent to ReadStatus
func ConvertReadStatus(event *meta.MessagingEvent) *ReadStatus {
	if event.Read == nil {
		return nil
	}

	return &ReadStatus{
		Watermark: msToTime(event.Read.Watermark),
	}
}

// ConvertPostback converts a meta.MessagingEvent to Postback
func ConvertPostback(event *meta.MessagingEvent) *Postback {
	if event.Postback == nil {
		return nil
	}

	return &Postback{
		Title:   event.Postback.Title,
		Payload: event.Postback.Payload,
	}
}

// msToTime converts milliseconds timestamp to time.Time
func msToTime(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// GetAttachmentType returns the normalized attachment type
func GetAttachmentType(fbType string) string {
	switch fbType {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "file":
		return "document"
	case "location":
		return "location"
	case "fallback":
		return "link"
	default:
		return "file"
	}
}

// OAuthHelper provides OAuth-related functionality
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

// GetLoginURL generates the Facebook Login URL
func (h *OAuthHelper) GetLoginURL(redirectURI, state string, scopes []string) string {
	scopeStr := "pages_messaging,pages_read_engagement,pages_manage_metadata"
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

// GetUserPages retrieves pages the user has access to
func (h *OAuthHelper) GetUserPages(ctx context.Context, userAccessToken string) (*meta.PagesResponse, error) {
	client := meta.NewClient(userAccessToken, h.AppSecret)
	return client.GetMyPages(ctx)
}
