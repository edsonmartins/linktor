//go:build integration

package database

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// ---------------------------------------------------------------------------
// Reflexive round-trip test (G-005: campo declarado é campo persistido).
//
// The same defect occurred three times: a field present on a domain entity that
// the repository silently drops (source/is_imported on messages;
// Identifier/MessageTemplateNamespace on channels). This is a process defect,
// so the cover is a generic net, not case-by-case fixes: every registered
// entity is fully populated, saved, reloaded from a REAL database and compared
// field by field. A failure names Entity.Field.
//
// Registering a new entity = appending one roundTripCase to roundTripCases.
// Every exported field must either round-trip or appear in `exceptions` WITH a
// justification — an empty justification fails the test.
// ---------------------------------------------------------------------------

type roundTripCase struct {
	entity string
	// exceptions maps exported field name -> REQUIRED justification for not
	// being round-trip-compared (computed, transient, persisted elsewhere, or
	// pending human decision — say which).
	exceptions map[string]string
	// run persists a fully-populated instance and returns (saved, loaded).
	run func(t *testing.T, ctx context.Context, db *PostgresDB, tenantID string) (any, any)
}

// fixedTime returns deterministic UTC times (no monotonic clock, microsecond
// precision) so equality survives Postgres TIMESTAMPTZ and JSON round-trips.
func fixedTime(min int) time.Time {
	return time.Date(2026, 7, 23, 10, min, 5, 123456000, time.UTC)
}
func fixedTimePtr(min int) *time.Time { t := fixedTime(min); return &t }

