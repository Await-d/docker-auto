# 系统扩展指南

## 概述

本指南涵盖 Docker Auto 系统的扩展策略，包括水平扩展、垂直扩展和云原生扩展方案。

## 扩展策略概览

### 扩展类型

#### 垂直扩展 (Scale Up)
- **适用场景**: 小到中型部署，资源需求增长可预测
- **优势**: 简单直接，无需架构变更
- **劣势**: 有硬件上限，单点故障风险

#### 水平扩展 (Scale Out)
- **适用场景**: 大型部署，高可用性要求
- **优势**: 无理论上限，高可用性，成本灵活
- **劣势**: 架构复杂，数据一致性挑战

#### 混合扩展
- **适用场景**: 企业级部署，复杂工作负载
- **优势**: 灵活性最高，可针对性优化
- **劣势**: 管理复杂度最高

## 垂直扩展

### 硬件资源评估

#### 当前资源使用分析
```bash
#!/bin/bash
# resource-analysis.sh

echo "=== CPU 使用情况 ==="
echo "CPU 核心数: $(nproc)"
echo "CPU 使用率: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)%"

echo -e "\n=== 内存使用情况 ==="
free -h

echo -e "\n=== 磁盘使用情况 ==="
df -h

echo -e "\n=== 网络连接数 ==="
echo "TCP 连接数: $(netstat -an | grep tcp | wc -l)"
echo "ESTABLISHED 连接数: $(netstat -an | grep ESTABLISHED | wc -l)"

echo -e "\n=== Docker 资源使用 ==="
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"

echo -e "\n=== 数据库连接数 ==="
docker exec docker-auto-postgres psql -U dockerauto -d dockerauto -c "SELECT count(*) as connections FROM pg_stat_activity;"
```

#### 资源需求预测
```python
#!/usr/bin/env python3
# capacity-planning.py

import pandas as pd
import numpy as np
from sklearn.linear_model import LinearRegression
import matplotlib.pyplot as plt

def predict_resource_needs(usage_data, prediction_days=30):
    """
    基于历史数据预测资源需求
    """
    # 准备数据
    df = pd.DataFrame(usage_data)
    df['timestamp'] = pd.to_datetime(df['timestamp'])
    df['days'] = (df['timestamp'] - df['timestamp'].min()).dt.days

    # CPU 预测
    X = df[['days']].values
    y_cpu = df['cpu_usage'].values

    model_cpu = LinearRegression()
    model_cpu.fit(X, y_cpu)

    # 预测未来需求
    future_days = np.array([[df['days'].max() + i] for i in range(1, prediction_days + 1)])
    predicted_cpu = model_cpu.predict(future_days)

    # 内存预测
    y_memory = df['memory_usage'].values
    model_memory = LinearRegression()
    model_memory.fit(X, y_memory)
    predicted_memory = model_memory.predict(future_days)

    return {
        'cpu': {
            'current': y_cpu[-1],
            'predicted_max': predicted_cpu.max(),
            'growth_rate': model_cpu.coef_[0]
        },
        'memory': {
            'current': y_memory[-1],
            'predicted_max': predicted_memory.max(),
            'growth_rate': model_memory.coef_[0]
        }
    }

# 示例使用
if __name__ == "__main__":
    # 模拟历史数据
    usage_data = [
        {'timestamp': '2024-01-01', 'cpu_usage': 45.2, 'memory_usage': 62.1},
        {'timestamp': '2024-01-15', 'cpu_usage': 48.7, 'memory_usage': 65.8},
        {'timestamp': '2024-02-01', 'cpu_usage': 52.3, 'memory_usage': 69.2},
        # ... 更多数据点
    ]

    predictions = predict_resource_needs(usage_data)
    print(f"CPU 当前使用率: {predictions['cpu']['current']:.1f}%")
    print(f"CPU 预测峰值: {predictions['cpu']['predicted_max']:.1f}%")
    print(f"建议 CPU 升级: {predictions['cpu']['predicted_max'] > 80}")
```

### 升级路径规划

