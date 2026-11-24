---
sidebar_position: 6
title: Backup e atualização
description: Como fazer backup e restaurar, atualizar a versão, entender as migrations, voltar atrás e trocar a chave de criptografia das credenciais.
---

# Backup e atualização

Os procedimentos desta página usam a pilha de `deploy/` descrita em [Produção](./producao.md):
repositório em `/opt/linktor`, variáveis em `/opt/linktor/.env`, Compose executado a partir de
`/opt/linktor/deploy`.

```bash
sudo -iu linktor
cd /opt/linktor/deploy
COMPOSE="docker compose -f docker-compose.prod.yml --env-file ../.env"
```

Os exemplos abaixo usam a variável `COMPOSE` definida acima.

## O que precisa de backup

| Dado | Onde está | Coberto por `backup.sh` |
|---|---|---|
| Banco (tenants, usuários, canais, conversas, mensagens) | Volume `linktor_postgres_data` | Sim (`pg_dump`) |
| Mídia | Volume `linktor_minio_data` | Sim (espelho do bucket) |
| Sessões do WhatsApp não-oficial | Volume `linktor_backend_storages` | Sim |
| Imagens geradas e uploads locais | Volume `linktor_backend_uploads` | Sim |
| Compose e configuração do Traefik | `deploy/docker-compose.prod.yml`, `deploy/traefik/` | Sim |
| **Arquivo `.env`** (inclui `CRYPTO_ENCRYPTION_KEY`) | `/opt/linktor/.env` | **Não** |
| Certificados TLS | Volume `linktor_traefik_acme` | Não (o Traefik emite de novo) |
| Filas do NATS | Volume `linktor_nats_data` | Não |
| Redis | Volume `linktor_redis_data` | Não |

:::danger Guarde o `.env` separadamente
As credenciais dos canais são gravadas cifradas com `CRYPTO_ENCRYPTION_KEY`. Um backup do banco
restaurado sem essa chave deixa **todos os canais inutilizáveis**: será preciso recadastrar cada
token. Mantenha uma cópia do `.env` em um cofre de senhas, fora do servidor e fora do arquivo de
backup.
:::

O que fica de fora e a consequência ao restaurar:

- **NATS**: mensagens que estavam na fila no momento do desastre são perdidas. O que já estava
  gravado no banco continua lá.
- **Redis**: perdem-se os contadores de rate limit, as chaves de deduplicação de webhook (um reenvio
  do provedor na janela de 6 horas pode ser processado de novo; a deduplicação por mensagem no banco
  ainda vale) e a **lista de revogação de mídia** — URLs revogadas por `POST /api/v1/media/revoke`
  voltam a funcionar. Se você usa revogação, inclua o volume do Redis no seu backup.

## Fazer backup

```bash
sudo -u linktor /opt/linktor/deploy/scripts/backup.sh
ls -lh /var/backups/linktor/
```

O script gera um arquivo `linktor-<data UTC>.tar.gz` com:

| Arquivo | Conteúdo |
|---|---|
| `postgres.dump` | `pg_dump` em formato custom, compressão 9 |
| `minio/` | Espelho de todos os buckets, feito com `mc mirror` |
| `backend-data.tar.gz` | Diretórios `uploads` e `storages` do backend |
| `config.tar.gz` | `docker-compose.prod.yml` e a pasta `traefik/` |
| `SHA256SUMS` | Somas de verificação dos itens acima |

Depois ele apaga os arquivos com mais de `BACKUP_RETENTION_DAYS` dias.

| Variável (no `.env`) | Padrão | Efeito |
|---|---|---|
| `BACKUP_DIR` | `/var/backups/linktor` | Onde os arquivos são gravados. |
| `BACKUP_RETENTION_DAYS` | `14` | Dias de retenção local. |
| `BACKUP_S3_ENDPOINT` | vazio | Endereço de um S3 externo, só esquema e host (`https://s3.exemplo.com`). Vazio desliga o envio externo. |
| `BACKUP_S3_BUCKET` | vazio | Bucket de destino. |
| `BACKUP_S3_ACCESS_KEY`, `BACKUP_S3_SECRET_KEY` | vazio | Credenciais do destino. Sem as duas, o envio externo é pulado com aviso no log. |
| `ENV_FILE` (do ambiente) | `/opt/linktor/.env` | Caminho alternativo do arquivo de variáveis. |

