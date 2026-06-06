package entity

import "time"

// CampaignStatus tracks the lifecycle of a bulk campaign.
type CampaignStatus string

const (
	CampaignStatusDraft      CampaignStatus = "draft"
	CampaignStatusScheduled  CampaignStatus = "scheduled"
	CampaignStatusProcessing CampaignStatus = "processing"
	CampaignStatusCompleted  CampaignStatus = "completed"
	CampaignStatusFailed     CampaignStatus = "failed"
	CampaignStatusCancelled  CampaignStatus = "cancelled"
)

// RecipientStatus tracks per-recipient delivery state.
type RecipientStatus string

const (
	RecipientPending   RecipientStatus = "pending"
	RecipientSent      RecipientStatus = "sent"
	RecipientDelivered RecipientStatus = "delivered"
	RecipientRead      RecipientStatus = "read"
	RecipientFailed    RecipientStatus = "failed"
)

// Campaign is a bulk template send to many recipients.
type Campaign struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id"`
	ChannelID        string            `json:"channel_id"`
	Name             string            `json:"name"`
	TemplateName     string            `json:"template_name"`
	TemplateLanguage string            `json:"template_language"`
	TemplateParams   map[string]string `json:"template_params,omitempty"`
	Status           CampaignStatus    `json:"status"`
	TotalRecipients  int               `json:"total_recipients"`
	SentCount        int               `json:"sent_count"`
	DeliveredCount   int               `json:"delivered_count"`
	ReadCount        int               `json:"read_count"`
	FailedCount      int               `json:"failed_count"`
	ScheduledAt      *time.Time        `json:"scheduled_at,omitempty"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CreatedBy        string            `json:"created_by,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// NewCampaign creates a draft campaign.
func NewCampaign(tenantID, channelID, name, templateName, language string) *Campaign {
	now := time.Now()
	return &Campaign{
		TenantID:         tenantID,
		ChannelID:        channelID,
		Name:             name,
		TemplateName:     templateName,
		TemplateLanguage: language,
		TemplateParams:   map[string]string{},
		Status:           CampaignStatusDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// CampaignRecipient is one target of a campaign.
type CampaignRecipient struct {
	ID          string          `json:"id"`
	CampaignID  string          `json:"campaign_id"`
	ContactID   string          `json:"contact_id,omitempty"`
	Phone       string          `json:"phone"`
	Params      []string        `json:"params,omitempty"`
	Status      RecipientStatus `json:"status"`
	MessageID   string          `json:"message_id,omitempty"`
	ErrorReason string          `json:"error_reason,omitempty"`
	Attempts    int             `json:"attempts"`
	SentAt      *time.Time      `json:"sent_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
