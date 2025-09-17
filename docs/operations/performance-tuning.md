# 性能调优指南

## 概述

本指南提供 Docker Auto 系统的性能优化策略，包括应用程序、数据库、缓存和基础设施的调优建议。

## 性能监控基准

### 关键性能指标 (KPI)

#### 应用层指标
```yaml
application_kpis:
  response_time:
    api_p95: "<200ms"          # 95% API 响应时间
    api_p99: "<500ms"          # 99% API 响应时间
    page_load: "<2s"           # 页面加载时间

  throughput:
    concurrent_users: "1000+"   # 并发用户数
    requests_per_second: "500+" # 每秒请求数
    containers_managed: "10000+" # 管理容器数

  reliability:
    uptime: "99.9%"            # 系统可用性
    error_rate: "<0.1%"        # 错误率
    update_success_rate: "99%" # 更新成功率
```

#### 系统层指标
```yaml
system_kpis:
  cpu_usage: "<70%"            # CPU 使用率
  memory_usage: "<80%"         # 内存使用率
  disk_io_wait: "<10%"         # 磁盘 I/O 等待
  network_latency: "<10ms"     # 网络延迟
```

## 应用程序优化

### Go 后端优化

#### 1. 内存管理优化
```go
// internal/config/app.go
package config

import (
    "runtime"
    "runtime/debug"
    "time"
)

func init() {
    // 设置 GC 目标百分比
    debug.SetGCPercent(100)

    // 设置最大 P 数量 (通常等于 CPU 核心数)
    runtime.GOMAXPROCS(runtime.NumCPU())

    // 定期强制 GC (生产环境谨慎使用)
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                runtime.GC()
                debug.FreeOSMemory()
            }
        }
    }()
}
```

#### 2. 数据库连接池优化
```go
// internal/database/config.go
package database

import (
    "database/sql"
    "time"
)

func NewDBConnection() *sql.DB {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        panic(err)
    }

    // 连接池配置
    db.SetMaxOpenConns(100)              // 最大打开连接数
    db.SetMaxIdleConns(10)               // 最大空闲连接数
    db.SetConnMaxLifetime(time.Hour)     // 连接最大生命周期
    db.SetConnMaxIdleTime(30 * time.Minute) // 连接最大空闲时间

    return db
}
```

#### 3. HTTP 服务器优化
```go
// cmd/server/main.go
package main

import (
    "net/http"
    "time"
    "github.com/gin-gonic/gin"
)

func main() {
    // Gin 模式设置
    gin.SetMode(gin.ReleaseMode)

    router := gin.New()

    // 使用高性能中间件
    router.Use(gin.Recovery())
    router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
        SkipPaths: []string{"/health", "/metrics"},
    }))

    server := &http.Server{
        Addr:    ":8080",
        Handler: router,

        // 超时配置
        ReadTimeout:    10 * time.Second,
        WriteTimeout:   10 * time.Second,
        IdleTimeout:    120 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1MB
    }

    server.ListenAndServe()
}
```

#### 4. 并发处理优化
```go
// internal/service/container.go
package service

import (
    "context"
    "sync"
    "golang.org/x/sync/semaphore"
)

type ContainerService struct {
    // 限制并发操作数
    concurrencySem *semaphore.Weighted
    // 工作池
    workerPool chan struct{}
}

func NewContainerService() *ContainerService {
    return &ContainerService{
        concurrencySem: semaphore.NewWeighted(10), // 最多10个并发操作
        workerPool:     make(chan struct{}, 50),   // 50个工作协程
    }
}

func (s *ContainerService) BatchUpdate(ctx context.Context, containers []Container) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(containers))

    for _, container := range containers {
        // 获取信号量
        if err := s.concurrencySem.Acquire(ctx, 1); err != nil {
            return err
        }

        wg.Add(1)
        go func(c Container) {
            defer wg.Done()
            defer s.concurrencySem.Release(1)

            if err := s.UpdateContainer(ctx, c); err != nil {
                errChan <- err
            }
        }(container)
    }

    wg.Wait()
    close(errChan)

    // 收集错误
    for err := range errChan {
        if err != nil {
            return err
        }
    }

    return nil
}
```

