# Linktor — Deploy

Production deployment for **linktor.dev**. Single-VPS Docker Compose stack with Traefik for TLS/routing, Postgres + Redis + NATS + MinIO self-hosted, and GitHub Actions pushing images to `ghcr.io/integrall-tech`.

## Topology

```
                                  Internet
                                     │
                            ┌────────▼────────┐
                            │     Traefik     │  (80/443, ACME EC256)
                            │  ACME / routing │
                            └──┬──────────┬───┘
                               │          │
        ┌──────────────────────┼──────────┼────────────────────────┐
        │                      │          │                        │
   linktor.dev          app.linktor.dev   api.linktor.dev    s3.linktor.dev
   www.linktor.dev      ──────────────    ─────────────      s3-console.linktor.dev
        │                      │                │                  │
   ┌────▼────┐           ┌─────▼─────┐    ┌─────▼─────┐      ┌─────▼─────┐
   │ landing │           │   admin   │    │  backend  │      │   minio   │
   │ (nginx) │           │ (next.js) │    │   (Go)    │      │ (s3 api)  │
   └─────────┘           └───────────┘    └─────┬─────┘      └─────┬─────┘
                                                │                  │
                                  ┌─────────────┼──────────────┐   │
                                  │             │              │   │
                              ┌───▼──┐      ┌───▼──┐      ┌────▼───▼──┐
                              │  pg  │      │redis │      │   nats    │
                              └──────┘      └──────┘      └───────────┘
                              (linktor-internal — no external ports)
```

Two Docker networks:

- `linktor-edge` — Traefik + every container that needs ingress (backend, admin, landing, MinIO).
- `linktor-internal` — fully internal (`internal: true`); Postgres, Redis, NATS only reachable from inside.

## Layout

```
deploy/
├── README.md                 ← this file
├── docker-compose.prod.yml   ← production stack
├── .env.example              ← env template (copy to /opt/linktor/.env)
├── traefik/
│   ├── traefik.yml           ← static config
│   ├── dynamic/              ← middlewares, basic-auth file
│   └── .gitignore            ← keeps acme.json + htpasswd out of git
├── scripts/
│   ├── install-vps.sh        ← bootstraps Ubuntu 24.04/25.04
│   ├── deploy.sh             ← pull + up locally on the VPS
│   ├── backup.sh             ← daily pg_dump + minio mirror + offsite
│   └── restore.sh            ← restore from backup archive
└── docs/
    ├── vps-setup.md          ← step-by-step server provisioning
    ├── dns.md                ← DNS records for linktor.dev
    ├── secrets.md            ← GitHub + VPS secrets checklist
    └── runbook.md            ← deploy/rollback/incident playbook
```

CI/CD lives in `.github/workflows/`:

- `build-and-push.yml` — builds backend, admin and landing images on every push to `main` (and on tags), publishes to GHCR under `ghcr.io/integrall-tech/linktor-*`.
- `deploy.yml` — called by `build-and-push.yml`, SSHes into the VPS and rolls the stack.

## Quickstart

1. **Provision the VPS** — follow [`docs/vps-setup.md`](docs/vps-setup.md). The short version: copy `scripts/install-vps.sh` to the box and run it as root.
2. **DNS** — point the records in [`docs/dns.md`](docs/dns.md) at the VPS public IP.
3. **GitHub secrets** — add the values listed in [`docs/secrets.md`](docs/secrets.md).
4. **Bootstrap** — clone this repo to `/opt/linktor` on the VPS, copy `deploy/.env.example` to `/opt/linktor/.env`, fill in real secrets, then `chmod 600 /opt/linktor/.env`.
5. **First boot**:
   ```bash
   cd /opt/linktor/deploy
   docker compose -f docker-compose.prod.yml --env-file ../.env up -d
   docker compose -f docker-compose.prod.yml --env-file ../.env logs -f traefik backend
   ```
6. **Subsequent rollouts** are automatic on every merge to `main`. Manual deploys: re-run the `deploy` workflow with a specific image tag, or run `deploy/scripts/deploy.sh` on the VPS.

## What runs where

| Subdomain                  | Service          | Port (internal) | Auth                     |
| -------------------------- | ---------------- | --------------- | ------------------------ |
| `linktor.dev`, `www.…`     | landing (nginx)  | 80              | public                   |
| `app.linktor.dev`          | admin (Next.js)  | 3000            | app login                |
| `api.linktor.dev`          | backend (Go)     | 8081            | JWT                      |
| `traefik.linktor.dev`      | Traefik dashboard| —               | basic auth (htpasswd)    |
| `s3.linktor.dev`           | MinIO S3 API     | 9000            | access key               |
| `s3-console.linktor.dev`   | MinIO console    | 9001            | MinIO root credentials   |

If you don't want MinIO publicly reachable, drop the `traefik.enable=true` labels from the `minio` service and access it through the VPS via SSH tunnel.

## Conventions

- All secrets live in `/opt/linktor/.env` (mode `600`, owned by `linktor:linktor`). Never commit, never ship in images.
- Image tags in `.env` (`*_IMAGE_TAG`) are the source of truth. The deploy workflow rewrites them on every rollout.
- Postgres data is in the named volume `linktor_postgres_data`. **Do not** delete it without a backup.
- Backups live under `/var/backups/linktor`. Cron entry suggested in [`docs/runbook.md`](docs/runbook.md).
