package model

import (
	"time"

	"gorm.io/gorm"
)

// NotificationTemplate represents notification templates
type NotificationTemplate struct {
	ID             int                   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string                `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Type           NotificationType      `json:"type" gorm:"not null"`
	TemplateSubject string               `json:"template_subject,omitempty" gorm:"size:255"`
	TemplateBody   string                `json:"template_body" gorm:"type:text;not null"`
	Variables      string                `json:"variables,omitempty" gorm:"type:jsonb;default:'[]'"`
	IsActive       bool                  `json:"is_active" gorm:"not null;default:true"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`

	// Relationships
	NotificationLogs []NotificationLog `json:"-" gorm:"foreignKey:TemplateID"`
}

// NotificationLog represents notification sending logs
type NotificationLog struct {
	ID           int              `json:"id" gorm:"primaryKey;autoIncrement"`
	TemplateID   *int             `json:"template_id,omitempty"`
	Recipient    string           `json:"recipient" gorm:"not null;size:255;index:idx_notification_logs_recipient"`
	Subject      string           `json:"subject,omitempty" gorm:"size:255"`
	Content      string           `json:"content" gorm:"type:text;not null"`
	Type         NotificationType `json:"type" gorm:"not null;index:idx_notification_logs_type"`
	Status       NotificationStatus `json:"status" gorm:"not null;default:'pending';index:idx_notification_logs_status"`
	ErrorMessage string           `json:"error_message,omitempty" gorm:"type:text"`
	SentAt       *time.Time       `json:"sent_at,omitempty" gorm:"index:idx_notification_logs_sent_at,sort:desc"`
	Metadata     string           `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time        `json:"created_at"`

	// Relationships
	Template *NotificationTemplate `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
}

// NotificationType defines notification types
type NotificationType string

const (
	NotificationTypeEmail            NotificationType = "email"
	NotificationTypeWebhook          NotificationType = "webhook"
	NotificationTypeWeChat           NotificationType = "wechat"
	NotificationTypeSlack            NotificationType = "slack"
	NotificationTypeDiscord          NotificationType = "discord"
	NotificationTypeBackup           NotificationType = "backup"
	NotificationTypeImageUpdate      NotificationType = "image_update"
	NotificationTypeSecurityUpdate   NotificationType = "security_update"
	NotificationTypeSystemMaintenance NotificationType = "system_maintenance"
	NotificationTypeContainerUpdate   NotificationType = "container_update"
)

// NotificationStatus defines notification status
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

// NotificationPriority defines notification priority levels
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityNormal NotificationPriority = "normal"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// Notification represents a runtime notification (not stored in database)
type Notification struct {
	Type     NotificationType     `json:"type"`
	Title    string               `json:"title"`
	Message  string               `json:"message"`
	Priority NotificationPriority `json:"priority"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// NotificationEvent represents different types of notification events
type NotificationEvent string

const (
	NotificationEventContainerUpdateSuccess NotificationEvent = "container_update_success"
	NotificationEventContainerUpdateFailed  NotificationEvent = "container_update_failed"
	NotificationEventContainerUpdateStarted NotificationEvent = "container_update_started"
	NotificationEventContainerStopped       NotificationEvent = "container_stopped"
	NotificationEventContainerStarted       NotificationEvent = "container_started"
	NotificationEventContainerError         NotificationEvent = "container_error"
	NotificationEventSystemAlert           NotificationEvent = "system_alert"
	NotificationEventImageCheckFailed      NotificationEvent = "image_check_failed"
	NotificationEventNewImageAvailable     NotificationEvent = "new_image_available"
)

// NotificationTemplateFilter represents filters for querying notification templates
type NotificationTemplateFilter struct {
	Name     string           `json:"name,omitempty"`
	Type     NotificationType `json:"type,omitempty"`
	IsActive *bool            `json:"is_active,omitempty"`
	Limit    int              `json:"limit,omitempty"`
	Offset   int              `json:"offset,omitempty"`
	OrderBy  string           `json:"order_by,omitempty"`
}

// NotificationLogFilter represents filters for querying notification logs
type NotificationLogFilter struct {
	TemplateID    *int               `json:"template_id,omitempty"`
	Recipient     string             `json:"recipient,omitempty"`
	Type          NotificationType   `json:"type,omitempty"`
	Status        NotificationStatus `json:"status,omitempty"`
	SentAfter     *time.Time         `json:"sent_after,omitempty"`
	SentBefore    *time.Time         `json:"sent_before,omitempty"`
	CreatedAfter  *time.Time         `json:"created_after,omitempty"`
	CreatedBefore *time.Time         `json:"created_before,omitempty"`
	Limit         int                `json:"limit,omitempty"`
	Offset        int                `json:"offset,omitempty"`
	OrderBy       string             `json:"order_by,omitempty"`
}

// NotificationConfig represents notification configuration for different channels
type NotificationConfig struct {
	Email   *EmailConfig   `json:"email,omitempty"`
	Webhook *WebhookConfig `json:"webhook,omitempty"`
	Slack   *SlackConfig   `json:"slack,omitempty"`
	WeChat  *WeChatConfig  `json:"wechat,omitempty"`
	Discord *DiscordConfig `json:"discord,omitempty"`
}

// EmailConfig represents email notification configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout"`
}

