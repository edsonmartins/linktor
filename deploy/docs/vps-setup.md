# VPS setup — Ubuntu 25.04

Target: a fresh Ubuntu 24.04 LTS or 25.04 host with public IPv4, root SSH, and at least **4 vCPU / 8 GB RAM / 80 GB SSD** for a starter production deploy. Adjust upward as traffic grows.

## 1. Initial SSH hardening (from your laptop)

```bash
ssh root@<VPS_IP>
```

On the box:

```bash
adduser --disabled-password --gecos "" deploy
usermod -aG sudo deploy
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys
```

Then edit `/etc/ssh/sshd_config`:

```
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
```

Reload SSH: `systemctl reload ssh`. From now on, log in as `deploy` and use `sudo`.

## 2. Bootstrap script

Copy the install script over and run it. You can either rsync the whole repo or just the script:

```bash
# from your laptop
scp deploy/scripts/install-vps.sh deploy@<VPS_IP>:/tmp/

# on the VPS
ssh deploy@<VPS_IP>
sudo bash /tmp/install-vps.sh
```

What it does:

- updates apt, installs base packages (curl, jq, ufw, fail2ban, htop, etc.)
- enables unattended security upgrades
- creates a 2 GB swap file
- installs Docker Engine + Compose plugin from Docker's official repo
- creates the `linktor` system user (added to `docker` group)
- prepares `/opt/linktor` and `/var/backups/linktor`
- configures UFW (allow 22/80/443 only)
- enables fail2ban with an SSH jail
- applies a small set of network/sysctl hardenings

The script is idempotent — re-run it any time you change settings.

## 3. Get the repo onto the VPS

```bash
sudo -u linktor bash
cd /opt/linktor
git clone https://github.com/integrall-tech/linktor.git .
```

If the repo is private, set up a deploy key first:

```bash
sudo -u linktor ssh-keygen -t ed25519 -f /home/linktor/.ssh/id_ed25519 -N ""
sudo -u linktor cat /home/linktor/.ssh/id_ed25519.pub
# add the printed key as a Deploy Key on the GitHub repo (read-only is fine)
```

## 4. Authenticate to GHCR

The deploy workflow ships images to `ghcr.io/integrall-tech`. The VPS must be able to pull them:

```bash
sudo -u linktor docker login ghcr.io \
  -u <github-username> \
  -p <PAT-with-read:packages>
```

Use a PAT scoped to `read:packages` only — store it in 1Password, not in `.env`.

## 5. Configure the environment file

```bash
sudo -u linktor cp /opt/linktor/deploy/.env.example /opt/linktor/.env
sudo -u linktor chmod 600 /opt/linktor/.env
sudoedit /opt/linktor/.env
```

Replace every `CHANGE_ME_*` value. Generate strong values with:

```bash
openssl rand -base64 48   # for POSTGRES_PASSWORD, REDIS_PASSWORD, MINIO_ROOT_PASSWORD
openssl rand -base64 64   # for JWT_SECRET
```

## 6. Traefik dashboard credentials

```bash
sudo apt-get install -y apache2-utils
htpasswd -nbB admin 'YOUR-DASHBOARD-PASSWORD' \
  | sudo -u linktor tee /opt/linktor/deploy/traefik/dynamic/dashboard.htpasswd
```

## 7. DNS

Point the records in [`dns.md`](dns.md) at the VPS public IP **before** starting the stack — Traefik will request Let's Encrypt certificates on first boot and needs the records resolving to succeed.

## 8. First boot

```bash
sudo -u linktor bash
cd /opt/linktor/deploy
docker compose -f docker-compose.prod.yml --env-file ../.env up -d
docker compose -f docker-compose.prod.yml --env-file ../.env logs -f traefik
```

Watch Traefik logs until you see successful ACME challenges for every host. Then visit:

- https://linktor.dev
- https://app.linktor.dev
- https://api.linktor.dev/health
- https://traefik.linktor.dev (basic auth)

## 9. Schedule backups

```bash
sudo -u linktor crontab -e
```

Add:

```
0 3 * * * /opt/linktor/deploy/scripts/backup.sh >> /var/log/linktor-backup.log 2>&1
```

`/var/log/linktor-backup.log` should be readable by the `linktor` user — run `sudo touch /var/log/linktor-backup.log && sudo chown linktor:linktor /var/log/linktor-backup.log` once.

## 10. Done

The VPS is now production-ready. Future deploys come in automatically through the `build-and-push` → `deploy` GitHub Actions chain on every merge to `main`. See [`runbook.md`](runbook.md) for day-to-day operations.
