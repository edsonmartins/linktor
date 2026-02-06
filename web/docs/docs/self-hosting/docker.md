---
sidebar_position: 1
title: Docker Deployment
---

# Docker Deployment

Deploy Linktor using Docker Compose for development, testing, or small production deployments. This guide covers everything from basic setup to production-ready configuration.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- 4GB RAM minimum (8GB recommended)
- 20GB disk space

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/linktor/linktor.git
cd linktor
```

### 2. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit configuration
nano .env
```

Essential variables to configure:

```bash
# Application
APP_URL=http://localhost:3000
API_URL=http://localhost:8080

# Database
POSTGRES_USER=linktor
POSTGRES_PASSWORD=your-secure-password
POSTGRES_DB=linktor

# Redis
REDIS_PASSWORD=your-redis-password

# Encryption (generate with: openssl rand -hex 32)
ENCRYPTION_KEY=your-32-byte-hex-key

# JWT Secret (generate with: openssl rand -hex 64)
JWT_SECRET=your-jwt-secret

# Admin User
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your-admin-password
```

### 3. Start Services

```bash
docker-compose up -d
```

### 4. Access Linktor

- **Admin Dashboard**: http://localhost:3000
- **API**: http://localhost:8080
- **API Documentation**: http://localhost:8080/docs

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Docker Network                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│  │  Nginx  │  │   API   │  │  Admin  │  │ Worker  │  │ Webhook │   │
│  │ :80/443 │  │  :8080  │  │  :3000  │  │         │  │ Worker  │   │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘   │
│       │            │            │            │            │         │
│       └────────────┴────────────┴────────────┴────────────┘         │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Data Services                             │   │
│  ├──────────┬──────────┬──────────┬──────────┬─────────────────┤   │
│  │PostgreSQL│  Redis   │   NATS   │  MinIO   │  PgVector       │   │
│  │  :5432   │  :6379   │  :4222   │  :9000   │  (in Postgres)  │   │
│  └──────────┴──────────┴──────────┴──────────┴─────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Docker Compose Configuration

### Basic Configuration

```yaml
# docker-compose.yml
version: '3.8'

services:
  # PostgreSQL Database with pgvector
  postgres:
    image: pgvector/pgvector:pg16
    container_name: linktor-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis for caching and pub/sub
  redis:
    image: redis:7-alpine
    container_name: linktor-redis
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # NATS for message queuing
  nats:
    image: nats:2-alpine
    container_name: linktor-nats
    restart: unless-stopped
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats_data:/data
    healthcheck:
      test: ["CMD", "nats-server", "--help"]
      interval: 10s
      timeout: 5s
      retries: 5

  # MinIO for object storage
  minio:
    image: minio/minio:latest
    container_name: linktor-minio
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin}
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Linktor API Server
  api:
    image: linktor/api:latest
    container_name: linktor-api
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
      NATS_URL: nats://nats:4222
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD:-minioadmin}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      JWT_SECRET: ${JWT_SECRET}
      APP_URL: ${APP_URL}
      API_URL: ${API_URL}
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Linktor Admin Dashboard
  admin:
    image: linktor/admin:latest
    container_name: linktor-admin
    restart: unless-stopped
    depends_on:
      api:
        condition: service_healthy
    environment:
      NEXT_PUBLIC_API_URL: ${API_URL}
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Background Worker
  worker:
    image: linktor/worker:latest
    container_name: linktor-worker
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
      NATS_URL: nats://nats:4222
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD:-minioadmin}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}

volumes:
  postgres_data:
  redis_data:
  nats_data:
  minio_data:
```

## Production Configuration

### Enable HTTPS with Nginx

Add an Nginx reverse proxy with SSL:

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    container_name: linktor-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - ./certbot/www:/var/www/certbot:ro
    depends_on:
      - api
      - admin

  certbot:
    image: certbot/certbot
    container_name: linktor-certbot
    volumes:
      - ./ssl:/etc/letsencrypt
      - ./certbot/www:/var/www/certbot
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 12h & wait $${!}; done;'"
```

**nginx.conf:**

```nginx
events {
    worker_connections 1024;
}

