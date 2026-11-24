---
sidebar_position: 4
title: Configuração
description: Referência de todas as variáveis de ambiente e chaves do config.yaml lidas pelo backend, pelo painel e pelos arquivos Compose.
---

# Configuração

Referência das variáveis que o código realmente lê. Para o passo a passo de implantação, veja
[Produção](./producao.md).

## Como a configuração é carregada

O backend combina três fontes, da menor para a maior precedência:

1. **Padrões** embutidos no código.
2. **`config.yaml`**, procurado em `.` (diretório de trabalho), `./config` e `/etc/linktor`. É
   opcional.
3. **Variáveis de ambiente**.

Antes disso, o backend carrega um arquivo `.env` do diretório de trabalho, se existir, sem
sobrescrever variáveis já definidas no ambiente.

Há dois estilos de variável:

- **Com prefixo `LINKTOR_`**, mapeadas para chaves do `config.yaml`: `LINKTOR_<SEÇÃO>_<CHAVE>`
  corresponde a `<seção>.<chave>`. Exemplo: `LINKTOR_DATABASE_HOST` é `database.host`.
- **Lidas diretamente** do ambiente, sem equivalente no `config.yaml` (`BASE_URL`, `MINIO_*`,
  `MEDIA_*`, chaves de IA e outras). Algumas delas também começam com `LINKTOR_`; as tabelas abaixo
  indicam a chave YAML só quando ela existe.

:::note A imagem Docker não contém `config.yaml`
O `Dockerfile` não copia o `config.yaml` da raiz do repositório. Dentro do container valem os
padrões do código e as variáveis de ambiente — por isso a porta padrão é `8080` no container e
`8081` quando você roda `go run ./cmd/server` na raiz (onde o `config.yaml` existe). Os arquivos
Compose definem `LINKTOR_SERVER_PORT` explicitamente.
:::

A coluna **Obrigatória** indica o que impede o backend de iniciar ou de funcionar em produção.

## Servidor

