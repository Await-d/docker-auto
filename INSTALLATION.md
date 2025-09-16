# Docker Auto-Update System - Installation Guide

This comprehensive installation guide will walk you through setting up the Docker Auto-Update System in various environments, from development to production.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Pre-Installation Checklist](#pre-installation-checklist)
3. [Quick Install (Docker Compose)](#quick-install-docker-compose)
4. [Manual Installation](#manual-installation)
5. [Database Setup](#database-setup)
6. [Configuration](#configuration)
7. [First Run](#first-run)
8. [Production Setup](#production-setup)
9. [Verification](#verification)
10. [Troubleshooting](#troubleshooting)
11. [Uninstallation](#uninstallation)

## System Requirements

### Minimum Requirements
- **CPU**: 2 cores
- **RAM**: 2GB available
- **Storage**: 10GB free space
- **Network**: Internet access for image downloads

### Recommended Requirements
- **CPU**: 4+ cores
- **RAM**: 4GB+ available
- **Storage**: 50GB+ SSD storage
- **Network**: Stable internet connection (100Mbps+)

### Software Dependencies

#### Required
- **Docker**: 20.10+ (latest stable recommended)
- **Docker Compose**: v2.0+ (latest stable recommended)
- **Operating System**:
  - Ubuntu 20.04+ / Debian 11+
  - CentOS 8+ / RHEL 8+
  - macOS 10.15+
  - Windows 10+ with WSL2

#### Optional (for manual installation)
- **Go**: 1.21+ (if building from source)
- **Node.js**: 18+ with npm/yarn (if building frontend)
- **PostgreSQL**: 13+ (if not using Docker)
- **Redis**: 6+ (for caching)

## Pre-Installation Checklist

Before installing, ensure you have:

- [ ] Root or sudo access to the target system
- [ ] Docker and Docker Compose installed and running
- [ ] Available ports: 3000, 8080, 5432, 6379
- [ ] Internet access for downloading images
- [ ] At least 10GB free disk space
- [ ] Access to Docker socket (`/var/run/docker.sock`)

### Check Docker Installation

```bash
# Verify Docker installation
docker --version
docker-compose --version

# Test Docker functionality
docker run hello-world

# Verify Docker Compose V2
docker compose version
```

### Check System Resources

```bash
# Check available memory
free -h

# Check disk space
df -h

# Check available ports
netstat -tlnp | grep -E ':3000|:8080|:5432|:6379'
```

## Quick Install (Docker Compose)

This is the recommended installation method for most users.

### Step 1: Download the Project

```bash
# Option 1: Clone from Git (if repository is available)
git clone https://github.com/your-org/docker-auto.git
cd docker-auto

# Option 2: Download release archive
wget https://github.com/your-org/docker-auto/archive/refs/tags/v2.0.0.tar.gz
tar -xzf v2.0.0.tar.gz
cd docker-auto-2.0.0
```

### Step 2: Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit configuration (see Configuration section for details)
nano .env
```

### Step 3: Start the System

```bash
# Start all services
docker-compose up -d

# View logs (optional)
docker-compose logs -f

# Check service status
docker-compose ps
```

### Step 4: Initialize Database

```bash
# Run database migrations
docker-compose exec backend ./docker-auto migrate

# Create initial admin user (optional)
docker-compose exec backend ./docker-auto create-admin
```

### Step 5: Access the System

- **Web Interface**: http://localhost:3000
- **API**: http://localhost:8080/api
- **Health Check**: http://localhost:8080/api/health

## Manual Installation

For advanced users who prefer manual installation or need custom configurations.

### Step 1: Install Dependencies

#### Ubuntu/Debian
```bash
# Update package list
sudo apt update

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt install docker-compose-plugin

# Install PostgreSQL (optional)
sudo apt install postgresql postgresql-contrib

# Install Redis (optional)
sudo apt install redis-server

# Install Go (for building from source)
sudo apt install golang-go

# Install Node.js (for frontend development)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install nodejs
```

#### CentOS/RHEL
```bash
# Install Docker
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Install PostgreSQL
sudo yum install postgresql-server postgresql-contrib
sudo postgresql-setup initdb
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### macOS
```bash
# Install Docker Desktop
# Download from https://docs.docker.com/desktop/mac/install/

# Install using Homebrew (alternative)
brew install --cask docker

# Install PostgreSQL
brew install postgresql
brew services start postgresql

# Install Redis
brew install redis
brew services start redis
```

### Step 2: Build from Source

```bash
# Clone repository
git clone https://github.com/your-org/docker-auto.git
cd docker-auto

# Build backend
cd backend
go mod download
go build -o ../bin/docker-auto ./cmd/main.go

# Build frontend
cd ../frontend
npm install
npm run build

# Copy built files
cp -r dist/* ../static/
```

### Step 3: Setup Database

```bash
# Connect to PostgreSQL
sudo -u postgres psql

# Create database and user
CREATE DATABASE dockerauto;
CREATE USER dockerauto WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE dockerauto TO dockerauto;
\q

# Run migrations
cd /path/to/docker-auto
./bin/docker-auto migrate --config config.yaml
```

### Step 4: Create Configuration

```bash
# Create config directory
sudo mkdir -p /etc/docker-auto

# Create configuration file
sudo nano /etc/docker-auto/config.yaml
```

Configuration content:
```yaml
database:
  host: localhost
  port: 5432
  name: dockerauto
  user: dockerauto
  password: your_password
  ssl_mode: disable

redis:
  host: localhost
  port: 6379
  password: ""
  database: 0

server:
  host: 0.0.0.0
  port: 8080
  static_dir: /opt/docker-auto/static

jwt:
  secret: your-jwt-secret-key
  expire_hours: 24

docker:
  host: unix:///var/run/docker.sock
  api_version: "1.41"
```

### Step 5: Create System Service

```bash
# Create systemd service file
sudo nano /etc/systemd/system/docker-auto.service
```

Service file content:
```ini
[Unit]
Description=Docker Auto-Update System
After=network.target docker.service postgresql.service
Requires=docker.service

[Service]
Type=simple
User=docker-auto
Group=docker-auto
WorkingDirectory=/opt/docker-auto
ExecStart=/opt/docker-auto/bin/docker-auto server --config /etc/docker-auto/config.yaml
Restart=always
RestartSec=5

# Environment
Environment=GIN_MODE=release

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/docker-auto

[Install]
WantedBy=multi-user.target
```

### Step 6: Install and Start Service

```bash
# Create user for service
sudo useradd -r -s /bin/false docker-auto
sudo usermod -aG docker docker-auto

# Install application
sudo mkdir -p /opt/docker-auto
sudo cp -r bin static /opt/docker-auto/
sudo chown -R docker-auto:docker-auto /opt/docker-auto

# Start service
sudo systemctl daemon-reload
sudo systemctl enable docker-auto
sudo systemctl start docker-auto

# Check status
sudo systemctl status docker-auto
```

## Database Setup

### PostgreSQL Configuration

#### Using Docker (Recommended)
```bash
# Start PostgreSQL container
docker run -d \
  --name postgres-docker-auto \
  -e POSTGRES_DB=dockerauto \
  -e POSTGRES_USER=dockerauto \
  -e POSTGRES_PASSWORD=secure_password \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:15

# Verify connection
docker exec -it postgres-docker-auto psql -U dockerauto -d dockerauto -c "SELECT version();"
```

#### Manual PostgreSQL Setup
```bash
# Install PostgreSQL
sudo apt install postgresql postgresql-contrib  # Ubuntu/Debian
sudo yum install postgresql-server postgresql-contrib  # CentOS/RHEL

# Initialize database (CentOS/RHEL only)
sudo postgresql-setup initdb

# Start PostgreSQL service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres createuser --pwprompt dockerauto
sudo -u postgres createdb -O dockerauto dockerauto

# Configure PostgreSQL
sudo nano /etc/postgresql/13/main/postgresql.conf
# Uncomment and modify:
# listen_addresses = 'localhost'
# port = 5432

# Configure authentication
sudo nano /etc/postgresql/13/main/pg_hba.conf
# Add line:
# local   dockerauto      dockerauto                              md5

# Restart PostgreSQL
sudo systemctl restart postgresql
```

### Redis Configuration

#### Using Docker (Recommended)
```bash
# Start Redis container
docker run -d \
  --name redis-docker-auto \
  -p 6379:6379 \
  -v redis_data:/data \
  redis:7-alpine redis-server --appendonly yes

# Verify connection
docker exec -it redis-docker-auto redis-cli ping
```

#### Manual Redis Setup
```bash
# Install Redis
sudo apt install redis-server  # Ubuntu/Debian
sudo yum install redis  # CentOS/RHEL

# Configure Redis
sudo nano /etc/redis/redis.conf
# Modify settings:
# bind 127.0.0.1
# port 6379
# save 900 1

# Start Redis service
sudo systemctl start redis
sudo systemctl enable redis

# Test Redis
redis-cli ping
```

## Configuration

### Environment Variables

The system uses the following environment variables:

#### Database Configuration
```bash
# PostgreSQL settings
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=dockerauto
DATABASE_USER=dockerauto
DATABASE_PASSWORD=secure_password
DATABASE_SSL_MODE=disable

# Connection pool settings
DATABASE_MAX_OPEN_CONNECTIONS=25
DATABASE_MAX_IDLE_CONNECTIONS=25
DATABASE_CONNECTION_MAX_LIFETIME=5m
```

#### Redis Configuration
```bash
# Redis settings
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DATABASE=0
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10
```

#### Application Configuration
```bash
# Server settings
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# JWT authentication
JWT_SECRET=your-secure-jwt-secret-key-256-bits-minimum
JWT_EXPIRE_HOURS=24
JWT_REFRESH_EXPIRE_HOURS=168

# Docker configuration
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_API_VERSION=1.41
DOCKER_TIMEOUT=30s
```

#### Monitoring and Logging
```bash
# Logging
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE=/var/log/docker-auto/app.log

# Metrics
PROMETHEUS_ENABLED=true
METRICS_PORT=9090
METRICS_PATH=/metrics

# Health check
HEALTH_CHECK_INTERVAL=30s
HEALTH_CHECK_TIMEOUT=10s
```

#### Notification Configuration
```bash
# Email notifications
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@docker-auto.com

# Slack notifications
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK

# Webhook notifications
WEBHOOK_URL=https://your-webhook-endpoint.com/notify
WEBHOOK_SECRET=your-webhook-secret
```

### Configuration File

Create `/etc/docker-auto/config.yaml`:

```yaml
# Application configuration
app:
  name: "Docker Auto-Update System"
  version: "2.0.0"
  environment: "production"  # development, staging, production

# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  static_dir: "/opt/docker-auto/static"

# Database configuration
database:
  host: "localhost"
  port: 5432
  name: "dockerauto"
  user: "dockerauto"
  password: "secure_password"
  ssl_mode: "disable"
  max_open_connections: 25
  max_idle_connections: 25
  connection_max_lifetime: "5m"

# Redis configuration
redis:
  host: "localhost"
  port: 6379
  password: ""
  database: 0
  max_retries: 3
  pool_size: 10

# JWT configuration
jwt:
  secret: "your-secure-jwt-secret-key-256-bits-minimum"
  expire_hours: 24
  refresh_expire_hours: 168

# Docker configuration
docker:
  host: "unix:///var/run/docker.sock"
  api_version: "1.41"
  timeout: "30s"

# Logging configuration
logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # json, text
  file: "/var/log/docker-auto/app.log"
  max_size: "100MB"
  max_backups: 3
  max_age: 30            # days

# Monitoring configuration
monitoring:
  prometheus_enabled: true
  metrics_port: 9090
  metrics_path: "/metrics"
  health_check_interval: "30s"
  health_check_timeout: "10s"

# Notification configuration
notifications:
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    smtp_user: "your-email@gmail.com"
    smtp_password: "your-app-password"
    from_address: "noreply@docker-auto.com"

  slack:
    enabled: false
    webhook_url: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"

  webhook:
    enabled: false
    url: "https://your-webhook-endpoint.com/notify"
    secret: "your-webhook-secret"

# Update configuration
updates:
  check_interval: "1h"           # How often to check for updates
  max_concurrent_updates: 5      # Maximum concurrent container updates
  default_strategy: "rolling"    # rolling, blue-green, canary
  health_check_timeout: "5m"     # Health check timeout after update
  rollback_on_failure: true     # Auto-rollback on health check failure

# Security configuration
security:
  enable_audit_log: true
  rate_limit_requests: 100       # Requests per minute per IP
  cors_allowed_origins: ["http://localhost:3000", "https://your-domain.com"]
  csrf_protection: true
  content_security_policy: true
```

## First Run

After installation and configuration, follow these steps for the first run:

### Step 1: Start the System

```bash
# Using Docker Compose
docker-compose up -d

# Using systemd (manual installation)
sudo systemctl start docker-auto
```

### Step 2: Check System Health

```bash
# Check API health
curl http://localhost:8080/api/health

# Expected response:
# {"status":"ok","version":"2.0.0","timestamp":"2024-09-16T10:00:00Z"}

# Check database connection
curl http://localhost:8080/api/health/db

# Check Redis connection
curl http://localhost:8080/api/health/redis

# Check Docker connection
curl http://localhost:8080/api/health/docker
```

### Step 3: Access Web Interface

1. Open your browser to http://localhost:3000
2. You should see the login page
3. Use default credentials (change immediately):
   - Email: `admin@example.com`
   - Password: `admin123`

### Step 4: Create Admin User (If Needed)

```bash
# Using Docker Compose
docker-compose exec backend ./docker-auto create-admin \
  --email admin@yourdomain.com \
  --password YourSecurePassword123! \
  --name "System Administrator"

# Using manual installation
/opt/docker-auto/bin/docker-auto create-admin \
  --config /etc/docker-auto/config.yaml \
  --email admin@yourdomain.com \
  --password YourSecurePassword123! \
  --name "System Administrator"
```

### Step 5: Initial Configuration

1. **Change Admin Password**: Go to Settings > Account and change the default password
2. **Configure Docker**: Go to Settings > Docker and verify Docker connection
3. **Setup Notifications**: Configure email/Slack notifications in Settings > Notifications
4. **Add First Container**: Add your first container to manage

## Production Setup

### SSL/TLS Configuration

For production deployment, configure SSL/TLS using a reverse proxy:

#### Nginx Configuration

Create `/etc/nginx/sites-available/docker-auto`:

```nginx
server {
    listen 80;
    server_name your-domain.com www.your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com www.your-domain.com;

    # SSL configuration
    ssl_certificate /path/to/your/certificate.crt;
    ssl_certificate_key /path/to/your/private.key;
    ssl_session_timeout 5m;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Frontend
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Backend API
    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Static files
    location /static {
        alias /opt/docker-auto/static;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

Enable the site:
```bash
sudo ln -s /etc/nginx/sites-available/docker-auto /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### Apache Configuration

Create `/etc/apache2/sites-available/docker-auto.conf`:

```apache
<VirtualHost *:80>
    ServerName your-domain.com
    ServerAlias www.your-domain.com
    Redirect permanent / https://your-domain.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName your-domain.com
    ServerAlias www.your-domain.com

    # SSL Configuration
    SSLEngine on
    SSLCertificateFile /path/to/your/certificate.crt
    SSLCertificateKeyFile /path/to/your/private.key
    SSLProtocol all -SSLv2 -SSLv3 -TLSv1 -TLSv1.1
    SSLCipherSuite ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384

    # Security Headers
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
    Header always set X-Frame-Options DENY
    Header always set X-Content-Type-Options nosniff

    # Frontend proxy
    ProxyPreserveHost On
    ProxyRequests Off
    ProxyPass /api/ http://localhost:8080/api/
    ProxyPassReverse /api/ http://localhost:8080/api/
    ProxyPass / http://localhost:3000/
    ProxyPassReverse / http://localhost:3000/

    # WebSocket support
    ProxyPass /api/ws ws://localhost:8080/api/ws
    ProxyPassReverse /api/ws ws://localhost:8080/api/ws

    # Static files
    Alias /static /opt/docker-auto/static
    <Directory /opt/docker-auto/static>
        Require all granted
        ExpiresActive On
        ExpiresDefault "access plus 1 year"
    </Directory>
</VirtualHost>
```

### Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https
sudo ufw deny 8080  # Block direct backend access
sudo ufw deny 5432  # Block direct database access
sudo ufw enable

# iptables (CentOS/RHEL)
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

### Backup Configuration

Create automated backup script `/opt/docker-auto/scripts/backup.sh`:

```bash
#!/bin/bash

BACKUP_DIR="/var/backups/docker-auto"
DATE=$(date +%Y%m%d_%H%M%S)
DB_BACKUP="${BACKUP_DIR}/db_backup_${DATE}.sql"
CONFIG_BACKUP="${BACKUP_DIR}/config_backup_${DATE}.tar.gz"

# Create backup directory
mkdir -p ${BACKUP_DIR}

# Database backup
docker exec postgres-docker-auto pg_dump -U dockerauto dockerauto > ${DB_BACKUP}

# Configuration backup
tar -czf ${CONFIG_BACKUP} /etc/docker-auto /opt/docker-auto

# Remove old backups (keep 7 days)
find ${BACKUP_DIR} -name "*.sql" -mtime +7 -delete
find ${BACKUP_DIR} -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: ${DATE}"
```

Add to crontab:
```bash
sudo crontab -e
# Add line:
0 2 * * * /opt/docker-auto/scripts/backup.sh >> /var/log/docker-auto/backup.log 2>&1
```

## Verification

### System Health Verification

```bash
# Check all services are running
docker-compose ps

# Expected output:
# NAME                COMMAND                  SERVICE             STATUS              PORTS
# docker-auto-backend   "./docker-auto serve…"   backend             running             0.0.0.0:8080->8080/tcp
# docker-auto-frontend  "/docker-entrypoint.…"   frontend            running             0.0.0.0:3000->80/tcp
# docker-auto-db        "docker-entrypoint.s…"   db                  running             0.0.0.0:5432->5432/tcp
# docker-auto-redis     "docker-entrypoint.s…"   redis               running             0.0.0.0:6379->6379/tcp

# Check logs for any errors
docker-compose logs --tail=50

# Test API endpoints
curl -f http://localhost:8080/api/health
curl -f http://localhost:8080/api/version

# Test web interface
curl -I http://localhost:3000
```

### Performance Verification

```bash
# Check resource usage
docker stats --no-stream

# Expected resource usage:
# CONTAINER             CPU %     MEM USAGE / LIMIT     MEM %
# docker-auto-backend   0.50%     128MiB / 2GiB         6.25%
# docker-auto-frontend  0.10%     32MiB / 2GiB          1.56%
# docker-auto-db        0.30%     256MiB / 2GiB         12.5%
# docker-auto-redis     0.05%     16MiB / 2GiB          0.78%

# Test API response time
time curl http://localhost:8080/api/health
# Should complete in < 100ms
```

### Security Verification

```bash
# Check file permissions
ls -la /opt/docker-auto/
ls -la /etc/docker-auto/

# Verify SSL certificate (if configured)
openssl s_client -connect your-domain.com:443 -servername your-domain.com

# Test authentication
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'

# Should return JWT token
```

## Troubleshooting

### Common Installation Issues

#### Docker Permission Denied
```bash
# Error: permission denied while trying to connect to Docker daemon socket
sudo usermod -aG docker $USER
newgrp docker

# Test Docker access
docker ps
```

#### Port Already in Use
```bash
# Error: port is already allocated
# Check what's using the port
sudo netstat -tlnp | grep :8080

# Kill the process or use different port
sudo kill -9 PID
# Or modify docker-compose.yml to use different port
```

#### Database Connection Failed
```bash
# Check database container
docker-compose logs db

# Test database connection
docker-compose exec db psql -U dockerauto -d dockerauto -c "SELECT 1;"

# Check database configuration in .env file
cat .env | grep DATABASE
```

#### Frontend Not Loading
```bash
# Check frontend container
docker-compose logs frontend

# Check if backend is accessible
curl http://localhost:8080/api/health

# Verify proxy configuration
curl -I http://localhost:3000
```

### Log Locations

```bash
# Application logs
docker-compose logs backend
docker-compose logs frontend

# System logs (manual installation)
tail -f /var/log/docker-auto/app.log
tail -f /var/log/docker-auto/error.log

# Database logs
docker-compose logs db

# System service logs
sudo journalctl -u docker-auto -f
```

### Debug Mode

Enable debug mode for troubleshooting:

```bash
# In .env file
LOG_LEVEL=debug
GIN_MODE=debug

# Restart services
docker-compose restart backend

# View debug logs
docker-compose logs -f backend
```

## Uninstallation

### Docker Compose Uninstallation

```bash
# Stop and remove containers
docker-compose down -v

# Remove images
docker-compose down --rmi all

# Remove volumes (WARNING: This will delete all data)
docker volume rm docker-auto_postgres_data docker-auto_redis_data

# Remove project directory
cd ..
rm -rf docker-auto
```

### Manual Uninstallation

```bash
# Stop service
sudo systemctl stop docker-auto
sudo systemctl disable docker-auto

# Remove service file
sudo rm /etc/systemd/system/docker-auto.service
sudo systemctl daemon-reload

# Remove application files
sudo rm -rf /opt/docker-auto
sudo rm -rf /etc/docker-auto
sudo rm -rf /var/log/docker-auto

# Remove user
sudo userdel docker-auto

# Remove database (if installed manually)
sudo -u postgres dropdb dockerauto
sudo -u postgres dropuser dockerauto
```

### Clean Up Docker

```bash
# Remove unused Docker resources
docker system prune -af
docker volume prune -f
docker network prune -f
```

## Next Steps

After successful installation:

1. **Security**: Change default passwords and configure SSL/TLS
2. **Configuration**: Review and customize settings in the web interface
3. **Monitoring**: Set up monitoring and alerting
4. **Backup**: Configure automated backups
5. **Documentation**: Read the [User Guide](USER_GUIDE.md) for operating instructions

## Support

If you encounter issues during installation:

1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review the logs for specific error messages
3. Search existing issues on GitHub
4. Create a new issue with detailed information:
   - Operating system and version
   - Docker and Docker Compose versions
   - Complete error messages
   - Steps to reproduce the problem

---

**Last Updated**: September 16, 2024
**Version**: 2.0.0