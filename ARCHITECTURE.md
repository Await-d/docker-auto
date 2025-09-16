# Docker Auto Update System - 技术架构文档

## 1. 项目概述

### 1.1 项目名称
Docker Auto Update System (简称: Docker-Auto)

### 1.2 项目目标
开发一个Docker容器自动更新系统，能够自动检测镜像更新、管理容器生命周期，并提供完整的Web管理界面。

### 1.3 核心功能
- 容器管理：添加、删除、启停、配置管理
- 自动更新：定时检查镜像更新并自动更新容器
- Web面板：提供可视化管理界面
- 监控告警：容器状态监控和更新通知

## 2. 系统架构设计

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Docker Auto Update System                │
├─────────────────────────────────────────────────────────────────┤
│                          Frontend Layer                         │
│   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐│
│   │   Dashboard     │    │Container Manager│    │   Settings  ││
│   │   (Vue 3 + TS)  │    │   (Vue 3 + TS)  │    │(Vue 3 + TS) ││
│   └─────────────────┘    └─────────────────┘    └─────────────┘│
├─────────────────────────────────────────────────────────────────┤
│                         API Gateway Layer                       │
│   ┌─────────────────────────────────────────────────────────────┐│
│   │                API Gateway (Go + Gin)                      ││
│   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   ││
│   │  │    Auth     │ │   Router    │ │     Middleware      │   ││
│   │  │  (JWT)      │ │   Handler   │ │  (CORS/Log/Rate)    │   ││
│   │  └─────────────┘ └─────────────┘ └─────────────────────┘   ││
│   └─────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│                        Service Layer                            │
│   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│   │Container Manager│ │ Image Checker   │ │   Scheduler     │  │
│   │    Service      │ │    Service      │ │   Service       │  │
│   │                 │ │                 │ │                 │  │
│   │• CRUD操作       │ │• 镜像版本检查    │ │• 定时任务调度    │  │
│   │• 状态监控       │ │• 多源支持        │ │• 更新策略执行    │  │
│   │• 健康检查       │ │• 缓存优化        │ │• 回滚机制        │  │
│   └─────────────────┘ └─────────────────┘ └─────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                         Data Layer                              │
│   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│   │   PostgreSQL    │ │      Redis      │ │   File System   │  │
│   │                 │ │                 │ │                 │  │
│   │• 容器配置        │ │• 任务队列        │ │• 日志文件        │  │
│   │• 更新历史        │ │• 缓存数据        │ │• 配置文件        │  │
│   │• 用户数据        │ │• 会话存储        │ │• 备份文件        │  │
│   └─────────────────┘ └─────────────────┘ └─────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      External Integration                       │
│   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│   │  Docker Engine  │ │ Image Registry  │ │ Notification    │  │
│   │                 │ │                 │ │                 │  │
│   │• Docker API     │ │• Docker Hub     │ │• Email/SMTP     │  │
│   │• Container运行   │ │• Harbor/私有仓库 │ │• Webhook       │  │
│   │• 镜像管理        │ │• 镜像拉取        │ │• 企业微信       │  │
│   └─────────────────┘ └─────────────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 架构分层说明

#### 2.2.1 前端层 (Frontend Layer)
- **技术栈**: Vue 3 + TypeScript + Element Plus
- **主要模块**:
  - Dashboard: 系统概览、统计图表
  - Container Manager: 容器管理界面
  - Settings: 系统配置管理
- **通信方式**: REST API + WebSocket (实时状态)

#### 2.2.2 API网关层 (API Gateway Layer)
- **技术栈**: Go + Gin框架
- **职责**:
  - 统一API入口
  - 身份认证授权 (JWT)
  - 请求路由分发
  - 中间件处理 (CORS、日志、限流)

#### 2.2.3 服务层 (Service Layer)
- **Container Manager Service**: 容器生命周期管理
- **Image Checker Service**: 镜像版本检查和比较
- **Scheduler Service**: 定时任务调度和更新执行

