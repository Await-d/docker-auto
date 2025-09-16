package service

import (
	"docker-auto/internal/model"
	"fmt"
)

// EmailService handles email notifications
type EmailService struct {
	enabled bool
}

// NewEmailService creates a new email service
func NewEmailService(enabled bool) *EmailService {
	return &EmailService{
		enabled: enabled,
	}
}

// SendNotificationEmail sends a notification via email
func (es *EmailService) SendNotificationEmail(email string, notification *model.Notification) error {
	if !es.enabled {
		return nil // Email service disabled
	}

	// TODO: Implement actual email sending
	// This is a placeholder implementation
	fmt.Printf("Sending email to %s: %s - %s\n", email, notification.Title, notification.Message)

	return nil
}

// IsEnabled returns whether email service is enabled
func (es *EmailService) IsEnabled() bool {
	return es.enabled
}