# 应急响应手册

## 概述

本手册提供 Docker Auto 系统发生故障时的应急响应流程和恢复步骤，确保快速诊断问题并恢复服务。

## 应急响应流程

### 事件分级

#### P0 - 严重事故
- **定义**: 系统完全不可用，影响所有用户
- **影响**: 生产环境完全中断
- **响应时间**: 立即响应（15分钟内）
- **解决目标**: 4小时内恢复

#### P1 - 高影响事故
- **定义**: 核心功能不可用，影响大部分用户
- **影响**: 主要功能中断或严重性能问题
- **响应时间**: 1小时内响应
- **解决目标**: 24小时内解决

#### P2 - 中等影响事故
- **定义**: 部分功能异常，影响部分用户
- **影响**: 非核心功能问题或轻微性能影响
- **响应时间**: 4小时内响应
- **解决目标**: 72小时内解决

#### P3 - 低影响事故
- **定义**: 轻微问题或潜在风险
- **影响**: 最小用户影响
- **响应时间**: 1个工作日内响应
- **解决目标**: 1周内解决

### 应急响应团队

#### 角色定义
```yaml
incident_response_team:
  incident_commander:
    role: "事故指挥官"
    responsibilities:
      - 协调整体响应工作
      - 决策权威
      - 对外沟通
    contacts:
      - name: "John Doe"
        phone: "+1-555-0101"
        email: "john.doe@company.com"

  technical_lead:
    role: "技术负责人"
    responsibilities:
      - 技术诊断和修复
      - 系统恢复操作
      - 技术决策支持
    contacts:
      - name: "Jane Smith"
        phone: "+1-555-0102"
        email: "jane.smith@company.com"

  communication_lead:
    role: "沟通协调员"
    responsibilities:
      - 用户沟通
      - 状态页面更新
      - 媒体应对
    contacts:
      - name: "Mike Johnson"
        phone: "+1-555-0103"
        email: "mike.johnson@company.com"

  sre_team:
    role: "SRE 团队"
    responsibilities:
      - 系统监控
      - 基础设施修复
      - 性能优化
    contacts:
      - name: "SRE On-call"
        phone: "+1-555-0999"
        email: "sre-oncall@company.com"
```

## 常见故障排查

### 系统无法访问

#### 诊断步骤
```bash
#!/bin/bash
# system-diagnosis.sh

echo "=== Docker Auto 系统诊断 ==="
echo "时间: $(date)"
echo

# 1. 检查 Docker 容器状态
echo "1. 容器状态检查:"
docker ps -a --filter "name=docker-auto" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo

# 2. 检查服务端口
echo "2. 端口状态检查:"
netstat -tlnp | grep -E ":80|:443|:8080|:5432|:6379"
echo

# 3. 检查服务健康状态
echo "3. 服务健康检查:"
curl -s -o /dev/null -w "HTTP Status: %{http_code}, Response Time: %{time_total}s\n" http://localhost/health
echo

# 4. 检查日志错误
echo "4. 最近日志错误:"
docker logs docker-auto-app --tail=20 2>&1 | grep -i error
echo

# 5. 检查系统资源
echo "5. 系统资源状态:"
echo "CPU: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)%"
echo "内存: $(free | grep Mem | awk '{printf "%.1f%", $3/$2 * 100.0}')"
echo "磁盘: $(df -h / | tail -1 | awk '{print $5}')"
echo

# 6. 检查数据库连接
echo "6. 数据库连接检查:"
docker exec docker-auto-postgres pg_isready -U dockerauto 2>/dev/null && echo "数据库连接正常" || echo "数据库连接失败"
echo

# 7. 检查 Redis 连接
echo "7. Redis 连接检查:"
docker exec docker-auto-redis redis-cli ping 2>/dev/null && echo "Redis连接正常" || echo "Redis连接失败"
echo

echo "=== 诊断完成 ==="
```

#### 快速修复步骤
```bash
#!/bin/bash
# quick-fix.sh

# 1. 重启应用容器
echo "重启应用容器..."
docker restart docker-auto-app

# 等待容器启动
sleep 30

# 2. 检查服务状态
if curl -s http://localhost/health > /dev/null; then
    echo "✅ 服务恢复正常"
    exit 0
fi

# 3. 重启整个服务栈
echo "重启整个服务栈..."
cd /opt/docker-auto
docker-compose -f docker-compose.production.yml restart

# 等待服务启动
sleep 60

# 4. 验证服务
if curl -s http://localhost/health > /dev/null; then
    echo "✅ 服务恢复正常"
else
    echo "❌ 服务仍然异常，需要深入调查"
    exit 1
fi
```

