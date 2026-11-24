---
sidebar_position: 7
title: Segurança
description: Como o Linktor isola tenants, autentica usuários e integrações, protege credenciais, valida webhooks e mídia, o que o deploy de referência endurece e o checklist de quem opera.
---

# Segurança

Esta página explica os mecanismos de segurança que existem no código e o que fica por conta de quem
opera. O [checklist](#checklist) no final resume as ações.

## Modelo em uma frase

Toda requisição é atribuída a **um tenant** pela credencial que a acompanha — nunca por um parâmetro
— e tudo o que entra de fora (webhook de provedor, URL de mídia, origem de navegador) é validado
antes de ser processado. O modo `release` transforma várias dessas validações de opcionais em
obrigatórias.

## Isolamento por tenant

Todos os tenants compartilham o mesmo banco, com uma coluna `tenant_id` em cada tabela de negócio. A
separação é feita pela aplicação:

- O middleware de autenticação extrai o tenant do JWT ou da chave de API e o coloca no contexto da
  requisição. Os handlers leem o tenant **só** desse contexto.
- Nenhuma rota aceita `tenant_id` no corpo, na query ou na URL.
- Em webhooks de provedores, o tenant é o dono do canal indicado em `:channelId`.
- No proxy de mídia, uma sessão ou chave de API só lê objetos cujo canal pertence ao seu tenant.

Limites que quem opera uma instalação com vários tenants precisa conhecer:

- **As filas são da instalação, não do tenant.** As rotas de **Observabilidade → Filas de
  Mensagens** (`GET /api/v1/observability/queue` e `POST /api/v1/observability/queue/reset-consumer`)
  mostram e reiniciam consumidores compartilhados. Um administrador de qualquer tenant enxerga o
  volume global das filas e pode reiniciar um consumidor que atende a todos.
- **O rate limit de API é por tenant** (veja [abaixo](#rate-limiting)): um integrador barulhento
  consome a cota dos atendentes do mesmo tenant, mas não a de outros tenants.
- Não há separação por schema ou por banco. Para isolamento físico, use instalações separadas.

## Autenticação

O backend aceita três credenciais, tentadas nesta ordem:

| Credencial | Como é enviada | Para quem |
|---|---|---|
| JWT de acesso | Cabeçalho `Authorization: Bearer <token>` | Scripts e CLI que fizeram login |
| JWT de acesso | Cookie `access_token` | Navegador (painel) |
| Chave de API | Cabeçalho `X-API-Key: lk_...` | Integrações servidor a servidor |

### Sessão do painel: cookies HttpOnly

O login (`POST /api/v1/auth/login`) grava dois cookies: `access_token` (15 minutos por padrão) e
`refresh_token` (7 dias). Ambos são `HttpOnly` — o JavaScript da página não consegue lê-los — com
`Path=/`, e os atributos `Secure`, `SameSite` e `Domain` vêm da configuração `jwt.cookie_*`.

| Configuração | Padrão | Recomendação |
|---|---|---|
| `LINKTOR_JWT_COOKIE_SECURE` | `true` | Mantenha. `false` só em HTTP local. |
| `LINKTOR_JWT_COOKIE_SAME_SITE` | `lax` | Mantenha quando painel e API são subdomínios do mesmo domínio (`app.` e `api.`). |
| `LINKTOR_JWT_COOKIE_DOMAIN` | vazio | Vazio restringe o cookie ao host da API. |

A proteção contra requisições forjadas de outros sites combina `SameSite=Lax` com a checagem de
origem do CORS: qualquer requisição que traga um cabeçalho `Origin` fora da lista recebe `403` antes
de chegar ao handler.

Os tokens também vêm no corpo da resposta de login, para clientes que não são navegador. Trocar
`LINKTOR_JWT_SECRET` invalida todos os tokens emitidos.

### Chaves de API e scopes

- A chave é gerada com 24 bytes aleatórios, no formato `lk_` seguido de 48 caracteres hexadecimais. O banco guarda
  apenas o **hash bcrypt** e um prefixo para localização; o valor completo aparece uma vez, na
  criação.
- Cada chave pertence a um tenant e carrega uma lista de **scopes** no formato `recurso:ação`
  (`channels:read`, `messages:send`, `contacts:write`...). `*` libera tudo; `recurso:*` libera
  todas as ações do recurso; `recurso:write` inclui `recurso:read`.
- Uma chave de API recebe o papel interno `api`. Rotas restritas aos papéis `admin` e `owner` —
  gestão de usuários, **Observabilidade**, `POST /channels/reencrypt`, alteração do tenant —
  **recusam chaves de API** com `403`, qualquer que seja o scope.
- Scopes não se aplicam a usuários humanos; para eles valem os papéis.

Detalhes de uso em [Autenticação da API](../api/autenticacao.md) e
[Chaves de API](../painel/chaves-de-api.md).

## Credenciais de canal cifradas

Tokens de provedor, senhas SMTP e segredos de webhook são gravados cifrados:

- Algoritmo AES-256-GCM, com a chave derivada por SHA-256 de `LINKTOR_CRYPTO_ENCRYPTION_KEY`.
- Cada valor é gravado como `enc:v1:<base64>`, com nonce aleatório próprio.
- O mapa `credentials` do canal é cifrado por inteiro. Do mapa `config`, só as chaves sensíveis
  (`access_token`, `bot_token`, `app_secret`, `api_key`, `auth_token`, `webhook_secret`,
  `widget_secret` e similares); o restante fica em claro para poder ser consultado.
- Nas respostas da API, `credentials` nunca é devolvido e os valores sensíveis de `config` aparecem
  mascarados.

Em `release`, o backend não inicia com a chave ausente, com um valor de exemplo, com menos de 32
caracteres ou igual ao segredo JWT.

O que isso protege: um vazamento do banco ou de um backup do banco não expõe os tokens dos
provedores. O que não protege: quem tem o banco **e** o `.env` tem tudo. Guarde os dois em lugares
diferentes. A troca de chave está em
[Backup e atualização](./backup-e-atualizacao.md#trocar-chave).

:::warning Sessões do WhatsApp não-oficial não são cifradas
Os arquivos em `/app/storages` (volume `backend_storages`) contêm as chaves das sessões pareadas.
Quem os copia consegue operar o número. Restrinja o acesso ao volume e aos arquivos de backup.
:::

## Webhooks recebidos dos provedores

As rotas `/api/v1/webhooks/*` são públicas por natureza. A autenticação é a prova de origem que cada
provedor oferece:

| Canal | Como é validado |
|---|---|
| WhatsApp oficial | HMAC-SHA256 do corpo no cabeçalho `X-Hub-Signature-256`; na verificação inicial (`GET`), o `hub.verify_token` do canal |
| Facebook e Instagram | Assinatura da Meta com o App Secret do canal; `hub.verify_token` na verificação inicial |
| Telegram | Cabeçalho `X-Telegram-Bot-Api-Secret-Token` igual ao segredo gerado ao conectar o canal |
| SMS e voz (Twilio) | Assinatura do Twilio calculada com o Auth Token |
| Slack | `X-Slack-Signature` com `X-Slack-Request-Timestamp`, usando o Signing Secret |
| Teams | JWT do Bot Connector, com a audiência igual ao App ID do canal |
| E-mail | Assinatura HMAC do Mailgun; nos demais provedores, segredo compartilhado |
| RCS | Assinatura do provedor (`X-Signature` ou `X-Hub-Signature-256`) |
| Direto e genérico | HMAC-SHA256 do corpo com o `webhook_secret` do canal, em `X-Linktor-Signature` ou `X-Hub-Signature-256` |

O comportamento quando o canal **não tem segredo configurado** depende do modo:

| Modo | Sem segredo no canal |
|---|---|
| `debug` | O webhook é aceito sem validação. |
| `release`, ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true` | O webhook é recusado com `401`. |

Assinatura inválida é sempre `401`, em qualquer modo. A configuração do segredo de cada canal está
na página do canal, em [Canais](../canais/visao-geral.md).

### Deduplicação

Provedores reenviam o mesmo webhook quando a resposta demora ou falha. O backend descarta repetições
em duas camadas:

1. **No recebimento (Redis).** Para cada `POST` em `/api/v1/webhooks/*`, calcula-se um identificador
   — o primeiro presente entre `X-Hub-Signature-256`, `X-Webhook-Id`, `X-Event-Id`,
   `X-Twilio-Idempotency-Token` e `Idempotency-Key`, ou o SHA-256 do corpo — combinado com o caminho
   da requisição. Uma repetição em até **6 horas** recebe `200` com `{"status":"duplicate"}` e o
   cabeçalho `X-Webhook-Dedup: duplicate`, sem ser processada. Se o processamento original responder
   `4xx` ou `5xx`, a marca é removida para que a próxima tentativa do provedor seja processada.
2. **No processamento (banco).** Uma mensagem com o mesmo identificador externo no mesmo canal é
   reconhecida como duplicata e contada em `linktor_inbound_messages_total{result="duplicate"}`.

A primeira camada depende do Redis: sem ele, só a segunda atua.

## Webhook de saída

As entregas para a sua aplicação levam `X-Linktor-Signature` (HMAC-SHA256) e `X-Linktor-Timestamp`,
calculados com o segredo configurado no canal. A verificação de TLS do destino é sempre feita; a
única exceção é `LINKTOR_WEBHOOK_INSECURE_LOOPBACK=true`, restrita a destinos em loopback. Como
validar a assinatura do seu lado: [Receber eventos](../api/receber-eventos.md).

## URLs de mídia

A mídia é servida pelo proxy `GET /api/v1/media/<chave>`. A URL guardada na mensagem é uma
referência permanente; o que autoriza a leitura é uma destas três coisas:

| Autorização | Quando vale |
|---|---|
| Assinatura na URL (`?exp=<unix>&sig=<hmac>`) | Com `MEDIA_SIGNING_SECRET` definido. A assinatura é adicionada quando a URL é entregue a um leitor (resposta da API, envio a um provedor, `POST /api/v1/media/sign`) e vale por `MEDIA_URL_TTL` (7 dias por padrão). Expirada, a resposta é `410`. |
| Sessão ou chave de API do tenant dono do objeto | Sempre. É como o painel carrega imagens. |
| Nenhuma (URL sem assinatura) | Sem `MEDIA_SIGNING_SECRET`: **sempre**. Com ele: só até a data de `MEDIA_UNSIGNED_UNTIL`, ou indefinidamente se a variável estiver vazia. |

Consequência: sem `MEDIA_SIGNING_SECRET`, qualquer pessoa com a URL lê o arquivo para sempre. As
chaves são longas e não enumeráveis, mas uma URL que vaza continua valendo. Em produção, defina o
segredo e, depois de migrar os consumidores, uma data em `MEDIA_UNSIGNED_UNTIL`.

Para invalidar um objeto ou um prefixo inteiro (por exemplo, toda a mídia de um canal),
`POST /api/v1/media/revoke` registra a chave em uma lista de bloqueio no Redis, sem prazo de
validade, que prevalece sobre qualquer assinatura. Detalhes em [Mídia](../api/midia.md).

No sentido contrário — quando o WhatsApp não-oficial baixa uma mídia para enviar — o backend recusa
URLs que resolvem para endereços privados (proteção contra SSRF). Exceções são declaradas em
`MEDIA_FETCH_ALLOWED_HOSTS`.

## CORS e origens

| Mecanismo | Variável | Comportamento |
|---|---|---|
| CORS da API | `LINKTOR_CORS_ALLOWED_ORIGINS` | Requisição com `Origin` de outro host e fora da lista recebe `403`. Mesmo host da API é sempre aceito. Lista vazia: em `release` nega tudo; em `debug` aceita `localhost` e `127.0.0.1`. |
| WebSocket do painel (`/api/v1/ws`) | `LINKTOR_WS_ALLOWED_ORIGINS` | Aceita a lista, o mesmo host, conexões sem `Origin` e origens `localhost`. Exige JWT válido em todos os casos. |
| WebSocket do WebChat (`/ws/:channelId`) | `LINKTOR_WS_ALLOWED_ORIGINS`, `LINKTOR_WS_ENFORCE_ORIGIN`, `LINKTOR_WS_ALLOW_EMPTY_ORIGIN` | Recusa conexões sem `Origin` (a menos de `ALLOW_EMPTY_ORIGIN`). Aceita a lista e, se `ENFORCE_ORIGIN` não estiver ligado, também `localhost`. |

Em produção com WebChat, coloque em `LINKTOR_WS_ALLOWED_ORIGINS` a origem do painel **e** a de cada
site que incorpora o widget, e ligue `LINKTOR_WS_ENFORCE_ORIGIN=true`.

A lista de cabeçalhos liberados pelo CORS não inclui `X-API-Key`: chaves de API são para chamadas
servidor a servidor, não para JavaScript em páginas de terceiros.

O backend acrescenta a todas as respostas `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
`Referrer-Policy: strict-origin-when-cross-origin` e, quando a requisição chegou por HTTPS,
`Strict-Transport-Security`.

## Rate limiting {#rate-limiting}

Há duas camadas, independentes.

**No backend (Redis).** Janela fixa de **100 requisições por minuto**, não configurável:

| Rotas | Chave do limite |
|---|---|
| `/api/v1/auth/*` (login, renovação, logout) | IP do cliente |
| `/api/v1/webhooks/*` | IP do cliente |
| Todas as rotas autenticadas | **Tenant** — usuários do painel e chaves de API do mesmo tenant dividem a mesma cota |

Acima do limite a resposta é `429`, com o código `RATE_LIMITED` e o campo `retry_after`. As respostas
trazem `X-RateLimit-Limit`, `X-RateLimit-Remaining` e `X-RateLimit-Reset`. Se o Redis estiver
indisponível, o limite **deixa de ser aplicado** (as requisições passam).

**No Traefik (pilha de `deploy/`).** O middleware `rate-limit` no roteador da API: média de 100
requisições por segundo por IP, com rajada de 200. Há um `rate-limit-strict` (10 por segundo, rajada
de 20) definido em `deploy/traefik/dynamic/middlewares.yml` e não usado por nenhum roteador.

:::note
A cota por tenant é compartilhada entre o painel e as integrações. Uma integração que faz muitas
chamadas por minuto pode levar o painel do mesmo tenant a receber `429`. Trate `429` no cliente
respeitando `retry_after`.
:::

## O que o deploy de referência endurece

Na pilha de `deploy/` (veja [Produção](./producao.md)):

| Camada | Medida |
|---|---|
| Containers | Backend roda como usuário sem privilégios (`linktor`, uid 1000); painel, como `node`. |
| Rede | PostgreSQL, Redis e NATS em rede Docker `internal`, sem portas no host. Redis com senha. |
| Borda | Só o Traefik publica portas (`80`, `443`); HTTP redireciona para HTTPS; certificados Let's Encrypt com chave EC256; HSTS com `includeSubDomains` e `preload`. |
| Servidor | `install-vps.sh`: UFW liberando SSH, `80` e `443`; `fail2ban` para SSH; atualizações de segurança automáticas; ajustes de `sysctl`. |
| Segredos | Tudo em `/opt/linktor/.env`, modo `600`; nada nas imagens. |
| Logs | Rotação de 10 MB por 5 arquivos em cada container. |

O que essa pilha **não** fecha e você deve tratar:

### `/metrics` e `/swagger` ficam públicos

O backend serve `GET /metrics` e `GET /swagger/*` sem autenticação, e o roteador de `api.<domínio>`
encaminha todos os caminhos. Uma forma de restringir, acrescentando rótulos ao serviço `backend` no
Compose (não faz parte do repositório; ajuste a faixa de IP):

```yaml
      - "traefik.http.routers.backend-ops.rule=Host(`api.${ROOT_DOMAIN}`) && (PathPrefix(`/metrics`) || PathPrefix(`/swagger`))"
      - "traefik.http.routers.backend-ops.entrypoints=websecure"
      - "traefik.http.routers.backend-ops.tls.certresolver=letsencrypt"
      - "traefik.http.routers.backend-ops.service=backend"
      - "traefik.http.routers.backend-ops.middlewares=ops-allowlist"
      - "traefik.http.middlewares.ops-allowlist.ipallowlist.sourcerange=203.0.113.10/32"
```

Um Prometheus na mesma rede Docker coleta `backend:8081/metrics` diretamente, sem passar pelo
Traefik, e não é afetado.

### MinIO exposto

A API S3 (`s3.<domínio>`) e o console (`s3-console.<domínio>`) têm rota pública, e o backend usa o
**usuário root** do MinIO como credencial. Se ninguém fora do servidor precisa do console, remova os
rótulos `minio-console` do Compose e acesse por túnel SSH. Use uma senha root longa e exclusiva.

### Socket do Docker no Traefik

O Traefik monta `/var/run/docker.sock` (somente leitura) para descobrir os serviços. Quem compromete
o container do Traefik consegue ler a configuração de todos os containers, inclusive variáveis de
ambiente com segredos. Mantenha a imagem do Traefik atualizada e o dashboard protegido.

### Logs com tokens

O log de acesso do backend grava a query string. Ela pode conter o token do WebSocket do painel
(`?token=`), o token do Linktor Bridge e assinaturas de mídia. Restrinja quem lê os logs e por
quanto tempo eles são guardados.

### Usuários de demonstração

O seed cria `admin@demo.com` e `agent@demo.com` com a mesma senha. Em produção, remova os dois
depois de criar o seu administrador (passo 7 de [Produção](./producao.md)) e nunca compile o painel
com `NEXT_PUBLIC_SHOW_DEMO_CREDENTIALS=true`.

## Checklist {#checklist}

**Antes do go-live**

- [ ] `LINKTOR_SERVER_MODE=release`.
- [ ] `LINKTOR_JWT_SECRET` e `LINKTOR_CRYPTO_ENCRYPTION_KEY` aleatórios, com 32 caracteres ou mais,
      diferentes entre si.
- [ ] Cópia do `.env` em cofre de senhas, separada dos backups do banco.
- [ ] Senhas de PostgreSQL, Redis e MinIO fortes e exclusivas deste ambiente.
- [ ] `LINKTOR_JWT_COOKIE_SECURE` no padrão (`true`) e todo o acesso por HTTPS.
- [ ] `LINKTOR_CORS_ALLOWED_ORIGINS` e `LINKTOR_WS_ALLOWED_ORIGINS` com as origens exatas;
      `LINKTOR_WS_ENFORCE_ORIGIN=true` se houver WebChat.
- [ ] `MEDIA_SIGNING_SECRET` definido; data de `MEDIA_UNSIGNED_UNTIL` combinada com os consumidores.
- [ ] Todo canal com segredo de webhook configurado; um teste com assinatura inválida devolve `401`.
- [ ] Usuários de demonstração removidos; `LINKTOR_SEED_DEMO` e `SEED_RESET_DESTRUCTIVE` ausentes.
- [ ] `/metrics` e `/swagger` restritos por IP ou rede.
- [ ] Console do MinIO e dashboard do Traefik sem rota pública ou com senha forte.
- [ ] Porta `8081` do backend inacessível de fora; só `22`, `80` e `443` abertas.
- [ ] Redis no ar e sem aviso `HTTP rate limiting disabled` no log do backend.
- [ ] `LINKTOR_ENABLED_CHANNEL_TYPES` limitado aos canais que você realmente oferece.

**Rotina**

- [ ] Chaves de API com o menor conjunto de scopes necessário; chaves sem uso revogadas.
- [ ] Usuários desligados removidos ou desativados.
- [ ] Backups cifrados ou em armazenamento com acesso restrito (eles contêm as sessões do WhatsApp
      e o banco).
- [ ] Imagens de Traefik, PostgreSQL, Redis, NATS e MinIO atualizadas.
- [ ] Rotação periódica de `LINKTOR_JWT_SECRET`, da senha do banco e da chave SSH de deploy.
- [ ] Limpeza dos logs de canal agendada (veja [Monitoramento](./monitoramento.md)).

**Após um incidente**

- [ ] Vazou o `.env`: troque `LINKTOR_JWT_SECRET`, as senhas de infraestrutura e a chave de
      criptografia (com recifragem), e gere novos tokens nos provedores.
- [ ] Vazou uma URL de mídia: `POST /api/v1/media/revoke` com a chave ou o prefixo.
- [ ] Vazou uma chave de API: revogue-a no painel e crie outra.
- [ ] Vazou o volume `backend_storages`: desconecte os aparelhos pelo WhatsApp no celular e pareie
      de novo.
