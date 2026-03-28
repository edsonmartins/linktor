package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// CreateContactInput represents input for creating a contact
type CreateContactInput struct {
	TenantID     string
	Name         string
	Email        string
	Phone        string
	AvatarURL    string
	CustomFields map[string]string
	Tags         []string
}

// UpdateContactInput represents input for updating a contact
type UpdateContactInput struct {
	Name         *string
	Email        *string
	Phone        *string
	AvatarURL    *string
	CustomFields map[string]string
	Tags         []string
}

// ContactService handles contact operations
type ContactService struct {
	contactRepo repository.ContactRepository
}

// NewContactService creates a new contact service
func NewContactService(contactRepo repository.ContactRepository) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
	}
}

// List returns all contacts for a tenant
func (s *ContactService) List(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Contact, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}
	return s.contactRepo.FindByTenant(ctx, tenantID, params)
}

// Create creates a new contact
func (s *ContactService) Create(ctx context.Context, input *CreateContactInput) (*entity.Contact, error) {
	if input.Name == "" {
		return nil, errors.Validation("contact name is required")
	}

	// Check for duplicate email within tenant
	if input.Email != "" {
		existing, err := s.contactRepo.FindByEmail(ctx, input.TenantID, input.Email)
		if err == nil && existing != nil {
			return nil, errors.Conflict("contact with this email already exists")
		}
	}

	// Check for duplicate phone within tenant
	if input.Phone != "" {
		existing, err := s.contactRepo.FindByPhone(ctx, input.TenantID, input.Phone)
		if err == nil && existing != nil {
			return nil, errors.Conflict("contact with this phone already exists")
		}
	}

	now := time.Now()
	contact := &entity.Contact{
		ID:           uuid.New().String(),
		TenantID:     input.TenantID,
		Name:         input.Name,
		Email:        input.Email,
		Phone:        input.Phone,
		AvatarURL:    input.AvatarURL,
		CustomFields: input.CustomFields,
		Tags:         input.Tags,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create contact")
	}

	return contact, nil
}

// GetByID returns a contact by ID
func (s *ContactService) GetByID(ctx context.Context, id string) (*entity.Contact, error) {
	contact, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	// Load identities
	identities, err := s.contactRepo.FindIdentitiesByContact(ctx, id)
	if err == nil {
		contact.Identities = identities
	}

	return contact, nil
}

// Update updates a contact
func (s *ContactService) Update(ctx context.Context, id string, input *UpdateContactInput) (*entity.Contact, error) {
	contact, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	if input.Name != nil {
		contact.Name = *input.Name
	}
	if input.Email != nil {
		contact.Email = *input.Email
	}
	if input.Phone != nil {
		contact.Phone = *input.Phone
	}
	if input.AvatarURL != nil {
		contact.AvatarURL = *input.AvatarURL
	}
	if input.CustomFields != nil {
		contact.CustomFields = input.CustomFields
	}
	if input.Tags != nil {
		contact.Tags = input.Tags
	}
	contact.UpdatedAt = time.Now()

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update contact")
	}

	return contact, nil
}

// Delete deletes a contact
func (s *ContactService) Delete(ctx context.Context, id string) error {
	_, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	if err := s.contactRepo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete contact")
	}

	return nil
}

// AddIdentity adds an identity to a contact
func (s *ContactService) AddIdentity(ctx context.Context, contactID, channelType, identifier string, metadata map[string]string) (*entity.Contact, error) {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	identity := &entity.ContactIdentity{
		ID:          uuid.New().String(),
		ContactID:   contactID,
		ChannelType: channelType,
		Identifier:  identifier,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}

	if err := s.contactRepo.AddIdentity(ctx, identity); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to add identity")
	}

	// Reload identities
	identities, err := s.contactRepo.FindIdentitiesByContact(ctx, contactID)
	if err == nil {
		contact.Identities = identities
	}

	return contact, nil
}

// RemoveIdentity removes an identity from a contact
func (s *ContactService) RemoveIdentity(ctx context.Context, contactID, identityID string) (*entity.Contact, error) {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	if err := s.contactRepo.RemoveIdentity(ctx, contactID, identityID); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to remove identity")
	}

	// Reload identities
	identities, err := s.contactRepo.FindIdentitiesByContact(ctx, contactID)
	if err == nil {
		contact.Identities = identities
	}

	return contact, nil
}

// BlockContact blocks a contact
func (s *ContactService) BlockContact(ctx context.Context, contactID string) error {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	contact.Block()

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to block contact")
	}

	return nil
}

// UnblockContact unblocks a contact
func (s *ContactService) UnblockContact(ctx context.Context, contactID string) error {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	contact.Unblock()

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to unblock contact")
	}

	return nil
}

// IsBlocked checks if a contact is blocked
func (s *ContactService) IsBlocked(ctx context.Context, contactID string) (bool, error) {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return false, errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	return contact.IsBlocked(), nil
}
