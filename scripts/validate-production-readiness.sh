#!/bin/bash

# Docker Auto Update System - Production Readiness Validation
# Comprehensive validation script to ensure the system is ready for production deployment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
VALIDATION_REPORT="$PROJECT_DIR/production-readiness-report-$(date +%Y%m%d-%H%M%S).txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test result counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_WARNING=0

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$VALIDATION_REPORT"
}

success() {
    echo -e "${GREEN}[✓ PASS]${NC} $1" | tee -a "$VALIDATION_REPORT"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}[✗ FAIL]${NC} $1" | tee -a "$VALIDATION_REPORT"
    ((TESTS_FAILED++))
}

warning() {
    echo -e "${YELLOW}[! WARN]${NC} $1" | tee -a "$VALIDATION_REPORT"
    ((TESTS_WARNING++))
}

# Initialize validation report
init_validation_report() {
    {
        echo "======================================"
        echo "Docker Auto Update System"
        echo "Production Readiness Validation Report"
        echo "======================================"
        echo ""
        echo "Generated: $(date)"
        echo "System: $(uname -a)"
        echo "User: $(whoami)"
        echo "Directory: $PROJECT_DIR"
        echo ""
    } > "$VALIDATION_REPORT"

    log "Starting production readiness validation"
}

# Test 1: Docker Environment Validation
validate_docker_environment() {
    log "=== Docker Environment Validation ==="

    # Check Docker installation
    if command -v docker &> /dev/null; then
        local docker_version=$(docker --version)
        success "Docker is installed: $docker_version"
    else
        fail "Docker is not installed or not in PATH"
        return
    fi

    # Check Docker daemon access
    if docker ps &> /dev/null; then
        success "Docker daemon is accessible"
    else
        fail "Cannot access Docker daemon. Check permissions and daemon status"
    fi

    # Check Docker Compose
    if docker compose version &> /dev/null; then
        local compose_version=$(docker compose version)
        success "Docker Compose is available: $compose_version"
    else
        fail "Docker Compose is not available"
    fi

    # Check available disk space
    local disk_usage=$(df -h "$PROJECT_DIR" | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ $disk_usage -lt 80 ]]; then
        success "Adequate disk space available (${disk_usage}% used)"
    else
        warning "Disk usage is high: ${disk_usage}% used"
    fi

    # Check available memory
    local mem_available=$(free -m | awk 'NR==2{printf "%.0f", $7*100/$2}')
    if [[ $mem_available -gt 20 ]]; then
        success "Sufficient memory available"
    else
        warning "Low memory availability: ${mem_available}%"
    fi
}

# Test 2: File Structure Validation
validate_file_structure() {
    log "=== File Structure Validation ==="

    # Check critical files exist
    local required_files=(
        "Dockerfile"
        "Dockerfile.production"
        "docker-compose.yml"
        "docker-compose.production.yml"
        ".env.production.example"
        "backend/cmd/server/main.go"
        "frontend/package.json"
    )

    for file in "${required_files[@]}"; do
        if [[ -f "$PROJECT_DIR/$file" ]]; then
            success "Required file exists: $file"
        else
            fail "Missing required file: $file"
        fi
    done

    # Check directory structure
    local required_dirs=(
        "backend/cmd/server"
        "backend/internal"
        "backend/pkg"
        "frontend/src"
        "scripts"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ -d "$PROJECT_DIR/$dir" ]]; then
            success "Required directory exists: $dir"
        else
            fail "Missing required directory: $dir"
        fi
    done

    # Check for sensitive files that shouldn't be in production
    local sensitive_files=(
        ".env"
        ".env.local"
        ".env.development"
        "*.key"
        "*.pem"
    )

    for pattern in "${sensitive_files[@]}"; do
        if find "$PROJECT_DIR" -name "$pattern" -type f | grep -q .; then
            warning "Potentially sensitive files found matching: $pattern"
        fi
    done
}

