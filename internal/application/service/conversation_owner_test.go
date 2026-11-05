package service

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P5 (VCR-36): conversation.owner_changed só quando o dono efetivo muda.
func novoServicoDono(t *testing.T, defaultName string) (*ConversationOwnerService, *testutil.MockProducer) {
	t.Helper()
	channels := testutil.NewMockChannelRepository()
	ch := &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: entity.ChannelTypeWhatsAppOfficial, Config: map[string]string{}}
	if defaultName != "" {
		ch.Config[entity.ChannelConfigDefaultConsumerName] = defaultName
	}
	channels.Channels["ch-1"] = ch
	pub := testutil.NewMockProducer()
	svc := NewConversationOwnerService(testutil.NewMockConversationOwnerRuleRepository(), channels).WithPublisher(pub)
	return svc, pub
}

func declara(t *testing.T, s *ConversationOwnerService, consumidor, ref string, desde time.Time) bool {
	t.Helper()
	applied, _, err := s.Declare(context.Background(), &DeclareOwnerInput{TenantID: "t-1", ChannelID: "ch-1",
		Identifier: "+5511987654321", Consumer: consumidor, ConversaRef: ref, Desde: desde})
	require.NoError(t, err)
	return applied
}

func eventosDono(pub *testutil.MockProducer) []*nats.Event {
	var out []*nats.Event
	for _, e := range pub.Events {
		if e.Type == nats.EventConversationOwnerChanged {
			out = append(out, e)
		}
	}
	return out
}

func TestOwnerChanged_SoQuandoODonoMuda(t *testing.T) {
	s, pub := novoServicoDono(t, "crm")
	t0 := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

	require.True(t, declara(t, s, "copilot", "conv-1", t0))
	ev := eventosDono(pub)
	require.Len(t, ev, 1, "sem regra (padrão crm) → copilot: muda")
	assert.Equal(t, "crm", ev[0].Payload["previous_consumer"])
	assert.Equal(t, "copilot", ev[0].Payload["consumer"])
	assert.Equal(t, "declarado", ev[0].Payload["motivo"])
	assert.Equal(t, "+5511987654321", ev[0].Payload["contact_identifier"])
	assert.Equal(t, "t-1", ev[0].TenantID)

	require.True(t, declara(t, s, "copilot", "conv-1", t0))
	assert.Len(t, eventosDono(pub), 1, "PUT idempotente não emite")
	require.True(t, declara(t, s, "copilot", "conv-2", t0.Add(time.Hour)))
	assert.Len(t, eventosDono(pub), 1, "conversa nova do mesmo dono não emite")
	require.False(t, declara(t, s, "crm", "conv-velha", t0.Add(-time.Hour)))
	assert.Len(t, eventosDono(pub), 1, "declaração atrasada não aplicada não emite")

	liberou, err := s.Release(context.Background(), "t-1", "ch-1", "+5511987654321", "conv-outra")
	require.NoError(t, err)
	require.False(t, liberou)
	assert.Len(t, eventosDono(pub), 1, "DELETE que não libera não emite")

	liberou, err = s.Release(context.Background(), "t-1", "ch-1", "+5511987654321", "conv-2")
	require.NoError(t, err)
	require.True(t, liberou)
	ev = eventosDono(pub)
	require.Len(t, ev, 2, "liberar o copilot volta ao padrão: muda")
	assert.Equal(t, "copilot", ev[1].Payload["previous_consumer"])
	assert.Equal(t, "crm", ev[1].Payload["consumer"])
	assert.Equal(t, "liberado", ev[1].Payload["motivo"])
	assert.Equal(t, "conv-2", ev[1].Payload["previous_conversa_ref"])
}

func TestOwnerChanged_DonoIgualAoPadraoNaoEmite(t *testing.T) {
	s, pub := novoServicoDono(t, "crm")
	t0 := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	require.True(t, declara(t, s, "crm", "conv-1", t0))
	assert.Empty(t, eventosDono(pub), "o CRM declarar a própria conversa não muda quem recebe")
	liberou, err := s.Release(context.Background(), "t-1", "ch-1", "+5511987654321", "conv-1")
	require.NoError(t, err)
	require.True(t, liberou)
	assert.Empty(t, eventosDono(pub))
}