// SlackConfig represents Slack notification configuration
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

// WeChatConfig represents WeChat notification configuration
type WeChatConfig struct {
	CorpID     string `json:"corp_id"`
	CorpSecret string `json:"corp_secret"`
	AgentID    int    `json:"agent_id"`
}

// DiscordConfig represents Discord notification configuration
type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
}

// TableName returns the table name for NotificationTemplate model
func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

// TableName returns the table name for NotificationLog model
func (NotificationLog) TableName() string {
	return "notification_logs"
}

// IsSuccessful checks if the notification was sent successfully
func (nl *NotificationLog) IsSuccessful() bool {
	return nl.Status == NotificationStatusSent
}

// IsFailed checks if the notification failed to send
func (nl *NotificationLog) IsFailed() bool {
	return nl.Status == NotificationStatusFailed
}

// IsPending checks if the notification is pending
func (nl *NotificationLog) IsPending() bool {
	return nl.Status == NotificationStatusPending
}

// GetDuration returns the time taken to send the notification
func (nl *NotificationLog) GetDuration() *time.Duration {
	if nl.SentAt != nil {
		duration := nl.SentAt.Sub(nl.CreatedAt)
		return &duration
	}
	return nil
}

// MarkAsSent marks the notification as sent
func (nl *NotificationLog) MarkAsSent() {
	nl.Status = NotificationStatusSent
	now := time.Now()
	nl.SentAt = &now
}

// MarkAsFailed marks the notification as failed with error message
func (nl *NotificationLog) MarkAsFailed(errorMessage string) {
	nl.Status = NotificationStatusFailed
	nl.ErrorMessage = errorMessage
}

// GetValidNotificationTypes returns all valid notification types
func GetValidNotificationTypes() []NotificationType {
	return []NotificationType{
		NotificationTypeEmail,
		NotificationTypeWebhook,
		NotificationTypeWeChat,
		NotificationTypeSlack,
		NotificationTypeDiscord,
	}
}

// GetValidNotificationStatuses returns all valid notification statuses
func GetValidNotificationStatuses() []NotificationStatus {
	return []NotificationStatus{
		NotificationStatusPending,
		NotificationStatusSent,
		NotificationStatusFailed,
	}
}

// GetValidNotificationEvents returns all valid notification events
func GetValidNotificationEvents() []NotificationEvent {
	return []NotificationEvent{
		NotificationEventContainerUpdateSuccess,
		NotificationEventContainerUpdateFailed,
		NotificationEventContainerUpdateStarted,
		NotificationEventContainerStopped,
		NotificationEventContainerStarted,
		NotificationEventContainerError,
		NotificationEventSystemAlert,
		NotificationEventImageCheckFailed,
		NotificationEventNewImageAvailable,
	}
}

// GetDefaultNotificationTemplates returns default notification templates
func GetDefaultNotificationTemplates() []NotificationTemplate {
	return []NotificationTemplate{
		{
			Name:            "container_update_success",
			Type:            NotificationTypeEmail,
			TemplateSubject: "Container {{.ContainerName}} Updated Successfully",
			TemplateBody:    "Container {{.ContainerName}} has been successfully updated from {{.OldImage}} to {{.NewImage}}. Update completed at {{.CompletedAt}}.",
			Variables:       `["ContainerName", "OldImage", "NewImage", "CompletedAt"]`,
			IsActive:        true,
		},
		{
			Name:            "container_update_failed",
			Type:            NotificationTypeEmail,
			TemplateSubject: "Container {{.ContainerName}} Update Failed",
			TemplateBody:    "Container {{.ContainerName}} update failed. Error: {{.ErrorMessage}}. Please check the logs for more details.",
			Variables:       `["ContainerName", "ErrorMessage"]`,
			IsActive:        true,
		},
		{
			Name:            "system_alert",
			Type:            NotificationTypeEmail,
			TemplateSubject: "Docker Auto Update System Alert",
			TemplateBody:    "System alert: {{.Message}}. Time: {{.Timestamp}}",
			Variables:       `["Message", "Timestamp"]`,
			IsActive:        true,
		},
		{
			Name:            "new_image_available",
			Type:            NotificationTypeEmail,
			TemplateSubject: "New Image Available for {{.ContainerName}}",
			TemplateBody:    "A new version {{.NewVersion}} is available for container {{.ContainerName}} (current: {{.CurrentVersion}}). Consider updating the container.",
			Variables:       `["ContainerName", "NewVersion", "CurrentVersion"]`,
			IsActive:        true,
		},
	}
}

// BeforeCreate hook for NotificationTemplate model
func (nt *NotificationTemplate) BeforeCreate(tx *gorm.DB) error {
	if nt.Type == "" {
		nt.Type = NotificationTypeEmail
	}
	return nil
}

// BeforeCreate hook for NotificationLog model
func (nl *NotificationLog) BeforeCreate(tx *gorm.DB) error {
	if nl.Status == "" {
		nl.Status = NotificationStatusPending
	}
	if nl.Type == "" {
		nl.Type = NotificationTypeEmail
	}
	return nil
}