package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTemplateRepository implements repository.TemplateRepository for testing
type mockTemplateRepository struct {
	Templates   map[string]*entity.Template
	ReturnError error
}

func newMockTemplateRepository() *mockTemplateRepository {
	return &mockTemplateRepository{
		Templates: make(map[string]*entity.Template),
	}
}

func (m *mockTemplateRepository) Create(ctx context.Context, template *entity.Template) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Templates[template.ID] = template
	return nil
}

func (m *mockTemplateRepository) FindByID(ctx context.Context, id string) (*entity.Template, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	t, ok := m.Templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return t, nil
}

func (m *mockTemplateRepository) FindByExternalID(ctx context.Context, externalID string) (*entity.Template, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, t := range m.Templates {
		if t.ExternalID == externalID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template not found by external ID: %s", externalID)
}

func (m *mockTemplateRepository) FindByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, t := range m.Templates {
		if t.TenantID == tenantID && t.ChannelID == channelID && t.Name == name && t.Language == language {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template not found: %s/%s", name, language)
}

func (m *mockTemplateRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Template
	for _, t := range m.Templates {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockTemplateRepository) FindByChannel(ctx context.Context, channelID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Template
	for _, t := range m.Templates {
		if t.ChannelID == channelID {
			result = append(result, t)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockTemplateRepository) FindByStatus(ctx context.Context, tenantID string, status entity.TemplateStatus, params *repository.ListParams) ([]*entity.Template, int64, error) {
	return nil, 0, nil
}

func (m *mockTemplateRepository) FindNeedsSync(ctx context.Context, channelID string, syncInterval int64) ([]*entity.Template, error) {
	return nil, nil
}

func (m *mockTemplateRepository) Update(ctx context.Context, template *entity.Template) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Templates[template.ID] = template
	return nil
}

func (m *mockTemplateRepository) UpdateStatus(ctx context.Context, id string, status entity.TemplateStatus, reason string) error {
	return nil
}

func (m *mockTemplateRepository) UpdateQuality(ctx context.Context, id string, quality entity.TemplateQuality) error {
	return nil
}

func (m *mockTemplateRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Templates, id)
	return nil
}

func (m *mockTemplateRepository) DeleteByExternalID(ctx context.Context, externalID string) error {
	return nil
}

func (m *mockTemplateRepository) CountByChannel(ctx context.Context, channelID string) (int64, error) {
	return 0, nil
}

func (m *mockTemplateRepository) UpsertByExternalID(ctx context.Context, template *entity.Template) error {
	return nil
}

func setupTemplateService() (*TemplateService, *mockTemplateRepository) {
	templateRepo := newMockTemplateRepository()
	channelRepo := testutil.NewMockChannelRepository()
	svc := NewTemplateService(templateRepo, channelRepo)
	return svc, templateRepo
}

func TestTemplateService_Create(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	template, err := svc.Create(context.Background(), &CreateTemplateInput{
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
		Category:  entity.TemplateCategoryMarketing,
		Components: []entity.TemplateComponent{
			{
				Type: "BODY",
				Text: "Hello {{1}}!",
				Example: &entity.TemplateExample{
					BodyText: [][]string{{"Ana"}},
				},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, template)
	assert.NotEmpty(t, template.ID)
	assert.Equal(t, "welcome", template.Name)
	assert.Equal(t, "en", template.Language)
	assert.Equal(t, entity.TemplateStatusPending, template.Status)
	assert.Len(t, templateRepo.Templates, 1)
}

func TestTemplateService_Create_RejectsVariablesWithoutExample(t *testing.T) {
	// Meta rejects this payload anyway — validating locally saves a
	// round-trip and gives the admin a clear error instead of 400 from Meta.
	svc, _ := setupTemplateService()

	_, err := svc.Create(context.Background(), &CreateTemplateInput{
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
		Category:  entity.TemplateCategoryMarketing,
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Hello {{1}}!"}, // no example — must fail
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
	assert.Contains(t, err.Error(), "example")
}

func TestTemplateService_Create_Duplicate(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["existing"] = &entity.Template{
		ID:        "existing",
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
	}

	_, err := svc.Create(context.Background(), &CreateTemplateInput{
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestTemplateService_GetByID(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["t1"] = &entity.Template{
		ID:   "t1",
		Name: "welcome",
	}

	template, err := svc.GetByID(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "welcome", template.Name)
}

func TestTemplateService_GetByID_NotFound(t *testing.T) {
	svc, _ := setupTemplateService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestTemplateService_List(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["t1"] = &entity.Template{ID: "t1", TenantID: "tenant-1", Name: "template1"}
	templateRepo.Templates["t2"] = &entity.Template{ID: "t2", TenantID: "tenant-1", Name: "template2"}
	templateRepo.Templates["t3"] = &entity.Template{ID: "t3", TenantID: "other", Name: "template3"}

	templates, count, err := svc.List(context.Background(), "tenant-1", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, templates, 2)
}

func TestTemplateService_ListByChannel(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["t1"] = &entity.Template{ID: "t1", ChannelID: "ch-1", Name: "t1"}
	templateRepo.Templates["t2"] = &entity.Template{ID: "t2", ChannelID: "ch-1", Name: "t2"}
	templateRepo.Templates["t3"] = &entity.Template{ID: "t3", ChannelID: "ch-2", Name: "t3"}

	templates, count, err := svc.ListByChannel(context.Background(), "ch-1", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, templates, 2)
}

func TestTemplateService_UpdateStatus(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["t1"] = &entity.Template{
		ID:         "t1",
		ExternalID: "ext-1",
		Status:     entity.TemplateStatusPending,
	}

	err := svc.UpdateStatus(context.Background(), "ext-1", entity.TemplateStatusApproved, "")
	require.NoError(t, err)

	// Verify updated
	assert.Equal(t, entity.TemplateStatusApproved, templateRepo.Templates["t1"].Status)
}

func TestTemplateService_UpdateStatus_NotFound(t *testing.T) {
	svc, _ := setupTemplateService()

	err := svc.UpdateStatus(context.Background(), "nonexistent", entity.TemplateStatusApproved, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestTemplateService_GetByName(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["t1"] = &entity.Template{
		ID:        "t1",
		TenantID:  "tenant-1",
		ChannelID: "ch-1",
		Name:      "welcome",
		Language:  "en",
	}

	template, err := svc.GetByName(context.Background(), "tenant-1", "ch-1", "welcome", "en")
	require.NoError(t, err)
	assert.Equal(t, "welcome", template.Name)
}

func TestTemplateService_SyncFromMeta_UsesChannelCredentialsFallback(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID:       "ch-1",
		TenantID: "tenant-1",
		Type:     entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{
			"access_token": "test-token",
			"waba_id":      "waba-123",
		},
	}

	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "/"+whatsappofficial.DefaultAPIVersion+"/waba-123/message_templates", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"data":[{"id":"tpl-1","name":"welcome","language":"pt_BR","status":"APPROVED","category":"UTILITY"}]}`,
			)),
		}, nil
	})

	err := svc.SyncFromMeta(context.Background(), "ch-1")
	require.NoError(t, err)

	require.Len(t, templateRepo.Templates, 1)
	for _, template := range templateRepo.Templates {
		assert.Equal(t, "tpl-1", template.ExternalID)
		assert.Equal(t, entity.TemplateStatusApproved, template.Status)
		assert.Equal(t, entity.TemplateCategoryUtility, template.Category)
	}
}

func TestTemplateService_ProcessTemplateCategoryWebhook(t *testing.T) {
	svc, templateRepo := setupTemplateService()

	templateRepo.Templates["t1"] = &entity.Template{
		ID:         "t1",
		ExternalID: "42",
		Category:   entity.TemplateCategoryUtility,
	}

	err := svc.ProcessTemplateCategoryWebhook(context.Background(), &TemplateCategoryEvent{
		TemplateID:       42,
		TemplateName:     "welcome",
		Language:         "pt_BR",
		PreviousCategory: "UTILITY",
		NewCategory:      "MARKETING",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.TemplateCategoryMarketing, templateRepo.Templates["t1"].Category)
}

// -----------------------------------------------------------------------------
// Create payload — new P1 fields (parameter_format, sub_category, TTL, allow_category_change)
// -----------------------------------------------------------------------------

func TestTemplateService_Create_SendsNewFieldsToMeta(t *testing.T) {
	svc, _ := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID:       "ch-1",
		TenantID: "tenant-1",
		Type:     entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{
			"access_token": "test-token",
			"waba_id":      "waba-123",
		},
	}

	var captured map[string]interface{}
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"tpl-new"}`)),
		}, nil
	})

	_, err := svc.Create(context.Background(), &CreateTemplateInput{
		TenantID:              "tenant-1",
		ChannelID:             "ch-1",
		Name:                  "order_update",
		Language:              "pt_BR",
		Category:              entity.TemplateCategoryUtility,
		SubCategory:           "ORDER_STATUS",
		ParameterFormat:       entity.TemplateParameterFormatNamed,
		MessageSendTTLSeconds: 3600,
		AllowCategoryChange:   true,
		Components: []entity.TemplateComponent{
			{
				Type: "BODY",
				Text: "Oi {{customer_name}}, o pedido {{order_id}} foi atualizado.",
				Example: &entity.TemplateExample{
					BodyText: [][]string{{"Ana", "ORD-42"}},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)

	assert.Equal(t, "order_update", captured["name"])
	assert.Equal(t, "pt_BR", captured["language"])
	assert.Equal(t, "UTILITY", captured["category"])
	assert.Equal(t, "ORDER_STATUS", captured["sub_category"])
	assert.Equal(t, "NAMED", captured["parameter_format"])
	assert.Equal(t, float64(3600), captured["message_send_ttl_seconds"])
	assert.Equal(t, true, captured["allow_category_change"])
}

func TestTemplateService_Create_OmitsEmptyOptionalFields(t *testing.T) {
	// When the caller doesn't set the new optional fields, the payload
	// should not carry them (Meta would otherwise interpret zero values as
	// explicit settings).
	svc, _ := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "t", "waba_id": "w"},
	}

	var captured map[string]interface{}
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"t"}`))}, nil
	})

	_, err := svc.Create(context.Background(), &CreateTemplateInput{
		TenantID:  "tenant-1",
		ChannelID: "ch-1",
		Name:      "plain",
		Language:  "pt_BR",
		Category:  entity.TemplateCategoryMarketing,
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "No variables"},
		},
	})
	require.NoError(t, err)

	_, hasSub := captured["sub_category"]
	_, hasFmt := captured["parameter_format"]
	_, hasTTL := captured["message_send_ttl_seconds"]
	_, hasAllow := captured["allow_category_change"]
	assert.False(t, hasSub, "sub_category must be omitted when empty")
	assert.False(t, hasFmt, "parameter_format must be omitted when empty")
	assert.False(t, hasTTL, "message_send_ttl_seconds must be omitted when zero")
	assert.False(t, hasAllow, "allow_category_change must be omitted when false")
}

// -----------------------------------------------------------------------------
// FetchNamespace — GET /{waba-id}?fields=message_template_namespace
// -----------------------------------------------------------------------------

func TestTemplateService_FetchNamespace_PersistsOnChannel(t *testing.T) {
	svc, _ := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "t", "waba_id": "waba-1"},
	}

	var capturedURL string
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedURL = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"message_template_namespace":"ns-xyz-123"}`)),
		}, nil
	})

	ns, err := svc.FetchNamespace(context.Background(), "ch-1")
	require.NoError(t, err)
	assert.Equal(t, "ns-xyz-123", ns)
	assert.Contains(t, capturedURL, "/waba-1")
	assert.Contains(t, capturedURL, "message_template_namespace")

	// Must have persisted on the channel
	assert.Equal(t, "ns-xyz-123", channelRepo.Channels["ch-1"].MessageTemplateNamespace)
}

func TestTemplateService_FetchNamespace_MissingCreds(t *testing.T) {
	svc, _ := setupTemplateService()
	_, err := svc.FetchNamespace(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials")
}

// -----------------------------------------------------------------------------
// DeleteBulk — DELETE /message_templates?hsm_ids=[...]
// -----------------------------------------------------------------------------

func TestTemplateService_DeleteBulk_SingleCallToMeta(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "t", "waba_id": "w"},
	}
	templateRepo.Templates["a"] = &entity.Template{ID: "a", ChannelID: "ch-1", ExternalID: "hsm-1", Name: "a"}
	templateRepo.Templates["b"] = &entity.Template{ID: "b", ChannelID: "ch-1", ExternalID: "hsm-2", Name: "b"}
	templateRepo.Templates["c"] = &entity.Template{ID: "c", ChannelID: "ch-1", ExternalID: "", Name: "c"} // unsynced

	var calls int
	var capturedURL string
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		capturedURL = r.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"success":true}`))}, nil
	})

	require.NoError(t, svc.DeleteBulk(context.Background(), []string{"a", "b", "c"}))
	assert.Equal(t, 1, calls, "bulk delete must hit Meta exactly once")
	assert.Contains(t, capturedURL, "hsm_ids=")
	// All three rows are gone locally — even the unsynced one
	assert.Empty(t, templateRepo.Templates)
}

func TestTemplateService_DeleteBulk_RejectsCrossChannel(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	templateRepo.Templates["a"] = &entity.Template{ID: "a", ChannelID: "ch-1", ExternalID: "hsm-1"}
	templateRepo.Templates["b"] = &entity.Template{ID: "b", ChannelID: "ch-2", ExternalID: "hsm-2"}

	err := svc.DeleteBulk(context.Background(), []string{"a", "b"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same channel")
}

func TestTemplateService_DeleteBulk_EmptyNoOp(t *testing.T) {
	svc, _ := setupTemplateService()
	assert.NoError(t, svc.DeleteBulk(context.Background(), nil))
}

// -----------------------------------------------------------------------------
// Edit — POST /{template_id}
// -----------------------------------------------------------------------------

func TestTemplateService_Edit_SendsChangesToMeta(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "test-token", "waba_id": "w"},
	}
	templateRepo.Templates["t1"] = &entity.Template{
		ID:         "t1",
		TenantID:   "tenant-1",
		ChannelID:  "ch-1",
		ExternalID: "hsm-42",
		Name:       "welcome",
		Language:   "pt_BR",
		Category:   entity.TemplateCategoryUtility,
		Status:     entity.TemplateStatusApproved,
	}

	var capturedURL string
	var captured map[string]interface{}
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedURL = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"success":true}`))}, nil
	})

	newComponents := []entity.TemplateComponent{
		{Type: "BODY", Text: "New body no vars"},
	}
	result, err := svc.Edit(context.Background(), &EditTemplateInput{
		ID:                    "t1",
		Category:              entity.TemplateCategoryMarketing,
		Components:            newComponents,
		MessageSendTTLSeconds: 7200,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Status resets to PENDING after edit — mirrors Meta's behaviour
	assert.Equal(t, entity.TemplateStatusPending, result.Status)
	assert.Equal(t, entity.TemplateCategoryMarketing, result.Category)
	assert.Contains(t, capturedURL, "/hsm-42", "edit must target /{template_id}")
	assert.Equal(t, "MARKETING", captured["category"])
	assert.Equal(t, float64(7200), captured["message_send_ttl_seconds"])
}

func TestTemplateService_Edit_RejectsInvalidComponents(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	templateRepo.Templates["t1"] = &entity.Template{
		ID: "t1", ChannelID: "ch-1", ExternalID: "hsm-1",
	}

	_, err := svc.Edit(context.Background(), &EditTemplateInput{
		ID: "t1",
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Hi {{1}}"}, // no example
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestTemplateService_Edit_RequiresSyncedTemplate(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	templateRepo.Templates["t1"] = &entity.Template{
		ID: "t1", ChannelID: "ch-1", // no ExternalID
	}

	_, err := svc.Edit(context.Background(), &EditTemplateInput{
		ID:         "t1",
		Components: []entity.TemplateComponent{{Type: "BODY", Text: "ok"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "synced")
}

// -----------------------------------------------------------------------------
// Refresh — GET /{template_id}
// -----------------------------------------------------------------------------

func TestTemplateService_RefreshFromMeta_UpdatesLocalRow(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "t", "waba_id": "w"},
	}
	templateRepo.Templates["t1"] = &entity.Template{
		ID: "t1", TenantID: "tenant-1", ChannelID: "ch-1",
		ExternalID: "hsm-99",
		Status:     entity.TemplateStatusPending,
	}

	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		assert.Contains(t, r.URL.Path, "/hsm-99")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"hsm-99","name":"welcome","language":"en_US","status":"APPROVED","category":"UTILITY"}`)),
		}, nil
	})

	result, err := svc.RefreshFromMeta(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, entity.TemplateStatusApproved, result.Status)
	assert.Equal(t, entity.TemplateCategoryUtility, result.Category)
	assert.Equal(t, "en_US", result.Language)
	assert.NotNil(t, result.LastSyncedAt)
}

func TestTemplateService_RefreshFromMeta_NoExternalID(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	templateRepo.Templates["t1"] = &entity.Template{ID: "t1", ChannelID: "ch-1"}

	_, err := svc.RefreshFromMeta(context.Background(), "t1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "external_id")
}

// -----------------------------------------------------------------------------
// Status mapper — newly added LIMIT_EXCEEDED and ARCHIVED
// -----------------------------------------------------------------------------

func TestMapMetaStatusToEntity_NewValues(t *testing.T) {
	cases := map[string]entity.TemplateStatus{
		"LIMIT_EXCEEDED": entity.TemplateStatusLimitExceeded,
		"ARCHIVED":       entity.TemplateStatusArchived,
	}
	for meta, want := range cases {
		assert.Equal(t, want, mapMetaStatusToEntity(meta), "meta status=%s", meta)
	}
}

func TestMapMetaStatusToEntity_UnknownFallsBackToPending(t *testing.T) {
	assert.Equal(t, entity.TemplateStatusPending, mapMetaStatusToEntity("SOMETHING_NEW"))
}

// -----------------------------------------------------------------------------
// Delete — must pass hsm_id so Meta only removes the specific language variant
// -----------------------------------------------------------------------------

func TestTemplateService_Delete_PassesHSMIDToMeta(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID:       "ch-1",
		TenantID: "tenant-1",
		Type:     entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{
			"access_token": "test-token",
			"waba_id":      "waba-123",
		},
	}
	templateRepo.Templates["t1"] = &entity.Template{
		ID:         "t1",
		TenantID:   "tenant-1",
		ChannelID:  "ch-1",
		ExternalID: "hsm-42",
		Name:       "welcome",
		Language:   "pt_BR",
	}

	var capturedURL string
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		capturedURL = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
		}, nil
	})

	require.NoError(t, svc.Delete(context.Background(), "t1"))
	assert.Contains(t, capturedURL, "name=welcome", "must include name param")
	assert.Contains(t, capturedURL, "hsm_id=hsm-42",
		"must include hsm_id so Meta only deletes this language variant, not every variant sharing the name")
	// Verify the row is gone locally too
	_, exists := templateRepo.Templates["t1"]
	assert.False(t, exists)
}

func TestTemplateService_Delete_SkipsMetaWhenNoExternalID(t *testing.T) {
	// A template that never synced to Meta (no ExternalID) shouldn't trigger
	// an HTTP call on delete — otherwise we'd call Meta with an empty hsm_id
	// and (worse) fall back to name-only delete, wiping other variants.
	svc, templateRepo := setupTemplateService()
	channelRepo := svc.channelRepo.(*testutil.MockChannelRepository)
	channelRepo.Channels["ch-1"] = &entity.Channel{
		ID: "ch-1", TenantID: "tenant-1", Type: entity.ChannelTypeWhatsAppOfficial,
		Credentials: map[string]string{"access_token": "t", "waba_id": "w"},
	}
	templateRepo.Templates["t1"] = &entity.Template{
		ID: "t1", TenantID: "tenant-1", ChannelID: "ch-1",
		Name: "welcome", Language: "pt_BR",
		// no ExternalID
	}

	called := false
	svc.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})

	require.NoError(t, svc.Delete(context.Background(), "t1"))
	assert.False(t, called, "no Meta call expected when template was never synced")
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
