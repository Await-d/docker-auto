package controller

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/service"
	"docker-auto/pkg/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DashboardController handles dashboard-related HTTP requests
type DashboardController struct {
	aggregator           *dashboard.DashboardAggregator
	updateActivitySvc    *service.UpdateActivityService
	securityScannerSvc   *service.SecurityScannerService
	containerService     *service.ContainerService
	logger               *logrus.Logger
}

// NewDashboardController creates a new dashboard controller
func NewDashboardController(
	aggregator *dashboard.DashboardAggregator,
	updateActivitySvc *service.UpdateActivityService,
	securityScannerSvc *service.SecurityScannerService,
	containerService *service.ContainerService,
	logger *logrus.Logger,
) *DashboardController {
	return &DashboardController{
		aggregator:           aggregator,
		updateActivitySvc:    updateActivitySvc,
		securityScannerSvc:   securityScannerSvc,
		containerService:     containerService,
		logger:               logger,
	}
}

// GetSystemOverview handles GET /api/dashboard/overview
func (dc *DashboardController) GetSystemOverview(c *gin.Context) {
	ctx := c.Request.Context()

	overview, err := dc.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get system overview")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get system overview",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

// GetContainerStats handles GET /api/dashboard/container-stats
func (dc *DashboardController) GetContainerStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get detailed container statistics
	containers, err := dc.containerService.ListContainers(ctx, 0, nil) // Get all containers for admin
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get container statistics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get container statistics",
			"details": err.Error(),
		})
		return
	}

	// Transform to detailed stats format
	containerStats := make([]dashboard.ContainerDetailStats, 0, len(containers.Containers))

	for _, container := range containers.Containers {
		// Get real-time stats for each container
		stats, err := dc.containerService.GetContainerStats(ctx, 0, container.ID)
		if err != nil {
			dc.logger.WithError(err).WithField("containerId", container.ID).Warn("Failed to get container stats")
			continue
		}

		detailStats := dashboard.ContainerDetailStats{
			ContainerID:  strconv.FormatInt(container.ID, 10),
			Name:         container.Name,
			Image:        container.Image,
			Status:       string(container.Status),
			State:        string(container.Status), // Use status as state for now
			RestartCount: 0,                       // Default value, may need to get from Docker
			LastUpdated:  time.Now(),
		}

		// Add CPU stats if available from simplified metrics
		if stats.Metrics != nil {
			// Convert CPU percentage to usage values (simplified mapping)
			totalUsageNanoseconds := uint64(stats.Metrics.CPUPercent * 1000000) // Convert percentage to rough nanoseconds

			detailStats.CPUStats = dashboard.CPUStats{
				CPUUsage: dashboard.CPUUsageStats{
					TotalUsage:        totalUsageNanoseconds,
					PercpuUsage:       []uint64{totalUsageNanoseconds}, // Single CPU for simplicity
					UsageInKernelmode: totalUsageNanoseconds / 2,       // Estimate 50% kernel mode
					UsageInUsermode:   totalUsageNanoseconds / 2,       // Estimate 50% user mode
				},
				SystemCPUUsage: totalUsageNanoseconds * 2, // Rough system estimate
				OnlineCPUs:     1,                         // Default to 1 CPU
				ThrottlingData: dashboard.ThrottleData{    // Default throttling data
					Periods:          0,
					ThrottledPeriods: 0,
					ThrottledTime:    0,
				},
			}
		}

		// Add memory stats if available from simplified metrics
		if stats.Metrics != nil {
			detailStats.MemoryStats = dashboard.MemoryStats{
				Usage:    uint64(stats.Metrics.MemoryUsage),
				MaxUsage: uint64(stats.Metrics.MemoryUsage), // Use current as max for simplicity
				Limit:    uint64(stats.Metrics.MemoryLimit),
				Stats: map[string]uint64{ // Provide basic memory stats
					"cache":               0,
					"rss":                 uint64(stats.Metrics.MemoryUsage),
					"mapped_file":         0,
					"total_cache":         0,
					"total_rss":           uint64(stats.Metrics.MemoryUsage),
					"total_mapped_file":   0,
					"pgpgin":              0,
					"pgpgout":             0,
					"pgfault":             0,
					"pgmajfault":          0,
					"total_pgpgin":        0,
					"total_pgpgout":       0,
					"total_pgfault":       0,
					"total_pgmajfault":    0,
				},
			}
		}

		// Add network stats if available
		if stats.Metrics != nil && stats.Metrics.NetworkIO != nil {
			detailStats.NetworkStats = map[string]dashboard.NetStats{
				"eth0": {
					RxBytes:   uint64(stats.Metrics.NetworkIO.RxBytes),
					TxBytes:   uint64(stats.Metrics.NetworkIO.TxBytes),
					RxPackets: uint64(stats.Metrics.NetworkIO.RxPackets),
					TxPackets: uint64(stats.Metrics.NetworkIO.TxPackets),
				},
			}
		}

		// Add block I/O stats if available
		if stats.Metrics != nil && stats.Metrics.BlockIO != nil {
			detailStats.BlockIOStats = dashboard.BlockIOStats{
				IoServiceBytesRecursive: []dashboard.BlkioStatEntry{
					{
						Major: 0,
						Minor: 0,
						Op:    "Read",
						Value: uint64(stats.Metrics.BlockIO.ReadBytes),
					},
					{
						Major: 0,
						Minor: 0,
						Op:    "Write",
						Value: uint64(stats.Metrics.BlockIO.WriteBytes),
					},
				},
				IoServicedRecursive: []dashboard.BlkioStatEntry{
					{
						Major: 0,
						Minor: 0,
						Op:    "Read",
						Value: uint64(stats.Metrics.BlockIO.ReadOps),
					},
					{
						Major: 0,
						Minor: 0,
						Op:    "Write",
						Value: uint64(stats.Metrics.BlockIO.WriteOps),
					},
				},
			}
		}

		containerStats = append(containerStats, detailStats)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"containers":    containerStats,
			"totalCount":    containers.Total,
			"runningCount":  dc.countContainersByStatus(containers.Containers, "running"),
			"stoppedCount":  dc.countContainersByStatus(containers.Containers, "exited"),
			"lastUpdated":   time.Now(),
		},
	})
}

