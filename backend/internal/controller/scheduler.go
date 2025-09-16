package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"docker-auto/internal/middleware"
	"docker-auto/internal/model"
	"docker-auto/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SchedulerController handles HTTP requests for scheduled tasks
type SchedulerController struct {
	schedulerService *service.SchedulerService
}

// NewSchedulerController creates a new scheduler controller
func NewSchedulerController(schedulerService *service.SchedulerService) *SchedulerController {
	return &SchedulerController{
		schedulerService: schedulerService,
	}
}

// RegisterRoutes registers scheduler routes
func (c *SchedulerController) RegisterRoutes(router *gin.RouterGroup) {
	scheduler := router.Group("/scheduler")
	scheduler.Use(middleware.JWT())

	// Scheduler status and control
	scheduler.GET("/status", c.GetSchedulerStatus)
	scheduler.POST("/start", middleware.Permission("admin"), c.StartScheduler)
	scheduler.POST("/stop", middleware.Permission("admin"), c.StopScheduler)

	// Task management
	tasks := scheduler.Group("/tasks")
	{
		tasks.POST("", c.CreateTask)
		tasks.GET("", c.ListTasks)
		tasks.GET("/:id", c.GetTask)
		tasks.PUT("/:id", c.UpdateTask)
		tasks.DELETE("/:id", c.DeleteTask)

		// Task control
		tasks.POST("/:id/pause", c.PauseTask)
		tasks.POST("/:id/resume", c.ResumeTask)
		tasks.POST("/:id/trigger", c.TriggerTask)

		// Task executions
		tasks.GET("/:id/executions", c.GetTaskExecutions)
	}

	// Task types information
	scheduler.GET("/task-types", c.GetTaskTypes)
	scheduler.GET("/cron-expressions", c.GetCronExpressions)
}

// GetSchedulerStatus returns the current scheduler status
func (c *SchedulerController) GetSchedulerStatus(ctx *gin.Context) {
	status, err := c.schedulerService.GetSchedulerStatus(ctx.Request.Context())
	if err != nil {
		logrus.WithError(err).Error("Failed to get scheduler status")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get scheduler status",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}

// StartScheduler starts the scheduler
func (c *SchedulerController) StartScheduler(ctx *gin.Context) {
	if c.schedulerService.IsRunning() {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Scheduler is already running",
			"running": true,
		})
		return
	}

	if err := c.schedulerService.Start(ctx.Request.Context()); err != nil {
		logrus.WithError(err).Error("Failed to start scheduler")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start scheduler",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Scheduler started successfully",
		"running": true,
	})
}

// StopScheduler stops the scheduler
func (c *SchedulerController) StopScheduler(ctx *gin.Context) {
	if !c.schedulerService.IsRunning() {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Scheduler is not running",
			"running": false,
		})
		return
	}

	if err := c.schedulerService.Stop(ctx.Request.Context()); err != nil {
		logrus.WithError(err).Error("Failed to stop scheduler")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to stop scheduler",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Scheduler stopped successfully",
		"running": false,
	})
}

// CreateTask creates a new scheduled task
func (c *SchedulerController) CreateTask(ctx *gin.Context) {
	userID := getUserID(ctx)

	var req service.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	task, err := c.schedulerService.CreateTask(ctx.Request.Context(), userID, &req)
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to create task")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to create task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"task":    task,
	})
}

// GetTask retrieves a specific task
func (c *SchedulerController) GetTask(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	task, err := c.schedulerService.GetTask(ctx.Request.Context(), userID, taskID)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to get task")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to get task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"task": task,
	})
}

// UpdateTask updates an existing task
func (c *SchedulerController) UpdateTask(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	var req service.UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := c.schedulerService.UpdateTask(ctx.Request.Context(), userID, taskID, &req); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to update task")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to update task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Task updated successfully",
	})
}