#### 2.2.4 数据层 (Data Layer)
- **PostgreSQL**: 主要业务数据存储
- **Redis**: 缓存和任务队列
- **File System**: 日志和配置文件

## 3. 技术栈选择

### 3.1 后端技术栈

#### 核心框架
- **语言**: Go 1.21+
- **Web框架**: Gin (高性能HTTP框架)
- **ORM**: GORM (Go语言ORM库)

#### 选择理由
1. **Go语言优势**:
   - Docker生态原生支持，API兼容性最佳
   - 高并发性能，适合长期运行的系统服务
   - 编译后单文件部署，运维友好
   - 丰富的Docker SDK支持

2. **Gin框架优势**:
   - 高性能HTTP路由
   - 中间件支持完善
   - 社区活跃，文档完整

### 3.2 前端技术栈

#### 核心框架
- **框架**: Vue 3 + Composition API
- **语言**: TypeScript
- **UI库**: Element Plus
- **构建工具**: Vite
- **状态管理**: Pinia

#### 选择理由
1. **Vue 3优势**:
   - 响应式系统优化，性能提升
   - Composition API提供更好的代码组织
   - TypeScript支持更完善

2. **Element Plus优势**:
   - 企业级UI组件库
   - 组件丰富，设计统一
   - Vue 3原生支持

### 3.3 数据存储

#### 主数据库
- **生产环境**: PostgreSQL 15+
- **开发环境**: SQLite (可选)

#### 缓存和队列
- **Redis 7+**: 缓存、会话存储、任务队列

#### 选择理由
1. **PostgreSQL优势**:
   - 功能强大的关系型数据库
   - 支持JSON数据类型
   - 高并发性能优秀
   - 丰富的扩展插件

2. **Redis优势**:
   - 高性能内存数据库
   - 丰富的数据结构支持
   - 发布订阅功能
   - 任务队列支持

## 4. 数据模型设计

### 4.1 核心数据表

#### 容器配置表 (containers)
```sql
CREATE TABLE containers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,          -- 容器名称
    image VARCHAR(255) NOT NULL,                -- 镜像名称
    tag VARCHAR(100) DEFAULT 'latest',          -- 镜像标签
    container_id VARCHAR(64),                   -- Docker容器ID
    status VARCHAR(50) DEFAULT 'stopped',       -- 容器状态
    config_json JSONB,                          -- 容器配置(环境变量、端口映射等)
    update_policy VARCHAR(50) DEFAULT 'auto',   -- 更新策略
    registry_url VARCHAR(255),                  -- 镜像仓库URL
    registry_auth JSONB,                        -- 仓库认证信息(加密存储)
    health_check JSONB,                         -- 健康检查配置
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_status (status),
    INDEX idx_update_policy (update_policy)
);
```

#### 更新历史表 (update_history)
```sql
CREATE TABLE update_history (
    id SERIAL PRIMARY KEY,
    container_id INTEGER REFERENCES containers(id) ON DELETE CASCADE,
    old_image VARCHAR(255),                     -- 更新前镜像
    new_image VARCHAR(255),                     -- 更新后镜像
    old_digest VARCHAR(71),                     -- 更新前镜像digest
    new_digest VARCHAR(71),                     -- 更新后镜像digest
    status VARCHAR(50),                         -- 更新状态: success/failed/rollback
    error_message TEXT,                         -- 错误信息
    duration_seconds INTEGER,                   -- 更新耗时(秒)
    triggered_by VARCHAR(50),                   -- 触发方式: auto/manual/schedule
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    INDEX idx_container_id (container_id),
    INDEX idx_status (status),
    INDEX idx_started_at (started_at)
);
```

