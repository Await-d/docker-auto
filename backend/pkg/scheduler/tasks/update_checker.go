package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/registry"
	"docker-auto/pkg/scheduler"

	"github.com/sirupsen/logrus"
)

// UpdateCheckerTask implements the Task interface for checking image updates
type UpdateCheckerTask struct {
	containerRepo repository.ContainerRepository
	imageRepo     repository.ImageVersionRepository
	registryChecker *registry.Checker
	containerService *service.ContainerService
	imageService     *service.ImageService
	notificationService *service.NotificationService
}

// NewUpdateCheckerTask creates a new update checker task
func NewUpdateCheckerTask(
	containerRepo repository.ContainerRepository,
	imageRepo repository.ImageVersionRepository,
	registryChecker *registry.Checker,
	containerService *service.ContainerService,
	imageService *service.ImageService,
	notificationService *service.NotificationService,
) *UpdateCheckerTask {
	return &UpdateCheckerTask{
		containerRepo:       containerRepo,
		imageRepo:          imageRepo,
		registryChecker:    registryChecker,
		containerService:   containerService,
		imageService:       imageService,
		notificationService: notificationService,
	}
}

// Execute runs the image update checking task
func (t *UpdateCheckerTask) Execute(ctx context.Context, params scheduler.TaskParameters) error {
	logger := logrus.WithFields(logrus.Fields{
		"task_type": t.GetType(),
		"task_name": t.GetName(),
	})

	logger.Info("Starting image update check task")

	// Parse task-specific parameters
	checkParams, err := t.parseParameters(params)
	if err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Get containers to check
	containers, err := t.getContainersToCheck(ctx, params.TargetContainers)
	if err != nil {
		return fmt.Errorf("failed to get containers to check: %w", err)
	}

	logger.WithField("container_count", len(containers)).Info("Found containers to check for updates")

	// Check for updates
	results, err := t.checkForUpdates(ctx, containers, checkParams)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Process results
	if err := t.processResults(ctx, results, checkParams); err != nil {
		return fmt.Errorf("failed to process results: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"containers_checked": len(containers),
		"updates_found":      results.UpdatesFound,
		"errors":            len(results.Errors),
	}).Info("Image update check task completed")

	return nil
}

// GetName returns the task name
func (t *UpdateCheckerTask) GetName() string {
	return "Image Update Checker"
}

// GetType returns the task type
func (t *UpdateCheckerTask) GetType() model.TaskType {
	return model.TaskTypeImageCheck
}

