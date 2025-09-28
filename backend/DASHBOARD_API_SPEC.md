# Dashboard API 完整规格说明

## 概述
Dashboard API 提供完整的Docker容器生命周期管理系统数据聚合和实时监控功能。所有端点返回真实的Docker和系统数据，支持高并发访问和实时数据推送。

## 基础信息
- **基础URL**: `http://localhost:8080/api/dashboard`
- **认证方式**: JWT Token (Bearer Token)
- **数据格式**: JSON
- **响应编码**: UTF-8

## 认证
所有API端点都需要JWT认证。在请求头中包含：
```
Authorization: Bearer <JWT_TOKEN>
```

## 核心API端点

### 1. 系统概览
**GET** `/api/dashboard/overview`

获取系统整体概览数据，包含容器统计、资源使用、安全状态、更新活动和系统健康状况。

**响应示例:**
```json
{
  "success": true,
  "data": {
    "containerStats": {
      "total": 15,
      "running": 12,
      "stopped": 2,
      "paused": 1,
      "restarting": 0,
      "unhealthy": 0
    },
    "resourceUsage": {
      "cpu": {
        "used": 45.2,
        "total": 100,
        "percentage": 45.2,
        "unit": "percent"
      },
      "memory": {
        "used": 8589934592,
        "total": 17179869184,
        "percentage": 50.0,
        "unit": "bytes"
      },
      "disk": {
        "used": 107374182400,
        "total": 536870912000,
        "percentage": 20.0,
        "unit": "bytes"
      },
      "network": {
        "bytesIn": 1073741824,
        "bytesOut": 2147483648,
        "packetsIn": 1000000,
        "packetsOut": 1200000
      }
    },
    "securityStatus": {
      "vulnerabilitiesFound": 5,
      "criticalVulns": 0,
      "highVulns": 1,
      "mediumVulns": 2,
      "lowVulns": 2,
      "securityScore": 85.5,
      "lastScanTime": "2023-12-25T10:30:00Z"
    },
    "updateActivity": {
      "pendingUpdates": 3,
      "recentUpdates": 8,
      "failedUpdates": 1,
      "lastUpdateTime": "2023-12-25T09:15:00Z",
      "autoUpdateEnabled": true
    },
    "systemHealth": {
      "overallStatus": "healthy",
      "healthyServices": 4,
      "unhealthyServices": 0,
      "healthChecks": [
        {
          "name": "Docker Daemon",
          "status": "healthy",
          "message": "Docker daemon is responsive",
          "timestamp": "2023-12-25T10:35:00Z"
        }
      ]
    },
    "lastUpdated": "2023-12-25T10:35:00Z"
  }
}
```

### 2. 容器统计详情
**GET** `/api/dashboard/container-stats`

获取详细的容器统计信息，包含每个容器的资源使用情况。

**响应示例:**
```json
{
  "success": true,
  "data": {
    "containers": [
      {
        "containerId": "abc123def456",
        "name": "nginx-web",
        "image": "nginx:1.21",
        "status": "running",
        "state": "running",
        "restartCount": 0,
        "cpuStats": {
          "cpuUsage": {
            "totalUsage": 1234567890,
            "percpuUsage": [123456789, 987654321],
            "usageInKernelmode": 100000000,
            "usageInUsermode": 200000000
          },
          "systemCpuUsage": 9876543210,
          "onlineCpus": 2
        },
        "memoryStats": {
          "usage": 134217728,
          "maxUsage": 268435456,
          "limit": 1073741824,
          "stats": {
            "cache": 12345678,
            "rss": 87654321
          }
        },
        "lastUpdated": "2023-12-25T10:35:00Z"
      }
    ],
    "totalCount": 15,
    "runningCount": 12,
    "stoppedCount": 2,
    "lastUpdated": "2023-12-25T10:35:00Z"
  }
}
```

### 3. 资源使用趋势
**GET** `/api/dashboard/resource-metrics`

获取系统资源使用的当前状态和历史趋势。

**查询参数:**
- `hours` (可选): 获取过去N小时的数据，默认1小时

**响应示例:**
```json
{
  "success": true,
  "data": {
    "current": {
      "cpu": {
        "used": 45.2,
        "total": 100,
        "percentage": 45.2,
        "unit": "percent"
      },
      "memory": {
        "used": 8589934592,
        "total": 17179869184,
        "percentage": 50.0,
        "unit": "bytes"
      }
    },
    "trends": {
      "cpu": {
        "timestamps": ["2023-12-25T09:35:00Z", "2023-12-25T10:35:00Z"],
        "values": [42.1, 45.2],
        "metricName": "CPU Usage",
        "unit": "percent"
      },
      "memory": {
        "timestamps": ["2023-12-25T09:35:00Z", "2023-12-25T10:35:00Z"],
        "values": [48.5, 50.0],
        "metricName": "Memory Usage",
        "unit": "percent"
      }
    },
    "lastUpdated": "2023-12-25T10:35:00Z"
  }
}
```

### 4. 安全状态
**GET** `/api/dashboard/security-status`

获取系统安全状态，包括漏洞扫描结果和合规状态。

