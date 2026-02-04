package whatsapp_official

import (
	"time"
)

// Config holds WhatsApp Business Cloud API configuration
type Config struct {
	AccessToken   string `json:"access_token"`
	PhoneNumberID string `json:"phone_number_id"`
	BusinessID    string `json:"business_id"`
	VerifyToken   string `json:"verify_token"`
	WebhookSecret string `json:"webhook_secret"`
	APIVersion    string `json:"api_version"` // e.g., v18.0
}

// DefaultAPIVersion is the default Meta Graph API version
const DefaultAPIVersion = "v21.0"

// BaseURL is the Meta Graph API base URL
const BaseURL = "https://graph.facebook.com"

// WebhookPayload represents the incoming webhook payload from Meta
type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

// WebhookEntry represents a single entry in the webhook payload
type WebhookEntry struct {
	ID      string          `json:"id"`
	Changes []WebhookChange `json:"changes"`
}

// WebhookChange represents a change in the webhook entry
type WebhookChange struct {
	Field string             `json:"field"`
	Value WebhookChangeValue `json:"value"`
}

// WebhookChangeValue represents the value of a webhook change
type WebhookChangeValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         WebhookMetadata  `json:"metadata"`
	Contacts         []ContactInfo    `json:"contacts,omitempty"`
	Messages         []IncomingMessage `json:"messages,omitempty"`
	Statuses         []StatusUpdate   `json:"statuses,omitempty"`
	Errors           []WebhookError   `json:"errors,omitempty"`
}

// WebhookMetadata represents metadata in the webhook
type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// WebhookError represents an error in the webhook
type WebhookError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Details string `json:"error_data,omitempty"`
}

// ContactInfo represents contact information in webhooks
type ContactInfo struct {
	WaID    string       `json:"wa_id"`
	Profile ContactProfile `json:"profile"`
}

// ContactProfile represents the contact profile
type ContactProfile struct {
	Name string `json:"name"`
}

// IncomingMessage represents an incoming message from WhatsApp
type IncomingMessage struct {
	ID        string          `json:"id"`
	From      string          `json:"from"`
	Timestamp string          `json:"timestamp"`
	Type      MessageType     `json:"type"`
	Text      *TextContent    `json:"text,omitempty"`
	Image     *MediaContent   `json:"image,omitempty"`
	Video     *MediaContent   `json:"video,omitempty"`
	Audio     *MediaContent   `json:"audio,omitempty"`
	Document  *DocumentContent `json:"document,omitempty"`
	Sticker   *StickerContent `json:"sticker,omitempty"`
	Location  *LocationContent `json:"location,omitempty"`
	Contacts  []ContactContent `json:"contacts,omitempty"`
	Interactive *InteractiveResponse `json:"interactive,omitempty"`
	Button    *ButtonResponse `json:"button,omitempty"`
	Context   *MessageContext `json:"context,omitempty"`
	Reaction  *ReactionContent `json:"reaction,omitempty"`
	Referral  *ReferralContent `json:"referral,omitempty"`
	Errors    []WebhookError   `json:"errors,omitempty"`
}

// MessageType represents the type of WhatsApp message
type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeImage       MessageType = "image"
	MessageTypeVideo       MessageType = "video"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeDocument    MessageType = "document"
	MessageTypeSticker     MessageType = "sticker"
	MessageTypeLocation    MessageType = "location"
	MessageTypeContacts    MessageType = "contacts"
	MessageTypeInteractive MessageType = "interactive"
	MessageTypeButton      MessageType = "button"
	MessageTypeReaction    MessageType = "reaction"
	MessageTypeUnknown     MessageType = "unknown"
	MessageTypeOrder       MessageType = "order"
	MessageTypeSystem      MessageType = "system"
	MessageTypeUnsupported MessageType = "unsupported"
)

// TextContent represents text message content
type TextContent struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

