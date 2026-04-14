#!/usr/bin/env bash
# Local helper: pull new images and roll the stack on the VPS.
# Run from /opt/linktor as the linktor user.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$DEPLOY_DIR/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

cd "$DEPLOY_DIR"

echo "[deploy] Pulling new images"
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" pull

echo "[deploy] Applying stack"
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" up -d --remove-orphans

echo "[deploy] Pruning dangling images"
docker image prune -f

echo "[deploy] Status:"
docker compose -f docker-compose.prod.yml --env-file "$ENV_FILE" ps
