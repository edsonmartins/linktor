# Prompt — Fase 0.1: smoke real contra a Meta e fechamento de lacunas

> Cole no Claude Code com o repositório do Linktor aberto.
> Ajuste os trechos `<<< >>>` antes de executar.

---

## Contexto

A fase 0 do canal de teste foi concluída em `feat/sandbox-channel-fase0` (9 commits, `324379f`..`1697a02`): `environment` como atributo imutável de canal, guarda de allowlist no funil único de envio, propagação da marcação até a persistência, e avaliação de janela de 24h e status de template em modo `dry_run`.

**Nada disso foi exercitado contra a Meta.** A capacidade inteira está validada contra mock e banco, não contra o provider real.

Esta fase tem duas metades:

1. **Validar a premissa** — provar que a barreira funciona contra o WhatsApp real, não só contra o teste.
2. **Fechar as lacunas** que a revisão da fase 0 identificou, com prioridade nas que deixam o sistema cego sobre si mesmo.

### Insumos obrigatórios

1. `<<< caminho >>>/CONSTITUTION.md` **v0.2** — os IDs `INV-xxx` e `G-xxx` são normativos. Leia a legenda de status: `PARCIAL` e `DECLARADO` são dívida, não garantia.
2. `<<< caminho >>>/canal-de-teste-homologacao.md` — mapeamento arquitetural.
3. `docs/sandbox-channel-operacao.md` — nota de operação produzida na fase 0.
4. O resumo de entrega da fase 0, incluindo a lista de defeitos adjacentes observados e não corrigidos.

---

## Escopo

### Dentro

| WP | Entrega | Invariante |
|---|---|---|
| WP-A | Roteiro executável de smoke real contra a Meta | INV-016/017/018 |
| WP-B | Observabilidade de fail-open das políticas de provider | INV-024 |
| WP-C | Critério objetivo de saída do dry-run | INV-015 |
| WP-D | Auditoria de criação/alteração de canal | INV-023 |
| WP-E | Teste reflexivo de round-trip por entidade + dois campos órfãos | G-005 |
| WP-F | Mascaramento de número em log de falha de provider | INV-002 |
| WP-G | Item rastreável do campo `environment` no corpus do VendaX | INV-018 |

### Fora — não implemente

- Retenção, expurgo e views filtradas para analytics (INV-019) — fase seguinte
- Adapter loopback, injeção de falha, simulação de inbound, cliente de chat
- Shadow channel (INV-022 o inviabiliza hoje)
- Reconciliação de `pending` órfão e outbox transacional no envio (INV-010)
- Correção do IDOR de tenant (INV-001) — workstream próprio
- Endpoint de capabilities (INV-014)
- `Registry.ConfigureChannel` reutilizando template sem clonar — bug documentado, deixe

### Regra que rege a fase

**G-004, não-agravamento.** Você trabalha sobre invariantes `PARCIAL`. Nada aqui pode aprofundar lacuna existente. Onde não puder fechar, registre.

---

## WP-A — Roteiro de smoke real contra a Meta

**Objetivo.** Produzir um roteiro que um operador executa manualmente contra o número de teste real, provando ponta a ponta que a barreira funciona contra o provider — e não apenas contra o mock.

**Entregável:** `docs/sandbox-channel-smoke.md`, mais qualquer script auxiliar de verificação em `<<< caminho de scripts >>>`.

**Formato:** checklist executável. Cada item traz o comando exato (curl com o endpoint real do repositório, query SQL, consulta de métrica), o resultado esperado e como distinguir sucesso de falso positivo. Nada de "verifique se funcionou".

**Cenários que o roteiro deve cobrir, nesta ordem:**

```
1. Criar canal sandbox whatsapp_official com credencial do número de teste
   → rejeitado se credential_environment ausente ou incoerente
   → rejeitado se phone_number_id fora da lista declarada

2. Tentar alterar environment do canal criado
   → rejeitado; valor persistido inalterado (confirmar no banco, não só na resposta)

3. Adicionar o número do operador à allowlist
   → persistido em E.164 normalizado
   → registro de auditoria presente

4. Enviar para o número na allowlist
   → mensagem CHEGA no aparelho (esta é a prova; status "sent" na API não basta)
   → conversations.environment = "sandbox"
   → envelope no NATS carrega environment
   → webhook de saída carrega environment

5. Enviar para um número FORA da allowlist (use um segundo número da equipe)
   → NADA chega no aparelho
   → status failed, motivo identificável, blocked_by preenchido em message_logs
   → linktor_outbound_guard_blocked_total incrementado com reason correto
   → NENHUMA chamada saiu para a Graph API (confirmar por log de saída, não por inferência)

6. Remover o número da allowlist e enviar imediatamente
   → bloqueado no primeiro envio, sem esperar os 5 min do cache do Resolver

7. Responder do aparelho (inbound real)
   → conversa marcada como sandbox
   → dedup verificado (reenvio do mesmo evento não duplica)

8. Enviar free-form com janela de 24h expirada, em dry_run
   → mensagem SAI (dry-run não bloqueia)
   → métrica registra o que seria bloqueado, com mode=dry_run

9. Repetir o item 8 com enforcement ativo (ambiente descartável)
   → bloqueado antes da chamada à Meta
   → motivo distinguível do bloqueio de allowlist

10. Canal production preexistente: enviar normalmente
    → nenhuma guarda aplicada, comportamento idêntico ao anterior
```

