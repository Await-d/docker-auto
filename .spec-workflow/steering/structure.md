# Project Structure

## Directory Organization

```
docker-auto/
├── backend/                     # Go后端服务
│   ├── cmd/                    # 应用程序入口点
│   │   └── server/            # 主服务器应用
│   ├── internal/              # 私有应用代码
│   │   ├── api/              # WebSocket和API处理器
│   │   ├── config/           # 配置管理
│   │   ├── controller/       # HTTP控制器层
│   │   ├── middleware/       # HTTP中间件
│   │   ├── model/           # 数据模型和GORM实体
│   │   ├── repository/      # 数据访问层接口
│   │   └── service/         # 业务逻辑层
│   ├── pkg/                   # 可重用包
│   │   ├── docker/          # Docker客户端和操作
│   │   ├── scheduler/       # 任务调度（Cron）
│   │   ├── security/        # 安全相关功能
│   │   ├── monitoring/      # 监控和指标
│   │   ├── utils/           # 工具函数
│   │   └── ...              # 其他可重用包
│   ├── migrations/          # 数据库迁移文件
│   ├── go.mod              # Go模块定义
│   └── go.sum              # Go依赖校验和
├── frontend/                   # Vue 3前端应用
│   ├── src/                  # 源代码
│   │   ├── api/             # API调用封装
│   │   ├── components/      # 可重用组件
│   │   │   ├── common/     # 通用组件
│   │   │   ├── container/  # 容器相关组件
│   │   │   ├── dashboard/  # 仪表板组件
│   │   │   └── settings/   # 设置组件
│   │   ├── composables/    # 组合式函数
│   │   ├── router/         # 路由配置
│   │   ├── store/          # Pinia状态管理
│   │   ├── types/          # TypeScript类型定义
│   │   ├── utils/          # 工具函数
│   │   ├── views/          # 页面组件
│   │   └── styles/         # 全局样式
│   ├── public/             # 静态资源
│   ├── package.json        # NPM依赖和脚本
│   └── vite.config.ts      # Vite构建配置
├── database/               # 数据库相关文件
├── data/                  # 运行时数据目录
├── .env.example           # 环境配置示例
├── docker-compose.yml     # Docker编排配置
└── README.md             # 项目文档
```

## Naming Conventions

### 文件命名
- **Go文件**: `snake_case.go` (如 `container_service.go`)
- **Go包**: `小写单数` (如 `service`, `model`)
- **Vue组件**: `PascalCase.vue` (如 `ContainerCard.vue`)
- **TypeScript文件**: `camelCase.ts` 或 `kebab-case.ts`
- **测试文件**: `[filename]_test.go` (Go), `[filename].test.ts` (TS)

### 代码命名
- **Go类型/结构体**: `PascalCase` (如 `ContainerService`, `UpdateHistory`)
- **Go函数/方法**: `PascalCase` (如 `GetContainers`, `UpdateContainer`)
- **Go常量**: `PascalCase` 或 `UPPER_SNAKE_CASE`
- **Go变量**: `camelCase`
- **TypeScript**: `camelCase` (变量/函数), `PascalCase` (类型/接口)

## Import Patterns

### Go Import Order
1. 标准库包
2. 第三方包
3. 项目内部包 (按相对路径深度排序)

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"

    "docker-auto/internal/config"
    "docker-auto/internal/model"
    "docker-auto/pkg/docker"
)
```

### TypeScript Import Order
1. 外部依赖 (node_modules)
2. 内部模块 (绝对路径 @/)
3. 相对导入
4. 类型导入

```typescript
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'

import { useAuthStore } from '@/store/auth'
import { ContainerAPI } from '@/api/containers'

import type { Container } from './types'
```

## Code Structure Patterns

### Go文件组织模式
```go
// 1. 包声明和导入
package service

import (...)

// 2. 常量和类型定义
const (
    DefaultTimeout = 30 * time.Second
)

type ContainerService struct {
    // 字段定义
}

// 3. 构造函数
func NewContainerService(...) *ContainerService {
    // 初始化逻辑
}

// 4. 公共方法 (按字母顺序)
func (s *ContainerService) CreateContainer(...) error {
    // 核心逻辑
}

