#!/bin/bash

# Docker Auto Update System - Production Deployment Script
# This script handles complete production deployment with safety checks

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DOCKER_COMPOSE_FILE="$PROJECT_DIR/docker-compose.production.yml"
ENV_FILE="$PROJECT_DIR/.env.production"
BACKUP_DIR="$PROJECT_DIR/backups/production"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    # Check if Docker is installed and running
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed or not in PATH"
    fi

    # Check if Docker Compose is available
    if ! docker compose version &> /dev/null; then
        error "Docker Compose is not available"
    fi

    # Check if running as non-root user with Docker access
    if ! docker ps &> /dev/null; then
        error "Cannot access Docker. Make sure your user is in the docker group"
    fi

    success "Prerequisites check passed"
}

# Validate environment configuration
validate_environment() {
    log "Validating environment configuration..."

    if [[ ! -f "$ENV_FILE" ]]; then
        warning "Production environment file not found: $ENV_FILE"
        log "Creating from example file..."
        cp "$PROJECT_DIR/.env.production.example" "$ENV_FILE"
        error "Please edit $ENV_FILE with your production settings before running this script"
    fi

    # Source environment file
    set -a
    source "$ENV_FILE"
    set +a

    # Check critical environment variables
    local required_vars=(
        "JWT_SECRET"
        "POSTGRES_PASSWORD"
        "DOMAIN"
    )

    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]] || [[ "${!var}" == *"change-me"* ]]; then
            error "Required environment variable $var is not set or contains default value"
        fi
    done

    success "Environment validation passed"
}

# Create required directories
create_directories() {
    log "Creating required directories..."

    local dirs=(
        "$PROJECT_DIR/data/production"
        "$PROJECT_DIR/logs/production"
        "$PROJECT_DIR/backups/production"
        "$PROJECT_DIR/config/production"
        "$PROJECT_DIR/ssl"
        "$PROJECT_DIR/nginx/conf.d"
    )

    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        log "Created directory: $dir"
    done

    success "Required directories created"
}

# Build production images
build_images() {
    log "Building production Docker images..."

    cd "$PROJECT_DIR"

    # Build with no cache for production
    docker compose -f "$DOCKER_COMPOSE_FILE" build \
        --no-cache \
        --pull \
        --progress=plain \
        docker-auto

    success "Production images built successfully"
}

# Create backup before deployment
create_backup() {
    log "Creating pre-deployment backup..."

    local backup_file="$BACKUP_DIR/pre-deployment-$(date +%Y%m%d-%H%M%S).tar.gz"

    # Backup current data and configuration
    if [[ -d "$PROJECT_DIR/data/production" ]]; then
        tar -czf "$backup_file" \
            -C "$PROJECT_DIR" \
            data/production \
            config/production \
            .env.production 2>/dev/null || true

        success "Backup created: $backup_file"
    else
        log "No existing data to backup (first deployment)"
    fi
}

# Deploy services
deploy_services() {
    log "Deploying production services..."

    cd "$PROJECT_DIR"

    # Pull latest base images
    docker compose -f "$DOCKER_COMPOSE_FILE" pull postgres redis nginx

    # Start services with proper ordering
    docker compose -f "$DOCKER_COMPOSE_FILE" up -d \
        --remove-orphans \
        --force-recreate

    success "Services deployed successfully"
}

# Wait for services to be healthy
wait_for_health() {
    log "Waiting for services to become healthy..."

    local max_attempts=30
    local attempt=0

    while [[ $attempt -lt $max_attempts ]]; do
        if docker compose -f "$DOCKER_COMPOSE_FILE" ps --filter status=running | grep -q "healthy"; then
            success "Services are healthy"
            return 0
        fi

        attempt=$((attempt + 1))
        log "Attempt $attempt/$max_attempts - Waiting for services to be healthy..."
        sleep 10
    done

    error "Services did not become healthy within expected time"
}

# Perform health checks
perform_health_checks() {
    log "Performing application health checks..."

    # Check main application
    local app_url="http://localhost:8080/api/health"
    if curl -f -s "$app_url" > /dev/null; then
        success "Application health check passed"
    else
        error "Application health check failed"
    fi

    # Check database connection (if PostgreSQL is used)
    if docker compose -f "$DOCKER_COMPOSE_FILE" exec -T postgres pg_isready -U docker_auto_user -d docker_auto_prod > /dev/null; then
        success "Database health check passed"
    else
        warning "Database health check failed or not using PostgreSQL"
    fi

    # Check Redis
    if docker compose -f "$DOCKER_COMPOSE_FILE" exec -T redis redis-cli ping > /dev/null; then
        success "Redis health check passed"
    else
        warning "Redis health check failed"
    fi
}

