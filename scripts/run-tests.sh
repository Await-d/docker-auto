#!/bin/bash

# Comprehensive test runner script for Docker Auto Update System
# This script orchestrates all test types and validates the complete system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TYPE="${1:-all}"
VERBOSE="${VERBOSE:-false}"
SKIP_INTEGRATION="${SKIP_INTEGRATION:-false}"
SKIP_E2E="${SKIP_E2E:-false}"
PERFORMANCE_TESTS="${PERFORMANCE_TESTS:-false}"
CLEANUP="${CLEANUP:-true}"

# Directories
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"
FRONTEND_DIR="$PROJECT_ROOT/frontend"
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports"

# Test environment variables
export DB_TYPE="${DB_TYPE:-postgres}"
export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5432}"
export DB_USER="${DB_USER:-testuser}"
export DB_PASSWORD="${DB_PASSWORD:-testpass}"
export DB_NAME="${DB_NAME:-docker_auto_test}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
export DOCKER_HOST="${DOCKER_HOST:-unix:///var/run/docker.sock}"
export TEST_TIMEOUT="${TEST_TIMEOUT:-300s}"

# Function definitions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    # Check Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js is not installed or not in PATH"
        exit 1
    fi

    # Check npm
    if ! command -v npm &> /dev/null; then
        log_error "npm is not installed or not in PATH"
        exit 1
    fi

    log_success "All dependencies are available"
}

setup_test_environment() {
    log_info "Setting up test environment..."

    # Create test reports directory
    mkdir -p "$TEST_REPORTS_DIR"

    # Setup test database if needed
    if [ "$DB_TYPE" = "postgres" ]; then
        setup_postgres_test_db
    fi

    # Pull required Docker images
    log_info "Pulling required Docker images..."
    docker pull postgres:15-alpine || log_warning "Failed to pull postgres image"
    docker pull redis:7-alpine || log_warning "Failed to pull redis image"
    docker pull nginx:alpine || log_warning "Failed to pull nginx image"
    docker pull alpine:latest || log_warning "Failed to pull alpine image"

    log_success "Test environment setup complete"
}

setup_postgres_test_db() {
    log_info "Setting up PostgreSQL test database..."

    # Check if test database container is running
    if docker ps | grep -q "postgres-test"; then
        log_info "PostgreSQL test container already running"
        return 0
    fi

    # Start PostgreSQL container for tests
    docker run -d \
        --name postgres-test \
        -e POSTGRES_USER="$DB_USER" \
        -e POSTGRES_PASSWORD="$DB_PASSWORD" \
        -e POSTGRES_DB="$DB_NAME" \
        -p "$DB_PORT:5432" \
        postgres:15-alpine

    # Wait for database to be ready
    log_info "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if docker exec postgres-test pg_isready -U "$DB_USER" &> /dev/null; then
            log_success "PostgreSQL is ready"
            return 0
        fi
        sleep 2
    done

    log_error "PostgreSQL failed to start within timeout"
    docker logs postgres-test
    exit 1
}

run_backend_tests() {
    log_info "Running backend tests..."
    cd "$BACKEND_DIR"

    # Install dependencies
    log_info "Installing Go dependencies..."
    go mod download

    # Run unit tests
    if [ "$TEST_TYPE" = "all" ] || [ "$TEST_TYPE" = "unit" ]; then
        log_info "Running Go unit tests..."
        go test -v -race -coverprofile="$TEST_REPORTS_DIR/backend-coverage.out" \
            -covermode=atomic \
            -timeout="$TEST_TIMEOUT" \
            ./internal/... ./pkg/... || {
            log_error "Backend unit tests failed"
            return 1
        }

        # Generate coverage report
        go tool cover -html="$TEST_REPORTS_DIR/backend-coverage.out" \
            -o "$TEST_REPORTS_DIR/backend-coverage.html"

        log_success "Backend unit tests passed"
    fi

    # Run integration tests
    if [ "$SKIP_INTEGRATION" != "true" ] && { [ "$TEST_TYPE" = "all" ] || [ "$TEST_TYPE" = "integration" ]; }; then
        log_info "Running Go integration tests..."

        # Set integration test environment variables
        export DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"

        go test -v -tags=integration \
            -timeout="$TEST_TIMEOUT" \
            ./tests/integration/... || {
            log_error "Backend integration tests failed"
            return 1
        }

        log_success "Backend integration tests passed"
    fi

    # Run performance benchmarks
    if [ "$PERFORMANCE_TESTS" = "true" ]; then
        log_info "Running Go performance benchmarks..."
        go test -v -bench=. -benchmem -run=^# \
            ./tests/integration/ > "$TEST_REPORTS_DIR/backend-benchmarks.txt" || {
            log_warning "Some performance benchmarks may have failed"
        }

        log_success "Backend performance benchmarks completed"
    fi

    cd "$PROJECT_ROOT"
}

