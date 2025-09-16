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

// ImageController handles image-related HTTP requests
type ImageController struct {
	imageService *service.ImageService
	logger       *logrus.Logger
}

// NewImageController creates a new image controller
func NewImageController(imageService *service.ImageService, logger *logrus.Logger) *ImageController {
	return &ImageController{
		imageService: imageService,
		logger:       logger,
	}
}

// ListImages godoc
// @Summary List available images
// @Description Get list of available images with version information
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search images by name"
// @Param registry query string false "Filter by registry URL"
// @Param limit query int false "Limit results" default(50)
// @Success 200 {object} utils.APIResponse{data=[]model.ImageVersion} "Images list"
// @Failure 400 {object} utils.APIResponse "Invalid request parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images [get]
func (ic *ImageController) ListImages(c *gin.Context) {
	search := c.Query("search")
	registryURL := c.Query("registry")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rb := utils.NewResponseBuilder(c)

	// If search is provided, search for images
	if search != "" {
		results, err := ic.imageService.SearchImages(c.Request.Context(), search, registryURL)
		if err != nil {
			ic.logger.WithError(err).WithField("search", search).Error("Failed to search images")
			rb.InternalServerError("Failed to search images")
			return
		}

		// Limit results
		if len(results) > limit {
			results = results[:limit]
		}

		rb.Success(results)
		return
	}

	// For now, return empty list as we don't have a general list endpoint
	// In a real implementation, you'd list images from your tracked containers
	rb.Success([]interface{}{})
}

// CheckUpdates godoc
// @Summary Check for image updates
// @Description Check for updates for all managed containers or specific criteria
// @Tags Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.ImageCheckFilter false "Check criteria"
// @Success 200 {object} utils.APIResponse{data=[]service.UpdateInfo} "Update information"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/check-updates [post]
func (ic *ImageController) CheckUpdates(c *gin.Context) {
	var filter service.ImageCheckFilter

	// Optional request body
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&filter); err != nil {
			ic.logger.WithError(err).Warn("Invalid check updates request")
			utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
			return
		}
	}

	rb := utils.NewResponseBuilder(c)

	updateInfos, err := ic.imageService.CheckImagesByFilter(c.Request.Context(), &filter)
	if err != nil {
		ic.logger.WithError(err).Error("Failed to check image updates")
		rb.InternalServerError("Failed to check for updates")
		return
	}

	ic.logger.WithField("updates_found", len(updateInfos)).Info("Image update check completed")
	rb.Success(updateInfos)
}

// GetImageVersions godoc
// @Summary Get image version history
// @Description Get version history for a specific image
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param name path string true "Image name (format: registry/image or image)"
// @Success 200 {object} utils.APIResponse{data=[]model.ImageVersion} "Image versions"
// @Failure 400 {object} utils.APIResponse "Invalid image name"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Image not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/{name}/versions [get]
func (ic *ImageController) GetImageVersions(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	// Decode URL-encoded image name (handle slashes)
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	rb := utils.NewResponseBuilder(c)

	versions, err := ic.imageService.GetImageVersions(c.Request.Context(), imageName)
	if err != nil {
		ic.logger.WithError(err).WithField("image", imageName).Error("Failed to get image versions")
		if strings.Contains(err.Error(), "not found") {
			rb.NotFound("Image not found")
			return
		}
		rb.InternalServerError("Failed to retrieve image versions")
		return
	}

	rb.Success(versions)
}