// MediaContent represents media content (image, video, audio)
type MediaContent struct {
	ID       string `json:"id"`
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// DocumentContent represents document content
type DocumentContent struct {
	ID       string `json:"id"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
}

// StickerContent represents sticker content
type StickerContent struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Animated bool   `json:"animated,omitempty"`
}

// LocationContent represents location content
type LocationContent struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ContactContent represents contact content
type ContactContent struct {
	Addresses []ContactAddress `json:"addresses,omitempty"`
	Birthday  string           `json:"birthday,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Name      ContactName      `json:"name"`
	Org       *ContactOrg      `json:"org,omitempty"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	URLs      []ContactURL     `json:"urls,omitempty"`
}

// ContactAddress represents a contact address
type ContactAddress struct {
	City        string `json:"city,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	State       string `json:"state,omitempty"`
	Street      string `json:"street,omitempty"`
	Type        string `json:"type,omitempty"`
	Zip         string `json:"zip,omitempty"`
}

// ContactEmail represents a contact email
type ContactEmail struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

// ContactName represents a contact name
type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

// ContactOrg represents a contact organization
type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

// ContactPhone represents a contact phone
type ContactPhone struct {
	Phone string `json:"phone,omitempty"`
	Type  string `json:"type,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
}

// ContactURL represents a contact URL
type ContactURL struct {
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

// InteractiveResponse represents an interactive message response
type InteractiveResponse struct {
	Type       string            `json:"type"`
	ButtonReply *ButtonReplyData `json:"button_reply,omitempty"`
	ListReply  *ListReplyData    `json:"list_reply,omitempty"`
}

// ButtonReplyData represents button reply data
type ButtonReplyData struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ListReplyData represents list reply data
type ListReplyData struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ButtonResponse represents a button template response
type ButtonResponse struct {
	Text    string `json:"text"`
	Payload string `json:"payload"`
}

// MessageContext represents the context of a message (reply-to)
type MessageContext struct {
	MessageID string `json:"id"`
	From      string `json:"from"`
	Forwarded bool   `json:"forwarded,omitempty"`
	FrequentlyForwarded bool `json:"frequently_forwarded,omitempty"`
	ReferredProduct *ReferredProduct `json:"referred_product,omitempty"`
}

// ReferredProduct represents a referred product in context
type ReferredProduct struct {
	CatalogID         string `json:"catalog_id"`
	ProductRetailerID string `json:"product_retailer_id"`
}

// ReactionContent represents a reaction message
type ReactionContent struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// ReferralContent represents referral data (ads, click-to-WhatsApp)
type ReferralContent struct {
	SourceURL  string `json:"source_url,omitempty"`
	SourceType string `json:"source_type,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	Headline   string `json:"headline,omitempty"`
	Body       string `json:"body,omitempty"`
	MediaType  string `json:"media_type,omitempty"`
	ImageURL   string `json:"image_url,omitempty"`
	VideoURL   string `json:"video_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

// StatusUpdate represents a message status update
type StatusUpdate struct {
	ID          string        `json:"id"`
	RecipientID string        `json:"recipient_id"`
	Status      MessageStatus `json:"status"`
	Timestamp   string        `json:"timestamp"`
	Conversation *ConversationInfo `json:"conversation,omitempty"`
	Pricing     *PricingInfo  `json:"pricing,omitempty"`
	Errors      []WebhookError `json:"errors,omitempty"`
}

// MessageStatus represents the status of a message
type MessageStatus string

const (
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
)

// ConversationInfo represents conversation info in status updates
type ConversationInfo struct {
	ID     string            `json:"id"`
	Origin *ConversationOrigin `json:"origin,omitempty"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
}

// ConversationOrigin represents the origin of a conversation
type ConversationOrigin struct {
	Type string `json:"type"` // business_initiated, user_initiated, referral_conversion
}

// PricingInfo represents pricing info in status updates
type PricingInfo struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"` // authentication, marketing, utility, service
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	MessagingProduct string      `json:"messaging_product"`
	RecipientType    string      `json:"recipient_type,omitempty"`
	To               string      `json:"to"`
	Type             MessageType `json:"type"`
	Text             *TextContent `json:"text,omitempty"`
	Image            *MediaObject `json:"image,omitempty"`
	Video            *MediaObject `json:"video,omitempty"`
	Audio            *MediaObject `json:"audio,omitempty"`
	Document         *DocumentObject `json:"document,omitempty"`
	Sticker          *MediaObject `json:"sticker,omitempty"`
	Location         *LocationObject `json:"location,omitempty"`
	Contacts         []ContactContent `json:"contacts,omitempty"`
	Interactive      *InteractiveObject `json:"interactive,omitempty"`
	Template         *TemplateObject `json:"template,omitempty"`
	Reaction         *ReactionObject `json:"reaction,omitempty"`
	Context          *ContextObject `json:"context,omitempty"`
}