**O item 5 é o teste que justifica a fase inteira.** Deixe explícito no documento que ele exige um segundo número real e que a verificação é a ausência de mensagem no aparelho — não a resposta da API.

**Inclua uma seção de pré-requisitos:** o que precisa estar provisionado na Meta (número de teste registrado, destinatários verificados, template `hello_world`), variáveis de ambiente necessárias, e como reverter o estado ao final.

**Não execute o smoke.** Você não tem acesso ao provider nem aos aparelhos. Produza o roteiro.

---

## WP-B — Observabilidade de fail-open

**Objetivo.** INV-024. As políticas de WP6/WP7 falham abertas em incerteza — decisão correta (G-006), mas hoje invisível.

**O problema concreto:** `linktor_outbound_guard_blocked_total` conta bloqueios. Não conta avaliações que **não puderam ser feitas** — erro na consulta de `LastInboundAt`, status de template indisponível ou não sincronizado. Uma falha sistemática na consulta desliga o enforcement na prática, e o painel mostra queda de bloqueios: indistinguível de melhoria.

**Implementar:** contador próprio, distinto do de bloqueios, com dimensão de motivo da incerteza. Respeite INV-021 — sem label de tenant. Log correlato com informação suficiente para o operador diagnosticar a causa.

**Critérios de aceite**

```
WHEN a consulta de LastInboundAt falha e a política deixa passar
THEN o contador de fail-open é incrementado com o motivo
 AND o log registra a causa da incerteza
 AND o contador de bloqueios NÃO é incrementado

WHEN o status de um template é desconhecido e o envio é permitido
THEN o mesmo tratamento se aplica, com motivo distinto

WHEN a política avalia normalmente e permite
THEN nenhum contador de fail-open é incrementado
```

O último cenário importa: fail-open é exceção observável, não o caminho feliz.

---

## WP-C — Critério de saída do dry-run

**Objetivo.** INV-015 está `PARCIAL` porque o default é `dry_run`. Sem critério objetivo de virada, ele permanece em dry-run indefinidamente e a política nunca liga.

**Entregável:** seção em `docs/sandbox-channel-operacao.md` contendo:

- Janela mínima de observação, justificada pelo volume real do piloto.
- Consultas prontas (PromQL e SQL) que respondem: quantos envios seriam bloqueados, por qual motivo, em qual canal — e quantas avaliações falharam abertas (WP-B).
- Taxa esperada de bloqueio e o que caracteriza anomalia que **impede** a virada.
- Procedimento de virada e de rollback, ambos por configuração e sem deploy.
- Registro explícito de que a virada promove INV-015 de `PARCIAL` para `IMPOSTO`, exigindo atualização do CONSTITUTION (G-002).

**Não decida os números sozinho.** Proponha valores com a justificativa e marque como pendente de confirmação humana.

---

## WP-D — Auditoria de criação e alteração de canal

**Objetivo.** INV-023 foi rebaixado a `PARCIAL` na v0.2 pela assimetria: a allowlist é auditada, a criação de canal não. `environment` é o atributo que define toda a fronteira de sandbox e não tem trilha.

**Implementar:** registro em `internal/application/service/audit.go` para criação e alteração de canal, contendo autor, `environment` declarado, `credential_environment` declarado e — para `whatsapp_official` — o `phone_number_id`. **Sem credencial, sem segredo** (INV-002).

**Critérios de aceite**

```
WHEN um canal sandbox é criado
THEN há registro de auditoria com autor, environment e credential_environment declarados
 AND nenhum valor de credencial aparece no registro

WHEN um canal é alterado
THEN há registro com autor e o delta dos campos relevantes

WHEN a criação é rejeitada por validação
THEN <<< decidir: auditar a tentativa ou não >>> — proponha e pare para decisão
```

