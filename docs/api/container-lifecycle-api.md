# Docker Container Lifecycle Management API Documentation

**Version**: 2.3.0
**Base URL**: `http://localhost:8080/api`
**Authentication**: Bearer Token (JWT)

## Overview

The Docker Container Lifecycle Management API provides comprehensive endpoints for managing Docker containers, registries, monitoring, and automated updates. All endpoints require JWT authentication unless otherwise specified.

## Authentication

All API requests must include a valid JWT token in the Authorization header:

```http
Authorization: Bearer <your-jwt-token>
```

## Common Response Format

All API responses follow a consistent format:

```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": { ... },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

Error responses:
```json
{
  "success": false,
  "error": "Error description",
  "code": 400,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Container Management

### List Containers

Get a paginated list of containers with filtering options.

**Endpoint**: `GET /containers`

**Query Parameters**:
- `status` (string, optional): Filter by container status (`running`, `stopped`, `paused`, `exited`)
- `image` (string, optional): Filter by image name
- `limit` (integer, optional): Number of containers to return (default: 50)
- `offset` (integer, optional): Number of containers to skip (default: 0)

**Response**:
```json
{
  "success": true,
  "data": {
    "containers": [
      {
        "id": 1,
        "name": "my-nginx",
        "container_id": "1234567890abcdef",
        "image": "nginx:latest",
        "status": "running",
        "ports": [
          {
            "container_port": 80,
            "host_port": 8080,
            "protocol": "tcp"
          }
        ],
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z",
        "runtime_state": {
          "started_at": "2024-01-01T00:00:00Z",
          "finished_at": null,
          "exit_code": 0,
          "pid": 1234
        }
      }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
  }
}
```

### Get Container Details

Get detailed information about a specific container.

**Endpoint**: `GET /containers/{id}`

**Path Parameters**:
- `id` (integer, required): Container ID

**Response**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "my-nginx",
    "container_id": "1234567890abcdef",
    "image": "nginx:latest",
    "tag": "latest",
    "status": "running",
    "config": {
      "environment": {
        "NGINX_HOST": "localhost"
      },
      "ports": [
        {
          "container_port": 80,
          "host_port": 8080,
          "protocol": "tcp"
        }
      ],
      "volumes": [
        {
          "host_path": "/data/nginx",
          "container_path": "/usr/share/nginx/html",
          "mode": "ro"
        }
      ]
    },
    "health_check": {
      "command": "curl -f http://localhost/health || exit 1",
      "interval": "30s",
      "timeout": "10s",
      "retries": 3
    },
    "runtime_state": {
      "started_at": "2024-01-01T00:00:00Z",
      "pid": 1234,
      "exit_code": 0
    },
    "resource_usage": {
      "cpu_percent": 2.5,
      "memory_usage": 128000000,
      "memory_limit": 512000000,
      "network_rx": 1024,
      "network_tx": 2048
    },
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### Container Operations

#### Start Container

**Endpoint**: `POST /containers/{id}/start`

**Response**:
```json
{
  "success": true,
  "message": "Container started successfully"
}
```

#### Stop Container

**Endpoint**: `POST /containers/{id}/stop`

**Request Body** (optional):
```json
{
  "timeout": 10
}
```

#### Restart Container

**Endpoint**: `POST /containers/{id}/restart`

**Request Body** (optional):
```json
{
  "timeout": 10
}
```

#### Remove Container

**Endpoint**: `DELETE /containers/{id}`

**Query Parameters**:
- `force` (boolean, optional): Force removal of running container
- `volumes` (boolean, optional): Remove associated volumes

## Container Monitoring

### Real-time Container Metrics

Get real-time performance metrics for a container.

**Endpoint**: `GET /monitoring/containers/{id}/metrics`

**Response**:
```json
{
  "success": true,
  "data": {
    "container_id": "1234567890abcdef",
    "timestamp": "2024-01-01T00:00:00Z",
    "cpu": {
      "usage_percent": 2.5,
      "system_usage": 1234567890,
      "online_cpus": 4
    },
    "memory": {
      "usage": 128000000,
      "limit": 512000000,
      "usage_percent": 25.0
    },
    "network": {
      "rx_bytes": 1024,
      "tx_bytes": 2048,
      "rx_packets": 10,
      "tx_packets": 15
    },
    "block_io": {
      "read_bytes": 4096,
      "write_bytes": 8192
    }
  }
}
```

### Historical Metrics

Get historical performance data for a container.

**Endpoint**: `GET /monitoring/containers/{id}/metrics/historical`

**Query Parameters**:
- `since` (string, required): Start time (RFC3339 format)
- `until` (string, optional): End time (RFC3339 format, defaults to now)
- `resolution` (string, optional): Data resolution (`1m`, `5m`, `15m`, `1h`)
- `limit` (integer, optional): Maximum number of data points

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "cpu_percent": 2.5,
      "memory_usage": 128000000,
      "memory_percent": 25.0,
      "network_rx": 1024,
      "network_tx": 2048
    }
  ]
}
```

### Start/Stop Monitoring

**Endpoint**: `POST /monitoring/containers/{id}/start`

**Request Body**:
```json
{
  "update_interval": 5,
  "enable_alerts": true,
  "alert_cpu_threshold": 80.0,
  "alert_memory_threshold": 90.0
}
```

**Endpoint**: `POST /monitoring/containers/{id}/stop`

## WebSocket Real-time Updates

### Container Metrics Stream

**Endpoint**: `ws://localhost:8080/ws/monitoring/containers/{id}/metrics`

Real-time metrics are streamed as JSON messages:

```json
{
  "type": "metrics",
  "container_id": "1234567890abcdef",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "cpu_percent": 2.5,
    "memory_usage": 128000000,
    "memory_percent": 25.0
  }
}
```

### Container Events Stream

**Endpoint**: `ws://localhost:8080/ws/containers/{id}/events`

Container lifecycle events:

```json
{
  "type": "status_change",
  "container_id": "1234567890abcdef",
  "timestamp": "2024-01-01T00:00:00Z",
  "from_status": "stopped",
  "to_status": "running"
}
```

## Web Terminal

### Create Terminal Session

**Endpoint**: `POST /containers/{id}/terminal`

**Request Body**:
```json
{
  "shell": "/bin/bash",
  "cols": 80,
  "rows": 24
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "session_id": "session-uuid",
    "websocket_url": "ws://localhost:8080/ws/terminal/session-uuid"
  }
}
```

### WebSocket Terminal Connection

**Endpoint**: `ws://localhost:8080/ws/terminal/{session_id}`

Terminal I/O is handled through WebSocket messages:

```json
// Input (from client)
{
  "type": "input",
  "data": "ls -la\n"
}

// Output (to client)
{
  "type": "output",
  "data": "total 8\ndrwxr-xr-x 2 root root 4096 Jan  1 00:00 .\n"
}

// Resize
{
  "type": "resize",
  "cols": 80,
  "rows": 24
}
```

## Registry Management

### List Registries

**Endpoint**: `GET /registries`

**Query Parameters**:
- `enabled` (boolean, optional): Filter by enabled status
- `type` (string, optional): Filter by registry type
- `limit` (integer, optional): Number of registries to return
- `offset` (integer, optional): Number of registries to skip

**Response**:
```json
{
  "success": true,
  "data": {
    "registries": [
      {
        "id": 1,
        "name": "Docker Hub",
        "url": "https://registry-1.docker.io",
        "type": "dockerhub",
        "description": "Official Docker registry",
        "is_default": true,
        "enabled": true,
        "status": "connected",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "count": 1
  }
}
```

### Create Registry

**Endpoint**: `POST /registries`

**Request Body**:
```json
{
  "name": "My Private Registry",
  "url": "https://registry.mycompany.com",
  "type": "harbor",
  "description": "Company private registry",
  "username": "myuser",
  "password": "mypassword",
  "is_default": false
}
```

### Test Registry Connection

**Endpoint**: `POST /registries/{id}/test`

**Response**:
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Connection successful",
    "response_time": "150ms",
    "capabilities": ["push", "pull", "search"],
    "info": {
      "registry_version": "2.0",
      "supports_v2": true
    }
  }
}
```

### Registry Statistics

**Endpoint**: `GET /registries/{id}/stats`

**Response**:
```json
{
  "success": true,
  "data": {
    "registry_id": 1,
    "total_containers": 5,
    "active_containers": 3,
    "total_pulls": 150,
    "recent_pulls": 25,
    "last_pull_time": "2024-01-01T00:00:00Z",
    "average_response_time": "120ms",
    "success_rate": 99.5
  }
}
```

## Container Updates

### Check for Updates

**Endpoint**: `GET /containers/{id}/updates/check`

**Response**:
```json
{
  "success": true,
  "data": {
    "has_updates": true,
    "current_version": "nginx:1.20",
    "latest_version": "nginx:1.21",
    "update_available": true,
    "security_updates": true,
    "changelog": "Bug fixes and security improvements"
  }
}
```

### Perform Update

**Endpoint**: `POST /containers/{id}/updates/apply`

**Request Body**:
```json
{
  "strategy": "rolling",
  "backup": true,
  "auto_rollback": true,
  "health_check_timeout": 60
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "update_id": "update-uuid",
    "status": "in_progress",
    "estimated_duration": "2m30s"
  }
}
```

### Update History

**Endpoint**: `GET /containers/{id}/updates/history`

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "container_id": 1,
      "from_version": "nginx:1.20",
      "to_version": "nginx:1.21",
      "status": "completed",
      "strategy": "rolling",
      "started_at": "2024-01-01T00:00:00Z",
      "completed_at": "2024-01-01T00:02:30Z",
      "duration": "2m30s"
    }
  ]
}
```

## Error Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 422 | Unprocessable Entity |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

## Rate Limiting

API requests are rate-limited to:
- **Authenticated users**: 1000 requests per hour
- **Container operations**: 100 operations per minute per container
- **WebSocket connections**: 10 concurrent connections per user

## WebSocket Connection Management

WebSocket connections automatically:
- Reconnect on connection failure
- Send ping/pong messages every 30 seconds
- Timeout after 5 minutes of inactivity
- Support graceful disconnection

## Security Considerations

- All sensitive data (passwords, tokens) are encrypted at rest
- JWT tokens expire after 24 hours by default
- API access logs are maintained for audit purposes
- Rate limiting prevents abuse
- Input validation prevents injection attacks

## SDK and Client Libraries

Official client libraries are available for:
- JavaScript/Node.js
- Python
- Go
- Java

Example usage with JavaScript:

```javascript
import { DockerAutoClient } from '@docker-auto/client';

const client = new DockerAutoClient({
  baseURL: 'http://localhost:8080',
  token: 'your-jwt-token'
});

// List containers
const containers = await client.containers.list();

// Start monitoring
const metrics = await client.monitoring.startContainer(containerId);

// WebSocket connection
const ws = client.monitoring.streamMetrics(containerId);
ws.on('metrics', (data) => {
  console.log('Real-time metrics:', data);
});
```

## Support

For API support, please:
1. Check this documentation
2. Review the [troubleshooting guide](../troubleshooting.md)
3. Submit issues on GitHub
4. Join our Discord community for real-time help