#### 分阶段升级策略
```yaml
# upgrade-stages.yml
stages:
  stage_1:
    name: "CPU 升级"
    trigger: "cpu_usage > 70%"
    action:
      - upgrade_cpu: "4 cores -> 8 cores"
      - expected_improvement: "50% more processing capacity"
      - downtime: "5 minutes"

  stage_2:
    name: "内存升级"
    trigger: "memory_usage > 80%"
    action:
      - upgrade_memory: "8GB -> 16GB"
      - expected_improvement: "Better caching, reduced disk I/O"
      - downtime: "10 minutes"

  stage_3:
    name: "存储升级"
    trigger: "disk_usage > 85% OR iops > 1000"
    action:
      - upgrade_storage: "HDD -> NVMe SSD"
      - upgrade_capacity: "100GB -> 500GB"
      - expected_improvement: "10x faster I/O performance"
      - downtime: "30 minutes"

  stage_4:
    name: "网络升级"
    trigger: "network_utilization > 60%"
    action:
      - upgrade_bandwidth: "100Mbps -> 1Gbps"
      - expected_improvement: "Better concurrent user support"
      - downtime: "15 minutes"
```

## 水平扩展

### 应用层扩展

#### 无状态应用设计
```go
// internal/config/stateless.go
package config

import (
    "os"
    "context"
    "github.com/go-redis/redis/v8"
)

// 外部化会话存储
type SessionStore struct {
    client *redis.Client
}

func NewSessionStore() *SessionStore {
    client := redis.NewClient(&redis.Options{
        Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       0,
    })

    return &SessionStore{client: client}
}

func (s *SessionStore) Set(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error {
    return s.client.Set(ctx, sessionID, data, ttl).Err()
}

// 外部化配置
type ConfigManager struct {
    source string // "env", "consul", "etcd"
}

func (c *ConfigManager) Get(key string) string {
    switch c.source {
    case "consul":
        // 从 Consul 获取配置
        return c.getFromConsul(key)
    case "etcd":
        // 从 etcd 获取配置
        return c.getFromEtcd(key)
    default:
        // 从环境变量获取
        return os.Getenv(key)
    }
}

// 健康检查端点
func (app *App) HealthCheck(c *gin.Context) {
    checks := map[string]string{
        "database": "ok",
        "redis":    "ok",
        "docker":   "ok",
    }

    // 数据库检查
    if err := app.db.Ping(); err != nil {
        checks["database"] = "error"
    }

    // Redis 检查
    if err := app.redis.Ping(c.Request.Context()).Err(); err != nil {
        checks["redis"] = "error"
    }

    // Docker 检查
    if _, err := app.dockerClient.Ping(c.Request.Context()); err != nil {
        checks["docker"] = "error"
    }

    status := 200
    for _, check := range checks {
        if check != "ok" {
            status = 503
            break
        }
    }

    c.JSON(status, gin.H{
        "status": "healthy",
        "checks": checks,
        "timestamp": time.Now(),
        "instance": os.Getenv("HOSTNAME"),
    })
}
```

#### 负载均衡配置
```nginx
# nginx-lb.conf
upstream docker_auto_cluster {
    # 负载均衡算法
    least_conn;

    # 应用实例
    server docker-auto-app-1:8080 weight=3 max_fails=3 fail_timeout=30s;
    server docker-auto-app-2:8080 weight=3 max_fails=3 fail_timeout=30s;
    server docker-auto-app-3:8080 weight=3 max_fails=3 fail_timeout=30s;

    # 备用实例
    server docker-auto-app-backup:8080 backup;

    # 连接保持
    keepalive 32;
    keepalive_requests 100;
    keepalive_timeout 60s;
}

# 健康检查配置
server {
    listen 8888;
    location /health {
        access_log off;
        return 200 "healthy\n";
    }
}

# 主服务配置
server {
    listen 80;
    listen 443 ssl http2;

    # 负载均衡
    location / {
        proxy_pass http://docker_auto_cluster;

        # 健康检查
        proxy_next_upstream error timeout invalid_header http_500 http_502 http_503;
        proxy_next_upstream_timeout 3s;
        proxy_next_upstream_tries 3;

        # 连接设置
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 会话亲和性（如果需要）
        # ip_hash;
    }

    # WebSocket 负载均衡
    location /ws {
        proxy_pass http://docker_auto_cluster;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;

        # WebSocket 会话保持
        ip_hash;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        proxy_pass http://docker_auto_cluster;
        proxy_cache static_cache;
        proxy_cache_valid 200 1d;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### 数据库扩展

#### 读写分离配置
```yaml
# docker-compose.db-cluster.yml
version: '3.8'

