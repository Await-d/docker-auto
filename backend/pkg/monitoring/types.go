package monitoring

import (
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric with metadata
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description"`
	Unit        string                 `json:"unit,omitempty"`
	Help        string                 `json:"help,omitempty"`
}

// Counter represents a counter metric that only increases
type Counter struct {
	mu          sync.RWMutex
	name        string
	description string
	value       float64
	labels      map[string]string
}

// Gauge represents a gauge metric that can increase or decrease
type Gauge struct {
	mu          sync.RWMutex
	name        string
	description string
	value       float64
	labels      map[string]string
}

// Histogram represents a histogram metric for measuring distributions
type Histogram struct {
	mu          sync.RWMutex
	name        string
	description string
	buckets     []float64
	counts      []uint64
	sum         float64
	count       uint64
	labels      map[string]string
}

// HistogramBucket represents a single bucket in a histogram
type HistogramBucket struct {
	UpperBound float64 `json:"upper_bound"`
	Count      uint64  `json:"count"`
}

// Summary represents a summary metric with quantiles
type Summary struct {
	mu          sync.RWMutex
	name        string
	description string
	observations []float64
	sum          float64
	count        uint64
	quantiles    []float64
	labels       map[string]string
	maxAge       time.Duration
	timestamps   []time.Time
}

// MetricsConfig represents configuration for the metrics system
type MetricsConfig struct {
	Enabled            bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	MaxMetrics         int           `json:"max_metrics"`
	BufferSize         int           `json:"buffer_size"`
	Storage            StorageConfig `json:"storage"`
	Export             ExportConfig  `json:"export"`
}

// StorageConfig represents configuration for metric storage
type StorageConfig struct {
	Type       string `json:"type"` // memory, file, database
	Path       string `json:"path,omitempty"`
	MaxSize    int64  `json:"max_size,omitempty"`
	Compression bool  `json:"compression,omitempty"`
}

// ExportConfig represents configuration for metric export
type ExportConfig struct {
	Enabled   bool          `json:"enabled"`
	Format    string        `json:"format"` // prometheus, json, influxdb
	Endpoint  string        `json:"endpoint,omitempty"`
	Interval  time.Duration `json:"interval"`
	BatchSize int           `json:"batch_size"`
}

// MetricUpdate represents an update to a metric
type MetricUpdate struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
	Operation string            `json:"operation"` // inc, dec, set, observe
}