#### 5. 缓存策略优化
```go
// internal/cache/manager.go
package cache

import (
    "context"
    "encoding/json"
    "time"
    "github.com/go-redis/redis/v8"
)

type CacheManager struct {
    client *redis.Client
}

// 多级缓存策略
func (c *CacheManager) GetWithFallback(
    ctx context.Context,
    key string,
    fallbackFunc func() (interface{}, error),
    ttl time.Duration,
) (interface{}, error) {

    // 1. 尝试从 Redis 获取
    val, err := c.client.Get(ctx, key).Result()
    if err == nil {
        var result interface{}
        json.Unmarshal([]byte(val), &result)
        return result, nil
    }

    // 2. 缓存未命中，执行回退函数
    result, err := fallbackFunc()
    if err != nil {
        return nil, err
    }

    // 3. 异步存储到缓存
    go func() {
        data, _ := json.Marshal(result)
        c.client.Set(context.Background(), key, data, ttl)
    }()

    return result, nil
}

// 缓存预热
func (c *CacheManager) WarmUpCache() error {
    // 预加载常用数据
    commonQueries := []string{
        "containers:list",
        "images:popular",
        "stats:dashboard",
    }

    for _, query := range commonQueries {
        go func(q string) {
            // 执行查询并缓存结果
            // ...
        }(query)
    }

    return nil
}
```

### 前端性能优化

#### 1. 构建优化配置
```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { splitVendorChunkPlugin } from 'vite'

export default defineConfig({
  plugins: [
    vue(),
    splitVendorChunkPlugin()
  ],

  build: {
    // 生产构建优化
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    },

    // 代码分割
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router', 'pinia'],
          ui: ['element-plus'],
          utils: ['axios', 'dayjs']
        }
      }
    },

    // 优化选项
    target: 'esnext',
    sourcemap: false,
    chunkSizeWarningLimit: 1000
  },

  // 开发服务器优化
  server: {
    hmr: {
      overlay: false
    }
  }
})
```

#### 2. 组件优化
```vue
<!-- components/ContainerList.vue -->
<template>
  <div class="container-list">
    <!-- 虚拟滚动优化大列表 -->
    <VirtualList
      :items="containers"
      :item-height="80"
      :container-height="600"
      v-slot="{ item }"
    >
      <ContainerCard :container="item" />
    </VirtualList>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import VirtualList from './VirtualList.vue'
import ContainerCard from './ContainerCard.vue'

// 防抖搜索
const searchTerm = ref('')
const debouncedSearch = useDebounceFn((term: string) => {
  // 执行搜索
  searchContainers(term)
}, 300)

watch(searchTerm, debouncedSearch)

// 计算属性优化
const filteredContainers = computed(() => {
  if (!searchTerm.value) return containers.value

  return containers.value.filter(container =>
    container.name.toLowerCase().includes(searchTerm.value.toLowerCase())
  )
})

// 异步组件加载
const ContainerCard = defineAsyncComponent(
  () => import('./ContainerCard.vue')
)
</script>
```

#### 3. 状态管理优化
```typescript
// stores/containers.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useContainerStore = defineStore('containers', () => {
  const containers = ref<Container[]>([])
  const loading = ref(false)

  // 使用 Map 优化查找性能
  const containerMap = computed(() => {
    const map = new Map<number, Container>()
    containers.value.forEach(container => {
      map.set(container.id, container)
    })
    return map
  })

  // 批量更新状态
  const updateContainerStatuses = (updates: { id: number; status: string }[]) => {
    updates.forEach(update => {
      const container = containerMap.value.get(update.id)
      if (container) {
        container.status = update.status
      }
    })
  }

  // 缓存查询结果
  const cachedQueries = new Map()

  const getContainers = async (filters?: ContainerFilters) => {
    const cacheKey = JSON.stringify(filters)

    if (cachedQueries.has(cacheKey)) {
      return cachedQueries.get(cacheKey)
    }

    loading.value = true
    try {
      const result = await api.getContainers(filters)
      cachedQueries.set(cacheKey, result)
      return result
    } finally {
      loading.value = false
    }
  }

  return {
    containers,
    loading,
    containerMap,
    updateContainerStatuses,
    getContainers
  }
})
```