services:
  postgres-primary:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=dockerauto
      - POSTGRES_USER=dockerauto
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_REPLICATION_MODE=master
      - POSTGRES_REPLICATION_USER=replicator
      - POSTGRES_REPLICATION_PASSWORD=${REPLICATION_PASSWORD}
    volumes:
      - postgres-primary-data:/var/lib/postgresql/data
      - ./config/postgres-primary.conf:/etc/postgresql/postgresql.conf
    ports:
      - "5432:5432"
    command: postgres -c config_file=/etc/postgresql/postgresql.conf

  postgres-replica-1:
    image: postgres:15-alpine
    environment:
      - POSTGRES_REPLICATION_MODE=slave
      - POSTGRES_MASTER_HOST=postgres-primary
      - POSTGRES_MASTER_PORT=5432
      - POSTGRES_REPLICATION_USER=replicator
      - POSTGRES_REPLICATION_PASSWORD=${REPLICATION_PASSWORD}
    volumes:
      - postgres-replica-1-data:/var/lib/postgresql/data
    ports:
      - "5433:5432"
    depends_on:
      - postgres-primary

  postgres-replica-2:
    image: postgres:15-alpine
    environment:
      - POSTGRES_REPLICATION_MODE=slave
      - POSTGRES_MASTER_HOST=postgres-primary
      - POSTGRES_MASTER_PORT=5432
      - POSTGRES_REPLICATION_USER=replicator
      - POSTGRES_REPLICATION_PASSWORD=${REPLICATION_PASSWORD}
    volumes:
      - postgres-replica-2-data:/var/lib/postgresql/data
    ports:
      - "5434:5432"
    depends_on:
      - postgres-primary

  pgpool:
    image: pgpool/pgpool:latest
    environment:
      - PGPOOL_BACKEND_HOSTNAME0=postgres-primary
      - PGPOOL_BACKEND_PORT0=5432
      - PGPOOL_BACKEND_HOSTNAME1=postgres-replica-1
      - PGPOOL_BACKEND_PORT1=5432
      - PGPOOL_BACKEND_HOSTNAME2=postgres-replica-2
      - PGPOOL_BACKEND_PORT2=5432
      - PGPOOL_POSTGRES_USERNAME=dockerauto
      - PGPOOL_POSTGRES_PASSWORD=${DB_PASSWORD}
    ports:
      - "5430:5432"
    depends_on:
      - postgres-primary
      - postgres-replica-1
      - postgres-replica-2

volumes:
  postgres-primary-data:
  postgres-replica-1-data:
  postgres-replica-2-data:
```

#### 数据库连接路由
```go
// internal/database/router.go
package database

import (
    "database/sql"
    "context"
    "sync"
)

type DatabaseRouter struct {
    master   *sql.DB
    replicas []*sql.DB
    current  int
    mu       sync.RWMutex
}

func NewDatabaseRouter(masterDSN string, replicaDSNs []string) *DatabaseRouter {
    master, err := sql.Open("postgres", masterDSN)
    if err != nil {
        panic(err)
    }

    var replicas []*sql.DB
    for _, dsn := range replicaDSNs {
        replica, err := sql.Open("postgres", dsn)
        if err != nil {
            panic(err)
        }
        replicas = append(replicas, replica)
    }

    return &DatabaseRouter{
        master:   master,
        replicas: replicas,
    }
}

// 写操作使用主库
func (r *DatabaseRouter) Master() *sql.DB {
    return r.master
}

