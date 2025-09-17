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

# Install build dependencies (removed sqlite-dev for external DB only)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app/backend

# Copy go mod files first for better caching
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

# Copy backend source
COPY ./backend/ ./

# Build backend with CGO support
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -o docker-auto-server ./cmd/server

# Main application stage
FROM alpine:3.18

# Install runtime dependencies (external database optimized)
RUN apk add --no-cache \
    supervisor \
    nginx \
    curl \
    tzdata \
    postgresql-client \
    netcat-openbsd \
    ca-certificates \
    libc6-compat

# Set timezone
ENV TZ=Asia/Shanghai

# Create application directories
RUN mkdir -p /app/backend /app/frontend /app/docs /app/logs /var/log/supervisor

# Copy frontend build from frontend-builder stage
COPY --from=frontend-builder /app/dist/ /app/frontend/

# Copy backend binary from backend-builder stage
COPY --from=backend-builder /app/backend/docker-auto-server /app/backend/

# Copy documentation files
COPY *.md /app/docs/
RUN mkdir -p /app/docs && (cp -r docs/* /app/docs/ 2>/dev/null || true)

# Create Nginx configuration
RUN mkdir -p /etc/nginx/conf.d && cat > /etc/nginx/conf.d/default.conf << 'EOF'
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

echo "🚀 Starting Docker Auto Update System v2.3.0 (External Database Edition)"

# Validate required database environment variables
if [ -z "$DB_HOST" ]; then
    echo "❌ ERROR: DB_HOST environment variable is required"
    echo "   Please provide external PostgreSQL database connection details:"
    echo "   - DB_HOST: PostgreSQL server hostname/IP"
    echo "   - DB_PORT: PostgreSQL server port (default: 5432)"
    echo "   - DB_NAME: Database name"
    echo "   - DB_USER: Database username"
    echo "   - DB_PASSWORD: Database password"
    exit 1
fi

# Set default database port if not provided
DB_PORT=${DB_PORT:-5432}

echo "📡 Connecting to external PostgreSQL database:"
echo "   Host: $DB_HOST:$DB_PORT"
echo "   Database: ${DB_NAME:-dockerauto}"
echo "   User: ${DB_USER:-dockerauto}"

# Wait for external database connection
echo "⏳ Waiting for database connection..."
timeout=60
while ! nc -z "$DB_HOST" "$DB_PORT" > /dev/null 2>&1; do
    timeout=$((timeout - 1))
    if [ $timeout -le 0 ]; then
        echo "❌ Database connection timeout after 60 seconds"
        echo "   Please verify:"
        echo "   1. PostgreSQL server is running"
        echo "   2. Network connectivity to $DB_HOST:$DB_PORT"
        echo "   3. Firewall allows connection on port $DB_PORT"
        exit 1
    fi
    echo "   Retrying... (${timeout}s remaining)"
    sleep 1
done

echo "✅ Database connection established"

# Test database authentication
echo "🔐 Testing database authentication..."
export PGPASSWORD="$DB_PASSWORD"
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "${DB_USER:-dockerauto}" -d "${DB_NAME:-dockerauto}" > /dev/null 2>&1; then
    echo "⚠️  Database authentication test failed, but continuing startup..."
    echo "   Please verify database credentials in environment variables"
fi

# Create directories and set permissions
mkdir -p /app/logs /app/data
chmod 755 /app/logs /app/data
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