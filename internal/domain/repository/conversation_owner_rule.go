package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// ConversationOwnerRuleRepository persists the owner of the open conversation
// per (tenant, channel, contact). Every operation takes the tenant explicitly.
type ConversationOwnerRuleRepository interface {
	// Upsert writes the rule. A declaration for the same conversa_ref always
	// applies; one for another conversation applies only if its desde is not
	// older than the rule in place, so a late declaration of an old conversation
	// cannot overwrite a newer one. Returns whether it applied and the rule now
	// in place.
	Upsert(ctx context.Context, rule *entity.ConversationOwnerRule) (applied bool, current *entity.ConversationOwnerRule, err error)
	// Find returns the stored rule, or nil when there is none (expired rules are
	// returned as stored; callers decide with VigenteEm).
	Find(ctx context.Context, tenantID, channelID, contactIdentifier string) (*entity.ConversationOwnerRule, error)
	// Release deletes the rule only if it still belongs to conversaRef. Returns
	// whether a rule was deleted.
	Release(ctx context.Context, tenantID, channelID, contactIdentifier, conversaRef string) (bool, error)
}