#### 镜像版本缓存表 (image_versions)
```sql
CREATE TABLE image_versions (
    id SERIAL PRIMARY KEY,
    image_name VARCHAR(255) NOT NULL,           -- 镜像名称
    tag VARCHAR(100) NOT NULL,                  -- 标签
    digest VARCHAR(71) NOT NULL,                -- 镜像digest
    size_bytes BIGINT,                          -- 镜像大小
    published_at TIMESTAMP,                     -- 发布时间
    architecture VARCHAR(50),                   -- 架构信息
    os VARCHAR(50),                             -- 操作系统
    checked_at TIMESTAMP DEFAULT NOW(),         -- 检查时间
    registry_url VARCHAR(255),                  -- 仓库URL
    UNIQUE(image_name, tag, registry_url),
    INDEX idx_image_tag (image_name, tag),
    INDEX idx_checked_at (checked_at)
);
```

#### 系统配置表 (system_configs)
```sql
CREATE TABLE system_configs (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,    -- 配置键
    config_value JSONB,                         -- 配置值
    description TEXT,                           -- 配置描述
    is_encrypted BOOLEAN DEFAULT FALSE,         -- 是否加密存储
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 4.2 数据关系说明

1. **容器与更新历史**: 一对多关系，每个容器可以有多条更新记录
2. **镜像版本缓存**: 独立表，用于缓存镜像仓库信息，减少API调用
3. **系统配置**: 存储全局配置，如通知设置、仓库配置等

## 5. API设计

### 5.1 RESTful API规范

#### 基础URL结构
```
Base URL: http://localhost:8080/api/v1
Authentication: Bearer Token (JWT)
Content-Type: application/json
```

#### 响应格式
```json
{
    "code": 200,
    "message": "success",
    "data": {},
    "timestamp": "2024-01-01T12:00:00Z"
}
```

### 5.2 核心API端点

#### 5.2.1 容器管理API
```http
# 获取容器列表
GET /api/v1/containers
Query Parameters:
  - page: 页码 (默认1)
  - limit: 每页数量 (默认20)
  - status: 状态过滤 (running/stopped/all)
  - search: 搜索关键词

# 获取容器详情
GET /api/v1/containers/{id}

# 创建容器
POST /api/v1/containers
Body: {
    "name": "my-app",
    "image": "nginx",
    "tag": "latest",
    "config": {
        "ports": ["80:8080"],
        "env": {"ENV": "production"},
        "volumes": ["/data:/app/data"]
    },
    "update_policy": "auto"
}

# 更新容器配置
PUT /api/v1/containers/{id}

# 删除容器
DELETE /api/v1/containers/{id}

# 启动容器
POST /api/v1/containers/{id}/start

# 停止容器
POST /api/v1/containers/{id}/stop

# 重启容器
POST /api/v1/containers/{id}/restart

# 手动更新容器
POST /api/v1/containers/{id}/update
Body: {
    "force": false,
    "backup": true
}
```

#### 5.2.2 镜像管理API
```http
# 检查镜像更新
GET /api/v1/images/check
Query Parameters:
  - container_id: 特定容器ID (可选)
  - force: 强制检查 (忽略缓存)

# 获取镜像版本历史
GET /api/v1/images/{image}/versions

# 获取镜像详细信息
GET /api/v1/images/{image}/info
```

#### 5.2.3 更新管理API
```http
# 获取更新历史
GET /api/v1/updates/history
Query Parameters:
  - container_id: 容器ID过滤
  - status: 状态过滤
  - start_date: 开始日期
  - end_date: 结束日期

# 获取更新详情
GET /api/v1/updates/{id}

# 回滚更新
POST /api/v1/updates/{id}/rollback

# 批量更新
POST /api/v1/updates/batch
Body: {
    "container_ids": [1, 2, 3],
    "strategy": "rolling"
}
```

#### 5.2.4 系统管理API
```http
# 获取系统状态
GET /api/v1/system/status

# 获取系统配置
GET /api/v1/system/config

# 更新系统配置
PUT /api/v1/system/config
Body: {
    "notification": {
        "email": {
            "enabled": true,
            "smtp_host": "smtp.gmail.com",
            "smtp_port": 587
        }
    },
    "schedule": {
        "check_interval": "1h"
    }
}

# 获取系统日志
GET /api/v1/system/logs

