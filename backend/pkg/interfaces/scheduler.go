package interfaces

import (
	"context"
	"time"

	"docker-auto/pkg/types"
)

// Scheduler defines the interface for task scheduling systems
type Scheduler interface {
	// Scheduler lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	IsRunning() bool
	GetStatus() (*SchedulerStatus, error)

	// Task management
	AddTask(ctx context.Context, task *types.ScheduledTask) error
	RemoveTask(ctx context.Context, taskID int) error
	UpdateTask(ctx context.Context, task *types.ScheduledTask) error
	GetTask(ctx context.Context, taskID int) (*types.ScheduledTask, error)
	ListTasks(ctx context.Context, filter *TaskFilter) ([]*types.ScheduledTask, error)

	// Task execution control
	PauseTask(ctx context.Context, taskID int) error
	ResumeTask(ctx context.Context, taskID int) error
	TriggerTask(ctx context.Context, taskID int) error
	CancelTask(ctx context.Context, executionID string) error
	KillTask(ctx context.Context, executionID string) error

	// Task status and monitoring
	GetTaskStatus(ctx context.Context, taskID int) (*types.TaskStatus, error)
	GetRunningTasks() []*types.TaskExecution
	GetTaskHistory(ctx context.Context, taskID int, filter *TaskHistoryFilter) ([]*types.TaskHistory, error)
	GetExecutionDetails(ctx context.Context, executionID string) (*types.TaskExecution, error)

	// Bulk operations
	BulkPauseTasks(ctx context.Context, taskIDs []int) (*BulkOperationResult, error)
	BulkResumeTasks(ctx context.Context, taskIDs []int) (*BulkOperationResult, error)
	BulkRemoveTasks(ctx context.Context, taskIDs []int) (*BulkOperationResult, error)
	BulkTriggerTasks(ctx context.Context, taskIDs []int) (*BulkOperationResult, error)

	// Configuration and settings
	UpdateConfig(ctx context.Context, config *types.SchedulerConfig) error
	GetConfig(ctx context.Context) (*types.SchedulerConfig, error)
	GetMetrics(ctx context.Context) (*types.SchedulerMetrics, error)

	// Event handling
	RegisterEventListener(listener SchedulerEventListener) error
	UnregisterEventListener(listener SchedulerEventListener) error
	PublishEvent(event *types.SchedulerEvent) error
}

// Task defines the interface for executable tasks
type Task interface {
	// Task execution
	Execute(ctx context.Context, params types.TaskParameters) (*types.TaskResult, error)
	Validate(params types.TaskParameters) error
	CanExecute(ctx context.Context, params types.TaskParameters) (bool, error)

	// Task metadata
	GetName() string
	GetType() types.TaskType
	GetVersion() string
	GetDescription() string
	GetDefaultTimeout() time.Duration
	GetDefaultPriority() types.TaskPriority

	// Task capabilities
	CanRunConcurrently() bool
	GetRequiredPermissions() []string
	GetRequiredResources() *TaskResourceRequirements
	GetSupportedParameters() []TaskParameterDefinition

	// Task lifecycle hooks
	BeforeExecute(ctx context.Context, params types.TaskParameters) error
	AfterExecute(ctx context.Context, params types.TaskParameters, result *types.TaskResult) error
	OnError(ctx context.Context, params types.TaskParameters, err error) error
	OnTimeout(ctx context.Context, params types.TaskParameters) error
	OnCancel(ctx context.Context, params types.TaskParameters) error

	// Task monitoring
	GetProgress(ctx context.Context) (float64, error)
	GetCurrentStatus(ctx context.Context) (string, error)
	IsHealthy(ctx context.Context) (bool, error)
}

// TaskRegistry defines the interface for task registration and discovery
type TaskRegistry interface {
	// Task registration
	RegisterTask(taskType types.TaskType, factory TaskFactory) error
	UnregisterTask(taskType types.TaskType) error
	GetTask(taskType types.TaskType) (Task, error)
	ListRegisteredTasks() []types.TaskType

	// Task metadata
	GetTaskInfo(taskType types.TaskType) (*TaskInfo, error)
	GetTaskCategories() []string
	GetTasksByCategory(category string) []types.TaskType

	// Task validation
	ValidateTaskType(taskType types.TaskType) error
	ValidateTaskConfig(config *types.TaskConfig) error
	GetTaskDependencies(taskType types.TaskType) ([]types.TaskType, error)

	// Task discovery
	DiscoverTasks(ctx context.Context) ([]types.TaskType, error)
	RefreshRegistry(ctx context.Context) error
}

