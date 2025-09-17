package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/pkg/events"

	"github.com/sirupsen/logrus"
)

// RealtimeService integrates real-time features with existing services
type RealtimeService struct {
	publisher           events.Publisher
	notificationService NotificationServiceInterface
	logger              *logrus.Logger
	subscribers         map[string]*events.Subscription
	mu                  sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
}

// RealtimeServiceInterface defines the realtime service interface
type RealtimeServiceInterface interface {
	Start() error
	Stop() error
	PublishContainerEvent(eventType events.EventType, containerID, containerName string, data map[string]interface{}) error
	PublishImageEvent(eventType events.EventType, imageName string, data map[string]interface{}) error
	PublishSystemEvent(eventType events.EventType, message string, data map[string]interface{}) error
	PublishTaskEvent(eventType events.EventType, taskID int64, taskName string, data map[string]interface{}) error
	PublishUserEvent(eventType events.EventType, userID int64, data map[string]interface{}) error
	NotifyContainerUpdate(containerID int64, containerName, oldImage, newImage string, success bool, errorMsg string)
	NotifyImageUpdateAvailable(imageName, currentVersion, newVersion string, containerIDs []int64)
	NotifySystemAlert(severity events.EventSeverity, title, message string, data map[string]interface{})
	NotifyTaskCompletion(taskID int64, taskName string, success bool, duration time.Duration, errorMsg string)
}

// NewRealtimeService creates a new realtime service
func NewRealtimeService(
	publisher events.Publisher,
	notificationService NotificationServiceInterface,
	logger *logrus.Logger,
) *RealtimeService {
	if logger == nil {
		logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RealtimeService{
		publisher:           publisher,
		notificationService: notificationService,
		logger:              logger,
		subscribers:         make(map[string]*events.Subscription),
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start initializes the realtime service and starts listening for events
func (rs *RealtimeService) Start() error {
	rs.logger.Info("Starting realtime service")

	// Subscribe to all events for notification processing
	rs.subscribeToEvents()

	rs.logger.Info("Realtime service started successfully")
	return nil
}

// Stop shuts down the realtime service
func (rs *RealtimeService) Stop() error {
	rs.logger.Info("Stopping realtime service")

	rs.cancel()

	// Unsubscribe from all events
	rs.mu.Lock()
	for _, subscription := range rs.subscribers {
		rs.publisher.Unsubscribe(subscription.ID)
	}
	rs.subscribers = make(map[string]*events.Subscription)
	rs.mu.Unlock()

	rs.logger.Info("Realtime service stopped")
	return nil
}

// PublishContainerEvent publishes a container-related event
func (rs *RealtimeService) PublishContainerEvent(eventType events.EventType, containerID, containerName string, data map[string]interface{}) error {
	event := events.NewEvent(
		eventType,
		rs.getEventSeverity(eventType),
		"container-service",
		rs.getEventTitle(eventType, containerName),
		rs.getEventMessage(eventType, containerName, data),
	).WithResource("container", containerID).
		WithData("container_id", containerID).
		WithData("container_name", containerName)

	// Add additional data
	for k, v := range data {
		event.WithData(k, v)
	}

	rs.publisher.PublishAsync(event)

	rs.logger.WithFields(logrus.Fields{
		"event_type":     eventType,
		"container_id":   containerID,
		"container_name": containerName,
	}).Debug("Container event published")

	return nil
}

// PublishImageEvent publishes an image-related event
func (rs *RealtimeService) PublishImageEvent(eventType events.EventType, imageName string, data map[string]interface{}) error {
	event := events.NewEvent(
		eventType,
		rs.getEventSeverity(eventType),
		"image-service",
		rs.getEventTitle(eventType, imageName),
		rs.getEventMessage(eventType, imageName, data),
	).WithResource("image", imageName).
		WithData("image_name", imageName)

	// Add additional data
	for k, v := range data {
		event.WithData(k, v)
	}

	rs.publisher.PublishAsync(event)

	rs.logger.WithFields(logrus.Fields{
		"event_type":  eventType,
		"image_name": imageName,
	}).Debug("Image event published")

	return nil
}

// PublishSystemEvent publishes a system-related event
func (rs *RealtimeService) PublishSystemEvent(eventType events.EventType, message string, data map[string]interface{}) error {
	event := events.NewEvent(
		eventType,
		rs.getEventSeverity(eventType),
		"system-service",
		rs.getEventTitle(eventType, ""),
		message,
	)

	// Add additional data
	for k, v := range data {
		event.WithData(k, v)
	}

	rs.publisher.PublishAsync(event)

	rs.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"message":   message,
	}).Debug("System event published")

	return nil
}

// PublishTaskEvent publishes a task-related event
func (rs *RealtimeService) PublishTaskEvent(eventType events.EventType, taskID int64, taskName string, data map[string]interface{}) error {
	event := events.NewEvent(
		eventType,
		rs.getEventSeverity(eventType),
		"task-service",
		rs.getEventTitle(eventType, taskName),
		rs.getEventMessage(eventType, taskName, data),
	).WithResource("task", fmt.Sprintf("%d", taskID)).
		WithData("task_id", taskID).
		WithData("task_name", taskName)

	// Add additional data
	for k, v := range data {
		event.WithData(k, v)
	}

	rs.publisher.PublishAsync(event)

	rs.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"task_id":   taskID,
		"task_name": taskName,
	}).Debug("Task event published")

	return nil
}

