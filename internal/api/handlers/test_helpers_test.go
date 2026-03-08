package handlers

import (
	"context"
	"fmt"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// ============================================================================
// mockFlowRepository
// ============================================================================

type mockFlowRepository struct {
	Flows       map[string]*entity.Flow
	ReturnError error
}

func newMockFlowRepository() *mockFlowRepository {
	return &mockFlowRepository{
		Flows: make(map[string]*entity.Flow),
	}
}

func (m *mockFlowRepository) Create(ctx context.Context, flow *entity.Flow) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Flows[flow.ID] = flow
	return nil
}

func (m *mockFlowRepository) FindByID(ctx context.Context, id string) (*entity.Flow, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	flow, ok := m.Flows[id]
	if !ok {
		return nil, fmt.Errorf("flow not found: %s", id)
	}
	return flow, nil
}

func (m *mockFlowRepository) FindByTenant(ctx context.Context, tenantID string, filter *entity.FlowFilter, params *repository.ListParams) ([]*entity.Flow, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Flow
	for _, f := range m.Flows {
		if f.TenantID == tenantID {
			result = append(result, f)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockFlowRepository) FindByBot(ctx context.Context, botID string) ([]*entity.Flow, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Flow
	for _, f := range m.Flows {
		if f.BotID != nil && *f.BotID == botID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockFlowRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Flow, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Flow
	for _, f := range m.Flows {
		if f.TenantID == tenantID && f.IsActive {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockFlowRepository) FindByTrigger(ctx context.Context, tenantID string, trigger entity.FlowTriggerType, triggerValue string) ([]*entity.Flow, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.Flow
	for _, f := range m.Flows {
		if f.TenantID == tenantID && f.Trigger == trigger && f.TriggerValue == triggerValue {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockFlowRepository) Update(ctx context.Context, flow *entity.Flow) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Flows[flow.ID] = flow
	return nil
}

func (m *mockFlowRepository) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	flow, ok := m.Flows[id]
	if !ok {
		return fmt.Errorf("flow not found: %s", id)
	}
	flow.IsActive = isActive
	return nil
}

func (m *mockFlowRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Flows, id)
	return nil
}

func (m *mockFlowRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, f := range m.Flows {
		if f.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockFlowRepository) CountByBot(ctx context.Context, botID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, f := range m.Flows {
		if f.BotID != nil && *f.BotID == botID {
			count++
		}
	}
	return count, nil
}

// ============================================================================
// mockTemplateRepository
// ============================================================================

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
	tmpl, ok := m.Templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return tmpl, nil
}

func (m *mockTemplateRepository) FindByExternalID(ctx context.Context, externalID string) (*entity.Template, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, tmpl := range m.Templates {
		if tmpl.ExternalID == externalID {
			return tmpl, nil
		}
	}
	return nil, fmt.Errorf("template not found by external ID: %s", externalID)
}

func (m *mockTemplateRepository) FindByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, tmpl := range m.Templates {
		if tmpl.TenantID == tenantID && tmpl.ChannelID == channelID && tmpl.Name == name && tmpl.Language == language {
			return tmpl, nil
		}
	}
	return nil, fmt.Errorf("template not found by name: %s", name)
}

func (m *mockTemplateRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Template
	for _, tmpl := range m.Templates {
		if tmpl.TenantID == tenantID {
			result = append(result, tmpl)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockTemplateRepository) FindByChannel(ctx context.Context, channelID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Template
	for _, tmpl := range m.Templates {
		if tmpl.ChannelID == channelID {
			result = append(result, tmpl)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockTemplateRepository) FindByStatus(ctx context.Context, tenantID string, status entity.TemplateStatus, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Template
	for _, tmpl := range m.Templates {
		if tmpl.TenantID == tenantID && tmpl.Status == status {
			result = append(result, tmpl)
		}
	}
	return result, int64(len(result)), nil
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
	if m.ReturnError != nil {
		return m.ReturnError
	}
	tmpl, ok := m.Templates[id]
	if !ok {
		return fmt.Errorf("template not found: %s", id)
	}
	tmpl.Status = status
	return nil
}

func (m *mockTemplateRepository) UpdateQuality(ctx context.Context, id string, quality entity.TemplateQuality) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	tmpl, ok := m.Templates[id]
	if !ok {
		return fmt.Errorf("template not found: %s", id)
	}
	tmpl.QualityScore = quality
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
	if m.ReturnError != nil {
		return m.ReturnError
	}
	for id, tmpl := range m.Templates {
		if tmpl.ExternalID == externalID {
			delete(m.Templates, id)
			return nil
		}
	}
	return nil
}

func (m *mockTemplateRepository) CountByChannel(ctx context.Context, channelID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, tmpl := range m.Templates {
		if tmpl.ChannelID == channelID {
			count++
		}
	}
	return count, nil
}

func (m *mockTemplateRepository) UpsertByExternalID(ctx context.Context, template *entity.Template) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	for id, tmpl := range m.Templates {
		if tmpl.ExternalID == template.ExternalID {
			m.Templates[id] = template
			return nil
		}
	}
	m.Templates[template.ID] = template
	return nil
}
