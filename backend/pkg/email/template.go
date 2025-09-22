package email

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	textTemplate "text/template"
	"sync"

	"github.com/sirupsen/logrus"
)

// TemplateManager manages email templates
type TemplateManager struct {
	templateDir   string
	defaultLocale string
	templates     map[string]*EmailTemplate
	mu            sync.RWMutex
	logger        *logrus.Logger
}

// EmailTemplate represents a complete email template with multiple formats
type EmailTemplate struct {
	Name         string
	Subject      string
	TextTemplate *textTemplate.Template
	HTMLTemplate *template.Template
	Locales      map[string]*LocalizedTemplate
}

// LocalizedTemplate represents a template for a specific locale
type LocalizedTemplate struct {
	Subject      string
	TextTemplate *textTemplate.Template
	HTMLTemplate *template.Template
}

// TemplateData represents the data structure for template rendering
type TemplateData struct {
	// Common fields
	RecipientName  string                 `json:"recipient_name"`
	RecipientEmail string                 `json:"recipient_email"`
	SenderName     string                 `json:"sender_name"`
	SenderEmail    string                 `json:"sender_email"`
	Timestamp      string                 `json:"timestamp"`
	AppName        string                 `json:"app_name"`
	AppURL         string                 `json:"app_url"`

	// Notification specific
	NotificationTitle   string                 `json:"notification_title"`
	NotificationMessage string                 `json:"notification_message"`
	NotificationType    string                 `json:"notification_type"`
	Severity            string                 `json:"severity"`

	// Container specific
	ContainerName   string `json:"container_name,omitempty"`
	ContainerID     string `json:"container_id,omitempty"`
	ContainerImage  string `json:"container_image,omitempty"`
	ContainerStatus string `json:"container_status,omitempty"`

	// Update specific
	OldVersion    string `json:"old_version,omitempty"`
	NewVersion    string `json:"new_version,omitempty"`
	UpdateStatus  string `json:"update_status,omitempty"`
	UpdateSummary string `json:"update_summary,omitempty"`

	// Custom data
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(templateDir, defaultLocale string, logger *logrus.Logger) *TemplateManager {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	return &TemplateManager{
		templateDir:   templateDir,
		defaultLocale: defaultLocale,
		templates:     make(map[string]*EmailTemplate),
		logger:        logger,
	}
}

// LoadTemplates loads all templates from the template directory
func (tm *TemplateManager) LoadTemplates() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Clear existing templates
	tm.templates = make(map[string]*EmailTemplate)

	// Check if template directory exists
	if _, err := os.Stat(tm.templateDir); os.IsNotExist(err) {
		tm.logger.Warnf("Template directory does not exist: %s", tm.templateDir)
		return tm.loadDefaultTemplates()
	}

	// Walk through template directory
	err := filepath.WalkDir(tm.templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Process template files
		if strings.HasSuffix(path, ".subject.txt") {
			return tm.loadTemplate(path)
		}

		return nil
	})

	if err != nil {
		tm.logger.WithError(err).Error("Failed to load templates")
		return tm.loadDefaultTemplates()
	}

	// Load default templates if none were found
	if len(tm.templates) == 0 {
		return tm.loadDefaultTemplates()
	}

	tm.logger.Infof("Loaded %d email templates", len(tm.templates))
	return nil
}