// TaskExecutor defines the interface for task execution management
type TaskExecutor interface {
	// Task execution
	ExecuteTask(ctx context.Context, task Task, params types.TaskParameters) (*types.TaskResult, error)
	ExecuteTaskAsync(ctx context.Context, task Task, params types.TaskParameters) (*types.TaskExecution, error)
	ExecuteTaskWithCallback(ctx context.Context, task Task, params types.TaskParameters, callback TaskExecutionCallback) error

	// Execution control
	CancelExecution(executionID string) error
	PauseExecution(executionID string) error
	ResumeExecution(executionID string) error
	KillExecution(executionID string) error

	// Execution monitoring
	GetExecution(executionID string) (*types.TaskExecution, error)
	GetActiveExecutions() []*types.TaskExecution
	GetExecutionHistory(filter *ExecutionHistoryFilter) ([]*types.TaskExecution, error)
	StreamExecutionUpdates(executionID string) (<-chan *types.TaskExecution, error)

	// Resource management
	SetConcurrencyLimit(limit int)
	GetConcurrencyLimit() int
	GetResourceUsage() (*ExecutorResourceUsage, error)
	SetResourceLimits(limits *ExecutorResourceLimits) error

	// Execution policies
	SetRetryPolicy(policy *types.TaskRetryPolicy) error
	GetRetryPolicy() (*types.TaskRetryPolicy, error)
	SetTimeoutPolicy(policy *TimeoutPolicy) error
	GetTimeoutPolicy() (*TimeoutPolicy, error)
}

// TaskQueue defines the interface for task queue management
type TaskQueue interface {
	// Queue operations
	Enqueue(ctx context.Context, task *QueuedTask) error
	Dequeue(ctx context.Context, workerID string) (*QueuedTask, error)
	Peek(ctx context.Context, count int) ([]*QueuedTask, error)
	Size(ctx context.Context) (int64, error)

	// Priority queue operations
	EnqueueWithPriority(ctx context.Context, task *QueuedTask, priority types.TaskPriority) error
	DequeueByPriority(ctx context.Context, workerID string, minPriority types.TaskPriority) (*QueuedTask, error)
	ReorderByPriority(ctx context.Context) error

	// Queue management
	Clear(ctx context.Context) error
	Remove(ctx context.Context, taskID string) error
	Update(ctx context.Context, taskID string, task *QueuedTask) error
	Contains(ctx context.Context, taskID string) (bool, error)

	// Queue monitoring
	GetQueueStats(ctx context.Context) (*TaskQueueStats, error)
	GetQueuedTasks(ctx context.Context, filter *QueuedTaskFilter) ([]*QueuedTask, error)
	GetStuckTasks(ctx context.Context, threshold time.Duration) ([]*QueuedTask, error)

	// Dead letter queue
	GetDeadLetterQueue(ctx context.Context) ([]*QueuedTask, error)
	RetryDeadLetterTasks(ctx context.Context, taskIDs []string) error
	PurgeDeadLetterQueue(ctx context.Context) (int64, error)
}

// TaskSchedulerEngine defines the interface for the core scheduling engine
type TaskSchedulerEngine interface {
	// Scheduling engine lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool

	// Schedule management
	AddSchedule(ctx context.Context, schedule *TaskSchedule) error
	RemoveSchedule(ctx context.Context, scheduleID string) error
	UpdateSchedule(ctx context.Context, schedule *TaskSchedule) error
	GetSchedule(ctx context.Context, scheduleID string) (*TaskSchedule, error)
	ListSchedules(ctx context.Context, filter *ScheduleFilter) ([]*TaskSchedule, error)

	// Schedule evaluation
	EvaluateSchedules(ctx context.Context, currentTime time.Time) ([]*types.ScheduledTask, error)
	GetNextRunTime(ctx context.Context, scheduleID string) (*time.Time, error)
	GetScheduleHistory(ctx context.Context, scheduleID string, limit int) ([]*ScheduleExecution, error)

	// Schedule monitoring
	GetScheduleStats(ctx context.Context) (*ScheduleStats, error)
	GetMissedSchedules(ctx context.Context, since time.Time) ([]*MissedSchedule, error)
	GetUpcomingSchedules(ctx context.Context, duration time.Duration) ([]*UpcomingSchedule, error)

	// Timezone and calendar support
	SetTimezone(timezone string) error
	GetTimezone() string
	AddHoliday(ctx context.Context, holiday *Holiday) error
	RemoveHoliday(ctx context.Context, holidayID string) error
	IsHoliday(ctx context.Context, date time.Time) (bool, error)
}