**响应示例:**
```json
{
  "success": true,
  "data": {
    "totalContainers": 15,
    "scannedContainers": 15,
    "vulnerableContainers": 3,
    "totalVulnerabilities": 5,
    "criticalVulns": 0,
    "highVulns": 1,
    "mediumVulns": 2,
    "lowVulns": 2,
    "securityScore": 85.5,
    "lastScanTime": "2023-12-25T10:30:00Z",
    "topVulnerabilities": [
      {
        "id": "1",
        "cve": "CVE-2023-1234",
        "severity": "high",
        "description": "Critical vulnerability in OpenSSL",
        "package": "openssl",
        "version": "1.1.1f",
        "fixedIn": "1.1.1g",
        "container": "nginx-web"
      }
    ],
    "riskDistribution": {
      "critical": 0,
      "high": 1,
      "medium": 2,
      "low": 2
    },
    "complianceStatus": {
      "overallScore": 85.0,
      "policies": {
        "no_critical_vulns": true,
        "limited_high_vulns": true,
        "regular_scanning": true
      },
      "recommendations": [
        "Update OpenSSL to latest version"
      ],
      "lastAssessment": "2023-12-25T10:30:00Z"
    }
  }
}
```

## 更新活动API

### 5. 获取最近更新
**GET** `/api/dashboard/updates/recent`

**查询参数:**
- `limit` (可选): 限制返回数量，默认10

### 6. 获取待处理更新
**GET** `/api/dashboard/updates/pending`

### 7. 触发更新
**POST** `/api/dashboard/updates/trigger`

**请求体:**
```json
{
  "containerId": "abc123def456",
  "strategy": {
    "autoUpdate": false,
    "rollbackOnFailure": true,
    "healthCheckTimeout": "60s",
    "preUpdateCommands": ["docker exec container backup"],
    "postUpdateCommands": ["docker exec container verify"]
  }
}
```

### 8. 更新历史
**GET** `/api/dashboard/updates/history`

**查询参数:**
- `page` (可选): 页码，默认1
- `limit` (可选): 每页数量，默认20

## 安全扫描API

### 9. 安全概览
**GET** `/api/dashboard/security/overview`

### 10. 触发安全扫描
**POST** `/api/dashboard/security/scan`

**请求体:**
```json
{
  "containerId": "abc123def456",  // 可选，扫描特定容器
  "scanAll": true                 // 扫描所有容器
}
```

## WebSocket实时数据

### WebSocket连接
**WebSocket** `/ws/dashboard`

**连接参数:**
- `token`: JWT认证令牌

**支持的订阅主题:**
- `dashboard:overview` - 系统概览更新 (30秒间隔)
- `dashboard:containers` - 容器状态更新 (10秒间隔)
- `dashboard:resources` - 资源使用更新 (5秒间隔)
- `dashboard:security` - 安全状态更新 (5分钟间隔)
- `dashboard:updates` - 更新活动通知 (1分钟间隔)
- `dashboard:health` - 健康检查更新 (15秒间隔)
- `dashboard:alerts` - 系统告警 (实时)

**订阅消息格式:**
```json
{
  "type": "subscribe",
  "topics": ["dashboard:overview", "dashboard:resources"]
}
```

**数据消息格式:**
```json
{
  "type": "data",
  "topic": "dashboard:overview",
  "data": { /* 相应API端点的数据 */ },
  "timestamp": "2023-12-25T10:35:00Z",
  "messageId": "dashboard:overview_1703509500123456789"
}
```

### WebSocket管理API

### 11. WebSocket统计
**GET** `/api/dashboard/ws/stats`

### 12. 广播告警
**POST** `/api/dashboard/ws/broadcast-alert`

**请求体:**
```json
{
  "type": "security",
  "message": "发现高危漏洞",
  "severity": "critical"
}
```

## 错误响应格式

所有API错误响应遵循统一格式：
```json
{
  "success": false,
  "error": "错误描述",
  "details": "详细错误信息",
  "code": "ERROR_CODE"
}
```

**常见HTTP状态码:**
- `200 OK` - 成功
- `202 Accepted` - 异步操作已接受
- `400 Bad Request` - 请求参数错误
- `401 Unauthorized` - 未授权
- `403 Forbidden` - 权限不足
- `404 Not Found` - 资源不存在
- `500 Internal Server Error` - 服务器内部错误

## 性能特性

- **缓存策略**: Redis缓存，不同数据类型使用不同TTL
  - 系统概览: 30秒
  - 容器统计: 10秒
  - 资源指标: 5秒
  - 安全状态: 5分钟
  - 健康检查: 15秒

- **并发支持**: 支持500+并发连接
- **响应时间**: API响应 < 200ms，WebSocket推送延迟 < 100ms
- **数据刷新**: 后台自动数据刷新，减少实时查询压力

## 扩展性

系统支持以下扩展：
- 自定义安全扫描器集成
- 多Docker主机监控
- 自定义告警规则
- 第三方漏洞数据库集成
- 自定义仪表板指标

## 使用示例

### JavaScript/TypeScript 客户端
```javascript
// REST API调用
const response = await fetch('/api/dashboard/overview', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
});
const data = await response.json();

// WebSocket连接
const ws = new WebSocket(`ws://localhost:8080/ws/dashboard?token=${token}`);
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

// 订阅主题
ws.send(JSON.stringify({
  type: 'subscribe',
  topics: ['dashboard:overview', 'dashboard:resources']
}));
```

### cURL 示例
```bash
# 获取系统概览
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/dashboard/overview

# 触发安全扫描
curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"scanAll": true}' \
     http://localhost:8080/api/dashboard/security/scan
```

此API规格提供了完整的Dashboard数据访问能力，支持实时监控、安全扫描、更新管理等企业级功能。