#### 4. 网络优化
```typescript
// utils/request.ts
import axios from 'axios'

// 请求拦截器 - 添加缓存头
axios.interceptors.request.use(config => {
  // 对于 GET 请求添加缓存控制
  if (config.method === 'get') {
    config.headers['Cache-Control'] = 'max-age=300'
  }

  // 添加请求去重
  const requestKey = `${config.method}:${config.url}:${JSON.stringify(config.params)}`
  if (pendingRequests.has(requestKey)) {
    // 返回正在进行的请求
    return pendingRequests.get(requestKey)
  }

  return config
})

// 响应拦截器 - 缓存处理
axios.interceptors.response.use(response => {
  // 缓存 GET 请求响应
  if (response.config.method === 'get') {
    responseCache.set(
      `${response.config.url}:${JSON.stringify(response.config.params)}`,
      response.data,
      300 // 5分钟缓存
    )
  }

  return response
})

// 请求合并
class RequestBatcher {
  private batches = new Map<string, any[]>()
  private timers = new Map<string, NodeJS.Timeout>()

  batch(endpoint: string, data: any, delay = 50) {
    if (!this.batches.has(endpoint)) {
      this.batches.set(endpoint, [])
    }

    this.batches.get(endpoint)!.push(data)

    if (!this.timers.has(endpoint)) {
      const timer = setTimeout(() => {
        this.flush(endpoint)
      }, delay)
      this.timers.set(endpoint, timer)
    }
  }

  private async flush(endpoint: string) {
    const batch = this.batches.get(endpoint)
    if (!batch || batch.length === 0) return

    this.batches.delete(endpoint)
    this.timers.delete(endpoint)

    // 发送批量请求
    await axios.post(`${endpoint}/batch`, batch)
  }
}
```

## 数据库优化

### PostgreSQL 性能调优

#### 1. 配置参数优化
```sql
-- postgresql.conf 优化配置

-- 内存配置
shared_buffers = '256MB'                    -- 共享缓冲区
effective_cache_size = '1GB'                -- 有效缓存大小
work_mem = '4MB'                            -- 工作内存
maintenance_work_mem = '64MB'               -- 维护工作内存

-- 连接配置
max_connections = 200                       -- 最大连接数
superuser_reserved_connections = 3          -- 超级用户保留连接

-- 检查点配置
checkpoint_completion_target = 0.7          -- 检查点完成目标
wal_buffers = '16MB'                        -- WAL 缓冲区
checkpoint_timeout = '10min'                -- 检查点超时

-- 日志配置
logging_collector = on
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_min_duration_statement = 1000           -- 记录慢查询(1秒)
log_checkpoints = on
log_connections = on
log_disconnections = on

-- 统计配置
track_activities = on
track_counts = on
track_io_timing = on
track_functions = pl
```

#### 2. 索引优化
```sql
-- 分析查询性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT * FROM containers WHERE status = 'running';

-- 创建复合索引
CREATE INDEX CONCURRENTLY idx_containers_status_updated
ON containers (status, updated_at)
WHERE status IN ('running', 'updating');

-- 创建部分索引
CREATE INDEX CONCURRENTLY idx_containers_failed
ON containers (id, name, updated_at)
WHERE status = 'failed';

-- 创建表达式索引
CREATE INDEX CONCURRENTLY idx_containers_name_lower
ON containers (lower(name));

-- 创建 GIN 索引用于 JSONB 查询
CREATE INDEX CONCURRENTLY idx_containers_config
ON containers USING gin (configuration);

-- 定期重建索引
REINDEX INDEX CONCURRENTLY idx_containers_status_updated;
```

