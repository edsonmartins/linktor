# Runbook

Day-to-day operations for the Linktor production VPS.

## Cheat sheet

```bash
# all commands assume you're the linktor user inside /opt/linktor
sudo -iu linktor
cd /opt/linktor/deploy
COMPOSE="docker compose -f docker-compose.prod.yml --env-file ../.env"

$COMPOSE ps                    # what's running
$COMPOSE logs -f backend       # tail backend logs
$COMPOSE logs -f traefik       # tail traefik logs (look for ACME)
$COMPOSE restart backend       # restart a single service
$COMPOSE pull && $COMPOSE up -d # roll the whole stack to current image tags
$COMPOSE down                  # stop everything (data volumes preserved)
```

## 1. Deploys

### Automatic (default)
Every push to `main` triggers `build-and-push.yml` → builds and publishes images to `ghcr.io/integrall-tech/linktor-*` → calls `deploy.yml` → SSHes into the VPS, rewrites `BACKEND_IMAGE_TAG` / `ADMIN_IMAGE_TAG` / `LANDING_IMAGE_TAG` in `.env` to the new commit SHA, then `pull` + `up -d`.

### Manual
Re-run the `deploy` workflow from the Actions tab and provide the image tag (e.g. `sha-abc1234` or `v1.2.0`).

Or, on the VPS:

```bash
cd /opt/linktor/deploy
sudoedit ../.env              # change *_IMAGE_TAG values
./scripts/deploy.sh
```

## 2. Rollback

Every successful CI run leaves a tagged image in GHCR. To roll back:

```bash
# find the previous image tag
docker images ghcr.io/integrall-tech/linktor-backend

# OR from the laptop, list the workflow runs
gh run list -w build-and-push.yml -L 10

# trigger deploy with that tag
gh workflow run deploy.yml -f image_tag=sha-<previous-sha>
```

If you don't have `gh`, edit `/opt/linktor/.env` on the VPS to set the old tag and run `./scripts/deploy.sh`.

> ⚠️ Rolling back **does not** roll back database migrations. The Go backend applies schema changes additively in `internal/infrastructure/database/postgres.go`; a rollback to an older binary against a newer schema is usually safe, but a destructive migration (column drop, table drop) requires a database restore as well — see §4.

## 3. Rotating database / Redis password

```bash
sudo -iu linktor
cd /opt/linktor/deploy
COMPOSE="docker compose -f docker-compose.prod.yml --env-file ../.env"

# 1. Rotate inside Postgres
$COMPOSE exec postgres psql -U linktor -c "ALTER USER linktor WITH PASSWORD 'NEW_PASSWORD';"

# 2. Update /opt/linktor/.env
sudoedit ../.env   # set POSTGRES_PASSWORD=NEW_PASSWORD

# 3. Recreate backend so it picks up the new env
$COMPOSE up -d --force-recreate backend
```

Same pattern for Redis: change `requirepass` via `redis-cli`, update `.env`, recreate `backend`.

## 4. Backups

### Manual snapshot

```bash
sudo -u linktor /opt/linktor/deploy/scripts/backup.sh
ls -lh /var/backups/linktor/
```

### Restore

```bash
sudo -u linktor /opt/linktor/deploy/scripts/restore.sh /var/backups/linktor/linktor-20260414T030000Z.tar.gz
```

`restore.sh` stops `backend`, restores Postgres via `pg_restore --clean`, mirrors MinIO objects back, then starts `backend`.

### Verifying backups

A backup you've never restored is not a backup. Once a quarter, restore the latest archive into a throwaway VM and confirm the admin can log in. Document the date in this section.

| Date verified | Verified by | Notes |
| ------------- | ----------- | ----- |
|               |             |       |

## 5. Certificates

Traefik handles Let's Encrypt automatically. To inspect:

```bash
sudo -u linktor docker volume inspect linktor_traefik_acme
sudo cat $(docker volume inspect linktor_traefik_acme -f '{{.Mountpoint}}')/acme.json | jq .letsencrypt.Certificates[].domain
```

If a certificate isn't renewing:

1. Check Traefik logs: `$COMPOSE logs --since 1h traefik | grep -i acme`.
2. Confirm DNS still points to the VPS.
3. Confirm port 80 is reachable (Let's Encrypt uses HTTP-01 fallback when TLS-ALPN fails).
4. As a last resort, delete the `acme.json` entry for that host and restart Traefik — a fresh cert will be issued.

## 6. Common incidents

### Backend won't start

```bash
$COMPOSE logs --tail=200 backend
```

Common causes:
- `.env` value missing or contains an unescaped `$` — Compose interpolates `$` in `--env-file`.
- Postgres not yet healthy on first boot — backend's healthcheck retries 6× with 30s start period; if it still fails, check `$COMPOSE logs postgres`.
- `JWT_SECRET` empty.

### 502 from Traefik

Usually means the target container is unhealthy. `$COMPOSE ps` should show one of admin/backend/landing as `unhealthy` or `restarting`. Tail its logs.

### Disk filling up

```bash
docker system df
docker image prune -af   # safe — CI re-pulls on next deploy
docker volume ls         # check for orphaned volumes
journalctl --vacuum-size=200M
```

Postgres growth: `$COMPOSE exec postgres psql -U linktor -c "\l+"` and look at the `linktor` database size.

### Locked out of SSH

UFW + fail2ban can lock you out after repeated failed attempts. Recover via your VPS provider's web console (Hetzner: Console; DigitalOcean: Recovery console). Then:

```bash
sudo fail2ban-client unban <YOUR_IP>
```

## 7. Updates

### Updating Linktor itself
Merge to `main`. CI handles the rest.

### Updating Traefik / Postgres / Redis / NATS
Edit `deploy/docker-compose.prod.yml`, bump the image tag, commit, push. CI redeploys.
**For Postgres major upgrades** (e.g. pg16 → pg17), you must `pg_dump` first, switch the image, then `pg_restore`. Don't just bump the tag.

### Updating the host OS

```bash
sudo apt-get update && sudo apt-get upgrade -y
sudo reboot   # if kernel/libc updated
```

`unattended-upgrades` already applies security patches automatically.
