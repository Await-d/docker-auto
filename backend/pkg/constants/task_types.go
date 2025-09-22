// Package constants provides centralized constant definitions for the docker-auto system.
// This package ensures consistency across the codebase and avoids magic strings.
package constants

// Task type constants for the scheduler system
const (
	// TaskTypeImageCleanup represents image cleanup tasks
	TaskTypeImageCleanup = "image_cleanup"

	// TaskTypeSystemMaintenance represents system maintenance tasks
	TaskTypeSystemMaintenance = "system_maintenance"

	// TaskTypeContainerUpdate represents container update tasks
	TaskTypeContainerUpdate = "container_update"

	// TaskTypeHealthCheck represents health check tasks
	TaskTypeHealthCheck = "health_check"

	// TaskTypeBackup represents backup tasks
	TaskTypeBackup = "backup"

	// TaskTypeMonitoring represents monitoring tasks
	TaskTypeMonitoring = "monitoring"

	// TaskTypeCleanup represents general cleanup tasks
	TaskTypeCleanup = "cleanup"

	// TaskTypeNotification represents notification tasks
	TaskTypeNotification = "notification"

	// TaskTypeSecurityScan represents security scan tasks
	TaskTypeSecurityScan = "security_scan"

	// TaskTypeLogRotation represents log rotation tasks
	TaskTypeLogRotation = "log_rotation"

	// TaskTypeImagePull represents image pull tasks
	TaskTypeImagePull = "image_pull"

	// TaskTypeContainerRestart represents container restart tasks
	TaskTypeContainerRestart = "container_restart"

	// TaskTypeVolumeCleanup represents volume cleanup tasks
	TaskTypeVolumeCleanup = "volume_cleanup"

	// TaskTypeNetworkCleanup represents network cleanup tasks
	TaskTypeNetworkCleanup = "network_cleanup"

	// TaskTypeRegistrySync represents registry synchronization tasks
	TaskTypeRegistrySync = "registry_sync"

	// TaskTypeMetricsCollection represents metrics collection tasks
	TaskTypeMetricsCollection = "metrics_collection"

	// TaskTypeAlertProcessing represents alert processing tasks
	TaskTypeAlertProcessing = "alert_processing"

	// TaskTypeConfigSync represents configuration synchronization tasks
	TaskTypeConfigSync = "config_sync"

	// TaskTypeResourceOptimization represents resource optimization tasks
	TaskTypeResourceOptimization = "resource_optimization"
)

// Task execution status constants
const (
	// ExecutionStatusPending indicates task is pending execution
	ExecutionStatusPending = "pending"

	// ExecutionStatusRunning indicates task is currently running
	ExecutionStatusRunning = "running"

	// ExecutionStatusCompleted indicates task has completed successfully
	ExecutionStatusCompleted = "completed"

	// ExecutionStatusFailed indicates task has failed
	ExecutionStatusFailed = "failed"

	// ExecutionStatusCancelled indicates task was cancelled
	ExecutionStatusCancelled = "cancelled"

	// ExecutionStatusTimeout indicates task timed out
	ExecutionStatusTimeout = "timeout"

	// ExecutionStatusRetrying indicates task is being retried
	ExecutionStatusRetrying = "retrying"

	// ExecutionStatusPaused indicates task is paused
	ExecutionStatusPaused = "paused"

	// ExecutionStatusSkipped indicates task was skipped
	ExecutionStatusSkipped = "skipped"

	// ExecutionStatusQueued indicates task is queued for execution
	ExecutionStatusQueued = "queued"
)

// Scheduler event type constants
const (
	// EventSchedulerStarted indicates scheduler has started
	EventSchedulerStarted = "scheduler_started"

	// EventSchedulerStopped indicates scheduler has stopped
	EventSchedulerStopped = "scheduler_stopped"

	// EventTaskAdded indicates a task was added
	EventTaskAdded = "task_added"

	// EventTaskRemoved indicates a task was removed
	EventTaskRemoved = "task_removed"

	// EventTaskUpdated indicates a task was updated
	EventTaskUpdated = "task_updated"

	// EventTaskPaused indicates a task was paused
	EventTaskPaused = "task_paused"

	// EventTaskResumed indicates a task was resumed
	EventTaskResumed = "task_resumed"

	// EventTaskStarted indicates a task started execution
	EventTaskStarted = "task_started"

	// EventTaskCompleted indicates a task completed successfully
	EventTaskCompleted = "task_completed"

	// EventTaskFailed indicates a task failed
	EventTaskFailed = "task_failed"

	// EventTaskTimeout indicates a task timed out
	EventTaskTimeout = "task_timeout"

	// EventTaskRetried indicates a task was retried
	EventTaskRetried = "task_retried"

	// EventTaskCancelled indicates a task was cancelled
	EventTaskCancelled = "task_cancelled"

	// EventTaskSkipped indicates a task was skipped
	EventTaskSkipped = "task_skipped"
)

