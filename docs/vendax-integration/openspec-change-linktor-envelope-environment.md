# [ARTEFATO PARA O CORPUS DO VENDAX] Change: add-environment-to-linktor-envelope

> **Origem:** Linktor, fase 0.1 do canal de teste (`feat/sandbox-channel-fase0`, 2026-07-23).
> **Destino:** repositório do VendaX Core — copiar este conteúdo para um pacote de mudança
> OpenSpec (`openspec/changes/add-environment-to-linktor-envelope/`) seguindo o formato do
> corpus de lá. Escrito sem o repositório do VendaX aberto: os itens marcados
> **[CONFIRMAR NO CÓDIGO DO VENDAX]** precisam ser verificados antes do aceite.

## Why

O Linktor introduziu canais de **sandbox** (homologação por tenant contra o WhatsApp real,
com allowlist de destinatários). Toda mensagem originada em canal sandbox carrega marcação
imutável de ambiente (INV-018 do CONSTITUTION do Linktor), propagada até os contratos de
saída — **inclusive o envelope consumido pelo Core**.

O produtor (Linktor) **já emite** o campo: `LinktorEnvelope.environment`
(`internal/integration/vendax/envelope.go`), publicado em `tenant.{id}.linktor.inbound`.
O DTO Java do Core ainda não o conhece.

**Dependência temporal:** o plano de integração Linktor↔VendaX está em **PROPOSTA**
(`PLANO-integracao-linktor-vendax.md`, 2026-07-08). Enquanto não congelar, esta inclusão é
aditiva e barata. Depois do freeze, vira versionamento de contrato entre dois produtos.
Este item precisa ser aceito **antes do freeze**.

## What Changes

- `br.com.vendax.core.<...>.dto.LinktorEnvelope` **[CONFIRMAR NO CÓDIGO DO VENDAX: FQCN
  exato — o plano o descreve como DTO de 7 campos String]** ganha o 8º campo:

  ```java
  /** "production" | "sandbox". Ausente/null = production (produtores antigos). */
  private String environment;
  ```

- **Semântica no Core:** mensagem com `environment == "sandbox"` é tráfego sintético de
  homologação do tenant. Mínimo obrigatório nesta mudança: **aceitar e persistir/propagar**
  o campo sem quebrar. Recomendado (pode ser mudança separada): excluir conversas sandbox
  de métricas comerciais/analytics do Core e sinalizá-las na UI do vendedor.
- **Compatibilidade retroativa (bidirecional):**
  - Envelope **antigo** (sem o campo) → `environment = null` → tratar como `production`.
  - Envelope **novo** contra Core antigo: o Linktor emite com `omitempty` — mensagens de
    produção continuam **byte a byte idênticas**; só tráfego sandbox carrega a chave.
    **[CONFIRMAR NO CÓDIGO DO VENDAX: config Jackson do consumidor —
    `FAIL_ON_UNKNOWN_PROPERTIES` deve estar desabilitado ou o DTO anotado com
    `@JsonIgnoreProperties(ignoreUnknown = true)`; caso contrário, qualquer campo novo do
    produtor derruba o consumo e esta mudança vira pré-requisito URGENTE.]**
- `LinktorOutbound` (Core → Linktor) **não** muda nesta fase: o Linktor resolve o ambiente
  pelo canal, nunca pelo envelope de entrada.

## Impact

- **Specs afetadas:** o contrato de `tenant.{id}.linktor.inbound` (envelope inbound) —
  **[CONFIRMAR NO CÓDIGO DO VENDAX: qual spec do corpus descreve esse subject; o plano o
  referencia na seção L0/L1]**. Alinhado ao ADR-010 (vocabulário de messageType) e
  ADR-012 (config não-secreta) do VendaX; `environment` é metadado não-secreto.
- **Código:** DTO + mapeamento de persistência da conversa/mensagem no Core; nenhum
  breaking change para produtores/consumidores existentes.
- **Risco de NÃO fazer:** após o freeze, tráfego sandbox do Linktor chega ao Core
  indistinguível de tráfego real — contamina métricas comerciais e conversas de vendedor
  com dado sintético, exatamente o que a marcação existe para impedir.

## Tasks (sugestão)

1. Confirmar FQCN do DTO e a config Jackson do consumidor (itens marcados acima).
2. Adicionar o campo `environment` ao DTO com default semântico `production`.
3. Persistir/propagar a marcação até onde o Core correlaciona conversas.
4. Teste de contrato: envelope com e sem o campo; envelope com campo desconhecido extra.
5. Registrar no spec do subject que ausência do campo = `production`.
