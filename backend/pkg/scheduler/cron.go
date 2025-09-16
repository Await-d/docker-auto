package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// cronLoggerWrapper adapts logrus.Logger to cron.Logger interface
type cronLoggerWrapper struct {
	logger *logrus.Logger
}

func (w *cronLoggerWrapper) Error(err error, msg string, keysAndValues ...interface{}) {
	fields := logrus.Fields{}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			fields[fmt.Sprintf("%v", keysAndValues[i])] = keysAndValues[i+1]
		}
	}
	w.logger.WithFields(fields).WithError(err).Error(msg)
}

func (w *cronLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	fields := logrus.Fields{}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			fields[fmt.Sprintf("%v", keysAndValues[i])] = keysAndValues[i+1]
		}
	}
	w.logger.WithFields(fields).Info(msg)
}

// CronScheduler implements the Scheduler interface using robfig/cron
type CronScheduler struct {
	cron          *cron.Cron
	taskRegistry  TaskRegistry
	taskExecutor  TaskExecutor
	taskRepo      repository.ScheduledTaskRepository
	executionRepo repository.TaskExecutionLogRepository
	config        *SchedulerConfig
	eventListener EventListener
	hooks         []TaskHook

	// Internal state
	isRunning        bool
	tasks            map[int]*scheduledTaskEntry
	executions       map[string]*TaskExecution
	cronEntries      map[int]cron.EntryID
	mu               sync.RWMutex
	cancelCtx        context.Context
	cancelFunc       context.CancelFunc
	workerPool       chan struct{}
	cleanupTicker    *time.Ticker
	metrics          *SchedulerMetrics
	startTime        time.Time
}

// scheduledTaskEntry represents a task entry in the scheduler
type scheduledTaskEntry struct {
	task      *model.ScheduledTask
	cronEntry cron.EntryID
	isPaused  bool
	lastRun   *time.Time
	runCount  int
	errorCount int
}

// NewCronScheduler creates a new cron-based scheduler
func NewCronScheduler(
	taskRegistry TaskRegistry,
	taskExecutor TaskExecutor,
	taskRepo repository.ScheduledTaskRepository,
	executionRepo repository.TaskExecutionLogRepository,
	config *SchedulerConfig,
	eventListener EventListener,
	hooks []TaskHook,
) *CronScheduler {
	if config == nil {
		config = &SchedulerConfig{
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
	}

	// Create timezone location
	location, err := time.LoadLocation(config.TimeZone)
	if err != nil {
		logrus.WithError(err).Warn("Invalid timezone, using UTC")
		location = time.UTC
	}

	// Create cron scheduler with timezone support
	cronScheduler := cron.New(
		cron.WithLocation(location),
		cron.WithSeconds(),
		cron.WithChain(cron.Recover(&cronLoggerWrapper{logger: logrus.StandardLogger()})),
	)

	return &CronScheduler{
		cron:          cronScheduler,
		taskRegistry:  taskRegistry,
		taskExecutor:  taskExecutor,
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		config:        config,
		eventListener: eventListener,
		hooks:         hooks,
		tasks:         make(map[int]*scheduledTaskEntry),
		executions:    make(map[string]*TaskExecution),
		cronEntries:   make(map[int]cron.EntryID),
		workerPool:    make(chan struct{}, config.MaxConcurrentTasks),
		metrics: &SchedulerMetrics{
			UptimeSeconds: 0,
		},
	}
}

// Start starts the scheduler
func (s *CronScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	// Create cancellation context
	s.cancelCtx, s.cancelFunc = context.WithCancel(ctx)
	s.startTime = time.Now()

	// Load existing tasks from database
	if err := s.loadTasksFromDatabase(s.cancelCtx); err != nil {
		return fmt.Errorf("failed to load tasks from database: %w", err)
	}

	// Start the cron scheduler
	s.cron.Start()

	// Start cleanup ticker
	s.cleanupTicker = time.NewTicker(s.config.CleanupInterval)
	go s.cleanupRoutine()

	// Start metrics updater
	if s.config.EnableMetrics {
		go s.metricsRoutine()
	}

	s.isRunning = true

	logrus.Info("Cron scheduler started successfully")
	s.publishEvent(EventSchedulerStarted, nil, "Scheduler started", nil)

	return nil
}

// Stop stops the scheduler gracefully
func (s *CronScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	logrus.Info("Stopping cron scheduler...")

	// Stop accepting new tasks
	s.isRunning = false

	// Stop cron scheduler
	cronCtx := s.cron.Stop()

	// Cancel all running tasks
	s.cancelFunc()

	// Stop cleanup ticker
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	// Wait for cron to stop
	select {
	case <-cronCtx.Done():
		logrus.Info("Cron scheduler stopped")
	case <-ctx.Done():
		logrus.Warn("Context cancelled while waiting for cron to stop")
	}

	// Wait for running tasks to complete or timeout
	s.waitForRunningTasks(ctx)

	s.publishEvent(EventSchedulerStopped, nil, "Scheduler stopped", nil)

	logrus.Info("Cron scheduler stopped successfully")
	return nil
}

// AddTask adds a new task to the scheduler
func (s *CronScheduler) AddTask(task *model.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %d already exists", task.ID)
	}

	// Validate cron expression
	if err := task.ValidateCronExpression(); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Add to cron scheduler
	entryID, err := s.cron.AddFunc(task.CronExpression, s.createTaskRunner(task))
	if err != nil {
		return fmt.Errorf("failed to add task to cron: %w", err)
	}

	// Store task entry
	s.tasks[task.ID] = &scheduledTaskEntry{
		task:      task,
		cronEntry: entryID,
		isPaused:  !task.IsActive,
	}
	s.cronEntries[task.ID] = entryID

	s.metrics.TotalTasks++
	if task.IsActive {
		s.metrics.ActiveTasks++
	}

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"task_type": task.Type,
		"cron_expr": task.CronExpression,
	}).Info("Task added to scheduler")

	s.publishEvent(EventTaskAdded, &task.ID, fmt.Sprintf("Task '%s' added", task.Name), map[string]interface{}{
		"task_type": task.Type,
		"cron_expr": task.CronExpression,
	})

	return nil
}

