package metrics

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PerformanceCollector collects and manages performance metrics
type PerformanceCollector struct {
	mu                sync.RWMutex
	enabled           bool
	collectionInterval time.Duration

	// System metrics
	systemMetrics     SystemMetrics

	// Application metrics
	requestMetrics    RequestMetrics
	databaseMetrics   DatabaseMetrics
	websocketMetrics  WebSocketMetrics
	dockerMetrics     DockerMetrics
	cacheMetrics      CacheMetrics

	// Performance history
	history           []PerformanceSnapshot
	maxHistorySize    int

	// Channels for metric updates
	requestCh         chan RequestMetric
	databaseCh        chan DatabaseMetric
	websocketCh       chan WebSocketMetric
	dockerCh          chan DockerMetric
	cacheCh           chan CacheMetric

	// Control channels
	stopCh            chan struct{}
	done              chan struct{}
}

// SystemMetrics tracks system-level performance
type SystemMetrics struct {
	CPUUsagePercent   float64   `json:"cpu_usage_percent"`
	MemoryUsedBytes   uint64    `json:"memory_used_bytes"`
	MemoryTotalBytes  uint64    `json:"memory_total_bytes"`
	GoroutineCount    int       `json:"goroutine_count"`
	GCPauseNanoseconds uint64   `json:"gc_pause_nanoseconds"`
	HeapSizeBytes     uint64    `json:"heap_size_bytes"`
	HeapObjectCount   uint64    `json:"heap_object_count"`
	Timestamp         time.Time `json:"timestamp"`
}

// RequestMetrics tracks API request performance
type RequestMetrics struct {
	TotalRequests     int64                    `json:"total_requests"`
	RequestsPerSecond float64                  `json:"requests_per_second"`
	AverageLatency    time.Duration            `json:"average_latency"`
	P95Latency        time.Duration            `json:"p95_latency"`
	P99Latency        time.Duration            `json:"p99_latency"`
	ErrorRate         float64                  `json:"error_rate"`
	EndpointMetrics   map[string]EndpointStats `json:"endpoint_metrics"`
	StatusCodeCounts  map[int]int64           `json:"status_code_counts"`
	ActiveRequests    int64                   `json:"active_requests"`
}

// EndpointStats tracks per-endpoint statistics
type EndpointStats struct {
	Count        int64         `json:"count"`
	TotalTime    time.Duration `json:"total_time"`
	MinTime      time.Duration `json:"min_time"`
	MaxTime      time.Duration `json:"max_time"`
	AverageTime  time.Duration `json:"average_time"`
	ErrorCount   int64         `json:"error_count"`
	ErrorRate    float64       `json:"error_rate"`
}

// DatabaseMetrics tracks database performance
type DatabaseMetrics struct {
	ActiveConnections     int           `json:"active_connections"`
	IdleConnections      int           `json:"idle_connections"`
	MaxConnections       int           `json:"max_connections"`
	QueryCount           int64         `json:"query_count"`
	SlowQueryCount       int64         `json:"slow_query_count"`
	AverageQueryTime     time.Duration `json:"average_query_time"`
	ConnectionWaitTime   time.Duration `json:"connection_wait_time"`
	TransactionCount     int64         `json:"transaction_count"`
	TransactionErrorCount int64        `json:"transaction_error_count"`
}

// WebSocketMetrics tracks WebSocket performance
type WebSocketMetrics struct {
	ActiveConnections    int64   `json:"active_connections"`
	TotalConnections     int64   `json:"total_connections"`
	MessagesPerSecond    float64 `json:"messages_per_second"`
	AverageMessageLatency time.Duration `json:"average_message_latency"`
	ConnectionUptime     time.Duration `json:"connection_uptime"`
	ReconnectionCount    int64   `json:"reconnection_count"`
	MessageBatchingRate  float64 `json:"message_batching_rate"`
	CompressionRatio     float64 `json:"compression_ratio"`
}

