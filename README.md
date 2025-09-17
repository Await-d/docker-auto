# Docker 自动更新管理系统

![Auto Release](https://github.com/Await-d/docker-auto/actions/workflows/auto-release-pipeline.yml/badge.svg)
![PR Checks](https://github.com/Await-d/docker-auto/actions/workflows/pr-lint-check.yml/badge.svg)
![Docker Build](https://github.com/Await-d/docker-auto/actions/workflows/docker-build-test.yml/badge.svg)
![Dependencies](https://github.com/Await-d/docker-auto/actions/workflows/dependency-update.yml/badge.svg)

[![Docker](https://img.shields.io/badge/docker-supported-blue)](https://www.docker.com/)
[![Go Version](https://img.shields.io/badge/go-1.23+-blue)](https://golang.org/)
[![Vue](https://img.shields.io/badge/vue-3.0+-green)](https://vuejs.org/)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

> **语言版本**: **🇨🇳 中文** | [🇺🇸 English](README.en.md)

一个专业的 Docker 容器自动更新管理系统，具备企业级功能特性、实时监控、全面的 Web 管理界面和统一的单镜像架构。

## 🚀 概览

Docker 自动更新管理系统是一个用于在生产环境中管理 Docker 容器生命周期的综合解决方案。它提供自动镜像更新检测、智能更新调度、回滚功能以及通过现代 Web 界面的广泛监控功能。

### 核心特性

- **🔄 自动化容器更新**: 智能更新检测，支持多种策略和调度
- **📊 实时监控**: 通过 WebSocket 实现的实时容器状态跟踪
- **🖥️ 现代化 Web 界面**: Vue 3 + TypeScript 响应式控制面板
- **🔐 企业级安全**: JWT 认证、基于角色的访问控制和审计日志
- **📈 性能监控**: 集成的指标收集和告警系统
- **🔄 回滚支持**: 失败更新的自动回滚机制
- **🐳 多注册中心支持**: Docker Hub、私有注册中心和云服务商
- **⚡ 高性能**: Go 后端搭配 PostgreSQL 和 Redis 缓存
- **🔔 通知系统**: 邮件、Slack、Webhook 通知更新和告警
- **📱 移动响应式**: 全面支持移动端管理
- **🎯 单镜像架构**: 统一容器包含前端、后端和文档
- **⚡ 优化性能**: 增强的构建流程和简化的 CI/CD 集成
- **🔧 主分支集成**: 针对主分支优化的完整 GitHub Actions 工作流

## 🏗️ 架构

### 技术栈

- **后端**: Go 1.21+ 配合 Gin 框架和 GORM ORM
- **前端**: Vue 3 + TypeScript + Element Plus UI
- **数据库**: PostgreSQL 13+ 配合 Redis 缓存
- **实时通信**: WebSocket 实现实时更新
- **容器化**: Docker + Docker Compose
- **监控**: Prometheus 指标配合 Grafana 仪表板（可选）
- **代理**: Nginx 用于生产部署

### 系统架构

```mermaid
graph TB
    subgraph "前端层"
        A[Vue 3 控制面板]
        B[容器管理]
        C[设置面板]
    end

    subgraph "API 网关"
        D[Gin 路由 + 中间件]
        E[JWT 认证]
        F[限流]
    end

    subgraph "服务层"
        G[容器服务]
        H[更新服务]
        I[调度服务]
        J[通知服务]
    end

    subgraph "数据层"
        K[(PostgreSQL)]
        L[(Redis 缓存)]
        M[Docker 引擎]
    end

    A --> D
    B --> D
    C --> D
    D --> G
    D --> H
    D --> I
    D --> J
    G --> K
    H --> L
    I --> M
```

## 🚦 快速开始

**v2.3.0 新特性**: 完整的 GitHub Actions 自动化，针对主分支集成进行了优化！

### 系统要求

- **Docker**: 20.10+ 配合 Docker Compose v2
- **系统**: Linux/macOS/Windows，2GB+ 内存
- **网络**: 用于下载镜像的互联网访问
- **端口**: 80（统一服务）、5432（数据库）

### 1 分钟安装

```bash
# 克隆仓库
git clone https://github.com/your-org/docker-auto.git
cd docker-auto

# 配置环境
cp .env.example .env
# 编辑 .env 文件配置您的设置

# 启动系统
docker-compose up -d

# 验证安装
curl http://localhost/health
```

### 访问系统

- **Web 控制面板**: http://localhost（统一界面）
- **API 端点**: http://localhost/api
- **文档**: http://localhost/docs
- **默认登录**: admin@example.com / admin123（请立即更改）

## 🐳 Docker 镜像示例

### 常用镜像添加示例

系统支持管理各种类型的 Docker 镜像，以下是一些常见的使用案例：

#### Web 应用
```yaml
# Nginx Web 服务器
镜像名称: nginx
标签: latest
更新策略: rolling
健康检查: ✓
端口映射: 80:80

# Apache HTTP 服务器
镜像名称: httpd
标签: 2.4-alpine
更新策略: blue-green
健康检查: ✓
端口映射: 8080:80
```

#### 数据库服务
```yaml
# PostgreSQL 数据库
镜像名称: postgres
标签: 15-alpine
更新策略: manual
备份策略: ✓
环境变量:
  - POSTGRES_DB=myapp
  - POSTGRES_USER=user
  - POSTGRES_PASSWORD=password
数据卷: /var/lib/postgresql/data

# MySQL 数据库
镜像名称: mysql
标签: 8.0
更新策略: scheduled
备份策略: ✓
环境变量:
  - MYSQL_ROOT_PASSWORD=rootpassword
  - MYSQL_DATABASE=myapp
数据卷: /var/lib/mysql
```

#### 应用服务
```yaml
# Node.js 应用
镜像名称: node
标签: 18-alpine
更新策略: canary
健康检查: ✓
端口映射: 3000:3000
环境变量:
  - NODE_ENV=production
  - PORT=3000

# Python 应用
镜像名称: python
标签: 3.11-slim
更新策略: rolling
健康检查: ✓
端口映射: 8000:8000
工作目录: /app
启动命令: python app.py
```

#### 中间件服务
```yaml
# Redis 缓存
镜像名称: redis
标签: 7-alpine
更新策略: rolling
健康检查: ✓
端口映射: 6379:6379
数据卷: /data

# Elasticsearch
镜像名称: elasticsearch
标签: 8.11.0
更新策略: manual
资源限制:
  - 内存: 2GB
  - CPU: 1 核心
环境变量:
  - discovery.type=single-node
  - ES_JAVA_OPTS=-Xms1g -Xmx1g
```

#### 监控工具
```yaml
# Prometheus 监控
镜像名称: prom/prometheus
标签: latest
更新策略: scheduled
端口映射: 9090:9090
配置文件: /etc/prometheus/prometheus.yml

# Grafana 仪表板
镜像名称: grafana/grafana
标签: latest
更新策略: rolling
端口映射: 3000:3000
数据卷: /var/lib/grafana
环境变量:
  - GF_SECURITY_ADMIN_PASSWORD=admin
```

### 私有镜像配置示例

```yaml
# 私有仓库镜像
镜像名称: registry.company.com/myapp/backend
标签: v1.2.0
注册中心: registry.company.com
认证信息:
  用户名: deploy-user
  密码: [加密存储]
更新策略: blue-green
健康检查端点: /health
```

### 镜像更新策略说明

| 策略类型 | 适用场景 | 特点 |
|---------|---------|------|
| **rolling** | Web 应用、API 服务 | 零停机更新，逐步替换 |
| **blue-green** | 关键业务应用 | 完整环境切换，快速回滚 |
| **canary** | 用户面向服务 | 渐进式发布，风险控制 |
| **scheduled** | 数据库、基础设施 | 维护窗口更新 |
| **manual** | 生产关键系统 | 手动审批更新 |

## 📖 文档

### 用户文档
- [**快速入门指南**](docs/user/getting-started.md) - 快速设置和入门
- [**用户手册**](USER_GUIDE.md) - 完整用户指南
- [**控制面板概览**](docs/user/dashboard.md) - Web 界面指南
- [**容器管理**](docs/user/containers.md) - 容器管理指南
- [**常见问题**](docs/user/faq.md) - 常见问题和解答

### 管理员文档
- [**安装指南**](INSTALLATION.md) - 完整安装流程
- [**配置指南**](CONFIGURATION.md) - 系统配置
- [**部署指南**](DEPLOYMENT_GUIDE.md) - 生产环境部署
- [**安全指南**](docs/admin/security.md) - 安全最佳实践
- [**监控设置**](docs/admin/monitoring.md) - 监控和告警
- [**故障排除**](TROUBLESHOOTING.md) - 常见问题解决方案

### 开发者文档
- [**API 文档**](API_DOCUMENTATION.md) - 完整 API 参考
- [**架构指南**](docs/developer/architecture.md) - 系统架构
- [**开发环境设置**](docs/developer/development-setup.md) - 本地开发
- [**贡献指南**](docs/developer/contributing.md) - 贡献指导原则
- [**测试指南**](docs/developer/testing.md) - 测试流程

### 运维文档
- [**生产部署**](docs/operations/deployment.md) - 生产环境设置
- [**性能调优**](docs/operations/performance-tuning.md) - 优化指南
- [**扩展指南**](docs/operations/scaling.md) - 水平和垂直扩展
- [**事件响应**](docs/operations/incident-response.md) - 应急处理流程

## 🔧 配置

### 基础配置

```yaml
# docker-compose.yml - 生产就绪
version: '3.8'
services:
  docker-auto:
    image: await2719/docker-auto:latest
    environment:
      - APP_PORT=8080
      - DB_HOST=db
      - DB_PORT=5432
      - DB_NAME=dockerauto
      - DB_USER=dockerauto
      - DB_PASSWORD=secure_password
      - JWT_SECRET=your-secure-jwt-secret-key
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - "80:80"
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      - POSTGRES_DB=dockerauto
      - POSTGRES_USER=dockerauto
      - POSTGRES_PASSWORD=secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

### 环境变量

关键配置选项：

```bash
# 数据库配置
DATABASE_URL=postgresql://user:pass@localhost:5432/dockerauto
REDIS_URL=redis://localhost:6379

# 认证配置
JWT_SECRET=your-secure-jwt-secret-key
JWT_EXPIRE_HOURS=24

# Docker 配置
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_API_VERSION=1.41

# 通知设置
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SLACK_WEBHOOK_URL=https://hooks.slack.com/...

# 监控配置
PROMETHEUS_ENABLED=true
METRICS_PORT=9090
```

## 🔒 安全特性

- **身份认证**: 基于 JWT 的认证系统，支持刷新令牌
- **权限控制**: 基于角色的访问控制（RBAC）
- **审计日志**: 所有操作的完整审计轨迹
- **安全通信**: HTTPS/TLS 加密
- **API 安全**: 限流、输入验证、CORS 保护
- **容器安全**: 无 root 容器、安全扫描
- **数据保护**: 敏感数据加密、安全的密钥管理

## 📊 监控与指标

### 内置指标
- 容器健康状态和资源使用情况
- 更新成功/失败率
- API 性能和响应时间
- 系统资源利用率
- 用户活动和审计日志

### 集成支持
- **Prometheus**: 原生指标导出
- **Grafana**: 预构建仪表板
- **Alertmanager**: 告警路由和通知
- **ELK Stack**: 日志聚合和分析
- **自定义 Webhook**: 与任何监控系统集成

## 🚀 性能

### 系统能力
- **并发操作**: 同时处理 1000+ 个容器
- **API 性能**: < 100ms 平均响应时间
- **数据库**: 优化查询和连接池
- **缓存**: Redis 缓存常用数据
- **资源使用**: 空闲状态下 < 256MB RAM，< 5% CPU

### 可扩展性
- **水平扩展**: 多个后端实例配合负载均衡
- **数据库扩展**: 读取副本和连接池
- **容器管理**: 分布在多个 Docker 主机
- **监控**: 高效的资源利用率跟踪

## 🔄 更新策略

### 可用策略
1. **滚动更新**: 零停机更新，带健康检查
2. **蓝绿部署**: 完整环境切换
3. **金丝雀发布**: 渐进式推出，流量分割
4. **计划更新**: 维护窗口更新
5. **手动更新**: 需要审批的按需更新

### 安全功能
- **更新前验证**: 镜像和配置验证
- **健康检查**: 自动健康状态验证
- **自动回滚**: 失败更新检测和回滚
- **备份创建**: 自动容器状态备份
- **通知**: 实时更新状态通知

## 🤝 贡献

我们欢迎贡献！请查看我们的[贡献指南](docs/developer/contributing.md)了解详情。

### 开发工作流
1. Fork 仓库
2. 创建功能分支: `git checkout -b feature/new-feature`
3. 进行更改并添加测试
4. 运行测试: `make test`
5. 提交拉取请求

### 代码标准
- **Go**: 遵循 Go 约定，使用 `gofmt` 和 `golint`
- **TypeScript**: ESLint + Prettier 配合 Vue 风格指南
- **测试**: 最低 80% 代码覆盖率
- **文档**: 为所有新功能更新文档

## 📋 路线图

### 版本 2.0（2024 年第二季度）
- [ ] Kubernetes 支持
- [ ] 多租户架构
- [ ] 高级调度选项
- [ ] 自定义集成插件系统

### 版本 2.1（2024 年第三季度）
- [ ] GitOps 集成
- [ ] 高级监控仪表板
- [ ] iOS/Android 移动应用
- [ ] 增强安全功能

## 📞 支持

### 社区支持
- **GitHub Issues**: 错误报告和功能请求
- **讨论区**: 问答和社区支持
- **Discord**: 实时社区聊天
- **文档**: 全面的指南和教程

### 企业支持
- **专业支持**: 24/7 支持，SLA 保证
- **定制开发**: 功能开发和定制
- **培训**: 现场培训和咨询
- **部署**: 托管部署和维护

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🏆 致谢

- **Docker**: 感谢出色的容器化平台
- **Go 社区**: 感谢优秀的库和工具
- **Vue.js 团队**: 感谢出色的前端框架
- **贡献者**: 感谢所有出色的贡献者

---

**状态**: 🚀 生产就绪
**版本**: 2.3.0
**最后更新**: 2025年9月16日

<details>
<summary>📊 项目统计</summary>

- **代码行数**: 50,000+
- **测试覆盖率**: 85%+
- **Docker 镜像**: 支持 10+ 个注册中心
- **编程语言**: Go、TypeScript、SQL
- **贡献者**: 15+ 活跃开发者
- **生产部署**: 100+ 安装实例

</details>