# Setup monitoring
setup_monitoring() {
    log "Setting up monitoring..."

    # Create monitoring configuration
    cat > "$PROJECT_DIR/config/production/monitoring.yml" << EOF
monitoring:
  enabled: true
  metrics_port: 9090
  health_check_interval: 30
  alerts:
    - name: high_cpu
      threshold: 80
      duration: 5m
    - name: high_memory
      threshold: 90
      duration: 2m
    - name: disk_space
      threshold: 85
      duration: 1m
EOF

    success "Monitoring configuration created"
}

# Setup SSL certificates (placeholder)
setup_ssl() {
    log "Setting up SSL certificates..."

    if [[ ! -f "$PROJECT_DIR/ssl/cert.pem" ]] || [[ ! -f "$PROJECT_DIR/ssl/private.key" ]]; then
        warning "SSL certificates not found"
        log "Generating self-signed certificates for testing (NOT FOR PRODUCTION USE)"

        # Create self-signed certificate (development only)
        openssl req -x509 -newkey rsa:4096 -keyout "$PROJECT_DIR/ssl/private.key" \
            -out "$PROJECT_DIR/ssl/cert.pem" -days 365 -nodes \
            -subj "/CN=localhost" 2>/dev/null || {
            warning "Could not generate SSL certificates. Please provide valid certificates."
        }
    fi

    success "SSL setup completed"
}

# Setup nginx configuration
setup_nginx() {
    log "Setting up Nginx configuration..."

    cat > "$PROJECT_DIR/nginx/nginx.conf" << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log notice;
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
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    # Performance
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Security
    server_tokens off;
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_min_length 10240;
    gzip_proxied expired no-cache no-store private must-revalidate auth;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

    include /etc/nginx/conf.d/*.conf;
}
EOF

    # Create application configuration
    cat > "$PROJECT_DIR/nginx/conf.d/docker-auto.conf" << 'EOF'
server {
    listen 80;
    server_name localhost;

    # Rate limiting
    limit_req zone=api burst=20 nodelay;

    # Proxy settings
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # Health check endpoint
    location /health {
        proxy_pass http://docker-auto:8080/api/health;
    }

    # API endpoints
    location /api/ {
        proxy_pass http://docker-auto:8080;
        proxy_read_timeout 86400;
    }

    # WebSocket endpoint
    location /api/ws {
        proxy_pass http://docker-auto:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }

    # Static files
    location / {
        proxy_pass http://docker-auto:8080;
    }
}
EOF

    success "Nginx configuration created"
}

# Display deployment summary
display_summary() {
    log "Deployment Summary"
    echo "===================="

    # Show running services
    docker compose -f "$DOCKER_COMPOSE_FILE" ps

    echo ""
    log "Application URLs:"
    echo "  - Main application: http://localhost:8080"
    echo "  - Health check: http://localhost:8080/api/health"
    echo "  - Metrics: http://localhost:9090 (internal)"

    echo ""
    log "To check logs:"
    echo "  docker compose -f $DOCKER_COMPOSE_FILE logs -f docker-auto"

    echo ""
    log "To stop services:"
    echo "  docker compose -f $DOCKER_COMPOSE_FILE down"

    echo ""
    success "Production deployment completed successfully!"
}

# Rollback function
rollback() {
    error "Deployment failed. Rolling back..."

    # Stop failed services
    docker compose -f "$DOCKER_COMPOSE_FILE" down || true

    # Restore from backup if exists
    local latest_backup=$(ls -t "$BACKUP_DIR"/pre-deployment-*.tar.gz 2>/dev/null | head -n1 || echo "")
    if [[ -n "$latest_backup" ]]; then
        log "Restoring from backup: $latest_backup"
        tar -xzf "$latest_backup" -C "$PROJECT_DIR" || true
    fi

    error "Rollback completed. Please check the logs and fix issues before retrying."
}

# Main deployment function
main() {
    log "Starting Docker Auto Update System Production Deployment"

    # Trap errors for rollback
    trap rollback ERR

    check_prerequisites
    validate_environment
    create_directories
    create_backup
    setup_ssl
    setup_nginx
    setup_monitoring
    build_images
    deploy_services
    wait_for_health
    perform_health_checks
    display_summary

    # Remove error trap on successful completion
    trap - ERR
}

# Run main function
main "$@"