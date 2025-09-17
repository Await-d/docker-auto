# 开发环境设置

## 概述

本指南将帮助开发者搭建 Docker Auto 的本地开发环境，包括前端、后端和数据库的配置。

## 前置要求

### 系统要求
- **操作系统**: Linux, macOS, Windows (WSL2)
- **内存**: 8GB+ 推荐
- **磁盘空间**: 20GB+ 可用空间

### 软件依赖

#### 必需软件
```bash
# Go 环境
Go 1.21+

# Node.js 环境
Node.js 18+
npm 9+

# 数据库
PostgreSQL 13+

# 容器环境
Docker 20.10+
Docker Compose 2.0+

# 版本控制
Git 2.30+
```

#### 开发工具推荐
```bash
# 代码编辑器
VS Code + Go 扩展
VS Code + Vue 扩展

# API 测试
Postman 或 Insomnia

# 数据库管理
pgAdmin 或 DBeaver
```

## 环境搭建

### 1. 克隆代码仓库
```bash
git clone https://github.com/your-org/docker-auto.git
cd docker-auto
```

### 2. 数据库环境搭建

#### 方式一：Docker 快速启动
```bash
# 启动 PostgreSQL 和 Redis
docker-compose -f docker-compose.dev.yml up -d postgres redis

# 验证服务状态
docker-compose -f docker-compose.dev.yml ps
```

#### 方式二：本地安装
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql-13 redis-server

# macOS
brew install postgresql redis

# 启动服务
sudo systemctl start postgresql redis-server

# 创建开发数据库
sudo -u postgres createuser -P dockerauto_dev
sudo -u postgres createdb -O dockerauto_dev dockerauto_dev
```

### 3. 后端开发环境

#### 配置环境变量
```bash
cd backend
cp .env.example .env
```

编辑 `.env` 文件：
```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=dockerauto_dev
DB_USER=dockerauto_dev
DB_PASSWORD=dev_password

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT 配置
JWT_SECRET=development-secret-key-change-in-production

# 应用配置
APP_ENV=development
APP_PORT=8080
APP_LOG_LEVEL=debug

# Docker 配置
DOCKER_HOST=unix:///var/run/docker.sock
```

#### 安装依赖
```bash
cd backend

# 下载 Go 模块
go mod download

# 验证依赖
go mod verify
```

#### 数据库迁移
```bash
# 运行数据库迁移
go run cmd/migrate/main.go

# 或使用 make 命令
make migrate
```

#### 启动后端服务
```bash
# 开发模式启动（热重载）
go run cmd/server/main.go

# 或使用 air 工具热重载
air -c .air.toml
```

### 4. 前端开发环境

#### 安装依赖
```bash
cd frontend

# 安装 npm 依赖
npm install

# 验证依赖
npm audit
```

#### 配置环境变量
```bash
# 创建环境配置文件
cp .env.example .env.local
```

编辑 `.env.local`：
```bash
# API 配置
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_BASE_URL=ws://localhost:8080/ws

# 应用配置
VITE_APP_TITLE=Docker Auto (Dev)
VITE_APP_VERSION=dev
```

#### 启动前端服务
```bash
# 开发服务器
npm run dev

# 指定端口启动
npm run dev -- --port 3000
```

## 开发工具配置

### VS Code 配置

#### 推荐扩展
```json
{
  "recommendations": [
    "golang.go",
    "vue.volar",
    "bradlc.vscode-tailwindcss",
    "esbenp.prettier-vscode",
    "ms-vscode.vscode-typescript-next",
    "formulahendry.auto-rename-tag"
  ]
}
```

#### 工作区配置
```json
{
  "go.toolsManagement.checkForUpdates": "local",
  "go.useLanguageServer": true,
  "go.formatTool": "gofmt",
  "go.lintTool": "golangci-lint",

  "typescript.preferences.importModuleSpecifier": "relative",
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  }
}
```

### Git 配置

#### Git Hooks
```bash
# 安装 pre-commit hooks
npm install -g pre-commit
pre-commit install

# 或使用 husky
npm install --save-dev husky
npx husky install
```

#### .gitignore 配置
```gitignore
# 后端
backend/.env
backend/tmp/
backend/logs/
backend/docker-auto

# 前端
frontend/node_modules/
frontend/dist/
frontend/.env.local
frontend/.vite/