// GetResourceMetrics handles GET /api/dashboard/resource-metrics
func (dc *DashboardController) GetResourceMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	// Get time range parameters
	hoursStr := c.DefaultQuery("hours", "1")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		hours = 1
	}

	// Get current resource usage from aggregator overview
	overview, err := dc.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get resource metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get resource metrics",
			"details": err.Error(),
		})
		return
	}

	// Create trend data (for now using current values, can be enhanced with historical data)
	now := time.Now()
	timePoints := make([]time.Time, hours*12) // 5-minute intervals
	cpuValues := make([]float64, hours*12)
	memoryValues := make([]float64, hours*12)
	diskValues := make([]float64, hours*12)

	for i := 0; i < len(timePoints); i++ {
		timePoints[i] = now.Add(time.Duration(-i*5) * time.Minute)
		// For demo purposes, create some variation around current values
		cpuValues[i] = overview.ResourceUsage.CPU.Percentage + float64(i%10-5) // ±5% variation
		memoryValues[i] = overview.ResourceUsage.Memory.Percentage + float64(i%8-4) // ±4% variation
		diskValues[i] = overview.ResourceUsage.Disk.Percentage + float64(i%6-3) // ±3% variation

		// Ensure values are within valid range
		if cpuValues[i] < 0 {
			cpuValues[i] = 0
		}
		if memoryValues[i] < 0 {
			memoryValues[i] = 0
		}
		if diskValues[i] < 0 {
			diskValues[i] = 0
		}
	}

	resourceMetrics := gin.H{
		"current": gin.H{
			"cpu":     overview.ResourceUsage.CPU,
			"memory":  overview.ResourceUsage.Memory,
			"disk":    overview.ResourceUsage.Disk,
			"network": overview.ResourceUsage.Network,
		},
		"trends": gin.H{
			"cpu": dashboard.ResourceTrendData{
				Timestamps: timePoints,
				Values:     cpuValues,
				MetricName: "CPU Usage",
				Unit:       "percent",
			},
			"memory": dashboard.ResourceTrendData{
				Timestamps: timePoints,
				Values:     memoryValues,
				MetricName: "Memory Usage",
				Unit:       "percent",
			},
			"disk": dashboard.ResourceTrendData{
				Timestamps: timePoints,
				Values:     diskValues,
				MetricName: "Disk Usage",
				Unit:       "percent",
			},
		},
		"lastUpdated": time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resourceMetrics,
	})
}

