#!/bin/bash

# Docker Auto Update System - Load Testing Script
# Tests the application under various load conditions

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BASE_URL="${BASE_URL:-http://localhost:8080}"
RESULTS_DIR="$PROJECT_DIR/test-results/load-testing"
REPORT_FILE="$RESULTS_DIR/load-test-report-$(date +%Y%m%d-%H%M%S).txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$REPORT_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$REPORT_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$REPORT_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$REPORT_FILE"
}

# Setup test environment
setup_test_environment() {
    log "Setting up load testing environment..."

    mkdir -p "$RESULTS_DIR"

    # Check if application is running
    if ! curl -f -s "$BASE_URL/api/health" > /dev/null; then
        error "Application is not running at $BASE_URL"
        exit 1
    fi

    success "Test environment setup completed"
}

# Test 1: Basic health check load
test_health_endpoint() {
    log "Test 1: Health endpoint load test..."

    local concurrent_users=50
    local duration=60
    local test_name="health_endpoint"

    log "Running health endpoint test with $concurrent_users concurrent users for ${duration}s"

    # Use curl with GNU parallel if available, otherwise use a loop
    if command -v parallel &> /dev/null; then
        seq 1 $((concurrent_users * duration)) | \
        parallel -j $concurrent_users --delay 1 \
            "curl -s -o /dev/null -w '%{http_code},%{time_total},%{time_connect},%{time_starttransfer}\\n' $BASE_URL/api/health" > \
            "$RESULTS_DIR/${test_name}_results.csv"
    else
        # Fallback method using background processes
        for i in $(seq 1 $concurrent_users); do
            {
                for j in $(seq 1 $duration); do
                    curl -s -o /dev/null -w '%{http_code},%{time_total},%{time_connect},%{time_starttransfer}\n' \
                        "$BASE_URL/api/health"
                    sleep 1
                done
            } > "$RESULTS_DIR/${test_name}_user_${i}.csv" &
        done

        # Wait for all background processes
        wait

        # Combine results
        cat "$RESULTS_DIR/${test_name}_user_"*.csv > "$RESULTS_DIR/${test_name}_results.csv"
        rm "$RESULTS_DIR/${test_name}_user_"*.csv
    fi

    # Analyze results
    analyze_results "$test_name" "$RESULTS_DIR/${test_name}_results.csv"
}

# Test 2: API endpoints stress test
test_api_endpoints() {
    log "Test 2: API endpoints stress test..."

    local endpoints=(
        "/api/health"
        "/api/system/info"
        "/api/containers"
        "/api/images"
    )

    for endpoint in "${endpoints[@]}"; do
        local test_name="api_$(echo ${endpoint##*/} | tr '/' '_')"
        log "Testing endpoint: $endpoint"

        # Simple stress test with 20 concurrent requests for 30 seconds
        {
            for i in $(seq 1 20); do
                {
                    for j in $(seq 1 30); do
                        curl -s -o /dev/null -w '%{http_code},%{time_total}\n' \
                            -H "Accept: application/json" \
                            "$BASE_URL$endpoint" || echo "000,0"
                        sleep 1
                    done
                } &
            done
            wait
        } > "$RESULTS_DIR/${test_name}_results.csv"

        log "Completed test for $endpoint"
    done

    success "API endpoints stress test completed"
}

# Test 3: WebSocket connection test
test_websocket_connections() {
    log "Test 3: WebSocket connections test..."

    # Test multiple WebSocket connections (if wscat is available)
    if command -v wscat &> /dev/null; then
        local ws_url="ws://localhost:8080/api/ws"
        local num_connections=10

        log "Testing $num_connections concurrent WebSocket connections"

        for i in $(seq 1 $num_connections); do
            {
                timeout 30 wscat -c "$ws_url" --execute 'ping' > \
                    "$RESULTS_DIR/websocket_${i}.log" 2>&1 || true
            } &
        done

        wait
        success "WebSocket connections test completed"
    else
        warning "wscat not available, skipping WebSocket test"
    fi
}

# Test 4: Memory and CPU usage monitoring
monitor_resource_usage() {
    log "Test 4: Resource usage monitoring..."

    local duration=120
    local container_name="docker-auto-prod"

    if docker ps --filter "name=$container_name" --format "table {{.Names}}" | grep -q "$container_name"; then
        log "Monitoring resource usage for $duration seconds"

        # Monitor CPU and memory usage
        {
            for i in $(seq 1 $duration); do
                docker stats "$container_name" --no-stream --format \
                    "table {{.MemUsage}},{{.CPUPerc}},{{.MemPerc}},{{.NetIO}},{{.BlockIO}}" | \
                    tail -n 1
                sleep 1
            done
        } > "$RESULTS_DIR/resource_usage.csv"

        success "Resource monitoring completed"
    else
        warning "Container $container_name not found, skipping resource monitoring"
    fi
}

