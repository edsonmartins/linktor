package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ============================================================================
// MockCampaignRepository
// ============================================================================

type MockCampaignRepository struct {
	Campaigns   map[string]*entity.Campaign
	Recipients  map[string]*entity.CampaignRecipient
	ReturnError error
}

func NewMockCampaignRepository() *MockCampaignRepository {
	return &MockCampaignRepository{
		Campaigns:  make(map[string]*entity.Campaign),
		Recipients: make(map[string]*entity.CampaignRecipient),
	}
}

func (m *MockCampaignRepository) Create(ctx context.Context, c *entity.Campaign) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	cp := *c
	m.Campaigns[c.ID] = &cp
	return nil
}

func (m *MockCampaignRepository) FindByID(ctx context.Context, id string) (*entity.Campaign, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	c, ok := m.Campaigns[id]
	if !ok {
		return nil, fmt.Errorf("campaign not found: %s", id)
	}
	cp := *c
	return &cp, nil
}

func (m *MockCampaignRepository) FindByTenant(ctx context.Context, tenantID string, _ *repository.ListParams) ([]*entity.Campaign, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var out []*entity.Campaign
	for _, c := range m.Campaigns {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, int64(len(out)), nil
}

func (m *MockCampaignRepository) Update(ctx context.Context, c *entity.Campaign) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	cp := *c
	m.Campaigns[c.ID] = &cp
	return nil
}

func (m *MockCampaignRepository) Delete(ctx context.Context, id string) error {
	delete(m.Campaigns, id)
	return nil
}

func (m *MockCampaignRepository) AddRecipients(ctx context.Context, recipients []*entity.CampaignRecipient) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	for _, r := range recipients {
		rc := *r
		m.Recipients[r.ID] = &rc
	}
	return nil
}

