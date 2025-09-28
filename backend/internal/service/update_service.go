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
	"docker-auto/pkg/docker"
	"docker-auto/pkg/security"
	dockerTypes "docker-auto/pkg/types"

	"github.com/sirupsen/logrus"
)

// UpdateService manages intelligent container updates with automated detection and rollback
type UpdateService struct {
	containerRepo      repository.ContainerRepository
	updateHistoryRepo  repository.UpdateHistoryRepository
	imageVersionRepo   repository.ImageVersionRepository
	activityRepo       repository.ActivityLogRepository
	dockerClient       *docker.DockerClient
	dockerManager      *docker.ClientManager
	updateEngine       *docker.UpdateEngine
	imageScanner       *docker.ImageScanner
	rollbackManager    *docker.RollbackManager
	notificationManager *docker.NotificationManager
	cache              *CacheService
	config             *config.Config
	userService        *UserService

	// Update orchestration
	activeUpdates      map[string]*UpdateOperation
	updateQueue        chan *UpdateRequest
	updatesMu          sync.RWMutex

	// Monitoring and health check
	monitoringService  *ContainerMonitoringService
	healthChecker      *HealthChecker

	// Update detection and scheduling
	updateDetector     *UpdateDetector
	updateScheduler    *UpdateScheduler

	logger             *logrus.Entry
}

// UpdateOperation represents an active update operation
type UpdateOperation struct {
	ID              string                              `json:"id"`
	ContainerID     string                              `json:"container_id"`
	ContainerName   string                              `json:"container_name"`
	UserID          int64                               `json:"user_id"`
	Strategy        dockerTypes.UpdateStrategy          `json:"strategy"`
	Options         *dockerTypes.ContainerUpdateOptions `json:"options"`
	Status          UpdateOperationStatus               `json:"status"`
	StartTime       time.Time                           `json:"start_time"`
	EndTime         *time.Time                          `json:"end_time,omitempty"`
	Progress        int                                 `json:"progress"` // 0-100
	CurrentStep     string                              `json:"current_step"`
	Result          *dockerTypes.ContainerUpdateResult  `json:"result,omitempty"`
	Error           string                              `json:"error,omitempty"`

	// Health monitoring
	PreUpdateHealth  *HealthSnapshot                    `json:"pre_update_health,omitempty"`
	PostUpdateHealth *HealthSnapshot                    `json:"post_update_health,omitempty"`

	// Context and cancellation
	cancel          context.CancelFunc                  `json:"-"`
	mu              sync.RWMutex                       `json:"-"`
}

// UpdateRequest represents a request to update a container
type UpdateRequest struct {
	ContainerID     string                              `json:"container_id"`
	UserID          int64                               `json:"user_id"`
	NewImage        string                              `json:"new_image,omitempty"`
	Strategy        dockerTypes.UpdateStrategy          `json:"strategy"`
	Options         *dockerTypes.ContainerUpdateOptions `json:"options,omitempty"`
	ScheduledTime   *time.Time                          `json:"scheduled_time,omitempty"`
	AutoDetected    bool                                `json:"auto_detected"`
}

// UpdateOperationStatus represents the status of an update operation
type UpdateOperationStatus string

const (
	UpdateStatusPending    UpdateOperationStatus = "pending"
	UpdateStatusRunning    UpdateOperationStatus = "running"
	UpdateStatusCompleted  UpdateOperationStatus = "completed"
	UpdateStatusFailed     UpdateOperationStatus = "failed"
	UpdateStatusRolledBack UpdateOperationStatus = "rolled_back"
	UpdateStatusCancelled  UpdateOperationStatus = "cancelled"
)

// HealthSnapshot represents container health at a point in time
type HealthSnapshot struct {
	Timestamp       time.Time                `json:"timestamp"`
	ContainerStatus model.ContainerStatus    `json:"container_status"`
	Metrics         *ContainerMetrics        `json:"metrics,omitempty"`
	HealthCheck     *HealthCheckResult       `json:"health_check,omitempty"`
	IsHealthy       bool                     `json:"is_healthy"`
	HealthScore     float64                  `json:"health_score"` // 0-100
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Success         bool          `json:"success"`
	ResponseTime    time.Duration `json:"response_time"`
	StatusCode      int           `json:"status_code,omitempty"`
	Error           string        `json:"error,omitempty"`
	CheckType       string        `json:"check_type"`
	Endpoint        string        `json:"endpoint,omitempty"`
}

// HealthChecker performs container health checks
type HealthChecker struct {
	dockerClient    *docker.DockerClient
	config          *config.Config
	logger          *logrus.Entry
}

// UpdateDetector detects available updates for containers
type UpdateDetector struct {
	imageScanner       *docker.ImageScanner
	imageVersionRepo   repository.ImageVersionRepository
	containerRepo      repository.ContainerRepository
	config             *config.Config
	logger             *logrus.Entry
}

// UpdateScheduler schedules and manages update operations
type UpdateScheduler struct {
	updateQueue        chan *UpdateRequest
	config             *config.Config
	logger             *logrus.Entry
	mu                 sync.RWMutex

	// Advanced scheduling features
	scheduledUpdates   map[string]*ScheduledUpdate
	maintenanceWindows []*MaintenanceWindow
	updatePriorities   map[string]UpdatePriority
	resourceLimits     *ResourceLimits
}

