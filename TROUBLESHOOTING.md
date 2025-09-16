# Docker Auto-Update System - Troubleshooting Guide

Comprehensive troubleshooting guide for common issues, debugging procedures, and solutions for the Docker Auto-Update System.

## Table of Contents

1. [Quick Diagnostic Tools](#quick-diagnostic-tools)
2. [Installation Issues](#installation-issues)
3. [Container Management Issues](#container-management-issues)
4. [Update and Rollback Issues](#update-and-rollback-issues)
5. [Performance Issues](#performance-issues)
6. [Network and Connectivity Issues](#network-and-connectivity-issues)
7. [Database Issues](#database-issues)
8. [Authentication and Authorization Issues](#authentication-and-authorization-issues)
9. [Monitoring and Logging Issues](#monitoring-and-logging-issues)
10. [API and WebSocket Issues](#api-and-websocket-issues)
11. [System Resource Issues](#system-resource-issues)
12. [Emergency Recovery Procedures](#emergency-recovery-procedures)

## Quick Diagnostic Tools

### System Health Check

Run this comprehensive health check script to identify common issues:

```bash
#!/bin/bash
# health-check.sh - Quick system diagnostic

echo "=== Docker Auto-Update System Health Check ==="
echo "Timestamp: $(date)"
echo ""

# Check Docker
echo "1. Docker Status:"
if systemctl is-active --quiet docker; then
    echo "   ✓ Docker service is running"
    echo "   Docker version: $(docker --version)"
else
    echo "   ✗ Docker service is not running"
fi

# Check containers
echo ""
echo "2. Container Status:"
if docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -q docker-auto; then
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep docker-auto
else
    echo "   ✗ No Docker Auto containers running"
fi

# Check API health
echo ""
echo "3. API Health:"
if curl -f -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "   ✓ API is responding"
    API_RESPONSE=$(curl -s http://localhost:8080/api/health | jq -r '.data.status // "unknown"')
    echo "   Status: $API_RESPONSE"
else
    echo "   ✗ API is not responding"
fi

# Check database connectivity
echo ""
echo "4. Database Connectivity:"
if docker exec docker-auto-postgres pg_isready -U dockerauto > /dev/null 2>&1; then
    echo "   ✓ Database is responding"
else
    echo "   ✗ Database is not responding"
fi

# Check Redis connectivity
echo ""
echo "5. Redis Connectivity:"
if docker exec docker-auto-redis-primary redis-cli ping | grep -q PONG; then
    echo "   ✓ Redis is responding"
else
    echo "   ✗ Redis is not responding"
fi

# Check disk space
echo ""
echo "6. Disk Usage:"
df -h / | grep -v Filesystem
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 90 ]; then
    echo "   ⚠ Warning: Disk usage is above 90%"
elif [ $DISK_USAGE -gt 80 ]; then
    echo "   ⚠ Warning: Disk usage is above 80%"
else
    echo "   ✓ Disk usage is acceptable"
fi

# Check memory usage
echo ""
echo "7. Memory Usage:"
free -h | head -2
MEMORY_USAGE=$(free | awk 'NR==2{printf "%.1f", $3*100/$2}')
echo "   Memory usage: ${MEMORY_USAGE}%"
if (( $(echo "$MEMORY_USAGE > 90" | bc -l) )); then
    echo "   ⚠ Warning: High memory usage"
fi

# Check recent logs for errors
echo ""
echo "8. Recent Error Logs (last 10 lines):"
docker-compose logs --tail=10 backend 2>/dev/null | grep -i error || echo "   No recent errors found"

echo ""
echo "=== Health Check Complete ==="
```

### Log Analysis Script

```bash
#!/bin/bash
# analyze-logs.sh - Log analysis tool

SERVICE=${1:-"backend"}
LINES=${2:-100}

echo "=== Log Analysis for $SERVICE ==="

# Get recent logs
echo "Recent logs (last $LINES lines):"
docker-compose logs --tail=$LINES $SERVICE

echo ""
echo "=== Error Analysis ==="

# Count error types
echo "Error summary (last 1000 lines):"
docker-compose logs --tail=1000 $SERVICE | grep -i error | \
    awk '{print $NF}' | sort | uniq -c | sort -nr

echo ""
echo "=== Performance Analysis ==="

# Response time analysis
echo "Slow requests (>1000ms):"
docker-compose logs --tail=1000 $SERVICE | grep -i "took.*ms" | \
    awk '{for(i=1;i<=NF;i++) if($i ~ /took/) print $(i+1)}' | \
    sed 's/ms//' | awk '$1 > 1000' | wc -l

echo ""
echo "=== Connection Analysis ==="

# Database connection issues
echo "Database connection errors:"
docker-compose logs --tail=1000 $SERVICE | grep -c "database.*error\|connection.*refused\|timeout"

# Redis connection issues
echo "Redis connection errors:"
docker-compose logs --tail=1000 $SERVICE | grep -c "redis.*error\|cache.*error"
```

## Installation Issues

### Issue: Docker Compose V2 Not Found

**Symptoms:**
```
docker-compose: command not found
```

**Solutions:**
1. **Install Docker Compose V2:**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install docker-compose-plugin

# Or via Docker CLI
mkdir -p ~/.docker/cli-plugins/
curl -SL https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-linux-x86_64 -o ~/.docker/cli-plugins/docker-compose
chmod +x ~/.docker/cli-plugins/docker-compose
```

2. **Use docker compose instead of docker-compose:**
```bash
# Old way
docker-compose up -d

# New way
docker compose up -d
```

### Issue: Permission Denied - Docker Socket

**Symptoms:**
```
permission denied while trying to connect to Docker daemon socket
```

**Solutions:**
1. **Add user to docker group:**
```bash
sudo usermod -aG docker $USER
newgrp docker
```

2. **Fix Docker socket permissions:**
```bash
sudo chmod 666 /var/run/docker.sock
```

3. **Restart Docker service:**
```bash
sudo systemctl restart docker
```

### Issue: Port Already in Use

**Symptoms:**
```
Error response from daemon: driver failed programming external connectivity:
Bind for 0.0.0.0:8080 failed: port is already allocated
```

**Solutions:**
1. **Find process using the port:**
```bash
sudo netstat -tlnp | grep :8080
sudo lsof -i :8080
```

2. **Kill the process:**
```bash
sudo kill -9 <PID>
```

3. **Use different ports in docker-compose.yml:**
```yaml
ports:
  - "8081:8080"  # Use 8081 instead of 8080
```

### Issue: Environment File Not Found

**Symptoms:**
```
WARN[0000] The "DATABASE_URL" variable is not set. Defaulting to a blank string.
```

**Solutions:**
1. **Create environment file:**
```bash
cp .env.example .env
nano .env  # Edit with your values
```

2. **Verify environment file location:**
```bash
ls -la .env*
```

3. **Check file permissions:**
```bash
chmod 644 .env
```

## Container Management Issues

### Issue: Container Won't Start

**Symptoms:**
- Container exits immediately with code 1 or 125
- Container shows "Exited" status

**Diagnostic Steps:**
```bash
# Check container logs
docker logs docker-auto-backend

# Check container configuration
docker inspect docker-auto-backend

# Check resource usage
docker stats --no-stream

# Check available disk space
df -h
```

**Common Solutions:**

1. **Missing environment variables:**
```bash
# Check if all required env vars are set
docker exec docker-auto-backend env | grep -E "DATABASE|REDIS|JWT"
```

2. **Database connection issues:**
```bash
# Test database connectivity
docker exec docker-auto-backend nc -zv postgres 5432
```

3. **Port conflicts:**
```bash
# Check for port conflicts
netstat -tulpn | grep :8080
```

4. **Resource constraints:**
```bash
# Check memory limits
docker exec docker-auto-backend free -h
```

### Issue: Container Keeps Restarting

**Symptoms:**
- Container status shows "Restarting"
- High restart count in `docker ps`

**Diagnostic Steps:**
```bash
# Check restart policy
docker inspect docker-auto-backend | grep -A 5 RestartPolicy

# Monitor container events
docker events --filter container=docker-auto-backend

# Check resource usage over time
watch 'docker stats --no-stream docker-auto-backend'
```

**Solutions:**

1. **Fix application crashes:**
```bash
# Check for panic/crash logs
docker logs docker-auto-backend | grep -i "panic\|fatal\|crash"
```

2. **Adjust restart policy:**
```yaml
restart: "no"  # Temporarily disable restart to debug
```

3. **Increase resource limits:**
```yaml
deploy:
  resources:
    limits:
      memory: 1G
      cpus: '1.0'
```

### Issue: Container Health Checks Failing

**Symptoms:**
- Container shows "unhealthy" status
- Health check endpoint returns errors

**Diagnostic Steps:**
```bash
# Check health check configuration
docker inspect docker-auto-backend | grep -A 10 Healthcheck

# Manual health check
curl -f http://localhost:8080/api/health

# Check health check logs
docker logs docker-auto-backend | grep healthcheck
```

**Solutions:**

1. **Fix health check endpoint:**
```bash
# Test health endpoint manually
docker exec docker-auto-backend curl -f http://localhost:8080/api/health
```

2. **Adjust health check timeouts:**
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## Update and Rollback Issues

### Issue: Container Updates Fail

**Symptoms:**
- Update process hangs or fails
- Container rollback doesn't work
- New image not being pulled

**Diagnostic Steps:**
```bash
# Check update logs
docker-compose logs backend | grep -i update

# Check Docker image versions
docker images | grep docker-auto

# Check available disk space for images
docker system df

# Check network connectivity to registry
curl -I https://registry-1.docker.io/v2/
```

**Solutions:**

1. **Network connectivity issues:**
```bash
# Test registry connectivity
docker pull hello-world

# Check DNS resolution
nslookup registry-1.docker.io
```

2. **Insufficient disk space:**
```bash
# Clean up unused images
docker image prune -f

# Clean up system
docker system prune -f
```

3. **Image pull authentication:**
```bash
# Login to registry
docker login

# Check registry credentials
cat ~/.docker/config.json
```

4. **Update strategy issues:**
```bash
# Try different update strategy
# In container settings, change from "rolling" to "recreate"
```

### Issue: Rollback Not Working

**Symptoms:**
- Rollback command fails
- Previous image version not available
- Service remains in failed state

**Solutions:**

1. **Manual rollback:**
```bash
# Find previous image
docker images docker-auto/backend --format "table {{.Tag}}\t{{.CreatedAt}}"

# Update compose file with previous tag
# docker-compose.yml:
# image: docker-auto/backend:previous-tag

# Restart with previous version
docker-compose up -d backend
```

2. **Backup container recovery:**
```bash
# If backup containers exist
docker start docker-auto-backend-backup
docker stop docker-auto-backend
docker rename docker-auto-backend-backup docker-auto-backend
```

### Issue: Update Takes Too Long

**Symptoms:**
- Update process doesn't complete within expected time
- Health checks timeout during update
- Users experience service interruption

**Solutions:**

1. **Increase timeout values:**
```yaml
update_policy:
  health_check_timeout: 600  # 10 minutes
  start_grace_period: 60     # 1 minute
```

2. **Optimize image size:**
```bash
# Check image layers
docker history docker-auto/backend:latest

# Use multi-stage builds
# Use smaller base images (alpine)
```

3. **Pre-pull images:**
```bash
# Pull new image before update
docker pull docker-auto/backend:latest
```

## Performance Issues

### Issue: Slow API Response Times

**Symptoms:**
- API responses take > 2 seconds
- Web interface feels sluggish
- Timeout errors in logs

**Diagnostic Steps:**
```bash
# Check API response times
time curl http://localhost:8080/api/containers

# Monitor resource usage
docker stats docker-auto-backend

# Check database performance
docker exec docker-auto-postgres pg_stat_activity

# Check for slow queries
docker exec docker-auto-postgres psql -U dockerauto -c "
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;"
```

**Solutions:**

1. **Database optimization:**
```sql
-- Add indexes for common queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_containers_status ON containers(status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_updates_created_at ON updates(created_at);

-- Update table statistics
ANALYZE;
```

2. **Increase database connections:**
```yaml
environment:
  DATABASE_MAX_OPEN_CONNECTIONS: 50
  DATABASE_MAX_IDLE_CONNECTIONS: 25
```

3. **Enable Redis caching:**
```yaml
environment:
  REDIS_CACHE_ENABLED: "true"
  REDIS_CACHE_TTL: "300"
```

4. **Scale backend instances:**
```yaml
# docker-compose.yml
backend:
  scale: 3  # Run 3 backend instances
```

### Issue: High Memory Usage

**Symptoms:**
- Container memory usage > 1GB
- System becomes unresponsive
- Out of memory (OOM) kills

**Diagnostic Steps:**
```bash
# Check memory usage by container
docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"

# Check system memory
free -h

# Check for memory leaks
docker exec docker-auto-backend cat /proc/meminfo

# Monitor memory over time
watch 'free -h && docker stats --no-stream'
```

**Solutions:**

1. **Set memory limits:**
```yaml
deploy:
  resources:
    limits:
      memory: 512M
    reservations:
      memory: 256M
```

2. **Optimize Go application:**
```bash
# Set Go garbage collector
environment:
  GOGC: 100
  GOMEMLIMIT: 512MB
```

3. **Add swap space:**
```bash
# Add 2GB swap file
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### Issue: High CPU Usage

**Symptoms:**
- CPU usage consistently > 80%
- System becomes sluggish
- High load averages

**Solutions:**

1. **Profile CPU usage:**
```bash
# Install pprof
go tool pprof http://localhost:8080/debug/pprof/profile

# Check top CPU consumers
docker exec docker-auto-backend top
```

2. **Optimize polling intervals:**
```yaml
environment:
  CONTAINER_CHECK_INTERVAL: "60s"  # Reduce checking frequency
  METRICS_COLLECTION_INTERVAL: "30s"
```

3. **Add CPU limits:**
```yaml
deploy:
  resources:
    limits:
      cpus: '1.0'
```

## Network and Connectivity Issues

### Issue: Cannot Access Web Interface

**Symptoms:**
- Browser shows "This site can't be reached"
- Connection timeout errors
- 502 Bad Gateway errors

**Diagnostic Steps:**
```bash
# Check if frontend container is running
docker ps | grep frontend

# Check port mappings
docker port docker-auto-frontend

# Test local connectivity
curl -I http://localhost:3000

# Check firewall rules
sudo ufw status
sudo iptables -L
```

**Solutions:**

1. **Check port mapping:**
```yaml
# docker-compose.yml
frontend:
  ports:
    - "3000:80"  # Ensure correct port mapping
```

2. **Fix firewall rules:**
```bash
# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 3000/tcp
```

3. **Check proxy configuration:**
```nginx
# nginx.conf
location / {
    proxy_pass http://frontend:80;
    proxy_set_header Host $host;
}
```

### Issue: API Not Responding

**Symptoms:**
- API endpoints return 503 errors
- Connection refused errors
- WebSocket connections fail

**Solutions:**

1. **Check backend container:**
```bash
# Ensure backend is running
docker ps | grep backend

# Check backend logs
docker logs docker-auto-backend
```

2. **Test API directly:**
```bash
# Bypass proxy
curl http://localhost:8080/api/health

# Check from inside container
docker exec docker-auto-backend curl http://localhost:8080/api/health
```

3. **Fix load balancer configuration:**
```nginx
upstream backend {
    server backend:8080 max_fails=3 fail_timeout=30s;
}
```

### Issue: Docker Socket Permission Denied

**Symptoms:**
- Container management operations fail
- "Permission denied" errors when accessing Docker API
- Docker socket not accessible from container

**Solutions:**

1. **Fix Docker socket permissions:**
```bash
# Method 1: Change socket permissions (less secure)
sudo chmod 666 /var/run/docker.sock

# Method 2: Add user to docker group (recommended)
sudo usermod -aG docker $(whoami)
newgrp docker
```

2. **Mount Docker socket correctly:**
```yaml
# docker-compose.yml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:rw
```

3. **Use rootless Docker (advanced):**
```bash
# Install rootless Docker
curl -fsSL https://get.docker.com/rootless | sh
```

## Database Issues

### Issue: Database Connection Failed

**Symptoms:**
- "Connection refused" errors
- "Password authentication failed"
- "Database does not exist" errors

**Diagnostic Steps:**
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Test connection from backend container
docker exec docker-auto-backend nc -zv postgres 5432

# Check PostgreSQL logs
docker logs docker-auto-postgres

# Verify database exists
docker exec docker-auto-postgres psql -U postgres -l
```

**Solutions:**

1. **Fix connection parameters:**
```yaml
environment:
  DATABASE_URL: "postgresql://dockerauto:password@postgres:5432/dockerauto?sslmode=disable"
```

2. **Reset database password:**
```bash
# Connect as postgres superuser
docker exec -it docker-auto-postgres psql -U postgres

# Reset password
ALTER USER dockerauto PASSWORD 'new_password';
```

3. **Recreate database:**
```bash
# Backup existing data if needed
docker exec docker-auto-postgres pg_dump -U dockerauto dockerauto > backup.sql

# Drop and recreate database
docker exec -it docker-auto-postgres psql -U postgres -c "DROP DATABASE IF EXISTS dockerauto;"
docker exec -it docker-auto-postgres psql -U postgres -c "CREATE DATABASE dockerauto OWNER dockerauto;"
```

### Issue: Database Migration Fails

**Symptoms:**
- Migration scripts fail to run
- Database schema is outdated
- Foreign key constraint errors

**Solutions:**

1. **Run migrations manually:**
```bash
# Check migration status
docker exec docker-auto-backend ./docker-auto migrate status

# Run pending migrations
docker exec docker-auto-backend ./docker-auto migrate up

# Rollback migration if needed
docker exec docker-auto-backend ./docker-auto migrate down 1
```

2. **Fix migration conflicts:**
```bash
# Reset migration state (CAUTION: Only in development)
docker exec docker-auto-backend ./docker-auto migrate reset

# Force migration version
docker exec docker-auto-backend ./docker-auto migrate force <version>
```

### Issue: Database Performance Problems

**Symptoms:**
- Slow query responses
- High database CPU usage
- Connection timeouts

**Solutions:**

1. **Optimize database configuration:**
```postgresql
# postgresql.conf
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 16MB
maintenance_work_mem = 256MB
```

2. **Add database indexes:**
```sql
-- Common indexes for better performance
CREATE INDEX CONCURRENTLY idx_containers_updated_at ON containers(updated_at);
CREATE INDEX CONCURRENTLY idx_containers_status ON containers(status);
CREATE INDEX CONCURRENTLY idx_updates_container_id ON updates(container_id);
```

3. **Enable query optimization:**
```bash
# Install pg_stat_statements extension
docker exec docker-auto-postgres psql -U dockerauto -c "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;"
```

## Authentication and Authorization Issues

### Issue: JWT Token Expired

**Symptoms:**
- 401 Unauthorized errors
- "Token expired" messages
- Users get logged out frequently

**Solutions:**

1. **Increase token expiry time:**
```yaml
environment:
  JWT_EXPIRE_HOURS: 24
  JWT_REFRESH_EXPIRE_HOURS: 168
```

2. **Implement token refresh:**
```javascript
// Frontend token refresh logic
if (response.status === 401) {
  const refreshToken = localStorage.getItem('refresh_token');
  const newToken = await refreshAccessToken(refreshToken);
  // Retry original request with new token
}
```

3. **Check system time:**
```bash
# Ensure system time is correct
timedatectl status
ntpq -p
```

### Issue: User Cannot Login

**Symptoms:**
- Invalid credentials errors
- Account locked messages
- Login form doesn't work

**Diagnostic Steps:**
```bash
# Check user in database
docker exec docker-auto-postgres psql -U dockerauto -c "SELECT id, email, active FROM users WHERE email='user@example.com';"

# Check authentication logs
docker logs docker-auto-backend | grep -i "login\|auth"

# Test API endpoint directly
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

**Solutions:**

1. **Reset user password:**
```bash
# Using API or database directly
docker exec docker-auto-backend ./docker-auto user reset-password user@example.com
```

2. **Activate user account:**
```sql
-- In PostgreSQL
UPDATE users SET active = true WHERE email = 'user@example.com';
```

3. **Check password hash:**
```bash
# Ensure password is hashed correctly
docker exec docker-auto-backend ./docker-auto user verify-password user@example.com
```

### Issue: Permission Denied Errors

**Symptoms:**
- 403 Forbidden responses
- "Insufficient permissions" errors
- Users cannot access certain features

**Solutions:**

1. **Check user roles:**
```sql
-- Check user permissions
SELECT u.email, r.name as role, p.permission
FROM users u
JOIN roles r ON u.role_id = r.id
JOIN role_permissions rp ON r.id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id
WHERE u.email = 'user@example.com';
```

2. **Update user permissions:**
```bash
# Grant admin permissions
docker exec docker-auto-backend ./docker-auto user grant-role user@example.com admin
```

## Monitoring and Logging Issues

### Issue: Metrics Not Available

**Symptoms:**
- Prometheus shows targets as down
- Grafana dashboards empty
- No metrics at `/metrics` endpoint

**Solutions:**

1. **Check metrics endpoint:**
```bash
# Test metrics endpoint
curl http://localhost:8080/metrics

# Check if Prometheus is enabled
docker exec docker-auto-backend env | grep PROMETHEUS
```

2. **Fix Prometheus configuration:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'docker-auto-api'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: /metrics
```

3. **Enable metrics in application:**
```yaml
environment:
  PROMETHEUS_ENABLED: "true"
  METRICS_PORT: "8080"
```

### Issue: Logs Not Showing

**Symptoms:**
- Empty logs in Docker
- Log aggregation not working
- Missing log entries

**Solutions:**

1. **Check logging configuration:**
```yaml
# docker-compose.yml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

2. **Fix log levels:**
```yaml
environment:
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
```

3. **Check log file permissions:**
```bash
# Ensure log directory is writable
docker exec docker-auto-backend ls -la /var/log/docker-auto/
```

## API and WebSocket Issues

### Issue: WebSocket Connection Fails

**Symptoms:**
- Real-time updates not working
- WebSocket connection refused
- Connection drops frequently

**Solutions:**

1. **Check WebSocket endpoint:**
```javascript
// Test WebSocket connection
const ws = new WebSocket('ws://localhost:8080/api/ws?token=<jwt_token>');
ws.onopen = () => console.log('Connected');
ws.onerror = (error) => console.error('WebSocket error:', error);
```

2. **Fix proxy configuration for WebSocket:**
```nginx
location /api/ws {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400;
}
```

3. **Check authentication:**
```bash
# Ensure JWT token is valid
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/auth/profile
```

### Issue: API Rate Limiting

**Symptoms:**
- 429 Too Many Requests errors
- API responses slow down
- Some requests get blocked

**Solutions:**

1. **Adjust rate limiting:**
```yaml
environment:
  RATE_LIMIT_REQUESTS: 1000
  RATE_LIMIT_WINDOW: "60s"
```

2. **Whitelist internal services:**
```nginx
# In Nginx configuration
location /api/ {
    # Skip rate limiting for internal IPs
    set $limit_rate $binary_remote_addr;
    if ($remote_addr ~ "^10\.0\.0\.") {
        set $limit_rate "";
    }

    limit_req zone=api:$limit_rate burst=20 nodelay;
}
```

## System Resource Issues

### Issue: Disk Space Full

**Symptoms:**
- "No space left on device" errors
- Container creation fails
- Database writes fail

**Solutions:**

1. **Clean up Docker resources:**
```bash
# Remove unused containers, networks, images
docker system prune -af

# Remove unused volumes (CAUTION: Data loss)
docker volume prune -f

# Remove old images
docker image prune -a --filter "until=72h"
```

2. **Clean up logs:**
```bash
# Truncate large log files
sudo truncate -s 0 /var/lib/docker/containers/*/*-json.log

# Configure log rotation
# /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

3. **Move Docker data directory:**
```bash
# Stop Docker
sudo systemctl stop docker

# Move Docker data
sudo mv /var/lib/docker /new/location/docker

# Create symlink
sudo ln -s /new/location/docker /var/lib/docker

# Start Docker
sudo systemctl start docker
```

### Issue: Out of Memory

**Symptoms:**
- Process killed by OOM killer
- System becomes unresponsive
- Memory usage alerts

**Solutions:**

1. **Add swap space:**
```bash
# Create 4GB swap file
sudo dd if=/dev/zero of=/swapfile bs=1M count=4096
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

2. **Optimize container memory:**
```yaml
# Set memory limits and reservations
deploy:
  resources:
    limits:
      memory: 1G
    reservations:
      memory: 512M
```

3. **Configure OOM handling:**
```yaml
# Prevent OOM killer
oom_kill_disable: true
oom_score_adj: -1000
```

## Emergency Recovery Procedures

### Complete System Recovery

When the entire system is down:

1. **Stop all services:**
```bash
docker compose -f docker-compose.prod.yml down
```

2. **Check system resources:**
```bash
df -h
free -h
docker system df
```

3. **Restore from backup:**
```bash
# Find latest backup
ls -la /opt/docker-auto/backups/

# Restore database
zcat /opt/docker-auto/backups/latest/database.sql.gz | \
  docker exec -i docker-auto-postgres psql -U dockerauto -d dockerauto

# Restore configuration
cp -r /opt/docker-auto/backups/latest/config/* /opt/docker-auto/
```

4. **Start services one by one:**
```bash
# Start database first
docker compose -f docker-compose.prod.yml up -d postgres redis

# Wait and verify
sleep 30
docker compose -f docker-compose.prod.yml ps

# Start application
docker compose -f docker-compose.prod.yml up -d backend frontend
```

### Data Recovery

For corrupted data or accidental deletion:

1. **Stop write operations:**
```bash
# Set system to read-only mode
docker exec docker-auto-backend touch /tmp/maintenance_mode
```

2. **Create immediate backup:**
```bash
# Backup current state before recovery
docker exec docker-auto-postgres pg_dump -U dockerauto dockerauto > emergency_backup.sql
```

3. **Restore from point-in-time backup:**
```bash
# List available backups
ls -la /opt/docker-auto/backups/

# Choose backup point
BACKUP_DATE="20240916_140000"
zcat /opt/docker-auto/backups/full_backup_$BACKUP_DATE/database.sql.gz | \
  docker exec -i docker-auto-postgres psql -U dockerauto -d dockerauto
```

### Network Connectivity Recovery

For network-related issues:

1. **Reset Docker networks:**
```bash
# Remove all containers
docker compose down

# Remove custom networks
docker network prune -f

# Restart Docker
sudo systemctl restart docker

# Recreate services
docker compose up -d
```

2. **Reset iptables rules:**
```bash
# Backup current rules
sudo iptables-save > iptables_backup

# Reset to default
sudo iptables -F
sudo iptables -X
sudo iptables -t nat -F
sudo iptables -t nat -X
sudo iptables -t mangle -F
sudo iptables -t mangle -X

# Restart Docker to recreate rules
sudo systemctl restart docker
```

### Performance Recovery

For severe performance degradation:

1. **Identify resource bottleneck:**
```bash
# Check all resource usage
htop
iotop
nethogs
docker stats
```

2. **Emergency scaling:**
```bash
# Scale up critical services
docker compose up -d --scale backend=3

# Reduce resource usage
docker update --memory=512m docker-auto-backend
```

3. **Clear caches:**
```bash
# Clear Redis cache
docker exec docker-auto-redis-primary redis-cli FLUSHALL

# Clear system caches
sudo sync
sudo sysctl -w vm.drop_caches=3
```

---

**Getting Additional Help:**

If these troubleshooting steps don't resolve your issue:

1. **Collect diagnostic information:**
```bash
# Run full diagnostic
./health-check.sh > diagnostic_report.txt
./analyze-logs.sh backend 500 >> diagnostic_report.txt
```

2. **Check community resources:**
   - GitHub Issues: Search existing issues
   - Documentation: Review relevant sections
   - Community Discord: Ask for help

3. **Contact support:**
   - Include diagnostic report
   - Describe steps to reproduce
   - Include system information

**Last Updated**: September 16, 2024
**Version**: 2.0.0