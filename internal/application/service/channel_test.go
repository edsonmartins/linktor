package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newChannelService() (*ChannelService, *testutil.MockChannelRepository, *testutil.MockProducer) {
	repo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	registry := plugin.NewRegistry()
	svc := NewChannelService(repo, registry, producer)
	return svc, repo, producer
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestChannelService_Create(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:   "tenant1",
		Type:       "webchat",
		Name:       "Support Chat",
		Identifier: "support-widget",
		Config:     map[string]string{"color": "blue"},
		Credentials: map[string]string{"api_key": "secret"},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, ch.ID)
	assert.Equal(t, "tenant1", ch.TenantID)
	assert.Equal(t, entity.ChannelTypeWebChat, ch.Type)
	assert.Equal(t, "Support Chat", ch.Name)
	assert.Equal(t, "support-widget", ch.Identifier)
	assert.True(t, ch.Enabled, "new channel should be enabled by default")
	assert.Equal(t, entity.ConnectionStatusDisconnected, ch.ConnectionStatus)
	assert.Equal(t, "blue", ch.Config["color"])
	assert.Equal(t, "secret", ch.Credentials["api_key"])
	assert.False(t, ch.CreatedAt.IsZero())
	assert.False(t, ch.UpdatedAt.IsZero())

	// Verify it was persisted
	assert.Len(t, repo.Channels, 1)
}

func TestChannelService_Create_MinimalInput(t *testing.T) {
	svc, _, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1",
		Type:     "email",
		Name:     "Email",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, ch.ID)
	assert.Equal(t, entity.ChannelTypeEmail, ch.Type)
}

func TestChannelService_Create_RepoError(t *testing.T) {
	svc, repo, _ := newChannelService()
	repo.ReturnError = assert.AnError

	_, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1",
		Type:     "webchat",
		Name:     "Chat",
	})

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestChannelService_GetByID_Found(t *testing.T) {
	svc, _, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1",
		Type:     "webchat",
		Name:     "Chat",
	})
	require.NoError(t, err)

	found, err := svc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Chat", found.Name)
}

func TestChannelService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	_, err := svc.GetByID(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// List (by tenant)
// ---------------------------------------------------------------------------

func TestChannelService_List(t *testing.T) {
	svc, _, _ := newChannelService()

	_, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat 1",
	})
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "email", Name: "Email 1",
	})
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant2", Type: "webchat", Name: "Other Tenant Chat",
	})
	require.NoError(t, err)

	channels, err := svc.List(context.Background(), "tenant1")
	require.NoError(t, err)
	assert.Len(t, channels, 2)
}

func TestChannelService_List_Empty(t *testing.T) {
	svc, _, _ := newChannelService()

	channels, err := svc.List(context.Background(), "tenant-no-channels")
	require.NoError(t, err)
	assert.Empty(t, channels)
}

func TestChannelService_List_NilRepo(t *testing.T) {
	svc := &ChannelService{repo: nil}

	_, err := svc.List(context.Background(), "tenant1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestChannelService_Update(t *testing.T) {
	svc, _, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:   "tenant1",
		Type:       "webchat",
		Name:       "Old Name",
		Identifier: "old-id",
	})
	require.NoError(t, err)

	newName := "New Name"
	newIdent := "new-id"
	updated, err := svc.Update(context.Background(), created.ID, &UpdateChannelInput{
		Name:        &newName,
		Identifier:  &newIdent,
		Config:      map[string]string{"theme": "dark"},
		Credentials: map[string]string{"token": "new-token"},
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "new-id", updated.Identifier)
	assert.Equal(t, "dark", updated.Config["theme"])
	assert.Equal(t, "new-token", updated.Credentials["token"])
	assert.True(t, updated.UpdatedAt.After(created.CreatedAt) || updated.UpdatedAt.Equal(created.CreatedAt))
}

func TestChannelService_Update_PartialFields(t *testing.T) {
	svc, _, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID:   "tenant1",
		Type:       "webchat",
		Name:       "Chat",
		Identifier: "widget-1",
	})
	require.NoError(t, err)

	// Only update Name, leave Identifier untouched
	newName := "Updated Chat"
	updated, err := svc.Update(context.Background(), created.ID, &UpdateChannelInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Chat", updated.Name)
	assert.Equal(t, "widget-1", updated.Identifier, "identifier should remain unchanged")
}

