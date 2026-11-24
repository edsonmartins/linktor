---
sidebar_position: 2
title: Instalação
description: Do clone ao primeiro login no painel, em ambiente local com Docker Compose, e como rodar em modo de desenvolvimento.
---

# Instalação

Este tutorial sobe o Linktor completo na sua máquina com Docker Compose e termina com você logado no
painel. Ele serve para conhecer e desenvolver; para um servidor exposto à internet, siga
[Produção](./producao.md).

## Antes de começar

- Docker com o plugin Compose.
- Git.
- Portas livres: `3000` (painel), `8081` (API), `5432`, `6379`, `4222`, `8222`, `9000` e `9001`.

Só para o modo de desenvolvimento (última seção): Go na versão indicada em `go.mod` e Node.js 24
(a versão usada no CI e na imagem do painel).

## 1. Clone o repositório

```bash
git clone https://github.com/edsonmartins/linktor.git
cd linktor
```

## 2. Crie o arquivo `.env`

```bash
cp .env.example .env
```

O arquivo já funciona como está. O `docker compose` lê dele as portas publicadas, as URLs do painel
e a senha do usuário inicial (`SEED_ADMIN_PASSWORD`, que vem como `admin123` no exemplo).

## 3. Suba os serviços

```bash
docker compose up -d --build
```

A primeira execução compila o backend e o painel e leva alguns minutos. O que sobe:

| Serviço | Container | Porta no host | Variável para mudar a porta |
|---|---|---|---|
| Painel (Next.js) | `linktor-admin` | `3000` | `LINKTOR_ADMIN_PORT` |
| API (Go) | `linktor-backend` | `8081` | `LINKTOR_API_PORT` |
| PostgreSQL | `linktor-postgres` | `5432` | `LINKTOR_POSTGRES_PORT` |
| Redis | `linktor-redis` | `6379` | `LINKTOR_REDIS_PORT` |
| NATS | `linktor-nats` | `4222` (cliente), `8222` (monitoramento) | `LINKTOR_NATS_PORT`, `LINKTOR_NATS_MONITOR_PORT` |
| MinIO | `linktor-minio` | `9000` (API S3), `9001` (console) | `LINKTOR_MINIO_PORT`, `LINKTOR_MINIO_CONSOLE_PORT` |

Acompanhe até todos ficarem saudáveis:

```bash
docker compose ps
```

O painel só inicia depois que o backend responde em `/health`, e o backend só inicia depois que
banco, Redis, NATS e MinIO estão saudáveis.

:::note Portas ocupadas
Se alguma porta já estiver em uso, mude-a no `.env` **antes** de subir. Ao mudar `LINKTOR_API_PORT`,
ajuste também `LINKTOR_BASE_URL`, `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_WS_URL` e
`NEXT_PUBLIC_WEBHOOK_BASE_URL` para a mesma porta — as três últimas são gravadas no painel durante o
build, então refaça o build (`docker compose up -d --build admin`).
:::

## 4. Confira a API

```bash
curl http://localhost:8081/health
```

```json
{"service":"linktor","status":"ok","timestamp":"2026-11-24T12:00:00Z","version":""}
```

```bash
curl http://localhost:8081/ready
```

`/ready` responde `200` com o estado de cada dependência, ou `503` se alguma estiver fora.

## 5. Entre no painel

No primeiro boot, com o banco vazio, o backend aplica as migrations e cria os dados iniciais:

| O quê | Valor |
|---|---|
| Tenant | `Demo Company` (slug `demo`) |
| Administrador | `admin@demo.com` |
| Atendente | `agent@demo.com` |
| Senha dos dois | O valor de `SEED_ADMIN_PASSWORD` (`admin123` no `.env.example`) |
| Canal | Um WebChat chamado `Website Chat` |
| Exemplo | Um contato e uma conversa com três mensagens |

1. Abra `http://localhost:3000`.
2. Informe `admin@demo.com` em **E-mail** e a senha em **Senha**.
3. Clique em **Entrar**.

Se você apagou `SEED_ADMIN_PASSWORD` do `.env`, a senha é aleatória e aparece **uma única vez** no
log do backend:

```bash
docker compose logs backend | grep "Password (shown once)"
```

O seed só roda quando a tabela `users` está vazia. Para trocar a senha depois, use o próprio painel;
mudar a variável não tem efeito em um banco já populado.

