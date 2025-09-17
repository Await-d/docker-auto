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
	"docker-auto/pkg/registry"
	"docker-auto/pkg/scheduler"
	// "docker-auto/pkg/scheduler/tasks" // Temporarily commented to fix import cycle

	"github.com/sirupsen/logrus"
)

// SchedulerService manages scheduled tasks and their execution
type SchedulerService struct {
	// Dependencies
	taskRepo            repository.ScheduledTaskRepository
	executionLogRepo    repository.TaskExecutionLogRepository
	containerRepo       repository.ContainerRepository
	updateHistoryRepo   repository.UpdateHistoryRepository
	activityLogRepo     repository.ActivityLogRepository
	imageVersionRepo    repository.ImageVersionRepository
	notificationRepo    repository.NotificationRepository
	containerService    *ContainerService
	imageService        *ImageService
	notificationService *NotificationService
	userService         *UserService
	dockerClient        *docker.DockerClient
	registryChecker     *registry.Checker
	config              *config.Config

	// Scheduler components
	scheduler      scheduler.Scheduler
	taskRegistry   scheduler.TaskRegistry
	taskExecutor   scheduler.TaskExecutor
	eventListener  *SchedulerEventListener

	// Internal state
	isRunning bool
	mu        sync.RWMutex
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(
	taskRepo repository.ScheduledTaskRepository,
	executionLogRepo repository.TaskExecutionLogRepository,
	containerRepo repository.ContainerRepository,
	updateHistoryRepo repository.UpdateHistoryRepository,
	activityLogRepo repository.ActivityLogRepository,
	imageVersionRepo repository.ImageVersionRepository,
	notificationRepo repository.NotificationRepository,
	containerService *ContainerService,
	imageService *ImageService,
	notificationService *NotificationService,
	userService *UserService,
	dockerClient *docker.DockerClient,
	registryChecker *registry.Checker,
	config *config.Config,
) *SchedulerService {
	service := &SchedulerService{
		taskRepo:            taskRepo,
		executionLogRepo:    executionLogRepo,
		containerRepo:       containerRepo,
		updateHistoryRepo:   updateHistoryRepo,
		activityLogRepo:     activityLogRepo,
		imageVersionRepo:    imageVersionRepo,
		notificationRepo:    notificationRepo,
		containerService:    containerService,
		imageService:        imageService,
		notificationService: notificationService,
		userService:         userService,
		dockerClient:        dockerClient,
		registryChecker:     registryChecker,
		config:              config,
	}

	// Initialize scheduler components
	service.taskRegistry = scheduler.NewTaskRegistry()
	service.taskExecutor = scheduler.NewTaskExecutor(config)
	service.eventListener = NewSchedulerEventListener(service)

	// Create scheduler instance
	schedulerConfig := &scheduler.SchedulerConfig{
		MaxConcurrentTasks: 10,
		TaskTimeout:        30 * time.Minute,
		RetryDelay:         5 * time.Minute,
		MaxRetries:         3,
		CleanupInterval:    1 * time.Hour,
		HistoryRetention:   24 * time.Hour,
		LogLevel:           "info",
		EnableMetrics:      true,
		TimeZone:           "UTC",
	}

	// Override with config values if available
	if config != nil && config.Scheduler != nil {
		if config.Scheduler.MaxConcurrentTasks > 0 {
			schedulerConfig.MaxConcurrentTasks = config.Scheduler.MaxConcurrentTasks
		}
		if config.Scheduler.TaskTimeout > 0 {
			schedulerConfig.TaskTimeout = config.Scheduler.TaskTimeout
		}
		if config.Scheduler.RetryDelay > 0 {
			schedulerConfig.RetryDelay = config.Scheduler.RetryDelay
		}
		if config.Scheduler.MaxRetries > 0 {
			schedulerConfig.MaxRetries = config.Scheduler.MaxRetries
		}
		if config.Scheduler.CleanupInterval > 0 {
			schedulerConfig.CleanupInterval = config.Scheduler.CleanupInterval
		}
		if config.Scheduler.HistoryRetention > 0 {
			schedulerConfig.HistoryRetention = config.Scheduler.HistoryRetention
		}
		if config.Scheduler.LogLevel != "" {
			schedulerConfig.LogLevel = config.Scheduler.LogLevel
		}
		schedulerConfig.EnableMetrics = config.Scheduler.EnableMetrics
		if config.Scheduler.TimeZone != "" {
			schedulerConfig.TimeZone = config.Scheduler.TimeZone
		}
	}

	service.scheduler = scheduler.NewCronScheduler(
		service.taskRegistry,
		service.taskExecutor,
		taskRepo,
		executionLogRepo,
		schedulerConfig,
		service.eventListener,
		[]scheduler.TaskHook{
			NewLoggingHook(),
			NewMetricsHook(),
		},
	)

	// Register task types
	// service.registerTaskTypes() // Temporarily commented to fix import cycle

	return service
}

// Start starts the scheduler service
func (s *SchedulerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler service is already running")
	}

	// Start the scheduler
	if err := s.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	s.isRunning = true

	logrus.Info("Scheduler service started successfully")
	return nil
}

