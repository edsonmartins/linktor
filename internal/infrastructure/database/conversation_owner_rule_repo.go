package database

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// ConversationOwnerRuleRepository implements repository.ConversationOwnerRuleRepository
// with PostgreSQL (migration 00021). Every query filters by tenant_id.
type ConversationOwnerRuleRepository struct {
	db *PostgresDB
}

// NewConversationOwnerRuleRepository creates the repository.
func NewConversationOwnerRuleRepository(db *PostgresDB) *ConversationOwnerRuleRepository {
	return &ConversationOwnerRuleRepository{db: db}
}

const ownerRuleColumns = `tenant_id, channel_id, contact_identifier, consumer, conversa_ref, desde, expira_em, updated_at`

// Upsert writes the rule in one statement: the WHERE of ON CONFLICT keeps a
// newer declaration of another conversation in place (see the interface).
func (r *ConversationOwnerRuleRepository) Upsert(ctx context.Context, rule *entity.ConversationOwnerRule) (bool, *entity.ConversationOwnerRule, error) {
	query := `
		INSERT INTO conversation_owner_rules (` + ownerRuleColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (tenant_id, channel_id, contact_identifier) DO UPDATE
		   SET consumer = EXCLUDED.consumer,
		       conversa_ref = EXCLUDED.conversa_ref,
		       desde = EXCLUDED.desde,
		       expira_em = EXCLUDED.expira_em,
		       updated_at = NOW()
		 WHERE conversation_owner_rules.conversa_ref = EXCLUDED.conversa_ref
		    OR conversation_owner_rules.desde <= EXCLUDED.desde
		RETURNING ` + ownerRuleColumns
	row := r.db.Pool.QueryRow(ctx, query, rule.TenantID, rule.ChannelID, rule.ContactIdentifier, rule.Consumer,
		rule.ConversaRef, rule.Desde, rule.ExpiraEm)
	written, err := scanOwnerRule(row)
	if err == nil {
		return true, written, nil
	}
	if !stderrors.Is(err, pgx.ErrNoRows) {
		return false, nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to write conversation owner rule")
	}
	// The WHERE refused the update: a newer rule of another conversation stays.
	current, ferr := r.Find(ctx, rule.TenantID, rule.ChannelID, rule.ContactIdentifier)
	if ferr != nil {
		return false, nil, ferr
	}
	return false, current, nil
}

// Find returns the stored rule or nil.
func (r *ConversationOwnerRuleRepository) Find(ctx context.Context, tenantID, channelID, contactIdentifier string) (*entity.ConversationOwnerRule, error) {
	row := r.db.Pool.QueryRow(ctx, `SELECT `+ownerRuleColumns+` FROM conversation_owner_rules
		WHERE tenant_id = $1 AND channel_id = $2 AND contact_identifier = $3`, tenantID, channelID, contactIdentifier)
	rule, err := scanOwnerRule(row)
	if stderrors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to read conversation owner rule")
	}
	return rule, nil
}

// Release deletes the rule only while it still belongs to conversaRef.
func (r *ConversationOwnerRuleRepository) Release(ctx context.Context, tenantID, channelID, contactIdentifier, conversaRef string) (bool, error) {
	res, err := r.db.Pool.Exec(ctx, `DELETE FROM conversation_owner_rules
		WHERE tenant_id = $1 AND channel_id = $2 AND contact_identifier = $3 AND conversa_ref = $4`,
		tenantID, channelID, contactIdentifier, conversaRef)
	if err != nil {
		return false, errors.Wrap(err, errors.ErrCodeInternal, "failed to release conversation owner rule")
	}
	return res.RowsAffected() > 0, nil
}

func scanOwnerRule(row pgx.Row) (*entity.ConversationOwnerRule, error) {
	var r entity.ConversationOwnerRule
	var expira *time.Time
	if err := row.Scan(&r.TenantID, &r.ChannelID, &r.ContactIdentifier, &r.Consumer, &r.ConversaRef, &r.Desde, &expira,
		&r.UpdatedAt); err != nil {
		return nil, err
	}
	r.ExpiraEm = expira
	return &r, nil
}
