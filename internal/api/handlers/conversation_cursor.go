package handlers

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// O cursor da lista de conversas viaja como texto opaco, no mesmo formato do
// cursor do histórico: base64url de "<instante>|<id>". Por dentro é o par que
// ordena a lista — a marca de atividade e o id —, para que o formato possa
// mudar sem quebrar quem guardou um cursor antigo.

// conversationActivityAt is the ordering key: a conversa sem mensagem nenhuma
// se ordena pela criação, que é quando ela de fato apareceu na caixa.
func conversationActivityAt(c *entity.Conversation) time.Time {
	if c.LastMessageAt != nil {
		return *c.LastMessageAt
	}
	return c.CreatedAt
}

// encodeConversationCursor serializes a list position for the client.
func encodeConversationCursor(c repository.ConversationCursor) string {
	raw := c.ActivityAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeConversationCursor parses what encodeConversationCursor produced. Um
// cursor ilegível é erro de quem chama, não página vazia: recomeçar do topo em
// silêncio faria a rolagem repetir a lista para sempre.
func decodeConversationCursor(s string) (*repository.ConversationCursor, error) {
	if s == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Validation("invalid cursor")
	}

	at, id, found := strings.Cut(string(raw), "|")
	if !found || id == "" {
		return nil, errors.Validation("invalid cursor")
	}

	activityAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return nil, errors.Validation("invalid cursor")
	}

	return &repository.ConversationCursor{ActivityAt: activityAt, ID: id}, nil
}
