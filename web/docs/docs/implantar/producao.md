---
sidebar_position: 3
title: Implantar em produção
description: Como subir o Linktor em um servidor com o Compose de produção, Traefik e TLS automático, criar o primeiro usuário, expor os webhooks e automatizar o deploy.
---

# Implantar em produção

O diretório `deploy/` do repositório contém uma pilha de produção para **um único servidor**: Docker
Compose, Traefik na frente com certificados Let's Encrypt, e PostgreSQL, Redis, NATS e MinIO no
mesmo host. Este guia leva de um servidor vazio ao painel no ar em HTTPS.

Não há manifests de Kubernetes nem chart Helm no repositório.

:::warning A pilha vem configurada para o domínio do projeto
Os arquivos de `deploy/` foram escritos para a instalação em `linktor.dev`. Antes do primeiro boot
no seu domínio, três pontos precisam de ajuste manual — todos descritos nos passos abaixo:

1. O e-mail do Let's Encrypt está escrito diretamente em `deploy/traefik/traefik.yml`.
2. As imagens apontam para `ghcr.io/integrall-tech/linktor-*`, e a imagem do painel publicada pelo
   CI foi compilada com as URLs de `api.linktor.dev`.
3. Os workflows de `.github/workflows/` chamam `api.linktor.dev` e `deploy.linktor.dev`.
:::

## O que sobe

| Endereço | Serviço | Porta interna | Proteção |
|---|---|---|---|
| `https://app.<domínio>` | Painel (Next.js) | 3000 | Login do Linktor |
| `https://api.<domínio>` | Backend (Go) | 8081 | JWT, chave de API ou assinatura do provedor |
| `https://<domínio>` e `https://www.<domínio>` | Site institucional (`landing`) | 8080 | Público |
| `https://s3.<domínio>` | API S3 do MinIO | 9000 | Chave de acesso |
| `https://s3-console.<domínio>` | Console do MinIO | 9001 | Usuário root do MinIO |
| `https://traefik.<domínio>` | Dashboard do Traefik | — | Basic auth (arquivo htpasswd) |

`<domínio>` é o valor de `ROOT_DOMAIN`. Nos exemplos desta documentação, `LINKTOR_URL` é o endereço
da API — aqui, `https://api.suaempresa.com`.

Duas redes Docker separam o tráfego:

- `linktor-edge`: Traefik e os serviços com entrada externa (backend, painel, landing, MinIO).
- `linktor-internal`: marcada como `internal: true`. PostgreSQL, Redis e NATS só existem nela e não
  publicam nenhuma porta no host.

## Requisitos

| Item | Valor |
|---|---|
| Sistema | Ubuntu 24.04 LTS ou 25.04 (o script de preparo usa `apt` e o repositório Docker para Ubuntu) |
| Máquina inicial | 4 vCPU, 8 GB de RAM, 80 GB de SSD |
| Rede | IPv4 público; portas `80` e `443` abertas |
| DNS | Controle da zona do domínio |
| Arquitetura | `linux/amd64` (as imagens publicadas pelo CI são só para essa plataforma) |

O backend roda em uma única instância (veja [Arquitetura](./arquitetura.md)); dimensione a máquina
para cima em vez de adicionar réplicas.

## 1. Aponte o DNS

Crie os registros **antes** de subir a pilha. O Traefik pede os certificados no primeiro boot, pelo
desafio TLS-ALPN, e cada nome precisa resolver para o servidor.

| Tipo | Nome | Valor |
|---|---|---|
| `A` | `@` | IP do servidor |
| `A` | `www` | IP do servidor |
| `A` | `app` | IP do servidor |
| `A` | `api` | IP do servidor |
| `A` | `traefik` | IP do servidor |
| `A` | `s3` | IP do servidor |
| `A` | `s3-console` | IP do servidor |
| `CAA` (opcional) | `@` | `0 issue "letsencrypt.org"` |

Confira a propagação:

```bash
for sub in "" www. app. api. traefik. s3. s3-console.; do
  echo -n "${sub}suaempresa.com -> "; dig +short "${sub}suaempresa.com" A | head -n1
done
```

## 2. Prepare o servidor

Copie o script e rode como root:

```bash
scp deploy/scripts/install-vps.sh usuario@IP_DO_SERVIDOR:/tmp/
ssh usuario@IP_DO_SERVIDOR sudo bash /tmp/install-vps.sh
```

O script é idempotente e faz o seguinte:

