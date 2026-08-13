-- +goose Up
-- Idempotência lógica do envio direto (POST /api/v1/messages/send).
--
-- A chave é fornecida pelo integrador em metadata.idempotency_key e vale por
-- tenant: duas requisições com a mesma chave no mesmo tenant produzem UMA
-- mensagem. A unicidade mora aqui, e não em messages, porque messages não tem
-- tenant_id (o tenant é alcançado via conversation) e porque a reserva precisa
-- acontecer ANTES de a mensagem existir — é ela que serializa duas requisições
-- concorrentes com a mesma chave.
--
-- message_id não é FK para messages: a linha é escrita antes do INSERT da
-- mensagem (reserva) e é removida quando o envio falha (liberação), então uma
-- FK só criaria uma janela em que a reserva não pode existir.
CREATE TABLE IF NOT EXISTS message_idempotency_keys (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(255) NOT NULL,
    message_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_message_idempotency_message
    ON message_idempotency_keys(message_id);

-- +goose Down
DROP TABLE IF EXISTS message_idempotency_keys;
