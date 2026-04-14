#!/usr/bin/env bash
# Restore a Linktor backup archive produced by backup.sh.
# Usage: restore.sh /var/backups/linktor/linktor-20260414T030000Z.tar.gz
set -euo pipefail

ARCHIVE="${1:-}"
if [[ -z "$ARCHIVE" || ! -f "$ARCHIVE" ]]; then
  echo "Usage: $0 <archive.tar.gz>" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$DEPLOY_DIR/.." && pwd)"

ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

log() { printf "\033[1;34m[restore]\033[0m %s\n" "$*"; }

log "Extracting $ARCHIVE → $WORK"
tar -xzf "$ARCHIVE" -C "$WORK"
SNAP_DIR="$(find "$WORK" -mindepth 1 -maxdepth 1 -type d | head -n1)"

log "Verifying checksums"
( cd "$SNAP_DIR" && sha256sum -c SHA256SUMS )

read -r -p "This will WIPE the current Postgres database and MinIO bucket. Continue? [yes/NO] " confirm
[[ "$confirm" == "yes" ]] || { echo "Aborted."; exit 1; }

log "Stopping backend so nothing writes during restore"
( cd "$DEPLOY_DIR" && docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" stop backend )

log "Restoring Postgres"
cat "$SNAP_DIR/postgres.dump" | docker exec -i -e PGPASSWORD="$POSTGRES_PASSWORD" linktor-postgres \
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists --no-owner

if [[ -d "$SNAP_DIR/minio" ]]; then
  log "Restoring MinIO objects"
  docker run --rm \
    --network linktor_linktor-internal \
    -v "$SNAP_DIR/minio:/snap:ro" \
    -e MC_HOST_minio="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@minio:9000" \
    minio/mc:latest \
    mirror --quiet --overwrite /snap/ minio/
fi

log "Restarting backend"
( cd "$DEPLOY_DIR" && docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" start backend )

log "Restore complete."