// NewUpdateService creates a new update service
func NewUpdateService(
	containerRepo repository.ContainerRepository,
	updateHistoryRepo repository.UpdateHistoryRepository,
	imageVersionRepo repository.ImageVersionRepository,
	activityRepo repository.ActivityLogRepository,
	dockerClient *docker.DockerClient,
	dockerManager *docker.ClientManager,
	cache *CacheService,
	config *config.Config,
	userService *UserService,
	monitoringService *ContainerMonitoringService,
) *UpdateService {

	logger := logrus.WithField("component", "update_service")

	// Initialize update engine
	var updateEngine *docker.UpdateEngine
	if dockerManager != nil {
		updateEngine = docker.NewUpdateEngine(dockerManager, logrus.StandardLogger())
	}

	// Initialize image scanner
	imageScanner := docker.NewImageScanner(dockerManager, &docker.ScannerConfig{
		ScanTimeout:        30 * time.Second,
		MaxConcurrentScans: 10,
		CacheExpiration:    24 * time.Hour,
		EnabledScanners:    []string{"basic", "cve", "compliance"},
	}, logrus.StandardLogger())

	// Initialize notification manager
	notificationManager := docker.NewNotificationManager(logrus.StandardLogger())

	// Initialize rollback manager
	rollbackManager := docker.NewRollbackManager(dockerManager, notificationManager, &docker.RollbackConfig{
		MaxRollbackHistory:    10,
		AutoRollbackEnabled:   true,
		RollbackTimeout:       10 * time.Minute,
		HealthCheckRetries:    3,
		HealthCheckInterval:   30 * time.Second,
		BackupRetentionPeriod: 24 * time.Hour,
		FailureThreshold:      3,
	}, logrus.StandardLogger())

	// Create health checker
	healthChecker := &HealthChecker{
		dockerClient: dockerClient,
		config:       config,
		logger:       logger.WithField("subcomponent", "health_checker"),
	}

	// Create update detector
	updateDetector := &UpdateDetector{
		imageScanner:     imageScanner,
		imageVersionRepo: imageVersionRepo,
		containerRepo:    containerRepo,
		config:           config,
		logger:           logger.WithField("subcomponent", "update_detector"),
	}

	// Create update scheduler
	updateScheduler := &UpdateScheduler{
		updateQueue: make(chan *UpdateRequest, 1000),
		config:      config,
		logger:      logger.WithField("subcomponent", "update_scheduler"),
	}

	service := &UpdateService{
		containerRepo:       containerRepo,
		updateHistoryRepo:   updateHistoryRepo,
		imageVersionRepo:    imageVersionRepo,
		activityRepo:        activityRepo,
		dockerClient:        dockerClient,
		dockerManager:       dockerManager,
		updateEngine:        updateEngine,
		imageScanner:        imageScanner,
		rollbackManager:     rollbackManager,
		notificationManager: notificationManager,
		cache:               cache,
		config:              config,
		userService:         userService,
		activeUpdates:       make(map[string]*UpdateOperation),
		updateQueue:         updateScheduler.updateQueue,
		monitoringService:   monitoringService,
		healthChecker:       healthChecker,
		updateDetector:      updateDetector,
		updateScheduler:     updateScheduler,
		logger:              logger,
	}

	// Start background workers
	go service.updateWorker()
	go service.healthMonitoringWorker()
	go service.updateDetectionWorker()
	go service.scheduleMaintenanceWorker()

	logger.Info("Update service initialized")
	return service
}

// UpdateContainer initiates a container update operation
func (s *UpdateService) UpdateContainer(ctx context.Context, userID int64, req *UpdateRequest) (*UpdateOperation, error) {
	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid update request: %w", err)
	}

	// Get container
	container, err := s.containerRepo.GetByContainerID(ctx, req.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	// Check permissions
	if err := s.checkUpdatePermissions(userID, container); err != nil {
		return nil, fmt.Errorf("permission denied: %w", err)
	}

	// Check if container is already being updated
	s.updatesMu.RLock()
	if _, exists := s.activeUpdates[container.ContainerID]; exists {
		s.updatesMu.RUnlock()
		return nil, fmt.Errorf("container is already being updated")
	}
	s.updatesMu.RUnlock()

	// Determine update image if not specified
	if req.NewImage == "" {
		latestImage, err := s.updateDetector.GetLatestImage(ctx, container.Image, container.Tag)
		if err != nil {
			return nil, fmt.Errorf("failed to determine latest image: %w", err)
		}
		req.NewImage = latestImage
	}

	// Create update options if not provided
	if req.Options == nil {
		req.Options = s.createDefaultUpdateOptions(container, req)
	}

	// Generate operation ID
	operationID := fmt.Sprintf("update_%s_%d", container.ContainerID, time.Now().UnixNano())

	// Create operation context with timeout
	operationCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)

	// Create update operation
	operation := &UpdateOperation{
		ID:            operationID,
		ContainerID:   container.ContainerID,
		ContainerName: container.Name,
		UserID:        userID,
		Strategy:      req.Strategy,
		Options:       req.Options,
		Status:        UpdateStatusPending,
		StartTime:     time.Now(),
		Progress:      0,
		CurrentStep:   "Preparing update",
		cancel:        cancel,
	}

	// Store operation
	s.updatesMu.Lock()
	s.activeUpdates[container.ContainerID] = operation
	s.updatesMu.Unlock()

	// Take pre-update health snapshot
	if preHealth, err := s.takeHealthSnapshot(operationCtx, container.ContainerID); err == nil {
		operation.PreUpdateHealth = preHealth
	}

	// Queue update request
	select {
	case s.updateQueue <- req:
		s.logger.WithFields(logrus.Fields{
			"operation_id":   operationID,
			"container_id":   container.ContainerID,
			"container_name": container.Name,
			"user_id":        userID,
			"new_image":      req.NewImage,
			"strategy":       req.Strategy,
		}).Info("Update operation queued")
	default:
		// Queue is full
		s.updatesMu.Lock()
		delete(s.activeUpdates, container.ContainerID)
		s.updatesMu.Unlock()
		cancel()
		return nil, fmt.Errorf("update queue is full")
	}

	// Log activity
	s.logUpdateActivity(userID, int64(container.ID), "update_initiated", "Container update initiated", map[string]interface{}{
		"operation_id":   operationID,
		"new_image":      req.NewImage,
		"strategy":       req.Strategy,
		"auto_detected":  req.AutoDetected,
	})

	return operation, nil
}

// GetUpdateOperation retrieves an update operation by ID
func (s *UpdateService) GetUpdateOperation(operationID string) (*UpdateOperation, error) {
	s.updatesMu.RLock()
	defer s.updatesMu.RUnlock()

	for _, operation := range s.activeUpdates {
		if operation.ID == operationID {
			operation.mu.RLock()
			defer operation.mu.RUnlock()

			// Create safe copy
			return &UpdateOperation{
				ID:               operation.ID,
				ContainerID:      operation.ContainerID,
				ContainerName:    operation.ContainerName,
				UserID:           operation.UserID,
				Strategy:         operation.Strategy,
				Status:           operation.Status,
				StartTime:        operation.StartTime,
				EndTime:          operation.EndTime,
				Progress:         operation.Progress,
				CurrentStep:      operation.CurrentStep,
				Result:           operation.Result,
				Error:            operation.Error,
				PreUpdateHealth:  operation.PreUpdateHealth,
				PostUpdateHealth: operation.PostUpdateHealth,
			}, nil
		}
	}

	return nil, fmt.Errorf("update operation not found")
}

