# Docker Auto Update System v2.4.0 - Single Process Architecture
# Simplified build with embedded frontend and Go backend only

# Frontend builder stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy frontend files
COPY frontend/ ./

# Install dependencies and build
RUN npm install --frozen-lockfile && npm run build

# Go backend builder stage
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files for better caching
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy frontend build to backend for embedding
COPY --from=frontend-builder /app/dist ./frontend/dist

# Build backend with embedded frontend
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags embed \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=2.4.0" \
    -o docker-auto-server ./cmd/server

# Final stage - minimal runtime
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# Set timezone
ENV TZ=Asia/Shanghai

# Create non-root user for security
RUN adduser -D -s /bin/sh dockerauto

# Create application directory
RUN mkdir -p /app/logs /app/data && \
    chown -R dockerauto:dockerauto /app

# Copy binary
COPY --from=backend-builder /app/backend/docker-auto-server /app/
RUN chmod +x /app/docker-auto-server && \
    chown dockerauto:dockerauto /app/docker-auto-server

# Switch to non-root user
USER dockerauto
WORKDIR /app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Set environment defaults
ENV APP_PORT=8080 \
    APP_ENV=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    DB_PORT=5432 \
    DB_NAME=dockerauto \
    DB_USER=dockerauto \
    DB_SSL_MODE=disable

# Start the application
CMD ["./docker-auto-server"]