// DockerMetrics tracks Docker operations performance
type DockerMetrics struct {
	OperationCount       int64         `json:"operation_count"`
	AverageOperationTime time.Duration `json:"average_operation_time"`
	FailureRate          float64       `json:"failure_rate"`
	ActiveOperations     int64         `json:"active_operations"`
	ParallelOperations   int64         `json:"parallel_operations"`
	ContainerCount       int           `json:"container_count"`
	ImageCount           int           `json:"image_count"`
	ConnectionPoolSize   int           `json:"connection_pool_size"`
	ConnectionsInUse     int           `json:"connections_in_use"`
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	HitRatio         float64       `json:"hit_ratio"`
	ItemCount        int64         `json:"item_count"`
	MaxItems         int64         `json:"max_items"`
	MemoryUsageBytes int64         `json:"memory_usage_bytes"`
	AverageLatency   time.Duration `json:"average_latency"`
	EvictionRate     float64       `json:"eviction_rate"`
	SetOperations    int64         `json:"set_operations"`
	GetOperations    int64         `json:"get_operations"`
	DeleteOperations int64         `json:"delete_operations"`
}

// Individual metric events
type RequestMetric struct {
	Endpoint   string
	Method     string
	StatusCode int
	Duration   time.Duration
	Error      error
	Timestamp  time.Time
}

type DatabaseMetric struct {
	QueryType      string
	Duration       time.Duration
	Error          error
	ConnectionWait time.Duration
	Timestamp      time.Time
}

type WebSocketMetric struct {
	EventType string
	Duration  time.Duration
	Size      int64
	Error     error
	Timestamp time.Time
}

type DockerMetric struct {
	Operation string
	Duration  time.Duration
	Error     error
	Parallel  bool
	Timestamp time.Time
}

type CacheMetric struct {
	Operation string
	Hit       bool
	Duration  time.Duration
	Timestamp time.Time
}

// PerformanceSnapshot represents a point-in-time performance snapshot
type PerformanceSnapshot struct {
	Timestamp        time.Time        `json:"timestamp"`
	SystemMetrics    SystemMetrics    `json:"system_metrics"`
	RequestMetrics   RequestMetrics   `json:"request_metrics"`
	DatabaseMetrics  DatabaseMetrics  `json:"database_metrics"`
	WebSocketMetrics WebSocketMetrics `json:"websocket_metrics"`
	DockerMetrics    DockerMetrics    `json:"docker_metrics"`
	CacheMetrics     CacheMetrics     `json:"cache_metrics"`
}

// NewPerformanceCollector creates a new performance collector
func NewPerformanceCollector(collectionInterval time.Duration) *PerformanceCollector {
	pc := &PerformanceCollector{
		enabled:           true,
		collectionInterval: collectionInterval,
		maxHistorySize:    1000, // Keep last 1000 snapshots
		history:          make([]PerformanceSnapshot, 0, 1000),

		// Initialize channels
		requestCh:    make(chan RequestMetric, 10000),
		databaseCh:   make(chan DatabaseMetric, 10000),
		websocketCh:  make(chan WebSocketMetric, 10000),
		dockerCh:     make(chan DockerMetric, 10000),
		cacheCh:      make(chan CacheMetric, 10000),
		stopCh:       make(chan struct{}),
		done:         make(chan struct{}),
	}

	// Initialize endpoint metrics
	pc.requestMetrics.EndpointMetrics = make(map[string]EndpointStats)
	pc.requestMetrics.StatusCodeCounts = make(map[int]int64)

	return pc
}

// Start begins performance monitoring
func (pc *PerformanceCollector) Start() {
	if !pc.enabled {
		return
	}

	go pc.metricsCollectionLoop()
	go pc.eventProcessingLoop()

	logrus.WithField("interval", pc.collectionInterval).Info("Performance monitoring started")
}

// Stop halts performance monitoring
func (pc *PerformanceCollector) Stop() {
	if !pc.enabled {
		return
	}

	close(pc.stopCh)
	<-pc.done

	logrus.Info("Performance monitoring stopped")
}

// metricsCollectionLoop periodically collects system metrics
func (pc *PerformanceCollector) metricsCollectionLoop() {
	defer close(pc.done)

	ticker := time.NewTicker(pc.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.collectSystemMetrics()
			pc.takeSnapshot()
		case <-pc.stopCh:
			return
		}
	}
}

