---
sidebar_position: 2
title: Kubernetes Deployment
---

# Kubernetes Deployment

Deploy Linktor on Kubernetes for high availability, auto-scaling, and enterprise-grade resilience. This guide covers Helm chart deployment and manual configuration.

## Prerequisites

- Kubernetes 1.25+
- kubectl configured
- Helm 3.10+
- 8GB RAM minimum across nodes
- Storage class with dynamic provisioning

## Quick Start with Helm

### 1. Add the Helm Repository

```bash
helm repo add linktor https://charts.linktor.io
helm repo update
```

### 2. Create Namespace

```bash
kubectl create namespace linktor
```

### 3. Configure Values

Create a `values.yaml` file:

```yaml
# values.yaml
global:
  domain: linktor.example.com
  storageClass: standard

api:
  replicas: 2
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "2Gi"
      cpu: "1000m"

admin:
  replicas: 2
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "512Mi"
      cpu: "500m"

worker:
  replicas: 3
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "1Gi"
      cpu: "500m"

postgresql:
  enabled: true
  auth:
    postgresPassword: your-postgres-password
    database: linktor
  primary:
    persistence:
      size: 50Gi

redis:
  enabled: true
  auth:
    password: your-redis-password
  master:
    persistence:
      size: 10Gi

nats:
  enabled: true
  jetstream:
    enabled: true
    fileStorage:
      size: 10Gi

minio:
  enabled: true
  auth:
    rootUser: minioadmin
    rootPassword: your-minio-password
  persistence:
    size: 100Gi

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  tls:
    enabled: true

secrets:
  encryptionKey: your-32-byte-hex-key
  jwtSecret: your-jwt-secret
```

### 4. Install the Chart

```bash
helm install linktor linktor/linktor \
  --namespace linktor \
  --values values.yaml
```

### 5. Verify Installation

```bash
kubectl get pods -n linktor
kubectl get svc -n linktor
kubectl get ingress -n linktor
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Ingress Controller                           │   │
│  │                     (nginx/traefik/istio)                            │   │
│  └───────────────────────────────┬─────────────────────────────────────┘   │
│                                  │                                          │
│         ┌────────────────────────┼────────────────────────┐                │
│         │                        │                        │                │
│         ▼                        ▼                        ▼                │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐          │
│  │   Admin     │         │     API     │         │   Webhooks  │          │
│  │  Deployment │         │  Deployment │         │  Deployment │          │
│  │  (2 pods)   │         │  (3 pods)   │         │  (2 pods)   │          │
│  └─────────────┘         └──────┬──────┘         └─────────────┘          │
│                                 │                                          │
│  ┌─────────────┐               │               ┌─────────────┐            │
│  │   Worker    │◄──────────────┴──────────────►│   Worker    │            │
│  │  Deployment │                               │  Deployment │            │
│  │  (5 pods)   │                               │  (AI Tasks) │            │
│  └─────────────┘                               └─────────────┘            │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        StatefulSets                                  │   │
│  ├─────────────┬─────────────┬─────────────┬─────────────┬─────────────┤   │
│  │ PostgreSQL  │    Redis    │    NATS     │   MinIO     │  PgVector   │   │
│  │  Primary +  │   Primary + │   Cluster   │   Cluster   │ (Postgres)  │   │
│  │  Replicas   │   Replicas  │   (3 pods)  │   (4 pods)  │             │   │
│  └─────────────┴─────────────┴─────────────┴─────────────┴─────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Manual Kubernetes Manifests

### Namespace and Secrets

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: linktor
---
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: linktor-secrets
  namespace: linktor
type: Opaque
stringData:
  POSTGRES_PASSWORD: your-postgres-password
  REDIS_PASSWORD: your-redis-password
  ENCRYPTION_KEY: your-32-byte-hex-key
  JWT_SECRET: your-jwt-secret
  MINIO_ROOT_PASSWORD: your-minio-password
```

### ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: linktor-config
  namespace: linktor
