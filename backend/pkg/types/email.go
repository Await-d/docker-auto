package types

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EmailProvider defines the interface for email service providers
type EmailProvider interface {
	Send(ctx context.Context, message *EmailMessage) error
	ValidateConfig() error
	GetProviderName() string
}

// EmailMessage represents an email message with all possible fields
type EmailMessage struct {
	To          []string              `json:"to"`
	CC          []string              `json:"cc,omitempty"`
	BCC         []string              `json:"bcc,omitempty"`
	From        string                `json:"from"`
	ReplyTo     string                `json:"reply_to,omitempty"`
	Subject     string                `json:"subject"`
	TextBody    string                `json:"text_body,omitempty"`
	HTMLBody    string                `json:"html_body,omitempty"`
	Headers     map[string]string     `json:"headers,omitempty"`
	Attachments []EmailAttachment     `json:"attachments,omitempty"`
	Priority    EmailPriority         `json:"priority,omitempty"`
	Tracking    *EmailTrackingOptions `json:"tracking,omitempty"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
	Inline      bool   `json:"inline,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// EmailPriority represents email priority levels
type EmailPriority int

const (
	// EmailPriorityLow represents low priority email
	EmailPriorityLow EmailPriority = iota
	// EmailPriorityNormal represents normal priority email
	EmailPriorityNormal
	// EmailPriorityHigh represents high priority email
	EmailPriorityHigh
	// EmailPriorityUrgent represents urgent priority email
	EmailPriorityUrgent
)

// String returns the string representation of EmailPriority
func (p EmailPriority) String() string {
	switch p {
	case EmailPriorityLow:
		return "low"
	case EmailPriorityNormal:
		return "normal"
	case EmailPriorityHigh:
		return "high"
	case EmailPriorityUrgent:
		return "urgent"
	default:
		return "normal"
	}
}

// EmailTrackingOptions represents email tracking options
type EmailTrackingOptions struct {
	OpenTracking  bool `json:"open_tracking,omitempty"`
	ClickTracking bool `json:"click_tracking,omitempty"`
	Unsubscribe   bool `json:"unsubscribe,omitempty"`
}

// EmailSendResult represents the result of sending an email
type EmailSendResult struct {
	MessageID   string     `json:"message_id"`
	Status      string     `json:"status"`
	ProviderID  string     `json:"provider_id,omitempty"`
	SentAt      time.Time  `json:"sent_at"`
	Error       string     `json:"error,omitempty"`
	RetryCount  int        `json:"retry_count"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// EmailQueuedMessage represents a message in the send queue
type EmailQueuedMessage struct {
	ID          string        `json:"id"`
	Message     *EmailMessage `json:"message"`
	ScheduledAt time.Time     `json:"scheduled_at"`
	Attempts    int           `json:"attempts"`
	LastError   string        `json:"last_error,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// EmailDeliveryStatus represents email delivery status
type EmailDeliveryStatus struct {
	MessageID   string                 `json:"message_id"`
	Status      string                 `json:"status"` // sent, delivered, bounced, failed, etc.
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EmailProviderConfig represents email provider configuration
type EmailProviderConfig struct {
	ProviderType   string            `json:"provider_type"`   // smtp, sendgrid, mailgun, etc.
	Host           string            `json:"host,omitempty"`
	Port           int               `json:"port,omitempty"`
	Username       string            `json:"username,omitempty"`
	Password       string            `json:"password,omitempty"`
	APIKey         string            `json:"api_key,omitempty"`
	APISecret      string            `json:"api_secret,omitempty"`
	UseTLS         bool              `json:"use_tls"`
	UseSSL         bool              `json:"use_ssl"`
	FromEmail      string            `json:"from_email"`
	FromName       string            `json:"from_name,omitempty"`
	ReplyToEmail   string            `json:"reply_to_email,omitempty"`
	MaxRetries     int               `json:"max_retries"`
	RetryDelay     time.Duration     `json:"retry_delay"`
	Timeout        time.Duration     `json:"timeout"`
	RateLimit      int               `json:"rate_limit,omitempty"`      // emails per minute
	DailyLimit     int               `json:"daily_limit,omitempty"`     // emails per day
	Webhook        *EmailWebhook     `json:"webhook,omitempty"`
	CustomHeaders  map[string]string `json:"custom_headers,omitempty"`
	Enabled        bool              `json:"enabled"`
}

// EmailWebhook represents webhook configuration for email events
type EmailWebhook struct {
	URL             string            `json:"url"`
	Secret          string            `json:"secret,omitempty"`
	Events          []string          `json:"events"`          // delivered, bounced, clicked, opened, etc.
	Headers         map[string]string `json:"headers,omitempty"`
	VerifySSL       bool              `json:"verify_ssl"`
	MaxRetries      int               `json:"max_retries"`
	RetryDelay      time.Duration     `json:"retry_delay"`
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Subject     string                 `json:"subject"`
	TextBody    string                 `json:"text_body,omitempty"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	Variables   []string               `json:"variables,omitempty"`   // list of template variables
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Version     int                    `json:"version"`
	IsActive    bool                   `json:"is_active"`
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// EmailTemplateData represents data for template rendering
type EmailTemplateData struct {
	Variables map[string]interface{} `json:"variables"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EmailQueue represents an email queue with priority and scheduling
type EmailQueue struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Priority     EmailPriority           `json:"priority"`
	MaxRetries   int                     `json:"max_retries"`
	RetryDelay   time.Duration           `json:"retry_delay"`
	BatchSize    int                     `json:"batch_size"`
	RateLimit    int                     `json:"rate_limit"`    // emails per minute
	DailyLimit   int                     `json:"daily_limit"`   // emails per day
	Processors   int                     `json:"processors"`    // number of concurrent processors
	IsActive     bool                    `json:"is_active"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Stats        *EmailQueueStats        `json:"stats,omitempty"`
}

// EmailQueueStats represents statistics for an email queue
type EmailQueueStats struct {
	TotalSent       int64     `json:"total_sent"`
	TotalFailed     int64     `json:"total_failed"`
	TotalQueued     int64     `json:"total_queued"`
	TotalProcessing int64     `json:"total_processing"`
	QueueDepth      int64     `json:"queue_depth"`
	AvgProcessTime  time.Duration `json:"avg_process_time"`
	LastProcessed   *time.Time    `json:"last_processed,omitempty"`
	LastError       string        `json:"last_error,omitempty"`
	SuccessRate     float64       `json:"success_rate"`
}

// EmailNotificationConfig represents configuration for email notifications
type EmailNotificationConfig struct {
	Enabled              bool              `json:"enabled"`
	DefaultProvider      string            `json:"default_provider"`
	FallbackProviders    []string          `json:"fallback_providers,omitempty"`
	DefaultQueue         string            `json:"default_queue"`
	Templates            map[string]string `json:"templates"`            // event -> template mapping
	Recipients           map[string][]string `json:"recipients"`         // event -> recipients mapping
	Filters              map[string]interface{} `json:"filters,omitempty"` // event filters
	RateLimits           map[string]int    `json:"rate_limits,omitempty"` // per-event rate limits
	Batching             *EmailBatchConfig `json:"batching,omitempty"`
	RetryPolicy          *EmailRetryPolicy `json:"retry_policy,omitempty"`
	UnsubscribeURL       string            `json:"unsubscribe_url,omitempty"`
	TrackingEnabled      bool              `json:"tracking_enabled"`
	SchedulingEnabled    bool              `json:"scheduling_enabled"`
	MaxAttachmentSize    int64             `json:"max_attachment_size"` // bytes
	AllowedAttachments   []string          `json:"allowed_attachments,omitempty"` // file extensions
}

// EmailBatchConfig represents configuration for email batching
type EmailBatchConfig struct {
	Enabled       bool          `json:"enabled"`
	MaxBatchSize  int           `json:"max_batch_size"`
	MaxWaitTime   time.Duration `json:"max_wait_time"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// EmailRetryPolicy represents retry policy for failed emails
type EmailRetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// EmailMetrics represents email system metrics
type EmailMetrics struct {
	TotalSent            int64     `json:"total_sent"`
	TotalFailed          int64     `json:"total_failed"`
	TotalQueued          int64     `json:"total_queued"`
	TotalBounced         int64     `json:"total_bounced"`
	TotalDelivered       int64     `json:"total_delivered"`
	TotalOpened          int64     `json:"total_opened"`
	TotalClicked         int64     `json:"total_clicked"`
	TotalUnsubscribed    int64     `json:"total_unsubscribed"`
	AvgDeliveryTime      time.Duration `json:"avg_delivery_time"`
	DeliveryRate         float64   `json:"delivery_rate"`
	OpenRate             float64   `json:"open_rate"`
	ClickRate            float64   `json:"click_rate"`
	BounceRate           float64   `json:"bounce_rate"`
	UnsubscribeRate      float64   `json:"unsubscribe_rate"`
	LastSentAt           *time.Time `json:"last_sent_at,omitempty"`
	LastDeliveredAt      *time.Time `json:"last_delivered_at,omitempty"`
	QueueDepthByPriority map[EmailPriority]int64 `json:"queue_depth_by_priority"`
}

// EmailEvent represents an email-related event
type EmailEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // sent, delivered, bounced, opened, clicked, etc.
	MessageID   string                 `json:"message_id"`
	Recipient   string                 `json:"recipient"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ProviderID  string                 `json:"provider_id,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	Location    string                 `json:"location,omitempty"`
}

// EmailFilter represents filters for querying emails
type EmailFilter struct {
	Provider    string                 `json:"provider,omitempty"`
	Queue       string                 `json:"queue,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Priority    EmailPriority          `json:"priority,omitempty"`
	Recipient   string                 `json:"recipient,omitempty"`
	Subject     string                 `json:"subject,omitempty"`
	FromTime    *time.Time             `json:"from_time,omitempty"`
	ToTime      *time.Time             `json:"to_time,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
	SortBy      string                 `json:"sort_by,omitempty"`     // created_at, sent_at, priority, etc.
	SortOrder   string                 `json:"sort_order,omitempty"`  // asc, desc
}

// Validate validates the email message
func (m *EmailMessage) Validate() error {
	if len(m.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if m.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if m.TextBody == "" && m.HTMLBody == "" {
		return fmt.Errorf("either text body or HTML body is required")
	}

	// Validate email addresses (basic validation)
	for _, email := range m.To {
		if !isValidEmail(email) {
			return fmt.Errorf("invalid recipient email: %s", email)
		}
	}

	for _, email := range m.CC {
		if !isValidEmail(email) {
			return fmt.Errorf("invalid CC email: %s", email)
		}
	}

	for _, email := range m.BCC {
		if !isValidEmail(email) {
			return fmt.Errorf("invalid BCC email: %s", email)
		}
	}

	if m.From != "" && !isValidEmail(m.From) {
		return fmt.Errorf("invalid from email: %s", m.From)
	}

	if m.ReplyTo != "" && !isValidEmail(m.ReplyTo) {
		return fmt.Errorf("invalid reply-to email: %s", m.ReplyTo)
	}

	return nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Basic email validation - should be enhanced with proper regex
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// GetPriorityValue returns the numeric value of the priority
func (p EmailPriority) GetPriorityValue() int {
	return int(p)
}

// IsHigherPriority checks if this priority is higher than another
func (p EmailPriority) IsHigherPriority(other EmailPriority) bool {
	return p > other
}

// EstimateSize estimates the size of the email in bytes
func (m *EmailMessage) EstimateSize() int64 {
	size := int64(len(m.Subject) + len(m.TextBody) + len(m.HTMLBody))

	for _, attachment := range m.Attachments {
		size += int64(len(attachment.Content))
	}

	return size
}

// HasAttachments returns true if the message has attachments
func (m *EmailMessage) HasAttachments() bool {
	return len(m.Attachments) > 0
}

// IsHTML returns true if the message has HTML body
func (m *EmailMessage) IsHTML() bool {
	return m.HTMLBody != ""
}

// GetRecipientCount returns the total number of recipients
func (m *EmailMessage) GetRecipientCount() int {
	return len(m.To) + len(m.CC) + len(m.BCC)
}