- atualiza os pacotes e ativa `unattended-upgrades`;
- cria 2 GB de swap, se não houver;
- instala Docker Engine e o plugin Compose;
- cria o usuário de serviço `linktor` (no grupo `docker`) e os diretórios `/opt/linktor` e
  `/var/backups/linktor`;
- configura o UFW liberando apenas SSH, `80` e `443`;
- ativa o `fail2ban` para SSH e aplica ajustes de `sysctl`.

Variáveis aceitas: `SSH_PORT` (padrão `22`) e `LINKTOR_USER` (padrão `linktor`).

:::warning
O script executa `ufw --force reset`. Se o servidor já tem regras de firewall próprias, elas são
apagadas.
:::

## 3. Coloque o repositório no servidor

```bash
sudo -u linktor bash
cd /opt/linktor
git clone https://github.com/edsonmartins/linktor.git .
```

Os scripts assumem esta estrutura: repositório em `/opt/linktor`, arquivo de ambiente em
`/opt/linktor/.env` e Compose executado de dentro de `/opt/linktor/deploy`.

## 4. Preencha o `.env`

```bash
cp /opt/linktor/deploy/.env.example /opt/linktor/.env
chmod 600 /opt/linktor/.env
```

| Variável | Como gerar | Observação |
|---|---|---|
| `ROOT_DOMAIN` | — | Ex.: `suaempresa.com`. Define todos os subdomínios. |
| `POSTGRES_USER`, `POSTGRES_DB` | — | `linktor` por padrão. |
| `POSTGRES_PASSWORD` | `openssl rand -base64 48` | |
| `REDIS_PASSWORD` | `openssl rand -base64 48` | O Redis sobe com `requirepass`. |
| `JWT_SECRET` | `openssl rand -base64 64` | Mínimo de 32 caracteres. Trocar invalida todas as sessões. |
| `CRYPTO_ENCRYPTION_KEY` | `openssl rand -base64 48` | Mínimo de 32 caracteres e **diferente** de `JWT_SECRET`. Cifra as credenciais dos canais. |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | `openssl rand -base64 48` para a senha | Mínimo de 8 caracteres na senha. |
| `MINIO_BUCKET` | — | `linktor-media` por padrão. |
| `BACKEND_IMAGE_TAG`, `ADMIN_IMAGE_TAG`, `LANDING_IMAGE_TAG` | — | Tag das imagens. O deploy automatizado reescreve estas linhas. |
| `BACKUP_*` | — | Veja [Backup e atualização](./backup-e-atualizacao.md). |

:::warning `CRYPTO_ENCRYPTION_KEY` não está no `.env.example`
O `deploy/.env.example` não traz a linha `CRYPTO_ENCRYPTION_KEY`, mas o Compose a repassa ao backend
e, em modo `release`, o backend **não inicia** sem ela (`crypto.encryption_key must be changed to a
unique value in release mode`). Adicione a linha manualmente.
:::

O Compose interpola `$` dentro do arquivo de ambiente. Se um segredo contiver `$`, gere outro.

`ACME_EMAIL` existe no `.env.example`, mas **não é lido**: o Traefik não expande variáveis no arquivo
de configuração estática. Troque o e-mail direto no arquivo:

```yaml title="deploy/traefik/traefik.yml"
certificatesResolvers:
  letsencrypt:
    acme:
      email: voce@suaempresa.com
```

## 5. Defina as imagens

O Compose de produção não compila nada: ele baixa `ghcr.io/integrall-tech/linktor-backend`,
`linktor-admin` e `linktor-landing`. Escolha um caminho.

### Usar as imagens publicadas

Autentique o servidor no registro, se as imagens forem privadas para você:

```bash
docker login ghcr.io -u SEU_USUARIO_GITHUB
```

A imagem do painel foi compilada com as URLs da instalação original. Sobreponha em tempo de
execução, adicionando ao serviço `admin` em `deploy/docker-compose.prod.yml`:

```yaml
  admin:
    environment:
      LINKTOR_ADMIN_API_URL: https://api.${ROOT_DOMAIN}/api/v1
      LINKTOR_ADMIN_WS_URL: wss://api.${ROOT_DOMAIN}/api/v1/ws
      LINKTOR_ADMIN_WEBHOOK_BASE_URL: https://api.${ROOT_DOMAIN}
```

As variáveis `NEXT_PUBLIC_*` que já estão nesse bloco não têm efeito em tempo de execução: o Next.js
as grava no bundle durante o build. Só as `LINKTOR_ADMIN_*` são lidas pelo container em execução.

### Compilar as suas

