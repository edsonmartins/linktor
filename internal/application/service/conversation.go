package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

// CreateConversationInput represents input for creating a conversation
type CreateConversationInput struct {
	TenantID  string
	ContactID string
	ChannelID string
	Subject   string
	Priority  string
	Tags      []string
}

// UpdateConversationInput represents input for updating a conversation
type UpdateConversationInput struct {
	Subject  *string
	Priority *string
	Status   *string
	Tags     []string
}

// ConversationFilters represents conversation filter options
type ConversationFilters struct {
	Status     string
	AssignedTo string
	ChannelID  string
	ContactID  string
	// Search casa com nome, telefone ou e-mail do contato — é por quem o
	// atendente procura ao abrir a caixa.
	Search string
	// Environment filters by the denormalized channel environment
	// ("production" | "sandbox", INV-018). Empty means no filter (all).
	Environment string
	Tags        []string
}

// ConversationService handles conversation operations
type ConversationService struct {
	conversationRepo repository.ConversationRepository
	contactRepo      repository.ContactRepository
	channelRepo      repository.ChannelRepository
	producer         nats.Publisher
}

// NewConversationService creates a new conversation service. producer may be nil
// (lifecycle events are simply not published in that case).
func NewConversationService(
	conversationRepo repository.ConversationRepository,
	contactRepo repository.ContactRepository,
	channelRepo repository.ChannelRepository,
	producer nats.Publisher,
) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		contactRepo:      contactRepo,
		channelRepo:      channelRepo,
		producer:         producer,
	}
}

// publishLifecycleEvent emits a conversation lifecycle event (assigned/resolved/
// reopened) so external consumers receive it via the outbound webhook. It is
// best-effort: a nil producer or publish failure never blocks the operation.
func (s *ConversationService) publishLifecycleEvent(ctx context.Context, eventType string, conversation *entity.Conversation) {
	if s.producer == nil || conversation == nil {
		return
	}
	_ = s.producer.PublishEvent(ctx, &nats.Event{
		Type:     eventType,
		TenantID: conversation.TenantID,
		Payload: map[string]interface{}{
			"conversation_id":  conversation.ID,
			"channel_id":       conversation.ChannelID,
			"contact_id":       conversation.ContactID,
			"status":           string(conversation.Status),
			"assigned_user_id": conversation.AssignedUserID,
		},
		Timestamp: time.Now(),
	})
}

// ConversationPage is one window of the conversation list, most recent
// activity first.
type ConversationPage struct {
	Conversations []*entity.Conversation
	// HasMore reports whether older conversations exist after the last one here.
	HasMore bool
	// Total is how many conversations match the filters. Só vem na primeira
	// página: contar a cada rolagem custaria varrer a caixa inteira para
	// responder um número que a tela já mostrou.
	Total int64
}

const (
	defaultConversationPageSize = 30
	maxConversationPageSize     = 100
)

// filterMap traduz os filtros da API para as chaves que o repositório conhece.
func (f *ConversationFilters) filterMap() (map[string]interface{}, error) {
	filters := map[string]interface{}{}
	if f == nil {
		return filters, nil
	}
	if f.Status != "" {
		filters["status"] = f.Status
	}
	if f.AssignedTo != "" {
		filters["assigned_user_id"] = f.AssignedTo
	}
	if f.ChannelID != "" {
		filters["channel_id"] = f.ChannelID
	}
	if f.ContactID != "" {
		filters["contact_id"] = f.ContactID
	}
	if f.Search != "" {
		filters["search"] = f.Search
	}
	if f.Environment != "" {
		// Validado na borda para que um valor arbitrário não chegue à consulta
		// como filtro, e para que erro de quem chama não se pareça com "não há
		// resultado".
		env, ok := entity.ParseChannelEnvironment(f.Environment)
		if !ok {
			return nil, errors.Validation("invalid environment filter: " + f.Environment)
		}
		filters["environment"] = string(env)
	}
	return filters, nil
}

