package monitoring

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

// APIMonitoring monitors HTTP API performance
type APIMonitoring struct {
	mu              sync.RWMutex
	collector       *MetricsCollector
	requestCounter  *Counter
	responseTime    *Histogram
	errorCounter    *Counter
	activeRequests  *Gauge
}

// NewAPIMonitoring creates a new API monitoring instance
func NewAPIMonitoring(collector *MetricsCollector) *APIMonitoring {
	return &APIMonitoring{
		collector: collector,
		requestCounter: collector.RegisterCounter(
			"http_requests_total",
			"Total number of HTTP requests",
			nil,
		),
		responseTime: collector.RegisterHistogram(
			"http_request_duration_seconds",
			"HTTP request duration in seconds",
			DefaultHistogramBuckets(),
			nil,
		),
		errorCounter: collector.RegisterCounter(
			"http_errors_total",
			"Total number of HTTP errors",
			nil,
		),
		activeRequests: collector.RegisterGauge(
			"http_active_requests",
			"Number of active HTTP requests",
			nil,
		),
	}
}

// Middleware returns a Gin middleware for API monitoring
func (am *APIMonitoring) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Track active requests
		am.activeRequests.Inc()
		defer am.activeRequests.Dec()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()

		labels := map[string]string{
			"method": method,
			"path":   path,
			"status": statusClass(status),
		}

		// Count requests
		requestCounter := am.collector.RegisterCounter("http_requests_total", "Total HTTP requests", labels)
		requestCounter.Inc()

		// Record response time
		responseTime := am.collector.RegisterHistogram("http_request_duration_seconds", "HTTP request duration", DefaultHistogramBuckets(), labels)
		responseTime.Observe(duration.Seconds())

		// Count errors
		if status >= 400 {
			errorCounter := am.collector.RegisterCounter("http_errors_total", "Total HTTP errors", labels)
			errorCounter.Inc()
		}
	}
}

// CollectMetrics implements ComponentCollector interface
func (am *APIMonitoring) CollectMetrics() (ComponentMetrics, error) {
	metrics := []Metric{
		am.requestCounter.ToMetric(),
		am.errorCounter.ToMetric(),
		am.activeRequests.ToMetric(),
	}

	return ComponentMetrics{
		Component: "api",
		Metrics:   metrics,
		Timestamp: time.Now(),
	}, nil
}

func (am *APIMonitoring) Name() string {
	return "api_monitoring"
}

// statusClass converts HTTP status code to class
func statusClass(status int) string {
	switch {
	case status < 200:
		return "1xx"
	case status < 300:
		return "2xx"
	case status < 400:
		return "3xx"
	case status < 500:
		return "4xx"
	default:
		return "5xx"
	}
}

// DatabaseMonitoring monitors database performance
type DatabaseMonitoring struct {
	mu               sync.RWMutex
	collector        *MetricsCollector
	db               *sql.DB
	queryCounter     *Counter
	queryDuration    *Histogram
	connectionGauge  *Gauge
	errorCounter     *Counter
}

// NewDatabaseMonitoring creates a new database monitoring instance
func NewDatabaseMonitoring(collector *MetricsCollector, db *sql.DB) *DatabaseMonitoring {
	dm := &DatabaseMonitoring{
		collector: collector,
		db:        db,
		queryCounter: collector.RegisterCounter(
			"database_queries_total",
			"Total number of database queries",
			nil,
		),
		queryDuration: collector.RegisterHistogram(
			"database_query_duration_seconds",
			"Database query duration in seconds",
			DefaultHistogramBuckets(),
			nil,
		),
		connectionGauge: collector.RegisterGauge(
			"database_connections_active",
			"Number of active database connections",
			nil,
		),
		errorCounter: collector.RegisterCounter(
			"database_errors_total",
			"Total number of database errors",
			nil,
		),
	}

	// Start periodic collection
	go dm.periodicCollection()

	return dm
}

// periodicCollection periodically collects database metrics
func (dm *DatabaseMonitoring) periodicCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		dm.collectConnectionMetrics()
	}
}