### 数据库故障

#### PostgreSQL 故障排查
```bash
#!/bin/bash
# postgres-diagnosis.sh

POSTGRES_CONTAINER="docker-auto-postgres"

echo "=== PostgreSQL 故障诊断 ==="

# 1. 容器状态检查
echo "1. 容器状态:"
docker ps -a --filter "name=$POSTGRES_CONTAINER" --format "table {{.Names}}\t{{.Status}}"

# 2. 数据库进程检查
echo "2. 数据库进程:"
docker exec $POSTGRES_CONTAINER ps aux | grep postgres

# 3. 连接测试
echo "3. 连接测试:"
docker exec $POSTGRES_CONTAINER pg_isready -U dockerauto -d dockerauto

# 4. 检查数据库大小
echo "4. 数据库大小:"
docker exec $POSTGRES_CONTAINER psql -U dockerauto -d dockerauto -c "
    SELECT
        pg_database.datname,
        pg_size_pretty(pg_database_size(pg_database.datname)) AS size
    FROM pg_database
    WHERE datname = 'dockerauto';
"

# 5. 检查连接数
echo "5. 当前连接数:"
docker exec $POSTGRES_CONTAINER psql -U dockerauto -d dockerauto -c "
    SELECT count(*) as connections, state
    FROM pg_stat_activity
    WHERE datname='dockerauto'
    GROUP BY state;
"

# 6. 检查长时间运行的查询
echo "6. 长时间运行的查询:"
docker exec $POSTGRES_CONTAINER psql -U dockerauto -d dockerauto -c "
    SELECT pid, now() - pg_stat_activity.query_start AS duration, query
    FROM pg_stat_activity
    WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
    AND state = 'active';
"

# 7. 检查磁盘空间
echo "7. 数据目录磁盘使用:"
docker exec $POSTGRES_CONTAINER df -h /var/lib/postgresql/data
```

#### 数据库恢复步骤
```bash
#!/bin/bash
# postgres-recovery.sh

BACKUP_DIR="/opt/docker-auto/backup"
POSTGRES_CONTAINER="docker-auto-postgres"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "=== PostgreSQL 恢复流程 ==="

# 1. 创建当前状态备份
echo "1. 创建当前状态备份..."
docker exec $POSTGRES_CONTAINER pg_dump -U dockerauto dockerauto > $BACKUP_DIR/emergency_backup_$TIMESTAMP.sql

# 2. 重启数据库服务
echo "2. 重启数据库服务..."
docker restart $POSTGRES_CONTAINER

# 等待数据库启动
sleep 30

# 3. 检查数据库状态
if docker exec $POSTGRES_CONTAINER pg_isready -U dockerauto -d dockerauto; then
    echo "✅ 数据库重启成功"
else
    echo "❌ 数据库重启失败，尝试从备份恢复..."

    # 4. 从最新备份恢复
    LATEST_BACKUP=$(ls -t $BACKUP_DIR/db_backup_*.sql | head -1)
    if [ -f "$LATEST_BACKUP" ]; then
        echo "从备份恢复: $LATEST_BACKUP"

        # 停止应用防止数据写入
        docker stop docker-auto-app

        # 重建数据库
        docker exec $POSTGRES_CONTAINER psql -U postgres -c "DROP DATABASE IF EXISTS dockerauto;"
        docker exec $POSTGRES_CONTAINER psql -U postgres -c "CREATE DATABASE dockerauto OWNER dockerauto;"

        # 恢复数据
        docker exec -i $POSTGRES_CONTAINER psql -U dockerauto dockerauto < $LATEST_BACKUP

        # 重启应用
        docker start docker-auto-app

        echo "✅ 数据库恢复完成"
    else
        echo "❌ 未找到备份文件，需要手动恢复"
    fi
fi
```

### 应用服务故障

