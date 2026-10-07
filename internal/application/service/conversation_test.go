package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
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

func TestConversationService_Delete(t *testing.T) {
	svc, convRepo := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	err := svc.Delete(context.Background(), conv.ID)
	require.NoError(t, err)
	assert.NotContains(t, convRepo.Conversations, conv.ID)
}

func TestConversationService_Delete_NotFound(t *testing.T) {
	svc, _ := setupConversationTest()

	err := svc.Delete(context.Background(), "nope")
	require.Error(t, err)
	assert.Equal(t, errors.ErrCodeConversationNotFound, errors.GetAppError(err).Code)
}

// A conversa de outro tenant não é apagada: o acesso cruzado responde o mesmo
// 404 de inexistente, para não revelar que o id existe em outro lugar.
func TestConversationService_DeleteForTenant_OtherTenant(t *testing.T) {
	svc, convRepo := setupConversationTest()

	conv, _ := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})

	err := svc.DeleteForTenant(context.Background(), "tenant2", conv.ID)
	require.Error(t, err)
	assert.Equal(t, errors.ErrCodeConversationNotFound, errors.GetAppError(err).Code)
	assert.Contains(t, convRepo.Conversations, conv.ID)
}

// O evento sai com a identidade da conversa que acabou de ser apagada — depois
// do delete não há mais linha de onde ler canal, contato e status.
func TestConversationService_Delete_PublishesEvent(t *testing.T) {
	convRepo := testutil.NewMockConversationRepository()
	contactRepo := testutil.NewMockContactRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo.Contacts["contact1"] = &entity.Contact{ID: "contact1", TenantID: "tenant1", Name: "Test"}
	channelRepo.Channels["channel1"] = &entity.Channel{ID: "channel1", TenantID: "tenant1", Type: entity.ChannelTypeWhatsApp}
	producer := testutil.NewMockProducer()
	svc := NewConversationService(convRepo, contactRepo, channelRepo, producer)

	conv, err := svc.Create(context.Background(), &CreateConversationInput{
		TenantID:  "tenant1",
		ContactID: "contact1",
		ChannelID: "channel1",
	})
	require.NoError(t, err)
	producer.Events = nil

	require.NoError(t, svc.DeleteForTenant(context.Background(), "tenant1", conv.ID))

	require.Len(t, producer.Events, 1)
	event := producer.Events[0]
	assert.Equal(t, nats.EventConversationDeleted, event.Type)
	assert.Equal(t, "tenant1", event.TenantID)
	assert.Equal(t, conv.ID, event.Payload["conversation_id"])
	assert.Equal(t, "channel1", event.Payload["channel_id"])
	assert.Equal(t, "contact1", event.Payload["contact_id"])
}