// ListActiveOperations returns all active update operations
func (s *UpdateService) ListActiveOperations() []*UpdateOperation {
	s.updatesMu.RLock()
	defer s.updatesMu.RUnlock()

	operations := make([]*UpdateOperation, 0, len(s.activeUpdates))
	for _, operation := range s.activeUpdates {
		operation.mu.RLock()
		// Create safe copy
		operationCopy := &UpdateOperation{
			ID:               operation.ID,
			ContainerID:      operation.ContainerID,
			ContainerName:    operation.ContainerName,
			UserID:           operation.UserID,
			Strategy:         operation.Strategy,
			Status:           operation.Status,
			StartTime:        operation.StartTime,
			EndTime:          operation.EndTime,
			Progress:         operation.Progress,
			CurrentStep:      operation.CurrentStep,
			Error:            operation.Error,
			PreUpdateHealth:  operation.PreUpdateHealth,
			PostUpdateHealth: operation.PostUpdateHealth,
		}
		operations = append(operations, operationCopy)
		operation.mu.RUnlock()
	}

	return operations
}

// CancelUpdateOperation cancels an active update operation
func (s *UpdateService) CancelUpdateOperation(operationID string, userID int64) error {
	s.updatesMu.Lock()
	defer s.updatesMu.Unlock()

	var operation *UpdateOperation
	for _, op := range s.activeUpdates {
		if op.ID == operationID {
			operation = op
			break
		}
	}

	if operation == nil {
		return fmt.Errorf("update operation not found")
	}

	// Check permissions
	if operation.UserID != userID {
		// Check if user has admin role
		user, err := s.userService.GetUserByID(context.Background(), userID)
		if err != nil || user.Role != model.UserRoleAdmin {
			return fmt.Errorf("permission denied: can only cancel own operations or admin required")
		}
	}

	operation.mu.Lock()
	if operation.Status != UpdateStatusPending && operation.Status != UpdateStatusRunning {
		operation.mu.Unlock()
		return fmt.Errorf("cannot cancel operation with status: %s", operation.Status)
	}

	operation.Status = UpdateStatusCancelled
	operation.Error = "Operation cancelled by user"
	endTime := time.Now()
	operation.EndTime = &endTime

	// Cancel context
	if operation.cancel != nil {
		operation.cancel()
	}
	operation.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"operation_id": operationID,
		"user_id":      userID,
	}).Info("Update operation cancelled")

	return nil
}

// RollbackContainer rolls back a container to its previous state
func (s *UpdateService) RollbackContainer(ctx context.Context, userID int64, containerID string) error {
	// Get container
	container, err := s.containerRepo.GetByContainerID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Check permissions
	if err := s.checkUpdatePermissions(userID, container); err != nil {
		return fmt.Errorf("permission denied: %w", err)
	}

	// Check if container is currently being updated
	s.updatesMu.RLock()
	if operation, exists := s.activeUpdates[container.ContainerID]; exists {
		if operation.Status == UpdateStatusRunning {
			s.updatesMu.RUnlock()
			return fmt.Errorf("cannot rollback: container is currently being updated")
		}
	}
	s.updatesMu.RUnlock()

	// Perform rollback using rollback manager
	// Create rollback request
	rollbackRequest := &docker.RollbackRequest{
		ContainerID: container.ContainerID,
		Reason:      "Manual rollback requested",
		Force:       false,
		HealthCheck: true,
		Timeout:     10 * time.Minute,
		UserContext: &security.DockerUserContext{
			UserID:   userID,
			Username: "user", // Would get from user service
			Role:     "user", // Would get from user service
		},
	}

	// Prepare rollback
	rollbackEntry, err := s.rollbackManager.PrepareRollback(ctx, rollbackRequest)
	if err != nil {
		return fmt.Errorf("rollback preparation failed: %w", err)
	}

	// Execute rollback
	result, err := s.rollbackManager.ExecuteRollback(ctx, rollbackEntry.ID)
	if err != nil {
		return fmt.Errorf("rollback execution failed: %w", err)
	}

	// Log activity
	s.logUpdateActivity(userID, int64(container.ID), "container_rollback", "Container rolled back", map[string]interface{}{
		"rollback_id":     rollbackEntry.ID,
		"success":         result.Success,
		"steps_completed": result.StepsCompleted,
	})

	s.logger.WithFields(logrus.Fields{
		"container_id":   containerID,
		"user_id":        userID,
		"rollback_id":    result.RollbackEntry.ID,
		"previous_image": result.RollbackEntry.OriginalImage,
	}).Info("Container rollback completed")

	return nil
}

// DetectUpdates detects available updates for containers
func (s *UpdateService) DetectUpdates(ctx context.Context) ([]*UpdateDetection, error) {
	return s.updateDetector.DetectAvailableUpdates(ctx)
}

// GetUpdateHistory returns update history for a container
func (s *UpdateService) GetUpdateHistory(ctx context.Context, containerID int64, limit, offset int) ([]*model.UpdateHistory, int64, error) {
	return s.updateHistoryRepo.GetByContainerID(ctx, containerID, limit, offset)
}

// updateWorker processes update requests from the queue
func (s *UpdateService) updateWorker() {
	for req := range s.updateQueue {
		if err := s.processUpdateRequest(req); err != nil {
			s.logger.WithError(err).WithField("container_id", req.ContainerID).Error("Failed to process update request")
		}
	}
}

