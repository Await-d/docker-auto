package repository

import (
	"context"
	"time"
	"docker-auto/internal/model"
)

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.UserFilter) ([]*model.User, int64, error)
	Exists(ctx context.Context, username, email string) (bool, error)
	GetActiveUsers(ctx context.Context) ([]*model.User, error)

	// Authentication operations
	UpdateLastLoginAt(ctx context.Context, userID int64) error
	SetUserStatus(ctx context.Context, userID int64, isActive bool) error

	// Batch operations
	CreateBatch(ctx context.Context, users []*model.User) error
	GetByIDs(ctx context.Context, ids []int64) ([]*model.User, error)
}

// UserSessionRepository defines the interface for user session repository operations
type UserSessionRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, session *model.UserSession) error
	GetByID(ctx context.Context, id string) (*model.UserSession, error)
	GetByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error)
	Update(ctx context.Context, session *model.UserSession) error
	Delete(ctx context.Context, id string) error

	// Query operations
	GetByUserID(ctx context.Context, userID int64) ([]*model.UserSession, error)
	DeleteExpiredSessions(ctx context.Context) error
	DeleteUserSessions(ctx context.Context, userID int64) error

	// Session management
	IsValidSession(ctx context.Context, refreshToken string) (bool, error)
	CleanupExpiredSessions(ctx context.Context) (int64, error)
}

// ActivityLogRepository defines the interface for activity log repository operations
type ActivityLogRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, log *model.ActivityLog) error
	GetByID(ctx context.Context, id int64) (*model.ActivityLog, error)
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.ActivityLogFilter) ([]*model.ActivityLog, int64, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*model.ActivityLog, int64, error)
	GetByResourceID(ctx context.Context, resourceType string, resourceID int64) ([]*model.ActivityLog, error)

	// Maintenance operations
	DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error)
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CreateBatch(ctx context.Context, logs []*model.ActivityLog) error
}

// ContainerRepository defines the interface for container repository operations
type ContainerRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, container *model.Container) error
	GetByID(ctx context.Context, id int64) (*model.Container, error)
	GetByName(ctx context.Context, name string) (*model.Container, error)
	GetByContainerID(ctx context.Context, containerID string) (*model.Container, error)
	Update(ctx context.Context, container *model.Container) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.ContainerFilter) ([]*model.Container, int64, error)
	GetByStatus(ctx context.Context, status model.ContainerStatus) ([]*model.Container, error)
	GetByUpdatePolicy(ctx context.Context, policy model.UpdatePolicy) ([]*model.Container, error)
	GetByCreatedBy(ctx context.Context, createdBy int64) ([]*model.Container, error)

	// Container management operations
	UpdateStatus(ctx context.Context, id int64, status model.ContainerStatus) error
	UpdateContainerID(ctx context.Context, id int64, containerID string) error
	GetAutoUpdateContainers(ctx context.Context) ([]*model.Container, error)

	// Batch operations
	UpdateStatusBatch(ctx context.Context, ids []int64, status model.ContainerStatus) error
	GetByIDs(ctx context.Context, ids []int64) ([]*model.Container, error)

	// Search operations
	SearchByImage(ctx context.Context, image string) ([]*model.Container, error)
	Exists(ctx context.Context, name string) (bool, error)
}

// RegistryCredentialsRepository defines the interface for registry credentials repository operations
type RegistryCredentialsRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, credentials *model.RegistryCredentials) error
	GetByID(ctx context.Context, id int64) (*model.RegistryCredentials, error)
	GetByName(ctx context.Context, name string) (*model.RegistryCredentials, error)
	Update(ctx context.Context, credentials *model.RegistryCredentials) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.RegistryCredentialsFilter) ([]*model.RegistryCredentials, int64, error)
	GetByRegistryURL(ctx context.Context, registryURL string) ([]*model.RegistryCredentials, error)
	GetDefault(ctx context.Context) (*model.RegistryCredentials, error)
	GetActive(ctx context.Context) ([]*model.RegistryCredentials, error)

	// Management operations
	SetDefault(ctx context.Context, id int64) error
	SetActive(ctx context.Context, id int64, isActive bool) error
	Exists(ctx context.Context, name string) (bool, error)
}