// 读操作使用从库（轮询）
func (r *DatabaseRouter) Replica() *sql.DB {
    r.mu.Lock()
    defer r.mu.Unlock()

    if len(r.replicas) == 0 {
        return r.master
    }

    replica := r.replicas[r.current]
    r.current = (r.current + 1) % len(r.replicas)
    return replica
}

// 检查从库健康状态
func (r *DatabaseRouter) HealthCheck() map[string]bool {
    status := make(map[string]bool)

    // 检查主库
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    status["master"] = r.master.PingContext(ctx) == nil

    // 检查从库
    for i, replica := range r.replicas {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        status[fmt.Sprintf("replica-%d", i)] = replica.PingContext(ctx) == nil
        cancel()
    }

    return status
}

// 故障转移
func (r *DatabaseRouter) RemoveUnhealthyReplicas() {
    r.mu.Lock()
    defer r.mu.Unlock()

    var healthyReplicas []*sql.DB
    for _, replica := range r.replicas {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        if replica.PingContext(ctx) == nil {
            healthyReplicas = append(healthyReplicas, replica)
        }
        cancel()
    }

    r.replicas = healthyReplicas
    r.current = 0
}
```

#### 缓存集群配置
```yaml
# redis-cluster.yml
version: '3.8'

services:
  redis-node-1:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes --port 7001
    ports:
      - "7001:7001"
    volumes:
      - redis-node-1-data:/data

  redis-node-2:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes --port 7002
    ports:
      - "7002:7002"
    volumes:
      - redis-node-2-data:/data

  redis-node-3:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf --cluster-node-timeout 5000 --appendonly yes --port 7003
    ports:
      - "7003:7003"
    volumes:
      - redis-node-3-data:/data

  redis-cluster-creator:
    image: redis:7-alpine
    command: redis-cli --cluster create redis-node-1:7001 redis-node-2:7002 redis-node-3:7003 --cluster-replicas 0 --cluster-yes
    depends_on:
      - redis-node-1
      - redis-node-2
      - redis-node-3

volumes:
  redis-node-1-data:
  redis-node-2-data:
  redis-node-3-data:
```

## Kubernetes 自动扩展

### 水平 Pod 自动扩展 (HPA)
```yaml
# hpa.yml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: docker-auto-hpa
  namespace: docker-auto
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: docker-auto-app
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "1k"

  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60

---
# 自定义指标
apiVersion: v1
kind: Service
metadata:
  name: docker-auto-metrics
  namespace: docker-auto
  labels:
    app: docker-auto-app
spec:
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
  selector:
    app: docker-auto-app

---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: docker-auto-monitor
  namespace: docker-auto
spec:
  selector:
    matchLabels:
      app: docker-auto-app
  endpoints:
  - port: metrics
    path: /metrics
```

### 垂直 Pod 自动扩展 (VPA)
```yaml
# vpa.yml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: docker-auto-vpa
  namespace: docker-auto
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: docker-auto-app

  updatePolicy:
    updateMode: "Auto"  # "Off", "Initial", "Recreation", "Auto"

  resourcePolicy:
    containerPolicies:
    - containerName: docker-auto-app
      minAllowed:
        cpu: 100m
        memory: 128Mi
      maxAllowed:
        cpu: 2
        memory: 4Gi
      controlledResources: ["cpu", "memory"]
      controlledValues: RequestsAndLimits
```

### 集群自动扩展 (CA)
```yaml
# cluster-autoscaler.yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cluster-autoscaler
  template:
    metadata:
      labels:
        app: cluster-autoscaler
    spec:
      serviceAccountName: cluster-autoscaler
      containers:
      - image: k8s.gcr.io/autoscaling/cluster-autoscaler:v1.21.0
        name: cluster-autoscaler
        resources:
          limits:
            cpu: 100m
            memory: 300Mi
          requests:
            cpu: 100m
            memory: 300Mi
        command:
        - ./cluster-autoscaler
        - --v=4
        - --stderrthreshold=info
        - --cloud-provider=aws
        - --skip-nodes-with-local-storage=false
        - --expander=least-waste
        - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/docker-auto-cluster
        - --balance-similar-node-groups
        - --scale-down-enabled=true
        - --scale-down-delay-after-add=10m
        - --scale-down-unneeded-time=10m
        - --max-node-provision-time=15m
        env:
        - name: AWS_REGION
          value: us-west-2