// processUpdateRequest processes a single update request
func (s *UpdateService) processUpdateRequest(req *UpdateRequest) error {
	// Get operation
	s.updatesMu.RLock()
	operation, exists := s.activeUpdates[req.ContainerID]
	s.updatesMu.RUnlock()

	if !exists {
		return fmt.Errorf("update operation not found for container %s", req.ContainerID)
	}

	// Update operation status
	operation.mu.Lock()
	operation.Status = UpdateStatusRunning
	operation.CurrentStep = "Starting update"
	operation.Progress = 10
	operation.mu.Unlock()

	// Create context with cancel function
	ctx := context.Background()
	if operation.cancel != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	// Get user context for permissions
	userContext := &security.DockerUserContext{
		UserID:   req.UserID,
		Username: "user", // Would get from user service
		Role:     "user", // Would get from user service
	}

	// Execute update using update engine
	result, err := s.updateEngine.UpdateContainer(ctx, req.ContainerID, req.Options, userContext)

	// Update operation with result
	operation.mu.Lock()
	operation.Result = result
	endTime := time.Now()
	operation.EndTime = &endTime

	if err != nil {
		operation.Status = UpdateStatusFailed
		operation.Error = err.Error()
		operation.Progress = 0
	} else {
		if result.RolledBack {
			operation.Status = UpdateStatusRolledBack
		} else {
			operation.Status = UpdateStatusCompleted
		}
		operation.Progress = 100
	}
	operation.mu.Unlock()

	// Take post-update health snapshot
	if operation.Status == UpdateStatusCompleted {
		if postHealth, healthErr := s.takeHealthSnapshot(ctx, req.ContainerID); healthErr == nil {
			operation.mu.Lock()
			operation.PostUpdateHealth = postHealth
			operation.mu.Unlock()

			// Check if rollback is needed based on health
			if !postHealth.IsHealthy && req.Options.RollbackOnFailure {
				s.logger.WithField("container_id", req.ContainerID).Warn("Post-update health check failed, initiating rollback")

				if rollbackErr := s.performHealthBasedRollback(ctx, req.ContainerID, userContext); rollbackErr != nil {
					s.logger.WithError(rollbackErr).WithField("container_id", req.ContainerID).Error("Health-based rollback failed")
				} else {
					operation.mu.Lock()
					operation.Status = UpdateStatusRolledBack
					operation.Error = "Rolled back due to failed health check"
					operation.mu.Unlock()
				}
			}
		}
	}

	// Persist update history
	s.persistUpdateHistory(req, operation, result)

	// Send notifications
	s.sendUpdateNotifications(operation, result)

	// Remove from active operations
	s.updatesMu.Lock()
	delete(s.activeUpdates, req.ContainerID)
	s.updatesMu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"container_id": req.ContainerID,
		"operation_id": operation.ID,
		"status":       operation.Status,
		"duration":     endTime.Sub(operation.StartTime),
	}).Info("Update operation completed")

	return nil
}

// Background workers

// healthMonitoringWorker monitors container health during updates
func (s *UpdateService) healthMonitoringWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.monitorActiveUpdates()
		}
	}
}

// updateDetectionWorker periodically checks for available updates
func (s *UpdateService) updateDetectionWorker() {
	ticker := time.NewTicker(6 * time.Hour) // Check for updates every 6 hours
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performAutomaticUpdateDetection()
		}
	}
}

// scheduleMaintenanceWorker handles scheduled maintenance tasks
func (s *UpdateService) scheduleMaintenanceWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performMaintenanceTasks()
		}
	}
}

// Helper methods

// validateUpdateRequest validates an update request
func (s *UpdateService) validateUpdateRequest(req *UpdateRequest) error {
	if req.ContainerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if req.UserID <= 0 {
		return fmt.Errorf("user ID is required")
	}
	if req.Strategy == "" {
		req.Strategy = dockerTypes.UpdateStrategyRecreate // Default strategy
	}
	return nil
}

// checkUpdatePermissions checks if user has permission to update container
func (s *UpdateService) checkUpdatePermissions(userID int64, container *model.Container) error {
	// Check if user owns the container or is admin
	if container.CreatedBy == nil || int64(*container.CreatedBy) != userID {
		// Check if user has admin role
		user, err := s.userService.GetUserByID(context.Background(), userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		if user.Role != model.UserRoleAdmin {
			return fmt.Errorf("access denied: container belongs to different user")
		}
	}
	return nil
}

// createDefaultUpdateOptions creates default update options for a container
func (s *UpdateService) createDefaultUpdateOptions(container *model.Container, req *UpdateRequest) *dockerTypes.ContainerUpdateOptions {
	return &dockerTypes.ContainerUpdateOptions{
		NewImage:            req.NewImage,
		Strategy:            req.Strategy,
		RollbackOnFailure:   true,
		HealthCheckTimeout:  30 * time.Second,
		BackupConfig: &dockerTypes.BackupConfig{
			Enabled:         true,
			BackupImage:     "backup-image",
			BackupContainer: "backup-container",
		},
		NotificationConfig: &dockerTypes.NotificationConfig{
			Enabled:   true,
			OnSuccess: true,
			OnFailure: true,
			OnRollback: true,
		},
	}
}

// takeHealthSnapshot takes a health snapshot of a container
func (s *UpdateService) takeHealthSnapshot(ctx context.Context, containerID string) (*HealthSnapshot, error) {
	// Get container status
	status, err := s.dockerClient.GetContainerStatus(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %w", err)
	}

	// Get container metrics if monitoring service is available
	var metrics *ContainerMetrics
	if s.monitoringService != nil {
		if m, err := s.monitoringService.GetContainerMetrics(ctx, containerID); err == nil {
			metrics = m
		}
	}

	// Perform health check
	healthCheck, err := s.healthChecker.CheckContainerHealth(ctx, containerID)
	if err != nil {
		s.logger.WithError(err).WithField("container_id", containerID).Warn("Health check failed")
	}

	// Calculate health score
	healthScore := s.calculateHealthScore(status, metrics, healthCheck)

	return &HealthSnapshot{
		Timestamp:       time.Now(),
		ContainerStatus: status,
		Metrics:         metrics,
		HealthCheck:     healthCheck,
		IsHealthy:       healthScore >= 70.0, // Threshold for healthy
		HealthScore:     healthScore,
	}, nil
}

// calculateHealthScore calculates a health score based on various factors
func (s *UpdateService) calculateHealthScore(status model.ContainerStatus, metrics *ContainerMetrics, healthCheck *HealthCheckResult) float64 {
	score := 0.0

	// Base score from container status
	switch status {
	case model.ContainerStatusRunning:
		score += 40.0
	case model.ContainerStatusRestarting:
		score += 20.0
	default:
		score += 0.0
	}

	// Score from metrics
	if metrics != nil {
		// CPU usage (lower is better)
		if metrics.CPUPercent < 50 {
			score += 20.0
		} else if metrics.CPUPercent < 80 {
			score += 10.0
		}

		// Memory usage (lower is better)
		if metrics.MemoryPercent < 60 {
			score += 20.0
		} else if metrics.MemoryPercent < 85 {
			score += 10.0
		}
	}

	// Score from health check
	if healthCheck != nil && healthCheck.Success {
		score += 20.0
	}

	return score
}

// performHealthBasedRollback performs rollback based on health check failure
func (s *UpdateService) performHealthBasedRollback(ctx context.Context, containerID string, userContext *security.DockerUserContext) error {
	// Create rollback request for health-based rollback
	rollbackRequest := &docker.RollbackRequest{
		ContainerID: containerID,
		Reason:      "Health check failure triggered rollback",
		Force:       true,
		HealthCheck: false, // Skip health check since it already failed
		Timeout:     5 * time.Minute,
		UserContext: userContext,
	}

	// Prepare rollback
	rollbackEntry, err := s.rollbackManager.PrepareRollback(ctx, rollbackRequest)
	if err != nil {
		return fmt.Errorf("rollback preparation failed: %w", err)
	}

	// Execute rollback
	_, err = s.rollbackManager.ExecuteRollback(ctx, rollbackEntry.ID)
	return err
}

// logUpdateActivity logs update-related activities
func (s *UpdateService) logUpdateActivity(userID int64, containerID int64, action, description string, metadata map[string]interface{}) {
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
		UserID:       &userID,
		Action:       action,
		ResourceType: "container_update",
		ResourceID:   func() *int { id := int(containerID); return &id }(),
		Description:  description,
		Metadata:     metadataJSON,
	}

	if err := s.activityRepo.Create(context.Background(), activity); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
			"action":       action,
		}).Warn("Failed to log update activity")
	}
}

