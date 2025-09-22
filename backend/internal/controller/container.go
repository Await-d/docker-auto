package controller

import (
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

// ContainerController handles container-related HTTP requests
type ContainerController struct {
	containerService *service.ContainerService
	logger           *logrus.Logger
}

// NewContainerController creates a new container controller
func NewContainerController(containerService *service.ContainerService, logger *logrus.Logger) *ContainerController {
	return &ContainerController{
		containerService: containerService,
		logger:           logger,
	}
}

// Container CRUD operations

// ListContainers godoc
// @Summary List managed containers
// @Description Get paginated list of managed containers with filtering and sorting
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search by name or image"
// @Param status query string false "Filter by status"
// @Param update_policy query string false "Filter by update policy"
// @Param has_update query boolean false "Filter containers with available updates"
// @Param sort_by query string false "Sort field" default(updated_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} utils.APIResponse{data=service.ContainerListResponse} "Containers list"
// @Failure 400 {object} utils.APIResponse "Invalid request parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers [get]
func (cc *ContainerController) ListContainers(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	status := c.Query("status")
	updatePolicy := c.Query("update_policy")
	hasUpdateStr := c.Query("has_update")
	sortBy := c.DefaultQuery("sort_by", "updated_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Parse boolean parameter
	var hasUpdate *bool
	if hasUpdateStr != "" {
		if hasUpdateStr == "true" {
			val := true
			hasUpdate = &val
		} else if hasUpdateStr == "false" {
			val := false
			hasUpdate = &val
		}
	}

	// Build filter
	filter := &service.ContainerFilter{
		SearchQuery: search,
		HasUpdate:   hasUpdate,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
	}

	// Set model filter properties
	filter.ContainerFilter = &model.ContainerFilter{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	if status != "" {
		filter.ContainerFilter.Status = model.ContainerStatus(status)
	}
	if updatePolicy != "" {
		filter.ContainerFilter.UpdatePolicy = model.UpdatePolicy(updatePolicy)
	}

	rb := utils.NewResponseBuilder(c)

	response, err := cc.containerService.ListContainers(c.Request.Context(), userID, filter)
	if err != nil {
		cc.logger.WithError(err).WithField("user_id", userID).Error("Failed to list containers")
		rb.InternalServerError("Failed to retrieve containers")
		return
	}

	rb.SuccessWithPagination(response.Containers, &utils.Pagination{
		Page:       response.Page,
		Limit:      response.Limit,
		Total:      response.Total,
		TotalPages: int((response.Total + int64(response.Limit) - 1) / int64(response.Limit)),
		HasNext:    response.HasNext,
		HasPrev:    response.HasPrev,
	})
}

// CreateContainer godoc
// @Summary Add container to management
// @Description Add a new container to the management system
// @Tags Containers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.CreateContainerRequest true "Container configuration"
// @Success 201 {object} utils.APIResponse{data=model.Container} "Container created successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 409 {object} utils.APIResponse "Container name already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers [post]
func (cc *ContainerController) CreateContainer(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req service.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		cc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid create container request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	container, err := cc.containerService.CreateContainer(c.Request.Context(), userID, &req)
	if err != nil {
		cc.logger.WithError(err).WithField("user_id", userID).Error("Failed to create container")
		if err.Error() == "container with name '"+req.Name+"' already exists" {
			rb.Conflict("Container name already exists")
			return
		}
		rb.InternalServerError("Failed to create container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": container.ID,
		"name":         container.Name,
	}).Info("Container created successfully")

	rb.Created(container)
}

// GetContainer godoc
// @Summary Get container details
// @Description Get detailed information about a specific container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse{data=service.ContainerDetail} "Container details"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id} [get]
func (cc *ContainerController) GetContainer(c *gin.Context) {
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

	detail, err := cc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container")
		rb.NotFound("Container not found")
		return
	}

	rb.Success(detail)
}

// UpdateContainer godoc
// @Summary Update container configuration
// @Description Update configuration of an existing container
// @Tags Containers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param request body service.UpdateContainerRequest true "Container update data"
// @Success 200 {object} utils.APIResponse "Container updated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id} [put]
func (cc *ContainerController) UpdateContainer(c *gin.Context) {
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

	var req service.UpdateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Warn("Invalid update container request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := cc.containerService.UpdateContainer(c.Request.Context(), userID, containerID, &req); err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to update container")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to update container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container updated successfully")

	rb.SuccessWithMessage(nil, "Container updated successfully")
}

// DeleteContainer godoc
// @Summary Remove container from management
// @Description Remove a container from the management system
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse "Container deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id} [delete]
func (cc *ContainerController) DeleteContainer(c *gin.Context) {
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

	if err := cc.containerService.DeleteContainer(c.Request.Context(), userID, containerID); err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to delete container")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to delete container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container deleted successfully")

	rb.SuccessWithMessage(nil, "Container deleted successfully")
}

// Container operations

// StartContainer godoc
// @Summary Start container
// @Description Start a stopped container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse "Container started successfully"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/start [post]
func (cc *ContainerController) StartContainer(c *gin.Context) {
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

	if err := cc.containerService.StartContainer(c.Request.Context(), userID, containerID); err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to start container")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to start container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container started successfully")

	rb.SuccessWithMessage(nil, "Container started successfully")
}

