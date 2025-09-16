package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/docker/docker/api/types"
)

// ContainerMetrics represents comprehensive container metrics
type ContainerMetrics struct {
	ContainerID     string            `json:"container_id"`
	Name            string            `json:"name"`
	Timestamp       time.Time         `json:"timestamp"`
	CPU             CPUMetrics        `json:"cpu"`
	Memory          MemoryMetrics     `json:"memory"`
	Network         NetworkMetrics    `json:"network"`
	BlockIO         BlockIOMetrics    `json:"block_io"`
	PIDs            PIDMetrics        `json:"pids"`
	PerformanceInfo PerformanceInfo   `json:"performance_info"`
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	CPUPercent      float64 `json:"cpu_percent"`
	SystemCPUUsage  uint64  `json:"system_cpu_usage"`
	OnlineCPUs      uint32  `json:"online_cpus"`
	ThrottledTime   uint64  `json:"throttled_time"`
	CPUShares       int64   `json:"cpu_shares"`
	CPUQuota        int64   `json:"cpu_quota"`
	CPUPeriod       int64   `json:"cpu_period"`
	Usage           uint64  `json:"usage"`
	UserUsage       uint64  `json:"user_usage"`
	SystemUsage     uint64  `json:"system_usage"`
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	Usage            uint64  `json:"usage"`
	MaxUsage         uint64  `json:"max_usage"`
	Limit            uint64  `json:"limit"`
	MemoryPercent    float64 `json:"memory_percent"`
	Cache            uint64  `json:"cache"`
	RSS              uint64  `json:"rss"`
	MappedFile       uint64  `json:"mapped_file"`
	Swap             uint64  `json:"swap"`
	SwapLimit        uint64  `json:"swap_limit"`
	KernelMemory     uint64  `json:"kernel_memory"`
	MemoryFailCount  uint64  `json:"memory_fail_count"`
	Available        uint64  `json:"available"`
	WorkingSet       uint64  `json:"working_set"`
}

// NetworkMetrics represents network I/O metrics
type NetworkMetrics struct {
	RxBytes   uint64             `json:"rx_bytes"`
	RxPackets uint64             `json:"rx_packets"`
	RxErrors  uint64             `json:"rx_errors"`
	RxDropped uint64             `json:"rx_dropped"`
	TxBytes   uint64             `json:"tx_bytes"`
	TxPackets uint64             `json:"tx_packets"`
	TxErrors  uint64             `json:"tx_errors"`
	TxDropped uint64             `json:"tx_dropped"`
	Networks  map[string]Network `json:"networks"`
}

// Network represents individual network interface metrics
type Network struct {
	Name      string `json:"name"`
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	RxDropped uint64 `json:"rx_dropped"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
	TxDropped uint64 `json:"tx_dropped"`
}

// BlockIOMetrics represents block I/O metrics
type BlockIOMetrics struct {
	ReadBytes       uint64              `json:"read_bytes"`
	WriteBytes      uint64              `json:"write_bytes"`
	ReadOperations  uint64              `json:"read_operations"`
	WriteOperations uint64              `json:"write_operations"`
	TotalBytes      uint64              `json:"total_bytes"`
	TotalOperations uint64              `json:"total_operations"`
	IoServicedRecursive []IOServiceBytes `json:"io_serviced_recursive"`
	IoServiceTimeRecursive []IOServiceBytes `json:"io_service_time_recursive"`
}

// IOServiceBytes represents I/O service statistics
type IOServiceBytes struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

// PIDMetrics represents process ID metrics
type PIDMetrics struct {
	Current uint64 `json:"current"`
	Limit   uint64 `json:"limit"`
}

// PerformanceInfo represents derived performance information
type PerformanceInfo struct {
	OverallHealth   string  `json:"overall_health"`   // healthy, warning, critical
	CPUHealthStatus string  `json:"cpu_health_status"`
	MemHealthStatus string  `json:"mem_health_status"`
	IOHealthStatus  string  `json:"io_health_status"`
	Efficiency      float64 `json:"efficiency"`       // Overall efficiency score (0-100)
	Recommendations []string `json:"recommendations"`
}

// MetricsHistory represents historical metrics for trend analysis
type MetricsHistory struct {
	ContainerID string             `json:"container_id"`
	Metrics     []ContainerMetrics `json:"metrics"`
	Duration    time.Duration      `json:"duration"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	Summary     MetricsSummary     `json:"summary"`
}