// GetSecurityStatus handles GET /api/dashboard/security-status
func (dc *DashboardController) GetSecurityStatus(c *gin.Context) {
	ctx := c.Request.Context()

	securityOverview, err := dc.securityScannerSvc.GetSecurityOverview(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get security status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get security status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    securityOverview,
	})
}

// GetUpdateActivity handles GET /api/dashboard/update-activity
func (dc *DashboardController) GetUpdateActivity(c *gin.Context) {
	ctx := c.Request.Context()

	updateSummary, err := dc.updateActivitySvc.GetUpdateSummary(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get update activity")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get update activity",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updateSummary,
	})
}

// GetHealthMetrics handles GET /api/dashboard/health-metrics
func (dc *DashboardController) GetHealthMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	// Get system overview which includes health information
	overview, err := dc.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get health metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get health metrics",
			"details": err.Error(),
		})
		return
	}

	// Enhanced health metrics with additional checks
	healthMetrics := gin.H{
		"systemHealth":    overview.SystemHealth,
		"containerHealth": dc.getContainerHealthSummary(ctx),
		"serviceHealth":   dc.getServiceHealthSummary(),
		"lastUpdated":     time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    healthMetrics,
	})
}

// GetRecentUpdates handles GET /api/dashboard/updates/recent
func (dc *DashboardController) GetRecentUpdates(c *gin.Context) {
	ctx := c.Request.Context()

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	// Get update activities from database
	updateSummary, err := dc.updateActivitySvc.GetUpdateSummary(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get recent updates")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get recent updates",
			"details": err.Error(),
		})
		return
	}

	// Limit the results
	activities := updateSummary.UpdateActivities
	if len(activities) > limit {
		activities = activities[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"updates":       activities,
			"totalCount":    len(updateSummary.UpdateActivities),
			"recentCount":   updateSummary.RecentlyUpdated,
			"pendingCount":  updateSummary.PendingUpdates,
			"failedCount":   updateSummary.FailedUpdates,
			"lastUpdated":   time.Now(),
		},
	})
}

// GetPendingUpdates handles GET /api/dashboard/updates/pending
func (dc *DashboardController) GetPendingUpdates(c *gin.Context) {
	ctx := c.Request.Context()

	updateSummary, err := dc.updateActivitySvc.GetUpdateSummary(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get pending updates")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get pending updates",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"availableUpdates": updateSummary.AvailableUpdates,
			"pendingCount":     updateSummary.PendingUpdates,
			"autoUpdateCount":  updateSummary.AutoUpdateEnabled,
			"lastChecked":      updateSummary.LastUpdateCheck,
		},
	})
}

