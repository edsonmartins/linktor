package meta

import "time"

// API Constants
const (
	// Graph API Base URL
	GraphAPIBaseURL = "https://graph.facebook.com"

	// Instagram Graph API Base URL
	InstagramAPIBaseURL = "https://graph.instagram.com"

	// Default API Version
	DefaultAPIVersion = "v22.0"
)

// Config holds common Meta API configuration
type Config struct {
	AccessToken   string `json:"access_token"`
	AppID         string `json:"app_id"`
	AppSecret     string `json:"app_secret"`
	VerifyToken   string `json:"verify_token"`
	APIVersion    string `json:"api_version"`
}

// WebhookVerification represents webhook verification request
type WebhookVerification struct {
	Mode      string `json:"hub.mode"`
	Token     string `json:"hub.verify_token"`
	Challenge string `json:"hub.challenge"`
}

// WebhookPayload represents the common webhook payload structure
type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

// WebhookEntry represents a single entry in the webhook payload
type WebhookEntry struct {
	ID        string           `json:"id"`
	Time      int64            `json:"time"`
	Messaging []MessagingEvent `json:"messaging,omitempty"`
	Standby   []MessagingEvent `json:"standby,omitempty"`
	Changes   []WebhookChange  `json:"changes,omitempty"`
}

// WebhookChange represents a change event (used for some webhook types)
type WebhookChange struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// MessagingEvent represents a messaging event from webhook
type MessagingEvent struct {
	Sender    MessagingParty   `json:"sender"`
	Recipient MessagingParty   `json:"recipient"`
	Timestamp int64            `json:"timestamp"`
	Message   *InboundMessage  `json:"message,omitempty"`
	Delivery  *DeliveryStatus  `json:"delivery,omitempty"`
	Read      *ReadStatus      `json:"read,omitempty"`
	Postback  *Postback        `json:"postback,omitempty"`
	Reaction  *ReactionEvent   `json:"reaction,omitempty"`
}

// MessagingParty represents a sender or recipient
type MessagingParty struct {
	ID string `json:"id"`
}

// InboundMessage represents an incoming message
type InboundMessage struct {
	MID         string            `json:"mid"`
	Text        string            `json:"text,omitempty"`
	IsEcho      bool              `json:"is_echo,omitempty"`
	IsDeleted   bool              `json:"is_deleted,omitempty"`
	Attachments []InboundAttachment `json:"attachments,omitempty"`
	QuickReply  *QuickReplyPayload `json:"quick_reply,omitempty"`
	ReplyTo     *ReplyTo          `json:"reply_to,omitempty"`
}

// InboundAttachment represents an attachment in incoming message
type InboundAttachment struct {
	Type    string            `json:"type"`
	Payload AttachmentPayload `json:"payload"`
}

// AttachmentPayload holds the attachment content
type AttachmentPayload struct {
	URL         string  `json:"url,omitempty"`
	StickerID   int64   `json:"sticker_id,omitempty"`
	Title       string  `json:"title,omitempty"`
	Coordinates *Coordinates `json:"coordinates,omitempty"`
}

// Coordinates for location attachments
type Coordinates struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

// QuickReplyPayload represents a quick reply response
type QuickReplyPayload struct {
	Payload string `json:"payload"`
}

// ReplyTo represents the message being replied to
type ReplyTo struct {
	MID string `json:"mid"`
}

// DeliveryStatus represents message delivery status
type DeliveryStatus struct {
	MIDs      []string `json:"mids,omitempty"`
	Watermark int64    `json:"watermark,omitempty"`
}

// ReadStatus represents message read status
type ReadStatus struct {
	Watermark int64 `json:"watermark,omitempty"`
}

// Postback represents a postback event
type Postback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

// ReactionEvent represents a reaction to a message
type ReactionEvent struct {
	MID      string `json:"mid"`
	Action   string `json:"action"` // "react" or "unreact"
	Reaction string `json:"reaction,omitempty"`
	Emoji    string `json:"emoji,omitempty"`
}