#### 应用诊断脚本
```bash
#!/bin/bash
# app-diagnosis.sh

APP_CONTAINER="docker-auto-app"

echo "=== 应用服务诊断 ==="

# 1. 容器状态
echo "1. 容器状态:"
docker inspect $APP_CONTAINER --format '{{.State.Status}}: {{.State.Error}}'

# 2. 最近日志
echo "2. 最近日志 (错误):"
docker logs $APP_CONTAINER --tail=50 | grep -i "error\|fatal\|panic"

# 3. 内存使用
echo "3. 内存使用:"
docker stats $APP_CONTAINER --no-stream --format "table {{.MemUsage}}\t{{.CPUPerc}}"

# 4. 端口检查
echo "4. 端口检查:"
docker port $APP_CONTAINER

# 5. 健康检查
echo "5. 健康检查:"
docker inspect $APP_CONTAINER --format '{{range .State.Health.Log}}{{.Output}}{{end}}'

# 6. 进程列表
echo "6. 容器内进程:"
docker exec $APP_CONTAINER ps aux

# 7. 磁盘使用
echo "7. 容器磁盘使用:"
docker exec $APP_CONTAINER df -h

# 8. 网络连接
echo "8. 网络连接:"
docker exec $APP_CONTAINER netstat -tlnp
```

### 性能问题排查

#### 性能诊断工具
```bash
#!/bin/bash
# performance-diagnosis.sh

echo "=== 性能问题诊断 ==="

# 1. 系统负载
echo "1. 系统负载:"
uptime
echo

# 2. CPU 使用详情
echo "2. CPU 使用详情:"
top -bn1 | head -20
echo

# 3. 内存使用详情
echo "3. 内存使用详情:"
free -h
cat /proc/meminfo | grep -E "MemAvailable|Buffers|Cached"
echo

# 4. 磁盘 I/O
echo "4. 磁盘 I/O 统计:"
iostat -x 1 5
echo

# 5. 网络统计
echo "5. 网络统计:"
sar -n DEV 1 5
echo

# 6. 数据库性能
echo "6. 数据库慢查询:"
docker exec docker-auto-postgres psql -U dockerauto -d dockerauto -c "
    SELECT query, mean_time, calls, total_time
    FROM pg_stat_statements
    WHERE mean_time > 1000
    ORDER BY total_time DESC
    LIMIT 10;
" 2>/dev/null || echo "pg_stat_statements 扩展未启用"

# 7. 应用响应时间
echo "7. 应用响应时间测试:"
for i in {1..5}; do
    curl -s -o /dev/null -w "请求 $i: %{time_total}s (状态: %{http_code})\n" http://localhost/health
done
```

## 灾难恢复

### 数据备份策略

#### 自动备份脚本
```bash
#!/bin/bash
# disaster-recovery-backup.sh

BACKUP_DIR="/opt/docker-auto/backup"
S3_BUCKET="docker-auto-backups"
RETENTION_DAYS=30

echo "=== 灾难恢复备份 ==="

# 1. 创建备份目录
mkdir -p $BACKUP_DIR
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 2. 数据库备份
echo "备份数据库..."
docker exec docker-auto-postgres pg_dump -U dockerauto dockerauto | gzip > $BACKUP_DIR/db_$TIMESTAMP.sql.gz

# 3. 应用数据备份
echo "备份应用数据..."
tar -czf $BACKUP_DIR/app_data_$TIMESTAMP.tar.gz /opt/docker-auto/data/app

# 4. 配置文件备份
echo "备份配置文件..."
tar -czf $BACKUP_DIR/config_$TIMESTAMP.tar.gz /opt/docker-auto/config /opt/docker-auto/.env.production

# 5. Docker 镜像备份
echo "备份 Docker 镜像..."
docker save await2719/docker-auto:latest | gzip > $BACKUP_DIR/docker_image_$TIMESTAMP.tar.gz

# 6. 上传到云存储
echo "上传备份到 S3..."
aws s3 sync $BACKUP_DIR s3://$S3_BUCKET/daily-backups/$(date +%Y-%m-%d)/

# 7. 清理旧备份
echo "清理本地旧备份..."
find $BACKUP_DIR -name "*.gz" -mtime +$RETENTION_DAYS -delete

# 8. 验证备份
echo "验证备份完整性..."
for backup in $BACKUP_DIR/*_$TIMESTAMP.*; do
    if [ -f "$backup" ] && [ -s "$backup" ]; then
        echo "✅ $(basename $backup)"
    else
        echo "❌ $(basename $backup) - 备份失败或为空"
    fi
done

echo "备份完成: $TIMESTAMP"
```

### 完整系统恢复