// MetricsSummary represents aggregated metrics summary
type MetricsSummary struct {
	AvgCPUPercent    float64 `json:"avg_cpu_percent"`
	MaxCPUPercent    float64 `json:"max_cpu_percent"`
	AvgMemoryPercent float64 `json:"avg_memory_percent"`
	MaxMemoryPercent float64 `json:"max_memory_percent"`
	TotalNetworkRx   uint64  `json:"total_network_rx"`
	TotalNetworkTx   uint64  `json:"total_network_tx"`
	TotalDiskRead    uint64  `json:"total_disk_read"`
	TotalDiskWrite   uint64  `json:"total_disk_write"`
	SampleCount      int     `json:"sample_count"`
}

// Container monitoring operations

// GetContainerStats gets real-time stats for a container
func (d *DockerClient) GetContainerStats(ctx context.Context, containerID string) (*types.StatsJSON, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	stats, err := d.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for container %s: %w", containerID, err)
	}
	defer stats.Body.Close()

	var statsJSON types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &statsJSON, nil
}

// GetContainerMetrics gets processed container metrics
func (d *DockerClient) GetContainerMetrics(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	stats, err := d.GetContainerStats(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Get container info for name
	containerInfo, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	return d.processStatsToMetrics(stats, containerInfo.Name), nil
}

// StreamContainerStats streams container stats in real-time
func (d *DockerClient) StreamContainerStats(ctx context.Context, containerID string) (<-chan *ContainerMetrics, <-chan error) {
	metricsChan := make(chan *ContainerMetrics, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(metricsChan)
		defer close(errChan)

		stats, err := d.client.ContainerStats(ctx, containerID, true)
		if err != nil {
			errChan <- fmt.Errorf("failed to start stats stream: %w", err)
			return
		}
		defer stats.Body.Close()

		// Get container info once
		containerInfo, err := d.GetContainer(ctx, containerID)
		if err != nil {
			errChan <- fmt.Errorf("failed to get container info: %w", err)
			return
		}

		decoder := json.NewDecoder(stats.Body)
		for {
			var statsJSON types.StatsJSON
			if err := decoder.Decode(&statsJSON); err != nil {
				if err == io.EOF {
					return
				}
				errChan <- fmt.Errorf("failed to decode stats: %w", err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case metricsChan <- d.processStatsToMetrics(&statsJSON, containerInfo.Name):
			}
		}
	}()

	return metricsChan, errChan
}

// processStatsToMetrics converts Docker stats to our metrics format
func (d *DockerClient) processStatsToMetrics(stats *types.StatsJSON, containerName string) *ContainerMetrics {
	metrics := &ContainerMetrics{
		ContainerID: stats.ID,
		Name:        containerName,
		Timestamp:   stats.Read,
	}

	// Process CPU metrics
	metrics.CPU = d.processCPUStats(stats)

	// Process memory metrics
	metrics.Memory = d.processMemoryStats(stats)

	// Process network metrics
	metrics.Network = d.processNetworkStats(stats)

	// Process block I/O metrics
	metrics.BlockIO = d.processBlockIOStats(stats)

	// Process PID metrics
	metrics.PIDs = d.processPIDStats(stats)

	// Generate performance information
	metrics.PerformanceInfo = d.generatePerformanceInfo(metrics)

	return metrics
}

// processCPUStats processes CPU statistics
func (d *DockerClient) processCPUStats(stats *types.StatsJSON) CPUMetrics {
	cpuMetrics := CPUMetrics{
		SystemCPUUsage: stats.CPUStats.SystemUsage,
		OnlineCPUs:     stats.CPUStats.OnlineCPUs,
	}

	// Calculate CPU percentage
	if len(stats.CPUStats.CPUUsage.PercpuUsage) > 0 {
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

		if systemDelta > 0 && cpuDelta > 0 {
			cpuMetrics.CPUPercent = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
		}
	}

	// Set CPU constraints
	if stats.CPUStats.CPUUsage.TotalUsage > 0 {
		cpuMetrics.Usage = stats.CPUStats.CPUUsage.TotalUsage
		cpuMetrics.UserUsage = stats.CPUStats.CPUUsage.UsageInUsermode
		cpuMetrics.SystemUsage = stats.CPUStats.CPUUsage.UsageInKernelmode
	}

	// Throttling information
	if stats.CPUStats.ThrottlingData.ThrottledTime > 0 {
		cpuMetrics.ThrottledTime = stats.CPUStats.ThrottlingData.ThrottledTime
	}

	return cpuMetrics
}

// processMemoryStats processes memory statistics
func (d *DockerClient) processMemoryStats(stats *types.StatsJSON) MemoryMetrics {
	memMetrics := MemoryMetrics{
		Usage:    stats.MemoryStats.Usage,
		MaxUsage: stats.MemoryStats.MaxUsage,
		Limit:    stats.MemoryStats.Limit,
	}

	// Calculate memory percentage
	if stats.MemoryStats.Limit > 0 {
		memMetrics.MemoryPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	// Process detailed memory stats
	if stats.MemoryStats.Stats != nil {
		if cache, ok := stats.MemoryStats.Stats["cache"]; ok {
			memMetrics.Cache = cache
		}
		if rss, ok := stats.MemoryStats.Stats["rss"]; ok {
			memMetrics.RSS = rss
		}
		if mappedFile, ok := stats.MemoryStats.Stats["mapped_file"]; ok {
			memMetrics.MappedFile = mappedFile
		}
		if swap, ok := stats.MemoryStats.Stats["swap"]; ok {
			memMetrics.Swap = swap
		}
		if failcnt, ok := stats.MemoryStats.Stats["failcnt"]; ok {
			memMetrics.MemoryFailCount = failcnt
		}
	}

	// Calculate available memory
	memMetrics.Available = memMetrics.Limit - memMetrics.Usage
	memMetrics.WorkingSet = memMetrics.Usage - memMetrics.Cache

	return memMetrics
}

// processNetworkStats processes network statistics
func (d *DockerClient) processNetworkStats(stats *types.StatsJSON) NetworkMetrics {
	netMetrics := NetworkMetrics{
		Networks: make(map[string]Network),
	}

	// Aggregate network stats across all interfaces
	for ifaceName, ifaceStats := range stats.Networks {
		network := Network{
			Name:      ifaceName,
			RxBytes:   ifaceStats.RxBytes,
			RxPackets: ifaceStats.RxPackets,
			RxErrors:  ifaceStats.RxErrors,
			RxDropped: ifaceStats.RxDropped,
			TxBytes:   ifaceStats.TxBytes,
			TxPackets: ifaceStats.TxPackets,
			TxErrors:  ifaceStats.TxErrors,
			TxDropped: ifaceStats.TxDropped,
		}

		netMetrics.Networks[ifaceName] = network

		// Aggregate totals
		netMetrics.RxBytes += ifaceStats.RxBytes
		netMetrics.RxPackets += ifaceStats.RxPackets
		netMetrics.RxErrors += ifaceStats.RxErrors
		netMetrics.RxDropped += ifaceStats.RxDropped
		netMetrics.TxBytes += ifaceStats.TxBytes
		netMetrics.TxPackets += ifaceStats.TxPackets
		netMetrics.TxErrors += ifaceStats.TxErrors
		netMetrics.TxDropped += ifaceStats.TxDropped
	}

	return netMetrics
}

// processBlockIOStats processes block I/O statistics
func (d *DockerClient) processBlockIOStats(stats *types.StatsJSON) BlockIOMetrics {
	ioMetrics := BlockIOMetrics{}

	// Process I/O service bytes
	for _, blkioStatEntry := range stats.BlkioStats.IoServiceBytesRecursive {
		switch blkioStatEntry.Op {
		case "Read":
			ioMetrics.ReadBytes += blkioStatEntry.Value
		case "Write":
			ioMetrics.WriteBytes += blkioStatEntry.Value
		}
		ioMetrics.TotalBytes += blkioStatEntry.Value

		ioMetrics.IoServicedRecursive = append(ioMetrics.IoServicedRecursive, IOServiceBytes{
			Major: blkioStatEntry.Major,
			Minor: blkioStatEntry.Minor,
			Op:    blkioStatEntry.Op,
			Value: blkioStatEntry.Value,
		})
	}

	// Process I/O serviced operations
	for _, blkioStatEntry := range stats.BlkioStats.IoServicedRecursive {
		switch blkioStatEntry.Op {
		case "Read":
			ioMetrics.ReadOperations += blkioStatEntry.Value
		case "Write":
			ioMetrics.WriteOperations += blkioStatEntry.Value
		}
		ioMetrics.TotalOperations += blkioStatEntry.Value
	}

	return ioMetrics
}

// processPIDStats processes PID statistics
func (d *DockerClient) processPIDStats(stats *types.StatsJSON) PIDMetrics {
	return PIDMetrics{
		Current: stats.PidsStats.Current,
		Limit:   stats.PidsStats.Limit,
	}
}

// generatePerformanceInfo generates performance analysis
func (d *DockerClient) generatePerformanceInfo(metrics *ContainerMetrics) PerformanceInfo {
	perf := PerformanceInfo{
		Recommendations: []string{},
	}

	// Analyze CPU health
	if metrics.CPU.CPUPercent > 80 {
		perf.CPUHealthStatus = "critical"
		perf.Recommendations = append(perf.Recommendations, "High CPU usage detected. Consider scaling or optimizing the application.")
	} else if metrics.CPU.CPUPercent > 60 {
		perf.CPUHealthStatus = "warning"
		perf.Recommendations = append(perf.Recommendations, "Moderate CPU usage. Monitor for trends.")
	} else {
		perf.CPUHealthStatus = "healthy"
	}

	// Analyze memory health
	if metrics.Memory.MemoryPercent > 90 {
		perf.MemHealthStatus = "critical"
		perf.Recommendations = append(perf.Recommendations, "Very high memory usage. Risk of OOM kills.")
	} else if metrics.Memory.MemoryPercent > 75 {
		perf.MemHealthStatus = "warning"
		perf.Recommendations = append(perf.Recommendations, "High memory usage. Consider increasing memory limits.")
	} else {
		perf.MemHealthStatus = "healthy"
	}

	// Analyze I/O health (simplified)
	totalIO := metrics.BlockIO.ReadBytes + metrics.BlockIO.WriteBytes
	if totalIO > 1e9 { // > 1GB
		perf.IOHealthStatus = "warning"
		perf.Recommendations = append(perf.Recommendations, "High I/O activity detected.")
	} else {
		perf.IOHealthStatus = "healthy"
	}

	// Calculate overall health
	healthScores := []string{perf.CPUHealthStatus, perf.MemHealthStatus, perf.IOHealthStatus}
	criticalCount := 0
	warningCount := 0

	for _, status := range healthScores {
		switch status {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		}
	}

	if criticalCount > 0 {
		perf.OverallHealth = "critical"
	} else if warningCount > 0 {
		perf.OverallHealth = "warning"
	} else {
		perf.OverallHealth = "healthy"
	}

	// Calculate efficiency score (simplified)
	cpuEfficiency := math.Max(0, 100-metrics.CPU.CPUPercent)
	memEfficiency := math.Max(0, 100-metrics.Memory.MemoryPercent)
	perf.Efficiency = (cpuEfficiency + memEfficiency) / 2

	return perf
}

// System resource monitoring

// GetSystemResources gets system resource information
func (d *DockerClient) GetSystemResources(ctx context.Context) (*types.Info, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	info, err := d.client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	return &info, nil
}

// GetAllContainerMetrics gets metrics for all running containers
func (d *DockerClient) GetAllContainerMetrics(ctx context.Context) (map[string]*ContainerMetrics, error) {
	containers, err := d.ListRunningContainers(ctx)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]*ContainerMetrics)
	for _, container := range containers {
		containerMetrics, err := d.GetContainerMetrics(ctx, container.ID)
		if err != nil {
			// Log error but continue with other containers
			continue
		}
		metrics[container.ID] = containerMetrics
	}

	return metrics, nil
}

// Container health checking

// CheckContainerHealth performs comprehensive health check
func (d *DockerClient) CheckContainerHealth(ctx context.Context, containerID string) (*ContainerHealthCheck, error) {
	// Get basic container info
	containerInfo, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Get metrics
	metrics, err := d.GetContainerMetrics(ctx, containerID)
	if err != nil {
		return nil, err
	}

	health := &ContainerHealthCheck{
		ContainerID:   containerID,
		Name:          containerInfo.Name,
		Timestamp:     time.Now(),
		Status:        containerInfo.State.Status,
		IsRunning:     containerInfo.State.Running,
		Metrics:       metrics,
		Issues:        []string{},
		Recommendations: []string{},
	}

	// Check for issues
	if !health.IsRunning {
		health.Issues = append(health.Issues, "Container is not running")
	}

	if containerInfo.State.ExitCode != 0 {
		health.Issues = append(health.Issues, fmt.Sprintf("Container exited with code %d", containerInfo.State.ExitCode))
	}

	if containerInfo.State.OOMKilled {
		health.Issues = append(health.Issues, "Container was killed due to out of memory")
		health.Recommendations = append(health.Recommendations, "Increase memory limits")
	}

	// Check metrics-based issues
	if metrics.CPU.CPUPercent > 95 {
		health.Issues = append(health.Issues, "Extremely high CPU usage")
	}

	if metrics.Memory.MemoryPercent > 95 {
		health.Issues = append(health.Issues, "Extremely high memory usage")
	}

	if metrics.Memory.MemoryFailCount > 0 {
		health.Issues = append(health.Issues, "Memory allocation failures detected")
	}

	// Set overall health
	if len(health.Issues) == 0 {
		health.OverallHealth = "healthy"
	} else if len(health.Issues) <= 2 {
		health.OverallHealth = "warning"
	} else {
		health.OverallHealth = "critical"
	}

	return health, nil
}

// ContainerHealthCheck represents comprehensive container health information
type ContainerHealthCheck struct {
	ContainerID     string             `json:"container_id"`
	Name            string             `json:"name"`
	Timestamp       time.Time          `json:"timestamp"`
	Status          string             `json:"status"`
	IsRunning       bool               `json:"is_running"`
	OverallHealth   string             `json:"overall_health"`
	Metrics         *ContainerMetrics  `json:"metrics"`
	Issues          []string           `json:"issues"`
	Recommendations []string           `json:"recommendations"`
}

// Historical metrics collection

// CollectMetricsHistory collects metrics over a period of time
func (d *DockerClient) CollectMetricsHistory(ctx context.Context, containerID string, duration time.Duration, interval time.Duration) (*MetricsHistory, error) {
	history := &MetricsHistory{
		ContainerID: containerID,
		Duration:    duration,
		StartTime:   time.Now(),
		Metrics:     []ContainerMetrics{},
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			history.EndTime = time.Now()
			history.Summary = d.calculateMetricsSummary(history.Metrics)
			return history, nil
		case <-ticker.C:
			metrics, err := d.GetContainerMetrics(ctx, containerID)
			if err != nil {
				// Log error but continue collecting
				continue
			}
			history.Metrics = append(history.Metrics, *metrics)
		}
	}
}