// OutboundMessage represents a message to send
type OutboundMessage struct {
	Recipient    MessageRecipient     `json:"recipient"`
	Message      MessageContent       `json:"message"`
	MessagingType string              `json:"messaging_type,omitempty"`
	Tag          string               `json:"tag,omitempty"`
}

// MessageRecipient specifies the message recipient
type MessageRecipient struct {
	ID string `json:"id"`
}

// MessageContent holds the message content
type MessageContent struct {
	Text         string            `json:"text,omitempty"`
	Attachment   *MessageAttachment `json:"attachment,omitempty"`
	QuickReplies []QuickReply      `json:"quick_replies,omitempty"`
}

// MessageAttachment for sending media
type MessageAttachment struct {
	Type    string                   `json:"type"`
	Payload MessageAttachmentPayload `json:"payload"`
}

// MessageAttachmentPayload holds attachment details
type MessageAttachmentPayload struct {
	URL        string `json:"url,omitempty"`
	IsReusable bool   `json:"is_reusable,omitempty"`
}

// QuickReply represents a quick reply button
type QuickReply struct {
	ContentType string `json:"content_type"` // "text", "user_phone_number", "user_email"
	Title       string `json:"title,omitempty"`
	Payload     string `json:"payload,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// SendMessageResponse represents the API response when sending a message
type SendMessageResponse struct {
	RecipientID string `json:"recipient_id,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
	Error       *APIError `json:"error,omitempty"`
}

// APIError represents a Meta API error response
type APIError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode,omitempty"`
	FBTraceID    string `json:"fbtrace_id,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// UserProfile represents user profile data
type UserProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	ProfilePic string `json:"profile_pic,omitempty"`
	Locale    string `json:"locale,omitempty"`
	Timezone  int    `json:"timezone,omitempty"`
	Gender    string `json:"gender,omitempty"`
}

// PageInfo represents Facebook Page information
type PageInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token,omitempty"`
	Category    string `json:"category,omitempty"`
	Picture     *PagePicture `json:"picture,omitempty"`
}

// PagePicture represents page profile picture
type PagePicture struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
}

// PagesResponse represents the response from /me/accounts
type PagesResponse struct {
	Data   []PageInfo `json:"data"`
	Paging *Paging    `json:"paging,omitempty"`
}

// Paging represents pagination info
type Paging struct {
	Cursors struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"cursors"`
	Next string `json:"next,omitempty"`
}

// InstagramAccount represents an Instagram business account
type InstagramAccount struct {
	ID          string `json:"id"`
	Username    string `json:"username,omitempty"`
	Name        string `json:"name,omitempty"`
	ProfilePic  string `json:"profile_picture_url,omitempty"`
	FollowersCount int `json:"followers_count,omitempty"`
}

// OAuthTokenResponse represents the OAuth token exchange response
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
	Error       *OAuthError `json:"error,omitempty"`
}

// OAuthError represents an OAuth error
type OAuthError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

// LongLivedTokenResponse represents the response for long-lived token exchange
type LongLivedTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// SubscribedAppsResponse represents the response from subscribing to app webhooks
type SubscribedAppsResponse struct {
	Success bool      `json:"success"`
	Error   *APIError `json:"error,omitempty"`
}

// SenderAction represents typing indicator or mark seen action
type SenderAction struct {
	Recipient    MessageRecipient `json:"recipient"`
	SenderAction string           `json:"sender_action"` // "typing_on", "typing_off", "mark_seen"
}

// ParsedInboundMessage represents a normalized incoming message
type ParsedInboundMessage struct {
	ID            string
	ExternalID    string
	SenderID      string
	RecipientID   string
	Text          string
	Attachments   []ParsedAttachment
	IsEcho        bool
	IsDeleted     bool
	QuickReply    string
	ReplyToMID    string
	Timestamp     time.Time
}

// ParsedAttachment represents a normalized attachment
type ParsedAttachment struct {
	Type     string
	URL      string
	Title    string
	Lat      float64
	Long     float64
}
