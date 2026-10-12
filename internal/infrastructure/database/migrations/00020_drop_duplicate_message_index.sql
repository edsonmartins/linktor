-- +goose Up
-- Remove índice duplicado.
--
-- A 00018 criou idx_messages_conversation_keyset em
-- (conversation_id, created_at DESC, id DESC) para a paginação do histórico.
-- A 00010 já havia criado idx_messages_conversation_created_id com exatamente
-- as mesmas colunas, nas mesmas direções — eu olhei a lista de índices do
-- postgres.go e não as migrações, e concluí que só existia o índice por
-- conversation_id.
--
-- Os dois convivendo custam duas vezes: espaço (2,2 MB na instalação onde isso
-- apareceu, sobre 17 MB de mensagens) e, o que importa mais, uma segunda
-- escrita de índice a cada mensagem inserida — na tabela que mais recebe
-- escrita do sistema.
--
-- Fica o mais antigo, que é o mais usado pelo planner (41.676 varreduras
-- contra 8.303) e serve tanto o desempate da ordenação quanto o keyset: a
-- consulta é a mesma.
DROP INDEX IF EXISTS idx_messages_conversation_keyset;

-- +goose Down
CREATE INDEX IF NOT EXISTS idx_messages_conversation_keyset
    ON messages(conversation_id, created_at DESC, id DESC);
