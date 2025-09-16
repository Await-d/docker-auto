# Docker Auto-Update System - API Documentation

Complete API reference for the Docker Auto-Update System REST API and WebSocket interface.

## Table of Contents

1. [API Overview](#api-overview)
2. [Authentication](#authentication)
3. [Request/Response Format](#requestresponse-format)
4. [Error Handling](#error-handling)
5. [Rate Limiting](#rate-limiting)
6. [REST API Endpoints](#rest-api-endpoints)
7. [WebSocket API](#websocket-api)
8. [SDK Examples](#sdk-examples)
9. [Webhook Integration](#webhook-integration)
10. [API Best Practices](#api-best-practices)

## API Overview

### Base URL

```
Production: https://your-domain.com/api
Development: http://localhost:8080/api
```

### API Version

Current API version: **v1**

All endpoints are prefixed with `/api` unless otherwise specified.

### Supported Formats

- **Request**: JSON, Form Data (file uploads)
- **Response**: JSON
- **Content-Type**: `application/json` (required for all requests)

### API Features

- **RESTful Design**: Standard HTTP methods and status codes
- **JWT Authentication**: Secure token-based authentication
- **Real-time Updates**: WebSocket support for live data
- **Pagination**: Cursor and offset-based pagination
- **Filtering**: Advanced filtering and search capabilities
- **Rate Limiting**: Protection against abuse
- **Comprehensive Logging**: Request/response logging and audit trail

## Authentication

### JWT-Based Authentication

The API uses JSON Web Tokens (JWT) for authentication. All protected endpoints require a valid JWT token in the `Authorization` header.

#### Header Format

```http
Authorization: Bearer <jwt_token>
```

#### Token Lifecycle

- **Access Token**: Valid for 24 hours (configurable)
- **Refresh Token**: Valid for 7 days (configurable)
- **Automatic Refresh**: Tokens can be refreshed before expiration

### Authentication Endpoints

#### Login

```http
POST /api/auth/login
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "user_password"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400,
    "token_type": "Bearer",
    "user": {
      "id": 1,
      "email": "user@example.com",
      "name": "John Doe",
      "role": "admin",
      "permissions": ["container:read", "container:write", "user:manage"]
    }
  }
}
```

#### Refresh Token

```http
POST /api/auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "success": true,
  "message": "Token refreshed successfully",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400,
    "token_type": "Bearer"
  }
}
```

#### Logout

```http
POST /api/auth/logout
```

**Headers:**
```http
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "success": true,
  "message": "Logout successful"
}
```

## Request/Response Format

### Standard Response Format

All API responses follow a consistent format:

```json
{
  "success": true|false,
  "message": "Human-readable message",
  "data": {...},
  "meta": {
    "request_id": "uuid",
    "timestamp": "2024-09-16T10:00:00Z",
    "version": "v1"
  }
}
```

### Pagination Response

For paginated endpoints:

```json
{
  "success": true,
  "message": "Data retrieved successfully",
  "data": {
    "items": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100,
      "pages": 5,
      "has_next": true,
      "has_prev": false
    }
  }
}
```

### Query Parameters

#### Common Parameters

- **page**: Page number (default: 1)
- **limit**: Items per page (default: 20, max: 100)
- **search**: Search query string
- **sort_by**: Field to sort by
- **sort_order**: `asc` or `desc` (default: `desc`)

#### Filter Parameters

- **created_after**: ISO 8601 timestamp
- **created_before**: ISO 8601 timestamp
- **updated_after**: ISO 8601 timestamp
- **updated_before**: ISO 8601 timestamp

## Error Handling

### HTTP Status Codes

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **204 No Content**: Request successful, no content returned
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource conflict
- **422 Unprocessable Entity**: Validation errors
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server error

### Error Response Format

```json
{
  "success": false,
  "message": "Error description",
  "error": {
    "code": "ERROR_CODE",
    "details": "Detailed error information",
    "fields": {
      "field_name": ["field-specific error messages"]
    }
  },
  "meta": {
    "request_id": "uuid",
    "timestamp": "2024-09-16T10:00:00Z"
  }
}
```

### Common Error Codes

- **INVALID_REQUEST**: Malformed request
- **AUTHENTICATION_REQUIRED**: Missing or invalid authentication
- **INSUFFICIENT_PERMISSIONS**: Insufficient permissions for operation
- **RESOURCE_NOT_FOUND**: Requested resource not found
- **VALIDATION_ERROR**: Request validation failed
- **RATE_LIMIT_EXCEEDED**: Too many requests
- **DOCKER_ERROR**: Docker operation failed
- **INTERNAL_ERROR**: Internal server error

## Rate Limiting

### Rate Limit Policy

- **Default**: 100 requests per minute per IP address
- **Authenticated**: 1000 requests per minute per user
- **Admin**: 5000 requests per minute

### Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1694865600
```

## REST API Endpoints

### System Endpoints

#### Health Check

```http
GET /api/health
```

**Response:**
```json
{
  "success": true,
  "message": "System is healthy",
  "data": {
    "status": "ok",
    "version": "2.0.0",
    "timestamp": "2024-09-16T10:00:00Z",
    "uptime": "72h30m15s",
    "components": {
      "database": "healthy",
      "redis": "healthy",
      "docker": "healthy"
    }
  }
}
```

#### System Information

```http
GET /api/system/info
```

**Authentication Required**

**Response:**
```json
{
  "success": true,
  "data": {
    "version": "2.0.0",
    "build_date": "2024-09-16T10:00:00Z",
    "go_version": "1.21.0",
    "docker_version": "24.0.0",
    "system": {
      "os": "linux",
      "arch": "amd64",
      "cpu_cores": 4,
      "memory_total": "8GB"
    },
    "features": {
      "monitoring": true,
      "notifications": true,
      "auto_updates": true
    }
  }
}
```

### Container Management

#### List Containers

```http
GET /api/containers
```

**Authentication Required**

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `search` (string): Search by name or image
- `status` (string): Filter by status (`running`, `stopped`, `error`)
- `update_policy` (string): Filter by update policy
- `has_update` (bool): Filter containers with available updates
- `sort_by` (string): Sort field (default: `updated_at`)
- `sort_order` (string): Sort order (`asc`/`desc`, default: `desc`)

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "web-server",
        "image": "nginx:latest",
        "image_digest": "sha256:abc123...",
        "status": "running",
        "created_at": "2024-09-16T10:00:00Z",
        "updated_at": "2024-09-16T12:00:00Z",
        "config": {
          "ports": [{"host": 8080, "container": 80, "protocol": "tcp"}],
          "environment": {"ENV": "production"},
          "volumes": [{"host": "/data", "container": "/app/data", "mode": "rw"}],
          "labels": {"app": "web", "version": "1.0.0"}
        },
        "update_policy": {
          "policy": "auto",
          "strategy": "rolling",
          "schedule": "0 2 * * *"
        },
        "health": {
          "status": "healthy",
          "last_check": "2024-09-16T12:00:00Z"
        },
        "stats": {
          "cpu_usage": 15.5,
          "memory_usage": "128MB",
          "network_rx": "1.2GB",
          "network_tx": "800MB"
        },
        "update_available": true,
        "latest_version": "nginx:1.25.0"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 5,
      "pages": 1,
      "has_next": false,
      "has_prev": false
    }
  }
}
```

#### Get Container Details

```http
GET /api/containers/{id}
```

**Authentication Required**

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "web-server",
    "image": "nginx:latest",
    "image_digest": "sha256:abc123...",
    "docker_id": "container_id_hash",
    "status": "running",
    "created_at": "2024-09-16T10:00:00Z",
    "updated_at": "2024-09-16T12:00:00Z",
    "config": {
      "image": "nginx:latest",
      "ports": [{"host": 8080, "container": 80, "protocol": "tcp"}],
      "environment": {"ENV": "production", "DEBUG": "false"},
      "volumes": [{"host": "/data", "container": "/app/data", "mode": "rw"}],
      "networks": ["bridge"],
      "labels": {"app": "web", "version": "1.0.0"},
      "restart_policy": "unless-stopped",
      "cpu_limit": "1.0",
      "memory_limit": "512MB"
    },
    "update_policy": {
      "policy": "auto",
      "strategy": "rolling",
      "schedule": "0 2 * * *",
      "rollback_on_failure": true,
      "health_check_timeout": 300
    },
    "health": {
      "status": "healthy",
      "last_check": "2024-09-16T12:00:00Z",
      "check_interval": 30,
      "retries": 3,
      "timeout": 10
    },
    "stats": {
      "uptime": "2h30m",
      "cpu_usage": 15.5,
      "memory_usage": "128MB",
      "memory_limit": "512MB",
      "network_rx": "1.2GB",
      "network_tx": "800MB",
      "block_io_read": "100MB",
      "block_io_write": "50MB"
    },
    "update_available": true,
    "latest_version": "nginx:1.25.0",
    "update_history": [
      {
        "id": 1,
        "from_image": "nginx:1.24.0",
        "to_image": "nginx:latest",
        "started_at": "2024-09-16T10:00:00Z",
        "completed_at": "2024-09-16T10:05:00Z",
        "status": "success",
        "strategy": "rolling"
      }
    ]
  }
}
```

#### Create Container

```http
POST /api/containers
```

**Authentication Required**

**Request Body:**
```json
{
  "name": "new-web-server",
  "image": "nginx:latest",
  "config": {
    "ports": [{"host": 8080, "container": 80, "protocol": "tcp"}],
    "environment": {"ENV": "production"},
    "volumes": [{"host": "/data", "container": "/app/data", "mode": "rw"}],
    "networks": ["bridge"],
    "labels": {"app": "web", "version": "1.0.0"},
    "restart_policy": "unless-stopped",
    "cpu_limit": "1.0",
    "memory_limit": "512MB"
  },
  "update_policy": {
    "policy": "auto",
    "strategy": "rolling",
    "schedule": "0 2 * * *",
    "rollback_on_failure": true
  },
  "health_check": {
    "enabled": true,
    "interval": 30,
    "timeout": 10,
    "retries": 3,
    "command": ["curl", "-f", "http://localhost/health"]
  },
  "start_immediately": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Container created successfully",
  "data": {
    "id": 2,
    "name": "new-web-server",
    "status": "starting",
    "created_at": "2024-09-16T14:00:00Z"
  }
}
```

#### Update Container

```http
PUT /api/containers/{id}
```

**Authentication Required**

**Request Body:**
```json
{
  "config": {
    "environment": {"ENV": "staging", "DEBUG": "true"},
    "memory_limit": "1GB"
  },
  "update_policy": {
    "schedule": "0 3 * * *"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Container updated successfully",
  "data": {
    "id": 1,
    "updated_at": "2024-09-16T14:30:00Z"
  }
}
```

#### Container Actions

##### Start Container

```http
POST /api/containers/{id}/start
```

**Authentication Required**

**Response:**
```json
{
  "success": true,
  "message": "Container started successfully",
  "data": {
    "id": 1,
    "status": "starting",
    "action_id": "action_uuid"
  }
}
```

##### Stop Container

```http
POST /api/containers/{id}/stop
```

**Authentication Required**

**Request Body (optional):**
```json
{
  "force": false,
  "timeout": 30
}
```

**Response:**
```json
{
  "success": true,
  "message": "Container stopped successfully",
  "data": {
    "id": 1,
    "status": "stopped",
    "action_id": "action_uuid"
  }
}
```

##### Restart Container

```http
POST /api/containers/{id}/restart
```

**Authentication Required**

**Request Body (optional):**
```json
{
  "timeout": 30
}
```

##### Update Container Image

```http
POST /api/containers/{id}/update
```

**Authentication Required**

**Request Body:**
```json
{
  "strategy": "rolling",
  "schedule": "immediate",
  "force": false,
  "backup": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Container update started",
  "data": {
    "update_id": "update_uuid",
    "status": "started",
    "estimated_duration": "5m"
  }
}
```

#### Container Logs

```http
GET /api/containers/{id}/logs
```

**Authentication Required**

**Query Parameters:**
- `lines` (int): Number of lines to retrieve (default: 100)
- `follow` (bool): Follow logs in real-time (use WebSocket for better performance)
- `timestamps` (bool): Include timestamps (default: true)
- `since` (string): Show logs since timestamp (ISO 8601)
- `until` (string): Show logs until timestamp (ISO 8601)
- `level` (string): Filter by log level

**Response:**
```json
{
  "success": true,
  "data": {
    "logs": [
      {
        "timestamp": "2024-09-16T14:00:00Z",
        "level": "info",
        "message": "Server started on port 80",
        "source": "stdout"
      },
      {
        "timestamp": "2024-09-16T14:01:00Z",
        "level": "info",
        "message": "Health check passed",
        "source": "stdout"
      }
    ],
    "meta": {
      "total_lines": 150,
      "truncated": false
    }
  }
}
```

#### Container Statistics

```http
GET /api/containers/{id}/stats
```

**Authentication Required**

**Response:**
```json
{
  "success": true,
  "data": {
    "current": {
      "cpu_usage": 15.5,
      "memory_usage": "128MB",
      "memory_limit": "512MB",
      "memory_percent": 25.0,
      "network_rx": "1.2GB",
      "network_tx": "800MB",
      "block_io_read": "100MB",
      "block_io_write": "50MB",
      "pids": 25
    },
    "history": [
      {
        "timestamp": "2024-09-16T14:00:00Z",
        "cpu_usage": 12.3,
        "memory_usage": "120MB",
        "network_rx_rate": "10MB/s",
        "network_tx_rate": "5MB/s"
      }
    ]
  }
}
```

### Image Management

#### List Images

```http
GET /api/images
```

**Authentication Required**

**Query Parameters:**
- `search` (string): Search by repository name
- `tag` (string): Filter by tag
- `registry` (string): Filter by registry

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "nginx:latest",
        "repository": "nginx",
        "tag": "latest",
        "registry": "docker.io",
        "digest": "sha256:abc123...",
        "size": "142MB",
        "created": "2024-09-16T10:00:00Z",
        "last_updated": "2024-09-16T12:00:00Z",
        "used_by": ["web-server", "api-server"],
        "update_available": true,
        "latest_digest": "sha256:def456...",
        "vulnerabilities": {
          "critical": 0,
          "high": 1,
          "medium": 3,
          "low": 5
        }
      }
    ]
  }
}
```

#### Check Image Updates

```http
POST /api/images/check-updates
```

**Authentication Required**

**Request Body:**
```json
{
  "images": ["nginx:latest", "redis:7"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "updates": [
      {
        "image": "nginx:latest",
        "current_digest": "sha256:abc123...",
        "latest_digest": "sha256:def456...",
        "update_available": true,
        "changelog_url": "https://github.com/nginx/nginx/releases",
        "vulnerability_changes": {
          "fixed": 2,
          "new": 0
        }
      }
    ]
  }
}
```

### Update Management

#### List Updates

```http
GET /api/updates
```

**Authentication Required**

**Query Parameters:**
- `status` (string): Filter by status (`pending`, `in_progress`, `completed`, `failed`)
- `container_id` (int): Filter by container ID
- `date_from` (string): Filter from date (ISO 8601)
- `date_to` (string): Filter to date (ISO 8601)

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "update_uuid",
        "container_id": 1,
        "container_name": "web-server",
        "from_image": "nginx:1.24.0",
        "to_image": "nginx:latest",
        "strategy": "rolling",
        "status": "completed",
        "started_at": "2024-09-16T10:00:00Z",
        "completed_at": "2024-09-16T10:05:00Z",
        "duration": "5m0s",
        "success": true,
        "rollback": false,
        "logs": [
          {
            "timestamp": "2024-09-16T10:00:00Z",
            "level": "info",
            "message": "Starting update process"
          }
        ]
      }
    ]
  }
}
```

#### Schedule Update

```http
POST /api/updates/schedule
```

**Authentication Required**

**Request Body:**
```json
{
  "container_ids": [1, 2, 3],
  "strategy": "rolling",
  "scheduled_at": "2024-09-16T02:00:00Z",
  "options": {
    "rollback_on_failure": true,
    "health_check_timeout": 300,
    "pre_update_script": "docker exec container backup_data.sh",
    "post_update_script": "docker exec container verify_data.sh"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Updates scheduled successfully",
  "data": {
    "batch_id": "batch_uuid",
    "scheduled_updates": 3,
    "scheduled_at": "2024-09-16T02:00:00Z"
  }
}
```

### User Management

#### List Users

```http
GET /api/users
```

**Authentication Required (Admin)**

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "email": "admin@example.com",
        "name": "Admin User",
        "role": "admin",
        "active": true,
        "last_login": "2024-09-16T12:00:00Z",
        "created_at": "2024-09-01T10:00:00Z",
        "permissions": ["*"]
      }
    ]
  }
}
```

#### Create User

```http
POST /api/users
```

**Authentication Required (Admin)**

**Request Body:**
```json
{
  "email": "newuser@example.com",
  "name": "New User",
  "password": "secure_password",
  "role": "operator",
  "permissions": ["container:read", "container:write"]
}
```

#### Update User Profile

```http
PUT /api/auth/profile
```

**Authentication Required**

**Request Body:**
```json
{
  "name": "Updated Name",
  "preferences": {
    "timezone": "UTC",
    "language": "en",
    "notifications": {
      "email": true,
      "push": false
    }
  }
}
```

### Notification Management

#### List Notifications

```http
GET /api/notifications
```

**Authentication Required**

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "type": "update_completed",
        "title": "Container Update Completed",
        "message": "web-server has been successfully updated to nginx:latest",
        "level": "info",
        "read": false,
        "created_at": "2024-09-16T10:05:00Z",
        "data": {
          "container_id": 1,
          "update_id": "update_uuid"
        }
      }
    ]
  }
}
```

#### Mark Notifications as Read

```http
PUT /api/notifications/{id}/read
```

**Authentication Required**

## WebSocket API

### Connection

Connect to WebSocket endpoint for real-time updates:

```
ws://localhost:8080/api/ws
```

**Authentication:**
Send JWT token as query parameter:
```
ws://localhost:8080/api/ws?token=<jwt_token>
```

### Message Format

All WebSocket messages use JSON format:

```json
{
  "type": "message_type",
  "timestamp": "2024-09-16T14:00:00Z",
  "data": {...}
}
```

### Event Types

#### Container Events

**Container Status Changed**
```json
{
  "type": "container_status_changed",
  "timestamp": "2024-09-16T14:00:00Z",
  "data": {
    "container_id": 1,
    "name": "web-server",
    "status": "running",
    "previous_status": "starting"
  }
}
```

**Container Update Started**
```json
{
  "type": "container_update_started",
  "timestamp": "2024-09-16T14:00:00Z",
  "data": {
    "update_id": "update_uuid",
    "container_id": 1,
    "container_name": "web-server",
    "strategy": "rolling"
  }
}
```

**Container Update Completed**
```json
{
  "type": "container_update_completed",
  "timestamp": "2024-09-16T14:05:00Z",
  "data": {
    "update_id": "update_uuid",
    "container_id": 1,
    "success": true,
    "duration": "5m0s"
  }
}
```

#### System Events

**System Alert**
```json
{
  "type": "system_alert",
  "timestamp": "2024-09-16T14:00:00Z",
  "data": {
    "level": "warning",
    "title": "High CPU Usage",
    "message": "System CPU usage is above 80%",
    "details": {
      "cpu_usage": 85.2,
      "threshold": 80
    }
  }
}
```

#### Log Streaming

**Container Logs**
```json
{
  "type": "container_logs",
  "timestamp": "2024-09-16T14:00:00Z",
  "data": {
    "container_id": 1,
    "log_entry": {
      "timestamp": "2024-09-16T14:00:00Z",
      "level": "info",
      "message": "Request processed successfully",
      "source": "stdout"
    }
  }
}
```

### Client Commands

#### Subscribe to Events

```json
{
  "type": "subscribe",
  "data": {
    "events": ["container_status_changed", "container_update_started"],
    "filters": {
      "container_ids": [1, 2, 3]
    }
  }
}
```

#### Unsubscribe from Events

```json
{
  "type": "unsubscribe",
  "data": {
    "events": ["container_logs"]
  }
}
```

## SDK Examples

### JavaScript/Node.js

#### Installation

```bash
npm install axios ws
```

#### Basic Usage

```javascript
const axios = require('axios');
const WebSocket = require('ws');

class DockerAutoAPI {
  constructor(baseURL, apiKey) {
    this.baseURL = baseURL;
    this.apiKey = apiKey;
    this.client = axios.create({
      baseURL: baseURL,
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json'
      }
    });
  }

  // Authentication
  async login(email, password) {
    const response = await this.client.post('/auth/login', {
      email,
      password
    });

    if (response.data.success) {
      this.apiKey = response.data.data.access_token;
      this.client.defaults.headers.Authorization = `Bearer ${this.apiKey}`;
    }

    return response.data;
  }

  // Container management
  async listContainers(params = {}) {
    const response = await this.client.get('/containers', { params });
    return response.data;
  }

  async getContainer(id) {
    const response = await this.client.get(`/containers/${id}`);
    return response.data;
  }

  async createContainer(containerData) {
    const response = await this.client.post('/containers', containerData);
    return response.data;
  }

  async updateContainer(id, updateData) {
    const response = await this.client.put(`/containers/${id}`, updateData);
    return response.data;
  }

  async startContainer(id) {
    const response = await this.client.post(`/containers/${id}/start`);
    return response.data;
  }

  async stopContainer(id, options = {}) {
    const response = await this.client.post(`/containers/${id}/stop`, options);
    return response.data;
  }

  async updateContainerImage(id, options = {}) {
    const response = await this.client.post(`/containers/${id}/update`, options);
    return response.data;
  }

  // WebSocket connection
  connectWebSocket() {
    const wsURL = this.baseURL.replace('http', 'ws') + `/ws?token=${this.apiKey}`;
    this.ws = new WebSocket(wsURL);

    this.ws.on('open', () => {
      console.log('WebSocket connected');
    });

    this.ws.on('message', (data) => {
      const message = JSON.parse(data);
      this.handleWebSocketMessage(message);
    });

    this.ws.on('error', (error) => {
      console.error('WebSocket error:', error);
    });

    return this.ws;
  }

  handleWebSocketMessage(message) {
    switch (message.type) {
      case 'container_status_changed':
        console.log('Container status changed:', message.data);
        break;
      case 'container_update_completed':
        console.log('Update completed:', message.data);
        break;
      default:
        console.log('Received message:', message);
    }
  }

  subscribeToEvents(events, filters = {}) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'subscribe',
        data: { events, filters }
      }));
    }
  }
}

