# Docker Auto Management System - WebSocket API 文档

## 概述

本文档详细描述了Docker Auto Management System的实时WebSocket通信API，提供了容器监控、事件订阅和实时通知的全面技术规范。

## 连接协议

### WebSocket 连接地址
- **协议**: `ws://` 或 `wss://`
- **端点**: `/ws`

### 认证方式
WebSocket连接要求JWT认证。可通过以下两种方式提供令牌：

1. **查询参数**:
   ```
   ws://localhost:8080/ws?token=your_jwt_token
   ```

2. **Authorization头**:
   ```
   Authorization: Bearer your_jwt_token
   ```

## 消息类型

### 客户端消息类型 (ClientMessage)

| 类型 | 描述 | 必需字段 |
|------|------|----------|
| `subscribe` | 订阅特定事件主题 | `topic` |
| `unsubscribe` | 取消订阅特定事件主题 | `topic` |
| `ping` | 保持连接活跃的心跳消息 | - |
| `ack` | 确认接收消息 | - |

### 服务器消息类型 (ServerMessage)

| 类型 | 描述 |
|------|------|
| `event` | 实时事件通知 |
| `metrics_update` | 容器指标更新 |
| `container_event` | 容器状态变更事件 |
| `monitoring_alert` | 系统监控告警 |
| `error` | 错误消息 |
| `pong` | 心跳响应 |

## 订阅主题

### 容器状态主题 (`container.status`)
订阅容器生命周期事件：启动、停止、更新、错误等。

### 容器日志主题 (`container.logs`)
订阅容器日志流和错误事件。

### 容器指标主题 (`container.metrics`)
实时接收容器资源使用指标。

### 容器事件主题 (`container.events`)
获取Docker容器的所有关键事件。

### 监控告警主题 (`monitoring.alerts`)
接收系统和容器的健康告警。

### 镜像更新主题 (`image.update`)
获取镜像更新状态和进度。

### 系统健康主题 (`system.health`)
监控整体系统健康状态。

### 任务进度主题 (`task.progress`)
跟踪后台任务的执行状态。

### 用户通知主题 (`user.notification`)
接收针对特定用户的个性化通知。

## 消息示例

### 订阅请求示例
```json
{
  "type": "subscribe",
  "topic": "container.status",
  "message_id": "unique_message_id"
}
```

### 服务器事件响应示例
```json
{
  "type": "event",
  "topic": "container.status",
  "data": {
    "container_id": "abc123",
    "status": "started",
    "timestamp": 1632825600
  },
  "timestamp": 1632825600
}
```

## 速率限制

- 每分钟最多发送100条消息
- 超出限制将收到错误消息

## 错误处理

### 常见错误代码

| 错误类型 | 描述 |
|----------|------|
| `unauthorized` | 认证失败 |
| `rate_limit_exceeded` | 超过消息发送速率 |
| `invalid_topic` | 无效的订阅主题 |
| `connection_error` | 连接异常 |

## 最佳实践

1. 始终使用安全的WebSocket连接 (wss://)
2. 实现自动重连机制
3. 处理所有可能的错误场景
4. 控制订阅的主题数量
5. 及时取消不再需要的订阅

## 安全建议

- 妥善保管JWT令牌
- 使用短期令牌
- 实现令牌刷新机制
- 遵守最小权限原则

## SDK和客户端库

- 提供官方Go和JavaScript WebSocket客户端
- 支持自动重连和高级订阅管理

## 版本信息

- **当前API版本**: 1.0
- **最后更新**: 2025-09-26
- **兼容性**: 与Docker Auto Management System v1.x完全兼容