# Canal de teste (sandbox) WhatsApp — nota de operação

- **Escopo:** fase 0 — modalidade "espelho do provider" para WhatsApp (Cloud API oficial e sessão não oficial). Invariantes: `docs/spec/CONSTITUTION.md` (INV-015 a INV-018).
- **Propriedade de segurança:** canal `sandbox` só entrega a destinatários na allowlist do tenant. Qualquer outro destino falha de forma ruidosa (status `failed`, log, métrica) — nunca silenciosamente.
- **Pré-condição:** as garantias herdam o isolamento de tenant (INV-001, hoje `PARCIAL` — IDOR P0 da auditoria 2026-07 em correção). A capacidade só é confiável após o P0 fechar.

## Criar um canal sandbox

`POST /api/v1/channels` com `environment: "sandbox"`. O campo é **imutável** após a criação (update que tente alterá-lo é rejeitado; o UPDATE do repositório nem inclui a coluna).

```json
{
  "type": "whatsapp_official",
  "name": "Homologação ACME",
  "environment": "sandbox",
  "config": {
    "phone_number_id": "111222333",
    "sandbox_test_phone_number_ids": "111222333, 444555666"
  },
  "credentials": {
    "access_token": "<token do número de teste>",
    "credential_environment": "sandbox"
  }
}
```

Validações na criação (rejeitam antes de persistir):

- `credentials.credential_environment = "sandbox"` é **obrigatório** em canal sandbox; um canal `production` rejeita credencial declarada como sandbox (INV-002). A checagem é **declarativa** — não é possível inferir de um token se ele é de produção; a garantia dura contra alcance indevido é a allowlist, não esta validação.
- Para `whatsapp_official` sandbox: `config.phone_number_id` deve constar da lista declarada `config.sandbox_test_phone_number_ids` (sem chamada à Graph API nesta fase). Revalidado em todo update sobre o estado pós-merge.

## Gerir a allowlist de destinatários

Endpoints (admin/owner do tenant apenas; toda alteração é auditada com autor, ação e valor):

| Operação | Endpoint |
|---|---|
| Listar | `GET /api/v1/sandbox/allowlist` |
| Adicionar | `POST /api/v1/sandbox/allowlist` — `{"recipient": "+55 44 99999-9999", "channel_id": "<opcional>", "note": "..."}` |
| Remover | `DELETE /api/v1/sandbox/allowlist/{id}` |

- Números são normalizados para E.164 na escrita **e** na comparação — `+55 44 9...` e `5544 9...` são o mesmo destinatário.
- `channel_id` vazio = entrada vale para todos os canais sandbox do tenant; preenchido = só para aquele canal (que deve ser sandbox e do tenant).
- **Remoção vale no próximo envio.** A guarda consulta a allowlist a cada envio; não há cache a expirar.
- A guarda fica no funil único de envio (worker → Resolver → Sender), cobrindo API, campanha, bot e retries. Canais `production` não são afetados.

## Alternar dry-run / enforcement (janela 24h e template)

Duas políticas de provider para `whatsapp_official`, ambas com rollout em dois estágios via env var (reler exige restart do processo; rollback é troca de configuração, sem deploy):

| Política | Env var | Valores |
|---|---|---|
| Janela de 24h (free-form fora da janela) | `LINKTOR_WA_WINDOW_ENFORCEMENT` | `off` \| `dry_run` (**default**) \| `enforce` |
| Template com status inutilizável (rejeitado/pausado/...) | `LINKTOR_WA_TEMPLATE_ENFORCEMENT` | `off` \| `dry_run` (**default**) \| `enforce` |

- `dry_run`: avalia, **não bloqueia**; registra métrica com `mode="dry_run"` e log do que *seria* bloqueado. Deixe rodar até a taxa observada ser compatível com o esperado antes de ligar `enforce`.
- `enforce`: bloqueia localmente antes da chamada à Meta (erro permanente, sem retry).
- Casos de incerteza **falham abertos** nessas duas políticas (template não sincronizado permite com log; erro na consulta da janela permite com log) — são políticas de provider, não fronteira de segurança. A guarda sandbox é o oposto: **falha fechada** em qualquer incerteza.
- Template aprovado é isento da janela de 24h. Status de template é consultado **no envio** (a Meta recategoriza por conta própria).

## Observabilidade dos bloqueios

- **Métrica:** `linktor_outbound_guard_blocked_total{channel_type, reason, mode}` — `reason ∈ {allowlist, invalid_recipient, unsupported_channel_type, window_24h, template_rejected}`, `mode ∈ {enforce, dry_run}`. Sem label de tenant (decisão deliberada de cardinalidade, INV-021).
- **Por tenant:** tela de logs do admin (`message_logs`). Bloqueio de guarda aparece como `"Envio bloqueado por guarda (<reason>)"` com metadata `blocked_by=<reason>` — distinguível de `"Envio falhou (permanente)"` (falha de provider) sem ler código.
- Números de destinatário aparecem sempre **mascarados** (`+55*********99`) em logs, métricas e mensagens de erro (INV-002).
- A mensagem bloqueada fica com status `failed` e o motivo no campo de erro.

