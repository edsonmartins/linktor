package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
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
	page, err := svc.ListPage(ctx, "tenant1", nil, nil, 0)
	require.NoError(t, err)
	require.Len(t, page.Conversations, 1)
	require.NotNil(t, page.Conversations[0].Contact, "contact relation must be enriched")
	assert.Equal(t, "Test", page.Conversations[0].Contact.Name)
	require.NotNil(t, page.Conversations[0].Channel, "channel relation must be enriched")
	assert.Equal(t, entity.ChannelTypeWhatsApp, page.Conversations[0].Channel.Type)
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

// seedConversationAt cria uma conversa com marca de atividade explícita.
func seedConversationAt(repo *testutil.MockConversationRepository, id, tenantID string, at time.Time) *entity.Conversation {
	conv := &entity.Conversation{
		ID: id, TenantID: tenantID, ContactID: "contact1", ChannelID: "channel1",
		Status: entity.ConversationStatusOpen, CreatedAt: at, UpdatedAt: at,
		LastMessageAt: &at,
	}
	repo.Conversations[id] = conv
	return conv
}

// A caixa é percorrida de página em página, da mais ativa para a mais parada,
// sem repetir nem perder conversa.
func TestConversationService_ListPage_WalksTheWholeInbox(t *testing.T) {
	svc, repo := setupConversationTest()
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	for i := 1; i <= 5; i++ {
		seedConversationAt(repo, fmt.Sprintf("conv-%d", i), "tenant1", base.Add(time.Duration(i)*time.Minute))
	}

	vistas := map[string]bool{}
	var cursor *repository.ConversationCursor
	for passo := 0; passo < 3; passo++ {
		page, err := svc.ListPage(context.Background(), "tenant1", nil, cursor, 2)
		require.NoError(t, err)

		for _, c := range page.Conversations {
			require.False(t, vistas[c.ID], "conversa %s apareceu duas vezes", c.ID)
			vistas[c.ID] = true
		}

		if passo == 0 {
			assert.Equal(t, "conv-5", page.Conversations[0].ID, "a mais ativa abre a lista")
			assert.Equal(t, int64(5), page.Total, "a primeira página conta o total")
		} else {
			assert.Zero(t, page.Total, "contar a cada rolagem custaria varrer a caixa")
		}

		if !page.HasMore {
			break
		}
		ultima := page.Conversations[len(page.Conversations)-1]
		cursor = &repository.ConversationCursor{ActivityAt: *ultima.LastMessageAt, ID: ultima.ID}
	}

	assert.Len(t, vistas, 5, "as cinco conversas saíram, cada uma uma vez")
}

// A conversa sem mensagem nenhuma se ordena pela criação — senão sumiria do
// fim da lista, ou apareceria sempre no topo.
func TestConversationService_ListPage_ConversationWithoutMessages(t *testing.T) {
	svc, repo := setupConversationTest()
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	seedConversationAt(repo, "com-mensagem", "tenant1", base.Add(time.Hour))
	repo.Conversations["sem-mensagem"] = &entity.Conversation{
		ID: "sem-mensagem", TenantID: "tenant1", ContactID: "contact1", ChannelID: "channel1",
		Status: entity.ConversationStatusOpen, CreatedAt: base.Add(2 * time.Hour),
	}

	page, err := svc.ListPage(context.Background(), "tenant1", nil, nil, 10)
	require.NoError(t, err)
	require.Len(t, page.Conversations, 2)
	assert.Equal(t, "sem-mensagem", page.Conversations[0].ID,
		"criada depois da última mensagem da outra, então vem antes")
}

// O termo de busca precisa chegar ao repositório. Ele era aceito pela tela,
// enviado pela API e descartado no caminho — a lista voltava inteira.
func TestConversationService_ListPage_SearchReachesTheRepository(t *testing.T) {
	svc, repo := setupConversationTest()
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	seedConversationAt(repo, "da-maria", "tenant1", base).Subject = "Maria"
	seedConversationAt(repo, "do-joao", "tenant1", base.Add(time.Minute)).Subject = "João"

	page, err := svc.ListPage(context.Background(), "tenant1", &ConversationFilters{Search: "maria"}, nil, 10)
	require.NoError(t, err)
	require.Len(t, page.Conversations, 1)
	assert.Equal(t, "da-maria", page.Conversations[0].ID)
	assert.Equal(t, int64(1), page.Total, "o total também respeita a busca")
}

func TestConversationService_ListPage_RejectsInvalidEnvironment(t *testing.T) {
	svc, _ := setupConversationTest()

	_, err := svc.ListPage(context.Background(), "tenant1",
		&ConversationFilters{Environment: "producao"}, nil, 10)
	require.Error(t, err)
}

func TestConversationService_ListPage_CapsTheLimit(t *testing.T) {
	svc, repo := setupConversationTest()
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 120; i++ {
		seedConversationAt(repo, fmt.Sprintf("conv-%03d", i), "tenant1", base.Add(time.Duration(i)*time.Second))
	}

	page, err := svc.ListPage(context.Background(), "tenant1", nil, nil, 500)
	require.NoError(t, err)
	assert.Len(t, page.Conversations, 100, "o teto é 100 por página")
	assert.True(t, page.HasMore)
}
