package entity

import "time"

// CannedResponse is a reusable quick reply an agent can insert by typing its
// shortcut (e.g. "/greeting"). Content may contain {{placeholders}} that the
// frontend or agent fills in before sending.
type CannedResponse struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Shortcut   string    `json:"shortcut"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Tags       []string  `json:"tags"`
	UsageCount int       `json:"usage_count"`
	CreatedBy  string    `json:"created_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewCannedResponse creates a canned response with timestamps set.
func NewCannedResponse(tenantID, shortcut, title, content string) *CannedResponse {
	now := time.Now()
	return &CannedResponse{
		TenantID:  tenantID,
		Shortcut:  shortcut,
		Title:     title,
		Content:   content,
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
