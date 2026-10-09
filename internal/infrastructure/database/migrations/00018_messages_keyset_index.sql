-- +goose Up
-- Índice para a paginação por cursor do histórico da conversa. A tela de
-- mensagens pede uma página por vez com `(created_at, id) < (cursor)` e ordena
-- por `created_at DESC, id DESC` — o mesmo par que desempata mensagens que
-- chegam no mesmo segundo. O índice existente é só por conversation_id, o que
-- obriga o Postgres a ordenar a conversa inteira para devolver 30 linhas: numa
-- conversa de 1.500 mensagens isso é um sort a cada rolagem.
CREATE INDEX IF NOT EXISTS idx_messages_conversation_keyset
    ON messages(conversation_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_conversation_keyset;
