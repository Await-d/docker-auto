# 贡献指南

感谢您对 Docker Auto 项目的关注！我们欢迎各种形式的贡献。

## 贡献方式

### 🐛 报告 Bug
1. 搜索 [GitHub Issues](https://github.com/your-org/docker-auto/issues) 确认问题未被报告
2. 创建新的 Issue，包含：
   - 详细的问题描述
   - 重现步骤
   - 期望的行为
   - 实际的行为
   - 环境信息（OS、Docker 版本等）

### 💡 功能建议
1. 在 [GitHub Discussions](https://github.com/your-org/docker-auto/discussions) 中讨论想法
2. 创建功能请求 Issue，包含：
   - 功能描述
   - 使用场景
   - 预期收益
   - 实现建议（可选）

### 🔧 代码贡献
1. Fork 仓库
2. 创建功能分支：`git checkout -b feature/awesome-feature`
3. 编写代码和测试
4. 提交更改：`git commit -m "feat: add awesome feature"`
5. 推送分支：`git push origin feature/awesome-feature`
6. 创建 Pull Request

## 开发环境设置

### 前置条件
- Go 1.21+
- Node.js 18+
- Docker 20.10+
- PostgreSQL 13+

### 本地开发
```bash
# 克隆仓库
git clone https://github.com/your-org/docker-auto.git
cd docker-auto

# 启动数据库
docker run -d --name postgres-dev \
  -e POSTGRES_DB=dockerauto \
  -e POSTGRES_USER=dev \
  -e POSTGRES_PASSWORD=dev \
  -p 5432:5432 \
  postgres:15-alpine

# 后端开发
cd backend
cp .env.example .env
go mod download
go run cmd/server/main.go

# 前端开发
cd frontend
npm install
npm run dev
```

## 代码规范

### Go 代码规范
- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码质量
- 遵循 Go 官方编码规范
- 添加适当的注释和文档

### TypeScript 代码规范
- 使用 ESLint + Prettier
- 遵循 Vue 3 组合式 API 风格
- 使用 TypeScript 严格模式
- 组件命名使用 PascalCase

### 提交消息规范
使用 [Conventional Commits](https://conventionalcommits.org/) 格式：

```
type(scope): description

body

footer
```

类型：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式化
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

示例：
```
feat(containers): add batch update functionality

Add support for updating multiple containers simultaneously
with different update strategies.

Closes #123
```

## 测试要求

### 后端测试
```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test ./internal/service

# 生成测试覆盖率报告
go test -cover ./...
```

### 前端测试
```bash
# 单元测试
npm run test:unit

# 端到端测试
npm run test:e2e

# 测试覆盖率
npm run test:coverage
```

### 测试要求
- 新功能必须包含测试
- 测试覆盖率不低于 80%
- 所有测试必须通过 CI 检查

## Pull Request 流程

### 提交前检查清单
- [ ] 代码遵循项目规范
- [ ] 添加了适当的测试
- [ ] 测试全部通过
- [ ] 文档已更新
- [ ] 提交消息符合规范

### 审核流程
1. 自动化检查（CI/CD）
2. 代码审核（至少 1 个维护者）
3. 测试验证
4. 合并到主分支

### 审核标准
- 功能完整性
- 代码质量
- 测试充分性
- 文档完整性
- 向后兼容性

## 文档贡献

### 文档结构
```
docs/
├── user/           # 用户文档
├── admin/          # 管理员文档
├── developer/      # 开发者文档
└── operations/     # 运维文档
```

### 文档规范
- 使用清晰的标题结构
- 提供代码示例
- 包含截图（如适用）
- 保持中英双语同步

## 社区准则

### 行为准则
- 尊重他人，友善交流
- 建设性反馈和讨论
- 包容不同观点和经验水平
- 遵循开源社区最佳实践

### 沟通渠道
- [GitHub Issues](https://github.com/your-org/docker-auto/issues) - Bug 报告和功能请求
- [GitHub Discussions](https://github.com/your-org/docker-auto/discussions) - 一般讨论
- [Pull Requests](https://github.com/your-org/docker-auto/pulls) - 代码审核

## 维护者

当前项目维护者：
- [@maintainer1](https://github.com/maintainer1)
- [@maintainer2](https://github.com/maintainer2)

## 致谢

感谢所有为 Docker Auto 项目贡献代码、文档、测试和想法的贡献者！

---

**准备开始贡献了吗？** 查看 [Good First Issues](https://github.com/your-org/docker-auto/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) 找到适合的入门任务。