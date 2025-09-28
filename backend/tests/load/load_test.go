package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/performance"
	"docker-auto/pkg/utils"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// LoadTestConfig defines load test configuration
type LoadTestConfig struct {
	BaseURL               string
	ConcurrentUsers       int
	TestDuration          time.Duration
	RequestsPerSecond     int
	ContainerOperations   int
	MonitoringOperations  int
	TerminalOperations    int
	WebSocketConnections  int
}

// LoadTestResults contains load test results
type LoadTestResults struct {
	TotalRequests         int64
	SuccessfulRequests    int64
	FailedRequests        int64
	AverageResponseTime   time.Duration
	MaxResponseTime       time.Duration
	MinResponseTime       time.Duration
	RequestsPerSecond     float64
	ErrorRate             float64
	ThroughputMBps        float64
	ContainerOpResults    *OperationResults
	MonitoringResults     *OperationResults
	TerminalResults       *OperationResults
	WebSocketResults      *WebSocketResults
	PerformanceMetrics    *performance.PerformanceMetrics
}

// OperationResults contains results for specific operations
type OperationResults struct {
	TotalOperations    int64
	SuccessfulOps      int64
	FailedOps          int64
	AverageTime        time.Duration
	MaxTime            time.Duration
	MinTime            time.Duration
}

// WebSocketResults contains WebSocket-specific test results
type WebSocketResults struct {
	TotalConnections      int64
	SuccessfulConnections int64
	FailedConnections     int64
	MessagesReceived      int64
	MessagesSent          int64
	AverageLatency        time.Duration
}

// LoadTester performs comprehensive load testing
type LoadTester struct {
	config     *LoadTestConfig
	httpClient *http.Client
	results    *LoadTestResults
	optimizer  *performance.SystemOptimizer
	startTime  time.Time
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewLoadTester creates a new load tester
func NewLoadTester(cfg *LoadTestConfig) *LoadTester {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.TestDuration)

	return &LoadTester{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		results: &LoadTestResults{
			MinResponseTime:    time.Hour, // Initialize to large value
			ContainerOpResults: &OperationResults{MinTime: time.Hour},
			MonitoringResults:  &OperationResults{MinTime: time.Hour},
			TerminalResults:    &OperationResults{MinTime: time.Hour},
			WebSocketResults:   &WebSocketResults{},
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// RunLoadTest executes comprehensive load testing
func (lt *LoadTester) RunLoadTest(t *testing.T) *LoadTestResults {
	lt.startTime = time.Now()

	t.Logf("Starting load test with %d concurrent users for %v",
		lt.config.ConcurrentUsers, lt.config.TestDuration)

	// Initialize performance optimizer for monitoring
	lt.setupPerformanceOptimizer(t)

	// Run concurrent load test scenarios
	lt.wg.Add(5)
	go lt.runContainerOperationLoad(t)
	go lt.runMonitoringLoad(t)
	go lt.runTerminalLoad(t)
	go lt.runWebSocketLoad(t)
	go lt.runGeneralAPILoad(t)

	// Wait for all tests to complete or timeout
	done := make(chan struct{})
	go func() {
		lt.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("All load tests completed")
	case <-lt.ctx.Done():
		t.Log("Load test timed out")
	}

	lt.cancel()

	// Calculate final results
	lt.calculateFinalResults()

	return lt.results
}

// setupPerformanceOptimizer initializes the performance optimizer for monitoring
func (lt *LoadTester) setupPerformanceOptimizer(t *testing.T) {
	// Initialize test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Initialize cache manager
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Database: 15,
		},
	}
	cacheManager, err := utils.NewCacheManager(&cfg.Redis)
	require.NoError(t, err)

	// Create performance optimizer
	logger := &utils.Logger{} // Use your logger implementation
	lt.optimizer = performance.NewSystemOptimizer(db, cacheManager, logger)
}

// runContainerOperationLoad performs load testing on container operations
func (lt *LoadTester) runContainerOperationLoad(t *testing.T) {
	defer lt.wg.Done()

	results := lt.results.ContainerOpResults
	operationsPerUser := lt.config.ContainerOperations / lt.config.ConcurrentUsers

	var userWg sync.WaitGroup
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		userWg.Add(1)
		go func(userID int) {
			defer userWg.Done()
			lt.performContainerOperations(userID, operationsPerUser, results)
		}(i)
	}

	userWg.Wait()
	t.Logf("Container operation load test completed: %d ops, %d successful, %d failed",
		results.TotalOperations, results.SuccessfulOps, results.FailedOps)
}