# Test 5: Database connection pool test
test_database_connections() {
    log "Test 5: Database connection pool test..."

    # Test concurrent database operations through API
    local concurrent_requests=25
    local requests_per_user=20

    log "Testing database with $concurrent_requests concurrent users"

    for i in $(seq 1 $concurrent_requests); do
        {
            for j in $(seq 1 $requests_per_user); do
                # Test endpoints that involve database queries
                curl -s -o /dev/null -w '%{http_code},%{time_total}\n' \
                    "$BASE_URL/api/containers" || echo "000,0"
                sleep 0.5
            done
        } > "$RESULTS_DIR/db_test_user_${i}.csv" &
    done

    wait

    # Combine results
    cat "$RESULTS_DIR/db_test_user_"*.csv > "$RESULTS_DIR/database_stress_results.csv"
    rm "$RESULTS_DIR/db_test_user_"*.csv

    success "Database connection pool test completed"
}

# Analyze test results
analyze_results() {
    local test_name="$1"
    local results_file="$2"

    if [[ ! -f "$results_file" ]]; then
        warning "Results file not found: $results_file"
        return
    fi

    log "Analyzing results for $test_name..."

    # Count total requests
    local total_requests=$(wc -l < "$results_file")

    # Count successful requests (HTTP 200)
    local successful_requests=$(grep -c "^200," "$results_file" || echo "0")

    # Count failed requests
    local failed_requests=$((total_requests - successful_requests))

    # Calculate success rate
    local success_rate=0
    if [[ $total_requests -gt 0 ]]; then
        success_rate=$(( (successful_requests * 100) / total_requests ))
    fi

    # Calculate average response time (for successful requests)
    local avg_response_time="0"
    if [[ $successful_requests -gt 0 ]]; then
        avg_response_time=$(grep "^200," "$results_file" | \
            cut -d',' -f2 | \
            awk '{sum+=$1; count++} END {if(count>0) print sum/count; else print 0}')
    fi

    # Log results
    log "Results for $test_name:"
    log "  Total requests: $total_requests"
    log "  Successful requests: $successful_requests"
    log "  Failed requests: $failed_requests"
    log "  Success rate: ${success_rate}%"
    log "  Average response time: ${avg_response_time}s"

    # Determine if test passed
    if [[ $success_rate -ge 95 ]] && (( $(echo "$avg_response_time < 2.0" | bc -l) )); then
        success "$test_name PASSED (Success rate: ${success_rate}%, Avg response: ${avg_response_time}s)"
    else
        warning "$test_name NEEDS ATTENTION (Success rate: ${success_rate}%, Avg response: ${avg_response_time}s)"
    fi
}

# Generate performance report
generate_performance_report() {
    log "Generating performance report..."

    local summary_file="$RESULTS_DIR/performance_summary.txt"

    {
        echo "======================================"
        echo "Docker Auto Update System - Performance Report"
        echo "Generated: $(date)"
        echo "Test Duration: $(date --date='1 hour ago') to $(date)"
        echo "======================================"
        echo ""

        echo "System Information:"
        echo "  OS: $(uname -o)"
        echo "  Kernel: $(uname -r)"
        echo "  CPU: $(nproc) cores"
        echo "  Memory: $(free -h | awk '/^Mem:/ {print $2}') total"
        echo ""

        echo "Docker Information:"
        docker --version
        docker compose version
        echo ""

        echo "Container Status:"
        docker compose -f "$PROJECT_DIR/docker-compose.production.yml" ps || echo "Production containers not running"
        echo ""

        echo "Test Results Summary:"
        echo "  Health Endpoint: See health_endpoint_results.csv"
        echo "  API Endpoints: See api_*_results.csv files"
        echo "  WebSocket: See websocket_*.log files"
        echo "  Resource Usage: See resource_usage.csv"
        echo "  Database Load: See database_stress_results.csv"
        echo ""

        echo "Recommendations:"
        echo "  - Monitor CPU usage during peak load"
        echo "  - Consider connection pooling optimization if needed"
        echo "  - Review slow endpoints (>2s response time)"
        echo "  - Monitor memory usage trends"
        echo ""

    } > "$summary_file"

    success "Performance report generated: $summary_file"
}

# Cleanup function
cleanup() {
    log "Cleaning up load testing processes..."
    # Kill any remaining background processes
    pkill -f "curl.*$BASE_URL" 2>/dev/null || true
    pkill -f "wscat" 2>/dev/null || true
}

# Main function
main() {
    log "Starting Docker Auto Update System Load Testing"

    # Set up cleanup on exit
    trap cleanup EXIT

    setup_test_environment

    # Run load tests
    test_health_endpoint
    test_api_endpoints
    test_websocket_connections
    monitor_resource_usage
    test_database_connections

    # Generate reports
    generate_performance_report

    success "Load testing completed successfully!"
    log "Results are available in: $RESULTS_DIR"
}

# Check if bc calculator is available (needed for floating point comparison)
if ! command -v bc &> /dev/null; then
    warning "bc calculator not available, some calculations may be limited"
fi

# Run main function
main "$@"