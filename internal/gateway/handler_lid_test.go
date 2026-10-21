package gateway

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
)

// O bridge traduz remetente LID para o número quando conhece o mapeamento;
// quando não conhece, manda só o LID. O contato tem de ser identificado pelo
// número no primeiro caso (com telefone) e pelo JID @lid inteiro no segundo,
// senão a resposta do atendente é endereçada a um número que não existe.
func TestHandleInbound_SenderIdentity(t *testing.T) {
	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-1"] = &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: entity.ChannelType(plugin.ChannelTypeWhatsAppRemote)}
	producer := testutil.NewMockProducer()
	h := NewHandler(NewHub(), channels, producer, nil, nil)

	cases := []struct {
		name, from, lid, wantSender, wantPhone, wantLID string
	}{
		{"número com LID traduzido", "554499806761@s.whatsapp.net", "252415295144120@lid", "554499806761", "554499806761", "252415295144120@lid"},
		{"número puro", "5511999999999@s.whatsapp.net", "", "5511999999999", "5511999999999", ""},
		{"só LID, sem tradução", "252415295144120@lid", "252415295144120@lid", "252415295144120@lid", "", "252415295144120@lid"},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(InboundEvent{Type: "message", ID: "M" + tc.name, From: tc.from, Chat: tc.from, SenderLID: tc.lid, Text: "oi", Timestamp: 1})
			require.NoError(t, h.handleInbound("ch-1", raw))
			in := producer.InboundMessages[i]
			assert.Equal(t, tc.wantSender, in.Metadata["sender_id"])
			assert.Equal(t, tc.wantPhone, in.Metadata["phone"])
			assert.Equal(t, tc.wantLID, in.Metadata["sender_lid"])
			assert.Equal(t, tc.from, in.Metadata["sender_jid"])
		})
	}
}
