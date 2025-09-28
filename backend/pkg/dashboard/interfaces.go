package dashboard

import (
	"context"
	"time"
)

// DockerClientInterface defines Docker client operations
type DockerClientInterface interface {
	ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error)
	GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error)
	Ping(ctx context.Context) error
}

// SystemMonitorInterface defines system monitoring operations
type SystemMonitorInterface interface {
	GetCPUUsage() (float64, error)
	GetMemoryUsage() (*MemoryUsage, error)
	GetDiskUsage() (*DiskUsage, error)
	GetNetworkUsage() (*NetworkUsage, error)
}

// DatabaseInterface defines database operations for dashboard
type DatabaseInterface interface {
	GetSecurityVulnerabilities(ctx context.Context) ([]Vulnerability, error)
	GetLastSecurityScanTime(ctx context.Context) (time.Time, error)
	GetPendingUpdatesCount(ctx context.Context) (int, error)
	GetRecentUpdatesCount(ctx context.Context, since time.Time) (int, error)
	GetFailedUpdatesCount(ctx context.Context, since time.Time) (int, error)
	GetLastUpdateTime(ctx context.Context) (time.Time, error)
	IsAutoUpdateEnabled(ctx context.Context) (bool, error)
	Ping(ctx context.Context) error
}

// ContainerInfo represents Docker container information
type ContainerInfo struct {
	ID      string            `json:"id"`
	Names   []string          `json:"names"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Created time.Time         `json:"created"`
	Ports   []Port           `json:"ports"`
	Labels  map[string]string `json:"labels"`
	Health  *HealthInfo      `json:"health,omitempty"`
}

// Port represents container port mapping
type Port struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort,omitempty"`
	Type        string `json:"type"`
	IP          string `json:"ip,omitempty"`
}

// HealthInfo represents container health information
type HealthInfo struct {
	Status        string    `json:"status"`
	FailingStreak int       `json:"failingStreak"`
	Log           []LogEntry `json:"log,omitempty"`
}

// LogEntry represents health check log entry
type LogEntry struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	ExitCode int       `json:"exitCode"`
	Output   string    `json:"output"`
}

// ContainerStats represents container resource statistics
type ContainerStats struct {
	ContainerID string    `json:"containerId"`
	Name        string    `json:"name"`
	CPUPercent  float64   `json:"cpuPercent"`
	MemoryUsage uint64    `json:"memoryUsage"`
	MemoryLimit uint64    `json:"memoryLimit"`
	NetworkRx   uint64    `json:"networkRx"`
	NetworkTx   uint64    `json:"networkTx"`
	BlockRead   uint64    `json:"blockRead"`
	BlockWrite  uint64    `json:"blockWrite"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemoryUsage represents system memory usage
type MemoryUsage struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// DiskUsage represents system disk usage
type DiskUsage struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// NetworkUsage represents system network usage
type NetworkUsage struct {
	BytesIn    uint64 `json:"bytesIn"`
	BytesOut   uint64 `json:"bytesOut"`
	PacketsIn  uint64 `json:"packetsIn"`
	PacketsOut uint64 `json:"packetsOut"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string    `json:"id"`
	CVE         string    `json:"cve"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Package     string    `json:"package"`
	Version     string    `json:"version"`
	FixedIn     string    `json:"fixedIn,omitempty"`
	Container   string    `json:"container"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ContainerDetailStats provides detailed container statistics
type ContainerDetailStats struct {
	ContainerID    string              `json:"containerId"`
	Name           string              `json:"name"`
	Image          string              `json:"image"`
	Status         string              `json:"status"`
	State          string              `json:"state"`
	RestartCount   int                 `json:"restartCount"`
	CPUStats       CPUStats           `json:"cpuStats"`
	MemoryStats    MemoryStats        `json:"memoryStats"`
	NetworkStats   map[string]NetStats `json:"networkStats"`
	BlockIOStats   BlockIOStats       `json:"blockIOStats"`
	LastUpdated    time.Time          `json:"lastUpdated"`
}

// CPUStats provides detailed CPU statistics
type CPUStats struct {
	CPUUsage        CPUUsageStats `json:"cpuUsage"`
	SystemCPUUsage  uint64        `json:"systemCpuUsage"`
	OnlineCPUs      uint32        `json:"onlineCpus"`
	ThrottlingData  ThrottleData  `json:"throttlingData"`
}

// CPUUsageStats provides CPU usage details
type CPUUsageStats struct {
	TotalUsage        uint64   `json:"totalUsage"`
	PercpuUsage       []uint64 `json:"percpuUsage"`
	UsageInKernelmode uint64   `json:"usageInKernelmode"`
	UsageInUsermode   uint64   `json:"usageInUsermode"`
}

// ThrottleData provides CPU throttling information
type ThrottleData struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttledPeriods"`
	ThrottledTime    uint64 `json:"throttledTime"`
}

// MemoryStats provides detailed memory statistics
type MemoryStats struct {
	Usage       uint64            `json:"usage"`
	MaxUsage    uint64            `json:"maxUsage"`
	Stats       map[string]uint64 `json:"stats"`
	Limit       uint64            `json:"limit"`
}

// NetStats provides network statistics per interface
type NetStats struct {
	RxBytes   uint64 `json:"rxBytes"`
	RxPackets uint64 `json:"rxPackets"`
	RxErrors  uint64 `json:"rxErrors"`
	RxDropped uint64 `json:"rxDropped"`
	TxBytes   uint64 `json:"txBytes"`
	TxPackets uint64 `json:"txPackets"`
	TxErrors  uint64 `json:"txErrors"`
	TxDropped uint64 `json:"txDropped"`
}

// BlockIOStats provides block I/O statistics
type BlockIOStats struct {
	IoServiceBytesRecursive []BlkioStatEntry `json:"ioServiceBytesRecursive"`
	IoServicedRecursive     []BlkioStatEntry `json:"ioServicedRecursive"`
	IoQueueRecursive        []BlkioStatEntry `json:"ioQueueRecursive"`
	IoServiceTimeRecursive  []BlkioStatEntry `json:"ioServiceTimeRecursive"`
	IoWaitTimeRecursive     []BlkioStatEntry `json:"ioWaitTimeRecursive"`
	IoMergedRecursive       []BlkioStatEntry `json:"ioMergedRecursive"`
	IoTimeRecursive         []BlkioStatEntry `json:"ioTimeRecursive"`
	SectorsRecursive        []BlkioStatEntry `json:"sectorsRecursive"`
}

// BlkioStatEntry represents a block I/O stat entry
type BlkioStatEntry struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

// ResourceTrendData represents resource usage over time
type ResourceTrendData struct {
	Timestamps []time.Time `json:"timestamps"`
	Values     []float64   `json:"values"`
	MetricName string      `json:"metricName"`
	Unit       string      `json:"unit"`
}

// DashboardMetrics aggregates all dashboard metrics
type DashboardMetrics struct {
	Overview       *SystemOverview                     `json:"overview"`
	ContainerStats []ContainerDetailStats              `json:"containerStats"`
	ResourceTrends map[string]*ResourceTrendData       `json:"resourceTrends"`
	SecurityStatus *SecurityOverviewStatus             `json:"securityStatus"`
	UpdateActivity *UpdateOverviewActivity             `json:"updateActivity"`
	HealthMetrics  *SystemHealthOverview               `json:"healthMetrics"`
	Timestamp      time.Time                           `json:"timestamp"`
}