```bash
docker build -t registro.suaempresa.com/linktor-backend:1.0.0 .

docker build -t registro.suaempresa.com/linktor-admin:1.0.0 \
  --build-arg NEXT_PUBLIC_API_URL=https://api.suaempresa.com/api/v1 \
  --build-arg NEXT_PUBLIC_WS_URL=wss://api.suaempresa.com/api/v1/ws \
  --build-arg NEXT_PUBLIC_WEBHOOK_BASE_URL=https://api.suaempresa.com \
  web/admin
```

O build do painel falha de propósito se `NEXT_PUBLIC_API_URL` ou `NEXT_PUBLIC_WS_URL` estiverem
ausentes. Depois, troque as linhas `image:` de `backend` e `admin` no Compose pelo seu registro (só
a tag é variável no arquivo).

O serviço `landing` é o site institucional do projeto. Se você não precisa dele, remova o bloco do
Compose — e os registros `@` e `www` do DNS.

## 6. Crie a senha do dashboard do Traefik

```bash
htpasswd -nbB admin 'UMA-SENHA-FORTE' > /opt/linktor/deploy/traefik/dynamic/dashboard.htpasswd
```

O `htpasswd` vem do pacote `apache2-utils`, instalado pelo script de preparo. O arquivo fica fora do
Git (`deploy/traefik/.gitignore`).

## 7. Primeiro boot e primeiro usuário

Em modo `release` o backend **não cria dados iniciais**, e a API não tem rota para cadastrar tenant.
O único caminho que existe no código é ligar o seed no primeiro boot, com `LINKTOR_SEED_DEMO=true`.
O Compose de produção não repassa essa variável, então use um arquivo de sobreposição temporário:

```yaml title="/opt/linktor/deploy/docker-compose.seed.yml"
services:
  backend:
    environment:
      LINKTOR_SEED_DEMO: "true"
      SEED_ADMIN_PASSWORD: "UMA-SENHA-FORTE-TEMPORARIA"
```

Suba a pilha com os dois arquivos:

```bash
cd /opt/linktor/deploy
docker compose -f docker-compose.prod.yml -f docker-compose.seed.yml --env-file ../.env up -d
docker compose -f docker-compose.prod.yml --env-file ../.env logs -f traefik backend
```

Espere o Traefik emitir os certificados e o backend registrar `Database seeding completed!`. Depois:

1. Abra `https://app.suaempresa.com` e entre com `admin@demo.com` e a senha do arquivo acima.
2. Crie o seu usuário administrador (veja [Usuários e papéis](../painel/usuarios-e-papeis.md)) e
   entre com ele.
3. Remova os usuários `admin@demo.com` e `agent@demo.com`, o canal `Website Chat` e o contato e a
   conversa de exemplo.
4. Troque o **Nome da organização** (o seed cria `Demo Company`) em
   [Configurações](../painel/configuracoes.md).
5. Recrie o backend sem a sobreposição e apague o arquivo:

   ```bash
   docker compose -f docker-compose.prod.yml --env-file ../.env up -d
   rm docker-compose.seed.yml
   ```

O seed só age quando a tabela `users` está vazia, então deixá-lo ligado não recria nada — mas
removê-lo evita surpresas se o banco for trocado. O e-mail de um usuário não pode ser alterado pela
API; por isso o passo 2 cria um usuário novo em vez de editar o de demonstração. O slug do tenant
permanece `demo`.

Confira os endereços:

```bash
curl https://api.suaempresa.com/health
curl https://api.suaempresa.com/ready
```

## 8. Agende o backup

```bash
sudo touch /var/log/linktor-backup.log && sudo chown linktor:linktor /var/log/linktor-backup.log
sudo -u linktor crontab -e
```

```text
0 3 * * * /opt/linktor/deploy/scripts/backup.sh >> /var/log/linktor-backup.log 2>&1
```

Detalhes e restauração em [Backup e atualização](./backup-e-atualizacao.md).

## Variáveis que o Compose de produção não repassa

O bloco `environment` do backend em `deploy/docker-compose.prod.yml` define só o essencial. Estes
recursos ficam **desligados** até você acrescentar a variável ao bloco (a referência completa está
em [Configuração](./configuracao.md)):