// UpdateHistoryRepository defines the interface for update history repository operations
type UpdateHistoryRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, history *model.UpdateHistory) error
	GetByID(ctx context.Context, id int64) (*model.UpdateHistory, error)
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.UpdateHistoryFilter) ([]*model.UpdateHistory, int64, error)
	GetByContainerID(ctx context.Context, containerID int64, limit, offset int) ([]*model.UpdateHistory, int64, error)
	GetByStatus(ctx context.Context, status model.UpdateStatus) ([]*model.UpdateHistory, error)
	GetRecent(ctx context.Context, limit int) ([]*model.UpdateHistory, error)

	// Statistics operations
	GetUpdateStats(ctx context.Context, containerID int64) (*model.UpdateStats, error)
	GetSuccessRate(ctx context.Context, containerID int64) (float64, error)

	// Maintenance operations
	DeleteOldHistory(ctx context.Context, retentionDays int) (int64, error)
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CreateBatch(ctx context.Context, histories []*model.UpdateHistory) error
}

// ImageVersionRepository defines the interface for image version repository operations
type ImageVersionRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, version *model.ImageVersion) error
	GetByID(ctx context.Context, id int64) (*model.ImageVersion, error)
	Update(ctx context.Context, version *model.ImageVersion) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.ImageVersionFilter) ([]*model.ImageVersion, int64, error)
	GetByImageName(ctx context.Context, imageName string) ([]*model.ImageVersion, error)
	GetByImageAndTag(ctx context.Context, imageName, tag string) (*model.ImageVersion, error)
	GetLatest(ctx context.Context, imageName string) (*model.ImageVersion, error)

	// Version management
	UpsertVersion(ctx context.Context, version *model.ImageVersion) error
	GetVersionHistory(ctx context.Context, imageName string, limit int) ([]*model.ImageVersion, error)
	DeleteOldVersions(ctx context.Context, imageName string, keepCount int) error

	// Cache operations
	RefreshImageCache(ctx context.Context, imageName string) error
	GetCachedVersions(ctx context.Context, imageName string) ([]*model.ImageVersion, error)

	// Cleanup operations
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)

	// Search and filter operations
	SearchByName(ctx context.Context, namePattern string) ([]*model.ImageVersion, error)
	GetOutdatedImages(ctx context.Context) ([]*model.ImageVersion, error)
}

// SystemConfigRepository defines the interface for system configuration repository operations
type SystemConfigRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, config *model.SystemConfig) error
	GetByID(ctx context.Context, id int64) (*model.SystemConfig, error)
	GetByKey(ctx context.Context, configKey string) (*model.SystemConfig, error)
	Update(ctx context.Context, config *model.SystemConfig) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.SystemConfigFilter) ([]*model.SystemConfig, int64, error)
	GetByCategory(ctx context.Context, category string) ([]*model.SystemConfig, error)
	GetPublicConfigs(ctx context.Context) ([]*model.SystemConfig, error)

	// Configuration management
	SetValue(ctx context.Context, configKey, value string) error
	GetValue(ctx context.Context, configKey string) (string, error)
	GetValueWithDefault(ctx context.Context, configKey, defaultValue string) (string, error)

	// Batch operations
	SetValues(ctx context.Context, configs map[string]string) error
	GetValues(ctx context.Context, keys []string) (map[string]string, error)

	// Cache operations
	RefreshConfigCache(ctx context.Context) error
	GetCachedValue(ctx context.Context, configKey string) (string, bool, error)

	// Validation and management
	ValidateConfig(ctx context.Context, configKey, value string) error
	ResetToDefault(ctx context.Context, configKey string) error
	GetConfigSchema(ctx context.Context) (map[string]interface{}, error)
}

// NotificationTemplateRepository defines the interface for notification template repository operations
type NotificationTemplateRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, template *model.NotificationTemplate) error
	GetByID(ctx context.Context, id int64) (*model.NotificationTemplate, error)
	GetByName(ctx context.Context, name string) (*model.NotificationTemplate, error)
	Update(ctx context.Context, template *model.NotificationTemplate) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.NotificationTemplateFilter) ([]*model.NotificationTemplate, int64, error)
	GetByType(ctx context.Context, notificationType model.NotificationType) ([]*model.NotificationTemplate, error)
	GetActive(ctx context.Context) ([]*model.NotificationTemplate, error)

	// Template management
	GetDefault(ctx context.Context, notificationType model.NotificationType) (*model.NotificationTemplate, error)
	SetActive(ctx context.Context, id int64, isActive bool) error
	ValidateTemplate(ctx context.Context, template *model.NotificationTemplate) error
}

// NotificationRepository defines the interface for user notification repository operations
type NotificationRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, notification *model.UserNotification) error
	GetByID(ctx context.Context, id int64) (*model.UserNotification, error)
	Update(ctx context.Context, notification *model.UserNotification) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.UserNotificationFilter) ([]*model.UserNotification, int64, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*model.UserNotification, error)
	GetByUserIDAndType(ctx context.Context, userID int64, notificationType string, limit, offset int) ([]*model.UserNotification, error)
	GetUnreadCount(ctx context.Context, userID int64) (int64, error)
	GetTotalCount(ctx context.Context, userID int64) (int64, error)
	GetCountByType(ctx context.Context, userID int64, notificationType string) (int64, error)

	// Notification management
	MarkAsRead(ctx context.Context, notificationID int64, userID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) (int64, error)

	// Cleanup operations
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CreateBatch(ctx context.Context, notifications []*model.UserNotification) error
}

