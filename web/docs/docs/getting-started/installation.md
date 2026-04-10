---
sidebar_position: 1
title: Installation
---

# Installation

This guide covers different ways to install and run Linktor.

## Prerequisites

- Docker and Docker Compose (recommended)
- Or: Go 1.21+, Node.js 20+, PostgreSQL 15+, Redis 7+

## Docker Installation (Recommended)

The easiest way to get started is with Docker Compose:

```bash
# Clone the repository
git clone https://github.com/linktor/linktor.git
cd linktor

# Copy environment file
cp .env.example .env

# Start all services
docker compose up --build
```

This will start:
- API Server on `http://localhost:8081`
- Admin Dashboard on `http://localhost:3000`
- PostgreSQL on port `5432`
- Redis on port `6379`
- NATS on port `4222`
- MinIO on port `9000`

## Environment Variables

Configure Linktor by editing the `.env` file:

```bash
# Database
LINKTOR_DATABASE_HOST=localhost
LINKTOR_DATABASE_PORT=5432
LINKTOR_DATABASE_USER=linktor
LINKTOR_DATABASE_PASSWORD=linktor
LINKTOR_DATABASE_DATABASE=linktor
LINKTOR_DATABASE_SSL_MODE=disable

# Redis
LINKTOR_REDIS_HOST=localhost
LINKTOR_REDIS_PORT=6379

# NATS
LINKTOR_NATS_URL=nats://localhost:4222

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# JWT Secret (generate a strong secret)
LINKTOR_JWT_SECRET=your-super-secret-key-change-in-production

# Admin frontend
NEXT_PUBLIC_API_URL=http://localhost:8081/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:8081/api/v1/ws
NEXT_PUBLIC_WEBHOOK_BASE_URL=http://localhost:8081

# AI Providers (optional)
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Channel Credentials (configure as needed)
TWILIO_ACCOUNT_SID=...
TWILIO_AUTH_TOKEN=...
```

## Manual Installation

For development or custom deployments:

### 1. Start Dependencies

```bash
# PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_USER=linktor \
  -e POSTGRES_PASSWORD=linktor \
  -e POSTGRES_DB=linktor \
  -p 5432:5432 postgres:15

# Redis
docker run -d --name redis -p 6379:6379 redis:7

# NATS
docker run -d --name nats -p 4222:4222 nats:latest

# MinIO
docker run -d --name minio \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  -p 9000:9000 minio/minio server /data
```

### 2. Build and Run API Server

```bash
cd internal
go mod download
go build -o linktor ./cmd/server
./linktor
```

### 3. Build and Run Admin Dashboard

```bash
cd web/admin
npm install
npm run dev
```

## Kubernetes Deployment

See [Kubernetes Guide](/self-hosting/kubernetes) for production-ready Kubernetes manifests.

## Health Check

Verify the installation:

```bash
# Check API health
curl http://localhost:8081/health

# Expected response
{"status": "healthy"}
```

## Next Steps

- [Quick Start](/getting-started/quick-start) - Create your first chatbot
- [Authentication](/getting-started/authentication) - Set up API authentication
