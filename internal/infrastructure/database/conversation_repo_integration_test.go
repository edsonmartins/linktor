//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// seedConversationGraph inserts the minimal tenant/channel/contact rows a
// conversation references and returns their IDs.
func seedConversationGraph(t *testing.T, ctx context.Context, db *PostgresDB) (tenantID, channelID, contactID string) {
	t.Helper()

	tenantID = uuid.New().String()
	channelID = uuid.New().String()
	contactID = uuid.New().String()
	slug := "t-" + uuid.New().String()[:8]

	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, "test", slug); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO channels (id, tenant_id, name, type) VALUES ($1, $2, $3, $4)`,
		channelID, tenantID, "ch", "whatsapp"); err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO contacts (id, tenant_id, name) VALUES ($1, $2, $3)`,
		contactID, tenantID, "contact"); err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	return tenantID, channelID, contactID
}

// TestConversationTagsMetadata_RoundTrip is the regression guard for
// WS10-PERSIST-FIELDS: tags and metadata set on a conversation must survive
// Create -> Update -> FindByID instead of being silently dropped by the repo.
func TestConversationTagsMetadata_RoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewConversationRepository(db)
	tenantID, channelID, contactID := seedConversationGraph(t, ctx, db)

	// Create with initial tags + metadata.
	conv := entity.NewConversation(tenantID, contactID, channelID)
	conv.ID = uuid.New().String()
	conv.Tags = []string{"vip", "billing"}
	conv.Metadata = map[string]string{"source": "webchat", "priority_reason": "sla"}

	if err := repo.Create(ctx, conv); err != nil {
		t.Fatalf("create: %v", err)
	}

	created, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find after create: %v", err)
	}
	assertStringSet(t, "tags after create", created.Tags, []string{"vip", "billing"})
	if got := created.Metadata["source"]; got != "webchat" {
		t.Fatalf("metadata source after create: want webchat, got %q", got)
	}
	if got := created.Metadata["priority_reason"]; got != "sla" {
		t.Fatalf("metadata priority_reason after create: want sla, got %q", got)
	}

	// Update tags (the primary fix path) and mutate metadata.
	created.Tags = []string{"vip", "resolved-later", "escalated"}
	created.Metadata["escalation_reason"] = "timeout"
	if err := repo.Update(ctx, created); err != nil {
		t.Fatalf("update: %v", err)
	}

	reloaded, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find after update: %v", err)
	}
	assertStringSet(t, "tags after update", reloaded.Tags,
		[]string{"vip", "resolved-later", "escalated"})
	if got := reloaded.Metadata["escalation_reason"]; got != "timeout" {
		t.Fatalf("metadata escalation_reason after update: want timeout, got %q", got)
	}
	if got := reloaded.Metadata["source"]; got != "webchat" {
		t.Fatalf("metadata source should persist through update: want webchat, got %q", got)
	}
}

// TestConversationTagsMetadata_EmptyDefaults verifies a conversation created
// with no tags/metadata reads back as empty (non-nil) values, never a nil map
// or a scan panic on NULL columns.
func TestConversationTagsMetadata_EmptyDefaults(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewConversationRepository(db)
	tenantID, channelID, contactID := seedConversationGraph(t, ctx, db)

	conv := entity.NewConversation(tenantID, contactID, channelID)
	conv.ID = uuid.New().String()
	if err := repo.Create(ctx, conv); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Fatalf("empty tags expected, got %v", got.Tags)
	}
	if got.Metadata == nil {
		t.Fatalf("metadata must default to a non-nil map")
	}
	if len(got.Metadata) != 0 {
		t.Fatalf("empty metadata expected, got %v", got.Metadata)
	}
}

func assertStringSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: want %v, got %v", label, want, got)
	}
	wantSet := make(map[string]bool, len(want))
	for _, w := range want {
		wantSet[w] = true
	}
	for _, g := range got {
		if !wantSet[g] {
			t.Fatalf("%s: unexpected value %q (want %v, got %v)", label, g, want, got)
		}
	}
}

