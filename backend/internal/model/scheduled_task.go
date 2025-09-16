package model

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// ScheduledTask represents scheduled tasks in the system
type ScheduledTask struct {
	ID               int              `json:"id" gorm:"primaryKey;autoIncrement"`
	Name             string           `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Type             TaskType         `json:"type" gorm:"not null;index:idx_scheduled_tasks_type"`
	CronExpression   string           `json:"cron_expression" gorm:"not null;size:100"`
	TargetContainers string           `json:"target_containers,omitempty" gorm:"type:jsonb;default:'[]'"`
	Parameters       string           `json:"parameters,omitempty" gorm:"type:jsonb;default:'{}'"`
	IsActive         bool             `json:"is_active" gorm:"not null;default:true;index:idx_scheduled_tasks_is_active"`
	LastRunAt        *time.Time       `json:"last_run_at,omitempty"`
	NextRunAt        *time.Time       `json:"next_run_at,omitempty" gorm:"index:idx_scheduled_tasks_next_run_at"`
	RunCount         int              `json:"run_count" gorm:"default:0"`
	FailureCount     int              `json:"failure_count" gorm:"default:0"`
	CreatedBy        *int             `json:"created_by,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`

	// Relationships
	CreatedByUser     *User                `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`
	TaskExecutionLogs []TaskExecutionLog   `json:"execution_logs,omitempty" gorm:"foreignKey:TaskID"`
}

// TaskExecutionLog represents task execution logs
type TaskExecutionLog struct {
	ID              int              `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID          int              `json:"task_id" gorm:"not null;index:idx_task_execution_logs_task_id"`
	Status          ExecutionStatus  `json:"status" gorm:"not null;default:'running';index:idx_task_execution_logs_status"`
	Message         string           `json:"message,omitempty" gorm:"type:text"`
	DurationSeconds int              `json:"duration_seconds" gorm:"default:0"`
	StartedAt       time.Time        `json:"started_at" gorm:"index:idx_task_execution_logs_started_at,sort:desc"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty"`

	// Relationships
	Task ScheduledTask `json:"-" gorm:"foreignKey:TaskID"`
}

// TaskType defines types of scheduled tasks
type TaskType string

const (
	TaskTypeImageCheck    TaskType = "image_check"
	TaskTypeContainerUpdate TaskType = "container_update"
	TaskTypeCleanup       TaskType = "cleanup"
	TaskTypeBackup        TaskType = "backup"
	TaskTypeHealthCheck   TaskType = "health_check"
)

// ExecutionStatus defines task execution status
type ExecutionStatus string

const (
	ExecutionStatusRunning ExecutionStatus = "running"
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailed  ExecutionStatus = "failed"
	ExecutionStatusTimeout ExecutionStatus = "timeout"
)

// TaskParameters represents different task parameter structures
type TaskParameters struct {
	ImageCheck    *ImageCheckParams    `json:"image_check,omitempty"`
	ContainerUpdate *ContainerUpdateParams `json:"container_update,omitempty"`
	Cleanup       *CleanupParams       `json:"cleanup,omitempty"`
	Backup        *BackupParams        `json:"backup,omitempty"`
	HealthCheck   *HealthCheckParams   `json:"health_check,omitempty"`
}

// ImageCheckParams represents parameters for image check tasks
type ImageCheckParams struct {
	RegistryTimeout   int      `json:"registry_timeout"`
	MaxConcurrent     int      `json:"max_concurrent"`
	CheckTags         []string `json:"check_tags"`
	IgnoreArchs       []string `json:"ignore_archs"`
	NotifyOnNewImage  bool     `json:"notify_on_new_image"`
}

// ContainerUpdateParams represents parameters for container update tasks
type ContainerUpdateParams struct {
	UpdateStrategy    string   `json:"update_strategy"`
	MaxConcurrent     int      `json:"max_concurrent"`
	RollbackOnFailure bool     `json:"rollback_on_failure"`
	PreUpdateBackup   bool     `json:"pre_update_backup"`
	UpdateTimeout     int      `json:"update_timeout"`
	ExcludeTags       []string `json:"exclude_tags"`
}

