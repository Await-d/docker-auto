package scheduler

import (
	"context"
	"time"

	"docker-auto/internal/model"
)

// Scheduler defines the interface for task scheduling systems
type Scheduler interface {
	// Start starts the scheduler
	Start(ctx context.Context) error

	// Stop stops the scheduler gracefully
	Stop(ctx context.Context) error

	// AddTask adds a new task to the scheduler
	AddTask(task *model.ScheduledTask) error

	// RemoveTask removes a task from the scheduler
	RemoveTask(taskID int) error

	// UpdateTask updates an existing task
	UpdateTask(task *model.ScheduledTask) error

	// PauseTask pauses a specific task
	PauseTask(taskID int) error

	// ResumeTask resumes a paused task
	ResumeTask(taskID int) error

	// TriggerTask manually triggers a task execution
	TriggerTask(taskID int) error

	// GetRunningTasks returns currently running tasks
	GetRunningTasks() []*TaskExecution

	// GetTaskStatus returns the status of a specific task
	GetTaskStatus(taskID int) (*TaskStatus, error)

	// IsRunning returns true if the scheduler is running
	IsRunning() bool
}

// Task defines the interface for executable tasks
type Task interface {
	// Execute runs the task
	Execute(ctx context.Context, params TaskParameters) error

	// GetName returns the task name
	GetName() string

	// GetType returns the task type
	GetType() model.TaskType

	// Validate validates task parameters
	Validate(params TaskParameters) error

	// GetDefaultTimeout returns the default timeout for this task
	GetDefaultTimeout() time.Duration

	// CanRunConcurrently returns true if this task can run concurrently with other instances
	CanRunConcurrently() bool
}

// TaskRegistry defines the interface for task registration
type TaskRegistry interface {
	// RegisterTask registers a new task type
	RegisterTask(taskType model.TaskType, factory TaskFactory) error

	// GetTask creates a task instance by type
	GetTask(taskType model.TaskType) (Task, error)

	// GetRegisteredTypes returns all registered task types
	GetRegisteredTypes() []model.TaskType

	// UnregisterTask removes a task type from registry
	UnregisterTask(taskType model.TaskType) error
}

// TaskFactory creates task instances
type TaskFactory func() Task

// TaskExecutor defines the interface for task execution
type TaskExecutor interface {
	// ExecuteTask executes a task with the given parameters
	ExecuteTask(ctx context.Context, task Task, params TaskParameters) (*TaskResult, error)

	// CancelTask cancels a running task
	CancelTask(executionID string) error

	// GetExecution returns information about a task execution
	GetExecution(executionID string) (*TaskExecution, error)

	// GetActiveExecutions returns all currently active task executions
	GetActiveExecutions() []*TaskExecution

	// SetConcurrencyLimit sets the maximum number of concurrent task executions
	SetConcurrencyLimit(limit int)
}

// TaskParameters represents parameters for task execution
type TaskParameters struct {
	TaskType        model.TaskType         `json:"task_type"`
	TargetContainers []int64               `json:"target_containers,omitempty"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
	Timeout          time.Duration         `json:"timeout,omitempty"`
	MaxRetries       int                   `json:"max_retries,omitempty"`
	RetryDelay       time.Duration         `json:"retry_delay,omitempty"`
}

// TaskResult represents the result of task execution
type TaskResult struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	Error        error                  `json:"error,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Duration     time.Duration          `json:"duration"`
	RetryCount   int                    `json:"retry_count"`
	AffectedItems []string              `json:"affected_items,omitempty"`
}

// TaskExecution represents an active or completed task execution
type TaskExecution struct {
	ID           string                 `json:"id"`
	TaskID       int                    `json:"task_id"`
	TaskName     string                 `json:"task_name"`
	TaskType     model.TaskType         `json:"task_type"`
	Status       model.ExecutionStatus  `json:"status"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Progress     float64                `json:"progress"` // 0-100
	Message      string                 `json:"message,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Result       *TaskResult            `json:"result,omitempty"`
	Parameters   TaskParameters         `json:"parameters"`
	CancelFunc   context.CancelFunc     `json:"-"`
}

