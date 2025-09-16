# GitHub Actions CI/CD Pipeline

This directory contains the complete CI/CD automation for the Docker Auto Update System, providing comprehensive testing, building, security scanning, and deployment capabilities.

## 🔄 Available Workflows

### 1. 📋 PR Lint and Format Check

**File**: `pr-lint-check.yml`
**Triggers**: Pull requests affecting code files

#### Features
- **Frontend Quality Checks**:
  - ESLint code linting
  - Prettier code formatting verification
  - TypeScript type checking
  - Vue component validation

- **Backend Quality Checks**:
  - Go code formatting (`gofmt`)
  - Go vet static analysis
  - Go build verification
  - Module dependency validation

- **Security Scanning**:
  - Gosec security scanner for Go code
  - Trivy filesystem vulnerability scanning
  - SARIF report generation

- **Automated Feedback**:
  - PR status comments
  - Detailed error reporting
  - Security vulnerability alerts

---

### 2. 🚀 Auto Release Pipeline

**File**: `auto-release-pipeline.yml`
**Triggers**: Push to `main` or `master` branch

#### Smart Release Logic
- **Skip Conditions**: Ignores commits with `[skip ci]`, `[skip release]`, or from `github-actions[bot]`
- **Change Detection**: Only processes commits affecting core functionality
- **Version Bumping**:
  - `[major]` or `BREAKING CHANGE`: Major version bump
  - `[minor]` or `feat:`: Minor version bump
  - Default: Patch version bump

#### Build Process
1. **Frontend Build**: Vue 3 + TypeScript compilation
2. **Backend Build**: Go binary compilation with static linking
3. **Docker Images**: Multi-platform (AMD64/ARM64) container builds
4. **Documentation**: Automated changelog generation

#### Deployment Artifacts
- **Docker Images**: Published to GitHub Container Registry (GHCR)
  - `ghcr.io/await-d/docker-auto-backend:latest`
  - `ghcr.io/await-d/docker-auto-frontend:latest`
  - `ghcr.io/await-d/docker-auto-docs:latest`
- **Release Archive**: Complete deployment package with quick-start script
- **GitHub Release**: Automated release with changelog and assets

#### Notifications
- **Telegram Integration**: Optional release notifications (requires secrets)
- **Release Notes**: Detailed changelog with technical specifications

---

### 3. 🐳 Docker Build and Test

**File**: `docker-build-test.yml`
**Triggers**: PRs and pushes affecting Docker-related files

#### Multi-Service Testing
- **Backend Container**: Health checks, API readiness
- **Frontend Container**: Build verification, HTTP response testing
- **Database Integration**: PostgreSQL initialization and table creation
- **Docker Compose**: Full stack integration testing

#### Security Validation
- **Image Scanning**: Trivy vulnerability assessment
- **Container Security**: Best practices validation
- **Dependency Checks**: Known vulnerability detection

#### Test Coverage
- Docker image build verification
- Container startup and health checks
- Service interconnectivity testing
- Database schema validation

---

### 4. 📦 Dependency Update Check

**File**: `dependency-update.yml`
**Triggers**: Weekly schedule (Mondays 9:00 AM UTC) + manual dispatch

#### Automated Dependency Management
- **Go Dependencies**: Automatic updates with PR creation
- **Node.js Dependencies**: Manual review process with issue creation
- **Security Audits**: Vulnerability scanning for all dependencies

#### Security Monitoring
- **Go Security**: `govulncheck` vulnerability detection
- **npm Security**: High-severity vulnerability alerts
- **Dockerfile Linting**: Hadolint best practices validation

#### Automation Features
- **Pull Request Creation**: Automated Go dependency updates
- **Issue Management**: Security vulnerability tracking
- **Audit Reports**: Comprehensive security analysis

## 🔧 Configuration Requirements

### Required Secrets (Optional)
- `TELEGRAM_BOT_TOKEN`: For release notifications
- `TELEGRAM_CHAT_ID`: Telegram chat for notifications

### Permissions Required
- `contents: write` - For creating releases and tags
- `packages: write` - For publishing Docker images
- `pull-requests: write` - For PR comments
- `security-events: write` - For security scan results

## 📊 Workflow Status Badges

Add these badges to your README.md:

```markdown
![Auto Release](https://github.com/Await-d/docker-auto/actions/workflows/auto-release-pipeline.yml/badge.svg)
![PR Checks](https://github.com/Await-d/docker-auto/actions/workflows/pr-lint-check.yml/badge.svg)
![Docker Build](https://github.com/Await-d/docker-auto/actions/workflows/docker-build-test.yml/badge.svg)
![Dependencies](https://github.com/Await-d/docker-auto/actions/workflows/dependency-update.yml/badge.svg)
```

## 🚦 Usage Examples

### Triggering Releases

```bash
# Patch release (default)
git commit -m "fix: resolve container restart issue"

# Minor release
git commit -m "feat: add new dashboard widget"

# Major release
git commit -m "feat: new authentication system [major]"

# Skip CI/Release
git commit -m "docs: update README [skip ci]"
```

### Manual Workflow Dispatch

```bash
# Trigger dependency update check
gh workflow run dependency-update.yml

# Check specific workflow status
gh workflow list
gh workflow run <workflow-id>
```

## 🔍 Troubleshooting

### Common Issues

1. **Release Pipeline Fails**
   - Check Docker build logs
   - Verify version bump logic
   - Ensure all tests pass

2. **PR Checks Fail**
   - Run `npm run lint:fix` in frontend
   - Run `gofmt -w .` in backend
   - Check security scan results

3. **Docker Build Issues**
   - Verify Dockerfile syntax
   - Check dependency availability
   - Review build context

### Debug Commands

```bash
# Local testing
docker compose -f docker-compose.yml config
docker compose up --build

# Lint checks
cd frontend && npm run lint:check
cd backend && go vet ./...

# Security scans
trivy fs .
govulncheck ./backend/...
```

## 🏗️ Architecture Integration

These workflows are designed specifically for the Docker Auto Update System architecture:

- **Go Backend**: Gin framework with Docker API integration
- **Vue Frontend**: TypeScript with Element Plus UI
- **PostgreSQL Database**: GORM ORM with migrations
- **Docker Compose**: Multi-service orchestration
- **Security**: JWT authentication, RBAC, audit logging

## 📈 Metrics and Monitoring

The workflows provide comprehensive metrics:

- **Build Success Rate**: Track CI/CD pipeline reliability
- **Security Scan Results**: Monitor vulnerability trends
- **Dependency Health**: Track outdated packages
- **Release Frequency**: Monitor delivery velocity
- **Test Coverage**: Ensure code quality standards

---

## 🤝 Contributing

When contributing to this project:

1. **Create Feature Branches**: Use descriptive branch names
2. **Write Clear Commits**: Follow conventional commit format
3. **Test Locally**: Run lint and build checks before pushing
4. **Review Security**: Address any security scan findings
5. **Update Documentation**: Keep README and docs current

The CI/CD pipeline will automatically validate your changes and provide feedback through PR comments and status checks.

---

*This CI/CD pipeline is based on the claude-relay-service project patterns and adapted for the Docker Auto Update System. 🚀*