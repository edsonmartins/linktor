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
// This is a union type that can contain different fields depending on the webhook field type
type WebhookChangeValue struct {
	// Common fields
	MessagingProduct string           `json:"messaging_product,omitempty"`
	Metadata         WebhookMetadata  `json:"metadata,omitempty"`
	Errors           []WebhookError   `json:"errors,omitempty"`

	// messages field
	Contacts         []ContactInfo     `json:"contacts,omitempty"`
	Messages         []IncomingMessage `json:"messages,omitempty"`
	Statuses         []StatusUpdate    `json:"statuses,omitempty"`

	// message_template_status_update field
	Event                   string `json:"event,omitempty"`
	MessageTemplateID       int64  `json:"message_template_id,omitempty"`
	MessageTemplateName     string `json:"message_template_name,omitempty"`
	MessageTemplateLanguage string `json:"message_template_language,omitempty"`
	Reason                  string `json:"reason,omitempty"`
	DisableInfo             string `json:"disable_info,omitempty"`
	OtherInfo               string `json:"other_info,omitempty"`

	// message_template_quality_update field
	PreviousQualityScore string `json:"previous_quality_score,omitempty"`
	NewQualityScore      string `json:"new_quality_score,omitempty"`

	// template_category_update field
	PreviousCategory string `json:"previous_category,omitempty"`
	NewCategory      string `json:"new_category,omitempty"`

	// account_alerts field
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`

	// account_update field
	PhoneNumber     string            `json:"phone_number,omitempty"`
	BanInfo         *BanInfo          `json:"ban_info,omitempty"`
	RestrictionInfo []RestrictionInfo `json:"restriction_info,omitempty"`

	// account_review_update field
	Decision string `json:"decision,omitempty"`

	// phone_number_name_update field
	DisplayPhoneNumber    string `json:"display_phone_number,omitempty"`
	RequestedVerifiedName string `json:"requested_verified_name,omitempty"`
	RejectionReason       string `json:"rejection_reason,omitempty"`

	// phone_number_quality_update field
	CurrentLimit string `json:"current_limit,omitempty"`

	// flows field
	FlowID       string                 `json:"flow_id,omitempty"`
	FlowName     string                 `json:"flow_name,omitempty"`
	OldStatus    string                 `json:"old_status,omitempty"`
	NewStatus    string                 `json:"new_status,omitempty"`
	ErrorType    string                 `json:"error_type,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`

	// business_capability_update field
	MaxDailyConversationPerPhone int    `json:"max_daily_conversation_per_phone,omitempty"`
	MaxPhoneNumbersPerBusiness   int    `json:"max_phone_numbers_per_business,omitempty"`
	MaxPhoneNumbersPerWaba       int    `json:"max_phone_numbers_per_waba,omitempty"`
	CoexistenceStatus            string `json:"coexistence_status,omitempty"`
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
	Order     *OrderContent    `json:"order,omitempty"` // Commerce order message
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

// NfmReplyData represents a Native Flow Message (NFM) reply from WhatsApp Flows
type NfmReplyData struct {
	Name         string `json:"name"`
	Body         string `json:"body"`
	ResponseJSON string `json:"response_json"` // JSON string with Flow data
	FlowToken    string `json:"flow_token,omitempty"`
}

// InteractiveResponse represents an interactive message response
type InteractiveResponse struct {
	Type        string            `json:"type"` // button_reply, list_reply, nfm_reply
	ButtonReply *ButtonReplyData  `json:"button_reply,omitempty"`
	ListReply   *ListReplyData    `json:"list_reply,omitempty"`
	NfmReply    *NfmReplyData     `json:"nfm_reply,omitempty"` // WhatsApp Flows response
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

// =============================================================================
// Webhook Field Types - All 13 subscription fields
// =============================================================================

// WebhookField represents the different webhook subscription fields
type WebhookField string

const (
	WebhookFieldMessages                   WebhookField = "messages"
	WebhookFieldMessageTemplateStatusUpdate WebhookField = "message_template_status_update"
	WebhookFieldMessageTemplateQualityUpdate WebhookField = "message_template_quality_update"
	WebhookFieldAccountAlerts              WebhookField = "account_alerts"
	WebhookFieldAccountUpdate              WebhookField = "account_update"
	WebhookFieldAccountReviewUpdate        WebhookField = "account_review_update"
	WebhookFieldPhoneNumberNameUpdate      WebhookField = "phone_number_name_update"
	WebhookFieldPhoneNumberQualityUpdate   WebhookField = "phone_number_quality_update"
	WebhookFieldTemplateCategoryUpdate     WebhookField = "template_category_update"
	WebhookFieldSecurity                   WebhookField = "security"
	WebhookFieldFlows                      WebhookField = "flows"
	WebhookFieldBusinessCapabilityUpdate   WebhookField = "business_capability_update"
	WebhookFieldMessageEchoes              WebhookField = "message_echoes"
)

// TemplateStatusUpdateValue represents a template status update webhook value
type TemplateStatusUpdateValue struct {
	Event                   string `json:"event"` // APPROVED, REJECTED, PENDING, PAUSED, DISABLED, IN_APPEAL, PENDING_DELETION, DELETED, REINSTATED
	MessageTemplateID       int64  `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	Reason                  string `json:"reason,omitempty"`
	DisableInfo             string `json:"disable_info,omitempty"`
	OtherInfo               string `json:"other_info,omitempty"`
}

// TemplateQualityUpdateValue represents a template quality score update
type TemplateQualityUpdateValue struct {
	MessageTemplateID       int64  `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	PreviousQualityScore    string `json:"previous_quality_score"` // GREEN, YELLOW, RED, UNKNOWN
	NewQualityScore         string `json:"new_quality_score"`
}

// AccountAlertValue represents an account alert webhook value
type AccountAlertValue struct {
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
}

// AccountUpdateValue represents an account update webhook value
type AccountUpdateValue struct {
	PhoneNumber    string `json:"phone_number,omitempty"`
	Event          string `json:"event,omitempty"`
	BanInfo        *BanInfo `json:"ban_info,omitempty"`
	RestrictionInfo []RestrictionInfo `json:"restriction_info,omitempty"`
}

// BanInfo represents ban information for an account
type BanInfo struct {
	WabaBanState string `json:"waba_ban_state,omitempty"` // SCHEDULE_FOR_DISABLE, DISABLE, REINSTATE
	WabaBanDate  string `json:"waba_ban_date,omitempty"`
}

// RestrictionInfo represents restriction information for an account
type RestrictionInfo struct {
	RestrictionType string `json:"restriction_type,omitempty"`
	Expiration      string `json:"expiration,omitempty"`
}

// AccountReviewUpdateValue represents an account review update webhook value
type AccountReviewUpdateValue struct {
	Decision string `json:"decision,omitempty"` // APPROVED, REJECTED
}

// PhoneNumberNameUpdateValue represents a phone number name update webhook value
type PhoneNumberNameUpdateValue struct {
	DisplayPhoneNumber   string `json:"display_phone_number"`
	Decision             string `json:"decision"`             // APPROVED, REJECTED
	RequestedVerifiedName string `json:"requested_verified_name"`
	RejectionReason      string `json:"rejection_reason,omitempty"`
}

// PhoneNumberQualityUpdateValue represents a phone number quality update webhook value
type PhoneNumberQualityUpdateValue struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	Event              string `json:"event"` // FLAGGED, UNFLAGGED
	CurrentLimit       string `json:"current_limit"` // TIER_50, TIER_250, TIER_1K, TIER_10K, TIER_100K, TIER_UNLIMITED
}

// TemplateCategoryUpdateValue represents a template category update webhook value
type TemplateCategoryUpdateValue struct {
	MessageTemplateID       int64  `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	PreviousCategory        string `json:"previous_category"` // AUTHENTICATION, MARKETING, UTILITY
	NewCategory             string `json:"new_category"`
}

// SecurityValue represents a security event webhook value
type SecurityValue struct {
	Event          string `json:"event,omitempty"` // TWO_STEP_VERIFICATION_DISABLED, ACCOUNT_LINKED, ACCOUNT_UNLINKED
	DisplayPhoneNumber string `json:"display_phone_number,omitempty"`
}

// FlowsValue represents a flow lifecycle event webhook value
type FlowsValue struct {
	Event    string `json:"event,omitempty"` // ENDPOINT_UNREACHABLE, ENDPOINT_TIMEOUT, ENDPOINT_ERROR, FLOW_STATUS_CHANGE
	FlowID   string `json:"flow_id,omitempty"`
	FlowName string `json:"flow_name,omitempty"`
	OldStatus string `json:"old_status,omitempty"` // DRAFT, PUBLISHED, DEPRECATED
	NewStatus string `json:"new_status,omitempty"`
	ErrorType string `json:"error_type,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// BusinessCapabilityUpdateValue represents a business capability update webhook value
type BusinessCapabilityUpdateValue struct {
	MaxDailyConversationPerPhone int    `json:"max_daily_conversation_per_phone,omitempty"`
	MaxPhoneNumbersPerBusiness   int    `json:"max_phone_numbers_per_business,omitempty"`
	MaxPhoneNumbersPerWaba       int    `json:"max_phone_numbers_per_waba,omitempty"`
	CoexistenceStatus            string `json:"coexistence_status,omitempty"`
}

// MessageEchoValue represents a message echo webhook value (messages sent from WhatsApp Business app)
type MessageEchoValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         WebhookMetadata   `json:"metadata"`
	Messages         []IncomingMessage `json:"messages,omitempty"`
	Statuses         []StatusUpdate    `json:"statuses,omitempty"`
}

// =============================================================================
// Parsed Webhook Events for each field type
// =============================================================================

// ParsedTemplateStatusEvent represents a parsed template status update
type ParsedTemplateStatusEvent struct {
	TemplateID   int64     `json:"template_id"`
	TemplateName string    `json:"template_name"`
	Language     string    `json:"language"`
	Event        string    `json:"event"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ParsedTemplateQualityEvent represents a parsed template quality update
type ParsedTemplateQualityEvent struct {
	TemplateID       int64     `json:"template_id"`
	TemplateName     string    `json:"template_name"`
	Language         string    `json:"language"`
	PreviousQuality  string    `json:"previous_quality"`
	NewQuality       string    `json:"new_quality"`
	Timestamp        time.Time `json:"timestamp"`
}

// ParsedAccountAlertEvent represents a parsed account alert
type ParsedAccountAlertEvent struct {
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ParsedPhoneQualityEvent represents a parsed phone quality update
type ParsedPhoneQualityEvent struct {
	PhoneNumber  string    `json:"phone_number"`
	Event        string    `json:"event"`
	CurrentLimit string    `json:"current_limit"`
	Timestamp    time.Time `json:"timestamp"`
}

// ParsedFlowEvent represents a parsed flow lifecycle event
type ParsedFlowEvent struct {
	FlowID       string    `json:"flow_id"`
	FlowName     string    `json:"flow_name"`
	Event        string    `json:"event"`
	OldStatus    string    `json:"old_status,omitempty"`
	NewStatus    string    `json:"new_status,omitempty"`
	ErrorType    string    `json:"error_type,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ParsedSecurityEvent represents a parsed security event
type ParsedSecurityEvent struct {
	Event       string    `json:"event"`
	PhoneNumber string    `json:"phone_number"`
	Timestamp   time.Time `json:"timestamp"`
}

// =============================================================================
// Order Message Types (Commerce)
// =============================================================================

// OrderContent represents an order message from WhatsApp Commerce
type OrderContent struct {
	CatalogID    string      `json:"catalog_id"`
	Text         string      `json:"text,omitempty"`
	ProductItems []OrderItem `json:"product_items"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductRetailerID string  `json:"product_retailer_id"`
	Quantity          int     `json:"quantity"`
	ItemPrice         float64 `json:"item_price"`
	Currency          string  `json:"currency"`
}
