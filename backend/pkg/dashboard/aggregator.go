package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// DashboardAggregator handles multi-source data aggregation and caching for dashboard
type DashboardAggregator struct {
	dockerClient   DockerClientInterface
	systemMonitor  SystemMonitorInterface
	dbManager      DatabaseInterface
	redisClient    *redis.Client
	logger         *logrus.Logger
	cacheConfig    *CacheConfig
	mutex          sync.RWMutex
	lastUpdate     time.Time
	updateInterval time.Duration
}

// CacheConfig defines caching configuration
type CacheConfig struct {
	OverviewTTL        time.Duration
	ContainerStatsTTL  time.Duration
	ResourceMetricsTTL time.Duration
	SecurityStatusTTL  time.Duration
	UpdateActivityTTL  time.Duration
	HealthMetricsTTL   time.Duration
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		OverviewTTL:        30 * time.Second,
		ContainerStatsTTL:  10 * time.Second,
		ResourceMetricsTTL: 5 * time.Second,
		SecurityStatusTTL:  5 * time.Minute,
		UpdateActivityTTL:  1 * time.Minute,
		HealthMetricsTTL:   15 * time.Second,
	}
}

// NewDashboardAggregator creates a new dashboard aggregator
func NewDashboardAggregator(
	dockerClient DockerClientInterface,
	systemMonitor SystemMonitorInterface,
	dbManager DatabaseInterface,
	redisClient *redis.Client,
	logger *logrus.Logger,
) *DashboardAggregator {
	return &DashboardAggregator{
		dockerClient:   dockerClient,
		systemMonitor:  systemMonitor,
		dbManager:      dbManager,
		redisClient:    redisClient,
		logger:         logger,
		cacheConfig:    DefaultCacheConfig(),
		updateInterval: 30 * time.Second,
	}
}

// SystemOverview aggregates system overview data
type SystemOverview struct {
	ContainerStats    ContainerOverviewStats `json:"containerStats"`
	ResourceUsage     SystemResourceUsage    `json:"resourceUsage"`
	SecurityStatus    SecurityOverviewStatus `json:"securityStatus"`
	UpdateActivity    UpdateOverviewActivity `json:"updateActivity"`
	SystemHealth      SystemHealthOverview   `json:"systemHealth"`
	LastUpdated       time.Time              `json:"lastUpdated"`
}

// ContainerOverviewStats provides container statistics
type ContainerOverviewStats struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Stopped   int `json:"stopped"`
	Paused    int `json:"paused"`
	Restarting int `json:"restarting"`
	Unhealthy int `json:"unhealthy"`
}

// SystemResourceUsage provides system resource metrics
type SystemResourceUsage struct {
	CPU    ResourceMetric `json:"cpu"`
	Memory ResourceMetric `json:"memory"`
	Disk   ResourceMetric `json:"disk"`
	Network NetworkMetric `json:"network"`
}

// ResourceMetric represents a resource usage metric
type ResourceMetric struct {
	Used      float64 `json:"used"`
	Total     float64 `json:"total"`
	Percentage float64 `json:"percentage"`
	Unit      string  `json:"unit"`
}

// NetworkMetric represents network usage
type NetworkMetric struct {
	BytesIn   uint64  `json:"bytesIn"`
	BytesOut  uint64  `json:"bytesOut"`
	PacketsIn uint64  `json:"packetsIn"`
	PacketsOut uint64 `json:"packetsOut"`
}

// SecurityOverviewStatus provides security status overview
type SecurityOverviewStatus struct {
	VulnerabilitiesFound int       `json:"vulnerabilitiesFound"`
	CriticalVulns        int       `json:"criticalVulns"`
	HighVulns           int       `json:"highVulns"`
	MediumVulns         int       `json:"mediumVulns"`
	LowVulns            int       `json:"lowVulns"`
	LastScanTime        time.Time `json:"lastScanTime"`
	SecurityScore       float64   `json:"securityScore"`
}

