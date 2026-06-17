package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// CampaignRepository implements repository.CampaignRepository.
type CampaignRepository struct {
	db *PostgresDB
}

// NewCampaignRepository creates a new PostgreSQL campaign repository.
func NewCampaignRepository(db *PostgresDB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

const campaignColumns = `id, tenant_id, channel_id, name, template_name, template_language, template_params,
	status, total_recipients, sent_count, delivered_count, read_count, failed_count,
	scheduled_at, started_at, completed_at, created_by, created_at, updated_at`

func (r *CampaignRepository) Create(ctx context.Context, c *entity.Campaign) error {
	params, _ := json.Marshal(c.TemplateParams)
	query := `INSERT INTO campaigns (` + campaignColumns + `)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`
	_, err := r.db.Pool.Exec(ctx, query,
		c.ID, c.TenantID, c.ChannelID, c.Name, c.TemplateName, c.TemplateLanguage, params,
		string(c.Status), c.TotalRecipients, c.SentCount, c.DeliveredCount, c.ReadCount, c.FailedCount,
		c.ScheduledAt, c.StartedAt, c.CompletedAt, nullString(c.CreatedBy), c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create campaign")
	}
	return nil
}

func (r *CampaignRepository) FindByID(ctx context.Context, id string) (*entity.Campaign, error) {
	c, err := scanCampaign(r.db.Pool.QueryRow(ctx, `SELECT `+campaignColumns+` FROM campaigns WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "campaign not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find campaign")
	}
	return c, nil
}

func (r *CampaignRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Campaign, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}
	var total int64
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaigns WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count campaigns")
	}
	rows, err := r.db.Pool.Query(ctx,
		`SELECT `+campaignColumns+` FROM campaigns WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query campaigns")
	}
	defer rows.Close()

	var list []*entity.Campaign
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, nil
}

func (r *CampaignRepository) Update(ctx context.Context, c *entity.Campaign) error {
	params, _ := json.Marshal(c.TemplateParams)
	query := `UPDATE campaigns SET
		name=$1, template_name=$2, template_language=$3, template_params=$4, status=$5,
		total_recipients=$6, sent_count=$7, delivered_count=$8, read_count=$9, failed_count=$10,
		scheduled_at=$11, started_at=$12, completed_at=$13 WHERE id=$14`
	result, err := r.db.Pool.Exec(ctx, query,
		c.Name, c.TemplateName, c.TemplateLanguage, params, string(c.Status),
		c.TotalRecipients, c.SentCount, c.DeliveredCount, c.ReadCount, c.FailedCount,
		c.ScheduledAt, c.StartedAt, c.CompletedAt, c.ID,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update campaign")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "campaign not found")
	}
	return nil
}

func (r *CampaignRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, `DELETE FROM campaigns WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete campaign")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "campaign not found")
	}
	return nil
}

func (r *CampaignRepository) AddRecipients(ctx context.Context, recipients []*entity.CampaignRecipient) error {
	if len(recipients) == 0 {
		return nil
	}
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to begin tx")
	}
	defer tx.Rollback(ctx)

	for _, rec := range recipients {
		params, _ := json.Marshal(rec.Params)
		_, err := tx.Exec(ctx,
			`INSERT INTO campaign_recipients (id, campaign_id, contact_id, phone, params, status, attempts, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			rec.ID, rec.CampaignID, nullString(rec.ContactID), rec.Phone, params,
			string(rec.Status), rec.Attempts, rec.CreatedAt, rec.UpdatedAt,
		)
		if err != nil {
			return errors.Wrap(err, errors.ErrCodeInternal, "failed to insert recipient")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to commit recipients")
	}
	return nil
}

func (r *CampaignRepository) FindPendingRecipients(ctx context.Context, campaignID string, limit int) ([]*entity.CampaignRecipient, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, campaign_id, contact_id, phone, params, status, message_id, error_reason, attempts, sent_at, created_at, updated_at
		 FROM campaign_recipients WHERE campaign_id = $1 AND status = 'pending' ORDER BY created_at ASC LIMIT $2`,
		campaignID, limit)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query pending recipients")
	}
	defer rows.Close()
	return scanRecipients(rows)
}

func (r *CampaignRepository) ListRecipients(ctx context.Context, campaignID, status string, params *repository.ListParams) ([]*entity.CampaignRecipient, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}

	where := "WHERE campaign_id = $1"
	args := []interface{}{campaignID}
	if status != "" {
		where += " AND status = $2"
		args = append(args, status)
	}

	var total int64
	if err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM campaign_recipients "+where, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count recipients")
	}

	limitIdx := len(args) + 1
	query := "SELECT id, campaign_id, contact_id, phone, params, status, message_id, error_reason, attempts, sent_at, created_at, updated_at" +
		" FROM campaign_recipients " + where + " ORDER BY created_at ASC LIMIT $" + itoa(limitIdx) + " OFFSET $" + itoa(limitIdx+1)
	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query recipients")
	}
	defer rows.Close()
	list, err := scanRecipients(rows)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *CampaignRepository) UpdateRecipientStatus(ctx context.Context, recipientID string, status entity.RecipientStatus, messageID, errReason string) error {
	query := `UPDATE campaign_recipients SET status=$1, message_id=$2, error_reason=$3, attempts=attempts+1,
		sent_at = CASE WHEN $1 = 'sent' THEN NOW() ELSE sent_at END WHERE id=$4`
	_, err := r.db.Pool.Exec(ctx, query, string(status), nullString(messageID), nullString(errReason), recipientID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update recipient status")
	}
	return nil
}

func (r *CampaignRepository) MarkRecipientQueued(ctx context.Context, recipientID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE campaign_recipients SET status = 'queued' WHERE id = $1 AND status = 'pending'`,
		recipientID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to mark recipient queued")
	}
	return nil
}

func (r *CampaignRepository) UpdateRecipientStatusByMessageID(ctx context.Context, messageID string, status entity.RecipientStatus) error {
	if messageID == "" {
		return nil
	}
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE campaign_recipients SET status = $1 WHERE message_id = $2`,
		string(status), messageID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update recipient by message id")
	}
	return nil
}

func (r *CampaignRepository) ResetFailedRecipients(ctx context.Context, campaignID string) (int64, error) {
	result, err := r.db.Pool.Exec(ctx,
		`UPDATE campaign_recipients SET status='pending', error_reason=NULL WHERE campaign_id=$1 AND status='failed'`,
		campaignID)
	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to reset failed recipients")
	}
	return result.RowsAffected(), nil
}

func (r *CampaignRepository) SweepStaleQueued(ctx context.Context, olderThan time.Duration) ([]string, error) {
	seconds := int(olderThan.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	query := `
		UPDATE campaign_recipients
		SET status = 'failed', error_reason = 'delivery not confirmed (timed out / retries exhausted)'
		WHERE status = 'queued'
		  AND updated_at < NOW() - ($1 * INTERVAL '1 second')
		RETURNING campaign_id`
	rows, err := r.db.Pool.Query(ctx, query, seconds)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to sweep stale queued recipients")
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var campaignIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan campaign id")
		}
		if !seen[id] {
			seen[id] = true
			campaignIDs = append(campaignIDs, id)
		}
	}
	return campaignIDs, nil
}

func (r *CampaignRepository) RecountStatuses(ctx context.Context, campaignID string) error {
	query := `
		UPDATE campaigns c SET
			sent_count = s.sent, delivered_count = s.delivered, read_count = s.read, failed_count = s.failed
		FROM (
			SELECT
				COUNT(*) FILTER (WHERE status IN ('sent','delivered','read')) AS sent,
				COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
				COUNT(*) FILTER (WHERE status = 'read') AS read,
				COUNT(*) FILTER (WHERE status = 'failed') AS failed
			FROM campaign_recipients WHERE campaign_id = $1
		) s
		WHERE c.id = $1`
	_, err := r.db.Pool.Exec(ctx, query, campaignID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to recount statuses")
	}
	return nil
}

func scanCampaign(row pgx.Row) (*entity.Campaign, error) {
	var c entity.Campaign
	var status string
	var params []byte
	var createdBy *string
	err := row.Scan(
		&c.ID, &c.TenantID, &c.ChannelID, &c.Name, &c.TemplateName, &c.TemplateLanguage, &params,
		&status, &c.TotalRecipients, &c.SentCount, &c.DeliveredCount, &c.ReadCount, &c.FailedCount,
		&c.ScheduledAt, &c.StartedAt, &c.CompletedAt, &createdBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Status = entity.CampaignStatus(status)
	c.CreatedBy = derefString(createdBy)
	if len(params) > 0 {
		_ = json.Unmarshal(params, &c.TemplateParams)
	}
	if c.TemplateParams == nil {
		c.TemplateParams = map[string]string{}
	}
	return &c, nil
}

func scanRecipients(rows pgx.Rows) ([]*entity.CampaignRecipient, error) {
	var list []*entity.CampaignRecipient
	for rows.Next() {
		var rec entity.CampaignRecipient
		var status string
		var params []byte
		var contactID, messageID, errReason *string
		err := rows.Scan(
			&rec.ID, &rec.CampaignID, &contactID, &rec.Phone, &params, &status,
			&messageID, &errReason, &rec.Attempts, &rec.SentAt, &rec.CreatedAt, &rec.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan recipient")
		}
		rec.Status = entity.RecipientStatus(status)
		rec.ContactID = derefString(contactID)
		rec.MessageID = derefString(messageID)
		rec.ErrorReason = derefString(errReason)
		if len(params) > 0 {
			_ = json.Unmarshal(params, &rec.Params)
		}
		list = append(list, &rec)
	}
	return list, nil
}