# Test 3: Build System Validation
validate_build_system() {
    log "=== Build System Validation ==="

    # Validate Go backend build
    if [[ -d "$PROJECT_DIR/backend" ]]; then
        cd "$PROJECT_DIR/backend"

        # Check Go version
        if command -v go &> /dev/null; then
            local go_version=$(go version)
            success "Go is available: $go_version"

            # Test Go modules
            if go mod verify; then
                success "Go modules are valid"
            else
                fail "Go modules verification failed"
            fi

            # Test compilation
            if CGO_ENABLED=0 go build -o /tmp/test-build ./cmd/server; then
                success "Go backend compiles successfully"
                rm -f /tmp/test-build
            else
                fail "Go backend compilation failed"
            fi
        else
            fail "Go is not installed or not in PATH"
        fi

        cd "$PROJECT_DIR"
    fi

    # Validate frontend build
    if [[ -d "$PROJECT_DIR/frontend" ]]; then
        cd "$PROJECT_DIR/frontend"

        # Check Node.js
        if command -v node &> /dev/null; then
            local node_version=$(node --version)
            success "Node.js is available: $node_version"
        else
            fail "Node.js is not installed or not in PATH"
        fi

        # Check npm
        if command -v npm &> /dev/null; then
            local npm_version=$(npm --version)
            success "npm is available: $npm_version"
        else
            fail "npm is not installed or not in PATH"
        fi

        # Check package.json
        if [[ -f "package.json" ]]; then
            success "Frontend package.json exists"

            # Validate dependencies
            if npm ls --production --parseable &> /dev/null; then
                success "Frontend production dependencies are valid"
            else
                warning "Some frontend dependencies may have issues"
            fi
        else
            fail "Frontend package.json is missing"
        fi

        cd "$PROJECT_DIR"
    fi
}

# Test 4: Configuration Validation
validate_configuration() {
    log "=== Configuration Validation ==="

    # Check Docker Compose files
    local compose_files=(
        "docker-compose.yml"
        "docker-compose.production.yml"
    )

    for compose_file in "${compose_files[@]}"; do
        if [[ -f "$PROJECT_DIR/$compose_file" ]]; then
            if docker compose -f "$PROJECT_DIR/$compose_file" config --quiet; then
                success "Docker Compose file is valid: $compose_file"
            else
                fail "Docker Compose file has errors: $compose_file"
            fi
        fi
    done

    # Check Dockerfile syntax
    local dockerfiles=(
        "Dockerfile"
        "Dockerfile.production"
    )

    for dockerfile in "${dockerfiles[@]}"; do
        if [[ -f "$PROJECT_DIR/$dockerfile" ]]; then
            # Basic syntax check
            if grep -q "FROM" "$PROJECT_DIR/$dockerfile" && grep -q "CMD\|ENTRYPOINT" "$PROJECT_DIR/$dockerfile"; then
                success "Dockerfile syntax appears valid: $dockerfile"
            else
                warning "Dockerfile may have syntax issues: $dockerfile"
            fi
        fi
    done

    # Check environment configuration
    if [[ -f "$PROJECT_DIR/.env.production.example" ]]; then
        success "Production environment example file exists"

        # Check for placeholder values that need to be changed
        if grep -q "change-me\|your-domain\|password" "$PROJECT_DIR/.env.production.example"; then
            warning "Environment file contains placeholder values that should be customized"
        fi
    else
        fail "Production environment example file is missing"
    fi
}