# IDE
.vscode/settings.json
.idea/

# 系统文件
.DS_Store
Thumbs.db
```

## 数据库管理

### 数据库迁移
```bash
# 创建新迁移
go run cmd/migrate/main.go create add_user_table

# 执行迁移
go run cmd/migrate/main.go up

# 回滚迁移
go run cmd/migrate/main.go down

# 查看迁移状态
go run cmd/migrate/main.go status
```

### 种子数据
```bash
# 加载测试数据
go run cmd/seed/main.go

# 清空数据库
go run cmd/seed/main.go --reset
```

### 数据库连接测试
```bash
# 测试连接
psql -h localhost -U dockerauto_dev -d dockerauto_dev

# 或使用 Go 测试
go run cmd/test-db/main.go
```

## 调试配置

### Go 后端调试

#### VS Code 调试配置
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/backend/cmd/server/main.go",
      "env": {
        "APP_ENV": "development"
      },
      "args": []
    }
  ]
}
```

#### Delve 命令行调试
```bash
# 安装 delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试会话
dlv debug cmd/server/main.go

# 调试命令
(dlv) break main.main
(dlv) continue
(dlv) next
(dlv) print variable_name
```

### 前端调试

#### 浏览器调试
```javascript
// 开发工具中设置断点
debugger;

// 或使用 console 调试
console.log('Debug info:', variable);
console.table(arrayData);
console.trace('Call stack');
```

#### Vue Devtools
```bash
# 安装浏览器扩展
# Chrome: Vue.js devtools
# Firefox: Vue.js devtools
```

## 测试环境

### 单元测试

#### Go 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/service

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### 前端测试
```bash
# 单元测试
npm run test:unit

# 监听模式
npm run test:unit -- --watch

# 覆盖率报告
npm run test:unit -- --coverage
```

### 集成测试

#### API 测试
```bash
# 启动测试数据库
docker-compose -f docker-compose.test.yml up -d

# 运行集成测试
go test ./tests/integration/...
```

#### E2E 测试
```bash
# 启动完整环境
docker-compose -f docker-compose.dev.yml up -d

# 运行 E2E 测试
npm run test:e2e
```

## 开发工作流

### 分支策略
```bash
# 功能开发
git checkout -b feature/container-monitoring
git push -u origin feature/container-monitoring

# Bug 修复
git checkout -b fix/login-validation
git push -u origin fix/login-validation

# 热修复
git checkout -b hotfix/security-patch
git push -u origin hotfix/security-patch
```

### 代码规范

#### Go 代码检查
```bash
# 格式化代码
gofmt -w .

# 代码检查
golangci-lint run

# 静态分析
go vet ./...
```

#### 前端代码检查
```bash
# ESLint 检查
npm run lint

# 修复可自动修复的问题
npm run lint:fix

# 格式化代码
npm run format
```

### 提交规范
```bash
# 功能提交
git commit -m "feat(containers): add batch update functionality"

# Bug 修复
git commit -m "fix(auth): resolve JWT token validation issue"

# 文档更新
git commit -m "docs(api): update container API documentation"
```

## 故障排除

### 常见开发问题

#### 数据库连接失败
```bash
# 检查数据库服务状态
sudo systemctl status postgresql

# 检查端口占用
netstat -tlnp | grep 5432

# 重启数据库服务
sudo systemctl restart postgresql
```

#### Go 模块问题
```bash
# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download

# 验证模块
go mod verify
```

#### 前端构建问题
```bash
# 清理 node_modules
rm -rf node_modules package-lock.json
npm install

# 清理构建缓存
npm run build --clean
```

#### Docker 权限问题
```bash
# 添加用户到 docker 组
sudo usermod -aG docker $USER

# 重新登录或刷新组权限
newgrp docker

# 验证权限
docker ps
```

## 性能优化

### 开发环境优化
```bash
# 使用 air 热重载
go install github.com/cosmtrek/air@latest

# 前端热重载优化
# vite.config.ts
export default defineConfig({
  server: {
    hmr: {
      overlay: false
    }
  }
})
```

### 构建优化
```bash
# Go 构建优化
go build -ldflags="-s -w" -o docker-auto cmd/server/main.go

# 前端构建优化
npm run build -- --mode development
```

---

**下一步**: 查看 [贡献指南](contributing.md) 了解如何为项目贡献代码