// CleanupParams represents parameters for cleanup tasks
type CleanupParams struct {
	LogRetentionDays     int  `json:"log_retention_days"`
	HistoryRetentionCount int `json:"history_retention_count"`
	ImageCacheRetentionDays int `json:"image_cache_retention_days"`
	CleanupUnusedImages  bool `json:"cleanup_unused_images"`
	CleanupDanglingImages bool `json:"cleanup_dangling_images"`
}

// BackupParams represents parameters for backup tasks
type BackupParams struct {
	BackupType        string   `json:"backup_type"`
	StoragePath       string   `json:"storage_path"`
	RetentionDays     int      `json:"retention_days"`
	CompressBackups   bool     `json:"compress_backups"`
	IncludeVolumes    bool     `json:"include_volumes"`
	ExcludeContainers []string `json:"exclude_containers"`
}

// HealthCheckParams represents parameters for health check tasks
type HealthCheckParams struct {
	CheckTimeout      int      `json:"check_timeout"`
	MaxRetries        int      `json:"max_retries"`
	NotifyOnFailure   bool     `json:"notify_on_failure"`
	RestartOnFailure  bool     `json:"restart_on_failure"`
	CheckServices     []string `json:"check_services"`
}

// ScheduledTaskFilter represents filters for querying scheduled tasks
type ScheduledTaskFilter struct {
	Name      string   `json:"name,omitempty"`
	Type      TaskType `json:"type,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
	CreatedBy *int     `json:"created_by,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
	OrderBy   string   `json:"order_by,omitempty"`
}

// TaskExecutionLogFilter represents filters for querying task execution logs
type TaskExecutionLogFilter struct {
	TaskID          *int            `json:"task_id,omitempty"`
	Status          ExecutionStatus `json:"status,omitempty"`
	StartedAfter    *time.Time      `json:"started_after,omitempty"`
	StartedBefore   *time.Time      `json:"started_before,omitempty"`
	CompletedAfter  *time.Time      `json:"completed_after,omitempty"`
	CompletedBefore *time.Time      `json:"completed_before,omitempty"`
	Limit           int             `json:"limit,omitempty"`
	Offset          int             `json:"offset,omitempty"`
	OrderBy         string          `json:"order_by,omitempty"`
}

// TableName returns the table name for ScheduledTask model
func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}

// TableName returns the table name for TaskExecutionLog model
func (TaskExecutionLog) TableName() string {
	return "task_execution_logs"
}

// IsTaskActive checks if the task is active
func (st *ScheduledTask) IsTaskActive() bool {
	return st.IsActive
}

// ShouldRun checks if the task should run now
func (st *ScheduledTask) ShouldRun() bool {
	if !st.IsActive {
		return false
	}
	if st.NextRunAt == nil {
		return true
	}
	return time.Now().After(*st.NextRunAt)
}

// CalculateNextRun calculates the next run time based on cron expression
func (st *ScheduledTask) CalculateNextRun() error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(st.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}

	nextRun := schedule.Next(time.Now())
	st.NextRunAt = &nextRun
	return nil
}

// ValidateCronExpression validates the cron expression
func (st *ScheduledTask) ValidateCronExpression() error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(st.CronExpression)
	return err
}

// GetSuccessRate returns the success rate of the task
func (st *ScheduledTask) GetSuccessRate() float64 {
	if st.RunCount == 0 {
		return 0
	}
	successCount := st.RunCount - st.FailureCount
	return float64(successCount) / float64(st.RunCount) * 100
}

// IncrementRunCount increments the run count
func (st *ScheduledTask) IncrementRunCount() {
	st.RunCount++
	st.LastRunAt = &time.Time{}
	*st.LastRunAt = time.Now()
}

// IncrementFailureCount increments the failure count
func (st *ScheduledTask) IncrementFailureCount() {
	st.FailureCount++
}

// IsCompleted checks if the task execution is completed
func (tel *TaskExecutionLog) IsCompleted() bool {
	return tel.Status == ExecutionStatusSuccess ||
		tel.Status == ExecutionStatusFailed ||
		tel.Status == ExecutionStatusTimeout
}

// IsSuccessful checks if the task execution was successful
func (tel *TaskExecutionLog) IsSuccessful() bool {
	return tel.Status == ExecutionStatusSuccess
}

