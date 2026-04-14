# Secrets

Two places hold secrets:

1. **`/opt/linktor/.env` on the VPS** — runtime config consumed by Docker Compose. Mode `600`, owned by `linktor:linktor`. Never committed.
2. **GitHub repository secrets** — used by the `build-and-push` and `deploy` workflows.

## On the VPS — `/opt/linktor/.env`

| Key                       | Generate with                            | Notes                              |
| ------------------------- | ---------------------------------------- | ---------------------------------- |
| `ROOT_DOMAIN`             | n/a                                      | `linktor.dev`                      |
| `ACME_EMAIL`              | n/a                                      | inbox you actually read            |
| `POSTGRES_PASSWORD`       | `openssl rand -base64 48`                | rotate via runbook                 |
| `REDIS_PASSWORD`          | `openssl rand -base64 48`                | required (auth is enforced)        |
| `JWT_SECRET`              | `openssl rand -base64 64`                | rotating invalidates all sessions  |
| `MINIO_ROOT_USER`         | n/a                                      | e.g. `linktor`                     |
| `MINIO_ROOT_PASSWORD`     | `openssl rand -base64 48`                | min 8 chars                        |
| `BACKUP_S3_*` (optional)  | from offsite provider                    | only if pushing backups offsite    |

The `*_IMAGE_TAG` keys are not secrets — the deploy workflow rewrites them on every rollout. Leave the defaults (`latest`) in the file you check into git.

## In GitHub — repository or environment secrets

Add these under **Settings → Secrets and variables → Actions → New repository secret** (or, better, an `production` environment with required reviewers).

| Secret             | Value                                                                 |
| ------------------ | --------------------------------------------------------------------- |
| `DEPLOY_HOST`      | VPS public IP or hostname                                             |
| `DEPLOY_USER`      | `linktor`                                                             |
| `DEPLOY_PORT`      | `22` (omit if default)                                                |
| `DEPLOY_SSH_KEY`   | OpenSSH **private** key (the matching public key is in `~linktor/.ssh/authorized_keys` on the VPS) |

`GITHUB_TOKEN` is provided automatically and has `packages:write` because the workflow declares it — no extra setup needed for GHCR.

### Generating the deploy key

On your laptop:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/linktor-deploy -C "linktor-gha-deploy" -N ""
```

Add the **public** key to the VPS:

```bash
ssh-copy-id -i ~/.ssh/linktor-deploy.pub linktor@<VPS_IP>
# or manually:
cat ~/.ssh/linktor-deploy.pub | ssh deploy@<VPS_IP> "sudo -u linktor tee -a /home/linktor/.ssh/authorized_keys"
```

Add the **private** key (`~/.ssh/linktor-deploy`) to GitHub as `DEPLOY_SSH_KEY`. Paste the entire file including `-----BEGIN OPENSSH PRIVATE KEY-----` / `-----END OPENSSH PRIVATE KEY-----`.

## Rotation

| Item                | Cadence  | Procedure                                                                       |
| ------------------- | -------- | ------------------------------------------------------------------------------- |
| `JWT_SECRET`        | yearly   | edit `.env`, restart `backend`, all users re-login                              |
| Database password   | yearly   | runbook §3 (rotate inside Postgres + update `.env` + restart `backend`)         |
| `DEPLOY_SSH_KEY`    | quarterly | generate new key, append to `authorized_keys`, update GH secret, remove old key |
| Traefik dashboard   | quarterly | regenerate htpasswd, copy to `/opt/linktor/deploy/traefik/dynamic/dashboard.htpasswd`, reload Traefik |
