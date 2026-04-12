package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const apiKeyPrefixLength = 12

// APIKeyService handles API key generation and persistence.
type APIKeyService struct {
	apiKeyRepo repository.APIKeyRepository
}

// CreateAPIKeyInput represents input for creating an API key.
type CreateAPIKeyInput struct {
	TenantID  string
	UserID    string
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
}

// CreateAPIKeyResult returns the persisted metadata and the one-time raw key.
type CreateAPIKeyResult struct {
	APIKey *entity.APIKey
	Key    string
}

// NewAPIKeyService creates a new API key service.
func NewAPIKeyService(apiKeyRepo repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{apiKeyRepo: apiKeyRepo}
}

// Create generates and stores a new API key. The raw key is returned only from this method.
func (s *APIKeyService) Create(ctx context.Context, input *CreateAPIKeyInput) (*CreateAPIKeyResult, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New(errors.ErrCodeValidation, "API key name is required")
	}
	if input.TenantID == "" {
		return nil, errors.New(errors.ErrCodeValidation, "Tenant ID is required")
	}

	rawKey, err := generateRawAPIKey()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to generate API key")
	}

	keyHash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to hash API key")
	}

	scopes := input.Scopes
	if len(scopes) == 0 {
		scopes = []string{"*"}
	}

	now := time.Now()
	var userID *string
	if input.UserID != "" {
		userID = &input.UserID
	}

	apiKey := &entity.APIKey{
		ID:        uuid.New().String(),
		TenantID:  input.TenantID,
		UserID:    userID,
		Name:      name,
		KeyHash:   string(keyHash),
		KeyPrefix: rawKey[:apiKeyPrefixLength],
		Scopes:    scopes,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: now,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "Failed to create API key")
	}

	return &CreateAPIKeyResult{APIKey: apiKey, Key: rawKey}, nil
}

// List returns API key metadata for a tenant without exposing key hashes.
func (s *APIKeyService) List(ctx context.Context, tenantID string) ([]*entity.APIKey, error) {
	if tenantID == "" {
		return nil, errors.New(errors.ErrCodeValidation, "Tenant ID is required")
	}
	return s.apiKeyRepo.ListByTenant(ctx, tenantID)
}

// Delete removes an API key for the current tenant.
func (s *APIKeyService) Delete(ctx context.Context, tenantID, id string) error {
	if tenantID == "" || id == "" {
		return errors.New(errors.ErrCodeValidation, "Tenant ID and API key ID are required")
	}
	return s.apiKeyRepo.Delete(ctx, tenantID, id)
}

func generateRawAPIKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "lk_" + hex.EncodeToString(bytes), nil
}
