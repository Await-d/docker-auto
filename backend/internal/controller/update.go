package controller

import (
	"net/http"
	"strconv"
	"time"

	"docker-auto/internal/middleware"
	"docker-auto/internal/model"
	"docker-auto/internal/service"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// UpdateController handles update-related HTTP requests
type UpdateController struct {
	containerService *service.ContainerService
	imageService     *service.ImageService
	logger           *logrus.Logger
}

// NewUpdateController creates a new update controller
func NewUpdateController(containerService *service.ContainerService, imageService *service.ImageService, logger *logrus.Logger) *UpdateController {
	return &UpdateController{
		containerService: containerService,
		imageService:     imageService,
		logger:           logger,
	}
}

// GetUpdateHistory godoc
// @Summary Get update history
// @Description Get paginated list of update history records
// @Tags Updates
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param container_id query int false "Filter by container ID"
// @Param status query string false "Filter by status"
// @Param start_date query string false "Filter by start date (RFC3339)"
// @Param end_date query string false "Filter by end date (RFC3339)"
// @Param sort_by query string false "Sort field" default(started_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} utils.APIResponse{data=[]model.UpdateHistory} "Update history"
// @Failure 400 {object} utils.APIResponse "Invalid request parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/history [get]
func (uc *UpdateController) GetUpdateHistory(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	containerIDStr := c.Query("container_id")
	status := c.Query("status")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	sortBy := c.DefaultQuery("sort_by", "started_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Parse container ID if provided
	var containerID *int64
	if containerIDStr != "" {
		if id, err := strconv.ParseInt(containerIDStr, 10, 64); err == nil {
			containerID = &id
		} else {
			utils.BadRequestJSON(c, "Invalid container ID")
			return
		}
	}

	// Parse dates if provided
	var startDate, endDate *time.Time
	if startDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = &parsed
		} else {
			utils.BadRequestJSON(c, "Invalid start date format (use RFC3339)")
			return
		}
	}
	if endDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = &parsed
		} else {
			utils.BadRequestJSON(c, "Invalid end date format (use RFC3339)")
			return
		}
	}

	// Build filter
	filter := &model.UpdateHistoryFilter{
		ContainerID: containerID,
		Status:      status,
		StartDate:   startDate,
		EndDate:     endDate,
		Limit:       limit,
		Offset:      (page - 1) * limit,
		OrderBy:     sortBy,
	}

	if sortOrder == "desc" {
		filter.OrderBy += " DESC"
	}

	rb := utils.NewResponseBuilder(c)

	// For now, return empty list as we need to implement the update history repository methods
	// In a real implementation, you'd call something like:
	// history, total, err := uc.updateHistoryRepo.List(c.Request.Context(), filter)
	uc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
		"status":       status,
		"page":         page,
		"limit":        limit,
	}).Info("Update history requested (placeholder implementation)")

	// Placeholder response
	emptyHistory := []interface{}{}
	pagination := utils.CreatePagination(page, limit, 0)

	rb.SuccessWithPagination(emptyHistory, pagination)
}

// TriggerBatchUpdate godoc
// @Summary Trigger batch updates
// @Description Trigger updates for multiple containers
// @Tags Updates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.BulkUpdateRequest true "Batch update request"
// @Success 200 {object} utils.APIResponse{data=[]service.OperationResult} "Update results"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/batch [post]
func (uc *UpdateController) TriggerBatchUpdate(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req service.BulkUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid batch update request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	// Validate request
	if len(req.ContainerIDs) == 0 {
		utils.BadRequestJSON(c, "At least one container ID is required")
		return
	}

	// For batch updates, we only support the update action
	if req.Action != "update" {
		utils.BadRequestJSON(c, "Only 'update' action is supported for batch updates")
		return
	}

	if req.UpdateImage == nil {
		utils.BadRequestJSON(c, "Update image configuration is required for batch updates")
		return
	}

	rb := utils.NewResponseBuilder(c)

	results := make([]service.OperationResult, 0, len(req.ContainerIDs))

	// Process each container
	for _, containerID := range req.ContainerIDs {
		result := service.OperationResult{
			ContainerID: containerID,
			Success:     false,
		}

		// Get container name for result
		if container, err := uc.containerService.GetContainer(c.Request.Context(), userID, containerID); err == nil {
			result.Name = container.Container.Name
		}

		// Trigger update
		updateHistory, err := uc.containerService.UpdateContainerImage(c.Request.Context(), userID, containerID, req.UpdateImage)
		if err != nil {
			result.Error = err.Error()
			uc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":      userID,
				"container_id": containerID,
			}).Warn("Batch update failed for container")
		} else {
			result.Success = true
			result.Message = "Update initiated successfully"

			// Add update ID to result metadata if available
			if updateHistory != nil {
				result.Message += " (Update ID: " + strconv.FormatUint(uint64(updateHistory.ID), 10) + ")"
			}
		}

		results = append(results, result)
	}

	// Count successes and failures
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	uc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"total":        len(req.ContainerIDs),
		"success":      successCount,
		"failed":       len(req.ContainerIDs) - successCount,
	}).Info("Batch update completed")

	rb.Success(results)
}

