package service

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/pkg/webhook"

	"github.com/sirupsen/logrus"
)

// WebhookService handles webhook notifications using the enhanced webhook package
type WebhookService struct {
	webhookService *webhook.Service
	config         *config.WebhookConfig
	logger         *logrus.Logger
}

// NewWebhookService creates a new webhook service
func NewWebhookService(config *config.WebhookConfig, logger *logrus.Logger) (*WebhookService, error) {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Create the enhanced webhook service
	webhookService, err := webhook.NewService(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook service: %w", err)
	}

	return &WebhookService{
		webhookService: webhookService,
		config:         config,
		logger:         logger,
	}, nil
}

// Start starts the webhook service
func (ws *WebhookService) Start(ctx context.Context) error {
	return ws.webhookService.Start(ctx)
}

// Stop stops the webhook service
func (ws *WebhookService) Stop() error {
	return ws.webhookService.Stop()
}

// SendNotificationWebhook sends a notification via webhook
func (ws *WebhookService) SendNotificationWebhook(notification *model.Notification) error {
	if !ws.webhookService.IsEnabled() {
		ws.logger.Debug("Webhook service disabled, skipping notification")
		return nil
	}

	// Determine event type based on notification
	event := ws.getEventForNotification(notification)

	// Prepare webhook data
	data := map[string]interface{}{
		"title":       notification.Title,
		"message":     notification.Message,
		"type":        string(notification.Type),
		"priority":    string(notification.Priority),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	// Add container-specific data if available from Data field
	if notification.Data != nil {
		if containerID, ok := notification.Data["container_id"]; ok {
			data["container_id"] = containerID
		}
		if containerName, ok := notification.Data["container_name"]; ok {
			data["container_name"] = containerName
		}
	}

	// Add additional data if available
	if notification.Data != nil {
		for key, value := range notification.Data {
			data[key] = value
		}
	}

	// Determine priority based on notification priority
	priority := ws.getPriorityForLevel(notification.Priority)

	// Queue the webhook
	err := ws.webhookService.QueueWebhook(event, data, priority)
	if err != nil {
		ws.logger.WithError(err).WithFields(logrus.Fields{
			"notification_title": notification.Title,
			"event":             event,
			"type":              string(notification.Type),
		}).Error("Failed to queue notification webhook")
		return fmt.Errorf("failed to queue notification webhook: %w", err)
	}

	ws.logger.WithFields(logrus.Fields{
		"notification_title": notification.Title,
		"event":             event,
		"type":              string(notification.Type),
		"priority":          string(priority),
	}).Info("Notification webhook queued successfully")

	return nil
}

// SendContainerUpdateWebhook sends a container update webhook
func (ws *WebhookService) SendContainerUpdateWebhook(containerID, containerName, oldImage, newImage, status string) error {
	if !ws.webhookService.IsEnabled() {
		ws.logger.Debug("Webhook service disabled, skipping container update")
		return nil
	}

	additionalData := map[string]interface{}{
		"old_image": oldImage,
		"new_image": newImage,
		"status":    status,
	}

	return ws.webhookService.SendContainerEvent(
		webhook.EventContainerUpdate,
		containerID,
		containerName,
		newImage,
		additionalData,
	)
}

// SendContainerFailureWebhook sends a container failure webhook
func (ws *WebhookService) SendContainerFailureWebhook(containerID, containerName, image, errorMessage string) error {
	if !ws.webhookService.IsEnabled() {
		ws.logger.Debug("Webhook service disabled, skipping container failure")
		return nil
	}

	additionalData := map[string]interface{}{
		"error_message": errorMessage,
		"status":        "failed",
	}

	return ws.webhookService.SendContainerEvent(
		webhook.EventContainerFailure,
		containerID,
		containerName,
		image,
		additionalData,
	)
}

// SendSystemAlertWebhook sends a system alert webhook
func (ws *WebhookService) SendSystemAlertWebhook(alertType, message, severity string, additionalData map[string]interface{}) error {
	if !ws.webhookService.IsEnabled() {
		ws.logger.Debug("Webhook service disabled, skipping system alert")
		return nil
	}

	return ws.webhookService.SendSystemAlert(alertType, message, severity, additionalData)
}

// SendDirectWebhook sends a webhook immediately with custom data
func (ws *WebhookService) SendDirectWebhook(ctx context.Context, event webhook.WebhookEvent, data map[string]interface{}) error {
	if !ws.webhookService.IsEnabled() {
		ws.logger.Debug("Webhook service disabled, skipping direct webhook")
		return nil
	}

	return ws.webhookService.SendWebhook(ctx, event, data)
}

// IsEnabled returns whether webhook service is enabled
func (ws *WebhookService) IsEnabled() bool {
	return ws.webhookService.IsEnabled()
}

// GetMetrics returns webhook service metrics
func (ws *WebhookService) GetMetrics() *webhook.WebhookMetrics {
	return ws.webhookService.GetMetrics()
}

// GetQueueStats returns webhook queue statistics
func (ws *WebhookService) GetQueueStats() *webhook.WebhookQueueStats {
	return ws.webhookService.GetQueueStats()
}

// TestConnection tests the webhook endpoint
func (ws *WebhookService) TestConnection(ctx context.Context) error {
	return ws.webhookService.TestConnection(ctx)
}

// getEventForNotification determines which webhook event to use for a notification
func (ws *WebhookService) getEventForNotification(notification *model.Notification) webhook.WebhookEvent {
	switch notification.Type {
	case model.NotificationTypeContainerUpdate:
		return webhook.EventContainerUpdate
	case model.NotificationTypeImageUpdate:
		return webhook.EventContainerFailure
	case model.NotificationTypeSystemMaintenance:
		return webhook.EventSystemAlert
	case model.NotificationTypeSecurityUpdate:
		return webhook.EventSecurityAlert
	default:
		return webhook.EventSystemAlert
	}
}

// getPriorityForLevel determines webhook priority based on notification priority
func (ws *WebhookService) getPriorityForLevel(priority model.NotificationPriority) webhook.Priority {
	switch priority {
	case model.NotificationPriorityCritical:
		return webhook.PriorityUrgent
	case model.NotificationPriorityHigh:
		return webhook.PriorityHigh
	case model.NotificationPriorityNormal:
		return webhook.PriorityNormal
	case model.NotificationPriorityLow:
		return webhook.PriorityLow
	default:
		return webhook.PriorityNormal
	}
}