# 测试通知
POST /api/v1/system/test-notification
```

### 5.3 WebSocket API

#### 实时状态推送
```javascript
// 连接WebSocket
const ws = new WebSocket('ws://localhost:8080/ws/status');

// 消息格式
{
    "type": "container_status",
    "data": {
        "container_id": 1,
        "status": "running",
        "timestamp": "2024-01-01T12:00:00Z"
    }
}

// 消息类型
- container_status: 容器状态变化
- update_progress: 更新进度
- system_alert: 系统告警
- log_stream: 实时日志
```

## 6. 服务模块设计

### 6.1 Container Manager Service

#### 主要职责
- 容器生命周期管理 (CRUD)
- 容器状态监控
- 容器健康检查
- Docker API交互

#### 核心接口
```go
type ContainerService interface {
    // 容器管理
    CreateContainer(ctx context.Context, req *CreateContainerRequest) (*Container, error)
    GetContainer(ctx context.Context, id int64) (*Container, error)
    UpdateContainer(ctx context.Context, id int64, req *UpdateContainerRequest) error
    DeleteContainer(ctx context.Context, id int64) error
    ListContainers(ctx context.Context, filter *ContainerFilter) ([]*Container, error)

    // 容器操作
    StartContainer(ctx context.Context, id int64) error
    StopContainer(ctx context.Context, id int64) error
    RestartContainer(ctx context.Context, id int64) error

    // 状态监控
    GetContainerStatus(ctx context.Context, id int64) (*ContainerStatus, error)
    HealthCheck(ctx context.Context, id int64) (*HealthStatus, error)
    StreamLogs(ctx context.Context, id int64) (<-chan string, error)
}
```

### 6.2 Image Checker Service

#### 主要职责
- 定时检查镜像更新
- 多镜像仓库支持
- 版本比较和缓存
- 镜像信息获取

#### 核心接口
```go
type ImageService interface {
    // 镜像检查
    CheckImageUpdate(ctx context.Context, image string, currentDigest string) (*ImageUpdateInfo, error)
    CheckAllImages(ctx context.Context) ([]*ImageUpdateInfo, error)

    // 镜像信息
    GetImageInfo(ctx context.Context, image string) (*ImageInfo, error)
    GetImageVersions(ctx context.Context, image string) ([]*ImageVersion, error)

    // 缓存管理
    RefreshCache(ctx context.Context, image string) error
    ClearCache(ctx context.Context) error
}
```

### 6.3 Scheduler Service

#### 主要职责
- 定时任务调度
- 更新策略执行
- 回滚机制
- 通知发送

#### 核心接口
```go
type SchedulerService interface {
    // 调度管理
    ScheduleUpdate(ctx context.Context, containerID int64, strategy UpdateStrategy) error
    CancelSchedule(ctx context.Context, containerID int64) error

    // 更新执行
    ExecuteUpdate(ctx context.Context, updatePlan *UpdatePlan) error
    RollbackUpdate(ctx context.Context, historyID int64) error

    // 策略管理
    SetUpdatePolicy(ctx context.Context, containerID int64, policy UpdatePolicy) error
    GetUpdatePolicies(ctx context.Context) ([]*UpdatePolicy, error)
}
```

## 7. 部署架构

### 7.1 开发环境部署

#### Docker Compose配置
```yaml
version: '3.8'
services:
  # 后端服务
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=dockerauto
      - DB_USER=postgres
      - DB_PASSWORD=password
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=your-jwt-secret
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./logs:/app/logs
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  # 前端服务
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:80"
    depends_on:
      - backend
    restart: unless-stopped

  # 数据库
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: dockerauto
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_INITDB_ARGS: "--encoding=UTF-8"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./database/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    restart: unless-stopped

  # Redis缓存
  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    restart: unless-stopped

  # Nginx反向代理
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
    depends_on:
      - frontend
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### 7.2 生产环境部署

