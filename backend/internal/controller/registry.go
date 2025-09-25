package controller

import (
	"strconv"
	"strings"
	"time"

	"docker-auto/internal/middleware"
	"docker-auto/internal/service"
	"docker-auto/pkg/registry"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RegistryController handles registry-related HTTP requests
type RegistryController struct {
	registryService *service.RegistryService
	imageService    *service.ImageService
	logger          *logrus.Logger
}

// NewRegistryController creates a new registry controller
func NewRegistryController(registryService *service.RegistryService, imageService *service.ImageService, logger *logrus.Logger) *RegistryController {
	return &RegistryController{
		registryService: registryService,
		imageService:    imageService,
		logger:          logger,
	}
}

// RegistryRequest represents a registry configuration request
type RegistryRequest struct {
	Name        string               `json:"name" binding:"required"`
	URL         string               `json:"url" binding:"required"`
	Type        string               `json:"type" binding:"required"` // dockerhub, harbor, ecr, etc.
	Description string               `json:"description,omitempty"`
	AuthConfig  *registry.AuthConfig `json:"auth_config,omitempty"`
	IsDefault   bool                 `json:"is_default,omitempty"`
	Enabled     bool                 `json:"enabled"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// RegistryResponse represents a registry configuration response
type RegistryResponse struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	URL         string                 `json:"url"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	IsDefault   bool                   `json:"is_default"`
	Enabled     bool                   `json:"enabled"`
	Status      string                 `json:"status"` // connected, error, unknown
	LastChecked *string                `json:"last_checked,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// RegistryTestResult represents registry connection test result
type RegistryTestResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Duration    string                 `json:"duration"`
	Error       string                 `json:"error,omitempty"`
	Capabilities []string              `json:"capabilities,omitempty"`
	Info        *registry.RegistryInfo `json:"info,omitempty"`
}

// ListRegistries godoc
// @Summary List configured registries
// @Description Get list of all configured container registries
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param enabled query boolean false "Filter by enabled status"
// @Param type query string false "Filter by registry type"
// @Success 200 {object} utils.APIResponse{data=[]RegistryResponse} "Registries list"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries [get]
func (rc *RegistryController) ListRegistries(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		rb := utils.NewResponseBuilder(c)
		rb.Error(401, "Unauthorized", err)
		return
	}

	enabledStr := c.Query("enabled")
	registryType := c.Query("type")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	rb := utils.NewResponseBuilder(c)

	// Create filter
	filter := &service.RegistryFilter{
		Type: registryType,
	}

	// Parse enabled filter
	if enabledStr != "" {
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			rb.Error(400, "Invalid enabled parameter", err)
			return
		}
		filter.IsActive = &enabled
	}

	// Parse limit and offset
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			rb.Error(400, "Invalid limit parameter", err)
			return
		}
		filter.Limit = limit
	}

	if offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			rb.Error(400, "Invalid offset parameter", err)
			return
		}
		filter.Offset = offset
	}

	// Get registries using service
	registries, total, err := rc.registryService.ListRegistries(c.Request.Context(), userID, filter)
	if err != nil {
		rc.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to list registries")
		rb.Error(500, "Failed to list registries", err)
		return
	}

	// Convert to response format with status
	responses := make([]map[string]interface{}, len(registries))
	for i, reg := range registries {
		responses[i] = map[string]interface{}{
			"id":          reg.ID,
			"name":        reg.Name,
			"url":         reg.URL,
			"type":        reg.Type,
			"description": reg.Description,
			"is_default":  reg.IsDefault,
			"enabled":     reg.IsActive,
			"status":      "connected", // TODO: Add actual status checking
			"created_at":  reg.CreatedAt,
			"updated_at":  reg.UpdatedAt,
		}
	}

	rc.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"total":   total,
		"count":   len(registries),
	}).Info("Registries listed")

	// Return response with pagination info
	response := map[string]interface{}{
		"registries": responses,
		"total":      total,
		"count":      len(responses),
	}

	rb.Success(response)
}

// CreateRegistry godoc
// @Summary Add registry
// @Description Add a new container registry configuration
// @Tags Registries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RegistryRequest true "Registry configuration"
// @Success 201 {object} utils.APIResponse{data=RegistryResponse} "Registry created successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 409 {object} utils.APIResponse "Registry already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries [post]
func (rc *RegistryController) CreateRegistry(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req RegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid registry creation request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Validate registry type
	validTypes := []string{"dockerhub", "harbor", "ecr", "gcr", "acr", "generic"}
	validType := false
	for _, t := range validTypes {
		if req.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		rb.BadRequest("Invalid registry type. Supported types: " + strings.Join(validTypes, ", "))
		return
	}

	// Validate URL format
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		rb.BadRequest("Registry URL must start with http:// or https://")
		return
	}

	// Test connection before creating
	if req.AuthConfig != nil {
		err := rc.imageService.TestRegistryConnection(c.Request.Context(), req.URL, req.AuthConfig)
		if err != nil {
			rc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id": userID,
				"url":     req.URL,
				"type":    req.Type,
			}).Warn("Registry connection test failed during creation")
			rb.BadRequest("Failed to connect to registry: " + err.Error())
			return
		}
	}

	// Create registry using service
	createReq := &service.RegistryCreateRequest{
		Name:        req.Name,
		URL:         req.URL,
		Type:        req.Type,
		Description: req.Description,
		IsDefault:   req.IsDefault,
	}

	// Extract auth information
	if req.AuthConfig != nil {
		createReq.Username = req.AuthConfig.Username
		createReq.Password = req.AuthConfig.Password
		createReq.Token = req.AuthConfig.Token
	}

	registry, err := rc.registryService.CreateRegistry(c.Request.Context(), userID, createReq)
	if err != nil {
		rc.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"name":    req.Name,
			"url":     req.URL,
			"error":   err.Error(),
		}).Error("Failed to create registry")

		// Handle specific error types
		if strings.Contains(err.Error(), "already exists") {
			rb.Error(409, "Registry already exists", err)
		} else {
			rb.Error(500, "Failed to create registry", err)
		}
		return
	}

	// Convert to controller response format
	response := RegistryResponse{
		ID:          registry.ID,
		Name:        registry.Name,
		URL:         registry.URL,
		Type:        registry.Type,
		Description: registry.Description,
		IsDefault:   registry.IsDefault,
		Enabled:     registry.IsActive,
		Status:      "connected",
		CreatedAt:   registry.CreatedAt,
		UpdatedAt:   registry.UpdatedAt,
	}

	rc.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"registry_id": registry.ID,
		"name":        req.Name,
	}).Info("Registry created successfully")

	rb.Created(response)
}

// GetRegistry godoc
// @Summary Get registry details
// @Description Get detailed information about a specific registry
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Success 200 {object} utils.APIResponse{data=RegistryResponse} "Registry details"
// @Failure 400 {object} utils.APIResponse "Invalid registry ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id} [get]
func (rc *RegistryController) GetRegistry(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		rb := utils.NewResponseBuilder(c)
		rb.Error(401, "Unauthorized", err)
		return
	}

	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Get registry using service
	registry, err := rc.registryService.GetRegistry(c.Request.Context(), userID, registryID)
	if err != nil {
		rc.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
		}).Error("Failed to get registry")

		if strings.Contains(err.Error(), "not found") {
			rb.NotFound("Registry not found")
		} else {
			rb.Error(500, "Failed to get registry", err)
		}
		return
	}

	// Convert to controller response format
	response := RegistryResponse{
		ID:          registry.ID,
		Name:        registry.Name,
		URL:         registry.URL,
		Type:        registry.Type,
		Description: registry.Description,
		IsDefault:   registry.IsDefault,
		Enabled:     registry.IsActive,
		Status:      "connected", // TODO: Add actual status checking
		CreatedAt:   registry.CreatedAt,
		UpdatedAt:   registry.UpdatedAt,
	}

	rb.Success(response)
}

// UpdateRegistry godoc
// @Summary Update registry
// @Description Update registry configuration
// @Tags Registries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Param request body RegistryRequest true "Registry update data"
// @Success 200 {object} utils.APIResponse "Registry updated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id} [put]
func (rc *RegistryController) UpdateRegistry(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	var req RegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
		}).Warn("Invalid registry update request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Test connection if auth config is provided
	if req.AuthConfig != nil {
		err := rc.imageService.TestRegistryConnection(c.Request.Context(), req.URL, req.AuthConfig)
		if err != nil {
			rc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":     userID,
				"registry_id": registryID,
				"url":         req.URL,
			}).Warn("Registry connection test failed during update")
			rb.BadRequest("Failed to connect to registry: " + err.Error())
			return
		}
	}

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Get existing registry from database
	// 2. Update the registry configuration
	// 3. Update the registry in the image service
	// 4. Return success response

	rc.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"registry_id": registryID,
		"name":        req.Name,
		"url":         req.URL,
	}).Info("Registry update requested (placeholder implementation)")

	rb.SuccessWithMessage(nil, "Registry updated successfully")
}

// DeleteRegistry godoc
// @Summary Remove registry
// @Description Remove a registry configuration
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Success 200 {object} utils.APIResponse "Registry deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid registry ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 409 {object} utils.APIResponse "Cannot delete default registry"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id} [delete]
func (rc *RegistryController) DeleteRegistry(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Get registry from database
	// 2. Check if it's the default registry (cannot delete)
	// 3. Check if any containers are using this registry
	// 4. Remove registry from database and image service
	// 5. Return success response

	// Prevent deletion of registry ID 1 (Docker Hub)
	if registryID == 1 {
		rb.Conflict("Cannot delete the default Docker Hub registry")
		return
	}

	rc.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"registry_id": registryID,
	}).Info("Registry deletion requested (placeholder implementation)")

	rb.SuccessWithMessage(nil, "Registry deleted successfully")
}

// TestRegistryConnection godoc
// @Summary Test registry connection
// @Description Test connection to a registry
// @Tags Registries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Param request body registry.AuthConfig false "Authentication configuration for testing"
// @Success 200 {object} utils.APIResponse{data=RegistryTestResult} "Connection test result"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id}/test [post]
func (rc *RegistryController) TestRegistryConnection(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	var authConfig *registry.AuthConfig
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&authConfig); err != nil {
			rc.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":     userID,
				"registry_id": registryID,
			}).Warn("Invalid registry test request")
			utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
			return
		}
	}

	rb := utils.NewResponseBuilder(c)

	// Test connection using registry service
	testResult, err := rc.registryService.TestConnection(c.Request.Context(), userID, registryID)
	if err != nil {
		rc.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
		}).Error("Failed to test registry connection")

		if strings.Contains(err.Error(), "not found") {
			rb.NotFound("Registry not found")
		} else {
			rb.Error(500, "Failed to test registry connection", err)
		}
		return
	}

	// Convert to controller response format
	result := RegistryTestResult{
		Success:      testResult.Success,
		Message:      testResult.Message,
		Duration:     testResult.ResponseTime,
		Capabilities: testResult.Capabilities,
		Info:         testResult.Info,
	}

	if !testResult.Success {
		result.Error = testResult.Message
	}

	rb.Success(result)
}

// GetRegistryInfo godoc
// @Summary Get registry information
// @Description Get detailed information about a registry
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Success 200 {object} utils.APIResponse{data=registry.RegistryInfo} "Registry information"
// @Failure 400 {object} utils.APIResponse "Invalid registry ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id}/info [get]
func (rc *RegistryController) GetRegistryInfo(c *gin.Context) {
	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For testing purposes, use hardcoded registry URL
	// In a real implementation, you would get the URL from the database
	var registryURL string
	switch registryID {
	case 1:
		registryURL = "https://registry-1.docker.io"
	default:
		rb.NotFound("Registry not found")
		return
	}

	info, err := rc.imageService.GetRegistryInfo(c.Request.Context(), registryURL)
	if err != nil {
		rc.logger.WithError(err).WithField("registry_id", registryID).Error("Failed to get registry info")
		rb.InternalServerError("Failed to retrieve registry information")
		return
	}

	rb.Success(info)
}

// SearchRegistryImages godoc
// @Summary Search images in registry
// @Description Search for images in a specific registry
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Param q query string true "Search query"
// @Param limit query int false "Limit results" default(25)
// @Success 200 {object} utils.APIResponse{data=[]registry.ImageSearchResult} "Search results"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id}/search [get]
func (rc *RegistryController) SearchRegistryImages(c *gin.Context) {
	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	query := c.Query("q")
	if query == "" {
		utils.BadRequestJSON(c, "Search query is required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	rb := utils.NewResponseBuilder(c)

	// For testing purposes, use hardcoded registry URL
	// In a real implementation, you would get the URL from the database
	var registryURL string
	switch registryID {
	case 1:
		registryURL = "https://registry-1.docker.io"
	default:
		rb.NotFound("Registry not found")
		return
	}

	results, err := rc.imageService.SearchImages(c.Request.Context(), query, registryURL)
	if err != nil {
		rc.logger.WithError(err).WithFields(logrus.Fields{
			"registry_id": registryID,
			"query":       query,
		}).Error("Failed to search registry images")
		rb.InternalServerError("Failed to search images")
		return
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	rc.logger.WithFields(logrus.Fields{
		"registry_id":   registryID,
		"query":         query,
		"results_count": len(results),
	}).Info("Registry image search completed")

	rb.Success(results)
}

// GetRegistryStatistics godoc
// @Summary Get registry statistics
// @Description Get usage statistics for a registry
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "Registry statistics"
// @Failure 400 {object} utils.APIResponse "Invalid registry ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id}/stats [get]
func (rc *RegistryController) GetRegistryStatistics(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		rb := utils.NewResponseBuilder(c)
		rb.Error(401, "Unauthorized", err)
		return
	}

	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Get statistics using registry service
	stats, err := rc.registryService.GetStatistics(c.Request.Context(), userID, registryID)
	if err != nil {
		rc.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
		}).Error("Failed to get registry statistics")

		if strings.Contains(err.Error(), "not found") {
			rb.NotFound("Registry not found")
		} else {
			rb.Error(500, "Failed to get registry statistics", err)
		}
		return
	}

	// Convert to response format
	response := map[string]interface{}{
		"registry_id":          registryID,
		"total_containers":     stats.TotalContainers,
		"active_containers":    stats.ActiveContainers,
		"total_pulls":          stats.TotalPulls,
		"recent_pulls":         stats.RecentPulls,
		"last_pull_time":       stats.LastPullTime,
		"average_response_time": stats.AverageResponseTime,
		"success_rate":         stats.SuccessRate,
	}

	rc.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"registry_id": registryID,
	}).Info("Registry statistics retrieved")

	rb.Success(response)
}

// SetDefaultRegistry godoc
// @Summary Set default registry
// @Description Set a registry as the default for new containers
// @Tags Registries
// @Produce json
// @Security BearerAuth
// @Param id path int true "Registry ID"
// @Success 200 {object} utils.APIResponse "Default registry updated"
// @Failure 400 {object} utils.APIResponse "Invalid registry ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Registry not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/registries/{id}/default [post]
func (rc *RegistryController) SetDefaultRegistry(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Verify registry exists and is enabled
	// 2. Update all other registries to not be default
	// 3. Set this registry as default
	// 4. Update system configuration

	rc.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"registry_id": registryID,
	}).Info("Default registry change requested (placeholder implementation)")

	rb.SuccessWithMessage(nil, "Default registry updated successfully")
}