// PublishUserEvent publishes a user-related event
func (rs *RealtimeService) PublishUserEvent(eventType events.EventType, userID int64, data map[string]interface{}) error {
	event := events.NewEvent(
		eventType,
		rs.getEventSeverity(eventType),
		"user-service",
		rs.getEventTitle(eventType, ""),
		rs.getEventMessage(eventType, "", data),
	).WithResource("user", fmt.Sprintf("%d", userID)).
		WithUserID(fmt.Sprintf("%d", userID))

	// Add additional data
	for k, v := range data {
		event.WithData(k, v)
	}

	rs.publisher.PublishAsync(event)

	rs.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"user_id":   userID,
	}).Debug("User event published")

	return nil
}

// NotifyContainerUpdate creates notifications for container updates
func (rs *RealtimeService) NotifyContainerUpdate(containerID int64, containerName, oldImage, newImage string, success bool, errorMsg string) {
	var eventType events.EventType
	var notificationType NotificationType
	var templateID string

	if success {
		eventType = events.EventContainerUpdated
		notificationType = NotificationTypeSuccess
		templateID = "container_update_success"
	} else {
		eventType = events.EventContainerError
		notificationType = NotificationTypeError
		templateID = "container_update_failed"
	}

	_ = templateID // Use templateID to avoid unused variable error

	// Publish event
	data := map[string]interface{}{
		"container_id":   containerID,
		"container_name": containerName,
		"old_image":      oldImage,
		"new_image":      newImage,
		"success":        success,
	}

	if errorMsg != "" {
		data["error"] = errorMsg
	}

	rs.PublishContainerEvent(eventType, fmt.Sprintf("%d", containerID), containerName, data)

	// Create notification using template
	if rs.notificationService != nil {
		// For container updates, we might want to notify the container owner or all admins
		// This is a simplified implementation - in production, you'd determine the target users
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			templateData := map[string]interface{}{
				"container_name": containerName,
				"old_image":      oldImage,
				"new_image":      newImage,
				"completed_at":   time.Now().Format(time.RFC3339),
			}

			if errorMsg != "" {
				templateData["error"] = errorMsg
			}

			// Create broadcast notification for container updates
			title := fmt.Sprintf("Container %s Update", containerName)
			message := fmt.Sprintf("Container %s update completed", containerName)
			if !success {
				message = fmt.Sprintf("Container %s update failed: %s", containerName, errorMsg)
			}

			if err := rs.notificationService.BroadcastNotification(
				ctx,
				notificationType,
				title,
				message,
				templateData,
			); err != nil {
				rs.logger.WithError(err).Error("Failed to create container update notification")
			}
		}()
	}
}

// NotifyImageUpdateAvailable creates notifications for available image updates
func (rs *RealtimeService) NotifyImageUpdateAvailable(imageName, currentVersion, newVersion string, containerIDs []int64) {
	// Publish event
	data := map[string]interface{}{
		"image_name":      imageName,
		"current_version": currentVersion,
		"new_version":     newVersion,
		"container_ids":   containerIDs,
		"container_count": len(containerIDs),
	}

	rs.PublishImageEvent(events.EventImageUpdateAvailable, imageName, data)

	// Create notification
	if rs.notificationService != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			title := fmt.Sprintf("New Version Available: %s", imageName)
			message := fmt.Sprintf("A new version %s is available for image %s (current: %s). %d container(s) can be updated.",
				newVersion, imageName, currentVersion, len(containerIDs))

			if err := rs.notificationService.BroadcastNotification(
				ctx,
				NotificationTypeInfo,
				title,
				message,
				data,
			); err != nil {
				rs.logger.WithError(err).Error("Failed to create image update notification")
			}
		}()
	}
}

