#!/bin/bash
# Script para rodar apenas o frontend

cd "$(dirname "$0")/../web/admin"

echo "=== Iniciando Frontend Linktor ==="
echo ""

# Verificar se node_modules existe
if [ ! -d "node_modules" ]; then
    echo "Instalando dependências..."
    npm install
fi

# Iniciar frontend
npm run dev