| Para ter | Acrescente ao backend |
|---|---|
| URLs de mídia assinadas e com validade | `MEDIA_SIGNING_SECRET` (e, se quiser, `MEDIA_URL_TTL`, `MEDIA_UNSIGNED_UNTIL`) |
| Bots com IA | `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` ou `OLLAMA_BASE_URL` |
| WhatsApp Embedded Signup | `WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET` |
| Versão em `/health` | `LINKTOR_VERSION` |
| Checagem do Redis em `/ready` | `REDIS_URL` (ex.: `redis://:${REDIS_PASSWORD}@redis:6379`) |
| Restringir os tipos de canal | `LINKTOR_ENABLED_CHANNEL_TYPES` |

Depois de editar, aplique com `docker compose -f docker-compose.prod.yml --env-file ../.env up -d`.

## Expor os webhooks

Os provedores (Meta, Telegram, Twilio, Slack, Teams...) chamam o Linktor pela internet e **exigem
HTTPS com certificado válido**. Com esta pilha, o endereço é:

```text
https://api.suaempresa.com/api/v1/webhooks/<tipo>/CHANNEL_ID
```

Três valores precisam concordar com esse endereço público:

| Onde | Variável | Para quê |
|---|---|---|
| Backend | `BASE_URL` (o Compose define `https://api.${ROOT_DOMAIN}`) | Registro automático do webhook do Telegram, URLs de retorno do OAuth da Meta, URLs de mídia |
| Painel | `NEXT_PUBLIC_WEBHOOK_BASE_URL` no build, ou `LINKTOR_ADMIN_WEBHOOK_BASE_URL` em execução | URL de webhook exibida nas telas de canal e snippet do WebChat |
| Backend | `LINKTOR_CORS_ALLOWED_ORIGINS` e `LINKTOR_WS_ALLOWED_ORIGINS` | Origem do painel (`https://app.<domínio>`); o Compose já preenche |

Pontos de atenção:

- Em `release`, webhook de canal sem segredo configurado é **recusado**. Configure o segredo de cada
  canal antes de apontar o provedor (veja a página do canal em [Canais](../canais/visao-geral.md)).
- O roteador da API passa pelo middleware `rate-limit` do Traefik (média de 100 requisições por
  segundo por IP, rajada de 200). Além disso, o backend limita as rotas `/api/v1/webhooks` a **100
  requisições por minuto por IP de origem** — o limite é fixo no código. Provedores de alto volume,
  que concentram as chamadas em poucos IPs, podem receber `429` e reenviar depois.
- O WebChat usa WebSocket em `wss://api.<domínio>/ws/CHANNEL_ID`, e o widget é servido em
  `https://api.<domínio>/widget/`. Ambos passam pelo mesmo roteador.
- Nunca publique a porta `8081` direto na internet: o backend confia no cabeçalho
  `X-Forwarded-For` para identificar o cliente, e é o Traefik que o torna confiável.

## Deploy contínuo com GitHub Actions

Há dois workflows de entrega no repositório.

### `build-and-push.yml`

Dispara em push para `main`, em tags `v*.*.*` e manualmente.

1. Executa o CI (`ci.yml`): `go vet`, build, testes com `-race`, migrations contra um PostgreSQL
   real e testes E2E do painel.
2. Compila e publica `linktor-backend`, `linktor-admin` e `linktor-landing` em
   `ghcr.io/integrall-tech`, para `linux/amd64`. Em push para `main`, as tags são `latest` e
   `sha-<7 primeiros caracteres do commit>`; em tag de versão, a própria tag e `latest`.
3. Chama um receptor de webhook em `https://deploy.linktor.dev/hooks/deploy`, com o cabeçalho
   `X-Deploy-Token`, pedindo a implantação da tag publicada.
4. Aguarda até 7 minutos `GET /health` reportar o campo `version` igual à tag.

O receptor do passo 3 **não está neste repositório**: ele faz parte de um repositório operacional
separado (`linktor-devops`). O que dá para afirmar daqui é o contrato da chamada — um `POST` com
`{"env":"production","component":"all","tag":"..."}`.

Secrets usados: `GHCR_ORG_TOKEN` (PAT com `write:packages`), `GHCR_ORG_USERNAME` (opcional) e
`DEPLOY_TOKEN`.

### `deploy.yml`

Dispara manualmente (com a tag da imagem como entrada) ou chamado por outro workflow. É o caminho
que combina com a pilha de `deploy/`:

1. Conecta por SSH no servidor.
2. Em `/opt/linktor`, executa `git fetch` e `git reset --hard origin/main` — alterações locais em
   arquivos versionados (inclusive edições no Compose e no `traefik.yml`) **são descartadas**.