## Limitações conhecidas da fase 0

- Só WhatsApp tem semântica sandbox definida; canal sandbox de outro tipo **falha fechado** em qualquer envio.
- Retenção/expurgo de dado sandbox e bloqueio de exportação (analytics/bridge VendaX) ficam para a fase seguinte (INV-019) — o dado já nasce marcado (`conversations.environment`, envelopes) para viabilizá-los.
- Espelho não oficial roda na rede real do WhatsApp: risco de banimento do número é inerente à modalidade; use número dedicado e descartável.

## Critério de saída do dry-run (janela 24h e template) — INV-015

> Números **confirmados** em decisão humana de 2026-07-23 (fase 0.1). A virada promove
> INV-015 de `PARCIAL` para `IMPOSTO` e **exige atualização do CONSTITUTION** (G-002):
> registre a versão nova com o ponto de imposição (guards no funil) e os testes correspondentes.

### Janela mínima de observação

**14 dias corridos E ≥500 envios free-form avaliados no canal piloto — o que ocorrer por último.**
Justificativa: o piloto tem volume baixo; menos que isso não gera amostra com as variações
de semana (fim de semana, horário comercial) que dominam a expiração de janela de 24h.

### Consultas de acompanhamento

**PromQL — o que SERIA bloqueado, por motivo (14 dias):**

```promql
sum by (reason) (increase(linktor_outbound_guard_blocked_total{mode="dry_run"}[14d]))
```

**PromQL — avaliações que falharam abertas (INV-024), por política e causa:**

```promql
sum by (policy, cause) (increase(linktor_outbound_guard_fail_open_total[14d]))
-- alerta de fail-open sistemático (impede a virada):
sum by (policy) (increase(linktor_outbound_guard_fail_open_total{cause="lookup_error"}[1h])) > 0
```

**SQL — denominador (free-form avaliados no período, canais whatsapp_official):**

```sql
SELECT count(*) FROM messages m
JOIN conversations c ON c.id = m.conversation_id
JOIN channels ch ON ch.id = c.channel_id
WHERE ch.type = 'whatsapp_official'
  AND m.sender_type IN ('user', 'bot')
  AND m.content_type <> 'template'
  AND m.created_at > now() - interval '14 days';
```

**Por canal:** a métrica não tem label de canal (cardinalidade, INV-021) e o dry-run não
escreve em `message_logs`. A granularidade por canal sai do log do backend: grep por
`[dry-run]` (a linha carrega o `message id`) e resolução do canal por SQL:

```bash
grep -o 'dry-run.*message [0-9a-f-]*' backend.log | awk '{print $NF}' | sort -u > /tmp/ids
```
```sql
SELECT c.channel_id, count(*) FROM messages m JOIN conversations c ON c.id=m.conversation_id
WHERE m.id = ANY(:ids) GROUP BY 1;
```

*(Limitação registrada: entrada de dry-run em `message_logs` facilitaria isto; candidata à próxima fase.)*

### Taxa esperada e anomalias que IMPEDEM a virada

- **Esperado:** bloqueios `window_24h` em dry-run ≤ **2%** dos free-form avaliados
  (homologação disciplinada responde dentro da janela); `template_rejected` ≈ 0.
- **Anomalia — não vire enforcement se qualquer um ocorrer:**
  - taxa de would-block > **5%** (indica fluxo legítimo fora da janela; investigar antes de bloquear clientes);
  - **fail-open sistemático**: `cause="lookup_error"` > 0 sustentado por **> 1 hora** (a política está cega; ligar enforcement seria fingir proteção);
  - fail-open total > **0,5%** das avaliações do período;
  - qualquer would-block cuja investigação manual conclua que a mensagem era legítima e a janela estava aberta (falso positivo → bug antes de virada).

### Procedimento de virada e rollback (ambos por configuração, sem deploy)

1. Confirmar janela + consultas acima dentro dos limites.
2. Virada: `LINKTOR_WA_WINDOW_ENFORCEMENT=enforce` (e/ou `LINKTOR_WA_TEMPLATE_ENFORCEMENT=enforce`) e reiniciar o processo.
3. Pós-virada (primeiras 48h): acompanhar `mode="enforce"` nas mesmas consultas; taxa deve permanecer na faixa observada no dry-run.
4. **Rollback:** voltar a env var para `dry_run` e reiniciar. Nenhum deploy, nenhuma migração.
5. Atualizar o CONSTITUTION (INV-015 → `IMPOSTO`, com data e evidência) — sem isso a virada não está concluída (G-002).