```

## 云原生扩展

### AWS EKS 扩展配置
```yaml
# eks-nodegroup.yml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: docker-auto-cluster
  region: us-west-2
  version: "1.21"

nodeGroups:
  - name: docker-auto-workers
    instanceType: m5.large
    minSize: 3
    maxSize: 20
    desiredCapacity: 5

    # 自动扩展配置
    iam:
      attachPolicyARNs:
        - arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy
        - arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy
        - arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly
        - arn:aws:iam::aws:policy/AutoScalingFullAccess

    # Spot 实例支持
    spot: true
    instancesDistribution:
      maxPrice: 0.10
      instanceTypes: ["m5.large", "m5.xlarge", "m4.large"]
      onDemandBaseCapacity: 2
      onDemandPercentageAboveBaseCapacity: 20

    # 自定义启动模板
    launchTemplate:
      id: lt-123456789abcdef0
      version: "1"

    # 标签
    tags:
      Environment: production
      Application: docker-auto

  - name: docker-auto-spot
    instanceTypes: ["m5.large", "m5.xlarge", "c5.large", "c5.xlarge"]
    spot: true
    minSize: 0
    maxSize: 50
    desiredCapacity: 0

    taints:
      - key: spot-instance
        value: "true"
        effect: NoSchedule

addons:
  - name: aws-load-balancer-controller
  - name: cluster-autoscaler
  - name: metrics-server
```

### Google GKE 扩展配置
```yaml
# gke-cluster.yml
apiVersion: container.v1
kind: Cluster
metadata:
  name: docker-auto-cluster
spec:
  location: us-central1-a

  # 节点池
  nodePools:
  - name: default-pool
    initialNodeCount: 3
    autoscaling:
      enabled: true
      minNodeCount: 3
      maxNodeCount: 20
    config:
      machineType: n1-standard-2
      diskType: pd-ssd
      diskSizeGb: 100
      preemptible: false

  - name: spot-pool
    initialNodeCount: 0
    autoscaling:
      enabled: true
      minNodeCount: 0
      maxNodeCount: 50
    config:
      machineType: n1-standard-2
      preemptible: true
      taints:
      - key: spot-instance
        value: "true"
        effect: NO_SCHEDULE

  # 集群配置
  clusterAutoscaling:
    enabled: true
    resourceLimits:
    - resourceType: cpu
      maximum: 1000
    - resourceType: memory
      maximum: 1000

  addonsConfig:
    horizontalPodAutoscaling:
      disabled: false
    httpLoadBalancing:
      disabled: false
    networkPolicyConfig:
      disabled: false
```

### 多云扩展策略
```yaml
# multi-cloud-deployment.yml
regions:
  primary:
    provider: aws
    region: us-west-2
    min_instances: 5
    max_instances: 20

  secondary:
    provider: gcp
    region: us-central1
    min_instances: 2
    max_instances: 10

  tertiary:
    provider: azure
    region: eastus
    min_instances: 0
    max_instances: 5

scaling_policies:
  - name: primary_region_scale
    trigger: "avg_response_time > 500ms"
    action: scale_up
    target: primary
    scale_factor: 2

  - name: secondary_region_activation
    trigger: "primary_region_utilization > 80%"
    action: scale_up
    target: secondary
    scale_factor: 1

  - name: global_failover
    trigger: "primary_region_availability < 95%"
    action: failover
    target: secondary

traffic_distribution:
  - region: primary
    weight: 70
  - region: secondary
    weight: 25
  - region: tertiary
    weight: 5