// GetDuration returns the execution duration
func (tel *TaskExecutionLog) GetDuration() time.Duration {
	if tel.CompletedAt != nil {
		return tel.CompletedAt.Sub(tel.StartedAt)
	}
	if tel.IsCompleted() {
		return time.Duration(tel.DurationSeconds) * time.Second
	}
	return time.Since(tel.StartedAt)
}

// MarkAsCompleted marks the task execution as completed
func (tel *TaskExecutionLog) MarkAsCompleted(status ExecutionStatus, message string) {
	tel.Status = status
	tel.Message = message
	now := time.Now()
	tel.CompletedAt = &now
	tel.DurationSeconds = int(now.Sub(tel.StartedAt).Seconds())
}

// GetValidTaskTypes returns all valid task types
func GetValidTaskTypes() []TaskType {
	return []TaskType{
		TaskTypeImageCheck,
		TaskTypeContainerUpdate,
		TaskTypeCleanup,
		TaskTypeBackup,
		TaskTypeHealthCheck,
	}
}

// GetValidExecutionStatuses returns all valid execution statuses
func GetValidExecutionStatuses() []ExecutionStatus {
	return []ExecutionStatus{
		ExecutionStatusRunning,
		ExecutionStatusSuccess,
		ExecutionStatusFailed,
		ExecutionStatusTimeout,
	}
}

// GetDefaultCronExpressions returns commonly used cron expressions
func GetDefaultCronExpressions() map[string]string {
	return map[string]string{
		"every_minute":      "* * * * *",
		"every_5_minutes":   "*/5 * * * *",
		"every_15_minutes":  "*/15 * * * *",
		"every_30_minutes":  "*/30 * * * *",
		"every_hour":        "0 * * * *",
		"every_2_hours":     "0 */2 * * *",
		"every_6_hours":     "0 */6 * * *",
		"every_12_hours":    "0 */12 * * *",
		"daily_at_midnight": "0 0 * * *",
		"daily_at_6am":      "0 6 * * *",
		"daily_at_noon":     "0 12 * * *",
		"weekly_sunday":     "0 0 * * 0",
		"monthly_1st":       "0 0 1 * *",
	}
}

// BeforeCreate hook for ScheduledTask model
func (st *ScheduledTask) BeforeCreate(tx *gorm.DB) error {
	if err := st.ValidateCronExpression(); err != nil {
		return err
	}
	if err := st.CalculateNextRun(); err != nil {
		return err
	}
	return nil
}

// BeforeUpdate hook for ScheduledTask model
func (st *ScheduledTask) BeforeUpdate(tx *gorm.DB) error {
	if err := st.ValidateCronExpression(); err != nil {
		return err
	}
	if err := st.CalculateNextRun(); err != nil {
		return err
	}
	return nil
}

// BeforeCreate hook for TaskExecutionLog model
func (tel *TaskExecutionLog) BeforeCreate(tx *gorm.DB) error {
	if tel.Status == "" {
		tel.Status = ExecutionStatusRunning
	}
	if tel.StartedAt.IsZero() {
		tel.StartedAt = time.Now()
	}
	return nil
}

// BeforeUpdate hook for TaskExecutionLog model
func (tel *TaskExecutionLog) BeforeUpdate(tx *gorm.DB) error {
	// Calculate duration when status changes to completed
	if tel.IsCompleted() && tel.CompletedAt == nil {
		now := time.Now()
		tel.CompletedAt = &now
		tel.DurationSeconds = int(now.Sub(tel.StartedAt).Seconds())
	}
	return nil
}

// TaskStatus defines task status (using ExecutionStatus for now)
type TaskStatus = ExecutionStatus

// ExecutionStats represents execution statistics for a task
type ExecutionStats struct {
	TotalExecutions     int     `json:"total_executions"`
	SuccessfulExecutions int    `json:"successful_executions"`
	FailedExecutions    int     `json:"failed_executions"`
	TimeoutExecutions   int     `json:"timeout_executions"`
	SuccessRate         float64 `json:"success_rate"`
	AverageExecutionTime int    `json:"average_execution_time"`
	LastExecutionTime   *time.Time `json:"last_execution_time,omitempty"`
	LastSuccessTime     *time.Time `json:"last_success_time,omitempty"`
	LastFailureTime     *time.Time `json:"last_failure_time,omitempty"`
}