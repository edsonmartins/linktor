package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/whatsapp/flows"
	"github.com/msgfy/linktor/pkg/errors"
)

// WhatsAppFlowsService connects the WhatsApp Flows client to the application layer.
type WhatsAppFlowsService struct {
	flowRepo    repository.FlowRepository
	channelRepo repository.ChannelRepository
}

// NewWhatsAppFlowsService creates a new WhatsAppFlowsService.
func NewWhatsAppFlowsService(flowRepo repository.FlowRepository, channelRepo repository.ChannelRepository) *WhatsAppFlowsService {
	return &WhatsAppFlowsService{
		flowRepo:    flowRepo,
		channelRepo: channelRepo,
	}
}

// createFlowClient builds a FlowClient from the channel's stored credentials.
// It enforces tenant isolation: the channel must belong to tenantID, otherwise a
// not-found error is returned so a tenant cannot use another tenant's channel
// access_token.
func (s *WhatsAppFlowsService) createFlowClient(ctx context.Context, tenantID, channelID string) (*flows.FlowClient, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "channel not found")
	}
	if channel == nil || channel.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "channel not found")
	}

	accessToken := channel.Config["access_token"]
	if accessToken == "" {
		accessToken = channel.Credentials["access_token"]
	}
	if accessToken == "" {
		return nil, fmt.Errorf("channel %s has no access_token configured", channelID)
	}

	wabaID := channel.Config["waba_id"]
	if wabaID == "" {
		wabaID = channel.WABAID
	}
	if wabaID == "" {
		return nil, fmt.Errorf("channel %s has no waba_id configured", channelID)
	}

	client := flows.NewFlowClient(&flows.FlowClientConfig{
		AccessToken: accessToken,
		WABAID:      wabaID,
	})

	return client, nil
}

// CreateFlow creates a new WhatsApp Flow via the Meta API and persists it locally.
func (s *WhatsAppFlowsService) CreateFlow(ctx context.Context, tenantID, channelID, name string, categories []string, endpointURI string) (*entity.Flow, error) {
	client, err := s.createFlowClient(ctx, tenantID, channelID)
	if err != nil {
		return nil, err
	}

	metaFlow, err := client.CreateFlow(ctx, &flows.CreateFlowRequest{
		Name:        name,
		Categories:  categories,
		EndpointURI: endpointURI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create flow on Meta: %w", err)
	}

	now := time.Now()
	flow := &entity.Flow{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        name,
		Description: fmt.Sprintf("WhatsApp Flow %s", name),
		MetaFlowID:  metaFlow.ID,
		Trigger:     entity.FlowTriggerManual,
		Nodes:       []entity.FlowNode{},
		IsActive:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.flowRepo.Create(ctx, flow); err != nil {
		return nil, fmt.Errorf("failed to save flow: %w", err)
	}

	return flow, nil
}

// GetFlow retrieves a flow from the local repository, scoped to the tenant.
// A flow belonging to another tenant is reported as not found.
func (s *WhatsAppFlowsService) GetFlow(ctx context.Context, tenantID, flowID string) (*entity.Flow, error) {
	flow, err := s.flowRepo.FindByID(ctx, flowID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "flow not found")
	}
	if flow == nil || flow.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "flow not found")
	}
	return flow, nil
}

// ListFlows lists flows for a tenant with optional filtering and pagination.
func (s *WhatsAppFlowsService) ListFlows(ctx context.Context, tenantID string, filter *entity.FlowFilter, params *repository.ListParams) ([]*entity.Flow, int64, error) {
	return s.flowRepo.FindByTenant(ctx, tenantID, filter, params)
}

// UpdateFlow updates a WhatsApp Flow via the Meta API and the local repository.
func (s *WhatsAppFlowsService) UpdateFlow(ctx context.Context, tenantID, channelID, flowID, name string, categories []string, endpointURI string) (*entity.Flow, error) {
	flow, err := s.flowRepo.FindByID(ctx, flowID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "flow not found")
	}
	if flow == nil || flow.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "flow not found")
	}

	if flow.MetaFlowID == "" {
		return nil, errors.New(errors.ErrCodeValidation, "flow is not synced to Meta")
	}

	client, err := s.createFlowClient(ctx, tenantID, channelID)
	if err != nil {
		return nil, err
	}

	_, err = client.UpdateFlow(ctx, flow.MetaFlowID, &flows.UpdateFlowRequest{
		Name:        name,
		Categories:  categories,
		EndpointURI: endpointURI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update flow on Meta: %w", err)
	}

	if name != "" {
		flow.Name = name
	}
	flow.UpdatedAt = time.Now()

	if err := s.flowRepo.Update(ctx, flow); err != nil {
		return nil, fmt.Errorf("failed to update flow in repository: %w", err)
	}

	return flow, nil
}