// UpdateOverviewActivity provides update activity overview
type UpdateOverviewActivity struct {
	PendingUpdates    int       `json:"pendingUpdates"`
	RecentUpdates     int       `json:"recentUpdates"`
	FailedUpdates     int       `json:"failedUpdates"`
	LastUpdateTime    time.Time `json:"lastUpdateTime"`
	AutoUpdateEnabled bool      `json:"autoUpdateEnabled"`
}

// SystemHealthOverview provides system health metrics
type SystemHealthOverview struct {
	OverallStatus     string              `json:"overallStatus"`
	HealthyServices   int                 `json:"healthyServices"`
	UnhealthyServices int                 `json:"unhealthyServices"`
	HealthChecks      []HealthCheckStatus `json:"healthChecks"`
}

// HealthCheckStatus represents individual health check status
type HealthCheckStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// GetSystemOverview returns aggregated system overview data
func (da *DashboardAggregator) GetSystemOverview(ctx context.Context) (*SystemOverview, error) {
	cacheKey := "dashboard:overview"

	// Try cache first
	if cached, err := da.getCachedData(ctx, cacheKey); err == nil {
		var overview SystemOverview
		if json.Unmarshal([]byte(cached), &overview) == nil {
			return &overview, nil
		}
	}

	// Aggregate fresh data
	overview, err := da.aggregateSystemOverview(ctx)
	if err != nil {
		da.logger.WithError(err).Error("Failed to aggregate system overview")
		return nil, err
	}

	// Cache the result
	da.cacheData(ctx, cacheKey, overview, da.cacheConfig.OverviewTTL)

	return overview, nil
}

// aggregateSystemOverview performs actual data aggregation
func (da *DashboardAggregator) aggregateSystemOverview(ctx context.Context) (*SystemOverview, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	overview := &SystemOverview{
		LastUpdated: time.Now(),
	}

	// Aggregate container stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats, err := da.aggregateContainerStats(ctx)
		if err != nil {
			da.logger.WithError(err).Error("Failed to aggregate container stats")
			return
		}
		mutex.Lock()
		overview.ContainerStats = *stats
		mutex.Unlock()
	}()

	// Aggregate resource usage
	wg.Add(1)
	go func() {
		defer wg.Done()
		usage, err := da.aggregateResourceUsage(ctx)
		if err != nil {
			da.logger.WithError(err).Error("Failed to aggregate resource usage")
			return
		}
		mutex.Lock()
		overview.ResourceUsage = *usage
		mutex.Unlock()
	}()

	// Aggregate security status
	wg.Add(1)
	go func() {
		defer wg.Done()
		security, err := da.aggregateSecurityStatus(ctx)
		if err != nil {
			da.logger.WithError(err).Error("Failed to aggregate security status")
			return
		}
		mutex.Lock()
		overview.SecurityStatus = *security
		mutex.Unlock()
	}()

	// Aggregate update activity
	wg.Add(1)
	go func() {
		defer wg.Done()
		activity, err := da.aggregateUpdateActivity(ctx)
		if err != nil {
			da.logger.WithError(err).Error("Failed to aggregate update activity")
			return
		}
		mutex.Lock()
		overview.UpdateActivity = *activity
		mutex.Unlock()
	}()

	// Aggregate system health
	wg.Add(1)
	go func() {
		defer wg.Done()
		health, err := da.aggregateSystemHealth(ctx)
		if err != nil {
			da.logger.WithError(err).Error("Failed to aggregate system health")
			return
		}
		mutex.Lock()
		overview.SystemHealth = *health
		mutex.Unlock()
	}()

	wg.Wait()
	return overview, nil
}

// aggregateContainerStats gets real container statistics from Docker
func (da *DashboardAggregator) aggregateContainerStats(ctx context.Context) (*ContainerOverviewStats, error) {
	containers, err := da.dockerClient.ListContainers(ctx, true) // true = include all
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	stats := &ContainerOverviewStats{}
	stats.Total = len(containers)

	for _, container := range containers {
		switch container.State {
		case "running":
			stats.Running++
		case "exited":
			stats.Stopped++
		case "paused":
			stats.Paused++
		case "restarting":
			stats.Restarting++
		}

		// Check health status
		if container.Health != nil && container.Health.Status == "unhealthy" {
			stats.Unhealthy++
		}
	}

	return stats, nil
}

