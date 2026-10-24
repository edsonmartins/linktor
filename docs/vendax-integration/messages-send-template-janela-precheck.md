# `POST /api/v1/messages/send`: template, precheck e janela de 24h

Mudanças aditivas pedidas pelo VendaX CRM (WA-02 e WA-03). Nenhum campo ou rota existente muda de comportamento:
quem não usa os campos novos recebe exatamente o que recebia.

## Janela de 24h — `GET /api/v1/channels/:id/window?to=<destinatário>`

Escopo `messages:send`. Responde se uma mensagem livre ainda cabe na janela de atendimento da Meta, pela mesma regra
do guard de saída (`outbound.WindowOpen`): a última mensagem recebida do contato na conversa aberta do canal, que é a
conversa que o próximo envio usa.

```json
{"success": true, "data": {"channel_id": "…", "to": "5511…", "applies": true, "open": true,
  "last_inbound_at": "2026-10-24T10:00:00Z", "closes_at": "2026-10-25T10:00:00Z"}}
```

- `applies=false` (com `open=true`) em canal sem a regra: tudo que não é `whatsapp_official`.
- Contato desconhecido ou sem conversa aberta → `open=false` e datas nulas. A consulta não cria nada.
- Canal de outro tenant → 404 `CHANNEL_NOT_FOUND`.
- O `+` do destinatário vai codificado (`%2B`): na query, `+` cru é lido como espaço.

## Template — `content_type: "template"`

```json
{"channel_id": "…", "to": "+5511…", "content_type": "template",
 "template": {"name": "reativacao", "language": "pt_BR",
   "components": [{"type": "body", "parameters": [{"type": "text", "text": "Ana"}]}]}}
```

- `name` e `language` são obrigatórios. `components` é o array nativo da Meta e passa sem alteração.
- Vira a metadata `template_name`, `template_language` e `template_components` que o pipeline de saída já lê
  (`outbound.Template`). O conteúdo gravado é `template:<nome>`.

## Precheck — `"precheck": true` (opt-in)

Em canal `whatsapp_official`, recusa **antes de gravar qualquer coisa**, com códigos distintos:

| `error.code` | HTTP | Quando |
|---|---|---|
| `INVALID_RECIPIENT` | 400 | o destinatário não é um telefone válido (E.164) |
| `TEMPLATE_NOT_APPROVED` | 422 | o registro local diz que a Meta recusaria o template (mesma lista de bloqueio do guard; `details.template_status`) |
| `OUTSIDE_WINDOW` | 422 | texto livre com a janela de 24h fechada |

- Template sem registro local, aprovado ou pendente segue: a Meta decide, como no guard.
- Sem `precheck`, nada muda; os guards de saída decidem depois e a falha chega como `message.failed`.

## Opt-out

O Linktor não mantém registro de opt-out. A recusa por opt-out só existe quando a Meta a informa: código 131050 no
status. O dono do opt-in/opt-out é decisão do VendaX CRM (WA-10, VCR-147).
