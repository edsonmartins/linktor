package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAllowlistService() (*SandboxAllowlistService, *testutil.MockSandboxAllowlistRepository, *testutil.MockChannelRepository) {
	repo := testutil.NewMockSandboxAllowlistRepository()
	channelRepo := testutil.NewMockChannelRepository()
	return NewSandboxAllowlistService(repo, channelRepo), repo, channelRepo
}

func TestSandboxAllowlist_Add_NormalizesToE164(t *testing.T) {
	svc, repo, _ := newAllowlistService()

	entry, err := svc.Add(context.Background(), &AddSandboxAllowlistInput{
		TenantID:  "tenant1",
		Recipient: "+55 44 99999-9999",
		CreatedBy: "user1",
	})

	require.NoError(t, err)
	assert.Equal(t, "+5544999999999", entry.Recipient)
	assert.Equal(t, "+5544999999999", repo.Entries[entry.ID].Recipient,
		"stored recipient must be the normalized form")
}

func TestSandboxAllowlist_Add_RejectsInvalidNumber(t *testing.T) {
	svc, repo, _ := newAllowlistService()

	_, err := svc.Add(context.Background(), &AddSandboxAllowlistInput{
		TenantID:  "tenant1",
		Recipient: "not-a-phone",
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
	assert.Empty(t, repo.Entries)
}

func TestSandboxAllowlist_Add_ChannelScopedRequiresSandboxChannelOfTenant(t *testing.T) {
	svc, _, channelRepo := newAllowlistService()

	sandboxCh := entity.NewChannel("tenant1", entity.ChannelTypeWhatsAppOfficial, "wa-sandbox", "")
	sandboxCh.ID = "ch-sandbox"
	sandboxCh.Environment = entity.ChannelEnvironmentSandbox
	prodCh := entity.NewChannel("tenant1", entity.ChannelTypeWhatsAppOfficial, "wa-prod", "")
	prodCh.ID = "ch-prod"
	otherTenantCh := entity.NewChannel("tenant2", entity.ChannelTypeWhatsAppOfficial, "wa-other", "")
	otherTenantCh.ID = "ch-other"
	otherTenantCh.Environment = entity.ChannelEnvironmentSandbox
	channelRepo.Channels[sandboxCh.ID] = sandboxCh
	channelRepo.Channels[prodCh.ID] = prodCh
	channelRepo.Channels[otherTenantCh.ID] = otherTenantCh

	base := func(channelID string) *AddSandboxAllowlistInput {
		return &AddSandboxAllowlistInput{
			TenantID:  "tenant1",
			ChannelID: channelID,
			Recipient: "+5544999999999",
		}
	}

	_, err := svc.Add(context.Background(), base(sandboxCh.ID))
	require.NoError(t, err, "sandbox channel of the tenant must be accepted")

	_, err = svc.Add(context.Background(), base(prodCh.ID))
	require.Error(t, err, "production channel must be rejected")
	assert.True(t, errors.IsValidation(err))

	_, err = svc.Add(context.Background(), base(otherTenantCh.ID))
	require.Error(t, err, "another tenant's channel must be rejected, not found")
}

func TestSandboxAllowlist_Remove_CrossTenantIsRejected(t *testing.T) {
	svc, repo, _ := newAllowlistService()

	entry, err := svc.Add(context.Background(), &AddSandboxAllowlistInput{
		TenantID:  "tenant1",
		Recipient: "+5544999999999",
	})
	require.NoError(t, err)

	_, err = svc.Remove(context.Background(), "tenant2", entry.ID)
	require.Error(t, err, "cross-tenant removal must be rejected, not silently succeed")
	assert.True(t, errors.IsNotFound(err))
	assert.Contains(t, repo.Entries, entry.ID, "entry must still exist")

	removed, err := svc.Remove(context.Background(), "tenant1", entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, removed.ID)
	assert.NotContains(t, repo.Entries, entry.ID)
}

func TestSandboxAllowlist_List_IsTenantScoped(t *testing.T) {
	svc, _, _ := newAllowlistService()

	_, err := svc.Add(context.Background(), &AddSandboxAllowlistInput{
		TenantID: "tenant1", Recipient: "+5544999999999"})
	require.NoError(t, err)
	_, err = svc.Add(context.Background(), &AddSandboxAllowlistInput{
		TenantID: "tenant2", Recipient: "+5544888888888"})
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), "tenant1")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "+5544999999999", entries[0].Recipient)
}
