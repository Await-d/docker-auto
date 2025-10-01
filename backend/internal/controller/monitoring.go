package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"docker-auto/internal/middleware"
	"docker-auto/internal/model"
	"docker-auto/internal/service"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocket upgrader for monitoring connections
var monitoringUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, you should check the origin properly
		return true
	},
}

// MonitoringController handles container monitoring and metrics endpoints
type MonitoringController struct {
	monitoringService *service.ContainerMonitoringService
	containerService  *service.ContainerService
	logger            *logrus.Logger
}

// NewMonitoringController creates a new monitoring controller
func NewMonitoringController(
	monitoringService *service.ContainerMonitoringService,
	containerService *service.ContainerService,
	logger *logrus.Logger,
) *MonitoringController {
	return &MonitoringController{
		monitoringService: monitoringService,
		containerService:  containerService,
		logger:            logger,
	}
}

// GetContainerMetrics godoc
// @Summary Get real-time container metrics
// @Description Get current resource usage metrics for a specific container
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse{data=service.ContainerMetrics} "Container metrics"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/containers/{id}/metrics [get]
func (mc *MonitoringController) GetContainerMetrics(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	containerIDStr := c.Param("id")
	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Verify user has access to container
	container, err := mc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container for metrics")
		rb.NotFound("Container not found")
		return
	}

	if container.Container.ContainerID == "" {
		rb.BadRequest("Container has no Docker instance")
		return
	}

	// Get current metrics
	metrics, err := mc.monitoringService.GetContainerMetrics(c.Request.Context(), container.Container.ContainerID)
	if err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
			"docker_id":    container.Container.ContainerID,
		}).Error("Failed to get container metrics")
		rb.InternalServerError("Failed to retrieve container metrics")
		return
	}

	rb.Success(metrics)
}

// GetAllContainerMetrics godoc
// @Summary Get metrics for all monitored containers
// @Description Get current resource usage metrics for all containers being monitored by the user
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=map[string]service.ContainerMetrics} "All container metrics"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/containers/metrics [get]
func (mc *MonitoringController) GetAllContainerMetrics(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Get all containers for user
	containers, err := mc.containerService.ListContainers(c.Request.Context(), userID, &service.ContainerFilter{
		ContainerFilter: &model.ContainerFilter{
			Limit: 1000, // Get all containers
		},
	})
	if err != nil {
		mc.logger.WithError(err).WithField("user_id", userID).Error("Failed to list containers for metrics")
		rb.InternalServerError("Failed to retrieve containers")
		return
	}

	// Get metrics for each container that has monitoring enabled
	result := make(map[string]*service.ContainerMetrics)
	for _, containerSummary := range containers.Containers {
		// Get full container details to access ContainerID
		containerDetail, err := mc.containerService.GetContainer(c.Request.Context(), userID, containerSummary.ID)
		if err != nil {
			mc.logger.WithError(err).WithField("container_id", containerSummary.ID).Debug("Failed to get container details for monitoring")
			continue
		}

		// Check if container has Docker ID and get metrics
		if containerDetail.Container.ContainerID != "" {
			if metrics, metricsErr := mc.monitoringService.GetContainerMetrics(c.Request.Context(), containerDetail.Container.ContainerID); metricsErr == nil {
				result[containerDetail.Container.ContainerID] = metrics
			}
		}
	}

	mc.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"containers":     len(containers.Containers),
		"with_metrics":   len(result),
	}).Debug("Retrieved metrics for all containers")

	rb.Success(result)
}

// StreamContainerMetrics godoc
// @Summary Stream real-time container metrics via WebSocket
// @Description Establish WebSocket connection for real-time container metrics streaming
// @Tags Monitoring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param interval query int false "Update interval in seconds" default(2) minimum(1) maximum(60)
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/containers/{id}/metrics/stream [get]
func (mc *MonitoringController) StreamContainerMetrics(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	containerIDStr := c.Param("id")
	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	// Parse update interval
	intervalStr := c.DefaultQuery("interval", "2")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 || interval > 60 {
		utils.BadRequestJSON(c, "Invalid interval, must be between 1 and 60 seconds")
		return
	}

	// Verify user has access to container
	container, err := mc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container for metrics streaming")
		utils.NotFoundJSON(c, "Container not found")
		return
	}

	if container.Container.ContainerID == "" {
		utils.BadRequestJSON(c, "Container has no Docker instance")
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := monitoringUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		mc.logger.WithError(err).Error("Failed to upgrade WebSocket connection for metrics streaming")
		utils.InternalServerErrorJSON(c, "Failed to establish WebSocket connection")
		return
	}
	defer conn.Close()

	// Start metrics streaming session
	mc.streamMetricsSession(conn, container.Container.ContainerID, time.Duration(interval)*time.Second)
}

