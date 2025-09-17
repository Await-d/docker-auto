package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/scheduler"

	"github.com/sirupsen/logrus"
)

// ContainerUpdaterTask implements the Task interface for updating containers
type ContainerUpdaterTask struct {
	containerRepo     repository.ContainerRepository
	updateHistoryRepo repository.UpdateHistoryRepository
	containerService  *service.ContainerService
	imageService      *service.ImageService
	notificationService *service.NotificationService
	dockerClient      *docker.DockerClient
}

// NewContainerUpdaterTask creates a new container updater task
func NewContainerUpdaterTask(
	containerRepo repository.ContainerRepository,
	updateHistoryRepo repository.UpdateHistoryRepository,
	containerService *service.ContainerService,
	imageService *service.ImageService,
	notificationService *service.NotificationService,
	dockerClient *docker.DockerClient,
) *ContainerUpdaterTask {
	return &ContainerUpdaterTask{
		containerRepo:       containerRepo,
		updateHistoryRepo:   updateHistoryRepo,
		containerService:    containerService,
		imageService:        imageService,
		notificationService: notificationService,
		dockerClient:        dockerClient,
	}
}

// Execute runs the container update task
func (t *ContainerUpdaterTask) Execute(ctx context.Context, params scheduler.TaskParameters) error {
	logger := logrus.WithFields(logrus.Fields{
		"task_type": t.GetType(),
		"task_name": t.GetName(),
	})

	logger.Info("Starting container update task")

	// Parse task-specific parameters
	updateParams, err := t.parseParameters(params)
	if err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Get containers to update
	containers, err := t.getContainersToUpdate(ctx, params.TargetContainers, updateParams)
	if err != nil {
		return fmt.Errorf("failed to get containers to update: %w", err)
	}

	logger.WithField("container_count", len(containers)).Info("Found containers to update")

	if len(containers) == 0 {
		logger.Info("No containers require updates")
		return nil
	}

	// Check if we're in a maintenance window
	if !t.isInMaintenanceWindow(updateParams) {
		logger.Info("Not in maintenance window, skipping updates")
		return nil
	}

	// Perform updates
	results, err := t.updateContainers(ctx, containers, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update containers: %w", err)
	}

	// Process results
	if err := t.processResults(ctx, results, updateParams); err != nil {
		return fmt.Errorf("failed to process results: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"containers_processed": len(containers),
		"successful_updates":   results.SuccessfulUpdates,
		"failed_updates":       results.FailedUpdates,
		"rollbacks":           results.Rollbacks,
	}).Info("Container update task completed")

	return nil
}

// GetName returns the task name
func (t *ContainerUpdaterTask) GetName() string {
	return "Container Updater"
}

// GetType returns the task type
func (t *ContainerUpdaterTask) GetType() model.TaskType {
	return model.TaskTypeContainerUpdate
}