// MediaObject represents media to send (by ID or URL)
type MediaObject struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// DocumentObject represents a document to send
type DocumentObject struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename"`
}

// LocationObject represents a location to send
type LocationObject struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ReactionObject represents a reaction to send
type ReactionObject struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// ContextObject represents context for reply-to
type ContextObject struct {
	MessageID string `json:"message_id"`
}

// SendMessageResponse represents the response from sending a message
type SendMessageResponse struct {
	MessagingProduct string               `json:"messaging_product"`
	Contacts         []SendMessageContact `json:"contacts"`
	Messages         []SendMessageResult  `json:"messages"`
}

// SendMessageContact represents a contact in the send response
type SendMessageContact struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

// SendMessageResult represents a message result in the send response
type SendMessageResult struct {
	ID            string `json:"id"`
	MessageStatus string `json:"message_status,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError represents an API error
type APIError struct {
	Message      string    `json:"message"`
	Type         string    `json:"type"`
	Code         int       `json:"code"`
	ErrorSubcode int       `json:"error_subcode,omitempty"`
	FBTraceID    string    `json:"fbtrace_id,omitempty"`
	ErrorData    *ErrorData `json:"error_data,omitempty"`
}

// ErrorData represents additional error data
type ErrorData struct {
	MessagingProduct string `json:"messaging_product,omitempty"`
	Details          string `json:"details,omitempty"`
}

// MediaUploadResponse represents the response from uploading media
type MediaUploadResponse struct {
	ID string `json:"id"`
}

// MediaInfoResponse represents the response from getting media info
type MediaInfoResponse struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	MimeType     string `json:"mime_type"`
	SHA256       string `json:"sha256"`
	FileSize     int64  `json:"file_size"`
	MessagingProduct string `json:"messaging_product,omitempty"`
}

// HealthStatus represents the health status of WhatsApp
type HealthStatus struct {
	Health []HealthInfo `json:"health"`
}

// HealthInfo represents health information
type HealthInfo struct {
	EntityType  string            `json:"entity_type"`
	CanSendMessage string         `json:"can_send_message"`
	Errors      []HealthError     `json:"errors,omitempty"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
}

// HealthError represents a health error
type HealthError struct {
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	PossibleSolution string `json:"possible_solution,omitempty"`
}

// BusinessProfile represents a WhatsApp Business Profile
type BusinessProfile struct {
	About             string   `json:"about,omitempty"`
	Address           string   `json:"address,omitempty"`
	Description       string   `json:"description,omitempty"`
	Email             string   `json:"email,omitempty"`
	MessagingProduct  string   `json:"messaging_product,omitempty"`
	ProfilePictureURL string   `json:"profile_picture_url,omitempty"`
	Websites          []string `json:"websites,omitempty"`
	Vertical          string   `json:"vertical,omitempty"`
}

// PhoneNumberInfo represents phone number information
type PhoneNumberInfo struct {
	ID                    string `json:"id"`
	VerifiedName          string `json:"verified_name"`
	DisplayPhoneNumber    string `json:"display_phone_number"`
	QualityRating         string `json:"quality_rating"`
	Status                string `json:"status,omitempty"`
	NameStatus            string `json:"name_status,omitempty"`
	NewNameStatus         string `json:"new_name_status,omitempty"`
	MessagingLimitTier    string `json:"messaging_limit_tier,omitempty"`
	CodeVerificationStatus string `json:"code_verification_status,omitempty"`
}

// SessionInfo tracks the 24-hour messaging window
type SessionInfo struct {
	ContactID              string    `json:"contact_id"`
	LastCustomerMessageAt  time.Time `json:"last_customer_message_at"`
	SessionExpiresAt       time.Time `json:"session_expires_at"`
	CanSendSessionMessage  bool      `json:"can_send_session_message"`
}

// IsSessionValid checks if the 24-hour session is still valid
func (s *SessionInfo) IsSessionValid() bool {
	return s.CanSendSessionMessage && time.Now().Before(s.SessionExpiresAt)
}

// UpdateSession updates the session after receiving a customer message
func (s *SessionInfo) UpdateSession() {
	s.LastCustomerMessageAt = time.Now()
	s.SessionExpiresAt = s.LastCustomerMessageAt.Add(24 * time.Hour)
	s.CanSendSessionMessage = true
}

// RateLimitInfo tracks rate limiting
type RateLimitInfo struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}