// NotifySystemAlert creates notifications for system alerts
func (rs *RealtimeService) NotifySystemAlert(severity events.EventSeverity, title, message string, data map[string]interface{}) {
	// Publish event
	rs.PublishSystemEvent(events.EventSystemHealthChanged, message, data)

	// Create notification
	if rs.notificationService != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var notifType NotificationType
			switch severity {
			case events.SeverityError:
				notifType = NotificationTypeError
			case events.SeverityWarning:
				notifType = NotificationTypeWarning
			case events.SeveritySuccess:
				notifType = NotificationTypeSuccess
			default:
				notifType = NotificationTypeInfo
			}

			if err := rs.notificationService.BroadcastNotification(
				ctx,
				notifType,
				title,
				message,
				data,
			); err != nil {
				rs.logger.WithError(err).Error("Failed to create system alert notification")
			}
		}()
	}
}

// NotifyTaskCompletion creates notifications for task completion
func (rs *RealtimeService) NotifyTaskCompletion(taskID int64, taskName string, success bool, duration time.Duration, errorMsg string) {
	var eventType events.EventType
	var notificationType NotificationType

	if success {
		eventType = events.EventTaskCompleted
		notificationType = NotificationTypeSuccess
	} else {
		eventType = events.EventTaskFailed
		notificationType = NotificationTypeError
	}

	// Publish event
	data := map[string]interface{}{
		"task_id":   taskID,
		"task_name": taskName,
		"success":   success,
		"duration":  duration.String(),
	}

	if errorMsg != "" {
		data["error"] = errorMsg
	}

	rs.PublishTaskEvent(eventType, taskID, taskName, data)

	// Create notification
	if rs.notificationService != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			title := fmt.Sprintf("Task %s Completed", taskName)
			message := fmt.Sprintf("Task %s completed successfully in %s", taskName, duration.String())
			if !success {
				title = fmt.Sprintf("Task %s Failed", taskName)
				message = fmt.Sprintf("Task %s failed after %s: %s", taskName, duration.String(), errorMsg)
			}

			if err := rs.notificationService.BroadcastNotification(
				ctx,
				notificationType,
				title,
				message,
				data,
			); err != nil {
				rs.logger.WithError(err).Error("Failed to create task completion notification")
			}
		}()
	}
}

// subscribeToEvents subscribes to relevant events for processing
func (rs *RealtimeService) subscribeToEvents() {
	// Subscribe to container events
	containerFilter := events.EventFilter{
		Types: []events.EventType{
			events.EventContainerStarted,
			events.EventContainerStopped,
			events.EventContainerUpdated,
			events.EventContainerError,
		},
	}
	containerSub := rs.publisher.Subscribe(containerFilter)

	// Subscribe to image events
	imageFilter := events.EventFilter{
		Types: []events.EventType{
			events.EventImageUpdateAvailable,
			events.EventImageUpdateCompleted,
			events.EventImageUpdateFailed,
		},
	}
	imageSub := rs.publisher.Subscribe(imageFilter)

	// Subscribe to system events
	systemFilter := events.EventFilter{
		Types: []events.EventType{
			events.EventSystemHealthChanged,
			events.EventSystemResourceAlert,
		},
	}
	systemSub := rs.publisher.Subscribe(systemFilter)

	// Store subscriptions
	rs.mu.Lock()
	rs.subscribers["container"] = containerSub
	rs.subscribers["image"] = imageSub
	rs.subscribers["system"] = systemSub
	rs.mu.Unlock()

	// Start event processing goroutines
	go rs.processContainerEvents(containerSub)
	go rs.processImageEvents(imageSub)
	go rs.processSystemEvents(systemSub)
}

// processContainerEvents processes container-related events
func (rs *RealtimeService) processContainerEvents(subscription *events.Subscription) {
	for {
		select {
		case <-rs.ctx.Done():
			return
		case event := <-subscription.Channel:
			rs.handleContainerEvent(event)
		}
	}
}

// processImageEvents processes image-related events
func (rs *RealtimeService) processImageEvents(subscription *events.Subscription) {
	for {
		select {
		case <-rs.ctx.Done():
			return
		case event := <-subscription.Channel:
			rs.handleImageEvent(event)
		}
	}
}

// processSystemEvents processes system-related events
func (rs *RealtimeService) processSystemEvents(subscription *events.Subscription) {
	for {
		select {
		case <-rs.ctx.Done():
			return
		case event := <-subscription.Channel:
			rs.handleSystemEvent(event)
		}
	}
}