#### 3. 查询优化
```sql
-- 优化前的查询
SELECT c.*, u.name as user_name,
       COUNT(h.id) as update_count
FROM containers c
LEFT JOIN users u ON c.created_by = u.id
LEFT JOIN update_history h ON h.container_id = c.id
WHERE c.status = 'running'
GROUP BY c.id, u.name
ORDER BY c.updated_at DESC;

-- 优化后的查询
WITH container_stats AS (
  SELECT container_id, COUNT(*) as update_count
  FROM update_history
  GROUP BY container_id
)
SELECT c.*, u.name as user_name,
       COALESCE(cs.update_count, 0) as update_count
FROM containers c
LEFT JOIN users u ON c.created_by = u.id
LEFT JOIN container_stats cs ON cs.container_id = c.id
WHERE c.status = 'running'
ORDER BY c.updated_at DESC;

-- 使用物化视图优化复杂查询
CREATE MATERIALIZED VIEW container_summary AS
SELECT
  c.id,
  c.name,
  c.image,
  c.status,
  c.updated_at,
  u.name as user_name,
  COUNT(h.id) as update_count,
  MAX(h.completed_at) as last_update
FROM containers c
LEFT JOIN users u ON c.created_by = u.id
LEFT JOIN update_history h ON h.container_id = c.id
GROUP BY c.id, u.name;

-- 定期刷新物化视图
REFRESH MATERIALIZED VIEW CONCURRENTLY container_summary;
```

#### 4. 连接池配置
```yaml
# pgbouncer.ini
[databases]
dockerauto = host=localhost port=5432 dbname=dockerauto

[pgbouncer]
listen_addr = *
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt

# 连接池配置
pool_mode = transaction                      # 事务级连接池
max_client_conn = 1000                      # 最大客户端连接数
default_pool_size = 100                     # 默认池大小
min_pool_size = 10                          # 最小池大小
reserve_pool_size = 10                      # 保留池大小

# 超时配置
server_connect_timeout = 15
server_login_retry = 15
client_login_timeout = 60
```

### Redis 缓存优化

#### 1. 内存优化配置
```redis
# redis.conf

# 内存配置
maxmemory 1gb
maxmemory-policy allkeys-lru               # LRU 淘汰策略

# 持久化配置
save 900 1                                 # 900秒内有1次写入则保存
save 300 10                                # 300秒内有10次写入则保存
save 60 10000                              # 60秒内有10000次写入则保存

# AOF 配置
appendonly yes
appendfsync everysec
no-appendfsync-on-rewrite yes
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb

# 网络配置
tcp-keepalive 300
timeout 0

# 客户端配置
maxclients 10000
```

#### 2. 缓存策略优化
```go
// internal/cache/strategies.go
package cache

import (
    "context"
    "fmt"
    "time"
)

// 多级缓存策略
type MultiLevelCache struct {
    l1 *LocalCache  // 本地内存缓存
    l2 *RedisCache  // Redis 缓存
    l3 *DatabaseCache // 数据库缓存
}

func (m *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, error) {
    // L1: 本地缓存
    if val, ok := m.l1.Get(key); ok {
        return val, nil
    }

    // L2: Redis 缓存
    if val, err := m.l2.Get(ctx, key); err == nil {
        // 回填到 L1
        m.l1.Set(key, val, 5*time.Minute)
        return val, nil
    }

    // L3: 数据库
    val, err := m.l3.Get(ctx, key)
    if err != nil {
        return nil, err
    }

    // 回填到上级缓存
    go func() {
        m.l2.Set(context.Background(), key, val, 30*time.Minute)
        m.l1.Set(key, val, 5*time.Minute)
    }()

    return val, nil
}

// 缓存预热策略
type CacheWarmUpStrategy struct {
    cache Cache
    db    Database
}

func (s *CacheWarmUpStrategy) WarmUp() error {
    // 预加载热点数据
    hotKeys := []string{
        "containers:running",
        "containers:stats",
        "images:popular",
        "users:active",
    }

    for _, key := range hotKeys {
        go func(k string) {
            data, err := s.db.Get(k)
            if err == nil {
                s.cache.Set(k, data, 1*time.Hour)
            }
        }(key)
    }

    return nil
}

// 缓存更新策略
type CacheUpdateStrategy struct {
    cache Cache
    ttl   map[string]time.Duration
}

func (s *CacheUpdateStrategy) InvalidatePattern(pattern string) {
    // 使用 Redis SCAN 命令安全地删除匹配的键
    keys := s.cache.ScanKeys(pattern)
    for _, key := range keys {
        s.cache.Delete(key)
    }
}

// 自动刷新策略
func (s *CacheUpdateStrategy) AutoRefresh(key string, refreshFunc func() (interface{}, error)) {
    ttl := s.ttl[key]
    if ttl == 0 {
        ttl = 30 * time.Minute
    }

    ticker := time.NewTicker(ttl - 5*time.Minute) // 提前5分钟刷新
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if data, err := refreshFunc(); err == nil {
                s.cache.Set(key, data, ttl)
            }
        }
    }
}
```