var roundTripCases = []roundTripCase{
	{
		entity:     "Channel",
		exceptions: map[string]string{
			// none — every Channel field must round-trip; Environment is
			// immutable on UPDATE by design but Create→FindByID covers it.
		},
		run: func(t *testing.T, ctx context.Context, db *PostgresDB, tenantID string) (any, any) {
			repo := NewChannelRepository(db, nil)
			ch := &entity.Channel{
				ID:                       uuid.New().String(),
				TenantID:                 tenantID,
				Type:                     entity.ChannelTypeWhatsAppOfficial,
				Name:                     "roundtrip",
				Identifier:               "+5511999999999",
				Enabled:                  true,
				ConnectionStatus:         entity.ConnectionStatusConnected,
				Environment:              entity.ChannelEnvironmentSandbox,
				Config:                   map[string]string{"phone_number_id": "111"},
				Credentials:              map[string]string{"credential_environment": "sandbox"},
				WebhookURL:               "https://example.com/hook",
				CreatedAt:                fixedTime(1),
				UpdatedAt:                fixedTime(2),
				IsCoexistence:            true,
				WABAID:                   "waba-1",
				LastEchoAt:               fixedTimePtr(3),
				CoexistenceStatus:        entity.CoexistenceStatusActive,
				MessageTemplateNamespace: "ns-123",
			}
			if err := repo.Create(ctx, ch); err != nil {
				t.Fatalf("create channel: %v", err)
			}
			got, err := repo.FindByID(ctx, ch.ID)
			if err != nil {
				t.Fatalf("find channel: %v", err)
			}
			return ch, got
		},
	},
	{
		entity: "Conversation",
		exceptions: map[string]string{
			"LastMessageAt": "calculado por subquery sobre messages no SELECT; não é coluna própria",
		},
		run: func(t *testing.T, ctx context.Context, db *PostgresDB, tenantID string) (any, any) {
			channelRepo := NewChannelRepository(db, nil)
			convRepo := NewConversationRepository(db)
			ch := newTestChannel(tenantID, entity.ChannelEnvironmentSandbox)
			if err := channelRepo.Create(ctx, ch); err != nil {
				t.Fatalf("seed channel: %v", err)
			}
			contactID := uuid.New().String()
			if _, err := db.Pool.Exec(ctx,
				`INSERT INTO contacts (id, tenant_id, name) VALUES ($1, $2, $3)`,
				contactID, tenantID, "rt-contact"); err != nil {
				t.Fatalf("seed contact: %v", err)
			}
			userID := uuid.New().String()
			if _, err := db.Pool.Exec(ctx,
				`INSERT INTO users (id, tenant_id, email, password_hash, name) VALUES ($1, $2, $3, 'x', 'rt-user')`,
				userID, tenantID, "rt-"+userID[:8]+"@test.local"); err != nil {
				t.Fatalf("seed user: %v", err)
			}

			conv := &entity.Conversation{
				ID:             uuid.New().String(),
				TenantID:       tenantID,
				ContactID:      contactID,
				ChannelID:      ch.ID,
				Environment:    entity.ChannelEnvironmentSandbox,
				AssignedUserID: &userID,
				Status:         entity.ConversationStatusPending,
				Priority:       entity.ConversationPriorityHigh,
				Subject:        "roundtrip subject",
				Tags:           []string{"a", "b"},
				Metadata:       map[string]string{"k": "v"},
				UnreadCount:    3,
				LastMessageAt:  fixedTimePtr(4),
				FirstReplyAt:   fixedTimePtr(5),
				ResolvedAt:     fixedTimePtr(6),
				CreatedAt:      fixedTime(7),
				UpdatedAt:      fixedTime(8),
			}
			if err := convRepo.Create(ctx, conv); err != nil {
				t.Fatalf("create conversation: %v", err)
			}
			got, err := convRepo.FindByID(ctx, conv.ID)
			if err != nil {
				t.Fatalf("find conversation: %v", err)
			}
			return conv, got
		},
	},
	{
		entity: "Message",
		exceptions: map[string]string{
			"Attachments": "persistidos em message_attachments via CreateAttachment e carregados por FindByID; round-trip coberto por testes próprios do message_repo (comparação de slice de ponteiros com timestamps é frágil aqui)",
			// --- Órfãos conhecidos, PENDENTES DE DECISÃO HUMANA (fase 0.1). ---
			// Colunas existem mas o repositório não as grava/lê:
			"Source":     "ÓRFÃO com coluna: repo não persiste. Pendente de decisão (pode ter razão histórica) — não corrigir sem decisão",
			"IsImported": "ÓRFÃO com coluna: repo não persiste. Pendente de decisão — idem Source",
			"ImportedAt": "ÓRFÃO com coluna: repo não persiste. Pendente de decisão — idem Source",
			// Campos sem coluna alguma (trafegam só em eventos NATS):
			"EditedAt":  "ÓRFÃO sem coluna: edição só trafega em evento NATS, nunca persistida. Pendente de decisão",
			"DeletedAt": "ÓRFÃO sem coluna: idem EditedAt",
			"IsEdited":  "ÓRFÃO sem coluna: idem EditedAt",
			"IsDeleted": "ÓRFÃO sem coluna: idem EditedAt",
			"ReplyToID": "ÓRFÃO sem coluna: reply só trafega em evento NATS. Pendente de decisão",
		},
		run: func(t *testing.T, ctx context.Context, db *PostgresDB, tenantID string) (any, any) {
			repo := NewMessageRepository(db)
			convID := seedConversation(t, ctx, db)
			msg := &entity.Message{
				ID:             uuid.New().String(),
				ConversationID: convID,
				SenderType:     entity.SenderTypeContact,
				SenderID:       uuid.New().String(),
				ContentType:    entity.ContentTypeText,
				Content:        "roundtrip body",
				Metadata:       map[string]string{"k": "v"},
				Status:         entity.MessageStatusDelivered,
				ExternalID:     "ext-" + uuid.New().String()[:8],
				ErrorMessage:   "err-note",
				SentAt:         fixedTimePtr(1),
				DeliveredAt:    fixedTimePtr(2),
				ReadAt:         fixedTimePtr(3),
				CreatedAt:      fixedTime(4),
				Reactions: []entity.Reaction{
					{UserID: uuid.New().String(), Emoji: "👍", Timestamp: fixedTime(5)},
				},
				// Orphan fields still populated so the exception list, not the
				// zero-check, is what documents them:
				Source:     entity.MessageSourceBusinessApp,
				IsImported: true,
				ImportedAt: fixedTimePtr(6),
				EditedAt:   fixedTimePtr(7),
				DeletedAt:  fixedTimePtr(8),
				IsEdited:   true,
				IsDeleted:  true,
				ReplyToID:  uuid.New().String(),
			}
			if err := repo.Create(ctx, msg); err != nil {
				t.Fatalf("create message: %v", err)
			}
			got, err := repo.FindByID(ctx, msg.ID)
			if err != nil {
				t.Fatalf("find message: %v", err)
			}
			return msg, got
		},
	},
	{
		entity:     "SandboxAllowlistEntry",
		exceptions: map[string]string{},
		run: func(t *testing.T, ctx context.Context, db *PostgresDB, tenantID string) (any, any) {
			channelRepo := NewChannelRepository(db, nil)
			repo := NewSandboxAllowlistRepository(db)
			ch := newTestChannel(tenantID, entity.ChannelEnvironmentSandbox)
			if err := channelRepo.Create(ctx, ch); err != nil {
				t.Fatalf("seed channel: %v", err)
			}
			e := &entity.SandboxAllowlistEntry{
				ID:        uuid.New().String(),
				TenantID:  tenantID,
				ChannelID: ch.ID,
				Recipient: "+5544999999999",
				Note:      "roundtrip",
				CreatedBy: uuid.New().String(),
				CreatedAt: fixedTime(9),
			}
			if err := repo.Create(ctx, e); err != nil {
				t.Fatalf("create allowlist entry: %v", err)
			}
			got, err := repo.FindByID(ctx, tenantID, e.ID)
			if err != nil {
				t.Fatalf("find allowlist entry: %v", err)
			}
			return e, got
		},
	},
}

