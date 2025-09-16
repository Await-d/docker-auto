package service

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/events"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
)

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID       string           `json:"id"`
	Type     NotificationType `json:"type"`
	Title    string           `json:"title"`
	Message  string           `json:"message"`
	Template *template.Template
}

// NotificationService handles notifications
type NotificationService struct {
	db                *gorm.DB
	logger            *logrus.Logger
	publisher         events.Publisher
	templates         map[string]*NotificationTemplate
	templatesMu       sync.RWMutex
	userRepo          repository.UserRepository
	notificationRepo  repository.NotificationRepository
	emailService      *EmailService
	webhookService    *WebhookService
}

// NotificationServiceInterface defines the notification service interface
type NotificationServiceInterface interface {
	CreateNotification(ctx context.Context, userID *int64, notificationType NotificationType, title, message string, data map[string]interface{}) (*model.Notification, error)
	CreateNotificationFromTemplate(ctx context.Context, userID *int64, templateID string, data map[string]interface{}) (*model.Notification, error)
	GetNotifications(ctx context.Context, userID int64, limit, offset int) ([]*model.Notification, error)
	GetUnreadCount(ctx context.Context, userID int64) (int64, error)
	MarkAsRead(ctx context.Context, notificationID int64, userID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error
	DeleteNotification(ctx context.Context, notificationID int64, userID int64) error
	BroadcastNotification(ctx context.Context, notificationType NotificationType, title, message string, data map[string]interface{}) error
	RegisterTemplate(templateID string, notificationType NotificationType, title, message string) error
	GetNotificationsByType(ctx context.Context, userID int64, notificationType NotificationType, limit, offset int) ([]*model.Notification, error)
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	db *gorm.DB,
	logger *logrus.Logger,
	publisher events.Publisher,
	userRepo repository.UserRepository,
	notificationRepo repository.NotificationRepository,
	emailService *EmailService,
	webhookService *WebhookService,
) *NotificationService {
	if logger == nil {
		logger = logrus.New()
	}

	service := &NotificationService{
		db:               db,
		logger:           logger,
		publisher:        publisher,
		templates:        make(map[string]*NotificationTemplate),
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		emailService:     emailService,
		webhookService:   webhookService,
	}

	// Register default templates
	service.registerDefaultTemplates()

	return service
}

// CreateNotification creates a new notification
func (ns *NotificationService) CreateNotification(
	ctx context.Context,
	userID *int64,
	notificationType NotificationType,
	title, message string,
	data map[string]interface{},
) (*model.Notification, error) {
	notification := &model.Notification{
		UserID:    userID,
		Type:      string(notificationType),
		Title:     title,
		Message:   message,
		Data:      data,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	// Save to database
	if err := ns.notificationRepo.Create(ctx, notification); err != nil {
		ns.logger.WithError(err).Error("Failed to create notification")
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Publish event
	event := events.NewEvent(
		events.EventNotificationCreated,
		ns.mapTypeToSeverity(notificationType),
		"notification-service",
		title,
		message,
	).WithData("notification_id", notification.ID).
		WithData("notification_type", notificationType)

	if userID != nil {
		userIDStr := fmt.Sprintf("%d", *userID)
		event = event.WithUserID(userIDStr)
	}

	ns.publisher.PublishAsync(event)

	ns.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":        userID,
		"type":           notificationType,
		"title":          title,
	}).Info("Notification created")

	// Send external notifications if configured
	go ns.sendExternalNotifications(notification)

	return notification, nil
}

// CreateNotificationFromTemplate creates a notification using a template
func (ns *NotificationService) CreateNotificationFromTemplate(
	ctx context.Context,
	userID *int64,
	templateID string,
	data map[string]interface{},
) (*model.Notification, error) {
	ns.templatesMu.RLock()
	tmpl, exists := ns.templates[templateID]
	ns.templatesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Execute template
	title, message, err := ns.executeTemplate(tmpl, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return ns.CreateNotification(ctx, userID, tmpl.Type, title, message, data)
}

// GetNotifications retrieves notifications for a user
func (ns *NotificationService) GetNotifications(ctx context.Context, userID int64, limit, offset int) ([]*model.Notification, error) {
	notifications, err := ns.notificationRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to get notifications")
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	return notifications, nil
}

// GetUnreadCount returns the count of unread notifications for a user
func (ns *NotificationService) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	count, err := ns.notificationRepo.GetUnreadCount(ctx, userID)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to get unread count")
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a notification as read
func (ns *NotificationService) MarkAsRead(ctx context.Context, notificationID int64, userID int64) error {
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	// Check if user owns the notification
	if notification.UserID == nil || *notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}

	if notification.IsRead {
		return nil // Already read
	}

	notification.IsRead = true
	notification.ReadAt = &time.Time{}
	*notification.ReadAt = time.Now()

	if err := ns.notificationRepo.Update(ctx, notification); err != nil {
		ns.logger.WithError(err).Error("Failed to mark notification as read")
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	// Publish event
	event := events.NewEvent(
		events.EventNotificationRead,
		events.SeverityInfo,
		"notification-service",
		"Notification Read",
		"Notification marked as read",
	).WithData("notification_id", notificationID).
		WithUserID(fmt.Sprintf("%d", userID))

	ns.publisher.PublishAsync(event)

	ns.logger.WithFields(logrus.Fields{
		"notification_id": notificationID,
		"user_id":        userID,
	}).Debug("Notification marked as read")

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (ns *NotificationService) MarkAllAsRead(ctx context.Context, userID int64) error {
	count, err := ns.notificationRepo.MarkAllAsRead(ctx, userID)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to mark all notifications as read")
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	if count > 0 {
		ns.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"count":   count,
		}).Info("Marked all notifications as read")
	}

	return nil
}

// DeleteNotification deletes a notification
func (ns *NotificationService) DeleteNotification(ctx context.Context, notificationID int64, userID int64) error {
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	// Check if user owns the notification
	if notification.UserID == nil || *notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}

	if err := ns.notificationRepo.Delete(ctx, notificationID); err != nil {
		ns.logger.WithError(err).Error("Failed to delete notification")
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	// Publish event
	event := events.NewEvent(
		events.EventNotificationDeleted,
		events.SeverityInfo,
		"notification-service",
		"Notification Deleted",
		"Notification was deleted",
	).WithData("notification_id", notificationID).
		WithUserID(fmt.Sprintf("%d", userID))

	ns.publisher.PublishAsync(event)

	ns.logger.WithFields(logrus.Fields{
		"notification_id": notificationID,
		"user_id":        userID,
	}).Debug("Notification deleted")

	return nil
}

// BroadcastNotification creates a notification for all users
func (ns *NotificationService) BroadcastNotification(
	ctx context.Context,
	notificationType NotificationType,
	title, message string,
	data map[string]interface{},
) error {
	// Get all active users
	users, err := ns.userRepo.GetActiveUsers(ctx)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to get active users for broadcast")
		return fmt.Errorf("failed to get active users: %w", err)
	}

	// Create notifications for each user
	for _, user := range users {
		_, err := ns.CreateNotification(ctx, &user.ID, notificationType, title, message, data)
		if err != nil {
			ns.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to create broadcast notification")
		}
	}

	ns.logger.WithFields(logrus.Fields{
		"type":       notificationType,
		"title":      title,
		"user_count": len(users),
	}).Info("Broadcast notification sent")

	return nil
}