// persistUpdateHistory persists update operation to database
func (s *UpdateService) persistUpdateHistory(req *UpdateRequest, operation *UpdateOperation, result *dockerTypes.ContainerUpdateResult) {
	if s.updateHistoryRepo == nil {
		return
	}

	container, err := s.containerRepo.GetByContainerID(context.Background(), req.ContainerID)
	if err != nil {
		s.logger.WithError(err).WithField("container_id", req.ContainerID).Warn("Failed to get container for history")
		return
	}

	var status model.UpdateStatus
	switch operation.Status {
	case UpdateStatusCompleted:
		status = model.UpdateStatusSuccess
	case UpdateStatusFailed:
		status = model.UpdateStatusFailed
	case UpdateStatusRolledBack:
		status = model.UpdateStatusRollback
	case UpdateStatusCancelled:
		status = model.UpdateStatusCancelled
	default:
		status = model.UpdateStatusFailed
	}

	history := &model.UpdateHistory{
		ContainerID:    int(container.ID),
		OldImage:       operation.Result.OldImage,
		NewImage:       operation.Result.NewImage,
		Strategy:       model.UpdateStrategy(operation.Strategy),
		Status:         status,
		StartedAt:      operation.StartTime,
		CompletedAt:    operation.EndTime,
		ErrorMessage:   operation.Error,
		TriggeredBy:    model.TriggerTypeManual,
	}

	if err := s.updateHistoryRepo.Create(context.Background(), history); err != nil {
		s.logger.WithError(err).WithField("operation_id", operation.ID).Warn("Failed to persist update history")
	}
}

// sendUpdateNotifications sends notifications for update operations
func (s *UpdateService) sendUpdateNotifications(operation *UpdateOperation, result *dockerTypes.ContainerUpdateResult) {
	if s.notificationManager == nil {
		return
	}

	// This would send notifications based on the operation result
	s.logger.WithFields(logrus.Fields{
		"operation_id": operation.ID,
		"status":       operation.Status,
	}).Debug("Update notifications sent")
}

// Health checker methods

// CheckContainerHealth performs a health check on a container
func (hc *HealthChecker) CheckContainerHealth(ctx context.Context, containerID string) (*HealthCheckResult, error) {
	start := time.Now()

	// Get container info
	containerInfo, err := hc.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return &HealthCheckResult{
			Success:      false,
			ResponseTime: time.Since(start),
			Error:        err.Error(),
			CheckType:    "container_inspect",
		}, err
	}

	// Check if container is running
	if !containerInfo.State.Running {
		return &HealthCheckResult{
			Success:      false,
			ResponseTime: time.Since(start),
			Error:        "container is not running",
			CheckType:    "container_status",
		}, nil
	}

	// Additional health checks would go here (HTTP endpoints, etc.)

	return &HealthCheckResult{
		Success:      true,
		ResponseTime: time.Since(start),
		CheckType:    "container_status",
	}, nil
}

// Update detector methods

// UpdateDetection represents a detected update
type UpdateDetection struct {
	ContainerID     string    `json:"container_id"`
	ContainerName   string    `json:"container_name"`
	CurrentImage    string    `json:"current_image"`
	AvailableImage  string    `json:"available_image"`
	UpdateType      string    `json:"update_type"` // "patch", "minor", "major"
	DetectedAt      time.Time `json:"detected_at"`
	AutoUpdateable  bool      `json:"auto_updateable"`
}

