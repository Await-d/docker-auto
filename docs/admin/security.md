# 安全配置指南

## 概述

Docker Auto 系统的安全配置涵盖用户认证、权限管理、数据加密、网络安全等多个方面。本指南帮助管理员建立完善的安全防护体系。

## 用户认证与授权

### JWT 认证配置

#### JWT 密钥管理
```bash
# 生成强密钥
JWT_SECRET=$(openssl rand -base64 32)
echo "JWT_SECRET=$JWT_SECRET" >> .env
```

#### JWT 配置参数
```yaml
jwt:
  secret: "your-256-bit-secret-key"
  expires_in: "24h"          # Token 有效期
  refresh_expires_in: "7d"   # 刷新 Token 有效期
  issuer: "docker-auto"      # 签发者
  algorithm: "HS256"         # 加密算法
```

### 密码策略

#### 密码复杂度要求
```yaml
password_policy:
  min_length: 8              # 最小长度
  require_uppercase: true    # 需要大写字母
  require_lowercase: true    # 需要小写字母
  require_numbers: true      # 需要数字
  require_symbols: true      # 需要特殊字符
  max_age_days: 90          # 密码最大使用期限
  history_size: 5           # 不能重复使用的历史密码数量
```

#### 账户安全策略
```yaml
account_security:
  max_login_attempts: 5      # 最大登录尝试次数
  lockout_duration: "30m"    # 账户锁定时间
  session_timeout: "2h"      # 会话超时时间
  force_password_change: true # 首次登录强制修改密码
```

### 角色权限管理

#### 内置角色定义
```yaml
roles:
  admin:
    description: "系统管理员"
    permissions:
      - "system:*"           # 系统全部权限
      - "user:*"             # 用户管理权限
      - "container:*"        # 容器全部权限
      - "monitoring:*"       # 监控全部权限

  operator:
    description: "运维人员"
    permissions:
      - "container:read"     # 容器查看权限
      - "container:update"   # 容器更新权限
      - "container:restart"  # 容器重启权限
      - "monitoring:read"    # 监控查看权限

  viewer:
    description: "只读用户"
    permissions:
      - "container:read"     # 容器查看权限
      - "monitoring:read"    # 监控查看权限
```

#### 自定义权限
```yaml
custom_permissions:
  - name: "container:deploy"
    description: "容器部署权限"
    resources: ["containers"]
    actions: ["create", "deploy"]

  - name: "system:backup"
    description: "系统备份权限"
    resources: ["system"]
    actions: ["backup", "restore"]
```

## 网络安全

### HTTPS 配置

#### SSL/TLS 证书配置
```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # SSL 协议和加密套件
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;

    # HSTS
    add_header Strict-Transport-Security "max-age=63072000" always;

    location / {
        proxy_pass http://docker-auto-backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}
```

#### Let's Encrypt 自动化
```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo crontab -e
# 添加：0 12 * * * /usr/bin/certbot renew --quiet
```

### API 安全

#### 请求频率限制
```yaml
rate_limiting:
  global:
    requests_per_minute: 1000
    burst: 100

  per_user:
    requests_per_minute: 100
    burst: 20

  per_ip:
    requests_per_minute: 60
    burst: 10

  api_endpoints:
    "/api/auth/login":
      requests_per_minute: 5
      burst: 2
```

#### API 访问控制
```yaml
api_security:
  cors:
    allowed_origins:
      - "https://your-domain.com"
      - "https://admin.your-domain.com"
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["Authorization", "Content-Type"]
    expose_headers: ["X-Total-Count"]
    credentials: true

  csrf_protection:
    enabled: true
    token_header: "X-CSRF-Token"
    cookie_name: "csrf_token"
```

## 数据安全

### 数据库安全

#### 连接加密
```yaml
database:
  host: "postgres-server"
  port: 5432
  database: "dockerauto"
  username: "app_user"
  password: "${DB_PASSWORD}"
  ssl_mode: "require"           # 强制 SSL 连接
  ssl_cert: "/certs/client.crt"
  ssl_key: "/certs/client.key"
  ssl_root_cert: "/certs/ca.crt"
```

#### 数据加密
```yaml
encryption:
  # 敏感字段加密
  field_encryption:
    key: "${ENCRYPTION_KEY}"
    algorithm: "AES-256-GCM"

  # 数据库级别加密（PostgreSQL）
  database_encryption:
    enabled: true
    key_management: "vault"  # 或 "local"
```

### 敏感数据处理

#### Docker Secrets 集成
```yaml
version: '3.8'
services:
  app:
    image: docker-auto:latest
    secrets:
      - db_password
      - jwt_secret
      - encryption_key
    environment:
      DB_PASSWORD_FILE: /run/secrets/db_password
      JWT_SECRET_FILE: /run/secrets/jwt_secret
      ENCRYPTION_KEY_FILE: /run/secrets/encryption_key

secrets:
  db_password:
    external: true
  jwt_secret:
    external: true
  encryption_key:
    external: true
```

