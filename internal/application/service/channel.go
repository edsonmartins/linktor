package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// DefaultListParams returns default pagination parameters
func DefaultListParams() *repository.ListParams {
	return &repository.ListParams{
		Page:     1,
		PageSize: 100,
		SortBy:   "created_at",
		SortDir:  "desc",
	}
}

// CreateChannelInput represents input for creating a channel
type CreateChannelInput struct {
	TenantID    string
	Type        string
	Name        string
	Identifier  string
	Config      map[string]string
	Credentials map[string]string
}

// UpdateChannelInput represents input for updating a channel
type UpdateChannelInput struct {
	Name        *string
	Identifier  *string
	Config      map[string]string
	Credentials map[string]string
}

// ConnectResult represents the result of connecting a channel
type ConnectResult struct {
	Channel *entity.Channel `json:"channel"`
	QRCode  string          `json:"qr_code,omitempty"`
}

// ChannelService handles channel operations
type ChannelService struct {
	repo repository.ChannelRepository
}

// NewChannelService creates a new channel service
func NewChannelService(repo repository.ChannelRepository) *ChannelService {
	return &ChannelService{
		repo: repo,
	}
}

// List returns all channels for a tenant
func (s *ChannelService) List(ctx context.Context, tenantID string) ([]*entity.Channel, error) {
	if s.repo == nil {
		return nil, errors.New(errors.ErrCodeInternal, "channel repository not initialized")
	}
	// Use default pagination params
	params := repository.NewListParams()
	params.Page = 1
	params.PageSize = 100
	params.SortBy = "created_at"
	params.SortDir = "desc"

	channels, _, err := s.repo.FindByTenant(ctx, tenantID, params)
	return channels, err
}

// Create creates a new channel
func (s *ChannelService) Create(ctx context.Context, input *CreateChannelInput) (*entity.Channel, error) {
	now := time.Now()
	channel := &entity.Channel{
		ID:          uuid.New().String(),
		TenantID:    input.TenantID,
		Type:        entity.ChannelType(input.Type),
		Name:        input.Name,
		Identifier:  input.Identifier,
		Status:      entity.ChannelStatusDisconnected,
		Config:      input.Config,
		Credentials: input.Credentials,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetByID returns a channel by ID
func (s *ChannelService) GetByID(ctx context.Context, id string) (*entity.Channel, error) {
	return s.repo.FindByID(ctx, id)
}

// Update updates a channel
func (s *ChannelService) Update(ctx context.Context, id string, input *UpdateChannelInput) (*entity.Channel, error) {
	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		channel.Name = *input.Name
	}
	if input.Identifier != nil {
		channel.Identifier = *input.Identifier
	}
	if input.Config != nil {
		channel.Config = input.Config
	}
	if input.Credentials != nil {
		channel.Credentials = input.Credentials
	}
	channel.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// Delete deletes a channel
func (s *ChannelService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Connect connects a channel
func (s *ChannelService) Connect(ctx context.Context, id string) (*ConnectResult, error) {
	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	channel.Status = entity.ChannelStatusActive
	channel.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}

	// For WhatsApp, we need to return a QR code
	// This would be handled by the adapter
	return &ConnectResult{
		Channel: channel,
	}, nil
}

// Disconnect disconnects a channel
func (s *ChannelService) Disconnect(ctx context.Context, id string) error {
	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	channel.Status = entity.ChannelStatusDisconnected
	channel.UpdatedAt = time.Now()

	return s.repo.Update(ctx, channel)
}