# Test 5: Security Validation
validate_security() {
    log "=== Security Validation ==="

    # Check for hardcoded secrets
    log "Checking for potential hardcoded secrets..."

    local secret_patterns=(
        "password.*=.*[^{]"
        "secret.*=.*[^{]"
        "key.*=.*[^{]"
        "token.*=.*[^{]"
    )

    local found_secrets=false
    for pattern in "${secret_patterns[@]}"; do
        if find "$PROJECT_DIR" -type f -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.vue" | \
           xargs grep -i "$pattern" 2>/dev/null | grep -v "example\|template\|placeholder" | head -5; then
            found_secrets=true
        fi
    done

    if [[ "$found_secrets" == "false" ]]; then
        success "No obvious hardcoded secrets found"
    else
        fail "Potential hardcoded secrets detected - please review above results"
    fi

    # Check Dockerfile security practices
    if [[ -f "$PROJECT_DIR/Dockerfile.production" ]]; then
        local security_checks=0
        local security_total=0

        # Check for non-root user
        ((security_total++))
        if grep -q "USER" "$PROJECT_DIR/Dockerfile.production"; then
            success "Dockerfile uses non-root user"
            ((security_checks++))
        else
            fail "Dockerfile should specify non-root user"
        fi

        # Check for HEALTHCHECK
        ((security_total++))
        if grep -q "HEALTHCHECK" "$PROJECT_DIR/Dockerfile.production"; then
            success "Dockerfile includes health check"
            ((security_checks++))
        else
            warning "Dockerfile should include health check"
        fi

        # Check for minimal base image
        ((security_total++))
        if grep -q "alpine\|scratch\|distroless" "$PROJECT_DIR/Dockerfile.production"; then
            success "Dockerfile uses minimal base image"
            ((security_checks++))
        else
            warning "Consider using minimal base image for security"
        fi

        local security_score=$((security_checks * 100 / security_total))
        log "Security score: ${security_score}%"
    fi
}

# Test 6: Performance Validation
validate_performance() {
    log "=== Performance Validation ==="

    # Check if production builds exist
    if [[ -f "$PROJECT_DIR/frontend/dist/index.html" ]]; then
        success "Frontend production build exists"

        # Check bundle size
        local bundle_size=$(du -sh "$PROJECT_DIR/frontend/dist" | cut -f1)
        log "Frontend bundle size: $bundle_size"

        if [[ $(du -s "$PROJECT_DIR/frontend/dist" | cut -f1) -lt 10240 ]]; then  # Less than 10MB
            success "Frontend bundle size is reasonable"
        else
            warning "Frontend bundle size is large - consider optimization"
        fi
    else
        fail "Frontend production build not found - run 'npm run build:prod' in frontend directory"
    fi

    # Check Go binary if it exists
    if [[ -f "$PROJECT_DIR/backend/docker-auto-server-cgo" ]]; then
        success "Go production binary exists"

        local binary_size=$(du -sh "$PROJECT_DIR/backend/docker-auto-server-cgo" | cut -f1)
        log "Go binary size: $binary_size"
    else
        warning "Go production binary not found - consider building for validation"
    fi

    # Check for static asset optimization
    if [[ -d "$PROJECT_DIR/frontend/dist" ]]; then
        local uncompressed_assets=$(find "$PROJECT_DIR/frontend/dist" -name "*.js" -o -name "*.css" | wc -l)
        local gzipped_assets=$(find "$PROJECT_DIR/frontend/dist" -name "*.gz" | wc -l)

        if [[ $gzipped_assets -gt 0 ]]; then
            success "Static assets appear to be compressed"
        else
            warning "Static assets may benefit from compression"
        fi
    fi
}

# Test 7: Monitoring and Observability
validate_monitoring() {
    log "=== Monitoring and Observability Validation ==="

    # Check for health check endpoints
    local health_patterns=(
        "/health"
        "/api/health"
        "healthcheck"
    )

    local health_found=false
    for pattern in "${health_patterns[@]}"; do
        if find "$PROJECT_DIR" -type f -name "*.go" -o -name "*.js" -o -name "*.ts" | \
           xargs grep -l "$pattern" &> /dev/null; then
            health_found=true
            break
        fi
    done

    if [[ "$health_found" == "true" ]]; then
        success "Health check endpoints found in code"
    else
        fail "No health check endpoints found"
    fi

    # Check for logging configuration
    if find "$PROJECT_DIR" -name "*.go" | xargs grep -l "logrus\|log\|logger" &> /dev/null; then
        success "Logging framework detected"
    else
        warning "No logging framework detected"
    fi

    # Check for metrics endpoints
    if find "$PROJECT_DIR" -name "*.go" | xargs grep -l "metrics\|prometheus" &> /dev/null; then
        success "Metrics/monitoring code detected"
    else
        warning "No metrics/monitoring code detected"
    fi
}

