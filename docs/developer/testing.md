# 测试指南

## 测试策略

Docker Auto 采用分层测试策略，确保代码质量和系统稳定性：

- **单元测试**: 测试单个函数和组件
- **集成测试**: 测试模块间交互
- **API 测试**: 测试 HTTP 接口
- **E2E 测试**: 测试完整用户流程
- **性能测试**: 测试系统性能指标

## 后端测试

### 单元测试

#### 测试结构
```
backend/
├── internal/
│   ├── service/
│   │   ├── container.go
│   │   └── container_test.go
│   ├── repository/
│   │   ├── user.go
│   │   └── user_test.go
│   └── handler/
│       ├── auth.go
│       └── auth_test.go
└── tests/
    ├── fixtures/
    ├── mocks/
    └── testutil/
```

#### 示例：Service 层测试
```go
// internal/service/container_test.go
package service

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "docker-auto/internal/model"
    "docker-auto/tests/mocks"
)

func TestContainerService_Create(t *testing.T) {
    // 准备测试数据
    container := &model.Container{
        Name:  "test-nginx",
        Image: "nginx",
        Tag:   "latest",
    }

    // 创建模拟依赖
    mockRepo := &mocks.ContainerRepository{}
    mockDocker := &mocks.DockerClient{}

    // 设置模拟行为
    mockRepo.On("Create", mock.Anything, container).Return(nil)
    mockDocker.On("ImageExists", "nginx:latest").Return(true, nil)

    // 创建服务实例
    service := &ContainerService{
        repo:         mockRepo,
        dockerClient: mockDocker,
    }

    // 执行测试
    err := service.Create(context.Background(), container)

    // 验证结果
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
    mockDocker.AssertExpectations(t)
}

func TestContainerService_Create_ImageNotFound(t *testing.T) {
    container := &model.Container{
        Name:  "test-nginx",
        Image: "nginx",
        Tag:   "nonexistent",
    }

    mockRepo := &mocks.ContainerRepository{}
    mockDocker := &mocks.DockerClient{}

    // 模拟镜像不存在
    mockDocker.On("ImageExists", "nginx:nonexistent").Return(false, nil)

    service := &ContainerService{
        repo:         mockRepo,
        dockerClient: mockDocker,
    }

    err := service.Create(context.Background(), container)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "image not found")
}
```

#### 测试工具函数
```go
// tests/testutil/database.go
package testutil

import (
    "database/sql"
    "testing"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func SetupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("postgres", "postgres://test:test@localhost/dockerauto_test?sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }

    // 运行迁移
    m, err := migrate.New(
        "file://../../migrations",
        "postgres://test:test@localhost/dockerauto_test?sslmode=disable",
    )
    if err != nil {
        t.Fatalf("Failed to create migration: %v", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        t.Fatalf("Failed to run migrations: %v", err)
    }

    t.Cleanup(func() {
        db.Close()
    })

    return db
}
```

### 集成测试

#### 数据库集成测试
```go
// tests/integration/repository_test.go
package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "docker-auto/internal/repository"
    "docker-auto/internal/model"
    "docker-auto/tests/testutil"
)

func TestUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := testutil.SetupTestDB(t)
    repo := repository.NewUserRepository(db)

    t.Run("Create and Get User", func(t *testing.T) {
        user := &model.User{
            Email:    "test@example.com",
            Password: "hashedpassword",
            Role:     model.RoleUser,
        }

        // 创建用户
        err := repo.Create(context.Background(), user)
        assert.NoError(t, err)
        assert.NotZero(t, user.ID)

        // 获取用户
        retrieved, err := repo.GetByEmail(context.Background(), user.Email)
        assert.NoError(t, err)
        assert.Equal(t, user.Email, retrieved.Email)
        assert.Equal(t, user.Role, retrieved.Role)
    })
}
```

### API 测试