| Variável | Chave YAML | Padrão | Obrigatória | Efeito |
|---|---|---|---|---|
| `LINKTOR_SERVER_PORT` | `server.port` | `8080` | Não | Porta HTTP. O `config.yaml` do repositório e os Compose usam `8081`. |
| `LINKTOR_SERVER_HOST` | `server.host` | `0.0.0.0` | Não | Endereço de escuta. |
| `LINKTOR_SERVER_MODE` | `server.mode` | `debug` | Em produção: `release` | Veja [o que muda em `release`](#modo-release). |
| `LINKTOR_SERVER_SHUTDOWN_TIMEOUT` | `server.shutdown_timeout` | `30` | Não | Segundos de espera pelas requisições em andamento ao encerrar. |
| `BASE_URL` | — | `http://<host>:<porta>` | Em produção | Origem pública da API. Usada no registro do webhook do Telegram, nas URLs de retorno do OAuth do Facebook e do Instagram, na URL de webhook devolvida pelo Embedded Signup e como base das URLs de mídia. |
| `LINKTOR_VERSION` | — | vazio | Não | Texto devolvido no campo `version` de `GET /health`. |
| `LINKTOR_LOG_LEVEL` | `log.level` | `info` | Não | `debug`, `info`, `warn` ou `error`. |
| `LINKTOR_LOG_FORMAT` | `log.format` | `json` | Não | `json` ou `console` (legível, colorido). |

### Modo `release` {#modo-release}

Com `LINKTOR_SERVER_MODE=release`:

- o backend **recusa iniciar** se `jwt.secret` estiver vazio, for um dos valores de exemplo ou tiver
  menos de 32 caracteres;
- o mesmo vale para `crypto.encryption_key`, que além disso precisa ser diferente de `jwt.secret`;
- os dados de demonstração não são criados (a menos de `LINKTOR_SEED_DEMO=true`);
- webhooks de provedores sem segredo configurado no canal são recusados;
- com `LINKTOR_CORS_ALLOWED_ORIGINS` vazio, toda origem de outro host é negada (em `debug`, origens
  `localhost` e `127.0.0.1` são aceitas).

## Banco de dados

| Variável | Chave YAML | Padrão | Obrigatória | Efeito |
|---|---|---|---|---|
| `LINKTOR_DATABASE_HOST` | `database.host` | `localhost` | Sim | Host do PostgreSQL. |
| `LINKTOR_DATABASE_PORT` | `database.port` | `5432` | Não | |
| `LINKTOR_DATABASE_USER` | `database.user` | `linktor` | Sim | |
| `LINKTOR_DATABASE_PASSWORD` | `database.password` | `linktor` | Sim | |
| `LINKTOR_DATABASE_DATABASE` | `database.database` | `linktor` | Sim | Nome do banco. |
| `LINKTOR_DATABASE_SSL_MODE` | `database.ssl_mode` | `disable` | Não | Valor de `sslmode` na conexão (`disable`, `require`, `verify-full`...). |
| `LINKTOR_DATABASE_MAX_OPEN_CONNS` | `database.max_open_conns` | `50` | Não | Tamanho máximo do pool. |
| `LINKTOR_DATABASE_MAX_IDLE_CONNS` | `database.max_idle_conns` | `10` | Não | Conexões mínimas mantidas abertas. |
| `LINKTOR_DATABASE_MAX_LIFETIME` | `database.max_lifetime` | `5` | Não | Vida máxima de uma conexão, em minutos. |

O usuário do banco precisa criar extensões (`pgcrypto`, `uuid-ossp`, `vector`) e tabelas: as
migrations rodam na inicialização. Use uma imagem com `pgvector`.

## Redis

| Variável | Chave YAML | Padrão | Obrigatória | Efeito |
|---|---|---|---|---|
| `LINKTOR_REDIS_HOST` | `redis.host` | `localhost` | Recomendada | Redis do rate limit, da deduplicação de webhooks e da revogação de mídia. Se a conexão falhar, o backend sobe com esses três recursos desligados. |
| `LINKTOR_REDIS_PORT` | `redis.port` | `6379` | Não | |
| `LINKTOR_REDIS_PASSWORD` | `redis.password` | vazio | Se o Redis exigir | |
| `LINKTOR_REDIS_DB` | `redis.db` | `0` | Não | |
| `REDIS_URL` | — | vazio | Não | Cliente Redis **adicional**, no formato `redis://:senha@host:porta/db`. Liga o cache do VRE e dos papéis e inclui o Redis na checagem de `GET /ready`. Não substitui as variáveis acima. |

## NATS

| Variável | Chave YAML | Padrão | Obrigatória | Efeito |
|---|---|---|---|---|
| `LINKTOR_NATS_URL` | `nats.url` | `nats://localhost:4222` | Sim | Servidor NATS com JetStream habilitado. Sem conexão, o backend sobe mas não processa mensagens. |
| `LINKTOR_NATS_CLIENT_ID` | `nats.client_id` | `linktor-server` | Não | Nome da conexão, visível no monitoramento do NATS. |
| `LINKTOR_NATS_CLUSTER_ID` | `nats.cluster_id` | `linktor-cluster` | Não | Lido, mas sem efeito no comportamento atual. |
| `LINKTOR_NATS_STREAM_REPLICAS` | `nats.stream_replicas` | `1` | Não | Réplicas de todos os streams. Use `3` apenas em um cluster NATS com pelo menos três nós; em servidor único, a criação dos streams falha com valor maior que `1`. |

## Autenticação, JWT e cookies

| Variável | Chave YAML | Padrão | Obrigatória | Efeito |
|---|---|---|---|---|
| `LINKTOR_JWT_SECRET` | `jwt.secret` | `change-me-in-production` | Sim em `release` | Chave de assinatura dos tokens. Trocar encerra todas as sessões. |
| `LINKTOR_JWT_ACCESS_TOKEN_TTL` | `jwt.access_token_ttl` | `15` | Não | Validade do token de acesso, em **minutos**. |
| `LINKTOR_JWT_REFRESH_TOKEN_TTL` | `jwt.refresh_token_ttl` | `168` | Não | Validade do token de renovação, em **horas**. |
| `LINKTOR_JWT_ISSUER` | `jwt.issuer` | `linktor` | Não | Emissor gravado nos tokens. |
| `LINKTOR_JWT_COOKIE_SECURE` | `jwt.cookie_secure` | `true` | Não | Marca os cookies `access_token` e `refresh_token` como `Secure`. Use `false` somente em HTTP local, senão o navegador descarta o cookie. |
| `LINKTOR_JWT_COOKIE_DOMAIN` | `jwt.cookie_domain` | vazio | Não | Atributo `Domain` do cookie, ex.: `.suaempresa.com`. Vazio restringe ao host da API. |
| `LINKTOR_JWT_COOKIE_SAME_SITE` | `jwt.cookie_same_site` | `lax` | Não | `lax`, `strict` ou `none`. `lax` funciona quando painel e API são subdomínios do mesmo site. |

## Criptografia de credenciais

| Variável | Chave YAML | Padrão | Obrigatória | Efeito |
|---|---|---|---|---|
| `LINKTOR_CRYPTO_ENCRYPTION_KEY` | `crypto.encryption_key` | `change-me-in-production` | Sim em `release` | Segredo do qual se deriva a chave AES-256 que cifra as credenciais dos canais. Mínimo absoluto de 16 caracteres; em `release`, 32 e diferente do segredo JWT. |
| — | `crypto.previous_keys` | lista vazia | Não | Chaves antigas ainda aceitas para decifrar, durante uma troca de chave. |

:::warning `previous_keys` só funciona com `config.yaml`
A variável `LINKTOR_CRYPTO_PREVIOUS_KEYS` é **ignorada** quando não existe um `config.yaml` que
declare a chave `crypto.previous_keys` — que é o caso da imagem Docker. Para trocar a chave em um
deploy com containers, monte um `config.yaml` em `/etc/linktor/config.yaml` com a lista. O
procedimento completo está em [Backup e atualização](./backup-e-atualizacao.md).
:::

## Dados iniciais

| Variável | Padrão | Efeito |
|---|---|---|
| `LINKTOR_SEED_DEMO` | vazio | `true` executa o seed também em `release`. Fora de `release` o seed sempre é tentado. |
| `SEED_ADMIN_PASSWORD` | vazio | Senha de `admin@demo.com` e `agent@demo.com`. Vazia: senha aleatória, impressa uma vez no log. |
| `SEED_RESET_DESTRUCTIVE` | vazio | `true` **apaga** mensagens, conversas, contatos, canais, usuários e tenants antes de recriar o seed. Só tem efeito quando a tabela `users` está vazia. Nunca use fora de desenvolvimento. |

O seed só age quando a tabela `users` está vazia.

## Mídia e armazenamento

O backend escolhe o armazenamento nesta ordem: MinIO/S3 se `MINIO_ENDPOINT` estiver definido; senão
disco local se `MEDIA_UPLOAD_DIR` estiver definido; senão nenhum (a mídia recebida mantém a URL do
provedor).

| Variável | Padrão | Obrigatória | Efeito |
|---|---|---|---|
| `MINIO_ENDPOINT` | vazio | Recomendada | `host:porta` do serviço S3, sem esquema. |
| `MINIO_ACCESS_KEY` | vazio | Com MinIO | |
| `MINIO_SECRET_KEY` | vazio | Com MinIO | |
| `MINIO_BUCKET` | `linktor-media` | Não | |
| `MINIO_REGION` | vazio | Não | |
| `MINIO_USE_SSL` | `false` | Não | `true` para HTTPS. |
| `MEDIA_UPLOAD_DIR` | vazio | Não | Diretório local, usado quando não há MinIO. |
| `MEDIA_UPLOAD_BASE_URL` | `/uploads/media` | Não | Prefixo de URL do armazenamento local. |
| `MEDIA_PUBLIC_BASE_URL` | valor de `BASE_URL` | Não | Origem pública usada para montar as URLs do proxy de mídia (`/api/v1/media/...`). Se esta e `BASE_URL` estiverem vazias, as URLs passam a ser links pré-assinados do S3, que embutem o `MINIO_ENDPOINT`. |
| `MEDIA_FETCH_ALLOWED_HOSTS` | vazio | Não | Hosts em rede privada dos quais o WhatsApp não-oficial pode baixar mídia para envio, separados por vírgula. Com porta (`192.168.0.10:9000`) vale só para aquela porta. Vazio mantém a proteção contra SSRF integral. |

### URLs de mídia assinadas

| Variável | Padrão | Efeito |
|---|---|---|
| `MEDIA_SIGNING_SECRET` | vazio | Liga a assinatura: as URLs de mídia saem como `/api/v1/media/<chave>?exp=<unix>&sig=<hmac>`. Vazio: as URLs não são assinadas e **nunca expiram** (o backend registra um aviso ao iniciar). |
| `MEDIA_URL_TTL` | `168h` | Validade de cada assinatura, em duração do Go (`24h`, `30m`). |
| `MEDIA_UNSIGNED_UNTIL` | vazio | Data limite em que URLs antigas, sem assinatura, ainda são aceitas. RFC 3339 ou `AAAA-MM-DD` (meia-noite UTC). Vazio ou inválido: URLs sem assinatura continuam aceitas indefinidamente. |

Para fechar o acesso sem assinatura são necessárias as duas: `MEDIA_SIGNING_SECRET` definido **e**
`MEDIA_UNSIGNED_UNTIL` com uma data já passada. Combine a data com quem consome as URLs antes de
defini-la.

### VRE (respostas visuais)

| Variável | Padrão | Efeito |
|---|---|---|
| `VRE_TEMPLATES_PATH` | `./templates` | Diretório dos modelos SVG. A imagem Docker já traz `/app/templates`. |
| `VRE_UPLOAD_DIR` | vazio | Diretório onde as imagens geradas são gravadas. Vazio: o armazenamento do VRE fica desligado. |
| `VRE_UPLOAD_BASE_URL` | `/uploads/vre` | Prefixo de URL das imagens geradas. |

:::note
O backend não publica uma rota `/uploads`. Se você depender das URLs de `VRE_UPLOAD_BASE_URL`,
aponte-a para um servidor que sirva o diretório.
:::

## CORS, WebSocket e webhooks

| Variável | Padrão | Efeito |
|---|---|---|
| `LINKTOR_CORS_ALLOWED_ORIGINS` | vazio | Origens autorizadas a chamar a API pelo navegador, separadas por vírgula, com esquema e sem barra final (`https://app.suaempresa.com`). Requisições em que a origem tem o mesmo host da API são sempre aceitas. |
| `LINKTOR_WS_ALLOWED_ORIGINS` | vazio | Origens autorizadas a abrir WebSocket — o do painel (`/api/v1/ws`) e o do WebChat (`/ws/:channelId`). Inclua a origem do painel e a de cada site que hospeda o widget. |
| `LINKTOR_WS_ENFORCE_ORIGIN` | vazio | Só WebChat. `true` aceita apenas a lista acima; sem isso, origens `localhost` também passam. |
| `LINKTOR_WS_ALLOW_EMPTY_ORIGIN` | vazio | Só WebChat. `true` aceita conexões sem cabeçalho `Origin` (clientes que não são navegador). |
| `LINKTOR_REQUIRE_WEBHOOK_SECRETS` | vazio | `true` recusa webhooks de canais sem segredo configurado também em `debug`. Em `release` isso já vale. |
| `LINKTOR_WEBHOOK_INSECURE_LOOPBACK` | vazio | `true` desliga a verificação de TLS do webhook de saída **somente** para destinos em loopback (consumidor no mesmo host com certificado autoassinado). |

As quatro variáveis booleanas de WebSocket aceitam `1`, `true`, `yes` ou `on`; as demais, apenas
`true`.

## Canais

| Variável | Padrão | Efeito |
|---|---|---|
| `LINKTOR_ENABLED_CHANNEL_TYPES` | vazio (todos) | Lista de tipos de canal que podem ser criados e conectados, separados por vírgula (ex.: `whatsapp_official,telegram,webchat`). |
| `LINKTOR_WA_WINDOW_ENFORCEMENT` | `dry_run` | Janela de 24 horas do WhatsApp oficial: `off`, `dry_run` (só registra o que seria bloqueado) ou `enforce` (bloqueia antes de chamar a Meta). Valor desconhecido vira `dry_run`. |
| `LINKTOR_WA_TEMPLATE_ENFORCEMENT` | `dry_run` | Mesmo contrato, para o envio de modelos com status inutilizável (rejeitado, pausado). |
| `LINKTOR_CONTACT_AVATAR_INTERVAL` | `1s` | Intervalo mínimo entre consultas de foto de contato no mesmo canal, em duração do Go. |
| `WHATSAPP_VERSION_AUTO_UPDATE` | ligado | `false`, `0`, `no` ou `off` impedem o WhatsApp não-oficial de adotar sozinho uma versão de cliente mais nova. O monitor continua verificando e alertando. Veja [Monitoramento](./monitoramento.md). |
| `LINKTOR_GRAPH_API_URL` | `https://graph.facebook.com` | Sobrescreve a URL da Graph API da Meta. Serve para apontar para o mock de desenvolvimento. |
| `LINKTOR_VENDAX_BRIDGE_ENABLED` | vazio | `true` liga a ponte NATS com o VendaX Core. Só faz sentido para quem opera os dois sistemas no mesmo NATS. |

### Meta, OAuth e Embedded Signup

| Variável | Padrão | Efeito |
|---|---|---|
| `WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET` | vazio | App Secret do aplicativo Meta usado no Embedded Signup do WhatsApp. Sem ele, a conclusão do cadastro responde erro. |
| `WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET_<APP_ID>` | vazio | Segredo específico por aplicativo, quando há mais de um. O sufixo é o App ID em maiúsculas, com qualquer caractere que não seja letra ou dígito trocado por `_`. Tem precedência sobre a variável sem sufixo. |

Não há variáveis globais de App ID ou App Secret para os fluxos OAuth do Facebook e do Instagram: as
credenciais ficam em cada canal. Do ambiente, esses fluxos usam apenas `BASE_URL`, para montar as
URLs de retorno `/api/v1/oauth/facebook/callback` e `/api/v1/oauth/instagram/callback`.

### Sandbox

O sandbox não tem variável de ambiente: é um atributo de cada canal, definido na criação, com a
lista de destinatários permitidos gerenciada pelo painel. Veja [Sandbox](../painel/sandbox.md). As
duas variáveis `LINKTOR_WA_*_ENFORCEMENT` acima controlam outras proteções de envio, não o sandbox.

## Provedores de IA

Todos opcionais. Sem nenhum, os recursos de IA dos bots ficam indisponíveis.

| Variável | Efeito |
|---|---|
| `OPENAI_API_KEY` | Registra o provedor OpenAI. |
| `OPENAI_ORG_ID` | Organização da OpenAI. |
| `OPENAI_DEFAULT_MODEL` | Modelo usado quando o bot não especifica. |
| `ANTHROPIC_API_KEY` | Registra o provedor Anthropic. |
| `ANTHROPIC_DEFAULT_MODEL` | Modelo padrão. |
| `OLLAMA_BASE_URL` | Endereço de um servidor Ollama. O provedor é sempre registrado; fica disponível quando o endereço responde. |
| `OLLAMA_DEFAULT_MODEL` | Modelo padrão. |

## Observabilidade

Não há variáveis para ligar métricas ou health checks: `GET /metrics`, `GET /health` e `GET /ready`
estão sempre ativos na mesma porta da API. As variáveis relacionadas são `LINKTOR_LOG_LEVEL`,
`LINKTOR_LOG_FORMAT` e `LINKTOR_VERSION`, na seção [Servidor](#servidor).

## Painel (Next.js)

### No build

Gravadas no bundle quando a imagem é compilada. Mudar o valor exige novo build.

| Variável | Padrão | Obrigatória | Efeito |
|---|---|---|---|
| `NEXT_PUBLIC_API_URL` | — | Sim (o build falha sem ela) | Base da API, ex.: `https://api.suaempresa.com/api/v1`. |
| `NEXT_PUBLIC_WS_URL` | — | Sim (o build falha sem ela) | WebSocket do painel, ex.: `wss://api.suaempresa.com/api/v1/ws`. |
| `NEXT_PUBLIC_WEBHOOK_BASE_URL` | `NEXT_PUBLIC_API_URL` sem `/api/v1` | Não | Origem pública da API, usada nas URLs de webhook exibidas nas telas de canal e no snippet do WebChat. |
| `NEXT_PUBLIC_SHOW_DEMO_CREDENTIALS` | vazio | Não | `true` mostra a dica de credenciais de demonstração na tela de login. Não use em produção. |
| `NEXT_PROXY_API_ORIGIN` | `http://localhost:8081` | Não | Destino do proxy `/api` do servidor de desenvolvimento, quando `NEXT_PUBLIC_API_URL` não está definida. |

### Em execução

Lidas pelo container em cada requisição; permitem que a mesma imagem sirva instalações diferentes.
Quando ausentes, valem os valores do build.

| Variável | Efeito |
|---|---|
| `LINKTOR_ADMIN_API_URL` | Substitui `NEXT_PUBLIC_API_URL`. Aceita caminho relativo (`/api/v1`), resolvido contra o endereço em que a página foi aberta — útil quando painel e API ficam atrás do mesmo proxy. |
| `LINKTOR_ADMIN_WS_URL` | Substitui `NEXT_PUBLIC_WS_URL`. Também aceita caminho relativo (`/api/v1/ws`). |
| `LINKTOR_ADMIN_WEBHOOK_BASE_URL` | Substitui `NEXT_PUBLIC_WEBHOOK_BASE_URL`. Deve ser absoluta: é exibida na tela e copiada para o console dos provedores. Se ausente e `LINKTOR_ADMIN_API_URL` estiver definida, é derivada dela. |
| `LINKTOR_ADMIN_VERSION` | Versão do painel exposta à página, para conferir o que está no ar. |

## Variáveis dos arquivos Compose

Estas não são lidas pelo backend: são interpoladas pelo Docker Compose e repassadas (às vezes com
outro nome) aos containers.

### `docker-compose.yml` (local)

| Variável | Padrão | Vira no container |
|---|---|---|
| `LINKTOR_API_PORT` | `8081` | `LINKTOR_SERVER_PORT` e a porta publicada |
| `LINKTOR_ADMIN_PORT` | `3000` | Porta publicada do painel |
| `LINKTOR_POSTGRES_PORT`, `LINKTOR_REDIS_PORT`, `LINKTOR_NATS_PORT`, `LINKTOR_NATS_MONITOR_PORT`, `LINKTOR_MINIO_PORT`, `LINKTOR_MINIO_CONSOLE_PORT` | `5432`, `6379`, `4222`, `8222`, `9000`, `9001` | Portas publicadas |
| `LINKTOR_BASE_URL` | `http://localhost:<porta da API>` | `BASE_URL` |
| `LINKTOR_MINIO_ENDPOINT`, `LINKTOR_MINIO_ACCESS_KEY`, `LINKTOR_MINIO_SECRET_KEY`, `LINKTOR_MINIO_BUCKET`, `LINKTOR_MINIO_USE_SSL` | `minio:9000`, `linktor`, `linktor123`, `linktor-media`, `false` | `MINIO_*` |
| `LINKTOR_MEDIA_SIGNING_SECRET`, `LINKTOR_MEDIA_URL_TTL`, `LINKTOR_MEDIA_UNSIGNED_UNTIL` | vazio, `168h`, vazio | `MEDIA_*` |
| `SEED_ADMIN_PASSWORD` | vazio | `SEED_ADMIN_PASSWORD` |
| `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_WS_URL`, `NEXT_PUBLIC_WEBHOOK_BASE_URL` | apontam para `localhost` na porta da API | Argumentos de build do painel |

Banco, Redis, NATS, segredo JWT e modo (`debug`) são fixos nesse arquivo.

### `deploy/docker-compose.prod.yml`

Lidas de `/opt/linktor/.env`.

| Variável | Vira no container |
|---|---|
| `ROOT_DOMAIN` | Regras de roteamento do Traefik, `BASE_URL`, `MINIO_ENDPOINT` (`s3.<domínio>`), origens de CORS e WebSocket |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | Credenciais do PostgreSQL e `LINKTOR_DATABASE_*` |
| `REDIS_PASSWORD` | `requirepass` do Redis e `LINKTOR_REDIS_PASSWORD` |
| `JWT_SECRET` | `LINKTOR_JWT_SECRET` |
| `CRYPTO_ENCRYPTION_KEY` | `LINKTOR_CRYPTO_ENCRYPTION_KEY` |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Usuário root do MinIO e `MINIO_ACCESS_KEY`/`MINIO_SECRET_KEY` |
| `MINIO_BUCKET` | `MINIO_BUCKET` (padrão `linktor-media`) |
| `BACKEND_IMAGE_TAG`, `ADMIN_IMAGE_TAG`, `LANDING_IMAGE_TAG` | Tag de cada imagem (padrão `latest`) |
| `BACKUP_DIR`, `BACKUP_RETENTION_DAYS`, `BACKUP_S3_ENDPOINT`, `BACKUP_S3_BUCKET`, `BACKUP_S3_ACCESS_KEY`, `BACKUP_S3_SECRET_KEY` | Lidas só por `deploy/scripts/backup.sh` |

`ACME_EMAIL` consta no `.env.example` mas não é usada por nenhum serviço.

## Exemplo de `config.yaml`

```yaml
server:
  port: 8081
  host: "0.0.0.0"
  mode: "release"
  shutdown_timeout: 30

database:
  host: "postgres"
  port: 5432
  user: "linktor"
  password: "TROQUE"
  database: "linktor"
  ssl_mode: "disable"
  max_open_conns: 50
  max_idle_conns: 10
  max_lifetime: 5

redis:
  host: "redis"
  port: 6379
  password: "TROQUE"
  db: 0

nats:
  url: "nats://nats:4222"
  client_id: "linktor-server"
  stream_replicas: 1

jwt:
  secret: "TROQUE-32-CARACTERES-OU-MAIS"
  access_token_ttl: 15
  refresh_token_ttl: 168
  issuer: "linktor"
  cookie_secure: true
  cookie_domain: ""
  cookie_same_site: "lax"

crypto:
  encryption_key: "TROQUE-OUTROS-32-CARACTERES-OU-MAIS"
  previous_keys: []

log:
  level: "info"
  format: "json"
```

Uma variável de ambiente definida sempre vence o valor do arquivo.