#### 系统恢复脚本
```bash
#!/bin/bash
# full-system-recovery.sh

BACKUP_DIR="/opt/docker-auto/backup"
RECOVERY_DIR="/opt/docker-auto-recovery"

if [ -z "$1" ]; then
    echo "使用方法: $0 <备份时间戳>"
    echo "可用的备份:"
    ls $BACKUP_DIR/db_*.sql.gz | sed 's/.*db_\(.*\).sql.gz/\1/'
    exit 1
fi

BACKUP_TIMESTAMP=$1

echo "=== 完整系统恢复 ==="
echo "恢复时间点: $BACKUP_TIMESTAMP"

# 1. 停止当前服务
echo "1. 停止当前服务..."
cd /opt/docker-auto
docker-compose -f docker-compose.production.yml down

# 2. 备份当前状态
echo "2. 备份当前状态..."
mkdir -p $RECOVERY_DIR
cp -r /opt/docker-auto/data $RECOVERY_DIR/data_current
cp -r /opt/docker-auto/config $RECOVERY_DIR/config_current

# 3. 恢复数据库
echo "3. 恢复数据库..."
docker-compose -f docker-compose.production.yml up -d postgres
sleep 30

# 删除现有数据库
docker exec docker-auto-postgres psql -U postgres -c "DROP DATABASE IF EXISTS dockerauto;"
docker exec docker-auto-postgres psql -U postgres -c "CREATE DATABASE dockerauto OWNER dockerauto;"

# 恢复数据
if [ -f "$BACKUP_DIR/db_$BACKUP_TIMESTAMP.sql.gz" ]; then
    zcat $BACKUP_DIR/db_$BACKUP_TIMESTAMP.sql.gz | docker exec -i docker-auto-postgres psql -U dockerauto dockerauto
    echo "✅ 数据库恢复完成"
else
    echo "❌ 数据库备份文件不存在"
    exit 1
fi

# 4. 恢复应用数据
echo "4. 恢复应用数据..."
if [ -f "$BACKUP_DIR/app_data_$BACKUP_TIMESTAMP.tar.gz" ]; then
    rm -rf /opt/docker-auto/data/app
    tar -xzf $BACKUP_DIR/app_data_$BACKUP_TIMESTAMP.tar.gz -C /
    echo "✅ 应用数据恢复完成"
else
    echo "⚠️  应用数据备份不存在，跳过"
fi

# 5. 恢复配置文件
echo "5. 恢复配置文件..."
if [ -f "$BACKUP_DIR/config_$BACKUP_TIMESTAMP.tar.gz" ]; then
    tar -xzf $BACKUP_DIR/config_$BACKUP_TIMESTAMP.tar.gz -C /
    echo "✅ 配置文件恢复完成"
else
    echo "⚠️  配置备份不存在，跳过"
fi

# 6. 恢复 Docker 镜像
echo "6. 恢复 Docker 镜像..."
if [ -f "$BACKUP_DIR/docker_image_$BACKUP_TIMESTAMP.tar.gz" ]; then
    zcat $BACKUP_DIR/docker_image_$BACKUP_TIMESTAMP.tar.gz | docker load
    echo "✅ Docker 镜像恢复完成"
else
    echo "⚠️  Docker 镜像备份不存在，将从仓库拉取"
fi

# 7. 启动服务
echo "7. 启动服务..."
docker-compose -f docker-compose.production.yml up -d

# 8. 等待服务启动
echo "8. 等待服务启动..."
sleep 60

# 9. 验证恢复
echo "9. 验证系统恢复..."
if curl -s http://localhost/health > /dev/null; then
    echo "✅ 系统恢复成功"

    # 发送恢复通知
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"Docker Auto 系统已从备份 $BACKUP_TIMESTAMP 成功恢复\"}" \
        $SLACK_WEBHOOK_URL
else
    echo "❌ 系统恢复失败，检查日志"
    docker-compose -f docker-compose.production.yml logs
fi

echo "恢复过程完成"
```

## 通信和协调

### 事故通信模板