// ensureFlowTenant loads the flow and verifies that it belongs to tenantID,
// reporting a not-found error otherwise (prevents cross-tenant flow
// operations). The loaded flow is returned so callers can address the Meta API
// by its persisted MetaFlowID.
func (s *WhatsAppFlowsService) ensureFlowTenant(ctx context.Context, tenantID, flowID string) (*entity.Flow, error) {
	flow, err := s.flowRepo.FindByID(ctx, flowID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "flow not found")
	}
	if flow == nil || flow.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "flow not found")
	}
	return flow, nil
}

// DeleteFlow deletes a WhatsApp Flow via the Meta API and removes it locally.
func (s *WhatsAppFlowsService) DeleteFlow(ctx context.Context, tenantID, channelID, flowID string) error {
	flow, err := s.ensureFlowTenant(ctx, tenantID, flowID)
	if err != nil {
		return err
	}
	if flow.MetaFlowID == "" {
		return errors.New(errors.ErrCodeValidation, "flow is not synced to Meta")
	}

	client, err := s.createFlowClient(ctx, tenantID, channelID)
	if err != nil {
		return err
	}

	if err := client.DeleteFlow(ctx, flow.MetaFlowID); err != nil {
		return fmt.Errorf("failed to delete flow on Meta: %w", err)
	}

	if err := s.flowRepo.Delete(ctx, flowID); err != nil {
		return fmt.Errorf("failed to delete flow from repository: %w", err)
	}

	return nil
}

// PublishFlow publishes a WhatsApp Flow via the Meta API and updates its status locally.
func (s *WhatsAppFlowsService) PublishFlow(ctx context.Context, tenantID, channelID, flowID string) error {
	flow, err := s.ensureFlowTenant(ctx, tenantID, flowID)
	if err != nil {
		return err
	}
	if flow.MetaFlowID == "" {
		return errors.New(errors.ErrCodeValidation, "flow is not synced to Meta")
	}

	client, err := s.createFlowClient(ctx, tenantID, channelID)
	if err != nil {
		return err
	}

	if err := client.PublishFlow(ctx, flow.MetaFlowID); err != nil {
		return fmt.Errorf("failed to publish flow on Meta: %w", err)
	}

	if err := s.flowRepo.UpdateStatus(ctx, flowID, true); err != nil {
		return fmt.Errorf("failed to update flow status in repository: %w", err)
	}

	return nil
}

// GetFlowPreview retrieves the preview URL for a WhatsApp Flow from Meta.
func (s *WhatsAppFlowsService) GetFlowPreview(ctx context.Context, tenantID, channelID, flowID string) (string, error) {
	flow, err := s.ensureFlowTenant(ctx, tenantID, flowID)
	if err != nil {
		return "", err
	}
	if flow.MetaFlowID == "" {
		return "", errors.New(errors.ErrCodeValidation, "flow is not synced to Meta")
	}

	client, err := s.createFlowClient(ctx, tenantID, channelID)
	if err != nil {
		return "", err
	}

	previewURL, err := client.GetFlowPreviewURL(ctx, flow.MetaFlowID)
	if err != nil {
		return "", fmt.Errorf("failed to get flow preview URL: %w", err)
	}

	return previewURL, nil
}

// SyncFlowsFromMeta lists all flows from the Meta API and syncs them to the local repository.
func (s *WhatsAppFlowsService) SyncFlowsFromMeta(ctx context.Context, tenantID, channelID string) ([]*entity.Flow, error) {
	client, err := s.createFlowClient(ctx, tenantID, channelID)
	if err != nil {
		return nil, err
	}

	metaFlows, err := client.ListFlows(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list flows from Meta: %w", err)
	}

	var synced []*entity.Flow
	now := time.Now()

	for _, mf := range metaFlows {
		isActive := mf.Status == flows.FlowStatusPublished

		flow := &entity.Flow{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Name:        mf.Name,
			Description: fmt.Sprintf("Synced from Meta (Status: %s)", mf.Status),
			MetaFlowID:  mf.ID,
			Trigger:     entity.FlowTriggerManual,
			Nodes:       []entity.FlowNode{},
			IsActive:    isActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := s.flowRepo.Create(ctx, flow); err != nil {
			return nil, fmt.Errorf("failed to save synced flow %s: %w", mf.Name, err)
		}

		synced = append(synced, flow)
	}

	return synced, nil
}