// eventProcessingLoop processes individual metric events
func (pc *PerformanceCollector) eventProcessingLoop() {
	for {
		select {
		case metric := <-pc.requestCh:
			pc.processRequestMetric(metric)
		case metric := <-pc.databaseCh:
			pc.processDatabaseMetric(metric)
		case metric := <-pc.websocketCh:
			pc.processWebSocketMetric(metric)
		case metric := <-pc.dockerCh:
			pc.processDockerMetric(metric)
		case metric := <-pc.cacheCh:
			pc.processCacheMetric(metric)
		case <-pc.stopCh:
			return
		}
	}
}

// collectSystemMetrics gathers current system performance data
func (pc *PerformanceCollector) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.systemMetrics = SystemMetrics{
		CPUUsagePercent:    pc.getCPUUsage(),
		MemoryUsedBytes:    m.Alloc,
		MemoryTotalBytes:   m.Sys,
		GoroutineCount:     runtime.NumGoroutine(),
		GCPauseNanoseconds: m.PauseNs[(m.NumGC+255)%256],
		HeapSizeBytes:      m.HeapAlloc,
		HeapObjectCount:    m.HeapObjects,
		Timestamp:          time.Now(),
	}
}

// getCPUUsage calculates CPU usage percentage (simplified)
func (pc *PerformanceCollector) getCPUUsage() float64 {
	// This is a simplified CPU usage calculation
	// In production, you would use more sophisticated methods
	return float64(runtime.NumGoroutine()) * 0.1 // Placeholder calculation
}

// takeSnapshot creates a performance snapshot
func (pc *PerformanceCollector) takeSnapshot() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	snapshot := PerformanceSnapshot{
		Timestamp:        time.Now(),
		SystemMetrics:    pc.systemMetrics,
		RequestMetrics:   pc.requestMetrics,
		DatabaseMetrics:  pc.databaseMetrics,
		WebSocketMetrics: pc.websocketMetrics,
		DockerMetrics:    pc.dockerMetrics,
		CacheMetrics:     pc.cacheMetrics,
	}

	// Add to history
	pc.history = append(pc.history, snapshot)

	// Trim history if it exceeds max size
	if len(pc.history) > pc.maxHistorySize {
		pc.history = pc.history[1:]
	}
}

// Processing methods for different metric types
func (pc *PerformanceCollector) processRequestMetric(metric RequestMetric) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Update total requests
	pc.requestMetrics.TotalRequests++

	// Update status code counts
	pc.requestMetrics.StatusCodeCounts[metric.StatusCode]++

	// Update endpoint-specific metrics
	endpoint := metric.Method + " " + metric.Endpoint
	stats, exists := pc.requestMetrics.EndpointMetrics[endpoint]
	if !exists {
		stats = EndpointStats{
			MinTime: metric.Duration,
			MaxTime: metric.Duration,
		}
	}

	stats.Count++
	stats.TotalTime += metric.Duration
	stats.AverageTime = stats.TotalTime / time.Duration(stats.Count)

	if metric.Duration < stats.MinTime {
		stats.MinTime = metric.Duration
	}
	if metric.Duration > stats.MaxTime {
		stats.MaxTime = metric.Duration
	}

	if metric.Error != nil {
		stats.ErrorCount++
	}
	stats.ErrorRate = float64(stats.ErrorCount) / float64(stats.Count)

	pc.requestMetrics.EndpointMetrics[endpoint] = stats

	// Update overall error rate
	var totalErrors int64
	for _, s := range pc.requestMetrics.EndpointMetrics {
		totalErrors += s.ErrorCount
	}
	pc.requestMetrics.ErrorRate = float64(totalErrors) / float64(pc.requestMetrics.TotalRequests)
}

func (pc *PerformanceCollector) processDatabaseMetric(metric DatabaseMetric) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.databaseMetrics.QueryCount++

	if pc.databaseMetrics.AverageQueryTime == 0 {
		pc.databaseMetrics.AverageQueryTime = metric.Duration
	} else {
		// Exponential moving average
		pc.databaseMetrics.AverageQueryTime =
			time.Duration(float64(pc.databaseMetrics.AverageQueryTime) * 0.9 + float64(metric.Duration) * 0.1)
	}

	if metric.Duration > 100*time.Millisecond {
		pc.databaseMetrics.SlowQueryCount++
	}

	if metric.Error != nil {
		pc.databaseMetrics.TransactionErrorCount++
	}
}

