package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MonitoringMetrics represents container performance metrics stored in time-series format
type MonitoringMetrics struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ContainerID int       `json:"container_id" gorm:"not null;index:idx_monitoring_container_time,priority:1"`
	Timestamp   time.Time `json:"timestamp" gorm:"not null;index:idx_monitoring_container_time,priority:2;index:idx_monitoring_timestamp"`

	// CPU Metrics
	CPUPercent      float64 `json:"cpu_percent" gorm:"not null;default:0"`
	CPUUsage        uint64  `json:"cpu_usage"`
	SystemCPUUsage  uint64  `json:"system_cpu_usage"`
	OnlineCPUs      uint32  `json:"online_cpus"`
	ThrottledTime   uint64  `json:"throttled_time"`
	UserCPUTime     uint64  `json:"user_cpu_time"`
	SystemCPUTime   uint64  `json:"system_cpu_time"`

	// Memory Metrics
	MemoryPercent    float64 `json:"memory_percent" gorm:"not null;default:0"`
	MemoryUsage      uint64  `json:"memory_usage"`
	MemoryLimit      uint64  `json:"memory_limit"`
	MemoryMaxUsage   uint64  `json:"memory_max_usage"`
	MemoryCache      uint64  `json:"memory_cache"`
	MemoryRSS        uint64  `json:"memory_rss"`
	MemorySwap       uint64  `json:"memory_swap"`
	MemoryFailCount  uint64  `json:"memory_fail_count"`
	MemoryAvailable  uint64  `json:"memory_available"`
	MemoryWorkingSet uint64  `json:"memory_working_set"`

	// Network Metrics
	NetworkRxBytes   uint64 `json:"network_rx_bytes"`
	NetworkTxBytes   uint64 `json:"network_tx_bytes"`
	NetworkRxPackets uint64 `json:"network_rx_packets"`
	NetworkTxPackets uint64 `json:"network_tx_packets"`
	NetworkRxErrors  uint64 `json:"network_rx_errors"`
	NetworkTxErrors  uint64 `json:"network_tx_errors"`
	NetworkRxDropped uint64 `json:"network_rx_dropped"`
	NetworkTxDropped uint64 `json:"network_tx_dropped"`

	// Block I/O Metrics
	BlockReadBytes   uint64 `json:"block_read_bytes"`
	BlockWriteBytes  uint64 `json:"block_write_bytes"`
	BlockReadOps     uint64 `json:"block_read_ops"`
	BlockWriteOps    uint64 `json:"block_write_ops"`
	BlockTotalBytes  uint64 `json:"block_total_bytes"`
	BlockTotalOps    uint64 `json:"block_total_ops"`

	// Process Metrics
	PIDs      uint64 `json:"pids"`
	PIDsLimit uint64 `json:"pids_limit"`

	// Health and Performance Indicators
	OverallHealth   string  `json:"overall_health" gorm:"size:20;not null;default:'unknown';index:idx_monitoring_health"`
	CPUHealthStatus string  `json:"cpu_health_status" gorm:"size:20;not null;default:'unknown'"`
	MemHealthStatus string  `json:"mem_health_status" gorm:"size:20;not null;default:'unknown'"`
	IOHealthStatus  string  `json:"io_health_status" gorm:"size:20;not null;default:'unknown'"`
	Efficiency      float64 `json:"efficiency" gorm:"default:0"`

	// Derived metrics for analysis
	CPUTrend       string                `json:"cpu_trend" gorm:"size:20;default:'stable'"`
	MemoryTrend    string                `json:"memory_trend" gorm:"size:20;default:'stable'"`
	NetworkActivity string               `json:"network_activity" gorm:"size:20;default:'low'"`
	IOActivity     string                `json:"io_activity" gorm:"size:20;default:'low'"`

	// Extended metrics and metadata
	ExtendedMetrics *ExtendedMetricsData `json:"extended_metrics,omitempty" gorm:"type:jsonb"`
	Anomalies       *AnomaliesData       `json:"anomalies,omitempty" gorm:"type:jsonb"`
	Recommendations string               `json:"recommendations,omitempty" gorm:"type:jsonb"`

	// Metadata
	DataSource string `json:"data_source" gorm:"size:50;not null;default:'docker-stats'"`
	Version    string `json:"version" gorm:"size:20;default:'1.0'"`

	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Container *Container `json:"container,omitempty" gorm:"foreignKey:ContainerID"`
}