#### Kubernetes部署 (可选)
```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: docker-auto

---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: docker-auto-backend
  namespace: docker-auto
spec:
  replicas: 3
  selector:
    matchLabels:
      app: docker-auto-backend
  template:
    metadata:
      labels:
        app: docker-auto-backend
    spec:
      containers:
      - name: backend
        image: docker-auto/backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: REDIS_HOST
          value: "redis-service"
        volumeMounts:
        - name: docker-sock
          mountPath: /var/run/docker.sock
      volumes:
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
```

### 7.3 项目目录结构

```
docker-auto/
├── README.md                    # 项目说明文档
├── ARCHITECTURE.md             # 架构设计文档 (本文件)
├── docker-compose.yml          # 开发环境配置
├── docker-compose.prod.yml     # 生产环境配置
├── .env.example               # 环境变量模板
├── .gitignore                 # Git忽略文件
│
├── backend/                   # Go后端服务
│   ├── cmd/                  # 程序入口
│   │   └── server/
│   │       └── main.go
│   ├── internal/             # 内部模块
│   │   ├── api/             # API控制器
│   │   ├── service/         # 业务逻辑层
│   │   ├── repository/      # 数据访问层
│   │   ├── model/           # 数据模型
│   │   ├── config/          # 配置管理
│   │   └── middleware/      # 中间件
│   ├── pkg/                 # 公共包
│   │   ├── docker/          # Docker客户端封装
│   │   ├── registry/        # 镜像仓库客户端
│   │   ├── notification/    # 通知服务
│   │   └── utils/           # 工具函数
│   ├── migrations/          # 数据库迁移脚本
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                 # Vue前端应用
│   ├── src/
│   │   ├── components/      # 组件
│   │   ├── views/           # 页面
│   │   ├── router/          # 路由配置
│   │   ├── store/           # 状态管理
│   │   ├── api/             # API调用
│   │   ├── types/           # TypeScript类型定义
│   │   └── utils/           # 工具函数
│   ├── public/
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── Dockerfile
│
├── database/                 # 数据库相关
│   ├── init.sql             # 初始化脚本
│   ├── migrations/          # 迁移脚本
│   └── seeds/               # 种子数据
│
├── nginx/                    # Nginx配置
│   ├── nginx.conf
│   └── ssl/
│
├── scripts/                  # 部署和工具脚本
│   ├── build.sh
│   ├── deploy.sh
│   └── backup.sh
│
├── docs/                     # 项目文档
│   ├── api.md               # API文档
│   ├── deployment.md        # 部署文档
│   └── development.md       # 开发文档
│
└── tests/                    # 测试代码
    ├── e2e/                 # 端到端测试
    ├── integration/         # 集成测试
    └── unit/                # 单元测试
```

## 8. 安全设计

### 8.1 身份认证与授权

#### JWT Token认证
```go
// JWT Claims结构
type Claims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.StandardClaims
}

// Token过期时间: 24小时
// 刷新Token过期时间: 7天
```

#### RBAC权限控制
```sql
-- 用户表
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    role VARCHAR(20) DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 权限定义
- admin: 系统管理员，全部权限
- operator: 操作员，容器管理权限
- viewer: 查看者，只读权限
```

### 8.2 数据安全

#### 敏感信息加密
```go
// 镜像仓库凭据加密存储
type RegistryAuth struct {
    Username string `json:"username"`
    Password string `json:"password"` // AES加密存储
    Token    string `json:"token"`    // AES加密存储
}

// 使用AES-256-GCM加密
func EncryptSensitiveData(data string, key []byte) (string, error)
func DecryptSensitiveData(encrypted string, key []byte) (string, error)
```

#### Docker Socket安全
```yaml
# 只读挂载Docker socket
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro

# 容器运行用户 (非root)
user: "1000:1000"

# 限制容器权限
security_opt:
  - no-new-privileges:true
cap_drop:
  - ALL
cap_add:
  - NET_BIND_SERVICE
```

### 8.3 网络安全

#### HTTPS配置
```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # 其他安全头
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
}
```