// GetMonitoringStatus godoc
// @Summary Get monitoring system status
// @Description Get status of all active monitoring sessions and system metrics
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "Monitoring system status"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/status [get]
func (mc *MonitoringController) GetMonitoringStatus(c *gin.Context) {
	rb := utils.NewResponseBuilder(c)

	// Get monitoring sessions status
	sessions := mc.monitoringService.GetMonitoringStatus()

	// Get system monitoring metrics
	systemMetrics := mc.monitoringService.GetSystemMetrics()

	status := map[string]interface{}{
		"active_sessions":    len(sessions),
		"monitoring_sessions": sessions,
		"system_metrics":     systemMetrics,
		"timestamp":          time.Now().Unix(),
	}

	mc.logger.WithField("active_sessions", len(sessions)).Debug("Retrieved monitoring status")

	rb.Success(status)
}

// StartContainerMonitoring godoc
// @Summary Start monitoring for a container
// @Description Start real-time monitoring for a specific container with configurable options
// @Tags Monitoring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param config body service.MonitoringSessionConfig false "Monitoring configuration"
// @Success 200 {object} utils.APIResponse "Monitoring started successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 409 {object} utils.APIResponse "Container already being monitored"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/containers/{id}/start [post]
func (mc *MonitoringController) StartContainerMonitoring(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	containerIDStr := c.Param("id")
	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	// Parse optional monitoring configuration
	var config *service.MonitoringSessionConfig
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&config); err != nil {
			mc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid monitoring config")
			utils.BadRequestJSON(c, "Invalid configuration format: "+err.Error())
			return
		}
	}

	rb := utils.NewResponseBuilder(c)

	// Start monitoring
	if err := mc.monitoringService.StartMonitoring(c.Request.Context(), userID, containerID, config); err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to start container monitoring")

		if err.Error() == "container is already being monitored" {
			rb.Conflict("Container is already being monitored")
			return
		}
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to start monitoring")
		return
	}

	mc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container monitoring started")

	rb.SuccessWithMessage(nil, "Monitoring started successfully")
}

// StopContainerMonitoring godoc
// @Summary Stop monitoring for a container
// @Description Stop real-time monitoring for a specific container
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse "Monitoring stopped successfully"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found or not being monitored"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/containers/{id}/stop [post]
func (mc *MonitoringController) StopContainerMonitoring(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	containerIDStr := c.Param("id")
	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Get container to find Docker ID
	container, err := mc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container for stopping monitoring")
		rb.NotFound("Container not found")
		return
	}

	if container.Container.ContainerID == "" {
		rb.BadRequest("Container has no Docker instance")
		return
	}

	// Stop monitoring
	if err := mc.monitoringService.StopMonitoring(container.Container.ContainerID); err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
			"docker_id":    container.Container.ContainerID,
		}).Error("Failed to stop container monitoring")

		if err.Error() == "container is not being monitored" {
			rb.NotFound("Container is not being monitored")
			return
		}
		rb.InternalServerError("Failed to stop monitoring")
		return
	}

	mc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container monitoring stopped")

	rb.SuccessWithMessage(nil, "Monitoring stopped successfully")
}

// GetHistoricalMetrics godoc
// @Summary Get historical container metrics
// @Description Retrieve historical metrics data for a container within a time range
// @Tags Monitoring
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param since query string false "Start time (RFC3339)" default("1h ago")
// @Param until query string false "End time (RFC3339)" default("now")
// @Param resolution query string false "Data resolution (1m, 5m, 15m, 1h)" default("5m")
// @Param limit query int false "Maximum number of data points" default(100)
// @Success 200 {object} utils.APIResponse{data=[]service.HistoricalMetric} "Historical metrics"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/monitoring/containers/{id}/metrics/historical [get]
func (mc *MonitoringController) GetHistoricalMetrics(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	containerIDStr := c.Param("id")
	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	// Parse time parameters
	var since, until time.Time
	sinceStr := c.DefaultQuery("since", "")
	if sinceStr != "" {
		if parsed, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = parsed
		} else {
			utils.BadRequestJSON(c, "Invalid since timestamp format, use RFC3339")
			return
		}
	} else {
		since = time.Now().Add(-time.Hour) // Default to 1 hour ago
	}

	untilStr := c.DefaultQuery("until", "")
	if untilStr != "" {
		if parsed, err := time.Parse(time.RFC3339, untilStr); err == nil {
			until = parsed
		} else {
			utils.BadRequestJSON(c, "Invalid until timestamp format, use RFC3339")
			return
		}
	} else {
		until = time.Now()
	}

	// Parse other parameters
	resolution := c.DefaultQuery("resolution", "5m")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 10000 {
		limit = 100
	}

	rb := utils.NewResponseBuilder(c)

	// Verify user has access to container
	container, err := mc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		mc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container for historical metrics")
		rb.NotFound("Container not found")
		return
	}

	// Get historical metrics (placeholder implementation)
	// In a real implementation, this would query the metrics repository
	historicalMetrics := mc.generateHistoricalMetrics(container.Container.ContainerID, since, until, resolution, limit)

	mc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
		"since":        since,
		"until":        until,
		"resolution":   resolution,
		"data_points":  len(historicalMetrics),
	}).Debug("Retrieved historical metrics")

	rb.Success(historicalMetrics)
}