// RegisterTemplate registers a notification template
func (ns *NotificationService) RegisterTemplate(templateID string, notificationType NotificationType, title, message string) error {
	titleTmpl, err := template.New(templateID + "_title").Parse(title)
	if err != nil {
		return fmt.Errorf("failed to parse title template: %w", err)
	}

	messageTmpl, err := template.New(templateID + "_message").Parse(message)
	if err != nil {
		return fmt.Errorf("failed to parse message template: %w", err)
	}

	// Combine templates
	combined := template.New(templateID)
	combined.AddParseTree("title", titleTmpl.Tree)
	combined.AddParseTree("message", messageTmpl.Tree)

	ns.templatesMu.Lock()
	ns.templates[templateID] = &NotificationTemplate{
		ID:       templateID,
		Type:     notificationType,
		Title:    title,
		Message:  message,
		Template: combined,
	}
	ns.templatesMu.Unlock()

	ns.logger.WithField("template_id", templateID).Debug("Notification template registered")

	return nil
}

// GetNotificationsByType retrieves notifications by type for a user
func (ns *NotificationService) GetNotificationsByType(ctx context.Context, userID int64, notificationType NotificationType, limit, offset int) ([]*model.Notification, error) {
	notifications, err := ns.notificationRepo.GetByUserIDAndType(ctx, userID, string(notificationType), limit, offset)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to get notifications by type")
		return nil, fmt.Errorf("failed to get notifications by type: %w", err)
	}

	return notifications, nil
}

// executeTemplate executes a notification template
func (ns *NotificationService) executeTemplate(tmpl *NotificationTemplate, data map[string]interface{}) (string, string, error) {
	var titleBuf, messageBuf strings.Builder

	// Execute title template
	if err := tmpl.Template.ExecuteTemplate(&titleBuf, "title", data); err != nil {
		return "", "", fmt.Errorf("failed to execute title template: %w", err)
	}

	// Execute message template
	if err := tmpl.Template.ExecuteTemplate(&messageBuf, "message", data); err != nil {
		return "", "", fmt.Errorf("failed to execute message template: %w", err)
	}

	return titleBuf.String(), messageBuf.String(), nil
}

