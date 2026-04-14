#!/usr/bin/env bash
# Bootstrap a fresh Ubuntu 24.04/25.04 VPS for Linktor.
# Run as root or via sudo. Idempotent: safe to re-run.
set -euo pipefail

LINKTOR_USER="${LINKTOR_USER:-linktor}"
LINKTOR_HOME="/opt/linktor"
SSH_PORT="${SSH_PORT:-22}"

log() { printf "\033[1;34m[install-vps]\033[0m %s\n" "$*"; }
require_root() {
  if [[ $EUID -ne 0 ]]; then
    echo "Run as root (or via sudo)." >&2
    exit 1
  fi
}

require_root

log "Updating apt cache and base packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get upgrade -y -qq
apt-get install -y -qq \
  ca-certificates curl gnupg lsb-release \
  ufw fail2ban unattended-upgrades \
  htop ncdu jq tmux git rsync \
  apache2-utils

log "Enabling unattended security upgrades"
dpkg-reconfigure -f noninteractive unattended-upgrades

log "Configuring swap (2G if missing)"
if ! swapon --show | grep -q '/swapfile'; then
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

log "Installing Docker Engine + Compose plugin"
if ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  CODENAME=$(. /etc/os-release && echo "$VERSION_CODENAME")
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $CODENAME stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
fi

log "Creating service user '$LINKTOR_USER'"
if ! id -u "$LINKTOR_USER" >/dev/null 2>&1; then
  useradd --system --create-home --shell /bin/bash --groups docker "$LINKTOR_USER"
else
  usermod -aG docker "$LINKTOR_USER"
fi

log "Preparing $LINKTOR_HOME"
mkdir -p "$LINKTOR_HOME" /var/backups/linktor
chown -R "$LINKTOR_USER:$LINKTOR_USER" "$LINKTOR_HOME" /var/backups/linktor
chmod 750 "$LINKTOR_HOME"

log "Configuring UFW firewall"
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow "$SSH_PORT"/tcp comment 'SSH'
ufw allow 80/tcp comment 'HTTP'
ufw allow 443/tcp comment 'HTTPS'
ufw --force enable

log "Configuring fail2ban (sshd jail)"
cat > /etc/fail2ban/jail.d/sshd.local <<EOF
[sshd]
enabled = true
port = $SSH_PORT
maxretry = 5
findtime = 10m
bantime = 1h
EOF
systemctl enable --now fail2ban
systemctl restart fail2ban

log "Hardening sysctl"
cat > /etc/sysctl.d/99-linktor.conf <<'EOF'
net.ipv4.tcp_syncookies = 1
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
fs.file-max = 2097152
vm.swappiness = 10
EOF
sysctl --system >/dev/null

log "Done. Next steps:"
cat <<EOF

  1. Copy the deploy/ folder to $LINKTOR_HOME on this server (or git clone the repo).
  2. Copy deploy/.env.example to $LINKTOR_HOME/.env, fill in real values, then chmod 600.
  3. Generate Traefik dashboard credentials:
       htpasswd -nbB admin 'YOUR-PASSWORD' > $LINKTOR_HOME/deploy/traefik/dynamic/dashboard.htpasswd
  4. Point linktor.dev DNS records at this server (see deploy/docs/dns.md).
  5. Authenticate to GHCR so docker pull works for private images:
       docker login ghcr.io -u <gh-user> -p <PAT-with-read:packages>
  6. Bring the stack up:
       cd $LINKTOR_HOME/deploy && docker compose -f docker-compose.prod.yml --env-file ../.env up -d

EOF
