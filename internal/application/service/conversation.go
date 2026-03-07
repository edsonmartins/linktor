package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
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
	Tags       []string
}

// ConversationService handles conversation operations
type ConversationService struct {
	conversationRepo repository.ConversationRepository
	contactRepo      repository.ContactRepository
	channelRepo      repository.ChannelRepository
}

// NewConversationService creates a new conversation service
func NewConversationService(
	conversationRepo repository.ConversationRepository,
	contactRepo repository.ContactRepository,
	channelRepo repository.ChannelRepository,
) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		contactRepo:      contactRepo,
		channelRepo:      channelRepo,
	}
}

// List returns all conversations for a tenant
func (s *ConversationService) List(ctx context.Context, tenantID string, filters *ConversationFilters, params *repository.ListParams) ([]*entity.Conversation, int64, error) {
	if params == nil {
		params = repository.NewListParams()
		params.SortBy = "updated_at"
	}

	// Apply filters to ListParams
	if filters != nil {
		if filters.Status != "" {
			params.Filters["status"] = filters.Status
		}
		if filters.AssignedTo != "" {
			params.Filters["assigned_user_id"] = filters.AssignedTo
		}
		if filters.ChannelID != "" {
			params.Filters["channel_id"] = filters.ChannelID
		}
		if filters.ContactID != "" {
			params.Filters["contact_id"] = filters.ContactID
		}
	}

	return s.conversationRepo.FindByTenant(ctx, tenantID, params)
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
	_, err := s.contactRepo.FindByID(ctx, input.ContactID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	// Verify channel exists
	_, err = s.channelRepo.FindByID(ctx, input.ChannelID)
	if err != nil {
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
	}

	now := time.Now()
	conversation := &entity.Conversation{
		ID:        uuid.New().String(),
		TenantID:  input.TenantID,
		ContactID: input.ContactID,
		ChannelID: input.ChannelID,
		Status:    entity.ConversationStatusOpen,
		Priority:  priority,
		Subject:   input.Subject,
		Tags:      input.Tags,
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
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
		conversation.Priority = entity.ConversationPriority(*input.Priority)
	}
	if input.Status != nil {
		conversation.Status = entity.ConversationStatus(*input.Status)
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

// Assign assigns a conversation to a user
func (s *ConversationService) Assign(ctx context.Context, id, userID string) (*entity.Conversation, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}

	conversation.Assign(userID)
	conversation.UpdatedAt = time.Now()

	if err := s.conversationRepo.UpdateAssignee(ctx, id, conversation.AssignedUserID); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to assign conversation")
	}

	return conversation, nil
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

	return conversation, nil
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

	return conversation, nil
}