// ExtendedMetricsData represents additional detailed metrics
type ExtendedMetricsData struct {
	// CPU details
	CPUShares         int64             `json:"cpu_shares,omitempty"`
	CPUQuota          int64             `json:"cpu_quota,omitempty"`
	CPUPeriod         int64             `json:"cpu_period,omitempty"`
	CPUSetCPUs        string            `json:"cpuset_cpus,omitempty"`
	PerCPUUsage       []uint64          `json:"per_cpu_usage,omitempty"`

	// Memory details
	KernelMemory      uint64            `json:"kernel_memory,omitempty"`
	MemoryReservation uint64            `json:"memory_reservation,omitempty"`
	MemorySwapLimit   uint64            `json:"memory_swap_limit,omitempty"`
	MemoryStats       map[string]uint64 `json:"memory_stats,omitempty"`

	// Network interface details
	NetworkInterfaces map[string]NetworkInterfaceMetrics `json:"network_interfaces,omitempty"`

	// Block I/O device details
	BlockIODevices    map[string]BlockIODeviceMetrics    `json:"block_io_devices,omitempty"`

	// Process information
	ProcessCount      int               `json:"process_count,omitempty"`
	ThreadCount       int               `json:"thread_count,omitempty"`
	FileDescriptors   int               `json:"file_descriptors,omitempty"`
	SocketCount       int               `json:"socket_count,omitempty"`

	// Resource limits and usage
	Ulimits           map[string]int64  `json:"ulimits,omitempty"`
	CgroupMemory      CgroupMemoryStats `json:"cgroup_memory,omitempty"`
	CgroupCPU         CgroupCPUStats    `json:"cgroup_cpu,omitempty"`
}

// NetworkInterfaceMetrics represents metrics for a specific network interface
type NetworkInterfaceMetrics struct {
	Name         string `json:"name"`
	RxBytes      uint64 `json:"rx_bytes"`
	TxBytes      uint64 `json:"tx_bytes"`
	RxPackets    uint64 `json:"rx_packets"`
	TxPackets    uint64 `json:"tx_packets"`
	RxErrors     uint64 `json:"rx_errors"`
	TxErrors     uint64 `json:"tx_errors"`
	RxDropped    uint64 `json:"rx_dropped"`
	TxDropped    uint64 `json:"tx_dropped"`
	MTU          int    `json:"mtu,omitempty"`
	Speed        int64  `json:"speed,omitempty"`
}