Agende a execução diária:

```text
0 3 * * * /opt/linktor/deploy/scripts/backup.sh >> /var/log/linktor-backup.log 2>&1
```

Pontos de atenção:

- O script localiza os recursos pelos nomes fixos do Compose de produção: container
  `linktor-postgres`, rede `linktor_linktor-internal` e volumes `linktor_backend_*`. Ele não serve
  para a pilha local de desenvolvimento.
- As etapas do MinIO e dos diretórios do backend **não interrompem** o script se falharem. Um backup
  pode terminar com `Backup complete` e sem a mídia. Leia o log e confira o conteúdo do arquivo.
- Um backup apenas local some junto com o servidor. Configure `BACKUP_S3_*` ou copie os arquivos
  para fora.
- O banco é exportado com o sistema no ar; a mídia é copiada logo depois. Uma mensagem com anexo
  recebida entre as duas etapas pode ficar com o anexo no backup sem o registro, ou o contrário.

## Restaurar

```bash
sudo -u linktor /opt/linktor/deploy/scripts/restore.sh /var/backups/linktor/linktor-20261124T030000Z.tar.gz
```

O script:

1. extrai o arquivo e confere o `SHA256SUMS`;
2. pede confirmação — digite `yes`;
3. para o backend;
4. restaura o banco com `pg_restore --clean --if-exists --no-owner` (os objetos existentes são
   apagados e recriados);
5. copia os objetos de volta ao MinIO, sobrescrevendo os de mesmo nome;
6. inicia o backend.

:::warning As sessões do WhatsApp e os uploads não são restaurados pelo script
`restore.sh` não toca em `backend-data.tar.gz` nem em `config.tar.gz`. Sem restaurar o primeiro, os
canais de WhatsApp não-oficial precisam ser pareados de novo.
:::

Para restaurar os diretórios do backend, com o backend parado:

```bash
ARQUIVO=/var/backups/linktor/linktor-20261124T030000Z.tar.gz
mkdir -p /tmp/linktor-restore && tar -xzf "$ARQUIVO" -C /tmp/linktor-restore
SNAP=$(find /tmp/linktor-restore -mindepth 1 -maxdepth 1 -type d | head -n1)

$COMPOSE stop backend
docker run --rm \
  -v linktor_backend_uploads:/dst/uploads \
  -v linktor_backend_storages:/dst/storages \
  -v "$SNAP:/backup:ro" \
  alpine:3.22 \
  tar -xzf /backup/backend-data.tar.gz -C /dst
$COMPOSE start backend
rm -rf /tmp/linktor-restore
```

### Restaurar em um servidor novo

1. Prepare o servidor e o DNS como em [Produção](./producao.md), passos 1 a 6.
2. Coloque em `/opt/linktor/.env` **o mesmo** `.env` do servidor antigo — no mínimo com o mesmo
   `CRYPTO_ENCRYPTION_KEY`, `POSTGRES_USER` e `POSTGRES_DB`.
3. Suba a pilha (`$COMPOSE up -d`), sem o passo de seed. O backend cria o esquema vazio.
4. Copie o arquivo de backup para o servidor e execute `restore.sh`.
5. Restaure os diretórios do backend com o bloco acima.
6. Confira `GET /ready`, entre no painel e verifique o estado dos canais.

Um backup que nunca foi restaurado não é garantia de nada. Repita este roteiro em uma máquina
descartável antes do go-live e depois periodicamente.

## Atualizar a versão

Cada versão é uma tag das imagens `linktor-backend`, `linktor-admin` e `linktor-landing`. Atualizar
é trocar as tags e recriar os containers.

1. Leia o `CHANGELOG.md` do repositório.
2. Faça um backup:

   ```bash
   /opt/linktor/deploy/scripts/backup.sh
   ```

3. Atualize os arquivos de `deploy/` e as tags:

   ```bash
   cd /opt/linktor && git pull
   # edite BACKEND_IMAGE_TAG, ADMIN_IMAGE_TAG e LANDING_IMAGE_TAG em /opt/linktor/.env
   ```

