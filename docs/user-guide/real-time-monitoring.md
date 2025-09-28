# Docker Auto Management System - 实时监控用户指南

## 简介

Docker Auto Management System提供全面的实时监控功能，使您能够持续追踪容器和系统的运行状态。本指南将帮助您充分利用实时监控特性。

## 监控功能概览

### 容器状态监控
实时追踪Docker容器的生命周期状态，包括：
- 容器启动
- 容器停止
- 容器重启
- 容器错误
- 容器资源使用情况

### 系统资源监控
获取系统级别的资源使用情况：
- CPU利用率
- 内存消耗
- 网络流量
- 磁盘I/O

### 告警和通知系统
智能告警机制，提供：
- 资源阈值告警
- 容器健康状态预警
- 异常事件通知

## 实时监控接入方式

### Web界面监控
1. 登录Docker Auto Management System
2. 导航至"仪表板"
3. 选择"实时监控"标签页
4. 实时查看容器和系统状态

### WebSocket实时通信
通过WebSocket获取实时数据流，支持多种编程语言：

#### JavaScript客户端示例
```javascript
const socket = new WebSocket('wss://your-domain.com/ws?token=YOUR_JWT_TOKEN');

socket.onopen = function() {
    // 订阅容器状态
    socket.send(JSON.stringify({
        type: 'subscribe',
        topic: 'container.status'
    }));
};

socket.onmessage = function(event) {
    const message = JSON.parse(event.data);
    switch(message.type) {
        case 'container_event':
            handleContainerEvent(message.data);
            break;
        case 'metrics_update':
            updateContainerMetrics(message.data);
            break;
    }
};
```

#### Go客户端示例
```go
func connectWebSocket(token string) {
    url := fmt.Sprintf("wss://your-domain.com/ws?token=%s", token)

    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    if err != nil {
        log.Fatal("WebSocket连接失败:", err)
    }
    defer conn.Close()

    // 订阅容器指标
    conn.WriteJSON(map[string]string{
        "type": "subscribe",
        "topic": "container.metrics",
    })
}
```

## 监控主题详解

### 容器状态主题 (`container.status`)
监控容器生命周期关键事件。

### 容器指标主题 (`container.metrics`)
实时获取容器资源使用情况：
- CPU使用率
- 内存消耗
- 网络吞吐量
- 磁盘读写速率

### 监控告警主题 (`monitoring.alerts`)
接收系统和容器的重要告警：
- 资源使用超过阈值
- 容器异常重启
- 性能瓶颈预警

## 告警配置

### 告警阈值设置
1. 进入"系统设置"
2. 选择"监控配置"
3. 设置资源阈值：
   - CPU利用率阈值
   - 内存使用率阈值
   - 网络流量限制

### 通知渠道
- 站内通知
- 电子邮件
- 企业微信/钉钉集成
- Webhook自定义通知

## 高级监控技巧

### 性能优化建议
- 合理设置告警阈值
- 定期审查容器资源配置
- 使用指标趋势图分析系统瓶颈

### 安全监控
- 监控异常登录
- 追踪容器访问日志
- 实时检测潜在安全风险

## 常见问题排查

### WebSocket连接问题
- 检查网络连接
- 验证JWT令牌有效性
- 确认服务器地址正确

### 指标数据不准确
- 重启监控服务
- 检查Docker守护进程状态
- 验证agent监控组件

## 系统要求

### 兼容性
- 支持Docker 20.10+
- 推荐使用现代浏览器
- 建议使用最新版本Docker Auto Management System

## 许可和订阅

- **社区版**：基础监控功能
- **专业版**：高级告警和长期数据存储
- **企业版**：全面监控、合规性报告和多集群管理

## 版本信息
- **文档版本**：1.0
- **最后更新**：2025-09-26
- **适用产品版本**：Docker Auto Management System v1.x