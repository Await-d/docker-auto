package webhook

import (
	"context"
	"sync"
	"time"
)

// Webhook defines the interface for webhook delivery
type Webhook interface {
	Send(ctx context.Context, payload *Payload) error
	ValidateConfig() error
	GetName() string
}

// Payload represents a webhook payload
type Payload struct {
	ID          string                 `json:"id"`
	Event       string                 `json:"event"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	Priority    Priority               `json:"priority,omitempty"`
}

// Priority represents webhook priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// DeliveryResult represents the result of webhook delivery
type DeliveryResult struct {
	ID              string        `json:"id"`
	URL             string        `json:"url"`
	StatusCode      int           `json:"status_code"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ResponseBody    string        `json:"response_body,omitempty"`
	DeliveredAt     time.Time     `json:"delivered_at"`
	Duration        time.Duration `json:"duration"`
	Success         bool          `json:"success"`
	Error           string        `json:"error,omitempty"`
	RetryCount      int           `json:"retry_count"`
}

// QueuedWebhook represents a webhook in the delivery queue
type QueuedWebhook struct {
	ID          string    `json:"id"`
	Payload     *Payload  `json:"payload"`
	URL         string    `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL             string            `json:"url"`
	Secret          string            `json:"secret,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	SignatureHeader string            `json:"signature_header,omitempty"`
	VerifySSL       bool              `json:"verify_ssl"`
	Timeout         time.Duration     `json:"timeout"`
	RetryAttempts   int               `json:"retry_attempts"`
	RetryDelay      time.Duration     `json:"retry_delay"`

	// Rate limiting
	RateLimit       int           `json:"rate_limit,omitempty"`        // requests per minute
	RateLimitBurst  int           `json:"rate_limit_burst,omitempty"`  // burst size
	RateLimitWindow time.Duration `json:"rate_limit_window,omitempty"` // window size
}

// DeliveryStatus represents webhook delivery status
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSent      DeliveryStatus = "sent"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
	DeliveryStatusCancelled DeliveryStatus = "cancelled"
)

// WebhookEvent represents different webhook event types
type WebhookEvent string

const (
	EventContainerUpdate  WebhookEvent = "container.update"
	EventContainerFailure WebhookEvent = "container.failure"
	EventContainerStart   WebhookEvent = "container.start"
	EventContainerStop    WebhookEvent = "container.stop"
	EventImageUpdate      WebhookEvent = "image.update"
	EventSystemAlert      WebhookEvent = "system.alert"
	EventSecurityAlert    WebhookEvent = "security.alert"
	EventUserAction       WebhookEvent = "user.action"
	EventScheduledTask    WebhookEvent = "task.scheduled"
	EventTaskComplete     WebhookEvent = "task.complete"
	EventTaskFailed       WebhookEvent = "task.failed"
)

// WebhookMetrics tracks webhook delivery metrics
type WebhookMetrics struct {
	mu                  sync.RWMutex  `json:"-"`
	TotalSent           int64     `json:"total_sent"`
	TotalFailed         int64     `json:"total_failed"`
	TotalQueued         int64     `json:"total_queued"`
	CurrentQueueSize    int64     `json:"current_queue_size"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	SuccessRate         float64   `json:"success_rate"`
	LastSentAt          *time.Time `json:"last_sent_at,omitempty"`
	LastFailureAt       *time.Time `json:"last_failure_at,omitempty"`

	// Status code distribution
	StatusCodes map[int]int64 `json:"status_codes,omitempty"`

	// Response time percentiles
	ResponseTimeP50 time.Duration `json:"response_time_p50,omitempty"`
	ResponseTimeP95 time.Duration `json:"response_time_p95,omitempty"`
	ResponseTimeP99 time.Duration `json:"response_time_p99,omitempty"`
}

// WebhookFilter represents filters for webhook queries
type WebhookFilter struct {
	Event      WebhookEvent    `json:"event,omitempty"`
	Status     DeliveryStatus  `json:"status,omitempty"`
	URL        string          `json:"url,omitempty"`
	From       *time.Time      `json:"from,omitempty"`
	To         *time.Time      `json:"to,omitempty"`
	Limit      int             `json:"limit,omitempty"`
	Offset     int             `json:"offset,omitempty"`
}