// BlockIODeviceMetrics represents metrics for a specific block I/O device
type BlockIODeviceMetrics struct {
	Major      uint64 `json:"major"`
	Minor      uint64 `json:"minor"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadOps    uint64 `json:"read_ops"`
	WriteOps   uint64 `json:"write_ops"`
}

// CgroupMemoryStats represents cgroup memory statistics
type CgroupMemoryStats struct {
	Cache                     uint64 `json:"cache,omitempty"`
	RSS                       uint64 `json:"rss,omitempty"`
	RSSHuge                   uint64 `json:"rss_huge,omitempty"`
	MappedFile               uint64 `json:"mapped_file,omitempty"`
	Dirty                    uint64 `json:"dirty,omitempty"`
	Writeback                uint64 `json:"writeback,omitempty"`
	PgPgIn                   uint64 `json:"pgpgin,omitempty"`
	PgPgOut                  uint64 `json:"pgpgout,omitempty"`
	PgFault                  uint64 `json:"pgfault,omitempty"`
	PgMajFault               uint64 `json:"pgmajfault,omitempty"`
	InactiveAnon             uint64 `json:"inactive_anon,omitempty"`
	ActiveAnon               uint64 `json:"active_anon,omitempty"`
	InactiveFile             uint64 `json:"inactive_file,omitempty"`
	ActiveFile               uint64 `json:"active_file,omitempty"`
	Unevictable              uint64 `json:"unevictable,omitempty"`
	HierarchicalMemoryLimit  uint64 `json:"hierarchical_memory_limit,omitempty"`
	TotalCache               uint64 `json:"total_cache,omitempty"`
	TotalRSS                 uint64 `json:"total_rss,omitempty"`
	TotalMappedFile          uint64 `json:"total_mapped_file,omitempty"`
	TotalPgPgIn              uint64 `json:"total_pgpgin,omitempty"`
	TotalPgPgOut             uint64 `json:"total_pgpgout,omitempty"`
	TotalPgFault             uint64 `json:"total_pgfault,omitempty"`
	TotalPgMajFault          uint64 `json:"total_pgmajfault,omitempty"`
	TotalInactiveAnon        uint64 `json:"total_inactive_anon,omitempty"`
	TotalActiveAnon          uint64 `json:"total_active_anon,omitempty"`
	TotalInactiveFile        uint64 `json:"total_inactive_file,omitempty"`
	TotalActiveFile          uint64 `json:"total_active_file,omitempty"`
	TotalUnevictable         uint64 `json:"total_unevictable,omitempty"`
}

// CgroupCPUStats represents cgroup CPU statistics
type CgroupCPUStats struct {
	CPUAcctUsage           uint64 `json:"cpuacct_usage,omitempty"`
	CPUAcctUsageInUserMode uint64 `json:"cpuacct_usage_in_user_mode,omitempty"`
	CPUAcctUsageInKernelMode uint64 `json:"cpuacct_usage_in_kernel_mode,omitempty"`
	CPUAcctUsagePerCPU     []uint64 `json:"cpuacct_usage_per_cpu,omitempty"`
	ThrottlingPeriods      uint64 `json:"throttling_periods,omitempty"`
	ThrottledPeriods       uint64 `json:"throttled_periods,omitempty"`
	ThrottledTime          uint64 `json:"throttled_time,omitempty"`
}

// AnomaliesData represents detected anomalies in the metrics
type AnomaliesData struct {
	CPUAnomalies     []AnomalyEntry `json:"cpu_anomalies,omitempty"`
	MemoryAnomalies  []AnomalyEntry `json:"memory_anomalies,omitempty"`
	NetworkAnomalies []AnomalyEntry `json:"network_anomalies,omitempty"`
	IOAnomalies      []AnomalyEntry `json:"io_anomalies,omitempty"`
	SystemAnomalies  []AnomalyEntry `json:"system_anomalies,omitempty"`
	Score            float64        `json:"score"` // Overall anomaly score (0-100)
}

// AnomalyEntry represents a single detected anomaly
type AnomalyEntry struct {
	Type        string    `json:"type"`        // spike, drop, trend, outlier
	Severity    string    `json:"severity"`    // low, medium, high, critical
	Description string    `json:"description"`
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected,omitempty"`
	Deviation   float64   `json:"deviation,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// MetricsAggregation represents aggregated metrics for analysis
type MetricsAggregation struct {
	ContainerID     int       `json:"container_id"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Interval        string    `json:"interval"` // minute, hour, day, week, month
	SampleCount     int       `json:"sample_count"`

	// CPU aggregations
	CPUPercent      MetricStats `json:"cpu_percent"`
	CPUUsage        MetricStats `json:"cpu_usage"`
	ThrottledTime   MetricStats `json:"throttled_time"`

	// Memory aggregations
	MemoryPercent   MetricStats `json:"memory_percent"`
	MemoryUsage     MetricStats `json:"memory_usage"`
	MemoryCache     MetricStats `json:"memory_cache"`
	MemoryFailCount MetricStats `json:"memory_fail_count"`

	// Network aggregations
	NetworkRxBytes  MetricStats `json:"network_rx_bytes"`
	NetworkTxBytes  MetricStats `json:"network_tx_bytes"`
	NetworkErrors   MetricStats `json:"network_errors"`

	// Block I/O aggregations
	BlockReadBytes  MetricStats `json:"block_read_bytes"`
	BlockWriteBytes MetricStats `json:"block_write_bytes"`
	BlockTotalOps   MetricStats `json:"block_total_ops"`

	// Health statistics
	HealthDistribution map[string]int `json:"health_distribution"`
	EfficiencyScore    MetricStats    `json:"efficiency_score"`
}

