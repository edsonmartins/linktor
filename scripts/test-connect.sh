#!/bin/bash
# Script para testar o endpoint de conexão WhatsApp

# Configuração
API_URL="${API_URL:-http://localhost:8081}"
CHANNEL_ID="${1:-}"

if [ -z "$CHANNEL_ID" ]; then
    echo "Uso: ./scripts/test-connect.sh <channel_id>"
    echo ""
    echo "Exemplo:"
    echo "  ./scripts/test-connect.sh 0af1959e-4d86-4ec6-af34-24d3dcaa275c"
    echo ""
    echo "Primeiro, faça login para obter o token:"
    echo "  curl -X POST $API_URL/api/v1/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"admin@example.com\",\"password\":\"senha\"}'"
    exit 1
fi

# Token (você precisa obter um token válido)
TOKEN="${TOKEN:-}"

if [ -z "$TOKEN" ]; then
    echo "ERRO: Defina a variável TOKEN com um token JWT válido"
    echo ""
    echo "Exemplo:"
    echo "  export TOKEN='eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'"
    echo "  ./scripts/test-connect.sh $CHANNEL_ID"
    exit 1
fi

echo "=== Testando conexão do canal WhatsApp ==="
echo "URL: $API_URL/api/v1/channels/$CHANNEL_ID/connect"
echo ""

curl -v -X POST "$API_URL/api/v1/channels/$CHANNEL_ID/connect" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    2>&1

echo ""
echo ""
echo "=== Resposta esperada ==="
echo "Se funcionar, deve retornar um objeto com 'qr_code' para WhatsApp Unofficial:"
echo '{
  "success": true,
  "data": {
    "channel": { ... },
    "qr_code": "1@...",
    "expires_in": 30
  }
}'
