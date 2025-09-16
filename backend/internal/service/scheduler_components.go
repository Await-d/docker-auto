package service

import (
	"context"
	"encoding/json"
	"time"

	"docker-auto/internal/model"
	"docker-auto/pkg/scheduler"

	"github.com/sirupsen/logrus"
)

// SchedulerEventListener handles scheduler events
type SchedulerEventListener struct {
	schedulerService *SchedulerService
}

// NewSchedulerEventListener creates a new event listener
func NewSchedulerEventListener(service *SchedulerService) *SchedulerEventListener {
	return &SchedulerEventListener{
		schedulerService: service,
	}
}

// OnEvent handles scheduler events
func (l *SchedulerEventListener) OnEvent(event scheduler.SchedulerEvent) {
	logger := logrus.WithFields(logrus.Fields{
		"event_type": event.Type,
		"timestamp":  event.Timestamp,
	})

	if event.TaskID != nil {
		logger = logger.WithField("task_id", *event.TaskID)
	}

	switch event.Type {
	case scheduler.EventSchedulerStarted:
		logger.Info("Scheduler started")
		l.logActivity("scheduler_started", "Scheduler service started", event.Data)

	case scheduler.EventSchedulerStopped:
		logger.Info("Scheduler stopped")
		l.logActivity("scheduler_stopped", "Scheduler service stopped", event.Data)

	case scheduler.EventTaskStarted:
		logger.Info("Task execution started")
		l.logActivity("task_execution_started", "Task execution started", event.Data)

	case scheduler.EventTaskCompleted:
		logger.Info("Task execution completed successfully")
		l.logActivity("task_execution_completed", "Task execution completed", event.Data)

	case scheduler.EventTaskFailed:
		logger.Warn("Task execution failed")
		l.logActivity("task_execution_failed", "Task execution failed", event.Data)

	case scheduler.EventTaskTimeout:
		logger.Warn("Task execution timed out")
		l.logActivity("task_execution_timeout", "Task execution timed out", event.Data)

	case scheduler.EventTaskRetried:
		logger.Info("Task execution retried")
		l.logActivity("task_execution_retried", "Task execution retried", event.Data)

	case scheduler.EventTaskCancelled:
		logger.Info("Task execution cancelled")
		l.logActivity("task_execution_cancelled", "Task execution cancelled", event.Data)

	default:
		logger.Debug("Received unknown scheduler event")
	}
}

// logActivity logs scheduler activity to the activity log
func (l *SchedulerEventListener) logActivity(action, description string, data map[string]interface{}) {
	if l.schedulerService.activityLogRepo == nil {
		return
	}

	// Use system user ID (0) for scheduler activities
	userID := int64(0)

	metadataJSON := "{}"
	if data != nil {
		if jsonData, err := json.Marshal(data); err == nil {
			metadataJSON = string(jsonData)
		}
	}

	log := &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: "scheduler",
		Description:  description,
		Metadata:     metadataJSON,
	}

	if err := l.schedulerService.activityLogRepo.Create(context.Background(), log); err != nil {
		logrus.WithError(err).Warn("Failed to log scheduler activity")
	}
}

// LoggingHook implements TaskHook interface for logging
type LoggingHook struct{}

// NewLoggingHook creates a new logging hook
func NewLoggingHook() scheduler.TaskHook {
	return &LoggingHook{}
}

// BeforeExecution is called before task execution starts
func (h *LoggingHook) BeforeExecution(ctx context.Context, execution *scheduler.TaskExecution) error {
	logrus.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"task_name":    execution.TaskName,
		"task_type":    execution.TaskType,
	}).Debug("Task execution starting")
	return nil
}

// AfterExecution is called after task execution completes
func (h *LoggingHook) AfterExecution(ctx context.Context, execution *scheduler.TaskExecution, result *scheduler.TaskResult) error {
	logger := logrus.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"task_name":    execution.TaskName,
		"task_type":    execution.TaskType,
		"success":      result.Success,
		"duration":     result.Duration,
		"retry_count":  result.RetryCount,
	})

	if result.Success {
		logger.Info("Task execution completed successfully")
	} else {
		logger.WithError(result.Error).Error("Task execution failed")
	}

	return nil
}

// OnError is called when task execution fails
func (h *LoggingHook) OnError(ctx context.Context, execution *scheduler.TaskExecution, err error) error {
	logrus.WithError(err).WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"task_name":    execution.TaskName,
		"task_type":    execution.TaskType,
	}).Error("Task execution error")
	return nil
}

// OnRetry is called when task execution is being retried
func (h *LoggingHook) OnRetry(ctx context.Context, execution *scheduler.TaskExecution, retryCount int) error {
	logrus.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"task_name":    execution.TaskName,
		"task_type":    execution.TaskType,
		"retry_count":  retryCount,
	}).Info("Retrying task execution")
	return nil
}

// MetricsHook implements TaskHook interface for metrics collection
type MetricsHook struct {
	taskMetrics map[string]*TaskMetrics
}

// TaskMetrics represents metrics for a specific task type
type TaskMetrics struct {
	TotalExecutions      int64
	SuccessfulExecutions int64
	FailedExecutions     int64
	TotalDuration        time.Duration
	LastExecution        time.Time
}

// NewMetricsHook creates a new metrics hook
func NewMetricsHook() scheduler.TaskHook {
	return &MetricsHook{
		taskMetrics: make(map[string]*TaskMetrics),
	}
}

// BeforeExecution is called before task execution starts
func (h *MetricsHook) BeforeExecution(ctx context.Context, execution *scheduler.TaskExecution) error {
	return nil
}

// AfterExecution is called after task execution completes
func (h *MetricsHook) AfterExecution(ctx context.Context, execution *scheduler.TaskExecution, result *scheduler.TaskResult) error {
	taskType := string(execution.TaskType)

	if h.taskMetrics[taskType] == nil {
		h.taskMetrics[taskType] = &TaskMetrics{}
	}

	metrics := h.taskMetrics[taskType]
	metrics.TotalExecutions++
	metrics.TotalDuration += result.Duration
	metrics.LastExecution = time.Now()

	if result.Success {
		metrics.SuccessfulExecutions++
	} else {
		metrics.FailedExecutions++
	}

	// Log metrics periodically (simplified)
	if metrics.TotalExecutions%10 == 0 {
		logrus.WithFields(logrus.Fields{
			"task_type":             taskType,
			"total_executions":      metrics.TotalExecutions,
			"successful_executions": metrics.SuccessfulExecutions,
			"failed_executions":     metrics.FailedExecutions,
			"average_duration":      metrics.TotalDuration / time.Duration(metrics.TotalExecutions),
		}).Info("Task metrics update")
	}

	return nil
}

// OnError is called when task execution fails
func (h *MetricsHook) OnError(ctx context.Context, execution *scheduler.TaskExecution, err error) error {
	return nil
}

// OnRetry is called when task execution is being retried
func (h *MetricsHook) OnRetry(ctx context.Context, execution *scheduler.TaskExecution, retryCount int) error {
	return nil
}

// GetMetrics returns current metrics for all task types
func (h *MetricsHook) GetMetrics() map[string]*TaskMetrics {
	result := make(map[string]*TaskMetrics)
	for taskType, metrics := range h.taskMetrics {
		// Return a copy to avoid race conditions
		result[taskType] = &TaskMetrics{
			TotalExecutions:      metrics.TotalExecutions,
			SuccessfulExecutions: metrics.SuccessfulExecutions,
			FailedExecutions:     metrics.FailedExecutions,
			TotalDuration:        metrics.TotalDuration,
			LastExecution:        metrics.LastExecution,
		}
	}
	return result
}