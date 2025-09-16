# Monitoring and Logging Implementation Guide

## Overview

This document describes the comprehensive monitoring and logging system implemented for the Docker Auto-Update System. The system provides complete observability through structured logging, metrics collection, health checks, and alerting.

## Architecture

### Components

1. **Structured Logging** (`pkg/logging/`)
   - Context-aware logging with request/user correlation
   - JSON and text output formats
   - Log rotation and retention
   - Security and audit logging

2. **Metrics Collection** (`pkg/monitoring/`)
   - Prometheus-compatible metrics
   - Counter, Gauge, Histogram, and Summary metrics
   - System and application metrics
   - Component-specific collectors

3. **Health Checks** (`pkg/health/`)
   - Database, Docker, HTTP, filesystem, memory, and CPU checks
   - Configurable thresholds and intervals
   - Historical data and trends
   - Automated recovery actions

4. **Alerting System** (`pkg/alerting/`)
   - Multi-channel notifications (email, Slack, Discord, webhooks)
   - Rule-based alert generation
   - Alert grouping and suppression
   - Escalation policies

## Quick Start

### 1. Configuration

Create or update `config/monitoring.yaml`:

```yaml
logging:
  level: "INFO"
  format: "json"
  output: "stdout"

monitoring:
  enabled: true
  collection_interval: "30s"

health_checks:
  enabled: true
  check_interval: "30s"

alerting:
  enabled: true
  evaluation_interval: "30s"
```

### 2. Integration

```go
package main

import (
    "database/sql"
    "github.com/gin-gonic/gin"

    "github.com/docker-auto-update/backend/pkg/config"
    "github.com/docker-auto-update/backend/pkg/monitoring"
)

func main() {
    // Load configuration
    cfg := config.DefaultMonitoringConfig()

    // Initialize observability
    om, err := monitoring.NewObservabilityManager(cfg, db)
    if err != nil {
        panic(err)
    }

    // Setup Gin router
    router := gin.New()
    om.SetupGinMiddleware(router)

    // Setup API routes
    api := router.Group("/api/v1")
    om.SetupAPIRoutes(api)

    // Start server
    router.Run(":8080")
}
```

### 3. Usage Examples

#### Structured Logging

```go
// Create logger with context
logger := logger.WithComponent("container-updater")
           .WithRequestID(requestID)
           .WithUserID(userID)

// Log with structured fields
logger.Info("Container update started", map[string]interface{}{
    "container": "nginx",
    "image":     "nginx:latest",
})

// Log performance
logger.LogPerformance("container_update", duration, map[string]interface{}{
    "container": "nginx",
    "success":   true,
})

// Log security events
logger.LogSecurity("container_access", "authorized", map[string]interface{}{
    "container": "nginx",
    "user_id":   userID,
})
```

#### Metrics Collection

```go
// Track business operations
err := businessMonitor.TrackContainerUpdate("nginx", "update", func() error {
    // Your container update logic
    return updateContainer("nginx")
})

// Track database operations
err := databaseMonitor.TrackQuery("SELECT", query, func() error {
    return db.Query(query)
})

// Track Docker operations
err := dockerMonitor.TrackOperation("pull", func() error {
    return dockerClient.ImagePull(ctx, "nginx:latest", options)
})
```

#### Health Checks

```go
// Manual health check
result, err := healthChecker.CheckHealth("database")
if err != nil {
    log.Error("Health check failed", err)
}

// Get aggregate health
aggregate := healthChecker.GetAggregateHealth()
fmt.Printf("Overall status: %s\n", aggregate.Status)
```

#### Custom Alerts

```go
// Create custom alert
alert := alerting.Alert{
    Name:        "High CPU Usage",
    Description: "CPU usage exceeded threshold",
    Severity:    alerting.AlertSeverityWarning,
    Component:   "system",
    Value:       85.5,
    Threshold:   80.0,
}

err := alertManager.CreateAlert(alert)
```

## API Endpoints

### Monitoring Endpoints

- `GET /api/v1/monitoring/metrics` - Get all metrics
- `GET /api/v1/monitoring/metrics/prometheus` - Get Prometheus format metrics
- `GET /api/v1/monitoring/system` - Get system metrics
- `GET /api/v1/monitoring/components` - Get component metrics
- `GET /api/v1/monitoring/status` - Get overall system status

### Health Check Endpoints

- `GET /api/v1/health` - Overall health status
- `GET /api/v1/health/ready` - Readiness probe
- `GET /api/v1/health/live` - Liveness probe
- `GET /api/v1/health/checks` - All health checks
- `GET /api/v1/health/checks/:name` - Specific health check
- `GET /api/v1/health/history/:name` - Health check history
- `POST /api/v1/health/checks/:name/run` - Run health check manually