// Default task configuration constants
const (
	// DefaultTaskTimeout is the default timeout for task execution
	DefaultTaskTimeout = "30m"

	// DefaultMaxRetries is the default maximum number of retries
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default delay between retries
	DefaultRetryDelay = "1m"

	// DefaultCleanupInterval is the default interval for cleanup operations
	DefaultCleanupInterval = "1h"

	// DefaultHistoryRetention is the default retention period for task history
	DefaultHistoryRetention = "7d"

	// DefaultMaxConcurrentTasks is the default maximum number of concurrent tasks
	DefaultMaxConcurrentTasks = 10

	// DefaultQueueSize is the default size of the task queue
	DefaultQueueSize = 1000

	// DefaultWorkerPoolSize is the default size of the worker pool
	DefaultWorkerPoolSize = 5
)

// Task category constants
const (
	// TaskCategoryMaintenance represents maintenance tasks
	TaskCategoryMaintenance = "maintenance"

	// TaskCategoryMonitoring represents monitoring tasks
	TaskCategoryMonitoring = "monitoring"

	// TaskCategorySecurity represents security-related tasks
	TaskCategorySecurity = "security"

	// TaskCategoryBackup represents backup tasks
	TaskCategoryBackup = "backup"

	// TaskCategoryUpdate represents update tasks
	TaskCategoryUpdate = "update"

	// TaskCategoryCleanup represents cleanup tasks
	TaskCategoryCleanup = "cleanup"

	// TaskCategoryNotification represents notification tasks
	TaskCategoryNotification = "notification"

	// TaskCategoryOptimization represents optimization tasks
	TaskCategoryOptimization = "optimization"
)

// Task dependency condition constants
const (
	// DependencyConditionSuccess requires dependent task to succeed
	DependencyConditionSuccess = "success"

	// DependencyConditionFailure requires dependent task to fail
	DependencyConditionFailure = "failure"

	// DependencyConditionCompletion requires dependent task to complete (success or failure)
	DependencyConditionCompletion = "completion"

	// DependencyConditionSkipped requires dependent task to be skipped
	DependencyConditionSkipped = "skipped"
)

// Task notification event constants
const (
	// NotificationEventTaskStarted triggers when task starts
	NotificationEventTaskStarted = "task_started"

	// NotificationEventTaskCompleted triggers when task completes successfully
	NotificationEventTaskCompleted = "task_completed"

	// NotificationEventTaskFailed triggers when task fails
	NotificationEventTaskFailed = "task_failed"

	// NotificationEventTaskTimeout triggers when task times out
	NotificationEventTaskTimeout = "task_timeout"

	// NotificationEventTaskRetried triggers when task is retried
	NotificationEventTaskRetried = "task_retried"

	// NotificationEventTaskCancelled triggers when task is cancelled
	NotificationEventTaskCancelled = "task_cancelled"

	// NotificationEventTaskStuck triggers when task appears stuck
	NotificationEventTaskStuck = "task_stuck"

	// NotificationEventHighFailureRate triggers when failure rate is high
	NotificationEventHighFailureRate = "high_failure_rate"
)

// Predefined task names for common operations
const (
	// TaskNameDailyCleanup represents daily cleanup task
	TaskNameDailyCleanup = "daily_cleanup"

	// TaskNameWeeklyMaintenance represents weekly maintenance task
	TaskNameWeeklyMaintenance = "weekly_maintenance"

	// TaskNameHourlyHealthCheck represents hourly health check task
	TaskNameHourlyHealthCheck = "hourly_health_check"

	// TaskNameImageUpdateCheck represents image update check task
	TaskNameImageUpdateCheck = "image_update_check"

	// TaskNameSystemMonitoring represents system monitoring task
	TaskNameSystemMonitoring = "system_monitoring"

	// TaskNameSecurityScan represents security scan task
	TaskNameSecurityScan = "security_scan"

	// TaskNameBackupCreation represents backup creation task
	TaskNameBackupCreation = "backup_creation"

	// TaskNameLogRotation represents log rotation task
	TaskNameLogRotation = "log_rotation"

	// TaskNameMetricsCollection represents metrics collection task
	TaskNameMetricsCollection = "metrics_collection"

	// TaskNameAlertProcessing represents alert processing task
	TaskNameAlertProcessing = "alert_processing"
)

// Common cron expressions for predefined schedules
const (
	// CronDaily represents daily execution at midnight
	CronDaily = "0 0 * * *"

	// CronHourly represents hourly execution
	CronHourly = "0 * * * *"

	// CronWeekly represents weekly execution on Sunday at midnight
	CronWeekly = "0 0 * * 0"

	// CronMonthly represents monthly execution on the 1st at midnight
	CronMonthly = "0 0 1 * *"

	// CronEvery5Minutes represents execution every 5 minutes
	CronEvery5Minutes = "*/5 * * * *"

	// CronEvery15Minutes represents execution every 15 minutes
	CronEvery15Minutes = "*/15 * * * *"

	// CronEvery30Minutes represents execution every 30 minutes
	CronEvery30Minutes = "*/30 * * * *"

	// CronTwiceDaily represents execution twice daily (6 AM and 6 PM)
	CronTwiceDaily = "0 6,18 * * *"

	// CronBusinessHours represents execution during business hours (9 AM - 5 PM)
	CronBusinessHours = "0 9-17 * * 1-5"

	// CronNightlyMaintenance represents execution at 2 AM daily
	CronNightlyMaintenance = "0 2 * * *"
)