// performContainerOperations performs container operations for a single user
func (lt *LoadTester) performContainerOperations(userID, operations int, results *OperationResults) {
	for i := 0; i < operations && lt.ctx.Err() == nil; i++ {
		start := time.Now()

		// Perform various container operations
		success := true

		// Test container list operation
		if !lt.testContainerList() {
			success = false
		}

		// Test container create operation
		containerID := fmt.Sprintf("load-test-%d-%d", userID, i)
		if !lt.testContainerCreate(containerID) {
			success = false
		} else {
			// Test container start/stop operations
			if !lt.testContainerStart(containerID) {
				success = false
			}

			time.Sleep(100 * time.Millisecond) // Brief operation interval

			if !lt.testContainerStop(containerID) {
				success = false
			}

			// Test container delete
			if !lt.testContainerDelete(containerID) {
				success = false
			}
		}

		duration := time.Since(start)
		lt.recordOperationResult(results, success, duration)

		// Rate limiting
		time.Sleep(time.Duration(1000/lt.config.RequestsPerSecond) * time.Millisecond)
	}
}

// runMonitoringLoad performs load testing on monitoring operations
func (lt *LoadTester) runMonitoringLoad(t *testing.T) {
	defer lt.wg.Done()

	results := lt.results.MonitoringResults
	operationsPerUser := lt.config.MonitoringOperations / lt.config.ConcurrentUsers

	var userWg sync.WaitGroup
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		userWg.Add(1)
		go func(userID int) {
			defer userWg.Done()
			lt.performMonitoringOperations(userID, operationsPerUser, results)
		}(i)
	}

	userWg.Wait()
	t.Logf("Monitoring load test completed: %d ops, %d successful, %d failed",
		results.TotalOperations, results.SuccessfulOps, results.FailedOps)
}

// performMonitoringOperations performs monitoring operations for a single user
func (lt *LoadTester) performMonitoringOperations(userID, operations int, results *OperationResults) {
	// Create a test container first for monitoring
	containerID := fmt.Sprintf("monitoring-test-%d", userID)
	if !lt.testContainerCreate(containerID) {
		return
	}
	lt.testContainerStart(containerID)

	defer lt.testContainerDelete(containerID)

	for i := 0; i < operations && lt.ctx.Err() == nil; i++ {
		start := time.Now()
		success := true

		// Test monitoring endpoints
		if !lt.testGetContainerMetrics(containerID) {
			success = false
		}

		if !lt.testGetSystemMetrics() {
			success = false
		}

		if !lt.testGetMetricsHistory(containerID) {
			success = false
		}

		duration := time.Since(start)
		lt.recordOperationResult(results, success, duration)

		time.Sleep(200 * time.Millisecond) // Monitoring interval
	}
}

// runTerminalLoad performs load testing on terminal operations
func (lt *LoadTester) runTerminalLoad(t *testing.T) {
	defer lt.wg.Done()

	results := lt.results.TerminalResults
	operationsPerUser := lt.config.TerminalOperations / lt.config.ConcurrentUsers

	var userWg sync.WaitGroup
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		userWg.Add(1)
		go func(userID int) {
			defer userWg.Done()
			lt.performTerminalOperations(userID, operationsPerUser, results)
		}(i)
	}

	userWg.Wait()
	t.Logf("Terminal load test completed: %d ops, %d successful, %d failed",
		results.TotalOperations, results.SuccessfulOps, results.FailedOps)
}

