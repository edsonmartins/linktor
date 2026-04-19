#!/usr/bin/env bash
# Daily backup of Postgres + MinIO data. Optionally syncs to an S3 target.
# Designed to run from cron as the linktor user:
#   0 3 * * * /opt/linktor/deploy/scripts/backup.sh >> /var/log/linktor-backup.log 2>&1
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$DEPLOY_DIR/.." && pwd)"

ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file at $ENV_FILE" >&2
  exit 1
fi
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

BACKUP_DIR="${BACKUP_DIR:-/var/backups/linktor}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
WORK_DIR="$BACKUP_DIR/$TIMESTAMP"

log() { printf "\033[1;34m[backup]\033[0m %s\n" "$*"; }

mkdir -p "$WORK_DIR"

log "Dumping Postgres → postgres.sql.gz"
docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" linktor-postgres \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom --compress=9 \
  > "$WORK_DIR/postgres.dump"

log "Snapshotting MinIO data via mc mirror"
docker run --rm \
  --network linktor_linktor-internal \
  -v "$WORK_DIR:/backup" \
  -e MC_HOST_minio="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@minio:9000" \
  minio/mc:latest \
  mirror --quiet --overwrite minio/ /backup/minio/ || true

log "Capturing compose + traefik config"
tar -czf "$WORK_DIR/config.tar.gz" \
  -C "$DEPLOY_DIR" docker-compose.prod.yml traefik

log "Computing checksums"
( cd "$WORK_DIR" && sha256sum -- * > SHA256SUMS )

ARCHIVE="$BACKUP_DIR/linktor-$TIMESTAMP.tar.gz"
log "Sealing archive → $ARCHIVE"
tar -czf "$ARCHIVE" -C "$BACKUP_DIR" "$TIMESTAMP"
rm -rf "$WORK_DIR"

if [[ -n "${BACKUP_S3_ENDPOINT:-}" && -n "${BACKUP_S3_BUCKET:-}" ]]; then
  if [[ -z "${BACKUP_S3_ACCESS_KEY:-}" || -z "${BACKUP_S3_SECRET_KEY:-}" ]]; then
    log "Skipping offsite push: BACKUP_S3_ACCESS_KEY / BACKUP_S3_SECRET_KEY not set"
  else
    # mc authenticates via MC_HOST_<alias> which MUST embed the credentials
    # in the URL: https://KEY:SECRET@host. Setting AWS_ACCESS_KEY_ID is
    # ignored. Build the URL here so BACKUP_S3_ENDPOINT can stay a bare
    # scheme+host string in .env.
    proto="${BACKUP_S3_ENDPOINT%%://*}"
    hostpart="${BACKUP_S3_ENDPOINT#*://}"
    if [[ "$proto" == "$BACKUP_S3_ENDPOINT" ]]; then
      # No scheme provided — assume https.
      proto="https"
      hostpart="$BACKUP_S3_ENDPOINT"
    fi
    # urlencode the secret so '+', '/', '=' characters in it don't break parsing
    encoded_secret=$(printf '%s' "$BACKUP_S3_SECRET_KEY" | python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.stdin.read(), safe=""))' 2>/dev/null || printf '%s' "$BACKUP_S3_SECRET_KEY")
    mc_host="${proto}://${BACKUP_S3_ACCESS_KEY}:${encoded_secret}@${hostpart}"

    log "Pushing to offsite S3: ${proto}://${hostpart}/${BACKUP_S3_BUCKET}"
    docker run --rm \
      -v "$BACKUP_DIR:/backup:ro" \
      -e MC_HOST_offsite="$mc_host" \
      minio/mc:latest \
      cp "/backup/$(basename "$ARCHIVE")" "offsite/$BACKUP_S3_BUCKET/"
  fi
fi

log "Pruning archives older than ${RETENTION_DAYS}d"
find "$BACKUP_DIR" -maxdepth 1 -name 'linktor-*.tar.gz' -mtime "+${RETENTION_DAYS}" -delete

log "Backup complete: $ARCHIVE"