// DetectAvailableUpdates detects available updates for all containers
func (ud *UpdateDetector) DetectAvailableUpdates(ctx context.Context) ([]*UpdateDetection, error) {
	// Get all containers with auto-update policy
	containers, err := ud.containerRepo.GetAutoUpdateContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	var detections []*UpdateDetection
	for _, container := range containers {
		if detection, err := ud.checkContainerForUpdates(ctx, container); err == nil && detection != nil {
			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// checkContainerForUpdates checks a single container for available updates
func (ud *UpdateDetector) checkContainerForUpdates(ctx context.Context, container *model.Container) (*UpdateDetection, error) {
	// Get latest image version
	latestImage, err := ud.GetLatestImage(ctx, container.Image, container.Tag)
	if err != nil {
		return nil, err
	}

	// Compare with current image
	currentImage := container.GetFullImageName()
	if currentImage == latestImage {
		return nil, nil // No update available
	}

	return &UpdateDetection{
		ContainerID:     container.ContainerID,
		ContainerName:   container.Name,
		CurrentImage:    currentImage,
		AvailableImage:  latestImage,
		UpdateType:      "patch", // Would determine based on version comparison
		DetectedAt:      time.Now(),
		AutoUpdateable:  container.UpdatePolicy == model.UpdatePolicyAuto,
	}, nil
}

// GetLatestImage gets the latest image version for a base image
func (ud *UpdateDetector) GetLatestImage(ctx context.Context, image, tag string) (string, error) {
	// Get latest version from image version repository
	latest, err := ud.imageVersionRepo.GetLatest(ctx, image)
	if err != nil {
		return "", fmt.Errorf("failed to get latest image version: %w", err)
	}

	if latest != nil {
		return fmt.Sprintf("%s:%s", latest.ImageName, latest.Tag), nil
	}

	// Fallback to current image if no newer version found
	return fmt.Sprintf("%s:%s", image, tag), nil
}

// Maintenance methods

// monitorActiveUpdates monitors active updates for health issues
func (s *UpdateService) monitorActiveUpdates() {
	s.updatesMu.RLock()
	operations := make([]*UpdateOperation, 0, len(s.activeUpdates))
	for _, op := range s.activeUpdates {
		operations = append(operations, op)
	}
	s.updatesMu.RUnlock()

	for _, operation := range operations {
		operation.mu.RLock()
		if operation.Status == UpdateStatusRunning {
			// Check if operation has been running too long
			if time.Since(operation.StartTime) > 30*time.Minute {
				operation.mu.RUnlock()
				s.logger.WithField("operation_id", operation.ID).Warn("Update operation running for too long")
				// Could cancel or investigate further
				continue
			}
		}
		operation.mu.RUnlock()
	}
}

// performAutomaticUpdateDetection performs automatic update detection
func (s *UpdateService) performAutomaticUpdateDetection() {
	ctx := context.Background()
	detections, err := s.updateDetector.DetectAvailableUpdates(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to detect available updates")
		return
	}

	for _, detection := range detections {
		if detection.AutoUpdateable {
			// Queue automatic update
			req := &UpdateRequest{
				ContainerID:  detection.ContainerID,
				UserID:       0, // System user
				NewImage:     detection.AvailableImage,
				Strategy:     dockerTypes.UpdateStrategyRecreate,
				AutoDetected: true,
			}

			select {
			case s.updateQueue <- req:
				s.logger.WithField("container_id", detection.ContainerID).Info("Automatic update queued")
			default:
				s.logger.WithField("container_id", detection.ContainerID).Warn("Failed to queue automatic update: queue full")
			}
		}
	}

	s.logger.WithField("detections_count", len(detections)).Info("Update detection completed")
}

// performMaintenanceTasks performs periodic maintenance tasks
func (s *UpdateService) performMaintenanceTasks() {
	// Clean up old update history
	if s.updateHistoryRepo != nil {
		cutoff := time.Now().AddDate(0, 0, -s.config.System.MaxLogRetentionDays)
		if deleted, err := s.updateHistoryRepo.DeleteOlderThan(context.Background(), cutoff); err != nil {
			s.logger.WithError(err).Warn("Failed to clean up old update history")
		} else if deleted > 0 {
			s.logger.WithField("deleted_count", deleted).Info("Cleaned up old update history")
		}
	}

	// Clean up completed operations older than 24 hours
	s.cleanupOldOperations()
}

// cleanupOldOperations removes old completed operations from memory
func (s *UpdateService) cleanupOldOperations() {
	s.updatesMu.Lock()
	defer s.updatesMu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for containerID, operation := range s.activeUpdates {
		operation.mu.RLock()
		isCompleted := operation.Status == UpdateStatusCompleted ||
			operation.Status == UpdateStatusFailed ||
			operation.Status == UpdateStatusRolledBack ||
			operation.Status == UpdateStatusCancelled

		isOld := operation.EndTime != nil && operation.EndTime.Before(cutoff)
		operation.mu.RUnlock()

		if isCompleted && isOld {
			delete(s.activeUpdates, containerID)
		}
	}
}

// Advanced update strategy types and structures

// ScheduledUpdate represents a scheduled update operation
type ScheduledUpdate struct {
	ID              string                     `json:"id"`
	ContainerID     string                     `json:"container_id"`
	ScheduledTime   time.Time                  `json:"scheduled_time"`
	Request         *UpdateRequest             `json:"request"`
	Priority        UpdatePriority             `json:"priority"`
	CreatedAt       time.Time                  `json:"created_at"`
	Status          ScheduledUpdateStatus      `json:"status"`
}

// MaintenanceWindow defines a maintenance window for updates
type MaintenanceWindow struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	StartTime       time.Time      `json:"start_time"`
	EndTime         time.Time      `json:"end_time"`
	Recurring       bool           `json:"recurring"`
	RecurringType   RecurringType  `json:"recurring_type,omitempty"`
	AllowedServices []string       `json:"allowed_services"`
	MaxConcurrent   int            `json:"max_concurrent"`
	Enabled         bool           `json:"enabled"`
}

// ResourceLimits defines resource limits for update operations
type ResourceLimits struct {
	MaxConcurrentUpdates int     `json:"max_concurrent_updates"`
	MaxCPUUsage         float64 `json:"max_cpu_usage"`
	MaxMemoryUsage      float64 `json:"max_memory_usage"`
	MinDiskSpace        int64   `json:"min_disk_space"`
}

// UpdatePriority defines update priority levels
type UpdatePriority int

const (
	PriorityLow UpdatePriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// ScheduledUpdateStatus represents the status of a scheduled update
type ScheduledUpdateStatus string

const (
	ScheduledStatusPending   ScheduledUpdateStatus = "pending"
	ScheduledStatusExecuting ScheduledUpdateStatus = "executing"
	ScheduledStatusCompleted ScheduledUpdateStatus = "completed"
	ScheduledStatusFailed    ScheduledUpdateStatus = "failed"
	ScheduledStatusCancelled ScheduledUpdateStatus = "cancelled"
)

// RecurringType defines types of recurring maintenance windows
type RecurringType string

const (
	RecurringDaily   RecurringType = "daily"
	RecurringWeekly  RecurringType = "weekly"
	RecurringMonthly RecurringType = "monthly"
)

// Smart Update Strategy Methods

// ScheduleUpdate schedules an update for a specific time
func (s *UpdateService) ScheduleUpdate(ctx context.Context, userID int64, req *UpdateRequest, scheduledTime time.Time, priority UpdatePriority) (*ScheduledUpdate, error) {
	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid update request: %w", err)
	}

	// Check if update can be scheduled during the requested time
	if !s.updateScheduler.CanScheduleAt(scheduledTime, req.ContainerID) {
		// Find next available maintenance window
		nextWindow := s.updateScheduler.FindNextMaintenanceWindow(scheduledTime, req.ContainerID)
		if nextWindow == nil {
			return nil, fmt.Errorf("no suitable maintenance window found")
		}
		scheduledTime = nextWindow.StartTime
	}

	scheduled := &ScheduledUpdate{
		ID:            fmt.Sprintf("scheduled_%s_%d", req.ContainerID, time.Now().UnixNano()),
		ContainerID:   req.ContainerID,
		ScheduledTime: scheduledTime,
		Request:       req,
		Priority:      priority,
		CreatedAt:     time.Now(),
		Status:        ScheduledStatusPending,
	}

	s.updateScheduler.mu.Lock()
	s.updateScheduler.scheduledUpdates[scheduled.ID] = scheduled
	s.updateScheduler.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"scheduled_id":   scheduled.ID,
		"container_id":   req.ContainerID,
		"scheduled_time": scheduledTime,
		"priority":       priority,
	}).Info("Update scheduled")

	return scheduled, nil
}

