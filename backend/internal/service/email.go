package service

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/pkg/email"

	"github.com/sirupsen/logrus"
)

// EmailService handles email notifications using the enhanced email package
type EmailService struct {
	emailService *email.Service
	config       *config.EmailConfig
	logger       *logrus.Logger
}

// NewEmailService creates a new email service
func NewEmailService(config *config.EmailConfig, logger *logrus.Logger) (*EmailService, error) {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Create the enhanced email service
	emailService, err := email.NewService(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create email service: %w", err)
	}

	return &EmailService{
		emailService: emailService,
		config:       config,
		logger:       logger,
	}, nil
}

// Start starts the email service
func (es *EmailService) Start(ctx context.Context) error {
	return es.emailService.Start(ctx)
}

// Stop stops the email service
func (es *EmailService) Stop() error {
	return es.emailService.Stop()
}

// SendNotificationEmail sends a notification via email using templates
func (es *EmailService) SendNotificationEmail(emailAddress string, notification *model.Notification) error {
	if !es.emailService.IsEnabled() {
		es.logger.Debug("Email service disabled, skipping notification")
		return nil
	}

	// Determine template based on notification type
	templateName := es.getTemplateForNotification(notification)

	// Prepare template data
	templateData := &email.TemplateData{
		RecipientEmail:      emailAddress,
		RecipientName:       es.extractNameFromEmail(emailAddress),
		SenderEmail:         es.config.From,
		SenderName:          "Docker Auto",
		AppName:             "Docker Auto",
		AppURL:              "http://localhost:8080", // TODO: Get from config
		Timestamp:           time.Now().Format("2006-01-02 15:04:05"),
		NotificationTitle:   notification.Title,
		NotificationMessage: notification.Message,
		NotificationType:    string(notification.Type),
		Severity:            es.getSeverityString(notification.Priority),
	}

	// Add container-specific data if available from Data field
	if notification.Data != nil {
		if containerID, ok := notification.Data["container_id"].(string); ok {
			templateData.ContainerID = containerID
		}
		if containerName, ok := notification.Data["container_name"].(string); ok {
			templateData.ContainerName = containerName
		}
	}

	// Queue the templated email
	err := es.emailService.QueueTemplatedEmail(templateName, templateData, es.config.DefaultLocale)
	if err != nil {
		es.logger.WithError(err).WithFields(logrus.Fields{
			"email":    emailAddress,
			"template": templateName,
			"title":    notification.Title,
		}).Error("Failed to queue notification email")
		return fmt.Errorf("failed to queue notification email: %w", err)
	}

	es.logger.WithFields(logrus.Fields{
		"email":    emailAddress,
		"template": templateName,
		"title":    notification.Title,
		"type":     notification.Type,
	}).Info("Notification email queued successfully")

	return nil
}

// SendDirectEmail sends a direct email without templates
func (es *EmailService) SendDirectEmail(ctx context.Context, to []string, subject, textBody, htmlBody string) error {
	if !es.emailService.IsEnabled() {
		es.logger.Debug("Email service disabled, skipping direct email")
		return nil
	}

	message := &email.Message{
		To:       to,
		From:     es.config.From,
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}

	return es.emailService.SendEmail(ctx, message)
}

// SendTemplatedEmail sends a templated email
func (es *EmailService) SendTemplatedEmail(ctx context.Context, templateName string, data *email.TemplateData) error {
	if !es.emailService.IsEnabled() {
		es.logger.Debug("Email service disabled, skipping templated email")
		return nil
	}

	return es.emailService.SendTemplatedEmail(ctx, templateName, data, es.config.DefaultLocale)
}

// IsEnabled returns whether email service is enabled
func (es *EmailService) IsEnabled() bool {
	return es.emailService.IsEnabled()
}

// GetMetrics returns email service metrics
func (es *EmailService) GetMetrics() *email.ServiceMetrics {
	return es.emailService.GetMetrics()
}

// GetQueueSize returns the current email queue size
func (es *EmailService) GetQueueSize() int {
	return es.emailService.GetQueueSize()
}

// TestConnection tests the email service connection
func (es *EmailService) TestConnection(ctx context.Context) error {
	return es.emailService.TestConnection(ctx)
}

// getTemplateForNotification determines which template to use for a notification
func (es *EmailService) getTemplateForNotification(notification *model.Notification) string {
	switch notification.Type {
	case model.NotificationTypeContainerUpdate:
		return "container_update"
	case model.NotificationTypeImageUpdate:
		return "container_failure"
	case model.NotificationTypeSystemMaintenance:
		return "system_notification"
	case model.NotificationTypeSecurityUpdate:
		return "system_notification"
	default:
		return "system_notification"
	}
}

// extractNameFromEmail extracts a name from an email address
func (es *EmailService) extractNameFromEmail(email string) string {
	// Simple extraction - take the part before @ and clean it up
	if atIndex := len(email); atIndex > 0 {
		for i, char := range email {
			if char == '@' {
				atIndex = i
				break
			}
		}
		if atIndex > 0 {
			name := email[:atIndex]
			// Replace common separators with spaces and title case
			for _, sep := range []string{".", "_", "-"} {
				if len(name) > 0 {
					name = replaceAll(name, sep, " ")
				}
			}
			return name
		}
	}
	return "User"
}

// getSeverityString converts notification level to string
func (es *EmailService) getSeverityString(priority model.NotificationPriority) string {
	switch priority {
	case model.NotificationPriorityLow:
		return "Low"
	case model.NotificationPriorityNormal:
		return "Normal"
	case model.NotificationPriorityHigh:
		return "High"
	case model.NotificationPriorityCritical:
		return "Critical"
	default:
		return "Info"
	}
}

// replaceAll is a simple string replacement function
func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}