// GetUpdateStatus godoc
// @Summary Get current update status
// @Description Get status of ongoing and recent updates
// @Tags Updates
// @Produce json
// @Security BearerAuth
// @Param active_only query boolean false "Show only active updates" default(false)
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "Update status"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/status [get]
func (uc *UpdateController) GetUpdateStatus(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	activeOnly, _ := strconv.ParseBool(c.DefaultQuery("active_only", "false"))

	rb := utils.NewResponseBuilder(c)

	// Build status response
	// In a real implementation, you'd query the update history repository
	status := map[string]interface{}{
		"active_updates":    0,
		"pending_updates":   0,
		"completed_updates": 0,
		"failed_updates":    0,
		"last_check":        time.Now(),
		"updates":          []interface{}{},
	}

	// Add summary statistics
	if !activeOnly {
		// Get recent update statistics
		// This would involve querying the update history repository
		status["recent_stats"] = map[string]interface{}{
			"last_24h": map[string]int{
				"completed": 0,
				"failed":    0,
				"total":     0,
			},
			"last_7d": map[string]int{
				"completed": 0,
				"failed":    0,
				"total":     0,
			},
		}
	}

	uc.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"active_only": activeOnly,
	}).Info("Update status requested (placeholder implementation)")

	rb.Success(status)
}

// RollbackUpdate godoc
// @Summary Rollback update
// @Description Rollback a container to its previous version
// @Tags Updates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Update History ID"
// @Param request body map[string]interface{} false "Rollback options"
// @Success 200 {object} utils.APIResponse{data=model.UpdateHistory} "Rollback initiated"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Update record not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/rollback/{id} [post]
func (uc *UpdateController) RollbackUpdate(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	updateIDStr := c.Param("id")
	updateID, err := strconv.ParseInt(updateIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid update ID")
		return
	}

	var req struct {
		Force  bool   `json:"force,omitempty"`
		Reason string `json:"reason,omitempty"`
	}

	// Optional request body
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			uc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":   userID,
				"update_id": updateID,
			}).Warn("Invalid rollback request")
			utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
			return
		}
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Get the update history record
	// 2. Validate that rollback is possible
	// 3. Create a new update history record for the rollback
	// 4. Perform the actual rollback operation
	// 5. Update the status accordingly

	uc.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"update_id": updateID,
		"force":     req.Force,
		"reason":    req.Reason,
	}).Info("Rollback requested (placeholder implementation)")

	// Placeholder response
	rb.Error(http.StatusNotImplemented, "Rollback functionality not yet implemented")
}

// GetUpdateDetails godoc
// @Summary Get update details
// @Description Get detailed information about a specific update
// @Tags Updates
// @Produce json
// @Security BearerAuth
// @Param id path int true "Update History ID"
// @Success 200 {object} utils.APIResponse{data=model.UpdateHistory} "Update details"
// @Failure 400 {object} utils.APIResponse "Invalid update ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Update not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/{id} [get]
func (uc *UpdateController) GetUpdateDetails(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	updateIDStr := c.Param("id")
	updateID, err := strconv.ParseInt(updateIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid update ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this is a placeholder implementation
	// In a real implementation, you would query the update history repository
	uc.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"update_id": updateID,
	}).Info("Update details requested (placeholder implementation)")

	rb.Error(http.StatusNotImplemented, "Update details retrieval not yet implemented")
}

// CancelUpdate godoc
// @Summary Cancel ongoing update
// @Description Cancel an ongoing update operation
// @Tags Updates
// @Produce json
// @Security BearerAuth
// @Param id path int true "Update History ID"
// @Success 200 {object} utils.APIResponse "Update cancelled"
// @Failure 400 {object} utils.APIResponse "Invalid update ID or cannot cancel"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Update not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/{id}/cancel [post]
func (uc *UpdateController) CancelUpdate(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	updateIDStr := c.Param("id")
	updateID, err := strconv.ParseInt(updateIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid update ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Get the update history record
	// 2. Check if the update is in a cancellable state
	// 3. Cancel the update operation
	// 4. Update the status accordingly

	uc.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"update_id": updateID,
	}).Info("Update cancellation requested (placeholder implementation)")

	rb.Error(http.StatusNotImplemented, "Update cancellation not yet implemented")
}