#### 内部通知模板
```yaml
# incident-notification-templates.yml
internal_templates:
  initial_alert:
    subject: "[P{{priority}}] Docker Auto 事故 - {{title}}"
    body: |
      事故级别: P{{priority}}
      事故标题: {{title}}
      检测时间: {{detected_at}}
      影响范围: {{impact}}

      当前状态: 调查中
      负责人: {{incident_commander}}

      初步行动:
      - {{initial_actions}}

      下次更新时间: {{next_update_time}}

  status_update:
    subject: "[P{{priority}}] Docker Auto 事故更新 - {{title}}"
    body: |
      事故: {{title}}
      状态: {{current_status}}

      进展更新:
      {{progress_update}}

      已完成行动:
      {{completed_actions}}

      下一步计划:
      {{next_steps}}

      预计解决时间: {{eta}}

  resolution:
    subject: "[已解决] Docker Auto 事故 - {{title}}"
    body: |
      事故: {{title}}
      解决时间: {{resolved_at}}
      持续时间: {{duration}}

      根本原因:
      {{root_cause}}

      解决措施:
      {{resolution_steps}}

      后续行动:
      {{follow_up_actions}}

external_templates:
  service_degradation:
    subject: "Docker Auto 服务受到影响"
    body: |
      我们正在经历影响 Docker Auto 服务的技术问题。

      影响: {{user_impact}}
      开始时间: {{start_time}}

      我们的团队正在积极调查并解决这个问题。
      预计解决时间: {{eta}}

      我们将在状态页面提供定期更新: {{status_page_url}}

  service_restoration:
    subject: "Docker Auto 服务已恢复"
    body: |
      Docker Auto 服务已完全恢复正常操作。

      事故持续时间: {{duration}}
      问题原因: {{brief_cause}}

      我们对此次中断给您带来的不便深表歉意。

      完整的事故报告将在 72 小时内在我们的博客发布。
```

#### 状态页面更新脚本
```bash
#!/bin/bash
# update-status-page.sh

STATUS_PAGE_API="https://api.statuspage.io/v1/pages/$PAGE_ID"
API_KEY=$STATUSPAGE_API_KEY

case "$1" in
    "investigating")
        curl -X POST "$STATUS_PAGE_API/incidents" \
        -H "Authorization: OAuth $API_KEY" \
        -d "incident[name]=$2" \
        -d "incident[status]=investigating" \
        -d "incident[impact_override]=major" \
        -d "incident[body]=$3"
        ;;

    "identified")
        curl -X PATCH "$STATUS_PAGE_API/incidents/$2" \
        -H "Authorization: OAuth $API_KEY" \
        -d "incident[status]=identified" \
        -d "incident[body]=$3"
        ;;

    "monitoring")
        curl -X PATCH "$STATUS_PAGE_API/incidents/$2" \
        -H "Authorization: OAuth $API_KEY" \
        -d "incident[status]=monitoring" \
        -d "incident[body]=$3"
        ;;

    "resolved")
        curl -X PATCH "$STATUS_PAGE_API/incidents/$2" \
        -H "Authorization: OAuth $API_KEY" \
        -d "incident[status]=resolved" \
        -d "incident[body]=$3"
        ;;
esac
```

## 事后分析

### 事故报告模板
```markdown
# 事故报告 - {{incident_title}}

## 执行摘要
- **事故时间**: {{start_time}} - {{end_time}} (UTC)
- **持续时间**: {{duration}}
- **影响级别**: P{{priority}}
- **受影响用户**: {{affected_users}}
- **根本原因**: {{root_cause_summary}}

## 事故时间线
| 时间 (UTC) | 事件 |
|-----------|------|
| {{timeline_entries}} |

## 根本原因分析
### 直接原因
{{immediate_cause}}

### 根本原因
{{root_cause_detailed}}

### 贡献因素
{{contributing_factors}}

## 影响评估
### 用户影响
- 受影响用户数: {{affected_user_count}}
- 功能影响: {{feature_impact}}
- 数据影响: {{data_impact}}

### 业务影响
- 收入影响: {{revenue_impact}}
- SLA 违约: {{sla_breach}}

## 响应分析
### 做得好的地方
{{what_went_well}}

### 需要改进的地方
{{what_went_wrong}}

## 纠正措施
| 措施 | 负责人 | 截止日期 | 状态 |
|------|--------|----------|------|
| {{corrective_actions}} |

## 预防措施
| 措施 | 负责人 | 截止日期 | 状态 |
|------|--------|----------|------|
| {{preventive_actions}} |

## 经验教训
{{lessons_learned}}
```