// StopContainer godoc
// @Summary Stop container
// @Description Stop a running container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse "Container stopped successfully"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/stop [post]
func (cc *ContainerController) StopContainer(c *gin.Context) {
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

	if err := cc.containerService.StopContainer(c.Request.Context(), userID, containerID); err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to stop container")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to stop container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container stopped successfully")

	rb.SuccessWithMessage(nil, "Container stopped successfully")
}

// RestartContainer godoc
// @Summary Restart container
// @Description Restart a container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse "Container restarted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/restart [post]
func (cc *ContainerController) RestartContainer(c *gin.Context) {
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

	if err := cc.containerService.RestartContainer(c.Request.Context(), userID, containerID); err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to restart container")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to restart container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
	}).Info("Container restarted successfully")

	rb.SuccessWithMessage(nil, "Container restarted successfully")
}

// UpdateContainerImage godoc
// @Summary Trigger manual update
// @Description Trigger a manual update for a container
// @Tags Containers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param request body service.UpdateImageRequest false "Update options"
// @Success 200 {object} utils.APIResponse{data=model.UpdateHistory} "Update initiated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/update [post]
func (cc *ContainerController) UpdateContainerImage(c *gin.Context) {
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

	var req service.UpdateImageRequest
	// Optional request body
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			cc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":      userID,
				"container_id": containerID,
			}).Warn("Invalid update image request")
			utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
			return
		}
	}

	rb := utils.NewResponseBuilder(c)

	updateHistory, err := cc.containerService.UpdateContainerImage(c.Request.Context(), userID, containerID, &req)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to update container image")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to update container")
		return
	}

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"container_id": containerID,
		"update_id":    updateHistory.ID,
	}).Info("Container update initiated successfully")

	rb.Success(updateHistory)
}