// CanScheduleAt checks if an update can be scheduled at the given time
func (us *UpdateScheduler) CanScheduleAt(scheduledTime time.Time, containerID string) bool {
	us.mu.RLock()
	defer us.mu.RUnlock()

	// Check if within maintenance window
	for _, window := range us.maintenanceWindows {
		if !window.Enabled {
			continue
		}

		if us.isTimeInWindow(scheduledTime, window) {
			// Check concurrent update limits
			concurrent := us.countConcurrentUpdates(scheduledTime, containerID)
			if concurrent < window.MaxConcurrent {
				return true
			}
		}
	}

	return false
}

// FindNextMaintenanceWindow finds the next available maintenance window
func (us *UpdateScheduler) FindNextMaintenanceWindow(after time.Time, containerID string) *MaintenanceWindow {
	us.mu.RLock()
	defer us.mu.RUnlock()

	var nextWindow *MaintenanceWindow
	var nextTime time.Time

	for _, window := range us.maintenanceWindows {
		if !window.Enabled {
			continue
		}

		windowStart := us.getNextWindowStart(after, window)
		if nextWindow == nil || windowStart.Before(nextTime) {
			nextWindow = window
			nextTime = windowStart
		}
	}

	return nextWindow
}

// OptimizeUpdateStrategy optimizes the update strategy based on system state
func (s *UpdateService) OptimizeUpdateStrategy(ctx context.Context, req *UpdateRequest) *dockerTypes.UpdateStrategy {
	// Get container metrics
	metrics, err := s.monitoringService.GetContainerMetrics(ctx, req.ContainerID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get container metrics for optimization")
		return &dockerTypes.UpdateStrategyRecreate // Fallback to safe strategy
	}

	// Get system load
	systemLoad := s.getSystemLoad(ctx)

	// Optimize based on container characteristics
	container, err := s.containerRepo.GetByContainerID(ctx, req.ContainerID)
	if err != nil {
		return &dockerTypes.UpdateStrategyRecreate
	}

	// Decision algorithm
	if s.shouldUseRollingUpdate(container, metrics, systemLoad) {
		return &dockerTypes.UpdateStrategyRolling
	} else if s.shouldUseBlueGreenUpdate(container, metrics, systemLoad) {
		return &dockerTypes.UpdateStrategyBlueGreen
	} else {
		return &dockerTypes.UpdateStrategyRecreate
	}
}

// shouldUseRollingUpdate determines if rolling update is optimal
func (s *UpdateService) shouldUseRollingUpdate(container *model.Container, metrics *ContainerMetrics, systemLoad *SystemLoad) bool {
	// Rolling update is good for:
	// 1. Low resource usage containers
	// 2. Stateless services
	// 3. Services with multiple replicas

	if metrics != nil && metrics.CPUPercent < 30 && metrics.MemoryPercent < 50 {
		return true
	}

	// Check if container has health check
	if container.HealthCheck != nil && container.HealthCheck != "" {
		return true
	}

	return false
}

// shouldUseBlueGreenUpdate determines if blue-green update is optimal
func (s *UpdateService) shouldUseBlueGreenUpdate(container *model.Container, metrics *ContainerMetrics, systemLoad *SystemLoad) bool {
	// Blue-green is good for:
	// 1. Critical services
	// 2. Services requiring zero downtime
	// 3. When system has sufficient resources

	if systemLoad != nil && systemLoad.CPUUsage < 60 && systemLoad.MemoryUsage < 70 {
		// Check if container is critical
		if container.Priority != nil && *container.Priority == "high" {
			return true
		}
	}

	return false
}

// BatchUpdateOptimization performs intelligent batch updates
func (s *UpdateService) BatchUpdateOptimization(ctx context.Context, userID int64, requests []*UpdateRequest) ([]*UpdateOperation, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no update requests provided")
	}

	// Group requests by dependency and priority
	batches := s.groupUpdatesByDependency(requests)

	var allOperations []*UpdateOperation

	// Process batches in order
	for i, batch := range batches {
		s.logger.WithFields(logrus.Fields{
			"batch_number": i + 1,
			"batch_size":   len(batch),
		}).Info("Processing update batch")

		var batchOperations []*UpdateOperation

		// Process batch with optimal concurrency
		concurrency := s.calculateOptimalConcurrency(batch)
		semaphore := make(chan struct{}, concurrency)

		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, req := range batch {
			wg.Add(1)
			go func(request *UpdateRequest) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				operation, err := s.UpdateContainer(ctx, userID, request)
				if err != nil {
					s.logger.WithError(err).WithField("container_id", request.ContainerID).Error("Batch update failed")
					return
				}

				mu.Lock()
				batchOperations = append(batchOperations, operation)
				mu.Unlock()
			}(req)
		}

		wg.Wait()
		allOperations = append(allOperations, batchOperations...)

		// Wait for batch completion before next batch
		s.waitForBatchCompletion(batchOperations)
	}

	return allOperations, nil
}

// groupUpdatesByDependency groups updates by their dependencies
func (s *UpdateService) groupUpdatesByDependency(requests []*UpdateRequest) [][]*UpdateRequest {
	// Simple implementation - could be enhanced with dependency analysis
	var batches [][]*UpdateRequest

	// Group by priority first
	priorityGroups := make(map[UpdatePriority][]*UpdateRequest)

	for _, req := range requests {
		priority := s.getUpdatePriority(req.ContainerID)
		priorityGroups[priority] = append(priorityGroups[priority], req)
	}

	// Create batches in priority order
	priorities := []UpdatePriority{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow}
	for _, priority := range priorities {
		if group, exists := priorityGroups[priority]; exists {
			// Split large groups into smaller batches
			batchSize := 5 // Configurable
			for i := 0; i < len(group); i += batchSize {
				end := i + batchSize
				if end > len(group) {
					end = len(group)
				}
				batches = append(batches, group[i:end])
			}
		}
	}

	return batches
}

