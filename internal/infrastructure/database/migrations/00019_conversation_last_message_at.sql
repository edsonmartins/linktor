-- +goose Up
-- Quando a última mensagem chegou, gravado na própria conversa.
--
-- A listagem calculava isso por subconsulta a cada linha devolvida. Serve para
-- exibir, não para ordenar nem paginar: ordenar por um valor calculado obriga o
-- Postgres a computá-lo para a tabela inteira antes de devolver os primeiros 20,
-- e não há cursor possível sobre ele. Por isso a tela ordenava por updated_at,
-- que muda por atribuição e mudança de status — deixando conversa parada acima
-- de conversa ativa.
--
-- NULL significa conversa sem nenhuma mensagem, e é diferente de "nunca houve
-- atividade": a ordenação usa coalesce(last_message_at, created_at), e o índice
-- abaixo cobre exatamente essa expressão.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMP WITH TIME ZONE;

UPDATE conversations c
   SET last_message_at = sub.ultima
  FROM (SELECT conversation_id, max(created_at) AS ultima FROM messages GROUP BY conversation_id) sub
 WHERE sub.conversation_id = c.id
   AND c.last_message_at IS DISTINCT FROM sub.ultima;

-- +goose StatementBegin
-- Gatilho, e não escrita no repositório: há dois caminhos de inserção de
-- mensagem hoje (um deles transacional, com outbox) e a importação de histórico
-- amanhã. Uma coluna materializada que um caminho esquece de atualizar ordena a
-- tela errado e ninguém percebe — o banco é o único lugar por onde todos passam.
--
-- GREATEST porque mensagem antiga pode ser gravada depois (importação): a marca
-- de atividade só anda para a frente.
CREATE OR REPLACE FUNCTION conversations_touch_last_message_at()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE conversations
       SET last_message_at = GREATEST(COALESCE(last_message_at, NEW.created_at), NEW.created_at)
     WHERE id = NEW.conversation_id
       AND (last_message_at IS NULL OR last_message_at < NEW.created_at);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS messages_touch_conversation_activity ON messages;
CREATE TRIGGER messages_touch_conversation_activity
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION conversations_touch_last_message_at();

-- Índice sobre a mesma expressão que a listagem ordena, na mesma direção: é o
-- que permite devolver uma página sem varrer e ordenar a caixa inteira.
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_activity
    ON conversations(tenant_id, (COALESCE(last_message_at, created_at)) DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_conversations_tenant_activity;
DROP TRIGGER IF EXISTS messages_touch_conversation_activity ON messages;
DROP FUNCTION IF EXISTS conversations_touch_last_message_at();
ALTER TABLE conversations DROP COLUMN IF EXISTS last_message_at;
