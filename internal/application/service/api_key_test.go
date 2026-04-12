package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type mockAPIKeyRepository struct {
	created *entity.APIKey
	items   []*entity.APIKey
}

func (m *mockAPIKeyRepository) Create(ctx context.Context, apiKey *entity.APIKey) error {
	m.created = apiKey
	return nil
}

func (m *mockAPIKeyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*entity.APIKey, error) {
	return m.items, nil
}

func (m *mockAPIKeyRepository) Delete(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestAPIKeyServiceCreateStoresHashAndReturnsRawKeyOnce(t *testing.T) {
	repo := &mockAPIKeyRepository{}
	service := NewAPIKeyService(repo)

	result, err := service.Create(context.Background(), &CreateAPIKeyInput{
		TenantID: "tenant-1",
		UserID:   "user-1",
		Name:     "Admin API Key",
		Scopes:   []string{"*"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, repo.created)
	require.NotEmpty(t, result.Key)
	require.NotEqual(t, result.Key, repo.created.KeyHash)
	require.Equal(t, result.Key[:apiKeyPrefixLength], repo.created.KeyPrefix)
	require.Equal(t, "Admin API Key", repo.created.Name)
	require.Equal(t, []string{"*"}, repo.created.Scopes)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(repo.created.KeyHash), []byte(result.Key)))
}

func TestAPIKeyServiceCreateDefaultsScopes(t *testing.T) {
	repo := &mockAPIKeyRepository{}
	service := NewAPIKeyService(repo)

	_, err := service.Create(context.Background(), &CreateAPIKeyInput{
		TenantID: "tenant-1",
		Name:     "Default scopes",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"*"}, repo.created.Scopes)
}

func TestAPIKeyServiceCreateRequiresName(t *testing.T) {
	service := NewAPIKeyService(&mockAPIKeyRepository{})

	_, err := service.Create(context.Background(), &CreateAPIKeyInput{
		TenantID: "tenant-1",
		Name:     "   ",
	})

	require.Error(t, err)
}