// collectConnectionMetrics collects database connection metrics
func (dm *DatabaseMonitoring) collectConnectionMetrics() {
	if dm.db == nil {
		return
	}

	stats := dm.db.Stats()

	// Update connection metrics
	dm.collector.RegisterGauge("database_connections_open", "Open database connections", nil).Set(float64(stats.OpenConnections))
	dm.collector.RegisterGauge("database_connections_in_use", "Database connections in use", nil).Set(float64(stats.InUse))
	dm.collector.RegisterGauge("database_connections_idle", "Idle database connections", nil).Set(float64(stats.Idle))

	dm.collector.RegisterCounter("database_connections_wait_count", "Database connection wait count", nil).Add(float64(stats.WaitCount))
	dm.collector.RegisterGauge("database_connections_wait_duration", "Database connection wait duration", nil).Set(stats.WaitDuration.Seconds())

	dm.collector.RegisterCounter("database_connections_max_idle_closed", "Max idle connections closed", nil).Add(float64(stats.MaxIdleClosed))
	dm.collector.RegisterCounter("database_connections_max_lifetime_closed", "Max lifetime connections closed", nil).Add(float64(stats.MaxLifetimeClosed))
}

// TrackQuery tracks a database query execution
func (dm *DatabaseMonitoring) TrackQuery(operation string, query string, fn func() error) error {
	start := time.Now()
	labels := map[string]string{
		"operation": operation,
	}

	// Track active query
	queryGauge := dm.collector.RegisterGauge("database_queries_active", "Active database queries", labels)
	queryGauge.Inc()
	defer queryGauge.Dec()

	// Execute query
	err := fn()
	duration := time.Since(start)

	// Record metrics
	queryCounter := dm.collector.RegisterCounter("database_queries_total", "Total database queries", labels)
	queryCounter.Inc()

	queryDuration := dm.collector.RegisterHistogram("database_query_duration_seconds", "Database query duration", DefaultHistogramBuckets(), labels)
	queryDuration.Observe(duration.Seconds())

	if err != nil {
		errorCounter := dm.collector.RegisterCounter("database_errors_total", "Database errors", labels)
		errorCounter.Inc()
	}

	return err
}

// CollectMetrics implements ComponentCollector interface
func (dm *DatabaseMonitoring) CollectMetrics() (ComponentMetrics, error) {
	metrics := []Metric{
		dm.queryCounter.ToMetric(),
		dm.errorCounter.ToMetric(),
		dm.connectionGauge.ToMetric(),
	}

	return ComponentMetrics{
		Component: "database",
		Metrics:   metrics,
		Timestamp: time.Now(),
	}, nil
}

func (dm *DatabaseMonitoring) Name() string {
	return "database_monitoring"
}

// DockerMonitoring monitors Docker operations
type DockerMonitoring struct {
	mu                sync.RWMutex
	collector         *MetricsCollector
	dockerClient      *client.Client
	operationCounter  *Counter
	operationDuration *Histogram
	containerGauge    *Gauge
	imageGauge        *Gauge
}

// NewDockerMonitoring creates a new Docker monitoring instance
func NewDockerMonitoring(collector *MetricsCollector) *DockerMonitoring {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	dm := &DockerMonitoring{
		collector:    collector,
		dockerClient: dockerClient,
		operationCounter: collector.RegisterCounter(
			"docker_operations_total",
			"Total number of Docker operations",
			nil,
		),
		operationDuration: collector.RegisterHistogram(
			"docker_operation_duration_seconds",
			"Docker operation duration in seconds",
			DefaultHistogramBuckets(),
			nil,
		),
		containerGauge: collector.RegisterGauge(
			"docker_containers_running",
			"Number of running Docker containers",
			nil,
		),
		imageGauge: collector.RegisterGauge(
			"docker_images_total",
			"Total number of Docker images",
			nil,
		),
	}

	// Start periodic collection
	go dm.periodicCollection()

	return dm
}

// periodicCollection periodically collects Docker metrics
func (dm *DockerMonitoring) periodicCollection() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		dm.collectDockerMetrics()
	}
}

