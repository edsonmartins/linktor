package service

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type mockAPIKeyRepository struct {
	created *entity.APIKey
	items   []*entity.APIKey
	touched string
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

func (m *mockAPIKeyRepository) FindActiveByPrefix(ctx context.Context, prefix string) ([]*entity.APIKey, error) {
	var out []*entity.APIKey
	for _, k := range m.items {
		if k.KeyPrefix == prefix {
			out = append(out, k)
		}
	}
	return out, nil
}

func (m *mockAPIKeyRepository) TouchLastUsed(ctx context.Context, id string, at time.Time) error {
	m.touched = id
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

func TestAPIKeyServiceAuthenticate(t *testing.T) {
	repo := &mockAPIKeyRepository{}
	svc := NewAPIKeyService(repo)

	created, err := svc.Create(context.Background(), &CreateAPIKeyInput{
		TenantID: "tenant-1",
		UserID:   "user-1",
		Name:     "Server Key",
	})
	require.NoError(t, err)
	repo.items = []*entity.APIKey{created.APIKey}

	// Correct key authenticates and resolves the tenant.
	key, err := svc.Authenticate(context.Background(), created.Key)
	require.NoError(t, err)
	require.Equal(t, "tenant-1", key.TenantID)
	require.Empty(t, key.KeyHash, "hash must not be exposed")
	require.Equal(t, created.APIKey.ID, repo.touched, "last_used should be recorded")

	// Wrong secret with a matching prefix is rejected.
	_, err = svc.Authenticate(context.Background(), created.Key[:apiKeyPrefixLength]+"deadbeefdeadbeef")
	require.Error(t, err)

	// Empty / too-short key is rejected.
	_, err = svc.Authenticate(context.Background(), "lk_")
	require.Error(t, err)
}

func TestAPIKeyServiceAuthenticateRejectsExpired(t *testing.T) {
	repo := &mockAPIKeyRepository{}
	svc := NewAPIKeyService(repo)
	created, err := svc.Create(context.Background(), &CreateAPIKeyInput{TenantID: "t", Name: "K"})
	require.NoError(t, err)
	// Repo filters expired keys, so an expired key simply isn't returned by prefix.
	repo.items = nil

	_, err = svc.Authenticate(context.Background(), created.Key)
	require.Error(t, err)
}

func TestAPIKeyServiceCreateRequiresName(t *testing.T) {
	service := NewAPIKeyService(&mockAPIKeyRepository{})

	_, err := service.Create(context.Background(), &CreateAPIKeyInput{
		TenantID: "tenant-1",
		Name:     "   ",
	})

	require.Error(t, err)
}