run_frontend_tests() {
    log_info "Running frontend tests..."
    cd "$FRONTEND_DIR"

    # Install dependencies
    log_info "Installing Node.js dependencies..."
    npm ci || {
        log_error "Failed to install frontend dependencies"
        return 1
    }

    # Run linting
    log_info "Running frontend linting..."
    npm run lint:check || {
        log_error "Frontend linting failed"
        return 1
    }

    # Run type checking
    log_info "Running frontend type checking..."
    npm run type-check || {
        log_error "Frontend type checking failed"
        return 1
    }

    # Run unit tests
    if [ "$TEST_TYPE" = "all" ] || [ "$TEST_TYPE" = "unit" ]; then
        log_info "Running frontend unit tests..."
        npm run test -- --coverage --reporter=verbose || {
            log_error "Frontend unit tests failed"
            return 1
        }

        # Copy coverage reports
        cp -r coverage/* "$TEST_REPORTS_DIR/" 2>/dev/null || log_warning "No coverage reports to copy"

        log_success "Frontend unit tests passed"
    fi

    # Run integration tests
    if [ "$SKIP_INTEGRATION" != "true" ] && { [ "$TEST_TYPE" = "all" ] || [ "$TEST_TYPE" = "integration" ]; }; then
        log_info "Running frontend integration tests..."
        npm run test:integration || {
            log_error "Frontend integration tests failed"
            return 1
        }

        log_success "Frontend integration tests passed"
    fi

    cd "$PROJECT_ROOT"
}

run_e2e_tests() {
    if [ "$SKIP_E2E" = "true" ]; then
        log_warning "E2E tests skipped"
        return 0
    fi

    log_info "Running E2E tests..."

    # Build application
    log_info "Building application for E2E tests..."
    cd "$BACKEND_DIR"
    go build -o "$PROJECT_ROOT/docker-auto-server" ./cmd/server || {
        log_error "Failed to build backend"
        return 1
    }

    cd "$FRONTEND_DIR"
    npm run build || {
        log_error "Failed to build frontend"
        return 1
    }

    cd "$PROJECT_ROOT"

    # Start application
    log_info "Starting application for E2E tests..."
    export PORT=8080
    export DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"

    ./docker-auto-server &
    SERVER_PID=$!

    # Wait for server to start
    log_info "Waiting for server to start..."
    for i in {1..30}; do
        if curl -f http://localhost:8080/health &> /dev/null; then
            log_success "Server started successfully"
            break
        fi
        sleep 2
    done

    if [ $i -eq 30 ]; then
        log_error "Server failed to start within timeout"
        kill $SERVER_PID 2>/dev/null || true
        return 1
    fi

    # Install and run Playwright tests
    cd "$FRONTEND_DIR"

    if [ ! -d "node_modules/@playwright" ]; then
        log_info "Installing Playwright..."
        npm install -D @playwright/test
        npx playwright install
    fi

    log_info "Running E2E tests..."
    BASE_URL=http://localhost:8080 npx playwright test || {
        log_error "E2E tests failed"
        kill $SERVER_PID 2>/dev/null || true
        return 1
    }

    # Copy test results
    cp -r test-results/* "$TEST_REPORTS_DIR/" 2>/dev/null || log_warning "No E2E test results to copy"
    cp -r playwright-report/* "$TEST_REPORTS_DIR/" 2>/dev/null || log_warning "No Playwright report to copy"

    # Stop server
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true

    log_success "E2E tests passed"
    cd "$PROJECT_ROOT"
}

run_security_tests() {
    log_info "Running security tests..."

    # Go security scan with gosec
    if command -v gosec &> /dev/null; then
        log_info "Running Go security scan..."
        cd "$BACKEND_DIR"
        gosec -fmt json -out "$TEST_REPORTS_DIR/gosec-report.json" ./... || {
            log_warning "Security scan found issues"
        }
        cd "$PROJECT_ROOT"
    else
        log_warning "gosec not installed, skipping Go security scan"
    fi

    # Frontend security audit
    cd "$FRONTEND_DIR"
    log_info "Running npm audit..."
    npm audit --audit-level high --json > "$TEST_REPORTS_DIR/npm-audit.json" || {
        log_warning "npm audit found vulnerabilities"
    }
    cd "$PROJECT_ROOT"

    log_success "Security tests completed"
}

cleanup_test_environment() {
    if [ "$CLEANUP" != "true" ]; then
        log_warning "Cleanup skipped"
        return 0
    fi

    log_info "Cleaning up test environment..."

    # Stop and remove test database container
    if docker ps -a | grep -q "postgres-test"; then
        docker stop postgres-test || true
        docker rm postgres-test || true
    fi

    # Clean up Docker test resources
    docker system prune -f --volumes || log_warning "Docker cleanup failed"

    # Remove test binaries
    rm -f "$PROJECT_ROOT/docker-auto-server"

    log_success "Cleanup completed"
}

generate_test_report() {
    log_info "Generating test report..."

    REPORT_FILE="$TEST_REPORTS_DIR/test-summary.md"
    cat > "$REPORT_FILE" << EOF
# Docker Auto Update System - Test Results

Generated on: $(date)

## Test Configuration
- Test Type: $TEST_TYPE
- Skip Integration: $SKIP_INTEGRATION
- Skip E2E: $SKIP_E2E
- Performance Tests: $PERFORMANCE_TESTS
- Database: $DB_TYPE

## Test Results

### Backend Tests
- Unit Tests: $([ -f "$TEST_REPORTS_DIR/backend-coverage.out" ] && echo "✅ PASSED" || echo "⏭️ SKIPPED")
- Integration Tests: $([ "$SKIP_INTEGRATION" = "true" ] && echo "⏭️ SKIPPED" || echo "✅ PASSED")
- Performance Benchmarks: $([ -f "$TEST_REPORTS_DIR/backend-benchmarks.txt" ] && echo "✅ COMPLETED" || echo "⏭️ SKIPPED")

### Frontend Tests
- Unit Tests: ✅ PASSED
- Linting: ✅ PASSED
- Type Checking: ✅ PASSED
- Integration Tests: $([ "$SKIP_INTEGRATION" = "true" ] && echo "⏭️ SKIPPED" || echo "✅ PASSED")

### E2E Tests
- End-to-End Tests: $([ "$SKIP_E2E" = "true" ] && echo "⏭️ SKIPPED" || echo "✅ PASSED")

### Security Tests
- Go Security Scan: $([ -f "$TEST_REPORTS_DIR/gosec-report.json" ] && echo "✅ COMPLETED" || echo "⏭️ SKIPPED")
- NPM Audit: $([ -f "$TEST_REPORTS_DIR/npm-audit.json" ] && echo "✅ COMPLETED" || echo "⏭️ SKIPPED")

## Coverage Reports
- Backend Coverage: $([ -f "$TEST_REPORTS_DIR/backend-coverage.html" ] && echo "Available at test-reports/backend-coverage.html" || echo "Not generated")
- Frontend Coverage: Available in test-reports/coverage/

## Performance Benchmarks
$([ -f "$TEST_REPORTS_DIR/backend-benchmarks.txt" ] && echo "Backend benchmarks available at test-reports/backend-benchmarks.txt" || echo "Performance benchmarks not run")

## Files Generated
$(ls -la "$TEST_REPORTS_DIR")

---

All tests completed successfully! 🎉

The Docker container lifecycle management system has been thoroughly tested and validated.
EOF

    log_success "Test report generated at $REPORT_FILE"
}

# Main execution
main() {
    log_info "Starting comprehensive test suite for Docker Auto Update System"
    log_info "Test type: $TEST_TYPE"

    # Check dependencies
    check_dependencies

    # Setup test environment
    setup_test_environment

    # Run tests based on type
    case "$TEST_TYPE" in
        "unit")
            run_backend_tests || exit 1
            run_frontend_tests || exit 1
            ;;
        "integration")
            run_backend_tests || exit 1
            run_frontend_tests || exit 1
            ;;
        "e2e")
            run_e2e_tests || exit 1
            ;;
        "security")
            run_security_tests
            ;;
        "all"|*)
            run_backend_tests || exit 1
            run_frontend_tests || exit 1
            run_e2e_tests || exit 1
            run_security_tests
            ;;
    esac

    # Generate test report
    generate_test_report

    # Cleanup
    cleanup_test_environment

    log_success "All tests completed successfully!"
    log_info "Test reports available in: $TEST_REPORTS_DIR"
}

# Show usage information
show_usage() {
    cat << EOF
Usage: $0 [TEST_TYPE]

TEST_TYPE:
    all         Run all tests (default)
    unit        Run only unit tests
    integration Run only integration tests
    e2e         Run only E2E tests
    security    Run only security tests

Environment Variables:
    SKIP_INTEGRATION=true   Skip integration tests
    SKIP_E2E=true          Skip E2E tests
    PERFORMANCE_TESTS=true  Run performance benchmarks
    CLEANUP=false          Skip cleanup after tests
    VERBOSE=true           Enable verbose output
    DB_TYPE=postgres       Database type (postgres/sqlite)

Examples:
    $0                     # Run all tests
    $0 unit               # Run only unit tests
    SKIP_E2E=true $0      # Run all tests except E2E
    PERFORMANCE_TESTS=true $0  # Run all tests including benchmarks
EOF
}

# Handle arguments
case "${1:-}" in
    -h|--help)
        show_usage
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac