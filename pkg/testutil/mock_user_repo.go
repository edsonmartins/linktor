package testutil

import (
	"context"
	"fmt"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// MockUserRepository is a mock implementation of repository.UserRepository
type MockUserRepository struct {
	Users       map[string]*entity.User
	ReturnError error
}

// NewMockUserRepository creates a new MockUserRepository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		Users: make(map[string]*entity.User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	user, ok := m.Users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, u := range m.Users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found by email: %s", email)
}

func (m *MockUserRepository) FindByTenantAndEmail(ctx context.Context, tenantID, email string) (*entity.User, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, u := range m.Users {
		if u.TenantID == tenantID && u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found: %s/%s", tenantID, email)
}

func (m *MockUserRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.User, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.User
	for _, u := range m.Users {
		if u.TenantID == tenantID {
			result = append(result, u)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Users, id)
	return nil
}

func (m *MockUserRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, u := range m.Users {
		if u.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockUserRepository) FindAvailableAgents(ctx context.Context, tenantID, channelID string) ([]*entity.User, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.User
	for _, u := range m.Users {
		if u.TenantID == tenantID && u.Role == entity.UserRoleAgent {
			result = append(result, u)
		}
	}
	return result, nil
}

// MockTenantRepository is a mock implementation of repository.TenantRepository
type MockTenantRepository struct {
	Tenants     map[string]*entity.Tenant
	ReturnError error
}

// NewMockTenantRepository creates a new MockTenantRepository
func NewMockTenantRepository() *MockTenantRepository {
	return &MockTenantRepository{
		Tenants: make(map[string]*entity.Tenant),
	}
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *entity.Tenant) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantRepository) FindByID(ctx context.Context, id string) (*entity.Tenant, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	tenant, ok := m.Tenants[id]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}
	return tenant, nil
}

func (m *MockTenantRepository) FindBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, t := range m.Tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, fmt.Errorf("tenant not found by slug: %s", slug)
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *entity.Tenant) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Tenants, id)
	return nil
}