// streamMetricsSession handles WebSocket session for streaming container metrics
func (mc *MonitoringController) streamMetricsSession(conn *websocket.Conn, dockerID string, interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to metrics updates
	updatesChan := mc.monitoringService.SubscribeToMetrics(dockerID)
	defer mc.monitoringService.UnsubscribeFromMetrics(dockerID, updatesChan)

	// Send initial success message
	conn.WriteMessage(websocket.TextMessage, []byte(`{"type": "stream_started", "message": "Metrics stream initialized"}`))

	errChan := make(chan error, 2)

	// Handle WebSocket messages (ping/pong and control)
	go func() {
		defer cancel()
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						mc.logger.WithError(err).Error("WebSocket connection error during metrics streaming")
						errChan <- err
					}
					return
				}

				// Handle control messages
				var controlMsg map[string]interface{}
				if json.Unmarshal(message, &controlMsg) == nil {
					if msgType, ok := controlMsg["type"].(string); ok && msgType == "ping" {
						conn.WriteMessage(websocket.TextMessage, []byte(`{"type": "pong"}`))
					}
				}
			}
		}
	}()

	// Stream metrics updates
	go func() {
		defer cancel()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-updatesChan:
				if !ok {
					errChan <- fmt.Errorf("metrics stream closed")
					return
				}

				// Send metrics update to WebSocket client
				updateData, _ := json.Marshal(map[string]interface{}{
					"type": "metrics_update",
					"data": update,
				})

				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if writeErr := conn.WriteMessage(websocket.TextMessage, updateData); writeErr != nil {
					mc.logger.WithError(writeErr).Error("Failed to write metrics to WebSocket")
					errChan <- writeErr
					return
				}

			case <-ticker.C:
				// Send periodic ping
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if pingErr := conn.WriteMessage(websocket.PingMessage, nil); pingErr != nil {
					errChan <- pingErr
					return
				}
			}
		}
	}()

	// Wait for error or context cancellation
	select {
	case err := <-errChan:
		if err != context.Canceled {
			mc.logger.WithError(err).WithField("docker_id", dockerID).Error("Metrics streaming error")
		}
	case <-ctx.Done():
		mc.logger.WithField("docker_id", dockerID).Debug("Metrics streaming session ended")
	}
}

// generateHistoricalMetrics generates sample historical metrics data
// In a real implementation, this would query the metrics repository
func (mc *MonitoringController) generateHistoricalMetrics(dockerID string, since, until time.Time, resolution string, limit int) []map[string]interface{} {
	// Parse resolution
	var step time.Duration
	switch resolution {
	case "1m":
		step = time.Minute
	case "5m":
		step = 5 * time.Minute
	case "15m":
		step = 15 * time.Minute
	case "1h":
		step = time.Hour
	default:
		step = 5 * time.Minute
	}

	var metrics []map[string]interface{}
	current := since
	count := 0

	for current.Before(until) && count < limit {
		// Generate sample data points
		metrics = append(metrics, map[string]interface{}{
			"timestamp":      current.Unix(),
			"cpu_percent":    50.0 + (float64(count%20)-10)*2.5, // Simulated CPU usage
			"memory_percent": 60.0 + (float64(count%15)-7)*3.0,  // Simulated memory usage
			"memory_usage":   2147483648 + count*10485760,        // ~2GB + growth
			"memory_limit":   4294967296,                         // 4GB limit
			"network_io": map[string]interface{}{
				"rx_bytes": count * 1024 * 1024, // Simulated network I/O
				"tx_bytes": count * 512 * 1024,
			},
			"block_io": map[string]interface{}{
				"read_bytes":  count * 2048 * 1024,
				"write_bytes": count * 1024 * 1024,
			},
		})

		current = current.Add(step)
		count++
	}

	return metrics
}