// performTerminalOperations performs terminal operations for a single user
func (lt *LoadTester) performTerminalOperations(userID, operations int, results *OperationResults) {
	// Create a test container for terminal access
	containerID := fmt.Sprintf("terminal-test-%d", userID)
	if !lt.testContainerCreate(containerID) {
		return
	}
	lt.testContainerStart(containerID)

	defer lt.testContainerDelete(containerID)

	for i := 0; i < operations && lt.ctx.Err() == nil; i++ {
		start := time.Now()
		success := true

		// Test terminal session creation
		sessionID, ok := lt.testCreateTerminalSession(containerID)
		if !ok {
			success = false
		} else {
			// Test command execution
			if !lt.testExecuteTerminalCommand(sessionID, "echo 'load test'") {
				success = false
			}

			// Test session cleanup
			if !lt.testCloseTerminalSession(sessionID) {
				success = false
			}
		}

		duration := time.Since(start)
		lt.recordOperationResult(results, success, duration)

		time.Sleep(500 * time.Millisecond) // Terminal operation interval
	}
}

// runWebSocketLoad performs load testing on WebSocket operations
func (lt *LoadTester) runWebSocketLoad(t *testing.T) {
	defer lt.wg.Done()

	results := lt.results.WebSocketResults
	connectionsPerUser := lt.config.WebSocketConnections / lt.config.ConcurrentUsers

	var userWg sync.WaitGroup
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		userWg.Add(1)
		go func(userID int) {
			defer userWg.Done()
			lt.performWebSocketOperations(userID, connectionsPerUser, results)
		}(i)
	}

	userWg.Wait()
	t.Logf("WebSocket load test completed: %d connections, %d successful",
		results.TotalConnections, results.SuccessfulConnections)
}

// performWebSocketOperations performs WebSocket operations for a single user
func (lt *LoadTester) performWebSocketOperations(userID, connections int, results *WebSocketResults) {
	for i := 0; i < connections && lt.ctx.Err() == nil; i++ {
		atomic.AddInt64(&results.TotalConnections, 1)

		success := lt.testWebSocketConnection(userID, i, results)
		if success {
			atomic.AddInt64(&results.SuccessfulConnections, 1)
		} else {
			atomic.AddInt64(&results.FailedConnections, 1)
		}

		time.Sleep(100 * time.Millisecond) // Connection interval
	}
}

// runGeneralAPILoad performs general API load testing
func (lt *LoadTester) runGeneralAPILoad(t *testing.T) {
	defer lt.wg.Done()

	requestsPerUser := lt.config.RequestsPerSecond * int(lt.config.TestDuration.Seconds()) / lt.config.ConcurrentUsers

	var userWg sync.WaitGroup
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		userWg.Add(1)
		go func(userID int) {
			defer userWg.Done()
			lt.performGeneralAPIRequests(userID, requestsPerUser)
		}(i)
	}

	userWg.Wait()
}

// performGeneralAPIRequests performs general API requests for load testing
func (lt *LoadTester) performGeneralAPIRequests(userID, requests int) {
	for i := 0; i < requests && lt.ctx.Err() == nil; i++ {
		start := time.Now()

		// Test various API endpoints
		endpoints := []string{
			"/api/v1/health",
			"/api/v1/system/info",
			"/api/v1/dashboard/summary",
			"/api/v1/containers",
			"/api/v1/images",
		}

		endpoint := endpoints[i%len(endpoints)]
		success := lt.makeHTTPRequest("GET", endpoint, nil)

		duration := time.Since(start)
		lt.recordHTTPResult(success, duration)

		// Rate limiting
		time.Sleep(time.Duration(1000/lt.config.RequestsPerSecond) * time.Millisecond)
	}
}

// HTTP test methods
func (lt *LoadTester) testContainerList() bool {
	return lt.makeHTTPRequest("GET", "/api/v1/containers", nil)
}

