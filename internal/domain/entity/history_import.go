package entity

import (
	"time"
)

// HistoryImportStatus represents the status of a chat history import
type HistoryImportStatus string

const (
	HistoryImportStatusPending    HistoryImportStatus = "pending"
	HistoryImportStatusInProgress HistoryImportStatus = "in_progress"
	HistoryImportStatusCompleted  HistoryImportStatus = "completed"
	HistoryImportStatusFailed     HistoryImportStatus = "failed"
	HistoryImportStatusCancelled  HistoryImportStatus = "cancelled"
)

// HistoryImport represents a chat history import job for WhatsApp Coexistence
type HistoryImport struct {
	ID        string              `json:"id"`
	ChannelID string              `json:"channel_id"`
	TenantID  string              `json:"tenant_id"`
	Status    HistoryImportStatus `json:"status"`

	// Progress tracking
	TotalConversations    int `json:"total_conversations"`
	ImportedConversations int `json:"imported_conversations"`
	TotalMessages         int `json:"total_messages"`
	ImportedMessages      int `json:"imported_messages"`
	TotalContacts         int `json:"total_contacts"`
	ImportedContacts      int `json:"imported_contacts"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error handling
	ErrorMessage string                 `json:"error_message,omitempty"`
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`

	// Import configuration
	ImportSince *time.Time `json:"import_since,omitempty"` // How far back to import (max 6 months)

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewHistoryImport creates a new history import job
func NewHistoryImport(channelID, tenantID string) *HistoryImport {
	now := time.Now()
	sixMonthsAgo := now.AddDate(0, -6, 0) // Default to 6 months back

	return &HistoryImport{
		ChannelID:   channelID,
		TenantID:    tenantID,
		Status:      HistoryImportStatusPending,
		ImportSince: &sixMonthsAgo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Start marks the import as started
func (h *HistoryImport) Start() {
	now := time.Now()
	h.Status = HistoryImportStatusInProgress
	h.StartedAt = &now
	h.UpdatedAt = now
}

// Complete marks the import as completed
func (h *HistoryImport) Complete() {
	now := time.Now()
	h.Status = HistoryImportStatusCompleted
	h.CompletedAt = &now
	h.UpdatedAt = now
}

// Fail marks the import as failed
func (h *HistoryImport) Fail(errorMessage string, details map[string]interface{}) {
	now := time.Now()
	h.Status = HistoryImportStatusFailed
	h.ErrorMessage = errorMessage
	h.ErrorDetails = details
	h.CompletedAt = &now
	h.UpdatedAt = now
}

// Cancel marks the import as cancelled
func (h *HistoryImport) Cancel() {
	now := time.Now()
	h.Status = HistoryImportStatusCancelled
	h.CompletedAt = &now
	h.UpdatedAt = now
}

// UpdateProgress updates the import progress
func (h *HistoryImport) UpdateProgress(conversations, messages, contacts int) {
	h.ImportedConversations = conversations
	h.ImportedMessages = messages
	h.ImportedContacts = contacts
	h.UpdatedAt = time.Now()
}

// SetTotals sets the total counts for the import
func (h *HistoryImport) SetTotals(conversations, messages, contacts int) {
	h.TotalConversations = conversations
	h.TotalMessages = messages
	h.TotalContacts = contacts
	h.UpdatedAt = time.Now()
}

// Progress returns the progress percentage (0-100)
func (h *HistoryImport) Progress() float64 {
	if h.TotalConversations == 0 && h.TotalMessages == 0 {
		return 0
	}

	// Weight: 30% conversations, 70% messages
	convProgress := 0.0
	if h.TotalConversations > 0 {
		convProgress = float64(h.ImportedConversations) / float64(h.TotalConversations) * 30
	}

	msgProgress := 0.0
	if h.TotalMessages > 0 {
		msgProgress = float64(h.ImportedMessages) / float64(h.TotalMessages) * 70
	}

	return convProgress + msgProgress
}

// IsComplete returns true if the import is complete (success, failed, or cancelled)
func (h *HistoryImport) IsComplete() bool {
	return h.Status == HistoryImportStatusCompleted ||
		h.Status == HistoryImportStatusFailed ||
		h.Status == HistoryImportStatusCancelled
}

// IsRunning returns true if the import is currently running
func (h *HistoryImport) IsRunning() bool {
	return h.Status == HistoryImportStatusInProgress
}