// calculateMetricsSummary calculates aggregated metrics summary
func (d *DockerClient) calculateMetricsSummary(metrics []ContainerMetrics) MetricsSummary {
	if len(metrics) == 0 {
		return MetricsSummary{}
	}

	summary := MetricsSummary{
		SampleCount: len(metrics),
	}

	var totalCPU, totalMemory float64
	var maxCPU, maxMemory float64

	for _, metric := range metrics {
		totalCPU += metric.CPU.CPUPercent
		totalMemory += metric.Memory.MemoryPercent

		if metric.CPU.CPUPercent > maxCPU {
			maxCPU = metric.CPU.CPUPercent
		}
		if metric.Memory.MemoryPercent > maxMemory {
			maxMemory = metric.Memory.MemoryPercent
		}

		summary.TotalNetworkRx += metric.Network.RxBytes
		summary.TotalNetworkTx += metric.Network.TxBytes
		summary.TotalDiskRead += metric.BlockIO.ReadBytes
		summary.TotalDiskWrite += metric.BlockIO.WriteBytes
	}

	summary.AvgCPUPercent = totalCPU / float64(len(metrics))
	summary.AvgMemoryPercent = totalMemory / float64(len(metrics))
	summary.MaxCPUPercent = maxCPU
	summary.MaxMemoryPercent = maxMemory

	return summary
}

