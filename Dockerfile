# Docker Auto Update System - Unified Single Image Architecture
# Multi-stage build combining frontend, backend, and documentation services
# Optimized for production deployment with minimal resource footprint

# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci --only=production=false --silent

# Copy frontend source
COPY frontend/ ./

# Build frontend
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.23-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app/backend

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy backend source
COPY backend/ ./

# Build backend binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o docker-auto-server ./cmd/server

# Stage 3: Final Runtime Image
FROM nginx:alpine

# Install required packages
RUN apk add --no-cache \
    supervisor \
    curl \
    tzdata \
    postgresql-client

# Set timezone
ENV TZ=Asia/Shanghai

# Create application directories
RUN mkdir -p /app/backend /app/frontend /app/docs /app/logs /var/log/supervisor

# Copy built backend binary
COPY --from=backend-builder /app/backend/docker-auto-server /app/backend/

# Copy built frontend files
COPY --from=frontend-builder /app/frontend/dist/ /app/frontend/

# Copy documentation files
COPY *.md /app/docs/
COPY docs/ /app/docs/ 2>/dev/null || true

# Copy backend configuration files (if any)
COPY backend/configs/ /app/backend/configs/ 2>/dev/null || true

# Create Nginx configuration for unified service
COPY <<'EOF' /etc/nginx/conf.d/default.conf
server {
    listen 80;
    server_name localhost;

    # Enable gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # Frontend - Serve Vue.js SPA
    location / {
        root /app/frontend;
        index index.html;
        try_files $uri $uri/ /index.html;

        # Cache static assets
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }

    # API - Proxy to backend Go service
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeout settings
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # WebSocket - Direct proxy to backend
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Documentation - Serve markdown files and docs
    location /docs/ {
        alias /app/docs/;
        autoindex on;
        autoindex_exact_size off;
        autoindex_localtime on;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }

    # Backend health check proxy
    location /api/health {
        proxy_pass http://127.0.0.1:8080/health;
        proxy_set_header Host $host;
    }
}
EOF

# Create supervisor configuration
COPY <<'EOF' /etc/supervisor/conf.d/supervisord.conf
[supervisord]
nodaemon=true
user=root
logfile=/var/log/supervisor/supervisord.log
pidfile=/var/run/supervisord.pid

[program:nginx]
command=nginx -g 'daemon off;'
stdout_logfile=/var/log/supervisor/nginx.log
stderr_logfile=/var/log/supervisor/nginx.log
autorestart=true
priority=10

[program:backend]
command=/app/backend/docker-auto-server
directory=/app/backend
stdout_logfile=/var/log/supervisor/backend.log
stderr_logfile=/var/log/supervisor/backend.log
autorestart=true
priority=20
environment=
    APP_PORT=8080,
    APP_ENV=production,
    LOG_LEVEL=info,
    LOG_FORMAT=json
EOF

# Create startup script
COPY <<'EOF' /docker-entrypoint.sh
#!/bin/sh

echo "🚀 Starting Docker Auto Update System (Unified Image)"
echo "=================================================="

# Wait for database if DB_HOST is provided
if [ -n "$DB_HOST" ]; then
    echo "⏳ Waiting for database at $DB_HOST:${DB_PORT:-5432}..."

    timeout=60
    while ! nc -z "$DB_HOST" "${DB_PORT:-5432}" > /dev/null 2>&1; do
        timeout=$((timeout - 1))
        if [ $timeout -le 0 ]; then
            echo "❌ Database connection timeout"
            exit 1
        fi
        sleep 1
    done
    echo "✅ Database connection established"
fi

# Create necessary directories
mkdir -p /app/logs
chmod 755 /app/logs

# Set proper permissions
chown -R nginx:nginx /app/frontend
chmod +x /app/backend/docker-auto-server

echo "✅ Starting services with supervisor..."
exec supervisord -c /etc/supervisor/conf.d/supervisord.conf
EOF

RUN chmod +x /docker-entrypoint.sh

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost/health && curl -f http://localhost/api/health || exit 1

# Set environment variables
ENV APP_PORT=8080 \
    APP_ENV=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    NGINX_PORT=80

# Start services
ENTRYPOINT ["/docker-entrypoint.sh"]