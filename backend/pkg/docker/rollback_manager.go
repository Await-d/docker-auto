package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"docker-auto/pkg/security"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
)

// RollbackManager handles container rollback operations
type RollbackManager struct {
	clientManager       *ClientManager
	notificationManager *NotificationManager
	rollbackHistory     map[string]*RollbackEntry
	mutex               sync.RWMutex
	logger              *logrus.Logger
	config              *RollbackConfig
}

// RollbackConfig represents rollback configuration
type RollbackConfig struct {
	AutoRollbackEnabled   bool          `json:"auto_rollback_enabled"`
	RollbackTimeout       time.Duration `json:"rollback_timeout"`
	HealthCheckRetries    int           `json:"health_check_retries"`
	HealthCheckInterval   time.Duration `json:"health_check_interval"`
	MaxRollbackHistory    int           `json:"max_rollback_history"`
	BackupRetentionPeriod time.Duration `json:"backup_retention_period"`
	FailureThreshold      int           `json:"failure_threshold"`
}

// RollbackEntry represents a rollback history entry
type RollbackEntry struct {
	ID                string                 `json:"id"`
	ContainerID       string                 `json:"container_id"`
	OriginalImage     string                 `json:"original_image"`
	TargetImage       string                 `json:"target_image"`
	BackupImage       string                 `json:"backup_image,omitempty"`
	BackupContainer   string                 `json:"backup_container,omitempty"`
	RollbackReason    string                 `json:"rollback_reason"`
	RollbackTrigger   RollbackTrigger        `json:"rollback_trigger"`
	Status            RollbackStatus         `json:"status"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           *time.Time             `json:"end_time,omitempty"`
	Duration          time.Duration          `json:"duration"`
	Steps             []RollbackStep         `json:"steps"`
	OriginalConfig    *ContainerSnapshot     `json:"original_config"`
	UserID            int64                  `json:"user_id"`
	Error             string                 `json:"error,omitempty"`
	HealthCheckResult *HealthCheckResult     `json:"health_check_result,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// ContainerSnapshot represents a snapshot of container configuration
type ContainerSnapshot struct {
	Image        string                 `json:"image"`
	Config       *container.Config      `json:"config"`
	HostConfig   *container.HostConfig  `json:"host_config"`
	NetworkConfig map[string]interface{} `json:"network_config"`
	Mounts       []types.MountPoint     `json:"mounts"`
	Environment  []string               `json:"environment"`
	Labels       map[string]string      `json:"labels"`
	CreatedAt    time.Time              `json:"created_at"`
	State        *types.ContainerState  `json:"state"`
}

// RollbackStep represents a single rollback step
type RollbackStep struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
}

// HealthCheckResult represents health check results
type HealthCheckResult struct {
	Healthy           bool          `json:"healthy"`
	ChecksPerformed   int           `json:"checks_performed"`
	FailedChecks      int           `json:"failed_checks"`
	LastCheckTime     time.Time     `json:"last_check_time"`
	ResponseTime      time.Duration `json:"response_time"`
	Error             string        `json:"error,omitempty"`
	EndpointResults   []EndpointCheck `json:"endpoint_results,omitempty"`
}

// EndpointCheck represents a health check for a specific endpoint
type EndpointCheck struct {
	URL          string        `json:"url"`
	Method       string        `json:"method"`
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
}

// RollbackTrigger represents what triggered the rollback
type RollbackTrigger string

const (
	RollbackTriggerManual       RollbackTrigger = "manual"
	RollbackTriggerAutoFailure  RollbackTrigger = "auto_failure"
	RollbackTriggerHealthCheck  RollbackTrigger = "health_check"
	RollbackTriggerTimeout      RollbackTrigger = "timeout"
	RollbackTriggerError        RollbackTrigger = "error"
)

// RollbackStatus represents rollback status
type RollbackStatus string

const (
	RollbackStatusPending    RollbackStatus = "pending"
	RollbackStatusRunning    RollbackStatus = "running"
	RollbackStatusCompleted  RollbackStatus = "completed"
	RollbackStatusFailed     RollbackStatus = "failed"
	RollbackStatusCancelled  RollbackStatus = "cancelled"
)

