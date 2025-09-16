package controller

import (
	"net/http"
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
	imageService *service.ImageService
	logger       *logrus.Logger
}

// NewRegistryController creates a new registry controller
func NewRegistryController(imageService *service.ImageService, logger *logrus.Logger) *RegistryController {
	return &RegistryController{
		imageService: imageService,
		logger:       logger,
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
	enabledStr := c.Query("enabled")
	registryType := c.Query("type")

	rb := utils.NewResponseBuilder(c)

	// For now, return a mock list of registries
	// In a real implementation, you would query the registry repository
	registries := []RegistryResponse{
		{
			ID:          1,
			Name:        "Docker Hub",
			URL:         "https://registry-1.docker.io",
			Type:        "dockerhub",
			Description: "Official Docker registry",
			IsDefault:   true,
			Enabled:     true,
			Status:      "connected",
			CreatedAt:   "2023-01-01T00:00:00Z",
			UpdatedAt:   "2023-01-01T00:00:00Z",
		},
	}

	// Apply filters
	filteredRegistries := []RegistryResponse{}
	for _, reg := range registries {
		// Filter by enabled status
		if enabledStr != "" {
			enabled, _ := strconv.ParseBool(enabledStr)
			if reg.Enabled != enabled {
				continue
			}
		}

		// Filter by type
		if registryType != "" && reg.Type != registryType {
			continue
		}

		filteredRegistries = append(filteredRegistries, reg)
	}

	rc.logger.WithFields(logrus.Fields{
		"enabled": enabledStr,
		"type":    registryType,
		"count":   len(filteredRegistries),
	}).Info("Registries listed")

	rb.Success(filteredRegistries)
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

	// For now, this is a placeholder implementation
	// In a real implementation, you would:
	// 1. Check if registry with same name/URL already exists
	// 2. Save registry configuration to database
	// 3. Register the registry with the image service
	// 4. Return the created registry

	rc.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"name":    req.Name,
		"url":     req.URL,
		"type":    req.Type,
	}).Info("Registry creation requested (placeholder implementation)")

	// Mock response
	response := RegistryResponse{
		ID:          2, // Mock ID
		Name:        req.Name,
		URL:         req.URL,
		Type:        req.Type,
		Description: req.Description,
		IsDefault:   req.IsDefault,
		Enabled:     req.Enabled,
		Status:      "connected",
		Settings:    req.Settings,
		CreatedAt:   "2023-01-01T00:00:00Z",
		UpdatedAt:   "2023-01-01T00:00:00Z",
	}

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
	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, return mock data
	// In a real implementation, you would query the registry repository
	if registryID == 1 {
		response := RegistryResponse{
			ID:          1,
			Name:        "Docker Hub",
			URL:         "https://registry-1.docker.io",
			Type:        "dockerhub",
			Description: "Official Docker registry",
			IsDefault:   true,
			Enabled:     true,
			Status:      "connected",
			CreatedAt:   "2023-01-01T00:00:00Z",
			UpdatedAt:   "2023-01-01T00:00:00Z",
		}
		rb.Success(response)
		return
	}

	rc.logger.WithField("registry_id", registryID).Warn("Registry not found")
	rb.NotFound("Registry not found")
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

	// Measure connection test duration
	start := time.Now()
	err = rc.imageService.TestRegistryConnection(c.Request.Context(), registryURL, authConfig)
	duration := time.Since(start)

	result := RegistryTestResult{
		Success:  err == nil,
		Duration: duration.String(),
	}

	if err != nil {
		result.Message = "Connection failed"
		result.Error = err.Error()
		rc.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"duration":    duration,
		}).Warn("Registry connection test failed")
	} else {
		result.Message = "Connection successful"
		result.Capabilities = []string{"push", "pull", "search"} // Mock capabilities

		// Try to get registry info
		if info, err := rc.imageService.GetRegistryInfo(c.Request.Context(), registryURL); err == nil {
			result.Info = info
		}

		rc.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"duration":    duration,
		}).Info("Registry connection test successful")
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
	registryIDStr := c.Param("id")
	registryID, err := strconv.ParseInt(registryIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid registry ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// For now, return mock statistics
	// In a real implementation, you would:
	// 1. Query database for containers using this registry
	// 2. Get pull/push statistics
	// 3. Calculate usage metrics

	stats := map[string]interface{}{
		"registry_id":        registryID,
		"containers_using":   0, // Number of containers using this registry
		"images_tracked":     0, // Number of different images tracked
		"total_pulls":        0, // Total image pulls
		"total_pushes":       0, // Total image pushes (if supported)
		"last_pull":          nil, // Last pull timestamp
		"last_push":          nil, // Last push timestamp
		"popular_images":     []interface{}{}, // Most pulled images
		"recent_activity":    []interface{}{}, // Recent pull/push activity
		"storage_usage":      map[string]interface{}{ // Storage usage if available
			"total_size":     0,
			"image_count":    0,
			"layer_count":    0,
		},
		"error_rate":         0.0, // Error rate for registry operations
		"availability":       100.0, // Registry availability percentage
	}

	rc.logger.WithField("registry_id", registryID).Info("Registry statistics requested")
	rb.Success(stats)
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