// GetContainerLogs godoc
// @Summary Get container logs
// @Description Retrieve logs from a container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param tail query int false "Number of log lines to retrieve" default(100)
// @Param since query string false "Show logs since timestamp (RFC3339)"
// @Param until query string false "Show logs until timestamp (RFC3339)"
// @Param follow query boolean false "Follow log output"
// @Param timestamps query boolean false "Include timestamps" default(true)
// @Success 200 {object} utils.APIResponse{data=service.LogResponse} "Container logs"
// @Failure 400 {object} utils.APIResponse "Invalid request parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/logs [get]
func (cc *ContainerController) GetContainerLogs(c *gin.Context) {
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

	// Parse query parameters
	tail, _ := strconv.Atoi(c.DefaultQuery("tail", "100"))
	sinceStr := c.Query("since")
	untilStr := c.Query("until")
	follow, _ := strconv.ParseBool(c.DefaultQuery("follow", "false"))
	timestamps, _ := strconv.ParseBool(c.DefaultQuery("timestamps", "true"))

	// Create log options
	options := &service.LogOptions{
		Tail:       tail,
		Follow:     follow,
		Timestamps: timestamps,
	}

	// Parse timestamps
	if sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			options.Since = since
		}
	}
	if untilStr != "" {
		if until, err := time.Parse(time.RFC3339, untilStr); err == nil {
			options.Until = until
		}
	}

	rb := utils.NewResponseBuilder(c)

	logResponse, err := cc.containerService.GetContainerLogs(c.Request.Context(), userID, containerID, options)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container logs")
		if err.Error() == "container not found" || err.Error() == "container has no Docker instance" {
			rb.NotFound("Container not found or not running")
			return
		}
		rb.InternalServerError("Failed to retrieve logs")
		return
	}

	rb.Success(logResponse)
}

// GetContainerStats godoc
// @Summary Get container statistics
// @Description Get resource usage statistics for a container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse{data=service.ContainerStats} "Container statistics"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/stats [get]
func (cc *ContainerController) GetContainerStats(c *gin.Context) {
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

	stats, err := cc.containerService.GetContainerStats(c.Request.Context(), userID, containerID)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container stats")
		if err.Error() == "container not found" || err.Error() == "container has no Docker instance" {
			rb.NotFound("Container not found or not running")
			return
		}
		rb.InternalServerError("Failed to retrieve statistics")
		return
	}

	rb.Success(stats)
}

// GetContainerStatus godoc
// @Summary Get container status
// @Description Get current status information for a container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Success 200 {object} utils.APIResponse{data=service.ContainerStatus} "Container status"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/status [get]
func (cc *ContainerController) GetContainerStatus(c *gin.Context) {
	containerIDStr := c.Param("id")
	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	status, err := cc.containerService.GetContainerStatus(c.Request.Context(), containerID)
	if err != nil {
		cc.logger.WithError(err).WithField("container_id", containerID).Error("Failed to get container status")
		if err.Error() == "container not found" {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to retrieve status")
		return
	}

	rb.Success(status)
}

// Bulk operations

// BulkContainerOperation godoc
// @Summary Bulk container operation
// @Description Perform bulk operations on multiple containers
// @Tags Containers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.BulkUpdateRequest true "Bulk operation request"
// @Success 200 {object} utils.APIResponse{data=[]service.OperationResult} "Bulk operation results"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/bulk [post]
func (cc *ContainerController) BulkContainerOperation(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req service.BulkUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		cc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid bulk operation request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Validate container IDs
	if len(req.ContainerIDs) == 0 {
		rb.BadRequest("At least one container ID is required")
		return
	}

	results := make([]service.OperationResult, 0, len(req.ContainerIDs))

	// Process each container
	for _, containerID := range req.ContainerIDs {
		result := service.OperationResult{
			ContainerID: containerID,
			Success:     false,
		}

		// Get container name for result
		if container, err := cc.containerService.GetContainer(c.Request.Context(), userID, containerID); err == nil {
			result.Name = container.Container.Name
		}

		var err error
		switch req.Action {
		case "start":
			err = cc.containerService.StartContainer(c.Request.Context(), userID, containerID)
		case "stop":
			err = cc.containerService.StopContainer(c.Request.Context(), userID, containerID)
		case "restart":
			err = cc.containerService.RestartContainer(c.Request.Context(), userID, containerID)
		case "update":
			if req.UpdateImage != nil {
				_, err = cc.containerService.UpdateContainerImage(c.Request.Context(), userID, containerID, req.UpdateImage)
			} else {
				err = cc.containerService.UpdateContainer(c.Request.Context(), userID, containerID, &service.UpdateContainerRequest{
					Config: req.Config,
				})
			}
		default:
			err = fmt.Errorf("invalid action: %s", req.Action)
		}

		if err != nil {
			result.Error = err.Error()
			cc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":      userID,
				"container_id": containerID,
				"action":       req.Action,
			}).Warn("Bulk operation failed for container")
		} else {
			result.Success = true
			result.Message = "Operation completed successfully"
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

	cc.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"action":       req.Action,
		"total":        len(req.ContainerIDs),
		"success":      successCount,
		"failed":       len(req.ContainerIDs) - successCount,
	}).Info("Bulk container operation completed")

	rb.Success(results)
}

