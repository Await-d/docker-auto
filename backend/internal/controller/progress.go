package controller

import (
	"strconv"

	"docker-auto/pkg/docker"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ProgressController handles progress tracking for Docker operations
type ProgressController struct {
	serviceManager *docker.ServiceManager
	logger         *logrus.Logger
}

// NewProgressController creates a new progress controller
func NewProgressController(serviceManager *docker.ServiceManager, logger *logrus.Logger) *ProgressController {
	return &ProgressController{
		serviceManager: serviceManager,
		logger:         logger,
	}
}

// GetPullProgress godoc
// @Summary Get image pull progress
// @Description Get real-time progress for image pull operation
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param image_name query string true "Image name to check progress for"
// @Success 200 {object} utils.APIResponse{data=docker.PullProgress} "Pull progress"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Progress not found"
// @Router /api/progress/pull [get]
func (pc *ProgressController) GetPullProgress(c *gin.Context) {
	imageName := c.Query("image_name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	clientManager := pc.serviceManager.GetClientManager()
	if clientManager == nil {
		rb.InternalServerError("Docker service not available")
		return
	}

	progress, found := clientManager.GetPullProgress(imageName)
	if !found {
		rb.NotFound("Pull progress not found for image: " + imageName)
		return
	}

	rb.Success(progress)
}

// GetOperationProgress godoc
// @Summary Get operation progress
// @Description Get real-time progress for any Docker operation
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param operation_id path string true "Operation ID"
// @Success 200 {object} utils.APIResponse{data=docker.OperationProgress} "Operation progress"
// @Failure 400 {object} utils.APIResponse "Invalid operation ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Operation not found"
// @Router /api/progress/operation/{operation_id} [get]
func (pc *ProgressController) GetOperationProgress(c *gin.Context) {
	operationID := c.Param("operation_id")
	if operationID == "" {
		utils.BadRequestJSON(c, "Operation ID is required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	notificationManager := pc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	operation, found := notificationManager.GetOperation(operationID)
	if !found {
		rb.NotFound("Operation not found: " + operationID)
		return
	}

	rb.Success(operation)
}

// GetUserOperations godoc
// @Summary Get user operations
// @Description Get all operations for the current user
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (started, running, completed, failed)"
// @Param limit query int false "Limit results" default(50)
// @Success 200 {object} utils.APIResponse{data=[]docker.OperationProgress} "User operations"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/progress/user [get]
func (pc *ProgressController) GetUserOperations(c *gin.Context) {
	// Get user ID from middleware
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedJSON(c, "User not authenticated")
		return
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		utils.UnauthorizedJSON(c, "Invalid user ID")
		return
	}

	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rb := utils.NewResponseBuilder(c)

	notificationManager := pc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	operations := notificationManager.GetUserOperations(userID)

	// Filter by status if specified
	if status != "" {
		filteredOps := make([]*docker.OperationProgress, 0)
		for _, op := range operations {
			if op.Status == status {
				filteredOps = append(filteredOps, op)
			}
		}
		operations = filteredOps
	}

	// Apply limit
	if len(operations) > limit {
		operations = operations[:limit]
	}

	rb.Success(operations)
}

// GetActiveOperations godoc
// @Summary Get active operations
// @Description Get all currently active operations (admin only)
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=[]docker.OperationProgress} "Active operations"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/progress/active [get]
func (pc *ProgressController) GetActiveOperations(c *gin.Context) {
	// Check if user is admin
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		utils.ForbiddenJSON(c, "Admin access required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	notificationManager := pc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	operations := notificationManager.GetActiveOperations()
	rb.Success(operations)
}

// GetServiceInfo godoc
// @Summary Get Docker service information
// @Description Get comprehensive information about the Docker service
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=docker.ServiceInfo} "Service information"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/progress/service [get]
func (pc *ProgressController) GetServiceInfo(c *gin.Context) {
	rb := utils.NewResponseBuilder(c)

	serviceInfo := pc.serviceManager.GetServiceInfo()
	rb.Success(serviceInfo)
}

// GetNotificationStats godoc
// @Summary Get notification statistics
// @Description Get statistics about WebSocket connections and notifications
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "Notification statistics"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/progress/notifications/stats [get]
func (pc *ProgressController) GetNotificationStats(c *gin.Context) {
	rb := utils.NewResponseBuilder(c)

	notificationManager := pc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	stats := notificationManager.GetStats()
	rb.Success(stats)
}

// GetRollbackHistory godoc
// @Summary Get rollback history
// @Description Get rollback operation history
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit results" default(20)
// @Success 200 {object} utils.APIResponse{data=[]docker.RollbackEntry} "Rollback history"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/progress/rollbacks [get]
func (pc *ProgressController) GetRollbackHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rb := utils.NewResponseBuilder(c)

	rollbackManager := pc.serviceManager.GetRollbackManager()
	if rollbackManager == nil {
		rb.InternalServerError("Rollback service not available")
		return
	}

	history := rollbackManager.GetRollbackHistory()

	// Apply limit
	if len(history) > limit {
		history = history[:limit]
	}

	rb.Success(history)
}

// GetRollbackEntry godoc
// @Summary Get rollback entry
// @Description Get details of a specific rollback operation
// @Tags Progress
// @Produce json
// @Security BearerAuth
// @Param rollback_id path string true "Rollback ID"
// @Success 200 {object} utils.APIResponse{data=docker.RollbackEntry} "Rollback entry"
// @Failure 400 {object} utils.APIResponse "Invalid rollback ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Rollback not found"
// @Router /api/progress/rollbacks/{rollback_id} [get]
func (pc *ProgressController) GetRollbackEntry(c *gin.Context) {
	rollbackID := c.Param("rollback_id")
	if rollbackID == "" {
		utils.BadRequestJSON(c, "Rollback ID is required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	rollbackManager := pc.serviceManager.GetRollbackManager()
	if rollbackManager == nil {
		rb.InternalServerError("Rollback service not available")
		return
	}

	entry, found := rollbackManager.GetRollbackEntry(rollbackID)
	if !found {
		rb.NotFound("Rollback entry not found: " + rollbackID)
		return
	}

	rb.Success(entry)
}