// collectDockerMetrics collects Docker daemon metrics
func (dm *DockerMonitoring) collectDockerMetrics() {
	if dm.dockerClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get container statistics
	containers, err := dm.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err == nil {
		runningCount := 0
		stoppedCount := 0

		for _, container := range containers {
			if container.State == "running" {
				runningCount++
			} else {
				stoppedCount++
			}
		}

		dm.collector.RegisterGauge("docker_containers_running", "Running containers", nil).Set(float64(runningCount))
		dm.collector.RegisterGauge("docker_containers_stopped", "Stopped containers", nil).Set(float64(stoppedCount))
		dm.collector.RegisterGauge("docker_containers_total", "Total containers", nil).Set(float64(len(containers)))
	}

	// Get image statistics
	images, err := dm.dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err == nil {
		dm.collector.RegisterGauge("docker_images_total", "Total images", nil).Set(float64(len(images)))
	}
}

// TrackOperation tracks a Docker operation
func (dm *DockerMonitoring) TrackOperation(operation string, fn func() error) error {
	start := time.Now()
	labels := map[string]string{
		"operation": operation,
	}

	// Execute operation
	err := fn()
	duration := time.Since(start)

	// Record metrics
	operationCounter := dm.collector.RegisterCounter("docker_operations_total", "Total Docker operations", labels)
	operationCounter.Inc()

	operationDuration := dm.collector.RegisterHistogram("docker_operation_duration_seconds", "Docker operation duration", DefaultHistogramBuckets(), labels)
	operationDuration.Observe(duration.Seconds())

	if err != nil {
		errorCounter := dm.collector.RegisterCounter("docker_operation_errors_total", "Docker operation errors", labels)
		errorCounter.Inc()
	}

	return err
}

// CollectMetrics implements ComponentCollector interface
func (dm *DockerMonitoring) CollectMetrics() (ComponentMetrics, error) {
	metrics := []Metric{
		dm.operationCounter.ToMetric(),
		dm.containerGauge.ToMetric(),
		dm.imageGauge.ToMetric(),
	}

	return ComponentMetrics{
		Component: "docker",
		Metrics:   metrics,
		Timestamp: time.Now(),
	}, nil
}

func (dm *DockerMonitoring) Name() string {
	return "docker_monitoring"
}

// WebSocketMonitoring monitors WebSocket connections
type WebSocketMonitoring struct {
	mu                 sync.RWMutex
	collector          *MetricsCollector
	connectionCounter  *Counter
	messageCounter     *Counter
	errorCounter       *Counter
	activeConnections  *Gauge
	messageDuration    *Histogram
}

// NewWebSocketMonitoring creates a new WebSocket monitoring instance
func NewWebSocketMonitoring(collector *MetricsCollector) *WebSocketMonitoring {
	return &WebSocketMonitoring{
		collector: collector,
		connectionCounter: collector.RegisterCounter(
			"websocket_connections_total",
			"Total number of WebSocket connections",
			nil,
		),
		messageCounter: collector.RegisterCounter(
			"websocket_messages_total",
			"Total number of WebSocket messages",
			nil,
		),
		errorCounter: collector.RegisterCounter(
			"websocket_errors_total",
			"Total number of WebSocket errors",
			nil,
		),
		activeConnections: collector.RegisterGauge(
			"websocket_connections_active",
			"Number of active WebSocket connections",
			nil,
		),
		messageDuration: collector.RegisterHistogram(
			"websocket_message_duration_seconds",
			"WebSocket message processing duration in seconds",
			DefaultHistogramBuckets(),
			nil,
		),
	}
}

// TrackConnection tracks a WebSocket connection
func (wsm *WebSocketMonitoring) TrackConnection() func() {
	wsm.connectionCounter.Inc()
	wsm.activeConnections.Inc()

	return func() {
		wsm.activeConnections.Dec()
	}
}

// TrackMessage tracks a WebSocket message
func (wsm *WebSocketMonitoring) TrackMessage(messageType string, fn func() error) error {
	start := time.Now()
	labels := map[string]string{
		"type": messageType,
	}

	// Execute message processing
	err := fn()
	duration := time.Since(start)

	// Record metrics
	messageCounter := wsm.collector.RegisterCounter("websocket_messages_total", "WebSocket messages", labels)
	messageCounter.Inc()

	messageDuration := wsm.collector.RegisterHistogram("websocket_message_duration_seconds", "Message duration", DefaultHistogramBuckets(), labels)
	messageDuration.Observe(duration.Seconds())

	if err != nil {
		errorCounter := wsm.collector.RegisterCounter("websocket_errors_total", "WebSocket errors", labels)
		errorCounter.Inc()
	}

	return err
}

