//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Roda com: go test -tags=integration ./internal/infrastructure/database/ -run TestConversationOwnerRule
func TestConversationOwnerRuleRepository_Postgres(t *testing.T) {
	ctx := context.Background()
	db, err := NewPostgresDB(testDBConfig(t))
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.RunMigrations(ctx))

	channels := NewChannelRepository(db, nil)
	repo := NewConversationOwnerRuleRepository(db)
	tenantA, tenantB := seedTenant(t, ctx, db), seedTenant(t, ctx, db)
	chA := newTestChannel(tenantA, entity.ChannelEnvironmentProduction)
	chB := newTestChannel(tenantB, entity.ChannelEnvironmentProduction)
	require.NoError(t, channels.Create(ctx, chA))
	require.NoError(t, channels.Create(ctx, chB))
	defer func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM tenants WHERE id IN ($1, $2)`, tenantA, tenantB)
	}()

	contato := "+5511987654321"
	desde := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	regra := func(tenant, canal, consumidor, ref string, d time.Time) *entity.ConversationOwnerRule {
		return &entity.ConversationOwnerRule{TenantID: tenant, ChannelID: canal, ContactIdentifier: contato,
			Consumer: consumidor, ConversaRef: ref, Desde: d}
	}

	applied, cur, err := repo.Upsert(ctx, regra(tenantA, chA.ID, "copilot", "conv-1", desde))
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, "copilot", cur.Consumer)

	// Mesma conversa de novo: aplica (idempotente). Outra conversa mais antiga: não sobrescreve.
	applied, _, err = repo.Upsert(ctx, regra(tenantA, chA.ID, "copilot", "conv-1", desde))
	require.NoError(t, err)
	assert.True(t, applied)
	applied, cur, err = repo.Upsert(ctx, regra(tenantA, chA.ID, "crm", "conv-velha", desde.Add(-time.Hour)))
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, "conv-1", cur.ConversaRef)

	// O mesmo contato noutro tenant é outra regra.
	got, err := repo.Find(ctx, tenantB, chB.ID, contato)
	require.NoError(t, err)
	assert.Nil(t, got)
	got, err = repo.Find(ctx, tenantB, chA.ID, contato)
	require.NoError(t, err)
	assert.Nil(t, got, "o canal de A pelo tenant B não acha nada")

	// Liberação só da conversa dona; a atrasada não apaga.
	ok, err := repo.Release(ctx, tenantA, chA.ID, contato, "conv-velha")
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = repo.Release(ctx, tenantB, chA.ID, contato, "conv-1")
	require.NoError(t, err)
	assert.False(t, ok, "outro tenant não libera")
	ok, err = repo.Release(ctx, tenantA, chA.ID, contato, "conv-1")
	require.NoError(t, err)
	assert.True(t, ok)
	got, err = repo.Find(ctx, tenantA, chA.ID, contato)
	require.NoError(t, err)
	assert.Nil(t, got)
}