// loadTemplate loads a single template
func (tm *TemplateManager) loadTemplate(subjectPath string) error {
	// Extract template name and locale from path
	basePath := strings.TrimSuffix(subjectPath, ".subject.txt")
	templateName := filepath.Base(basePath)
	locale := tm.defaultLocale

	// Check for locale in path (e.g., template.en.subject.txt)
	if parts := strings.Split(templateName, "."); len(parts) > 1 {
		templateName = parts[0]
		locale = parts[1]
	}

	// Read subject
	subjectBytes, err := os.ReadFile(subjectPath)
	if err != nil {
		return fmt.Errorf("failed to read subject file %s: %w", subjectPath, err)
	}
	subject := strings.TrimSpace(string(subjectBytes))

	// Get or create template
	emailTemplate, exists := tm.templates[templateName]
	if !exists {
		emailTemplate = &EmailTemplate{
			Name:    templateName,
			Locales: make(map[string]*LocalizedTemplate),
		}
		tm.templates[templateName] = emailTemplate
	}

	// Create localized template
	localizedTemplate := &LocalizedTemplate{
		Subject: subject,
	}

	// Load text template
	textPath := basePath + ".text.txt"
	if _, err := os.Stat(textPath); err == nil {
		textContent, err := os.ReadFile(textPath)
		if err != nil {
			return fmt.Errorf("failed to read text template %s: %w", textPath, err)
		}

		textTmpl, err := textTemplate.New(templateName + "_text").Parse(string(textContent))
		if err != nil {
			return fmt.Errorf("failed to parse text template %s: %w", textPath, err)
		}
		localizedTemplate.TextTemplate = textTmpl
	}

	// Load HTML template
	htmlPath := basePath + ".html"
	if _, err := os.Stat(htmlPath); err == nil {
		htmlContent, err := os.ReadFile(htmlPath)
		if err != nil {
			return fmt.Errorf("failed to read HTML template %s: %w", htmlPath, err)
		}

		htmlTmpl, err := template.New(templateName + "_html").Parse(string(htmlContent))
		if err != nil {
			return fmt.Errorf("failed to parse HTML template %s: %w", htmlPath, err)
		}
		localizedTemplate.HTMLTemplate = htmlTmpl
	}

	// Store localized template
	emailTemplate.Locales[locale] = localizedTemplate

	// Set default locale templates
	if locale == tm.defaultLocale {
		emailTemplate.Subject = subject
		emailTemplate.TextTemplate = localizedTemplate.TextTemplate
		emailTemplate.HTMLTemplate = localizedTemplate.HTMLTemplate
	}

	return nil
}

// loadDefaultTemplates loads built-in default templates
func (tm *TemplateManager) loadDefaultTemplates() error {
	defaultTemplates := map[string]map[string]string{
		"container_update": {
			"subject": "Container Update: {{.ContainerName}}",
			"text": `Hello {{.RecipientName}},

Your container "{{.ContainerName}}" has been updated.

Details:
- Container: {{.ContainerName}}
- Old Version: {{.OldVersion}}
- New Version: {{.NewVersion}}
- Status: {{.UpdateStatus}}

{{if .UpdateSummary}}
Summary: {{.UpdateSummary}}
{{end}}

Best regards,
{{.AppName}} Team`,
			"html": `<html>
<body>
<h2>Container Update Notification</h2>
<p>Hello {{.RecipientName}},</p>
<p>Your container "<strong>{{.ContainerName}}</strong>" has been updated.</p>
<h3>Details:</h3>
<ul>
<li><strong>Container:</strong> {{.ContainerName}}</li>
<li><strong>Old Version:</strong> {{.OldVersion}}</li>
<li><strong>New Version:</strong> {{.NewVersion}}</li>
<li><strong>Status:</strong> {{.UpdateStatus}}</li>
</ul>
{{if .UpdateSummary}}
<p><strong>Summary:</strong> {{.UpdateSummary}}</p>
{{end}}
<p>Best regards,<br>{{.AppName}} Team</p>
</body>
</html>`,
		},
		"container_failure": {
			"subject": "Container Failure: {{.ContainerName}}",
			"text": `Hello {{.RecipientName}},

ALERT: Container "{{.ContainerName}}" has failed.

Details:
- Container: {{.ContainerName}}
- Status: {{.ContainerStatus}}
- Message: {{.NotificationMessage}}
- Severity: {{.Severity}}
- Time: {{.Timestamp}}

Please check your Docker Auto dashboard for more details.

Best regards,
{{.AppName}} Team`,
			"html": `<html>
<body>
<h2 style="color: red;">Container Failure Alert</h2>
<p>Hello {{.RecipientName}},</p>
<p><strong>ALERT:</strong> Container "<strong>{{.ContainerName}}</strong>" has failed.</p>
<h3>Details:</h3>
<ul>
<li><strong>Container:</strong> {{.ContainerName}}</li>
<li><strong>Status:</strong> <span style="color: red;">{{.ContainerStatus}}</span></li>
<li><strong>Message:</strong> {{.NotificationMessage}}</li>
<li><strong>Severity:</strong> {{.Severity}}</li>
<li><strong>Time:</strong> {{.Timestamp}}</li>
</ul>
<p>Please check your Docker Auto dashboard for more details.</p>
<p>Best regards,<br>{{.AppName}} Team</p>
</body>
</html>`,
		},
		"system_notification": {
			"subject": "{{.NotificationTitle}}",
			"text": `Hello {{.RecipientName}},

{{.NotificationMessage}}

Type: {{.NotificationType}}
{{if .Severity}}Severity: {{.Severity}}{{end}}
Time: {{.Timestamp}}

Best regards,
{{.AppName}} Team`,
			"html": `<html>
<body>
<h2>{{.NotificationTitle}}</h2>
<p>Hello {{.RecipientName}},</p>
<p>{{.NotificationMessage}}</p>
<h3>Details:</h3>
<ul>
<li><strong>Type:</strong> {{.NotificationType}}</li>
{{if .Severity}}<li><strong>Severity:</strong> {{.Severity}}</li>{{end}}
<li><strong>Time:</strong> {{.Timestamp}}</li>
</ul>
<p>Best regards,<br>{{.AppName}} Team</p>
</body>
</html>`,
		},
	}

	for templateName, templates := range defaultTemplates {
		emailTemplate := &EmailTemplate{
			Name:    templateName,
			Subject: templates["subject"],
			Locales: make(map[string]*LocalizedTemplate),
		}

		// Parse text template
		if textContent, exists := templates["text"]; exists {
			textTmpl, err := textTemplate.New(templateName + "_text").Parse(textContent)
			if err != nil {
				return fmt.Errorf("failed to parse default text template %s: %w", templateName, err)
			}
			emailTemplate.TextTemplate = textTmpl
		}

		// Parse HTML template
		if htmlContent, exists := templates["html"]; exists {
			htmlTmpl, err := template.New(templateName + "_html").Parse(htmlContent)
			if err != nil {
				return fmt.Errorf("failed to parse default HTML template %s: %w", templateName, err)
			}
			emailTemplate.HTMLTemplate = htmlTmpl
		}

		// Store default locale
		emailTemplate.Locales[tm.defaultLocale] = &LocalizedTemplate{
			Subject:      emailTemplate.Subject,
			TextTemplate: emailTemplate.TextTemplate,
			HTMLTemplate: emailTemplate.HTMLTemplate,
		}

		tm.templates[templateName] = emailTemplate
	}

	tm.logger.Info("Loaded default email templates")
	return nil
}

