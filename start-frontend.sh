#!/bin/bash

# 启动前端服务
echo "🎨 启动Docker Auto前端服务..."

# 检查Node.js环境
if ! command -v node &> /dev/null; then
    echo "❌ 错误: 未安装Node.js，请先安装Node.js 18+"
    exit 1
fi

# 检查项目结构
if [[ ! -d "frontend" || ! -f "frontend/package.json" ]]; then
    echo "❌ 错误: 请在项目根目录运行此脚本"
    exit 1
fi

# 进入前端目录
cd frontend

# 创建环境配置
if [[ ! -f ".env.development" ]]; then
    echo "📝 创建前端开发环境配置..."
    cat > .env.development << 'EOF'
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
VITE_APP_TITLE=Docker Auto Update System
VITE_DEV_MODE=true
VITE_HOST=0.0.0.0
VITE_PORT=5173
EOF
fi

# 检查依赖
if [[ ! -d "node_modules" ]]; then
    echo "📦 安装依赖..."
    if command -v yarn &> /dev/null && [[ -f "yarn.lock" ]]; then
        yarn install
    else
        npm install
    fi
fi

# 启动服务
echo "✅ 启动前端服务..."
echo "📍 前端地址: http://localhost:5173"
echo "📍 后端API: http://localhost:8080/api/v1"
echo "💡 按 Ctrl+C 停止服务"
echo ""

if command -v yarn &> /dev/null && [[ -f "yarn.lock" ]]; then
    yarn dev
else
    npm run dev
fi