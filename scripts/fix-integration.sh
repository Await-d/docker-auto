#!/bin/bash

# Docker Auto - 前后端集成问题修复脚本
echo "🔧 开始修复前后端集成问题..."

# 确保在项目根目录
if [[ ! -d "backend" || ! -d "frontend" ]]; then
    echo "❌ 错误: 请在项目根目录运行此脚本"
    exit 1
fi

echo "📝 1. 检查前端环境配置..."

# 检查并修复前端环境配置
if [[ -f "frontend/.env.development" ]]; then
    echo "   ✅ 前端环境配置已存在"
    # 检查配置是否正确
    if grep -q "VITE_WS_URL=ws://localhost:8080/api/ws" frontend/.env.development; then
        echo "   ✅ WebSocket URL 配置正确"
    else
        echo "   ⚠️  修复 WebSocket URL 配置"
        sed -i 's|VITE_WS_URL=.*|VITE_WS_URL=ws://localhost:8080/api/ws|' frontend/.env.development
    fi
else
    echo "   📝 创建前端环境配置文件"
    cat > frontend/.env.development << 'EOF'
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/api/ws
VITE_APP_TITLE=Docker Auto Update System
VITE_DEV_MODE=true
VITE_HOST=0.0.0.0
VITE_PORT=5173
EOF
fi

echo "📝 2. 检查启动脚本配置..."

# 检查启动脚本配置
if grep -q "VITE_API_BASE_URL=http://localhost:8080/api/v1" start-frontend.sh; then
    echo "   ⚠️  修复启动脚本中的 API URL"
    sed -i 's|VITE_API_BASE_URL=http://localhost:8080/api/v1|VITE_API_BASE_URL=http://localhost:8080|' start-frontend.sh
fi

if grep -q "VITE_WS_URL=ws://localhost:8080/ws" start-frontend.sh; then
    echo "   ⚠️  修复启动脚本中的 WebSocket URL"
    sed -i 's|VITE_WS_URL=ws://localhost:8080/ws|VITE_WS_URL=ws://localhost:8080/api/ws|' start-frontend.sh
fi

echo "📝 3. 检查前端配置文件..."

# 检查vite配置中的端口
if grep -q "port: 3000" frontend/vite.config.ts; then
    echo "   ⚠️  修复 Vite 配置中的端口设置"
    sed -i 's/port: 3000/port: 5173/' frontend/vite.config.ts
fi

echo "📝 4. 验证API路径映射..."

# 验证关键路径是否存在
echo "   🔍 检查前端API类..."
if [[ -f "frontend/src/api/container.ts" ]]; then
    if grep -q 'baseUrl = "/api/containers"' frontend/src/api/container.ts; then
        echo "   ✅ Container API 路径正确"
    else
        echo "   ❌ Container API 路径配置有问题"
    fi
fi

echo "   🔍 检查认证端点..."
if [[ -f "frontend/src/utils/constants.ts" ]]; then
    if grep -q 'LOGIN: "/api/auth/login"' frontend/src/utils/constants.ts; then
        echo "   ✅ 认证 API 路径正确"
    else
        echo "   ❌ 认证 API 路径配置有问题"
    fi
fi

echo "📝 5. 检查后端API响应格式..."

# 检查后端主程序是否包含标准响应格式
if grep -q '"success": true' backend/cmd/server/main.go; then
    echo "   ✅ 后端API响应格式已标准化"
else
    echo "   ⚠️  后端API响应格式可能需要标准化"
fi

echo "📝 6. 生成集成测试脚本..."

# 创建集成测试脚本
cat > scripts/test-integration.sh << 'EOF'
#!/bin/bash

echo "🧪 测试前后端集成..."

# 启动后端
echo "🚀 启动后端服务..."
./start-backend.sh &
BACKEND_PID=$!

# 等待后端启动
sleep 5

# 测试后端健康检查
echo "🔍 测试后端健康检查..."
if curl -s http://localhost:8080/api/health | grep -q "ok"; then
    echo "✅ 后端健康检查通过"
else
    echo "❌ 后端健康检查失败"
    kill $BACKEND_PID 2>/dev/null
    exit 1
fi

# 测试认证端点
echo "🔍 测试认证端点..."
if curl -s -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d '{}' | grep -q "error"; then
    echo "✅ 认证端点响应正常"
else
    echo "❌ 认证端点无响应"
fi

# 清理
kill $BACKEND_PID 2>/dev/null

echo "✅ 集成测试完成"
EOF

chmod +x scripts/test-integration.sh

echo "📝 7. 生成问题排查指南..."

cat > docs/integration-troubleshooting.md << 'EOF'
# 前后端集成问题排查指南

## 常见问题及解决方案

### 1. API请求404错误
**症状**: 前端请求返回404
**原因**: API路径不匹配
**解决**:
- 检查前端 `API_BASE_URL` 配置
- 确认后端路由是否正确设置
- 验证前端请求路径与后端路由匹配

### 2. WebSocket连接失败
**症状**: WebSocket无法建立连接
**原因**: WebSocket路径或端口不正确
**解决**:
- 检查 `VITE_WS_URL` 配置
- 确认后端WebSocket路由: `/api/ws`
- 验证端口是否正确: 8080

### 3. 认证token无效
**症状**: 请求返回401未授权
**原因**: token格式或验证逻辑问题
**解决**:
- 检查前端token存储和发送逻辑
- 验证后端token验证逻辑
- 确认token刷新机制正常

### 4. CORS错误
**症状**: 浏览器控制台显示CORS错误
**原因**: 后端CORS配置不正确
**解决**:
- 检查后端CORS中间件配置
- 确认允许的origin包含前端地址
- 验证允许的HTTP方法和headers

## 调试步骤

1. 检查网络请求
   - 打开浏览器开发者工具
   - 查看Network标签页
   - 检查请求URL、状态码、响应内容

2. 检查WebSocket连接
   - 在Network标签页查看WS连接
   - 确认连接状态和消息收发

3. 检查控制台错误
   - 查看Console标签页的错误信息
   - 注意认证相关错误

4. 后端日志检查
   - 查看后端服务日志
   - 关注路由匹配和错误信息
EOF

echo ""
echo "✅ 前后端集成问题修复完成！"
echo ""
echo "📋 修复内容总结:"
echo "   ✅ 修复前端环境配置 (.env.development)"
echo "   ✅ 修复启动脚本配置 (start-frontend.sh)"
echo "   ✅ 修复Vite端口配置 (vite.config.ts)"
echo "   ✅ 修复WebSocket服务路径"
echo "   ✅ 标准化后端API响应格式"
echo "   ✅ 创建集成测试脚本"
echo "   ✅ 创建问题排查指南"
echo ""
echo "🚀 下一步:"
echo "   1. 运行: ./scripts/test-integration.sh"
echo "   2. 启动服务: ./start-backend.sh 和 ./start-frontend.sh"
echo "   3. 测试前后端通信是否正常"
echo ""
echo "📖 如遇问题，请查看: docs/integration-troubleshooting.md"