package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthAPI provides HTTP endpoints for health checks
type HealthAPI struct {
	checker *HealthChecker
}

// NewHealthAPI creates a new health API instance
func NewHealthAPI(checker *HealthChecker) *HealthAPI {
	return &HealthAPI{
		checker: checker,
	}
}

// RegisterRoutes registers health check API routes
func (ha *HealthAPI) RegisterRoutes(router *gin.RouterGroup) {
	health := router.Group("/health")
	{
		health.GET("", ha.GetHealth)
		health.GET("/ready", ha.GetReadiness)
		health.GET("/live", ha.GetLiveness)
		health.GET("/checks", ha.GetAllChecks)
		health.GET("/checks/:name", ha.GetSingleCheck)
		health.GET("/history/:name", ha.GetCheckHistory)
		health.GET("/metrics", ha.GetHealthMetrics)
		health.POST("/checks/:name/run", ha.RunHealthCheck)
	}
}

// GetHealth returns overall health status
func (ha *HealthAPI) GetHealth(c *gin.Context) {
	aggregate := ha.checker.GetAggregateHealth()

	statusCode := http.StatusOK
	switch aggregate.Status {
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still OK, but degraded
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case HealthStatusUnknown:
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, aggregate)
}

// GetReadiness returns readiness probe status
func (ha *HealthAPI) GetReadiness(c *gin.Context) {
	aggregate := ha.checker.GetAggregateHealth()

	// Readiness check - service is ready to serve traffic
	ready := aggregate.Status == HealthStatusHealthy || aggregate.Status == HealthStatusDegraded

	status := gin.H{
		"status":    aggregate.Status,
		"ready":     ready,
		"timestamp": time.Now(),
		"checks":    len(aggregate.Checks),
	}

	// Add details about failed checks if not ready
	if !ready {
		var failedChecks []string
		for name, result := range aggregate.Checks {
			if result.Status != HealthStatusHealthy {
				failedChecks = append(failedChecks, name)
			}
		}
		status["failed_checks"] = failedChecks
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, status)
}

// GetLiveness returns liveness probe status
func (ha *HealthAPI) GetLiveness(c *gin.Context) {
	// Liveness check - basic service health
	// This should only fail if the service is completely broken
	alive := true
	timestamp := time.Now()

	// Basic liveness checks
	status := gin.H{
		"status":    "alive",
		"alive":     alive,
		"timestamp": timestamp,
		"uptime":    timestamp.Sub(timestamp.Add(-time.Hour)), // Placeholder
	}

	// Check if health checker is responsive
	aggregate := ha.checker.GetAggregateHealth()
	if aggregate.Status == HealthStatusUnknown {
		alive = false
		status["status"] = "dead"
		status["alive"] = false
		status["reason"] = "Health checker not responding"
	}

	statusCode := http.StatusOK
	if !alive {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, status)
}

// GetAllChecks returns status of all health checks
func (ha *HealthAPI) GetAllChecks(c *gin.Context) {
	aggregate := ha.checker.GetAggregateHealth()

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"timestamp": aggregate.Timestamp,
		"overall":   aggregate.Status,
		"checks":    aggregate.Checks,
		"summary": gin.H{
			"total":     len(aggregate.Checks),
			"healthy":   ha.countByStatus(aggregate.Checks, HealthStatusHealthy),
			"degraded":  ha.countByStatus(aggregate.Checks, HealthStatusDegraded),
			"unhealthy": ha.countByStatus(aggregate.Checks, HealthStatusUnhealthy),
			"unknown":   ha.countByStatus(aggregate.Checks, HealthStatusUnknown),
		},
	})
}

// GetSingleCheck returns status of a specific health check
func (ha *HealthAPI) GetSingleCheck(c *gin.Context) {
	checkName := c.Param("name")

	result, err := ha.checker.CheckHealth(checkName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"check":  result,
	})
}

// GetCheckHistory returns historical data for a health check
func (ha *HealthAPI) GetCheckHistory(c *gin.Context) {
	checkName := c.Param("name")

	history, err := ha.checker.GetHealthHistory(checkName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Limit history if requested
	limit := c.Query("limit")
	if limit != "" {
		// Parse limit and truncate history if needed
		// Implementation would go here
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"history": history,
	})
}

// GetHealthMetrics returns health check metrics
func (ha *HealthAPI) GetHealthMetrics(c *gin.Context) {
	allMetrics := ha.checker.GetAllHealthMetrics()

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"timestamp": time.Now(),
		"metrics":   allMetrics,
		"summary": gin.H{
			"total_checks": len(allMetrics),
		},
	})
}

// RunHealthCheck manually triggers a specific health check
func (ha *HealthAPI) RunHealthCheck(c *gin.Context) {
	checkName := c.Param("name")

	result, err := ha.checker.CheckHealth(checkName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"message":   "Health check executed",
		"check":     checkName,
		"result":    result,
		"timestamp": time.Now(),
	})
}

// Helper methods

// countByStatus counts health checks by status
func (ha *HealthAPI) countByStatus(checks map[string]HealthResult, status HealthStatus) int {
	count := 0
	for _, check := range checks {
		if check.Status == status {
			count++
		}
	}
	return count
}