// mapTypeToSeverity maps notification type to event severity
func (ns *NotificationService) mapTypeToSeverity(notificationType NotificationType) events.EventSeverity {
	switch notificationType {
	case NotificationTypeError:
		return events.SeverityError
	case NotificationTypeWarning:
		return events.SeverityWarning
	case NotificationTypeSuccess:
		return events.SeveritySuccess
	case NotificationTypeInfo:
		return events.SeverityInfo
	default:
		return events.SeverityInfo
	}
}

// sendExternalNotifications sends notifications via email/webhook if configured
func (ns *NotificationService) sendExternalNotifications(notification *model.Notification) {
	if notification.UserID == nil {
		return
	}

	// Get user for external notification settings
	ctx := context.Background()
	user, err := ns.userRepo.GetByID(ctx, *notification.UserID)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to get user for external notifications")
		return
	}

	// Send email notification if enabled
	if ns.emailService != nil && user.EmailNotifications {
		if err := ns.emailService.SendNotificationEmail(user.Email, notification); err != nil {
			ns.logger.WithError(err).Error("Failed to send email notification")
		}
	}

	// Send webhook notification if enabled
	if ns.webhookService != nil {
		if err := ns.webhookService.SendNotificationWebhook(notification); err != nil {
			ns.logger.WithError(err).Error("Failed to send webhook notification")
		}
	}
}

// registerDefaultTemplates registers default notification templates
func (ns *NotificationService) registerDefaultTemplates() {
	templates := map[string]struct {
		Type    NotificationType
		Title   string
		Message string
	}{
		"container_update_success": {
			Type:    NotificationTypeSuccess,
			Title:   "Container Updated Successfully",
			Message: "Container {{.container_name}} has been updated to version {{.new_version}}",
		},
		"container_update_failed": {
			Type:    NotificationTypeError,
			Title:   "Container Update Failed",
			Message: "Failed to update container {{.container_name}}: {{.error}}",
		},
		"image_update_available": {
			Type:    NotificationTypeInfo,
			Title:   "New Image Version Available",
			Message: "A new version {{.new_version}} is available for image {{.image_name}}",
		},
		"system_health_warning": {
			Type:    NotificationTypeWarning,
			Title:   "System Health Warning",
			Message: "System health check failed: {{.reason}}",
		},
		"user_login": {
			Type:    NotificationTypeInfo,
			Title:   "User Login",
			Message: "User {{.username}} logged in from {{.ip_address}}",
		},
		"task_completed": {
			Type:    NotificationTypeSuccess,
			Title:   "Task Completed",
			Message: "Task {{.task_name}} completed successfully",
		},
		"task_failed": {
			Type:    NotificationTypeError,
			Title:   "Task Failed",
			Message: "Task {{.task_name}} failed: {{.error}}",
		},
	}

	for templateID, tmpl := range templates {
		if err := ns.RegisterTemplate(templateID, tmpl.Type, tmpl.Title, tmpl.Message); err != nil {
			ns.logger.WithError(err).WithField("template_id", templateID).Error("Failed to register default template")
		}
	}
}

// GetNotificationStats returns notification statistics for a user
func (ns *NotificationService) GetNotificationStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total count
	total, err := ns.notificationRepo.GetTotalCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total"] = total

	// Get unread count
	unread, err := ns.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}
	stats["unread"] = unread

	// Get counts by type
	types := []NotificationType{NotificationTypeInfo, NotificationTypeWarning, NotificationTypeError, NotificationTypeSuccess}
	typeCounts := make(map[string]int64)
	for _, notifType := range types {
		count, err := ns.notificationRepo.GetCountByType(ctx, userID, string(notifType))
		if err != nil {
			ns.logger.WithError(err).Error("Failed to get count by type")
			continue
		}
		typeCounts[string(notifType)] = count
	}
	stats["by_type"] = typeCounts

	return stats, nil
}

// CleanupOldNotifications removes old notifications based on retention policy
func (ns *NotificationService) CleanupOldNotifications(ctx context.Context, retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	count, err := ns.notificationRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to cleanup old notifications")
		return fmt.Errorf("failed to cleanup old notifications: %w", err)
	}

	if count > 0 {
		ns.logger.WithFields(logrus.Fields{
			"count":           count,
			"retention_days":  retentionDays,
			"cutoff_date":     cutoffDate,
		}).Info("Cleaned up old notifications")
	}

	return nil
}