#### HTTP Handler 测试
```go
// internal/handler/auth_test.go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "docker-auto/internal/dto"
    "docker-auto/tests/mocks"
)

func TestAuthHandler_Login(t *testing.T) {
    gin.SetMode(gin.TestMode)

    mockService := &mocks.AuthService{}
    handler := &AuthHandler{authService: mockService}

    t.Run("Valid Login", func(t *testing.T) {
        loginReq := dto.LoginRequest{
            Email:    "admin@example.com",
            Password: "password123",
        }

        mockService.On("Login", loginReq.Email, loginReq.Password).
            Return("jwt-token", nil)

        reqBody, _ := json.Marshal(loginReq)
        req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(reqBody))
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router := gin.New()
        router.POST("/api/auth/login", handler.Login)
        router.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)

        var response dto.LoginResponse
        err := json.Unmarshal(w.Body.Bytes(), &response)
        assert.NoError(t, err)
        assert.Equal(t, "jwt-token", response.Token)
    })
}
```

### Mock 生成

#### 使用 mockery 生成 Mock
```bash
# 安装 mockery
go install github.com/vektra/mockery/v2@latest

# 生成 Mock 接口
mockery --dir=internal/repository --name=UserRepository --output=tests/mocks
mockery --dir=pkg/docker --name=Client --output=tests/mocks
```

#### Mock 配置文件
```yaml
# .mockery.yaml
with-expecter: true
dir: "tests/mocks"
packages:
  docker-auto/internal/repository:
    interfaces:
      UserRepository:
      ContainerRepository:
  docker-auto/pkg/docker:
    interfaces:
      Client:
```

## 前端测试

### 单元测试

#### Vue 组件测试
```typescript
// frontend/src/components/__tests__/ContainerCard.spec.ts
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ContainerCard from '../ContainerCard.vue'
import type { Container } from '@/types/container'

describe('ContainerCard', () => {
  const mockContainer: Container = {
    id: 1,
    name: 'test-nginx',
    image: 'nginx',
    tag: 'latest',
    status: 'running',
    created_at: '2024-01-01T00:00:00Z'
  }

  it('renders container information correctly', () => {
    const wrapper = mount(ContainerCard, {
      props: { container: mockContainer }
    })

    expect(wrapper.text()).toContain('test-nginx')
    expect(wrapper.text()).toContain('nginx:latest')
    expect(wrapper.find('.status-running')).toBeTruthy()
  })

  it('emits start event when start button clicked', async () => {
    const wrapper = mount(ContainerCard, {
      props: { container: { ...mockContainer, status: 'stopped' } }
    })

    await wrapper.find('.btn-start').trigger('click')
    expect(wrapper.emitted('start')).toHaveLength(1)
    expect(wrapper.emitted('start')?.[0]).toEqual([mockContainer.id])
  })
})
```

#### Composable 测试
```typescript
// frontend/src/composables/__tests__/useContainer.spec.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useContainer } from '../useContainer'
import { ref } from 'vue'

// Mock API
vi.mock('@/api/container', () => ({
  getContainers: vi.fn().mockResolvedValue([]),
  startContainer: vi.fn().mockResolvedValue({}),
  stopContainer: vi.fn().mockResolvedValue({})
}))

describe('useContainer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches containers on initialization', async () => {
    const { containers, loading, fetchContainers } = useContainer()

    expect(loading.value).toBe(false)

    await fetchContainers()

    expect(containers.value).toEqual([])
  })

  it('starts container successfully', async () => {
    const { startContainer } = useContainer()
    const { startContainer: apiStart } = await import('@/api/container')

    await startContainer(1)

    expect(apiStart).toHaveBeenCalledWith(1)
  })
})
```

### 集成测试