```

## 扩展监控和告警

### 扩展指标监控
```yaml
# scaling-metrics.yml
metrics:
  application:
    - name: response_time_p95
      query: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
      threshold: 500ms

    - name: requests_per_second
      query: rate(http_requests_total[5m])
      threshold: 1000

    - name: error_rate
      query: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
      threshold: 0.01

  infrastructure:
    - name: cpu_utilization
      query: 100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
      threshold: 80

    - name: memory_utilization
      query: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100
      threshold: 85

    - name: pod_count
      query: count(kube_pod_info)
      threshold: 100

  database:
    - name: connection_count
      query: pg_stat_database_numbackends
      threshold: 80

    - name: query_duration
      query: pg_stat_statements_mean_time
      threshold: 1000

    - name: replication_lag
      query: pg_replication_lag_seconds
      threshold: 10

alerts:
  - name: high_response_time
    condition: response_time_p95 > 500ms for 5m
    action: scale_up
    severity: warning

  - name: high_error_rate
    condition: error_rate > 0.01 for 3m
    action: [scale_up, notify_oncall]
    severity: critical

  - name: resource_exhaustion
    condition: cpu_utilization > 90% AND memory_utilization > 90% for 2m
    action: [scale_up_urgent, notify_oncall]
    severity: critical
```

### 自动化扩展脚本
```bash
#!/bin/bash
# auto-scaling.sh

NAMESPACE="docker-auto"
DEPLOYMENT="docker-auto-app"
MIN_REPLICAS=3
MAX_REPLICAS=20

# 获取当前指标
CPU_USAGE=$(kubectl top pods -n $NAMESPACE --no-headers | awk '{sum+=$2} END {print sum/NR}' | sed 's/m//')
MEMORY_USAGE=$(kubectl top pods -n $NAMESPACE --no-headers | awk '{sum+=$3} END {print sum/NR}' | sed 's/Mi//')
CURRENT_REPLICAS=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE -o jsonpath='{.spec.replicas}')

# 计算目标副本数
TARGET_REPLICAS=$CURRENT_REPLICAS

# CPU 基于扩展
if [ $CPU_USAGE -gt 700 ]; then
    TARGET_REPLICAS=$((CURRENT_REPLICAS * 2))
elif [ $CPU_USAGE -gt 500 ]; then
    TARGET_REPLICAS=$((CURRENT_REPLICAS + 1))
elif [ $CPU_USAGE -lt 200 ] && [ $CURRENT_REPLICAS -gt $MIN_REPLICAS ]; then
    TARGET_REPLICAS=$((CURRENT_REPLICAS - 1))
fi

# 内存基于扩展
if [ $MEMORY_USAGE -gt 800 ]; then
    TARGET_REPLICAS=$((TARGET_REPLICAS + 1))
fi

# 应用限制
if [ $TARGET_REPLICAS -lt $MIN_REPLICAS ]; then
    TARGET_REPLICAS=$MIN_REPLICAS
elif [ $TARGET_REPLICAS -gt $MAX_REPLICAS ]; then
    TARGET_REPLICAS=$MAX_REPLICAS
fi

# 执行扩展
if [ $TARGET_REPLICAS -ne $CURRENT_REPLICAS ]; then
    echo "Scaling from $CURRENT_REPLICAS to $TARGET_REPLICAS replicas"
    kubectl scale deployment $DEPLOYMENT -n $NAMESPACE --replicas=$TARGET_REPLICAS

    # 记录扩展事件
    echo "$(date): Scaled $DEPLOYMENT from $CURRENT_REPLICAS to $TARGET_REPLICAS (CPU: ${CPU_USAGE}m, Memory: ${MEMORY_USAGE}Mi)" >> /var/log/scaling.log

    # 发送通知
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"Docker Auto scaled from $CURRENT_REPLICAS to $TARGET_REPLICAS replicas\"}" \
        $SLACK_WEBHOOK_URL
fi
```

## 扩展测试验证

### 负载测试脚本
```python
#!/usr/bin/env python3
# scaling-test.py

import asyncio
import aiohttp
import time
import json
from dataclasses import dataclass
from typing import List

@dataclass
class TestResult:
    timestamp: float
    response_time: float
    status_code: int
    success: bool

