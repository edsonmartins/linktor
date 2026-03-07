package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func setupConversationTest() (*ConversationService, *testutil.MockConversationRepository) {
	convRepo := testutil.NewMockConversationRepository()
	contactRepo := testutil.NewMockContactRepository()
	channelRepo := testutil.NewMockChannelRepository()

	// Add fixtures
	contactRepo.Contacts["contact1"] = &entity.Contact{ID: "contact1", TenantID: "tenant1", Name: "Test"}
	channelRepo.Channels["channel1"] = &entity.Channel{ID: "channel1", TenantID: "tenant1", Type: entity.ChannelTypeWhatsApp}

	svc := NewConversationService(convRepo, contactRepo, channelRepo)
	return svc, convRepo
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