// RemoveTask removes a task from the scheduler
func (s *CronScheduler) RemoveTask(taskID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %d not found", taskID)
	}

	// Remove from cron scheduler
	s.cron.Remove(entry.cronEntry)

	// Cancel running execution if any
	for _, execution := range s.executions {
		if execution.TaskID == taskID && execution.Status == model.ExecutionStatusRunning {
			if execution.CancelFunc != nil {
				execution.CancelFunc()
			}
		}
	}

	// Remove from internal maps
	delete(s.tasks, taskID)
	delete(s.cronEntries, taskID)

	s.metrics.TotalTasks--
	if entry.task.IsActive {
		s.metrics.ActiveTasks--
	}

	logrus.WithFields(logrus.Fields{
		"task_id":   taskID,
		"task_name": entry.task.Name,
	}).Info("Task removed from scheduler")

	s.publishEvent(EventTaskRemoved, &taskID, fmt.Sprintf("Task '%s' removed", entry.task.Name), nil)

	return nil
}

// UpdateTask updates an existing task
func (s *CronScheduler) UpdateTask(task *model.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.tasks[task.ID]
	if !exists {
		return fmt.Errorf("task with ID %d not found", task.ID)
	}

	// Validate new cron expression
	if err := task.ValidateCronExpression(); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Remove old entry
	s.cron.Remove(entry.cronEntry)

	// Add new entry with updated task
	entryID, err := s.cron.AddFunc(task.CronExpression, s.createTaskRunner(task))
	if err != nil {
		// Re-add old entry on failure
		oldEntryID, _ := s.cron.AddFunc(entry.task.CronExpression, s.createTaskRunner(entry.task))
		entry.cronEntry = oldEntryID
		return fmt.Errorf("failed to update task in cron: %w", err)
	}

	// Update task entry
	oldActive := entry.task.IsActive
	entry.task = task
	entry.cronEntry = entryID
	entry.isPaused = !task.IsActive
	s.cronEntries[task.ID] = entryID

	// Update metrics
	if oldActive != task.IsActive {
		if task.IsActive {
			s.metrics.ActiveTasks++
		} else {
			s.metrics.ActiveTasks--
		}
	}

	logrus.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"task_type": task.Type,
		"cron_expr": task.CronExpression,
	}).Info("Task updated in scheduler")

	s.publishEvent(EventTaskUpdated, &task.ID, fmt.Sprintf("Task '%s' updated", task.Name), map[string]interface{}{
		"task_type": task.Type,
		"cron_expr": task.CronExpression,
	})

	return nil
}

// PauseTask pauses a specific task
func (s *CronScheduler) PauseTask(taskID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %d not found", taskID)
	}

	if entry.isPaused {
		return fmt.Errorf("task with ID %d is already paused", taskID)
	}

	entry.isPaused = true
	s.metrics.ActiveTasks--

	logrus.WithFields(logrus.Fields{
		"task_id":   taskID,
		"task_name": entry.task.Name,
	}).Info("Task paused")

	s.publishEvent(EventTaskPaused, &taskID, fmt.Sprintf("Task '%s' paused", entry.task.Name), nil)

	return nil
}