// RenderTemplate renders a template with the given data
func (tm *TemplateManager) RenderTemplate(templateName string, data *TemplateData, locale string) (*RenderedTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	template, exists := tm.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	// Get localized template or fall back to default
	localizedTemplate, exists := template.Locales[locale]
	if !exists {
		localizedTemplate = template.Locales[tm.defaultLocale]
		if localizedTemplate == nil {
			return nil, fmt.Errorf("template %s not available for locale %s", templateName, locale)
		}
	}

	rendered := &RenderedTemplate{}

	// Render subject
	subjectTmpl, err := textTemplate.New("subject").Parse(localizedTemplate.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subject template: %w", err)
	}

	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}
	rendered.Subject = subjectBuf.String()

	// Render text body
	if localizedTemplate.TextTemplate != nil {
		var textBuf bytes.Buffer
		if err := localizedTemplate.TextTemplate.Execute(&textBuf, data); err != nil {
			return nil, fmt.Errorf("failed to render text template: %w", err)
		}
		rendered.TextBody = textBuf.String()
	}

	// Render HTML body
	if localizedTemplate.HTMLTemplate != nil {
		var htmlBuf bytes.Buffer
		if err := localizedTemplate.HTMLTemplate.Execute(&htmlBuf, data); err != nil {
			return nil, fmt.Errorf("failed to render HTML template: %w", err)
		}
		rendered.HTMLBody = htmlBuf.String()
	}

	return rendered, nil
}

// GetTemplate returns a template by name
func (tm *TemplateManager) GetTemplate(templateName string) (*EmailTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	template, exists := tm.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	return template, nil
}

// ListTemplates returns a list of available template names
func (tm *TemplateManager) ListTemplates() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var names []string
	for name := range tm.templates {
		names = append(names, name)
	}

	return names
}

// RenderedTemplate represents a fully rendered email template
type RenderedTemplate struct {
	Subject  string `json:"subject"`
	TextBody string `json:"text_body,omitempty"`
	HTMLBody string `json:"html_body,omitempty"`
}

// Validate validates template data
func (td *TemplateData) Validate() error {
	if td.RecipientEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	if td.AppName == "" {
		td.AppName = "Docker Auto"
	}

	if td.SenderName == "" {
		td.SenderName = td.AppName
	}

	return nil
}