// DeleteTask deletes a task
func (c *SchedulerController) DeleteTask(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	if err := c.schedulerService.DeleteTask(ctx.Request.Context(), userID, taskID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to delete task")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to delete task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
	})
}

// ListTasks lists scheduled tasks
func (c *SchedulerController) ListTasks(ctx *gin.Context) {
	userID := getUserID(ctx)

	// Parse query parameters
	filter := &service.TaskFilter{}

	if name := ctx.Query("name"); name != "" {
		filter.Name = name
	}

	if taskType := ctx.Query("type"); taskType != "" {
		filter.Type = model.TaskType(taskType)
	}

	if isActiveStr := ctx.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	response, err := c.schedulerService.ListTasks(ctx.Request.Context(), userID, filter)
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to list tasks")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list tasks",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// PauseTask pauses a task
func (c *SchedulerController) PauseTask(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	if err := c.schedulerService.PauseTask(ctx.Request.Context(), userID, taskID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to pause task")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to pause task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Task paused successfully",
	})
}

// ResumeTask resumes a paused task
func (c *SchedulerController) ResumeTask(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	if err := c.schedulerService.ResumeTask(ctx.Request.Context(), userID, taskID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to resume task")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to resume task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Task resumed successfully",
	})
}

// TriggerTask manually triggers a task execution
func (c *SchedulerController) TriggerTask(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	if err := c.schedulerService.TriggerTask(ctx.Request.Context(), userID, taskID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to trigger task")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "scheduler is not running" {
			statusCode = http.StatusServiceUnavailable
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to trigger task",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Task triggered successfully",
	})
}

