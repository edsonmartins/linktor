package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// TemplateService handles template operations
type TemplateService struct {
	templateRepo repository.TemplateRepository
	channelRepo  repository.ChannelRepository
	httpClient   *http.Client
}

// NewTemplateService creates a new template service
func NewTemplateService(
	templateRepo repository.TemplateRepository,
	channelRepo repository.ChannelRepository,
) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		channelRepo:  channelRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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

// Create creates a new template (locally and syncs to Meta if credentials available)
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

	// Try to create template on Meta if credentials are available
	creds := s.getChannelCredentials(ctx, input.ChannelID)
	if creds != nil {
		externalID, err := s.createTemplateOnMeta(ctx, creds, template)
		if err == nil && externalID != "" {
			template.ExternalID = externalID
		}
	}

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

// Delete deletes a template (locally and from Meta if credentials available)
func (s *TemplateService) Delete(ctx context.Context, id string) error {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Try to delete from Meta
	if template.ExternalID != "" {
		creds := s.getChannelCredentials(ctx, template.ChannelID)
		if creds != nil {
			s.deleteTemplateOnMeta(ctx, creds, template.Name)
		}
	}

	return s.templateRepo.Delete(ctx, template.ID)
}

// SyncFromMeta syncs templates from Meta for a channel
func (s *TemplateService) SyncFromMeta(ctx context.Context, channelID string) error {
	creds := s.getChannelCredentials(ctx, channelID)
	if creds == nil {
		return fmt.Errorf("channel not found or missing credentials")
	}

	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	// Fetch templates from Meta
	metaTemplates, err := s.listTemplatesFromMeta(ctx, creds)
	if err != nil {
		return fmt.Errorf("failed to fetch templates from Meta: %w", err)
	}

	// Upsert each template
	for _, mt := range metaTemplates {
		existing, _ := s.templateRepo.FindByExternalID(ctx, mt.ID)
		if existing != nil {
			existing.Status = mapMetaStatusToEntity(mt.Status)
			existing.UpdatedAt = time.Now()
			now := time.Now()
			existing.LastSyncedAt = &now
			s.templateRepo.Update(ctx, existing)
		} else {
			now := time.Now()
			tpl := &entity.Template{
				ID:           generateID(),
				TenantID:     channel.TenantID,
				ChannelID:    channelID,
				ExternalID:   mt.ID,
				Name:         mt.Name,
				Language:     mt.Language,
				Category:     entity.TemplateCategory(mt.Category),
				Status:       mapMetaStatusToEntity(mt.Status),
				QualityScore: entity.TemplateQualityUnknown,
				CreatedAt:    now,
				UpdatedAt:    now,
				LastSyncedAt: &now,
			}
			s.templateRepo.Create(ctx, tpl)
		}
	}

	return nil
}

// SyncToMeta syncs a local template to Meta
func (s *TemplateService) SyncToMeta(ctx context.Context, id string) error {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	creds := s.getChannelCredentials(ctx, template.ChannelID)
	if creds != nil && template.ExternalID == "" {
		externalID, err := s.createTemplateOnMeta(ctx, creds, template)
		if err == nil && externalID != "" {
			template.ExternalID = externalID
			template.Status = entity.TemplateStatusPending
		}
	}

	now := time.Now()
	template.LastSyncedAt = &now
	return s.templateRepo.Update(ctx, template)
}

// ProcessTemplateStatusWebhook processes a template status webhook event
func (s *TemplateService) ProcessTemplateStatusWebhook(ctx context.Context, event *TemplateStatusEvent) error {
	template, err := s.templateRepo.FindByExternalID(ctx, fmt.Sprintf("%d", event.TemplateID))
	if err != nil {
		return nil
	}

	status := mapMetaStatusToEntity(event.Event)
	template.UpdateStatus(status, event.Reason)
	return s.templateRepo.Update(ctx, template)
}

// ProcessTemplateQualityWebhook processes a template quality webhook event
func (s *TemplateService) ProcessTemplateQualityWebhook(ctx context.Context, event *TemplateQualityEvent) error {
	template, err := s.templateRepo.FindByExternalID(ctx, fmt.Sprintf("%d", event.TemplateID))
	if err != nil {
		return nil
	}

	quality := mapMetaQualityToEntity(event.NewQuality)
	template.UpdateQuality(quality)
	return s.templateRepo.Update(ctx, template)
}

// ProcessTemplateCategoryWebhook processes a template category webhook event
func (s *TemplateService) ProcessTemplateCategoryWebhook(ctx context.Context, event *TemplateCategoryEvent) error {
	template, err := s.templateRepo.FindByExternalID(ctx, fmt.Sprintf("%d", event.TemplateID))
	if err != nil {
		return nil
	}

	template.Category = mapMetaCategoryToEntity(event.NewCategory)
	template.UpdatedAt = time.Now()
	return s.templateRepo.Update(ctx, template)
}

// --- Meta Graph API integration ---

type metaCredentials struct {
	accessToken string
	wabaID      string
}

func (s *TemplateService) getChannelCredentials(ctx context.Context, channelID string) *metaCredentials {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil
	}

	accessToken := firstNonEmpty(
		channel.Credentials["access_token"],
		channel.Config["access_token"],
	)
	wabaID := firstNonEmpty(
		channel.WABAID,
		channel.Credentials["waba_id"],
		channel.Config["waba_id"],
		channel.Config["business_id"],
		channel.Credentials["business_id"],
	)
	if accessToken == "" || wabaID == "" {
		return nil
	}

	return &metaCredentials{
		accessToken: accessToken,
		wabaID:      wabaID,
	}
}

func (s *TemplateService) createTemplateOnMeta(ctx context.Context, creds *metaCredentials, template *entity.Template) (string, error) {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates", whatsappofficial.DefaultAPIVersion, creds.wabaID)

	payload := map[string]interface{}{
		"name":     template.Name,
		"language": template.Language,
		"category": string(template.Category),
	}

	if len(template.Components) > 0 {
		payload["components"] = template.Components
	}

	respBody, err := s.metaRequest(ctx, "POST", url, creds.accessToken, payload)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ID, nil
}

func (s *TemplateService) deleteTemplateOnMeta(ctx context.Context, creds *metaCredentials, templateName string) error {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates?name=%s", whatsappofficial.DefaultAPIVersion, creds.wabaID, templateName)
	_, err := s.metaRequest(ctx, "DELETE", url, creds.accessToken, nil)
	return err
}

type metaTemplateInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

func (s *TemplateService) listTemplatesFromMeta(ctx context.Context, creds *metaCredentials) ([]metaTemplateInfo, error) {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates?limit=250", whatsappofficial.DefaultAPIVersion, creds.wabaID)

	respBody, err := s.metaRequest(ctx, "GET", url, creds.accessToken, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []metaTemplateInfo `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

func (s *TemplateService) metaRequest(ctx context.Context, method, url, accessToken string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("Meta API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
	}

	return respBody, nil
}

// --- Webhook event types ---

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

// TemplateCategoryEvent represents a template category webhook event
type TemplateCategoryEvent struct {
	TemplateID       int64
	TemplateName     string
	Language         string
	PreviousCategory string
	NewCategory      string
}

// --- Status/Quality mapping ---

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

func mapMetaCategoryToEntity(category string) entity.TemplateCategory {
	switch category {
	case "AUTHENTICATION":
		return entity.TemplateCategoryAuthentication
	case "MARKETING":
		return entity.TemplateCategoryMarketing
	case "UTILITY":
		return entity.TemplateCategoryUtility
	default:
		return entity.TemplateCategoryUtility
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func generateID() string {
	return uuid.New().String()
}
