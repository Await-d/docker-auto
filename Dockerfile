# Docker Auto Update System - Unified Single Image v2.2.0
# Optimized multi-stage build combining frontend and backend services
# Enhanced with streamlined CI/CD integration and improved performance
FROM node:20-alpine AS frontend-builder

WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci --only=production=false
COPY frontend/ ./
RUN npm run build

# Main application stage
FROM node:20-alpine

# Install system dependencies
RUN apk add --no-cache \
    supervisor \
    nginx \
    curl \
    tzdata \
    postgresql-client \
    go \
    gcc \
    musl-dev

# Set timezone
ENV TZ=Asia/Shanghai

# Create application directories
RUN mkdir -p /app/backend /app/frontend /app/docs /app/logs /var/log/supervisor

# Copy frontend build from previous stage
COPY --from=frontend-builder /app/dist/ /app/frontend/

# Copy backend source and build
WORKDIR /app/backend
COPY backend/ ./

# Build backend
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o docker-auto-server ./cmd/server

# Copy documentation files
COPY *.md /app/docs/
RUN mkdir -p /app/docs && (cp -r docs/* /app/docs/ 2>/dev/null || true)

# Create Nginx configuration
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
RUN cat > /etc/supervisor/conf.d/supervisord.conf << 'EOF'
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
RUN cat > /docker-entrypoint.sh << 'EOF'
#!/bin/sh

echo "🚀 Starting Docker Auto Update System"

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

# Create directories and set permissions
mkdir -p /app/logs
chmod 755 /app/logs
chmod +x /app/backend/docker-auto-server

echo "✅ Starting services with supervisor..."
exec supervisord -c /etc/supervisor/conf.d/supervisord.conf
EOF

RUN chmod +x /docker-entrypoint.sh

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Set environment variables
ENV APP_PORT=8080 \
    APP_ENV=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    NGINX_PORT=80

# Start services
ENTRYPOINT ["/docker-entrypoint.sh"]