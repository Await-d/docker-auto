package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultTaskExecutor implements the TaskExecutor interface
type DefaultTaskExecutor struct {
	executions       map[string]*TaskExecution
	concurrencyLimit int
	activeTasks      chan struct{}
	mu               sync.RWMutex
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(config *config.Config) TaskExecutor {
	concurrencyLimit := 10
	if config != nil && config.Scheduler.MaxConcurrentTasks > 0 {
		concurrencyLimit = config.Scheduler.MaxConcurrentTasks
	}

	return &DefaultTaskExecutor{
		executions:       make(map[string]*TaskExecution),
		concurrencyLimit: concurrencyLimit,
		activeTasks:      make(chan struct{}, concurrencyLimit),
	}
}

// ExecuteTask executes a task with the given parameters
func (e *DefaultTaskExecutor) ExecuteTask(ctx context.Context, task Task, params TaskParameters) (*TaskResult, error) {
	// Acquire execution slot
	select {
	case e.activeTasks <- struct{}{}:
		defer func() { <-e.activeTasks }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	executionID := uuid.New().String()
	startTime := time.Now()

	// Create execution context with timeout
	execCtx := ctx
	if params.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, params.Timeout)
		defer cancel()
	} else if task.GetDefaultTimeout() > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, task.GetDefaultTimeout())
		defer cancel()
	}

	// Create execution record
	execution := &TaskExecution{
		ID:        executionID,
		TaskName:  task.GetName(),
		TaskType:  task.GetType(),
		StartedAt: startTime,
		Parameters: params,
	}

	// Store execution
	e.mu.Lock()
	e.executions[executionID] = execution
	e.mu.Unlock()

	logger := logrus.WithFields(logrus.Fields{
		"execution_id": executionID,
		"task_name":    task.GetName(),
		"task_type":    task.GetType(),
	})

	logger.Info("Starting task execution")

	// Execute task with retries
	result := e.executeWithRetries(execCtx, task, params, execution, logger)

	// Update execution
	e.mu.Lock()
	execution.Duration = time.Since(startTime)
	execution.Result = result
	if result.Success {
		execution.Status = "success"
	} else {
		execution.Status = "failed"
		execution.Error = ""
		if result.Error != nil {
			execution.Error = result.Error.Error()
		}
	}
	if execution.CompletedAt == nil {
		now := time.Now()
		execution.CompletedAt = &now
	}
	e.mu.Unlock()

	// Clean up execution record after some time
	go func() {
		time.Sleep(10 * time.Minute)
		e.mu.Lock()
		delete(e.executions, executionID)
		e.mu.Unlock()
	}()

	logger.WithFields(logrus.Fields{
		"success":     result.Success,
		"duration":    result.Duration,
		"retry_count": result.RetryCount,
	}).Info("Task execution completed")

	return result, nil
}

// executeWithRetries executes a task with retry logic
func (e *DefaultTaskExecutor) executeWithRetries(ctx context.Context, task Task, params TaskParameters, execution *TaskExecution, logger *logrus.Entry) *TaskResult {
	maxRetries := params.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3 // Default retry count
	}

	retryDelay := params.RetryDelay
	if retryDelay == 0 {
		retryDelay = 5 * time.Second // Default retry delay
	}

	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Update execution progress
		e.mu.Lock()
		execution.Progress = float64(attempt) / float64(maxRetries+1) * 100
		e.mu.Unlock()

		attemptStart := time.Now()
		logger.WithField("attempt", attempt+1).Debug("Executing task attempt")

		// Execute task
		err := task.Execute(ctx, params)
		attemptDuration := time.Since(attemptStart)

		if err == nil {
			// Success
			return &TaskResult{
				Success:    true,
				Duration:   time.Since(startTime),
				RetryCount: attempt,
			}
		}

		lastErr = err
		logger.WithError(err).WithFields(logrus.Fields{
			"attempt":  attempt + 1,
			"duration": attemptDuration,
		}).Warn("Task execution attempt failed")

		// Check if we should retry
		if attempt < maxRetries {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return &TaskResult{
					Success:    false,
					Error:      ctx.Err(),
					Duration:   time.Since(startTime),
					RetryCount: attempt,
				}
			default:
			}

			// Wait before retrying
			timer := time.NewTimer(retryDelay)
			select {
			case <-timer.C:
				// Continue to next attempt
			case <-ctx.Done():
				timer.Stop()
				return &TaskResult{
					Success:    false,
					Error:      ctx.Err(),
					Duration:   time.Since(startTime),
					RetryCount: attempt,
				}
			}
		}
	}

	// All retries exhausted
	return &TaskResult{
		Success:    false,
		Error:      lastErr,
		Duration:   time.Since(startTime),
		RetryCount: maxRetries,
	}
}

// CancelTask cancels a running task
func (e *DefaultTaskExecutor) CancelTask(executionID string) error {
	e.mu.RLock()
	execution, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	if execution.CancelFunc != nil {
		execution.CancelFunc()
		return nil
	}

	return fmt.Errorf("execution %s cannot be cancelled", executionID)
}

// GetExecution returns information about a task execution
func (e *DefaultTaskExecutor) GetExecution(executionID string) (*TaskExecution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	execution, exists := e.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}

	// Return a copy to avoid race conditions
	execCopy := *execution
	return &execCopy, nil
}

// GetActiveExecutions returns all currently active task executions
func (e *DefaultTaskExecutor) GetActiveExecutions() []*TaskExecution {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var active []*TaskExecution
	for _, execution := range e.executions {
		if execution.CompletedAt == nil {
			// Return a copy to avoid race conditions
			execCopy := *execution
			active = append(active, &execCopy)
		}
	}

	return active
}

// SetConcurrencyLimit sets the maximum number of concurrent task executions
func (e *DefaultTaskExecutor) SetConcurrencyLimit(limit int) {
	if limit <= 0 {
		limit = 1
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.concurrencyLimit = limit

	// Create new channel with updated capacity
	close(e.activeTasks)
	e.activeTasks = make(chan struct{}, limit)
}