#### API 集成测试
```typescript
// frontend/src/api/__tests__/container.integration.spec.ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { setupServer } from 'msw/node'
import { rest } from 'msw'
import { getContainers, createContainer } from '../container'

const server = setupServer(
  rest.get('/api/v1/containers', (req, res, ctx) => {
    return res(ctx.json([
      {
        id: 1,
        name: 'test-container',
        image: 'nginx',
        status: 'running'
      }
    ]))
  }),

  rest.post('/api/v1/containers', (req, res, ctx) => {
    return res(ctx.json({ id: 2, name: 'new-container' }))
  })
)

beforeAll(() => server.listen())
afterAll(() => server.close())

describe('Container API Integration', () => {
  it('fetches containers', async () => {
    const containers = await getContainers()
    expect(containers).toHaveLength(1)
    expect(containers[0].name).toBe('test-container')
  })

  it('creates container', async () => {
    const newContainer = await createContainer({
      name: 'new-container',
      image: 'redis'
    })
    expect(newContainer.id).toBe(2)
  })
})
```

## E2E 测试

### Playwright 配置
```typescript
// frontend/playwright.config.ts
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',

  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
  ],

  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
})
```

### E2E 测试用例
```typescript
// frontend/tests/e2e/container-management.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Container Management', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('/login')
    await page.fill('[data-testid="email"]', 'admin@example.com')
    await page.fill('[data-testid="password"]', 'admin123')
    await page.click('[data-testid="login-btn"]')
    await expect(page).toHaveURL('/dashboard')
  })

  test('should display container list', async ({ page }) => {
    await page.goto('/containers')
    await expect(page.locator('[data-testid="container-list"]')).toBeVisible()
  })

  test('should create new container', async ({ page }) => {
    await page.goto('/containers')
    await page.click('[data-testid="add-container-btn"]')

    await page.fill('[data-testid="container-name"]', 'test-nginx')
    await page.fill('[data-testid="container-image"]', 'nginx')
    await page.selectOption('[data-testid="update-strategy"]', 'rolling')

    await page.click('[data-testid="create-btn"]')

    await expect(page.locator('text=Container created successfully')).toBeVisible()
    await expect(page.locator('text=test-nginx')).toBeVisible()
  })

  test('should start and stop container', async ({ page }) => {
    await page.goto('/containers')

    // 启动容器
    await page.click('[data-testid="container-1"] [data-testid="start-btn"]')
    await expect(page.locator('[data-testid="container-1"] .status-running')).toBeVisible()

    // 停止容器
    await page.click('[data-testid="container-1"] [data-testid="stop-btn"]')
    await expect(page.locator('[data-testid="container-1"] .status-stopped')).toBeVisible()
  })
})
```

## 性能测试

### 负载测试

#### K6 配置
```javascript
// tests/performance/load-test.js
import http from 'k6/http'
import { check, group } from 'k6'

export let options = {
  stages: [
    { duration: '2m', target: 100 }, // 升级到 100 用户
    { duration: '5m', target: 100 }, // 保持 100 用户
    { duration: '2m', target: 200 }, // 升级到 200 用户
    { duration: '5m', target: 200 }, // 保持 200 用户
    { duration: '2m', target: 0 },   // 降级到 0 用户
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% 请求响应时间小于 500ms
    http_req_failed: ['rate<0.1'],    // 错误率小于 10%
  },
}

const BASE_URL = 'http://localhost:8080/api/v1'
let authToken = ''

export function setup() {
  // 获取认证令牌
  const loginResponse = http.post(`${BASE_URL}/auth/login`, {
    email: 'admin@example.com',
    password: 'admin123'
  })

  authToken = JSON.parse(loginResponse.body).token
  return { token: authToken }
}

export default function(data) {
  const headers = {
    'Authorization': `Bearer ${data.token}`,
    'Content-Type': 'application/json'
  }

  group('Container API', () => {
    // 获取容器列表
    const listResponse = http.get(`${BASE_URL}/containers`, { headers })
    check(listResponse, {
      'list containers status is 200': (r) => r.status === 200,
      'list containers response time < 200ms': (r) => r.timings.duration < 200,
    })

    // 获取容器详情
    if (listResponse.json().length > 0) {
      const containerId = listResponse.json()[0].id
      const detailResponse = http.get(`${BASE_URL}/containers/${containerId}`, { headers })
      check(detailResponse, {
        'get container detail status is 200': (r) => r.status === 200,
      })
    }
  })
}
```

### 基准测试