http {
    upstream api {
        server api:8080;
    }

    upstream admin {
        server admin:3000;
    }

    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name your-domain.com;

        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }

        location / {
            return 301 https://$host$request_uri;
        }
    }

    # HTTPS Server
    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/ssl/live/your-domain.com/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/live/your-domain.com/privkey.pem;

        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
        ssl_prefer_server_ciphers off;

        # Admin Dashboard
        location / {
            proxy_pass http://admin;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_cache_bypass $http_upgrade;
        }

        # API
        location /api/ {
            proxy_pass http://api/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # WebSocket
        location /ws/ {
            proxy_pass http://api/ws/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_read_timeout 86400;
        }
    }
}
```

### Scaling Workers

Scale background workers for higher throughput:

```bash
docker-compose up -d --scale worker=3
```

Or in docker-compose.yml:

```yaml
services:
  worker:
    image: linktor/worker:latest
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
```

### Resource Limits

Configure resource constraints:

```yaml
services:
  api:
    image: linktor/api:latest
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M

  postgres:
    image: pgvector/pgvector:pg16
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
    command: >
      postgres
      -c shared_buffers=1GB
      -c effective_cache_size=3GB
      -c maintenance_work_mem=256MB
      -c work_mem=64MB
      -c max_connections=200
```

## Backup and Restore

### Database Backup

```bash
# Create backup
docker exec linktor-postgres pg_dump -U linktor linktor > backup.sql

# With compression
docker exec linktor-postgres pg_dump -U linktor linktor | gzip > backup.sql.gz
```

### Automated Backups

Add a backup service:

```yaml
services:
  backup:
    image: prodrigestivill/postgres-backup-local
    container_name: linktor-backup
    restart: unless-stopped
    depends_on:
      - postgres
    environment:
      POSTGRES_HOST: postgres
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      SCHEDULE: "@daily"
      BACKUP_KEEP_DAYS: 7
      BACKUP_KEEP_WEEKS: 4
      BACKUP_KEEP_MONTHS: 6
    volumes:
      - ./backups:/backups
```

### Restore from Backup

```bash
# Stop API and workers
docker-compose stop api worker

# Restore
gunzip < backup.sql.gz | docker exec -i linktor-postgres psql -U linktor linktor

# Restart services
docker-compose start api worker
```

### MinIO Backup

```bash
# Backup MinIO data
docker run --rm -v linktor_minio_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/minio-backup.tar.gz /data

# Restore
docker run --rm -v linktor_minio_data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/minio-backup.tar.gz -C /
```

## Monitoring

### Add Prometheus and Grafana

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: linktor-prometheus
    restart: unless-stopped
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    container_name: linktor-grafana
    restart: unless-stopped
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    ports:
      - "3001:3000"

volumes:
  prometheus_data:
  grafana_data:
```

**prometheus.yml:**

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'linktor-api'
    static_configs:
      - targets: ['api:8080']
    metrics_path: /metrics

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Health Checks

```bash
# Check all services
docker-compose ps

# Check specific service logs
docker-compose logs -f api

# Check service health
curl http://localhost:8080/health
```

## Updating

### Update to Latest Version

```bash
# Pull latest images
docker-compose pull

# Restart with new images
docker-compose up -d

# Clean up old images
docker image prune -f
```

### Zero-Downtime Updates

```bash
# Update API with rolling restart
docker-compose up -d --no-deps --scale api=2 api
sleep 30
docker-compose up -d --no-deps --scale api=1 api
```

## Troubleshooting

### Common Issues

**Database connection refused:**
```bash
# Check if PostgreSQL is running
docker-compose logs postgres

# Verify connection
docker exec -it linktor-postgres psql -U linktor -d linktor -c "SELECT 1"
```

**API not starting:**
```bash
# Check logs
docker-compose logs api

# Verify environment variables
docker-compose exec api env | grep DATABASE
```

**Out of disk space:**
```bash
# Clean up Docker
docker system prune -a --volumes

# Check disk usage
docker system df
```

### Logs

```bash
# All logs
docker-compose logs -f

# Specific service
docker-compose logs -f api

# Last 100 lines
docker-compose logs --tail=100 api
```

## Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_USER` | Database user | `linktor` |
| `POSTGRES_PASSWORD` | Database password | Required |
| `POSTGRES_DB` | Database name | `linktor` |
| `REDIS_PASSWORD` | Redis password | Required |
| `ENCRYPTION_KEY` | 32-byte hex key for encryption | Required |
| `JWT_SECRET` | Secret for JWT tokens | Required |
| `APP_URL` | Public URL for admin dashboard | Required |
| `API_URL` | Public URL for API | Required |
| `MINIO_ROOT_USER` | MinIO admin user | `minioadmin` |
| `MINIO_ROOT_PASSWORD` | MinIO admin password | `minioadmin` |

## Next Steps

- [Kubernetes Deployment](/self-hosting/kubernetes) - Scale with Kubernetes
- [API Reference](/api/overview) - Configure integrations
- [Channels](/channels/overview) - Set up messaging channels
