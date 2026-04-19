package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/graphapi"
)

// TemplateService handles template operations
type TemplateService struct {
	templateRepo repository.TemplateRepository
	channelRepo  repository.ChannelRepository
	httpClient   *http.Client
	writeLimiter *templateWriteLimiter
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
		// Meta's hard ceiling on WABA template writes is ~100/hour. We
		// shave a bit off the top so concurrent callers don't race into
		// an 80008 response.
		writeLimiter: newTemplateWriteLimiter(80, time.Hour),
	}
}

// CreateTemplateInput represents input for creating a template
type CreateTemplateInput struct {
	TenantID              string
	ChannelID             string
	Name                  string
	Language              string
	Category              entity.TemplateCategory
	SubCategory           string
	ParameterFormat       entity.TemplateParameterFormat
	MessageSendTTLSeconds int
	AllowCategoryChange   bool
	Components            []entity.TemplateComponent
}

// Create creates a new template (locally and syncs to Meta if credentials available)
func (s *TemplateService) Create(ctx context.Context, input *CreateTemplateInput) (*entity.Template, error) {
	// Validate components locally before spending a Graph API round-trip on
	// something Meta would reject anyway. This catches the most common
	// authoring mistake: declaring {{N}} variables without attaching
	// example values.
	if err := validateTemplateComponents(input.Components); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}
	if err := validateParameterFormat(input.ParameterFormat, input.Components); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Check if template already exists
	existing, _ := s.templateRepo.FindByName(ctx, input.TenantID, input.ChannelID, input.Name, input.Language)
	if existing != nil {
		return nil, fmt.Errorf("template with name '%s' and language '%s' already exists", input.Name, input.Language)
	}

	template := &entity.Template{
		ID:                    generateID(),
		TenantID:              input.TenantID,
		ChannelID:             input.ChannelID,
		Name:                  input.Name,
		Language:              input.Language,
		Category:              input.Category,
		SubCategory:           input.SubCategory,
		ParameterFormat:       input.ParameterFormat,
		MessageSendTTLSeconds: input.MessageSendTTLSeconds,
		AllowCategoryChange:   input.AllowCategoryChange,
		Status:                entity.TemplateStatusPending,
		QualityScore:          entity.TemplateQualityUnknown,
		Components:            input.Components,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Try to create template on Meta if credentials are available.
	creds := s.getChannelCredentials(ctx, input.ChannelID)
	if creds != nil {
		if err := s.writeLimiter.Allow(creds.wabaID); err != nil {
			return nil, err
		}
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

// EditTemplateInput carries the fields Meta will accept on a template edit.
// Components and Category are the two Meta actually allows to change on
// already-approved templates; MessageSendTTLSeconds is also editable.
// Other fields (name, language, parameter_format) are immutable once a
// template has been submitted — changing them requires delete + recreate.
//
// TenantID is the caller's tenant (extracted from the auth middleware).
// We verify it against the stored template's TenantID to prevent one
// tenant from editing another's templates via UUID guessing.
type EditTemplateInput struct {
	TenantID              string
	ID                    string
	Category              entity.TemplateCategory
	Components            []entity.TemplateComponent
	MessageSendTTLSeconds int
}

// Edit updates an existing template on Meta and syncs the local copy.
// After a successful edit Meta resets the status to PENDING regardless of
// the previous state, so we mirror that here.
func (s *TemplateService) Edit(ctx context.Context, input *EditTemplateInput) (*entity.Template, error) {
	if err := validateTemplateComponents(input.Components); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	template, err := s.templateRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	if input.TenantID != "" && template.TenantID != input.TenantID {
		return nil, fmt.Errorf("template not found: %s", input.ID)
	}
	if template.ExternalID == "" {
		return nil, fmt.Errorf("template must be synced with Meta before editing")
	}

	// Apply edits locally so the same components make the round-trip.
	if input.Category != "" {
		template.Category = input.Category
	}
	if len(input.Components) > 0 {
		template.Components = input.Components
	}
	if input.MessageSendTTLSeconds > 0 {
		template.MessageSendTTLSeconds = input.MessageSendTTLSeconds
	}
	if err := validateParameterFormat(template.ParameterFormat, template.Components); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	creds := s.getChannelCredentials(ctx, template.ChannelID)
	if creds == nil {
		return nil, fmt.Errorf("channel missing credentials")
	}

	if err := s.writeLimiter.Allow(creds.wabaID); err != nil {
		return nil, err
	}
	if err := s.editTemplateOnMeta(ctx, creds, template); err != nil {
		return nil, fmt.Errorf("failed to edit template on Meta: %w", err)
	}

	template.Status = entity.TemplateStatusPending
	template.UpdatedAt = time.Now()
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save edited template: %w", err)
	}
	return template, nil
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
// DeleteBulk removes multiple templates in a single Meta call. All templates
// must belong to the same channel (so we can resolve credentials once);
// mixing channels returns an error rather than silently partitioning.
//
// tenantID (if non-empty) scopes the lookup so one tenant can't bulk-delete
// another tenant's templates via UUID enumeration.
func (s *TemplateService) DeleteBulk(ctx context.Context, tenantID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Resolve all templates first so we can fail fast before touching Meta.
	templates := make([]*entity.Template, 0, len(ids))
	var channelID string
	for _, id := range ids {
		tpl, err := s.templateRepo.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("template %s not found: %w", id, err)
		}
		if tenantID != "" && tpl.TenantID != tenantID {
			return fmt.Errorf("template %s not found", id)
		}
		if channelID == "" {
			channelID = tpl.ChannelID
		} else if tpl.ChannelID != channelID {
			return fmt.Errorf("bulk delete requires all templates to share the same channel (got %s and %s)", channelID, tpl.ChannelID)
		}
		templates = append(templates, tpl)
	}

	// Collect external IDs and hit Meta once. Templates that were never
	// synced (no ExternalID) are dropped from the Meta call but still
	// removed locally.
	var hsmIDs []string
	for _, tpl := range templates {
		if tpl.ExternalID != "" {
			hsmIDs = append(hsmIDs, tpl.ExternalID)
		}
	}
	if len(hsmIDs) > 0 {
		creds := s.getChannelCredentials(ctx, channelID)
		if creds != nil {
			if err := s.deleteTemplatesBulkOnMeta(ctx, creds, hsmIDs); err != nil {
				return fmt.Errorf("bulk delete on Meta failed: %w", err)
			}
		}
	}

	// Remove local rows even if Meta was skipped (no creds or no external IDs).
	for _, tpl := range templates {
		if err := s.templateRepo.Delete(ctx, tpl.ID); err != nil {
			return fmt.Errorf("failed to delete %s locally: %w", tpl.ID, err)
		}
	}
	return nil
}

func (s *TemplateService) Delete(ctx context.Context, id string) error {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Try to delete from Meta. Pass the external (hsm) id so Meta only removes
	// this specific language variant — without hsm_id, Meta deletes every
	// language variant that shares the template name, which is almost never
	// what the caller wants.
	if template.ExternalID != "" {
		creds := s.getChannelCredentials(ctx, template.ChannelID)
		if creds != nil {
			s.deleteTemplateOnMeta(ctx, creds, template.Name, template.ExternalID)
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
	url := fmt.Sprintf("%s/%s/%s/message_templates", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, creds.wabaID)

	payload := map[string]interface{}{
		"name":     template.Name,
		"language": template.Language,
		"category": string(template.Category),
	}

	if len(template.Components) > 0 {
		payload["components"] = template.Components
	}
	if template.SubCategory != "" {
		payload["sub_category"] = template.SubCategory
	}
	if template.ParameterFormat != "" {
		payload["parameter_format"] = string(template.ParameterFormat)
	}
	if template.MessageSendTTLSeconds > 0 {
		payload["message_send_ttl_seconds"] = template.MessageSendTTLSeconds
	}
	if template.AllowCategoryChange {
		payload["allow_category_change"] = true
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

// deleteTemplateOnMeta removes a template from the WABA. When hsmID is
// non-empty, Meta deletes only the specific language variant identified by
// that id; when it's empty, Meta deletes every language variant that shares
// the template name.
func (s *TemplateService) deleteTemplateOnMeta(ctx context.Context, creds *metaCredentials, templateName, hsmID string) error {
	url := fmt.Sprintf("%s/%s/%s/message_templates?name=%s", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, creds.wabaID, templateName)
	if hsmID != "" {
		url += "&hsm_id=" + hsmID
	}
	_, err := s.metaRequest(ctx, "DELETE", url, creds.accessToken, nil)
	return err
}

// deleteTemplatesBulkOnMeta sends a single DELETE with an hsm_ids[] so
// Meta removes every listed template in one call. Useful for the admin
// "delete N selected templates" UI flow — avoids N round-trips and
// respects Meta's per-WABA rate limits better.
func (s *TemplateService) deleteTemplatesBulkOnMeta(ctx context.Context, creds *metaCredentials, hsmIDs []string) error {
	if len(hsmIDs) == 0 {
		return nil
	}
	// Meta accepts hsm_ids as a JSON array in the query string. We marshal
	// to JSON first (to get the `["id1","id2"]` shape) and then percent-
	// encode the whole value so brackets / quotes survive any middlebox
	// normalisation intact.
	ids, err := json.Marshal(hsmIDs)
	if err != nil {
		return fmt.Errorf("failed to encode hsm_ids: %w", err)
	}
	params := url.Values{}
	params.Set("hsm_ids", string(ids))
	fullURL := fmt.Sprintf("%s/%s/%s/message_templates?%s",
		graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, creds.wabaID, params.Encode())
	_, err = s.metaRequest(ctx, "DELETE", fullURL, creds.accessToken, nil)
	return err
}

type metaTemplateInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

// editTemplateOnMeta updates an existing template. Meta allows editing
// components, category, and a handful of other fields on templates that
// are APPROVED, IN_APPEAL or REJECTED. Status resets to PENDING after edit.
func (s *TemplateService) editTemplateOnMeta(ctx context.Context, creds *metaCredentials, template *entity.Template) error {
	url := fmt.Sprintf("%s/%s/%s", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, template.ExternalID)

	payload := map[string]interface{}{}
	if len(template.Components) > 0 {
		payload["components"] = template.Components
	}
	if template.Category != "" {
		payload["category"] = string(template.Category)
	}
	if template.MessageSendTTLSeconds > 0 {
		payload["message_send_ttl_seconds"] = template.MessageSendTTLSeconds
	}

	_, err := s.metaRequest(ctx, "POST", url, creds.accessToken, payload)
	return err
}

// FetchNamespace reads the WABA's message_template_namespace. Callers that
// pass forceRefresh=false get the value persisted on the channel when
// available, avoiding a Graph API round-trip. Passing forceRefresh=true
// (or calling on a channel that never cached the value) hits Meta's
// GET /{waba-id}?fields=message_template_namespace endpoint.
//
// tenantID (if non-empty) scopes the channel lookup so one tenant can't
// fetch another tenant's namespace.
func (s *TemplateService) FetchNamespace(ctx context.Context, tenantID, channelID string, forceRefresh bool) (string, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return "", fmt.Errorf("channel not found: %w", err)
	}
	if tenantID != "" && channel.TenantID != tenantID {
		return "", fmt.Errorf("channel not found: %s", channelID)
	}

	// Serve from the cached field unless the caller wants a fresh read.
	if !forceRefresh && channel.MessageTemplateNamespace != "" {
		return channel.MessageTemplateNamespace, nil
	}

	creds := s.getChannelCredentials(ctx, channelID)
	if creds == nil {
		return "", fmt.Errorf("channel missing credentials")
	}

	url := fmt.Sprintf("%s/%s/%s?fields=message_template_namespace", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, creds.wabaID)
	respBody, err := s.metaRequest(ctx, "GET", url, creds.accessToken, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		MessageTemplateNamespace string `json:"message_template_namespace"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse namespace response: %w", err)
	}

	// Persist on the channel so the fast path above can serve future reads.
	if result.MessageTemplateNamespace != "" {
		channel.MessageTemplateNamespace = result.MessageTemplateNamespace
		if err := s.channelRepo.Update(ctx, channel); err != nil {
			// Non-fatal: we still return the value the caller asked for.
			// Worst case next call re-fetches from Meta.
			_ = err
		}
	}

	return result.MessageTemplateNamespace, nil
}

// LibraryTemplate represents a pre-built template Meta exposes via the
// message_template_library endpoint. Each entry can be instantiated on a
// WABA by name — Meta handles approval because the wording is pre-vetted.
type LibraryTemplate struct {
	Name       string                   `json:"name"`
	Category   string                   `json:"category"`
	Language   string                   `json:"language,omitempty"`
	Industry   []string                 `json:"industry,omitempty"`
	Topic      string                   `json:"topic,omitempty"`
	Usecase    string                   `json:"usecase,omitempty"`
	BodyText   string                   `json:"body_text,omitempty"`
	HeaderText string                   `json:"header_text,omitempty"`
	Buttons    []map[string]interface{} `json:"buttons,omitempty"`
}

// LibraryQuery narrows the template library listing via Meta's filter
// parameters. All fields are optional; zero value returns the full catalog.
type LibraryQuery struct {
	Search   string // free-text match against template content/name
	Topic    string // e.g. "PAYMENT_REMINDER"
	Usecase  string
	Industry string
	Language string
}

// ListTemplateLibrary fetches Meta's pre-built template catalog. The list
// is the same regardless of WABA but we still take a channelID so we can
// authenticate with the right access token.
func (s *TemplateService) ListTemplateLibrary(ctx context.Context, channelID string, query LibraryQuery) ([]LibraryTemplate, error) {
	creds := s.getChannelCredentials(ctx, channelID)
	if creds == nil {
		return nil, fmt.Errorf("channel missing credentials")
	}

	// url.Values handles percent-encoding so a query like `Search="A&B"`
	// or `Search="hello world"` doesn't inject spurious params / break
	// the URL. Previously we concatenated raw values which was a
	// correctness + mild injection bug.
	params := url.Values{}
	if query.Search != "" {
		params.Set("search", query.Search)
	}
	if query.Topic != "" {
		params.Set("topic", query.Topic)
	}
	if query.Usecase != "" {
		params.Set("usecase", query.Usecase)
	}
	if query.Industry != "" {
		params.Set("industry", query.Industry)
	}
	if query.Language != "" {
		params.Set("language", query.Language)
	}
	suffix := ""
	if encoded := params.Encode(); encoded != "" {
		suffix = "?" + encoded
	}

	fullURL := fmt.Sprintf("%s/%s/message_template_library%s", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, suffix)
	respBody, err := s.metaRequest(ctx, "GET", fullURL, creds.accessToken, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []LibraryTemplate `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse library response: %w", err)
	}
	return result.Data, nil
}

// CreateFromLibraryInput carries the payload for creating a template from
// Meta's library. library_template_name is required; body/button inputs
// customise the optional placeholders the library exposes.
type CreateFromLibraryInput struct {
	TenantID                  string
	ChannelID                 string
	Name                      string
	Language                  string
	Category                  entity.TemplateCategory
	LibraryTemplateName       string
	LibraryTemplateBodyInputs map[string]interface{}
	LibraryTemplateButtonInputs []map[string]interface{}
}

// CreateFromLibrary instantiates a library template on the channel's WABA.
// The local row is persisted the same way as a hand-crafted template —
// subsequent sync/refresh/edit calls work identically.
func (s *TemplateService) CreateFromLibrary(ctx context.Context, input *CreateFromLibraryInput) (*entity.Template, error) {
	if input.LibraryTemplateName == "" {
		return nil, fmt.Errorf("library_template_name is required")
	}

	existing, _ := s.templateRepo.FindByName(ctx, input.TenantID, input.ChannelID, input.Name, input.Language)
	if existing != nil {
		return nil, fmt.Errorf("template with name '%s' and language '%s' already exists", input.Name, input.Language)
	}

	creds := s.getChannelCredentials(ctx, input.ChannelID)
	if creds == nil {
		return nil, fmt.Errorf("channel missing credentials")
	}
	if err := s.writeLimiter.Allow(creds.wabaID); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"name":                  input.Name,
		"language":              input.Language,
		"category":              string(input.Category),
		"library_template_name": input.LibraryTemplateName,
	}
	if len(input.LibraryTemplateBodyInputs) > 0 {
		payload["library_template_body_inputs"] = input.LibraryTemplateBodyInputs
	}
	if len(input.LibraryTemplateButtonInputs) > 0 {
		payload["library_template_button_inputs"] = input.LibraryTemplateButtonInputs
	}

	url := fmt.Sprintf("%s/%s/%s/message_templates", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, creds.wabaID)
	respBody, err := s.metaRequest(ctx, "POST", url, creds.accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create template from library: %w", err)
	}

	var result struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse library create response: %w", err)
	}

	template := &entity.Template{
		ID:           generateID(),
		TenantID:     input.TenantID,
		ChannelID:    input.ChannelID,
		ExternalID:   result.ID,
		Name:         input.Name,
		Language:     input.Language,
		Category:     input.Category,
		Status:       mapMetaStatusToEntity(result.Status),
		QualityScore: entity.TemplateQualityUnknown,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if template.Status == "" {
		template.Status = entity.TemplateStatusPending
	}
	if result.Category != "" {
		template.Category = entity.TemplateCategory(result.Category)
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}
	return template, nil
}

// getTemplateFromMeta fetches a single template by its Meta ID (hsm_id).
// Useful when the caller has a stale local copy and wants to refresh just
// that variant without paying for a 250-template list call.
func (s *TemplateService) getTemplateFromMeta(ctx context.Context, creds *metaCredentials, templateID string) (*metaTemplateInfo, error) {
	url := fmt.Sprintf("%s/%s/%s?fields=id,name,language,status,category", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, templateID)

	respBody, err := s.metaRequest(ctx, "GET", url, creds.accessToken, nil)
	if err != nil {
		return nil, err
	}

	var info metaTemplateInfo
	if err := json.Unmarshal(respBody, &info); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &info, nil
}

// RefreshFromMeta pulls the latest state of a single template from Meta
// and syncs the local row. Returns the updated template. Callers typically
// use this after a webhook hints that a specific template changed, or when
// an admin triggers a manual refresh from the UI.
//
// tenantID (if non-empty) scopes the lookup so one tenant can't refresh
// another tenant's template via UUID guessing.
func (s *TemplateService) RefreshFromMeta(ctx context.Context, tenantID, id string) (*entity.Template, error) {
	template, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	if tenantID != "" && template.TenantID != tenantID {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	if template.ExternalID == "" {
		return nil, fmt.Errorf("template has no external_id to refresh from")
	}

	creds := s.getChannelCredentials(ctx, template.ChannelID)
	if creds == nil {
		return nil, fmt.Errorf("channel missing credentials")
	}

	info, err := s.getTemplateFromMeta(ctx, creds, template.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Meta: %w", err)
	}

	template.Status = mapMetaStatusToEntity(info.Status)
	template.Category = entity.TemplateCategory(info.Category)
	template.Language = info.Language
	template.UpdatedAt = time.Now()
	now := time.Now()
	template.LastSyncedAt = &now

	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save refreshed template: %w", err)
	}
	return template, nil
}

func (s *TemplateService) listTemplatesFromMeta(ctx context.Context, creds *metaCredentials) ([]metaTemplateInfo, error) {
	url := fmt.Sprintf("%s/%s/%s/message_templates?limit=250", graphapi.BaseURL(), whatsappofficial.DefaultAPIVersion, creds.wabaID)

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
	case "LIMIT_EXCEEDED":
		return entity.TemplateStatusLimitExceeded
	case "ARCHIVED":
		return entity.TemplateStatusArchived
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