4. Aplique:

   ```bash
   /opt/linktor/deploy/scripts/deploy.sh
   ```

5. Confira:

   ```bash
   curl https://api.suaempresa.com/ready
   $COMPOSE logs --tail=100 backend
   ```

O backend roda em uma instância, então a troca do container causa alguns segundos de
indisponibilidade. Os webhooks recebidos nesse intervalo são reenviados pelos provedores. Ao
encerrar, o backend espera até `server.shutdown_timeout` segundos (padrão 30) pelas requisições em
andamento.

Se você usa as próprias imagens, reconstrua-as a partir do código atualizado antes do passo 4. O
caminho automatizado, pelo workflow `deploy.yml`, está em [Produção](./producao.md).

No ambiente local de [Instalação](./instalacao.md):

```bash
git pull
docker compose up -d --build
```

### Atualizar os serviços de infraestrutura

Para Traefik, Redis e NATS, troque a tag da imagem em `deploy/docker-compose.prod.yml` e aplique
com `$COMPOSE up -d`.

Para o **PostgreSQL, não basta trocar a tag em uma mudança de versão principal** (por exemplo, 16
para 17): o diretório de dados é incompatível. Faça `pg_dump`, troque a imagem, suba com volume
vazio e restaure. Mantenha uma imagem com `pgvector`.

## Migrations

As migrations são aplicadas **automaticamente** toda vez que o backend inicia, antes de abrir a
porta HTTP. Não há comando separado a executar em um deploy normal.