// Stop stops the scheduler service
func (s *SchedulerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler service is not running")
	}

	// Stop the scheduler
	if err := s.scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}

	s.isRunning = false

	logrus.Info("Scheduler service stopped successfully")
	return nil
}

// IsRunning returns true if the scheduler service is running
func (s *SchedulerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// Task Management

// CreateTask creates a new scheduled task
func (s *SchedulerService) CreateTask(ctx context.Context, userID int64, req *CreateTaskRequest) (*model.ScheduledTask, error) {
	if req == nil {
		return nil, fmt.Errorf("create task request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if task name already exists
	tasks, _, err := s.taskRepo.List(ctx, &model.ScheduledTaskFilter{
		Name:  req.Name,
		Limit: 1,
	})
	if err == nil && len(tasks) > 0 {
		return nil, fmt.Errorf("task with name '%s' already exists", req.Name)
	}

	// Create task model
	task := &model.ScheduledTask{
		Name:             req.Name,
		Type:             req.Type,
		CronExpression:   req.CronExpression,
		TargetContainers: s.serializeTargetContainers(req.TargetContainers),
		Parameters:       s.serializeParameters(req.Parameters),
		IsActive:         req.IsActive,
		CreatedBy:        func() *int { u := int(userID); return &u }(),
	}

	// Validate cron expression
	if err := task.ValidateCronExpression(); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	if err := task.CalculateNextRun(); err != nil {
		return nil, fmt.Errorf("failed to calculate next run: %w", err)
	}

	// Save to database
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Add to scheduler if active and scheduler is running
	if task.IsActive && s.isRunning {
		if err := s.scheduler.AddTask(task); err != nil {
			logrus.WithError(err).WithField("task_id", task.ID).Warn("Failed to add task to scheduler")
		}
	}

	// Log activity
	s.logTaskActivity(userID, int64(task.ID), "task_created", "Scheduled task created", map[string]interface{}{
		"task_name": task.Name,
		"task_type": task.Type,
		"cron_expr": task.CronExpression,
	})

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"task_type": task.Type,
		"user_id":   userID,
	}).Info("Scheduled task created")

	return task, nil
}

// GetTask retrieves a scheduled task by ID
func (s *SchedulerService) GetTask(ctx context.Context, userID int64, taskID int64) (*TaskDetail, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return nil, err
	}

	// Get task status from scheduler
	status, err := s.scheduler.GetTaskStatus(int(task.ID))
	if err != nil {
		logrus.WithError(err).WithField("task_id", task.ID).Warn("Failed to get task status from scheduler")
	}

	// Get recent execution logs
	executions, _, err := s.executionLogRepo.GetByTaskID(ctx, int64(task.ID), 10, 0)
	if err != nil {
		logrus.WithError(err).WithField("task_id", task.ID).Warn("Failed to get execution logs")
	}

	detail := &TaskDetail{
		Task:            task,
		Status:          status,
		RecentExecutions: executions,
	}

	return detail, nil
}