#### API限流
```go
// 使用gin-rate-limit中间件
import "github.com/gin-contrib/rate-limit"

// 限制: 每分钟100次请求
store := ratelimit.NewInMemoryStore(100)
ratelimiter := ratelimit.RateLimiter(store, &ratelimit.Options{
    ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
        c.JSON(429, gin.H{"error": "Too many requests"})
    },
})

router.Use(ratelimiter)
```

## 9. 监控与运维

### 9.1 健康检查

#### 应用健康检查
```go
// 健康检查端点
func HealthCheckHandler(c *gin.Context) {
    health := &HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks: map[string]interface{}{
            "database":    checkDatabase(),
            "redis":       checkRedis(),
            "docker":      checkDockerAPI(),
            "disk_space":  checkDiskSpace(),
        },
    }

    if !allHealthy(health.Checks) {
        health.Status = "unhealthy"
        c.JSON(503, health)
        return
    }

    c.JSON(200, health)
}
```

#### Docker Compose健康检查
```yaml
backend:
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 30s
    timeout: 10s
    retries: 3
    start_period: 40s
```

### 9.2 日志管理

#### 结构化日志
```go
import "github.com/sirupsen/logrus"

// 日志格式配置
log := logrus.New()
log.SetFormatter(&logrus.JSONFormatter{})
log.SetLevel(logrus.InfoLevel)

// 使用示例
log.WithFields(logrus.Fields{
    "container_id": 123,
    "action":       "update",
    "image":        "nginx:latest",
}).Info("Container update started")
```

#### 日志轮转
```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

### 9.3 性能监控

#### Prometheus指标暴露
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    containerTotal = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "docker_auto_containers_total",
            Help: "Total number of managed containers",
        },
        []string{"status"},
    )

    updateDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "docker_auto_update_duration_seconds",
            Help: "Duration of container updates",
        },
        []string{"container", "status"},
    )
)

// 注册指标
prometheus.MustRegister(containerTotal, updateDuration)
```

## 10. 开发规范

### 10.1 代码规范

#### Go代码规范
- 使用`gofmt`格式化代码
- 使用`golint`检查代码质量
- 遵循Go官方编码规范
- 函数和方法必须有注释
- 错误处理不能忽略

#### TypeScript代码规范
- 使用ESLint + Prettier
- 严格的TypeScript配置
- 组件使用Composition API
- 遵循Vue 3官方风格指南

### 10.2 Git工作流

#### 分支策略
```
main                # 主分支，生产环境代码
├── develop         # 开发分支
├── feature/*       # 功能分支
├── release/*       # 发布分支
└── hotfix/*        # 热修复分支
```

#### 提交规范
```
feat: 新功能
fix: 修复bug
docs: 文档更新
style: 代码格式化
refactor: 重构代码
test: 测试相关
chore: 构建工具或辅助工具的变动
```

### 10.3 测试策略

#### 测试金字塔
```
         /\
        /  \  E2E Tests (10%)
       /    \
      /------\ Integration Tests (20%)
     /        \
    /----------\ Unit Tests (70%)
   /            \
  /--------------\
```

#### 测试覆盖率要求
- 单元测试覆盖率: >= 80%
- 集成测试覆盖率: >= 60%
- 关键业务逻辑: >= 90%

## 11. 性能优化

### 11.1 数据库优化

#### 索引策略
```sql
-- 容器查询优化
CREATE INDEX CONCURRENTLY idx_containers_status ON containers(status);
CREATE INDEX CONCURRENTLY idx_containers_update_policy ON containers(update_policy);
CREATE INDEX CONCURRENTLY idx_containers_name_trgm ON containers USING gin(name gin_trgm_ops);

-- 更新历史查询优化
CREATE INDEX CONCURRENTLY idx_update_history_container_time ON update_history(container_id, started_at DESC);
CREATE INDEX CONCURRENTLY idx_update_history_status_time ON update_history(status, started_at DESC);

-- 镜像版本查询优化
CREATE INDEX CONCURRENTLY idx_image_versions_name_tag ON image_versions(image_name, tag);
CREATE INDEX CONCURRENTLY idx_image_versions_checked ON image_versions(checked_at DESC);
```

