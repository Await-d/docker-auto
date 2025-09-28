package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// UpdateActivityService manages container update detection, tracking, and execution
type UpdateActivityService struct {
	db            *gorm.DB
	dockerClient  DockerServiceInterface
	logger        *logrus.Logger
	mutex         sync.RWMutex
	updateCheckers map[string]*UpdateChecker
}

// UpdateChecker tracks update information for a specific container
type UpdateChecker struct {
	ContainerID     string
	ImageName       string
	CurrentTag      string
	LatestTag       string
	LastChecked     time.Time
	UpdateAvailable bool
	UpdateStrategy  UpdateStrategy
}

// UpdateStrategy defines how updates should be handled
type UpdateStrategy struct {
	AutoUpdate          bool          `json:"autoUpdate"`
	UpdateWindow        *TimeWindow   `json:"updateWindow,omitempty"`
	RollbackOnFailure   bool          `json:"rollbackOnFailure"`
	HealthCheckTimeout  time.Duration `json:"healthCheckTimeout"`
	PreUpdateCommands   []string      `json:"preUpdateCommands,omitempty"`
	PostUpdateCommands  []string      `json:"postUpdateCommands,omitempty"`
}

// TimeWindow defines when updates can occur
type TimeWindow struct {
	StartHour int `json:"startHour"` // 0-23
	EndHour   int `json:"endHour"`   // 0-23
	Days      []int `json:"days"`     // 0-6, 0=Sunday
}

