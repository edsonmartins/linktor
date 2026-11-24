---
sidebar_position: 5
title: Monitoramento
description: Health checks, métricas Prometheus, regras de alerta, dashboard Grafana, logs, a tela Observabilidade e o alerta de versão do cliente WhatsApp.
---

# Monitoramento

O backend expõe três endpoints de operação na mesma porta da API, sempre ativos e **sem
autenticação**: `/health`, `/ready` e `/metrics`. O repositório traz regras de alerta e um dashboard
prontos em `deploy/monitoring/`, mas **não sobe** Prometheus, Alertmanager nem Grafana — você os
conecta à sua pilha.

## Health checks

| Endpoint | Verifica | Resposta |
|---|---|---|
| `GET /health` | Que o processo responde. Não consulta dependências. | Sempre `200`. |
| `GET /ready` | PostgreSQL, NATS (se a conexão foi estabelecida na inicialização) e Redis (somente se `REDIS_URL` estiver definida). | `200` com `status: "ready"`, ou `503` com `status: "not ready"`. |

```bash
curl "$LINKTOR_URL/health"
```

```json
{"service":"linktor","status":"ok","timestamp":"2026-11-24T12:00:00Z","version":"sha-abc1234"}
```

O campo `version` repete a variável `LINKTOR_VERSION`; fica vazio se ela não estiver definida.

```bash
curl -i "$LINKTOR_URL/ready"
```

```json
{
  "service": "linktor",
  "status": "not ready",
  "timestamp": "2026-11-24T12:00:00Z",
  "checks": {
    "postgres": {"status": "healthy"},
    "nats": {"status": "unhealthy", "error": "not connected"}
  }
}
```

Como usar cada um:

- **Monitor externo de disponibilidade**: `/health`.
- **Decidir se a instância deve receber tráfego**: `/ready`. Uma instância com o NATS desconectado
  responde na API mas não processa nenhuma mensagem.
- Os healthchecks dos arquivos Compose usam `/health`. O Traefik remove do roteamento um container
  marcado como `unhealthy`.

:::note
Se o NATS estava fora no momento em que o backend iniciou, o backend sobe sem ele e `/ready` **não
inclui** a checagem `nats` — responde `200` mesmo sem fila. Procure no log por
`messaging features disabled` e reinicie o backend depois que o NATS voltar. Nesse estado a métrica
`linktor_nats_up` fica em `0`, então o alerta `LinktorNATSDown` cobre o caso.
:::

## Métricas Prometheus

`GET /metrics` devolve as métricas do aplicativo (prefixo `linktor_`) e os coletores padrão de Go e
do processo (`go_*`, `process_*`).

### Pipeline de mensagens

| Métrica | Tipo | Rótulos | Significado |
|---|---|---|---|
| `linktor_inbound_messages_total` | contador | `channel_type`, `result` | Mensagens recebidas, por resultado: `processed`, `duplicate` ou `failed`. |
| `linktor_inbound_processing_seconds` | histograma | `channel_type` | Tempo de processamento de uma mensagem recebida. |
| `linktor_outbound_messages_total` | contador | `channel_type`, `result` | Envios, por resultado: `sent` ou `failed`. |
| `linktor_outbound_send_seconds` | histograma | `channel_type` | Tempo da chamada ao provedor. |
| `linktor_outbound_guard_blocked_total` | contador | `channel_type`, `reason`, `mode` | Envios barrados por uma proteção. `reason`: `allowlist` (sandbox), `window_24h` e outros; `mode`: `enforce` (bloqueou) ou `dry_run` (só registrou). |
| `linktor_outbound_guard_fail_open_total` | contador | `channel_type`, `policy`, `cause`, `mode` | Vezes em que uma política não conseguiu avaliar o envio e o liberou. `cause="lookup_error"` indica falha de infraestrutura; `status_unknown`, dado ainda não sincronizado. |

### NATS

| Métrica | Tipo | Rótulos | Significado |
|---|---|---|---|
| `linktor_nats_up` | gauge | — | `1` com a conexão ativa, `0` sem. Atualizada a cada 10 segundos. |
| `linktor_nats_publish_failures_total` | contador | `kind` | Falhas ao publicar, por tipo de mensagem. |
| `linktor_stream_messages` | gauge | `stream` | Mensagens guardadas em cada stream. Em `LINKTOR_DLQ`, é o acúmulo de mensagens mortas. |
| `linktor_consumer_pending` | gauge | `stream`, `consumer` | Mensagens ainda não entregues ao consumidor (atraso). |
| `linktor_consumer_ack_pending` | gauge | `stream`, `consumer` | Mensagens em processamento, aguardando confirmação. |
| `linktor_consumer_redelivered` | gauge | `stream`, `consumer` | Mensagens reentregues. Crescimento contínuo indica uma mensagem que falha sempre. |

