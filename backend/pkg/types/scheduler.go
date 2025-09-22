package types

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TaskType represents different types of tasks that can be scheduled
type TaskType string

const (
	// TaskTypeImageCleanup represents image cleanup tasks
	TaskTypeImageCleanup TaskType = "image_cleanup"
	// TaskTypeSystemMaintenance represents system maintenance tasks
	TaskTypeSystemMaintenance TaskType = "system_maintenance"
	// TaskTypeContainerUpdate represents container update tasks
	TaskTypeContainerUpdate TaskType = "container_update"
	// TaskTypeHealthCheck represents health check tasks
	TaskTypeHealthCheck TaskType = "health_check"
	// TaskTypeBackup represents backup tasks
	TaskTypeBackup TaskType = "backup"
	// TaskTypeMonitoring represents monitoring tasks
	TaskTypeMonitoring TaskType = "monitoring"
	// TaskTypeCleanup represents general cleanup tasks
	TaskTypeCleanup TaskType = "cleanup"
	// TaskTypeNotification represents notification tasks
	TaskTypeNotification TaskType = "notification"
	// TaskTypeSecurityScan represents security scan tasks
	TaskTypeSecurityScan TaskType = "security_scan"
	// TaskTypeLogRotation represents log rotation tasks
	TaskTypeLogRotation TaskType = "log_rotation"
)

// ExecutionStatus represents the status of task execution
type ExecutionStatus string

const (
	// ExecutionStatusPending indicates task is pending execution
	ExecutionStatusPending ExecutionStatus = "pending"
	// ExecutionStatusRunning indicates task is currently running
	ExecutionStatusRunning ExecutionStatus = "running"
	// ExecutionStatusCompleted indicates task has completed successfully
	ExecutionStatusCompleted ExecutionStatus = "completed"
	// ExecutionStatusFailed indicates task has failed
	ExecutionStatusFailed ExecutionStatus = "failed"
	// ExecutionStatusCancelled indicates task was cancelled
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	// ExecutionStatusTimeout indicates task timed out
	ExecutionStatusTimeout ExecutionStatus = "timeout"
	// ExecutionStatusRetrying indicates task is being retried
	ExecutionStatusRetrying ExecutionStatus = "retrying"
	// ExecutionStatusPaused indicates task is paused
	ExecutionStatusPaused ExecutionStatus = "paused"
)

// TaskPriority defines task execution priority
type TaskPriority int

const (
	// TaskPriorityLow represents low priority tasks
	TaskPriorityLow TaskPriority = 1
	// TaskPriorityNormal represents normal priority tasks
	TaskPriorityNormal TaskPriority = 5
	// TaskPriorityHigh represents high priority tasks
	TaskPriorityHigh TaskPriority = 10
	// TaskPriorityCritical represents critical priority tasks
	TaskPriorityCritical TaskPriority = 15
)

// String returns the string representation of TaskPriority
func (p TaskPriority) String() string {
	switch p {
	case TaskPriorityLow:
		return "low"
	case TaskPriorityNormal:
		return "normal"
	case TaskPriorityHigh:
		return "high"
	case TaskPriorityCritical:
		return "critical"
	default:
		return "normal"
	}
}

// TaskParameters represents parameters for task execution
type TaskParameters struct {
	TaskType         TaskType               `json:"task_type"`
	TargetContainers []int64                `json:"target_containers,omitempty"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
	Timeout          time.Duration          `json:"timeout,omitempty"`
	MaxRetries       int                    `json:"max_retries,omitempty"`
	RetryDelay       time.Duration          `json:"retry_delay,omitempty"`
}

// TaskResult represents the result of task execution
type TaskResult struct {
	Success       bool                   `json:"success"`
	Message       string                 `json:"message,omitempty"`
	Error         error                  `json:"error,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Duration      time.Duration          `json:"duration"`
	RetryCount    int                    `json:"retry_count"`
	AffectedItems []string               `json:"affected_items,omitempty"`
}