# Test 8: Deployment Readiness
validate_deployment() {
    log "=== Deployment Readiness Validation ==="

    # Check for deployment scripts
    local deployment_scripts=(
        "scripts/deploy-production.sh"
        "scripts/load-test.sh"
    )

    for script in "${deployment_scripts[@]}"; do
        if [[ -f "$PROJECT_DIR/$script" ]]; then
            if [[ -x "$PROJECT_DIR/$script" ]]; then
                success "Deployment script exists and is executable: $script"
            else
                warning "Deployment script exists but is not executable: $script"
            fi
        else
            warning "Deployment script not found: $script"
        fi
    done

    # Check for backup procedures
    if grep -r "backup\|restore" "$PROJECT_DIR/scripts/" &> /dev/null; then
        success "Backup/restore procedures found in scripts"
    else
        warning "No backup/restore procedures detected"
    fi

    # Check for rollback capabilities
    if grep -r "rollback\|revert" "$PROJECT_DIR/scripts/" &> /dev/null; then
        success "Rollback capabilities found"
    else
        warning "No rollback procedures detected"
    fi
}

# Generate final report
generate_final_report() {
    log "=== Validation Summary ==="

    local total_tests=$((TESTS_PASSED + TESTS_FAILED + TESTS_WARNING))

    {
        echo ""
        echo "======================================"
        echo "PRODUCTION READINESS SUMMARY"
        echo "======================================"
        echo ""
        echo "Total Tests: $total_tests"
        echo "Passed: $TESTS_PASSED"
        echo "Failed: $TESTS_FAILED"
        echo "Warnings: $TESTS_WARNING"
        echo ""

        local pass_rate=0
        if [[ $total_tests -gt 0 ]]; then
            pass_rate=$((TESTS_PASSED * 100 / total_tests))
        fi

        echo "Pass Rate: ${pass_rate}%"
        echo ""

        if [[ $TESTS_FAILED -eq 0 ]] && [[ $pass_rate -ge 80 ]]; then
            echo "✅ SYSTEM IS READY FOR PRODUCTION DEPLOYMENT"
        elif [[ $TESTS_FAILED -eq 0 ]] && [[ $pass_rate -ge 60 ]]; then
            echo "⚠️  SYSTEM NEEDS MINOR IMPROVEMENTS BEFORE PRODUCTION"
        else
            echo "❌ SYSTEM REQUIRES SIGNIFICANT WORK BEFORE PRODUCTION"
        fi

        echo ""
        echo "Recommendations:"
        if [[ $TESTS_FAILED -gt 0 ]]; then
            echo "  - Address all failed tests before deployment"
        fi
        if [[ $TESTS_WARNING -gt 0 ]]; then
            echo "  - Review and address warning items"
        fi
        echo "  - Perform load testing with realistic traffic"
        echo "  - Set up monitoring and alerting"
        echo "  - Prepare incident response procedures"
        echo "  - Schedule regular security updates"
        echo ""

    } >> "$VALIDATION_REPORT"

    log "Validation completed. Full report: $VALIDATION_REPORT"

    # Display summary on console
    echo ""
    echo "======================================"
    echo "PRODUCTION READINESS VALIDATION COMPLETE"
    echo "======================================"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Warnings: $TESTS_WARNING"
    echo "Pass Rate: ${pass_rate}%"
    echo ""
    echo "Full report: $VALIDATION_REPORT"
    echo "======================================"
}

# Main execution
main() {
    init_validation_report

    validate_docker_environment
    validate_file_structure
    validate_build_system
    validate_configuration
    validate_security
    validate_performance
    validate_monitoring
    validate_deployment

    generate_final_report

    # Return appropriate exit code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"