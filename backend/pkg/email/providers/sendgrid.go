package providers

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

// SendGridProvider implements EmailProvider for SendGrid
type SendGridProvider struct {
	config SendGridConfig
	client *sendgrid.Client
	logger *logrus.Logger
}

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey    string
	From      string
	FromName  string
	ReplyTo   string

	// Template settings
	TemplateID       string
	UnsubscribeGroup int

	// Tracking settings
	ClickTracking bool
	OpenTracking  bool
	Subscription  bool

	// Sandbox mode (for testing)
	SandboxMode bool
}

// NewSendGridProvider creates a new SendGrid email provider
func NewSendGridProvider(config SendGridConfig, logger *logrus.Logger) *SendGridProvider {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	client := sendgrid.NewSendClient(config.APIKey)

	return &SendGridProvider{
		config: config,
		client: client,
		logger: logger,
	}
}

// Send sends an email via SendGrid
func (p *SendGridProvider) Send(ctx context.Context, message *Message) error {
	// Create the email message
	m := mail.NewV3Mail()

	// Set from address
	from := mail.NewEmail(p.config.FromName, p.getFromAddress(message))
	m.SetFrom(from)

	// Set subject
	m.Subject = message.Subject

	// Add recipients
	personalization := mail.NewPersonalization()

	// To recipients
	for _, to := range message.To {
		personalization.AddTos(mail.NewEmail("", to))
	}

	// CC recipients
	for _, cc := range message.CC {
		personalization.AddCCs(mail.NewEmail("", cc))
	}

	// BCC recipients
	for _, bcc := range message.BCC {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}

	// Custom headers
	if len(message.Headers) > 0 {
		personalization.Headers = message.Headers
	}

	m.AddPersonalizations(personalization)

	// Set content
	if message.TextBody != "" {
		m.AddContent(mail.NewContent("text/plain", message.TextBody))
	}
	if message.HTMLBody != "" {
		m.AddContent(mail.NewContent("text/html", message.HTMLBody))
	}

	// Reply-To
	if message.ReplyTo != "" {
		m.ReplyTo = mail.NewEmail("", message.ReplyTo)
	} else if p.config.ReplyTo != "" {
		m.ReplyTo = mail.NewEmail("", p.config.ReplyTo)
	}

	// Attachments
	for _, attachment := range message.Attachments {
		a := mail.NewAttachment()
		a.SetFilename(attachment.Filename)
		a.SetContent(string(attachment.Content))
		a.SetType(attachment.ContentType)

		if attachment.Inline {
			a.SetDisposition("inline")
			if attachment.ContentID != "" {
				a.SetContentID(attachment.ContentID)
			}
		}

		m.AddAttachment(a)
	}

	// Tracking settings
	if message.Tracking != nil || p.hasTrackingConfig() {
		trackingSettings := mail.NewTrackingSettings()

		// Click tracking
		clickTracking := mail.NewClickTrackingSetting()
		if message.Tracking != nil {
			clickTracking.SetEnable(message.Tracking.ClickTracking)
		} else {
			clickTracking.SetEnable(p.config.ClickTracking)
		}
		trackingSettings.SetClickTracking(clickTracking)

		// Open tracking
		openTracking := mail.NewOpenTrackingSetting()
		if message.Tracking != nil {
			openTracking.SetEnable(message.Tracking.OpenTracking)
		} else {
			openTracking.SetEnable(p.config.OpenTracking)
		}
		trackingSettings.SetOpenTracking(openTracking)

		// Subscription tracking
		subscriptionTracking := mail.NewSubscriptionTrackingSetting()
		if message.Tracking != nil {
			subscriptionTracking.SetEnable(message.Tracking.Unsubscribe)
		} else {
			subscriptionTracking.SetEnable(p.config.Subscription)
		}
		trackingSettings.SetSubscriptionTracking(subscriptionTracking)

		m.SetTrackingSettings(trackingSettings)
	}

	// Mail settings
	mailSettings := mail.NewMailSettings()

	// Sandbox mode
	if p.config.SandboxMode {
		sandBoxMode := mail.NewSetting(true)
		mailSettings.SetSandboxMode(sandBoxMode)
	}

	m.SetMailSettings(mailSettings)

	// Template ID (if using dynamic templates)
	if p.config.TemplateID != "" {
		m.SetTemplateID(p.config.TemplateID)
	}

	// Send the email
	response, err := p.client.SendWithContext(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	// Check response status
	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s",
			response.StatusCode, response.Body)
	}

	p.logger.WithFields(logrus.Fields{
		"to":          message.To,
		"subject":     message.Subject,
		"status_code": response.StatusCode,
		"from":        p.getFromAddress(message),
	}).Info("Email sent successfully via SendGrid")

	return nil
}

// SendTemplate sends an email using SendGrid dynamic templates
func (p *SendGridProvider) SendTemplate(ctx context.Context, message *Message, templateID string, templateData map[string]interface{}) error {
	// Create the email message
	m := mail.NewV3Mail()

	// Set from address
	from := mail.NewEmail(p.config.FromName, p.getFromAddress(message))
	m.SetFrom(from)

	// Set template ID
	m.SetTemplateID(templateID)

	// Add recipients with template data
	personalization := mail.NewPersonalization()

	// To recipients
	for _, to := range message.To {
		personalization.AddTos(mail.NewEmail("", to))
	}

	// CC recipients
	for _, cc := range message.CC {
		personalization.AddCCs(mail.NewEmail("", cc))
	}

	// BCC recipients
	for _, bcc := range message.BCC {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}

	// Template data
	if templateData != nil {
		for key, value := range templateData {
			personalization.SetDynamicTemplateData(key, value)
		}
	}

	// Custom headers
	if len(message.Headers) > 0 {
		personalization.Headers = message.Headers
	}

	m.AddPersonalizations(personalization)

	// Reply-To
	if message.ReplyTo != "" {
		m.ReplyTo = mail.NewEmail("", message.ReplyTo)
	} else if p.config.ReplyTo != "" {
		m.ReplyTo = mail.NewEmail("", p.config.ReplyTo)
	}

	// Send the email
	response, err := p.client.SendWithContext(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to send template email via SendGrid: %w", err)
	}

	// Check response status
	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s",
			response.StatusCode, response.Body)
	}

	p.logger.WithFields(logrus.Fields{
		"to":          message.To,
		"template_id": templateID,
		"status_code": response.StatusCode,
		"from":        p.getFromAddress(message),
	}).Info("Template email sent successfully via SendGrid")

	return nil
}

// ValidateConfig validates the SendGrid configuration
func (p *SendGridProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("SendGrid API key is required")
	}

	if p.config.From == "" {
		return fmt.Errorf("sender email address is required")
	}

	return nil
}

// GetProviderName returns the provider name
func (p *SendGridProvider) GetProviderName() string {
	return "sendgrid"
}

// getFromAddress returns the from address for the message
func (p *SendGridProvider) getFromAddress(message *Message) string {
	if message.From != "" {
		return message.From
	}
	return p.config.From
}

// hasTrackingConfig checks if tracking configuration is available
func (p *SendGridProvider) hasTrackingConfig() bool {
	return p.config.ClickTracking || p.config.OpenTracking || p.config.Subscription
}

// TestConnection tests the SendGrid API connection
func (p *SendGridProvider) TestConnection(ctx context.Context) error {
	// Create a simple test message (won't be sent due to empty recipients)
	m := mail.NewV3Mail()
	from := mail.NewEmail(p.config.FromName, p.config.From)
	m.SetFrom(from)
	m.Subject = "Test Connection"

	// Enable sandbox mode for testing
	mailSettings := mail.NewMailSettings()
	sandBoxMode := mail.NewSetting(true)
	mailSettings.SetSandboxMode(sandBoxMode)
	m.SetMailSettings(mailSettings)

	// Try to send (will fail due to no recipients, but validates API key)
	_, err := p.client.SendWithContext(ctx, m)

	// We expect an error about missing recipients, not authentication
	if err != nil && (err.Error() == "Provide at least one recipient" ||
		err.Error() == "The to array is required for all personalizations") {
		return nil // This means the API key is valid
	}

	return err
}

// GetStats retrieves SendGrid statistics
func (p *SendGridProvider) GetStats(ctx context.Context, startDate, endDate string) (*SendGridStats, error) {
	// This would require additional SendGrid stats API implementation
	// For now, return basic structure
	return &SendGridStats{
		Date:     startDate,
		Requests: 0,
		Bounces:  0,
		Delivered: 0,
		InvalidEmails: 0,
		Processed: 0,
		SpamReports: 0,
		Suppressed: 0,
		Unsubscribes: 0,
	}, nil
}

// SendGridStats represents SendGrid email statistics
type SendGridStats struct {
	Date          string `json:"date"`
	Requests      int    `json:"requests"`
	Bounces       int    `json:"bounces"`
	Delivered     int    `json:"delivered"`
	InvalidEmails int    `json:"invalid_emails"`
	Processed     int    `json:"processed"`
	SpamReports   int    `json:"spam_reports"`
	Suppressed    int    `json:"suppressed"`
	Unsubscribes  int    `json:"unsubscribes"`
}