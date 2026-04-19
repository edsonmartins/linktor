package entity

import (
	"time"
)

// TemplateStatus represents the approval status of a template
type TemplateStatus string

const (
	TemplateStatusPending          TemplateStatus = "PENDING"
	TemplateStatusApproved         TemplateStatus = "APPROVED"
	TemplateStatusRejected         TemplateStatus = "REJECTED"
	TemplateStatusPaused           TemplateStatus = "PAUSED"
	TemplateStatusDisabled         TemplateStatus = "DISABLED"
	TemplateStatusInAppeal         TemplateStatus = "IN_APPEAL"
	TemplateStatusPendingDeletion  TemplateStatus = "PENDING_DELETION"
	TemplateStatusDeleted          TemplateStatus = "DELETED"
	TemplateStatusReinstated       TemplateStatus = "REINSTATED"
	// Meta returns LIMIT_EXCEEDED for templates a WABA can no longer send
	// because it hit its category-specific send limit, and ARCHIVED for
	// templates Meta has retired from the library but that still exist on
	// the WABA for reference. Missing these meant the mapper silently fell
	// through and we'd persist the zero value instead of the real status.
	TemplateStatusLimitExceeded    TemplateStatus = "LIMIT_EXCEEDED"
	TemplateStatusArchived         TemplateStatus = "ARCHIVED"
)

// TemplateCategory represents the category of a template
type TemplateCategory string

const (
	TemplateCategoryAuthentication TemplateCategory = "AUTHENTICATION"
	TemplateCategoryMarketing      TemplateCategory = "MARKETING"
	TemplateCategoryUtility        TemplateCategory = "UTILITY"
)

// TemplateQuality represents the quality score of a template
type TemplateQuality string

const (
	TemplateQualityGreen   TemplateQuality = "GREEN"
	TemplateQualityYellow  TemplateQuality = "YELLOW"
	TemplateQualityRed     TemplateQuality = "RED"
	TemplateQualityUnknown TemplateQuality = "UNKNOWN"
)

// TemplateParameterFormat declares whether a template's placeholders are
// referenced by position ({{1}}, {{2}}) or by name ({{customer_name}}).
// Meta defaults to POSITIONAL when the field is omitted.
type TemplateParameterFormat string

const (
	TemplateParameterFormatPositional TemplateParameterFormat = "POSITIONAL"
	TemplateParameterFormatNamed      TemplateParameterFormat = "NAMED"
)

// Template represents a WhatsApp message template
type Template struct {
	ID                string           `json:"id"`
	TenantID          string           `json:"tenant_id"`
	ChannelID         string           `json:"channel_id"`
	ExternalID        string           `json:"external_id"`         // Meta's template ID (aka hsm_id)
	Name              string           `json:"name"`
	Language          string           `json:"language"`
	Category          TemplateCategory `json:"category"`
	// SubCategory refines some categories (e.g. UTILITY → ORDER_DETAILS,
	// ORDER_STATUS, RICH_ORDER_STATUS). Optional; Meta treats absence as
	// the generic category.
	SubCategory       string           `json:"sub_category,omitempty"`
	// ParameterFormat controls whether placeholders are positional ({{1}})
	// or named ({{first_name}}). Empty means positional per Meta's default.
	ParameterFormat   TemplateParameterFormat `json:"parameter_format,omitempty"`
	// MessageSendTTLSeconds bounds how long Meta will retry delivery of a
	// message that uses this template. Zero means Meta's default.
	MessageSendTTLSeconds int `json:"message_send_ttl_seconds,omitempty"`
	// AllowCategoryChange lets Meta auto-move a template to a different
	// creation category based on content during review. Useful for
	// marketing vs utility ambiguity.
	AllowCategoryChange bool             `json:"allow_category_change,omitempty"`
	Status            TemplateStatus   `json:"status"`
	QualityScore      TemplateQuality  `json:"quality_score"`
	Components        []TemplateComponent `json:"components"`
	RejectionReason   string           `json:"rejection_reason,omitempty"`
	LastSyncedAt      *time.Time       `json:"last_synced_at,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// TemplateComponent represents a component of a template at creation time.
// Allowed `Type` values:
//   - HEADER / BODY / FOOTER — plain text or media components
//   - BUTTONS — button row (QUICK_REPLY / URL / PHONE_NUMBER / OTP / FLOW)
//   - CAROUSEL — card-based carousel (fill Cards)
//   - LIMITED_TIME_OFFER — time-bound promo banner (fill LimitedTimeOffer)
type TemplateComponent struct {
	Type       string                 `json:"type"`
	Format     string                 `json:"format,omitempty"` // TEXT, IMAGE, VIDEO, DOCUMENT, LOCATION
	Text       string                 `json:"text,omitempty"`
	Example    *TemplateExample       `json:"example,omitempty"`
	Buttons    []TemplateButton       `json:"buttons,omitempty"`
	Parameters []TemplateParameter    `json:"parameters,omitempty"`
	// Cards applies to CAROUSEL components. Each card carries its own
	// header/body/buttons sub-components. Meta caps carousel at 10 cards.
	Cards []TemplateCarouselCard `json:"cards,omitempty"`
	// LimitedTimeOffer applies to the LIMITED_TIME_OFFER component.
	LimitedTimeOffer *TemplateLimitedTimeOffer `json:"limited_time_offer,omitempty"`
}

// TemplateCarouselCard is one card inside a carousel. A card must carry at
// least a header and body, and may carry a button row. Cards share the
// overall template's body; their own body can further vary per card.
type TemplateCarouselCard struct {
	Components []TemplateComponent `json:"components"`
}

// TemplateLimitedTimeOffer declares a countdown promo banner on a
// utility/marketing template. `ExpirationTimeMS` is an absolute unix
// millisecond timestamp; `HasExpiration=false` disables the countdown.
type TemplateLimitedTimeOffer struct {
	Text             string `json:"text,omitempty"`
	HasExpiration    bool   `json:"has_expiration"`
	ExpirationTimeMS int64  `json:"expiration_time_ms,omitempty"`
}

// TemplateExample represents example values for a template
type TemplateExample struct {
	HeaderText   []string   `json:"header_text,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
	HeaderHandle []string   `json:"header_handle,omitempty"` // For media headers
}