// ResumeTask resumes a paused task
func (s *CronScheduler) ResumeTask(taskID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %d not found", taskID)
	}

	if !entry.isPaused {
		return fmt.Errorf("task with ID %d is not paused", taskID)
	}

	entry.isPaused = false
	s.metrics.ActiveTasks++

	logrus.WithFields(logrus.Fields{
		"task_id":   taskID,
		"task_name": entry.task.Name,
	}).Info("Task resumed")

	s.publishEvent(EventTaskResumed, &taskID, fmt.Sprintf("Task '%s' resumed", entry.task.Name), nil)

	return nil
}

// TriggerTask manually triggers a task execution
func (s *CronScheduler) TriggerTask(taskID int) error {
	s.mu.RLock()
	entry, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task with ID %d not found", taskID)
	}

	// Execute task immediately
	go s.executeTask(entry.task)

	logrus.WithFields(logrus.Fields{
		"task_id":   taskID,
		"task_name": entry.task.Name,
	}).Info("Task triggered manually")

	return nil
}

// GetRunningTasks returns currently running tasks
func (s *CronScheduler) GetRunningTasks() []*TaskExecution {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var running []*TaskExecution
	for _, execution := range s.executions {
		if execution.Status == model.ExecutionStatusRunning {
			running = append(running, execution)
		}
	}

	return running
}

// GetTaskStatus returns the status of a specific task
func (s *CronScheduler) GetTaskStatus(taskID int) (*TaskStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task with ID %d not found", taskID)
	}

	// Find current execution
	var currentExecution *TaskExecution
	for _, execution := range s.executions {
		if execution.TaskID == taskID && execution.Status == model.ExecutionStatusRunning {
			currentExecution = execution
			break
		}
	}

	// Calculate success rate
	successRate := float64(0)
	if entry.runCount > 0 {
		successCount := entry.runCount - entry.errorCount
		successRate = float64(successCount) / float64(entry.runCount) * 100
	}

	status := &TaskStatus{
		TaskID:           taskID,
		Name:             entry.task.Name,
		Type:             entry.task.Type,
		IsActive:         entry.task.IsActive,
		IsPaused:         entry.isPaused,
		IsRunning:        currentExecution != nil,
		LastRun:          entry.lastRun,
		NextRun:          entry.task.NextRunAt,
		RunCount:         entry.runCount,
		FailureCount:     entry.errorCount,
		SuccessRate:      successRate,
		CurrentExecution: currentExecution,
	}

	return status, nil
}