func TestChannelService_Update_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	newName := "Whatever"
	_, err := svc.Update(context.Background(), "nonexistent", &UpdateChannelInput{
		Name: &newName,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestChannelService_Delete(t *testing.T) {
	svc, repo, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)
	assert.Len(t, repo.Channels, 1)

	err = svc.Delete(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Len(t, repo.Channels, 0)
}

func TestChannelService_Delete_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	// The mock's Delete removes from map silently even if not present (no error for missing key).
	// This is consistent with the mock behavior.
	err := svc.Delete(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UpdateEnabled (Enable / Disable)
// ---------------------------------------------------------------------------

func TestChannelService_UpdateEnabled_Enable(t *testing.T) {
	svc, repo, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// Disable first via repo to set up scenario
	repo.Channels[created.ID].Enabled = false

	ch, err := svc.UpdateEnabled(context.Background(), created.ID, true)
	require.NoError(t, err)
	assert.True(t, ch.Enabled)
}

func TestChannelService_UpdateEnabled_Disable(t *testing.T) {
	svc, _, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)
	assert.True(t, created.Enabled, "new channel should be enabled")

	ch, err := svc.UpdateEnabled(context.Background(), created.ID, false)
	require.NoError(t, err)
	assert.False(t, ch.Enabled)
}

func TestChannelService_UpdateEnabled_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	_, err := svc.UpdateEnabled(context.Background(), "nonexistent", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// UpdateStatus (backwards compatibility: "active" / "inactive")
// ---------------------------------------------------------------------------

func TestChannelService_UpdateStatus_Active(t *testing.T) {
	svc, repo, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)
	repo.Channels[created.ID].Enabled = false

	ch, err := svc.UpdateStatus(context.Background(), created.ID, "active")
	require.NoError(t, err)
	assert.True(t, ch.Enabled)
}

func TestChannelService_UpdateStatus_Inactive(t *testing.T) {
	svc, _, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	ch, err := svc.UpdateStatus(context.Background(), created.ID, "inactive")
	require.NoError(t, err)
	assert.False(t, ch.Enabled)
}

func TestChannelService_UpdateStatus_InvalidStatus(t *testing.T) {
	svc, _, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	_, err = svc.UpdateStatus(context.Background(), created.ID, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestChannelService_UpdateStatus_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	_, err := svc.UpdateStatus(context.Background(), "nonexistent", "active")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Connect (non-WhatsApp channels -- default path marks as connected)
// ---------------------------------------------------------------------------

func TestChannelService_Connect_NonWhatsApp(t *testing.T) {
	svc, repo, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusDisconnected, created.ConnectionStatus)

	result, err := svc.Connect(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotNil(t, result.Channel)
	assert.Equal(t, entity.ConnectionStatusConnected, result.Channel.ConnectionStatus)
	assert.Empty(t, result.QRCode)

	// Verify repo was updated
	assert.Equal(t, entity.ConnectionStatusConnected, repo.Channels[created.ID].ConnectionStatus)
}

func TestChannelService_Connect_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	_, err := svc.Connect(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Disconnect
// ---------------------------------------------------------------------------

func TestChannelService_Disconnect(t *testing.T) {
	svc, repo, _ := newChannelService()

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// First connect the channel
	_, err = svc.Connect(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusConnected, repo.Channels[created.ID].ConnectionStatus)

	// Now disconnect
	err = svc.Disconnect(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusDisconnected, repo.Channels[created.ID].ConnectionStatus)
}

func TestChannelService_Disconnect_NotFound(t *testing.T) {
	svc, _, _ := newChannelService()

	err := svc.Disconnect(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Disconnect with nil registry (code path guard)
// ---------------------------------------------------------------------------

func TestChannelService_Disconnect_NilRegistry(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	svc := NewChannelService(repo, nil, producer)

	created, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// Connect via repo directly since Connect for webchat uses repo
	repo.Channels[created.ID].ConnectionStatus = entity.ConnectionStatusConnected

	err = svc.Disconnect(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusDisconnected, repo.Channels[created.ID].ConnectionStatus)
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewChannelService(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	registry := plugin.NewRegistry()

	svc := NewChannelService(repo, registry, producer)
	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, registry, svc.registry)
	assert.Equal(t, producer, svc.producer)
}

func TestNewChannelService_NilRegistry(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()

	svc := NewChannelService(repo, nil, producer)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.registry)
}

// ---------------------------------------------------------------------------
// Multiple operations integration-style
// ---------------------------------------------------------------------------

func TestChannelService_CreateListUpdateDelete(t *testing.T) {
	svc, _, _ := newChannelService()
	ctx := context.Background()

	// Create
	ch, err := svc.Create(ctx, &CreateChannelInput{
		TenantID: "tenant1", Type: "telegram", Name: "TG Bot",
		Identifier: "bot123",
	})
	require.NoError(t, err)
	id := ch.ID

	// List
	channels, err := svc.List(ctx, "tenant1")
	require.NoError(t, err)
	assert.Len(t, channels, 1)

	// Update
	newName := "Updated TG Bot"
	updated, err := svc.Update(ctx, id, &UpdateChannelInput{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Updated TG Bot", updated.Name)

	// Delete
	err = svc.Delete(ctx, id)
	require.NoError(t, err)

	// Verify deleted
	_, err = svc.GetByID(ctx, id)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// DefaultListParams helper
// ---------------------------------------------------------------------------

func TestDefaultListParams(t *testing.T) {
	params := DefaultListParams()
	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 100, params.PageSize)
	assert.Equal(t, "created_at", params.SortBy)
	assert.Equal(t, "desc", params.SortDir)
}

// ---------------------------------------------------------------------------
// Connect and Disconnect round-trip for non-WhatsApp
// ---------------------------------------------------------------------------

func TestChannelService_ConnectDisconnect_RoundTrip(t *testing.T) {
	svc, repo, _ := newChannelService()
	ctx := context.Background()

	ch, err := svc.Create(ctx, &CreateChannelInput{
		TenantID: "tenant1", Type: "email", Name: "Email Channel",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusDisconnected, ch.ConnectionStatus)

	// Connect
	result, err := svc.Connect(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusConnected, result.Channel.ConnectionStatus)
	assert.Equal(t, entity.ConnectionStatusConnected, repo.Channels[ch.ID].ConnectionStatus)

	// Disconnect
	err = svc.Disconnect(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.ConnectionStatusDisconnected, repo.Channels[ch.ID].ConnectionStatus)
}

// ---------------------------------------------------------------------------
// Update with nil optional fields leaves values unchanged
// ---------------------------------------------------------------------------

func TestChannelService_Update_NilFieldsPreserved(t *testing.T) {
	svc, _, _ := newChannelService()
	ctx := context.Background()

	ch, err := svc.Create(ctx, &CreateChannelInput{
		TenantID:    "tenant1",
		Type:        "webchat",
		Name:        "Original",
		Identifier:  "orig-ident",
		Config:      map[string]string{"key": "val"},
		Credentials: map[string]string{"secret": "pw"},
	})
	require.NoError(t, err)

	// Update with all-nil optional fields
	updated, err := svc.Update(ctx, ch.ID, &UpdateChannelInput{})
	require.NoError(t, err)
	assert.Equal(t, "Original", updated.Name)
	assert.Equal(t, "orig-ident", updated.Identifier)
	assert.Equal(t, "val", updated.Config["key"])
	assert.Equal(t, "pw", updated.Credentials["secret"])
}

// ---------------------------------------------------------------------------
// Repo error propagation
// ---------------------------------------------------------------------------

func TestChannelService_List_RepoError(t *testing.T) {
	svc, repo, _ := newChannelService()
	repo.ReturnError = assert.AnError

	_, err := svc.List(context.Background(), "tenant1")
	assert.Error(t, err)
}

func TestChannelService_GetByID_RepoError(t *testing.T) {
	svc, repo, _ := newChannelService()
	repo.ReturnError = assert.AnError

	_, err := svc.GetByID(context.Background(), "any-id")
	assert.Error(t, err)
}

func TestChannelService_Update_RepoUpdateError(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// Set error after creation
	repo.ReturnError = assert.AnError

	newName := "Updated"
	_, err = svc.Update(context.Background(), ch.ID, &UpdateChannelInput{Name: &newName})
	assert.Error(t, err)
}

func TestChannelService_Delete_RepoError(t *testing.T) {
	svc, repo, _ := newChannelService()
	repo.ReturnError = assert.AnError

	err := svc.Delete(context.Background(), "any-id")
	assert.Error(t, err)
}

func TestChannelService_UpdateEnabled_RepoUpdateEnabledError(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// Set error after creation
	repo.ReturnError = assert.AnError

	_, err = svc.UpdateEnabled(context.Background(), ch.ID, false)
	assert.Error(t, err)
}

func TestChannelService_Connect_RepoUpdateConnectionStatusError(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// Set error after creation so UpdateConnectionStatus fails
	repo.ReturnError = assert.AnError

	_, err = svc.Connect(context.Background(), ch.ID)
	assert.Error(t, err)
}

func TestChannelService_Disconnect_RepoUpdateConnectionStatusError(t *testing.T) {
	svc, repo, _ := newChannelService()

	ch, err := svc.Create(context.Background(), &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat",
	})
	require.NoError(t, err)

	// Set error after creation
	repo.ReturnError = assert.AnError

	err = svc.Disconnect(context.Background(), ch.ID)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Create generates unique IDs
// ---------------------------------------------------------------------------

func TestChannelService_Create_UniqueIDs(t *testing.T) {
	svc, _, _ := newChannelService()
	ctx := context.Background()

	ch1, err := svc.Create(ctx, &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat 1",
	})
	require.NoError(t, err)

	ch2, err := svc.Create(ctx, &CreateChannelInput{
		TenantID: "tenant1", Type: "webchat", Name: "Chat 2",
	})
	require.NoError(t, err)

	assert.NotEqual(t, ch1.ID, ch2.ID)
}