// aggregateResourceUsage gets real system resource usage
func (da *DashboardAggregator) aggregateResourceUsage(ctx context.Context) (*SystemResourceUsage, error) {
	usage := &SystemResourceUsage{}

	// Get CPU usage
	cpuUsage, err := da.systemMonitor.GetCPUUsage()
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get CPU usage")
	} else {
		usage.CPU = ResourceMetric{
			Used:       cpuUsage,
			Total:      100,
			Percentage: cpuUsage,
			Unit:       "percent",
		}
	}

	// Get memory usage
	memUsage, err := da.systemMonitor.GetMemoryUsage()
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get memory usage")
	} else {
		usage.Memory = ResourceMetric{
			Used:       float64(memUsage.Used),
			Total:      float64(memUsage.Total),
			Percentage: (float64(memUsage.Used) / float64(memUsage.Total)) * 100,
			Unit:       "bytes",
		}
	}

	// Get disk usage
	diskUsage, err := da.systemMonitor.GetDiskUsage()
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get disk usage")
	} else {
		usage.Disk = ResourceMetric{
			Used:       float64(diskUsage.Used),
			Total:      float64(diskUsage.Total),
			Percentage: (float64(diskUsage.Used) / float64(diskUsage.Total)) * 100,
			Unit:       "bytes",
		}
	}

	// Get network usage
	netUsage, err := da.systemMonitor.GetNetworkUsage()
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get network usage")
	} else {
		usage.Network = NetworkMetric{
			BytesIn:    netUsage.BytesIn,
			BytesOut:   netUsage.BytesOut,
			PacketsIn:  netUsage.PacketsIn,
			PacketsOut: netUsage.PacketsOut,
		}
	}

	return usage, nil
}

// aggregateSecurityStatus gets security scan results
func (da *DashboardAggregator) aggregateSecurityStatus(ctx context.Context) (*SecurityOverviewStatus, error) {
	// Query database for vulnerability scan results
	vulns, err := da.dbManager.GetSecurityVulnerabilities(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get security vulnerabilities: %w", err)
	}

	status := &SecurityOverviewStatus{}

	for _, vuln := range vulns {
		status.VulnerabilitiesFound++
		switch vuln.Severity {
		case "critical":
			status.CriticalVulns++
		case "high":
			status.HighVulns++
		case "medium":
			status.MediumVulns++
		case "low":
			status.LowVulns++
		}
	}

	// Calculate security score (0-100, higher is better)
	if status.VulnerabilitiesFound == 0 {
		status.SecurityScore = 100
	} else {
		// Weight vulnerabilities by severity
		weightedScore := float64(status.CriticalVulns)*10 +
			float64(status.HighVulns)*5 +
			float64(status.MediumVulns)*2 +
			float64(status.LowVulns)*1

		// Normalize to 0-100 scale (lower is better for weighted score)
		status.SecurityScore = math.Max(0, 100 - weightedScore)
	}

	// Get last scan time
	lastScan, err := da.dbManager.GetLastSecurityScanTime(ctx)
	if err == nil {
		status.LastScanTime = lastScan
	}

	return status, nil
}

// aggregateUpdateActivity gets update activity information
func (da *DashboardAggregator) aggregateUpdateActivity(ctx context.Context) (*UpdateOverviewActivity, error) {
	activity := &UpdateOverviewActivity{}

	// Get pending updates
	pendingCount, err := da.dbManager.GetPendingUpdatesCount(ctx)
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get pending updates count")
	} else {
		activity.PendingUpdates = pendingCount
	}

	// Get recent updates (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	recentCount, err := da.dbManager.GetRecentUpdatesCount(ctx, since)
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get recent updates count")
	} else {
		activity.RecentUpdates = recentCount
	}

	// Get failed updates (last 7 days)
	failedSince := time.Now().Add(-7 * 24 * time.Hour)
	failedCount, err := da.dbManager.GetFailedUpdatesCount(ctx, failedSince)
	if err != nil {
		da.logger.WithError(err).Warn("Failed to get failed updates count")
	} else {
		activity.FailedUpdates = failedCount
	}

	// Get last update time
	lastUpdate, err := da.dbManager.GetLastUpdateTime(ctx)
	if err == nil {
		activity.LastUpdateTime = lastUpdate
	}

	// Check if auto-update is enabled
	autoUpdate, err := da.dbManager.IsAutoUpdateEnabled(ctx)
	if err == nil {
		activity.AutoUpdateEnabled = autoUpdate
	}

	return activity, nil
}