// CollectMetrics implements ComponentCollector interface
func (wsm *WebSocketMonitoring) CollectMetrics() (ComponentMetrics, error) {
	metrics := []Metric{
		wsm.connectionCounter.ToMetric(),
		wsm.messageCounter.ToMetric(),
		wsm.errorCounter.ToMetric(),
		wsm.activeConnections.ToMetric(),
	}

	return ComponentMetrics{
		Component: "websocket",
		Metrics:   metrics,
		Timestamp: time.Now(),
	}, nil
}

func (wsm *WebSocketMonitoring) Name() string {
	return "websocket_monitoring"
}

// BusinessLogicMonitoring monitors business-specific operations
type BusinessLogicMonitoring struct {
	mu                    sync.RWMutex
	collector             *MetricsCollector
	updateCounter         *Counter
	updateDuration        *Histogram
	containerUpdateGauge  *Gauge
	notificationCounter   *Counter
}

// NewBusinessLogicMonitoring creates a new business logic monitoring instance
func NewBusinessLogicMonitoring(collector *MetricsCollector) *BusinessLogicMonitoring {
	return &BusinessLogicMonitoring{
		collector: collector,
		updateCounter: collector.RegisterCounter(
			"container_updates_total",
			"Total number of container updates",
			nil,
		),
		updateDuration: collector.RegisterHistogram(
			"container_update_duration_seconds",
			"Container update duration in seconds",
			DefaultHistogramBuckets(),
			nil,
		),
		containerUpdateGauge: collector.RegisterGauge(
			"containers_managed_total",
			"Total number of managed containers",
			nil,
		),
		notificationCounter: collector.RegisterCounter(
			"notifications_sent_total",
			"Total number of notifications sent",
			nil,
		),
	}
}

// TrackContainerUpdate tracks a container update operation
func (blm *BusinessLogicMonitoring) TrackContainerUpdate(containerName string, updateType string, fn func() error) error {
	start := time.Now()
	labels := map[string]string{
		"container":   containerName,
		"update_type": updateType,
	}

	// Execute update
	err := fn()
	duration := time.Since(start)

	// Record metrics
	updateCounter := blm.collector.RegisterCounter("container_updates_total", "Container updates", labels)
	updateCounter.Inc()

	updateDuration := blm.collector.RegisterHistogram("container_update_duration_seconds", "Update duration", DefaultHistogramBuckets(), labels)
	updateDuration.Observe(duration.Seconds())

	if err != nil {
		errorCounter := blm.collector.RegisterCounter("container_update_errors_total", "Update errors", labels)
		errorCounter.Inc()
	} else {
		successCounter := blm.collector.RegisterCounter("container_updates_successful_total", "Successful updates", labels)
		successCounter.Inc()
	}

	return err
}

// TrackNotification tracks a notification sent
func (blm *BusinessLogicMonitoring) TrackNotification(notificationType, channel string, success bool) {
	labels := map[string]string{
		"type":    notificationType,
		"channel": channel,
	}

	notificationCounter := blm.collector.RegisterCounter("notifications_sent_total", "Notifications sent", labels)
	notificationCounter.Inc()

	if !success {
		errorCounter := blm.collector.RegisterCounter("notification_errors_total", "Notification errors", labels)
		errorCounter.Inc()
	}
}

// UpdateManagedContainers updates the count of managed containers
func (blm *BusinessLogicMonitoring) UpdateManagedContainers(count int) {
	blm.containerUpdateGauge.Set(float64(count))
}

// CollectMetrics implements ComponentCollector interface
func (blm *BusinessLogicMonitoring) CollectMetrics() (ComponentMetrics, error) {
	metrics := []Metric{
		blm.updateCounter.ToMetric(),
		blm.containerUpdateGauge.ToMetric(),
		blm.notificationCounter.ToMetric(),
	}

	return ComponentMetrics{
		Component: "business_logic",
		Metrics:   metrics,
		Timestamp: time.Now(),
	}, nil
}

func (blm *BusinessLogicMonitoring) Name() string {
	return "business_logic_monitoring"
}