// RollbackRequest represents a rollback request
type RollbackRequest struct {
	ContainerID     string                 `json:"container_id"`
	TargetImage     string                 `json:"target_image,omitempty"`
	Reason          string                 `json:"reason"`
	Force           bool                   `json:"force"`
	HealthCheck     bool                   `json:"health_check"`
	Timeout         time.Duration          `json:"timeout"`
	UserContext     *security.DockerUserContext `json:"user_context"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	Success         bool           `json:"success"`
	RollbackEntry   *RollbackEntry `json:"rollback_entry"`
	NewContainerID  string         `json:"new_container_id,omitempty"`
	Error           string         `json:"error,omitempty"`
	Duration        time.Duration  `json:"duration"`
	StepsCompleted  int            `json:"steps_completed"`
	TotalSteps      int            `json:"total_steps"`
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(clientManager *ClientManager, notificationManager *NotificationManager, config *RollbackConfig, logger *logrus.Logger) *RollbackManager {
	if config == nil {
		config = DefaultRollbackConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	rm := &RollbackManager{
		clientManager:       clientManager,
		notificationManager: notificationManager,
		rollbackHistory:     make(map[string]*RollbackEntry),
		logger:              logger,
		config:              config,
	}

	// Start cleanup routine
	go rm.cleanupRoutine()

	return rm
}

// DefaultRollbackConfig returns default rollback configuration
func DefaultRollbackConfig() *RollbackConfig {
	return &RollbackConfig{
		AutoRollbackEnabled:   true,
		RollbackTimeout:       10 * time.Minute,
		HealthCheckRetries:    3,
		HealthCheckInterval:   30 * time.Second,
		MaxRollbackHistory:    100,
		BackupRetentionPeriod: 7 * 24 * time.Hour,
		FailureThreshold:      3,
	}
}

// CreateSnapshot creates a snapshot of container configuration
func (rm *RollbackManager) CreateSnapshot(ctx context.Context, containerID string) (*ContainerSnapshot, error) {
	client := rm.clientManager.GetClient()

	// Get container information
	containerInfo, err := client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	snapshot := &ContainerSnapshot{
		Image:        containerInfo.Config.Image,
		Config:       containerInfo.Config,
		HostConfig:   containerInfo.HostConfig,
		Mounts:       containerInfo.Mounts,
		Environment:  containerInfo.Config.Env,
		Labels:       containerInfo.Config.Labels,
		CreatedAt:    time.Now(),
		State:        containerInfo.State,
	}

	// Get network configuration
	if containerInfo.NetworkSettings != nil {
		networkData, _ := json.Marshal(containerInfo.NetworkSettings)
		var networkConfig map[string]interface{}
		json.Unmarshal(networkData, &networkConfig)
		snapshot.NetworkConfig = networkConfig
	}

	rm.logger.WithFields(logrus.Fields{
		"container_id": containerID,
		"image":        snapshot.Image,
	}).Info("Container snapshot created")

	return snapshot, nil
}

// PrepareRollback prepares a rollback operation
func (rm *RollbackManager) PrepareRollback(ctx context.Context, request *RollbackRequest) (*RollbackEntry, error) {
	// Create snapshot of current state
	snapshot, err := rm.CreateSnapshot(ctx, request.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Determine target image
	targetImage := request.TargetImage
	if targetImage == "" {
		// Get the most recent backup or previous image
		targetImage, err = rm.findPreviousImage(ctx, request.ContainerID)
		if err != nil {
			return nil, fmt.Errorf("failed to find previous image: %w", err)
		}
	}

	// Create rollback entry
	rollbackID := fmt.Sprintf("rollback_%s_%d", request.ContainerID[:8], time.Now().Unix())
	entry := &RollbackEntry{
		ID:              rollbackID,
		ContainerID:     request.ContainerID,
		OriginalImage:   snapshot.Image,
		TargetImage:     targetImage,
		RollbackReason:  request.Reason,
		RollbackTrigger: RollbackTriggerManual,
		Status:          RollbackStatusPending,
		StartTime:       time.Now(),
		OriginalConfig:  snapshot,
		UserID:          request.UserContext.UserID,
		Metadata:        request.Metadata,
		Steps:           []RollbackStep{},
	}

	// Store rollback entry
	rm.mutex.Lock()
	rm.rollbackHistory[rollbackID] = entry
	rm.mutex.Unlock()

	rm.logger.WithFields(logrus.Fields{
		"rollback_id":   rollbackID,
		"container_id":  request.ContainerID,
		"target_image":  targetImage,
		"trigger":       entry.RollbackTrigger,
	}).Info("Rollback prepared")

	return entry, nil
}

// ExecuteRollback executes a rollback operation
func (rm *RollbackManager) ExecuteRollback(ctx context.Context, rollbackID string) (*RollbackResult, error) {
	rm.mutex.Lock()
	entry, exists := rm.rollbackHistory[rollbackID]
	if !exists {
		rm.mutex.Unlock()
		return nil, fmt.Errorf("rollback entry not found: %s", rollbackID)
	}
	entry.Status = RollbackStatusRunning
	rm.mutex.Unlock()

	// Start operation tracking
	if rm.notificationManager != nil {
		rm.notificationManager.StartOperation(rollbackID, "container_rollback", entry.UserID, map[string]interface{}{
			"container_id": entry.ContainerID,
			"target_image": entry.TargetImage,
			"reason":       entry.RollbackReason,
		})
	}

	result := &RollbackResult{
		RollbackEntry: entry,
	}

	var newContainerID string
	var err error

	// Execute rollback steps
	err = rm.executeRollbackSteps(ctx, entry, &newContainerID)

	// Complete the operation
	endTime := time.Now()
	entry.EndTime = &endTime
	entry.Duration = endTime.Sub(entry.StartTime)

	if err != nil {
		entry.Status = RollbackStatusFailed
		entry.Error = err.Error()
		result.Error = err.Error()

		if rm.notificationManager != nil {
			rm.notificationManager.CompleteOperation(rollbackID, false, err.Error())
		}

		rm.logger.WithError(err).WithFields(logrus.Fields{
			"rollback_id":  rollbackID,
			"container_id": entry.ContainerID,
		}).Error("Rollback failed")
	} else {
		entry.Status = RollbackStatusCompleted
		result.Success = true
		result.NewContainerID = newContainerID

		if rm.notificationManager != nil {
			rm.notificationManager.CompleteOperation(rollbackID, true, "")
		}

		rm.logger.WithFields(logrus.Fields{
			"rollback_id":      rollbackID,
			"container_id":     entry.ContainerID,
			"new_container_id": newContainerID,
			"duration":         entry.Duration,
		}).Info("Rollback completed successfully")
	}

	result.Duration = entry.Duration
	result.StepsCompleted = rm.countCompletedSteps(entry)
	result.TotalSteps = len(entry.Steps)

	return result, err
}

// executeRollbackSteps executes all rollback steps
func (rm *RollbackManager) executeRollbackSteps(ctx context.Context, entry *RollbackEntry, newContainerID *string) error {
	client := rm.clientManager.GetClient()

	// Step 1: Stop current container
	step := rm.addStep(entry, "stop_container", "Stopping current container")
	if err := rm.executeWithRetry(ctx, step, func() error {
		return client.ContainerStop(ctx, entry.ContainerID, container.StopOptions{
			Timeout: func() *int { t := 30; return &t }(),
		})
	}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	rm.completeStep(step, true, "")

	// Step 2: Create backup of current container (if needed)
	if entry.BackupContainer == "" {
		step = rm.addStep(entry, "create_backup", "Creating backup of current container")
		backupName := fmt.Sprintf("%s_backup_%d", entry.ContainerID[:8], time.Now().Unix())

		if err := rm.executeWithRetry(ctx, step, func() error {
			commitOptions := types.ContainerCommitOptions{
				Reference: backupName,
				Comment:   fmt.Sprintf("Backup before rollback: %s", entry.RollbackReason),
				Author:    "rollback-manager",
			}
			_, err := client.ContainerCommit(ctx, entry.ContainerID, commitOptions)
			if err == nil {
				entry.BackupImage = backupName
			}
			return err
		}); err != nil {
			rm.logger.WithError(err).Warn("Failed to create backup, continuing rollback")
			rm.completeStep(step, false, err.Error())
		} else {
			rm.completeStep(step, true, "")
		}
	}

	// Step 3: Remove current container
	step = rm.addStep(entry, "remove_container", "Removing current container")
	if err := rm.executeWithRetry(ctx, step, func() error {
		return client.ContainerRemove(ctx, entry.ContainerID, types.ContainerRemoveOptions{
			Force: true,
		})
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	rm.completeStep(step, true, "")

	// Step 4: Pull target image if not available
	step = rm.addStep(entry, "pull_target_image", "Pulling target image")
	if _, _, err := client.ImageInspectWithRaw(ctx, entry.TargetImage); err != nil {
		// Image not found locally, try to pull
		if err := rm.executeWithRetry(ctx, step, func() error {
			pullProgress, err := rm.clientManager.PullImage(ctx, entry.TargetImage, "", nil, false, &security.DockerUserContext{
				UserID:   entry.UserID,
				Username: "rollback-system",
				Role:     "admin",
			})
			if err != nil {
				return err
			}

			// Wait for pull to complete
			for !pullProgress.Completed {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(1 * time.Second):
					// Continue waiting
				}
			}

			if pullProgress.Error != "" {
				return fmt.Errorf("image pull failed: %s", pullProgress.Error)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("failed to pull target image: %w", err)
		}
		rm.completeStep(step, true, "")
	} else {
		rm.completeStep(step, true, "Image already available locally")
	}

	// Step 5: Create new container with target image
	step = rm.addStep(entry, "create_new_container", "Creating new container with target image")
	var createResponse container.CreateResponse
	if err := rm.executeWithRetry(ctx, step, func() error {
		// Use original configuration but with target image
		config := *entry.OriginalConfig.Config
		config.Image = entry.TargetImage

		var err error
		createResponse, err = client.ContainerCreate(
			ctx,
			&config,
			entry.OriginalConfig.HostConfig,
			nil, // Network config will be handled separately
			nil,
			"", // Auto-generate name
		)
		if err == nil {
			*newContainerID = createResponse.ID
		}
		return err
	}); err != nil {
		return fmt.Errorf("failed to create new container: %w", err)
	}
	rm.completeStep(step, true, "")

	// Step 6: Start new container
	step = rm.addStep(entry, "start_new_container", "Starting new container")
	if err := rm.executeWithRetry(ctx, step, func() error {
		return client.ContainerStart(ctx, createResponse.ID, types.ContainerStartOptions{})
	}); err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}
	rm.completeStep(step, true, "")

	// Step 7: Perform health check
	step = rm.addStep(entry, "health_check", "Performing health check")
	healthResult, err := rm.performHealthCheck(ctx, createResponse.ID)
	entry.HealthCheckResult = healthResult

	if err != nil || !healthResult.Healthy {
		errorMsg := "Health check failed"
		if err != nil {
			errorMsg = err.Error()
		}
		rm.completeStep(step, false, errorMsg)
		return fmt.Errorf("health check failed: %s", errorMsg)
	}
	rm.completeStep(step, true, "")

	return nil
}

// executeWithRetry executes a function with retry logic
func (rm *RollbackManager) executeWithRetry(ctx context.Context, step *RollbackStep, fn func() error) error {
	maxRetries := step.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			step.Retries = attempt
			rm.logger.WithFields(logrus.Fields{
				"step":    step.Name,
				"attempt": attempt,
			}).Info("Retrying step")

			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if !rm.shouldRetryError(err) {
			break
		}
	}

	return lastErr
}

// shouldRetryError determines if an error should be retried
func (rm *RollbackManager) shouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"network is unreachable",
		"temporary failure",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errorStr), retryable) {
			return true
		}
	}

	return false
}

// performHealthCheck performs health check on container
func (rm *RollbackManager) performHealthCheck(ctx context.Context, containerID string) (*HealthCheckResult, error) {
	client := rm.clientManager.GetClient()
	result := &HealthCheckResult{
		ChecksPerformed: 0,
		FailedChecks:    0,
		LastCheckTime:   time.Now(),
	}

	checkCtx, cancel := context.WithTimeout(ctx, rm.config.RollbackTimeout)
	defer cancel()

	for attempt := 0; attempt < rm.config.HealthCheckRetries; attempt++ {
		result.ChecksPerformed++

		startTime := time.Now()
		containerInfo, err := client.ContainerInspect(checkCtx, containerID)
		result.ResponseTime = time.Since(startTime)
		result.LastCheckTime = time.Now()

		if err != nil {
			result.FailedChecks++
			result.Error = err.Error()
			rm.logger.WithError(err).WithFields(logrus.Fields{
				"container_id": containerID,
				"attempt":      attempt + 1,
			}).Warn("Health check failed")

			if attempt < rm.config.HealthCheckRetries-1 {
				select {
				case <-checkCtx.Done():
					return result, checkCtx.Err()
				case <-time.After(rm.config.HealthCheckInterval):
					continue
				}
			}
			continue
		}

		// Check if container is running
		if !containerInfo.State.Running {
			result.FailedChecks++
			result.Error = fmt.Sprintf("Container not running, state: %s", containerInfo.State.Status)

			if attempt < rm.config.HealthCheckRetries-1 {
				select {
				case <-checkCtx.Done():
					return result, checkCtx.Err()
				case <-time.After(rm.config.HealthCheckInterval):
					continue
				}
			}
			continue
		}

		// Check container health if health check is defined
		if containerInfo.State.Health != nil {
			switch containerInfo.State.Health.Status {
			case "healthy":
				result.Healthy = true
				return result, nil
			case "unhealthy":
				result.FailedChecks++
				result.Error = "Container health check reports unhealthy"
			default:
				// Still starting up, wait more
				if attempt < rm.config.HealthCheckRetries-1 {
					select {
					case <-checkCtx.Done():
						return result, checkCtx.Err()
					case <-time.After(rm.config.HealthCheckInterval):
						continue
					}
				}
			}
		} else {
			// No health check defined, assume healthy if running
			result.Healthy = true
			return result, nil
		}

		if attempt < rm.config.HealthCheckRetries-1 {
			select {
			case <-checkCtx.Done():
				return result, checkCtx.Err()
			case <-time.After(rm.config.HealthCheckInterval):
			}
		}
	}

	if !result.Healthy {
		return result, fmt.Errorf("health check failed after %d attempts", rm.config.HealthCheckRetries)
	}

	return result, nil
}

// addStep adds a new step to rollback entry
func (rm *RollbackManager) addStep(entry *RollbackEntry, name, description string) *RollbackStep {
	step := RollbackStep{
		Name:       name,
		Status:     "running",
		StartTime:  time.Now(),
		Details:    map[string]interface{}{"description": description},
		Retries:    0,
		MaxRetries: 3,
	}

	entry.Steps = append(entry.Steps, step)

	// Notify step progress
	if rm.notificationManager != nil {
		rm.notificationManager.AddOperationStep(entry.ID, name, description)
	}

	return &entry.Steps[len(entry.Steps)-1]
}

// completeStep marks a step as completed
func (rm *RollbackManager) completeStep(step *RollbackStep, success bool, errorMsg string) {
	endTime := time.Now()
	step.EndTime = &endTime
	step.Duration = endTime.Sub(step.StartTime)

	if success {
		step.Status = "completed"
	} else {
		step.Status = "failed"
		step.Error = errorMsg
	}

	// Notify step completion
	if rm.notificationManager != nil {
		// Find rollback entry for this step (simplified lookup)
		for _, entry := range rm.rollbackHistory {
			for _, entryStep := range entry.Steps {
				if entryStep.Name == step.Name && entryStep.StartTime.Equal(step.StartTime) {
					rm.notificationManager.CompleteOperationStep(entry.ID, step.Name, success, errorMsg)
					return
				}
			}
		}
	}
}

// findPreviousImage finds the previous image for rollback
func (rm *RollbackManager) findPreviousImage(ctx context.Context, containerID string) (string, error) {
	// Look for backup images
	for _, entry := range rm.rollbackHistory {
		if entry.ContainerID == containerID && entry.BackupImage != "" {
			return entry.BackupImage, nil
		}
	}

	// If no backup found, try to find from original config
	for _, entry := range rm.rollbackHistory {
		if entry.ContainerID == containerID && entry.OriginalConfig != nil {
			return entry.OriginalConfig.Image, nil
		}
	}

	return "", fmt.Errorf("no previous image found for container %s", containerID)
}

// countCompletedSteps counts completed steps
func (rm *RollbackManager) countCompletedSteps(entry *RollbackEntry) int {
	completed := 0
	for _, step := range entry.Steps {
		if step.Status == "completed" {
			completed++
		}
	}
	return completed
}

// GetRollbackHistory returns rollback history
func (rm *RollbackManager) GetRollbackHistory() []*RollbackEntry {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var history []*RollbackEntry
	for _, entry := range rm.rollbackHistory {
		history = append(history, entry)
	}

	return history
}

// GetRollbackEntry returns a specific rollback entry
func (rm *RollbackManager) GetRollbackEntry(rollbackID string) (*RollbackEntry, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	entry, exists := rm.rollbackHistory[rollbackID]
	return entry, exists
}

// cleanupRoutine performs periodic cleanup
func (rm *RollbackManager) cleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rm.cleanupOldEntries()
		rm.cleanupOldBackups()
	}
}

// cleanupOldEntries removes old rollback entries
func (rm *RollbackManager) cleanupOldEntries() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if len(rm.rollbackHistory) <= rm.config.MaxRollbackHistory {
		return
	}

	// Convert to slice for sorting
	var entries []*RollbackEntry
	for _, entry := range rm.rollbackHistory {
		entries = append(entries, entry)
	}

	// Sort by start time (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].StartTime.After(entries[j].StartTime)
	})

	// Keep only the most recent entries
	toKeep := rm.config.MaxRollbackHistory
	for i := toKeep; i < len(entries); i++ {
		delete(rm.rollbackHistory, entries[i].ID)
		rm.logger.WithField("rollback_id", entries[i].ID).Debug("Cleaned up old rollback entry")
	}
}

// cleanupOldBackups removes old backup images (this would integrate with Docker client)
func (rm *RollbackManager) cleanupOldBackups() {
	client := rm.clientManager.GetClient()
	ctx := context.Background()

	cutoff := time.Now().Add(-rm.config.BackupRetentionPeriod)

	rm.mutex.RLock()
	var oldBackups []string
	for _, entry := range rm.rollbackHistory {
		if entry.BackupImage != "" && entry.StartTime.Before(cutoff) {
			oldBackups = append(oldBackups, entry.BackupImage)
		}
	}
	rm.mutex.RUnlock()

	for _, backupImage := range oldBackups {
		if _, err := client.ImageRemove(ctx, backupImage, types.ImageRemoveOptions{Force: true}); err != nil {
			rm.logger.WithError(err).WithField("backup_image", backupImage).Warn("Failed to remove old backup image")
		} else {
			rm.logger.WithField("backup_image", backupImage).Info("Removed old backup image")
		}
	}
}

// GetStats returns rollback manager statistics
func (rm *RollbackManager) GetStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_rollbacks":     len(rm.rollbackHistory),
		"successful_rollbacks": 0,
		"failed_rollbacks":     0,
		"pending_rollbacks":    0,
		"running_rollbacks":    0,
		"config":              rm.config,
	}

	for _, entry := range rm.rollbackHistory {
		switch entry.Status {
		case RollbackStatusCompleted:
			stats["successful_rollbacks"] = stats["successful_rollbacks"].(int) + 1
		case RollbackStatusFailed:
			stats["failed_rollbacks"] = stats["failed_rollbacks"].(int) + 1
		case RollbackStatusPending:
			stats["pending_rollbacks"] = stats["pending_rollbacks"].(int) + 1
		case RollbackStatusRunning:
			stats["running_rollbacks"] = stats["running_rollbacks"].(int) + 1
		}
	}

	return stats
}