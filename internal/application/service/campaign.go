package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
)

// dispatchBatchSize bounds how many recipients are loaded per dispatch loop.
const dispatchBatchSize = 200

// RecipientInput is one target supplied at campaign creation.
type RecipientInput struct {
	Phone     string
	ContactID string
	Params    []string
}

// CampaignInput carries fields to create a campaign.
type CampaignInput struct {
	Name             string
	ChannelID        string
	TemplateName     string
	TemplateLanguage string
	Recipients       []RecipientInput
}

// CampaignService manages bulk template campaigns. Delivery is dispatched over
// NATS through the existing outbound pipeline (one OutboundMessage per recipient).
type CampaignService struct {
	repo        repository.CampaignRepository
	channelRepo repository.ChannelRepository
	producer    nats.Publisher

	mu     sync.Mutex      // guards active
	active map[string]bool // campaign IDs currently being dispatched
}

// NewCampaignService creates a new CampaignService.
func NewCampaignService(repo repository.CampaignRepository, channelRepo repository.ChannelRepository, producer nats.Publisher) *CampaignService {
	return &CampaignService{
		repo:        repo,
		channelRepo: channelRepo,
		producer:    producer,
		active:      make(map[string]bool),
	}
}

// tryAcquire marks a campaign as actively dispatching. It returns false if a
// dispatch is already running for that campaign, preventing concurrent
// duplicate sends (e.g. Start racing RetryFailed).
func (s *CampaignService) tryAcquire(campaignID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active[campaignID] {
		return false
	}
	s.active[campaignID] = true
	return true
}

func (s *CampaignService) release(campaignID string) {
	s.mu.Lock()
	delete(s.active, campaignID)
	s.mu.Unlock()
}

// Create validates and persists a draft campaign with its recipients.
func (s *CampaignService) Create(ctx context.Context, tenantID, createdBy string, in *CampaignInput) (*entity.Campaign, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.Validation("name is required")
	}
	if in.TemplateName == "" {
		return nil, errors.Validation("template_name is required")
	}
	if len(in.Recipients) == 0 {
		return nil, errors.Validation("at least one recipient is required")
	}

	channel, err := s.channelRepo.FindByID(ctx, in.ChannelID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}
	if channel.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeChannelNotFound, "channel not found")
	}

	lang := in.TemplateLanguage
	if lang == "" {
		lang = "pt_BR"
	}

	campaign := entity.NewCampaign(tenantID, in.ChannelID, in.Name, in.TemplateName, lang)
	campaign.ID = uuid.New().String()
	campaign.CreatedBy = createdBy
	campaign.TotalRecipients = len(in.Recipients)

	if err := s.repo.Create(ctx, campaign); err != nil {
		return nil, err
	}

	now := time.Now()
	recipients := make([]*entity.CampaignRecipient, 0, len(in.Recipients))
	for _, ri := range in.Recipients {
		if strings.TrimSpace(ri.Phone) == "" {
			continue
		}
		recipients = append(recipients, &entity.CampaignRecipient{
			ID:         uuid.New().String(),
			CampaignID: campaign.ID,
			ContactID:  ri.ContactID,
			Phone:      ri.Phone,
			Params:     ri.Params,
			Status:     entity.RecipientPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	if err := s.repo.AddRecipients(ctx, recipients); err != nil {
		return nil, err
	}
	campaign.TotalRecipients = len(recipients)
	_ = s.repo.Update(ctx, campaign)

	return campaign, nil
}

// Get returns a campaign scoped to the tenant.
func (s *CampaignService) Get(ctx context.Context, tenantID, id string) (*entity.Campaign, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "campaign not found")
	}
	return c, nil
}

// List returns campaigns for a tenant.
func (s *CampaignService) List(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Campaign, int64, error) {
	return s.repo.FindByTenant(ctx, tenantID, params)
}

// ListRecipients returns a campaign's recipients.
func (s *CampaignService) ListRecipients(ctx context.Context, tenantID, campaignID string, params *repository.ListParams) ([]*entity.CampaignRecipient, int64, error) {
	if _, err := s.Get(ctx, tenantID, campaignID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListRecipients(ctx, campaignID, params)
}

// Start marks the campaign processing and dispatches pending recipients in the
// background. Returns immediately.
func (s *CampaignService) Start(ctx context.Context, tenantID, campaignID string) (*entity.Campaign, error) {
	campaign, err := s.Get(ctx, tenantID, campaignID)
	if err != nil {
		return nil, err
	}
	if campaign.Status == entity.CampaignStatusProcessing {
		return nil, errors.New(errors.ErrCodeConflict, "campaign is already processing")
	}

	if !s.tryAcquire(campaignID) {
		return nil, errors.New(errors.ErrCodeConflict, "campaign is already processing")
	}

	now := time.Now()
	campaign.Status = entity.CampaignStatusProcessing
	campaign.StartedAt = &now
	if err := s.repo.Update(ctx, campaign); err != nil {
		s.release(campaignID)
		return nil, err
	}

	go s.dispatch(campaign)
	return campaign, nil
}

// RetryFailed requeues failed recipients and resumes dispatch.
func (s *CampaignService) RetryFailed(ctx context.Context, tenantID, campaignID string) (int64, error) {
	campaign, err := s.Get(ctx, tenantID, campaignID)
	if err != nil {
		return 0, err
	}
	count, err := s.repo.ResetFailedRecipients(ctx, campaignID)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		// Skip silently if a dispatch is already running; the reset recipients
		// will be picked up by that in-flight loop.
		if !s.tryAcquire(campaignID) {
			return count, nil
		}
		now := time.Now()
		campaign.Status = entity.CampaignStatusProcessing
		campaign.StartedAt = &now
		_ = s.repo.Update(ctx, campaign)
		go s.dispatch(campaign)
	}
	return count, nil
}