// TemplateButton represents a button in a template at creation time.
// Meta has different shapes per button type — the struct carries the
// superset and only the fields relevant to `Type` are serialized.
type TemplateButton struct {
	Type        string `json:"type"` // QUICK_REPLY, URL, PHONE_NUMBER, COPY_CODE, OTP, FLOW
	Text        string `json:"text"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	FlowID      string `json:"flow_id,omitempty"`
	FlowAction  string `json:"flow_action,omitempty"` // navigate, data_exchange
	Example     string `json:"example,omitempty"`
	// OTP-specific fields. OTPType selects the autofill experience:
	//   - COPY_CODE → user manually taps "copy" (fallback for all apps)
	//   - ONE_TAP   → same as copy but Meta can autofill if supported_apps matches
	//   - ZERO_TAP  → fully automatic autofill; requires zero_tap_terms_accepted
	OTPType              string            `json:"otp_type,omitempty"`
	AutofillText         string            `json:"autofill_text,omitempty"`
	PackageName          string            `json:"package_name,omitempty"`
	SignatureHash        string            `json:"signature_hash,omitempty"`
	SupportedApps        []TemplateOTPApp  `json:"supported_apps,omitempty"`
	ZeroTapTermsAccepted bool              `json:"zero_tap_terms_accepted,omitempty"`
}

// TemplateOTPApp identifies an Android app that can autofill an OTP code
// via Meta's WhatsApp OTP API. Required for ONE_TAP and ZERO_TAP.
type TemplateOTPApp struct {
	PackageName   string `json:"package_name"`
	SignatureHash string `json:"signature_hash"`
}

// TemplateParameter represents a parameter in a template message
type TemplateParameter struct {
	Type     string                   `json:"type"` // text, currency, date_time, image, video, document
	Text     string                   `json:"text,omitempty"`
	Currency *TemplateCurrencyParam   `json:"currency,omitempty"`
	DateTime *TemplateDateTimeParam   `json:"date_time,omitempty"`
	Image    *TemplateMediaParam      `json:"image,omitempty"`
	Video    *TemplateMediaParam      `json:"video,omitempty"`
	Document *TemplateDocumentParam   `json:"document,omitempty"`
}

// TemplateCurrencyParam represents a currency parameter
type TemplateCurrencyParam struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int64  `json:"amount_1000"` // Amount in thousandths
}

// TemplateDateTimeParam represents a date/time parameter
type TemplateDateTimeParam struct {
	FallbackValue string `json:"fallback_value"`
}

// TemplateMediaParam represents a media parameter (image/video)
type TemplateMediaParam struct {
	ID   string `json:"id,omitempty"`   // Media ID
	Link string `json:"link,omitempty"` // Media URL
}

// TemplateDocumentParam represents a document parameter
type TemplateDocumentParam struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// TemplateSendRequest represents a request to send a template message
type TemplateSendRequest struct {
	TemplateName string                      `json:"template_name"`
	Language     string                      `json:"language"`
	Components   []TemplateComponentParams   `json:"components,omitempty"`
}

// TemplateComponentParams represents parameters for a template component when sending
type TemplateComponentParams struct {
	Type       string              `json:"type"` // header, body, button
	Parameters []TemplateParameter `json:"parameters,omitempty"`
	SubType    string              `json:"sub_type,omitempty"` // For buttons: quick_reply, url
	Index      int                 `json:"index,omitempty"`    // Button index (0-based)
}

// IsApproved returns true if the template is approved and can be used
func (t *Template) IsApproved() bool {
	return t.Status == TemplateStatusApproved
}

// NeedsSync returns true if the template needs to be synced with Meta
func (t *Template) NeedsSync(interval time.Duration) bool {
	if t.LastSyncedAt == nil {
		return true
	}
	return time.Since(*t.LastSyncedAt) > interval
}

// UpdateQuality updates the quality score and timestamp
func (t *Template) UpdateQuality(quality TemplateQuality) {
	t.QualityScore = quality
	t.UpdatedAt = time.Now()
}

// UpdateStatus updates the status and timestamp
func (t *Template) UpdateStatus(status TemplateStatus, reason string) {
	t.Status = status
	t.RejectionReason = reason
	t.UpdatedAt = time.Now()
}