// UpdateTask updates a scheduled task
func (s *SchedulerService) UpdateTask(ctx context.Context, userID int64, taskID int64, req *UpdateTaskRequest) error {
	if req == nil {
		return fmt.Errorf("update task request cannot be nil")
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return err
	}

	// Update fields
	updated := false
	changes := make(map[string]interface{})

	if req.CronExpression != nil && *req.CronExpression != task.CronExpression {
		task.CronExpression = *req.CronExpression
		changes["cron_expression"] = *req.CronExpression
		updated = true
	}

	if req.TargetContainers != nil {
		newTargets := s.serializeTargetContainers(*req.TargetContainers)
		if newTargets != task.TargetContainers {
			task.TargetContainers = newTargets
			changes["target_containers"] = *req.TargetContainers
			updated = true
		}
	}

	if req.Parameters != nil {
		newParams := s.serializeParameters(*req.Parameters)
		if newParams != task.Parameters {
			task.Parameters = newParams
			changes["parameters"] = *req.Parameters
			updated = true
		}
	}

	if req.IsActive != nil && *req.IsActive != task.IsActive {
		task.IsActive = *req.IsActive
		changes["is_active"] = *req.IsActive
		updated = true
	}

	if !updated {
		return nil // No changes made
	}

	// Validate cron expression if changed
	if _, exists := changes["cron_expression"]; exists {
		if err := task.ValidateCronExpression(); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// Recalculate next run if needed
	if err := task.CalculateNextRun(); err != nil {
		return fmt.Errorf("failed to calculate next run: %w", err)
	}

	// Update in database
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Update in scheduler
	if s.isRunning {
		if err := s.scheduler.UpdateTask(task); err != nil {
			logrus.WithError(err).WithField("task_id", task.ID).Warn("Failed to update task in scheduler")
		}
	}

	// Log activity
	s.logTaskActivity(userID, int64(task.ID), "task_updated", "Scheduled task updated", changes)

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"user_id":   userID,
		"changes":   changes,
	}).Info("Scheduled task updated")

	return nil
}

// DeleteTask deletes a scheduled task
func (s *SchedulerService) DeleteTask(ctx context.Context, userID int64, taskID int64) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return err
	}

	// Remove from scheduler
	if s.isRunning {
		if err := s.scheduler.RemoveTask(int(task.ID)); err != nil {
			logrus.WithError(err).WithField("task_id", task.ID).Warn("Failed to remove task from scheduler")
		}
	}

	// Delete from database
	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Log activity
	s.logTaskActivity(userID, int64(task.ID), "task_deleted", "Scheduled task deleted", map[string]interface{}{
		"task_name": task.Name,
		"task_type": task.Type,
	})

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"user_id":   userID,
	}).Info("Scheduled task deleted")

	return nil
}

// ListTasks lists scheduled tasks with filtering
func (s *SchedulerService) ListTasks(ctx context.Context, userID int64, filter *TaskFilter) (*TaskListResponse, error) {
	if filter == nil {
		filter = &TaskFilter{}
	}

	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Build database filter
	dbFilter := &model.ScheduledTaskFilter{
		Name:      filter.Name,
		Type:      filter.Type,
		IsActive:  filter.IsActive,
		CreatedBy: func() *int { u := int(userID); return &u }(), // Filter by user
		Limit:     filter.Limit,
		Offset:    filter.Offset,
		OrderBy:   "updated_at DESC",
	}

	// Get tasks from database
	tasks, total, err := s.taskRepo.List(ctx, dbFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Convert to summaries with status information
	summaries := make([]*TaskSummary, len(tasks))
	for i, task := range tasks {
		summary := &TaskSummary{
			ID:             int64(task.ID),
			Name:           task.Name,
			Type:           task.Type,
			CronExpression: task.CronExpression,
			IsActive:       task.IsActive,
			LastRunAt:      task.LastRunAt,
			NextRunAt:      task.NextRunAt,
			RunCount:       task.RunCount,
			FailureCount:   task.FailureCount,
			CreatedAt:      task.CreatedAt,
			UpdatedAt:      task.UpdatedAt,
		}

		// Get status from scheduler if running
		if s.isRunning {
			if status, err := s.scheduler.GetTaskStatus(int(task.ID)); err == nil {
				summary.IsRunning = status.IsRunning
				summary.IsPaused = status.IsPaused
				summary.SuccessRate = status.SuccessRate
			}
		}

		summaries[i] = summary
	}

	// Calculate pagination
	page := (filter.Offset / filter.Limit) + 1
	hasNext := filter.Offset+filter.Limit < int(total)
	hasPrev := filter.Offset > 0

	return &TaskListResponse{
		Tasks:   summaries,
		Total:   total,
		Page:    page,
		Limit:   filter.Limit,
		HasNext: hasNext,
		HasPrev: hasPrev,
	}, nil
}

// Task Control Operations

// PauseTask pauses a scheduled task
func (s *SchedulerService) PauseTask(ctx context.Context, userID int64, taskID int64) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return err
	}

	// Pause in scheduler
	if s.isRunning {
		if err := s.scheduler.PauseTask(int(task.ID)); err != nil {
			return fmt.Errorf("failed to pause task in scheduler: %w", err)
		}
	}

	// Update database
	task.IsActive = false
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Log activity
	s.logTaskActivity(userID, int64(task.ID), "task_paused", "Scheduled task paused", nil)

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"user_id":   userID,
	}).Info("Scheduled task paused")

	return nil
}

