#!/bin/bash

# Docker Auto - 运行前环境检查脚本
echo "🔍 Docker Auto 运行前环境检查..."

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

success_count=0
warning_count=0
error_count=0

check_ok() {
    echo -e "   ${GREEN}✅ $1${NC}"
    success_count=$((success_count + 1))
}

check_warning() {
    echo -e "   ${YELLOW}⚠️  $1${NC}"
    warning_count=$((warning_count + 1))
}

check_error() {
    echo -e "   ${RED}❌ $1${NC}"
    error_count=$((error_count + 1))
}

echo -e "${BLUE}📋 1. 基础环境检查${NC}"

# Node.js 检查
if command -v node >/dev/null 2>&1; then
    node_version=$(node --version)
    if [[ "${node_version#v}" =~ ^1[8-9]\.|^[2-9][0-9]\. ]]; then
        check_ok "Node.js 版本: $node_version"
    else
        check_warning "Node.js 版本较旧: $node_version (建议 18+)"
    fi
else
    check_error "Node.js 未安装"
fi

# Go 检查
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | cut -d' ' -f3)
    check_ok "Go 版本: $go_version"
else
    check_error "Go 未安装"
fi

# Docker 检查
if command -v docker >/dev/null 2>&1; then
    docker_version=$(docker --version | cut -d' ' -f3 | sed 's/,//')
    check_ok "Docker 版本: $docker_version"
else
    check_error "Docker 未安装"
fi

echo ""
echo -e "${BLUE}📋 2. Docker 权限检查${NC}"

# Docker socket 检查
if [ -S /var/run/docker.sock ]; then
    check_ok "Docker socket 存在"

    # 权限检查
    if groups | grep -q docker; then
        check_ok "用户在 docker 组中"
    else
        check_error "用户不在 docker 组中，需要运行: sudo usermod -aG docker \$USER"
    fi

    # Docker 连接测试
    if docker ps >/dev/null 2>&1; then
        check_ok "Docker API 连接正常"
    else
        check_error "Docker API 连接失败"
    fi
else
    check_error "Docker socket 不存在"
fi

echo ""
echo -e "${BLUE}📋 3. 端口检查${NC}"

# 检查端口是否被占用
check_port() {
    local port=$1
    local service=$2

    if netstat -tuln 2>/dev/null | grep -q ":$port "; then
        check_warning "端口 $port 已被占用 ($service)"
    else
        check_ok "端口 $port 可用 ($service)"
    fi
}

check_port 8080 "后端服务"
check_port 5173 "前端开发服务"

echo ""
echo -e "${BLUE}📋 4. 项目配置检查${NC}"

# 环境配置文件
if [ -f "frontend/.env.development" ]; then
    check_ok "前端开发环境配置存在"
else
    check_error "前端开发环境配置缺失"
fi

if [ -f ".env.dev" ]; then
    check_ok "后端开发环境配置存在"
else
    check_warning "后端开发环境配置缺失（启动脚本会自动创建）"
fi

# 依赖检查
if [ -d "frontend/node_modules" ]; then
    check_ok "前端依赖已安装"
else
    check_warning "前端依赖未安装（启动脚本会自动安装）"
fi

if [ -f "backend/go.mod" ]; then
    check_ok "Go 模块文件存在"
else
    check_error "Go 模块文件缺失"
fi

echo ""
echo -e "${BLUE}📋 5. 安全配置检查${NC}"

# JWT Secret 检查
if [ -f ".env.dev" ] && grep -q "JWT_SECRET" .env.dev; then
    jwt_secret=$(grep "JWT_SECRET" .env.dev | cut -d'=' -f2)
    if [ ${#jwt_secret} -ge 32 ]; then
        check_ok "JWT 密钥长度足够"
    else
        check_warning "JWT 密钥长度不足（建议至少32位）"
    fi
fi

# CORS 配置检查
if grep -q "Access-Control-Allow-Origin.*\*" backend/cmd/server/main.go; then
    check_warning "CORS 配置为允许所有来源（开发环境正常）"
else
    check_ok "CORS 配置限制来源"
fi

echo ""
echo -e "${BLUE}📋 6. 网络连接检查${NC}"

# 检查基本网络连接
if curl -s --max-time 5 http://localhost:8080/api/health >/dev/null 2>&1; then
    check_warning "后端服务已在运行"
elif curl -s --max-time 2 http://google.com >/dev/null 2>&1; then
    check_ok "网络连接正常"
else
    check_warning "网络连接可能有问题"
fi

echo ""
echo "=========================================="
echo -e "检查结果汇总:"
echo -e "   ${GREEN}✅ 成功: $success_count 项${NC}"
echo -e "   ${YELLOW}⚠️  警告: $warning_count 项${NC}"
echo -e "   ${RED}❌ 错误: $error_count 项${NC}"

if [ $error_count -eq 0 ]; then
    if [ $warning_count -eq 0 ]; then
        echo -e "${GREEN}🎉 所有检查通过，系统已准备就绪！${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠️  有警告项目，但可以继续运行${NC}"
        exit 0
    fi
else
    echo -e "${RED}❌ 发现严重问题，请先解决错误项目${NC}"
    echo ""
    echo "🔧 常见问题解决方案:"
    echo "   • Docker 权限: sudo usermod -aG docker \$USER && newgrp docker"
    echo "   • 端口占用: sudo lsof -i :8080 或 sudo lsof -i :5173"
    echo "   • 安装依赖: cd frontend && npm install"
    exit 1
fi