// handleContainerEvent handles individual container events
func (rs *RealtimeService) handleContainerEvent(event *events.Event) {
	rs.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
		"event_id":   event.ID,
		"source":     event.Source,
	}).Debug("Processing container event")

	// Additional processing logic can be added here
	// For example, updating statistics, triggering other services, etc.
}

// handleImageEvent handles individual image events
func (rs *RealtimeService) handleImageEvent(event *events.Event) {
	rs.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
		"event_id":   event.ID,
		"source":     event.Source,
	}).Debug("Processing image event")

	// Additional processing logic can be added here
}

// handleSystemEvent handles individual system events
func (rs *RealtimeService) handleSystemEvent(event *events.Event) {
	rs.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
		"event_id":   event.ID,
		"source":     event.Source,
	}).Debug("Processing system event")

	// Additional processing logic can be added here
}

// Helper methods

// getEventSeverity maps event types to severities
func (rs *RealtimeService) getEventSeverity(eventType events.EventType) events.EventSeverity {
	switch eventType {
	case events.EventContainerError, events.EventImageUpdateFailed, events.EventTaskFailed, events.EventSystemError:
		return events.SeverityError
	case events.EventSystemResourceAlert:
		return events.SeverityWarning
	case events.EventContainerUpdated, events.EventImageUpdateCompleted, events.EventTaskCompleted:
		return events.SeveritySuccess
	default:
		return events.SeverityInfo
	}
}

// getEventTitle generates appropriate titles for events
func (rs *RealtimeService) getEventTitle(eventType events.EventType, resourceName string) string {
	switch eventType {
	case events.EventContainerStarted:
		return fmt.Sprintf("Container Started: %s", resourceName)
	case events.EventContainerStopped:
		return fmt.Sprintf("Container Stopped: %s", resourceName)
	case events.EventContainerUpdated:
		return fmt.Sprintf("Container Updated: %s", resourceName)
	case events.EventContainerError:
		return fmt.Sprintf("Container Error: %s", resourceName)
	case events.EventImageUpdateAvailable:
		return fmt.Sprintf("Image Update Available: %s", resourceName)
	case events.EventImageUpdateCompleted:
		return fmt.Sprintf("Image Update Completed: %s", resourceName)
	case events.EventImageUpdateFailed:
		return fmt.Sprintf("Image Update Failed: %s", resourceName)
	case events.EventTaskStarted:
		return fmt.Sprintf("Task Started: %s", resourceName)
	case events.EventTaskCompleted:
		return fmt.Sprintf("Task Completed: %s", resourceName)
	case events.EventTaskFailed:
		return fmt.Sprintf("Task Failed: %s", resourceName)
	case events.EventSystemHealthChanged:
		return "System Health Alert"
	case events.EventSystemResourceAlert:
		return "System Resource Alert"
	default:
		return "System Event"
	}
}

// getEventMessage generates appropriate messages for events
func (rs *RealtimeService) getEventMessage(eventType events.EventType, resourceName string, data map[string]interface{}) string {
	switch eventType {
	case events.EventContainerStarted:
		return fmt.Sprintf("Container %s has been started successfully", resourceName)
	case events.EventContainerStopped:
		return fmt.Sprintf("Container %s has been stopped", resourceName)
	case events.EventContainerUpdated:
		if oldImage, ok := data["old_image"]; ok {
			if newImage, ok := data["new_image"]; ok {
				return fmt.Sprintf("Container %s has been updated from %s to %s", resourceName, oldImage, newImage)
			}
		}
		return fmt.Sprintf("Container %s has been updated", resourceName)
	case events.EventContainerError:
		if errorMsg, ok := data["error"]; ok {
			return fmt.Sprintf("Container %s encountered an error: %s", resourceName, errorMsg)
		}
		return fmt.Sprintf("Container %s encountered an error", resourceName)
	case events.EventImageUpdateAvailable:
		if newVersion, ok := data["new_version"]; ok {
			return fmt.Sprintf("A new version %s is available for image %s", newVersion, resourceName)
		}
		return fmt.Sprintf("A new version is available for image %s", resourceName)
	case events.EventTaskCompleted:
		if duration, ok := data["duration"]; ok {
			return fmt.Sprintf("Task %s completed successfully in %s", resourceName, duration)
		}
		return fmt.Sprintf("Task %s completed successfully", resourceName)
	case events.EventTaskFailed:
		if errorMsg, ok := data["error"]; ok {
			return fmt.Sprintf("Task %s failed: %s", resourceName, errorMsg)
		}
		return fmt.Sprintf("Task %s failed", resourceName)
	default:
		return fmt.Sprintf("Event for %s", resourceName)
	}
}