#### 连接池配置
```go
// GORM连接池配置
db.DB().SetMaxIdleConns(10)           // 最大空闲连接数
db.DB().SetMaxOpenConns(100)          // 最大打开连接数
db.DB().SetConnMaxLifetime(time.Hour) // 连接最大生存时间
```

### 11.2 缓存策略

#### Redis缓存设计
```go
// 缓存键命名规范
const (
    ContainerStatusKey  = "container:status:%d"     // 容器状态 (TTL: 30s)
    ImageInfoKey       = "image:info:%s"           // 镜像信息 (TTL: 1h)
    ImageVersionsKey   = "image:versions:%s"       // 镜像版本 (TTL: 6h)
    SystemConfigKey    = "system:config"           // 系统配置 (TTL: 5m)
)

// 缓存更新策略
- 写透 (Write Through): 系统配置
- 延迟写入 (Write Behind): 统计数据
- 旁路缓存 (Cache Aside): 镜像信息
```

### 11.3 前端优化

#### 代码分割
```typescript
// 路由懒加载
const Dashboard = () => import('@/views/Dashboard.vue')
const ContainerManage = () => import('@/views/ContainerManage.vue')
const Settings = () => import('@/views/Settings.vue')

// 组件懒加载
const HeavyComponent = defineAsyncComponent(() => import('@/components/HeavyComponent.vue'))
```

#### 资源优化
```typescript
// Vite配置优化
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router', 'pinia'],
          ui: ['element-plus'],
          utils: ['axios', 'dayjs']
        }
      }
    },
    cssCodeSplit: true,
    sourcemap: false
  },
  optimizeDeps: {
    include: ['vue', 'vue-router', 'pinia', 'element-plus']
  }
})
```

## 12. 扩展性设计

### 12.1 插件系统

#### 插件接口定义
```go
// 插件接口
type Plugin interface {
    Name() string
    Version() string
    Init(config map[string]interface{}) error
    Execute(ctx context.Context, data interface{}) (interface{}, error)
    Cleanup() error
}

// 更新策略插件
type UpdateStrategyPlugin interface {
    Plugin
    ShouldUpdate(container *Container, imageInfo *ImageInfo) bool
    ExecuteUpdate(ctx context.Context, plan *UpdatePlan) error
}

// 通知插件
type NotificationPlugin interface {
    Plugin
    SendNotification(ctx context.Context, notification *Notification) error
}
```

### 12.2 多租户支持

#### 租户隔离设计
```sql
-- 租户表
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    config JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 用户租户关联
CREATE TABLE user_tenants (
    user_id INTEGER REFERENCES users(id),
    tenant_id INTEGER REFERENCES tenants(id),
    role VARCHAR(20) DEFAULT 'member',
    PRIMARY KEY (user_id, tenant_id)
);

-- 为所有业务表添加tenant_id字段
ALTER TABLE containers ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
```

### 12.3 API版本管理

#### 版本策略
```go
// API版本路由
v1 := router.Group("/api/v1")
{
    v1.GET("/containers", v1ContainerHandler)
}

v2 := router.Group("/api/v2")
{
    v2.GET("/containers", v2ContainerHandler)
}

// 版本兼容性保证
- v1: 保持向后兼容，只修复关键bug
- v2: 新功能开发，可能包含breaking changes
- 每个版本至少维护2年
```

## 13. 部署与运维指南

### 13.1 环境要求

#### 最低系统要求
- **CPU**: 2 核心
- **内存**: 4GB RAM
- **磁盘**: 20GB 可用空间
- **操作系统**:
  - Ubuntu 20.04+ / CentOS 8+ / Debian 11+
  - Docker 20.10+
  - Docker Compose 2.0+

#### 推荐生产环境
- **CPU**: 4+ 核心
- **内存**: 8GB+ RAM
- **磁盘**: 100GB+ SSD
- **网络**: 100Mbps+

### 13.2 部署步骤