// Utility functions

// IsHealthy checks if metrics indicate a healthy container
func (m *ContainerMetrics) IsHealthy() bool {
	return m.PerformanceInfo.OverallHealth == "healthy"
}

// GetCPUUsageMB returns CPU usage in a human-readable format
func (m *ContainerMetrics) GetCPUUsageFormatted() string {
	return fmt.Sprintf("%.2f%%", m.CPU.CPUPercent)
}

// GetMemoryUsageFormatted returns memory usage in a human-readable format
func (m *ContainerMetrics) GetMemoryUsageFormatted() string {
	return fmt.Sprintf("%.2f%% (%s / %s)",
		m.Memory.MemoryPercent,
		formatBytes(m.Memory.Usage),
		formatBytes(m.Memory.Limit))
}

// GetNetworkUsageFormatted returns network usage in a human-readable format
func (m *ContainerMetrics) GetNetworkUsageFormatted() string {
	return fmt.Sprintf("RX: %s, TX: %s",
		formatBytes(m.Network.RxBytes),
		formatBytes(m.Network.TxBytes))
}

// GetBlockIOFormatted returns block I/O in a human-readable format
func (m *ContainerMetrics) GetBlockIOFormatted() string {
	return fmt.Sprintf("Read: %s, Write: %s",
		formatBytes(m.BlockIO.ReadBytes),
		formatBytes(m.BlockIO.WriteBytes))
}

// formatBytes formats bytes in human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}