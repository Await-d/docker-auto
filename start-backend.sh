#!/bin/bash

# 启动后端服务
echo "🚀 启动Docker Auto后端服务..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未安装Go，请先安装Go 1.23+"
    exit 1
fi

# 检查项目结构
if [[ ! -f "backend/go.mod" || ! -d "backend/cmd/server" ]]; then
    echo "❌ 错误: 请在项目根目录运行此脚本"
    exit 1
fi

# 创建环境配置
if [[ ! -f ".env.dev" ]]; then
    echo "📝 创建开发环境配置..."
    cat > .env.dev << 'EOF'
APP_PORT=8080
APP_ENV=development
LOG_LEVEL=debug
LOG_FORMAT=text

# 数据库配置 - 开发环境使用SQLite（零配置）
DB_TYPE=sqlite
DB_PATH=../data/docker-auto.db

# 如需使用PostgreSQL，取消注释并注释上面的SQLite配置：
# DB_TYPE=postgres
# DB_HOST=localhost
# DB_PORT=5432
# DB_NAME=dockerauto_dev
# DB_USER=postgres
# DB_PASSWORD=password
# DB_SSL_MODE=disable

# 如需使用MySQL，取消注释并注释上面的配置：
# DB_TYPE=mysql
# DB_HOST=localhost
# DB_PORT=3306
# DB_NAME=dockerauto_dev
# DB_USER=root
# DB_PASSWORD=password

JWT_SECRET=development-jwt-secret-key-32-chars-minimum-required-for-security
DOCKER_HOST=unix:///var/run/docker.sock
TZ=Asia/Shanghai
EOF
fi

# 创建必要目录
mkdir -p data logs tmp

# 进入后端目录
cd backend

# 下载依赖
echo "📦 下载依赖..."
go mod download
go mod tidy

# 导出环境变量
set -a  # 自动导出所有变量
source ../.env.dev
set +a  # 关闭自动导出

# 启动服务
echo "✅ 启动后端服务..."
echo "📍 服务地址: http://localhost:8080"
echo "📍 API文档: http://localhost:8080/swagger"
echo "💡 按 Ctrl+C 停止服务"
echo ""

go run ./cmd/server