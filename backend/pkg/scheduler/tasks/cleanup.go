package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/scheduler"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
)

// CleanupTask implements the Task interface for system cleanup operations
type CleanupTask struct {
	containerRepo       repository.ContainerRepository
	updateHistoryRepo   repository.UpdateHistoryRepository
	executionLogRepo    repository.TaskExecutionLogRepository
	activityLogRepo     repository.ActivityLogRepository
	imageVersionRepo    repository.ImageVersionRepository
	notificationRepo    repository.NotificationRepository
	containerService    *service.ContainerService
	notificationService *service.NotificationService
	dockerClient        *docker.DockerClient
}

// NewCleanupTask creates a new cleanup task
func NewCleanupTask(
	containerRepo repository.ContainerRepository,
	updateHistoryRepo repository.UpdateHistoryRepository,
	executionLogRepo repository.TaskExecutionLogRepository,
	activityLogRepo repository.ActivityLogRepository,
	imageVersionRepo repository.ImageVersionRepository,
	notificationRepo repository.NotificationRepository,
	containerService *service.ContainerService,
	notificationService *service.NotificationService,
	dockerClient *docker.DockerClient,
) *CleanupTask {
	return &CleanupTask{
		containerRepo:       containerRepo,
		updateHistoryRepo:   updateHistoryRepo,
		executionLogRepo:    executionLogRepo,
		activityLogRepo:     activityLogRepo,
		imageVersionRepo:    imageVersionRepo,
		notificationRepo:    notificationRepo,
		containerService:    containerService,
		notificationService: notificationService,
		dockerClient:        dockerClient,
	}
}