### 改进措施跟踪
```python
#!/usr/bin/env python3
# action-items-tracker.py

import json
import datetime
from dataclasses import dataclass
from typing import List, Dict
from enum import Enum

class ActionStatus(Enum):
    NOT_STARTED = "not_started"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    BLOCKED = "blocked"

@dataclass
class ActionItem:
    id: str
    title: str
    description: str
    assignee: str
    due_date: datetime.date
    status: ActionStatus
    incident_id: str
    created_date: datetime.date

class ActionTracker:
    def __init__(self, data_file: str = "action_items.json"):
        self.data_file = data_file
        self.actions: List[ActionItem] = []
        self.load_data()

    def load_data(self):
        try:
            with open(self.data_file, 'r') as f:
                data = json.load(f)
                self.actions = [
                    ActionItem(**item) for item in data
                ]
        except FileNotFoundError:
            self.actions = []

    def save_data(self):
        with open(self.data_file, 'w') as f:
            json.dump([
                {
                    **action.__dict__,
                    'due_date': action.due_date.isoformat(),
                    'created_date': action.created_date.isoformat(),
                    'status': action.status.value
                }
                for action in self.actions
            ], f, indent=2)

    def add_action(self, action: ActionItem):
        self.actions.append(action)
        self.save_data()

    def get_overdue_actions(self) -> List[ActionItem]:
        today = datetime.date.today()
        return [
            action for action in self.actions
            if action.due_date < today and action.status != ActionStatus.COMPLETED
        ]

    def get_actions_by_assignee(self, assignee: str) -> List[ActionItem]:
        return [
            action for action in self.actions
            if action.assignee == assignee and action.status != ActionStatus.COMPLETED
        ]

    def generate_report(self) -> Dict:
        total = len(self.actions)
        completed = len([a for a in self.actions if a.status == ActionStatus.COMPLETED])
        overdue = len(self.get_overdue_actions())

        return {
            "total_actions": total,
            "completed_actions": completed,
            "completion_rate": completed / total if total > 0 else 0,
            "overdue_actions": overdue,
            "actions_by_status": {
                status.value: len([a for a in self.actions if a.status == status])
                for status in ActionStatus
            }
        }

if __name__ == "__main__":
    tracker = ActionTracker()
    report = tracker.generate_report()

    print("=== 改进措施跟踪报告 ===")
    print(f"总计措施: {report['total_actions']}")
    print(f"已完成: {report['completed_actions']}")
    print(f"完成率: {report['completion_rate']:.1%}")
    print(f"逾期措施: {report['overdue_actions']}")

    overdue_actions = tracker.get_overdue_actions()
    if overdue_actions:
        print("\n逾期措施:")
        for action in overdue_actions:
            print(f"- {action.title} (负责人: {action.assignee}, 截止: {action.due_date})")
```

## 应急工具包

### 快速诊断工具
```bash
#!/bin/bash
# emergency-toolkit.sh

case "$1" in
    "status")
        echo "=== 快速状态检查 ==="
        curl -s http://localhost/health | jq '.'
        docker ps --filter "name=docker-auto" --format "table {{.Names}}\t{{.Status}}"
        ;;

    "logs")
        echo "=== 最近错误日志 ==="
        docker logs docker-auto-app --tail=50 | grep -i "error\|fatal\|panic"
        ;;

    "resources")
        echo "=== 资源使用情况 ==="
        docker stats --no-stream
        df -h
        free -h
        ;;

    "restart")
        echo "=== 重启服务 ==="
        docker restart docker-auto-app docker-auto-nginx
        ;;

    "emergency-stop")
        echo "=== 紧急停止 ==="
        docker stop docker-auto-app docker-auto-nginx
        ;;

    "backup")
        echo "=== 紧急备份 ==="
        /opt/docker-auto/scripts/disaster-recovery-backup.sh
        ;;

    *)
        echo "使用方法: $0 {status|logs|resources|restart|emergency-stop|backup}"
        ;;
esac
```

### 联系人信息
```yaml
# emergency-contacts.yml
emergency_contacts:
  primary_oncall:
    name: "John Doe"
    role: "SRE Lead"
    phone: "+1-555-0101"
    email: "john.doe@company.com"
    slack: "@johndoe"

  secondary_oncall:
    name: "Jane Smith"
    role: "DevOps Engineer"
    phone: "+1-555-0102"
    email: "jane.smith@company.com"
    slack: "@janesmith"

  incident_commander:
    name: "Mike Johnson"
    role: "Engineering Manager"
    phone: "+1-555-0103"
    email: "mike.johnson@company.com"
    slack: "@mikejohnson"

escalation_chain:
  level_1: "primary_oncall"
  level_2: "secondary_oncall"
  level_3: "incident_commander"

external_contacts:
  hosting_provider:
    name: "AWS Support"
    phone: "+1-800-123-4567"
    account: "123456789012"

  cdn_provider:
    name: "CloudFlare Support"
    phone: "+1-800-987-6543"
    account: "cloudflare-account-id"
```

---

**相关文档**: [监控配置](../admin/monitoring.md) | [部署指南](deployment.md) | [安全配置](../admin/security.md)