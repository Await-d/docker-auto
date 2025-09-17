# 快速入门指南

## 系统概述

Docker Auto 是一个专业的 Docker 容器自动更新管理系统，帮助您轻松管理生产环境中的容器生命周期。

## 快速安装

### 方法一：完整启动（推荐）

```bash
# 1. 启动 PostgreSQL 数据库
docker run -d \
  --name postgres-db \
  -e POSTGRES_DB=dockerauto \
  -e POSTGRES_USER=dockerauto \
  -e POSTGRES_PASSWORD=secure_password_123 \
  -p 5432:5432 \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine

# 2. 启动 Docker Auto 应用
docker run -d \
  --name docker-auto-system \
  -p 80:80 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v docker-auto-data:/app/data \
  -e APP_ENV=production \
  -e DB_HOST=host.docker.internal \
  -e DB_PASSWORD=secure_password_123 \
  -e JWT_SECRET=your-secure-jwt-secret \
  await2719/docker-auto:latest
```

### 方法二：使用 Docker Compose

```bash
git clone https://github.com/your-org/docker-auto.git
cd docker-auto
docker-compose up -d
```

## 首次登录

1. 打开浏览器访问 http://localhost
2. 使用默认凭据登录：
   - 邮箱：admin@example.com
   - 密码：admin123
3. **重要**：登录后立即修改默认密码

## 基础配置

### 添加第一个容器

1. 点击"添加容器"按钮
2. 填写基本信息：
   - 容器名称：nginx-test
   - 镜像：nginx
   - 标签：latest
   - 更新策略：rolling
3. 配置端口映射（如需要）
4. 点击"创建容器"

### 设置更新策略

- **rolling**: 滚动更新，零停机
- **blue-green**: 蓝绿部署，快速切换
- **canary**: 金丝雀发布，渐进式更新
- **manual**: 手动更新，需要审批

## 常用功能

### 容器管理
- 启动/停止容器
- 查看容器日志
- 监控资源使用
- 更新容器镜像

### 系统监控
- 实时状态看板
- 资源使用图表
- 更新历史记录
- 告警通知

## 下一步

- [控制面板使用指南](dashboard.md)
- [容器管理详解](containers.md)
- [常见问题解答](faq.md)