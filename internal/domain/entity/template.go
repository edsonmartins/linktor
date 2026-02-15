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

// Template represents a WhatsApp message template
type Template struct {
	ID                string           `json:"id"`
	TenantID          string           `json:"tenant_id"`
	ChannelID         string           `json:"channel_id"`
	ExternalID        string           `json:"external_id"`         // Meta's template ID
	Name              string           `json:"name"`
	Language          string           `json:"language"`
	Category          TemplateCategory `json:"category"`
	Status            TemplateStatus   `json:"status"`
	QualityScore      TemplateQuality  `json:"quality_score"`
	Components        []TemplateComponent `json:"components"`
	RejectionReason   string           `json:"rejection_reason,omitempty"`
	LastSyncedAt      *time.Time       `json:"last_synced_at,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// TemplateComponent represents a component of a template (header, body, footer, buttons)
type TemplateComponent struct {
	Type       string                 `json:"type"` // HEADER, BODY, FOOTER, BUTTONS
	Format     string                 `json:"format,omitempty"` // TEXT, IMAGE, VIDEO, DOCUMENT, LOCATION
	Text       string                 `json:"text,omitempty"`
	Example    *TemplateExample       `json:"example,omitempty"`
	Buttons    []TemplateButton       `json:"buttons,omitempty"`
	Parameters []TemplateParameter    `json:"parameters,omitempty"`
}

// TemplateExample represents example values for a template
type TemplateExample struct {
	HeaderText   []string   `json:"header_text,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
	HeaderHandle []string   `json:"header_handle,omitempty"` // For media headers
}

// TemplateButton represents a button in a template
type TemplateButton struct {
	Type        string `json:"type"` // QUICK_REPLY, URL, PHONE_NUMBER, COPY_CODE, OTP, FLOW
	Text        string `json:"text"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	FlowID      string `json:"flow_id,omitempty"`
	FlowAction  string `json:"flow_action,omitempty"` // navigate, data_exchange
	Example     string `json:"example,omitempty"`
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
