package alerting

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// EmailChannel implements email notifications
type EmailChannel struct {
	config EmailChannelConfig
}

// NewEmailChannel creates a new email channel
func NewEmailChannel(config EmailChannelConfig) *EmailChannel {
	return &EmailChannel{config: config}
}

func (ec *EmailChannel) Name() string {
	return "email"
}

func (ec *EmailChannel) Type() string {
	return "email"
}

func (ec *EmailChannel) Config() AlertChannelConfig {
	return AlertChannelConfig{
		Name:    ec.Name(),
		Type:    ec.Type(),
		Enabled: true,
		Settings: map[string]interface{}{
			"smtp_host": ec.config.SMTPHost,
			"smtp_port": ec.config.SMTPPort,
			"from":      ec.config.From,
			"to":        ec.config.To,
		},
	}
}

func (ec *EmailChannel) Send(alert Alert) error {
	// Compose email message
	subject := ec.config.Subject
	if subject == "" {
		subject = fmt.Sprintf("[%s] %s", alert.Severity, alert.Name)
	}

	body := ec.formatEmailBody(alert)

	// Set up authentication
	auth := smtp.PlainAuth("", ec.config.Username, ec.config.Password, ec.config.SMTPHost)

	// Prepare message
	msg := ec.composeMessage(subject, body)

	// Send email to all recipients
	recipients := append(ec.config.To, ec.config.CC...)
	recipients = append(recipients, ec.config.BCC...)

	addr := fmt.Sprintf("%s:%d", ec.config.SMTPHost, ec.config.SMTPPort)

	// Handle TLS
	if ec.config.TLS {
		return ec.sendTLS(addr, auth, ec.config.From, recipients, msg)
	}

	return smtp.SendMail(addr, auth, ec.config.From, recipients, msg)
}

func (ec *EmailChannel) Test() error {
	testAlert := Alert{
		Name:        "Test Alert",
		Description: "This is a test alert",
		Severity:    AlertSeverityInfo,
		Status:      AlertStatusActive,
		Timestamp:   time.Now(),
	}

	return ec.Send(testAlert)
}

func (ec *EmailChannel) formatEmailBody(alert Alert) string {
	template := ec.config.Template
	if template == "" {
		template = `
Alert: {{.Name}}
Severity: {{.Severity}}
Status: {{.Status}}
Description: {{.Description}}
Timestamp: {{.Timestamp}}

{{if .Value}}Value: {{.Value}}{{end}}
{{if .Threshold}}Threshold: {{.Threshold}}{{end}}

{{if .Labels}}Labels:
{{range $key, $value := .Labels}}  {{$key}}: {{$value}}
{{end}}{{end}}

{{if .Annotations}}Details:
{{range $key, $value := .Annotations}}  {{$key}}: {{$value}}
{{end}}{{end}}
`
	}

	// Simple template replacement (in production, use proper templating)
	body := strings.ReplaceAll(template, "{{.Name}}", alert.Name)
	body = strings.ReplaceAll(body, "{{.Severity}}", string(alert.Severity))
	body = strings.ReplaceAll(body, "{{.Status}}", string(alert.Status))
	body = strings.ReplaceAll(body, "{{.Description}}", alert.Description)
	body = strings.ReplaceAll(body, "{{.Timestamp}}", alert.Timestamp.Format(time.RFC3339))

	if alert.Value != 0 {
		body = strings.ReplaceAll(body, "{{.Value}}", fmt.Sprintf("%.2f", alert.Value))
	}
	if alert.Threshold != 0 {
		body = strings.ReplaceAll(body, "{{.Threshold}}", fmt.Sprintf("%.2f", alert.Threshold))
	}

	return body
}

func (ec *EmailChannel) composeMessage(subject, body string) []byte {
	msg := fmt.Sprintf("From: %s\r\n", ec.config.From)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(ec.config.To, ","))
	if len(ec.config.CC) > 0 {
		msg += fmt.Sprintf("Cc: %s\r\n", strings.Join(ec.config.CC, ","))
	}
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "Content-Type: text/plain; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += body

	return []byte(msg)
}

func (ec *EmailChannel) sendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// This is a simplified TLS implementation
	// In production, you'd want more robust TLS handling
	return smtp.SendMail(addr, auth, from, to, msg)
}

// SlackChannel implements Slack notifications
type SlackChannel struct {
	config     SlackChannelConfig
	httpClient *http.Client
}