// SyncContainerStatus godoc
// @Summary Sync container status
// @Description Synchronize container status with Docker daemon
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=service.SyncResult} "Sync completed"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/sync [post]
func (cc *ContainerController) SyncContainerStatus(c *gin.Context) {
	rb := utils.NewResponseBuilder(c)

	if err := cc.containerService.SyncContainerStatus(c.Request.Context()); err != nil {
		cc.logger.WithError(err).Error("Failed to sync container status")
		rb.InternalServerError("Failed to sync container status")
		return
	}

	cc.logger.Info("Container status sync completed")
	rb.SuccessWithMessage(nil, "Container status synchronized successfully")
}

// WebSocket upgrader for terminal connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, you should check the origin properly
		return true
	},
}

// ContainerTerminal godoc
// @Summary Connect to container terminal via WebSocket
// @Description Establish WebSocket connection for container terminal access
// @Tags Containers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param cmd query string false "Command to execute" default("/bin/sh")
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/terminal [get]
func (cc *ContainerController) ContainerTerminal(c *gin.Context) {
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

	// Get command from query parameter, default to /bin/sh
	cmd := c.DefaultQuery("cmd", "/bin/sh")

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		cc.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		utils.InternalServerErrorJSON(c, "Failed to establish WebSocket connection")
		return
	}
	defer conn.Close()

	// Get container information
	container, err := cc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container")
		conn.WriteMessage(websocket.TextMessage, []byte("Error: Container not found"))
		return
	}

	if container.Status != model.ContainerStatusRunning {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: Container is not running"))
		return
	}

	// Execute terminal session
	err = cc.executeTerminalSession(conn, container.Container.ContainerID, cmd)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
			"docker_id":    container.Container.ContainerID,
		}).Error("Terminal session failed")
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
	}
}