As gauges de stream e consumidor são amostradas a cada 15 segundos.

:::note
A métrica `linktor_nats_dlq_messages_total` está registrada mas não é incrementada pelo código
atual. Para acompanhar a DLQ, use `linktor_stream_messages{stream="LINKTOR_DLQ"}`.
:::

### HTTP

| Métrica | Tipo | Rótulos | Significado |
|---|---|---|---|
| `linktor_http_requests_total` | contador | `method`, `route`, `status` | Requisições. `route` é o padrão da rota (`/api/v1/channels/:id`), não a URL. |
| `linktor_http_request_seconds` | histograma | `method`, `route` | Duração das requisições. |

Nenhuma métrica tem rótulo de tenant.

### Coletar

O arquivo `deploy/monitoring/prometheus.yml` é um ponto de partida. O alvo padrão é `linktor:8081`;
ajuste para o nome do serviço na sua rede — na pilha de `deploy/`, o backend se chama `backend`:

```yaml title="prometheus.yml"
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - alerts.yml

alerting:
  alertmanagers:
    - static_configs:
        - targets: ["alertmanager:9093"]

scrape_configs:
  - job_name: linktor
    metrics_path: /metrics
    static_configs:
      - targets: ["backend:8081"]
        labels:
          service: linktor
```

Mantenha `job_name: linktor`: a regra `LinktorInstanceDown` filtra por esse nome. O Prometheus
precisa estar em uma rede Docker que alcance o backend (`linktor_linktor-internal` ou
`linktor_linktor-edge`).

:::warning `/metrics` fica público na pilha de produção
O roteador do Traefik para `api.<domínio>` encaminha todos os caminhos, inclusive `/metrics`.
Restrinja-o antes do go-live — o exemplo está em [Segurança](./seguranca.md).
:::

O Traefik também expõe métricas Prometheus próprias (por entrypoint, roteador e serviço), ativadas
em `deploy/traefik/traefik.yml`.

## Alertas

`deploy/monitoring/alerts.yml` define as regras abaixo. Os limiares são pontos de partida para uma
instância única; ajuste ao seu volume.

| Alerta | Dispara quando | Severidade | O que olhar |
|---|---|---|---|
| `LinktorInstanceDown` | O Prometheus não coleta o alvo há 1 minuto | crítica | Container do backend e rede. |
| `LinktorNATSDown` | `linktor_nats_up == 0` por 1 minuto | crítica | Container do NATS. Nada entra nem sai enquanto durar. |
| `LinktorDLQBacklog` | Há mensagem em `LINKTOR_DLQ` por 5 minutos | aviso | Log do backend, linhas do observador da DLQ. |
| `LinktorDLQBacklogCritical` | Mais de 50 mensagens em `LINKTOR_DLQ` por 5 minutos | crítica | Um estágio do pipeline está falhando de forma sistemática. |
| `LinktorConsumerLagHigh` | Mais de 1000 pendentes em um consumidor por 10 minutos | aviso | Processamento não acompanha a entrada. |
| `LinktorConsumerRedeliveriesRising` | Mais de 20 reentregas em 10 minutos | aviso | Mensagem que falha repetidamente. |
| `LinktorPublishFailures` | Há falhas de publicação no NATS por 5 minutos | aviso | Conectividade com o NATS, espaço em disco do stream. |
| `LinktorInboundFailureRatioHigh` | Mais de 5% das mensagens recebidas falham, por 10 minutos | aviso | **Observabilidade → Logs**, origem **Webhook**. |
| `LinktorOutboundFailureRatioHigh` | Mais de 5% dos envios falham, por 10 minutos | aviso | Credenciais do canal e limites do provedor. |
| `LinktorHTTP5xxHigh` | Mais de 5% das respostas são `5xx`, por 10 minutos | aviso | Log do backend. |
| `LinktorHTTPLatencyHigh` | p99 de latência acima de 1 s, por 10 minutos | aviso | Banco e carga da máquina. |
| `LinktorGuardFailOpenSystematic` | Uma política falha aberta por erro de consulta há mais de 1 hora | crítica | A política está cega: envios passam sem avaliação. Não ligue `enforce` enquanto durar. |
| `LinktorGuardDryRunBlockRateHigh` | Mais de 5% dos envios do WhatsApp oficial seriam bloqueados pela janela de 24 h, por 6 horas | aviso | Fluxo legítimo fora da janela; investigue antes de ligar `enforce`. |
| `LinktorSandboxBlockSurge` | Mais de 50 envios barrados pela lista do sandbox em 1 hora, por 30 minutos | aviso | Lista de destinatários do sandbox vazia ou incompleta. |

