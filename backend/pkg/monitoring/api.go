package monitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MonitoringAPI provides HTTP endpoints for monitoring and observability
type MonitoringAPI struct {
	collector    *MetricsCollector
	systemCollector *SystemMetricsCollector
}

// NewMonitoringAPI creates a new monitoring API instance
func NewMonitoringAPI(collector *MetricsCollector) *MonitoringAPI {
	return &MonitoringAPI{
		collector:       collector,
		systemCollector: NewSystemMetricsCollector(),
	}
}

// RegisterRoutes registers monitoring API routes
func (ma *MonitoringAPI) RegisterRoutes(router *gin.RouterGroup) {
	monitoring := router.Group("/monitoring")
	{
		monitoring.GET("/metrics", ma.GetMetrics)
		monitoring.GET("/metrics/prometheus", ma.GetPrometheusMetrics)
		monitoring.GET("/system", ma.GetSystemMetrics)
		monitoring.GET("/components", ma.GetComponentMetrics)
		monitoring.GET("/health", ma.GetHealthStatus)
		monitoring.GET("/status", ma.GetOverallStatus)
	}
}

// GetMetrics returns all collected metrics
func (ma *MonitoringAPI) GetMetrics(c *gin.Context) {
	metrics := ma.collector.GetAllMetrics()

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"timestamp": time.Now(),
		"metrics":   metrics,
		"count":     len(metrics),
	})
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (ma *MonitoringAPI) GetPrometheusMetrics(c *gin.Context) {
	metrics := ma.collector.GetAllMetrics()
	prometheusFormat := ma.convertToPrometheusFormat(metrics)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, prometheusFormat)
}

// GetSystemMetrics returns system-level metrics
func (ma *MonitoringAPI) GetSystemMetrics(c *gin.Context) {
	systemMetrics, err := ma.systemCollector.Collect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"timestamp":      time.Now(),
		"system_metrics": systemMetrics,
	})
}

// GetComponentMetrics returns metrics for all registered components
func (ma *MonitoringAPI) GetComponentMetrics(c *gin.Context) {
	componentMetrics := ma.collector.GetComponentMetrics()

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"timestamp":        time.Now(),
		"component_metrics": componentMetrics,
		"count":            len(componentMetrics),
	})
}

// GetHealthStatus returns basic health status
func (ma *MonitoringAPI) GetHealthStatus(c *gin.Context) {
	// Basic health check
	healthStatus := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now().Add(-time.Hour)), // Placeholder
		"version":   "1.0.0",
	}

	// Check if metrics collector is working
	metrics := ma.collector.GetAllMetrics()
	if len(metrics) == 0 {
		healthStatus["status"] = "degraded"
		healthStatus["issues"] = []string{"No metrics collected"}
	}

	c.JSON(http.StatusOK, healthStatus)
}

// GetOverallStatus returns comprehensive system status
func (ma *MonitoringAPI) GetOverallStatus(c *gin.Context) {
	// Get system metrics
	systemMetrics, systemErr := ma.systemCollector.Collect()

	// Get component metrics
	componentMetrics := ma.collector.GetComponentMetrics()

	// Get all metrics count
	allMetrics := ma.collector.GetAllMetrics()

	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"summary": gin.H{
			"total_metrics":     len(allMetrics),
			"components_count":  len(componentMetrics),
			"system_healthy":    systemErr == nil,
		},
	}

	// Add system metrics if available
	if systemErr == nil {
		status["system"] = systemMetrics
	} else {
		status["system_error"] = systemErr.Error()
		status["status"] = "degraded"
	}

	// Add component status
	if len(componentMetrics) > 0 {
		status["components"] = componentMetrics
	}

	c.JSON(http.StatusOK, status)
}

// convertToPrometheusFormat converts metrics to Prometheus text format
func (ma *MonitoringAPI) convertToPrometheusFormat(metrics []Metric) string {
	var output string

	for _, metric := range metrics {
		// Add HELP line
		output += "# HELP " + metric.Name + " " + metric.Description + "\n"

		// Add TYPE line
		metricType := "gauge"
		switch metric.Type {
		case MetricTypeCounter:
			metricType = "counter"
		case MetricTypeHistogram:
			metricType = "histogram"
		case MetricTypeSummary:
			metricType = "summary"
		}
		output += "# TYPE " + metric.Name + " " + metricType + "\n"

		// Add metric line
		metricLine := metric.Name
		if len(metric.Labels) > 0 {
			metricLine += "{"
			first := true
			for k, v := range metric.Labels {
				if !first {
					metricLine += ","
				}
				metricLine += k + "=\"" + v + "\""
				first = false
			}
			metricLine += "}"
		}
		metricLine += " " + strconv.FormatFloat(metric.Value, 'f', -1, 64)
		metricLine += " " + strconv.FormatInt(metric.Timestamp.UnixMilli(), 10)
		output += metricLine + "\n\n"
	}

	return output
}