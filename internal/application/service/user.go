package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// CreateUserInput represents input for creating a user
type CreateUserInput struct {
	TenantID string
	Email    string
	Password string
	Name     string
	Role     entity.UserRole
}

// UpdateUserInput represents input for updating a user
type UpdateUserInput struct {
	Name      *string
	Role      *entity.UserRole
	AvatarURL *string
	Status    *entity.UserStatus
}

// UserService handles user operations
type UserService struct {
	userRepo   repository.UserRepository
	tenantRepo repository.TenantRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo repository.UserRepository, tenantRepo repository.TenantRepository) *UserService {
	return &UserService{
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
	}
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, input *CreateUserInput) (*entity.User, error) {
	// Check if tenant exists
	tenant, err := s.tenantRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeTenantNotFound, "Tenant not found")
	}

	// Check tenant limits
	if tenant.Limits != nil && tenant.Limits.MaxUsers > 0 {
		count, err := s.userRepo.CountByTenant(ctx, input.TenantID)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to count users")
		}
		if int(count) >= tenant.Limits.MaxUsers {
			return nil, errors.New(errors.ErrCodeQuotaExceeded, "Maximum number of users reached")
		}
	}

	// Check if email already exists
	existing, _ := s.userRepo.FindByTenantAndEmail(ctx, input.TenantID, input.Email)
	if existing != nil {
		return nil, errors.New(errors.ErrCodeConflict, "Email already in use")
	}

	// Hash password
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to hash password")
	}

	// Create user
	user := entity.NewUser(input.TenantID, input.Email, passwordHash, input.Name, input.Role)
	user.ID = uuid.New().String()

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to create user")
	}

	return user, nil
}

// GetByID returns a user by ID
func (s *UserService) GetByID(ctx context.Context, id string) (*entity.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeUserNotFound, "User not found")
	}
	return user, nil
}

// List returns users for a tenant
func (s *UserService) List(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.User, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}
	return s.userRepo.FindByTenant(ctx, tenantID, params)
}

// Update updates a user
func (s *UserService) Update(ctx context.Context, id string, input *UpdateUserInput) (*entity.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.ErrCodeUserNotFound, "User not found")
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Role != nil {
		user.Role = *input.Role
	}
	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}
	if input.Status != nil {
		user.Status = *input.Status
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to update user")
	}

	return user, nil
}

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, id string) error {
	_, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.ErrCodeUserNotFound, "User not found")
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "Failed to delete user")
	}

	return nil
}
