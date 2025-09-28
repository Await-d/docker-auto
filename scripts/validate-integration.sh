#!/bin/bash

# Docker Auto - 最终集成验证脚本
echo "🔍 最终前后端集成验证..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success_count=0
total_count=0

check_item() {
    local description="$1"
    local check_command="$2"
    local expected_result="$3"

    total_count=$((total_count + 1))
    echo -n "   检查 $description... "

    if eval "$check_command" > /dev/null 2>&1; then
        if [ -n "$expected_result" ]; then
            if eval "$check_command" | grep -q "$expected_result"; then
                echo -e "${GREEN}✅ 通过${NC}"
                success_count=$((success_count + 1))
            else
                echo -e "${RED}❌ 失败 (结果不匹配)${NC}"
            fi
        else
            echo -e "${GREEN}✅ 通过${NC}"
            success_count=$((success_count + 1))
        fi
    else
        echo -e "${RED}❌ 失败${NC}"
    fi
}

echo "📋 1. 前端配置验证"

check_item "前端环境配置存在" "test -f frontend/.env.development"
check_item "API基础URL正确" "grep -q 'VITE_API_BASE_URL=http://localhost:8080' frontend/.env.development"
check_item "WebSocket URL正确" "grep -q 'VITE_WS_URL=ws://localhost:8080/api/ws' frontend/.env.development"
check_item "前端端口配置" "grep -q 'port: 5173' frontend/vite.config.ts"

echo ""
echo "📋 2. 后端配置验证"

check_item "后端认证路由" "grep -q 'Group(\"/auth\")' backend/cmd/server/main.go"
check_item "后端WebSocket路由" "grep -q '/api/ws' backend/cmd/server/main.go"
check_item "后端ping端点" "grep -q '/ping' backend/cmd/server/main.go"
check_item "后端容器路由" "grep -q '/api/containers' backend/cmd/server/main.go"

echo ""
echo "📋 3. API路径映射验证"

check_item "前端Container API路径" "grep -q 'baseUrl = \"/api/containers\"' frontend/src/api/container.ts"
check_item "前端认证端点" "grep -q 'LOGIN: \"/api/auth/login\"' frontend/src/utils/constants.ts"
check_item "前端token刷新路径" "grep -q '/api/auth/refresh' frontend/src/utils/request.ts"

echo ""
echo "📋 4. 响应格式验证"

check_item "后端标准响应格式" "grep -q '\"success\": true' backend/cmd/server/main.go"
check_item "前端响应接口定义" "grep -q 'success: boolean' frontend/src/utils/request.ts"

echo ""
echo "📋 5. WebSocket配置验证"

check_item "前端WebSocket构造" "grep -q 'api/ws' frontend/src/utils/websocket.ts"
check_item "updateWebSocket修复" "grep -q '/api/ws' frontend/src/services/updateWebSocket.ts"

echo ""
echo "📋 6. 文件结构验证"

check_item "启动脚本存在" "test -f start-frontend.sh && test -f start-backend.sh"
check_item "前端源码结构" "test -d frontend/src/api && test -d frontend/src/components"
check_item "后端源码结构" "test -d backend/internal && test -d backend/cmd/server"

echo ""
echo "📋 7. 生产环境配置"

check_item "生产环境配置文件" "test -f frontend/.env.production"
check_item "后端静态文件服务" "grep -q 'frontend/dist' backend/cmd/server/main.go"

echo ""
echo "=========================================="
echo -e "验证结果: ${GREEN}${success_count}/${total_count}${NC} 项通过"

if [ $success_count -eq $total_count ]; then
    echo -e "${GREEN}🎉 所有检查项目都通过了！前后端集成配置正确。${NC}"
    exit 0
else
    failed_count=$((total_count - success_count))
    echo -e "${YELLOW}⚠️  有 ${failed_count} 项检查失败，请查看上面的详细信息。${NC}"
    exit 1
fi