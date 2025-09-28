package controller

import (
	"net/http"
	"strconv"
	"time"

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
	containers, err := dc.containerService.ListContainers(ctx, 0) // Get all containers
	if err != nil {
		dc.logger.WithError(err).Error("Failed to get container statistics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get container statistics",
			"details": err.Error(),
		})
		return
	}

	// Transform to detailed stats format
	containerStats := make([]dashboard.ContainerDetailStats, 0, len(containers))

	for _, container := range containers {
		// Get real-time stats for each container
		stats, err := dc.containerService.GetContainerStats(ctx, container.ID)
		if err != nil {
			dc.logger.WithError(err).WithField("containerId", container.ID).Warn("Failed to get container stats")
			continue
		}

		detailStats := dashboard.ContainerDetailStats{
			ContainerID:  container.ID,
			Name:         container.Name,
			Image:        container.Image,
			Status:       container.Status,
			State:        container.State,
			RestartCount: container.RestartCount,
			LastUpdated:  time.Now(),
		}

		// Add CPU stats if available
		if stats.CPUStats != nil {
			detailStats.CPUStats = dashboard.CPUStats{
				CPUUsage: dashboard.CPUUsageStats{
					TotalUsage:        stats.CPUStats.TotalUsage,
					PercpuUsage:       stats.CPUStats.PercpuUsage,
					UsageInKernelmode: stats.CPUStats.UsageInKernelmode,
					UsageInUsermode:   stats.CPUStats.UsageInUsermode,
				},
				SystemCPUUsage: stats.CPUStats.SystemCPUUsage,
				OnlineCPUs:     stats.CPUStats.OnlineCPUs,
			}
		}

		// Add memory stats if available
		if stats.MemoryStats != nil {
			detailStats.MemoryStats = dashboard.MemoryStats{
				Usage:    stats.MemoryStats.Usage,
				MaxUsage: stats.MemoryStats.MaxUsage,
				Limit:    stats.MemoryStats.Limit,
				Stats:    stats.MemoryStats.Stats,
			}
		}

		containerStats = append(containerStats, detailStats)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"containers":    containerStats,
			"totalCount":    len(containers),
			"runningCount":  dc.countContainersByStatus(containers, "running"),
			"stoppedCount":  dc.countContainersByStatus(containers, "exited"),
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
		container, err := dc.containerService.GetContainerByID(ctx, request.ContainerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Container not found",
				"details": err.Error(),
			})
			return
		}

		go func() {
			if err := dc.securityScannerSvc.ScanContainer(context.Background(),
				container.ID, container.ImageID, container.Image, container.Name); err != nil {
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

func (dc *DashboardController) countContainersByStatus(containers []interface{}, status string) int {
	count := 0
	for _, container := range containers {
		// This would need to be adapted based on the actual container structure
		// For now, this is a placeholder
		count++
	}
	return count
}

func (dc *DashboardController) getContainerHealthSummary(ctx context.Context) gin.H {
	// Get container health information
	containers, err := dc.containerService.ListContainers(ctx, 0)
	if err != nil {
		return gin.H{
			"error": "Failed to get container health data",
		}
	}

	healthyCount := 0
	unhealthyCount := 0
	unknownCount := 0

	for _, container := range containers {
		switch container.Health {
		case "healthy":
			healthyCount++
		case "unhealthy":
			unhealthyCount++
		default:
			unknownCount++
		}
	}

	return gin.H{
		"totalContainers": len(containers),
		"healthy":         healthyCount,
		"unhealthy":       unhealthyCount,
		"unknown":         unknownCount,
		"healthyPercent":  float64(healthyCount) / float64(len(containers)) * 100,
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