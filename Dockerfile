# Docker Auto Update System - External Database Edition v2.3.0
# Optimized multi-stage build combining frontend and backend services
# Designed for external PostgreSQL database connection only
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy all frontend files
COPY frontend/ ./

# Install dependencies and build
RUN npm install --frozen-lockfile && npm run build

# Go backend builder stage
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app/backend

# Copy go mod files first for better caching
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

# Copy backend source
COPY ./backend/ ./

# Build backend without CGO (pure Go binary for external database only)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o docker-auto-server ./cmd/server

# Main application stage
FROM alpine:3.18

# Install runtime dependencies (pure application image)
RUN apk add --no-cache \
    supervisor \
    nginx \
    curl \
    tzdata \
    ca-certificates

# Set timezone
ENV TZ=Asia/Shanghai

# Create application directories
RUN mkdir -p /app/backend /app/frontend /app/docs /app/logs /var/log/supervisor

# Copy frontend build from frontend-builder stage
COPY --from=frontend-builder /app/dist/ /app/frontend/

# Copy backend binary from backend-builder stage
COPY --from=backend-builder /app/backend/docker-auto-server /app/backend/

# Set proper permissions for nginx to access frontend files
RUN chown -R nginx:nginx /app/frontend && \
    chmod -R 755 /app/frontend

# Copy documentation files
COPY *.md /app/docs/
RUN mkdir -p /app/docs && (cp -r docs/* /app/docs/ 2>/dev/null || true)

# Create nginx user and directories
RUN adduser -D -s /bin/sh nginx || true && \
    mkdir -p /etc/nginx/conf.d /var/log/nginx /var/cache/nginx /var/run/nginx && \
    chown -R nginx:nginx /var/log/nginx /var/cache/nginx /var/run/nginx

# Remove default nginx configuration and create new one
RUN rm -f /etc/nginx/nginx.conf /etc/nginx/conf.d/* && \
    cat > /etc/nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
pid /var/run/nginx/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;
    error_log  /var/log/nginx/error.log warn;

    sendfile        on;
    tcp_nopush      on;
    tcp_nodelay     on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    include /etc/nginx/conf.d/*.conf;
}
EOF

# Create site configuration
RUN cat > /etc/nginx/conf.d/default.conf << 'EOF'
server {
    listen 80;
    server_name localhost;

    # Frontend - Serve Vue.js SPA
    location / {
        root /app/frontend;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    # API - Proxy to backend Go service
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket - Direct proxy to backend
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }

    # Documentation - Serve markdown files
    location /docs/ {
        alias /app/docs/;
        autoindex on;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF

# Create supervisor configuration
RUN mkdir -p /etc/supervisor/conf.d /var/log/supervisor && cat > /etc/supervisor/conf.d/supervisord.conf << 'EOF'
[supervisord]
nodaemon=true
user=root
logfile=/var/log/supervisor/supervisord.log
pidfile=/var/run/supervisord.pid

[program:nginx]
command=nginx -g 'daemon off;'
stdout_logfile=/var/log/supervisor/nginx-stdout.log
stderr_logfile=/var/log/supervisor/nginx-stderr.log
autorestart=true
autostart=true
priority=10
redirect_stderr=false

[program:backend]
command=/app/backend/docker-auto-server
directory=/app/backend
stdout_logfile=/var/log/supervisor/backend-stdout.log
stderr_logfile=/var/log/supervisor/backend-stderr.log
autorestart=true
autostart=true
priority=20
redirect_stderr=false
environment=
    APP_PORT=8080,
    APP_ENV=%(ENV_APP_ENV)s,
    LOG_LEVEL=%(ENV_LOG_LEVEL)s,
    LOG_FORMAT=%(ENV_LOG_FORMAT)s,
    DB_HOST=%(ENV_DB_HOST)s,
    DB_PORT=%(ENV_DB_PORT)s,
    DB_NAME=%(ENV_DB_NAME)s,
    DB_USER=%(ENV_DB_USER)s,
    DB_PASSWORD=%(ENV_DB_PASSWORD)s,
    JWT_SECRET=%(ENV_JWT_SECRET)s
EOF

# Create startup script
RUN cat > /docker-entrypoint.sh << 'EOF'
#!/bin/sh

echo "🚀 Starting Docker Auto Update System v2.3.0 (Pure Application Image)"
echo "📋 Database connection is managed by the application"
echo "   Please ensure your external database is properly configured"

# Validate critical environment variables
if [ -z "$DB_HOST" ]; then
    echo "⚠️  WARNING: DB_HOST not set - application may fail to start"
fi

if [ -z "$JWT_SECRET" ]; then
    echo "⚠️  WARNING: JWT_SECRET not set - using insecure default"
fi

# Create directories and set permissions
mkdir -p /app/logs /app/data /var/run/nginx /var/log/nginx
chmod 755 /app/logs /app/data
chmod +x /app/backend/docker-auto-server

# Ensure nginx has proper permissions
chown -R nginx:nginx /var/run/nginx /var/log/nginx /app/frontend
chmod -R 755 /app/frontend

echo "🚀 Starting services with supervisor..."
exec supervisord -c /etc/supervisor/conf.d/supervisord.conf
EOF

RUN chmod +x /docker-entrypoint.sh

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Set environment variables (external database required)
ENV APP_PORT=8080 \
    APP_ENV=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    NGINX_PORT=80 \
    DB_PORT=5432 \
    DB_NAME=dockerauto \
    DB_USER=dockerauto

# Start services
ENTRYPOINT ["/docker-entrypoint.sh"]