func (pc *PerformanceCollector) processWebSocketMetric(metric WebSocketMetric) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.websocketMetrics.AverageMessageLatency == 0 {
		pc.websocketMetrics.AverageMessageLatency = metric.Duration
	} else {
		// Exponential moving average
		pc.websocketMetrics.AverageMessageLatency =
			time.Duration(float64(pc.websocketMetrics.AverageMessageLatency) * 0.9 + float64(metric.Duration) * 0.1)
	}
}

func (pc *PerformanceCollector) processDockerMetric(metric DockerMetric) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.dockerMetrics.OperationCount++

	if pc.dockerMetrics.AverageOperationTime == 0 {
		pc.dockerMetrics.AverageOperationTime = metric.Duration
	} else {
		// Exponential moving average
		pc.dockerMetrics.AverageOperationTime =
			time.Duration(float64(pc.dockerMetrics.AverageOperationTime) * 0.9 + float64(metric.Duration) * 0.1)
	}

	if metric.Parallel {
		pc.dockerMetrics.ParallelOperations++
	}

	if metric.Error != nil {
		errorCount := pc.dockerMetrics.OperationCount - int64(pc.dockerMetrics.FailureRate * float64(pc.dockerMetrics.OperationCount))
		errorCount++
		pc.dockerMetrics.FailureRate = float64(errorCount) / float64(pc.dockerMetrics.OperationCount)
	}
}

func (pc *PerformanceCollector) processCacheMetric(metric CacheMetric) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	switch metric.Operation {
	case "get":
		pc.cacheMetrics.GetOperations++
		if metric.Hit {
			// Update hit ratio using exponential moving average
			if pc.cacheMetrics.HitRatio == 0 {
				pc.cacheMetrics.HitRatio = 1.0
			} else {
				pc.cacheMetrics.HitRatio = pc.cacheMetrics.HitRatio * 0.99 + 0.01
			}
		} else {
			pc.cacheMetrics.HitRatio = pc.cacheMetrics.HitRatio * 0.99
		}
	case "set":
		pc.cacheMetrics.SetOperations++
	case "delete":
		pc.cacheMetrics.DeleteOperations++
	}

	if pc.cacheMetrics.AverageLatency == 0 {
		pc.cacheMetrics.AverageLatency = metric.Duration
	} else {
		pc.cacheMetrics.AverageLatency =
			time.Duration(float64(pc.cacheMetrics.AverageLatency) * 0.9 + float64(metric.Duration) * 0.1)
	}
}

// Public methods for recording metrics
func (pc *PerformanceCollector) RecordRequest(endpoint, method string, statusCode int, duration time.Duration, err error) {
	if !pc.enabled {
		return
	}

	select {
	case pc.requestCh <- RequestMetric{
		Endpoint:   endpoint,
		Method:     method,
		StatusCode: statusCode,
		Duration:   duration,
		Error:      err,
		Timestamp:  time.Now(),
	}:
	default:
		// Channel full, drop metric
	}
}