## Configuration Reference

### Logging Configuration

```yaml
logging:
  level: "INFO"           # DEBUG, INFO, WARN, ERROR, FATAL
  format: "json"          # json, text
  output: "stdout"        # stdout, stderr, or file path
  component: "docker-auto-update"

  rotation:
    enabled: true
    max_size: 100         # MB
    max_backups: 5
    max_age: 30          # days
    compress: true

  middleware:
    enabled: true
    log_request_body: false
    log_response_body: false
    max_body_size: 1024
    skip_paths:
      - "/health"
      - "/metrics"
    sensitive_headers:
      - "authorization"
      - "cookie"
```

### Monitoring Configuration

```yaml
monitoring:
  enabled: true
  collection_interval: "30s"
  retention_period: "7d"
  max_metrics: 10000
  buffer_size: 1000

  storage:
    type: "memory"        # memory, file, database
    path: "/var/lib/docker-auto/metrics"
    max_size: 1073741824  # bytes
    compression: true

  export:
    enabled: true
    format: "prometheus"   # prometheus, json, influxdb
    endpoint: ""
    interval: "60s"
    batch_size: 100
```

### Health Check Configuration

```yaml
health_checks:
  enabled: true
  check_interval: "30s"
  timeout: "10s"
  failure_threshold: 3
  success_threshold: 1
  grace_period: "60s"
  retry_attempts: 2
  retry_delay: "5s"
  enable_recovery: true
```

### Alert Configuration

```yaml
alerting:
  enabled: true
  evaluation_interval: "30s"
  alert_timeout: "5m"
  resolve_timeout: "5m"
  group_wait: "30s"
  group_interval: "5m"
  repeat_interval: "12h"
  max_alerts: 1000
```

## Metric Types and Examples

### Application Metrics

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - HTTP request duration
- `http_errors_total` - Total HTTP errors
- `http_active_requests` - Active HTTP requests

### Database Metrics

- `database_queries_total` - Total database queries
- `database_query_duration_seconds` - Query duration
- `database_connections_active` - Active connections
- `database_errors_total` - Database errors

### Docker Metrics

- `docker_operations_total` - Total Docker operations
- `docker_operation_duration_seconds` - Operation duration
- `docker_containers_running` - Running containers
- `docker_images_total` - Total images

### System Metrics

- `system_cpu_usage_percent` - CPU usage percentage
- `system_memory_usage_percent` - Memory usage percentage
- `system_disk_usage_percent` - Disk usage percentage
- `system_load_average_1` - 1-minute load average

### Business Metrics

- `container_updates_total` - Total container updates
- `container_update_duration_seconds` - Update duration
- `containers_managed_total` - Managed containers
- `notifications_sent_total` - Notifications sent

## Health Check Types

### Database Health Check
```yaml
database:
  name: "database"
  enabled: true
  interval: "60s"
  timeout: "5s"
  connection_string: "${DATABASE_URL}"
  test_query: "SELECT 1"
  max_connections: 100
```

### Docker Health Check
```yaml
docker:
  name: "docker"
  enabled: true
  interval: "60s"
  timeout: "10s"
  docker_host: "unix:///var/run/docker.sock"
  check_containers: true
  check_images: true
```

### HTTP Health Check
```yaml
api:
  name: "api"
  enabled: true
  interval: "30s"
  timeout: "5s"
  url: "http://localhost:8080/health"
  method: "GET"
  expected_status: [200]
```

### Filesystem Health Check
```yaml
filesystem:
  name: "filesystem"
  enabled: true
  interval: "300s"
  timeout: "5s"
  path: "/var/lib/docker-auto"
  min_free_space: 1073741824  # 1GB
  min_free_percent: 10.0
  check_writable: true
```

### Memory Health Check
```yaml
memory:
  name: "memory"
  enabled: true
  interval: "60s"
  timeout: "2s"
  max_memory_percent: 90.0
  check_swap: true
  max_swap_percent: 50.0
```

## Alert Configuration

### Email Alerts
```yaml
alert_receivers:
  - name: "default"
    email_configs:
      - to: ["admin@example.com"]
        from: "alerts@docker-auto-update.local"
        subject: "[Docker Auto-Update] {{ .Status }} - {{ .AlertName }}"
        smtp_host: "smtp.example.com"
        smtp_port: 587
        username: "alerts@example.com"
        password: "${SMTP_PASSWORD}"
```

### Slack Alerts
```yaml
    slack_configs:
      - webhook_url: "${SLACK_WEBHOOK_URL}"
        channel: "#alerts"
        username: "Docker Auto-Update"
        icon_emoji: ":warning:"
        title: "[{{ .Severity }}] {{ .AlertName }}"
```