## 系统级优化

### 容器运行时优化

#### 1. Docker 配置优化
```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ],
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65536,
      "Soft": 65536
    }
  },
  "live-restore": true,
  "userland-proxy": false,
  "experimental": true,
  "default-cgroupns-mode": "host"
}
```

#### 2. 系统内核参数优化
```bash
# /etc/sysctl.conf

# 网络优化
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 1200
net.ipv4.tcp_max_tw_buckets = 5000

# 内存优化
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5

# 文件系统优化
fs.file-max = 65535
fs.inotify.max_user_watches = 524288

# 应用限制
kernel.pid_max = 4194304
```

#### 3. 文件系统优化
```bash
# 使用 XFS 文件系统挂载选项
/dev/sdb1 /opt/docker-auto/data xfs defaults,noatime,nodiratime,logbufs=8,logbsize=256k 0 2

# 针对 Docker 存储的优化
/dev/sdc1 /var/lib/docker xfs defaults,noatime,nodiratime,prjquota 0 2
```

### 负载均衡优化

#### 1. Nginx 负载均衡配置
```nginx
upstream docker_auto_backend {
    # 负载均衡策略
    least_conn;

    # 后端服务器
    server app1:8080 weight=3 max_fails=3 fail_timeout=30s;
    server app2:8080 weight=3 max_fails=3 fail_timeout=30s;
    server app3:8080 weight=2 max_fails=3 fail_timeout=30s;

    # 保持连接
    keepalive 32;
}

server {
    listen 80;

    # 连接和请求优化
    client_max_body_size 100m;
    client_body_timeout 60s;
    client_header_timeout 60s;
    keepalive_timeout 65s;

    # Gzip 压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/javascript
        application/json
        application/xml+rss;

    # 缓存配置
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        add_header Vary Accept-Encoding;
    }

    # API 请求
    location /api {
        proxy_pass http://docker_auto_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时设置
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # 缓冲配置
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
        proxy_busy_buffers_size 8k;
    }

    # WebSocket 连接
    location /ws {
        proxy_pass http://docker_auto_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;

        # WebSocket 特定超时
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}

# 限流配置
http {
    # IP 限流
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=1r/s;

    server {
        location /api/auth/login {
            limit_req zone=login burst=3 nodelay;
            proxy_pass http://docker_auto_backend;
        }

        location /api {
            limit_req zone=api burst=20 nodelay;
            proxy_pass http://docker_auto_backend;
        }
    }
}
```

### 监控和诊断工具

#### 1. 性能监控脚本
```bash
#!/bin/bash
# performance-monitor.sh

LOG_FILE="/var/log/docker-auto-performance.log"
THRESHOLD_CPU=80
THRESHOLD_MEM=85

while true; do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

    # CPU 使用率
    CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)

    # 内存使用率
    MEM_USAGE=$(free | grep Mem | awk '{printf "%.2f", $3/$2 * 100.0}')

    # 磁盘 I/O
    IO_WAIT=$(iostat -x 1 1 | tail -n +4 | awk '{sum+=$10} END {print sum/NR}')

    # 网络连接数
    CONNECTIONS=$(netstat -an | grep ESTABLISHED | wc -l)

    # Docker 容器数量
    CONTAINERS_RUNNING=$(docker ps -q | wc -l)

    # 记录到日志
    echo "$TIMESTAMP,CPU:$CPU_USAGE,MEM:$MEM_USAGE,IO:$IO_WAIT,CONN:$CONNECTIONS,CONTAINERS:$CONTAINERS_RUNNING" >> $LOG_FILE

    # 检查阈值并告警
    if (( $(echo "$CPU_USAGE > $THRESHOLD_CPU" | bc -l) )); then
        echo "WARNING: High CPU usage: $CPU_USAGE%" | mail -s "Performance Alert" admin@company.com
    fi

    if (( $(echo "$MEM_USAGE > $THRESHOLD_MEM" | bc -l) )); then
        echo "WARNING: High memory usage: $MEM_USAGE%" | mail -s "Performance Alert" admin@company.com
    fi

    sleep 60
done
```