func (m *MockCampaignRepository) FindPendingRecipients(ctx context.Context, campaignID string, limit int) ([]*entity.CampaignRecipient, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var out []*entity.CampaignRecipient
	for _, r := range m.Recipients {
		if r.CampaignID == campaignID && r.Status == entity.RecipientPending {
			rc := *r
			out = append(out, &rc)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *MockCampaignRepository) ListRecipients(ctx context.Context, campaignID, status string, _ *repository.ListParams) ([]*entity.CampaignRecipient, int64, error) {
	var out []*entity.CampaignRecipient
	for _, r := range m.Recipients {
		if r.CampaignID == campaignID && (status == "" || string(r.Status) == status) {
			out = append(out, r)
		}
	}
	return out, int64(len(out)), nil
}

func (m *MockCampaignRepository) UpdateRecipientStatus(ctx context.Context, recipientID string, status entity.RecipientStatus, messageID, errReason string) error {
	if r, ok := m.Recipients[recipientID]; ok {
		r.Status = status
		if messageID != "" {
			r.MessageID = messageID
		}
		r.ErrorReason = errReason
		r.Attempts++
	}
	return nil
}

func (m *MockCampaignRepository) MarkRecipientQueued(ctx context.Context, recipientID string) error {
	if r, ok := m.Recipients[recipientID]; ok && r.Status == entity.RecipientPending {
		r.Status = entity.RecipientQueued
	}
	return nil
}

func (m *MockCampaignRepository) UpdateRecipientStatusByMessageID(ctx context.Context, messageID string, status entity.RecipientStatus) error {
	for _, r := range m.Recipients {
		if r.MessageID == messageID {
			r.Status = status
		}
	}
	return nil
}

func (m *MockCampaignRepository) ResetFailedRecipients(ctx context.Context, campaignID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var n int64
	for _, r := range m.Recipients {
		if r.CampaignID == campaignID && r.Status == entity.RecipientFailed {
			r.Status = entity.RecipientPending
			r.ErrorReason = ""
			n++
		}
	}
	return n, nil
}

func (m *MockCampaignRepository) SweepStaleQueued(ctx context.Context, _ time.Duration) ([]string, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	seen := map[string]bool{}
	var ids []string
	for _, r := range m.Recipients {
		if r.Status == entity.RecipientQueued {
			r.Status = entity.RecipientFailed
			r.ErrorReason = "delivery not confirmed"
			if !seen[r.CampaignID] {
				seen[r.CampaignID] = true
				ids = append(ids, r.CampaignID)
			}
		}
	}
	return ids, nil
}

func (m *MockCampaignRepository) RecountStatuses(ctx context.Context, campaignID string) error {
	c, ok := m.Campaigns[campaignID]
	if !ok {
		return nil
	}
	var sent, delivered, read, failed int
	for _, r := range m.Recipients {
		if r.CampaignID != campaignID {
			continue
		}
		switch r.Status {
		case entity.RecipientSent:
			sent++
		case entity.RecipientDelivered:
			sent++
			delivered++
		case entity.RecipientRead:
			sent++
			read++
		case entity.RecipientFailed:
			failed++
		}
	}
	c.SentCount, c.DeliveredCount, c.ReadCount, c.FailedCount = sent, delivered, read, failed
	return nil
}

// ============================================================================
// MockRoleRepository
// ============================================================================

type MockRoleRepository struct {
	Roles         map[string]*entity.Role // by ID
	ReturnError   error                   // returned by all methods
	FindByNameErr error                   // specific error for FindByName (e.g. transient)
}

func NewMockRoleRepository() *MockRoleRepository {
	return &MockRoleRepository{Roles: make(map[string]*entity.Role)}
}

func (m *MockRoleRepository) Create(ctx context.Context, role *entity.Role) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	r := *role
	m.Roles[role.ID] = &r
	return nil
}

func (m *MockRoleRepository) FindByID(ctx context.Context, id string) (*entity.Role, error) {
	if r, ok := m.Roles[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("role not found")
}

func (m *MockRoleRepository) FindByName(ctx context.Context, tenantID, name string) (*entity.Role, error) {
	if m.FindByNameErr != nil {
		return nil, m.FindByNameErr
	}
	for _, r := range m.Roles {
		if r.TenantID == tenantID && r.Name == name {
			return r, nil
		}
	}
	// Return a real NOT_FOUND so callers using errors.IsNotFound behave correctly.
	return nil, errors.New(errors.ErrCodeNotFound, "role not found")
}

func (m *MockRoleRepository) FindByTenant(ctx context.Context, tenantID string) ([]*entity.Role, error) {
	var out []*entity.Role
	for _, r := range m.Roles {
		if r.TenantID == tenantID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *MockRoleRepository) Update(ctx context.Context, role *entity.Role) error {
	r := *role
	m.Roles[role.ID] = &r
	return nil
}

func (m *MockRoleRepository) Delete(ctx context.Context, id string) error {
	delete(m.Roles, id)
	return nil
}

// ============================================================================
// MockCannedResponseRepository
// ============================================================================

type MockCannedResponseRepository struct {
	Items map[string]*entity.CannedResponse
}

func NewMockCannedResponseRepository() *MockCannedResponseRepository {
	return &MockCannedResponseRepository{Items: make(map[string]*entity.CannedResponse)}
}

func (m *MockCannedResponseRepository) Create(ctx context.Context, cr *entity.CannedResponse) error {
	c := *cr
	m.Items[cr.ID] = &c
	return nil
}

func (m *MockCannedResponseRepository) FindByID(ctx context.Context, id string) (*entity.CannedResponse, error) {
	if c, ok := m.Items[id]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("canned response not found")
}

func (m *MockCannedResponseRepository) FindByShortcut(ctx context.Context, tenantID, shortcut string) (*entity.CannedResponse, error) {
	for _, c := range m.Items {
		if c.TenantID == tenantID && c.Shortcut == shortcut {
			return c, nil
		}
	}
	return nil, fmt.Errorf("canned response not found")
}

func (m *MockCannedResponseRepository) FindByTenant(ctx context.Context, tenantID, _ string, _ *repository.ListParams) ([]*entity.CannedResponse, int64, error) {
	var out []*entity.CannedResponse
	for _, c := range m.Items {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, int64(len(out)), nil
}

func (m *MockCannedResponseRepository) Update(ctx context.Context, cr *entity.CannedResponse) error {
	m.Items[cr.ID] = cr
	return nil
}

func (m *MockCannedResponseRepository) Delete(ctx context.Context, id string) error {
	delete(m.Items, id)
	return nil
}

func (m *MockCannedResponseRepository) IncrementUsage(ctx context.Context, id string) error {
	if c, ok := m.Items[id]; ok {
		c.UsageCount++
	}
	return nil
}

// ============================================================================
// MockTenantSettingsRepository
// ============================================================================

type MockTenantSettingsRepository struct {
	Settings map[string]*entity.TenantSettings
}

func NewMockTenantSettingsRepository() *MockTenantSettingsRepository {
	return &MockTenantSettingsRepository{Settings: make(map[string]*entity.TenantSettings)}
}

func (m *MockTenantSettingsRepository) Get(ctx context.Context, tenantID string) (*entity.TenantSettings, error) {
	if s, ok := m.Settings[tenantID]; ok {
		return s, nil
	}
	return entity.DefaultTenantSettings(tenantID), nil
}

func (m *MockTenantSettingsRepository) Upsert(ctx context.Context, s *entity.TenantSettings) error {
	cp := *s
	m.Settings[s.TenantID] = &cp
	return nil
}

func (m *MockTenantSettingsRepository) ListWithSLA(ctx context.Context) ([]*entity.TenantSettings, error) {
	var out []*entity.TenantSettings
	for _, s := range m.Settings {
		if s.SLAFirstResponseMinutes > 0 || s.SLAResolutionMinutes > 0 || s.AutoCloseAfterMinutes > 0 {
			out = append(out, s)
		}
	}
	return out, nil
}

// ============================================================================
// MockAssignmentRepository
// ============================================================================

type MockAssignmentRepository struct {
	AgentID     string // returned by FindAssignableAgent ("" = none)
	ReturnError error
	Assigned    []string // userIDs passed to MarkAssigned
}

func NewMockAssignmentRepository() *MockAssignmentRepository {
	return &MockAssignmentRepository{}
}

func (m *MockAssignmentRepository) FindAssignableAgent(ctx context.Context, tenantID string, _ entity.AssignmentStrategy) (string, error) {
	if m.ReturnError != nil {
		return "", m.ReturnError
	}
	return m.AgentID, nil
}

func (m *MockAssignmentRepository) MarkAssigned(ctx context.Context, userID string) error {
	m.Assigned = append(m.Assigned, userID)
	return nil
}

// ============================================================================
// MockSLARepository
// ============================================================================

type MockSLARepository struct {
	AutoCloseIDs map[string][]string // tenantID -> recipient/conv ids
	BreachIDs    map[string][]string // tenantID -> conv ids
	Closed       []string
	Breached     []string
}

func NewMockSLARepository() *MockSLARepository {
	return &MockSLARepository{
		AutoCloseIDs: make(map[string][]string),
		BreachIDs:    make(map[string][]string),
	}
}

func (m *MockSLARepository) FindForAutoClose(ctx context.Context, tenantID string, _ int) ([]string, error) {
	return m.AutoCloseIDs[tenantID], nil
}

func (m *MockSLARepository) FindFirstResponseBreaches(ctx context.Context, tenantID string, _ int) ([]string, error) {
	return m.BreachIDs[tenantID], nil
}

func (m *MockSLARepository) MarkBreached(ctx context.Context, id string) error {
	m.Breached = append(m.Breached, id)
	return nil
}

func (m *MockSLARepository) Close(ctx context.Context, id string) error {
	m.Closed = append(m.Closed, id)
	return nil
}

// ============================================================================
// MockAuditLogRepository
// ============================================================================

type MockAuditLogRepository struct {
	Logs []*entity.AuditLog
}

func NewMockAuditLogRepository() *MockAuditLogRepository {
	return &MockAuditLogRepository{}
}

func (m *MockAuditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	m.Logs = append(m.Logs, log)
	return nil
}

func (m *MockAuditLogRepository) FindByTenant(ctx context.Context, tenantID string, _ *repository.AuditLogFilters, _ *repository.ListParams) ([]*entity.AuditLog, int64, error) {
	var out []*entity.AuditLog
	for _, l := range m.Logs {
		if l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, int64(len(out)), nil
}
