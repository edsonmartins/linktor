#!/bin/bash
# Script para rodar apenas o backend

cd "$(dirname "$0")/.."

echo "=== Iniciando Backend Linktor ==="
echo "Porta: 8081 (conforme config.yaml)"
echo ""

# Parar processo existente
pkill -f "bin/server" 2>/dev/null || true
sleep 1

# Verificar se binário existe
if [ ! -f "bin/server" ]; then
    echo "Binário não encontrado. Compilando..."
    go build -o bin/server ./cmd/server
fi

# Iniciar servidor
./bin/server 2>&1
