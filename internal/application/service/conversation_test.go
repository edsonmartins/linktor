package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupConversationTest() (*ConversationService, *testutil.MockConversationRepository) {
	convRepo := testutil.NewMockConversationRepository()
	contactRepo := testutil.NewMockContactRepository()
	channelRepo := testutil.NewMockChannelRepository()

	// Add fixtures
	contactRepo.Contacts["contact1"] = &entity.Contact{ID: "contact1", TenantID: "tenant1", Name: "Test"}
	channelRepo.Channels["channel1"] = &entity.Channel{ID: "channel1", TenantID: "tenant1", Type: entity.ChannelTypeWhatsApp}

	svc := NewConversationService(convRepo, contactRepo, channelRepo, nil)
	return svc, convRepo
}

func TestConversationService_List_EnrichesContactAndChannel(t *testing.T) {
	svc, _ := setupConversationTest()
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})
	require.NoError(t, err)

	// The repository scans only conversation columns; the service must attach the
	// contact and channel so the UI can label rows (not show "unknown").
	convs, _, err := svc.List(ctx, "tenant1", nil, nil)
	require.NoError(t, err)
	require.Len(t, convs, 1)
	require.NotNil(t, convs[0].Contact, "contact relation must be enriched")
	assert.Equal(t, "Test", convs[0].Contact.Name)
	require.NotNil(t, convs[0].Channel, "channel relation must be enriched")
	assert.Equal(t, entity.ChannelTypeWhatsApp, convs[0].Channel.Type)
}

func TestConversationService_Create(t *testing.T) {
	svc, _ := setupConversationTest()

	conv, err := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	assert.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, entity.ConversationStatusOpen, conv.Status)
}

func TestConversationService_Create_MissingContact(t *testing.T) {
	svc, _ := setupConversationTest()

	_, err := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ChannelID: "channel1",
	})

	assert.Error(t, err)
}

func TestConversationService_Resolve(t *testing.T) {
	svc, convRepo := setupConversationTest()

	// Create conversation
	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	resolved, err := svc.Resolve(context.Background(), conv.ID)
	assert.NoError(t, err)
	assert.Equal(t, entity.ConversationStatusResolved, resolved.Status)

	// Verify in repo
	stored := convRepo.Conversations[conv.ID]
	assert.Equal(t, entity.ConversationStatusResolved, stored.Status)
}

func TestConversationService_Reopen(t *testing.T) {
	svc, _ := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	svc.Resolve(context.Background(), conv.ID)
	reopened, err := svc.Reopen(context.Background(), conv.ID)
	assert.NoError(t, err)
	assert.Equal(t, entity.ConversationStatusOpen, reopened.Status)
}

func TestConversationService_Update_RejectsInvalidStatus(t *testing.T) {
	svc, convRepo := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	bad := "banana"
	_, err := svc.Update(context.Background(), conv.ID, &UpdateConversationInput{Status: &bad})
	assert.Error(t, err)

	// The invalid status must not have been persisted.
	stored := convRepo.Conversations[conv.ID]
	assert.Equal(t, entity.ConversationStatusOpen, stored.Status)
}

func TestConversationService_Update_RejectsInvalidPriority(t *testing.T) {
	svc, convRepo := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	bad := "banana"
	_, err := svc.Update(context.Background(), conv.ID, &UpdateConversationInput{Priority: &bad})
	assert.Error(t, err)

	stored := convRepo.Conversations[conv.ID]
	assert.Equal(t, entity.ConversationPriorityNormal, stored.Priority)
}

func TestConversationService_Update_AcceptsValidStatus(t *testing.T) {
	svc, _ := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	good := string(entity.ConversationStatusPending)
	updated, err := svc.Update(context.Background(), conv.ID, &UpdateConversationInput{Status: &good})
	assert.NoError(t, err)
	assert.Equal(t, entity.ConversationStatusPending, updated.Status)
}

func TestConversationService_Create_RejectsInvalidPriority(t *testing.T) {
	svc, _ := setupConversationTest()

	_, err := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
		Priority:  "banana",
	})
	assert.Error(t, err)
}

func TestConversationService_Assign_RejectsEmptyUser(t *testing.T) {
	svc, _ := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	_, err := svc.Assign(context.Background(), conv.ID, "")
	assert.Error(t, err)
}