// GetUpdateMetrics godoc
// @Summary Get update metrics
// @Description Get update performance and success metrics
// @Tags Updates
// @Produce json
// @Security BearerAuth
// @Param period query string false "Time period (24h, 7d, 30d)" default(7d)
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "Update metrics"
// @Failure 400 {object} utils.APIResponse "Invalid period"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/metrics [get]
func (uc *UpdateController) GetUpdateMetrics(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	period := c.DefaultQuery("period", "7d")

	// Validate period
	var duration time.Duration
	switch period {
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		utils.BadRequestJSON(c, "Invalid period. Use: 24h, 7d, or 30d")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Calculate metrics for the specified period
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	// Build metrics response
	// In a real implementation, you'd query the update history repository
	metrics := map[string]interface{}{
		"period": period,
		"start_time": startTime,
		"end_time": endTime,
		"total_updates": 0,
		"successful_updates": 0,
		"failed_updates": 0,
		"rollbacks": 0,
		"success_rate": 0.0,
		"average_duration": "0s",
		"by_strategy": map[string]int{
			"recreate":   0,
			"rolling":    0,
			"blue_green": 0,
		},
		"by_status": map[string]int{
			"completed": 0,
			"failed":    0,
			"cancelled": 0,
			"in_progress": 0,
		},
		"containers_updated": 0,
		"most_updated_images": []map[string]interface{}{},
	}

	uc.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"period":  period,
	}).Info("Update metrics requested (placeholder implementation)")

	rb.Success(metrics)
}

// CheckAvailableUpdates godoc
// @Summary Check for available updates
// @Description Check for available updates across all containers
// @Tags Updates
// @Produce json
// @Security BearerAuth
// @Param force query boolean false "Force refresh cache" default(false)
// @Success 200 {object} utils.APIResponse{data=[]service.UpdateInfo} "Available updates"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/available [get]
func (uc *UpdateController) CheckAvailableUpdates(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))

	rb := utils.NewResponseBuilder(c)

	// Check for available updates
	updateInfos, err := uc.imageService.CheckAllImages(c.Request.Context())
	if err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to check for available updates")
		rb.InternalServerError("Failed to check for updates")
		return
	}

	// Filter to only return containers with available updates
	availableUpdates := make([]*service.UpdateInfo, 0)
	for _, updateInfo := range updateInfos {
		if updateInfo.UpdateAvailable {
			availableUpdates = append(availableUpdates, updateInfo)
		}
	}

	uc.logger.WithFields(logrus.Fields{
		"user_id":          userID,
		"force":            force,
		"total_containers": len(updateInfos),
		"available_updates": len(availableUpdates),
	}).Info("Available updates checked")

	rb.Success(availableUpdates)
}

// ScheduleUpdate godoc
// @Summary Schedule update
// @Description Schedule an update to be performed at a specific time
// @Tags Updates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Schedule request"
// @Success 200 {object} utils.APIResponse "Update scheduled"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/updates/schedule [post]
func (uc *UpdateController) ScheduleUpdate(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req struct {
		ContainerIDs  []int64                      `json:"container_ids" binding:"required"`
		ScheduledTime time.Time                    `json:"scheduled_time" binding:"required"`
		UpdateImage   *service.UpdateImageRequest  `json:"update_image,omitempty"`
		Config        map[string]interface{}       `json:"config,omitempty"`
		Description   string                       `json:"description,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid schedule update request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	// Validate request
	if len(req.ContainerIDs) == 0 {
		utils.BadRequestJSON(c, "At least one container ID is required")
		return
	}

	// Validate scheduled time is in the future
	if req.ScheduledTime.Before(time.Now()) {
		utils.BadRequestJSON(c, "Scheduled time must be in the future")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Create scheduled task records
	// 2. Set up the scheduler to execute the updates at the specified time
	// 3. Return the scheduled task information

	uc.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"container_ids":  req.ContainerIDs,
		"scheduled_time": req.ScheduledTime,
		"description":    req.Description,
	}).Info("Update scheduling requested (placeholder implementation)")

	rb.Error(http.StatusNotImplemented, "Update scheduling not yet implemented")
}