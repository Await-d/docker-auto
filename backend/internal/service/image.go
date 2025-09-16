package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/registry"

	"github.com/sirupsen/logrus"
)

// ImageService manages image checking and version management
type ImageService struct {
	imageRepo       repository.ImageVersionRepository
	containerRepo   repository.ContainerRepository
	activityRepo    repository.ActivityLogRepository
	updateRepo      repository.UpdateHistoryRepository
	imageChecker    registry.ImageChecker
	cache           *CacheService
	config          *config.Config
	scheduledChecks map[int64]*scheduledCheck
	checksMutex     sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// scheduledCheck represents a scheduled image check
type scheduledCheck struct {
	containerID int64
	interval    time.Duration
	ticker      *time.Ticker
	lastCheck   time.Time
}

// ImageCheckFilter represents filters for image checking
type ImageCheckFilter struct {
	UpdatePolicy *model.UpdatePolicy `json:"update_policy,omitempty"`
	Images       []string            `json:"images,omitempty"`
	RegistryURL  string              `json:"registry_url,omitempty"`
	OnlyOutdated bool                `json:"only_outdated,omitempty"`
	MaxAge       *time.Duration      `json:"max_age,omitempty"`
}

// ImageUpdateInfo represents information about available updates (specific to image service)
type ImageUpdateInfo struct {
	ContainerID     int64                                `json:"container_id"`
	Name            string                               `json:"name"`
	CurrentImage    string                               `json:"current_image"`
	CurrentTag      string                               `json:"current_tag"`
	CurrentDigest   string                               `json:"current_digest,omitempty"`
	LatestImage     string                               `json:"latest_image,omitempty"`
	LatestTag       string                               `json:"latest_tag,omitempty"`
	LatestDigest    string                               `json:"latest_digest,omitempty"`
	UpdateAvailable bool                                 `json:"update_available"`
	UpdateType      string                               `json:"update_type,omitempty"` // major, minor, patch, unknown
	LastChecked     time.Time                            `json:"last_checked"`
	VersionInfo     *registry.VersionComparisonResult    `json:"version_info,omitempty"`
	SecurityIssues  []registry.SecurityVulnerability     `json:"security_issues,omitempty"`
	Recommendation  string                               `json:"recommendation,omitempty"`
}

// NewImageService creates a new image service instance
func NewImageService(
	imageRepo repository.ImageVersionRepository,
	containerRepo repository.ContainerRepository,
	activityRepo repository.ActivityLogRepository,
	updateRepo repository.UpdateHistoryRepository,
	cache *CacheService,
	config *config.Config,
) *ImageService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &ImageService{
		imageRepo:       imageRepo,
		containerRepo:   containerRepo,
		activityRepo:    activityRepo,
		updateRepo:      updateRepo,
		cache:           cache,
		config:          config,
		scheduledChecks: make(map[int64]*scheduledCheck),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize image checker
	service.initializeImageChecker()

	return service
}

// Start starts the image service
func (s *ImageService) Start(ctx context.Context) error {
	logrus.Info("Image service started")
	return nil
}

// Stop stops the image service
func (s *ImageService) Stop() error {
	s.cancel()

	// Stop all scheduled checks
	s.checksMutex.Lock()
	for _, check := range s.scheduledChecks {
		if check.ticker != nil {
			check.ticker.Stop()
		}
	}
	s.scheduledChecks = make(map[int64]*scheduledCheck)
	s.checksMutex.Unlock()

	logrus.Info("Image service stopped")
	return nil
}

// Image checking methods

// CheckImageUpdate checks for updates for a specific container
func (s *ImageService) CheckImageUpdate(ctx context.Context, containerID int64) (*ImageUpdateInfo, error) {
	// Get container information
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	// Get current digest from cache or database
	currentDigest, err := s.getCurrentImageDigest(ctx, container)
	if err != nil {
		logrus.WithError(err).WithField("container_id", containerID).Debug("Failed to get current digest, proceeding without it")
		currentDigest = ""
	}

	// Check for update using image checker
	fullImageName := container.GetFullImageName()
	updateResult, err := s.imageChecker.CheckImageUpdate(ctx, fullImageName, currentDigest, container.RegistryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check image update: %w", err)
	}

	// Convert to ImageUpdateInfo
	updateInfo := &ImageUpdateInfo{
		ContainerID:     containerID,
		Name:            container.Name,
		CurrentImage:    container.Image,
		CurrentTag:      container.Tag,
		CurrentDigest:   currentDigest,
		UpdateAvailable: updateResult.UpdateAvailable,
		UpdateType:      updateResult.UpdateType,
		LastChecked:     updateResult.LastChecked,
		SecurityIssues:  updateResult.SecurityIssues,
	}

	if updateResult.UpdateAvailable {
		updateInfo.LatestTag = updateResult.LatestTag
		updateInfo.LatestDigest = updateResult.LatestDigest
		updateInfo.LatestImage = container.Image // Same image, different tag

		// Get version comparison if available
		if currentImageVersion, err := s.getCurrentImageVersion(ctx, container); err == nil {
			if latestImageVersion, err := s.getLatestImageVersion(ctx, container); err == nil {
				if versionComparison, err := s.imageChecker.CompareVersions(currentImageVersion, latestImageVersion); err == nil {
					updateInfo.VersionInfo = versionComparison
					updateInfo.Recommendation = s.generateUpdateRecommendation(container, versionComparison)
				}
			}
		}
	}

	// Cache the result
	s.cacheUpdateInfo(containerID, updateInfo)

	// Log the check
	s.logImageActivity(containerID, "image_check", "Image update check performed", map[string]interface{}{
		"update_available": updateInfo.UpdateAvailable,
		"update_type":      updateInfo.UpdateType,
		"current_tag":      updateInfo.CurrentTag,
		"latest_tag":       updateInfo.LatestTag,
	})

	return updateInfo, nil
}

// CheckAllImages checks for updates for all containers
func (s *ImageService) CheckAllImages(ctx context.Context) ([]*ImageUpdateInfo, error) {
	// Get all containers
	containers, _, err := s.containerRepo.List(ctx, &model.ContainerFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	// Check updates for all containers using image checker
	updateResults, err := s.imageChecker.CheckAllImages(ctx, containers)
	if err != nil {
		return nil, fmt.Errorf("failed to check all images: %w", err)
	}

	// Convert results to ImageUpdateInfo
	updateInfos := make([]*ImageUpdateInfo, len(containers))
	for i, container := range containers {
		if i < len(updateResults) && updateResults[i] != nil {
			updateInfo := s.convertUpdateResultToInfo(container, updateResults[i])
			updateInfos[i] = updateInfo

			// Cache the result
			s.cacheUpdateInfo(int64(container.ID), updateInfo)
		} else {
			// Create empty result for failed checks
			updateInfos[i] = &ImageUpdateInfo{
				ContainerID:     int64(container.ID),
				Name:            container.Name,
				CurrentImage:    container.Image,
				CurrentTag:      container.Tag,
				UpdateAvailable: false,
				LastChecked:     time.Now(),
			}
		}
	}

	// Log bulk check
	successCount := 0
	for _, info := range updateInfos {
		if info.LastChecked.After(time.Now().Add(-5*time.Minute)) {
			successCount++
		}
	}

	s.logSystemActivity("bulk_image_check", fmt.Sprintf("Bulk image check completed: %d/%d successful", successCount, len(containers)), map[string]interface{}{
		"total_containers": len(containers),
		"success_count":    successCount,
	})

	return updateInfos, nil
}

// CheckImagesByFilter checks images based on filter criteria
func (s *ImageService) CheckImagesByFilter(ctx context.Context, filter *ImageCheckFilter) ([]*ImageUpdateInfo, error) {
	if filter == nil {
		return s.CheckAllImages(ctx)
	}

	// Build container filter
	containerFilter := &model.ContainerFilter{}
	if filter.UpdatePolicy != nil {
		containerFilter.UpdatePolicy = *filter.UpdatePolicy
	}

	// Get filtered containers
	containers, _, err := s.containerRepo.List(ctx, containerFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered containers: %w", err)
	}

	// Filter by specific images if provided
	if len(filter.Images) > 0 {
		filteredContainers := []*model.Container{}
		imageSet := make(map[string]bool)
		for _, image := range filter.Images {
			imageSet[image] = true
		}

		for _, container := range containers {
			if imageSet[container.GetFullImageName()] {
				filteredContainers = append(filteredContainers, container)
			}
		}
		containers = filteredContainers
	}

	// Filter by registry URL if provided
	if filter.RegistryURL != "" {
		filteredContainers := []*model.Container{}
		for _, container := range containers {
			if container.RegistryURL == filter.RegistryURL {
				filteredContainers = append(filteredContainers, container)
			}
		}
		containers = filteredContainers
	}

	// Check updates for filtered containers
	if len(containers) == 0 {
		return []*ImageUpdateInfo{}, nil
	}

	updateResults, err := s.imageChecker.CheckAllImages(ctx, containers)
	if err != nil {
		return nil, fmt.Errorf("failed to check filtered images: %w", err)
	}

	// Convert and filter results
	updateInfos := make([]*ImageUpdateInfo, 0, len(containers))
	for i, container := range containers {
		if i < len(updateResults) && updateResults[i] != nil {
			updateInfo := s.convertUpdateResultToInfo(container, updateResults[i])

			// Apply additional filters
			if filter.OnlyOutdated && !updateInfo.UpdateAvailable {
				continue
			}

			if filter.MaxAge != nil && time.Since(updateInfo.LastChecked) > *filter.MaxAge {
				continue
			}

			updateInfos = append(updateInfos, updateInfo)
			s.cacheUpdateInfo(int64(container.ID), updateInfo)
		}
	}

	return updateInfos, nil
}

// ScheduleImageCheck schedules regular image checks for a container
func (s *ImageService) ScheduleImageCheck(ctx context.Context, containerID int64, interval time.Duration) error {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()

	// Stop existing scheduled check if any
	if existingCheck, exists := s.scheduledChecks[containerID]; exists {
		if existingCheck.ticker != nil {
			existingCheck.ticker.Stop()
		}
	}

	// Create new scheduled check
	check := &scheduledCheck{
		containerID: containerID,
		interval:    interval,
		ticker:      time.NewTicker(interval),
		lastCheck:   time.Now(),
	}

	s.scheduledChecks[containerID] = check

	// Start the check routine
	go s.runScheduledCheck(check)

	logrus.WithFields(logrus.Fields{
		"container_id": containerID,
		"interval":     interval,
	}).Info("Scheduled image check created")

	return nil
}

// Image version management

// GetImageVersions gets all cached versions for an image
func (s *ImageService) GetImageVersions(ctx context.Context, image string) ([]*model.ImageVersion, error) {
	versions, err := s.imageRepo.GetByImageName(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image versions: %w", err)
	}

	return versions, nil
}

// GetLatestImageInfo gets the latest version information for an image
func (s *ImageService) GetLatestImageInfo(ctx context.Context, image string, registryURL string) (*model.ImageVersion, error) {
	// Try cache first
	if cachedInfo, found := s.imageChecker.GetCachedImageInfo(image); found {
		return cachedInfo, nil
	}

	// Get client for the registry
	client, err := s.imageChecker.GetClient(registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry client: %w", err)
	}

	// Get latest image info
	latestInfo, err := client.GetLatestImageInfo(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest image info: %w", err)
	}

	// Cache the result
	s.imageChecker.CacheImageInfo(image, latestInfo, 6*time.Hour)

	// Save to database
	if err := s.imageRepo.UpsertVersion(ctx, latestInfo); err != nil {
		logrus.WithError(err).WithField("image", image).Warn("Failed to save image version to database")
	}

	return latestInfo, nil
}

// RefreshImageCache refreshes cached image information
func (s *ImageService) RefreshImageCache(ctx context.Context, image string) error {
	// Invalidate current cache
	if err := s.imageChecker.InvalidateCache(image); err != nil {
		logrus.WithError(err).WithField("image", image).Warn("Failed to invalidate image cache")
	}

	// Get fresh image information
	registry, _, _, _ := registry.ParseImageRef(image)
	_, err := s.GetLatestImageInfo(ctx, image, registry)
	if err != nil {
		return fmt.Errorf("failed to refresh image cache: %w", err)
	}

	logrus.WithField("image", image).Info("Image cache refreshed")
	return nil
}

// CompareImageVersions compares two image versions
func (s *ImageService) CompareImageVersions(ctx context.Context, current, latest *model.ImageVersion) (*registry.VersionComparisonResult, error) {
	if current == nil || latest == nil {
		return nil, fmt.Errorf("both current and latest versions must be provided")
	}

	return s.imageChecker.CompareVersions(current, latest)
}

// Registry management

// TestRegistryConnection tests connection to a registry
func (s *ImageService) TestRegistryConnection(ctx context.Context, registryURL string, auth *registry.AuthConfig) error {
	// Get or create client for the registry
	client, err := s.imageChecker.GetClient(registryURL)
	if err != nil {
		// If no client is available, try to create one based on registry type
		registryType := s.detectRegistryType(registryURL)
		switch registryType {
		case "dockerhub":
			client = registry.NewDockerHubClient(auth)
		case "harbor":
			client = registry.NewHarborClient(registryURL, auth)
		default:
			return fmt.Errorf("unsupported registry type for URL: %s", registryURL)
		}
	}

	// Test connection
	return client.TestConnection(ctx)
}

// GetRegistryInfo gets information about a registry
func (s *ImageService) GetRegistryInfo(ctx context.Context, registryURL string) (*registry.RegistryInfo, error) {
	client, err := s.imageChecker.GetClient(registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry client: %w", err)
	}

	return client.GetRegistryInfo(ctx)
}

// SearchImages searches for images in a registry
func (s *ImageService) SearchImages(ctx context.Context, query string, registryURL string) ([]*registry.ImageSearchResult, error) {
	client, err := s.imageChecker.GetClient(registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry client: %w", err)
	}

	searchOptions := &registry.SearchOptions{
		Query: query,
		Limit: 50,
	}

	repositories, err := client.SearchRepositories(ctx, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to search repositories: %w", err)
	}

	// Convert to ImageSearchResult
	results := make([]*registry.ImageSearchResult, len(repositories))
	for i, repo := range repositories {
		results[i] = &registry.ImageSearchResult{
			Name:         repo.Name,
			Description:  repo.Description,
			Stars:        repo.Stars,
			IsOfficial:   repo.IsOfficial,
			IsAutomated:  repo.IsAutomated,
			RegistryType: s.detectRegistryType(registryURL),
		}
	}

	return results, nil
}

// Helper methods

// initializeImageChecker initializes the image checker with registry clients
func (s *ImageService) initializeImageChecker() {
	s.imageChecker = registry.NewImageChecker()

	// Register Docker Hub client
	dockerhubClient := registry.NewDockerHubClient(nil)
	s.imageChecker.RegisterClient("dockerhub", dockerhubClient)

	// Register Harbor clients if configured
	if s.config.Registry.Harbor.Enabled {
		harborAuth := &registry.AuthConfig{
			Username: s.config.Registry.Harbor.Username,
			Password: s.config.Registry.Harbor.Password,
			AuthType: "basic",
		}
		harborClient := registry.NewHarborClient(s.config.Registry.Harbor.URL, harborAuth)
		s.imageChecker.RegisterClient("harbor", harborClient)
	}

	// Set default registry
	s.imageChecker.SetDefaultRegistry("docker.io")

	// Configure cache
	cacheConfig := &registry.CacheConfig{
		TTL:             time.Duration(s.config.Cache.ImageCacheTTLHours) * time.Hour,
		MaxEntries:      1000,
		CleanupInterval: 30 * time.Minute,
	}
	s.imageChecker.SetCacheConfig(cacheConfig)
}

// runScheduledCheck runs a scheduled image check
func (s *ImageService) runScheduledCheck(check *scheduledCheck) {
	defer func() {
		if check.ticker != nil {
			check.ticker.Stop()
		}
	}()

	for {
		select {
		case <-check.ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
			updateInfo, err := s.CheckImageUpdate(ctx, check.containerID)
			cancel()

			if err != nil {
				logrus.WithError(err).WithField("container_id", check.containerID).Warn("Scheduled image check failed")
			} else {
				check.lastCheck = time.Now()
				logrus.WithFields(logrus.Fields{
					"container_id":     check.containerID,
					"update_available": updateInfo.UpdateAvailable,
				}).Debug("Scheduled image check completed")
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// getCurrentImageDigest gets the current image digest for a container
func (s *ImageService) getCurrentImageDigest(ctx context.Context, container *model.Container) (string, error) {
	// Try to get from cached image version
	fullImageName := container.GetFullImageName()
	if cachedVersion, found := s.imageChecker.GetCachedImageInfo(fullImageName); found {
		return cachedVersion.Digest, nil
	}

	// Try to get from database
	imageVersion, err := s.imageRepo.GetByImageAndTag(ctx, container.Image, container.Tag)
	if err == nil && imageVersion != nil {
		return imageVersion.Digest, nil
	}

	// No current digest available
	return "", fmt.Errorf("no current digest available")
}

// getCurrentImageVersion gets the current image version for a container
func (s *ImageService) getCurrentImageVersion(ctx context.Context, container *model.Container) (*model.ImageVersion, error) {
	return s.imageRepo.GetByImageAndTag(ctx, container.Image, container.Tag)
}

// getLatestImageVersion gets the latest image version for a container
func (s *ImageService) getLatestImageVersion(ctx context.Context, container *model.Container) (*model.ImageVersion, error) {
	fullImageName := container.GetFullImageName()
	return s.GetLatestImageInfo(ctx, fullImageName, container.RegistryURL)
}

// convertUpdateResultToInfo converts registry.UpdateCheckResult to ImageUpdateInfo
func (s *ImageService) convertUpdateResultToInfo(container *model.Container, result *registry.UpdateCheckResult) *ImageUpdateInfo {
	updateInfo := &ImageUpdateInfo{
		ContainerID:     int64(container.ID),
		Name:            container.Name,
		CurrentImage:    container.Image,
		CurrentTag:      container.Tag,
		CurrentDigest:   result.CurrentDigest,
		UpdateAvailable: result.UpdateAvailable,
		UpdateType:      result.UpdateType,
		LastChecked:     result.LastChecked,
		SecurityIssues:  result.SecurityIssues,
	}

	if result.UpdateAvailable {
		updateInfo.LatestTag = result.LatestTag
		updateInfo.LatestDigest = result.LatestDigest
		updateInfo.LatestImage = container.Image
	}

	return updateInfo
}

// generateUpdateRecommendation generates update recommendation
func (s *ImageService) generateUpdateRecommendation(container *model.Container, comparison *registry.VersionComparisonResult) string {
	// Check container's update policy
	switch container.UpdatePolicy {
	case model.UpdatePolicyAuto:
		if s.imageChecker.ShouldUpdate(comparison, "auto") {
			strategy := s.imageChecker.GetUpdateStrategy(comparison)
			return fmt.Sprintf("Auto-update recommended using %s strategy", strategy)
		}
		return "Auto-update policy active but update not recommended at this time"

	case model.UpdatePolicyManual:
		return "Manual review required - check change details before updating"

	case model.UpdatePolicyDisabled:
		return "Updates disabled for this container"

	case model.UpdatePolicyScheduled:
		return "Update will be applied during next scheduled maintenance window"

	default:
		return "Review update details and apply manually if needed"
	}
}

// detectRegistryType detects registry type from URL
func (s *ImageService) detectRegistryType(registryURL string) string {
	if registryURL == "" || registryURL == "docker.io" {
		return "dockerhub"
	}

	// Add logic to detect other registry types
	// This is a simplified version
	return "generic"
}

// cacheUpdateInfo caches update information
func (s *ImageService) cacheUpdateInfo(containerID int64, updateInfo *ImageUpdateInfo) {
	if s.cache == nil {
		return
	}

	cacheKey := fmt.Sprintf("update_info:%d", containerID)
	if err := s.cache.Set(cacheKey, updateInfo, 30*time.Minute); err != nil {
		logrus.WithError(err).WithField("container_id", containerID).Debug("Failed to cache update info")
	}
}

// getCachedUpdateInfo gets cached update information
func (s *ImageService) GetCachedUpdateInfo(containerID int64) (*ImageUpdateInfo, bool) {
	if s.cache == nil {
		return nil, false
	}

	cacheKey := fmt.Sprintf("update_info:%d", containerID)
	if value, found := s.cache.Get(cacheKey); found {
		if updateInfo, ok := value.(*ImageUpdateInfo); ok {
			return updateInfo, true
		}
	}

	return nil, false
}

// Activity logging

// logImageActivity logs image-related activities
func (s *ImageService) logImageActivity(containerID int64, action, description string, metadata map[string]interface{}) {
	if s.activityRepo == nil {
		return
	}

	metadataJSON := "{}"
	if metadata != nil {
		if jsonBytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	activity := &model.ActivityLog{
		UserID:       0, // System activity
		Action:       action,
		ResourceType: "container",
		ResourceID:   &containerID,
		Description:  description,
		Metadata:     metadataJSON,
	}

	if err := s.activityRepo.Create(context.Background(), activity); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"container_id": containerID,
			"action":       action,
		}).Warn("Failed to log image activity")
	}
}

// logSystemActivity logs system-level activities
func (s *ImageService) logSystemActivity(action, description string, metadata map[string]interface{}) {
	if s.activityRepo == nil {
		return
	}

	metadataJSON := "{}"
	if metadata != nil {
		if jsonBytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	activity := &model.ActivityLog{
		UserID:      0, // System activity
		Action:      action,
		Description: description,
		Metadata:    metadataJSON,
	}

	if err := s.activityRepo.Create(context.Background(), activity); err != nil {
		logrus.WithError(err).WithField("action", action).Warn("Failed to log system activity")
	}
}