#### 快速部署
```bash
# 1. 克隆项目
git clone https://github.com/your-org/docker-auto.git
cd docker-auto

# 2. 配置环境变量
cp .env.example .env
vim .env

# 3. 启动服务
docker-compose up -d

# 4. 初始化数据库
docker-compose exec backend ./migrate up

# 5. 创建管理员用户
docker-compose exec backend ./create-admin-user
```

#### 生产环境部署
```bash
# 使用生产配置
docker-compose -f docker-compose.prod.yml up -d

# 配置SSL证书
./scripts/setup-ssl.sh

# 配置备份策略
./scripts/setup-backup.sh
```

### 13.3 监控与告警

#### Prometheus + Grafana
```yaml
# 监控堆栈
monitoring:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
```

#### 告警规则
```yaml
# prometheus/alerts.yml
groups:
- name: docker-auto-alerts
  rules:
  - alert: ContainerUpdateFailed
    expr: docker_auto_update_failures_total > 0
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Container update failed"
      description: "Container {{ $labels.container }} update failed"

  - alert: HighMemoryUsage
    expr: docker_auto_memory_usage_percent > 85
    for: 10m
    labels:
      severity: critical
    annotations:
      summary: "High memory usage"
      description: "Memory usage is above 85%"
```

## 14. 故障排除

### 14.1 常见问题

#### Docker连接问题
```bash
# 检查Docker socket权限
ls -la /var/run/docker.sock

# 检查Docker服务状态
systemctl status docker

# 测试Docker API连接
curl --unix-socket /var/run/docker.sock http://localhost/version
```

#### 数据库连接问题
```bash
# 检查数据库连接
docker-compose exec backend pg_isready -h postgres -U postgres

# 查看数据库日志
docker-compose logs postgres

# 手动连接测试
docker-compose exec postgres psql -U postgres -d dockerauto
```

#### 性能问题排查
```bash
# 查看容器资源使用
docker stats

# 查看应用日志
docker-compose logs backend | grep ERROR

# 数据库性能分析
docker-compose exec postgres psql -U postgres -d dockerauto -c "
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;"
```

### 14.2 备份与恢复

#### 数据备份
```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# 备份数据库
docker-compose exec postgres pg_dump -U postgres dockerauto > $BACKUP_DIR/database.sql

# 备份配置文件
cp -r ./config $BACKUP_DIR/
cp .env $BACKUP_DIR/

# 压缩备份
tar -czf $BACKUP_DIR.tar.gz $BACKUP_DIR
rm -rf $BACKUP_DIR

echo "Backup completed: $BACKUP_DIR.tar.gz"
```

#### 数据恢复
```bash
#!/bin/bash
# restore.sh

BACKUP_FILE=$1
if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.tar.gz>"
    exit 1
fi

# 解压备份
tar -xzf $BACKUP_FILE

# 恢复数据库
docker-compose exec postgres psql -U postgres -c "DROP DATABASE IF EXISTS dockerauto;"
docker-compose exec postgres psql -U postgres -c "CREATE DATABASE dockerauto;"
docker-compose exec postgres psql -U postgres dockerauto < ./backup/database.sql

echo "Restore completed"
```

## 15. 更新日志

### Version 1.0.0 (计划)
- ✅ 基础容器管理功能
- ✅ 自动镜像检查和更新
- ✅ Web管理界面
- ✅ 基础通知系统

### Version 1.1.0 (计划)
- 🔄 高级更新策略 (蓝绿部署、金丝雀发布)
- 🔄 多镜像仓库支持
- 🔄 详细的操作审计日志
- 🔄 性能监控和告警

### Version 2.0.0 (计划)
- ⏳ 多租户架构
- ⏳ 插件系统
- ⏳ Kubernetes支持
- ⏳ 集群模式部署

---

## 文档维护

**文档版本**: 1.0.0
**最后更新**: 2024-01-01
**维护人员**: 开发团队
**更新频率**: 随项目开发进度更新

如有问题或建议，请提交Issue或Pull Request。