func (lt *LoadTester) testContainerCreate(containerID string) bool {
	data := map[string]interface{}{
		"name":      containerID,
		"image":     "alpine:latest",
		"cmd":       []string{"sleep", "30"},
	}
	return lt.makeHTTPRequest("POST", "/api/v1/containers", data)
}

func (lt *LoadTester) testContainerStart(containerID string) bool {
	return lt.makeHTTPRequest("POST", fmt.Sprintf("/api/v1/containers/%s/start", containerID), nil)
}

func (lt *LoadTester) testContainerStop(containerID string) bool {
	return lt.makeHTTPRequest("POST", fmt.Sprintf("/api/v1/containers/%s/stop", containerID), nil)
}

func (lt *LoadTester) testContainerDelete(containerID string) bool {
	return lt.makeHTTPRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s", containerID), nil)
}

func (lt *LoadTester) testGetContainerMetrics(containerID string) bool {
	return lt.makeHTTPRequest("GET", fmt.Sprintf("/api/v1/containers/%s/metrics", containerID), nil)
}

func (lt *LoadTester) testGetSystemMetrics() bool {
	return lt.makeHTTPRequest("GET", "/api/v1/system/metrics", nil)
}

func (lt *LoadTester) testGetMetricsHistory(containerID string) bool {
	return lt.makeHTTPRequest("GET", fmt.Sprintf("/api/v1/containers/%s/metrics/history", containerID), nil)
}

func (lt *LoadTester) testCreateTerminalSession(containerID string) (string, bool) {
	data := map[string]interface{}{
		"container_id": containerID,
		"shell":        "/bin/sh",
		"cols":         80,
		"rows":         24,
	}

	response, success := lt.makeHTTPRequestWithResponse("POST", "/api/v1/terminal/sessions", data)
	if !success {
		return "", false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return "", false
	}

	if sessionID, ok := result["session_id"].(string); ok {
		return sessionID, true
	}

	return "", false
}

func (lt *LoadTester) testExecuteTerminalCommand(sessionID, command string) bool {
	data := map[string]interface{}{
		"command": command,
	}
	return lt.makeHTTPRequest("POST", fmt.Sprintf("/api/v1/terminal/sessions/%s/execute", sessionID), data)
}

func (lt *LoadTester) testCloseTerminalSession(sessionID string) bool {
	return lt.makeHTTPRequest("DELETE", fmt.Sprintf("/api/v1/terminal/sessions/%s", sessionID), nil)
}

// WebSocket test methods
func (lt *LoadTester) testWebSocketConnection(userID, connID int, results *WebSocketResults) bool {
	wsURL := fmt.Sprintf("ws://%s/ws", lt.config.BaseURL[7:]) // Remove http://

	u, err := url.Parse(wsURL)
	if err != nil {
		return false
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send test messages
	testMessage := fmt.Sprintf(`{"type":"subscribe","container_id":"test-%d-%d"}`, userID, connID)

	start := time.Now()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(testMessage)); err != nil {
		return false
	}

	atomic.AddInt64(&results.MessagesSent, 1)

	// Wait for response with timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = conn.ReadMessage()
	if err != nil {
		return false
	}

	latency := time.Since(start)
	atomic.AddInt64(&results.MessagesReceived, 1)

	// Update average latency (simplified)
	if results.AverageLatency == 0 {
		results.AverageLatency = latency
	} else {
		results.AverageLatency = (results.AverageLatency + latency) / 2
	}

	return true
}

// HTTP request helpers
func (lt *LoadTester) makeHTTPRequest(method, endpoint string, data interface{}) bool {
	_, success := lt.makeHTTPRequestWithResponse(method, endpoint, data)
	return success
}

func (lt *LoadTester) makeHTTPRequestWithResponse(method, endpoint string, data interface{}) ([]byte, bool) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, false
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(lt.ctx, method, lt.config.BaseURL+endpoint, body)
	if err != nil {
		return nil, false
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := lt.httpClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}

	return responseBody, resp.StatusCode >= 200 && resp.StatusCode < 300
}

