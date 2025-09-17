# 容器管理详解

## 容器生命周期管理

### 创建容器

1. **基本信息配置**
   ```
   容器名称: my-nginx
   镜像名称: nginx
   镜像标签: latest
   描述: Web服务容器（可选）
   ```

2. **端口映射配置**
   - 主机端口: 8080
   - 容器端口: 80
   - 协议: TCP/UDP

3. **环境变量设置**
   ```
   变量名          值
   MYSQL_PASSWORD  secure_password
   APP_ENV         production
   DEBUG          false
   ```

4. **挂载卷配置**
   - 主机路径: `/host/data`
   - 容器路径: `/app/data`
   - 访问模式: 读写/只读

### 容器状态管理

#### 启动容器
- 点击容器列表中的"启动"按钮
- 或使用批量操作启动多个容器
- 系统将自动拉取镜像（如果本地不存在）

#### 停止容器
- **优雅停止**: 发送 SIGTERM 信号，等待容器自行关闭
- **强制停止**: 发送 SIGKILL 信号，立即终止容器
- **停止超时**: 默认30秒，可在设置中修改

#### 重启容器
- **重启策略**:
  - no: 不自动重启
  - on-failure: 故障时重启
  - always: 总是重启
  - unless-stopped: 除非手动停止

## 镜像更新管理

### 更新检测

#### 自动检测
- **定时检查**: 可配置检查间隔（小时/天/周）
- **Webhook 触发**: 支持 Docker Hub、Harbor 等仓库的 Webhook
- **API 触发**: 通过 REST API 手动触发检查

#### 手动检测
1. 在容器详情页面点击"检查更新"
2. 系统将查询远程仓库的最新镜像标签
3. 比较本地镜像与远程镜像的差异

### 更新策略详解

#### 1. 滚动更新 (Rolling Update)
```yaml
策略: rolling
参数:
  max_surge: 1        # 最多可以超出目标副本数
  max_unavailable: 0  # 最多可以不可用的副本数
  health_check_interval: 30s
  timeout: 300s
```

**适用场景**: 无状态应用，需要零停机时间

#### 2. 蓝绿部署 (Blue-Green)
```yaml
策略: blue-green
参数:
  switch_delay: 60s    # 切换前等待时间
  verification_time: 300s # 新版本验证时间
  auto_rollback: true  # 自动回滚开关
```

**适用场景**: 需要快速回滚，资源充足的环境

#### 3. 金丝雀发布 (Canary)
```yaml
策略: canary
参数:
  initial_percentage: 10  # 初始流量比例
  increment: 20          # 流量递增比例
  evaluation_interval: 300s # 评估间隔
  success_threshold: 99   # 成功率阈值
```

**适用场景**: 需要渐进式验证的关键应用

#### 4. 计划更新 (Scheduled)
```yaml
策略: scheduled
参数:
  cron: "0 2 * * 0"    # 每周日凌晨2点
  timezone: "Asia/Shanghai"
  batch_size: 5        # 批量更新数量
  delay_between_batches: 60s
```

**适用场景**: 维护窗口期间的批量更新

#### 5. 手动更新 (Manual)
```yaml
策略: manual
参数:
  require_approval: true    # 需要审批
  approval_timeout: 86400s  # 审批超时时间
  approvers: ["admin@example.com"]
```

**适用场景**: 生产环境的关键系统

## 容器配置管理

### 配置模板
创建可重用的配置模板：
```json
{
  "name": "web-service-template",
  "image": "nginx",
  "ports": [{"host": 80, "container": 80}],
  "environment": {
    "NGINX_HOST": "localhost",
    "NGINX_PORT": "80"
  },
  "volumes": [
    {
      "host": "/etc/nginx/conf.d",
      "container": "/etc/nginx/conf.d",
      "mode": "ro"
    }
  ],
  "restart_policy": "always",
  "update_policy": "rolling"
}
```

### 配置版本控制
- 自动保存配置历史版本
- 支持配置对比查看差异
- 可回滚到任意历史版本
- Git 集成（可选）

## 日志管理

### 实时日志查看
- **实时流**: WebSocket 连接实时显示日志
- **日志级别**: ERROR、WARN、INFO、DEBUG
- **过滤功能**: 按关键词、时间范围过滤
- **下载导出**: 支持导出指定时间段的日志

### 日志配置
```yaml
logging:
  driver: "json-file"
  options:
    max-size: "100m"
    max-file: "3"
    labels: "app,environment"
```

## 健康检查

### 健康检查配置
```yaml
health_check:
  test: ["CMD", "curl", "-f", "http://localhost/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 60s
```

### 健康状态
- **健康**: 所有检查通过
- **不健康**: 连续失败次数达到阈值
- **启动中**: 启动期间，暂时跳过检查
- **未配置**: 未设置健康检查

## 资源监控

### 实时指标
- **CPU 使用率**: 实时 CPU 使用百分比
- **内存使用**: 当前内存使用量和限制
- **网络 I/O**: 入站/出站流量统计
- **磁盘 I/O**: 读写操作统计

### 资源限制
```yaml
resources:
  limits:
    memory: "512m"
    cpu: "0.5"
  reservations:
    memory: "256m"
    cpu: "0.25"
```

## 故障排除

### 常见问题诊断

#### 容器启动失败
1. **检查镜像**: 验证镜像名称和标签是否正确
2. **端口冲突**: 确认端口未被其他服务占用
3. **权限问题**: 检查文件系统权限
4. **资源不足**: 确认系统有足够的内存和 CPU

#### 更新失败
1. **网络连接**: 检查容器仓库网络连通性
2. **镜像拉取**: 验证镜像拉取权限
3. **健康检查**: 确认新版本通过健康检查
4. **回滚机制**: 自动回滚到上一个稳定版本

### 诊断工具
- **容器日志**: 查看详细错误信息
- **系统事件**: 查看 Docker 事件日志
- **资源监控**: 监控资源使用异常
- **网络诊断**: 测试网络连通性

## 最佳实践

### 容器设计
1. **单一职责**: 每个容器只运行一个服务
2. **无状态设计**: 避免在容器中存储持久数据
3. **配置外部化**: 使用环境变量和配置文件
4. **健康检查**: 为所有服务容器配置健康检查

### 更新策略选择
1. **开发环境**: 使用滚动更新快速迭代
2. **测试环境**: 使用金丝雀发布验证功能
3. **生产环境**: 使用蓝绿部署确保稳定性
4. **关键系统**: 使用手动更新严格控制

### 安全考虑
1. **最小权限**: 使用非 root 用户运行容器
2. **镜像安全**: 定期扫描镜像漏洞
3. **网络隔离**: 使用 Docker 网络进行服务隔离
4. **敏感数据**: 使用 Docker Secrets 管理敏感信息

---

**相关文档**: [监控告警配置](../admin/monitoring.md) | [常见问题解答](faq.md)