// GetTaskExecutions retrieves execution history for a task
func (c *SchedulerController) GetTaskExecutions(ctx *gin.Context) {
	userID := getUserID(ctx)
	taskID, err := getTaskID(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid task ID",
			"details": err.Error(),
		})
		return
	}

	// Parse query parameters
	filter := &service.ExecutionFilter{}

	if statusStr := ctx.Query("status"); statusStr != "" {
		filter.Status = model.ExecutionStatus(statusStr)
	}

	if startedAfterStr := ctx.Query("started_after"); startedAfterStr != "" {
		if startedAfter, err := time.Parse(time.RFC3339, startedAfterStr); err == nil {
			filter.StartedAfter = startedAfter
		}
	}

	if startedBeforeStr := ctx.Query("started_before"); startedBeforeStr != "" {
		if startedBefore, err := time.Parse(time.RFC3339, startedBeforeStr); err == nil {
			filter.StartedBefore = startedBefore
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	response, err := c.schedulerService.GetTaskExecutions(ctx.Request.Context(), userID, taskID, filter)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"task_id": taskID,
		}).Error("Failed to get task executions")

		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}

		ctx.JSON(statusCode, gin.H{
			"error":   "Failed to get task executions",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetTaskTypes returns available task types and their information
func (c *SchedulerController) GetTaskTypes(ctx *gin.Context) {
	taskTypes := []map[string]interface{}{
		{
			"type":        model.TaskTypeImageCheck,
			"name":        "Image Update Checker",
			"description": "Checks for available container image updates",
			"parameters": map[string]interface{}{
				"registry_timeout":     "Duration to wait for registry response",
				"max_concurrent":       "Maximum concurrent registry checks",
				"check_tags":           "Image tags to check for updates",
				"notify_on_new_image":  "Send notification when updates are found",
				"check_beta":           "Include beta versions in update checks",
				"only_security_updates": "Only check for security updates",
			},
		},
		{
			"type":        model.TaskTypeContainerUpdate,
			"name":        "Container Updater",
			"description": "Automatically updates containers to newer image versions",
			"parameters": map[string]interface{}{
				"update_strategy":     "Update strategy (recreate, rolling, blue-green)",
				"rollback_on_failure": "Rollback to previous version on failure",
				"pre_update_backup":   "Create backup before updating",
				"maintenance_windows": "Time windows when updates are allowed",
				"health_check_timeout": "Timeout for health checks after update",
			},
		},
		{
			"type":        model.TaskTypeCleanup,
			"name":        "System Cleanup",
			"description": "Cleans up old logs, images, and unused resources",
			"parameters": map[string]interface{}{
				"activity_log_retention_days": "Days to keep activity logs",
				"cleanup_unused_images":       "Remove unused Docker images",
				"cleanup_stopped_containers":  "Remove old stopped containers",
				"image_retention_days":        "Days to keep Docker images",
				"dry_run":                     "Preview cleanup without actual deletion",
			},
		},
		{
			"type":        model.TaskTypeHealthCheck,
			"name":        "Container Health Checker",
			"description": "Monitors container health and takes corrective actions",
			"parameters": map[string]interface{}{
				"check_timeout":      "Timeout for individual health checks",
				"restart_on_failure": "Restart unhealthy containers",
				"http_checks":        "HTTP endpoint health checks",
				"tcp_checks":         "TCP connectivity checks",
				"command_checks":     "Command-based health checks",
			},
		},
		{
			"type":        model.TaskTypeBackup,
			"name":        "System Backup",
			"description": "Creates backups of system data and configurations",
			"parameters": map[string]interface{}{
				"backup_type":              "Type of backup (full, incremental)",
				"backup_database":          "Include database in backup",
				"backup_configurations":    "Include system configurations",
				"backup_volumes":           "Include Docker volumes",
				"compress_backups":         "Compress backup files",
				"retention_days":           "Days to keep backup files",
			},
		},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"task_types": taskTypes,
	})
}

// GetCronExpressions returns common cron expressions with descriptions
func (c *SchedulerController) GetCronExpressions(ctx *gin.Context) {
	expressions := map[string]string{
		"* * * * *":     "Every minute",
		"*/5 * * * *":   "Every 5 minutes",
		"*/15 * * * *":  "Every 15 minutes",
		"*/30 * * * *":  "Every 30 minutes",
		"0 * * * *":     "Every hour",
		"0 */2 * * *":   "Every 2 hours",
		"0 */6 * * *":   "Every 6 hours",
		"0 */12 * * *":  "Every 12 hours",
		"0 0 * * *":     "Daily at midnight",
		"0 6 * * *":     "Daily at 6 AM",
		"0 12 * * *":    "Daily at noon",
		"0 18 * * *":    "Daily at 6 PM",
		"0 0 * * 0":     "Weekly on Sunday at midnight",
		"0 0 * * 1":     "Weekly on Monday at midnight",
		"0 0 1 * *":     "Monthly on the 1st at midnight",
		"0 0 1 1 *":     "Yearly on January 1st at midnight",
		"0 2 * * 0":     "Weekly on Sunday at 2 AM",
		"0 3 1,15 * *":  "Twice monthly (1st and 15th) at 3 AM",
		"0 4 * * 1-5":   "Weekdays at 4 AM",
		"0 1 * * 6,0":   "Weekends at 1 AM",
	}

	ctx.JSON(http.StatusOK, gin.H{
		"cron_expressions": expressions,
		"format":           "minute hour day month day_of_week",
		"help":            "Use * for any value, */N for every N units, or specific values",
	})
}

// Helper functions

// getUserID extracts user ID from the JWT token in context
func getUserID(ctx *gin.Context) int64 {
	if userID, exists := ctx.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			return id
		}
		if id, ok := userID.(float64); ok {
			return int64(id)
		}
	}
	return 0 // Should never happen with proper JWT middleware
}

// getTaskID extracts and validates task ID from URL parameter
func getTaskID(ctx *gin.Context) (int64, error) {
	taskIDStr := ctx.Param("id")
	if taskIDStr == "" {
		return 0, fmt.Errorf("task ID is required")
	}

	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid task ID format")
	}

	if taskID <= 0 {
		return 0, fmt.Errorf("task ID must be positive")
	}

	return taskID, nil
}