### Webhook Alerts
```yaml
    webhook_configs:
      - url: "${WEBHOOK_URL}"
        method: "POST"
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer ${WEBHOOK_TOKEN}"
```

### Alert Rules
```yaml
alert_rules:
  - id: "high_error_rate"
    name: "High Error Rate"
    description: "HTTP error rate is above threshold"
    enabled: true
    condition: ">"
    threshold: 0.1
    severity: "warning"
    duration: "5m"
    channels: ["default"]
    cooldown: "30m"
```

## Best Practices

### Logging
1. Use structured logging with consistent field names
2. Include request/correlation IDs for tracing
3. Log at appropriate levels (DEBUG for development, INFO+ for production)
4. Avoid logging sensitive information
5. Use context-aware logging for better correlation

### Metrics
1. Use descriptive metric names with consistent naming conventions
2. Include relevant labels but avoid high cardinality
3. Use appropriate metric types (Counter, Gauge, Histogram, Summary)
4. Set reasonable retention periods based on storage capacity
5. Monitor metric collection performance impact

### Health Checks
1. Configure appropriate timeouts and thresholds
2. Include critical dependencies in health checks
3. Use different endpoints for liveness vs readiness
4. Monitor health check performance impact
5. Implement graceful degradation strategies

### Alerts
1. Set meaningful thresholds to avoid alert fatigue
2. Use alert grouping and suppression rules
3. Implement escalation policies for critical alerts
4. Test alert channels regularly
5. Document alert runbooks and resolution procedures

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   - Reduce metric retention period
   - Decrease collection interval
   - Limit max metrics count
   - Enable compression

2. **Log File Growth**
   - Enable log rotation
   - Reduce log level in production
   - Set appropriate retention policies
   - Monitor disk usage

3. **Health Check Failures**
   - Check network connectivity
   - Verify service dependencies
   - Adjust timeout settings
   - Review threshold configurations

4. **Missing Alerts**
   - Verify alert channel configurations
   - Check alert rule conditions
   - Test notification endpoints
   - Review alert suppression rules

### Debug Mode

Enable debug logging for troubleshooting:

```yaml
development:
  debug_logging: true
  debug_metrics: true
  mock_services: true
  test_alerts: true
```

### Monitoring the Monitoring

Monitor the monitoring system itself:
- Track metrics collection latency
- Monitor health check execution times
- Alert on monitoring system failures
- Set up external monitoring for critical paths

## Performance Considerations

1. **Metrics Collection**
   - Use appropriate collection intervals
   - Implement metric sampling for high-frequency events
   - Consider using separate storage for long-term retention
   - Monitor memory usage of metrics collector

2. **Logging**
   - Use asynchronous logging where possible
   - Buffer log writes to reduce I/O
   - Implement log sampling for high-volume events
   - Consider using centralized logging systems

3. **Health Checks**
   - Set reasonable check intervals
   - Implement circuit breakers for external dependencies
   - Use connection pooling for database checks
   - Cache health check results where appropriate

## Integration with External Systems

### Prometheus
```yaml
export:
  enabled: true
  format: "prometheus"
  endpoint: "http://prometheus:9090/api/v1/write"
```

### Grafana Dashboards
- System Overview Dashboard
- Application Performance Dashboard
- Docker Container Dashboard
- Alert Management Dashboard

### ELK Stack
```yaml
logging:
  output: "/var/log/docker-auto/app.log"
  format: "json"
```

### External Monitoring Services
- DataDog integration
- New Relic APM integration
- AWS CloudWatch integration
- Azure Monitor integration

## Deployment Considerations

### Docker Compose
```yaml
services:
  docker-auto-update:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./config:/app/config
      - ./logs:/var/log/docker-auto
    environment:
      - MONITORING_CONFIG=/app/config/monitoring.yaml
```

### Kubernetes
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: monitoring-config
data:
  monitoring.yaml: |
    # Your monitoring configuration
```

### Production Checklist

- [ ] Configure log rotation and retention
- [ ] Set up external metrics storage
- [ ] Configure alert channels and test
- [ ] Set up monitoring dashboards
- [ ] Implement backup strategies
- [ ] Configure resource limits
- [ ] Set up external health checks
- [ ] Test disaster recovery procedures

## Security Considerations

1. **Sensitive Data**
   - Avoid logging sensitive information
   - Use secure connections for metrics export
   - Encrypt stored logs and metrics
   - Implement access controls

2. **Network Security**
   - Use TLS for external connections
   - Implement network segmentation
   - Restrict access to monitoring endpoints
   - Use authentication for dashboards

3. **Audit Trail**
   - Log all administrative actions
   - Track configuration changes
   - Monitor access to sensitive endpoints
   - Implement change detection

For more details, see the individual package documentation and configuration examples in the `config/` directory.