// TaskDependencyManager defines the interface for managing task dependencies
type TaskDependencyManager interface {
	// Dependency management
	AddDependency(ctx context.Context, taskID int, dependency *types.TaskDependency) error
	RemoveDependency(ctx context.Context, taskID int, dependencyID string) error
	GetDependencies(ctx context.Context, taskID int) ([]*types.TaskDependency, error)
	UpdateDependency(ctx context.Context, taskID int, dependency *types.TaskDependency) error

	// Dependency resolution
	ResolveDependencies(ctx context.Context, taskID int) (*DependencyResolution, error)
	CanExecuteTask(ctx context.Context, taskID int) (bool, []string, error)
	GetExecutionOrder(ctx context.Context, taskIDs []int) ([]int, error)
	DetectCircularDependencies(ctx context.Context, taskID int) (*CircularDependencyResult, error)

	// Dependency monitoring
	GetDependencyGraph(ctx context.Context) (*DependencyGraph, error)
	GetBlockedTasks(ctx context.Context) ([]*types.ScheduledTask, error)
	GetDependentTasks(ctx context.Context, taskID int) ([]*types.ScheduledTask, error)

	// Dependency policies
	SetDependencyPolicy(ctx context.Context, policy *DependencyPolicy) error
	GetDependencyPolicy(ctx context.Context) (*DependencyPolicy, error)
}

// TaskNotificationManager defines the interface for task notifications
type TaskNotificationManager interface {
	// Notification management
	CreateNotification(ctx context.Context, notification *types.TaskNotification) error
	UpdateNotification(ctx context.Context, notificationID int, notification *types.TaskNotification) error
	DeleteNotification(ctx context.Context, notificationID int) error
	GetNotification(ctx context.Context, notificationID int) (*types.TaskNotification, error)
	ListNotifications(ctx context.Context, filter *SchedulerNotificationFilter) ([]*types.TaskNotification, error)

	// Notification triggers
	TriggerNotification(ctx context.Context, event *types.SchedulerEvent) error
	RegisterNotificationHandler(eventType types.SchedulerEventType, handler NotificationHandler) error
	UnregisterNotificationHandler(eventType types.SchedulerEventType) error

	// Notification channels
	AddNotificationChannel(ctx context.Context, channel NotificationChannel) error
	RemoveNotificationChannel(ctx context.Context, channelID string) error
	GetNotificationChannels(ctx context.Context) ([]NotificationChannel, error)
	TestNotificationChannel(ctx context.Context, channelID string, testMessage *TestNotificationMessage) error

	// Notification history
	GetNotificationHistory(ctx context.Context, filter *SchedulerNotificationHistoryFilter) ([]*SchedulerNotificationHistoryEntry, error)
	GetNotificationStats(ctx context.Context, period time.Duration) (*SchedulerNotificationStats, error)
}

// TaskMonitor defines the interface for task monitoring and observability
type TaskMonitor interface {
	// Task monitoring
	StartMonitoring(ctx context.Context) error
	StopMonitoring(ctx context.Context) error
	IsMonitoring() bool

	// Performance monitoring
	GetTaskPerformance(ctx context.Context, taskID int, period time.Duration) (*TaskPerformanceMetrics, error)
	GetSystemPerformance(ctx context.Context, period time.Duration) (*SystemPerformanceMetrics, error)
	GetResourceUtilization(ctx context.Context) (*ResourceUtilization, error)

	// Health monitoring
	CheckTaskHealth(ctx context.Context, taskID int) (*TaskHealthStatus, error)
	GetUnhealthyTasks(ctx context.Context) ([]*types.ScheduledTask, error)
	MonitorTaskHealth(ctx context.Context, taskID int, interval time.Duration) (<-chan *TaskHealthStatus, error)

	// Alert management
	CreateAlert(ctx context.Context, alert *TaskAlert) error
	UpdateAlert(ctx context.Context, alertID string, alert *TaskAlert) error
	DeleteAlert(ctx context.Context, alertID string) error
	GetAlerts(ctx context.Context, filter *TaskAlertFilter) ([]*TaskAlert, error)
	TriggerAlert(ctx context.Context, alert *TaskAlert, data map[string]interface{}) error

	// Metrics collection
	CollectMetrics(ctx context.Context) (*TaskMetricsSnapshot, error)
	GetMetricsHistory(ctx context.Context, period time.Duration) ([]*TaskMetricsSnapshot, error)
	ExportMetrics(ctx context.Context, format string) ([]byte, error)
}