// valuesEqual compares one field, treating time.Time and *time.Time by Equal
// (Postgres normalizes zone; DeepEqual would compare locations).
func valuesEqual(a, b reflect.Value) bool {
	switch av := a.Interface().(type) {
	case time.Time:
		return av.Equal(b.Interface().(time.Time))
	case *time.Time:
		bv := b.Interface().(*time.Time)
		if av == nil || bv == nil {
			return av == bv
		}
		return av.Equal(*bv)
	default:
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

func TestRoundTrip_AllPersistedEntities(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, tc := range roundTripCases {
		tc := tc
		t.Run(tc.entity, func(t *testing.T) {
			tenantID := seedTenant(t, ctx, db)
			saved, loaded := tc.run(t, ctx, db, tenantID)

			sv := reflect.ValueOf(saved).Elem()
			lv := reflect.ValueOf(loaded).Elem()
			st := sv.Type()

			seenExceptions := map[string]bool{}
			var failures []string
			for i := 0; i < st.NumField(); i++ {
				field := st.Field(i)
				if !field.IsExported() {
					continue
				}
				name := field.Name
				if why, ok := tc.exceptions[name]; ok {
					seenExceptions[name] = true
					if why == "" {
						failures = append(failures, fmt.Sprintf("%s.%s: exceção sem justificativa (G-005 exige comentário)", tc.entity, name))
					}
					continue
				}
				if sv.Field(i).IsZero() {
					failures = append(failures, fmt.Sprintf("%s.%s: campo não populado pelo caso de teste — popule com valor não-zero ou declare exceção justificada", tc.entity, name))
					continue
				}
				if !valuesEqual(sv.Field(i), lv.Field(i)) {
					failures = append(failures, fmt.Sprintf("%s.%s: não sobreviveu ao round-trip (gravado %#v, lido %#v)", tc.entity, name, sv.Field(i).Interface(), lv.Field(i).Interface()))
				}
			}
			// Stale exceptions rot the list: every declared exception must
			// still name a real field.
			for name := range tc.exceptions {
				if !seenExceptions[name] {
					failures = append(failures, fmt.Sprintf("%s.%s: exceção declarada para campo inexistente", tc.entity, name))
				}
			}
			for _, f := range failures {
				t.Error(f)
			}
		})
	}
}
