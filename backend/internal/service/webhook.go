package service

import (
	"docker-auto/internal/model"
	"fmt"
)

// WebhookService handles webhook notifications
type WebhookService struct {
	enabled bool
	url     string
}

// NewWebhookService creates a new webhook service
func NewWebhookService(enabled bool, url string) *WebhookService {
	return &WebhookService{
		enabled: enabled,
		url:     url,
	}
}

// SendNotificationWebhook sends a notification via webhook
func (ws *WebhookService) SendNotificationWebhook(notification *model.Notification) error {
	if !ws.enabled || ws.url == "" {
		return nil // Webhook service disabled or no URL configured
	}

	// TODO: Implement actual webhook sending
	// This is a placeholder implementation
	fmt.Printf("Sending webhook to %s: %s - %s\n", ws.url, notification.Title, notification.Message)

	return nil
}

// IsEnabled returns whether webhook service is enabled
func (ws *WebhookService) IsEnabled() bool {
	return ws.enabled
}