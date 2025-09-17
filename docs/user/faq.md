# 常见问题解答

## 安装和配置

### Q: 系统要求是什么？
A: 最低要求：
- Docker 20.10+
- 2GB+ 内存
- PostgreSQL 数据库
- 端口 80 可用

### Q: 如何连接到现有的 PostgreSQL 数据库？
A: 在启动容器时提供数据库连接参数：
```bash
docker run -d \
  -e DB_HOST=your-postgres-host \
  -e DB_PORT=5432 \
  -e DB_NAME=dockerauto \
  -e DB_USER=your-user \
  -e DB_PASSWORD=your-password \
  await2719/docker-auto:latest
```

### Q: 忘记了管理员密码怎么办？
A: 可以通过数据库直接重置：
```sql
UPDATE users SET password = '新密码哈希' WHERE email = 'admin@example.com';
```

## 容器管理

### Q: 为什么容器无法启动？
A: 常见原因：
1. 端口冲突 - 检查端口是否被占用
2. 镜像不存在 - 验证镜像名称和标签
3. 权限不足 - 确保 Docker sock 有正确权限
4. 资源不足 - 检查内存和 CPU 限制

### Q: 如何查看容器日志？
A: 在容器详情页面点击"查看日志"按钮，或者通过 API：
```bash
curl http://localhost/api/containers/{id}/logs
```

### Q: 支持哪些更新策略？
A: 系统支持以下更新策略：
- **rolling**: 滚动更新，逐步替换
- **blue-green**: 蓝绿部署，环境切换
- **canary**: 金丝雀发布，流量分割
- **scheduled**: 计划更新，指定时间
- **manual**: 手动更新，需要确认

## 监控和告警

### Q: 如何设置告警通知？
A: 在设置页面配置通知渠道：
1. 邮件通知 - 配置 SMTP 服务器
2. Slack 通知 - 添加 Webhook URL
3. 自定义 Webhook - 配置自定义接口

### Q: 监控指标包括哪些？
A: 主要监控指标：
- 容器健康状态
- CPU 和内存使用率
- 网络流量统计
- 磁盘使用情况
- 更新成功/失败率

## 安全相关

### Q: 如何启用 HTTPS？
A: 建议使用反向代理：
```nginx
server {
    listen 443 ssl;
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:80;
    }
}
```

### Q: 支持哪些认证方式？
A: 目前支持：
- 用户名/密码认证
- JWT Token 认证
- API Key 认证

## 故障排除

### Q: 服务无法启动，如何排查？
A: 按以下步骤排查：
1. 检查容器日志：`docker logs docker-auto-system`
2. 验证数据库连接：测试数据库连接参数
3. 检查端口占用：`netstat -tlnp | grep :80`
4. 验证权限：确保有 Docker socket 访问权限

### Q: 性能较慢，如何优化？
A: 优化建议：
1. 增加内存分配
2. 使用 Redis 缓存
3. 启用数据库连接池
4. 配置 CDN 加速静态资源

### Q: 数据备份策略？
A: 备份重要数据：
1. PostgreSQL 数据库定期备份
2. 容器配置文件备份
3. 日志文件归档保存

## 技术支持

如果以上解答不能解决您的问题，请：

1. 查看 [GitHub Issues](https://github.com/your-org/docker-auto/issues)
2. 提交新的 Issue 并提供详细信息
3. 参与 [GitHub Discussions](https://github.com/your-org/docker-auto/discussions)

---

**需要更多帮助？** 请查看完整的 [用户指南](../README.md) 或联系技术支持。