// Cancel stops a campaign (pending recipients are left untouched).
func (s *CampaignService) Cancel(ctx context.Context, tenantID, campaignID string) error {
	campaign, err := s.Get(ctx, tenantID, campaignID)
	if err != nil {
		return err
	}
	campaign.Status = entity.CampaignStatusCancelled
	return s.repo.Update(ctx, campaign)
}

// dispatch runs in the background, publishing one outbound template message per
// pending recipient through NATS, then recomputes counters.
func (s *CampaignService) dispatch(campaign *entity.Campaign) {
	defer s.release(campaign.ID)
	ctx := context.Background()

	channel, err := s.channelRepo.FindByID(ctx, campaign.ChannelID)
	if err != nil {
		logger.Error("campaign dispatch: channel lookup failed: " + err.Error())
		s.finish(ctx, campaign, entity.CampaignStatusFailed)
		return
	}

	// seen guards against infinite loops: if a recipient's status update fails
	// after a successful publish it stays 'pending' and would be re-selected
	// forever. Tracking handled IDs lets us stop once a batch makes no progress.
	seen := make(map[string]bool)

	for {
		// Honor a cancellation that landed mid-dispatch.
		if fresh, err := s.repo.FindByID(ctx, campaign.ID); err == nil && fresh.Status == entity.CampaignStatusCancelled {
			logger.Info("campaign " + campaign.ID + " dispatch stopped: cancelled")
			return
		}

		recipients, err := s.repo.FindPendingRecipients(ctx, campaign.ID, dispatchBatchSize)
		if err != nil {
			logger.Error("campaign dispatch: pending query failed: " + err.Error())
			break
		}
		if len(recipients) == 0 {
			break
		}

		progressed := false
		for _, rec := range recipients {
			if seen[rec.ID] {
				continue // already attempted this run; avoid re-sending in a stuck loop
			}
			seen[rec.ID] = true
			progressed = true

			msg := &nats.OutboundMessage{
				ID:          uuid.New().String(),
				TenantID:    campaign.TenantID,
				ChannelID:   campaign.ChannelID,
				ChannelType: string(channel.Type),
				RecipientID: rec.Phone,
				ContentType: "template",
				Metadata:    s.templateMetadata(campaign, rec),
				Timestamp:   time.Now(),
			}

			if err := s.producer.PublishOutbound(ctx, msg); err != nil {
				_ = s.repo.UpdateRecipientStatus(ctx, rec.ID, entity.RecipientFailed, "", err.Error())
				continue
			}
			_ = s.repo.UpdateRecipientStatus(ctx, rec.ID, entity.RecipientSent, msg.ID, "")
		}

		_ = s.repo.RecountStatuses(ctx, campaign.ID)

		// Every recipient in this batch was already handled this run but is still
		// pending (status writes are failing): stop to avoid a hot re-send loop.
		if !progressed {
			logger.Error("campaign dispatch: no progress, stopping to avoid resend loop for campaign " + campaign.ID)
			break
		}
	}

	s.finish(ctx, campaign, entity.CampaignStatusCompleted)
}

// templateMetadata builds the OutboundMessage metadata the WhatsApp adapter
// reads. Body parameters are encoded as a template_components JSON (a BODY
// component with one text parameter per value) — the adapter ignores Content
// for templates, so params must travel via template_components.
func (s *CampaignService) templateMetadata(campaign *entity.Campaign, rec *entity.CampaignRecipient) map[string]string {
	meta := map[string]string{
		"template_name":         campaign.TemplateName,
		"template_language":     campaign.TemplateLanguage,
		"campaign_id":           campaign.ID,
		"campaign_recipient_id": rec.ID,
	}
	if len(rec.Params) > 0 {
		if components := buildBodyComponents(rec.Params); components != "" {
			meta["template_components"] = components
		}
	}
	return meta
}

// templateTextParameter mirrors the wire shape the WhatsApp adapter unmarshals
// (whatsapp_official.TemplateParameter). Kept local to avoid an inbound→adapter
// layering dependency.
type templateTextParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type templateBodyComponent struct {
	Type       string                  `json:"type"`
	Parameters []templateTextParameter `json:"parameters"`
}

// buildBodyComponents encodes positional body params as the components JSON the
// adapter expects: [{"type":"body","parameters":[{"type":"text","text":"..."}]}].
func buildBodyComponents(params []string) string {
	body := templateBodyComponent{Type: "body"}
	for _, p := range params {
		body.Parameters = append(body.Parameters, templateTextParameter{Type: "text", Text: p})
	}
	raw, err := json.Marshal([]templateBodyComponent{body})
	if err != nil {
		return ""
	}
	return string(raw)
}

func (s *CampaignService) finish(ctx context.Context, campaign *entity.Campaign, status entity.CampaignStatus) {
	_ = s.repo.RecountStatuses(ctx, campaign.ID)
	fresh, err := s.repo.FindByID(ctx, campaign.ID)
	if err != nil {
		return
	}
	if fresh.Status == entity.CampaignStatusCancelled {
		return // respect a cancellation that landed mid-dispatch
	}
	now := time.Now()
	fresh.Status = status
	fresh.CompletedAt = &now
	_ = s.repo.Update(ctx, fresh)
	logger.Info("campaign " + campaign.ID + " finished with status " + string(status))
}
