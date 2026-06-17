package repository

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// CampaignRepository persists campaigns and their recipients.
type CampaignRepository interface {
	Create(ctx context.Context, c *entity.Campaign) error
	FindByID(ctx context.Context, id string) (*entity.Campaign, error)
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.Campaign, int64, error)
	Update(ctx context.Context, c *entity.Campaign) error
	Delete(ctx context.Context, id string) error

	// AddRecipients bulk-inserts recipients for a campaign.
	AddRecipients(ctx context.Context, recipients []*entity.CampaignRecipient) error
	// FindPendingRecipients returns recipients in pending status for a campaign.
	FindPendingRecipients(ctx context.Context, campaignID string, limit int) ([]*entity.CampaignRecipient, error)
	// ListRecipients returns recipients for a campaign with pagination, optionally
	// filtered by status ("" = all).
	ListRecipients(ctx context.Context, campaignID, status string, params *ListParams) ([]*entity.CampaignRecipient, int64, error)
	// UpdateRecipientStatus sets a recipient's status/messageID/error and bumps attempts.
	UpdateRecipientStatus(ctx context.Context, recipientID string, status entity.RecipientStatus, messageID, errReason string) error
	// MarkRecipientQueued moves a recipient to 'queued' without bumping attempts
	// (enqueue is not a delivery attempt).
	MarkRecipientQueued(ctx context.Context, recipientID string) error
	// UpdateRecipientStatusByMessageID updates a recipient by its provider message
	// ID, used to apply delivery/read status from webhooks. No-op if unmatched.
	UpdateRecipientStatusByMessageID(ctx context.Context, messageID string, status entity.RecipientStatus) error
	// ResetFailedRecipients moves failed recipients back to pending for retry; returns count.
	ResetFailedRecipients(ctx context.Context, campaignID string) (int64, error)
	// SweepStaleQueued fails recipients stuck in 'queued' longer than olderThan
	// (the delivery worker never confirmed them — exhausted retries / DLQ / down).
	// Returns the distinct campaign IDs that were touched.
	SweepStaleQueued(ctx context.Context, olderThan time.Duration) ([]string, error)
	// RecountStatuses recomputes the campaign counters from its recipients.
	RecountStatuses(ctx context.Context, campaignID string) error
}
