#!/usr/bin/env bash
# Verificações de banco e métrica para UM envio do smoke de canal sandbox
# (docs/sandbox-channel-smoke.md). Uso:
#   scripts/sandbox-smoke-check.sh <message_id>
# Config por env: PGHOST/PGPORT/PGUSER/PGDATABASE (padrões localhost/5432/linktor/linktor),
# METRICS_URL (padrão http://localhost:8081/metrics).
set -euo pipefail

MSG_ID="${1:?uso: $0 <message_id>}"
PGHOST="${PGHOST:-localhost}"; PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-linktor}"; PGDATABASE="${PGDATABASE:-linktor}"
METRICS_URL="${METRICS_URL:-http://localhost:8081/metrics}"
PSQL=(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" -tA)

echo "== mensagem $MSG_ID =="
"${PSQL[@]}" -c "SELECT 'status='||status||' error='||coalesce(error_message,'') FROM messages WHERE id='$MSG_ID'" \
  || echo "(sem linha em messages — envio via NATS direto/campanha não cria linha)"

echo "== conversa (environment) =="
"${PSQL[@]}" -c "SELECT 'conversation='||c.id||' environment='||c.environment
  FROM messages m JOIN conversations c ON c.id=m.conversation_id WHERE m.id='$MSG_ID'" || true

echo "== message_logs =="
"${PSQL[@]}" -c "SELECT level||' | '||message||' | blocked_by='||coalesce(metadata->>'blocked_by','-')
  FROM message_logs WHERE metadata->>'message_id'='$MSG_ID' ORDER BY created_at"

echo "== métricas de guarda (valores acumulados) =="
curl -s "$METRICS_URL" | grep -E 'linktor_outbound_(guard_blocked|messages)_total' || true

echo
echo "Interpretação: bloqueio de guarda => log 'Envio bloqueado por guarda (<reason>)' com blocked_by"
echo "preenchido e NENHUMA linha 'Mensagem entregue ao canal' para este message_id."
