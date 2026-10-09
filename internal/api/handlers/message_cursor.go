package handlers

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// O cursor do histórico viaja como um texto opaco: quem consome passa de volta
// o que recebeu, sem interpretar. Por dentro é só o par que ordena a conversa
// — created_at e id — para que o formato possa mudar sem quebrar clientes que
// guardaram um cursor antigo.
//
// Base64 URL-safe sem padding, porque o valor anda em query string.

// encodeMessageCursor serializes a thread position for the client.
func encodeMessageCursor(c repository.MessageCursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeMessageCursor parses what encodeMessageCursor produced. An unreadable
// cursor is a client error, not an empty page: silently starting from the
// newest message would make a broken "load older" button look like the end of
// the conversation.
func decodeMessageCursor(s string) (*repository.MessageCursor, error) {
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

	createdAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return nil, errors.Validation("invalid cursor")
	}

	return &repository.MessageCursor{CreatedAt: createdAt, ID: id}, nil
}
