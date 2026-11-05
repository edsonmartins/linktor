-- +goose Up
-- Dono da conversa aberta por (canal, contato): o inbound daquele contato naquele
-- canal vai só ao consumidor dono (P5 do acordo CRM ↔ Sales Copilot, VCR-36).
-- Sem regra vigente, vai ao consumidor padrão do canal — o comportamento de hoje.
--
-- Quem escreve e libera a regra são os produtos (o CRM, dono do cliente); ela não
-- anda com o estado da conversa no Linktor. contact_identifier é o identificador
-- normalizado do contato no canal (E.164 para telefone). expira_em é opcional e
-- vem de quem cria a regra; não há expiração padrão por canal.
CREATE TABLE IF NOT EXISTS conversation_owner_rules (
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id         UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    contact_identifier VARCHAR(255) NOT NULL,
    consumer           VARCHAR(64) NOT NULL,
    conversa_ref       VARCHAR(255) NOT NULL,
    desde              TIMESTAMP WITH TIME ZONE NOT NULL,
    expira_em          TIMESTAMP WITH TIME ZONE,
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, channel_id, contact_identifier)
);

-- +goose Down
DROP TABLE IF EXISTS conversation_owner_rules;