// 5. 私有方法 (按字母顺序)
func (s *ContainerService) validateConfig(...) error {
    // 辅助逻辑
}
```

### Vue组件组织模式
```vue
<template>
  <!-- 模板内容 -->
</template>

<script setup lang="ts">
// 1. 导入
import { ref, computed, onMounted } from 'vue'
import type { Container } from '@/types'

// 2. 接口定义
interface Props {
  containerId: string
}

// 3. Props和Emits
const props = defineProps<Props>()
const emit = defineEmits<{
  update: [container: Container]
}>()

// 4. 响应式数据
const loading = ref(false)
const containers = ref<Container[]>([])

// 5. 计算属性
const filteredContainers = computed(() => {
  // 计算逻辑
})

// 6. 方法
const handleUpdate = async () => {
  // 方法实现
}

// 7. 生命周期钩子
onMounted(() => {
  // 初始化逻辑
})
</script>

<style scoped>
/* 组件样式 */
</style>
```

## Code Organization Principles

1. **单一职责原则**: 每个文件、类、函数都有明确的单一职责
2. **模块化**: 按功能域组织代码，高内聚低耦合
3. **分层架构**: 严格的分层依赖关系，上层可以依赖下层，反之不可
4. **接口分离**: 使用接口定义契约，便于测试和替换实现

## Module Boundaries

### 后端分层边界
```
Controller Layer (HTTP处理)
    ↓ (只能调用)
Service Layer (业务逻辑)
    ↓ (只能调用)
Repository Layer (数据访问)
    ↓ (只能调用)
Model Layer (数据模型)
```

### 包依赖规则
- `internal/` 包只能被同一项目内部使用
- `pkg/` 包可以被外部项目引用
- `cmd/` 是应用入口，可以引用任何内部包
- 业务逻辑不应直接依赖HTTP/数据库细节

### 前端模块边界
- **Views**: 页面级组件，组装其他组件
- **Components**: 可复用组件，不包含业务逻辑
- **Composables**: 业务逻辑复用，无UI依赖
- **Stores**: 全局状态管理，单向数据流
- **API**: 网络请求封装，统一错误处理

## Code Size Guidelines

### 文件大小限制
- **Go文件**: 建议 <500行，最大 <1000行
- **Vue组件**: 建议 <300行，最大 <500行
- **TypeScript文件**: 建议 <400行，最大 <800行

### 函数复杂度
- **Go方法**: 建议 <50行，最大 <100行
- **Vue方法**: 建议 <30行，最大 <50行
- **嵌套深度**: 最大3层，建议2层以内

### 结构体/接口复杂度
- **Go结构体字段**: 建议 <20个
- **接口方法**: 建议 <10个
- **TypeScript接口**: 建议 <15个属性

## Dashboard/Monitoring Structure

### 前端监控组件结构
```
frontend/src/components/dashboard/
├── widgets/              # 仪表板小部件
│   ├── ContainerStats.vue
│   ├── SystemOverview.vue
│   └── RealtimeMonitor.vue
├── config/               # 小部件配置组件
│   ├── ContainerStatsConfig.vue
│   └── SystemOverviewConfig.vue
└── WidgetWrapper.vue     # 小部件容器组件
```

### 实时数据流架构
```
WebSocket连接 → Store状态 → 响应式组件 → UI更新
                    ↓
               本地缓存/持久化
```

### 关注点分离
- **数据获取**: API层和Store管理
- **状态管理**: Pinia响应式状态
- **UI渲染**: 纯展示组件
- **交互逻辑**: 组合式函数
- **样式**: Scoped CSS + 全局主题

## Documentation Standards

### Go代码文档
- 所有公共函数/类型必须有GoDoc注释
- 包级别文档在`doc.go`文件中
- 复杂算法添加行内注释
- API文档使用Swagger注解

### TypeScript代码文档
- 公共接口使用JSDoc注释
- 组件使用Vue文档规范
- 复杂业务逻辑添加说明注释
- README文件说明组件用法

### 数据库文档
- 表结构使用GORM注释
- 迁移文件包含变更说明
- 索引策略文档化
- 数据关系图维护

### API文档
- 使用OpenAPI/Swagger规范
- 包含请求/响应示例
- 错误码完整定义
- 认证方式说明