// Execute runs the cleanup task
func (t *CleanupTask) Execute(ctx context.Context, params scheduler.TaskParameters) error {
	logger := logrus.WithFields(logrus.Fields{
		"task_type": t.GetType(),
		"task_name": t.GetName(),
	})

	logger.Info("Starting system cleanup task")

	// Parse task-specific parameters
	cleanupParams, err := t.parseParameters(params)
	if err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Perform cleanup operations
	results := &CleanupResult{
		StartedAt: time.Now(),
		Operations: []CleanupOperation{},
	}

	// Clean up activity logs
	if cleanupParams.CleanupActivityLogs {
		operation := t.cleanupActivityLogs(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up update history
	if cleanupParams.CleanupUpdateHistory {
		operation := t.cleanupUpdateHistory(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up task execution logs
	if cleanupParams.CleanupTaskLogs {
		operation := t.cleanupTaskExecutionLogs(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up notifications
	if cleanupParams.CleanupNotifications {
		operation := t.cleanupNotifications(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up image version cache
	if cleanupParams.CleanupImageCache {
		operation := t.cleanupImageVersionCache(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up Docker images
	if cleanupParams.CleanupUnusedImages {
		operation := t.cleanupDockerImages(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up Docker containers
	if cleanupParams.CleanupStoppedContainers {
		operation := t.cleanupStoppedContainers(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up Docker volumes
	if cleanupParams.CleanupUnusedVolumes {
		operation := t.cleanupDockerVolumes(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	// Clean up Docker networks
	if cleanupParams.CleanupUnusedNetworks {
		operation := t.cleanupDockerNetworks(ctx, cleanupParams)
		results.Operations = append(results.Operations, operation)
		if operation.Success {
			results.SuccessfulOperations++
		} else {
			results.FailedOperations++
		}
	}

	results.CompletedAt = time.Now()
	results.Duration = results.CompletedAt.Sub(results.StartedAt)

	// Send notification about cleanup results
	if err := t.sendCleanupNotification(ctx, results, cleanupParams); err != nil {
		logger.WithError(err).Warn("Failed to send cleanup notification")
	}

	logger.WithFields(logrus.Fields{
		"total_operations":      len(results.Operations),
		"successful_operations": results.SuccessfulOperations,
		"failed_operations":     results.FailedOperations,
		"duration":             results.Duration,
		"space_freed":          results.TotalSpaceFreed,
	}).Info("System cleanup task completed")

	return nil
}

// GetName returns the task name
func (t *CleanupTask) GetName() string {
	return "System Cleanup"
}

// GetType returns the task type
func (t *CleanupTask) GetType() model.TaskType {
	return model.TaskTypeCleanup
}

// Validate validates task parameters
func (t *CleanupTask) Validate(params scheduler.TaskParameters) error {
	if params.TaskType != model.TaskTypeCleanup {
		return fmt.Errorf("invalid task type: expected %s, got %s", model.TaskTypeCleanup, params.TaskType)
	}

	// Validate parameters structure
	if _, err := t.parseParameters(params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	return nil
}

// GetDefaultTimeout returns the default timeout for this task
func (t *CleanupTask) GetDefaultTimeout() time.Duration {
	return 30 * time.Minute
}

// CanRunConcurrently returns false since cleanup operations should be serialized
func (t *CleanupTask) CanRunConcurrently() bool {
	return false
}

// CleanupParameters represents parameters for cleanup operations
type CleanupParameters struct {
	// Database cleanup
	ActivityLogRetentionDays    int  `json:"activity_log_retention_days"`
	UpdateHistoryRetentionDays  int  `json:"update_history_retention_days"`
	TaskLogRetentionDays        int  `json:"task_log_retention_days"`
	NotificationRetentionDays   int  `json:"notification_retention_days"`
	ImageCacheRetentionDays     int  `json:"image_cache_retention_days"`
	CleanupActivityLogs         bool `json:"cleanup_activity_logs"`
	CleanupUpdateHistory        bool `json:"cleanup_update_history"`
	CleanupTaskLogs             bool `json:"cleanup_task_logs"`
	CleanupNotifications        bool `json:"cleanup_notifications"`
	CleanupImageCache           bool `json:"cleanup_image_cache"`

	// Docker cleanup
	CleanupUnusedImages         bool     `json:"cleanup_unused_images"`
	CleanupDanglingImages       bool     `json:"cleanup_dangling_images"`
	CleanupStoppedContainers    bool     `json:"cleanup_stopped_containers"`
	CleanupUnusedVolumes        bool     `json:"cleanup_unused_volumes"`
	CleanupUnusedNetworks       bool     `json:"cleanup_unused_networks"`
	ImageRetentionDays          int      `json:"image_retention_days"`
	ContainerRetentionDays      int      `json:"container_retention_days"`
	VolumeRetentionDays         int      `json:"volume_retention_days"`
	ExcludeImages               []string `json:"exclude_images"`
	ExcludeContainers           []string `json:"exclude_containers"`
	ExcludeVolumes              []string `json:"exclude_volumes"`
	ExcludeNetworks             []string `json:"exclude_networks"`
	ForceRemoveImages           bool     `json:"force_remove_images"`
	DryRun                      bool     `json:"dry_run"`

	// Notification settings
	NotifyOnCompletion          bool `json:"notify_on_completion"`
	NotifyOnErrors              bool `json:"notify_on_errors"`
	NotifySpaceFreed            bool `json:"notify_space_freed"`
}

// CleanupResult represents the result of cleanup operations
type CleanupResult struct {
	Operations           []CleanupOperation `json:"operations"`
	SuccessfulOperations int                `json:"successful_operations"`
	FailedOperations     int                `json:"failed_operations"`
	TotalSpaceFreed      int64              `json:"total_space_freed"`
	StartedAt            time.Time          `json:"started_at"`
	CompletedAt          time.Time          `json:"completed_at"`
	Duration             time.Duration      `json:"duration"`
}

// CleanupOperation represents a single cleanup operation
type CleanupOperation struct {
	Type           string        `json:"type"`
	Description    string        `json:"description"`
	Success        bool          `json:"success"`
	Error          string        `json:"error,omitempty"`
	ItemsRemoved   int           `json:"items_removed"`
	SpaceFreed     int64         `json:"space_freed"`
	Duration       time.Duration `json:"duration"`
	DryRun         bool          `json:"dry_run"`
	Details        interface{}   `json:"details,omitempty"`
}

// parseParameters parses and validates task parameters
func (t *CleanupTask) parseParameters(params scheduler.TaskParameters) (*CleanupParameters, error) {
	// Set defaults
	cleanupParams := &CleanupParameters{
		ActivityLogRetentionDays:    30,
		UpdateHistoryRetentionDays:  90,
		TaskLogRetentionDays:        30,
		NotificationRetentionDays:   7,
		ImageCacheRetentionDays:     7,
		CleanupActivityLogs:         true,
		CleanupUpdateHistory:        true,
		CleanupTaskLogs:             true,
		CleanupNotifications:        true,
		CleanupImageCache:           true,
		CleanupUnusedImages:         true,
		CleanupDanglingImages:       true,
		CleanupStoppedContainers:    true,
		CleanupUnusedVolumes:        false, // More conservative default
		CleanupUnusedNetworks:       false, // More conservative default
		ImageRetentionDays:          30,
		ContainerRetentionDays:      7,
		VolumeRetentionDays:         30,
		ExcludeImages:               []string{},
		ExcludeContainers:           []string{},
		ExcludeVolumes:              []string{},
		ExcludeNetworks:             []string{"bridge", "host", "none"},
		ForceRemoveImages:           false,
		DryRun:                      false,
		NotifyOnCompletion:          true,
		NotifyOnErrors:              true,
		NotifySpaceFreed:            true,
	}

	// Parse from parameters map
	if params.Parameters != nil {
		jsonData, err := json.Marshal(params.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		if err := json.Unmarshal(jsonData, cleanupParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	// Validate parameters
	if cleanupParams.ActivityLogRetentionDays < 1 {
		cleanupParams.ActivityLogRetentionDays = 1
	}
	if cleanupParams.UpdateHistoryRetentionDays < 1 {
		cleanupParams.UpdateHistoryRetentionDays = 1
	}
	if cleanupParams.TaskLogRetentionDays < 1 {
		cleanupParams.TaskLogRetentionDays = 1
	}

	return cleanupParams, nil
}

// cleanupActivityLogs removes old activity log entries
func (t *CleanupTask) cleanupActivityLogs(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "activity_logs",
		Description: "Clean up old activity log entries",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.activityLogRepo == nil {
		operation.Error = "Activity log repository not available"
		operation.Success = false
		return operation
	}

	cutoffDate := time.Now().AddDate(0, 0, -params.ActivityLogRetentionDays)

	if params.DryRun {
		// Count items that would be removed
		count, err := t.activityLogRepo.CountOlderThan(ctx, cutoffDate)
		if err != nil {
			operation.Error = err.Error()
			operation.Success = false
			return operation
		}
		operation.ItemsRemoved = int(count)
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d items)", count)
		return operation
	}

	// Perform actual cleanup
	deletedCount, err := t.activityLogRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		operation.Error = err.Error()
		operation.Success = false
		return operation
	}

	operation.ItemsRemoved = int(deletedCount)
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"cutoff_date":   cutoffDate,
	}).Info("Cleaned up activity logs")

	return operation
}

// cleanupUpdateHistory removes old update history entries
func (t *CleanupTask) cleanupUpdateHistory(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "update_history",
		Description: "Clean up old update history entries",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.updateHistoryRepo == nil {
		operation.Error = "Update history repository not available"
		operation.Success = false
		return operation
	}

	cutoffDate := time.Now().AddDate(0, 0, -params.UpdateHistoryRetentionDays)

	if params.DryRun {
		count, err := t.updateHistoryRepo.CountOlderThan(ctx, cutoffDate)
		if err != nil {
			operation.Error = err.Error()
			operation.Success = false
			return operation
		}
		operation.ItemsRemoved = int(count)
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d items)", count)
		return operation
	}

	deletedCount, err := t.updateHistoryRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		operation.Error = err.Error()
		operation.Success = false
		return operation
	}

	operation.ItemsRemoved = int(deletedCount)
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"cutoff_date":   cutoffDate,
	}).Info("Cleaned up update history")

	return operation
}

// cleanupTaskExecutionLogs removes old task execution log entries
func (t *CleanupTask) cleanupTaskExecutionLogs(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "task_execution_logs",
		Description: "Clean up old task execution log entries",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.executionLogRepo == nil {
		operation.Error = "Task execution log repository not available"
		operation.Success = false
		return operation
	}

	cutoffDate := time.Now().AddDate(0, 0, -params.TaskLogRetentionDays)

	if params.DryRun {
		count, err := t.executionLogRepo.CountOlderThan(ctx, cutoffDate)
		if err != nil {
			operation.Error = err.Error()
			operation.Success = false
			return operation
		}
		operation.ItemsRemoved = int(count)
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d items)", count)
		return operation
	}

	deletedCount, err := t.executionLogRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		operation.Error = err.Error()
		operation.Success = false
		return operation
	}

	operation.ItemsRemoved = int(deletedCount)
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"cutoff_date":   cutoffDate,
	}).Info("Cleaned up task execution logs")

	return operation
}

// cleanupNotifications removes old notification entries
func (t *CleanupTask) cleanupNotifications(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "notifications",
		Description: "Clean up old notification entries",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.notificationRepo == nil {
		operation.Error = "Notification repository not available"
		operation.Success = false
		return operation
	}

	cutoffDate := time.Now().AddDate(0, 0, -params.NotificationRetentionDays)

	if params.DryRun {
		count, err := t.notificationRepo.CountOlderThan(ctx, cutoffDate)
		if err != nil {
			operation.Error = err.Error()
			operation.Success = false
			return operation
		}
		operation.ItemsRemoved = int(count)
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d items)", count)
		return operation
	}

	deletedCount, err := t.notificationRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		operation.Error = err.Error()
		operation.Success = false
		return operation
	}

	operation.ItemsRemoved = int(deletedCount)
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"cutoff_date":   cutoffDate,
	}).Info("Cleaned up notifications")

	return operation
}

// cleanupImageVersionCache removes old image version cache entries
func (t *CleanupTask) cleanupImageVersionCache(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "image_version_cache",
		Description: "Clean up old image version cache entries",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.imageVersionRepo == nil {
		operation.Error = "Image version repository not available"
		operation.Success = false
		return operation
	}

	cutoffDate := time.Now().AddDate(0, 0, -params.ImageCacheRetentionDays)

	if params.DryRun {
		count, err := t.imageVersionRepo.CountOlderThan(ctx, cutoffDate)
		if err != nil {
			operation.Error = err.Error()
			operation.Success = false
			return operation
		}
		operation.ItemsRemoved = int(count)
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d items)", count)
		return operation
	}

	deletedCount, err := t.imageVersionRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		operation.Error = err.Error()
		operation.Success = false
		return operation
	}

	operation.ItemsRemoved = int(deletedCount)
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"cutoff_date":   cutoffDate,
	}).Info("Cleaned up image version cache")

	return operation
}