// UpdateActivity represents update activity data
type UpdateActivity struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	ContainerID     string    `json:"containerId" gorm:"index"`
	ContainerName   string    `json:"containerName"`
	ImageName       string    `json:"imageName"`
	FromTag         string    `json:"fromTag"`
	ToTag           string    `json:"toTag"`
	Status          string    `json:"status"` // pending, in_progress, completed, failed, rolled_back
	Strategy        UpdateStrategy `json:"strategy" gorm:"type:text"`
	StartTime       *time.Time `json:"startTime"`
	EndTime         *time.Time `json:"endTime"`
	ErrorMessage    string    `json:"errorMessage,omitempty"`
	LogEntries      []UpdateLogEntry `json:"logEntries,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// UpdateLogEntry represents a log entry during update process
type UpdateLogEntry struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ActivityID uint      `json:"activityId" gorm:"index"`
	Timestamp  time.Time `json:"timestamp"`
	Level      string    `json:"level"` // info, warn, error
	Message    string    `json:"message"`
	Step       string    `json:"step"`  // pre_check, pull, stop, start, health_check, rollback
}

// UpdateSummary provides aggregate update information
type UpdateSummary struct {
	TotalContainers     int                    `json:"totalContainers"`
	PendingUpdates      int                    `json:"pendingUpdates"`
	RecentlyUpdated     int                    `json:"recentlyUpdated"`
	FailedUpdates       int                    `json:"failedUpdates"`
	AutoUpdateEnabled   int                    `json:"autoUpdateEnabled"`
	LastUpdateCheck     time.Time              `json:"lastUpdateCheck"`
	UpdateActivities    []UpdateActivity       `json:"updateActivities"`
	AvailableUpdates    []AvailableUpdate      `json:"availableUpdates"`
}

// AvailableUpdate represents an available update
type AvailableUpdate struct {
	ContainerID     string    `json:"containerId"`
	ContainerName   string    `json:"containerName"`
	ImageName       string    `json:"imageName"`
	CurrentTag      string    `json:"currentTag"`
	LatestTag       string    `json:"latestTag"`
	Size            int64     `json:"size"`
	LastChecked     time.Time `json:"lastChecked"`
	AutoUpdate      bool      `json:"autoUpdate"`
	Severity        string    `json:"severity"` // low, medium, high, critical
}

// NewUpdateActivityService creates a new update activity service
func NewUpdateActivityService(
	db *gorm.DB,
	dockerClient DockerServiceInterface,
	logger *logrus.Logger,
) *UpdateActivityService {
	return &UpdateActivityService{
		db:             db,
		dockerClient:   dockerClient,
		logger:         logger,
		updateCheckers: make(map[string]*UpdateChecker),
	}
}

// Initialize sets up the update activity service
func (uas *UpdateActivityService) Initialize(ctx context.Context) error {
	// Migrate database tables
	if err := uas.db.AutoMigrate(&UpdateActivity{}, &UpdateLogEntry{}); err != nil {
		return fmt.Errorf("failed to migrate update activity tables: %w", err)
	}

	// Initialize update checkers for existing containers
	if err := uas.initializeUpdateCheckers(ctx); err != nil {
		uas.logger.WithError(err).Warn("Failed to initialize update checkers")
	}

	return nil
}

// CheckForUpdates checks all containers for available updates
func (uas *UpdateActivityService) CheckForUpdates(ctx context.Context) error {
	uas.logger.Info("Starting update check for all containers")

	containers, err := uas.dockerClient.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent checks

	for _, container := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := uas.checkContainerForUpdates(ctx, containerID); err != nil {
				uas.logger.WithError(err).WithField("containerId", containerID).Error("Failed to check container for updates")
			}
		}(container.ID)
	}

	wg.Wait()
	uas.logger.Info("Completed update check for all containers")

	return nil
}

// checkContainerForUpdates checks a specific container for updates
func (uas *UpdateActivityService) checkContainerForUpdates(ctx context.Context, containerID string) error {
	container, err := uas.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}

	uas.mutex.Lock()
	checker, exists := uas.updateCheckers[containerID]
	if !exists {
		checker = &UpdateChecker{
			ContainerID: containerID,
			ImageName:   container.Image,
		}
		uas.updateCheckers[containerID] = checker
	}
	uas.mutex.Unlock()

	// Extract current tag
	currentTag := uas.extractTagFromImage(container.Image)
	checker.CurrentTag = currentTag

	// Check for latest tag
	latestTag, err := uas.dockerClient.GetLatestImageTag(ctx, container.Image)
	if err != nil {
		uas.logger.WithError(err).WithField("image", container.Image).Warn("Failed to get latest image tag")
		return err
	}

	checker.LatestTag = latestTag
	checker.LastChecked = time.Now()
	checker.UpdateAvailable = currentTag != latestTag && latestTag != ""

	// If update is available, create or update activity record
	if checker.UpdateAvailable {
		if err := uas.createOrUpdateActivityRecord(ctx, checker, container.Name); err != nil {
			uas.logger.WithError(err).Error("Failed to create activity record")
		}
	}

	return nil
}

// createOrUpdateActivityRecord creates or updates an update activity record
func (uas *UpdateActivityService) createOrUpdateActivityRecord(ctx context.Context, checker *UpdateChecker, containerName string) error {
	var activity UpdateActivity

	// Check if pending activity already exists
	err := uas.db.Where("container_id = ? AND status = ?", checker.ContainerID, "pending").First(&activity).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if err == gorm.ErrRecordNotFound {
		// Create new activity
		activity = UpdateActivity{
			ContainerID:   checker.ContainerID,
			ContainerName: containerName,
			ImageName:     checker.ImageName,
			FromTag:       checker.CurrentTag,
			ToTag:         checker.LatestTag,
			Status:        "pending",
			Strategy:      checker.UpdateStrategy,
		}

		if err := uas.db.Create(&activity).Error; err != nil {
			return fmt.Errorf("failed to create update activity: %w", err)
		}

		uas.logger.WithFields(logrus.Fields{
			"containerId":   checker.ContainerID,
			"containerName": containerName,
			"fromTag":       checker.CurrentTag,
			"toTag":         checker.LatestTag,
		}).Info("Created new update activity")
	} else {
		// Update existing activity
		activity.ToTag = checker.LatestTag
		if err := uas.db.Save(&activity).Error; err != nil {
			return fmt.Errorf("failed to update activity: %w", err)
		}
	}

	return nil
}

// GetUpdateSummary returns a summary of update activities
func (uas *UpdateActivityService) GetUpdateSummary(ctx context.Context) (*UpdateSummary, error) {
	summary := &UpdateSummary{
		LastUpdateCheck: time.Now(),
	}

	// Count total containers
	containers, err := uas.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	summary.TotalContainers = len(containers)

	// Count pending updates
	var pendingCount int64
	if err := uas.db.Model(&UpdateActivity{}).Where("status = ?", "pending").Count(&pendingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count pending updates: %w", err)
	}
	summary.PendingUpdates = int(pendingCount)

	// Count recently updated (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	var recentCount int64
	if err := uas.db.Model(&UpdateActivity{}).Where("status = ? AND updated_at > ?", "completed", since).Count(&recentCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count recent updates: %w", err)
	}
	summary.RecentlyUpdated = int(recentCount)

	// Count failed updates (last 7 days)
	failedSince := time.Now().Add(-7 * 24 * time.Hour)
	var failedCount int64
	if err := uas.db.Model(&UpdateActivity{}).Where("status IN ? AND updated_at > ?", []string{"failed", "rolled_back"}, failedSince).Count(&failedCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count failed updates: %w", err)
	}
	summary.FailedUpdates = int(failedCount)

	// Get recent update activities
	var activities []UpdateActivity
	if err := uas.db.Order("updated_at DESC").Limit(10).Preload("LogEntries").Find(&activities).Error; err != nil {
		return nil, fmt.Errorf("failed to get update activities: %w", err)
	}
	summary.UpdateActivities = activities

	// Get available updates
	availableUpdates := uas.getAvailableUpdates()
	summary.AvailableUpdates = availableUpdates

	// Count auto-update enabled containers
	autoUpdateCount := 0
	for _, update := range availableUpdates {
		if update.AutoUpdate {
			autoUpdateCount++
		}
	}
	summary.AutoUpdateEnabled = autoUpdateCount

	return summary, nil
}

// getAvailableUpdates returns list of available updates
func (uas *UpdateActivityService) getAvailableUpdates() []AvailableUpdate {
	uas.mutex.RLock()
	defer uas.mutex.RUnlock()

	var updates []AvailableUpdate
	for _, checker := range uas.updateCheckers {
		if checker.UpdateAvailable {
			// Get container name
			containerName := uas.getContainerName(checker.ContainerID)

			update := AvailableUpdate{
				ContainerID:   checker.ContainerID,
				ContainerName: containerName,
				ImageName:     checker.ImageName,
				CurrentTag:    checker.CurrentTag,
				LatestTag:     checker.LatestTag,
				LastChecked:   checker.LastChecked,
				AutoUpdate:    checker.UpdateStrategy.AutoUpdate,
				Severity:      uas.calculateUpdateSeverity(checker.CurrentTag, checker.LatestTag),
			}
			updates = append(updates, update)
		}
	}

	// Sort by severity and last checked
	sort.Slice(updates, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		if severityOrder[updates[i].Severity] != severityOrder[updates[j].Severity] {
			return severityOrder[updates[i].Severity] > severityOrder[updates[j].Severity]
		}
		return updates[i].LastChecked.After(updates[j].LastChecked)
	})

	return updates
}

// TriggerUpdate manually triggers an update for a specific container
func (uas *UpdateActivityService) TriggerUpdate(ctx context.Context, containerID string, strategy *UpdateStrategy) error {
	uas.logger.WithField("containerId", containerID).Info("Triggering manual update")

	// Get or create activity record
	var activity UpdateActivity
	err := uas.db.Where("container_id = ? AND status = ?", containerID, "pending").First(&activity).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to get update activity: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		return fmt.Errorf("no pending update found for container %s", containerID)
	}

	// Update strategy if provided
	if strategy != nil {
		activity.Strategy = *strategy
	}

	// Execute update
	return uas.executeUpdate(ctx, &activity)
}

// executeUpdate executes an update for the given activity
func (uas *UpdateActivityService) executeUpdate(ctx context.Context, activity *UpdateActivity) error {
	// Update status to in_progress
	activity.Status = "in_progress"
	now := time.Now()
	activity.StartTime = &now

	if err := uas.db.Save(activity).Error; err != nil {
		return fmt.Errorf("failed to update activity status: %w", err)
	}

	uas.addLogEntry(activity.ID, "info", "pre_check", "Starting update process")

	// Execute pre-update commands
	if len(activity.Strategy.PreUpdateCommands) > 0 {
		uas.addLogEntry(activity.ID, "info", "pre_check", "Executing pre-update commands")
		for _, cmd := range activity.Strategy.PreUpdateCommands {
			if err := uas.dockerClient.ExecCommand(ctx, activity.ContainerID, cmd); err != nil {
				uas.addLogEntry(activity.ID, "error", "pre_check", fmt.Sprintf("Pre-update command failed: %s", err.Error()))
				return uas.markUpdateFailed(activity, err)
			}
		}
	}

	// Pull new image
	uas.addLogEntry(activity.ID, "info", "pull", fmt.Sprintf("Pulling image %s:%s", activity.ImageName, activity.ToTag))
	if err := uas.dockerClient.PullImage(ctx, fmt.Sprintf("%s:%s", activity.ImageName, activity.ToTag)); err != nil {
		uas.addLogEntry(activity.ID, "error", "pull", fmt.Sprintf("Failed to pull image: %s", err.Error()))
		return uas.markUpdateFailed(activity, err)
	}

	// Store current container config for rollback
	containerConfig, err := uas.dockerClient.GetContainer(ctx, activity.ContainerID)
	if err != nil {
		uas.addLogEntry(activity.ID, "error", "pre_check", fmt.Sprintf("Failed to get container config: %s", err.Error()))
		return uas.markUpdateFailed(activity, err)
	}

	// Stop container
	uas.addLogEntry(activity.ID, "info", "stop", "Stopping container")
	if err := uas.dockerClient.StopContainer(ctx, activity.ContainerID, 30*time.Second); err != nil {
		uas.addLogEntry(activity.ID, "error", "stop", fmt.Sprintf("Failed to stop container: %s", err.Error()))
		return uas.markUpdateFailed(activity, err)
	}

	// Update container with new image
	uas.addLogEntry(activity.ID, "info", "start", "Starting container with new image")
	if err := uas.dockerClient.UpdateContainer(ctx, activity.ContainerID, fmt.Sprintf("%s:%s", activity.ImageName, activity.ToTag)); err != nil {
		uas.addLogEntry(activity.ID, "error", "start", fmt.Sprintf("Failed to update container: %s", err.Error()))

		// Attempt rollback
		if activity.Strategy.RollbackOnFailure {
			uas.rollbackUpdate(ctx, activity, containerConfig)
		}

		return uas.markUpdateFailed(activity, err)
	}

	// Health check
	if err := uas.performHealthCheck(ctx, activity); err != nil {
		if activity.Strategy.RollbackOnFailure {
			uas.rollbackUpdate(ctx, activity, containerConfig)
		}
		return uas.markUpdateFailed(activity, err)
	}

	// Execute post-update commands
	if len(activity.Strategy.PostUpdateCommands) > 0 {
		uas.addLogEntry(activity.ID, "info", "post_check", "Executing post-update commands")
		for _, cmd := range activity.Strategy.PostUpdateCommands {
			if err := uas.dockerClient.ExecCommand(ctx, activity.ContainerID, cmd); err != nil {
				uas.addLogEntry(activity.ID, "warn", "post_check", fmt.Sprintf("Post-update command failed: %s", err.Error()))
				// Don't fail the update for post-update command failures
			}
		}
	}

	// Mark update as completed
	activity.Status = "completed"
	endTime := time.Now()
	activity.EndTime = &endTime

	if err := uas.db.Save(activity).Error; err != nil {
		uas.logger.WithError(err).Error("Failed to mark update as completed")
	}

	uas.addLogEntry(activity.ID, "info", "completed", "Update completed successfully")
	uas.logger.WithFields(logrus.Fields{
		"containerId":   activity.ContainerID,
		"containerName": activity.ContainerName,
		"fromTag":       activity.FromTag,
		"toTag":         activity.ToTag,
	}).Info("Container update completed successfully")

	return nil
}

// performHealthCheck performs health check on updated container
func (uas *UpdateActivityService) performHealthCheck(ctx context.Context, activity *UpdateActivity) error {
	uas.addLogEntry(activity.ID, "info", "health_check", "Performing health check")

	timeout := activity.Strategy.HealthCheckTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check if container is running
	for {
		select {
		case <-checkCtx.Done():
			return fmt.Errorf("health check timeout after %v", timeout)
		default:
			container, err := uas.dockerClient.GetContainer(checkCtx, activity.ContainerID)
			if err != nil {
				return fmt.Errorf("failed to get container status: %w", err)
			}

			if container.Status == "running" {
				// Additional health checks can be added here
				uas.addLogEntry(activity.ID, "info", "health_check", "Container is healthy")
				return nil
			}

			time.Sleep(2 * time.Second)
		}
	}
}

// rollbackUpdate attempts to rollback a failed update
func (uas *UpdateActivityService) rollbackUpdate(ctx context.Context, activity *UpdateActivity, originalConfig interface{}) {
	uas.addLogEntry(activity.ID, "info", "rollback", "Starting rollback process")

	// Stop current container
	if err := uas.dockerClient.StopContainer(ctx, activity.ContainerID, 30*time.Second); err != nil {
		uas.addLogEntry(activity.ID, "error", "rollback", fmt.Sprintf("Failed to stop container for rollback: %s", err.Error()))
		return
	}

	// Restore original image
	originalImage := fmt.Sprintf("%s:%s", activity.ImageName, activity.FromTag)
	if err := uas.dockerClient.UpdateContainer(ctx, activity.ContainerID, originalImage); err != nil {
		uas.addLogEntry(activity.ID, "error", "rollback", fmt.Sprintf("Failed to rollback container: %s", err.Error()))
		return
	}

	activity.Status = "rolled_back"
	if err := uas.db.Save(activity).Error; err != nil {
		uas.logger.WithError(err).Error("Failed to update rollback status")
	}

	uas.addLogEntry(activity.ID, "info", "rollback", "Rollback completed successfully")
}

// markUpdateFailed marks an update as failed
func (uas *UpdateActivityService) markUpdateFailed(activity *UpdateActivity, err error) error {
	activity.Status = "failed"
	activity.ErrorMessage = err.Error()
	endTime := time.Now()
	activity.EndTime = &endTime

	if dbErr := uas.db.Save(activity).Error; dbErr != nil {
		uas.logger.WithError(dbErr).Error("Failed to mark update as failed")
	}

	uas.addLogEntry(activity.ID, "error", "failed", err.Error())

	return err
}

// addLogEntry adds a log entry to an update activity
func (uas *UpdateActivityService) addLogEntry(activityID uint, level, step, message string) {
	entry := UpdateLogEntry{
		ActivityID: activityID,
		Timestamp:  time.Now(),
		Level:      level,
		Step:       step,
		Message:    message,
	}

	if err := uas.db.Create(&entry).Error; err != nil {
		uas.logger.WithError(err).Error("Failed to create log entry")
	}
}

// Helper functions
func (uas *UpdateActivityService) initializeUpdateCheckers(ctx context.Context) error {
	containers, err := uas.dockerClient.ListContainers(ctx)
	if err != nil {
		return err
	}

	for _, container := range containers {
		checker := &UpdateChecker{
			ContainerID: container.ID,
			ImageName:   container.Image,
			CurrentTag:  uas.extractTagFromImage(container.Image),
		}
		uas.updateCheckers[container.ID] = checker
	}

	return nil
}

func (uas *UpdateActivityService) extractTagFromImage(image string) string {
	// Extract tag from image name (e.g., nginx:1.20 -> 1.20)
	parts := strings.Split(image, ":")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "latest"
}

func (uas *UpdateActivityService) getContainerName(containerID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	container, err := uas.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return containerID
	}

	if len(container.Names) > 0 {
		return strings.TrimPrefix(container.Names[0], "/")
	}

	return container.ID[:12]
}

func (uas *UpdateActivityService) calculateUpdateSeverity(currentTag, latestTag string) string {
	// Simple severity calculation based on tag patterns
	// This can be enhanced with more sophisticated version parsing

	if strings.Contains(latestTag, "security") || strings.Contains(latestTag, "critical") {
		return "critical"
	}

	// Major version change
	currentMajor := strings.Split(currentTag, ".")[0]
	latestMajor := strings.Split(latestTag, ".")[0]

	if currentMajor != latestMajor {
		return "high"
	}

	// Minor version change
	if strings.Contains(currentTag, ".") && strings.Contains(latestTag, ".") {
		return "medium"
	}

	return "low"
}