// ResumeTask resumes a paused task
func (s *SchedulerService) ResumeTask(ctx context.Context, userID int64, taskID int64) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return err
	}

	// Resume in scheduler
	if s.isRunning {
		if err := s.scheduler.ResumeTask(int(task.ID)); err != nil {
			return fmt.Errorf("failed to resume task in scheduler: %w", err)
		}
	}

	// Update database
	task.IsActive = true
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Log activity
	s.logTaskActivity(userID, int64(task.ID), "task_resumed", "Scheduled task resumed", nil)

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"user_id":   userID,
	}).Info("Scheduled task resumed")

	return nil
}

// TriggerTask manually triggers a task execution
func (s *SchedulerService) TriggerTask(ctx context.Context, userID int64, taskID int64) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return err
	}

	// Trigger in scheduler
	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	if err := s.scheduler.TriggerTask(int(task.ID)); err != nil {
		return fmt.Errorf("failed to trigger task: %w", err)
	}

	// Log activity
	s.logTaskActivity(userID, int64(task.ID), "task_triggered", "Scheduled task triggered manually", nil)

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"user_id":   userID,
	}).Info("Scheduled task triggered manually")

	return nil
}

// Monitoring and Status

// GetSchedulerStatus returns the current scheduler status
func (s *SchedulerService) GetSchedulerStatus(ctx context.Context) (*SchedulerStatus, error) {
	status := &SchedulerStatus{
		IsRunning: s.isRunning,
		Timestamp: time.Now(),
	}

	if s.isRunning {
		// Get running tasks
		runningTasks := s.scheduler.GetRunningTasks()
		status.RunningTasks = len(runningTasks)

		// Get metrics if available - this would need to be implemented in the scheduler interface
		// For now, we'll leave it empty
		status.Metrics = &scheduler.SchedulerMetrics{
			TotalTasks:   status.TotalTasks,
			ActiveTasks:  status.ActiveTasks,
			RunningTasks: status.RunningTasks,
		}

		// Get task counts
		activeTasks, err := s.taskRepo.GetActiveTasks(ctx)
		if err == nil {
			status.ActiveTasks = len(activeTasks)
		}

		totalTasks, _, err := s.taskRepo.List(ctx, &model.ScheduledTaskFilter{Limit: 1})
		if err == nil {
			status.TotalTasks = len(totalTasks)
		}
	}

	return status, nil
}