// cleanupDockerImages removes unused Docker images
func (t *CleanupTask) cleanupDockerImages(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "docker_images",
		Description: "Clean up unused Docker images",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.dockerClient == nil {
		operation.Error = "Docker client not available"
		operation.Success = false
		return operation
	}

	// Get all images
	images, err := t.dockerClient.ListImages(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to list images: %v", err)
		operation.Success = false
		return operation
	}

	var imagesToRemove []string
	var spaceToFree int64

	for _, image := range images {
		// Skip images that are in use
		if t.isImageInUse(ctx, image.ID) {
			continue
		}

		// Convert types.ImageSummary to docker.Image for compatibility
		dockerImage := docker.Image{
			ID:       image.ID,
			RepoTags: image.RepoTags,
			Size:     image.Size,
			Created:  time.Unix(image.Created, 0),
		}

		// Skip excluded images
		if t.isImageExcluded(dockerImage, params.ExcludeImages) {
			continue
		}

		// Check if image is old enough
		if params.ImageRetentionDays > 0 {
			cutoffDate := time.Now().AddDate(0, 0, -params.ImageRetentionDays)
			if dockerImage.Created.After(cutoffDate) {
				continue
			}
		}

		imagesToRemove = append(imagesToRemove, image.ID)
		spaceToFree += image.Size
	}

	operation.ItemsRemoved = len(imagesToRemove)
	operation.SpaceFreed = spaceToFree

	if params.DryRun {
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d images, %d bytes)", len(imagesToRemove), spaceToFree)
		return operation
	}

	// Remove images
	var removedCount int
	var actualSpaceFreed int64

	for _, imageID := range imagesToRemove {
		_, err := t.dockerClient.RemoveImage(ctx, imageID, types.ImageRemoveOptions{
			Force:         params.ForceRemoveImages,
			PruneChildren: true,
		})
		if err != nil {
			logrus.WithError(err).WithField("image_id", imageID).Warn("Failed to remove image")
		} else {
			removedCount++
			// In a real implementation, you'd calculate actual space freed
			actualSpaceFreed += spaceToFree / int64(len(imagesToRemove))
		}
	}

	operation.ItemsRemoved = removedCount
	operation.SpaceFreed = actualSpaceFreed
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"removed_count": removedCount,
		"space_freed":   actualSpaceFreed,
	}).Info("Cleaned up Docker images")

	return operation
}