// calculateOptimalConcurrency calculates optimal concurrency for a batch
func (s *UpdateService) calculateOptimalConcurrency(batch []*UpdateRequest) int {
	systemLoad := s.getSystemLoad(context.Background())

	baseConcurrency := 3
	if systemLoad != nil {
		if systemLoad.CPUUsage < 50 && systemLoad.MemoryUsage < 60 {
			baseConcurrency = 5
		} else if systemLoad.CPUUsage > 80 || systemLoad.MemoryUsage > 85 {
			baseConcurrency = 1
		}
	}

	// Don't exceed batch size
	if baseConcurrency > len(batch) {
		return len(batch)
	}

	return baseConcurrency
}

// SystemLoad represents current system load
type SystemLoad struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	LoadAvg     float64 `json:"load_avg"`
}

// getSystemLoad gets current system load
func (s *UpdateService) getSystemLoad(ctx context.Context) *SystemLoad {
	// This would integrate with monitoring service
	// For now, return mock data
	return &SystemLoad{
		CPUUsage:    45.0,
		MemoryUsage: 60.0,
		DiskUsage:   30.0,
		LoadAvg:     1.2,
	}
}

// getUpdatePriority gets the priority for a container update
func (s *UpdateService) getUpdatePriority(containerID string) UpdatePriority {
	s.updateScheduler.mu.RLock()
	defer s.updateScheduler.mu.RUnlock()

	if priority, exists := s.updateScheduler.updatePriorities[containerID]; exists {
		return priority
	}

	return PriorityNormal // Default priority
}

// waitForBatchCompletion waits for a batch of operations to complete
func (s *UpdateService) waitForBatchCompletion(operations []*UpdateOperation) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		allCompleted := true
		for _, op := range operations {
			op.mu.RLock()
			if op.Status == UpdateStatusPending || op.Status == UpdateStatusRunning {
				allCompleted = false
			}
			op.mu.RUnlock()
		}

		if allCompleted {
			break
		}

		<-ticker.C
	}
}

// Helper methods for maintenance windows

// isTimeInWindow checks if a time falls within a maintenance window
func (us *UpdateScheduler) isTimeInWindow(t time.Time, window *MaintenanceWindow) bool {
	if !window.Recurring {
		return t.After(window.StartTime) && t.Before(window.EndTime)
	}

	// Handle recurring windows
	switch window.RecurringType {
	case RecurringDaily:
		return us.isTimeInDailyWindow(t, window)
	case RecurringWeekly:
		return us.isTimeInWeeklyWindow(t, window)
	case RecurringMonthly:
		return us.isTimeInMonthlyWindow(t, window)
	}

	return false
}

// isTimeInDailyWindow checks if time is in daily recurring window
func (us *UpdateScheduler) isTimeInDailyWindow(t time.Time, window *MaintenanceWindow) bool {
	startHour := window.StartTime.Hour()
	startMin := window.StartTime.Minute()
	endHour := window.EndTime.Hour()
	endMin := window.EndTime.Minute()

	timeOfDay := t.Hour()*60 + t.Minute()
	startOfDay := startHour*60 + startMin
	endOfDay := endHour*60 + endMin

	if endOfDay > startOfDay {
		return timeOfDay >= startOfDay && timeOfDay <= endOfDay
	} else {
		// Window crosses midnight
		return timeOfDay >= startOfDay || timeOfDay <= endOfDay
	}
}

// isTimeInWeeklyWindow checks if time is in weekly recurring window
func (us *UpdateScheduler) isTimeInWeeklyWindow(t time.Time, window *MaintenanceWindow) bool {
	if t.Weekday() != window.StartTime.Weekday() {
		return false
	}
	return us.isTimeInDailyWindow(t, window)
}

// isTimeInMonthlyWindow checks if time is in monthly recurring window
func (us *UpdateScheduler) isTimeInMonthlyWindow(t time.Time, window *MaintenanceWindow) bool {
	if t.Day() != window.StartTime.Day() {
		return false
	}
	return us.isTimeInDailyWindow(t, window)
}

// countConcurrentUpdates counts concurrent updates at a given time
func (us *UpdateScheduler) countConcurrentUpdates(t time.Time, excludeContainer string) int {
	count := 0
	for _, scheduled := range us.scheduledUpdates {
		if scheduled.ContainerID == excludeContainer {
			continue
		}
		if scheduled.Status == ScheduledStatusExecuting {
			// Estimate if it would be running at time t
			estimatedDuration := 10 * time.Minute // Configurable
			if t.After(scheduled.ScheduledTime) && t.Before(scheduled.ScheduledTime.Add(estimatedDuration)) {
				count++
			}
		}
	}
	return count
}

// getNextWindowStart gets the next start time for a recurring window
func (us *UpdateScheduler) getNextWindowStart(after time.Time, window *MaintenanceWindow) time.Time {
	if !window.Recurring {
		if window.StartTime.After(after) {
			return window.StartTime
		}
		return time.Time{} // No future occurrence
	}

	switch window.RecurringType {
	case RecurringDaily:
		next := time.Date(after.Year(), after.Month(), after.Day(),
			window.StartTime.Hour(), window.StartTime.Minute(), 0, 0, after.Location())
		if next.Before(after) {
			next = next.Add(24 * time.Hour)
		}
		return next

	case RecurringWeekly:
		daysUntilWindow := int(window.StartTime.Weekday()) - int(after.Weekday())
		if daysUntilWindow <= 0 {
			daysUntilWindow += 7
		}
		next := after.Add(time.Duration(daysUntilWindow) * 24 * time.Hour)
		return time.Date(next.Year(), next.Month(), next.Day(),
			window.StartTime.Hour(), window.StartTime.Minute(), 0, 0, next.Location())

	case RecurringMonthly:
		next := time.Date(after.Year(), after.Month(), window.StartTime.Day(),
			window.StartTime.Hour(), window.StartTime.Minute(), 0, 0, after.Location())
		if next.Before(after) {
			next = next.AddDate(0, 1, 0)
		}
		return next
	}

	return time.Time{}
}

// Close gracefully shuts down the update service
func (s *UpdateService) Close() error {
	// Cancel all active operations
	s.updatesMu.Lock()
	for _, operation := range s.activeUpdates {
		if operation.cancel != nil {
			operation.cancel()
		}
	}
	s.updatesMu.Unlock()

	// Close update queue
	close(s.updateQueue)

	s.logger.Info("Update service shut down")
	return nil
}