// AlertRule represents a rule for alerting based on metrics
type AlertRule struct {
	Name        string                 `json:"name"`
	MetricName  string                 `json:"metric_name"`
	Condition   AlertCondition         `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Duration    time.Duration          `json:"duration"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

// AlertCondition represents the condition for triggering an alert
type AlertCondition string

const (
	AlertConditionGreaterThan    AlertCondition = ">"
	AlertConditionGreaterOrEqual AlertCondition = ">="
	AlertConditionLessThan       AlertCondition = "<"
	AlertConditionLessOrEqual    AlertCondition = "<="
	AlertConditionEqual          AlertCondition = "=="
	AlertConditionNotEqual       AlertCondition = "!="
)

// ComponentMetrics represents metrics for a specific component
type ComponentMetrics struct {
	Component string    `json:"component"`
	Metrics   []Metric  `json:"metrics"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemMetrics represents overall system metrics
type SystemMetrics struct {
	CPU              CPUMetrics         `json:"cpu"`
	Memory           MemoryMetrics      `json:"memory"`
	Disk             DiskMetrics        `json:"disk"`
	Network          NetworkMetrics     `json:"network"`
	Docker           DockerMetrics      `json:"docker"`
	Application      ApplicationMetrics `json:"application"`
	Database         DatabaseMetrics    `json:"database"`
	WebSocket        WebSocketMetrics   `json:"websocket"`
	Timestamp        time.Time          `json:"timestamp"`
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	Usage     float64 `json:"usage"`      // percentage
	LoadAvg1  float64 `json:"load_avg_1"` // 1-minute load average
	LoadAvg5  float64 `json:"load_avg_5"` // 5-minute load average
	LoadAvg15 float64 `json:"load_avg_15"` // 15-minute load average
	Cores     int     `json:"cores"`
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	Total     uint64  `json:"total"`     // bytes
	Used      uint64  `json:"used"`      // bytes
	Available uint64  `json:"available"` // bytes
	Usage     float64 `json:"usage"`     // percentage
	Swap      SwapMetrics `json:"swap"`
}

// SwapMetrics represents swap usage metrics
type SwapMetrics struct {
	Total uint64  `json:"total"` // bytes
	Used  uint64  `json:"used"`  // bytes
	Usage float64 `json:"usage"` // percentage
}

// DiskMetrics represents disk usage metrics
type DiskMetrics struct {
	Total       uint64  `json:"total"`        // bytes
	Used        uint64  `json:"used"`         // bytes
	Available   uint64  `json:"available"`    // bytes
	Usage       float64 `json:"usage"`        // percentage
	ReadOps     uint64  `json:"read_ops"`
	WriteOps    uint64  `json:"write_ops"`
	ReadBytes   uint64  `json:"read_bytes"`
	WriteBytes  uint64  `json:"write_bytes"`
}

// NetworkMetrics represents network usage metrics
type NetworkMetrics struct {
	BytesReceived    uint64 `json:"bytes_received"`
	BytesSent        uint64 `json:"bytes_sent"`
	PacketsReceived  uint64 `json:"packets_received"`
	PacketsSent      uint64 `json:"packets_sent"`
	ErrorsReceived   uint64 `json:"errors_received"`
	ErrorsSent       uint64 `json:"errors_sent"`
	DroppedReceived  uint64 `json:"dropped_received"`
	DroppedSent      uint64 `json:"dropped_sent"`
}

// DockerMetrics represents Docker-related metrics
type DockerMetrics struct {
	ContainersRunning int     `json:"containers_running"`
	ContainersStopped int     `json:"containers_stopped"`
	ContainersPaused  int     `json:"containers_paused"`
	Images           int     `json:"images"`
	Volumes          int     `json:"volumes"`
	Networks         int     `json:"networks"`
	CPUUsage         float64 `json:"cpu_usage"`
	MemoryUsage      uint64  `json:"memory_usage"`
}

// ApplicationMetrics represents application-specific metrics
type ApplicationMetrics struct {
	RequestsTotal      uint64        `json:"requests_total"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	ResponseTime       time.Duration `json:"response_time"`
	ErrorRate          float64       `json:"error_rate"`
	ActiveConnections  int           `json:"active_connections"`
	Uptime            time.Duration `json:"uptime"`
	Version           string        `json:"version"`
}

// DatabaseMetrics represents database metrics
type DatabaseMetrics struct {
	ConnectionsActive  int           `json:"connections_active"`
	ConnectionsIdle    int           `json:"connections_idle"`
	ConnectionsMax     int           `json:"connections_max"`
	QueriesTotal       uint64        `json:"queries_total"`
	QueriesPerSecond   float64       `json:"queries_per_second"`
	SlowQueries        uint64        `json:"slow_queries"`
	AvgQueryTime       time.Duration `json:"avg_query_time"`
	DeadlocksTotal     uint64        `json:"deadlocks_total"`
}

// WebSocketMetrics represents WebSocket metrics
type WebSocketMetrics struct {
	ActiveConnections int     `json:"active_connections"`
	TotalConnections  uint64  `json:"total_connections"`
	MessagesReceived  uint64  `json:"messages_received"`
	MessagesSent      uint64  `json:"messages_sent"`
	MessageRate       float64 `json:"message_rate"`
	AvgLatency        time.Duration `json:"avg_latency"`
	ErrorsTotal       uint64  `json:"errors_total"`
}

// BusinessMetrics represents business-specific metrics
type BusinessMetrics struct {
	UpdatesScheduled   uint64  `json:"updates_scheduled"`
	UpdatesCompleted   uint64  `json:"updates_completed"`
	UpdatesFailed      uint64  `json:"updates_failed"`
	UpdateSuccessRate  float64 `json:"update_success_rate"`
	ContainersManaged  int     `json:"containers_managed"`
	RegistriesMonitored int    `json:"registries_monitored"`
	NotificationsSent  uint64  `json:"notifications_sent"`
	AlertsTriggered    uint64  `json:"alerts_triggered"`
}