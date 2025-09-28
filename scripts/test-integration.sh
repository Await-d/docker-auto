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