Com o painel aberto, siga [Primeiros passos](../guias/primeiros-passos.md) para conectar um canal.

## Parar, ver logs e recomeçar

```bash
docker compose logs -f backend      # log do backend
docker compose down                 # para tudo, preserva os dados
docker compose down -v              # para tudo e APAGA os volumes (banco, mídia, sessões)
```

O `Makefile` tem atalhos equivalentes: `make docker-up`, `make docker-down`, `make docker-logs` e
`make docker-clean` (que apaga os volumes).

:::warning Atalhos do Makefile
Os alvos `docker-*` chamam o binário `docker-compose` (com hífen). Em máquinas que só têm o plugin
`docker compose`, use os comandos acima diretamente. O alvo `make db-reset` aplica um arquivo
`init.sql` que não existe mais no repositório e falha no último passo; para zerar o banco, use
`docker compose down -v` e suba de novo.
:::

## Receber webhooks na sua máquina

Meta, Telegram, Twilio, Slack e os demais provedores só chamam URLs HTTPS públicas. Para testar um
canal real em ambiente local:

1. Abra um túnel para a porta da API (ngrok, Cloudflare Tunnel ou similar).
2. Coloque a URL do túnel em `LINKTOR_BASE_URL` e em `NEXT_PUBLIC_WEBHOOK_BASE_URL` no `.env`.
3. Recrie os containers: `docker compose up -d --build`.

`LINKTOR_BASE_URL` vira o `BASE_URL` do backend (usado para registrar o webhook do Telegram, montar
as URLs do OAuth da Meta e as URLs de mídia). `NEXT_PUBLIC_WEBHOOK_BASE_URL` é a base das URLs de
webhook exibidas nas telas de canal.

## Modo de desenvolvimento

Para alterar código, rode a infraestrutura no Docker e o backend e o painel direto na máquina.

1. Suba só a infraestrutura:

   ```bash
   docker compose up -d postgres redis nats minio
   ```

2. Inicie o backend na raiz do repositório:

   ```bash
   go run ./cmd/server
   ```

   `make run` faz o mesmo. `make run-dev` sobe o Compose inteiro antes (inclusive os containers de
   backend e painel, que disputam as portas `8081` e `3000` com os processos locais) — prefira os
   dois comandos acima.

   O backend lê o `.env` da raiz e o `config.yaml`. As migrations rodam sozinhas e, fora do modo
   `release`, o seed cria os mesmos dados do passo 5.

3. Em outro terminal, inicie o painel:

   ```bash
   cd web/admin
   cp .env.example .env.local
   npm install
   npm run dev
   ```

   O painel abre em `http://localhost:3000`, apontando para `http://localhost:8081/api/v1`.

Comandos úteis durante o desenvolvimento:

| Comando | O que faz |
|---|---|
| `make build` | Compila o backend em `bin/linktor`. |
| `make test` | Roda os testes do backend (`go test -v ./...`). |
| `make lint` | Roda o `golangci-lint` (instale com `make install-tools`). |
| `make mock-whatsapp-up` | Sobe um mock da Graph API da Meta em `http://localhost:4010`; exporte `LINKTOR_GRAPH_API_URL=http://localhost:4010` para redirecionar as chamadas. |
| `make mock-whatsapp-down` | Para o mock. |
| `make help` | Lista todos os alvos. |

## Solução de problemas

| Sintoma | Causa | O que fazer |
|---|---|---|
| Login responde certo, mas o painel volta para a tela de login | O cookie de sessão exige HTTPS e você está em `http://` | Use `LINKTOR_JWT_COOKIE_SECURE=false` (o `docker-compose.yml` já define assim). |
| Painel mostra erro de rede ao chamar a API | `NEXT_PUBLIC_API_URL` aponta para outra porta ou host | Corrija no `.env` e refaça o build do painel. |
| `linktor-backend` reinicia em laço | Banco inacessível ou migration falhou | `docker compose logs backend` mostra `Failed to connect to database` ou `Failed to run migrations`. |
| Log com `Failed to connect to NATS ... messaging features disabled` | O NATS não estava pronto | Reinicie o backend: `docker compose restart backend`. Sem NATS, nenhuma mensagem é processada. |
| Não aparece `Password (shown once)` no log | O banco já tinha usuários; o seed não rodou | Use a senha definida no primeiro boot, ou zere com `docker compose down -v`. |