// ListPage returns one page of the tenant's conversations, most recent activity
// first, ending just before the cursor (nil = the top of the list).
//
// Pede uma linha além do limite para saber se há mais, em vez de contar a cada
// rolagem.
func (s *ConversationService) ListPage(
	ctx context.Context,
	tenantID string,
	filters *ConversationFilters,
	before *repository.ConversationCursor,
	limit int,
) (*ConversationPage, error) {
	filterMap, err := filters.filterMap()
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = defaultConversationPageSize
	}
	if limit > maxConversationPageSize {
		limit = maxConversationPageSize
	}

	conversations, err := s.conversationRepo.FindPageByTenant(ctx, tenantID, before, limit+1, filterMap)
	if err != nil {
		return nil, err
	}

	page := &ConversationPage{Conversations: conversations}
	if len(conversations) > limit {
		page.Conversations = conversations[:limit]
		page.HasMore = true
	}

	if before == nil {
		total, err := s.conversationRepo.CountByTenantFiltered(ctx, tenantID, filterMap)
		if err != nil {
			return nil, err
		}
		page.Total = total
	}

	s.enrichConversations(ctx, page.Conversations)
	return page, nil
}

// enrichConversations populates each conversation's Contact and Channel relation
// for API responses. The repository scans only conversation columns, so without
// this the list shows every contact as "unknown" and every channel unlabeled.
// Best-effort: a lookup miss leaves that relation nil (the UI localizes it)
// rather than failing the whole list. Channels are deduped (and negative results
// cached) since many conversations share a single channel.
func (s *ConversationService) enrichConversations(ctx context.Context, conversations []*entity.Conversation) {
	channelCache := make(map[string]*entity.Channel)
	contactCache := make(map[string]*entity.Contact)
	for _, conv := range conversations {
		if conv.ContactID != "" {
			contact, ok := contactCache[conv.ContactID]
			if !ok {
				contact, _ = s.contactRepo.FindByID(ctx, conv.ContactID)
				contactCache[conv.ContactID] = contact
			}
			conv.Contact = contact
		}
		if conv.ChannelID != "" {
			channel, ok := channelCache[conv.ChannelID]
			if !ok {
				channel, _ = s.channelRepo.FindByID(ctx, conv.ChannelID)
				channelCache[conv.ChannelID] = channel
			}
			conv.Channel = channel
		}
	}
}

// Create creates a new conversation
func (s *ConversationService) Create(ctx context.Context, input *CreateConversationInput) (*entity.Conversation, error) {
	if input.ContactID == "" {
		return nil, errors.Validation("contact_id is required")
	}
	if input.ChannelID == "" {
		return nil, errors.Validation("channel_id is required")
	}

	// Verify contact exists
	contact, err := s.contactRepo.FindByID(ctx, input.ContactID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}
	if contact.TenantID != input.TenantID {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	// Verify channel exists
	channel, err := s.channelRepo.FindByID(ctx, input.ChannelID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}
	if channel.TenantID != input.TenantID {
		return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	// Check for existing open conversation
	existing, err := s.conversationRepo.FindOpenByContactAndChannel(ctx, input.ContactID, input.ChannelID)
	if err == nil && existing != nil {
		return existing, nil
	}

	priority := entity.ConversationPriority(input.Priority)
	if priority == "" {
		priority = entity.ConversationPriorityNormal
	} else if !priority.IsValid() {
		return nil, errors.Validation("invalid conversation priority: " + input.Priority)
	}

	now := time.Now()
	conversation := &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    input.TenantID,
		ContactID:   input.ContactID,
		ChannelID:   input.ChannelID,
		Environment: channel.Environment, // denormalized at birth (INV-018)
		Status:      entity.ConversationStatusOpen,
		Priority:    priority,
		Subject:     input.Subject,
		Tags:        input.Tags,
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create conversation")
	}

	return conversation, nil
}

// GetByID returns a conversation by ID
func (s *ConversationService) GetByID(ctx context.Context, id string) (*entity.Conversation, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}
	return conversation, nil
}

// GetByTenantAndID returns a conversation only if it belongs to the tenant.
func (s *ConversationService) GetByTenantAndID(ctx context.Context, tenantID, id string) (*entity.Conversation, error) {
	conversation, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if conversation.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}
	s.enrichConversations(ctx, []*entity.Conversation{conversation})
	return conversation, nil
}