A última pergunta não é trivial: auditar tentativa rejeitada é sinal de segurança útil, mas polui a trilha. Recomende com base no volume esperado.

---

## WP-E — Teste reflexivo de round-trip + campos órfãos

**Objetivo.** G-005. Três ocorrências do mesmo defeito — campo na entidade que o repositório não persiste (`source`/`is_imported` em `messages`; `Channel.Identifier` e `MessageTemplateNamespace` em `channels`). É defeito de processo, não três bugs.

**Implementar, nesta ordem:**

1. **Teste reflexivo genérico** que, para cada entidade de domínio persistida, popule todos os campos exportados com valores não-zero, grave, releia e compare — falhando com o nome do campo que não sobreviveu. Contra banco real. Deve ser fácil registrar entidade nova nesse teste.
2. **Uma lista de exceções explícita e comentada** para campos legitimamente não persistidos (calculados, transientes, imutáveis por desenho como `environment` no UPDATE). Exceção sem comentário justificando é falha de teste.
3. **Só então** corrija `Channel.Identifier` e `MessageTemplateNamespace`.

A ordem importa: o teste primeiro deve **falhar** apontando os dois campos, provando que a rede pega o defeito. Se o teste passar antes da correção, ele não serve.

**Critérios de aceite**

```
WHEN o teste reflexivo roda antes da correção
THEN ele falha nomeando Channel.Identifier e MessageTemplateNamespace

WHEN o teste roda depois da correção
THEN passa

WHEN um campo novo é adicionado a uma entidade sem persistência nem exceção declarada
THEN o teste falha nomeando o campo
```

Não corrija `source`/`is_imported` nesta fase se o teste os apontar — registre e traga para decisão; podem ter razão histórica.

---

## WP-F — Mascaramento de número em log de falha

`logActivity` do worker registra `meta["to"]` com número completo em falhas de provider (pré-existente). Os bloqueios de guarda já mascaram — aplique o mesmo tratamento aqui, reutilizando a função existente.

```
WHEN uma falha de provider é registrada em message_logs
THEN o destinatário aparece mascarado, no mesmo formato usado pelos bloqueios de guarda
 AND o dado continua suficiente para o operador correlacionar
```

---

## WP-G — Rastreabilidade do contrato VendaX

`environment` está no `LinktorEnvelope`, mas **não** no DTO Java do Core VendaX. O contrato segue em PROPOSTA — enquanto isso, a inclusão é aditiva e barata; depois do freeze, vira versionamento entre dois produtos.

"Registrado no código" não sobrevive a um freeze decidido do outro lado.

**Entregável:** o texto de um item rastreável para o corpus do VendaX — nomeando o DTO Java, o campo, a compatibilidade retroativa e a dependência temporal com o freeze. Escreva no formato de mudança OpenSpec usado naquele projeto. Você não tem o repositório do VendaX aberto: produza o artefato para ser levado até lá, e deixe explícito o que precisa ser confirmado contra aquele código.

---

## Restrições

- **Não refatore fora do escopo.** Defeito adjacente vira nota, não commit.
- **Compatibilidade retroativa obrigatória** nos contratos de saída; campos novos aditivos com `omitempty`.
- **Nada muda o comportamento de canal `production`** — WP-F altera apenas o formato do log.
- **Sem segredo em log, métrica, evento ou erro** (INV-002).
- **Migrations só via goose**, seguindo a convenção da baseline congelada.
- **Cada WP é um commit coerente**, com os próprios testes verdes.
- Suíte completa (`go test -race ./...` e integração contra Postgres real) verde ao final, sem teste desabilitado.

## Ordem e pontos de parada

WP-A primeiro — é o de maior valor e não depende de nada. Depois WP-B e WP-E em paralelo, então WP-C (depende do contador do WP-B), WP-D, WP-F, WP-G.

**Pare e peça decisão humana quando:**

- Precisar fixar números do critério de saída do dry-run (WP-C).
- Decidir se tentativa rejeitada de criação de canal é auditada (WP-D).
- O teste reflexivo apontar campos além dos dois previstos (WP-E).
- Descobrir que o contrato do VendaX já congelou (WP-G).
- Qualquer entrega exigir flexibilizar invariante do CONSTITUTION.

Ao final, entregue: resumo por WP, decisões tomadas dentro da margem, **a lista de invariantes cujo status muda com esta fase** (com a justificativa de cada mudança, conforme G-002), defeitos adjacentes observados e não corrigidos, e o que permanece para a fase seguinte.