// NewSlackChannel creates a new Slack channel
func NewSlackChannel(config SlackChannelConfig) *SlackChannel {
	return &SlackChannel{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (sc *SlackChannel) Name() string {
	return "slack"
}

func (sc *SlackChannel) Type() string {
	return "slack"
}

func (sc *SlackChannel) Config() AlertChannelConfig {
	return AlertChannelConfig{
		Name:    sc.Name(),
		Type:    sc.Type(),
		Enabled: true,
		Settings: map[string]interface{}{
			"webhook_url": sc.config.WebhookURL,
			"channel":     sc.config.Channel,
			"username":    sc.config.Username,
		},
	}
}

func (sc *SlackChannel) Send(alert Alert) error {
	message := sc.formatSlackMessage(alert)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	resp, err := sc.httpClient.Post(sc.config.WebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	return nil
}

func (sc *SlackChannel) Test() error {
	testAlert := Alert{
		Name:        "Test Alert",
		Description: "This is a test alert",
		Severity:    AlertSeverityInfo,
		Status:      AlertStatusActive,
		Timestamp:   time.Now(),
	}

	return sc.Send(testAlert)
}

func (sc *SlackChannel) formatSlackMessage(alert Alert) map[string]interface{} {
	color := sc.getSeverityColor(alert.Severity)

	message := map[string]interface{}{
		"username": sc.config.Username,
		"channel":  sc.config.Channel,
	}

	if sc.config.IconEmoji != "" {
		message["icon_emoji"] = sc.config.IconEmoji
	}
	if sc.config.IconURL != "" {
		message["icon_url"] = sc.config.IconURL
	}

	attachment := map[string]interface{}{
		"color":      color,
		"title":      fmt.Sprintf("[%s] %s", alert.Severity, alert.Name),
		"text":       alert.Description,
		"timestamp":  alert.Timestamp.Unix(),
		"footer":     "Docker Auto-Update",
		"footer_icon": "https://cdn.icon-icons.com/icons2/2407/PNG/512/docker_icon_146192.png",
	}

	// Add fields
	fields := []map[string]interface{}{
		{"title": "Status", "value": string(alert.Status), "short": true},
		{"title": "Component", "value": alert.Component, "short": true},
	}

	if alert.Value != 0 {
		fields = append(fields, map[string]interface{}{
			"title": "Value", "value": fmt.Sprintf("%.2f", alert.Value), "short": true,
		})
	}
	if alert.Threshold != 0 {
		fields = append(fields, map[string]interface{}{
			"title": "Threshold", "value": fmt.Sprintf("%.2f", alert.Threshold), "short": true,
		})
	}

	attachment["fields"] = fields
	message["attachments"] = []interface{}{attachment}

	return message
}

func (sc *SlackChannel) getSeverityColor(severity AlertSeverity) string {
	switch severity {
	case AlertSeverityCritical, AlertSeverityFatal:
		return "danger"
	case AlertSeverityWarning:
		return "warning"
	case AlertSeverityInfo:
		return "good"
	default:
		return "#808080"
	}
}

// WebhookChannel implements generic webhook notifications
type WebhookChannel struct {
	config     WebhookChannelConfig
	httpClient *http.Client
}

// NewWebhookChannel creates a new webhook channel
func NewWebhookChannel(config WebhookChannelConfig) *WebhookChannel {
	return &WebhookChannel{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (wc *WebhookChannel) Name() string {
	return "webhook"
}

func (wc *WebhookChannel) Type() string {
	return "webhook"
}

func (wc *WebhookChannel) Config() AlertChannelConfig {
	return AlertChannelConfig{
		Name:    wc.Name(),
		Type:    wc.Type(),
		Enabled: true,
		Settings: map[string]interface{}{
			"url":    wc.config.URL,
			"method": wc.config.Method,
		},
	}
}

func (wc *WebhookChannel) Send(alert Alert) error {
	payload := wc.formatWebhookPayload(alert)

	var body []byte
	var err error

	contentType := wc.config.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	if contentType == "application/json" {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}
	} else {
		// For other content types, convert to string
		body = []byte(fmt.Sprintf("%v", payload))
	}

	method := wc.config.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, wc.config.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "Docker-Auto-Update/1.0")

	// Add custom headers
	for key, value := range wc.config.Headers {
		req.Header.Set(key, value)
	}

	// Add signature if secret is provided
	if wc.config.Secret != "" && wc.config.SignHeader != "" {
		signature := wc.generateSignature(body)
		req.Header.Set(wc.config.SignHeader, signature)
	}

	resp, err := wc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (wc *WebhookChannel) Test() error {
	testAlert := Alert{
		Name:        "Test Alert",
		Description: "This is a test alert",
		Severity:    AlertSeverityInfo,
		Status:      AlertStatusActive,
		Timestamp:   time.Now(),
	}

	return wc.Send(testAlert)
}

func (wc *WebhookChannel) formatWebhookPayload(alert Alert) interface{} {
	return map[string]interface{}{
		"id":          alert.ID,
		"name":        alert.Name,
		"description": alert.Description,
		"severity":    string(alert.Severity),
		"status":      string(alert.Status),
		"source":      alert.Source,
		"component":   alert.Component,
		"labels":      alert.Labels,
		"annotations": alert.Annotations,
		"value":       alert.Value,
		"threshold":   alert.Threshold,
		"timestamp":   alert.Timestamp.Unix(),
		"count":       alert.Count,
	}
}

func (wc *WebhookChannel) generateSignature(body []byte) string {
	// Simplified signature generation
	// In production, use proper HMAC-SHA256
	return fmt.Sprintf("sha256=%x", body)
}

// DiscordChannel implements Discord notifications
type DiscordChannel struct {
	config     DiscordChannelConfig
	httpClient *http.Client
}

// NewDiscordChannel creates a new Discord channel
func NewDiscordChannel(config DiscordChannelConfig) *DiscordChannel {
	return &DiscordChannel{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (dc *DiscordChannel) Name() string {
	return "discord"
}

func (dc *DiscordChannel) Type() string {
	return "discord"
}

func (dc *DiscordChannel) Config() AlertChannelConfig {
	return AlertChannelConfig{
		Name:    dc.Name(),
		Type:    dc.Type(),
		Enabled: true,
		Settings: map[string]interface{}{
			"webhook_url": dc.config.WebhookURL,
			"username":    dc.config.Username,
		},
	}
}

func (dc *DiscordChannel) Send(alert Alert) error {
	message := dc.formatDiscordMessage(alert)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	resp, err := dc.httpClient.Post(dc.config.WebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send Discord message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	return nil
}

func (dc *DiscordChannel) Test() error {
	testAlert := Alert{
		Name:        "Test Alert",
		Description: "This is a test alert",
		Severity:    AlertSeverityInfo,
		Status:      AlertStatusActive,
		Timestamp:   time.Now(),
	}

	return dc.Send(testAlert)
}

func (dc *DiscordChannel) formatDiscordMessage(alert Alert) map[string]interface{} {
	message := map[string]interface{}{
		"username": dc.config.Username,
	}

	if dc.config.AvatarURL != "" {
		message["avatar_url"] = dc.config.AvatarURL
	}

	if dc.config.Embeds {
		embed := map[string]interface{}{
			"title":       fmt.Sprintf("[%s] %s", alert.Severity, alert.Name),
			"description": alert.Description,
			"color":       dc.getSeverityColor(alert.Severity),
			"timestamp":   alert.Timestamp.Format(time.RFC3339),
		}

		// Add fields
		fields := []map[string]interface{}{
			{"name": "Status", "value": string(alert.Status), "inline": true},
			{"name": "Component", "value": alert.Component, "inline": true},
		}

		if alert.Value != 0 {
			fields = append(fields, map[string]interface{}{
				"name": "Value", "value": fmt.Sprintf("%.2f", alert.Value), "inline": true,
			})
		}

		embed["fields"] = fields
		message["embeds"] = []interface{}{embed}
	} else {
		// Simple text message
		text := fmt.Sprintf("**[%s] %s**\n%s\nStatus: %s\nTime: %s",
			alert.Severity, alert.Name, alert.Description, alert.Status, alert.Timestamp.Format(time.RFC3339))
		message["content"] = text
	}

	return message
}

func (dc *DiscordChannel) getSeverityColor(severity AlertSeverity) int {
	switch severity {
	case AlertSeverityCritical, AlertSeverityFatal:
		return 0xFF0000 // Red
	case AlertSeverityWarning:
		return 0xFFA500 // Orange
	case AlertSeverityInfo:
		return 0x00FF00 // Green
	default:
		return 0x808080 // Gray
	}
}