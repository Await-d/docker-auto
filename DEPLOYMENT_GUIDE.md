# Docker Auto-Update System - Production Deployment Guide

This guide covers production deployment of the Docker Auto-Update System, from infrastructure setup to monitoring and maintenance.

## Table of Contents

1. [Production Architecture](#production-architecture)
2. [Infrastructure Requirements](#infrastructure-requirements)
3. [Deployment Options](#deployment-options)
4. [Container Orchestration](#container-orchestration)
5. [Load Balancing and High Availability](#load-balancing-and-high-availability)
6. [SSL/TLS Configuration](#ssltls-configuration)
7. [Database Setup](#database-setup)
8. [Security Hardening](#security-hardening)
9. [Monitoring and Logging](#monitoring-and-logging)
10. [Backup and Disaster Recovery](#backup-and-disaster-recovery)
11. [CI/CD Pipeline](#cicd-pipeline)
12. [Performance Optimization](#performance-optimization)
13. [Troubleshooting](#troubleshooting)

## Production Architecture

### Recommended Architecture

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Nginx/HAProxy]
    end

    subgraph "Application Tier"
        API1[Backend Instance 1]
        API2[Backend Instance 2]
        API3[Backend Instance 3]
    end

    subgraph "Frontend Tier"
        FE1[Frontend Instance 1]
        FE2[Frontend Instance 2]
    end

    subgraph "Database Tier"
        DB1[(PostgreSQL Primary)]
        DB2[(PostgreSQL Replica)]
    end

    subgraph "Cache Tier"
        REDIS1[Redis Primary]
        REDIS2[Redis Replica]
    end

    subgraph "Monitoring"
        PROM[Prometheus]
        GRAF[Grafana]
        ALERT[Alertmanager]
    end

    LB --> API1
    LB --> API2
    LB --> API3
    LB --> FE1
    LB --> FE2

    API1 --> DB1
    API2 --> DB1
    API3 --> DB1
    DB1 --> DB2

    API1 --> REDIS1
    API2 --> REDIS1
    API3 --> REDIS1
    REDIS1 --> REDIS2

    PROM --> API1
    PROM --> API2
    PROM --> API3
    GRAF --> PROM
    ALERT --> PROM
```

### Key Components

1. **Load Balancer**: Nginx or HAProxy for traffic distribution
2. **Application Instances**: Multiple backend API instances for high availability
3. **Frontend Instances**: Static file serving with CDN
4. **Database**: PostgreSQL with read replicas
5. **Cache**: Redis cluster for performance
6. **Monitoring**: Prometheus, Grafana, and Alertmanager

## Infrastructure Requirements

### Minimum Production Environment

#### Single Node Deployment
- **CPU**: 4 cores
- **RAM**: 8GB
- **Storage**: 100GB SSD
- **Network**: 1Gbps connection
- **OS**: Ubuntu 20.04 LTS or RHEL 8+

#### Multi-Node High Availability
- **Load Balancer**: 2 nodes (2 cores, 4GB RAM each)
- **Application**: 3 nodes (4 cores, 8GB RAM each)
- **Database**: 3 nodes (4 cores, 16GB RAM, 500GB SSD each)
- **Cache**: 3 nodes (2 cores, 4GB RAM each)
- **Monitoring**: 2 nodes (2 cores, 8GB RAM each)

### Cloud Provider Requirements

#### AWS
```yaml
# Minimum recommended instance types
Load Balancer: t3.medium (2 vCPU, 4GB RAM)
Application: t3.large (2 vCPU, 8GB RAM)
Database: r5.xlarge (4 vCPU, 32GB RAM)
Cache: r5.large (2 vCPU, 16GB RAM)

# Storage
Database: gp3 SSD with 3000 IOPS
Application: gp3 SSD
Backup: S3 Standard-IA
```

#### Google Cloud Platform
```yaml
# Minimum recommended machine types
Load Balancer: n1-standard-2
Application: n1-standard-4
Database: n1-highmem-4
Cache: n1-standard-2

# Storage
Database: SSD persistent disks
Application: Standard persistent disks
Backup: Cloud Storage Nearline
```

#### Azure
```yaml
# Minimum recommended VM sizes
Load Balancer: Standard_B2s
Application: Standard_D4s_v3
Database: Standard_E4s_v3
Cache: Standard_B2ms

# Storage
Database: Premium SSD
Application: Standard SSD
Backup: Azure Blob Storage Cool tier
```

### Network Requirements

#### Ports and Protocols
```yaml
# External (Internet facing)
80/tcp: HTTP (redirects to HTTPS)
443/tcp: HTTPS

# Internal (between services)
8080/tcp: Backend API
5432/tcp: PostgreSQL
6379/tcp: Redis
9090/tcp: Prometheus metrics
3000/tcp: Frontend development (development only)

# Monitoring
9093/tcp: Alertmanager
3000/tcp: Grafana
```

#### Firewall Rules
```bash
# Allow external access
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Allow internal communication (adjust IPs for your network)
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 8080 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 5432 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 6379 -j ACCEPT

# Block direct access to internal services from internet
iptables -A INPUT -p tcp --dport 8080 -j DROP
iptables -A INPUT -p tcp --dport 5432 -j DROP
iptables -A INPUT -p tcp --dport 6379 -j DROP
```

## Deployment Options

### Option 1: Docker Compose (Recommended for Small-Medium)

#### Production Docker Compose

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  # Load Balancer
  nginx:
    image: nginx:alpine
    container_name: docker-auto-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - ./nginx/logs:/var/log/nginx
    depends_on:
      - backend-1
      - backend-2
      - frontend
    restart: unless-stopped
    networks:
      - docker-auto-network

  # Backend Instances
  backend-1:
    image: docker-auto/backend:${VERSION:-latest}
    container_name: docker-auto-backend-1
    environment:
      - DATABASE_URL=postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}
      - REDIS_URL=redis://redis-primary:6379
      - JWT_SECRET=${JWT_SECRET}
      - INSTANCE_ID=backend-1
    depends_on:
      - postgres
      - redis-primary
    restart: unless-stopped
    networks:
      - docker-auto-network
    deploy:
      resources:
        limits:
          memory: 1GB
          cpus: '1.0'

  backend-2:
    image: docker-auto/backend:${VERSION:-latest}
    container_name: docker-auto-backend-2
    environment:
      - DATABASE_URL=postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}
      - REDIS_URL=redis://redis-primary:6379
      - JWT_SECRET=${JWT_SECRET}
      - INSTANCE_ID=backend-2
    depends_on:
      - postgres
      - redis-primary
    restart: unless-stopped
    networks:
      - docker-auto-network
    deploy:
      resources:
        limits:
          memory: 1GB
          cpus: '1.0'

  # Frontend
  frontend:
    image: docker-auto/frontend:${VERSION:-latest}
    container_name: docker-auto-frontend
    environment:
      - NODE_ENV=production
      - API_URL=https://${DOMAIN}/api
    restart: unless-stopped
    networks:
      - docker-auto-network
    deploy:
      resources:
        limits:
          memory: 512MB
          cpus: '0.5'

  # Database Primary
  postgres:
    image: postgres:15
    container_name: docker-auto-postgres
    environment:
      - POSTGRES_DB=${DB_NAME}
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_INITDB_ARGS=--auth-host=md5
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/postgresql.conf:/etc/postgresql/postgresql.conf
      - ./postgres/pg_hba.conf:/etc/postgresql/pg_hba.conf
      - ./postgres/init:/docker-entrypoint-initdb.d
    command: postgres -c config_file=/etc/postgresql/postgresql.conf
    restart: unless-stopped
    networks:
      - docker-auto-network
    deploy:
      resources:
        limits:
          memory: 4GB
          cpus: '2.0'

  # Redis Primary
  redis-primary:
    image: redis:7-alpine
    container_name: docker-auto-redis-primary
    command: redis-server /etc/redis/redis.conf
    volumes:
      - redis_data:/data
      - ./redis/redis.conf:/etc/redis/redis.conf:ro
    restart: unless-stopped
    networks:
      - docker-auto-network
    deploy:
      resources:
        limits:
          memory: 1GB
          cpus: '0.5'

  # Monitoring
  prometheus:
    image: prom/prometheus:latest
    container_name: docker-auto-prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'
    volumes:
      - prometheus_data:/prometheus
      - ./monitoring/prometheus:/etc/prometheus
    restart: unless-stopped
    networks:
      - docker-auto-network
    deploy:
      resources:
        limits:
          memory: 2GB
          cpus: '1.0'

  grafana:
    image: grafana/grafana:latest
    container_name: docker-auto-grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana:/etc/grafana/provisioning
    depends_on:
      - prometheus
    restart: unless-stopped
    networks:
      - docker-auto-network

  # Log Management
  loki:
    image: grafana/loki:latest
    container_name: docker-auto-loki
    volumes:
      - loki_data:/tmp/loki
      - ./monitoring/loki:/etc/loki
    command: -config.file=/etc/loki/loki.yml
    restart: unless-stopped
    networks:
      - docker-auto-network

volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: none
      device: /opt/docker-auto/data/postgres
      o: bind
  redis_data:
    driver: local
    driver_opts:
      type: none
      device: /opt/docker-auto/data/redis
      o: bind
  prometheus_data:
    driver: local
  grafana_data:
    driver: local
  loki_data:
    driver: local

networks:
  docker-auto-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

#### Deployment Script

Create `deploy.sh`:

```bash
#!/bin/bash

set -e

# Configuration
DOMAIN=${DOMAIN:-"your-domain.com"}
VERSION=${VERSION:-"latest"}
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-7}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

# Pre-deployment checks
check_requirements() {
    log "Checking deployment requirements..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
    fi

    # Check Docker Compose
    if ! docker compose version &> /dev/null; then
        error "Docker Compose V2 is not installed"
    fi

    # Check environment file
    if [[ ! -f .env.prod ]]; then
        error ".env.prod file not found"
    fi

    # Check SSL certificates
    if [[ ! -f nginx/ssl/fullchain.pem ]] || [[ ! -f nginx/ssl/privkey.pem ]]; then
        warn "SSL certificates not found. HTTPS will not be available."
    fi

    log "Requirements check passed"
}

# Backup existing deployment
backup_deployment() {
    log "Creating backup of current deployment..."

    BACKUP_DIR="/opt/docker-auto/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"

    # Backup database
    if docker ps | grep -q docker-auto-postgres; then
        log "Backing up database..."
        docker exec docker-auto-postgres pg_dump -U ${DB_USER} ${DB_NAME} > "$BACKUP_DIR/database.sql"
    fi

    # Backup configuration
    log "Backing up configuration..."
    cp -r ./nginx "$BACKUP_DIR/"
    cp -r ./monitoring "$BACKUP_DIR/"
    cp .env.prod "$BACKUP_DIR/"

    log "Backup created at $BACKUP_DIR"
}

# Deploy application
deploy() {
    log "Starting deployment..."

    # Load environment variables
    export $(cat .env.prod | xargs)

    # Pull latest images
    log "Pulling latest images..."
    docker compose -f docker-compose.prod.yml pull

    # Start services
    log "Starting services..."
    docker compose -f docker-compose.prod.yml up -d

    # Wait for services to be healthy
    log "Waiting for services to be healthy..."
    sleep 30

    # Run health checks
    check_deployment_health

    log "Deployment completed successfully"
}

# Health checks
check_deployment_health() {
    log "Running health checks..."

    # Check backend health
    for i in {1..30}; do
        if curl -f http://localhost/api/health &> /dev/null; then
            log "Backend is healthy"
            break
        fi
        if [[ $i -eq 30 ]]; then
            error "Backend health check failed"
        fi
        sleep 5
    done

    # Check database connectivity
    if ! docker exec docker-auto-backend-1 sh -c 'echo "SELECT 1;" | nc postgres 5432' &> /dev/null; then
        error "Database connectivity check failed"
    fi

    # Check Redis connectivity
    if ! docker exec docker-auto-backend-1 sh -c 'echo "PING" | nc redis-primary 6379' &> /dev/null; then
        error "Redis connectivity check failed"
    fi

    log "All health checks passed"
}

# Cleanup old backups
cleanup_backups() {
    log "Cleaning up old backups..."
    find /opt/docker-auto/backups -type d -mtime +$BACKUP_RETENTION_DAYS -exec rm -rf {} \; 2>/dev/null || true
    log "Backup cleanup completed"
}

# Main deployment flow
main() {
    log "Starting Docker Auto-Update System deployment"

    check_requirements
    backup_deployment
    deploy
    cleanup_backups

    log "Deployment completed successfully!"
    log "Access the system at: https://$DOMAIN"
    log "Grafana dashboard: https://$DOMAIN/grafana"
    log "Check logs with: docker compose -f docker-compose.prod.yml logs -f"
}

# Handle script arguments
case ${1:-deploy} in
    "deploy")
        main
        ;;
    "backup")
        backup_deployment
        ;;
    "health")
        check_deployment_health
        ;;
    "cleanup")
        cleanup_backups
        ;;
    *)
        echo "Usage: $0 [deploy|backup|health|cleanup]"
        exit 1
        ;;
esac
```

### Option 2: Kubernetes Deployment

#### Kubernetes Manifests

Create `k8s/namespace.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: docker-auto
  labels:
    name: docker-auto
    app.kubernetes.io/name: docker-auto
    app.kubernetes.io/version: "2.0.0"
```

Create `k8s/configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: docker-auto-config
  namespace: docker-auto
data:
  DATABASE_HOST: "postgres"
  DATABASE_PORT: "5432"
  DATABASE_NAME: "dockerauto"
  REDIS_HOST: "redis"
  REDIS_PORT: "6379"
  LOG_LEVEL: "info"
  PROMETHEUS_ENABLED: "true"
  METRICS_PORT: "9090"
---
apiVersion: v1
kind: Secret
metadata:
  name: docker-auto-secrets
  namespace: docker-auto
type: Opaque
data:
  DATABASE_USER: ZG9ja2VyYXV0bw== # base64 encoded 'dockerauto'
  DATABASE_PASSWORD: c2VjdXJlX3Bhc3N3b3Jk # base64 encoded 'secure_password'
  JWT_SECRET: eW91ci1zZWN1cmUtand0LXNlY3JldC1rZXk= # base64 encoded jwt secret
```

Create `k8s/backend-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: docker-auto-backend
  namespace: docker-auto
  labels:
    app: docker-auto-backend
    version: "2.0.0"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: docker-auto-backend
  template:
    metadata:
      labels:
        app: docker-auto-backend
        version: "2.0.0"
    spec:
      containers:
      - name: backend
        image: docker-auto/backend:2.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: DATABASE_URL
          value: "postgresql://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)"
        - name: REDIS_URL
          value: "redis://$(REDIS_HOST):$(REDIS_PORT)"
        envFrom:
        - configMapRef:
            name: docker-auto-config
        - secretRef:
            name: docker-auto-secrets
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: docker-socket
          mountPath: /var/run/docker.sock
          readOnly: true
      volumes:
      - name: docker-socket
        hostPath:
          path: /var/run/docker.sock
          type: Socket
---
apiVersion: v1
kind: Service
metadata:
  name: docker-auto-backend
  namespace: docker-auto
  labels:
    app: docker-auto-backend
spec:
  selector:
    app: docker-auto-backend
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
```

Create `k8s/frontend-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: docker-auto-frontend
  namespace: docker-auto
  labels:
    app: docker-auto-frontend
    version: "2.0.0"
spec:
  replicas: 2
  selector:
    matchLabels:
      app: docker-auto-frontend
  template:
    metadata:
      labels:
        app: docker-auto-frontend
        version: "2.0.0"
    spec:
      containers:
      - name: frontend
        image: docker-auto/frontend:2.0.0
        ports:
        - containerPort: 80
          name: http
        env:
        - name: NODE_ENV
          value: "production"
        - name: API_URL
          value: "https://your-domain.com/api"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: docker-auto-frontend
  namespace: docker-auto
  labels:
    app: docker-auto-frontend
spec:
  selector:
    app: docker-auto-frontend
  ports:
  - name: http
    port: 80
    targetPort: 80
```

Create `k8s/postgres-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: docker-auto
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        ports:
        - containerPort: 5432
          name: postgres
        env:
        - name: POSTGRES_DB
          valueFrom:
            configMapKeyRef:
              name: docker-auto-config
              key: DATABASE_NAME
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: docker-auto-secrets
              key: DATABASE_USER
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: docker-auto-secrets
              key: DATABASE_PASSWORD
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - $(POSTGRES_USER)
            - -d
            - $(POSTGRES_DB)
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - $(POSTGRES_USER)
            - -d
            - $(POSTGRES_DB)
          initialDelaySeconds: 5
          periodSeconds: 5
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
      storageClassName: fast-ssd
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: docker-auto
  labels:
    app: postgres
spec:
  selector:
    app: postgres
  ports:
  - name: postgres
    port: 5432
    targetPort: 5432
  clusterIP: None
```

Create `k8s/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: docker-auto-ingress
  namespace: docker-auto
  annotations:
    kubernetes.io/ingress.class: "nginx"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/use-regex: "true"
    nginx.ingress.kubernetes.io/rewrite-target: /$1
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "300"
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: docker-auto-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /api/(.*)
        pathType: Prefix
        backend:
          service:
            name: docker-auto-backend
            port:
              number: 8080
      - path: /grafana/(.*)
        pathType: Prefix
        backend:
          service:
            name: grafana
            port:
              number: 3000
      - path: /(.*)
        pathType: Prefix
        backend:
          service:
            name: docker-auto-frontend
            port:
              number: 80
```

#### Kubernetes Deployment Script

Create `k8s/deploy.sh`:

```bash
#!/bin/bash

set -e

NAMESPACE="docker-auto"
VERSION=${VERSION:-"2.0.0"}

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Create namespace
log "Creating namespace..."
kubectl apply -f namespace.yaml

# Apply configurations
log "Applying configurations..."
kubectl apply -f configmap.yaml

# Deploy database
log "Deploying PostgreSQL..."
kubectl apply -f postgres-deployment.yaml

# Deploy Redis
log "Deploying Redis..."
kubectl apply -f redis-deployment.yaml

# Wait for database to be ready
log "Waiting for database to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s

# Deploy backend
log "Deploying backend..."
kubectl apply -f backend-deployment.yaml

# Deploy frontend
log "Deploying frontend..."
kubectl apply -f frontend-deployment.yaml

# Deploy ingress
log "Deploying ingress..."
kubectl apply -f ingress.yaml

# Wait for all deployments to be ready
log "Waiting for deployments to be ready..."
kubectl wait --for=condition=available deployment --all -n $NAMESPACE --timeout=300s

# Check deployment status
log "Checking deployment status..."
kubectl get all -n $NAMESPACE

log "Deployment completed successfully!"
log "Access the application at: https://your-domain.com"
```

## Load Balancing and High Availability

### Nginx Configuration

Create `nginx/nginx.conf`:

```nginx
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for" '
                    'rt=$request_time uct="$upstream_connect_time" '
                    'uht="$upstream_header_time" urt="$upstream_response_time"';

    access_log /var/log/nginx/access.log main;

    # Performance settings
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    client_max_body_size 100M;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/json
        application/javascript
        application/xml+rss
        application/atom+xml
        image/svg+xml;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=1r/s;

    # Upstream backend servers
    upstream backend {
        least_conn;
        server backend-1:8080 max_fails=3 fail_timeout=30s;
        server backend-2:8080 max_fails=3 fail_timeout=30s;
        server backend-3:8080 max_fails=3 fail_timeout=30s backup;
    }

    # Upstream frontend servers
    upstream frontend {
        least_conn;
        server frontend-1:80 max_fails=3 fail_timeout=30s;
        server frontend-2:80 max_fails=3 fail_timeout=30s;
    }

    # HTTP to HTTPS redirect
    server {
        listen 80 default_server;
        server_name _;
        return 301 https://$host$request_uri;
    }

    # HTTPS server
    server {
        listen 443 ssl http2 default_server;
        server_name your-domain.com;

        # SSL configuration
        ssl_certificate /etc/nginx/ssl/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/privkey.pem;
        ssl_session_timeout 1d;
        ssl_session_cache shared:SSL:50m;
        ssl_session_tickets off;

        # Modern configuration
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        # HSTS
        add_header Strict-Transport-Security "max-age=63072000" always;

        # Security headers
        add_header X-Frame-Options DENY always;
        add_header X-Content-Type-Options nosniff always;
        add_header X-XSS-Protection "1; mode=block" always;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;

        # API routes
        location /api/ {
            limit_req zone=api burst=20 nodelay;

            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
            proxy_buffering off;

            # Health check endpoint
            location /api/health {
                proxy_pass http://backend;
                access_log off;
            }
        }

        # Login endpoint with stricter rate limiting
        location /api/auth/login {
            limit_req zone=login burst=5 nodelay;
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # WebSocket support
        location /api/ws {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 86400;
        }

        # Frontend routes
        location / {
            proxy_pass http://frontend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # Cache static assets
            location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
                proxy_pass http://frontend;
                expires 1y;
                add_header Cache-Control "public, immutable";
            }
        }

        # Grafana (monitoring)
        location /grafana/ {
            proxy_pass http://grafana:3000/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Status page
        location /nginx_status {
            stub_status on;
            access_log off;
            allow 127.0.0.1;
            allow 10.0.0.0/8;
            deny all;
        }
    }

    # Health check endpoint for load balancer
    server {
        listen 8081;
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

## SSL/TLS Configuration

### Automatic SSL with Let's Encrypt

Create `scripts/setup-ssl.sh`:

```bash
#!/bin/bash

DOMAIN=${1:-"your-domain.com"}
EMAIL=${2:-"admin@your-domain.com"}

# Install Certbot
if ! command -v certbot &> /dev/null; then
    echo "Installing Certbot..."
    apt-get update
    apt-get install -y certbot python3-certbot-nginx
fi

# Stop nginx temporarily
docker compose -f docker-compose.prod.yml stop nginx

# Generate certificate
echo "Generating SSL certificate for $DOMAIN..."
certbot certonly --standalone \
    --non-interactive \
    --agree-tos \
    --email $EMAIL \
    --domains $DOMAIN \
    --cert-path /opt/docker-auto/nginx/ssl \
    --key-path /opt/docker-auto/nginx/ssl

# Create renewal script
cat > /etc/cron.daily/certbot-renewal << 'EOF'
#!/bin/bash
certbot renew --quiet --pre-hook "docker compose -f /opt/docker-auto/docker-compose.prod.yml stop nginx" --post-hook "docker compose -f /opt/docker-auto/docker-compose.prod.yml start nginx"
EOF

chmod +x /etc/cron.daily/certbot-renewal

# Start nginx
docker compose -f docker-compose.prod.yml start nginx

echo "SSL certificate configured successfully"
```

## Database Setup

### PostgreSQL Production Configuration

Create `postgres/postgresql.conf`:

```ini
# Connection settings
listen_addresses = '*'
port = 5432
max_connections = 200
superuser_reserved_connections = 3

# Memory settings
shared_buffers = 2GB                    # 25% of system RAM
effective_cache_size = 6GB              # 75% of system RAM
work_mem = 16MB
maintenance_work_mem = 512MB
dynamic_shared_memory_type = posix

# WAL settings
wal_level = replica
max_wal_senders = 3
max_replication_slots = 3
wal_buffers = 16MB
checkpoint_completion_target = 0.9
wal_compression = on

# Query planner
random_page_cost = 1.1                  # SSD storage
effective_io_concurrency = 200

# Logging
log_destination = 'stderr'
logging_collector = on
log_directory = 'pg_log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_min_duration_statement = 1000       # Log slow queries
log_line_prefix = '%t [%p-%l] %q%u@%d '
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on

# Performance monitoring
track_activities = on
track_counts = on
track_io_timing = on
track_functions = pl

# Autovacuum
autovacuum = on
log_autovacuum_min_duration = 0
autovacuum_max_workers = 3
autovacuum_naptime = 1min
```

### Database Backup and Replication

Create `scripts/backup-database.sh`:

```bash
#!/bin/bash

set -e

# Configuration
DB_NAME=${DB_NAME:-"dockerauto"}
DB_USER=${DB_USER:-"dockerauto"}
BACKUP_DIR="/opt/docker-auto/backups/database"
RETENTION_DAYS=30

# Create backup directory
mkdir -p $BACKUP_DIR

# Generate backup filename
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/db_backup_$TIMESTAMP.sql.gz"

echo "Creating database backup: $BACKUP_FILE"

# Create backup
docker exec docker-auto-postgres pg_dump \
    -U $DB_USER \
    -h localhost \
    -d $DB_NAME \
    --verbose \
    --no-acl \
    --no-owner \
    | gzip > $BACKUP_FILE

# Verify backup
if [[ -f $BACKUP_FILE ]] && [[ -s $BACKUP_FILE ]]; then
    echo "Backup created successfully: $BACKUP_FILE"

    # Upload to S3 (optional)
    if [[ -n $AWS_S3_BUCKET ]]; then
        aws s3 cp $BACKUP_FILE s3://$AWS_S3_BUCKET/database-backups/
        echo "Backup uploaded to S3"
    fi
else
    echo "Backup failed!"
    exit 1
fi

# Cleanup old backups
find $BACKUP_DIR -name "db_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete
echo "Old backups cleaned up"
```

## Security Hardening

### System Security

Create `scripts/harden-system.sh`:

```bash
#!/bin/bash

echo "Hardening system security..."

# Update system
apt-get update && apt-get upgrade -y

# Install security packages
apt-get install -y fail2ban ufw

# Configure UFW firewall
ufw --force reset
ufw default deny incoming
ufw default allow outgoing

# Allow SSH (change port if needed)
ufw allow 22/tcp

# Allow HTTP/HTTPS
ufw allow 80/tcp
ufw allow 443/tcp

# Enable firewall
ufw --force enable

# Configure Fail2Ban
cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[nginx-http-auth]
enabled = true
filter = nginx-http-auth
logpath = /var/log/nginx/error.log

[nginx-req-limit]
enabled = true
filter = nginx-req-limit
logpath = /var/log/nginx/error.log

[sshd]
enabled = true
port = ssh
logpath = %(sshd_log)s
backend = %(sshd_backend)s
EOF

# Restart services
systemctl restart fail2ban

# Secure Docker daemon
cat > /etc/docker/daemon.json << 'EOF'
{
  "live-restore": true,
  "userland-proxy": false,
  "no-new-privileges": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2"
}
EOF

systemctl restart docker

# Set file permissions
chmod 600 /opt/docker-auto/.env.prod
chown root:root /opt/docker-auto/.env.prod

echo "System hardening completed"
```

### Application Security

Create `security/security-headers.conf`:

```nginx
# Security headers for Nginx
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "no-referrer-when-downgrade" always;
add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
```

## Monitoring and Logging

### Prometheus Configuration

Create `monitoring/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # Docker Auto API
  - job_name: 'docker-auto-api'
    static_configs:
      - targets: ['backend-1:9090', 'backend-2:9090', 'backend-3:9090']
    scrape_interval: 15s
    metrics_path: /metrics

  # Node Exporter
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  # PostgreSQL Exporter
  - job_name: 'postgres-exporter'
    static_configs:
      - targets: ['postgres-exporter:9187']

  # Redis Exporter
  - job_name: 'redis-exporter'
    static_configs:
      - targets: ['redis-exporter:9121']

  # Nginx Exporter
  - job_name: 'nginx-exporter'
    static_configs:
      - targets: ['nginx-exporter:9113']

  # Docker Daemon
  - job_name: 'docker-daemon'
    static_configs:
      - targets: ['localhost:9323']
```

### Alert Rules

Create `monitoring/prometheus/rules/alerts.yml`:

```yaml
groups:
- name: docker-auto-alerts
  rules:
  # High CPU usage
  - alert: HighCPUUsage
    expr: 100 - (avg(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High CPU usage detected"
      description: "CPU usage is above 80% for more than 5 minutes"

  # High memory usage
  - alert: HighMemoryUsage
    expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100 > 85
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage detected"
      description: "Memory usage is above 85% for more than 5 minutes"

  # Database connection issues
  - alert: DatabaseDown
    expr: up{job="postgres-exporter"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Database is down"
      description: "PostgreSQL database is not responding"

  # API endpoint down
  - alert: APIDown
    expr: up{job="docker-auto-api"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "API is down"
      description: "Docker Auto API is not responding"

  # High error rate
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100 > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "API error rate is above 10% for more than 5 minutes"
```

## Backup and Disaster Recovery

### Comprehensive Backup Strategy

Create `scripts/full-backup.sh`:

```bash
#!/bin/bash

set -e

BACKUP_ROOT="/opt/docker-auto/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_ROOT/full_backup_$TIMESTAMP"

echo "Starting full backup: $BACKUP_DIR"

mkdir -p "$BACKUP_DIR"

# Backup database
echo "Backing up database..."
docker exec docker-auto-postgres pg_dump -U $DB_USER $DB_NAME | gzip > "$BACKUP_DIR/database.sql.gz"

# Backup Redis data
echo "Backing up Redis..."
docker exec docker-auto-redis-primary redis-cli SAVE
docker cp docker-auto-redis-primary:/data/dump.rdb "$BACKUP_DIR/"

# Backup configuration files
echo "Backing up configuration..."
cp -r /opt/docker-auto/nginx "$BACKUP_DIR/"
cp -r /opt/docker-auto/monitoring "$BACKUP_DIR/"
cp /opt/docker-auto/.env.prod "$BACKUP_DIR/"
cp /opt/docker-auto/docker-compose.prod.yml "$BACKUP_DIR/"

# Backup Docker volumes
echo "Backing up Docker volumes..."
mkdir -p "$BACKUP_DIR/volumes"
docker run --rm -v postgres_data:/source -v "$BACKUP_DIR/volumes":/backup alpine tar czf /backup/postgres_data.tar.gz -C /source .

# Create manifest
cat > "$BACKUP_DIR/manifest.json" << EOF
{
  "backup_date": "$TIMESTAMP",
  "version": "2.0.0",
  "components": [
    "database",
    "redis",
    "configuration",
    "volumes"
  ],
  "size": "$(du -sh $BACKUP_DIR | cut -f1)"
}
EOF

# Upload to cloud storage (optional)
if [[ -n $AWS_S3_BUCKET ]]; then
    echo "Uploading to S3..."
    tar czf "$BACKUP_ROOT/full_backup_$TIMESTAMP.tar.gz" -C "$BACKUP_ROOT" "full_backup_$TIMESTAMP"
    aws s3 cp "$BACKUP_ROOT/full_backup_$TIMESTAMP.tar.gz" "s3://$AWS_S3_BUCKET/backups/"
    rm "$BACKUP_ROOT/full_backup_$TIMESTAMP.tar.gz"
fi

echo "Full backup completed: $BACKUP_DIR"
```

### Disaster Recovery Plan

Create `scripts/disaster-recovery.sh`:

```bash
#!/bin/bash

set -e

BACKUP_FILE=${1}
RESTORE_DIR="/opt/docker-auto-restore"

if [[ -z $BACKUP_FILE ]]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

echo "Starting disaster recovery from: $BACKUP_FILE"

# Stop current services
echo "Stopping current services..."
docker compose -f docker-compose.prod.yml down

# Extract backup
echo "Extracting backup..."
mkdir -p "$RESTORE_DIR"
tar xzf "$BACKUP_FILE" -C "$RESTORE_DIR"

BACKUP_NAME=$(basename "$BACKUP_FILE" .tar.gz)
BACKUP_PATH="$RESTORE_DIR/$BACKUP_NAME"

# Restore configuration
echo "Restoring configuration..."
cp -r "$BACKUP_PATH/nginx" /opt/docker-auto/
cp -r "$BACKUP_PATH/monitoring" /opt/docker-auto/
cp "$BACKUP_PATH/.env.prod" /opt/docker-auto/
cp "$BACKUP_PATH/docker-compose.prod.yml" /opt/docker-auto/

# Restore database
echo "Restoring database..."
docker compose -f docker-compose.prod.yml up -d postgres
sleep 30
zcat "$BACKUP_PATH/database.sql.gz" | docker exec -i docker-auto-postgres psql -U $DB_USER -d $DB_NAME

# Restore Redis
echo "Restoring Redis..."
docker cp "$BACKUP_PATH/dump.rdb" docker-auto-redis-primary:/data/
docker compose -f docker-compose.prod.yml restart redis-primary

# Start all services
echo "Starting all services..."
docker compose -f docker-compose.prod.yml up -d

# Verify recovery
echo "Verifying recovery..."
sleep 30
if curl -f http://localhost/api/health; then
    echo "Disaster recovery completed successfully"
else
    echo "Recovery verification failed"
    exit 1
fi

# Cleanup
rm -rf "$RESTORE_DIR"

echo "Disaster recovery completed"
```

## CI/CD Pipeline

### GitHub Actions Workflow

Create `.github/workflows/deploy.yml`:

```yaml
name: Deploy to Production

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      environment:
        description: 'Deployment environment'
        required: true
        default: 'production'
        type: choice
        options:
          - production
          - staging

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
    - name: Checkout repository
      uses: actions/checkout@v4

    - name: Log in to Container Registry
      uses: docker/login-action@v3
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
        tags: |
          type=ref,event=tag
          type=ref,event=branch
          type=sha

    - name: Build and push backend image
      uses: docker/build-push-action@v5
      with:
        context: ./backend
        push: true
        tags: ${{ steps.meta.outputs.tags }}-backend
        labels: ${{ steps.meta.outputs.labels }}

    - name: Build and push frontend image
      uses: docker/build-push-action@v5
      with:
        context: ./frontend
        push: true
        tags: ${{ steps.meta.outputs.tags }}-frontend
        labels: ${{ steps.meta.outputs.labels }}

  deploy:
    runs-on: ubuntu-latest
    needs: build
    if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/')

    steps:
    - name: Deploy to production
      uses: appleboy/ssh-action@v1.0.0
      with:
        host: ${{ secrets.PROD_HOST }}
        username: ${{ secrets.PROD_USER }}
        key: ${{ secrets.PROD_SSH_KEY }}
        script: |
          cd /opt/docker-auto
          export VERSION=${{ github.ref_name }}
          ./deploy.sh
```

## Performance Optimization

### Application Performance

1. **Database Connection Pooling**
```yaml
database:
  max_open_connections: 25
  max_idle_connections: 25
  connection_max_lifetime: 5m
```

2. **Redis Caching Strategy**
```yaml
redis:
  maxmemory: 1gb
  maxmemory-policy: allkeys-lru
  save: "900 1 300 10 60 10000"
```

3. **API Rate Limiting**
```yaml
rate_limiting:
  requests_per_minute: 100
  burst: 20
```

### Infrastructure Optimization

1. **Nginx Tuning**
```nginx
worker_processes auto;
worker_connections 2048;
keepalive_timeout 65;
client_max_body_size 100M;
```

2. **Docker Resource Limits**
```yaml
deploy:
  resources:
    limits:
      memory: 1GB
      cpus: '1.0'
    reservations:
      memory: 512MB
      cpus: '0.5'
```

## Troubleshooting

### Common Deployment Issues

1. **Service Won't Start**
```bash
# Check logs
docker compose -f docker-compose.prod.yml logs service_name

# Check resource usage
docker stats

# Verify configuration
docker compose -f docker-compose.prod.yml config
```

2. **Database Connection Issues**
```bash
# Test database connectivity
docker exec docker-auto-backend-1 nc -zv postgres 5432

# Check database logs
docker compose -f docker-compose.prod.yml logs postgres

# Verify environment variables
docker exec docker-auto-backend-1 env | grep DATABASE
```

3. **SSL Certificate Issues**
```bash
# Check certificate validity
openssl s_client -connect your-domain.com:443 -servername your-domain.com

# Renew certificate manually
certbot renew --dry-run
```

### Performance Issues

1. **High Memory Usage**
```bash
# Monitor memory usage
docker stats --no-stream

# Check for memory leaks
docker exec docker-auto-backend-1 pprof -top -mem heap.prof
```

2. **High CPU Usage**
```bash
# Profile CPU usage
docker exec docker-auto-backend-1 pprof -top -cpu profile.prof

# Check system load
htop
```

### Recovery Procedures

1. **Service Recovery**
```bash
# Restart specific service
docker compose -f docker-compose.prod.yml restart service_name

# Full system restart
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d
```

2. **Database Recovery**
```bash
# Restore from backup
./scripts/disaster-recovery.sh backup_file.tar.gz

# Check database integrity
docker exec docker-auto-postgres pg_dump -U dockerauto dockerauto > /dev/null
```

---

**Last Updated**: September 16, 2024
**Version**: 2.0.0

This deployment guide ensures a robust, scalable, and secure production environment for the Docker Auto-Update System. Follow the procedures carefully and customize configurations based on your specific requirements.