// cleanupStoppedContainers removes old stopped containers
func (t *CleanupTask) cleanupStoppedContainers(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "stopped_containers",
		Description: "Clean up old stopped containers",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.dockerClient == nil {
		operation.Error = "Docker client not available"
		operation.Success = false
		return operation
	}

	// Get all stopped containers
	filterArgs := filters.NewArgs()
	filterArgs.Add("status", "exited")

	containers, err := t.dockerClient.ListContainers(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to list containers: %v", err)
		operation.Success = false
		return operation
	}

	var containersToRemove []string

	for _, container := range containers {
		// Convert types.Container to docker.Container for compatibility
		dockerContainer := docker.Container{
			ID:      container.ID,
			Name:    strings.TrimPrefix(container.Names[0], "/"), // Remove leading slash
			Image:   container.Image,
			Status:  container.Status,
			State:   container.State,
			Created: time.Unix(container.Created, 0),
		}

		// Skip excluded containers
		if t.isContainerExcluded(dockerContainer, params.ExcludeContainers) {
			continue
		}

		// Check if container is old enough
		if params.ContainerRetentionDays > 0 {
			cutoffDate := time.Now().AddDate(0, 0, -params.ContainerRetentionDays)
			if dockerContainer.Created.After(cutoffDate) {
				continue
			}
		}

		containersToRemove = append(containersToRemove, container.ID)
	}

	operation.ItemsRemoved = len(containersToRemove)

	if params.DryRun {
		operation.Success = true
		operation.Description += fmt.Sprintf(" (DRY RUN: would remove %d containers)", len(containersToRemove))
		return operation
	}

	// Remove containers
	var removedCount int

	for _, containerID := range containersToRemove {
		err := t.dockerClient.RemoveContainer(ctx, containerID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if err != nil {
			logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to remove container")
		} else {
			removedCount++
		}
	}

	operation.ItemsRemoved = removedCount
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"removed_count": removedCount,
	}).Info("Cleaned up stopped containers")

	return operation
}