data:
  DATABASE_URL: "postgres://linktor:$(POSTGRES_PASSWORD)@linktor-postgresql:5432/linktor"
  REDIS_URL: "redis://:$(REDIS_PASSWORD)@linktor-redis:6379"
  NATS_URL: "nats://linktor-nats:4222"
  MINIO_ENDPOINT: "linktor-minio:9000"
  APP_URL: "https://linktor.example.com"
  API_URL: "https://api.linktor.example.com"
```

### API Deployment

```yaml
# api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linktor-api
  namespace: linktor
spec:
  replicas: 3
  selector:
    matchLabels:
      app: linktor-api
  template:
    metadata:
      labels:
        app: linktor-api
    spec:
      containers:
        - name: api
          image: linktor/api:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: linktor-config
            - secretRef:
                name: linktor-secrets
          resources:
            requests:
              memory: "512Mi"
              cpu: "250m"
            limits:
              memory: "2Gi"
              cpu: "1000m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: linktor-api
  namespace: linktor
spec:
  selector:
    app: linktor-api
  ports:
    - port: 8080
      targetPort: 8080
  type: ClusterIP
```

### Admin Deployment

```yaml
# admin-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linktor-admin
  namespace: linktor
spec:
  replicas: 2
  selector:
    matchLabels:
      app: linktor-admin
  template:
    metadata:
      labels:
        app: linktor-admin
    spec:
      containers:
        - name: admin
          image: linktor/admin:latest
          ports:
            - containerPort: 3000
          env:
            - name: NEXT_PUBLIC_API_URL
              value: "https://api.linktor.example.com"
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: linktor-admin
  namespace: linktor
spec:
  selector:
    app: linktor-admin
  ports:
    - port: 3000
      targetPort: 3000
  type: ClusterIP
```

### Worker Deployment

```yaml
# worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linktor-worker
  namespace: linktor
spec:
  replicas: 5
  selector:
    matchLabels:
      app: linktor-worker
  template:
    metadata:
      labels:
        app: linktor-worker
    spec:
      containers:
        - name: worker
          image: linktor/worker:latest
          envFrom:
            - configMapRef:
                name: linktor-config
            - secretRef:
                name: linktor-secrets
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "1Gi"
              cpu: "500m"
```

### Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: linktor-ingress
  namespace: linktor
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  tls:
    - hosts:
        - linktor.example.com
        - api.linktor.example.com
      secretName: linktor-tls
  rules:
    - host: linktor.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: linktor-admin
                port:
                  number: 3000
    - host: api.linktor.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: linktor-api
                port:
                  number: 8080
```

## Auto-Scaling

### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: linktor-api-hpa
  namespace: linktor
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: linktor-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: linktor-worker-hpa
  namespace: linktor
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: linktor-worker
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### Vertical Pod Autoscaler

```yaml
# vpa.yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: linktor-api-vpa
  namespace: linktor
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: linktor-api
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
      - containerName: api
        minAllowed:
          cpu: "250m"
          memory: "512Mi"
        maxAllowed:
          cpu: "4"
          memory: "8Gi"
```

## High Availability

### Pod Disruption Budget

```yaml
# pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: linktor-api-pdb
  namespace: linktor
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: linktor-api
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: linktor-worker-pdb
  namespace: linktor
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: linktor-worker
```

### Pod Anti-Affinity

```yaml
# api-deployment.yaml (updated)
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - linktor-api
                topologyKey: kubernetes.io/hostname
```

### Multi-Zone Deployment

```yaml
# api-deployment.yaml (updated)
spec:
  template:
    spec:
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app: linktor-api
```

## Database Configuration

### PostgreSQL with Replication

Using the Bitnami PostgreSQL Helm chart:

```yaml
# values.yaml (postgresql section)
postgresql:
  enabled: true
  architecture: replication
  auth:
    postgresPassword: your-password
    replicationPassword: replication-password
    database: linktor
  primary:
    persistence:
      size: 100Gi
      storageClass: fast-ssd
    resources:
      requests:
        memory: 2Gi
        cpu: 1000m
      limits:
        memory: 8Gi
        cpu: 4000m
    extendedConfiguration: |
      shared_buffers = 2GB
      effective_cache_size = 6GB
      maintenance_work_mem = 512MB
      work_mem = 128MB
      max_connections = 500
  readReplicas:
    replicaCount: 2
    persistence:
      size: 100Gi
    resources:
      requests:
        memory: 1Gi
        cpu: 500m
```

### Redis Sentinel

```yaml
# values.yaml (redis section)
redis:
  enabled: true
  architecture: replication
  auth:
    password: your-redis-password
  sentinel:
    enabled: true
    masterSet: linktor-master
    quorum: 2
  master:
    persistence:
      size: 10Gi
  replica:
    replicaCount: 2
    persistence:
      size: 10Gi
```

## Monitoring

### ServiceMonitor for Prometheus

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: linktor-api
  namespace: linktor
spec:
  selector:
    matchLabels:
      app: linktor-api
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

### Grafana Dashboards

```yaml
# grafana-dashboard-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: linktor-grafana-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  linktor-dashboard.json: |
    {
      "title": "Linktor Overview",
      "panels": [
        {
          "title": "API Requests/sec",
          "type": "graph",
          "targets": [
            {
              "expr": "rate(linktor_http_requests_total[5m])"
            }
          ]
        }
      ]
    }
```

## Backup with Velero

### Install Velero

```bash
velero install \
  --provider aws \
  --bucket linktor-backups \
  --secret-file ./credentials-velero \
  --use-volume-snapshots=true \
  --backup-location-config region=us-east-1
```

### Schedule Backups

```yaml
# backup-schedule.yaml
apiVersion: velero.io/v1
kind: Schedule
metadata:
  name: linktor-daily-backup
  namespace: velero
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  template:
    includedNamespaces:
      - linktor
    storageLocation: default
    ttl: 720h  # 30 days
```

### Restore

```bash
# List backups
velero backup get

# Restore from backup
velero restore create --from-backup linktor-daily-backup-20240115
```

## Troubleshooting

### Check Pod Status

```bash
# List pods
kubectl get pods -n linktor

# Describe failing pod
kubectl describe pod <pod-name> -n linktor

# Check logs
kubectl logs <pod-name> -n linktor
kubectl logs <pod-name> -n linktor --previous  # Previous container logs
```

### Check Events

```bash
kubectl get events -n linktor --sort-by='.lastTimestamp'
```

### Debug Networking

```bash
# Test DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup linktor-api

# Test connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- curl http://linktor-api:8080/health
```

### Resource Issues

```bash
# Check resource usage
kubectl top pods -n linktor
kubectl top nodes

# Check PVC status
kubectl get pvc -n linktor
```

## Upgrading

### Helm Upgrade

```bash
# Update values if needed
helm upgrade linktor linktor/linktor \
  --namespace linktor \
  --values values.yaml

# Or upgrade to specific version
helm upgrade linktor linktor/linktor \
  --namespace linktor \
  --version 1.2.0 \
  --values values.yaml
```

### Rolling Update

```bash
# Update image
kubectl set image deployment/linktor-api api=linktor/api:v1.2.0 -n linktor

# Watch rollout
kubectl rollout status deployment/linktor-api -n linktor

# Rollback if needed
kubectl rollout undo deployment/linktor-api -n linktor
```

## Security

### Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: linktor-api-policy
  namespace: linktor
spec:
  podSelector:
    matchLabels:
      app: linktor-api
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: linktor-postgresql
      ports:
        - port: 5432
    - to:
        - podSelector:
            matchLabels:
              app: linktor-redis
      ports:
        - port: 6379
```

### Pod Security Standards

```yaml
# api-deployment.yaml (updated)
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
        - name: api
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
```

## Next Steps

- [Docker Deployment](/self-hosting/docker) - Simpler deployment option
- [API Reference](/api/overview) - Configure integrations
- [Channels](/channels/overview) - Set up messaging channels