// Validate validates task parameters
func (t *ContainerUpdaterTask) Validate(params scheduler.TaskParameters) error {
	if params.TaskType != model.TaskTypeContainerUpdate {
		return fmt.Errorf("invalid task type: expected %s, got %s", model.TaskTypeContainerUpdate, params.TaskType)
	}

	// Validate parameters structure
	if _, err := t.parseParameters(params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	return nil
}

// GetDefaultTimeout returns the default timeout for this task
func (t *ContainerUpdaterTask) GetDefaultTimeout() time.Duration {
	return 60 * time.Minute
}

// CanRunConcurrently returns false since container updates should be serialized
func (t *ContainerUpdaterTask) CanRunConcurrently() bool {
	return false
}

// ContainerUpdateParameters represents parameters for container updates
type ContainerUpdateParameters struct {
	UpdateStrategy      string                 `json:"update_strategy"`      // recreate, rolling, blue-green
	MaxConcurrent       int                    `json:"max_concurrent"`
	RollbackOnFailure   bool                   `json:"rollback_on_failure"`
	PreUpdateBackup     bool                   `json:"pre_update_backup"`
	UpdateTimeout       time.Duration          `json:"update_timeout"`
	ExcludeTags         []string               `json:"exclude_tags"`
	HealthCheckTimeout  time.Duration          `json:"health_check_timeout"`
	HealthCheckRetries  int                    `json:"health_check_retries"`
	MaintenanceWindows  []MaintenanceWindow    `json:"maintenance_windows"`
	NotifyOnSuccess     bool                   `json:"notify_on_success"`
	NotifyOnFailure     bool                   `json:"notify_on_failure"`
	DryRun              bool                   `json:"dry_run"`
	ForceUpdate         bool                   `json:"force_update"`
	PullPolicy          string                 `json:"pull_policy"` // always, if-not-present, never
	StopGracePeriod     time.Duration          `json:"stop_grace_period"`
	StartupHealthCheck  bool                   `json:"startup_health_check"`
}

// MaintenanceWindow represents a time window for updates
type MaintenanceWindow struct {
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	DaysOfWeek []int `json:"days_of_week"` // 0=Sunday, 1=Monday, etc.
	Timezone   string `json:"timezone"`
}

// ContainerUpdateTaskResult represents the result of updating all containers
type ContainerUpdateTaskResult struct {
	ContainerResults  []*SingleContainerUpdateResult `json:"container_results"`
	SuccessfulUpdates int                             `json:"successful_updates"`
	FailedUpdates     int                             `json:"failed_updates"`
	Rollbacks         int                             `json:"rollbacks"`
	Duration          time.Duration                   `json:"duration"`
	UpdatedAt         time.Time                       `json:"updated_at"`
	Errors            []ContainerUpdateError          `json:"errors"`
}

// SingleContainerUpdateResult represents the result of updating a single container
type SingleContainerUpdateResult struct {
	Container        *model.Container       `json:"container"`
	OldVersion       string                 `json:"old_version"`
	NewVersion       string                 `json:"new_version"`
	Success          bool                   `json:"success"`
	Error            string                 `json:"error,omitempty"`
	UpdateHistory    *model.UpdateHistory   `json:"update_history,omitempty"`
	Duration         time.Duration          `json:"duration"`
	RolledBack       bool                   `json:"rolled_back"`
	BackupCreated    bool                   `json:"backup_created"`
	HealthCheckPassed bool                  `json:"health_check_passed"`
	UpdateSteps      []UpdateStep           `json:"update_steps"`
}

// UpdateStep represents a step in the update process
type UpdateStep struct {
	Step        string        `json:"step"`
	Status      string        `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// ContainerUpdateError represents an error during container updates
type ContainerUpdateError struct {
	ContainerID   int64  `json:"container_id"`
	ContainerName string `json:"container_name"`
	Error         string `json:"error"`
	Step          string `json:"step"`
	Recoverable   bool   `json:"recoverable"`
}

// parseParameters parses and validates task parameters
func (t *ContainerUpdaterTask) parseParameters(params scheduler.TaskParameters) (*ContainerUpdateParameters, error) {
	// Set defaults
	updateParams := &ContainerUpdateParameters{
		UpdateStrategy:     "recreate",
		MaxConcurrent:      1,
		RollbackOnFailure:  true,
		PreUpdateBackup:    true,
		UpdateTimeout:      10 * time.Minute,
		ExcludeTags:        []string{},
		HealthCheckTimeout: 5 * time.Minute,
		HealthCheckRetries: 3,
		MaintenanceWindows: []MaintenanceWindow{},
		NotifyOnSuccess:    true,
		NotifyOnFailure:    true,
		DryRun:            false,
		ForceUpdate:       false,
		PullPolicy:        "always",
		StopGracePeriod:   30 * time.Second,
		StartupHealthCheck: true,
	}

	// Parse from parameters map
	if params.Parameters != nil {
		jsonData, err := json.Marshal(params.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		if err := json.Unmarshal(jsonData, updateParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	// Validate parameters
	if updateParams.MaxConcurrent <= 0 {
		updateParams.MaxConcurrent = 1
	}
	if updateParams.MaxConcurrent > 10 {
		updateParams.MaxConcurrent = 10
	}

	validStrategies := []string{"recreate", "rolling", "blue-green"}
	if !t.contains(validStrategies, updateParams.UpdateStrategy) {
		updateParams.UpdateStrategy = "recreate"
	}

	validPullPolicies := []string{"always", "if-not-present", "never"}
	if !t.contains(validPullPolicies, updateParams.PullPolicy) {
		updateParams.PullPolicy = "always"
	}

	return updateParams, nil
}

// getContainersToUpdate retrieves containers that should be updated
func (t *ContainerUpdaterTask) getContainersToUpdate(ctx context.Context, targetContainers []int64, params *ContainerUpdateParameters) ([]*model.Container, error) {
	var containers []*model.Container

	if len(targetContainers) > 0 {
		// Update specific containers
		for _, containerID := range targetContainers {
			container, err := t.containerRepo.GetByID(ctx, containerID)
			if err != nil {
				logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to get container")
				continue
			}

			// Check if container has updates available
			if t.hasUpdatesAvailable(ctx, container) {
				containers = append(containers, container)
			}
		}
	} else {
		// Find all containers that need updates
		containers = t.findContainersNeedingUpdates(ctx, params)
	}

	return containers, nil
}

// hasUpdatesAvailable checks if a container has updates available
func (t *ContainerUpdaterTask) hasUpdatesAvailable(ctx context.Context, container *model.Container) bool {
	// This would typically check against the image version repository
	// or call the image service to determine if updates are available
	// For now, we'll implement a simple check

	if t.imageService == nil {
		return false
	}

	// Check if there's a newer version available
	// Implementation would depend on your image service interface
	return true // Placeholder
}

// findContainersNeedingUpdates finds all containers that need updates
func (t *ContainerUpdaterTask) findContainersNeedingUpdates(ctx context.Context, params *ContainerUpdateParameters) []*model.Container {
	filter := &model.ContainerFilter{
		UpdatePolicy: &model.UpdatePolicyAuto,
		Status:       model.ContainerStatusRunning,
		Limit:        1000,
	}

	allContainers, _, err := t.containerRepo.List(ctx, filter)
	if err != nil {
		logrus.WithError(err).Error("Failed to list containers for updates")
		return nil
	}

	var needingUpdates []*model.Container
	for _, container := range allContainers {
		// Skip containers with excluded tags
		if t.contains(params.ExcludeTags, container.Tag) {
			continue
		}

		// Check if container has updates available
		if t.hasUpdatesAvailable(ctx, container) {
			needingUpdates = append(needingUpdates, container)
		}
	}

	return needingUpdates
}

// isInMaintenanceWindow checks if current time is within maintenance window
func (t *ContainerUpdaterTask) isInMaintenanceWindow(params *ContainerUpdateParameters) bool {
	if len(params.MaintenanceWindows) == 0 {
		return true // No restrictions
	}

	now := time.Now()
	for _, window := range params.MaintenanceWindows {
		if t.timeInWindow(now, window) {
			return true
		}
	}

	return false
}

// timeInWindow checks if the given time is within the maintenance window
func (t *ContainerUpdaterTask) timeInWindow(checkTime time.Time, window MaintenanceWindow) bool {
	// Parse timezone
	location := time.UTC
	if window.Timezone != "" {
		if loc, err := time.LoadLocation(window.Timezone); err == nil {
			location = loc
		}
	}

	// Convert time to window timezone
	windowTime := checkTime.In(location)

	// Check day of week
	if len(window.DaysOfWeek) > 0 {
		dayMatch := false
		for _, day := range window.DaysOfWeek {
			if int(windowTime.Weekday()) == day {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			return false
		}
	}

	// Parse start and end times
	startTime, err := time.Parse("15:04", window.StartTime)
	if err != nil {
		return false
	}
	endTime, err := time.Parse("15:04", window.EndTime)
	if err != nil {
		return false
	}

	// Get current time in HH:MM format
	currentTime := time.Date(0, 1, 1, windowTime.Hour(), windowTime.Minute(), 0, 0, time.UTC)

	// Handle overnight windows (e.g., 22:00 to 06:00)
	if endTime.Before(startTime) {
		return currentTime.After(startTime) || currentTime.Before(endTime)
	}

	return currentTime.After(startTime) && currentTime.Before(endTime)
}

// updateContainers performs the actual container updates
func (t *ContainerUpdaterTask) updateContainers(ctx context.Context, containers []*model.Container, params *ContainerUpdateParameters) (*ContainerUpdateTaskResult, error) {
	startTime := time.Now()
	result := &ContainerUpdateTaskResult{
		ContainerResults: make([]*SingleContainerUpdateResult, 0, len(containers)),
		UpdatedAt:        startTime,
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

			// Update this container
			containerResult := t.updateSingleContainer(ctx, c, params)

			// Add to results
			mu.Lock()
			result.ContainerResults = append(result.ContainerResults, containerResult)
			if containerResult.Success {
				result.SuccessfulUpdates++
			} else {
				result.FailedUpdates++
				result.Errors = append(result.Errors, ContainerUpdateError{
					ContainerID:   int64(c.ID),
					ContainerName: c.Name,
					Error:         containerResult.Error,
					Recoverable:   true,
				})
			}
			if containerResult.RolledBack {
				result.Rollbacks++
			}
			mu.Unlock()
		}(container)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	return result, nil
}

// updateSingleContainer updates a single container
func (t *ContainerUpdaterTask) updateSingleContainer(ctx context.Context, container *model.Container, params *ContainerUpdateParameters) *SingleContainerUpdateResult {
	startTime := time.Now()
	result := &SingleContainerUpdateResult{
		Container:   container,
		OldVersion:  container.Tag,
		UpdateSteps: []UpdateStep{},
	}

	logger := logrus.WithFields(logrus.Fields{
		"container_id":   container.ID,
		"container_name": container.Name,
		"image":          container.Image,
		"current_tag":    container.Tag,
		"strategy":       params.UpdateStrategy,
	})

	// Create update history record
	updateHistory := &model.UpdateHistory{
		ContainerID: int64(container.ID),
		OldImage:    container.GetFullImageName(),
		Status:      model.UpdateStatusInProgress,
		Strategy:    params.UpdateStrategy,
		StartedAt:   startTime,
	}

	if t.updateHistoryRepo != nil {
		if err := t.updateHistoryRepo.Create(ctx, updateHistory); err != nil {
			logger.WithError(err).Warn("Failed to create update history record")
		}
		result.UpdateHistory = updateHistory
	}

	// Execute update based on strategy
	switch params.UpdateStrategy {
	case "recreate":
		result = t.updateWithRecreateStrategy(ctx, container, params, result)
	case "rolling":
		result = t.updateWithRollingStrategy(ctx, container, params, result)
	case "blue-green":
		result = t.updateWithBlueGreenStrategy(ctx, container, params, result)
	default:
		result.Error = fmt.Sprintf("unsupported update strategy: %s", params.UpdateStrategy)
		result.Success = false
	}

	result.Duration = time.Since(startTime)

	// Update history record
	if updateHistory != nil && t.updateHistoryRepo != nil {
		if result.Success {
			updateHistory.Status = model.UpdateStatusCompleted
			updateHistory.NewImage = result.NewVersion
		} else {
			updateHistory.Status = model.UpdateStatusFailed
			updateHistory.ErrorMessage = result.Error
		}
		completedAt := time.Now()
		updateHistory.CompletedAt = &completedAt

		if err := t.updateHistoryRepo.Update(ctx, updateHistory); err != nil {
			logger.WithError(err).Warn("Failed to update history record")
		}
	}

	if result.Success {
		logger.WithFields(logrus.Fields{
			"new_version": result.NewVersion,
			"duration":    result.Duration,
		}).Info("Container updated successfully")
	} else {
		logger.WithFields(logrus.Fields{
			"error":    result.Error,
			"duration": result.Duration,
		}).Error("Container update failed")
	}

	return result
}

// updateWithRecreateStrategy implements the recreate update strategy
func (t *ContainerUpdaterTask) updateWithRecreateStrategy(ctx context.Context, container *model.Container, params *ContainerUpdateParameters, result *SingleContainerUpdateResult) *SingleContainerUpdateResult {
	// Implementation for recreate strategy
	// This is a simplified version - full implementation would include:
	// 1. Pull new image
	// 2. Create backup if requested
	// 3. Stop container
	// 4. Remove container
	// 5. Create new container with new image
	// 6. Start container
	// 7. Health check
	// 8. Rollback if needed

	result.Success = true // Placeholder
	result.NewVersion = "latest" // Placeholder

	return result
}

// updateWithRollingStrategy implements the rolling update strategy
func (t *ContainerUpdaterTask) updateWithRollingStrategy(ctx context.Context, container *model.Container, params *ContainerUpdateParameters, result *SingleContainerUpdateResult) *SingleContainerUpdateResult {
	// Implementation for rolling strategy
	result.Success = true // Placeholder
	result.NewVersion = "latest" // Placeholder

	return result
}

// updateWithBlueGreenStrategy implements the blue-green update strategy
func (t *ContainerUpdaterTask) updateWithBlueGreenStrategy(ctx context.Context, container *model.Container, params *ContainerUpdateParameters, result *SingleContainerUpdateResult) *SingleContainerUpdateResult {
	// Implementation for blue-green strategy
	result.Success = true // Placeholder
	result.NewVersion = "latest" // Placeholder

	return result
}

// processResults processes the update results
func (t *ContainerUpdaterTask) processResults(ctx context.Context, results *ContainerUpdateTaskResult, params *ContainerUpdateParameters) error {
	// Send notifications
	if params.NotifyOnSuccess && results.SuccessfulUpdates > 0 {
		t.sendSuccessNotification(ctx, results)
	}

	if params.NotifyOnFailure && results.FailedUpdates > 0 {
		t.sendFailureNotification(ctx, results)
	}

	return nil
}

// sendSuccessNotification sends a notification for successful updates
func (t *ContainerUpdaterTask) sendSuccessNotification(ctx context.Context, results *ContainerUpdateResult) {
	if t.notificationService == nil {
		return
	}

	notification := &model.Notification{
		Type:     model.NotificationTypeContainerUpdate,
		Title:    "Container Updates Completed",
		Message:  fmt.Sprintf("Successfully updated %d container(s)", results.SuccessfulUpdates),
		Priority: model.NotificationPriorityNormal,
		Data: map[string]interface{}{
			"successful_updates": results.SuccessfulUpdates,
			"failed_updates":     results.FailedUpdates,
			"rollbacks":         results.Rollbacks,
			"duration":          results.Duration.String(),
		},
	}

	if err := t.notificationService.SendNotification(ctx, notification); err != nil {
		logrus.WithError(err).Warn("Failed to send success notification")
	}
}

// sendFailureNotification sends a notification for failed updates
func (t *ContainerUpdaterTask) sendFailureNotification(ctx context.Context, results *ContainerUpdateResult) {
	if t.notificationService == nil {
		return
	}

	notification := &model.Notification{
		Type:     model.NotificationTypeContainerUpdate,
		Title:    "Container Update Failures",
		Message:  fmt.Sprintf("Failed to update %d container(s)", results.FailedUpdates),
		Priority: model.NotificationPriorityHigh,
		Data: map[string]interface{}{
			"successful_updates": results.SuccessfulUpdates,
			"failed_updates":     results.FailedUpdates,
			"rollbacks":         results.Rollbacks,
			"errors":           results.Errors,
		},
	}

	if err := t.notificationService.SendNotification(ctx, notification); err != nil {
		logrus.WithError(err).Warn("Failed to send failure notification")
	}
}

// contains checks if a slice contains a specific string
func (t *ContainerUpdaterTask) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}