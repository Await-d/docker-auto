# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Docker自动更新管理系统是一个企业级Docker容器生命周期管理平台，采用Go后端 + Vue 3前端的全栈架构。

## 开发命令

### 后端 (Go)
```bash
# 启动后端开发服务器
./start-backend.sh

# 手动启动（在backend目录）
cd backend
go run ./cmd/server

# 运行测试
go test ./...
go test -v ./pkg/...

# 构建二进制文件
go build -o docker-auto-server ./cmd/server

# 整理依赖
go mod tidy
```

### 前端 (Vue 3 + TypeScript)
```bash
# 启动前端开发服务器
./start-frontend.sh

# 手动启动（在frontend目录）
cd frontend
npm run dev
# 或
yarn dev

# 构建生产版本
npm run build

# 类型检查
npm run type-check

# 代码检查
npm run lint
```

### Docker部署
```bash
# 启动完整环境
docker-compose up -d

# 仅启动数据库
docker-compose up -d postgres redis

# 查看日志
docker-compose logs -f app
```

## 架构概览

### 后端架构 (Go)
采用领域驱动设计(DDD)的分层架构：

```
backend/
├── cmd/server/          # 应用入口点
├── internal/            # 私有应用代码
│   ├── api/            # WebSocket和API处理器
│   ├── config/         # 配置管理
│   ├── controller/     # HTTP控制器层
│   ├── middleware/     # HTTP中间件
│   ├── model/          # 数据模型和GORM实体
│   ├── repository/     # 数据访问层接口
│   └── service/        # 业务逻辑层
└── pkg/                # 可重用包
    ├── docker/         # Docker客户端和操作
    ├── scheduler/      # 任务调度（Cron）
    ├── security/       # 安全相关功能
    ├── monitoring/     # 监控和指标
    └── utils/          # 工具函数
```

**核心组件:**
- **ContainerService**: 容器生命周期管理
- **UpdateEngine**: 自动更新引擎和策略
- **DockerClient**: Docker API封装
- **WebSocketManager**: 实时通信
- **SchedulerService**: 基于Cron的任务调度

### 前端架构 (Vue 3)
采用组合式API和TypeScript的现代Vue架构：

```
frontend/src/
├── api/                # API调用封装
├── components/         # 可重用组件
│   ├── common/        # 通用组件
│   ├── container/     # 容器相关组件
│   ├── dashboard/     # 仪表板组件
│   └── settings/      # 设置组件
├── composables/       # 组合式函数
├── router/            # 路由配置
├── store/             # Pinia状态管理
├── types/             # TypeScript类型定义
├── utils/             # 工具函数
└── views/             # 页面组件
```

**状态管理 (Pinia):**
- `authStore`: 用户认证状态
- `containerStore`: 容器数据
- `dashboardStore`: 仪表板配置
- `settingsStore`: 系统设置

## 关键设计模式

### 后端模式
1. **Repository模式**: 数据访问抽象化（`internal/repository/`）
2. **Service层**: 业务逻辑封装（`internal/service/`）
3. **依赖注入**: 通过结构体组合实现
4. **中间件链**: Gin中间件用于横切关注点
5. **事件驱动**: 通过`pkg/events/`实现解耦

### 前端模式
1. **组合式API**: Vue 3的响应式状态管理
2. **组件组合**: 高阶组件和插槽模式
3. **类型安全**: 完整的TypeScript集成
4. **模块化**: 按功能域组织代码

## 数据库架构

**支持的数据库:**
- SQLite (开发环境默认)
- PostgreSQL (推荐生产环境)
- MySQL (可选)

**核心表:**
- `users`: 用户和认证
- `containers`: 容器配置
- `update_histories`: 更新记录
- `notifications`: 通知管理
- `scheduled_tasks`: 计划任务

## WebSocket通信

实时功能通过WebSocket实现：
- 容器状态更新
- 实时日志流
- 系统通知
- 更新进度

**连接地址:** `ws://localhost:8080/ws`

## 环境配置

### 开发环境
- 后端: `.env.dev` (由start-backend.sh自动创建)
- 前端: `.env.development` (由start-frontend.sh自动创建)

### 重要配置项
- `DB_TYPE`: 数据库类型 (sqlite/postgres/mysql)
- `DOCKER_HOST`: Docker守护进程连接
- `JWT_SECRET`: JWT签名密钥
- `LOG_LEVEL`: 日志级别

## 测试

### 后端测试
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./pkg/docker/
go test ./internal/service/

# 集成测试
go test ./internal/service/container_integration_test.go
```

### 前端测试
项目使用Element Plus组件库，测试主要关注业务逻辑和组件交互。

## Docker集成

项目深度集成Docker：
- **容器管理**: 启动、停止、重启、删除
- **镜像更新**: 自动检测和拉取新版本
- **健康检查**: 容器状态监控
- **日志收集**: 实时日志流式传输
- **资源监控**: CPU、内存使用率

## 安全考虑

- JWT令牌认证
- CORS配置
- Docker安全扫描
- 输入验证和清理
- 敏感信息加密存储

## 性能优化

- Go协程并发处理
- Redis缓存
- 数据库连接池
- 前端懒加载
- WebSocket连接复用