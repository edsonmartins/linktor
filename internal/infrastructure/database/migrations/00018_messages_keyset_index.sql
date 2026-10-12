-- +goose Up
-- OBSOLETA: este índice duplica idx_messages_conversation_created_id, criado
-- pela 00010 com as mesmas colunas e direções. A 00020 o remove. Mantida como
-- está porque migração aplicada não se reescreve — quem já rodou a 00018 tem o
-- índice, e é a 00020 que o desfaz.
--
-- Índice para a paginação por cursor do histórico da conversa.
CREATE INDEX IF NOT EXISTS idx_messages_conversation_keyset
    ON messages(conversation_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_conversation_keyset;