// A listagem paginada contra o Postgres de verdade: a ordem é a da atividade,
// o cursor percorre a caixa sem repetir, a prévia vem junto e a busca estreita.
// O teste unitário usa mock e não vê o SQL — CTE materializado, LATERAL e
// comparação de tupla só se provam aqui.
func TestConversationPage_KeysetOrderingAndPreview(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewConversationRepository(db)
	msgRepo := NewMessageRepository(db)
	tenantID, channelID, contactID := seedConversationGraph(t, ctx, db)

	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	// Cinco conversas criadas na mesma ordem, mas com atividade invertida: a
	// primeira criada é a que recebeu a mensagem mais recente. Se a ordenação
	// ainda fosse por created_at ou updated_at, a lista sairia ao contrário.
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		conv := entity.NewConversation(tenantID, contactID, channelID)
		conv.ID = uuid.New().String()
		if err := repo.Create(ctx, conv); err != nil {
			t.Fatalf("create conversa %d: %v", i, err)
		}
		ids[i] = conv.ID

		msg := &entity.Message{
			ID:             uuid.New().String(),
			ConversationID: conv.ID,
			SenderType:     entity.SenderTypeContact,
			ContentType:    entity.ContentTypeText,
			Content:        fmt.Sprintf("fala da conversa %d", i),
			Status:         entity.MessageStatusDelivered,
			CreatedAt:      base.Add(time.Duration(5-i) * time.Minute),
		}
		if err := msgRepo.Create(ctx, msg); err != nil {
			t.Fatalf("create mensagem %d: %v", i, err)
		}
	}

	// O gatilho precisa ter gravado a marca de atividade; sem ela a ordenação
	// cai de volta para a criação e a caixa mente.
	primeira, err := repo.FindByID(ctx, ids[0])
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if primeira.LastMessageAt == nil {
		t.Fatal("o gatilho não gravou last_message_at")
	}
	if !primeira.LastMessageAt.Equal(base.Add(5 * time.Minute)) {
		t.Fatalf("marca de atividade errada: %v", primeira.LastMessageAt)
	}

	vistas := map[string]bool{}
	var ordem []string
	var cursor *repository.ConversationCursor
	for passo := 0; passo < 4; passo++ {
		pagina, err := repo.FindPageByTenant(ctx, tenantID, cursor, 2, nil)
		if err != nil {
			t.Fatalf("página %d: %v", passo, err)
		}
		if len(pagina) == 0 {
			break
		}
		for _, c := range pagina {
			if vistas[c.ID] {
				t.Fatalf("conversa %s repetida entre páginas", c.ID)
			}
			vistas[c.ID] = true
			ordem = append(ordem, c.ID)
			if c.LastMessage == nil {
				t.Fatalf("conversa %s veio sem a prévia da última mensagem", c.ID)
			}
		}
		ultima := pagina[len(pagina)-1]
		at := ultima.CreatedAt
		if ultima.LastMessageAt != nil {
			at = *ultima.LastMessageAt
		}
		cursor = &repository.ConversationCursor{ActivityAt: at, ID: ultima.ID}
	}

	if len(ordem) != 5 {
		t.Fatalf("esperava as 5 conversas, vieram %d", len(ordem))
	}
	if ordem[0] != ids[0] {
		t.Fatal("a conversa com a mensagem mais recente tem de abrir a lista")
	}
	if ordem[4] != ids[4] {
		t.Fatal("a conversa mais parada tem de fechar a lista")
	}

	// A prévia é a última mensagem daquela conversa, não a de outra.
	pagina, err := repo.FindPageByTenant(ctx, tenantID, nil, 1, nil)
	if err != nil {
		t.Fatalf("primeira página: %v", err)
	}
	if pagina[0].LastMessage.Content != "fala da conversa 0" {
		t.Fatalf("prévia errada: %q", pagina[0].LastMessage.Content)
	}

	// Busca pelo contato: o nome semeado é "contact".
	achadas, err := repo.FindPageByTenant(ctx, tenantID, nil, 10, map[string]interface{}{"search": "ontac"})
	if err != nil {
		t.Fatalf("busca: %v", err)
	}
	if len(achadas) != 5 {
		t.Fatalf("a busca pelo nome do contato devia achar as 5, achou %d", len(achadas))
	}
	nenhuma, err := repo.FindPageByTenant(ctx, tenantID, nil, 10, map[string]interface{}{"search": "zzz-ninguem"})
	if err != nil {
		t.Fatalf("busca vazia: %v", err)
	}
	if len(nenhuma) != 0 {
		t.Fatalf("termo sem dono devia achar nada, achou %d", len(nenhuma))
	}

	total, err := repo.CountByTenantFiltered(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("contagem: %v", err)
	}
	if total != 5 {
		t.Fatalf("total esperado 5, veio %d", total)
	}
}

// Mensagem antiga gravada depois (importação de histórico) não pode puxar a
// marca de atividade para trás.
func TestConversationActivity_OlderMessageDoesNotRewind(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewConversationRepository(db)
	msgRepo := NewMessageRepository(db)
	tenantID, channelID, contactID := seedConversationGraph(t, ctx, db)

	conv := entity.NewConversation(tenantID, contactID, channelID)
	conv.ID = uuid.New().String()
	if err := repo.Create(ctx, conv); err != nil {
		t.Fatalf("create: %v", err)
	}

	recente := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	antiga := recente.Add(-72 * time.Hour)

	for _, quando := range []time.Time{recente, antiga} {
		msg := &entity.Message{
			ID:             uuid.New().String(),
			ConversationID: conv.ID,
			SenderType:     entity.SenderTypeContact,
			ContentType:    entity.ContentTypeText,
			Content:        "oi",
			Status:         entity.MessageStatusDelivered,
			CreatedAt:      quando,
		}
		if err := msgRepo.Create(ctx, msg); err != nil {
			t.Fatalf("create mensagem: %v", err)
		}
	}

	atual, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if atual.LastMessageAt == nil || !atual.LastMessageAt.Equal(recente) {
		t.Fatalf("a marca andou para trás: %v", atual.LastMessageAt)
	}
}