#### 环境变量安全
```bash
# 使用外部密钥管理系统
export VAULT_ADDR="https://vault.company.com"
export VAULT_TOKEN="$(vault auth -method=aws)"

# 在启动脚本中获取敏感信息
DB_PASSWORD=$(vault kv get -field=password secret/docker-auto/db)
JWT_SECRET=$(vault kv get -field=secret secret/docker-auto/jwt)
```

## 容器安全

### Docker 安全配置

#### 容器运行安全
```yaml
container_security:
  # 非 root 用户运行
  user: "1000:1000"

  # 安全选项
  security_opt:
    - "no-new-privileges:true"
    - "seccomp:unconfined"

  # 资源限制
  mem_limit: "512m"
  cpus: "0.5"
  pids_limit: 100

  # 只读根文件系统
  read_only: true
  tmpfs:
    - "/tmp:noexec,nosuid,size=100m"
    - "/var/run:noexec,nosuid,size=100m"
```

#### 镜像安全扫描
```bash
# 使用 Trivy 扫描镜像
trivy image --severity HIGH,CRITICAL docker-auto:latest

# 集成到 CI/CD 流程
jobs:
  security_scan:
    runs-on: ubuntu-latest
    steps:
      - uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'docker-auto:latest'
          format: 'sarif'
          output: 'trivy-results.sarif'
```

### Docker Socket 安全

#### Socket 权限控制
```bash
# 创建 Docker 组
sudo groupadd docker
sudo usermod -aG docker $USER

# 设置 socket 权限
sudo chown root:docker /var/run/docker.sock
sudo chmod 660 /var/run/docker.sock
```

#### Socket 代理
```yaml
# 使用 Docker Socket Proxy
version: '3.8'
services:
  socket-proxy:
    image: tecnativa/docker-socket-proxy:latest
    environment:
      CONTAINERS: 1
      IMAGES: 1
      NETWORKS: 1
      VOLUMES: 1
      POST: 0  # 禁用创建操作
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - "2375:2375"

  docker-auto:
    image: docker-auto:latest
    environment:
      DOCKER_HOST: "tcp://socket-proxy:2375"
```

## 监控与审计

### 安全事件监控

#### 登录监控
```yaml
security_monitoring:
  login_events:
    failed_login_threshold: 3
    suspicious_location_check: true
    notification_channels: ["email", "slack"]

  privilege_escalation:
    monitor_role_changes: true
    monitor_permission_changes: true
    alert_on_admin_creation: true
```

#### 操作审计日志
```yaml
audit_logging:
  enabled: true
  log_level: "INFO"

  events:
    - "user.login"
    - "user.logout"
    - "user.create"
    - "user.delete"
    - "container.create"
    - "container.update"
    - "container.delete"
    - "system.config_change"

  storage:
    type: "file"  # 或 "syslog", "elasticsearch"
    path: "/var/log/docker-auto/audit.log"
    rotation: "daily"
    retention: "90d"
```

### 安全告警

#### 告警规则配置
```yaml
security_alerts:
  rules:
    - name: "multiple_failed_logins"
      condition: "failed_logins > 5 in 10m"
      severity: "high"
      actions: ["email", "lock_account"]

    - name: "privilege_escalation"
      condition: "role_change to admin"
      severity: "critical"
      actions: ["email", "slack", "log"]

    - name: "suspicious_api_usage"
      condition: "api_requests > 1000 in 1m from single_ip"
      severity: "medium"
      actions: ["rate_limit", "email"]
```

## 安全检查清单

### 部署前检查
- [ ] 修改所有默认密码
- [ ] 配置强 JWT 密钥
- [ ] 启用 HTTPS
- [ ] 配置防火墙规则
- [ ] 设置数据库访问控制
- [ ] 启用审计日志
- [ ] 配置备份策略

### 运行时检查
- [ ] 定期更新系统补丁
- [ ] 监控安全事件
- [ ] 检查用户权限
- [ ] 扫描镜像漏洞
- [ ] 验证备份有效性
- [ ] 审查访问日志

### 应急响应
- [ ] 制定安全事件响应计划
- [ ] 准备系统隔离流程
- [ ] 建立通知联系人清单
- [ ] 定期演练应急流程

## 合规性要求

### 数据保护法规
- **GDPR**: 用户数据处理和隐私保护
- **SOX**: 财务数据安全控制
- **HIPAA**: 医疗数据安全要求
- **PCI DSS**: 支付数据安全标准

### 安全标准
- **ISO 27001**: 信息安全管理体系
- **NIST**: 网络安全框架
- **CIS Controls**: 网络安全控制措施

---

**相关文档**: [监控告警配置](monitoring.md) | [系统架构](../developer/architecture.md)