#### 2. 数据库性能诊断
```sql
-- 查找慢查询
SELECT query, mean_time, calls, total_time
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;

-- 查找未使用的索引
SELECT schemaname, tablename, indexname, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE idx_tup_read = 0;

-- 查找表膨胀
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
       pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
       pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as index_size
FROM pg_tables
WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 连接池状态
SELECT state, count(*)
FROM pg_stat_activity
GROUP BY state;
```

## 性能测试

### 负载测试场景

#### 1. 基准性能测试
```javascript
// k6-benchmark.js
import http from 'k6/http'
import { check, group } from 'k6'

export let options = {
  scenarios: {
    // 基准测试
    benchmark: {
      executor: 'constant-vus',
      vus: 50,
      duration: '10m',
    },

    // 峰值测试
    peak: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5m', target: 100 },
        { duration: '10m', target: 200 },
        { duration: '5m', target: 0 },
      ],
    },

    // 压力测试
    stress: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '30m',
      preAllocatedVUs: 100,
      maxVUs: 500,
    }
  },

  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
}

export default function() {
  group('Container Operations', () => {
    // 获取容器列表
    let response = http.get('http://localhost/api/containers')
    check(response, {
      'list containers status is 200': (r) => r.status === 200,
      'list containers response time < 200ms': (r) => r.timings.duration < 200,
    })

    // 创建容器
    response = http.post('http://localhost/api/containers', {
      name: `test-container-${__VU}-${__ITER}`,
      image: 'nginx',
      tag: 'latest'
    })
    check(response, {
      'create container status is 201': (r) => r.status === 201,
    })

    if (response.status === 201) {
      const container = JSON.parse(response.body)

      // 启动容器
      response = http.post(`http://localhost/api/containers/${container.id}/start`)
      check(response, {
        'start container status is 200': (r) => r.status === 200,
      })
    }
  })
}
```

#### 2. 自动化性能回归测试
```bash
#!/bin/bash
# performance-regression-test.sh

BASELINE_FILE="performance-baseline.json"
CURRENT_RESULTS="current-performance.json"

# 运行性能测试
k6 run --out json=$CURRENT_RESULTS k6-benchmark.js

# 比较结果
python3 << EOF
import json

# 加载基准和当前结果
with open('$BASELINE_FILE') as f:
    baseline = json.load(f)

with open('$CURRENT_RESULTS') as f:
    current = json.load(f)

# 比较关键指标
metrics = ['http_req_duration', 'http_req_failed']
regression_threshold = 0.1  # 10% 性能回归阈值

for metric in metrics:
    baseline_value = baseline['metrics'][metric]['avg']
    current_value = current['metrics'][metric]['avg']

    if metric == 'http_req_duration':
        # 响应时间变化
        change = (current_value - baseline_value) / baseline_value
        if change > regression_threshold:
            print(f"REGRESSION: {metric} increased by {change:.2%}")
            exit(1)
    elif metric == 'http_req_failed':
        # 错误率变化
        if current_value > baseline_value * (1 + regression_threshold):
            print(f"REGRESSION: {metric} increased from {baseline_value} to {current_value}")
            exit(1)

print("Performance regression test passed")
EOF
```

## 性能优化检查清单

### 应用层优化
- [ ] 启用 Gzip 压缩
- [ ] 实施缓存策略
- [ ] 优化数据库查询
- [ ] 使用连接池
- [ ] 实现请求去重
- [ ] 添加 CDN 支持

### 数据库优化
- [ ] 创建适当的索引
- [ ] 优化查询语句
- [ ] 配置连接池
- [ ] 启用查询缓存
- [ ] 设置合理的内存参数

### 系统优化
- [ ] 调整内核参数
- [ ] 优化文件系统
- [ ] 配置负载均衡
- [ ] 实施限流策略
- [ ] 监控系统指标

### 部署优化
- [ ] 使用容器编排
- [ ] 实现自动伸缩
- [ ] 配置健康检查
- [ ] 设置资源限制
- [ ] 优化镜像大小

---

**相关文档**: [监控配置](../admin/monitoring.md) | [系统架构](../developer/architecture.md)