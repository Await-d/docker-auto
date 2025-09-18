package model

import (
	"time"

	"gorm.io/gorm"
)

// UserNotification represents a user notification stored in database
type UserNotification struct {
	ID        int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    *int64                 `json:"user_id,omitempty" gorm:"index:idx_notifications_user_id"`
	Type      string                 `json:"type" gorm:"not null;size:50;index:idx_notifications_type"`
	Title     string                 `json:"title" gorm:"not null;size:255"`
	Message   string                 `json:"message" gorm:"type:text;not null"`
	Data      JSONMap `json:"data,omitempty" gorm:"type:text;default:'{}'"`
	IsRead    bool                   `json:"is_read" gorm:"not null;default:false;index:idx_notifications_is_read"`
	ReadAt    *time.Time             `json:"read_at,omitempty"`
	CreatedAt time.Time              `json:"created_at" gorm:"index:idx_notifications_created_at,sort:desc"`
	UpdatedAt time.Time              `json:"updated_at"`

	// Relationships
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// UserNotificationSettings represents user notification preferences
type UserNotificationSettings struct {
	ID                    int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                int64     `json:"user_id" gorm:"uniqueIndex;not null"`
	EmailNotifications    bool      `json:"email_notifications" gorm:"not null;default:true"`
	WebNotifications      bool      `json:"web_notifications" gorm:"not null;default:true"`
	ContainerUpdates      bool      `json:"container_updates" gorm:"not null;default:true"`
	SystemAlerts          bool      `json:"system_alerts" gorm:"not null;default:true"`
	TaskNotifications     bool      `json:"task_notifications" gorm:"not null;default:true"`
	SecurityAlerts        bool      `json:"security_alerts" gorm:"not null;default:true"`
	ImageUpdateAlerts     bool      `json:"image_update_alerts" gorm:"not null;default:true"`
	PerformanceAlerts     bool      `json:"performance_alerts" gorm:"not null;default:false"`
	MaintenanceNotices    bool      `json:"maintenance_notices" gorm:"not null;default:true"`
	WeeklyReports         bool      `json:"weekly_reports" gorm:"not null;default:false"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// UserNotificationFilter represents filters for querying user notifications
type UserNotificationFilter struct {
	UserID    *int64 `json:"user_id,omitempty"`
	Type      string `json:"type,omitempty"`
	IsRead    *bool  `json:"is_read,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	OrderBy   string `json:"order_by,omitempty"`
}

// TableName returns the table name for Notification model
func (Notification) TableName() string {
	return "user_notifications"
}

// TableName returns the table name for UserNotificationSettings model
func (UserNotificationSettings) TableName() string {
	return "user_notification_settings"
}

// IsUnread checks if the notification is unread
func (n *UserNotification) IsUnread() bool {
	return !n.IsRead
}

// MarkAsRead marks the notification as read
func (n *UserNotification) MarkAsRead() {
	n.IsRead = true
	now := time.Now()
	n.ReadAt = &now
}

// MarkAsUnread marks the notification as unread
func (n *UserNotification) MarkAsUnread() {
	n.IsRead = false
	n.ReadAt = nil
}

// IsBroadcast checks if the notification is a broadcast (no specific user)
func (n *UserNotification) IsBroadcast() bool {
	return n.UserID == nil
}

// GetAge returns the age of the notification
func (n *UserNotification) GetAge() time.Duration {
	return time.Since(n.CreatedAt)
}

// IsOlderThan checks if the notification is older than the specified duration
func (n *UserNotification) IsOlderThan(duration time.Duration) bool {
	return n.GetAge() > duration
}

// GetNotificationTypes returns common notification types
func GetNotificationTypes() []string {
	return []string{
		"info",
		"warning",
		"error",
		"success",
		"container_update",
		"system_alert",
		"task_completed",
		"task_failed",
		"image_update",
		"security_alert",
		"performance_alert",
		"maintenance",
	}
}

// BeforeCreate hook for Notification model
func (n *UserNotification) BeforeCreate(tx *gorm.DB) error {
	if n.Type == "" {
		n.Type = "info"
	}
	return nil
}

// BeforeCreate hook for UserNotificationSettings model
func (uns *UserNotificationSettings) BeforeCreate(tx *gorm.DB) error {
	// Default values are set in struct tags
	return nil
}

// GetDefaultNotificationSettings returns default notification settings for a new user
func GetDefaultNotificationSettings(userID int64) *UserNotificationSettings {
	return &UserNotificationSettings{
		UserID:                userID,
		EmailNotifications:    true,
		WebNotifications:      true,
		ContainerUpdates:      true,
		SystemAlerts:          true,
		TaskNotifications:     true,
		SecurityAlerts:        true,
		ImageUpdateAlerts:     true,
		PerformanceAlerts:     false,
		MaintenanceNotices:    true,
		WeeklyReports:         false,
	}
}

// ShouldSendEmail checks if email notifications are enabled for this type
func (uns *UserNotificationSettings) ShouldSendEmail(notificationType string) bool {
	if !uns.EmailNotifications {
		return false
	}

	switch notificationType {
	case "container_update":
		return uns.ContainerUpdates
	case "system_alert", "error":
		return uns.SystemAlerts
	case "task_completed", "task_failed":
		return uns.TaskNotifications
	case "security_alert":
		return uns.SecurityAlerts
	case "image_update":
		return uns.ImageUpdateAlerts
	case "performance_alert":
		return uns.PerformanceAlerts
	case "maintenance":
		return uns.MaintenanceNotices
	default:
		return true // Default to true for unknown types
	}
}

// ShouldSendWebNotification checks if web notifications are enabled for this type
func (uns *UserNotificationSettings) ShouldSendWebNotification(notificationType string) bool {
	if !uns.WebNotifications {
		return false
	}

	switch notificationType {
	case "container_update":
		return uns.ContainerUpdates
	case "system_alert", "error":
		return uns.SystemAlerts
	case "task_completed", "task_failed":
		return uns.TaskNotifications
	case "security_alert":
		return uns.SecurityAlerts
	case "image_update":
		return uns.ImageUpdateAlerts
	case "performance_alert":
		return uns.PerformanceAlerts
	case "maintenance":
		return uns.MaintenanceNotices
	default:
		return true // Default to true for unknown types
	}
}