// Update updates a conversation
func (s *ConversationService) Update(ctx context.Context, id string, input *UpdateConversationInput) (*entity.Conversation, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	if input.Subject != nil {
		conversation.Subject = *input.Subject
	}
	if input.Priority != nil {
		priority := entity.ConversationPriority(*input.Priority)
		if !priority.IsValid() {
			return nil, errors.Validation("invalid conversation priority: " + *input.Priority)
		}
		conversation.Priority = priority
	}
	if input.Status != nil {
		status := entity.ConversationStatus(*input.Status)
		if !status.IsValid() {
			return nil, errors.Validation("invalid conversation status: " + *input.Status)
		}
		conversation.Status = status
	}
	if input.Tags != nil {
		conversation.Tags = input.Tags
	}
	conversation.UpdatedAt = time.Now()

	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update conversation")
	}

	return conversation, nil
}

// UpdateForTenant updates a conversation only if it belongs to the tenant.
func (s *ConversationService) UpdateForTenant(ctx context.Context, tenantID, id string, input *UpdateConversationInput) (*entity.Conversation, error) {
	conversation, err := s.GetByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return s.Update(ctx, conversation.ID, input)
}

// Assign assigns a conversation to a user
func (s *ConversationService) Assign(ctx context.Context, id, userID string) (*entity.Conversation, error) {
	if userID == "" {
		return nil, errors.Validation("user_id is required")
	}

	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	conversation.Assign(userID)
	conversation.UpdatedAt = time.Now()

	if err := s.conversationRepo.UpdateAssignee(ctx, id, conversation.AssignedUserID); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to assign conversation")
	}

	s.publishLifecycleEvent(ctx, nats.EventConversationAssigned, conversation)

	return conversation, nil
}

// AssignForTenant assigns a conversation only if it belongs to the tenant.
func (s *ConversationService) AssignForTenant(ctx context.Context, tenantID, id, userID string) (*entity.Conversation, error) {
	conversation, err := s.GetByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return s.Assign(ctx, conversation.ID, userID)
}

// Resolve marks a conversation as resolved
func (s *ConversationService) Resolve(ctx context.Context, id string) (*entity.Conversation, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	if conversation.Status == entity.ConversationStatusResolved {
		return nil, errors.Validation("conversation is already resolved")
	}

	conversation.Resolve()
	conversation.UpdatedAt = time.Now()

	if err := s.conversationRepo.UpdateStatus(ctx, id, entity.ConversationStatusResolved); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to resolve conversation")
	}

	s.publishLifecycleEvent(ctx, nats.EventConversationResolved, conversation)

	return conversation, nil
}

// ResolveForTenant resolves a conversation only if it belongs to the tenant.
func (s *ConversationService) ResolveForTenant(ctx context.Context, tenantID, id string) (*entity.Conversation, error) {
	conversation, err := s.GetByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return s.Resolve(ctx, conversation.ID)
}

// Reopen reopens a resolved conversation
func (s *ConversationService) Reopen(ctx context.Context, id string) (*entity.Conversation, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	if conversation.IsOpen() {
		return nil, errors.Validation("conversation is already open")
	}

	conversation.Reopen()
	conversation.UpdatedAt = time.Now()

	if err := s.conversationRepo.UpdateStatus(ctx, id, entity.ConversationStatusOpen); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to reopen conversation")
	}

	s.publishLifecycleEvent(ctx, nats.EventConversationReopened, conversation)

	return conversation, nil
}

// ReopenForTenant reopens a conversation only if it belongs to the tenant.
func (s *ConversationService) ReopenForTenant(ctx context.Context, tenantID, id string) (*entity.Conversation, error) {
	conversation, err := s.GetByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return s.Reopen(ctx, conversation.ID)
}

// Delete permanently removes a conversation. The schema cascades from it, so the
// messages, attachments and tags of the conversation go with it — there is no
// undo and no soft-delete tombstone. Callers that only want it out of the agent's
// way should resolve it instead.
//
// The conversation is loaded before the delete so the lifecycle event can carry
// its identity (channel, contact, status): after the row is gone there is nothing
// left to read it from.
func (s *ConversationService) Delete(ctx context.Context, id string) error {
	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	if err := s.conversationRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.publishLifecycleEvent(ctx, nats.EventConversationDeleted, conversation)

	return nil
}

// DeleteForTenant deletes a conversation only if it belongs to the tenant.
func (s *ConversationService) DeleteForTenant(ctx context.Context, tenantID, id string) error {
	conversation, err := s.GetByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	return s.Delete(ctx, conversation.ID)
}