// Validate validates task parameters
func (t *UpdateCheckerTask) Validate(params scheduler.TaskParameters) error {
	if params.TaskType != model.TaskTypeImageCheck {
		return fmt.Errorf("invalid task type: expected %s, got %s", model.TaskTypeImageCheck, params.TaskType)
	}

	// Validate parameters structure
	if _, err := t.parseParameters(params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	return nil
}

// GetDefaultTimeout returns the default timeout for this task
func (t *UpdateCheckerTask) GetDefaultTimeout() time.Duration {
	return 15 * time.Minute
}

// CanRunConcurrently returns true if this task can run concurrently
func (t *UpdateCheckerTask) CanRunConcurrently() bool {
	return true
}

// ImageCheckParameters represents parameters for image checking
type ImageCheckParameters struct {
	RegistryTimeout   time.Duration `json:"registry_timeout"`
	MaxConcurrent     int           `json:"max_concurrent"`
	CheckTags         []string      `json:"check_tags"`
	IgnoreArchs       []string      `json:"ignore_archs"`
	NotifyOnNewImage  bool          `json:"notify_on_new_image"`
	CheckBeta         bool          `json:"check_beta"`
	CheckRC           bool          `json:"check_rc"`
	IncludePreRelease bool          `json:"include_pre_release"`
	OnlyMajorUpdates  bool          `json:"only_major_updates"`
	OnlySecurityUpdates bool        `json:"only_security_updates"`
}

// UpdateCheckResult represents the result of checking updates for all containers
type UpdateCheckResult struct {
	ContainerResults []*ContainerUpdateResult `json:"container_results"`
	UpdatesFound     int                      `json:"updates_found"`
	Errors           []UpdateCheckError       `json:"errors"`
	Duration         time.Duration            `json:"duration"`
	CheckedAt        time.Time                `json:"checked_at"`
}

// ContainerUpdateResult represents the result of checking updates for a single container
type ContainerUpdateResult struct {
	Container        *model.Container     `json:"container"`
	CurrentVersion   string               `json:"current_version"`
	LatestVersion    string               `json:"latest_version"`
	UpdateAvailable  bool                 `json:"update_available"`
	IsSecurityUpdate bool                 `json:"is_security_update"`
	IsMajorUpdate    bool                 `json:"is_major_update"`
	UpdateType       string               `json:"update_type"` // patch, minor, major
	RegistryMetadata map[string]interface{} `json:"registry_metadata,omitempty"`
	CheckedAt        time.Time            `json:"checked_at"`
	Error            string               `json:"error,omitempty"`
}

// UpdateCheckError represents an error during update checking
type UpdateCheckError struct {
	ContainerID   int64  `json:"container_id"`
	ContainerName string `json:"container_name"`
	Error         string `json:"error"`
	Recoverable   bool   `json:"recoverable"`
}

// parseParameters parses and validates task parameters
func (t *UpdateCheckerTask) parseParameters(params scheduler.TaskParameters) (*ImageCheckParameters, error) {
	// Set defaults
	checkParams := &ImageCheckParameters{
		RegistryTimeout:   30 * time.Second,
		MaxConcurrent:     5,
		CheckTags:         []string{"latest"},
		IgnoreArchs:       []string{},
		NotifyOnNewImage:  true,
		CheckBeta:         false,
		CheckRC:           false,
		IncludePreRelease: false,
		OnlyMajorUpdates:  false,
		OnlySecurityUpdates: false,
	}

	// Parse from parameters map
	if params.Parameters != nil {
		// Convert parameters to JSON and back to struct for type safety
		jsonData, err := json.Marshal(params.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		if err := json.Unmarshal(jsonData, checkParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	// Validate parameters
	if checkParams.MaxConcurrent <= 0 {
		checkParams.MaxConcurrent = 5
	}
	if checkParams.MaxConcurrent > 20 {
		checkParams.MaxConcurrent = 20
	}

	if checkParams.RegistryTimeout <= 0 {
		checkParams.RegistryTimeout = 30 * time.Second
	}

	if len(checkParams.CheckTags) == 0 {
		checkParams.CheckTags = []string{"latest"}
	}

	return checkParams, nil
}

// getContainersToCheck retrieves containers that should be checked for updates
func (t *UpdateCheckerTask) getContainersToCheck(ctx context.Context, targetContainers []int64) ([]*model.Container, error) {
	var containers []*model.Container

	if len(targetContainers) > 0 {
		// Check specific containers
		for _, containerID := range targetContainers {
			container, err := t.containerRepo.GetByID(ctx, containerID)
			if err != nil {
				logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to get container")
				continue
			}
			containers = append(containers, container)
		}
	} else {
		// Check all active containers with automatic update policy
		runningStatus := model.ContainerStatusRunning
		autoPolicy := model.UpdatePolicyAuto
		filter := &model.ContainerFilter{
			Status: runningStatus,
			// Only check containers with automatic update policies
			UpdatePolicy: autoPolicy,
			Limit:        1000, // Reasonable limit
		}

		allContainers, _, err := t.containerRepo.List(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to list containers: %w", err)
		}

		containers = allContainers
	}

	return containers, nil
}

// checkForUpdates checks for updates for all containers
func (t *UpdateCheckerTask) checkForUpdates(ctx context.Context, containers []*model.Container, params *ImageCheckParameters) (*UpdateCheckResult, error) {
	startTime := time.Now()
	result := &UpdateCheckResult{
		ContainerResults: make([]*ContainerUpdateResult, 0, len(containers)),
		CheckedAt:        startTime,
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, params.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, container := range containers {
		wg.Add(1)
		go func(c *model.Container) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			// Check this container for updates
			containerResult := t.checkContainerUpdate(ctx, c, params)

			// Add to results
			mu.Lock()
			result.ContainerResults = append(result.ContainerResults, containerResult)
			if containerResult.UpdateAvailable {
				result.UpdatesFound++
			}
			if containerResult.Error != "" {
				result.Errors = append(result.Errors, UpdateCheckError{
					ContainerID:   int64(c.ID),
					ContainerName: c.Name,
					Error:         containerResult.Error,
					Recoverable:   true,
				})
			}
			mu.Unlock()
		}(container)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	return result, nil
}

// checkContainerUpdate checks for updates for a single container
func (t *UpdateCheckerTask) checkContainerUpdate(ctx context.Context, container *model.Container, params *ImageCheckParameters) *ContainerUpdateResult {
	result := &ContainerUpdateResult{
		Container:     container,
		CheckedAt:     time.Now(),
		CurrentVersion: container.Tag,
	}

	logger := logrus.WithFields(logrus.Fields{
		"container_id":   container.ID,
		"container_name": container.Name,
		"image":          container.Image,
		"current_tag":    container.Tag,
	})

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, params.RegistryTimeout)
	defer cancel()

	// Check for latest version using image checker
	image := container.GetFullImageName()
	updateResult, err := (*t.registryChecker).CheckImageUpdate(checkCtx, image, "", container.RegistryURL)

	if err != nil {
		result.Error = err.Error()
		logger.WithError(err).Warn("Failed to check for image updates")
		return result
	}

	result.LatestVersion = updateResult.LatestTag
	result.UpdateAvailable = updateResult.UpdateAvailable
	result.UpdateType = updateResult.UpdateType

	// Determine if update is available
	if updateResult.UpdateAvailable {
		result.IsMajorUpdate = result.UpdateType == "major"
		result.IsSecurityUpdate = len(updateResult.SecurityIssues) > 0

		// Apply filtering based on parameters
		if params.OnlyMajorUpdates && !result.IsMajorUpdate {
			result.UpdateAvailable = false
		}
		if params.OnlySecurityUpdates && !result.IsSecurityUpdate {
			result.UpdateAvailable = false
		}

		logger.WithFields(logrus.Fields{
			"latest_version":    result.LatestVersion,
			"update_type":       result.UpdateType,
			"is_major":          result.IsMajorUpdate,
			"is_security":       result.IsSecurityUpdate,
		}).Info("Update available for container")
	}

	return result
}

// processResults processes the update check results
func (t *UpdateCheckerTask) processResults(ctx context.Context, results *UpdateCheckResult, params *ImageCheckParameters) error {
	// Save image version information
	for _, containerResult := range results.ContainerResults {
		if err := t.saveImageVersion(ctx, containerResult); err != nil {
			logrus.WithError(err).WithField("container_id", containerResult.Container.ID).
				Warn("Failed to save image version")
		}
	}

	// Send notifications if enabled
	if params.NotifyOnNewImage {
		if err := t.sendNotifications(ctx, results); err != nil {
			logrus.WithError(err).Warn("Failed to send notifications")
		}
	}

	// Update container update statuses
	for _, containerResult := range results.ContainerResults {
		if containerResult.UpdateAvailable {
			// Update container's last checked time and available update info
			// This would typically be stored in a separate table or field
			logrus.WithFields(logrus.Fields{
				"container_id":   containerResult.Container.ID,
				"container_name": containerResult.Container.Name,
				"latest_version": containerResult.LatestVersion,
				"update_type":    containerResult.UpdateType,
			}).Info("Update available for container")
		}
	}

	return nil
}

// saveImageVersion saves image version information to the database
func (t *UpdateCheckerTask) saveImageVersion(ctx context.Context, result *ContainerUpdateResult) error {
	if t.imageRepo == nil {
		return nil
	}

	imageVersion := &model.ImageVersion{
		ImageName:   result.Container.Image,
		Tag:         result.LatestVersion,
		RegistryURL: result.Container.RegistryURL,
		CheckedAt:   result.CheckedAt,
	}

	// Check if this version already exists
	existing, err := t.imageRepo.GetByImageAndTag(ctx, result.Container.Image, result.LatestVersion)
	if err == nil && existing != nil {
		// Update existing record
		existing.CheckedAt = result.CheckedAt
		existing.IsLatest = true
		existing.Metadata = imageVersion.Metadata
		return t.imageRepo.Update(ctx, existing)
	}

	// Create new record
	return t.imageRepo.Create(ctx, imageVersion)
}

// sendNotifications sends notifications about available updates
func (t *UpdateCheckerTask) sendNotifications(ctx context.Context, results *UpdateCheckResult) error {
	if t.notificationService == nil || results.UpdatesFound == 0 {
		return nil
	}

	// Prepare notification content
	var updatesAvailable []string
	var securityUpdates []string

	for _, result := range results.ContainerResults {
		if result.UpdateAvailable {
			updateMsg := fmt.Sprintf("%s: %s → %s (%s)",
				result.Container.Name,
				result.CurrentVersion,
				result.LatestVersion,
				result.UpdateType)

			updatesAvailable = append(updatesAvailable, updateMsg)

			if result.IsSecurityUpdate {
				securityUpdates = append(securityUpdates, updateMsg)
			}
		}
	}

	// Send security update notifications with high priority
	if len(securityUpdates) > 0 {
		notification := &model.Notification{
			Type:     model.NotificationTypeSecurityUpdate,
			Title:    "Security Updates Available",
			Message:  fmt.Sprintf("Security updates are available for %d container(s):\n%s",
				len(securityUpdates), strings.Join(securityUpdates, "\n")),
			Priority: model.NotificationPriorityHigh,
			Data:     map[string]interface{}{
				"security_updates": securityUpdates,
				"total_updates":    results.UpdatesFound,
			},
		}

		if err := t.notificationService.SendNotification(ctx, notification); err != nil {
			logrus.WithError(err).Warn("Failed to send security update notification")
		}
	}

	// Send general update notifications
	if len(updatesAvailable) > 0 {
		notification := &model.Notification{
			Type:     model.NotificationTypeImageUpdate,
			Title:    "Container Updates Available",
			Message:  fmt.Sprintf("Updates are available for %d container(s):\n%s",
				results.UpdatesFound, strings.Join(updatesAvailable, "\n")),
			Priority: model.NotificationPriorityNormal,
			Data:     map[string]interface{}{
				"updates_available": updatesAvailable,
				"total_updates":     results.UpdatesFound,
				"security_updates":  len(securityUpdates),
			},
		}

		if err := t.notificationService.SendNotification(ctx, notification); err != nil {
			logrus.WithError(err).Warn("Failed to send update notification")
		}
	}

	return nil
}

// parseRegistryAuth parses registry authentication from JSON string
func (t *UpdateCheckerTask) parseRegistryAuth(authJSON string) *registry.AuthConfig {
	if authJSON == "" {
		return nil
	}

	var auth registry.AuthConfig
	if err := json.Unmarshal([]byte(authJSON), &auth); err != nil {
		logrus.WithError(err).Warn("Failed to parse registry auth")
		return nil
	}

	return &auth
}

// determineUpdateType determines the type of update (patch, minor, major)
func (t *UpdateCheckerTask) determineUpdateType(currentTag, latestTag string) string {
	// Simple semantic version comparison
	// This is a basic implementation - you might want to use a proper semver library
	if strings.Contains(latestTag, "major") || strings.Contains(currentTag, "v1") && strings.Contains(latestTag, "v2") {
		return "major"
	}
	if strings.Contains(latestTag, "minor") {
		return "minor"
	}
	return "patch"
}

// isSecurityUpdate determines if an update is a security update
func (t *UpdateCheckerTask) isSecurityUpdate(imageInfo *registry.ImageInfo) bool {
	// Check metadata for security indicators
	if imageInfo.Metadata != nil {
		if labels, ok := imageInfo.Metadata["labels"].(map[string]interface{}); ok {
			// Check for security-related labels
			for key, value := range labels {
				keyLower := strings.ToLower(key)
				valueLower := strings.ToLower(fmt.Sprintf("%v", value))

				if strings.Contains(keyLower, "security") ||
					strings.Contains(keyLower, "cve") ||
					strings.Contains(keyLower, "vulnerability") ||
					strings.Contains(valueLower, "security") ||
					strings.Contains(valueLower, "cve") {
					return true
				}
			}
		}
	}

	return false
}

// serializeMetadata converts metadata map to JSON string
func (t *UpdateCheckerTask) serializeMetadata(metadata map[string]interface{}) string {
	if metadata == nil {
		return "{}"
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		logrus.WithError(err).Warn("Failed to serialize metadata")
		return "{}"
	}

	return string(jsonData)
}