- A ferramenta é o [goose](https://github.com/pressly/goose), embutido no binário.
- A versão `1` é a linha de base: cria o esquema inicial com comandos idempotentes
  (`IF NOT EXISTS`), de modo que aplicá-la sobre um banco já existente não altera nada.
- As versões seguintes são arquivos SQL numerados em
  `internal/infrastructure/database/migrations/` (`00002_...sql` em diante), também embutidos no
  binário.
- O controle fica na tabela `goose_db_version`.

Ver a versão aplicada:

```bash
$COMPOSE exec postgres psql -U linktor -d linktor \
  -c "SELECT version_id, tstamp FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 5;"
```

Se uma migration falhar, o backend registra `Failed to run migrations: ...` e encerra; o Compose o
reinicia em laço. A versão antiga já foi substituída nesse ponto — por isso o backup do passo 2.

## Voltar atrás

Reimplante a tag anterior:

```bash
# edite as linhas *_IMAGE_TAG em /opt/linktor/.env para a tag anterior
/opt/linktor/deploy/scripts/deploy.sh
```

Ou, com o workflow: `gh workflow run deploy.yml -f image_tag=sha-<commit anterior>`.

**Voltar a imagem não desfaz as migrations.** O backend só aplica migrations para frente. Na
prática:

- A maioria das migrations é aditiva (coluna nova, índice novo), e o binário anterior continua
  funcionando com o esquema mais novo.
- Se a versão que você está abandonando aplicou uma migration destrutiva ou incompatível, a volta
  segura é **restaurar o backup** feito antes da atualização — com perda do que entrou depois dele.

Cada arquivo SQL traz uma seção `-- +goose Down`, então é possível reverter uma migration à mão com
o binário `goose` apontado para o diretório de migrations. Esse caminho não é automatizado nem
exercitado pelos scripts do repositório: teste em uma cópia do banco antes, e nunca reverta a versão
`1` — o `Down` da linha de base apaga todas as tabelas.

## Trocar a chave de criptografia das credenciais {#trocar-chave}

`CRYPTO_ENCRYPTION_KEY` cifra as credenciais dos canais. A troca é feita em três tempos, sem
indisponibilidade dos canais: a chave nova passa a cifrar, a antiga continua aceita para decifrar,
e depois de recifrar tudo a antiga é removida.

A lista de chaves antigas (`crypto.previous_keys`) **só é lida de um arquivo `config.yaml`** — a
variável `LINKTOR_CRYPTO_PREVIOUS_KEYS` não tem efeito na imagem Docker. Por isso o procedimento
monta um arquivo no container.

1. Gere a chave nova:

   ```bash
   openssl rand -base64 48
   ```

2. Crie um `config.yaml` com a chave **atual** (a que será substituída) na lista:

   ```yaml title="/opt/linktor/linktor-config.yaml"
   crypto:
     previous_keys:
       - "CHAVE-ATUAL-QUE-VAI-SER-SUBSTITUIDA"
   ```

   O container roda com o usuário de uid `1000`, que precisa conseguir ler o arquivo — se não
   conseguir, o backend não inicia (`error reading config file`):

   ```bash
   sudo chown 1000:1000 /opt/linktor/linktor-config.yaml
   sudo chmod 400 /opt/linktor/linktor-config.yaml
   ```

3. Monte o arquivo no backend, acrescentando ao serviço em `deploy/docker-compose.prod.yml`:

   ```yaml
     backend:
       volumes:
         - ../linktor-config.yaml:/etc/linktor/config.yaml:ro
   ```

   As demais configurações continuam vindo das variáveis de ambiente, que têm precedência sobre o
   arquivo.

4. Troque `CRYPTO_ENCRYPTION_KEY` no `.env` pela chave nova e recrie o backend:

   ```bash
   $COMPOSE up -d backend
   ```

   A partir daqui o backend lê credenciais cifradas com qualquer uma das duas chaves e grava com a
   nova.

5. Recifre as credenciais. A rota age sobre o tenant de quem chama e exige um **usuário** com papel
   `admin` ou `owner` (chave de API recebe `403`):

   ```bash
   TOKEN=$(curl -s -X POST "https://api.suaempresa.com/api/v1/auth/login" \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@suaempresa.com","password":"SUA_SENHA"}' | jq -r '.data.access_token')

   curl -X POST "https://api.suaempresa.com/api/v1/channels/reencrypt" \
     -H "Authorization: Bearer $TOKEN"
   ```

   ```json
   {"success": true, "data": {"reencrypted": 7}}
   ```

   Repita com um administrador de **cada tenant** da instalação.

6. Só depois de recifrar todos os canais de todos os tenants, esvazie a lista `previous_keys` (ou
   remova o volume) e recrie o backend.

:::warning A rota recifra no máximo 20 canais por tenant
`POST /channels/reencrypt` carrega os canais do tenant com a paginação padrão: os **20 mais
recentes**. Chamá-la de novo processa os mesmos 20. Se `reencrypted` vier `20` e o tenant tiver mais
canais, os restantes continuam cifrados com a chave antiga. Nesse caso, salve cada canal restante
uma vez (editar e salvar o canal regrava as credenciais com a chave nova) ou mantenha a chave antiga
em `previous_keys`. Remover a chave antiga com canais ainda cifrados por ela os torna ilegíveis.
:::

Credenciais gravadas antes de a criptografia existir (sem o prefixo `enc:v1:`) continuam legíveis e
são cifradas na próxima vez em que o canal é salvo — a mesma rota serve para cifrá-las de uma vez.

## Trocar os outros segredos

| Segredo | Procedimento | Efeito |
|---|---|---|
| `JWT_SECRET` | Edite o `.env` e execute `$COMPOSE up -d backend`. | Todas as sessões são encerradas; os usuários entram de novo. Chaves de API não são afetadas. |
| Senha do PostgreSQL | `$COMPOSE exec postgres psql -U linktor -c "ALTER USER linktor WITH PASSWORD 'NOVA';"`, atualize `POSTGRES_PASSWORD` no `.env`, depois `$COMPOSE up -d --force-recreate backend`. | Breve indisponibilidade do backend. |
| Senha do Redis | Atualize `REDIS_PASSWORD` no `.env` e execute `$COMPOSE up -d redis backend`. | O Redis reinicia com a senha nova. |
| `MEDIA_SIGNING_SECRET` | Edite e recrie o backend. | Todas as URLs de mídia assinadas antes deixam de valer; quem as guarda precisa pedir novas. |
| Senha do dashboard do Traefik | Gere novo `dashboard.htpasswd` e execute `$COMPOSE restart traefik`. | Segundos sem roteamento. |
