# Checklist — Homologação / Produção

Verificações obrigatórias antes de apontar tráfego real para um ambiente.
Itens marcados **(prod)** só bloqueiam produção; o restante vale também para
homologação.

## 1. Modo e segredos

- [ ] `LINKTOR_SERVER_MODE=release` — em release o backend **recusa subir** com
      JWT secret/encryption key fracos e força validação de assinatura de
      webhooks.
- [ ] `JWT_SECRET` forte (`openssl rand -base64 64`), nunca o default.
- [ ] `CRYPTO_ENCRYPTION_KEY` forte (`openssl rand -base64 48`), ≥32 chars e
      **diferente** do JWT secret. Sem ela o boot em release falha.
- [ ] `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `MINIO_ROOT_PASSWORD` fortes e
      únicos por ambiente.
- [ ] `.env` com modo `600`, fora do git (ver `secrets.md`).
- [ ] `LINKTOR_SEED_DEMO` **não** definido (o usuário admin@demo.com/admin123 é
      só para desenvolvimento).

## 2. Rede e exposição

- [ ] `LINKTOR_CORS_ALLOWED_ORIGINS` com a(s) origem(ns) exata(s) do admin
      (ex.: `https://app.<dominio>`). Vazio = fallback para localhost.
- [ ] `LINKTOR_WS_ALLOWED_ORIGINS` idem para o WebSocket.
- [ ] Porta 8222 (monitoramento NATS) **não** exposta externamente.
- [ ] Postgres/Redis/MinIO apenas na rede interna (compose de prod já usa
      `linktor-internal`); em homolog, não publicar portas além do necessário.
- [ ] TLS via Traefik/Let's Encrypt funcionando (`https://api.<dominio>/health`).

## 3. Persistência

- [ ] Volumes montados no backend: `backend_uploads:/app/uploads` (mídia
      webchat + renders VRE) e `backend_storages:/app/storages` (sessões
      WhatsApp/whatsmeow). Sem o segundo, todo redeploy exige reparear QR code.
- [ ] `VRE_UPLOAD_DIR=/app/uploads/vre` definido (sem isso o storage do VRE
      fica desabilitado).
- [ ] Backup agendado (`deploy/scripts/backup.sh` via cron) — inclui Postgres,
      MinIO, uploads e sessões WhatsApp.
- [ ] **(prod)** Restore do backup testado pelo menos uma vez antes do go-live.

## 4. Frontend (admin)

- [ ] Imagem buildada com `NEXT_PUBLIC_API_URL` e `NEXT_PUBLIC_WS_URL` do
      ambiente (o build **falha** se ausentes — proposital).
- [ ] `NEXT_PUBLIC_SHOW_DEMO_CREDENTIALS` **não** definido (hint de credenciais
      demo só aparece quando explicitamente habilitado).

## 5. Smoke test pós-deploy

- [ ] `GET /health` → 200 e `GET /ready` → 200 (checa DB + NATS).
- [ ] Login no admin com usuário real (não demo).
- [ ] Criar canal de teste, enviar e receber uma mensagem ponta a ponta.
- [ ] Upload de mídia via webchat → derrubar e subir o container → arquivo
      ainda acessível (valida os volumes).
- [ ] Webhook com assinatura inválida → 401 (valida enforcement em release).

## 6. Observabilidade mínima

- [ ] `LINKTOR_LOG_FORMAT=json` e `LINKTOR_LOG_LEVEL=info` (ou `warn`).
- [ ] `docker compose logs` acessível; **(prod)** agregação de logs (Loki ou
      SaaS) recomendada antes do go-live.
- [ ] **(prod)** Alerta básico de disponibilidade no `/health` (uptime monitor
      externo serve).

## Pendências conhecidas (não bloqueiam homologação)

Registradas na auditoria de 2026-06-12; ver plano de fases:

- Consumer da DLQ do NATS (mensagens esgotadas hoje morrem em silêncio).
- Retry do AIConsumer sem limite (sem dead-letter).
- Tokens do admin em localStorage (migrar para cookie HttpOnly).
- Migrations sem versionamento/rollback.
- Imagens Docker sem pinning de versão exata.