// TaskExecution represents an active or completed task execution
type TaskExecution struct {
	ID          string                 `json:"id"`
	TaskID      int                    `json:"task_id"`
	TaskName    string                 `json:"task_name"`
	TaskType    TaskType               `json:"task_type"`
	Status      ExecutionStatus        `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Progress    float64                `json:"progress"` // 0-100
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Result      *TaskResult            `json:"result,omitempty"`
	Parameters  TaskParameters         `json:"parameters"`
	CancelFunc  context.CancelFunc     `json:"-"`
}

// TaskStatus represents the current status of a scheduled task
type TaskStatus struct {
	TaskID           int             `json:"task_id"`
	Name             string          `json:"name"`
	Type             TaskType        `json:"type"`
	IsActive         bool            `json:"is_active"`
	IsPaused         bool            `json:"is_paused"`
	IsRunning        bool            `json:"is_running"`
	LastRun          *time.Time      `json:"last_run,omitempty"`
	NextRun          *time.Time      `json:"next_run,omitempty"`
	RunCount         int             `json:"run_count"`
	FailureCount     int             `json:"failure_count"`
	SuccessRate      float64         `json:"success_rate"`
	LastResult       *TaskResult     `json:"last_result,omitempty"`
	LastError        string          `json:"last_error,omitempty"`
	AverageRunTime   time.Duration   `json:"average_run_time"`
	CurrentExecution *TaskExecution  `json:"current_execution,omitempty"`
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	// MaxConcurrentTasks limits the number of tasks that can run simultaneously
	MaxConcurrentTasks int `json:"max_concurrent_tasks"`

	// TaskTimeout sets the default timeout for task execution
	TaskTimeout time.Duration `json:"task_timeout"`

	// RetryDelay sets the default delay between retries
	RetryDelay time.Duration `json:"retry_delay"`

	// MaxRetries sets the default maximum number of retries
	MaxRetries int `json:"max_retries"`

	// CleanupInterval sets how often to clean up completed task executions
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// HistoryRetention sets how long to keep task execution history
	HistoryRetention time.Duration `json:"history_retention"`

	// LogLevel sets the logging level for scheduler operations
	LogLevel string `json:"log_level"`

	// EnableMetrics enables metrics collection
	EnableMetrics bool `json:"enable_metrics"`

	// TimeZone sets the timezone for cron scheduling
	TimeZone string `json:"time_zone"`
}

// SchedulerMetrics represents scheduler performance metrics
type SchedulerMetrics struct {
	TotalTasks           int           `json:"total_tasks"`
	ActiveTasks          int           `json:"active_tasks"`
	RunningTasks         int           `json:"running_tasks"`
	PausedTasks          int           `json:"paused_tasks"`
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecutionTime    *time.Time    `json:"last_execution_time,omitempty"`
	QueueDepth           int           `json:"queue_depth"`
	WorkerUtilization    float64       `json:"worker_utilization"`
	UptimeSeconds        int64         `json:"uptime_seconds"`
}

// TaskDependency represents task dependencies
type TaskDependency struct {
	TaskID    int        `json:"task_id"`
	DependsOn []int      `json:"depends_on"`
	WaitFor   []TaskType `json:"wait_for_types,omitempty"`
	Condition string     `json:"condition,omitempty"` // success, failure, completion
}

// ScheduleOptions provides options for task scheduling
type ScheduleOptions struct {
	Priority         TaskPriority     `json:"priority"`
	Dependencies     []TaskDependency `json:"dependencies,omitempty"`
	Tags             []string         `json:"tags,omitempty"`
	NotifyOnFailure  bool             `json:"notify_on_failure"`
	NotifyOnSuccess  bool             `json:"notify_on_success"`
	MaxDuration      time.Duration    `json:"max_duration,omitempty"`
	AllowOverlap     bool             `json:"allow_overlap"`
	Timezone         string           `json:"timezone,omitempty"`
}

// SchedulerEventType represents different types of scheduler events
type SchedulerEventType string

const (
	// EventSchedulerStarted indicates scheduler has started
	EventSchedulerStarted SchedulerEventType = "scheduler_started"
	// EventSchedulerStopped indicates scheduler has stopped
	EventSchedulerStopped SchedulerEventType = "scheduler_stopped"
	// EventTaskAdded indicates a task was added
	EventTaskAdded SchedulerEventType = "task_added"
	// EventTaskRemoved indicates a task was removed
	EventTaskRemoved SchedulerEventType = "task_removed"
	// EventTaskUpdated indicates a task was updated
	EventTaskUpdated SchedulerEventType = "task_updated"
	// EventTaskPaused indicates a task was paused
	EventTaskPaused SchedulerEventType = "task_paused"
	// EventTaskResumed indicates a task was resumed
	EventTaskResumed SchedulerEventType = "task_resumed"
	// EventTaskStarted indicates a task started execution
	EventTaskStarted SchedulerEventType = "task_started"
	// EventTaskCompleted indicates a task completed successfully
	EventTaskCompleted SchedulerEventType = "task_completed"
	// EventTaskFailed indicates a task failed
	EventTaskFailed SchedulerEventType = "task_failed"
	// EventTaskTimeout indicates a task timed out
	EventTaskTimeout SchedulerEventType = "task_timeout"
	// EventTaskRetried indicates a task was retried
	EventTaskRetried SchedulerEventType = "task_retried"
	// EventTaskCancelled indicates a task was cancelled
	EventTaskCancelled SchedulerEventType = "task_cancelled"
)

// SchedulerEvent represents an event that occurred in the scheduler
type SchedulerEvent struct {
	Type      SchedulerEventType         `json:"type"`
	TaskID    *int                       `json:"task_id,omitempty"`
	Message   string                     `json:"message"`
	Timestamp time.Time                  `json:"timestamp"`
	Data      map[string]interface{}     `json:"data,omitempty"`
}

// ScheduledTask represents a task that can be scheduled for execution
type ScheduledTask struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Type              TaskType               `json:"type"`
	CronExpression    string                 `json:"cron_expression"`
	Parameters        TaskParameters         `json:"parameters"`
	Priority          TaskPriority           `json:"priority"`
	IsActive          bool                   `json:"is_active"`
	IsPaused          bool                   `json:"is_paused"`
	MaxRetries        int                    `json:"max_retries"`
	RetryDelay        time.Duration          `json:"retry_delay"`
	Timeout           time.Duration          `json:"timeout"`
	Tags              []string               `json:"tags,omitempty"`
	NotifyOnFailure   bool                   `json:"notify_on_failure"`
	NotifyOnSuccess   bool                   `json:"notify_on_success"`
	Dependencies      []TaskDependency       `json:"dependencies,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	LastRun           *time.Time             `json:"last_run,omitempty"`
	NextRun           *time.Time             `json:"next_run,omitempty"`
	RunCount          int                    `json:"run_count"`
	FailureCount      int                    `json:"failure_count"`
	SuccessCount      int                    `json:"success_count"`
	LastResult        *TaskResult            `json:"last_result,omitempty"`
	CreatedBy         int                    `json:"created_by,omitempty"`         // User ID
	UpdatedBy         int                    `json:"updated_by,omitempty"`         // User ID
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// TaskConfig represents configuration for a specific task type
type TaskConfig struct {
	Type              TaskType               `json:"type"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	DefaultTimeout    time.Duration          `json:"default_timeout"`
	DefaultRetries    int                    `json:"default_retries"`
	DefaultRetryDelay time.Duration          `json:"default_retry_delay"`
	AllowConcurrent   bool                   `json:"allow_concurrent"`
	RequiredParams    []string               `json:"required_params,omitempty"`
	OptionalParams    []string               `json:"optional_params,omitempty"`
	Validations       map[string]interface{} `json:"validations,omitempty"`
	DefaultPriority   TaskPriority           `json:"default_priority"`
	Category          string                 `json:"category,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Documentation     string                 `json:"documentation,omitempty"`
	Version           string                 `json:"version,omitempty"`
	IsActive          bool                   `json:"is_active"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// TaskFilter represents filters for querying tasks
type TaskFilter struct {
	Types       []TaskType        `json:"types,omitempty"`
	Status      ExecutionStatus   `json:"status,omitempty"`
	Priority    TaskPriority      `json:"priority,omitempty"`
	IsActive    *bool             `json:"is_active,omitempty"`
	IsPaused    *bool             `json:"is_paused,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedBy   *int              `json:"created_by,omitempty"`
	FromTime    *time.Time        `json:"from_time,omitempty"`
	ToTime      *time.Time        `json:"to_time,omitempty"`
	NamePattern string            `json:"name_pattern,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
	SortBy      string            `json:"sort_by,omitempty"`      // name, created_at, priority, last_run, etc.
	SortOrder   string            `json:"sort_order,omitempty"`   // asc, desc
}

// TaskRetryPolicy represents retry policy for failed tasks
type TaskRetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// TaskNotification represents notification configuration for tasks
type TaskNotification struct {
	ID             int                    `json:"id"`
	TaskID         int                    `json:"task_id"`
	EventTypes     []SchedulerEventType   `json:"event_types"`
	Recipients     []string               `json:"recipients"`     // email addresses
	WebhookURLs    []string               `json:"webhook_urls,omitempty"`
	EmailTemplate  string                 `json:"email_template,omitempty"`
	IsActive       bool                   `json:"is_active"`
	Conditions     map[string]interface{} `json:"conditions,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// TaskHistory represents historical execution data for a task
type TaskHistory struct {
	ID            int                    `json:"id"`
	TaskID        int                    `json:"task_id"`
	ExecutionID   string                 `json:"execution_id"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Status        ExecutionStatus        `json:"status"`
	Result        *TaskResult            `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	Parameters    TaskParameters         `json:"parameters"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// GetPriorityValue returns the numeric value of the priority
func (p TaskPriority) GetPriorityValue() int {
	return int(p)
}

// IsHigherPriority checks if this priority is higher than another
func (p TaskPriority) IsHigherPriority(other TaskPriority) bool {
	return p > other
}

// IsCompleted returns true if the execution is in a completed state
func (s ExecutionStatus) IsCompleted() bool {
	return s == ExecutionStatusCompleted ||
		s == ExecutionStatusFailed ||
		s == ExecutionStatusCancelled ||
		s == ExecutionStatusTimeout
}

// IsFinal returns true if the execution status is final (cannot change)
func (s ExecutionStatus) IsFinal() bool {
	return s.IsCompleted()
}

// IsSuccess returns true if the execution completed successfully
func (s ExecutionStatus) IsSuccess() bool {
	return s == ExecutionStatusCompleted
}

// IsRunning returns true if the execution is currently running
func (s ExecutionStatus) IsRunning() bool {
	return s == ExecutionStatusRunning || s == ExecutionStatusRetrying
}

// CanRetry returns true if the execution can be retried
func (s ExecutionStatus) CanRetry() bool {
	return s == ExecutionStatusFailed || s == ExecutionStatusTimeout
}

// CalculateDelay calculates the retry delay based on attempt number
func (r *TaskRetryPolicy) CalculateDelay(attempt int) time.Duration {
	if r == nil {
		return time.Second
	}

	delay := r.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * r.BackoffFactor)
		if delay > r.MaxDelay {
			return r.MaxDelay
		}
	}
	return delay
}

// IsRetryable checks if an error is retryable based on retry policy
func (r *TaskRetryPolicy) IsRetryable(errorMsg string) bool {
	if r == nil || len(r.RetryableErrors) == 0 {
		return true // Default to retryable if no specific errors defined
	}

	for _, retryableError := range r.RetryableErrors {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(retryableError)) {
			return true
		}
	}
	return false
}

// GetSuccessRate calculates the success rate of the task
func (t *ScheduledTask) GetSuccessRate() float64 {
	if t.RunCount == 0 {
		return 0.0
	}
	return float64(t.SuccessCount) / float64(t.RunCount)
}

// GetFailureRate calculates the failure rate of the task
func (t *ScheduledTask) GetFailureRate() float64 {
	if t.RunCount == 0 {
		return 0.0
	}
	return float64(t.FailureCount) / float64(t.RunCount)
}

// IsOverdue checks if the task is overdue for execution
func (t *ScheduledTask) IsOverdue() bool {
	if t.NextRun == nil || !t.IsActive || t.IsPaused {
		return false
	}
	return time.Now().After(*t.NextRun)
}

// CanExecute checks if the task can be executed now
func (t *ScheduledTask) CanExecute() bool {
	return t.IsActive && !t.IsPaused && (t.NextRun == nil || time.Now().After(*t.NextRun))
}

// HasDependencies checks if the task has any dependencies
func (t *ScheduledTask) HasDependencies() bool {
	return len(t.Dependencies) > 0
}

// EstimateExecutionTime estimates the execution time based on historical data
func (t *ScheduledTask) EstimateExecutionTime() time.Duration {
	if t.LastResult != nil {
		return t.LastResult.Duration
	}
	return t.Timeout // fallback to timeout
}

// Validate validates the scheduled task configuration
func (t *ScheduledTask) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name is required")
	}

	if t.Type == "" {
		return fmt.Errorf("task type is required")
	}

	if t.CronExpression == "" {
		return fmt.Errorf("cron expression is required")
	}

	if t.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if t.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if t.RetryDelay <= 0 {
		return fmt.Errorf("retry delay must be positive")
	}

	return nil
}

// String returns the string representation of TaskType
func (t TaskType) String() string {
	return string(t)
}

// String returns the string representation of ExecutionStatus
func (s ExecutionStatus) String() string {
	return string(s)
}