// executeTerminalSession handles the terminal session between WebSocket and Docker container
func (cc *ContainerController) executeTerminalSession(conn *websocket.Conn, dockerID, cmd string) error {
	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Create a Docker exec session
	// 2. Attach to the exec session with TTY
	// 3. Pipe WebSocket data to/from the exec session

	// For now, we'll simulate a terminal session
	welcome := fmt.Sprintf("Connected to container %s\r\nTerminal session started with command: %s\r\n", dockerID[:12], cmd)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(welcome)); err != nil {
		return err
	}

	prompt := fmt.Sprintf("root@%s:~$ ", dockerID[:12])
	if err := conn.WriteMessage(websocket.TextMessage, []byte(prompt)); err != nil {
		return err
	}

	// Handle WebSocket messages
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				cc.logger.WithError(err).Error("WebSocket connection closed unexpectedly")
			}
			break
		}

		if messageType == websocket.TextMessage {
			input := string(data)

			// Echo input back and simulate command execution
			response := fmt.Sprintf("%s\r\n", input)

			// Simulate some common commands
			switch input {
			case "ls":
				response += "bin  boot  dev  etc  home  lib  lib64  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var\r\n"
			case "pwd":
				response += "/root\r\n"
			case "whoami":
				response += "root\r\n"
			case "date":
				response += fmt.Sprintf("%s\r\n", time.Now().Format("Mon Jan 2 15:04:05 MST 2006"))
			case "exit":
				response += "Connection closed\r\n"
				conn.WriteMessage(websocket.TextMessage, []byte(response))
				return nil
			default:
				if input != "" {
					response += fmt.Sprintf("bash: %s: command not found\r\n", input)
				}
			}

			response += prompt

			if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetContainerEvents godoc
// @Summary Get container events
// @Description Get real-time and historical events for a container
// @Tags Containers
// @Produce json
// @Security BearerAuth
// @Param id path int true "Container ID"
// @Param since query string false "Show events since timestamp (RFC3339)"
// @Param until query string false "Show events until timestamp (RFC3339)"
// @Success 200 {object} utils.APIResponse{data=[]ContainerEvent} "Container events"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/containers/{id}/events [get]
func (cc *ContainerController) GetContainerEvents(c *gin.Context) {
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
	var since, until *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		if parsed, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &parsed
		}
	}
	if untilStr := c.Query("until"); untilStr != "" {
		if parsed, err := time.Parse(time.RFC3339, untilStr); err == nil {
			until = &parsed
		}
	}

	rb := utils.NewResponseBuilder(c)

	// Get container information
	container, err := cc.containerService.GetContainer(c.Request.Context(), userID, containerID)
	if err != nil {
		cc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
		}).Error("Failed to get container")
		rb.NotFound("Container not found")
		return
	}

	// Generate sample events for now
	// In a real implementation, you would fetch actual Docker events
	events := cc.generateSampleEvents(container, since, until)

	rb.Success(events)
}

// ContainerEvent represents a container event
type ContainerEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Action      string                 `json:"action"`
	Actor       map[string]interface{} `json:"actor"`
	Time        time.Time              `json:"time"`
	TimeNano    int64                  `json:"timeNano"`
	Scope       string                 `json:"scope"`
	Message     string                 `json:"message"`
	Level       string                 `json:"level"`
}

// generateSampleEvents creates sample events for demonstration
func (cc *ContainerController) generateSampleEvents(container *service.ContainerDetail, since, until *time.Time) []ContainerEvent {
	now := time.Now()
	events := []ContainerEvent{
		{
			ID:      "1",
			Type:    "container",
			Action:  "start",
			Time:    now.Add(-time.Hour * 2),
			TimeNano: now.Add(-time.Hour * 2).UnixNano(),
			Scope:   "local",
			Message: "Container started successfully",
			Level:   "info",
			Actor: map[string]interface{}{
				"ID": container.Container.ContainerID,
				"Attributes": map[string]string{
					"image": container.Container.Image,
					"name":  container.Container.Name,
				},
			},
		},
		{
			ID:      "2",
			Type:    "container",
			Action:  "health_status",
			Time:    now.Add(-time.Minute * 30),
			TimeNano: now.Add(-time.Minute * 30).UnixNano(),
			Scope:   "local",
			Message: "Health check passed",
			Level:   "info",
			Actor: map[string]interface{}{
				"ID": container.Container.ContainerID,
				"Attributes": map[string]string{
					"image": container.Container.Image,
					"name":  container.Container.Name,
				},
			},
		},
		{
			ID:      "3",
			Type:    "container",
			Action:  "update_config",
			Time:    now.Add(-time.Minute * 15),
			TimeNano: now.Add(-time.Minute * 15).UnixNano(),
			Scope:   "local",
			Message: "Container configuration updated",
			Level:   "info",
			Actor: map[string]interface{}{
				"ID": container.Container.ContainerID,
				"Attributes": map[string]string{
					"image": container.Container.Image,
					"name":  container.Container.Name,
				},
			},
		},
	}

	// Filter by time range if specified
	if since != nil || until != nil {
		filtered := []ContainerEvent{}
		for _, event := range events {
			if since != nil && event.Time.Before(*since) {
				continue
			}
			if until != nil && event.Time.After(*until) {
				continue
			}
			filtered = append(filtered, event)
		}
		events = filtered
	}

	return events
}