Valide o arquivo depois de editar:

```bash
promtool check rules deploy/monitoring/alerts.yml
```

As regras só viram notificação com um Alertmanager apontado no `prometheus.yml`.

## Grafana

O repositório provisiona por arquivo:

| Arquivo | Conteúdo |
|---|---|
| `deploy/monitoring/grafana/provisioning/datasources/prometheus.yml` | Datasource `Prometheus` (uid `linktor-prometheus`) em `http://prometheus:9090`. Ajuste a URL. |
| `deploy/monitoring/grafana/provisioning/dashboards/provider.yml` | Carrega os JSON de `dashboards/` na pasta `LINKTOR`. Edição pela interface fica bloqueada. |
| `deploy/monitoring/grafana/dashboards/linktor-guards.json` | Dashboard das proteções de envio: bloqueios por motivo e modo, o que o `dry_run` bloquearia e falhas abertas separadas por causa. |

Subir um Grafana local com esse provisionamento, a partir da raiz do repositório:

```bash
docker run -d --name linktor-grafana -p 3000:3000 \
  -v "$PWD/deploy/monitoring/grafana/provisioning:/etc/grafana/provisioning" \
  -v "$PWD/deploy/monitoring/grafana/dashboards:/var/lib/grafana/dashboards" \
  grafana/grafana:latest
```

O único dashboard versionado cobre as proteções de envio. Para o pipeline, as filas e o HTTP, monte
painéis a partir das métricas listadas acima.

## Logs

O backend escreve em `stdout`. Com `LINKTOR_LOG_FORMAT=json` (padrão), cada linha é um objeto JSON;
com `console`, texto legível para desenvolvimento. O nível vem de `LINKTOR_LOG_LEVEL`.

Cada requisição HTTP gera uma linha `HTTP Request` com `request_id`, `method`, `path`, `query`,
`status`, `latency`, `client_ip`, `user_agent` e `body_size`.

```bash
cd /opt/linktor/deploy
COMPOSE="docker compose -f docker-compose.prod.yml --env-file ../.env"

$COMPOSE logs -f backend
$COMPOSE logs --since 1h backend | grep '"level":"error"'
$COMPOSE logs --since 1h traefik | grep -i acme
```

No Compose de produção, os logs usam o driver `json-file` com rotação de 10 MB e 5 arquivos por
container. Não há agregação de logs no repositório; para retenção longa, envie o `stdout` dos
containers ao seu coletor.

:::warning Os logs contêm dados sensíveis
A linha de acesso grava a query string inteira. Isso inclui o token do WebSocket do painel
(`/api/v1/ws?token=...`), o token do Linktor Bridge e as assinaturas de URL de mídia. Trate os logs
do backend como material sensível.
:::

Mensagens que valem um alerta por busca em log:

