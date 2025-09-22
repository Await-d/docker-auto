# Technology Stack

## Project Type
企业级Web应用 - Docker容器自动更新管理系统，采用前后端分离架构，提供Web界面和API服务。

## Core Technologies

### Primary Language(s)
- **后端语言**: Go 1.23+ (使用工具链 1.24.7)
- **前端语言**: TypeScript 5.3+
- **构建工具**: Go modules, Vite 5.0+
- **包管理器**: go mod (后端), npm/yarn (前端)

### Key Dependencies/Libraries

#### 后端核心依赖
- **Gin v1.9.1**: HTTP Web框架，提供REST API服务
- **GORM**: ORM框架，支持PostgreSQL/MySQL/SQLite
- **Docker SDK v25.0.0**: Docker客户端，容器操作核心
- **JWT v5.3.0**: 用户认证和授权
- **WebSocket v1.5.3**: 实时通信支持
- **Viper v1.18.2**: 配置管理
- **Logrus v1.9.3**: 结构化日志
- **Cron v3.0.1**: 任务调度

#### 前端核心依赖
- **Vue 3.4+**: 渐进式前端框架
- **Element Plus 2.4+**: UI组件库
- **Pinia 2.1+**: 状态管理
- **Vue Router 4.2+**: 路由管理
- **Axios 1.6+**: HTTP客户端
- **ECharts 5.4+**: 数据可视化
- **VueUse 13.9+**: 组合式工具库

### Application Architecture
采用**微服务单体架构** (Modular Monolith)：
- **前端**: SPA单页应用，组件化开发
- **后端**: 分层架构 + 领域驱动设计 (DDD)
- **通信**: RESTful API + WebSocket双向通信
- **数据流**: 事件驱动 + 响应式状态管理

### Data Storage
- **主数据库**: PostgreSQL 13+ (生产环境推荐)
- **开发数据库**: SQLite (零配置开发)
- **备选数据库**: MySQL 8.0+
- **缓存层**: Redis (可选，用于会话和数据缓存)
- **数据格式**: JSON (API), GORM模型 (ORM)

### External Integrations
- **Docker Registry**: Docker Hub, Harbor, AWS ECR, 私有注册中心
- **通知服务**: SendGrid (邮件), Slack API, Webhook
- **监控集成**: Prometheus指标, Grafana仪表板 (可选)
- **协议**: HTTP/HTTPS, WebSocket, Docker Remote API
- **认证**: JWT Bearer Token, Docker Registry认证

### Monitoring & Dashboard Technologies
- **前端框架**: Vue 3 + TypeScript + Element Plus
- **实时通信**: WebSocket双向通信
- **可视化**: ECharts图表库, Vue Grid Layout
- **状态管理**: Pinia响应式状态
- **路由**: Vue Router with guards

## Development Environment

### Build & Development Tools
- **后端构建**: Go原生编译 (`go build`)
- **前端构建**: Vite热重载开发服务器
- **包管理**: go mod tidy, npm/yarn install
- **开发脚本**: start-backend.sh, start-frontend.sh
- **容器化**: Docker + Docker Compose

### Code Quality Tools
- **Go静态分析**: go vet, golint (可配置)
- **TS/JS检查**: ESLint + TypeScript compiler
- **代码格式化**: gofmt (Go), Prettier (前端)
- **测试框架**: Go原生testing, Vue Test Utils (前端)
- **文档**: Go docs, JSDoc, Swagger/OpenAPI

### Version Control & Collaboration
- **版本控制**: Git with semantic commit messages
- **分支策略**: GitHub Flow (master + feature branches)
- **代码审查**: Pull Request + GitHub Actions CI
- **自动化**: GitHub Actions (CI/CD, 依赖更新, Docker构建)

### Dashboard Development
- **热重载**: Vite HMR (毫秒级前端更新)
- **端口管理**: 前端5173, 后端8080, 可配置
- **多实例**: 支持多环境同时开发
- **代理配置**: Vite代理后端API请求

## Deployment & Distribution

- **目标平台**: Linux服务器, Docker容器, Kubernetes
- **分发方式**: Docker镜像, GitHub Releases, 源码编译
- **安装要求**: Docker + Docker Compose 或 Go 1.23+ + Node 18+
- **更新机制**: Docker镜像标签, GitHub自动发布
- **配置**: 环境变量 + 配置文件

## Technical Requirements & Constraints

### Performance Requirements
- **API响应时间**: <200ms (90th percentile)
- **WebSocket延迟**: <50ms
- **内存占用**: 后端<512MB, 前端<100MB (打包后<10MB)
- **启动时间**: <10秒
- **并发支持**: 100+ WebSocket连接

### Compatibility Requirements
- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+), macOS (开发)
- **架构**: AMD64, ARM64
- **浏览器**: Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
- **Docker版本**: 20.10+
- **数据库**: PostgreSQL 13+, MySQL 8.0+, SQLite 3.35+

### Security & Compliance
- **认证**: JWT Bearer Token (256-bit签名)
- **传输加密**: HTTPS/TLS 1.3, WSS WebSocket
- **输入验证**: GORM验证, 前端表单验证
- **Docker安全**: 非root用户运行, 镜像安全扫描
- **审计日志**: 所有用户操作记录

### Scalability & Reliability
- **预期负载**: 10-100个用户, 1000+容器管理
- **可用性**: 99.9%运行时间目标
- **数据备份**: 数据库备份策略
- **故障恢复**: 自动重启, 健康检查, 回滚机制

## Technical Decisions & Rationale

### Decision Log

1. **Go后端 vs Node.js**:
   - 选择Go: 更好的并发性能, 静态编译, Docker生态匹配
   - 权衡: 学习曲线 vs 运行时性能和资源效率

2. **Vue 3 vs React**:
   - 选择Vue 3: 更简洁的语法, 优秀的中文文档, Element Plus生态
   - 权衡: 社区规模 vs 开发效率和团队技能匹配

3. **GORM vs 原生SQL**:
   - 选择GORM: 类型安全, 迁移管理, 多数据库支持
   - 权衡: 性能开销 vs 开发效率和维护性

4. **WebSocket vs SSE**:
   - 选择WebSocket: 双向通信需求, 实时日志流
   - 权衡: 复杂性 vs 功能完整性

5. **单体 vs 微服务**:
   - 选择模块化单体: 简化部署, 减少网络开销, 团队规模匹配
   - 权衡: 可扩展性 vs 运维复杂度

## Known Limitations

- **单点故障**: 当前为单实例部署, 未来需要集群支持
- **缓存策略**: 基础缓存实现, 可优化为分布式缓存
- **日志管理**: 本地文件日志, 未来可集成ELK或类似方案
- **监控深度**: 基础指标收集, 可增强APM和分布式追踪
- **多租户**: 当前单租户设计, 企业级需要多租户支持