// IsRunning returns true if the scheduler is running
func (s *CronScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// createTaskRunner creates a function that will be called by cron
func (s *CronScheduler) createTaskRunner(task *model.ScheduledTask) func() {
	return func() {
		s.mu.RLock()
		entry := s.tasks[task.ID]
		s.mu.RUnlock()

		if entry == nil || entry.isPaused || !entry.task.IsActive {
			return
		}

		s.executeTask(task)
	}
}

// executeTask executes a task
func (s *CronScheduler) executeTask(task *model.ScheduledTask) {
	// Acquire worker slot
	select {
	case s.workerPool <- struct{}{}:
		defer func() { <-s.workerPool }()
	case <-s.cancelCtx.Done():
		return
	}

	executionID := uuid.New().String()
	ctx, cancel := context.WithTimeout(s.cancelCtx, s.config.TaskTimeout)
	defer cancel()

	// Create execution record
	execution := &TaskExecution{
		ID:         executionID,
		TaskID:     task.ID,
		TaskName:   task.Name,
		TaskType:   task.Type,
		Status:     model.ExecutionStatusRunning,
		StartedAt:  time.Now(),
		Progress:   0,
		CancelFunc: cancel,
	}

	// Store execution
	s.mu.Lock()
	s.executions[executionID] = execution
	s.metrics.RunningTasks++
	s.mu.Unlock()

	// Update task entry
	s.mu.Lock()
	if entry := s.tasks[task.ID]; entry != nil {
		entry.runCount++
		now := time.Now()
		entry.lastRun = &now
	}
	s.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"execution_id": executionID,
		"task_id":      task.ID,
		"task_name":    task.Name,
		"task_type":    task.Type,
	}).Info("Task execution started")

	s.publishEvent(EventTaskStarted, &task.ID, fmt.Sprintf("Task '%s' started", task.Name), map[string]interface{}{
		"execution_id": executionID,
	})

	// Execute task with hooks
	result := s.executeTaskWithHooks(ctx, execution, task)

	// Update execution
	s.mu.Lock()
	execution.CompletedAt = &result.CompletedAt
	execution.Duration = result.Duration
	execution.Result = &result.TaskResult
	execution.Status = result.Status
	if result.Error != nil {
		execution.Error = result.Error.Error()
	}
	s.metrics.RunningTasks--
	s.metrics.TotalExecutions++
	if result.Status == model.ExecutionStatusSuccess {
		s.metrics.SuccessfulExecutions++
	} else {
		s.metrics.FailedExecutions++
	}
	s.mu.Unlock()

	// Update task failure count
	if result.Status != model.ExecutionStatusSuccess {
		s.mu.Lock()
		if entry := s.tasks[task.ID]; entry != nil {
			entry.errorCount++
		}
		s.mu.Unlock()
	}

	// Save execution log to database
	s.saveExecutionLog(execution, result)

	// Remove from active executions after some time
	go func() {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		delete(s.executions, executionID)
		s.mu.Unlock()
	}()

	eventType := EventTaskCompleted
	if result.Status == model.ExecutionStatusFailed {
		eventType = EventTaskFailed
	} else if result.Status == model.ExecutionStatusTimeout {
		eventType = EventTaskTimeout
	}

	s.publishEvent(eventType, &task.ID, fmt.Sprintf("Task '%s' %s", task.Name, result.Status), map[string]interface{}{
		"execution_id": executionID,
		"duration":     result.Duration.String(),
		"success":      result.Status == model.ExecutionStatusSuccess,
	})

	logrus.WithFields(logrus.Fields{
		"execution_id": executionID,
		"task_id":      task.ID,
		"task_name":    task.Name,
		"task_type":    task.Type,
		"status":       result.Status,
		"duration":     result.Duration,
		"success":      result.Status == model.ExecutionStatusSuccess,
	}).Info("Task execution completed")
}

// taskExecutionResult represents the result of task execution
type taskExecutionResult struct {
	TaskResult
	Status      model.ExecutionStatus
	CompletedAt time.Time
}

// executeTaskWithHooks executes a task with hooks
func (s *CronScheduler) executeTaskWithHooks(ctx context.Context, execution *TaskExecution, task *model.ScheduledTask) taskExecutionResult {
	startTime := time.Now()
	result := taskExecutionResult{
		CompletedAt: time.Now(),
		Status:      model.ExecutionStatusFailed,
	}

	// Execute before hooks
	for _, hook := range s.hooks {
		if err := hook.BeforeExecution(ctx, execution); err != nil {
			logrus.WithError(err).Warn("Before execution hook failed")
		}
	}

	// Get task implementation
	taskImpl, err := s.taskRegistry.GetTask(task.Type)
	if err != nil {
		result.TaskResult = TaskResult{
			Success: false,
			Error:   fmt.Errorf("failed to get task implementation: %w", err),
			Duration: time.Since(startTime),
		}
		result.Status = model.ExecutionStatusFailed
		result.CompletedAt = time.Now()
		return result
	}

	// Parse task parameters
	params, err := s.parseTaskParameters(task)
	if err != nil {
		result.TaskResult = TaskResult{
			Success: false,
			Error:   fmt.Errorf("failed to parse task parameters: %w", err),
			Duration: time.Since(startTime),
		}
		result.Status = model.ExecutionStatusFailed
		result.CompletedAt = time.Now()
		return result
	}

	// Execute task
	taskResult, err := s.taskExecutor.ExecuteTask(ctx, taskImpl, *params)
	if err != nil {
		result.TaskResult = TaskResult{
			Success: false,
			Error:   err,
			Duration: time.Since(startTime),
		}
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = model.ExecutionStatusTimeout
		} else {
			result.Status = model.ExecutionStatusFailed
		}
	} else {
		result.TaskResult = *taskResult
		if taskResult.Success {
			result.Status = model.ExecutionStatusSuccess
		} else {
			result.Status = model.ExecutionStatusFailed
		}
	}

	result.CompletedAt = time.Now()

	// Execute after hooks
	for _, hook := range s.hooks {
		if err := hook.AfterExecution(ctx, execution, &result.TaskResult); err != nil {
			logrus.WithError(err).Warn("After execution hook failed")
		}
	}

	// Execute error hooks if needed
	if result.TaskResult.Error != nil {
		for _, hook := range s.hooks {
			if err := hook.OnError(ctx, execution, result.TaskResult.Error); err != nil {
				logrus.WithError(err).Warn("Error hook failed")
			}
		}
	}

	return result
}