#### Go Benchmark 测试
```go
// internal/service/container_benchmark_test.go
package service

import (
    "context"
    "testing"
    "docker-auto/internal/model"
    "docker-auto/tests/testutil"
)

func BenchmarkContainerService_List(b *testing.B) {
    db := testutil.SetupTestDB(b)
    service := NewContainerService(db, nil, nil)

    // 创建测试数据
    for i := 0; i < 100; i++ {
        container := &model.Container{
            Name:  fmt.Sprintf("container-%d", i),
            Image: "nginx",
            Tag:   "latest",
        }
        service.Create(context.Background(), container)
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.List(context.Background(), &ListOptions{
                Limit: 10,
                Offset: 0,
            })
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## 测试数据管理

### 测试 Fixtures
```go
// tests/fixtures/containers.go
package fixtures

import "docker-auto/internal/model"

var TestContainers = []model.Container{
    {
        ID:     1,
        Name:   "nginx-web",
        Image:  "nginx",
        Tag:    "latest",
        Status: model.StatusRunning,
    },
    {
        ID:     2,
        Name:   "redis-cache",
        Image:  "redis",
        Tag:    "6-alpine",
        Status: model.StatusStopped,
    },
}

func CreateTestContainer() *model.Container {
    return &model.Container{
        Name:  "test-container",
        Image: "alpine",
        Tag:   "latest",
    }
}
```

### 数据库种子
```go
// tests/seed/main.go
package main

import (
    "context"
    "database/sql"
    "log"
    "docker-auto/internal/model"
    "docker-auto/internal/repository"
    "docker-auto/tests/fixtures"
)

func main() {
    db, err := sql.Open("postgres", "postgres://test:test@localhost/dockerauto_test?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    userRepo := repository.NewUserRepository(db)
    containerRepo := repository.NewContainerRepository(db)

    // 创建测试用户
    user := &model.User{
        Email:    "test@example.com",
        Password: "$2a$10$hashed_password",
        Role:     model.RoleAdmin,
    }
    userRepo.Create(context.Background(), user)

    // 创建测试容器
    for _, container := range fixtures.TestContainers {
        containerRepo.Create(context.Background(), &container)
    }

    log.Println("Test data seeded successfully")
}
```

## CI/CD 集成

### GitHub Actions 配置
```yaml
# .github/workflows/test.yml
name: Test Suite

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: dockerauto_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: |
        cd backend
        go mod download

    - name: Run unit tests
      run: |
        cd backend
        go test -v -race -cover ./...

    - name: Run integration tests
      run: |
        cd backend
        go test -v -tags integration ./tests/integration/...
      env:
        DB_HOST: localhost
        DB_PASSWORD: postgres

  frontend-test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Set up Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
        cache: 'npm'
        cache-dependency-path: frontend/package-lock.json

    - name: Install dependencies
      run: |
        cd frontend
        npm ci

    - name: Run unit tests
      run: |
        cd frontend
        npm run test:unit

    - name: Install Playwright
      run: |
        cd frontend
        npx playwright install --with-deps

    - name: Run E2E tests
      run: |
        cd frontend
        npm run test:e2e
```

## 测试最佳实践

### 测试原则
1. **Fast**: 测试应该快速执行
2. **Independent**: 测试之间不应该相互依赖
3. **Repeatable**: 测试结果应该一致
4. **Self-Validating**: 测试应该有清晰的通过/失败结果
5. **Timely**: 测试应该及时编写

### 命名规范
```go
// 功能测试
func TestUserService_CreateUser_Success(t *testing.T) {}
func TestUserService_CreateUser_EmailExists(t *testing.T) {}

// 边界测试
func TestUserService_CreateUser_EmptyEmail(t *testing.T) {}
func TestUserService_CreateUser_InvalidEmail(t *testing.T) {}
```

### 测试覆盖率目标
- **单元测试**: 80%+ 代码覆盖率
- **集成测试**: 覆盖关键业务流程
- **E2E 测试**: 覆盖主要用户场景

---

**相关文档**: [开发环境设置](development-setup.md) | [贡献指南](contributing.md)