// GetTaskExecutions returns execution history for a task
func (s *SchedulerService) GetTaskExecutions(ctx context.Context, userID int64, taskID int64, filter *ExecutionFilter) (*ExecutionListResponse, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Check permissions
	if err := s.checkTaskPermission(task, userID); err != nil {
		return nil, err
	}

	if filter == nil {
		filter = &ExecutionFilter{}
	}

	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Build database filter
	dbFilter := &model.TaskExecutionLogFilter{
		TaskID: func() *int { t := int(taskID); return &t }(),
		Status: filter.Status,
		Limit:  filter.Limit,
		Offset: filter.Offset,
		OrderBy: "started_at DESC",
	}

	if !filter.StartedAfter.IsZero() {
		dbFilter.StartedAfter = &filter.StartedAfter
	}
	if !filter.StartedBefore.IsZero() {
		dbFilter.StartedBefore = &filter.StartedBefore
	}

	// Get executions from database
	executions, total, err := s.executionLogRepo.List(ctx, dbFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	// Calculate pagination
	page := (filter.Offset / filter.Limit) + 1
	hasNext := filter.Offset+filter.Limit < int(total)
	hasPrev := filter.Offset > 0

	return &ExecutionListResponse{
		Executions: executions,
		Total:      total,
		Page:       page,
		Limit:      filter.Limit,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}, nil
}

// Helper methods

// registerTaskTypes registers available task implementations
// Temporarily commented to fix import cycle
/*
func (s *SchedulerService) registerTaskTypes() {
	// Register image update checker
	s.taskRegistry.RegisterTask(model.TaskTypeImageCheck, func() scheduler.Task {
		return tasks.NewUpdateCheckerTask(
			s.containerRepo,
			s.imageVersionRepo,
			s.registryChecker,
			s.containerService,
			s.imageService,
			s.notificationService,
		)
	})

	// Register container updater
	s.taskRegistry.RegisterTask(model.TaskTypeContainerUpdate, func() scheduler.Task {
		return tasks.NewContainerUpdaterTask(
			s.containerRepo,
			s.updateHistoryRepo,
			s.containerService,
			s.imageService,
			s.notificationService,
			s.dockerClient,
		)
	})

	// Register cleanup task
	s.taskRegistry.RegisterTask(model.TaskTypeCleanup, func() scheduler.Task {
		return tasks.NewCleanupTask(
			s.containerRepo,
			s.updateHistoryRepo,
			s.executionLogRepo,
			s.activityLogRepo,
			s.imageVersionRepo,
			s.notificationRepo,
			s.containerService,
			s.notificationService,
			s.dockerClient,
		)
	})

	// Register health checker
	s.taskRegistry.RegisterTask(model.TaskTypeHealthCheck, func() scheduler.Task {
		return tasks.NewHealthCheckerTask(
			s.containerRepo,
			s.containerService,
			s.notificationService,
			s.dockerClient,
		)
	})

	// Register backup task
	s.taskRegistry.RegisterTask(model.TaskTypeBackup, func() scheduler.Task {
		return tasks.NewBackupTask(
			s.containerRepo,
			s.updateHistoryRepo,
			s.taskRepo,
			s.containerService,
			s.notificationService,
			s.dockerClient,
		)
	})

	logrus.Info("Registered all task types")
}
*/

// checkTaskPermission checks if user has permission to access task
func (s *SchedulerService) checkTaskPermission(task *model.ScheduledTask, userID int64) error {
	// Admin users can access all tasks
	user, err := s.userService.GetUserByID(context.Background(), userID)
	if err == nil && user.IsAdmin() {
		return nil
	}

	// Users can only access their own tasks
	if task.CreatedBy == nil || int64(*task.CreatedBy) != userID {
		return fmt.Errorf("access denied: user %d cannot access task %d", userID, task.ID)
	}

	return nil
}

// logTaskActivity logs task-related activity
func (s *SchedulerService) logTaskActivity(userID, taskID int64, action, description string, metadata map[string]interface{}) {
	if s.activityLogRepo == nil {
		return
	}

	metadataJSON := "{}"
	if metadata != nil {
		if jsonData, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(jsonData)
		}
	}

	log := &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: "scheduled_task",
		ResourceID:   func() *int { r := int(taskID); return &r }(),
		Description:  description,
		Metadata:     metadataJSON,
	}

	if err := s.activityLogRepo.Create(context.Background(), log); err != nil {
		logrus.WithError(err).Warn("Failed to log task activity")
	}
}

// serializeTargetContainers serializes target container IDs to JSON
func (s *SchedulerService) serializeTargetContainers(containers []int64) string {
	if len(containers) == 0 {
		return "[]"
	}

	jsonData, err := json.Marshal(containers)
	if err != nil {
		logrus.WithError(err).Warn("Failed to serialize target containers")
		return "[]"
	}

	return string(jsonData)
}