// parseTaskParameters parses task parameters from JSON
func (s *CronScheduler) parseTaskParameters(task *model.ScheduledTask) (*TaskParameters, error) {
	// TODO: Implement parameter parsing based on task type
	params := &TaskParameters{
		TaskType: task.Type,
		Timeout:  s.config.TaskTimeout,
		MaxRetries: s.config.MaxRetries,
		RetryDelay: s.config.RetryDelay,
	}

	// Parse target containers
	// Implementation depends on how target containers are stored in the database

	return params, nil
}

// saveExecutionLog saves task execution log to database
func (s *CronScheduler) saveExecutionLog(execution *TaskExecution, result taskExecutionResult) {
	if s.executionRepo == nil {
		return
	}

	logEntry := &model.TaskExecutionLog{
		TaskID:          execution.TaskID,
		Status:          result.Status,
		Message:         result.Message,
		DurationSeconds: int(result.Duration.Seconds()),
		StartedAt:       execution.StartedAt,
		CompletedAt:     &result.CompletedAt,
	}

	if err := s.executionRepo.Create(context.Background(), logEntry); err != nil {
		logrus.WithError(err).WithField("execution_id", execution.ID).Error("Failed to save execution log")
	}
}

// loadTasksFromDatabase loads existing tasks from database
func (s *CronScheduler) loadTasksFromDatabase(ctx context.Context) error {
	if s.taskRepo == nil {
		return nil
	}

	tasks, _, err := s.taskRepo.List(ctx, &model.ScheduledTaskFilter{
		IsActive: boolPtr(true),
		Limit:    1000, // Load up to 1000 active tasks
	})
	if err != nil {
		return fmt.Errorf("failed to load tasks from database: %w", err)
	}

	for _, task := range tasks {
		if err := s.AddTask(task); err != nil {
			logrus.WithError(err).WithField("task_id", task.ID).Error("Failed to add task from database")
		}
	}

	logrus.WithField("task_count", len(tasks)).Info("Loaded tasks from database")
	return nil
}

// waitForRunningTasks waits for running tasks to complete
func (s *CronScheduler) waitForRunningTasks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Warn("Context cancelled while waiting for running tasks")
			return
		case <-ticker.C:
			s.mu.RLock()
			runningCount := len(s.executions)
			s.mu.RUnlock()

			if runningCount == 0 {
				return
			}

			logrus.WithField("running_tasks", runningCount).Info("Waiting for running tasks to complete")
		}
	}
}

// cleanupRoutine periodically cleans up old execution records
func (s *CronScheduler) cleanupRoutine() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.cleanupOldExecutions()
		case <-s.cancelCtx.Done():
			return
		}
	}
}

// cleanupOldExecutions removes old execution records
func (s *CronScheduler) cleanupOldExecutions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.config.HistoryRetention)

	for id, execution := range s.executions {
		if execution.CompletedAt != nil && execution.CompletedAt.Before(cutoff) {
			delete(s.executions, id)
		}
	}
}

// metricsRoutine updates scheduler metrics
func (s *CronScheduler) metricsRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateMetrics()
		case <-s.cancelCtx.Done():
			return
		}
	}
}

// updateMetrics updates scheduler metrics
func (s *CronScheduler) updateMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
	s.metrics.QueueDepth = len(s.workerPool)
	s.metrics.WorkerUtilization = float64(len(s.workerPool)) / float64(cap(s.workerPool)) * 100

	// Calculate average execution time
	if s.metrics.TotalExecutions > 0 {
		// This is a simplified calculation - in a real implementation,
		// you would track actual execution times
		s.metrics.AverageExecutionTime = 5 * time.Minute
	}

	if s.metrics.TotalExecutions > 0 {
		now := time.Now()
		s.metrics.LastExecutionTime = &now
	}
}

// publishEvent publishes a scheduler event
func (s *CronScheduler) publishEvent(eventType SchedulerEventType, taskID *int, message string, data map[string]interface{}) {
	if s.eventListener == nil {
		return
	}

	event := SchedulerEvent{
		Type:      eventType,
		TaskID:    taskID,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
	}

	go s.eventListener.OnEvent(event)
}

// GetMetrics returns current scheduler metrics
func (s *CronScheduler) GetMetrics() *SchedulerMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy of metrics
	metrics := *s.metrics
	return &metrics
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}