// Usage example
const api = new DockerAutoAPI('http://localhost:8080/api');

async function example() {
  // Login
  const loginResult = await api.login('admin@example.com', 'password');
  console.log('Login result:', loginResult);

  // List containers
  const containers = await api.listContainers({ limit: 10 });
  console.log('Containers:', containers);

  // Connect to WebSocket for real-time updates
  const ws = api.connectWebSocket();

  // Subscribe to container events
  api.subscribeToEvents(['container_status_changed', 'container_update_completed']);

  // Create a new container
  const newContainer = await api.createContainer({
    name: 'test-container',
    image: 'nginx:latest',
    config: {
      ports: [{ host: 8080, container: 80, protocol: 'tcp' }]
    },
    start_immediately: true
  });

  console.log('Created container:', newContainer);
}

example().catch(console.error);
```

### Python

#### Installation

```bash
pip install requests websockets asyncio
```

#### Basic Usage

```python
import requests
import asyncio
import websockets
import json
from typing import Optional, Dict, Any

class DockerAutoAPI:
    def __init__(self, base_url: str, api_key: Optional[str] = None):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()

        if api_key:
            self.session.headers.update({
                'Authorization': f'Bearer {api_key}',
                'Content-Type': 'application/json'
            })

    def login(self, email: str, password: str) -> Dict[str, Any]:
        """Login and get access token"""
        response = self.session.post(
            f'{self.base_url}/auth/login',
            json={'email': email, 'password': password}
        )
        response.raise_for_status()

        data = response.json()
        if data.get('success'):
            self.api_key = data['data']['access_token']
            self.session.headers.update({
                'Authorization': f'Bearer {self.api_key}'
            })

        return data

    def list_containers(self, **params) -> Dict[str, Any]:
        """List all containers"""
        response = self.session.get(f'{self.base_url}/containers', params=params)
        response.raise_for_status()
        return response.json()

    def get_container(self, container_id: int) -> Dict[str, Any]:
        """Get container details"""
        response = self.session.get(f'{self.base_url}/containers/{container_id}')
        response.raise_for_status()
        return response.json()

    def create_container(self, container_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new container"""
        response = self.session.post(
            f'{self.base_url}/containers',
            json=container_data
        )
        response.raise_for_status()
        return response.json()

    def start_container(self, container_id: int) -> Dict[str, Any]:
        """Start a container"""
        response = self.session.post(f'{self.base_url}/containers/{container_id}/start')
        response.raise_for_status()
        return response.json()

    def stop_container(self, container_id: int, **options) -> Dict[str, Any]:
        """Stop a container"""
        response = self.session.post(
            f'{self.base_url}/containers/{container_id}/stop',
            json=options
        )
        response.raise_for_status()
        return response.json()

    def update_container_image(self, container_id: int, **options) -> Dict[str, Any]:
        """Update container image"""
        response = self.session.post(
            f'{self.base_url}/containers/{container_id}/update',
            json=options
        )
        response.raise_for_status()
        return response.json()

    async def connect_websocket(self, message_handler=None):
        """Connect to WebSocket for real-time updates"""
        ws_url = self.base_url.replace('http', 'ws') + f'/ws?token={self.api_key}'

        async with websockets.connect(ws_url) as websocket:
            print("WebSocket connected")

            if message_handler:
                async for message in websocket:
                    data = json.loads(message)
                    await message_handler(data)
            else:
                async for message in websocket:
                    data = json.loads(message)
                    print(f"Received: {data}")

    async def subscribe_to_events(self, websocket, events, filters=None):
        """Subscribe to specific events"""
        await websocket.send(json.dumps({
            'type': 'subscribe',
            'data': {
                'events': events,
                'filters': filters or {}
            }
        }))

# Usage example
async def main():
    api = DockerAutoAPI('http://localhost:8080/api')

    # Login
    login_result = api.login('admin@example.com', 'password')
    print(f"Login result: {login_result}")

    # List containers
    containers = api.list_containers(limit=10)
    print(f"Containers: {containers}")

    # WebSocket message handler
    async def handle_message(message):
        if message['type'] == 'container_status_changed':
            print(f"Container status changed: {message['data']}")
        elif message['type'] == 'container_update_completed':
            print(f"Update completed: {message['data']}")
        else:
            print(f"Received message: {message}")

    # Connect to WebSocket (this will run indefinitely)
    await api.connect_websocket(handle_message)

if __name__ == '__main__':
    asyncio.run(main())
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"

    "github.com/gorilla/websocket"
)

type DockerAutoAPI struct {
    BaseURL string
    APIKey  string
    client  *http.Client
}

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Success bool `json:"success"`
    Data    struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    } `json:"data"`
}

func NewDockerAutoAPI(baseURL string) *DockerAutoAPI {
    return &DockerAutoAPI{
        BaseURL: baseURL,
        client:  &http.Client{},
    }
}

func (api *DockerAutoAPI) Login(email, password string) error {
    loginReq := LoginRequest{Email: email, Password: password}

    jsonData, err := json.Marshal(loginReq)
    if err != nil {
        return err
    }

    resp, err := api.client.Post(
        api.BaseURL+"/auth/login",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var loginResp LoginResponse
    if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
        return err
    }

    if loginResp.Success {
        api.APIKey = loginResp.Data.AccessToken
    }

    return nil
}

func (api *DockerAutoAPI) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
    var req *http.Request
    var err error

    if body != nil {
        jsonData, _ := json.Marshal(body)
        req, err = http.NewRequest(method, api.BaseURL+endpoint, bytes.NewBuffer(jsonData))
        req.Header.Set("Content-Type", "application/json")
    } else {
        req, err = http.NewRequest(method, api.BaseURL+endpoint, nil)
    }

    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+api.APIKey)

    return api.client.Do(req)
}

func (api *DockerAutoAPI) ListContainers() (map[string]interface{}, error) {
    resp, err := api.makeRequest("GET", "/containers", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result, nil
}

func (api *DockerAutoAPI) ConnectWebSocket() (*websocket.Conn, error) {
    u := url.URL{
        Scheme:   "ws",
        Host:     "localhost:8080",
        Path:     "/api/ws",
        RawQuery: "token=" + api.APIKey,
    }

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        return nil, err
    }

    return conn, nil
}

func main() {
    api := NewDockerAutoAPI("http://localhost:8080/api")

    // Login
    if err := api.Login("admin@example.com", "password"); err != nil {
        fmt.Printf("Login failed: %v\n", err)
        return
    }

    fmt.Println("Login successful")

    // List containers
    containers, err := api.ListContainers()
    if err != nil {
        fmt.Printf("Failed to list containers: %v\n", err)
        return
    }

    fmt.Printf("Containers: %+v\n", containers)

    // Connect to WebSocket
    conn, err := api.ConnectWebSocket()
    if err != nil {
        fmt.Printf("WebSocket connection failed: %v\n", err)
        return
    }
    defer conn.Close()

    fmt.Println("WebSocket connected")

    // Read messages
    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            fmt.Printf("Read error: %v\n", err)
            break
        }

        var wsMessage map[string]interface{}
        if err := json.Unmarshal(message, &wsMessage); err != nil {
            fmt.Printf("JSON decode error: %v\n", err)
            continue
        }

        fmt.Printf("Received: %+v\n", wsMessage)
    }
}
```

## Webhook Integration

### Webhook Configuration

Configure webhooks to receive notifications about container events:

```json
{
  "webhook_url": "https://your-service.com/webhook",
  "events": [
    "container_created",
    "container_updated",
    "container_started",
    "container_stopped",
    "update_completed",
    "update_failed"
  ],
  "secret": "your-webhook-secret",
  "headers": {
    "X-Custom-Header": "value"
  }
}
```

### Webhook Payload

All webhook payloads follow this format:

```json
{
  "event": "container_updated",
  "timestamp": "2024-09-16T14:00:00Z",
  "data": {
    "container": {
      "id": 1,
      "name": "web-server",
      "image": "nginx:latest"
    },
    "update": {
      "id": "update_uuid",
      "status": "completed",
      "duration": "5m0s"
    }
  },
  "signature": "sha256=hash_value"
}
```

### Webhook Verification

Verify webhook authenticity using HMAC-SHA256:

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');

  const actualSignature = signature.replace('sha256=', '');

  return crypto.timingSafeEqual(
    Buffer.from(expectedSignature),
    Buffer.from(actualSignature)
  );
}
```

## API Best Practices

### Authentication

1. **Store tokens securely**: Never store JWT tokens in localStorage or cookies without proper security measures
2. **Token refresh**: Implement automatic token refresh to maintain authentication
3. **Logout properly**: Always call the logout endpoint to invalidate tokens

### Error Handling

1. **Check response status**: Always check the `success` field in API responses
2. **Handle rate limits**: Implement exponential backoff for rate limit errors
3. **Graceful degradation**: Handle API failures gracefully in your application

### Performance

1. **Pagination**: Use pagination for large datasets
2. **Filtering**: Apply filters on the server side rather than client side
3. **Caching**: Cache frequently accessed data with appropriate TTL
4. **WebSocket for real-time**: Use WebSocket connections for real-time updates instead of polling

### Security

1. **HTTPS only**: Always use HTTPS in production
2. **Validate inputs**: Validate all inputs on both client and server side
3. **Rate limiting**: Respect rate limits and implement client-side throttling
4. **Webhook security**: Always verify webhook signatures

### Monitoring

1. **Request logging**: Log all API requests for debugging and monitoring
2. **Error tracking**: Implement proper error tracking and alerting
3. **Performance monitoring**: Monitor API response times and error rates

---

**Last Updated**: September 16, 2024
**Version**: 2.0.0

For additional support and examples, visit our [GitHub repository](https://github.com/your-org/docker-auto) or check the [developer documentation](docs/developer/).