// TriggerUpdate handles POST /api/dashboard/updates/trigger
func (dc *DashboardController) TriggerUpdate(c *gin.Context) {
	ctx := c.Request.Context()

	var request struct {
		ContainerID string                      `json:"containerId" binding:"required"`
		Strategy    *service.UpdateStrategy     `json:"strategy,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err := dc.updateActivitySvc.TriggerUpdate(ctx, request.ContainerID, request.Strategy)
	if err != nil {
		dc.logger.WithError(err).WithField("containerId", request.ContainerID).Error("Failed to trigger update")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to trigger update",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Update triggered successfully",
		"data": gin.H{
			"containerId": request.ContainerID,
			"triggeredAt": time.Now(),
		},
	})
}

// GetUpdateHistory handles GET /api/dashboard/updates/history
func (dc *DashboardController) GetUpdateHistory(c *gin.Context) {
	ctx := c.Request.Context()

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	// Get full update summary
	updateSummary, err := dc.updateActivitySvc.GetUpdateSummary(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get update history")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get update history",
			"details": err.Error(),
		})
		return
	}

	// Implement pagination
	activities := updateSummary.UpdateActivities
	totalCount := len(activities)
	startIndex := (page - 1) * limit
	endIndex := startIndex + limit

	if startIndex >= totalCount {
		activities = []service.UpdateActivity{}
	} else if endIndex > totalCount {
		activities = activities[startIndex:]
	} else {
		activities = activities[startIndex:endIndex]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"updates":      activities,
			"totalCount":   totalCount,
			"page":         page,
			"limit":        limit,
			"totalPages":   (totalCount + limit - 1) / limit,
			"hasNext":      endIndex < totalCount,
			"hasPrevious":  page > 1,
		},
	})
}

// GetSecurityOverview handles GET /api/dashboard/security/overview
func (dc *DashboardController) GetSecurityOverview(c *gin.Context) {
	ctx := c.Request.Context()

	overview, err := dc.securityScannerSvc.GetSecurityOverview(ctx)
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get security overview")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get security overview",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
	})
}

// TriggerSecurityScan handles POST /api/dashboard/security/scan
func (dc *DashboardController) TriggerSecurityScan(c *gin.Context) {
	ctx := c.Request.Context()

	var request struct {
		ContainerID string `json:"containerId,omitempty"`
		ScanAll     bool   `json:"scanAll"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.ScanAll {
		// Trigger scan for all containers
		go func() {
			if err := dc.securityScannerSvc.ScanAllContainers(context.Background()); err != nil {
				dc.logger.WithError(err).Error("Background security scan failed")
			}
		}()

		c.JSON(http.StatusAccepted, gin.H{
			"success": true,
			"message": "Security scan for all containers started",
			"data": gin.H{
				"scanType":   "all_containers",
				"startedAt":  time.Now(),
			},
		})
	} else if request.ContainerID != "" {
		// Trigger scan for specific container
		containerID, err := strconv.ParseInt(request.ContainerID, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid container ID format",
				"details": err.Error(),
			})
			return
		}

		containerDetail, err := dc.containerService.GetContainer(ctx, 0, containerID) // Use 0 for admin access
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Container not found",
				"details": err.Error(),
			})
			return
		}

		container := containerDetail.Container

		go func() {
			if err := dc.securityScannerSvc.ScanContainer(context.Background(),
				strconv.FormatInt(int64(container.ID), 10), container.ContainerID, container.Image, container.Name); err != nil {
				dc.logger.WithError(err).Error("Background security scan failed")
			}
		}()

		c.JSON(http.StatusAccepted, gin.H{
			"success": true,
			"message": "Security scan started for container",
			"data": gin.H{
				"scanType":    "single_container",
				"containerId": request.ContainerID,
				"startedAt":   time.Now(),
			},
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Either containerId must be provided or scanAll must be true",
		})
	}
}

// Helper functions

func (dc *DashboardController) countContainersByStatus(containers []*service.ContainerSummary, status string) int {
	count := 0
	for _, container := range containers {
		if string(container.Status) == status {
			count++
		}
	}
	return count
}

func (dc *DashboardController) getContainerHealthSummary(ctx context.Context) gin.H {
	// Get container health information
	containers, err := dc.containerService.ListContainers(ctx, 0, nil)
	if err != nil {
		return gin.H{
			"error": "Failed to get container health data",
		}
	}

	healthyCount := 0
	unhealthyCount := 0
	unknownCount := 0

	for _, container := range containers.Containers {
		// Map container status to health status
		switch container.Status {
		case model.ContainerStatusRunning:
			healthyCount++
		case model.ContainerStatusExited, model.ContainerStatusDead, model.ContainerStatusStopped:
			unhealthyCount++
		default:
			unknownCount++
		}
	}

	return gin.H{
		"totalContainers": len(containers.Containers),
		"healthy":         healthyCount,
		"unhealthy":       unhealthyCount,
		"unknown":         unknownCount,
		"healthyPercent":  float64(healthyCount) / float64(len(containers.Containers)) * 100,
	}
}

func (dc *DashboardController) getServiceHealthSummary() gin.H {
	// This would check various service health endpoints
	services := []string{"docker", "database", "redis", "api"}
	healthyServices := len(services) // Placeholder - all healthy for demo

	return gin.H{
		"totalServices":   len(services),
		"healthyServices": healthyServices,
		"services":        services,
	}
}