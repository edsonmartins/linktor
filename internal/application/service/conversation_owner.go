package service

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ConversationOwnerService declares, reads and releases the owner of the open
// conversation with a contact on a channel (P5 of the CRM ↔ Sales Copilot
// agreement, VCR-36). The rule is tenant-scoped: a channel of another tenant
// is reported as not found, never revealed.
type ConversationOwnerService struct {
	repo        repository.ConversationOwnerRuleRepository
	channelRepo repository.ChannelRepository
	now         func() time.Time
}

// NewConversationOwnerService creates the service.
func NewConversationOwnerService(repo repository.ConversationOwnerRuleRepository, channelRepo repository.ChannelRepository) *ConversationOwnerService {
	return &ConversationOwnerService{repo: repo, channelRepo: channelRepo, now: time.Now}
}

// DeclareOwnerInput is the PUT payload, already bound to the tenant and path.
type DeclareOwnerInput struct {
	TenantID    string
	ChannelID   string
	Identifier  string
	Consumer    string
	ConversaRef string
	Desde       time.Time
	ExpiraEm    *time.Time
}

// Declare writes the rule (idempotent). applied is false when a newer rule of
// another conversation is in place; current is the rule now in force.
func (s *ConversationOwnerService) Declare(ctx context.Context, in *DeclareOwnerInput) (applied bool, current *entity.ConversationOwnerRule, err error) {
	if !entity.ValidConsumerName(in.Consumer) {
		return false, nil, errors.New(errors.ErrCodeValidation, "consumidor must be a consumer name (lowercase letters, digits, - or _)")
	}
	if in.ConversaRef == "" {
		return false, nil, errors.New(errors.ErrCodeValidation, "conversa_ref is required")
	}
	if in.Desde.IsZero() {
		return false, nil, errors.New(errors.ErrCodeValidation, "desde is required (ISO-8601)")
	}
	if in.ExpiraEm != nil && !in.ExpiraEm.After(in.Desde) {
		return false, nil, errors.New(errors.ErrCodeValidation, "expira_em must be after desde")
	}
	identifier, err := s.chave(ctx, in.TenantID, in.ChannelID, in.Identifier)
	if err != nil {
		return false, nil, err
	}
	return s.repo.Upsert(ctx, &entity.ConversationOwnerRule{
		TenantID:          in.TenantID,
		ChannelID:         in.ChannelID,
		ContactIdentifier: identifier,
		Consumer:          in.Consumer,
		ConversaRef:       in.ConversaRef,
		Desde:             in.Desde.UTC(),
		ExpiraEm:          utcOrNil(in.ExpiraEm),
	})
}

// Get returns the rule in force, or not found when there is none or it expired.
func (s *ConversationOwnerService) Get(ctx context.Context, tenantID, channelID, rawIdentifier string) (*entity.ConversationOwnerRule, error) {
	identifier, err := s.chave(ctx, tenantID, channelID, rawIdentifier)
	if err != nil {
		return nil, err
	}
	rule, err := s.repo.Find(ctx, tenantID, channelID, identifier)
	if err != nil {
		return nil, err
	}
	if !rule.VigenteEm(s.now()) {
		return nil, errors.New(errors.ErrCodeNotFound, "no owner rule in force for this contact")
	}
	return rule, nil
}

// Release deletes the rule only if it still belongs to conversaRef (a late
// release of an old conversation does not erase a newer rule). Idempotent:
// released is false when there was nothing of that conversation to release.
func (s *ConversationOwnerService) Release(ctx context.Context, tenantID, channelID, rawIdentifier, conversaRef string) (released bool, err error) {
	if conversaRef == "" {
		return false, errors.New(errors.ErrCodeValidation, "conversa_ref is required")
	}
	identifier, err := s.chave(ctx, tenantID, channelID, rawIdentifier)
	if err != nil {
		return false, err
	}
	return s.repo.Release(ctx, tenantID, channelID, identifier, conversaRef)
}

// chave checks that the channel belongs to the tenant and normalizes the contact identifier.
func (s *ConversationOwnerService) chave(ctx context.Context, tenantID, channelID, rawIdentifier string) (string, error) {
	identifier := entity.NormalizeContactIdentifier(rawIdentifier)
	if identifier == "" {
		return "", errors.New(errors.ErrCodeValidation, "contact identifier is required")
	}
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil && errors.IsAppError(err) && !errors.IsNotFound(err) {
		return "", err // falha do banco não vira "não encontrado"
	}
	if err != nil || channel == nil || channel.TenantID != tenantID {
		// Same shape as ChannelService.GetByTenantAndID: never reveal that the id exists under another tenant.
		return "", errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}
	return identifier, nil
}

func utcOrNil(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
