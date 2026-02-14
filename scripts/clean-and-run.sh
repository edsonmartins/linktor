#!/bin/bash
# Script para limpar build/cache e rodar o servidor

set -e

cd "$(dirname "$0")/.."

echo "=== 1. Parando processos existentes ==="
pkill -f "bin/server" 2>/dev/null || true
sleep 1

echo "=== 2. Limpando build e cache ==="
rm -rf bin/server 2>/dev/null || true
go clean -cache
go clean -modcache 2>/dev/null || true

echo "=== 3. Testando conexão com PostgreSQL ==="
if nc -zv 192.168.1.110 5444 -w 3 2>&1 | grep -q "succeeded"; then
    echo "✓ PostgreSQL acessível em 192.168.1.110:5444"
else
    echo "✗ ERRO: Não conseguiu conectar ao PostgreSQL em 192.168.1.110:5444"
    exit 1
fi

echo "=== 4. Recompilando servidor ==="
go build -o bin/server ./cmd/server
echo "✓ Build completo"

echo "=== 5. Iniciando servidor ==="
echo "Logs serão exibidos abaixo. Use Ctrl+C para parar."
echo ""
./bin/server 2>&1
