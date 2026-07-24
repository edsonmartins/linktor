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