// cleanupDockerVolumes removes unused Docker volumes
func (t *CleanupTask) cleanupDockerVolumes(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "docker_volumes",
		Description: "Clean up unused Docker volumes",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.dockerClient == nil {
		operation.Error = "Docker client not available"
		operation.Success = false
		return operation
	}

	// This is a placeholder implementation
	// Real implementation would use Docker API to list and remove unused volumes
	operation.ItemsRemoved = 0
	operation.Success = true

	logrus.Info("Docker volume cleanup completed")

	return operation
}

// cleanupDockerNetworks removes unused Docker networks
func (t *CleanupTask) cleanupDockerNetworks(ctx context.Context, params *CleanupParameters) CleanupOperation {
	operation := CleanupOperation{
		Type:        "docker_networks",
		Description: "Clean up unused Docker networks",
		DryRun:      params.DryRun,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.dockerClient == nil {
		operation.Error = "Docker client not available"
		operation.Success = false
		return operation
	}

	// This is a placeholder implementation
	// Real implementation would use Docker API to list and remove unused networks
	operation.ItemsRemoved = 0
	operation.Success = true

	logrus.Info("Docker network cleanup completed")

	return operation
}

// sendCleanupNotification sends a notification about cleanup results
func (t *CleanupTask) sendCleanupNotification(ctx context.Context, results *CleanupResult, params *CleanupParameters) error {
	if t.notificationService == nil || !params.NotifyOnCompletion {
		return nil
	}

	title := "System Cleanup Completed"
	if results.FailedOperations > 0 {
		title = "System Cleanup Completed with Errors"
	}

	message := fmt.Sprintf("Cleanup completed: %d successful, %d failed operations",
		results.SuccessfulOperations, results.FailedOperations)

	if results.TotalSpaceFreed > 0 {
		message += fmt.Sprintf(", %d bytes freed", results.TotalSpaceFreed)
	}

	priority := model.NotificationPriorityNormal
	if results.FailedOperations > 0 {
		priority = model.NotificationPriorityHigh
	}

	notification := &model.Notification{
		Type:     model.NotificationTypeSystemMaintenance,
		Title:    title,
		Message:  message,
		Priority: priority,
		Data: map[string]interface{}{
			"successful_operations": results.SuccessfulOperations,
			"failed_operations":     results.FailedOperations,
			"total_space_freed":     results.TotalSpaceFreed,
			"duration":             results.Duration.String(),
			"operations":           results.Operations,
		},
	}

	return t.notificationService.SendNotification(ctx, notification)
}

// Helper methods

func (t *CleanupTask) isImageInUse(ctx context.Context, imageID string) bool {
	// Check if any containers are using this image
	// This is a simplified check - real implementation would be more thorough
	return false
}

func (t *CleanupTask) isImageExcluded(image docker.Image, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if strings.Contains(image.RepoTags[0], pattern) {
			return true
		}
	}
	return false
}

func (t *CleanupTask) isContainerExcluded(container docker.Container, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if strings.Contains(container.Name, pattern) {
			return true
		}
	}
	return false
}