// NotificationLogRepository defines the interface for notification log repository operations
type NotificationLogRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, log *model.NotificationLog) error
	GetByID(ctx context.Context, id int64) (*model.NotificationLog, error)
	Update(ctx context.Context, log *model.NotificationLog) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.NotificationLogFilter) ([]*model.NotificationLog, int64, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*model.NotificationLog, int64, error)
	GetByStatus(ctx context.Context, status model.NotificationStatus) ([]*model.NotificationLog, error)
	GetPending(ctx context.Context) ([]*model.NotificationLog, error)

	// Notification management
	UpdateStatus(ctx context.Context, id int64, status model.NotificationStatus) error
	MarkAsRead(ctx context.Context, userID int64, ids []int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error

	// Statistics and cleanup
	GetUnreadCount(ctx context.Context, userID int64) (int64, error)
	DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error)
	CreateBatch(ctx context.Context, logs []*model.NotificationLog) error
}

// ScheduledTaskRepository defines the interface for scheduled task repository operations
type ScheduledTaskRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, task *model.ScheduledTask) error
	GetByID(ctx context.Context, id int64) (*model.ScheduledTask, error)
	Update(ctx context.Context, task *model.ScheduledTask) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.ScheduledTaskFilter) ([]*model.ScheduledTask, int64, error)
	GetByType(ctx context.Context, taskType model.TaskType) ([]*model.ScheduledTask, error)
	GetByStatus(ctx context.Context, status model.TaskStatus) ([]*model.ScheduledTask, error)
	GetDueTasks(ctx context.Context) ([]*model.ScheduledTask, error)

	// Task management
	UpdateStatus(ctx context.Context, id int64, status model.TaskStatus) error
	UpdateLastRun(ctx context.Context, id int64) error
	UpdateNextRun(ctx context.Context, id int64) error
	SetEnabled(ctx context.Context, id int64, enabled bool) error

	// Execution tracking
	GetActiveTasks(ctx context.Context) ([]*model.ScheduledTask, error)
	GetOverdueTasks(ctx context.Context) ([]*model.ScheduledTask, error)
	CreateBatch(ctx context.Context, tasks []*model.ScheduledTask) error
}

// TaskExecutionLogRepository defines the interface for task execution log repository operations
type TaskExecutionLogRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, log *model.TaskExecutionLog) error
	GetByID(ctx context.Context, id int64) (*model.TaskExecutionLog, error)
	Update(ctx context.Context, log *model.TaskExecutionLog) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *model.TaskExecutionLogFilter) ([]*model.TaskExecutionLog, int64, error)
	GetByTaskID(ctx context.Context, taskID int64, limit, offset int) ([]*model.TaskExecutionLog, int64, error)
	GetByStatus(ctx context.Context, status model.ExecutionStatus) ([]*model.TaskExecutionLog, error)
	GetRecent(ctx context.Context, limit int) ([]*model.TaskExecutionLog, error)

	// Execution tracking
	GetRunningExecutions(ctx context.Context) ([]*model.TaskExecutionLog, error)
	GetFailedExecutions(ctx context.Context, since int) ([]*model.TaskExecutionLog, error)
	UpdateStatus(ctx context.Context, id int64, status model.ExecutionStatus) error

	// Statistics and cleanup
	GetExecutionStats(ctx context.Context, taskID int64) (*model.ExecutionStats, error)
	DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error)
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
	CreateBatch(ctx context.Context, logs []*model.TaskExecutionLog) error
}

// RepositoryManager aggregates all repositories
type RepositoryManager interface {
	// Repository getters
	User() UserRepository
	UserSession() UserSessionRepository
	ActivityLog() ActivityLogRepository
	Container() ContainerRepository
	RegistryCredentials() RegistryCredentialsRepository
	UpdateHistory() UpdateHistoryRepository
	ImageVersion() ImageVersionRepository
	SystemConfig() SystemConfigRepository
	NotificationTemplate() NotificationTemplateRepository
	Notification() NotificationRepository
	NotificationLog() NotificationLogRepository
	ScheduledTask() ScheduledTaskRepository
	TaskExecutionLog() TaskExecutionLogRepository

	// Transaction management
	WithTransaction(fn func(RepositoryManager) error) error

	// Health and maintenance
	HealthCheck() error
	Close() error
}