| Texto | Significado |
|---|---|
| `messaging features disabled` | O backend iniciou sem NATS. |
| `HTTP rate limiting disabled` | O backend iniciou sem Redis: sem rate limit, deduplicação de webhook nem revogação de mídia. |
| `MEDIA_SIGNING_SECRET not set` | URLs de mídia sem assinatura e sem validade. |
| `inbound message dropped (non-retryable)` | Mensagem recebida descartada por erro permanente. |
| `WhatsApp refused client version` | Veja [a seção abaixo](#whatsapp-versao). |

## Tela Observabilidade

No painel, **Observabilidade** (visível para os papéis administrador e proprietário) reúne o que o
operador precisa sem sair do Linktor. Descrição completa em
[Observabilidade](../painel/observabilidade.md).

| Aba | Mostra | Rota da API |
|---|---|---|
| **Logs** | Registro por canal de conexões, envios, falhas e webhooks recebidos, com filtro por canal, nível e origem (**Canal**, **Fila**, **Sistema**, **Webhook**). | `GET /api/v1/observability/logs` |
| **Filas de Mensagens** | Streams do NATS e seus consumidores: pendentes, aguardando ack, reentregues. Permite **Resetar Consumidor**. | `GET /api/v1/observability/queue`, `GET /api/v1/observability/queue/:stream`, `POST /api/v1/observability/queue/reset-consumer` |
| **Estatísticas** | Totais do período. | `GET /api/v1/observability/stats?period=day` |

As rotas `/api/v1/observability/*` exigem um **usuário** com papel `admin` ou `owner`. Uma chave de
API (`X-API-Key`) recebe `403` nelas, qualquer que seja o scope. Para chamá-las por script, obtenha
um token de acesso com o login (válido por 15 minutos, por padrão):

```bash
TOKEN=$(curl -s -X POST "$LINKTOR_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@suaempresa.com","password":"SUA_SENHA"}' | jq -r '.data.access_token')
```

Os logs de canal ficam no PostgreSQL e **não são apagados automaticamente**. Agende a limpeza:

```bash
curl -X POST "$LINKTOR_URL/api/v1/observability/logs/cleanup" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"retention_days": 30}'
```

A chamada apaga os logs do tenant do usuário com mais de `retention_days` dias (30 se o corpo for
omitido) e responde `{"success": true, "deleted_count": 123}`.

:::warning Resetar um consumidor afeta todos os tenants
As filas são compartilhadas pela instalação. **Resetar Consumidor** faz o consumidor reprocessar as
mensagens pendentes de todos os tenants. Use só durante uma investigação, sabendo o que está na fila.
:::

## Versão do cliente WhatsApp {#whatsapp-versao}

Os canais de WhatsApp não-oficial anunciam ao WhatsApp uma versão de cliente. Quando o WhatsApp
deixa de aceitar essa versão, ele recusa a conexão com o erro **405** e todos os canais não-oficiais
da instalação passam a "conectar e desconectar" ao mesmo tempo, sem que nada tenha mudado do seu
lado.

O backend tem um monitor para isso:

- consulta a versão atual publicada em `web.whatsapp.com` na inicialização, a cada 6 horas e antes
  de conectar um canal;
- quando encontra uma versão mais nova, passa a usá-la em todo o processo (a menos de
  `WHATSAPP_VERSION_AUTO_UPDATE=false`);
- ao receber um 405, força uma nova consulta (no máximo uma a cada 2 minutos), adota a versão nova
  e reconecta o canal **sem novo pareamento**;
- registra cada recusa para exibir no painel.

No topo de **Observabilidade** fica o cartão **Versão do cliente WhatsApp (canais não-oficiais)**,
com **Em uso**, **Atual no WhatsApp**, **Embutida no build**, **Última verificação** e o botão
**Verificar agora**.

| Estado no cartão | `status` na API | Significado | Ação |
|---|---|---|---|
| **Em dia** | `ok` | O último login foi aceito e não há versão mais nova pendente. | Nenhuma. |
| **Desatualizada** | `behind` | Existe versão mais nova e a atualização automática está desligada. | Remova `WHATSAPP_VERSION_AUTO_UPDATE=false` ou atualize o Linktor. |
| **Recusada pelo WhatsApp** | `outdated` | O WhatsApp recusou a versão (405) depois do último login bem-sucedido. | Clique em **Verificar agora**. Se continuar, atualize o Linktor. |
| **Não verificada** | `unknown` | Nenhuma consulta teve sucesso e nenhum canal conectou ainda. | Confira se o servidor alcança `web.whatsapp.com`. |

Pela API:

```bash
# estado atual (use o TOKEN obtido na seção anterior)
curl "$LINKTOR_URL/api/v1/observability/whatsapp-version" \
  -H "Authorization: Bearer $TOKEN"

# força uma verificação agora
curl -X POST "$LINKTOR_URL/api/v1/observability/whatsapp-version/check" \
  -H "Authorization: Bearer $TOKEN"
```

A resposta traz `status`, `builtin`, `current`, `latest`, `auto_update`, `last_check_at`,
`last_check_error`, `last_outdated_at`, `outdated_count` e `last_login_at`.

No log do backend:

| Linha | O que aconteceu |
|---|---|
| `WhatsApp client version updated <antiga> -> <nova> after 405; reconnecting` | O monitor resolveu sozinho. |
| `WhatsApp refused client version <versão> and no newer build is available; update whatsmeow` | Não há versão mais nova publicada que resolva. É preciso atualizar o Linktor para uma versão com a biblioteca `whatsmeow` mais recente. |

O estado do monitor vive na memória do processo e reinicia a cada boot. Para ser avisado sem abrir
o painel, consulte `GET /api/v1/observability/whatsapp-version` periodicamente e alerte quando
`status` for `outdated`, ou alerte pela segunda linha de log acima. Detalhes do canal em
[WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).