class ScalingTest:
    def __init__(self, base_url: str, max_concurrent: int = 1000):
        self.base_url = base_url
        self.max_concurrent = max_concurrent
        self.results: List[TestResult] = []

    async def make_request(self, session: aiohttp.ClientSession, url: str) -> TestResult:
        start_time = time.time()
        try:
            async with session.get(url) as response:
                await response.text()
                response_time = time.time() - start_time
                return TestResult(
                    timestamp=start_time,
                    response_time=response_time,
                    status_code=response.status,
                    success=response.status == 200
                )
        except Exception as e:
            response_time = time.time() - start_time
            return TestResult(
                timestamp=start_time,
                response_time=response_time,
                status_code=0,
                success=False
            )

    async def run_load_test(self, duration_minutes: int = 10):
        """
        运行负载测试，逐步增加并发数
        """
        async with aiohttp.ClientSession() as session:
            end_time = time.time() + (duration_minutes * 60)
            concurrent_users = 10

            while time.time() < end_time:
                print(f"Testing with {concurrent_users} concurrent users...")

                # 创建并发请求
                tasks = [
                    self.make_request(session, f"{self.base_url}/api/containers")
                    for _ in range(concurrent_users)
                ]

                # 执行请求
                batch_results = await asyncio.gather(*tasks, return_exceptions=True)

                # 收集结果
                valid_results = [r for r in batch_results if isinstance(r, TestResult)]
                self.results.extend(valid_results)

                # 计算指标
                success_rate = sum(1 for r in valid_results if r.success) / len(valid_results)
                avg_response_time = sum(r.response_time for r in valid_results) / len(valid_results)

                print(f"Success rate: {success_rate:.2%}, Avg response time: {avg_response_time:.3f}s")

                # 调整并发数
                if success_rate > 0.95 and avg_response_time < 0.5:
                    concurrent_users = min(concurrent_users + 50, self.max_concurrent)
                elif success_rate < 0.9 or avg_response_time > 1.0:
                    concurrent_users = max(concurrent_users - 10, 10)

                await asyncio.sleep(30)  # 30秒间隔

    def generate_report(self) -> dict:
        """
        生成测试报告
        """
        if not self.results:
            return {}

        success_count = sum(1 for r in self.results if r.success)
        total_requests = len(self.results)

        response_times = [r.response_time for r in self.results if r.success]
        response_times.sort()

        return {
            "total_requests": total_requests,
            "successful_requests": success_count,
            "success_rate": success_count / total_requests,
            "response_time": {
                "min": min(response_times) if response_times else 0,
                "max": max(response_times) if response_times else 0,
                "avg": sum(response_times) / len(response_times) if response_times else 0,
                "p50": response_times[len(response_times)//2] if response_times else 0,
                "p95": response_times[int(len(response_times)*0.95)] if response_times else 0,
                "p99": response_times[int(len(response_times)*0.99)] if response_times else 0,
            }
        }

async def main():
    test = ScalingTest("http://your-docker-auto-domain.com")
    await test.run_load_test(duration_minutes=15)

    report = test.generate_report()
    print("\n=== Scaling Test Report ===")
    print(json.dumps(report, indent=2))

if __name__ == "__main__":
    asyncio.run(main())
```

## 扩展最佳实践

### 扩展检查清单
- [ ] 应用无状态化设计
- [ ] 数据库读写分离
- [ ] 缓存集群部署
- [ ] 负载均衡配置
- [ ] 健康检查实现
- [ ] 监控指标设置
- [ ] 自动扩展策略
- [ ] 成本优化配置

### 扩展注意事项
1. **数据一致性**: 确保分布式环境下的数据一致性
2. **会话管理**: 外部化会话存储，避免会话亲和性
3. **配置管理**: 集中化配置管理，支持动态更新
4. **监控告警**: 完善的监控体系，及时发现问题
5. **成本控制**: 合理的扩展策略，避免资源浪费

---

**相关文档**: [部署指南](deployment.md) | [性能调优](performance-tuning.md) | [监控配置](../admin/monitoring.md)