// aggregateSystemHealth gets system health information
func (da *DashboardAggregator) aggregateSystemHealth(ctx context.Context) (*SystemHealthOverview, error) {
	health := &SystemHealthOverview{
		OverallStatus: "healthy",
		HealthChecks:  []HealthCheckStatus{},
	}

	// Check Docker daemon health
	dockerHealth := da.checkDockerHealth(ctx)
	health.HealthChecks = append(health.HealthChecks, dockerHealth)

	// Check database health
	dbHealth := da.checkDatabaseHealth(ctx)
	health.HealthChecks = append(health.HealthChecks, dbHealth)

	// Check Redis health (if configured)
	if da.redisClient != nil {
		redisHealth := da.checkRedisHealth(ctx)
		health.HealthChecks = append(health.HealthChecks, redisHealth)
	}

	// Calculate overall status
	for _, check := range health.HealthChecks {
		if check.Status == "healthy" {
			health.HealthyServices++
		} else {
			health.UnhealthyServices++
			if health.OverallStatus == "healthy" {
				health.OverallStatus = "degraded"
			}
		}
	}

	if health.UnhealthyServices > health.HealthyServices {
		health.OverallStatus = "unhealthy"
	}

	return health, nil
}

// Helper functions for health checks
func (da *DashboardAggregator) checkDockerHealth(ctx context.Context) HealthCheckStatus {
	check := HealthCheckStatus{
		Name:      "Docker Daemon",
		Timestamp: time.Now(),
	}

	if err := da.dockerClient.Ping(ctx); err != nil {
		check.Status = "unhealthy"
		check.Message = fmt.Sprintf("Docker daemon unreachable: %v", err)
	} else {
		check.Status = "healthy"
		check.Message = "Docker daemon is responsive"
	}

	return check
}

func (da *DashboardAggregator) checkDatabaseHealth(ctx context.Context) HealthCheckStatus {
	check := HealthCheckStatus{
		Name:      "Database",
		Timestamp: time.Now(),
	}

	if err := da.dbManager.Ping(ctx); err != nil {
		check.Status = "unhealthy"
		check.Message = fmt.Sprintf("Database unreachable: %v", err)
	} else {
		check.Status = "healthy"
		check.Message = "Database is responsive"
	}

	return check
}

func (da *DashboardAggregator) checkRedisHealth(ctx context.Context) HealthCheckStatus {
	check := HealthCheckStatus{
		Name:      "Redis Cache",
		Timestamp: time.Now(),
	}

	if err := da.redisClient.Ping(ctx).Err(); err != nil {
		check.Status = "unhealthy"
		check.Message = fmt.Sprintf("Redis unreachable: %v", err)
	} else {
		check.Status = "healthy"
		check.Message = "Redis cache is responsive"
	}

	return check
}

// Cache helper functions
func (da *DashboardAggregator) getCachedData(ctx context.Context, key string) (string, error) {
	if da.redisClient == nil {
		return "", fmt.Errorf("redis client not configured")
	}

	return da.redisClient.Get(ctx, key).Result()
}

func (da *DashboardAggregator) cacheData(ctx context.Context, key string, data interface{}, ttl time.Duration) {
	if da.redisClient == nil {
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		da.logger.WithError(err).Warn("Failed to marshal cache data")
		return
	}

	if err := da.redisClient.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		da.logger.WithError(err).Warn("Failed to cache data")
	}
}

// StartBackgroundRefresh starts background data refresh routine
func (da *DashboardAggregator) StartBackgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(da.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Refresh cache in background
			go func() {
				refreshCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if _, err := da.GetSystemOverview(refreshCtx); err != nil {
					da.logger.WithError(err).Error("Background refresh failed")
				}
			}()
		}
	}
}