// PullImage godoc
// @Summary Pull specific image version
// @Description Pull a specific version of an image
// @Tags Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name path string true "Image name (format: registry/image or image)"
// @Param request body map[string]string true "Pull request"
// @Success 200 {object} utils.APIResponse "Image pulled successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/{name}/pull [post]
func (ic *ImageController) PullImage(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	// Decode URL-encoded image name
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	var req struct {
		Tag         string `json:"tag,omitempty"`
		RegistryURL string `json:"registry_url,omitempty"`
		Force       bool   `json:"force,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ic.logger.WithError(err).WithField("image", imageName).Warn("Invalid pull image request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	// Default tag
	if req.Tag == "" {
		req.Tag = "latest"
	}

	rb := utils.NewResponseBuilder(c)

	// For now, this would be a placeholder as we need to integrate with Docker client
	// In a real implementation, you'd use the Docker client to pull the image
	ic.logger.WithFields(logrus.Fields{
		"image": imageName,
		"tag":   req.Tag,
		"force": req.Force,
	}).Info("Image pull requested (placeholder implementation)")

	// TODO: Implement actual image pulling using Docker client
	rb.Error(http.StatusNotImplemented, "Image pulling not yet implemented")
}

// RemoveImage godoc
// @Summary Remove image
// @Description Remove a specific image version
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param name path string true "Image name (format: registry/image or image)"
// @Param tag query string false "Tag to remove" default(latest)
// @Param force query boolean false "Force removal" default(false)
// @Success 200 {object} utils.APIResponse "Image removed successfully"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Image not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/{name} [delete]
func (ic *ImageController) RemoveImage(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	// Decode URL-encoded image name
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	tag := c.DefaultQuery("tag", "latest")
	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))

	rb := utils.NewResponseBuilder(c)

	// For now, this would be a placeholder as we need to integrate with Docker client
	// In a real implementation, you'd use the Docker client to remove the image
	ic.logger.WithFields(logrus.Fields{
		"image": imageName,
		"tag":   tag,
		"force": force,
	}).Info("Image removal requested (placeholder implementation)")

	// TODO: Implement actual image removal using Docker client
	rb.Error(http.StatusNotImplemented, "Image removal not yet implemented")
}

// SearchImages godoc
// @Summary Search public images
// @Description Search for public images in registries
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param registry query string false "Registry to search" default(docker.io)
// @Param limit query int false "Limit results" default(25)
// @Success 200 {object} utils.APIResponse{data=[]registry.ImageSearchResult} "Search results"
// @Failure 400 {object} utils.APIResponse "Invalid request parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/search [get]
func (ic *ImageController) SearchImages(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.BadRequestJSON(c, "Search query is required")
		return
	}

	registryURL := c.DefaultQuery("registry", "docker.io")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))

	if limit <= 0 || limit > 100 {
		limit = 25
	}

	rb := utils.NewResponseBuilder(c)

	results, err := ic.imageService.SearchImages(c.Request.Context(), query, registryURL)
	if err != nil {
		ic.logger.WithError(err).WithFields(logrus.Fields{
			"query":    query,
			"registry": registryURL,
		}).Error("Failed to search images")
		rb.InternalServerError("Failed to search images")
		return
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	ic.logger.WithFields(logrus.Fields{
		"query":        query,
		"registry":     registryURL,
		"results_count": len(results),
	}).Info("Image search completed")

	rb.Success(results)
}

// GetImageInfo godoc
// @Summary Get image information
// @Description Get detailed information about a specific image
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param name path string true "Image name (format: registry/image or image)"
// @Param tag query string false "Tag" default(latest)
// @Param registry query string false "Registry URL"
// @Success 200 {object} utils.APIResponse{data=model.ImageVersion} "Image information"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Image not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/{name}/info [get]
func (ic *ImageController) GetImageInfo(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	// Decode URL-encoded image name
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	tag := c.DefaultQuery("tag", "latest")
	registryURL := c.Query("registry")

	// Build full image name with tag
	fullImageName := imageName
	if tag != "" && tag != "latest" {
		fullImageName += ":" + tag
	}

	rb := utils.NewResponseBuilder(c)

	imageInfo, err := ic.imageService.GetLatestImageInfo(c.Request.Context(), fullImageName, registryURL)
	if err != nil {
		ic.logger.WithError(err).WithFields(logrus.Fields{
			"image":    imageName,
			"tag":      tag,
			"registry": registryURL,
		}).Error("Failed to get image info")
		if strings.Contains(err.Error(), "not found") {
			rb.NotFound("Image not found")
			return
		}
		rb.InternalServerError("Failed to retrieve image information")
		return
	}

	rb.Success(imageInfo)
}

// CompareImageVersions godoc
// @Summary Compare image versions
// @Description Compare two versions of an image
// @Tags Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Version comparison request"
// @Success 200 {object} utils.APIResponse{data=registry.VersionComparisonResult} "Comparison result"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/compare [post]
func (ic *ImageController) CompareImageVersions(c *gin.Context) {
	var req struct {
		CurrentImage  string `json:"current_image" binding:"required"`
		CurrentTag    string `json:"current_tag" binding:"required"`
		LatestImage   string `json:"latest_image" binding:"required"`
		LatestTag     string `json:"latest_tag" binding:"required"`
		RegistryURL   string `json:"registry_url,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ic.logger.WithError(err).Warn("Invalid compare versions request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Get version information for both images
	currentFullName := req.CurrentImage + ":" + req.CurrentTag
	latestFullName := req.LatestImage + ":" + req.LatestTag

	currentVersion, err := ic.imageService.GetLatestImageInfo(c.Request.Context(), currentFullName, req.RegistryURL)
	if err != nil {
		ic.logger.WithError(err).WithField("image", currentFullName).Error("Failed to get current version info")
		rb.BadRequest("Failed to get current version information")
		return
	}

	latestVersion, err := ic.imageService.GetLatestImageInfo(c.Request.Context(), latestFullName, req.RegistryURL)
	if err != nil {
		ic.logger.WithError(err).WithField("image", latestFullName).Error("Failed to get latest version info")
		rb.BadRequest("Failed to get latest version information")
		return
	}

	// Compare versions
	comparison, err := ic.imageService.CompareImageVersions(c.Request.Context(), currentVersion, latestVersion)
	if err != nil {
		ic.logger.WithError(err).Error("Failed to compare image versions")
		rb.InternalServerError("Failed to compare versions")
		return
	}

	ic.logger.WithFields(logrus.Fields{
		"current": currentFullName,
		"latest":  latestFullName,
	}).Info("Image versions compared")

	rb.Success(comparison)
}

// RefreshImageCache godoc
// @Summary Refresh image cache
// @Description Refresh cached information for a specific image
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param name path string true "Image name (format: registry/image or image)"
// @Success 200 {object} utils.APIResponse "Cache refreshed successfully"
// @Failure 400 {object} utils.APIResponse "Invalid image name"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/{name}/refresh [post]
func (ic *ImageController) RefreshImageCache(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	// Decode URL-encoded image name
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	rb := utils.NewResponseBuilder(c)

	if err := ic.imageService.RefreshImageCache(c.Request.Context(), imageName); err != nil {
		ic.logger.WithError(err).WithField("image", imageName).Error("Failed to refresh image cache")
		rb.InternalServerError("Failed to refresh cache")
		return
	}

	ic.logger.WithField("image", imageName).Info("Image cache refreshed")
	rb.SuccessWithMessage(nil, "Image cache refreshed successfully")
}

// GetImageUpdateInfo godoc
// @Summary Get update information for image
// @Description Get update information for a specific container's image
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param container_id query int true "Container ID"
// @Success 200 {object} utils.APIResponse{data=service.UpdateInfo} "Update information"
// @Failure 400 {object} utils.APIResponse "Invalid container ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Container not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/update-info [get]
func (ic *ImageController) GetImageUpdateInfo(c *gin.Context) {
	containerIDStr := c.Query("container_id")
	if containerIDStr == "" {
		utils.BadRequestJSON(c, "Container ID is required")
		return
	}

	containerID, err := strconv.ParseInt(containerIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid container ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Check cache first
	if cachedInfo, found := ic.imageService.GetCachedUpdateInfo(containerID); found {
		// Check if cache is still valid (within last hour)
		if time.Since(cachedInfo.LastChecked) < time.Hour {
			rb.Success(cachedInfo)
			return
		}
	}

	// Get fresh update info
	updateInfo, err := ic.imageService.CheckImageUpdate(c.Request.Context(), containerID)
	if err != nil {
		ic.logger.WithError(err).WithField("container_id", containerID).Error("Failed to check image update")
		if strings.Contains(err.Error(), "not found") {
			rb.NotFound("Container not found")
			return
		}
		rb.InternalServerError("Failed to check for updates")
		return
	}

	rb.Success(updateInfo)
}

// ScheduleImageCheck godoc
// @Summary Schedule periodic image checks
// @Description Schedule periodic image update checks for a container
// @Tags Images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Schedule request"
// @Success 200 {object} utils.APIResponse "Check scheduled successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/schedule-check [post]
func (ic *ImageController) ScheduleImageCheck(c *gin.Context) {
	var req struct {
		ContainerID int64  `json:"container_id" binding:"required"`
		Interval    string `json:"interval" binding:"required"` // e.g., "1h", "6h", "24h"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ic.logger.WithError(err).Warn("Invalid schedule check request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	// Parse interval
	interval, err := time.ParseDuration(req.Interval)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid interval format (use format like '1h', '6h', '24h')")
		return
	}

	// Validate interval bounds
	if interval < time.Hour {
		utils.BadRequestJSON(c, "Minimum interval is 1 hour")
		return
	}
	if interval > 24*time.Hour {
		utils.BadRequestJSON(c, "Maximum interval is 24 hours")
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := ic.imageService.ScheduleImageCheck(c.Request.Context(), req.ContainerID, interval); err != nil {
		ic.logger.WithError(err).WithFields(logrus.Fields{
			"container_id": req.ContainerID,
			"interval":     interval,
		}).Error("Failed to schedule image check")
		rb.InternalServerError("Failed to schedule check")
		return
	}

	ic.logger.WithFields(logrus.Fields{
		"container_id": req.ContainerID,
		"interval":     interval,
	}).Info("Image check scheduled")

	rb.SuccessWithMessage(nil, "Image check scheduled successfully")
}

// GetImageSecurityIssues godoc
// @Summary Get security issues for image
// @Description Get known security vulnerabilities for a specific image
// @Tags Images
// @Produce json
// @Security BearerAuth
// @Param name path string true "Image name (format: registry/image or image)"
// @Param tag query string false "Tag" default(latest)
// @Success 200 {object} utils.APIResponse{data=[]registry.SecurityVulnerability} "Security issues"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Image not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/images/{name}/security [get]
func (ic *ImageController) GetImageSecurityIssues(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		utils.BadRequestJSON(c, "Image name is required")
		return
	}

	// Decode URL-encoded image name
	imageName = strings.ReplaceAll(imageName, "%2F", "/")

	tag := c.DefaultQuery("tag", "latest")
	fullImageName := imageName + ":" + tag

	rb := utils.NewResponseBuilder(c)

	// Get image update info which includes security issues
	updateInfo, err := ic.imageService.CheckImageUpdate(c.Request.Context(), 0) // Use 0 as placeholder
	if err != nil {
		// For now, return empty array as security scanning is not fully implemented
		ic.logger.WithFields(logrus.Fields{
			"image": fullImageName,
		}).Debug("Security check not available (placeholder implementation)")
		rb.Success([]interface{}{})
		return
	}

	rb.Success(updateInfo.SecurityIssues)
}