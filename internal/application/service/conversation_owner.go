package service

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
)

// ConversationOwnerService declares, reads and releases the owner of the open
// conversation with a contact on a channel (P5 of the CRM ↔ Sales Copilot
// agreement, VCR-36). The rule is tenant-scoped: a channel of another tenant
// is reported as not found, never revealed.
type ConversationOwnerService struct {
	repo        repository.ConversationOwnerRuleRepository
	channelRepo repository.ChannelRepository
	publisher   nats.Publisher
	now         func() time.Time
}

// WithPublisher enables the conversation.owner_changed event (P5, VCR-36),
// published only when the effective owner actually changes.
func (s *ConversationOwnerService) WithPublisher(p nats.Publisher) *ConversationOwnerService {
	s.publisher = p
	return s
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
	channel, identifier, err := s.chave(ctx, in.TenantID, in.ChannelID, in.Identifier)
	if err != nil {
		return false, nil, err
	}
	previous, err := s.repo.Find(ctx, in.TenantID, in.ChannelID, identifier)
	if err != nil {
		return false, nil, err
	}
	applied, current, err = s.repo.Upsert(ctx, &entity.ConversationOwnerRule{
		TenantID:          in.TenantID,
		ChannelID:         in.ChannelID,
		ContactIdentifier: identifier,
		Consumer:          in.Consumer,
		ConversaRef:       in.ConversaRef,
		Desde:             in.Desde.UTC(),
		ExpiraEm:          utcOrNil(in.ExpiraEm),
	})
	if err == nil && applied {
		s.publishIfChanged(ctx, channel, identifier, previous, current, "declarado")
	}
	return applied, current, err
}

// Get returns the rule in force, or not found when there is none or it expired.
func (s *ConversationOwnerService) Get(ctx context.Context, tenantID, channelID, rawIdentifier string) (*entity.ConversationOwnerRule, error) {
	_, identifier, err := s.chave(ctx, tenantID, channelID, rawIdentifier)
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
	channel, identifier, err := s.chave(ctx, tenantID, channelID, rawIdentifier)
	if err != nil {
		return false, err
	}
	previous, err := s.repo.Find(ctx, tenantID, channelID, identifier)
	if err != nil {
		return false, err
	}
	released, err = s.repo.Release(ctx, tenantID, channelID, identifier, conversaRef)
	if err == nil && released {
		s.publishIfChanged(ctx, channel, identifier, previous, nil, "liberado")
	}
	return released, err
}

// efetivo is the consumer that receives the contact's events under rule: the
// rule's consumer while in force, the channel's default otherwise.
func (s *ConversationOwnerService) efetivo(channel *entity.Channel, rule *entity.ConversationOwnerRule) string {
	if rule.VigenteEm(s.now()) {
		return rule.Consumer
	}
	return entity.DefaultConsumerName(channel)
}

// publishIfChanged emits conversation.owner_changed only when the effective
// owner changes: an idempotent PUT, a new conversation of the same owner or a
// release that leaves the same consumer in charge emit nothing.
func (s *ConversationOwnerService) publishIfChanged(ctx context.Context, channel *entity.Channel, identifier string,
	previous, current *entity.ConversationOwnerRule, motivo string) {
	if s.publisher == nil {
		return
	}
	antes, depois := s.efetivo(channel, previous), s.efetivo(channel, current)
	if antes == depois {
		return
	}
	payload := map[string]interface{}{
		"channel_id":         channel.ID,
		"channel_type":       string(channel.Type),
		"contact_identifier": identifier,
		"previous_consumer":  antes,
		"consumer":           depois,
		"motivo":             motivo,
	}
	if current != nil {
		payload["conversa_ref"] = current.ConversaRef
	}
	if previous != nil {
		payload["previous_conversa_ref"] = previous.ConversaRef
	}
	if err := s.publisher.PublishEvent(ctx, &nats.Event{Type: nats.EventConversationOwnerChanged, TenantID: channel.TenantID,
		Payload: payload, Timestamp: s.now()}); err != nil {
		// The rule is already written; a lost notification must not fail the declaration.
		logger.Warn("routing: failed to publish conversation.owner_changed for channel " + channel.ID + ": " + err.Error())
	}
}

// chave checks that the channel belongs to the tenant and normalizes the contact identifier.
func (s *ConversationOwnerService) chave(ctx context.Context, tenantID, channelID, rawIdentifier string) (*entity.Channel, string, error) {
	identifier := entity.NormalizeContactIdentifier(rawIdentifier)
	if identifier == "" {
		return nil, "", errors.New(errors.ErrCodeValidation, "contact identifier is required")
	}
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil && errors.IsAppError(err) && !errors.IsNotFound(err) {
		return nil, "", err // falha do banco não vira "não encontrado"
	}
	if err != nil || channel == nil || channel.TenantID != tenantID {
		// Same shape as ChannelService.GetByTenantAndID: never reveal that the id exists under another tenant.
		return nil, "", errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}
	return channel, identifier, nil
}

func utcOrNil(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