// Supporting interfaces and types

// SchedulerEventListener defines the interface for scheduler event listeners
type SchedulerEventListener interface {
	OnEvent(event *types.SchedulerEvent) error
	GetListenerID() string
	GetSupportedEvents() []types.SchedulerEventType
}

// TaskFactory creates task instances
type TaskFactory func() Task

// TaskExecutionCallback defines callback for task execution
type TaskExecutionCallback func(ctx context.Context, execution *types.TaskExecution, result *types.TaskResult, err error)

// NotificationHandler defines the interface for notification handlers
type NotificationHandler interface {
	HandleNotification(ctx context.Context, notification *NotificationMessage) error
	GetHandlerName() string
	GetSupportedEvents() []types.SchedulerEventType
}

// NotificationChannel defines the interface for notification channels
type NotificationChannel interface {
	Send(ctx context.Context, message *NotificationMessage) error
	GetChannelID() string
	GetChannelType() string
	IsHealthy(ctx context.Context) (bool, error)
	ValidateConfig() error
}

// Supporting types and structures

// SchedulerStatus represents the status of the scheduler
type SchedulerStatus struct {
	IsRunning          bool      `json:"is_running"`
	StartTime          time.Time `json:"start_time"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
	ActiveTasks        int       `json:"active_tasks"`
	QueuedTasks        int       `json:"queued_tasks"`
	CompletedTasks     int64     `json:"completed_tasks"`
	FailedTasks        int64     `json:"failed_tasks"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	MemoryUsage        int64     `json:"memory_usage"`
	CPUUsage           float64   `json:"cpu_usage"`
}

// TaskFilter represents filters for task queries
type TaskFilter struct {
	Types       []types.TaskType        `json:"types,omitempty"`
	Status      []types.ExecutionStatus `json:"status,omitempty"`
	Priority    []types.TaskPriority    `json:"priority,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	IsActive    *bool                   `json:"is_active,omitempty"`
	IsPaused    *bool                   `json:"is_paused,omitempty"`
	CreatedBy   *int                    `json:"created_by,omitempty"`
	FromTime    *time.Time              `json:"from_time,omitempty"`
	ToTime      *time.Time              `json:"to_time,omitempty"`
	NamePattern string                  `json:"name_pattern,omitempty"`
	Limit       int                     `json:"limit,omitempty"`
	Offset      int                     `json:"offset,omitempty"`
	SortBy      string                  `json:"sort_by,omitempty"`
	SortOrder   string                  `json:"sort_order,omitempty"`
}

// Additional supporting types (abbreviated for space)
type TaskHistoryFilter struct{}
type BulkOperationResult struct{}
type TaskResourceRequirements struct{}
type TaskParameterDefinition struct{}
type TaskInfo struct{}
type ExecutionHistoryFilter struct{}
type ExecutorResourceUsage struct{}
type ExecutorResourceLimits struct{}
type TimeoutPolicy struct{}
type QueuedTask struct{}
type TaskQueueStats struct{}
type QueuedTaskFilter struct{}
type TaskSchedule struct{}
type ScheduleFilter struct{}
type ScheduleExecution struct{}
type ScheduleStats struct{}
type MissedSchedule struct{}
type UpcomingSchedule struct{}
type Holiday struct{}
type DependencyResolution struct{}
type CircularDependencyResult struct{}
type DependencyGraph struct{}
type DependencyPolicy struct{}
type SchedulerNotificationFilter struct{}
type TestNotificationMessage struct{}
type SchedulerNotificationHistoryFilter struct{}
type SchedulerNotificationHistoryEntry struct{}
type SchedulerNotificationStats struct{}
type NotificationMessage struct{}
type TaskPerformanceMetrics struct{}
type SystemPerformanceMetrics struct{}
type ResourceUtilization struct{}
type TaskHealthStatus struct{}
type TaskAlert struct{}
type TaskAlertFilter struct{}
type TaskMetricsSnapshot struct{}