// TaskStatus represents the current status of a scheduled task
type TaskStatus struct {
	TaskID           int                   `json:"task_id"`
	Name             string                `json:"name"`
	Type             model.TaskType        `json:"type"`
	IsActive         bool                  `json:"is_active"`
	IsPaused         bool                  `json:"is_paused"`
	IsRunning        bool                  `json:"is_running"`
	LastRun          *time.Time            `json:"last_run,omitempty"`
	NextRun          *time.Time            `json:"next_run,omitempty"`
	RunCount         int                   `json:"run_count"`
	FailureCount     int                   `json:"failure_count"`
	SuccessRate      float64               `json:"success_rate"`
	LastResult       *TaskResult           `json:"last_result,omitempty"`
	LastError        string                `json:"last_error,omitempty"`
	AverageRunTime   time.Duration         `json:"average_run_time"`
	CurrentExecution *TaskExecution        `json:"current_execution,omitempty"`
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
	TotalTasks          int           `json:"total_tasks"`
	ActiveTasks         int           `json:"active_tasks"`
	RunningTasks        int           `json:"running_tasks"`
	PausedTasks         int           `json:"paused_tasks"`
	TotalExecutions     int64         `json:"total_executions"`
	SuccessfulExecutions int64        `json:"successful_executions"`
	FailedExecutions    int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecutionTime   *time.Time    `json:"last_execution_time,omitempty"`
	QueueDepth          int           `json:"queue_depth"`
	WorkerUtilization   float64       `json:"worker_utilization"`
	UptimeSeconds       int64         `json:"uptime_seconds"`
}

// TaskPriority defines task execution priority
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityNormal TaskPriority = 5
	PriorityHigh   TaskPriority = 10
	PriorityCritical TaskPriority = 15
)

// TaskDependency represents task dependencies
type TaskDependency struct {
	TaskID      int              `json:"task_id"`
	DependsOn   []int            `json:"depends_on"`
	WaitFor     []model.TaskType `json:"wait_for_types,omitempty"`
	Condition   string           `json:"condition,omitempty"` // success, failure, completion
}

// ScheduleOptions provides options for task scheduling
type ScheduleOptions struct {
	Priority     TaskPriority     `json:"priority"`
	Dependencies []TaskDependency `json:"dependencies,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	NotifyOnFailure bool          `json:"notify_on_failure"`
	NotifyOnSuccess bool          `json:"notify_on_success"`
	MaxDuration  time.Duration    `json:"max_duration,omitempty"`
	AllowOverlap bool             `json:"allow_overlap"`
	Timezone     string           `json:"timezone,omitempty"`
}

// TaskHook defines hooks that can be executed at various points during task execution
type TaskHook interface {
	// BeforeExecution is called before task execution starts
	BeforeExecution(ctx context.Context, execution *TaskExecution) error

	// AfterExecution is called after task execution completes
	AfterExecution(ctx context.Context, execution *TaskExecution, result *TaskResult) error

	// OnError is called when task execution fails
	OnError(ctx context.Context, execution *TaskExecution, err error) error

	// OnRetry is called when task execution is being retried
	OnRetry(ctx context.Context, execution *TaskExecution, retryCount int) error
}

// SchedulerEventType represents different types of scheduler events
type SchedulerEventType string

const (
	EventSchedulerStarted     SchedulerEventType = "scheduler_started"
	EventSchedulerStopped     SchedulerEventType = "scheduler_stopped"
	EventTaskAdded           SchedulerEventType = "task_added"
	EventTaskRemoved         SchedulerEventType = "task_removed"
	EventTaskUpdated         SchedulerEventType = "task_updated"
	EventTaskPaused          SchedulerEventType = "task_paused"
	EventTaskResumed         SchedulerEventType = "task_resumed"
	EventTaskStarted         SchedulerEventType = "task_started"
	EventTaskCompleted       SchedulerEventType = "task_completed"
	EventTaskFailed          SchedulerEventType = "task_failed"
	EventTaskTimeout         SchedulerEventType = "task_timeout"
	EventTaskRetried         SchedulerEventType = "task_retried"
	EventTaskCancelled       SchedulerEventType = "task_cancelled"
)

// SchedulerEvent represents an event that occurred in the scheduler
type SchedulerEvent struct {
	Type      SchedulerEventType `json:"type"`
	TaskID    *int               `json:"task_id,omitempty"`
	Message   string             `json:"message"`
	Timestamp time.Time          `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventListener defines the interface for listening to scheduler events
type EventListener interface {
	OnEvent(event SchedulerEvent)
}

// SchedulerOptions provides configuration options for creating a scheduler
type SchedulerOptions struct {
	Config        *SchedulerConfig `json:"config"`
	TaskRegistry  TaskRegistry     `json:"-"`
	EventListener EventListener    `json:"-"`
	TaskHooks     []TaskHook       `json:"-"`
}