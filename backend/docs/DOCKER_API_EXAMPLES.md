# Docker核心功能API使用示例

本文档展示了新实现的Docker核心功能API的使用方法和响应示例。

## 目录
- [镜像拉取](#镜像拉取)
- [镜像删除](#镜像删除)
- [容器更新策略](#容器更新策略)
- [镜像安全扫描](#镜像安全扫描)
- [实时进度追踪](#实时进度追踪)
- [WebSocket连接](#websocket连接)
- [回滚操作](#回滚操作)

## 镜像拉取

### 拉取镜像
```http
POST /api/images/nginx/pull
Authorization: Bearer <token>
Content-Type: application/json

{
  "tag": "latest",
  "force": false,
  "registry_url": "docker.io"
}
```

**响应示例:**
```json
{
  "success": true,
  "message": "Image pull started",
  "data": {
    "message": "Image pull started",
    "image": "nginx:latest",
    "status": "starting",
    "progress_id": "nginx:latest",
    "start_time": "2024-01-15T10:30:00Z",
    "force": false
  }
}
```

### 获取拉取进度
```http
GET /api/progress/pull?image_name=nginx:latest
Authorization: Bearer <token>
```

**响应示例:**
```json
{
  "success": true,
  "data": {
    "image_name": "nginx:latest",
    "status": "downloading",
    "progress": {
      "sha256:abc123": {
        "id": "sha256:abc123",
        "status": "Downloading",
        "progress": "50%",
        "current": 52428800,
        "total": 104857600
      }
    },
    "start_time": "2024-01-15T10:30:00Z",
    "completed": false,
    "total_size": 104857600,
    "downloaded_size": 52428800
  }
}
```

## 镜像删除

### 删除镜像
```http
DELETE /api/images/nginx?tag=latest&force=false
Authorization: Bearer <token>
```

**响应示例:**
```json
{
  "success": true,
  "message": "Image removed successfully",
  "data": {
    "image": "nginx:latest",
    "image_id": "sha256:def456789",
    "deleted": [
      "sha256:layer1",
      "sha256:layer2"
    ],
    "untagged": [
      "nginx:latest"
    ],
    "dependency_info": [],
    "force": false,
    "success": true
  }
}
```

### 删除失败（有依赖）
```json
{
  "success": false,
  "message": "Cannot remove image: image has dependencies",
  "error": "image has dependencies: [container web-server (abc123) - running]"
}
```

## 容器更新策略

### 滚动更新
```http
POST /api/containers/123/update
Authorization: Bearer <token>
Content-Type: application/json

{
  "new_image": "nginx:1.21",
  "strategy": "rolling",
  "health_check_timeout": 60,
  "rollback_on_failure": true,
  "create_backup": true,
  "rolling_config": {
    "max_unavailable": 1,
    "max_surge": 1,
    "batch_size": 1,
    "batch_delay": 30
  }
}
```

**响应示例:**
```json
{
  "success": true,
  "message": "Container update started",
  "data": {
    "update_id": "update_123_1705318200",
    "strategy": "rolling",
    "status": "started",
    "container_id": 123,
    "old_image": "nginx:1.20",
    "new_image": "nginx:1.21",
    "estimated_duration": "5m"
  }
}
```

### 蓝绿部署
```http
POST /api/containers/123/update
Authorization: Bearer <token>
Content-Type: application/json

{
  "new_image": "nginx:1.21",
  "strategy": "blue_green",
  "health_check_timeout": 120,
  "blue_green_config": {
    "network_name": "app-network",
    "warmup_time": 30,
    "test_endpoints": [
      "http://localhost:8080/health",
      "http://localhost:8080/ready"
    ]
  }
}
```

### 金丝雀发布
```http
POST /api/containers/123/update
Authorization: Bearer <token>
Content-Type: application/json

{
  "new_image": "nginx:1.21",
  "strategy": "canary",
  "canary_config": {
    "traffic_percent": 10,
    "canary_duration": 300,
    "auto_promote": true,
    "metrics_threshold": {
      "error_rate": 0.05,
      "response_time": 500
    }
  }
}
```

## 镜像安全扫描

### 扫描镜像
```http
GET /api/images/nginx/security?tag=latest
Authorization: Bearer <token>
```

**响应示例:**
```json
{
  "success": true,
  "data": {
    "image_name": "nginx:latest",
    "image_id": "sha256:abc123",
    "scan_time": "2024-01-15T10:35:00Z",
    "scan_duration": "45s",
    "passed": true,
    "grade": "A",
    "score": 85.5,
    "total_vulns": 3,
    "critical_vulns": 0,
    "high_vulns": 0,
    "medium_vulns": 2,
    "low_vulns": 1,
    "vulnerabilities": [
      {
        "cve": "CVE-2023-1234",
        "severity": "medium",
        "description": "Buffer overflow in package xyz",
        "package": "libxyz",
        "version": "1.0.1",
        "fixed_in": "1.0.2"
      }
    ],
    "compliance_checks": [
      {
        "check_id": "CIS-4.1",
        "name": "User for container created",
        "category": "User Management",
        "severity": "Medium",
        "passed": true,
        "description": "Ensure that a user for the container has been created"
      }
    ],
    "recommendations": [
      "Update to newer base image to fix medium severity vulnerabilities",
      "Image meets security standards - monitor for updates"
    ]
  }
}
```

### 版本比较
```http
POST /api/images/compare
Authorization: Bearer <token>
Content-Type: application/json

{
  "current_image": "nginx",
  "current_tag": "1.20",
  "latest_image": "nginx",
  "latest_tag": "latest"
}
```

**响应示例:**
```json
{
  "success": true,
  "data": {
    "current_version": {
      "tag": "1.20",
      "created": "2023-12-01T10:00:00Z",
      "size": 104857600,
      "is_stable": true
    },
    "latest_version": {
      "tag": "1.21",
      "created": "2024-01-10T15:30:00Z",
      "size": 108003328,
      "is_latest": true,
      "is_stable": true
    },
    "update_available": true,
    "versions_behind": 2,
    "security_risk": "MEDIUM",
    "recommendation": "Schedule update within a week - moderate security risk"
  }
}
```

## 实时进度追踪

### 获取操作进度
```http
GET /api/progress/operation/update_123_1705318200
Authorization: Bearer <token>
```

**响应示例:**
```json
{
  "success": true,
  "data": {
    "id": "update_123_1705318200",
    "type": "container_update",
    "status": "running",
    "progress": 65.0,
    "message": "Performing health check",
    "start_time": "2024-01-15T10:30:00Z",
    "duration": "2m30s",
    "user_id": 1,
    "container_id": "123",
    "image_name": "nginx:1.21",
    "steps": [
      {
        "name": "stop_container",
        "status": "completed",
        "progress": 100.0,
        "message": "Container stopped successfully",
        "start_time": "2024-01-15T10:30:00Z",
        "end_time": "2024-01-15T10:30:15Z",
        "duration": "15s"
      },
      {
        "name": "create_new_container",
        "status": "completed",
        "progress": 100.0,
        "message": "New container created",
        "start_time": "2024-01-15T10:30:15Z",
        "end_time": "2024-01-15T10:31:00Z",
        "duration": "45s"
      },
      {
        "name": "health_check",
        "status": "running",
        "progress": 60.0,
        "message": "Performing health check",
        "start_time": "2024-01-15T10:31:00Z",
        "details": {
          "checks_performed": 3,
          "checks_remaining": 2
        }
      }
    ],
    "last_update": "2024-01-15T10:32:30Z"
  }
}
```

### 获取用户操作列表
```http
GET /api/progress/user?status=running&limit=10
Authorization: Bearer <token>
```

**响应示例:**
```json
{
  "success": true,
  "data": [
    {
      "id": "update_123_1705318200",
      "type": "container_update",
      "status": "running",
      "progress": 65.0,
      "start_time": "2024-01-15T10:30:00Z",
      "container_id": "123",
      "image_name": "nginx:1.21"
    },
    {
      "id": "pull_redis_1705318100",
      "type": "image_pull",
      "status": "completed",
      "progress": 100.0,
      "start_time": "2024-01-15T10:28:20Z",
      "end_time": "2024-01-15T10:29:45Z",
      "image_name": "redis:7"
    }
  ]
}
```

## WebSocket连接

### 建立WebSocket连接
```javascript
// 前端JavaScript示例
const token = localStorage.getItem('authToken');
const ws = new WebSocket(`wss://your-domain.com/api/ws/connect?channels=operations,alerts&token=${token}`);

ws.onopen = function(event) {
    console.log('WebSocket连接已建立');
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('收到消息:', message);

    switch(message.type) {
        case 'welcome':
            console.log('欢迎消息:', message.data);
            break;
        case 'progress':
            updateProgressUI(message.data);
            break;
        case 'alert':
            showAlert(message.data);
            break;
        case 'success':
            showSuccess(message.data);
            break;
        case 'error':
            showError(message.data);
            break;
    }
};
```

### WebSocket消息示例

**欢迎消息:**
```json
{
  "type": "welcome",
  "channel": "system",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "connection_id": "conn_abc123",
    "user_id": 1,
    "channels": ["operations", "alerts", "user"],
    "server_time": "2024-01-15T10:30:00Z",
    "features": [
      "real_time_progress",
      "operation_notifications",
      "security_alerts",
      "system_status"
    ]
  }
}
```

**进度更新消息:**
```json
{
  "type": "progress",
  "channel": "operations",
  "timestamp": "2024-01-15T10:32:15Z",
  "data": {
    "id": "update_123_1705318200",
    "type": "container_update",
    "status": "running",
    "progress": 75.0,
    "message": "Health check passed, finalizing update",
    "container_id": "123",
    "steps": [...]
  },
  "user_id": 1
}
```

**安全告警消息:**
```json
{
  "type": "alert",
  "channel": "alerts",
  "timestamp": "2024-01-15T10:35:00Z",
  "data": {
    "level": "critical",
    "title": "Critical Vulnerabilities Found",
    "message": "Image nginx:latest contains 5 critical vulnerabilities",
    "data": {
      "image": "nginx:latest",
      "critical_vulns": 5,
      "total_vulns": 12,
      "security_grade": "F"
    }
  },
  "user_id": 1
}
```

## 回滚操作

### 执行回滚
```http
POST /api/containers/123/rollback
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "Update failed health check",
  "force": false,
  "health_check": true,
  "timeout": 300
}
```

**响应示例:**
```json
{
  "success": true,
  "message": "Rollback initiated",
  "data": {
    "rollback_id": "rollback_123_1705318500",
    "container_id": "123",
    "target_image": "nginx:1.20",
    "status": "pending",
    "reason": "Update failed health check"
  }
}
```

### 获取回滚历史
```http
GET /api/progress/rollbacks?limit=10
Authorization: Bearer <token>
```

**响应示例:**
```json
{
  "success": true,
  "data": [
    {
      "id": "rollback_123_1705318500",
      "container_id": "123",
      "original_image": "nginx:1.21",
      "target_image": "nginx:1.20",
      "rollback_reason": "Update failed health check",
      "rollback_trigger": "auto_failure",
      "status": "completed",
      "start_time": "2024-01-15T10:35:00Z",
      "end_time": "2024-01-15T10:37:30Z",
      "duration": "2m30s",
      "steps": [
        {
          "name": "stop_container",
          "status": "completed",
          "duration": "10s"
        },
        {
          "name": "create_new_container",
          "status": "completed",
          "duration": "45s"
        },
        {
          "name": "health_check",
          "status": "completed",
          "duration": "1m30s"
        }
      ],
      "health_check_result": {
        "healthy": true,
        "checks_performed": 3,
        "response_time": "150ms"
      }
    }
  ]
}
```

## 错误处理

### 权限不足
```json
{
  "success": false,
  "message": "Permission denied: operation image_remove not allowed for role developer",
  "error": "Forbidden"
}
```

### 服务不可用
```json
{
  "success": false,
  "message": "Docker service unavailable",
  "error": "failed to connect to Docker daemon"
}
```

### 操作失败
```json
{
  "success": false,
  "message": "Container update failed: health check timeout",
  "error": "Health check failed after 3 attempts",
  "data": {
    "operation_id": "update_123_1705318200",
    "rollback_performed": true,
    "rollback_id": "rollback_123_1705318500"
  }
}
```

## 使用建议

1. **进度追踪**: 对于长时间运行的操作（如镜像拉取、容器更新），建议同时使用WebSocket和轮询API来确保进度更新的可靠性。

2. **错误处理**: 始终检查API响应的`success`字段，并妥善处理错误情况。

3. **安全扫描**: 在生产环境中部署新镜像前，建议先进行安全扫描。

4. **回滚策略**: 重要的生产环境更新应该启用自动回滚功能。

5. **权限管理**: 根据用户角色合理分配Docker操作权限。

6. **监控告警**: 订阅WebSocket告警频道以及时收到安全威胁通知。

这些API提供了完整的Docker容器生命周期管理功能，支持生产环境的高可用性和安全性要求。