// MetricStats represents statistical data for a metric
type MetricStats struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Avg     float64 `json:"avg"`
	Median  float64 `json:"median"`
	P95     float64 `json:"p95"`
	P99     float64 `json:"p99"`
	StdDev  float64 `json:"std_dev"`
	Count   int     `json:"count"`
	Sum     float64 `json:"sum"`
	Trend   string  `json:"trend"` // increasing, decreasing, stable, volatile
}

// MonitoringMetricsFilter represents filters for querying monitoring metrics
type MonitoringMetricsFilter struct {
	ContainerID     *int      `json:"container_id,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	OverallHealth   string    `json:"overall_health,omitempty"`
	CPUPercentMin   *float64  `json:"cpu_percent_min,omitempty"`
	CPUPercentMax   *float64  `json:"cpu_percent_max,omitempty"`
	MemoryPercentMin *float64  `json:"memory_percent_min,omitempty"`
	MemoryPercentMax *float64  `json:"memory_percent_max,omitempty"`
	DataSource      string    `json:"data_source,omitempty"`
	HasAnomalies    *bool     `json:"has_anomalies,omitempty"`
	Limit           int       `json:"limit,omitempty"`
	Offset          int       `json:"offset,omitempty"`
	OrderBy         string    `json:"order_by,omitempty"`
	GroupBy         string    `json:"group_by,omitempty"`
}

// TableName returns the table name for MonitoringMetrics model
func (MonitoringMetrics) TableName() string {
	return "monitoring_metrics"
}

// GORM hooks

// BeforeCreate hook for MonitoringMetrics model
func (mm *MonitoringMetrics) BeforeCreate(tx *gorm.DB) error {
	// Set default values
	if mm.DataSource == "" {
		mm.DataSource = "docker-stats"
	}
	if mm.Version == "" {
		mm.Version = "1.0"
	}
	if mm.OverallHealth == "" {
		mm.OverallHealth = "unknown"
	}
	if mm.CPUHealthStatus == "" {
		mm.CPUHealthStatus = "unknown"
	}
	if mm.MemHealthStatus == "" {
		mm.MemHealthStatus = "unknown"
	}
	if mm.IOHealthStatus == "" {
		mm.IOHealthStatus = "unknown"
	}

	// Calculate derived metrics
	mm.calculateDerivedMetrics()

	return nil
}

// BeforeSave hook for MonitoringMetrics model
func (mm *MonitoringMetrics) BeforeSave(tx *gorm.DB) error {
	// Validate JSON fields
	if mm.ExtendedMetrics != nil {
		if _, err := json.Marshal(mm.ExtendedMetrics); err != nil {
			return fmt.Errorf("invalid extended_metrics JSON: %w", err)
		}
	}

	if mm.Anomalies != nil {
		if _, err := json.Marshal(mm.Anomalies); err != nil {
			return fmt.Errorf("invalid anomalies JSON: %w", err)
		}
	}

	// Validate metric ranges
	if mm.CPUPercent < 0 || mm.CPUPercent > 100 {
		mm.CPUPercent = 0
	}
	if mm.MemoryPercent < 0 || mm.MemoryPercent > 100 {
		mm.MemoryPercent = 0
	}
	if mm.Efficiency < 0 || mm.Efficiency > 100 {
		mm.Efficiency = 0
	}

	return nil
}

// Derived metrics calculation

// calculateDerivedMetrics calculates derived metrics and trends
func (mm *MonitoringMetrics) calculateDerivedMetrics() {
	// Calculate network activity level
	totalNetworkBytes := mm.NetworkRxBytes + mm.NetworkTxBytes
	if totalNetworkBytes > 1e9 { // > 1GB
		mm.NetworkActivity = "high"
	} else if totalNetworkBytes > 1e8 { // > 100MB
		mm.NetworkActivity = "medium"
	} else {
		mm.NetworkActivity = "low"
	}

	// Calculate I/O activity level
	totalIOBytes := mm.BlockReadBytes + mm.BlockWriteBytes
	if totalIOBytes > 1e9 { // > 1GB
		mm.IOActivity = "high"
	} else if totalIOBytes > 1e8 { // > 100MB
		mm.IOActivity = "medium"
	} else {
		mm.IOActivity = "low"
	}

	// Calculate efficiency score
	cpuEfficiency := 100.0 - mm.CPUPercent
	memoryEfficiency := 100.0 - mm.MemoryPercent
	mm.Efficiency = (cpuEfficiency + memoryEfficiency) / 2

	// Set trends (would be calculated with historical data in practice)
	mm.CPUTrend = "stable"
	mm.MemoryTrend = "stable"
}

// Analysis and utility methods

// IsHealthy checks if the container metrics indicate healthy state
func (mm *MonitoringMetrics) IsHealthy() bool {
	return mm.OverallHealth == "healthy"
}

// HasCriticalIssues checks if there are critical performance issues
func (mm *MonitoringMetrics) HasCriticalIssues() bool {
	return mm.OverallHealth == "critical" ||
		mm.CPUPercent > 95 ||
		mm.MemoryPercent > 95 ||
		mm.MemoryFailCount > 0
}

// HasWarnings checks if there are performance warnings
func (mm *MonitoringMetrics) HasWarnings() bool {
	return mm.OverallHealth == "warning" ||
		mm.CPUPercent > 80 ||
		mm.MemoryPercent > 80 ||
		(mm.NetworkRxErrors+mm.NetworkTxErrors) > 0
}

// GetResourceUtilization returns overall resource utilization percentage
func (mm *MonitoringMetrics) GetResourceUtilization() float64 {
	return (mm.CPUPercent + mm.MemoryPercent) / 2
}

// GetFormattedMemoryUsage returns human-readable memory usage
func (mm *MonitoringMetrics) GetFormattedMemoryUsage() string {
	return fmt.Sprintf("%.1f%% (%s / %s)",
		mm.MemoryPercent,
		formatBytes(mm.MemoryUsage),
		formatBytes(mm.MemoryLimit))
}

// GetFormattedNetworkIO returns human-readable network I/O
func (mm *MonitoringMetrics) GetFormattedNetworkIO() string {
	return fmt.Sprintf("RX: %s, TX: %s",
		formatBytes(mm.NetworkRxBytes),
		formatBytes(mm.NetworkTxBytes))
}

// GetFormattedBlockIO returns human-readable block I/O
func (mm *MonitoringMetrics) GetFormattedBlockIO() string {
	return fmt.Sprintf("Read: %s, Write: %s",
		formatBytes(mm.BlockReadBytes),
		formatBytes(mm.BlockWriteBytes))
}

// DetectAnomalies detects anomalies in the current metrics (simplified)
func (mm *MonitoringMetrics) DetectAnomalies(previous *MonitoringMetrics) {
	if previous == nil {
		return
	}

	anomalies := &AnomaliesData{
		CPUAnomalies:     []AnomalyEntry{},
		MemoryAnomalies:  []AnomalyEntry{},
		NetworkAnomalies: []AnomalyEntry{},
		IOAnomalies:      []AnomalyEntry{},
		SystemAnomalies:  []AnomalyEntry{},
	}

	// Check for CPU spikes
	cpuDiff := mm.CPUPercent - previous.CPUPercent
	if cpuDiff > 50 {
		anomalies.CPUAnomalies = append(anomalies.CPUAnomalies, AnomalyEntry{
			Type:        "spike",
			Severity:    "high",
			Description: fmt.Sprintf("CPU usage spiked by %.1f%%", cpuDiff),
			Value:       mm.CPUPercent,
			Expected:    previous.CPUPercent,
			Deviation:   cpuDiff,
			Timestamp:   mm.Timestamp,
		})
	}

	// Check for memory spikes
	memDiff := mm.MemoryPercent - previous.MemoryPercent
	if memDiff > 30 {
		anomalies.MemoryAnomalies = append(anomalies.MemoryAnomalies, AnomalyEntry{
			Type:        "spike",
			Severity:    "high",
			Description: fmt.Sprintf("Memory usage spiked by %.1f%%", memDiff),
			Value:       mm.MemoryPercent,
			Expected:    previous.MemoryPercent,
			Deviation:   memDiff,
			Timestamp:   mm.Timestamp,
		})
	}

	// Check for memory failures
	if mm.MemoryFailCount > previous.MemoryFailCount {
		anomalies.MemoryAnomalies = append(anomalies.MemoryAnomalies, AnomalyEntry{
			Type:        "outlier",
			Severity:    "critical",
			Description: "Memory allocation failures detected",
			Value:       float64(mm.MemoryFailCount),
			Expected:    float64(previous.MemoryFailCount),
			Timestamp:   mm.Timestamp,
		})
	}

	// Calculate overall anomaly score
	score := 0.0
	totalAnomalies := len(anomalies.CPUAnomalies) + len(anomalies.MemoryAnomalies) +
	                  len(anomalies.NetworkAnomalies) + len(anomalies.IOAnomalies) +
	                  len(anomalies.SystemAnomalies)
	if totalAnomalies > 0 {
		score = float64(totalAnomalies) * 10.0 // Simple scoring
		if score > 100 {
			score = 100
		}
	}
	anomalies.Score = score

	if totalAnomalies > 0 {
		mm.Anomalies = anomalies
	}
}

// Data retention and cleanup methods

// GetRetentionPolicy returns the data retention policy based on age
func (mm *MonitoringMetrics) GetRetentionPolicy() time.Duration {
	age := time.Since(mm.Timestamp)

	// Keep detailed data for recent metrics, aggregate older data
	if age < 24*time.Hour {
		return 7 * 24 * time.Hour // Keep 7 days of hourly data
	} else if age < 7*24*time.Hour {
		return 30 * 24 * time.Hour // Keep 30 days of daily data
	} else {
		return 365 * 24 * time.Hour // Keep 1 year of weekly data
	}
}

// ShouldAggregate checks if this metric should be aggregated with others
func (mm *MonitoringMetrics) ShouldAggregate(interval string) bool {
	age := time.Since(mm.Timestamp)

	switch interval {
	case "hour":
		return age > 24*time.Hour
	case "day":
		return age > 7*24*time.Hour
	case "week":
		return age > 30*24*time.Hour
	case "month":
		return age > 365*24*time.Hour
	default:
		return false
	}
}

// Cleanup queries for data retention

// GetCleanupQuery returns query for cleaning up old metrics
func GetCleanupQuery(db *gorm.DB, containerID int, retentionPeriod time.Duration) *gorm.DB {
	cutoff := time.Now().Add(-retentionPeriod)
	return db.Where("container_id = ? AND timestamp < ?", containerID, cutoff)
}

// GetAggregationQuery returns query for metrics that need aggregation
func GetAggregationQuery(db *gorm.DB, containerID int, interval string, cutoff time.Time) *gorm.DB {
	return db.Where("container_id = ? AND timestamp < ?", containerID, cutoff).Order("timestamp ASC")
}

// Valid health statuses and trends
func GetValidHealthStatuses() []string {
	return []string{"healthy", "warning", "critical", "unknown"}
}

func GetValidTrends() []string {
	return []string{"increasing", "decreasing", "stable", "volatile"}
}

func GetValidActivityLevels() []string {
	return []string{"low", "medium", "high"}
}

// formatBytes formats bytes in human-readable format (defined in container.go)