package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// SandboxAllowlistService manages the tenant's sandbox recipient allowlist
// (INV-017). Entries are tenant-scoped; an entry may opt into a single sandbox
// channel via ChannelID. Recipients are normalized to E.164 on write, and the
// delivery guard normalizes again on comparison, so raw formatting differences
// can never cause a mismatch in either direction.
type SandboxAllowlistService struct {
	repo        repository.SandboxAllowlistRepository
	channelRepo repository.ChannelRepository
}

// NewSandboxAllowlistService creates a new SandboxAllowlistService.
func NewSandboxAllowlistService(repo repository.SandboxAllowlistRepository, channelRepo repository.ChannelRepository) *SandboxAllowlistService {
	return &SandboxAllowlistService{repo: repo, channelRepo: channelRepo}
}

// AddSandboxAllowlistInput is the input for adding an allowlist entry.
type AddSandboxAllowlistInput struct {
	TenantID  string
	ChannelID string // optional: narrows the entry to one sandbox channel
	Recipient string // any common phone formatting; normalized to E.164
	Note      string
	CreatedBy string
}

// Add validates, normalizes and persists an allowlist entry. When ChannelID is
// set, the channel must belong to the tenant and be a sandbox channel — an
// allowlist entry pointing at a production channel would be meaningless and
// likely a mistake worth failing on.
func (s *SandboxAllowlistService) Add(ctx context.Context, input *AddSandboxAllowlistInput) (*entity.SandboxAllowlistEntry, error) {
	recipient, ok := entity.NormalizeE164(input.Recipient)
	if !ok {
		return nil, errors.New(errors.ErrCodeValidation,
			"recipient must be a valid E.164 phone number")
	}

	if input.ChannelID != "" {
		channel, err := s.channelRepo.FindByID(ctx, input.ChannelID)
		if err != nil {
			return nil, err
		}
		if channel.TenantID != input.TenantID {
			// Same shape as ChannelService.GetByTenantAndID: never reveal that
			// the id exists under another tenant.
			return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
		}
		if !channel.IsSandbox() {
			return nil, errors.New(errors.ErrCodeValidation,
				"allowlist entries can only be scoped to sandbox channels")
		}
	}

	entry := &entity.SandboxAllowlistEntry{
		ID:        uuid.New().String(),
		TenantID:  input.TenantID,
		ChannelID: input.ChannelID,
		Recipient: recipient,
		Note:      input.Note,
		CreatedBy: input.CreatedBy,
		CreatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// List returns the tenant's allowlist entries.
func (s *SandboxAllowlistService) List(ctx context.Context, tenantID string) ([]*entity.SandboxAllowlistEntry, error) {
	return s.repo.FindByTenant(ctx, tenantID)
}

// Remove deletes an entry, returning it so callers can audit what was removed.
// Tenant scoping is enforced by the repository: a cross-tenant id fails as
// not-found. Removal takes effect on the next send — the delivery guard
// consults the allowlist at send time and never caches it.
func (s *SandboxAllowlistService) Remove(ctx context.Context, tenantID, id string) (*entity.SandboxAllowlistEntry, error) {
	entry, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return nil, err
	}
	return entry, nil
}