func (pc *PerformanceCollector) RecordDatabaseQuery(queryType string, duration time.Duration, err error) {
	if !pc.enabled {
		return
	}

	select {
	case pc.databaseCh <- DatabaseMetric{
		QueryType: queryType,
		Duration:  duration,
		Error:     err,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, drop metric
	}
}

func (pc *PerformanceCollector) RecordWebSocketEvent(eventType string, duration time.Duration, size int64, err error) {
	if !pc.enabled {
		return
	}

	select {
	case pc.websocketCh <- WebSocketMetric{
		EventType: eventType,
		Duration:  duration,
		Size:      size,
		Error:     err,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, drop metric
	}
}

func (pc *PerformanceCollector) RecordDockerOperation(operation string, duration time.Duration, parallel bool, err error) {
	if !pc.enabled {
		return
	}

	select {
	case pc.dockerCh <- DockerMetric{
		Operation: operation,
		Duration:  duration,
		Error:     err,
		Parallel:  parallel,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, drop metric
	}
}

func (pc *PerformanceCollector) RecordCacheOperation(operation string, hit bool, duration time.Duration) {
	if !pc.enabled {
		return
	}

	select {
	case pc.cacheCh <- CacheMetric{
		Operation: operation,
		Hit:       hit,
		Duration:  duration,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, drop metric
	}
}

// GetCurrentMetrics returns the current performance metrics
func (pc *PerformanceCollector) GetCurrentMetrics() PerformanceSnapshot {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return PerformanceSnapshot{
		Timestamp:        time.Now(),
		SystemMetrics:    pc.systemMetrics,
		RequestMetrics:   pc.requestMetrics,
		DatabaseMetrics:  pc.databaseMetrics,
		WebSocketMetrics: pc.websocketMetrics,
		DockerMetrics:    pc.dockerMetrics,
		CacheMetrics:     pc.cacheMetrics,
	}
}

// GetHistoricalMetrics returns historical performance data
func (pc *PerformanceCollector) GetHistoricalMetrics(since time.Time, limit int) []PerformanceSnapshot {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	var filtered []PerformanceSnapshot
	for i := len(pc.history) - 1; i >= 0 && len(filtered) < limit; i-- {
		if pc.history[i].Timestamp.After(since) {
			filtered = append([]PerformanceSnapshot{pc.history[i]}, filtered...)
		}
	}

	return filtered
}

// GetPerformanceReport generates a comprehensive performance report
func (pc *PerformanceCollector) GetPerformanceReport() map[string]interface{} {
	current := pc.GetCurrentMetrics()

	// Get historical data for comparison
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	historical := pc.GetHistoricalMetrics(oneDayAgo, 100)

	report := map[string]interface{}{
		"current":    current,
		"historical": historical,
		"summary":    pc.generateSummary(current, historical),
	}

	return report
}

// generateSummary creates a performance summary
func (pc *PerformanceCollector) generateSummary(current PerformanceSnapshot, historical []PerformanceSnapshot) map[string]interface{} {
	summary := map[string]interface{}{
		"collection_time": current.Timestamp,
		"health_status":   "healthy", // Default
	}

	// System health checks
	if current.SystemMetrics.CPUUsagePercent > 80 {
		summary["health_status"] = "degraded"
		summary["cpu_warning"] = "High CPU usage detected"
	}

	if current.SystemMetrics.MemoryUsedBytes > current.SystemMetrics.MemoryTotalBytes * 8 / 10 {
		summary["health_status"] = "degraded"
		summary["memory_warning"] = "High memory usage detected"
	}

	// Performance trends
	if len(historical) > 1 {
		first := historical[0]
		last := historical[len(historical)-1]

		// Request rate trend
		if last.RequestMetrics.RequestsPerSecond > first.RequestMetrics.RequestsPerSecond * 1.5 {
			summary["request_rate_trend"] = "increasing"
		} else if last.RequestMetrics.RequestsPerSecond < first.RequestMetrics.RequestsPerSecond * 0.5 {
			summary["request_rate_trend"] = "decreasing"
		} else {
			summary["request_rate_trend"] = "stable"
		}

		// Error rate analysis
		if current.RequestMetrics.ErrorRate > 0.05 { // 5% error rate
			summary["health_status"] = "degraded"
			summary["error_rate_warning"] = "High error rate detected"
		}
	}

	return summary
}

// ExportMetrics exports metrics in JSON format
func (pc *PerformanceCollector) ExportMetrics(ctx context.Context) ([]byte, error) {
	report := pc.GetPerformanceReport()
	return json.MarshalIndent(report, "", "  ")
}

// Global performance collector instance
var globalCollector *PerformanceCollector
var collectorOnce sync.Once

// GetGlobalCollector returns the global performance collector instance
func GetGlobalCollector() *PerformanceCollector {
	collectorOnce.Do(func() {
		globalCollector = NewPerformanceCollector(30 * time.Second) // Collect every 30 seconds
		globalCollector.Start()
	})
	return globalCollector
}