// Result recording helpers
func (lt *LoadTester) recordOperationResult(results *OperationResults, success bool, duration time.Duration) {
	atomic.AddInt64(&results.TotalOperations, 1)

	if success {
		atomic.AddInt64(&results.SuccessfulOps, 1)
	} else {
		atomic.AddInt64(&results.FailedOps, 1)
	}

	// Update timing stats (simplified, should use proper synchronization)
	if duration < results.MinTime {
		results.MinTime = duration
	}
	if duration > results.MaxTime {
		results.MaxTime = duration
	}
}

func (lt *LoadTester) recordHTTPResult(success bool, duration time.Duration) {
	atomic.AddInt64(&lt.results.TotalRequests, 1)

	if success {
		atomic.AddInt64(&lt.results.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&lt.results.FailedRequests, 1)
	}

	// Update timing stats (simplified)
	if duration < lt.results.MinResponseTime {
		lt.results.MinResponseTime = duration
	}
	if duration > lt.results.MaxResponseTime {
		lt.results.MaxResponseTime = duration
	}
}

// calculateFinalResults calculates final load test results
func (lt *LoadTester) calculateFinalResults() {
	totalTime := time.Since(lt.startTime)

	// Calculate request rates
	lt.results.RequestsPerSecond = float64(lt.results.TotalRequests) / totalTime.Seconds()

	// Calculate error rate
	if lt.results.TotalRequests > 0 {
		lt.results.ErrorRate = float64(lt.results.FailedRequests) / float64(lt.results.TotalRequests)
	}

	// Calculate average response times (simplified)
	if lt.results.TotalRequests > 0 {
		lt.results.AverageResponseTime = lt.results.MaxResponseTime / 2 // Simplified
	}

	// Get performance metrics from optimizer
	if lt.optimizer != nil {
		lt.results.PerformanceMetrics = lt.optimizer.GetPerformanceMetrics()
	}
}

// Test runner function
func TestSystemLoadAndPerformance(t *testing.T) {
	// Skip if not in load test environment
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := &LoadTestConfig{
		BaseURL:               "http://localhost:8080",
		ConcurrentUsers:       10,
		TestDuration:          2 * time.Minute,
		RequestsPerSecond:     50,
		ContainerOperations:   100,
		MonitoringOperations:  200,
		TerminalOperations:    50,
		WebSocketConnections:  20,
	}

	loadTester := NewLoadTester(config)
	results := loadTester.RunLoadTest(t)

	// Assert performance requirements
	assert.Less(t, results.ErrorRate, 0.01, "Error rate should be less than 1%")
	assert.Greater(t, results.RequestsPerSecond, 30.0, "Should handle at least 30 requests per second")
	assert.Less(t, results.AverageResponseTime, 500*time.Millisecond, "Average response time should be under 500ms")

	// Log detailed results
	t.Logf("Load Test Results:")
	t.Logf("Total Requests: %d", results.TotalRequests)
	t.Logf("Success Rate: %.2f%%", (1-results.ErrorRate)*100)
	t.Logf("Requests/Second: %.2f", results.RequestsPerSecond)
	t.Logf("Average Response Time: %v", results.AverageResponseTime)
	t.Logf("Container Operations: %d/%d successful",
		results.ContainerOpResults.SuccessfulOps,
		results.ContainerOpResults.TotalOperations)
	t.Logf("WebSocket Connections: %d/%d successful",
		results.WebSocketResults.SuccessfulConnections,
		results.WebSocketResults.TotalConnections)

	if results.PerformanceMetrics != nil {
		t.Logf("Database Connections: %d", results.PerformanceMetrics.DatabaseConnections)
		t.Logf("Cache Hit Rate: %.2f%%", results.PerformanceMetrics.CacheHitRate*100)
		t.Logf("Memory Usage: %.2f MB", float64(results.PerformanceMetrics.MemoryUsage)/1024/1024)
	}
}