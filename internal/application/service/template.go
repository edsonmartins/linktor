package service

import (
	"context"
	"fmt"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// TemplateService handles template operations
type TemplateService struct {
	templateRepo repository.TemplateRepository
	channelRepo  repository.ChannelRepository
	// TODO: Add WhatsApp client factory
}

// NewTemplateService creates a new template service
func NewTemplateService(
	templateRepo repository.TemplateRepository,
	channelRepo repository.ChannelRepository,
) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		channelRepo:  channelRepo,
	}
}

// CreateTemplateInput represents input for creating a template
type CreateTemplateInput struct {
	TenantID   string
	ChannelID  string
	Name       string
	Language   string
	Category   entity.TemplateCategory
	Components []entity.TemplateComponent
}

// Create creates a new template (locally and optionally syncs to Meta)
func (s *TemplateService) Create(ctx context.Context, input *CreateTemplateInput) (*entity.Template, error) {
	// Check if template already exists
	existing, _ := s.templateRepo.FindByName(ctx, input.TenantID, input.ChannelID, input.Name, input.Language)
	if existing != nil {
		return nil, fmt.Errorf("template with name '%s' and language '%s' already exists", input.Name, input.Language)
	}

	template := &entity.Template{
		ID:           generateID(),
		TenantID:     input.TenantID,
		ChannelID:    input.ChannelID,
		Name:         input.Name,
		Language:     input.Language,
		Category:     input.Category,
		Status:       entity.TemplateStatusPending,
		QualityScore: entity.TemplateQualityUnknown,
		Components:   input.Components,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// TODO: Create template on Meta and get external ID
	// client := s.getWhatsAppClient(ctx, input.ChannelID)
	// response, err := client.CreateTemplate(ctx, template)
	// template.ExternalID = response.ID

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetByID returns a template by ID
func (s *TemplateService) GetByID(ctx context.Context, id string) (*entity.Template, error) {
	return s.templateRepo.FindByID(ctx, id)
}

// GetByName returns a template by name and language
func (s *TemplateService) GetByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error) {
	return s.templateRepo.FindByName(ctx, tenantID, channelID, name, language)
}

// List returns all templates for a tenant
func (s *TemplateService) List(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	return s.templateRepo.FindByTenant(ctx, tenantID, params)
}

// ListByChannel returns all templates for a channel
func (s *TemplateService) ListByChannel(ctx context.Context, channelID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	return s.templateRepo.FindByChannel(ctx, channelID, params)
}

// UpdateStatus updates a template's status (from webhook)
func (s *TemplateService) UpdateStatus(ctx context.Context, externalID string, status entity.TemplateStatus, reason string) error {
	template, err := s.templateRepo.FindByExternalID(ctx, externalID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	template.UpdateStatus(status, reason)
	return s.templateRepo.Update(ctx, template)
}

// UpdateQuality updates a template's quality score (from webhook)
func (s *TemplateService) UpdateQuality(ctx context.Context, externalID string, quality entity.TemplateQuality) error {
	template, err := s.templateRepo.FindByExternalID(ctx, externalID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	template.UpdateQuality(quality)
	return s.templateRepo.Update(ctx, template)
}

// Delete deletes a template (locally and from Meta)
func (s *TemplateService) Delete(ctx context.Context, id string) error {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// TODO: Delete from Meta
	// client := s.getWhatsAppClient(ctx, template.ChannelID)
	// err = client.DeleteTemplate(ctx, template.ExternalID)

	return s.templateRepo.Delete(ctx, template.ID)
}

// SyncFromMeta syncs templates from Meta for a channel
func (s *TemplateService) SyncFromMeta(ctx context.Context, channelID string) error {
	// Get channel to obtain credentials
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	// TODO: Implement actual Meta API call
	// client := s.getWhatsAppClient(ctx, channelID)
	// templates, err := client.ListTemplates(ctx)

	// For now, mark sync time
	_ = channel
	return nil
}

// SyncToMeta syncs a local template to Meta
func (s *TemplateService) SyncToMeta(ctx context.Context, id string) error {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// TODO: Implement actual Meta API call
	// client := s.getWhatsAppClient(ctx, template.ChannelID)
	// response, err := client.CreateTemplate(ctx, template)
	// template.ExternalID = response.ID
	// template.Status = entity.TemplateStatusPending

	now := time.Now()
	template.LastSyncedAt = &now
	return s.templateRepo.Update(ctx, template)
}

// ProcessTemplateStatusWebhook processes a template status webhook event
func (s *TemplateService) ProcessTemplateStatusWebhook(ctx context.Context, event *TemplateStatusEvent) error {
	// Find template by external ID or name
	template, err := s.templateRepo.FindByExternalID(ctx, fmt.Sprintf("%d", event.TemplateID))
	if err != nil {
		// Template might not exist locally yet - create it
		return nil
	}

	// Update status
	status := mapMetaStatusToEntity(event.Event)
	template.UpdateStatus(status, event.Reason)
	return s.templateRepo.Update(ctx, template)
}

// ProcessTemplateQualityWebhook processes a template quality webhook event
func (s *TemplateService) ProcessTemplateQualityWebhook(ctx context.Context, event *TemplateQualityEvent) error {
	template, err := s.templateRepo.FindByExternalID(ctx, fmt.Sprintf("%d", event.TemplateID))
	if err != nil {
		return nil // Template not found locally
	}

	quality := mapMetaQualityToEntity(event.NewQuality)
	template.UpdateQuality(quality)
	return s.templateRepo.Update(ctx, template)
}

// TemplateStatusEvent represents a template status webhook event
type TemplateStatusEvent struct {
	TemplateID   int64
	TemplateName string
	Language     string
	Event        string
	Reason       string
}

// TemplateQualityEvent represents a template quality webhook event
type TemplateQualityEvent struct {
	TemplateID      int64
	TemplateName    string
	Language        string
	PreviousQuality string
	NewQuality      string
}

// mapMetaStatusToEntity maps Meta's status string to entity.TemplateStatus
func mapMetaStatusToEntity(status string) entity.TemplateStatus {
	switch status {
	case "APPROVED":
		return entity.TemplateStatusApproved
	case "REJECTED":
		return entity.TemplateStatusRejected
	case "PENDING":
		return entity.TemplateStatusPending
	case "PAUSED":
		return entity.TemplateStatusPaused
	case "DISABLED":
		return entity.TemplateStatusDisabled
	case "IN_APPEAL":
		return entity.TemplateStatusInAppeal
	case "PENDING_DELETION":
		return entity.TemplateStatusPendingDeletion
	case "DELETED":
		return entity.TemplateStatusDeleted
	case "REINSTATED":
		return entity.TemplateStatusReinstated
	default:
		return entity.TemplateStatusPending
	}
}

// mapMetaQualityToEntity maps Meta's quality string to entity.TemplateQuality
func mapMetaQualityToEntity(quality string) entity.TemplateQuality {
	switch quality {
	case "GREEN":
		return entity.TemplateQualityGreen
	case "YELLOW":
		return entity.TemplateQualityYellow
	case "RED":
		return entity.TemplateQualityRed
	default:
		return entity.TemplateQualityUnknown
	}
}

// generateID generates a unique ID (placeholder - should use UUID)
func generateID() string {
	return fmt.Sprintf("tpl_%d", time.Now().UnixNano())
}
