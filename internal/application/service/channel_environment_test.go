package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sandboxCredentials() map[string]string {
	return map[string]string{
		"access_token":           "test-token",
		CredentialEnvironmentKey: "sandbox",
	}
}

// ---------------------------------------------------------------------------
// Create — environment validation (INV-002, INV-016)
// ---------------------------------------------------------------------------

func TestChannelService_Create_DefaultsToProduction(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1",
		Type:     "webchat",
		Name:     "Support Chat",
	})

	require.NoError(t, err)
	assert.Equal(t, entity.ChannelEnvironmentProduction, ch.Environment)
	assert.Equal(t, entity.ChannelEnvironmentProduction, repo.Channels[ch.ID].Environment)
}

func TestChannelService_Create_InvalidEnvironmentRejected(t *testing.T) {
	svc, repo, _ := newChannelService()

	_, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Bad",
		Environment: "staging",
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
	assert.Empty(t, repo.Channels, "nothing may be persisted on validation failure")
}

func TestChannelService_Create_SandboxRequiresCredentialDeclaration(t *testing.T) {
	svc, repo, _ := newChannelService()

	_, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Sandbox",
		Environment: "sandbox",
		Credentials: map[string]string{"api_key": "secret"}, // no declaration
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
	assert.Contains(t, err.Error(), CredentialEnvironmentKey)
	assert.Empty(t, repo.Channels)
}

func TestChannelService_Create_SandboxWithDeclarationSucceeds(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Sandbox",
		Environment: "sandbox",
		Credentials: sandboxCredentials(),
	})

	require.NoError(t, err)
	assert.Equal(t, entity.ChannelEnvironmentSandbox, ch.Environment)
	assert.Equal(t, entity.ChannelEnvironmentSandbox, repo.Channels[ch.ID].Environment)
}

func TestChannelService_Create_ProductionRejectsSandboxCredentials(t *testing.T) {
	svc, repo, _ := newChannelService()

	_, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Prod",
		Credentials: sandboxCredentials(),
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
	assert.Empty(t, repo.Channels)
}

func TestChannelService_Create_SandboxWhatsAppOfficial_PhoneNumberIDValidation(t *testing.T) {
	base := func(config map[string]string) *CreateChannelInput {
		return &CreateChannelInput{
			TenantID:    "tenant1",
			Type:        string(entity.ChannelTypeWhatsAppOfficial),
			Name:        "WA Sandbox",
			Environment: "sandbox",
			Config:      config,
			Credentials: sandboxCredentials(),
		}
	}

	t.Run("phone_number_id in declared list is accepted", func(t *testing.T) {
		svc, _, _ := newChannelService()
		ch, err := svc.Create(context.Background(), base(map[string]string{
			"phone_number_id":            "111222333",
			SandboxTestPhoneNumberIDsKey: "111222333, 444555666",
		}))
		require.NoError(t, err)
		assert.True(t, ch.IsSandbox())
	})

	t.Run("phone_number_id outside declared list is rejected", func(t *testing.T) {
		svc, repo, _ := newChannelService()
		_, err := svc.Create(context.Background(), base(map[string]string{
			"phone_number_id":            "999888777",
			SandboxTestPhoneNumberIDsKey: "111222333",
		}))
		require.Error(t, err)
		assert.True(t, errors.IsValidation(err))
		assert.Empty(t, repo.Channels)
	})

	t.Run("missing phone_number_id is rejected", func(t *testing.T) {
		svc, _, _ := newChannelService()
		_, err := svc.Create(context.Background(), base(map[string]string{
			SandboxTestPhoneNumberIDsKey: "111222333",
		}))
		require.Error(t, err)
		assert.True(t, errors.IsValidation(err))
	})

	t.Run("missing declared list is rejected", func(t *testing.T) {
		svc, _, _ := newChannelService()
		_, err := svc.Create(context.Background(), base(map[string]string{
			"phone_number_id": "111222333",
		}))
		require.Error(t, err)
		assert.True(t, errors.IsValidation(err))
	})
}

// ---------------------------------------------------------------------------
// Update — immutability (INV-016) and binding re-validation on merge
// ---------------------------------------------------------------------------

func TestChannelService_Update_EnvironmentIsImmutable(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Sandbox",
		Environment: "sandbox",
		Credentials: sandboxCredentials(),
	})
	require.NoError(t, err)

	production := "production"
	_, err = svc.Update(context.Background(), ch.ID, &UpdateChannelInput{
		Environment: &production,
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
	assert.Contains(t, err.Error(), "immutable")
	assert.Equal(t, entity.ChannelEnvironmentSandbox, repo.Channels[ch.ID].Environment,
		"persisted environment must be unchanged after rejected update")
}

func TestChannelService_Update_SameEnvironmentIsNoop(t *testing.T) {
	svc, _, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Sandbox",
		Environment: "sandbox",
		Credentials: sandboxCredentials(),
	})
	require.NoError(t, err)

	sandbox := "sandbox"
	name := "renamed"
	updated, err := svc.Update(context.Background(), ch.ID, &UpdateChannelInput{
		Environment: &sandbox,
		Name:        &name,
	})

	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.Equal(t, entity.ChannelEnvironmentSandbox, updated.Environment)
}

func TestChannelService_Update_CannotMoveSandboxPhoneNumberIDOffList(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        string(entity.ChannelTypeWhatsAppOfficial),
		Name:        "WA Sandbox",
		Environment: "sandbox",
		Config: map[string]string{
			"phone_number_id":            "111222333",
			SandboxTestPhoneNumberIDsKey: "111222333",
		},
		Credentials: sandboxCredentials(),
	})
	require.NoError(t, err)

	updateCalls := repo.UpdateCalls
	_, err = svc.Update(context.Background(), ch.ID, &UpdateChannelInput{
		Config: map[string]string{
			"phone_number_id":            "999888777", // production number
			SandboxTestPhoneNumberIDsKey: "111222333",
		},
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
	assert.Equal(t, updateCalls, repo.UpdateCalls,
		"repository Update must not be reached after rejected validation")
}

func TestChannelService_Update_CannotAttachSandboxCredentialsToProduction(t *testing.T) {
	svc, _, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1",
		Type:     "webchat",
		Name:     "Prod",
	})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), ch.ID, &UpdateChannelInput{
		Credentials: map[string]string{CredentialEnvironmentKey: "sandbox"},
	})

	require.Error(t, err)
	assert.True(t, errors.IsValidation(err))
}