// serializeParameters serializes task parameters to JSON
func (s *SchedulerService) serializeParameters(params map[string]interface{}) string {
	if len(params) == 0 {
		return "{}"
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		logrus.WithError(err).Warn("Failed to serialize task parameters")
		return "{}"
	}

	return string(jsonData)
}

// Type definitions for requests and responses

// CreateTaskRequest represents a request to create a scheduled task
type CreateTaskRequest struct {
	Name             string                 `json:"name" binding:"required"`
	Type             model.TaskType         `json:"type" binding:"required"`
	CronExpression   string                 `json:"cron_expression" binding:"required"`
	TargetContainers []int64                `json:"target_containers,omitempty"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
	IsActive         bool                   `json:"is_active"`
}

// Validate validates the create task request
func (r *CreateTaskRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if r.CronExpression == "" {
		return fmt.Errorf("cron expression is required")
	}
	// Additional validation can be added here
	return nil
}

// UpdateTaskRequest represents a request to update a scheduled task
type UpdateTaskRequest struct {
	CronExpression   *string                 `json:"cron_expression,omitempty"`
	TargetContainers *[]int64                `json:"target_containers,omitempty"`
	Parameters       *map[string]interface{} `json:"parameters,omitempty"`
	IsActive         *bool                   `json:"is_active,omitempty"`
}

// TaskFilter represents filters for listing tasks
type TaskFilter struct {
	Name     string          `json:"name,omitempty"`
	Type     model.TaskType  `json:"type,omitempty"`
	IsActive *bool           `json:"is_active,omitempty"`
	Limit    int             `json:"limit,omitempty"`
	Offset   int             `json:"offset,omitempty"`
}

// ExecutionFilter represents filters for listing task executions
type ExecutionFilter struct {
	Status        model.ExecutionStatus `json:"status,omitempty"`
	StartedAfter  time.Time             `json:"started_after,omitempty"`
	StartedBefore time.Time             `json:"started_before,omitempty"`
	Limit         int                   `json:"limit,omitempty"`
	Offset        int                   `json:"offset,omitempty"`
}

// Response types

// TaskDetail represents detailed information about a task
type TaskDetail struct {
	Task             *model.ScheduledTask        `json:"task"`
	Status           *scheduler.TaskStatus       `json:"status,omitempty"`
	RecentExecutions []*model.TaskExecutionLog   `json:"recent_executions,omitempty"`
}

// TaskSummary represents summary information about a task
type TaskSummary struct {
	ID             int64              `json:"id"`
	Name           string             `json:"name"`
	Type           model.TaskType     `json:"type"`
	CronExpression string             `json:"cron_expression"`
	IsActive       bool               `json:"is_active"`
	IsRunning      bool               `json:"is_running"`
	IsPaused       bool               `json:"is_paused"`
	LastRunAt      *time.Time         `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time         `json:"next_run_at,omitempty"`
	RunCount       int                `json:"run_count"`
	FailureCount   int                `json:"failure_count"`
	SuccessRate    float64            `json:"success_rate"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// TaskListResponse represents the response for listing tasks
type TaskListResponse struct {
	Tasks   []*TaskSummary `json:"tasks"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	Limit   int            `json:"limit"`
	HasNext bool           `json:"has_next"`
	HasPrev bool           `json:"has_prev"`
}

// ExecutionListResponse represents the response for listing task executions
type ExecutionListResponse struct {
	Executions []*model.TaskExecutionLog `json:"executions"`
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	Limit      int                       `json:"limit"`
	HasNext    bool                      `json:"has_next"`
	HasPrev    bool                      `json:"has_prev"`
}

// SchedulerStatus represents the current status of the scheduler
type SchedulerStatus struct {
	IsRunning    bool                      `json:"is_running"`
	TotalTasks   int                       `json:"total_tasks"`
	ActiveTasks  int                       `json:"active_tasks"`
	RunningTasks int                       `json:"running_tasks"`
	Metrics      *scheduler.SchedulerMetrics `json:"metrics,omitempty"`
	Timestamp    time.Time                 `json:"timestamp"`
}