3. Reescreve `BACKEND_IMAGE_TAG`, `ADMIN_IMAGE_TAG` e `LANDING_IMAGE_TAG` no `.env`.
4. Executa `pull`, `up -d --remove-orphans` e `image prune`.
5. Testa `https://api.linktor.dev/health`.

Secrets do ambiente `production`: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY` e `DEPLOY_PORT`
(opcional).

```bash
gh workflow run deploy.yml -f image_tag=sha-abc1234
```

### Para usar no seu fork

Os dois workflows têm o domínio `linktor.dev` e a organização `integrall-tech` fixos. Em um fork,
ajuste: `env.ORG` e os `build_args` do painel em `build-and-push.yml`; a URL do receptor de deploy
(ou troque o job `deploy` por uma chamada a `deploy.yml`); e as URLs do teste de fumaça nos dois
arquivos. Como o `deploy.yml` faz `git reset --hard`, mantenha as suas alterações de `deploy/`
commitadas no fork, não editadas à mão no servidor.

### Deploy manual no servidor

```bash
sudo -iu linktor
cd /opt/linktor/deploy
# edite as linhas *_IMAGE_TAG em ../.env
./scripts/deploy.sh
```

O script faz `pull`, `up -d --remove-orphans`, remove imagens órfãs e mostra o estado dos containers.

## Checklist de go-live

**Modo e segredos**

- [ ] `LINKTOR_SERVER_MODE=release` (o Compose de produção já define).
- [ ] `JWT_SECRET` e `CRYPTO_ENCRYPTION_KEY` fortes, com 32 caracteres ou mais e diferentes entre si.
- [ ] `CRYPTO_ENCRYPTION_KEY` guardada fora do servidor, junto com a política de backup.
- [ ] `POSTGRES_PASSWORD`, `REDIS_PASSWORD` e `MINIO_ROOT_PASSWORD` fortes e únicos.
- [ ] `/opt/linktor/.env` com modo `600`, dono `linktor`.
- [ ] Usuários de demonstração removidos; `LINKTOR_SEED_DEMO` ausente.

**Rede**

- [ ] Todos os subdomínios resolvem e têm certificado válido.
- [ ] Só `22`, `80` e `443` abertas no firewall.
- [ ] `LINKTOR_CORS_ALLOWED_ORIGINS` e `LINKTOR_WS_ALLOWED_ORIGINS` contêm a origem exata do painel.
- [ ] `/metrics` e `/swagger` restritos (veja [Segurança](./seguranca.md)).
- [ ] Console do MinIO e dashboard do Traefik com senha forte, ou sem rota pública.

**Persistência**

- [ ] Volumes `backend_storages` e `backend_uploads` montados (o Compose já monta).
- [ ] Backup agendado e **restauração testada** em outra máquina.

**Funcionamento**

- [ ] `GET /health` e `GET /ready` respondem `200`.
- [ ] Login com usuário real.
- [ ] Um canal de teste envia e recebe de ponta a ponta.
- [ ] Um webhook com assinatura inválida recebe `401`.
- [ ] Reiniciar o backend não exige parear o WhatsApp não-oficial de novo.

**Observação**

- [ ] Monitor externo em `/health` e coleta de `/metrics` configurados (veja
  [Monitoramento](./monitoramento.md)).

## Solução de problemas

| Sintoma | Causa | O que fazer |
|---|---|---|
| Backend não inicia, log com `jwt.secret must be...` ou `crypto.encryption_key must be...` | Segredo ausente, fraco, padrão, ou os dois iguais | Corrija o `.env` e recrie o backend. |
| Traefik não emite certificado | DNS não propagou, porta `443` fechada ou e-mail inválido em `traefik.yml` | `docker compose ... logs traefik`; corrija e reinicie o Traefik. |
| `404` em todos os endereços | O container de destino está `unhealthy` e o Traefik o tirou do roteamento | `docker compose ... ps` e o log do serviço. |
| Painel abre, login falha com erro de rede | Painel compilado com a URL de outra instalação, ou CORS | Defina `LINKTOR_ADMIN_API_URL`/`LINKTOR_ADMIN_WS_URL`; confira `LINKTOR_CORS_ALLOWED_ORIGINS`. |
| Login funciona, mas o tempo real não | Origem ausente em `LINKTOR_WS_ALLOWED_ORIGINS` | Inclua `https://app.<domínio>`. |
| Tela de login sem nenhum usuário válido | Instalação nova em `release`, sem seed | Siga o passo 7. |
| Dashboard do Traefik responde erro | `dashboard.htpasswd` não existe | Siga o passo 6. |
