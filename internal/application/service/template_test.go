package service

import (
	"context"
	"fmt"
	"testing"

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
			{Type: "BODY", Text: "Hello {{1}}!"},
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
